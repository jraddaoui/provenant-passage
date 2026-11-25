#!/usr/bin/env sh
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
# Run inside each independent module after downloading its dependencies.
set -eu
notices_dir="$1"
mkdir -p "$notices_dir"
go list -mod=readonly -m -f '{{.Path}}@{{.Version}} {{.Dir}}' all > "$notices_dir/modules.txt"
while read -r module module_dir; do
  test -n "$module_dir" || continue
  for file in "$module_dir"/LICENSE* "$module_dir"/LICENCE* "$module_dir"/COPYING* "$module_dir"/NOTICE* "$module_dir"/COPYRIGHT*; do
    test -f "$file" || continue
    mkdir -p "$notices_dir/$module"
    cp "$file" "$notices_dir/$module/"
  done
done < "$notices_dir/modules.txt"
cp /usr/local/go/LICENSE "$notices_dir/Go-LICENSE"
