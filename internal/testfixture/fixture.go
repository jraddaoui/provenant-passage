// SPDX-FileCopyrightText: 2026 Justin Simpson / Artefactual Systems Limited
// SPDX-License-Identifier: Apache-2.0

// Package testfixture builds a small synthetic Archivematica-shaped AIP for tests, so
// that `go test ./...` exercises the reader, the profiles, the import-event facts
// and the CLI on a bare checkout, with no staged package.
//
// The bag is a BagIt bag with the parts the importer reads: bagit.txt,
// bag-info.txt with an External-Identifier, a payload manifest and a tag manifest
// under one digest algorithm, data/METS.<uuid>.xml with a metsHdr CREATEDATE, a
// README.html, and a handful of payload files chosen to hit the importer's rules:
//
//   - a path containing a space (the manifest's two-space column split);
//   - two byte-identical files (one digest, two paths);
//   - one file under objects/submissionDocumentation/ (the reasoned drop);
//   - a CREATEDATE without a timezone, as Archivematica writes it.
//
// A directory and a .tar are written in-test. The .7z cannot be: the sevenzip
// library is read-only and the CI image has no 7z binary, so one KB-scale .7z of
// the md5-only variant is committed under testdata/, with the script that made it
// and a provenance note. TestFixture7z_matchesGenerator (package aip) fails if
// that file and MD5Only().Files() ever disagree in the source importer project.
//
// Everything in the bag is invented for the tests. It is not derived from any
// real package.
package testfixture

import (
	"archive/tar"
	"crypto/md5" // #nosec G501 -- synthetic legacy BagIt fixture, not a security primitive.
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// Identity and expected values of the synthetic bag. Tests assert against these
// so the numbers live next to the bag that produces them.
const (
	UUID         = "11111111-2222-4333-8444-555555555555"
	TransferUUID = "99999999-8888-4777-8666-555555555555"
	BagName      = "synthetic-" + UUID

	// CreateDate is the METS metsHdr CREATEDATE, without a timezone as
	// Archivematica writes it; CreateDateUTC is how the importer reads it.
	CreateDate    = "2026-01-02T03:04:05"
	CreateDateUTC = "2026-01-02T03:04:05Z"

	// PayloadEntries is the number of files under data/ (the payload manifest's
	// line count): the METS, the README, five payload files and the one file the
	// lean profiles drop.
	PayloadEntries = 8
	// FixityBridged is how many of those carry the bag's digest into the OCFL
	// fixity block: all but the relocated METS and README and the dropped file.
	FixityBridged = 5

	// Digests of objects/alpha.txt ("alpha\n"), which objects/nested/alpha-copy.txt
	// shares byte for byte.
	AlphaSHA256 = "b6a98d9ce9a2d9149288fa3df42d377c3e42737afdcdaf714e33c0a100b51060"
	AlphaMD5    = "9f9f90dbe3e5ee1218c86b8839db1995"
	AlphaSHA512 = "62d0791d22f871ef4b4e8f6fa1374091f6d540ba5e3e9bc23b0e6fd2e3d6534f9087b8c195634c7627fc26a33f17576b4e107da4ab421d486acc2636538bb58f"
)

// Bag describes a synthetic AIP. Start from Default or MD5Only and change fields
// to build a variant.
type Bag struct {
	Name        string            // top-level bag directory name
	UUID        string            // External-Identifier, and the METS file name
	BaggingDate string            // bag-info.txt Bagging-Date
	CreateDate  string            // METS metsHdr CREATEDATE; "" omits the attribute
	Algorithm   string            // manifest digest algorithm: "sha256" or "md5"
	Payload     map[string]string // path relative to data/ -> content (METS excluded)

	// BadDigestFor lists payload paths (relative to data/) whose payload-manifest
	// digest is deliberately wrong, to test the source self-integrity check.
	BadDigestFor []string
	// OmitMETS and OmitExternalIdentifier produce bags the reader must reject.
	OmitMETS               bool
	OmitExternalIdentifier bool
}

// Default is the standard synthetic bag, with sha256 manifests.
func Default() Bag {
	return Bag{
		Name:        BagName,
		UUID:        UUID,
		BaggingDate: "2026-01-02",
		CreateDate:  CreateDate,
		Algorithm:   "sha256",
		Payload: map[string]string{
			"README.html":                       "<html><body>Synthetic AIP for provenant-import tests.</body></html>\n",
			"logs/filenameCleanup.log":          "synthetic log line\n",
			"objects/alpha.txt":                 "alpha\n",
			"objects/metadata/identifiers.json": "{\"synthetic\":true}\n",
			"objects/nested/alpha-copy.txt":     "alpha\n",
			"objects/nested/beta file.txt":      "beta\n",
			"objects/submissionDocumentation/transfer-synthetic-" + TransferUUID + "/METS.xml": "<mets xmlns=\"http://www.loc.gov/METS/\"/>\n",
		},
	}
}

// MD5Only is the same bag with md5 manifests and no sha256 manifest: the shape of
// an older Archivematica AIP, and the content of the committed .7z fixture.
func MD5Only() Bag {
	b := Default()
	b.Algorithm = "md5"
	return b
}

func (b Bag) digest(content []byte) string {
	switch b.Algorithm {
	case "md5":
		s := md5.Sum(content) // #nosec G401 -- reproduces a synthetic legacy manifest.
		return hex.EncodeToString(s[:])
	case "sha256":
		s := sha256.Sum256(content)
		return hex.EncodeToString(s[:])
	default:
		panic("aiptest: unsupported manifest algorithm " + b.Algorithm)
	}
}

func (b Bag) mets() string {
	attr := ""
	if b.CreateDate != "" {
		attr = fmt.Sprintf(" CREATEDATE=%q", b.CreateDate)
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<mets xmlns=\"http://www.loc.gov/METS/\"><metsHdr" + attr + "/></mets>\n"
}

// manifest renders BagIt manifest lines ("<digest>  <path>") for files, sorted.
func (b Bag) manifest(files map[string][]byte, bad map[string]bool) []byte {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var sb strings.Builder
	for _, n := range names {
		d := b.digest(files[n])
		if bad[n] {
			d = strings.Repeat("0", len(d))
		}
		fmt.Fprintf(&sb, "%s  %s\n", d, n)
	}
	return []byte(sb.String())
}

// Files returns every file of the bag, keyed by its path including the top-level
// bag directory (the shape aip.Open's readers produce), with the manifests
// computed from the actual bytes.
func (b Bag) Files() map[string][]byte {
	payload := map[string][]byte{} // bag-relative, "data/..."
	for rel, content := range b.Payload {
		payload["data/"+rel] = []byte(content)
	}
	if !b.OmitMETS {
		payload["data/METS."+b.UUID+".xml"] = []byte(b.mets())
	}
	var oxum int
	for _, c := range payload {
		oxum += len(c)
	}
	bad := map[string]bool{}
	for _, rel := range b.BadDigestFor {
		bad["data/"+rel] = true
	}

	info := "Bagging-Date: " + b.BaggingDate + "\n"
	if !b.OmitExternalIdentifier {
		info += "External-Identifier: " + b.UUID + "\n"
	}
	info += fmt.Sprintf("Payload-Oxum: %d.%d\n", oxum, len(payload))

	tags := map[string][]byte{
		"bagit.txt":                        []byte("BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"),
		"bag-info.txt":                     []byte(info),
		"manifest-" + b.Algorithm + ".txt": b.manifest(payload, bad),
	}
	tags["tagmanifest-"+b.Algorithm+".txt"] = b.manifest(tags, nil)

	out := map[string][]byte{}
	for n, c := range payload {
		out[b.Name+"/"+n] = c
	}
	for n, c := range tags {
		out[b.Name+"/"+n] = c
	}
	return out
}

// WriteDir writes the bag as an extracted directory under dir and returns the
// path of the bag directory.
func (b Bag) WriteDir(t testing.TB, dir string) string {
	t.Helper()
	p, err := b.WriteDirTo(dir)
	if err != nil {
		t.Fatalf("aiptest: %v", err)
	}
	return p
}

// WriteDirTo is WriteDir without a test handle, for the fixture generator
// (gen7z), so the committed .7z and the in-test bags share one definition.
func (b Bag) WriteDirTo(dir string) (string, error) {
	for name, content := range b.Files() {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return "", fmt.Errorf("mkdir for %s: %w", name, err)
		}
		if err := os.WriteFile(p, content, 0o600); err != nil {
			return "", fmt.Errorf("write %s: %w", name, err)
		}
	}
	return filepath.Join(dir, b.Name), nil
}

// WriteTar packs the bag as <Name>.tar under dir and returns its path. Entries
// are written in sorted order with a fixed mtime, so the archive bytes (and the
// sha256 profile (a) binds v1 to) are the same on every run.
func (b Bag) WriteTar(t testing.TB, dir string) string {
	t.Helper()
	files := b.Files()
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)

	p := filepath.Join(dir, b.Name+".tar")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("aiptest: create tar: %v", err)
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	mtime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, n := range names {
		hdr := &tar.Header{Name: n, Mode: 0o644, Size: int64(len(files[n])), Typeflag: tar.TypeReg, ModTime: mtime}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("aiptest: tar header %s: %v", n, err)
		}
		if _, err := tw.Write(files[n]); err != nil {
			t.Fatalf("aiptest: tar body %s: %v", n, err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("aiptest: close tar: %v", err)
	}
	return p
}

// Fixture7z returns the path of the committed synthetic .7z (the MD5Only bag).
// It fails the test if the file is missing: the fixture is part of the repository,
// so its absence is a broken checkout, not a reason to skip.
func Fixture7z(t testing.TB) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("aiptest: cannot locate the package directory")
	}
	p := filepath.Join(filepath.Dir(here), "testdata", "synthetic-md5.7z")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("aiptest: committed fixture missing: %v", err)
	}
	return p
}
