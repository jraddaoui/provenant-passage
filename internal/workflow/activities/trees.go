// SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

package activities

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/mholt/archives"

	"github.com/jraddaoui/provenant-passage/internal/aips"
)

type treeNode struct {
	name     string
	dir      bool
	children map[string]*treeNode
}

func (n *treeNode) add(name string, dir bool) error {
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	name = strings.TrimSuffix(name, "/")
	if (name == "" || name == ".") && dir {
		return nil
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid tree path %q", name)
		}
		isDir := i < len(parts)-1 || dir
		if n.children == nil {
			n.children = map[string]*treeNode{}
		}
		child, exists := n.children[part]
		if exists {
			if child.dir != isDir || !isDir {
				return fmt.Errorf("conflicting tree path %q", name)
			}
		} else {
			child = &treeNode{name: part, dir: isDir}
			n.children[part] = child
		}
		n = child
	}
	return nil
}

// Quote names with control characters or backslashes so they cannot forge tree lines.
func treeLabel(name string) string {
	for _, r := range name {
		if !unicode.IsPrint(r) || r == '\\' {
			return strconv.Quote(name)
		}
	}
	return name
}

func (n *treeNode) render(ctx context.Context) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "%s/\n", treeLabel(n.name))
	dirs, files := 0, 0
	var visit func(*treeNode, string) error
	visit = func(parent *treeNode, prefix string) error {
		children := make([]*treeNode, 0, len(parent.children))
		for _, child := range parent.children {
			children = append(children, child)
		}
		slices.SortFunc(children, func(a, b *treeNode) int {
			if a.dir != b.dir {
				if a.dir {
					return -1
				}
				return 1
			}
			return strings.Compare(a.name, b.name)
		})
		for i, child := range children {
			if err := ctx.Err(); err != nil {
				return err
			}
			branch, indent := "|-- ", "|   "
			if i == len(children)-1 {
				branch, indent = "`-- ", "    "
			}
			name := treeLabel(child.name)
			if child.dir {
				name += "/"
				dirs++
			} else {
				files++
			}
			fmt.Fprintf(&b, "%s%s%s\n", prefix, branch, name)
			if err := visit(child, prefix+indent); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(n, ""); err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "\n%d directories (excluding root), %d files\n", dirs, files)
	return b.String(), ctx.Err()
}

func archiveTree(ctx context.Context, filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var format archives.Extractor
	switch filepath.Ext(filename) {
	case ".tar":
		format = archives.Tar{}
	case ".7z":
		format = archives.SevenZip{}
	default:
		return "", errors.New("unsupported tree archive format")
	}
	root := &treeNode{name: strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), dir: true}
	// Extract invokes the callback for headers only. File contents are never opened or written.
	err = format.Extract(ctx, f, func(ctx context.Context, entry archives.FileInfo) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() && !entry.Mode().IsRegular() || entry.LinkTarget != "" {
			return fmt.Errorf("unsupported tree entry %q", entry.NameInArchive)
		}
		return root.add(entry.NameInArchive, entry.IsDir())
	})
	if err != nil {
		return "", err
	}
	if len(root.children) == 0 {
		return "", errors.New("empty source archive")
	}
	// Use the archive's own root when present; otherwise label its top level with the archive basename.
	if len(root.children) == 1 {
		for _, child := range root.children {
			if child.dir {
				root = child
			}
		}
	}
	return root.render(ctx)
}

func objectTree(ctx context.Context, dir string) (string, error) {
	root := &treeNode{name: "ocfl", dir: true}
	err := filepath.WalkDir(dir, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported tree entry %q", name)
		}
		if name == dir {
			if !entry.IsDir() {
				return errors.New("OCFL tree root is not a directory")
			}
			return nil
		}
		rel, err := filepath.Rel(dir, name)
		if err != nil {
			return err
		}
		return root.add(filepath.ToSlash(rel), entry.IsDir())
	})
	if err != nil {
		return "", err
	}
	return root.render(ctx)
}

func (a *Activities) GenerateTrees(ctx context.Context, p ImportParams) error {
	stop := heartbeat(ctx)
	defer stop()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := aips.ValidateName(p.Request.Name); err != nil {
		return reject(err)
	}
	source, err := archiveTree(ctx, filepath.Join(p.Workspace, p.Request.Name))
	if err != nil {
		return fmt.Errorf("build AIP tree: %w", err)
	}
	object, err := objectTree(ctx, filepath.Join(p.Workspace, "object"))
	if err != nil {
		return fmt.Errorf("build OCFL tree: %w", err)
	}
	if err := os.WriteFile(filepath.Join(p.Workspace, "am-aip-tree.txt"), []byte(source), 0o600); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.Workspace, "reports", "ocfl-tree.txt"), []byte(object), 0o600)
}
