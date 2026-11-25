#!/usr/bin/env bash
# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
# Restore pristine built assets on every start, then substitute the one public
# setting. Restrict it to a same-origin path so shell/JS quoting stays literal.
export VITE_API_PATH="${VITE_API_PATH:-/api}"
if [[ ! "$VITE_API_PATH" =~ ^/[a-zA-Z0-9_/-]+$ || "$VITE_API_PATH" == //* ]]; then
  echo 'VITE_API_PATH must be a same-origin absolute path.' >&2
  exit 1
fi
cp -a "$PROVENANT_UI_DIST/." "$PROVENANT_UI_ROOT/"
temp_dir="$(mktemp -d)"
trap 'rm -rf "$temp_dir"' EXIT
for file in "$PROVENANT_UI_ROOT"/assets/*.js; do
  envsubst '$VITE_API_PATH' < "$file" > "$temp_dir/asset.js"
  cat "$temp_dir/asset.js" > "$file"
done
