// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"archive/tar"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/jraddaoui/provenant-passage/internal/aips"
	"github.com/jraddaoui/provenant-passage/internal/enums"
	"github.com/jraddaoui/provenant-passage/internal/testfixture"
)

func writeTreeTar(t *testing.T, entries []tar.Header) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "source.tar")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	w := tar.NewWriter(f)
	for _, entry := range entries {
		if err := w.WriteHeader(&entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestArchiveTree(t *testing.T) {
	// Unsorted headers, implicit parents, an explicit empty directory, and hidden files.
	name := writeTreeTar(t, []tar.Header{
		{Name: "./bag/z.txt", Typeflag: tar.TypeReg},
		{Name: "bag/data/objects/a.txt", Typeflag: tar.TypeReg},
		{Name: "bag/empty/", Typeflag: tar.TypeDir},
		{Name: "bag/.hidden", Typeflag: tar.TypeReg},
		{Name: "bag/data/", Typeflag: tar.TypeDir},
	})
	got, err := archiveTree(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	want := "bag/\n|-- data/\n|   `-- objects/\n|       `-- a.txt\n|-- empty/\n|-- .hidden\n`-- z.txt\n\n3 directories (excluding root), 3 files\n"
	if got != want {
		t.Fatalf("tree mismatch:\n%s\nwant:\n%s", got, want)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := archiveTree(ctx, name); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestArchiveTreeRejectsAmbiguousEntries(t *testing.T) {
	for _, entries := range [][]tar.Header{
		{{Name: "../escape", Typeflag: tar.TypeReg}},
		{{Name: "/absolute", Typeflag: tar.TypeReg}},
		{{Name: "bag/a", Typeflag: tar.TypeReg}, {Name: "bag/a/b", Typeflag: tar.TypeReg}},
		{{Name: "bag/a", Typeflag: tar.TypeReg}, {Name: "bag/a", Typeflag: tar.TypeReg}},
		{{Name: "bag/link", Typeflag: tar.TypeSymlink, Linkname: "/outside"}},
	} {
		if _, err := archiveTree(t.Context(), writeTreeTar(t, entries)); err == nil {
			t.Fatal("ambiguous archive accepted")
		}
	}
}

func TestTreeEscapesNamesAndPreservesTopLevelEntries(t *testing.T) {
	name := writeTreeTar(t, []tar.Header{
		{Name: "data/line\nbreak.txt", Typeflag: tar.TypeReg},
		{Name: "bagit.txt", Typeflag: tar.TypeReg},
	})
	got, err := archiveTree(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"line\nbreak.txt"`) || !strings.HasPrefix(got, "source/\n") ||
		!strings.HasSuffix(got, "1 directories (excluding root), 2 files\n") {
		t.Fatal(got)
	}
}

func TestGenerateTreesForAllProfilesAndArchives(t *testing.T) {
	for _, ext := range []string{".tar", ".7z"} {
		for _, profile := range []enums.Profile{enums.ProfileFilesAndMetadata, enums.ProfileFilesMetadataAndManifests, enums.ProfileOriginalAndExtracted} {
			t.Run(ext+"/"+string(profile), func(t *testing.T) {
				root := t.TempDir()
				source := testfixture.MD5Only().WriteTar(t, root)
				if ext == ".7z" {
					data, err := os.ReadFile(testfixture.Fixture7z(t))
					if err != nil {
						t.Fatal(err)
					}
					sevenzipPath := filepath.Join(root, "source.7z")
					// #nosec G703 -- fixed basename inside t.TempDir; data is the checked-in synthetic fixture.
					if err := os.WriteFile(sevenzipPath, data, 0o600); err != nil {
						t.Fatal(err)
					}
					source = sevenzipPath
				}
				a := &Activities{ImporterBinary: importerBinary(t)}
				p := ImportParams{
					Workspace: root,
					Request: aips.ImportRequest{
						UUID:      uuid.New(),
						Name:      filepath.Base(source),
						Profile:   profile,
						CreatedAt: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC),
					},
				}
				if err := a.RunImporter(t.Context(), p); err != nil {
					t.Fatal(err)
				}
				if err := a.GenerateTrees(t.Context(), p); err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(filepath.Join(root, "am-aip-tree.txt"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(before), "bagit.txt") || !strings.Contains(string(before), "alpha.txt") {
					t.Fatal(string(before))
				}
				data, err := os.ReadFile(filepath.Join(root, "reports", "ocfl-tree.txt"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(string(data), "ocfl/\n") ||
					!strings.Contains(string(data), "0=ocfl_object_1.1") ||
					strings.Contains(string(data), "tree.txt") {
					t.Fatal(string(data))
				}
				if err := a.GenerateTrees(t.Context(), p); err != nil {
					t.Fatal(err)
				}
				after, err := os.ReadFile(filepath.Join(root, "am-aip-tree.txt"))
				if err != nil {
					t.Fatal(err)
				}
				if string(before) != string(after) {
					t.Fatal("retry changed tree")
				}
			})
		}
	}
}

func TestObjectTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "v1", "content", "empty"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "inventory.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := objectTree(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	want := "ocfl/\n|-- v1/\n|   `-- content/\n|       `-- empty/\n`-- inventory.json\n\n3 directories (excluding root), 1 files\n"
	if got != want {
		t.Fatal(got)
	}
	if err := os.Symlink("/outside", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := objectTree(t.Context(), root); err == nil {
		t.Fatal("symlink accepted")
	}
}
