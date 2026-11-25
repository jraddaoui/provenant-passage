// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/api"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/persistence"
)

type reportReader struct {
	*strings.Reader
	closed bool
}

func (r *reportReader) Close() error { r.closed = true; return nil }

type reportStore struct {
	Store
	reader *reportReader
	key    string
	err    error
}

func (s *reportStore) NewReader(_ context.Context, key string) (BlobReader, error) {
	s.key = key
	return s.reader, s.err
}

func TestReportHTTP(t *testing.T) {
	for _, format := range []string{"json", "md"} {
		for _, scenario := range []string{"completed", "unknown", "queued", "processing", "error", "canceled", "missing blob", "storage error", "invalid format", "invalid UUID"} {
			t.Run(format+"/"+scenario, func(t *testing.T) {
				s, _, db, _, _ := setupUpload()
				s.Logger = logr.Discard()
				id := uuid.New()
				text, filename, contentType := "{\n  \"status\": \"success\"\n}\n", "import-event.json", "application/json"
				if format == "md" {
					text, filename, contentType = "# Import report\n\n| Result | Count |\n| --- | --- |\n| Preserved | 18 |\n", "import-report.md", "text/markdown; charset=utf-8"
				}
				store := &reportStore{reader: &reportReader{Reader: strings.NewReader(text)}}
				s.Output = store
				db.rows[id] = &persistence.AIP{UUID: id, Status: enums.StatusCompleted}
				path := "/aips/" + id.String() + "/reports/" + format
				want := 200
				switch scenario {
				case "unknown":
					delete(db.rows, id)
					want = 404
				case "queued", "processing", "error", "canceled":
					db.rows[id].Status = enums.Status(scenario)
					want = 409
				case "missing blob":
					store.err = ErrObjectNotFound
					want = 404
				case "storage error":
					store.err = errInjected
					want = 500
				case "invalid format":
					path = "/aips/" + id.String() + "/reports/source.tar"
					want = 400
				case "invalid UUID":
					path = "/aips/invalid/reports/" + format
					want = 400
				}
				w := httptest.NewRecorder()
				api.Handler(s, logr.Discard()).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
				if w.Code != want {
					t.Fatalf("status=%d, want %d; body=%s", w.Code, want, w.Body.String())
				}
				if want == 200 {
					if w.Body.String() != text || !store.reader.closed || store.key != OutputKey(id, filename) {
						t.Fatal("report was not streamed unchanged from its fixed bucket key and closed")
					}
					if w.Header().Get("Content-Type") != contentType ||
						w.Header().Get("Content-Length") != strconv.Itoa(len(text)) {
						t.Fatalf("incorrect report headers: %v", w.Header())
					}
				} else if scenario != "missing blob" && scenario != "storage error" && store.key != "" {
					t.Fatal("invalid report request accessed storage")
				}
			})
		}
	}
}
