#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Lint a source-only staging copy, with all VCS executables unavailable."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

checker = Path(sys.argv[1] if len(sys.argv) > 1 else shutil.which("reuse") or "reuse").absolute()
if not checker.is_file():
    raise SystemExit("Install reuse==6.2.0, then pass its executable path.")
# These are build/dependency/local-evidence outputs, never distributed source.
skip = {".git", ".cache", "bin", "ui/node_modules", "ui/dist", "ui/coverage", "coverage", "covreport", "docs/evidence/local"}
private = {"provenant.local.toml", ".tilt.env", "ui/.env.local"}
with tempfile.TemporaryDirectory(prefix="provenant-reuse-") as tmp:
    root = Path(tmp) / "source"
    for parent, dirs, names in os.walk("."):
        rel = Path(parent)
        dirs[:] = [d for d in dirs if (rel / d).as_posix() not in skip]
        for name in names:
            path = rel / name
            if path.as_posix() in private or name.endswith(".tsbuildinfo"):
                continue
            target = root / path
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path, target)
    # REUSE discovers VCS binaries with shutil.which at import time. An empty
    # PATH and explicit root select its standard VCSStrategyNone, without Git.
    empty_path = Path(tmp) / "no-vcs"
    empty_path.mkdir()
    env = dict(os.environ, PATH=str(empty_path))
    result = subprocess.run([str(checker), "--root", str(root), "--no-multiprocessing", "lint"], env=env)
    raise SystemExit(result.returncode)
