// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateZIP(t *testing.T) {
	for _, scenario := range []string{"valid", "truncated", "crc", "outside object", "missing inventory"} {
		t.Run(scenario, func(t *testing.T) {
			var b bytes.Buffer
			z := zip.NewWriter(&b)
			name := "object/inventory.json"
			if scenario == "missing inventory" {
				name = "object/other.json"
			}
			if scenario == "outside object" {
				name = "../inventory.json"
			}
			w, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("payload")); err != nil {
				t.Fatal(err)
			}
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
			data := b.Bytes()
			if scenario == "truncated" {
				data = data[:len(data)-20]
			}
			if scenario == "crc" {
				data[bytes.Index(data, []byte("payload"))] = 'X'
			}
			p := filepath.Join(t.TempDir(), "object.zip")
			if err := os.WriteFile(p, data, 0o600); err != nil {
				t.Fatal(err)
			}
			err = (&Activities{}).ValidateZIP(t.Context(), p)
			if scenario == "valid" && err != nil {
				t.Fatal(err)
			}
			if scenario != "valid" && err == nil {
				t.Fatal("invalid ZIP accepted")
			}
		})
	}
}
