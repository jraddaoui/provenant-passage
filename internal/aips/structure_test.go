// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package aips

import (
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

func TestStructureHTTP(t *testing.T) {
	for _, kind := range []string{"aip", "ocfl"} {
		for _, scenario := range []string{"completed", "unknown", "queued", "processing", "error", "canceled", "missing blob", "storage error", "invalid kind", "invalid UUID"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				s, _, db, _, _ := setupUpload()
				s.Logger = logr.Discard()
				id := uuid.New()
				text := "bag/\n`-- <script>.txt\n\n0 directories (excluding root), 1 files\n"
				filename, contentType := "am-aip-tree.txt", "text/plain; charset=utf-8"
				if kind == "ocfl" {
					filename = "ocfl-tree.txt"
				}
				store := &reportStore{reader: &reportReader{Reader: strings.NewReader(text)}}
				other := &reportStore{}
				s.Source, s.Output = store, other
				if kind == "ocfl" {
					s.Source, s.Output = other, store
				}
				db.rows[id] = &persistence.AIP{UUID: id, Status: enums.StatusCompleted}
				path := "/aips/" + id.String() + "/structures/" + kind
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
				case "invalid kind":
					path = "/aips/" + id.String() + "/structures/source.tar"
					want = 400
				case "invalid UUID":
					path = "/aips/invalid/structures/" + kind
					want = 400
				}
				w := httptest.NewRecorder()
				api.Handler(s, logr.Discard()).ServeHTTP(w, httptest.NewRequest("GET", path, nil))
				if w.Code != want {
					t.Fatalf("status=%d, want %d; body=%s", w.Code, want, w.Body.String())
				}
				if other.key != "" {
					t.Fatal("accessed wrong bucket")
				}
				if want == 200 {
					if w.Body.String() != text || !store.reader.closed || store.key != OutputKey(id, filename) {
						t.Fatal("report was not streamed unchanged from its fixed bucket key and closed")
					}
					if w.Header().Get("Content-Type") != contentType ||
						w.Header().Get("Content-Length") != strconv.Itoa(len(text)) {
						t.Fatalf("incorrect structure headers: %v", w.Header())
					}
				} else if scenario != "missing blob" && scenario != "storage error" && store.key != "" {
					t.Fatal("invalid report request accessed storage")
				}
			})
		}
	}
}
