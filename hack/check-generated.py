#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Check deterministic generated outputs and module hygiene without Git."""
import hashlib
from pathlib import Path
import subprocess

roots = ["internal/enums", "internal/persistence/ent/db", "internal/api/gen",
         "internal/persistence/fake", "ui/src/openapi-generator",
         "internal/db/migrations/atlas.sum", "go.mod", "go.sum"]

def snapshot():
    result = {}
    for root in roots:
        path = Path(root)
        for file in ([path] if path.is_file() else path.rglob("*")):
            if file.is_file():
                result[str(file)] = hashlib.sha256(file.read_bytes()).hexdigest()
    return result

before = snapshot()
subprocess.run(["make", "gen-enums", "gen-ent", "gen-goa", "gen-ui-client", "gen-mock", "atlas-hash", "mod-tidy"], check=True)
after = snapshot()
changed = sorted(name for name in before.keys() | after.keys() if before.get(name) != after.get(name))
if changed:
    raise SystemExit("Generated/module files changed:\n" + "\n".join(changed))
print(f"Generation and module hygiene passed: {len(after)} unchanged files.")
