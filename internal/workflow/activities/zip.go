// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

func (a *Activities) ValidateZIP(ctx context.Context, name string) error {
	stop := heartbeat(ctx)
	defer stop()
	z, err := zip.OpenReader(name)
	if err != nil {
		return fmt.Errorf("reopen OCFL ZIP: %w", err)
	}
	defer z.Close()
	files := 0
	seen := map[string]bool{}
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !strings.HasPrefix(f.Name, "object/") || path.Clean(f.Name) != strings.TrimSuffix(f.Name, "/") ||
			strings.Contains(f.Name, "\\") ||
			seen[f.Name] {
			return fmt.Errorf("invalid ZIP entry %q", f.Name)
		}
		seen[f.Name] = true
		if f.FileInfo().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() {
			return errors.New("non-regular ZIP member")
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		// #nosec G110 -- reads our freshly generated ZIP, never an uploaded ZIP.
		_, readErr := io.Copy(io.Discard, r)
		closeErr := r.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return fmt.Errorf("read ZIP member: %w", err)
		}
		files++
	}
	if files == 0 || !seen["object/inventory.json"] {
		return errors.New("ZIP has no object inventory")
	}
	return nil
}
