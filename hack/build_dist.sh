#!/usr/bin/env bash
# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
read -r release < internal/version/VERSION.txt
version="${BUILD_VERSION:-${release}-local}"
if [[ ! "$version" =~ ^[A-Za-z0-9.+_-]+$ ]]; then
  echo 'BUILD_VERSION must be an explicit version label.' >&2
  exit 1
fi
exec go build -mod=readonly -buildvcs=false -trimpath \
  -ldflags="-s -w -X github.com/jraddaoui/provenant-passage/internal/version.Long=${version}" "$@"
