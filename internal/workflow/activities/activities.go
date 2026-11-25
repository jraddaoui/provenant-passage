// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/google/uuid"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	"gocloud.dev/blob"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/event"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

const (
	CreateWorkspaceName  = "create-workspace"
	RunImporterName      = "run-importer"
	InspectReportsName   = "inspect-reports"
	GenerateTreesName    = "generate-trees"
	UploadSourceTreeName = "upload-source-tree"
	ValidateZIPName      = "validate-zip"
	RetainReportsName    = "retain-reports"
	UpdateStatusName     = "update-aip-status"
)

type Activities struct {
	ImporterBinary, ScratchDir string
	Persistence                persistence.Service
	Events                     event.Service[*api.AIPEvent]
	Output                     *blob.Bucket
}
type (
	WorkspaceParams struct {
		UUID    uuid.UUID
		Attempt int
	}
	ImportParams struct {
		Workspace string
		Request   aips.ImportRequest
	}
	StatusParams struct {
		UUID   uuid.UUID
		Status enums.Status
		Key    string
	}
)

func heartbeat(ctx context.Context) func() {
	if !temporalsdk_activity.IsActivity(ctx) {
		return func() {}
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				temporalsdk_activity.RecordHeartbeat(ctx)
			}
		}
	})
	return func() { close(stop); wg.Wait() }
}

func (a *Activities) CreateWorkspace(ctx context.Context, p WorkspaceParams) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if p.UUID == uuid.Nil {
		return "", errors.New("missing request UUID")
	}
	if err := os.MkdirAll(a.ScratchDir, 0o700); err != nil {
		return "", err
	}
	return os.MkdirTemp(a.ScratchDir, fmt.Sprintf("%s-%d-", p.UUID, p.Attempt))
}

func (a *Activities) UpdateStatus(ctx context.Context, p StatusParams) error {
	item, err := a.Persistence.Update(ctx, p.UUID, p.Status, p.Key)
	if err != nil {
		return err
	}
	aips.PublishUpdated(ctx, a.Events, item)
	return nil
}

// ZIP delegates unchanged ZIP creation and adds heartbeats for long archives.
// Validation is a separate activity because the dependency defers close errors.
func (a *Activities) ZIP(ctx context.Context, p *archivezip.Params) (*archivezip.Result, error) {
	stop := heartbeat(ctx)
	defer stop()
	result, err := archivezip.New().Execute(ctx, p)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Activities) RetainReports(ctx context.Context, p ImportParams) error {
	stop := heartbeat(ctx)
	defer stop()
	var errs error
	for _, name := range []string{"import-event.json", "import-report.md"} {
		path := filepath.Join(p.Workspace, "reports", name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if !info.Mode().IsRegular() {
			errs = errors.Join(errs, fmt.Errorf("report is not a regular file: %s", name))
			continue
		}
		_, err = bucketupload.New(a.Output).
			Execute(ctx, &bucketupload.Params{Path: path, Key: aips.OutputKey(p.Request.UUID, name)})
		errs = errors.Join(errs, err)
	}
	return errs
}
