<!-- SPDX-FileCopyrightText: 2026 Justin Simpson / Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Provenant Passage development conventions

Adapted 2026-11-25 from Provenant's development conventions. The source copy
identifies private documentation revision
`d3f4d60b7615fb87e944df21a9947a2980845a50`, copied on 2026-09-22 under CC-BY-4.0.
This document is maintained locally for Passage. Its human-only Git rules,
GitHub workflow and commit-message format replace the upstream equivalents.

These conventions apply to human and agent development in this repository.
[AGENTS.md](AGENTS.md) is the agent entry point;
[CONTRIBUTING.md](CONTRIBUTING.md) defines the current contribution policy.

## 1. Test-first development and validation

- For production code changes, write a focused behavior test that fails for the
  right reason, implement the change, then refactor with the test passing.
- Run the applicable backend, UI, generation and licensing checks documented in
  the [README](README.md#development-checks). Use focused checks for documentation
  and configuration changes; a document-only edit needs no build or demo run.
- Keep local results and remote CI results distinct. The GitHub Actions
  definitions in [.github/workflows](.github/workflows) are prepared locally;
  they have not yet run remotely. A local pass does not satisfy the CI gate.
- A human configures the digest-pinned importer variables before enabling CI.
  Once enabled, the applicable checks must pass before a human merges a change.
  Do not publish or create commits merely to trigger CI.
- Record unavailable required checks as gaps. Use the project's
  [acceptance criteria](docs/acceptance-criteria.md) for integration changes and
  retain reproduced results in [local evidence](docs/acceptance.md).

## 2. Track work in this project

- The selected repository home is
  [jraddaoui/provenant-passage](https://github.com/jraddaoui/provenant-passage).
  Use GitHub issues and milestones for non-trivial work and release planning
  when the human maintainer establishes that workflow. Do not require access to
  an upstream private forge, roadmap or missing decision records.
- During local preparation, record requirements, decisions, evidence and
  remaining work in this project's `docs/`. Keep the record concise and current.
- Future human PRs link the relevant issue and include screenshots for visible
  UI changes. Describe the final behavior and validation, not session history.
- Agents prepare local changes and draft issue/PR text when requested. The
  current setup forbids Git operations, issues, PR submission and publication.
  Do not post messages or create external records without explicit authorization.

## 3. Licensing and source attribution

- Code, configuration and build files are Apache-2.0; prose is CC-BY-4.0. Keep
  the canonical license texts in [LICENSES](LICENSES).
- Add SPDX copyright and license headers at file creation. Use
  [REUSE.toml](REUSE.toml) for files that cannot carry headers. Preserve existing
  valid notices and the licenses of reused material.
- Run the REUSE checker documented in the README. The local source-only checker
  avoids Git discovery; licensing must also pass in CI when enabled.
- Record copied source paths, hashes and adaptations in
  [source reuse](docs/enduro-reuse.md). Leave upstream projects unchanged and
  retain required dependency notices in runtime images.

## 4. Human-controlled Git, DCO and AI assistance

- All commits, amendments, pushes, PR submissions and publication belong to the
  human developer, including local/WIP commits. Agents never create commits or
  sign on a human's behalf. The current setup also prohibits Git inspection;
  a later request must authorize it before staged-change review.
- Every commit reaching `main` requires a real human `Signed-off-by` trailer
  matching its author, asserting the [Developer Certificate of Origin](DCO).
  Use the actual human contributor's identity. Do not inherit an upstream
  maintainer's identity or historical permission to sign for them.
- Disclose AI assistance accurately, for example with an `Assisted-by` or
  `Co-Authored-By` trailer. The human signer takes responsibility for origin and
  licensing. AI assistance does not supply a DCO sign-off.
- The prepared compliance workflow checks the sign-off requirement for `main`.
  Human review remains necessary; no remote check or sign-off is claimed by
  local preparation.
- The default branch is `main`. The initial contribution policy is maintainer
  only; public pull requests are not accepted for now. Passage does not inherit
  the upstream private/public mirror procedure.
- Follow Chris Beams' seven commit-message rules: separate subject and body with
  a blank line; limit the subject to 50 characters; capitalize it; omit its final
  period; use the imperative; wrap the body at 72 characters; explain what and
  why. When authorized to inspect staged changes for a message, describe only
  the final staged change. Include generated artifacts only when their source
  changes require regeneration.

## 5. Session evidence and document portability

- Keep project requirements and handoff in this repository. Use relative links
  only within the project and public links to the
  [Provenant projects](https://opencommit.eu/provenant) for external context.
  Do not link to a sibling checkout or an external research directory.
- Keep bulky local evidence and any session audit material under ignored local
  storage. Do not publish private logs, credentials or environment files.
- Distinguish inspected code, reported outcomes, results reproduced locally and
  proposals. Record source/tool versions and trust inputs behind verification
  claims. Equality to CLI output does not establish truthful archive history.

## 6. House style

- Use plain ASCII in new project text, including prose and code comments.
  Write clear, concrete sentences. Use commas, parentheses, colons or separate
  sentences instead of typographic dashes.
- Apply style when writing or editing relevant content. Do not sweep unrelated
  files solely to change punctuation, and preserve canonical license/DCO text.

## 7. Quality and operability

- Review security boundaries regularly and when architecture changes. Record
  findings locally for human follow-up; use this project's security policy when
  present. Do not copy upstream reporting contacts or publish sensitive findings.
- After a bug, CI failure or near miss, record the cause and a concrete prevention
  measure. Fix or explicitly quarantine flaky tests; do not count an unreliable
  result as a pass.
- Prefer explicit types at module and interface boundaries. Avoid `any` or
  `interface{}` when a concrete type expresses the contract.
- Treat developer observability as part of implementation: add suitable
  structured logs, trace spans and behavior tests for the code being changed.
  Do not expand the current scope or modify the external importer to satisfy a
  new instrumentation proposal. Record remaining observability gaps honestly.
- Preserve deterministic Temporal workflows, idempotent activities, session
  affinity for local files and bounded cleanup. Changes to these behaviors need
  failure/recovery evidence as well as normal-path checks.
- Keep secrets out of source files. Checked-in examples contain development
  placeholders only. The local prototype's lack of authentication and upload-size
  limits is a scope constraint, not evidence of production readiness.
