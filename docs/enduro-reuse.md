<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Enduro source reuse

Reference project: [Enduro](https://github.com/artefactual-sdps/enduro), local source
inspected 2026-11-25.
The [inventory](enduro-reuse.json) records source SHA-256, source path,
destination and adaptations for copies. Original notices are preserved, with
REUSE annotations for sources lacking inline headers. No Git discovery is used.

Actual source metadata specifies Go 1.27.4, Goa 3.30.0, Ent 0.14.6 and Temporal
SDK 1.48.0. The new Goa generator matches its runtime, correcting the reference
tool file's 3.29.2 mismatch. Source-project guidance has older Go numbers;
the metadata controls this build. Dependency changes are limited to this project.

Reproduced generator exception: Bine's standalone Ent 0.14.6 binary uses
x/tools 0.30.0 and fails loading standard-library types. Running Ent with the
application graph's x/tools 0.49.0 still fails: Go 1.27.4 emits export data
version 5, while that loader supports at most 4. Module-pinned `go tool ent`
with x/tools 0.53.0 succeeds. Goa uses the same module-pinned tool approach.
Transitive x/mod, x/sync and x/sys versions follow that necessary loader update.
This is a tooling compatibility change, not a blanket dependency upgrade.

The [importer](https://opencommit.eu/provenant/provenant-import) is built as an
external executable. Its source fingerprint is in
[importer-source-sha256.json](importer-source-sha256.json), excluding Git data.
Application and importer have separate module graphs and use Go 1.27.4.

UI adaptations retain the generic Pager, ResultCounter, UUID, ThemeToggle and
Breadcrumb components (ASCII ellipsis and project theme storage key where needed).
Sidebar/header retain responsive Bootstrap offcanvas and collapse behavior, with
three routes and no auth. The upload component retains Uppy Dashboard/XHR and
reactive theme behavior, removes size/auth restrictions, and freezes the chosen
profile for retries. The Pinia store keeps paging and fetching behavior; request
sequence numbers prevent stale responses overwriting a newer page. Monitor events
invalidate database reads, with reconnect refresh and five-second active polling.
The UI uses an original SVG logo and a teal theme.
The header places breadcrumbs immediately after the brand, with the theme
control at the right. Home, upload, AIP details and reports use the shared content
width; the details grid removes Bootstrap's negative outer row margins.

Nginx/Vite proxy all API routes without request/response buffering or a body-size
ceiling. Runtime injection retains the distribution-copy approach but only accepts
a same-origin API path. Builds disable VCS discovery/stamping. Local Tilt extension
copies replace remote extension loading, preventing automatic repository cloning.
The new reset implements bucket deletion/recreation explicitly; the source reset
only recreated application databases. Temporal history remains outside reset.

The importer fixture provenance/adaptations are recorded separately in
[importer-reuse.json](importer-reuse.json). The archive bytes remain unchanged.
Shared Temporal activities are pinned external dependencies, not copied converter
code. Session affinity and report validation are Passage's responsibility.

Runtime images collect available LICENSE/NOTICE/COPYRIGHT files from their
independent module graphs; the UI distribution collects installed package notices.
Icon notices were retrieved 2026-11-25 from the original projects: [Clarity
license](https://github.com/vmware-archive/clarity-assets/blob/master/LICENSE),
[Clarity notice](https://github.com/vmware-archive/clarity-assets/blob/master/NOTICE.txt),
[Bootstrap Icons](https://github.com/twbs/icons/blob/main/LICENSE), and
[Akar Icons](https://github.com/artcoholic/akar-icons/blob/master/LICENSE).
Copies are in `ui/licenses/icons/`; versions/metadata come from the pinned Iconify
package lock. Project canonical license files remain byte-identical to Provenant.

A live worker-loss experiment exposed a distinction absent from the initial unit
simulation: a failed session's in-flight activity may return a heartbeat timeout.
The workflow now also checks SessionStateFailed before selecting whole-session
recovery. Both conversion and output-upload worker terminations subsequently
recovered with new workspaces, new worker queues and source redownloads.
