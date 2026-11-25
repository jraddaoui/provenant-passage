<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Contributing to Provenant Passage

The project follows Provenant's initial maintainer-only, read-only contribution
policy. Pull requests are not accepted for now. Questions, bug reports and
experiment results can be raised with the maintainer; assistants must not
submit issues or messages without explicit authorization.

## Human development practice

Every commit reaching `main` requires a human [DCO](DCO) sign-off matching its
author. AI assistance is disclosed, and an AI never signs on a human's behalf.
All commits, pushes, publication and PR submission belong to the human developer.
The current workspace setup prohibits all Git operations, including inspection.

Commit messages follow Chris Beams' seven rules: separate subject and body with
a blank line; limit the subject to 50 characters; capitalize it; omit its final
period; use the imperative; wrap the body at 72 characters; explain what and why.
Later commit-message requests inspect the staged final change only when Git
inspection has been authorized. Session history is relevant only to explain
the final result. Future human PRs link an issue and include UI screenshots for
visible changes. Generated artifacts accompany source changes requiring them.

Write focused failing tests before implementation. Local checks and future
GitHub Actions checks are recorded separately. Code is Apache-2.0 and prose is
CC-BY-4.0, with preserved upstream notices and REUSE coverage.

See [AGENTS.md](AGENTS.md) and this project's
[development conventions](development-conventions.md) for the local policy.
