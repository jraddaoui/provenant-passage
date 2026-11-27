#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Fetch the pinned edge charts, verifying their recorded SHA-256 digests."""
import hashlib
from pathlib import Path
import subprocess

import yaml

ROOT = Path(__file__).resolve().parents[1]


def main():
    charts = yaml.safe_load((ROOT / "hack/kube/overlays/ovh/edge/charts.yaml").read_text())
    destination = ROOT / ".cache/ovh-charts"
    destination.mkdir(parents=True, exist_ok=True)
    for name, chart in charts.items():
        path = destination / f"{name}-{chart['version']}.tgz"
        if path.exists():
            data = path.read_bytes()
        else:
            data = subprocess.run(
                ["curl", "--fail", "--location", "--silent", "--show-error",
                 "--max-time", "60", chart["url"]],
                check=True, capture_output=True,
            ).stdout
        if hashlib.sha256(data).hexdigest() != chart["sha256"]:
            raise SystemExit(f"Chart digest mismatch: {name}; no installation performed")
        if not path.exists():
            path.write_bytes(data)
        print(path.relative_to(ROOT))


if __name__ == "__main__":
    main()
