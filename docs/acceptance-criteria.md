<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Import workflow acceptance criteria

Scope established 2026-11-25: upload an Archivematica AIP, convert it with the
independently maintained [provenant-import](https://opencommit.eu/provenant/provenant-import)
CLI, monitor progress and download the OCFL object. The [README](../README.md)
documents operation and architecture; [local evidence](acceptance.md) records
reproduced results and gaps separately from these requirements.

## Application contract

- Accept one `.tar` or `.7z` AIP through the browser. Stream it to `am-aips`,
  preserve its bytes and safe original basename, and retain it after conversion.
- Use UUIDs for import requests without replacing the source BagIt identifier.
  Persist one whole-second UTC creation time and reuse it for every retry.
- Keep status to queued, processing, completed, error and canceled. Completion
  requires clean importer reports, a validated ZIP and all three output uploads.
  Failed or canceled imports offer no download.
- Keep all activities using local files in one Temporal session. Session loss
  requires a new workspace and source redownload. Redis carries transient UI
  notifications; reconnect refresh and polling recover missed events.
- Download a ZIP containing one `object/` directory with the unchanged OCFL
  object. Store `import-event.json` and `import-report.md` separately in the
  output bucket.

| Application profile | CLI profile |
| --- | --- |
| `files_and_metadata` (default) | `b` |
| `files_metadata_and_manifests` | `c` |
| `original_and_extracted` | `a` |

Keep the CLI and its report schema unchanged. Build it separately with its own
module graph, `-mod=readonly` and `-buildvcs=false`. Do not modify upstream source,
import its internal packages or bridge modules with `replace` or `go.work`.

## Exact CLI comparison

These are developer acceptance checks. The application does not run a comparison
or expose comparison controls.

For each profile, exercise browser upload, list, details and download with a
generated synthetic TAR, the copied synthetic 7z, the published CSVmultiLevel AIP
from [provenant-testdata](https://opencommit.eu/provenant/provenant-testdata), and
a fixture with a missing or unparseable METS creation date. Record missing
fixtures as gaps.

1. Build local and deployed importers from the same source state, dependencies
   and Go toolchain. Record source fingerprints and executable hashes.
2. Record the input SHA-256, original basename, application/CLI profiles, request
   UUID and exact persisted creation time.
3. Download through the browser and safely extract the object. Run the local
   CLI with the identical input bytes, basename, profile and creation time,
   using fresh output and report directories:

   ```sh
   ./bin/provenant-import --profile b \
     --import-time '<created_at from this upload>' \
     --reports /tmp/provenant-parity/local-reports \
     '/path/to/the/same-original-name.tar' /tmp/provenant-parity/local-object
   ```

4. Compare every relative path and file byte, including inventories, sidecars,
   declarations and all versions. Remove only the ZIP's outer `object/` directory
   from the comparison root. No JSON normalization or timestamp exclusions are
   allowed. ZIP container bytes and filesystem metadata are outside the policy.
5. Compare both reports separately. For profile `a`, verify the original archive
   bytes/basename in v1 and the exact extracted view in v2. The missing-date case
   exercises import-time fallback for profiles `b` and `c`.
6. Confirm the comparison detects a changed inventory, changed payload, missing
   file and extra file. Run independent OCFL validation with fixity enabled and
   record validator version/results, or explicitly record its unavailability.

Equality to CLI output does not itself establish OCFL validity. Neither check
establishes truthful pre-enrollment history, authentication or institutional
value. Trust inputs and verification policy remain external.

## Failure, recovery and operational checks

- Cover invalid inputs, upload copy/close/cancellation failures, database and
  workflow-start failures, and ambiguous acceptance without destructive cleanup
  of an import that may have started.
- Cover malformed or unclean importer reports, invalid ZIPs, each output-upload
  failure, bounded terminal writes, unknown requests and missing output blobs.
- Terminate workers during conversion and output upload. Verify new session
  queues/workspaces, source redownload and unchanged input identity/profile/time.
- Exercise Redis fan-out across instances, outage recovery, reconnect, duplicate
  and out-of-order events, and polling when completion notifications are missed.
- Check pagination, light/dark persistence, mobile navigation, empty/error states
  and browser upload/download in both Nginx and Vite modes.
- Verify migrations and repeated explicit reset in the intended local cluster.
  Reset stops active imports and clears only application data and its two
  buckets; Temporal databases/history remain and old workflows cannot repopulate
  cleared storage. Normal startup preserves data.
- Run applicable backend, UI, generation, build, deployment and REUSE checks.
  Record local checks separately from remote CI, which requires human setup.

Keep a concise result/limitation record in this project's `docs/` and bulky local
evidence under ignored `docs/evidence/local/`. Missing required checks remain
gaps. This is a local proof of concept without authentication or an upload-size
ceiling; acceptance does not claim production readiness. Human review, DCO
sign-off, commits and publication remain human responsibilities.
