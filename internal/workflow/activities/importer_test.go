// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/testfixture"
)

func importerBinary(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../../bin/provenant-import")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal("run make importer before activity tests: ", err)
	}
	return p
}

func TestExternalImporterProfilesAndReports(t *testing.T) {
	for _, profile := range []enums.Profile{enums.ProfileFilesAndMetadata, enums.ProfileFilesMetadataAndManifests, enums.ProfileOriginalAndExtracted} {
		t.Run(string(profile), func(t *testing.T) {
			root := t.TempDir()
			bag := testfixture.Default()
			bag.CreateDate = ""
			source := bag.WriteTar(t, root)
			r := aips.ImportRequest{
				UUID:      uuid.New(),
				Name:      filepath.Base(source),
				Profile:   profile,
				CreatedAt: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
			}
			a := &Activities{ImporterBinary: importerBinary(t), ScratchDir: root}
			p := ImportParams{Workspace: root, Request: r}
			if err := a.RunImporter(t.Context(), p); err != nil {
				t.Fatal(err)
			}
			if err := a.InspectReports(t.Context(), p); err != nil {
				t.Fatal(err)
			}
			// A retry must replace output directories, not append another version.
			if err := a.RunImporter(t.Context(), p); err != nil {
				t.Fatal(err)
			}
			if err := a.InspectReports(t.Context(), p); err != nil {
				t.Fatal(err)
			}
			for _, scenario := range []string{"warning", "missing", "profile", "time", "source", "equivalence", "absent accounting", "invalid json", "missing markdown"} {
				t.Run(scenario, func(t *testing.T) {
					path := filepath.Join(root, "reports", "import-event.json")
					original, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					defer os.WriteFile(path, original, 0o600)
					var report map[string]any
					if err := json.Unmarshal(original, &report); err != nil {
						t.Fatal(err)
					}
					eq := report["equivalence"].(map[string]any)
					switch scenario {
					case "warning":
						report["eventOutcome"] = "warning"
					case "missing":
						eq["missing"] = []string{"lost file"}
					case "profile":
						report["profile"] = "invalid"
					case "time":
						report["eventDateTime"] = "2020-01-01T00:00:00Z"
					case "source":
						report["source"].(map[string]any)["name"] = "renamed.tar"
					case "equivalence":
						eq["metsRoundTrip"] = false
					case "absent accounting":
						delete(eq, "missing")
					case "missing markdown":
						md := filepath.Join(root, "reports", "import-report.md")
						b, _ := os.ReadFile(md)
						os.Remove(md)
						defer os.WriteFile(md, b, 0o600)
					}
					data, _ := json.Marshal(report)
					if scenario == "invalid json" {
						data = []byte("{")
					}
					if err := os.WriteFile(path, data, 0o600); err != nil {
						t.Fatal(err)
					}
					if err := a.InspectReports(t.Context(), p); err == nil {
						t.Fatal("invalid report accepted")
					}
				})
			}
		})
	}
}

func TestImporterWarningAndCancellation(t *testing.T) {
	root := t.TempDir()
	bag := testfixture.Default()
	bag.BadDigestFor = []string{"objects/alpha.txt"}
	source := bag.WriteTar(t, root)
	a := &Activities{ImporterBinary: importerBinary(t), ScratchDir: root}
	p := ImportParams{
		Workspace: root,
		Request: aips.ImportRequest{
			UUID:      uuid.New(),
			Name:      filepath.Base(source),
			Profile:   enums.ProfileFilesAndMetadata,
			CreatedAt: time.Now().UTC().Truncate(time.Second),
		},
	}
	if err := a.RunImporter(t.Context(), p); err != nil {
		t.Fatal("warning CLI should exit zero: ", err)
	}
	if err := a.InspectReports(t.Context(), p); err == nil {
		t.Fatal("warning passed report gate")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.RunImporter(ctx, p); err == nil {
		t.Fatal("canceled import succeeded")
	}
	if err := os.WriteFile(source, []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := a.RunImporter(t.Context(), p); err == nil {
		t.Fatal("broken import succeeded")
	}
}
