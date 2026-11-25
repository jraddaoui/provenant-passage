#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Future human CI gate. Never execute during the Git-free workspace setup."""
import os
import re
import subprocess


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


head = os.environ.get("DCO_HEAD", "")
base = os.environ.get("DCO_BASE", "")
if not re.fullmatch(r"[0-9a-f]{40,64}", head):
    raise SystemExit("Missing valid head commit.")
if base and not re.fullmatch(r"[0-9a-f]{40,64}", base):
    raise SystemExit("Invalid base commit.")
revision = f"{base}..{head}" if base and set(base) != {"0"} else head
commits = git("rev-list", revision).splitlines()
for commit in commits:
    author = git("show", "-s", "--format=%an <%ae>", commit)
    trailers = git("show", "-s", "--format=%(trailers:key=Signed-off-by,valueonly)", commit).splitlines()
    if re.search(r"codex|openai|claude|anthropic|copilot|\[bot\]", author, re.I) or author.casefold() not in [line.strip().casefold() for line in trailers]:
        raise SystemExit(f"{commit}: missing human author's Signed-off-by.")
print(f"Human-author sign-off present on {len(commits)} commits. Identity authenticity still requires human review.")
