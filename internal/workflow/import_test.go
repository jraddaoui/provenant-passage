// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/artefactual-sdps/temporal-activities/archivezip"
	"github.com/artefactual-sdps/temporal-activities/bucketdownload"
	"github.com/artefactual-sdps/temporal-activities/bucketupload"
	"github.com/artefactual-sdps/temporal-activities/removepaths"
	"github.com/google/uuid"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	temporalsdk_testsuite "go.temporal.io/sdk/testsuite"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/workflow/activities"
)

func TestImportWorkflow(t *testing.T) {
	for _, scenario := range []string{"success", "import failure", "report failure", "zip failure", "upload object.zip", "upload import-event.json", "upload import-report.md", "terminal database failure", "activity retry", "session retry"} {
		t.Run(scenario, func(t *testing.T) {
			var suite temporalsdk_testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetWorkerOptions(temporalsdk_worker.Options{EnableSessionWorker: true})
			var mu sync.Mutex
			queues := map[string]string{}
			statuses := []activities.StatusParams{}
			uploads := []string{}
			workspaceCount, downloadCount, runCount := 0, 0, 0
			record := func(ctx context.Context, name string) {
				mu.Lock()
				defer mu.Unlock()
				queues[name] = temporalsdk_activity.GetInfo(ctx).TaskQueue
			}
			fail := func() error {
				return temporalsdk_temporal.NewNonRetryableApplicationError("injected failure", "test-failure", nil)
			}
			register := func(name string, fn any) {
				env.RegisterActivityWithOptions(fn, temporalsdk_activity.RegisterOptions{Name: name})
			}
			register(activities.UpdateStatusName, func(ctx context.Context, p activities.StatusParams) error {
				record(ctx, activities.UpdateStatusName)
				mu.Lock()
				defer mu.Unlock()
				statuses = append(statuses, p)
				if scenario == "terminal database failure" && p.Status != enums.StatusProcessing {
					return fail()
				}
				return nil
			})
			register(
				activities.CreateWorkspaceName,
				func(ctx context.Context, p activities.WorkspaceParams) (string, error) {
					record(ctx, activities.CreateWorkspaceName)
					workspaceCount++
					return "/scratch/attempt-" + strconv.Itoa(p.Attempt), nil
				},
			)
			register(
				bucketdownload.Name,
				func(ctx context.Context, p *bucketdownload.Params) (*bucketdownload.Result, error) {
					record(ctx, bucketdownload.Name)
					downloadCount++
					if p.FileName != "Original.tar" {
						return nil, errors.New("archive renamed")
					}
					if scenario == "activity retry" && downloadCount == 1 {
						return nil, errors.New("temporary transport failure")
					}
					return &bucketdownload.Result{FilePath: filepath.Join(p.DirPath, p.FileName)}, nil
				},
			)
			req := aips.ImportRequest{
				UUID:      uuid.New(),
				Name:      "Original.tar",
				SourceKey: "aips/source/Original.tar",
				Profile:   enums.ProfileOriginalAndExtracted,
				CreatedAt: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
			}
			register(activities.RunImporterName, func(ctx context.Context, p activities.ImportParams) error {
				record(ctx, activities.RunImporterName)
				runCount++
				if p.Request != req {
					return errors.New("request changed across workflow")
				}
				if scenario == "import failure" {
					return fail()
				}
				if scenario == "session retry" && runCount == 1 {
					return temporalsdk_temporal.NewCanceledError("session lost")
				}
				return nil
			})
			register(activities.InspectReportsName, func(ctx context.Context, p activities.ImportParams) error {
				record(ctx, activities.InspectReportsName)
				if scenario == "report failure" {
					return fail()
				}
				return nil
			})
			register(archivezip.Name, func(ctx context.Context, p *archivezip.Params) (*archivezip.Result, error) {
				record(ctx, archivezip.Name)
				return &archivezip.Result{Path: p.DestPath}, nil
			})
			register(activities.ValidateZIPName, func(ctx context.Context, p string) error {
				record(ctx, activities.ValidateZIPName)
				if scenario == "zip failure" {
					return fail()
				}
				return nil
			})
			register(
				bucketupload.Name,
				func(ctx context.Context, p *bucketupload.Params) (*bucketupload.Result, error) {
					record(ctx, bucketupload.Name)
					uploads = append(uploads, p.Key)
					if scenario == "upload "+filepath.Base(p.Path) {
						return nil, fail()
					}
					return &bucketupload.Result{Key: p.Key}, nil
				},
			)
			register(activities.RetainReportsName, func(ctx context.Context, p activities.ImportParams) error {
				record(ctx, activities.RetainReportsName)
				return nil
			})
			register(removepaths.Name, func(ctx context.Context, p *removepaths.Params) (*removepaths.Result, error) {
				record(ctx, removepaths.Name)
				return &removepaths.Result{}, nil
			})
			w := &Workflow{
				Options: Options{
					WorkflowQueue:           "workflow",
					ActivityQueue:           "sessions",
					SessionCreationTimeout:  time.Minute,
					SessionExecutionTimeout: time.Hour,
					ActivityTimeout:         time.Minute,
					HeartbeatTimeout:        10 * time.Second,
				},
			}
			env.ExecuteWorkflow(w.Execute, req)
			success := scenario == "success" || scenario == "activity retry" || scenario == "session retry"
			if success && env.GetWorkflowError() != nil {
				t.Fatal(env.GetWorkflowError())
			}
			if !success && env.GetWorkflowError() == nil {
				t.Fatal("failed pipeline completed")
			}
			last := statuses[len(statuses)-1]
			if success &&
				(last.Status != enums.StatusCompleted || last.Key != aips.OutputKey(req.UUID, "object.zip") || len(uploads) != 3) {
				t.Fatalf("incorrect success %+v uploads %v", statuses, uploads)
			}
			if !success && last.Status != enums.StatusError {
				t.Fatalf("failure status %v", statuses)
			}
			if scenario == "session retry" && (workspaceCount != 2 || downloadCount != 2) {
				t.Fatal("lost session reused local files")
			}
			if scenario == "activity retry" && downloadCount != 2 {
				t.Fatal("transient activity not retried")
			}
			var sessionQueue string
			for name, q := range queues {
				if name == activities.UpdateStatusName {
					if q != "workflow" {
						t.Error("DB activity wrong queue")
					}
					continue
				}
				if q == "" || q == "workflow" || q == "sessions" {
					t.Errorf("%s not session-routed: %s", name, q)
				}
				if sessionQueue == "" {
					sessionQueue = q
				}
				if q != sessionQueue {
					t.Errorf("workspace activity %s left owning worker", name)
				}
			}
		})
	}
}
