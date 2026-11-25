<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Local acceptance evidence

The [acceptance criteria](acceptance-criteria.md) define the project contract.

Reproduced 2026-11-25 in the user-provided local cluster. These results concern
Passage, not the importer's earlier research experiments. No commits,
remote CI execution or publication were performed. Source projects were
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

## Report viewing

Added 2026-11-25 with OpenAI Codex assistance: completed AIP details include
collapsed **Import event record** and **Content preservation report**
sections. A short paragraph before the JSON explains the source checksum,
import time/outcome, content checks, and omitted or unaccounted content.
`GET /aips/{uuid}/reports/{format}` accepts
only `json` or `md`, streams the fixed output-bucket report key, and closes the
reader. It returns 404 for missing AIPs/blobs and 409 for incomplete imports.
It uses the same access boundary as the existing AIP API; authentication remains
outside this local prototype. The browser loads each report on first expansion,
retains it for reopening, and cancels pending requests when leaving the AIP.

Markdown uses Marked 18.3.0 followed by DOMPurify 3.5.0 sanitization; JSON is
literal preformatted text. This follows [Marked's sanitization guidance](https://marked.js.org/#usage)
and [DOMPurify's HTML profile](https://github.com/cure53/DOMPurify#can-i-configure-dompurify).
The importer, stored report bytes and report schema are unchanged.

Reproduced locally: both formats through the HTTP transport, invalid formats
and UUIDs, incomplete imports, missing objects, storage failures, exact bytes,
headers and reader closure. All 35 UI tests pass, including lazy loading, cached
reopening, rendered headings/tables, unsafe HTML removal, missing-report retry
and request cancellation. Formatter/linter, backend tests/race tests,
workflowcheck, UI type checks/build, generation/module hygiene (67 unchanged
files) and REUSE (228 source files) passed. Both local Tilt images rebuilt.

The existing completed CSVmultiLevel AIP
`00f5c57b-553d-46f0-a8e7-53f2d80d5bcd` passed live report viewing: no report request
before expansion, one same-origin API request per format, JSON byte equality to
the API response, rendered Markdown tables, light/dark themes and 390px layout
without page overflow. Evidence and screenshots are
`docs/evidence/local/browser/reports-result.json` and `reports-*.png` beside it.
The first theme check assumed the initial theme incorrectly; the browser script
now explicitly selects a theme before assertions. Full import/CLI parity was
not rerun for this presentation-only change; remote CI remains unrun.

Layout follow-up: home, upload, AIP details and reports now fill the shared
content width. Breadcrumbs sit immediately after the brand; removing the detail
grid's negative row margins aligns its border with the surrounding content.
`npm run check` and REUSE passed. Browser measurements passed for all four pages
at 1440px, 1024px and 390px without horizontal page overflow. Screenshots and
measurements are `docs/evidence/local/browser/layout-*.png` and
`layout-result.json`. Browser checks disable cache to avoid reusing pre-rebuild
page assets. The local UI image was rebuilt; backend behavior was unchanged.

The header breadcrumb uses `ms-3` (16px), and the title link uses
`text-decoration-none`. UI checks and REUSE passed; the local UI rebuilt.
Browser checks confirmed the spacing and no title underline on hover or focus.
Evidence: `docs/evidence/local/browser/header-spacing-result.json` and
`header-title-hover.png` beside it.

Report naming follow-up: the section titles describe their contents, and the
JSON introduction appears inside the expanded section before the literal text.
`npm run check` and REUSE passed; the local UI image rebuilt. Browser checks
confirmed both labels, initial collapse, the explanation placement, rendered
content report, and 1440px/390px layouts without page overflow. Evidence is
`docs/evidence/local/browser/report-labels-result.json` and
`report-labels-1440.png` / `report-labels-390.png` beside it.

Page-height fix: the navbar's 64px minimum plus the header's 1px border made
short desktop/tablet pages one pixel taller than the viewport, because the
sidebar reserved only 64px. The navbar now accounts for that border, keeping the
whole header at 64px. Browser regression checks compare document and viewport
heights for short home/details pages at 1440px, plus home at 820px and 390px.
All now fit without main-page overflow; expanded structures still scroll and
the header/sidebar remain aligned. UI checks and REUSE passed; the local UI
rebuilt. Evidence is `docs/evidence/local/browser/page-height-before.json`,
`page-height-result.json`, `page-height-long-content.json`, and the
`page-height-*.png` screenshots beside them. Tree/report scrolling was unchanged.

## Package structure viewing

Added 2026-11-25 with OpenAI Codex assistance: the session generates
`am-aip-tree.txt` from TAR/7z archive entries and `ocfl-tree.txt` from the
complete generated object. The source tree is stored in
`am-aips/aips/{uuid}/`; the OCFL tree is stored in
`ocfl-objects/aips/{uuid}/` beside the reports. Neither is inserted into the
archive or OCFL object. Trees sort directories first, include implicit/empty
folders and end with counts excluding the root. Control characters and
backslashes in names are quoted; conflicting paths and unsupported special
entries fail generation instead of silently omitting entries.

`GET /aips/{uuid}/structures/{kind}` streams plain text from the fixed source
(`aip`) or output (`ocfl`) key. Completed details have an initially collapsed
**Package structures** section above the reports, with equal desktop columns
and independently scrollable `pre` elements. Medium and smaller screens (below
992px) stack the columns, as requested in the responsive layout follow-up.
Loading is lazy, reopening is cached, failures can be retried independently,
and leaving the record cancels pending requests. Missing older sidecars return
404; incomplete imports return 409. A Temporal version marker preserves the
old command sequence for histories that already passed the new generation
point. There is no backfill; re-upload creates tree sidecars for an older AIP.

Local checks passed: formatting/lint, backend tests and race tests, vet,
workflowcheck, UI lint/format/type checks/tests/build, generated/module hygiene
(67 unchanged files), and REUSE (235 files). Tests cover both archive formats
and all three profiles, exact trees/counts, empty/implicit folders, escaped
names, conflicting paths, cancellation, both bucket routes and HTTP errors,
new generation/upload failures, session retry and the legacy workflow branch.
Both local Tilt images rebuilt successfully.

Fresh browser upload `977f6bac-29a5-4b86-a329-2a4c9a343908` used the published
CSVmultiLevel TAR with `files_and_metadata` (CLI profile `b`) and fixed import
time `2026-11-25T19:28:50Z`. The AIP tree has 12 directories and 23 files; the
OCFL tree has 13 directories and 23 files. API text equals both displayed text
and stored bucket bytes. Independent Python listings of the TAR and downloaded
ZIP match the trees exactly. All 23 object files and both original reports
match a direct run of the unchanged local importer. This is one fresh parity
case, not a rerun of the twelve-case matrix below.

Browser checks passed initial collapse, section order, lazy loading, cached
reopening, full content width and equal columns at 1440/1024px, 390px stacking
without page overflow, both themes, existing reports and ZIP download.
Evidence: `docs/evidence/local/structures/result.json`,
`docs/evidence/local/browser/structures-result.json`, and `structures-*.png`
beside the browser record. Remote CI and human review remain pending.

The structure stacking breakpoint was extended from 768px to 992px at the
user's request. UI checks passed and the local UI rebuilt. Browser measurements
confirm two columns at 1200/992px and two rows at 991/820/768/390px, without
horizontal page overflow. Current evidence and screenshots are
`docs/evidence/local/browser/structures-responsive-result.json` and
`structures-responsive-992.png`, `structures-responsive-820.png`, and
`structures-responsive-390.png` beside it.

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

The initial local acceptance did not run GitHub Actions. The later reported CI
failure and local provisioning fix are recorded below. A human still reviews
origin/licensing and provides any DCO sign-off. This task claims no human sign-off.

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

## CI importer provisioning

On 2026-11-25, the user reported that GitHub Actions failed at importer
provisioning because both repository variables were empty. The build workflow
now carries the archive URL and SHA-256 directly, removing that manual setup
dependency. Local setup instructions use the same values. This supersedes the
initial requirement to configure repository variables before enabling CI.

The public forge's web archive URL returned HTTP 404; its API archive endpoint
worked for revision `21d9649813ac18031eee965e98aa7a54cc7be348`. The downloaded
archive SHA-256 is
`d2428d9cfd3cdfa7107461309c8884118dd69e3f47c120a4f31007fb63cd505c`.
All 39 files match `importer-source-sha256.json`, so this changes provisioning
without changing the previously tested importer source.

Reproduced locally: a real HTTPS download through `hack/provision-importer.py`
with the workflow's pinned values, exact extracted-source inventory equality,
digest-mismatch rejection using the cached archive, existing-directory
protection, workflow YAML parsing and matching README values. REUSE 6.2.0 passed
for 228 source files. Evidence: `docs/evidence/local/ci-importer/result.json`.
No application code changed; backend/UI suites and browser acceptance were not
repeated. The corrected workflow still needs a human commit/push and remote CI
run; no remote pass is claimed.
