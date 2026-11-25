// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"

	"github.com/jraddaoui/provenant-passage/internal/aips"
)

type tailBuffer struct{ data []byte }

func (b *tailBuffer) Write(p []byte) (int, error) {
	const capacity = 8192
	n := len(p)
	if len(p) > capacity {
		p = p[len(p)-capacity:]
	}
	b.data = append(b.data, p...)
	if len(b.data) > capacity {
		b.data = b.data[len(b.data)-capacity:]
	}
	return n, nil
}

func reject(err error) error {
	return temporalsdk_temporal.NewNonRetryableApplicationError(err.Error(), "invalid-import", err)
}

func (a *Activities) RunImporter(ctx context.Context, p ImportParams) error {
	stop := heartbeat(ctx)
	defer stop()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := aips.ValidateName(p.Request.Name); err != nil {
		return reject(err)
	}
	profile, err := p.Request.Profile.CLI()
	if err != nil {
		return reject(err)
	}
	if p.Request.CreatedAt.IsZero() || p.Request.CreatedAt.Nanosecond() != 0 {
		return reject(errors.New("import time must be fixed to a whole second"))
	}
	// Only this activity owns these two directories. Retry always starts fresh.
	for _, name := range []string{"object", "reports"} {
		if err := os.RemoveAll(filepath.Join(p.Workspace, name)); err != nil {
			return err
		}
	}
	// #nosec G204 -- trusted configured binary, validated basename and explicit arguments; no shell.
	cmd := exec.CommandContext(ctx, a.ImporterBinary, "--profile", profile,
		"--import-time", p.Request.CreatedAt.UTC().Format(time.RFC3339),
		"--reports", filepath.Join(p.Workspace, "reports"),
		filepath.Join(p.Workspace, p.Request.Name), filepath.Join(p.Workspace, "object"))
	cmd.Dir = p.Workspace
	var output tailBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return reject(fmt.Errorf("external importer failed: %w: %s", err, output.data))
	}
	return nil
}

// Report fields are an external CLI contract, not imported converter types.
type importReport struct {
	SchemaVersion string `json:"schemaVersion"`
	ObjectID      string `json:"objectId"`
	Profile       string `json:"profile"`
	EventType     string `json:"eventType"`
	EventDateTime string `json:"eventDateTime"`
	EventOutcome  string `json:"eventOutcome"`
	Source        struct {
		Name      string            `json:"name"`
		Kind      string            `json:"kind"`
		BagUUID   string            `json:"bagUuid"`
		Digest    map[string]string `json:"digest"`
		SizeBytes int64             `json:"sizeBytes"`
	} `json:"source"`
	Equivalence struct {
		SourcePayloadEntries *int            `json:"sourcePayloadEntries"`
		PreservedByDigest    *int            `json:"preservedByDigest"`
		MetsRoundTrip        *bool           `json:"metsRoundTrip"`
		Dropped              json.RawMessage `json:"dropped"`
		Missing              json.RawMessage `json:"missing"`
	} `json:"equivalence"`
}

func (a *Activities) InspectReports(ctx context.Context, p ImportParams) error {
	stop := heartbeat(ctx)
	defer stop()
	if err := inspectReports(ctx, p); err != nil {
		return reject(err)
	}
	return nil
}

func inspectReports(ctx context.Context, p ImportParams) error {
	data, err := os.ReadFile(filepath.Join(p.Workspace, "reports", "import-event.json"))
	if err != nil {
		return err
	}
	var report importReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("invalid import-event.json: %w", err)
	}
	profile, err := p.Request.Profile.CLI()
	if err != nil {
		return err
	}
	if report.SchemaVersion != "0.1" || report.EventType != "ingestion" || report.Profile != profile ||
		report.EventDateTime != p.Request.CreatedAt.UTC().Format(time.RFC3339) {
		return errors.New("import report schema/profile/time mismatch")
	}
	if report.EventOutcome != "success" {
		return fmt.Errorf("unclean importer outcome: %q", report.EventOutcome)
	}
	eq := report.Equivalence
	if eq.SourcePayloadEntries == nil || eq.PreservedByDigest == nil || eq.MetsRoundTrip == nil ||
		len(eq.Missing) == 0 ||
		len(eq.Dropped) == 0 {
		return errors.New("missing equivalence evidence")
	}
	var missing []string
	var dropped []struct{ Path, Reason string }
	if err := json.Unmarshal(eq.Missing, &missing); err != nil {
		return err
	}
	if err := json.Unmarshal(eq.Dropped, &dropped); err != nil {
		return err
	}
	if !*eq.MetsRoundTrip || len(missing) != 0 || *eq.SourcePayloadEntries < 0 || *eq.PreservedByDigest < 0 ||
		*eq.PreservedByDigest+len(dropped) != *eq.SourcePayloadEntries {
		return errors.New("failed METS equivalence or unaccounted source content")
	}
	for _, drop := range dropped {
		if drop.Path == "" || drop.Reason == "" {
			return errors.New("invalid loss accounting")
		}
	}
	if report.Source.Kind != "archive" || report.Source.Name != p.Request.Name || report.Source.BagUUID == "" ||
		report.ObjectID != "urn:uuid:"+report.Source.BagUUID {
		return errors.New("report changed source archive identity")
	}
	source, err := os.Open(filepath.Join(p.Workspace, p.Request.Name))
	if err != nil {
		return err
	}
	defer source.Close()
	h := sha256.New()
	n, err := io.Copy(h, source)
	if err != nil {
		return err
	}
	if report.Source.SizeBytes != n || report.Source.Digest["sha256"] != hex.EncodeToString(h.Sum(nil)) {
		return errors.New("report source digest mismatch")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	md, err := os.ReadFile(filepath.Join(p.Workspace, "reports", "import-report.md"))
	if err != nil {
		return err
	}
	if len(md) == 0 {
		return errors.New("empty import-report.md")
	}
	return nil
}
