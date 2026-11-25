// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package persistence

// Page retains Enduro's limit/offset semantics.
func Page(limit, offset int) (int, int) {
	switch {
	case limit == 0:
		limit = 20
	case limit < 0 || limit > 50000:
		limit = 50000
	}
	return limit, max(0, offset)
}
