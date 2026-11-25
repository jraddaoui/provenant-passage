// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package persistence

import "testing"

func TestPage(t *testing.T) {
	for _, tc := range []struct{ limit, offset, wantLimit, wantOffset int }{
		{0, 0, 20, 0},
		{-1, -3, 50000, 0},
		{1, 5, 1, 5},
		{50001, 0, 50000, 0},
		{50000, 0, 50000, 0},
	} {
		l, o := Page(tc.limit, tc.offset)
		if l != tc.wantLimit || o != tc.wantOffset {
			t.Fatalf("Page(%d,%d) = %d,%d", tc.limit, tc.offset, l, o)
		}
	}
}
