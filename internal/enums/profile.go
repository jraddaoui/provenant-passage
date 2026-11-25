// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package enums

import "fmt"

/*
ENUM(
files_and_metadata
files_metadata_and_manifests
original_and_extracted
)
*/
type Profile string

// CLI translates only at the external importer boundary.
func (p Profile) CLI() (string, error) {
	switch p {
	case ProfileFilesAndMetadata:
		return "b", nil
	case ProfileFilesMetadataAndManifests:
		return "c", nil
	case ProfileOriginalAndExtracted:
		return "a", nil
	default:
		return "", fmt.Errorf("invalid profile %q", p)
	}
}
