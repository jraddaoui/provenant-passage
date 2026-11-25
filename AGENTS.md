<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Provenant Passage

A local import proof of concept and home for Provenant workflows. Read
[acceptance criteria](docs/acceptance-criteria.md) for the project contract and
[local evidence](docs/acceptance.md) for reproduced results and remaining work.

## Boundaries and layout

- No Git operations, commits, pushes, publication, issues or PRs during setup.
  Human review, DCO sign-off and publication remain human responsibilities.
- `cmd/provenant-server`: lifecycle; `internal/aips`: API/domain;
  `internal/api/design`: Goa source; `internal/persistence/ent/schema`: schema;
  `internal/workflow`: Temporal orchestration; `ui/`: Vue application.
- Reference [Enduro](https://github.com/artefactual-sdps/enduro), copying unchanged
  where appropriate.
  Keep sources read-only and record source hashes/adaptations in `docs/enduro-reuse.md`.
- External [importer](https://opencommit.eu/provenant/provenant-import) source
  defaults to `.cache/provenant-import`; see [setup](README.md#run-locally).
  `IMPORTER_SOURCE` may select another source directory. Never import its
  internal packages, change it, or bridge its module with replace/go.work.
  Build with `-mod=readonly -buildvcs=false`; preserve original archive basename,
  bytes, profile and whole-second UTC import time. Exact CLI parity is developer
  acceptance, never an application feature.

## Development

Follow this project's [development conventions](development-conventions.md).
Write focused failing behavior tests before implementing production code changes.
Run local checks; future human-triggered CI is a separate gate.

- Generate enums, Ent, Goa, UI client and mocks via `make gen-enums`, `gen-ent`,
  `gen-goa`, `gen-ui-client`, `gen-mock`. Never edit generated outputs manually.
- MySQL only. Embed versioned SQL migrations and run `make atlas-hash` on changes.
- Run `make fmt lint test test-race workflowcheck`; use `make tools`/`go tool bine`.
  Disable VCS stamping for builds, tests and generation. Compare hashes for
  generation/module hygiene, without Git.
- Temporal workflows must be deterministic and activities idempotent. All local
  workspace activities belong to one session; session loss means new scratch and
  redownload. Never route an old local path to a new worker. Terminal writes are
  bounded and cancellation-safe. Redis carries transient UI events, not tasks.
- Use only verified context `k3d-provenant`, namespace `provenant`, named project
  resources and dedicated Temporal namespace. Do not change global context or
  create a cluster. Reset is explicit/manual and leaves Temporal DBs intact.
- Run UI commands from `ui/`: `npm run check`, `npm run dev`. Verify actual browser
  upload/list/details/download, themes, event reconnection and mobile layout.
- Code/config/build files are Apache-2.0; prose CC-BY-4.0. Preserve upstream
  notices and REUSE annotations. Disclose OpenAI Codex assistance accurately;
  never invent human sign-off. Run `reuse lint`.
- This is a prototype. Conversion/parity do not prove historical truth,
  authentication, signing, production readiness or institutional value.
- Keep project requirements, evidence and handoff in this repository's `docs/`.
  Use local links within this project and public links to the
  [Provenant projects](https://opencommit.eu/provenant) for external references.
  Do not link to sibling repositories or an external research workspace.

## Commit, PR, And Security
- Assistants must not create commits or submit pull requests; the human developer
  owns committing and PR submission. Pushes and publication are human-only.
- When asked for a commit message, follow Chris Beams' seven rules:
  1. Separate subject from body with a blank line.
  2. Limit the subject line to 50 characters.
  3. Capitalize the subject line.
  4. Do not end the subject line with a period.
  5. Use the imperative mood in the subject line.
  6. Wrap the body at 72 characters.
  7. Use the body to explain what and why, not how.
- For commit-message requests, inspect the staged changes and consider session
  context only as it explains the final state. Do not include intermediate
  back-and-forth or discarded approaches from the session.
- Link an issue in PR descriptions.
- Include UI screenshots for visible UI changes.
- Commit generated artifacts only when the source change requires regeneration.
- Report vulnerabilities through this project's `SECURITY.md` policy when present.
  Do not copy upstream reporting contacts or add a broken policy link.
- Keep secrets out of repo files and commits, including `provenant.local.toml`, credentials in `provenant.toml`, Kubernetes Secrets, and environment files. Checked-in examples contain development placeholders only.
- Update the project threat model when architecture or security boundaries change,
  if one is maintained. Do not add a threat-model implementation solely for this rule.

The staged-inspection rule applies only to later commit-message requests after
Git inspection is permitted. It does not permit Git during setup. Assistants
prepare generated outputs; humans commit them.
