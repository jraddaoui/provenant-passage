#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Local and CI importer provisioning from a digest-pinned source archive."""
import hashlib
import os
from pathlib import Path
import re
import sys
import tarfile
import tempfile
import urllib.request

url = os.environ.get("IMPORTER_ARCHIVE_URL", "")
digest = os.environ.get("IMPORTER_ARCHIVE_SHA256", "")
if not url.startswith("https://") or not re.fullmatch(r"[0-9a-f]{64}", digest):
    raise SystemExit("Set IMPORTER_ARCHIVE_URL (HTTPS) and IMPORTER_ARCHIVE_SHA256 environment variables to the selected importer source archive.")
destination = Path(sys.argv[1])
if destination.exists():
    raise SystemExit("Importer destination already exists; refusing to replace it.")
with tempfile.TemporaryDirectory(prefix="provenant-importer-") as tmp:
    archive = Path(tmp) / "importer.tar.gz"
    hashed = hashlib.sha256()
    with urllib.request.urlopen(url, timeout=60) as response, archive.open("wb") as output:
        while chunk := response.read(1024 * 1024):
            output.write(chunk)
            hashed.update(chunk)
    if hashed.hexdigest() != digest:
        raise SystemExit("Importer source archive digest mismatch.")
    unpacked = Path(tmp) / "unpacked"
    with tarfile.open(archive) as source:
        source.extractall(unpacked, filter="data")
    roots = list(unpacked.iterdir())
    if len(roots) != 1 or not (roots[0] / "go.mod").is_file():
        raise SystemExit("Archive must contain one top-level importer directory.")
    import shutil
    shutil.copytree(roots[0], destination)
print("Provisioned the digest-pinned external importer.")
