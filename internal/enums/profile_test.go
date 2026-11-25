// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package enums

import "testing"

func TestProfileContract(t *testing.T) {
	for _, tc := range []struct{ input, cli string }{
		{"", "b"},
		{"files_and_metadata", "b"},
		{"files_metadata_and_manifests", "c"},
		{"original_and_extracted", "a"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			p, err := ParseProfileWithDefault(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := p.CLI()
			if err != nil || got != tc.cli {
				t.Fatalf("CLI = %q, %v; want %q", got, err, tc.cli)
			}
		})
	}
	for _, input := range []string{"a", "b", "c", "FILES_AND_METADATA", "invalid"} {
		if _, err := ParseProfileWithDefault(input); err == nil {
			t.Errorf("accepted %q", input)
		}
		if _, err := Profile(input).CLI(); err == nil {
			t.Errorf("mapped invalid %q", input)
		}
	}
}
