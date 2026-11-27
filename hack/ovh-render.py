#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Render public OVH manifests by phase. Never contacts a Kubernetes cluster."""
import argparse
import ipaddress
from pathlib import Path
import re
import subprocess
import sys

import yaml

ROOT = Path(__file__).resolve().parents[1]
OVERLAY = ROOT / "hack/kube/overlays/ovh"
PHASES = ("foundation", "schema", "temporal", "namespace", "app", "tls", "routes")


def phase_of(obj):
    kind, name = obj["kind"], obj["metadata"]["name"]
    if kind == "Job":
        return "schema" if name.startswith("temporal-schema-") else "namespace"
    if kind == "Deployment":
        if name.startswith("temporal-"):
            return "temporal"
        if name in ("provenant-server", "provenant-ui"):
            return "app"
    return "foundation"


def image(settings, key):
    value = settings.get(key, "")
    if not isinstance(value, str) or not re.fullmatch(
        r"ghcr\.io/[a-z0-9._/-]+@sha256:[a-f0-9]{64}", value
    ):
        raise ValueError(f"{key} must be a published public GHCR image pinned by sha256")
    return value


def render(phase, settings):
    if phase == "tls":
        address = ipaddress.ip_address(settings.get("publicIP", ""))
        if address.version != 4 or not address.is_global:
            raise ValueError("publicIP must be the public IPv4 address of the edge LoadBalancer")
        email = settings.get("acmeEmail", "")
        if not isinstance(email, str) or not re.fullmatch(r"[^\s@]+@[^\s@]+\.[^\s@]+", email):
            raise ValueError("acmeEmail must be a contact email address")
        server = settings.get("acmeServer", "")
        if server not in (
            "https://acme-staging-v02.api.letsencrypt.org/directory",
            "https://acme-v02.api.letsencrypt.org/directory",
        ):
            raise ValueError("acmeServer must be the documented staging or production endpoint")
        replacements = {
            "REPLACE_PUBLIC_IP": str(address),
            "REPLACE_ACME_EMAIL": email,
            "REPLACE_ACME_SERVER": server,
            "REPLACE_ISSUER_NAME": "letsencrypt-ip-staging" if "staging" in server else "letsencrypt-ip",
            "REPLACE_ACCOUNT_SECRET": "letsencrypt-ip-staging-account" if "staging" in server else "letsencrypt-ip-account",
        }

        def replace(value):
            if isinstance(value, dict):
                return {key: replace(item) for key, item in value.items()}
            if isinstance(value, list):
                return [replace(item) for item in value]
            return replacements.get(value, value) if isinstance(value, str) else value

        return [replace(obj) for obj in yaml.safe_load_all((OVERLAY / "edge/tls.yaml").read_text())]
    if phase == "routes":
        return list(yaml.safe_load_all((OVERLAY / "edge/routes.yaml").read_text()))
    rendered = subprocess.run(
        ["kubectl", "kustomize", str(OVERLAY)], check=True, capture_output=True, text=True
    ).stdout
    objects = [obj for obj in yaml.safe_load_all(rendered) if obj and phase_of(obj) == phase]
    if phase == "app":
        images = {
            "provenant-server": image(settings, "serverImage"),
            "provenant-ui": image(settings, "uiImage"),
        }
        for obj in objects:
            name = obj["metadata"]["name"]
            for container in obj["spec"]["template"]["spec"]["containers"]:
                container["image"] = images[name]
    if any(obj["kind"] == "Secret" for obj in objects):
        raise ValueError("public OVH render must not contain Secret objects")
    return objects


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("phase", choices=PHASES)
    parser.add_argument("--settings", type=Path, default=OVERLAY / "settings.yaml")
    args = parser.parse_args()
    settings = yaml.safe_load(args.settings.read_text())
    if not isinstance(settings, dict):
        parser.error("settings must be a YAML mapping")
    try:
        objects = render(args.phase, settings)
    except (ValueError, subprocess.CalledProcessError) as error:
        parser.error(str(error))
    print("# Generated from versioned public configuration; contains no secret values.")
    yaml.safe_dump_all(objects, sys.stdout, sort_keys=False)


if __name__ == "__main__":
    main()
