#!/usr/bin/env bash
# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
# Disable Buildx's optional VCS metadata discovery as well as Go stamping.
export BUILDX_GIT_INFO=false BUILDX_GIT_LABELS=false
target="${1:-provenant-server}"
if [[ "$target" == provenant-ui ]]; then
  ui_target="${UI_BUILD_TARGET:-provenant-ui}"
  exec docker build --target "$ui_target" --tag "${EXPECTED_REF:-provenant-ui:${2:-dev}}" --file ui/Dockerfile ui
fi
if [[ "$target" != provenant-server ]]; then
  echo 'Accepted targets: provenant-server, provenant-ui' >&2
  exit 1
fi
importer_source="${IMPORTER_SOURCE:-.cache/provenant-import}"
test -f "$importer_source/go.mod"
image_ref="${EXPECTED_REF:-provenant-server:${2:-dev}}"
exec docker build --build-context "importer-source=$importer_source" \
  --build-arg GO_VERSION=1.27.4 --target provenant-server \
  --tag "$image_ref" --file Dockerfile .
