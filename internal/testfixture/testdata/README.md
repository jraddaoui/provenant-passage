<!-- SPDX-FileCopyrightText: 2026 Justin Simpson / Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Synthetic .7z fixture: provenance

`synthetic-md5.7z` is copied unchanged from the
[provenant-import](https://opencommit.eu/provenant/provenant-import) test fixtures. It is
**synthetic**: every byte in it was invented for the tests. It is not derived from
any real preservation package and holds no third-party content.

## What it is

A BagIt bag shaped like an Archivematica AIP, in the md5-only variant (a
`manifest-md5.txt` and no sha256 manifest, as older AIPs have). Its content is
defined in Go, in [`../fixture.go`](../fixture.go) (`MD5Only()`): 8 payload files of
a few bytes each, including a METS with a `metsHdr CREATEDATE`, a path with a
space, two byte-identical files, and one file under `submissionDocumentation/`.

## Why the archive is included

Tests build directory and `.tar` forms in temporary directories. The `.7z`
form cannot be built that way: the library the external importer uses to read
`.7z` with (`github.com/bodgit/sevenzip`) is read-only, and the CI image has no
`7z` binary. One small copied file keeps the `.7z` code path tested, offline.

## How it was made, and how to remake it

The upstream [generation script](https://opencommit.eu/provenant/provenant-import/src/branch/main/scripts/gen-synthetic-7z.sh)
is run from the importer project only when intentionally updating that project.
Passage does not regenerate or modify its source. The script writes the
`MD5Only()` bag with `go run ./internal/aip/aiptest/gen7z`,
pins every mtime to 2026-01-02T03:04:05Z, and packs it with `7z a -t7z -mx=9`.

| | |
|---|---|
| Generated | 2026-09-20 |
| Size | 1239 bytes |
| sha256 | `75a44de1751da7bf688e4dd8a172a8599c4692376d5d325cdaacd43dd0ee4860` |
| Tool | 7-Zip 23.01 (x64), p7zip on Linux |

Two runs with that 7-Zip version gave identical bytes. Another version may not, and
that is fine: what the tests pin is the **content**, not the archive bytes.
The importer's `TestFixture7z_matchesGenerator` (package `aip`) reads the committed file and fails
if its files differ in any byte from what `MD5Only()` describes. Regenerate only
when that test says the two have drifted, then update the table above.
