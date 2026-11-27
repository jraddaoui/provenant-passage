#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Create private OVH credentials outside the checkout, or apply them explicitly."""
import argparse
import base64
import getpass
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
from urllib.parse import urlparse

ROOT = Path(__file__).resolve().parents[1]
SPECS = {
    "mysql-secret": ["--from-env-file=mysql.env"],
    "seaweedfs-secret": ["--from-env-file=seaweedfs.env"],
    "provenant-runtime": ["--from-env-file=application.env"],
    "redis-secret": ["--from-env-file=redis.env"],
    "demo-basic-auth": ["--from-file=users=htpasswd"],
}
FILES = ("mysql.env", "seaweedfs.env", "application.env", "redis.env", "redis.conf", "htpasswd")


def private_directory(path):
    path = path.expanduser().resolve()
    if path == ROOT or ROOT in path.parents:
        raise ValueError("keep the credential directory outside the project checkout")
    return path


def initialize(directory, username):
    if not re.fullmatch(r"[A-Za-z0-9_.-]{1,64}", username):
        raise ValueError("username must contain only letters, digits, underscore, dot or hyphen")
    if directory.exists():
        raise ValueError("directory already exists; refusing to overwrite or regenerate credentials")
    password = getpass.getpass("Shared demo password (at least 16 characters): ")
    if (len(password) < 16 or len(password.encode()) > 72 or password != password.strip()
            or any(char in password for char in "\r\n\x00")):
        raise ValueError("use 16 or more characters, at most 72 bytes, without surrounding whitespace, newline or NUL")
    if password != getpass.getpass("Repeat password: "):
        raise ValueError("passwords do not match")
    # -i reads stdin; no password is placed in argv or shell history.
    hashed = subprocess.run(
        ["htpasswd", "-niB", "-C", "12", username], input=password + "\n",
        check=True, capture_output=True, text=True,
    ).stdout.strip() + "\n"
    mysql_password, root_password, redis_password, s3_key, s3_secret = (
        secrets.token_hex(32) for _ in range(5)
    )
    values = {
        "mysql.env": f"user=provenant\npassword={mysql_password}\nroot-password={root_password}\n",
        "seaweedfs.env": (
            f"access_key_id={s3_key}\nsecret_access_key={s3_secret}\n"
            f"admin_user={username}\nadmin_password={password}\n"
        ),
        "application.env": (
            f"PROVENANT_DATABASE_DSN=provenant:{mysql_password}@tcp(mysql.provenant:3306)/provenant\n"
            f"PROVENANT_EVENT_REDISADDRESS=redis://:{redis_password}@redis.provenant:6379\n"
        ),
        "redis.env": f"password={redis_password}\n",
        "redis.conf": f"bind 0.0.0.0\nprotected-mode yes\ndir /data\nappendonly yes\nrequirepass {redis_password}\n",
        "htpasswd": hashed,
    }
    directory.mkdir(mode=0o700, parents=True, exist_ok=False)
    for name, value in values.items():
        descriptor = os.open(directory / name, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w") as output:
            output.write(value)
    print(f"Created credentials in {directory}. Back up this directory securely.")


def apply(directory, context, expected_server):
    if not context or context.startswith(("k3d-", "kind-", "minikube", "docker-desktop")):
        raise ValueError("provide the explicit OVH context; local contexts are refused")
    parsed = urlparse(expected_server or "")
    if parsed.scheme != "https" or not parsed.hostname or parsed.hostname in ("localhost", "127.0.0.1", "::1"):
        raise ValueError("provide the expected remote HTTPS Kubernetes API URL with --api-server")
    for path in [directory, *(directory / name for name in FILES)]:
        if path.is_symlink() or stat.S_IMODE(path.stat().st_mode) & 0o077:
            raise ValueError(f"private path must not be a symlink or readable by group/others: {path}")
    kube = ["kubectl", "--context", context]
    configuration = json.loads(subprocess.run(
        kube + ["config", "view", "--minify", "-o", "json"],
        capture_output=True, text=True, check=True,
    ).stdout)
    if configuration["clusters"][0]["cluster"]["server"] != expected_server:
        raise ValueError("context API URL differs from --api-server; nothing applied")
    # Validate and render ALL credentials before the first mutation. Never print them.
    objects = []
    for name, sources in SPECS.items():
        result = subprocess.run(
            kube + ["-n", "provenant", "create", "secret", "generic", name, *sources,
                    "--dry-run=client", "-o", "json"],
            cwd=directory, capture_output=True, text=True, check=True,
        )
        obj = json.loads(result.stdout)
        if name == "redis-secret":
            # kubectl forbids combining --from-env-file with --from-file.
            obj["data"]["redis.conf"] = base64.b64encode(
                (directory / "redis.conf").read_bytes()
            ).decode("ascii")
        objects.append(obj)
    payload = json.dumps({"apiVersion": "v1", "kind": "List", "items": objects})
    result = subprocess.run(
        kube + ["-n", "provenant", "apply", "--server-side", "--field-manager=passage-secrets", "-f", "-"],
        input=payload, capture_output=True, text=True,
    )
    if result.returncode:
        # API errors can echo submitted data; do not print captured stderr/stdout.
        raise ValueError("Secret apply failed; inspect cluster status without exporting Secret contents")
    print("Applied five private Secrets in namespace provenant.")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("init", "apply"))
    parser.add_argument("directory", type=Path)
    parser.add_argument("--username", default="demo")
    parser.add_argument("--context")
    parser.add_argument("--api-server")
    args = parser.parse_args()
    try:
        directory = private_directory(args.directory)
        if args.action == "init":
            initialize(directory, args.username)
        else:
            apply(directory, args.context, args.api_server)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.error(str(error))


if __name__ == "__main__":
    main()
