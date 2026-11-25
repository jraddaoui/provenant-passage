// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"testing"

	"github.com/google/uuid"

	api "github.com/jraddaoui/provenant-passage/internal/api/gen/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

type missingOutput struct{ Store }

func (missingOutput) NewReader(context.Context, string) (BlobReader, error) {
	return nil, ErrObjectNotFound
}

func TestDownloadFailureStates(t *testing.T) {
	for _, scenario := range []string{"unknown", "queued", "processing", "error", "canceled", "missing blob"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, db, _, _ := setupUpload()
			s.Output = missingOutput{}
			id := uuid.New()
			want := "not_ready"
			if scenario == "unknown" {
				want = "not_found"
			} else {
				aip := &persistence.AIP{UUID: id, Status: enums.Status(scenario)}
				if scenario == "missing blob" {
					aip.Status = enums.StatusCompleted
					key := OutputKey(id, "object.zip")
					aip.OCFLObjectKey = &key
					want = "not_found"
				}
				db.rows[id] = aip
			}
			result, body, err := s.Download(t.Context(), &api.DownloadPayload{UUID: id.String()})
			if result != nil || body != nil || err == nil {
				t.Fatal("invalid download returned content")
			}
			if named, ok := err.(interface{ ErrorName() string }); !ok || named.ErrorName() != want {
				t.Fatalf("wanted %s, got %v", want, err)
			}
		})
	}
}
