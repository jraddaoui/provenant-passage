// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package design_test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestGeneratedRoutes(t *testing.T) {
	data, err := os.ReadFile("../gen/http/openapi3.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/aips", "/aips/{uuid}", "/aips/upload", "/aips/{uuid}/download", "/aips/{uuid}/reports/{format}", "/aips/monitor", "/aips/{uuid}/structures/{kind}"} {
		if _, ok := spec.Paths[path]; !ok {
			t.Errorf("missing contract route %s", path)
		}
	}
}
