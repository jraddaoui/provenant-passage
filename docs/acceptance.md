<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Local acceptance evidence

The [acceptance criteria](acceptance-criteria.md) define the project contract.

Reproduced 2026-11-25 in the user-provided local cluster. These results concern
Passage, not the importer's earlier research experiments. No Git operations,
commits, remote CI execution or publication were performed. Source projects were
read-only. Build images went only to the existing local k3d registry.

## Inputs and verification policy

Verified context `k3d-provenant`, node `k3d-provenant-server-0`, namespace
`provenant`, and `local-path` storage. The global kubeconfig context was unchanged.
Go 1.27.4 builds the server and external importer with separate module graphs,
read-only importer dependencies, and VCS stamping disabled. Goa 3.30.0, Ent
0.14.6 and Temporal SDK 1.48.0 follow inspected Enduro metadata. The necessary
x/tools 0.53.0 generator compatibility change is explained in
[enduro-reuse.md](enduro-reuse.md).

The local and deployed importer executable SHA-256 is
`4d6e4d93ed30bc82b9252db3f20528c071aeda9e500599a9234d320ed351882e`.
Final fingerprints matched all 39 importer source files and 52 Enduro reference
entries. Canonical Apache/CC-BY/DCO files match Provenant byte for byte.
See `docs/evidence/local/final-source-check.json` and the source inventories.

## Browser and exact CLI comparison

All twelve production-Nginx browser upload/list/show/download cases passed:
three output choices each for generated synthetic TAR, copied synthetic 7z,
published CSVmultiLevel TAR and generated missing-METS-date TAR. Each record
retains source SHA-256/basename, application profile, CLI profile, UUID and fixed
whole-second UTC creation time. Actual Uppy uploads, UI downloads and live events
were exercised using browser-use 1.0 in its isolated Chromium profile.

For every case the downloaded object's entire relative-path set and file bytes
equal the unchanged CLI output at that same import time. No JSON normalization,
inventory exclusions or timestamp substitutions were used. Only the ZIP's outer
`object/` directory was removed from the comparison root. ZIP container bytes and
filesystem metadata are outside this comparison. Both sidecar reports also match
byte for byte, separately. Profile a retains the original archive bytes under
its original basename in v1 and the exact extracted view in v2.

`rocfl 1.7.0` validated all twelve objects with fixity enabled. Validator logs
retain unsuppressed warnings. Deliberate inventory/payload changes, a missing
file and an extra file all failed the comparison helper. This is exact equality
to the CLI plus a separate OCFL validity check; it does not establish truthful
pre-enrollment history, authentication, production readiness or customer value.

Evidence: `docs/evidence/local/browser/matrix.json`, `matrix-events.json`,
`downloads/`, and `docs/evidence/local/parity/results.json`, per-case files,
validator logs and `negative-controls.json`. All twelve downloads and reports
were retained locally before reset removed the cluster's disposable imports.

Light/dark persistence, Uppy theme updates, mobile navigation, exact second-page
row counts and creation-event reset to page one (including query state) passed.
Unknown-UUID and failed-import details show errors. Importer error/warning pages
provide no download. Screenshots and `paging-result.json` are in the browser
folder. Vite-mode upload/list/show/download, empty state and a blocked-request error
followed by Retry also passed (`vite-result.json`).

## Failure and recovery evidence

Backend tests passed for malformed/profile/filename/archive inputs, one-file
multipart enforcement, copy/close failure, request cancellation with detached
cleanup, DB-create/start failures and ambiguous acceptance. Download tests cover
unknown/non-completed records and absent output blobs. Workflow/activity tests
cover invalid/missing reports, failed equivalence, importer failure/warning,
ZIP truncation/CRC, each of the three output-upload failures, terminal DB failure,
activity retries, session routing and fresh-workspace/redownload recovery.

Live malformed/profile requests returned 400. An AIP without METS and a bad
checksum AIP both ended in `error`, with no output key and download HTTP 409.
The latter's retained report says `warning`; warning was not treated as success.
See `docs/evidence/local/failures/results.json` and retained reports.
A live missing-output check returned 404; the exact downloaded ZIP was restored
and downloaded bytes checked again (`missing-blob-result.json`).

Live worker termination during conversion initially exposed a bug: the failed
session's running activity returned a heartbeat timeout, which was incorrectly
classified as a terminal import failure. The workflow now also checks
SessionStateFailed. A regression test failed before that correction, then passed.
Conversion retest `d59b2e3b-8009-49f5-a014-177935103a78` and output-upload retest
`c8cfb897-b970-4b89-ac59-ddf64bafcb77` passed. Histories prove different worker
session queues, distinct workspaces and fresh source downloads, with unchanged
input identity/profile/time. The output test used a temporary delaying S3 proxy.
Histories, summaries and the initial failure are retained under
`docs/evidence/local/recovery/`.

Live Redis fan-out passed with a publisher and two separately constructed
monitor services, including disconnect cleanup. During Redis shutdown the UI
recovered completion by polling without receiving its completion event, then
reconnected after a separate browser network interruption. Evidence:
`docs/evidence/local/browser/redis-result.json`, `redis-missed-events.json`
and screenshots. UI tests also cover duplicate/out-of-order invalidations,
reconnect, stale page responses and active polling.

## Local engineering checks

Passed: backend build, formatter/linter, full race tests, vet and workflowcheck;
external importer package tests; isolated live MySQL migration/constraint/time/
pagination tests; live Redis fan-out; UI lint/format/TypeScript/build and 31 tests;
Kustomize/Tilt validation and local server/UI image builds. Generation and module
hygiene passed with 67 unchanged files. REUSE 6.2.0 passed using a source-only
staging copy and empty executable PATH, preventing automatic Git discovery.
REUSE covered 223 source files. Runtime images retain upstream, dependency and
icon notices.

The first MySQL test mistakenly used the application schema, allowing startup
reconciliation to see fixture rows. It now requires an admin DSN without a schema,
creates/drops an isolated temporary schema and passed again. Live reset removed
the original stray fixture data. This was test isolation, not production evidence.

GitHub Actions definitions are prepared but have not run remotely. A human must
supply the digest-pinned importer archive variables before enabling CI, review
origin/licensing and provide any DCO sign-off. This task claims no human sign-off.

## Deployment side effect

The first Tilt start used its default automatic Docker prune, removing 103
build-cache entries (4.731 GB), beyond project-only resource scope. The user was
informed. `docker_prune_settings(disable=True)` now prevents automatic pruning.
Cached layers may need rebuilding; application data was not pruned.

## Final deployment handoff

Two live resets passed. The first stopped an active import, confirmed its
Temporal termination, emptied the application database and both buckets, reran
migrations and restarted the app. After 65 seconds no old import had repopulated
storage. The second reset repeated successfully. Both retained earlier completed
Temporal histories and the separate temporal/temporal_visibility databases.
See `docs/evidence/local/reset/results.json` and both reset Job logs.

The installed kubectl 1.24 client intermittently timed out forwarding streams
against this 1.31 cluster. Checks inside the cluster remained healthy. Reset
verification used direct in-cluster API reads after one tunnel timeout; the Vite
browser round trip used a temporary NodePort on the local Docker node address.
That service was removed. The delayed importer override/mount/ConfigMap and
upload-delay proxy were also removed. Redis has one healthy replica.

The final default Nginx deployment is healthy. Its configuration and shipped
notices passed checks. Tilt remains running on localhost:10355 with UI :8080,
API :9000 and MySQL :23306. One completed synthetic example remains after reset. Final production-browser
navigation passed; screenshots are `browser/final-home.png` and
`browser/final-detail.png`. The implementation goal is complete; future work
requires human review and a new scope.
All test archives, histories, downloads and comparison outputs remain local under
`docs/evidence/local/` (excluded from images and future version control).

## Documentation and source-location follow-up

On 2026-11-25, project guidance and acceptance criteria were made self-contained,
with public Provenant links replacing sibling-project and research references.
`AGENTS.md` is the sole agent entry point. Development conventions are maintained
locally, adapted for Passage's human-controlled Git and GitHub workflow, with
upstream attribution retained.
Make, Docker, Tilt and CI now default to ignored `.cache/provenant-import` source.
An ignored local Tilt override selects the previously used importer directory.

Focused checks passed: cached archive extraction with a mocked HTTPS response,
digest mismatch/existing-directory rejection, Make dry runs, Docker invocation
with a stub executable for both default and explicit paths (including spaces),
matching Tilt/CI defaults, and all 32 local links across eight Markdown files.
REUSE 6.2.0 passed for 224 source files. No application build, browser acceptance
or remote CI was rerun for this documentation/configuration change.
