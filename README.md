<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# Provenant Passage

Provenant Passage is a web application for Provenant workflows. Its first
workflow converts Archivematica AIPs into OCFL objects using provenant-import.

This import proof of concept uses Go, Goa, Temporal and Vue, with MySQL,
SeaweedFS and Redis. Upload an AIP, choose what the output should contain,
follow its progress and download the resulting OCFL object.

See [acceptance criteria](docs/acceptance-criteria.md) and
[local evidence](docs/acceptance.md) for this application's requirements,
reproduced results and limits, distinct from the importer's earlier experiments.
The related [Provenant projects](https://opencommit.eu/provenant) are maintained
independently.

## Run locally

Prerequisites: Go 1.27.4, Node 26, Python 3.12 or newer, Docker with BuildKit,
kubectl, Tilt and the user-provided local `provenant` cluster. Use a kubectl
version compatible with your cluster; Tilt supplies the normal development
port-forwards. Local acceptance verified context
`k3d-provenant`; commands never change your global current context.

Obtain the independently maintained
[provenant-import](https://opencommit.eu/provenant/provenant-import) source from
the public project. Provision the same immutable source archive used by CI into
this project's ignored cache:

```sh
export IMPORTER_ARCHIVE_URL='https://opencommit.eu/api/v1/repos/provenant/provenant-import/archive/21d9649813ac18031eee965e98aa7a54cc7be348.tar.gz'
export IMPORTER_ARCHIVE_SHA256='d2428d9cfd3cdfa7107461309c8884118dd69e3f47c120a4f31007fb63cd505c'
python3 hack/provision-importer.py .cache/provenant-import
```

The helper verifies the archive digest and refuses to replace an existing source
directory. Make, Docker and Tilt default to `.cache/provenant-import`. To use an
existing source directory elsewhere, export `IMPORTER_SOURCE` with its absolute
path before running commands; `.tilt.env` also supports this setting for Tilt.
The build compiles the importer separately, with its own module graph and
read-only module settings. No converter internals are imported into this
application.

```sh
make tools importer build
cd ui
npm ci --ignore-scripts
npm run check
cd ..
tilt up --context k3d-provenant
```

Tilt builds `provenant-server` and `provenant-ui`, and deploys MySQL, Temporal,
SeaweedFS and Redis in Kubernetes namespace `provenant`. It initializes Temporal's
schemas and namespace before starting the application. Normal startup preserves
data. Automatic Docker cache pruning is disabled.

Local endpoints: UI [localhost:8080](http://localhost:8080), API
[localhost:9000/aips](http://localhost:9000/aips), Temporal Web
[localhost:7440](http://localhost:7440), SeaweedFS admin
[localhost:23646](http://localhost:23646); MySQL forwards to port 3306.
These forwards exist while Tilt runs. Use `.tilt.env` to override `UI_PORT`,
`API_PORT`, `TEMPORAL_UI_PORT`, `SEAWEED_PORT` and `MYSQL_PORT` for collisions.
The context override is `KUBE_CONTEXT`; verify it resolves to the intended local
cluster before use. Manual reset is restricted to `k3d-provenant`.

Set `UI_DEV=true` in `.tilt.env` for Vite and source live updates; the default
serves a production build through Nginx. `TRIGGER_MODE_AUTO=true` enables automatic
application rebuilds; otherwise trigger them manually in Tilt. Both modes proxy
all `/api/` routes, including streaming uploads, downloads and SSE.

## Import an AIP

Open the UI, choose **Upload**, select one `.tar` or `.7z` Archivematica AIP and
choose its output contents:

| Choice | Contents |
| --- | --- |
| Files and preservation metadata (default) | Extracted files and original AIP METS. |
| Files, metadata and original checksum files | Extracted files, METS and original BagIt manifests/tag manifests. |
| Original AIP and extracted content | Complete archive in OCFL v1, extracted files and METS in v2. |

The upload page explains exclusions before submission. Every extracted view
omits submissionDocumentation (including transfer METS), bagit.txt and
bag-info.txt as separate files. The original AIP choice retains these within the
archive. All options receive OCFL inventories and checksums.

Click Uppy's upload button. The chosen option stays fixed for retries. The AIPs
page shows queued, processing, completed, error or canceled requests, newest
first. Open an AIP for details. A completed import offers **Download ZIP**,
containing one `object/` directory with the unchanged OCFL object. Below the
details, expand **JSON report** or **Markdown report** to view the separately
stored reports. Each section loads through the API on first expansion; JSON is
shown as preformatted text and Markdown is rendered with sanitized HTML.
An error or cancellation has no download or report sections.
If an upload response is interrupted, check the AIPs list before manually retrying;
the server may already have accepted it.

## Architecture and configuration

Goa's streaming API stores the original archive in `am-aips`, persists one AIP
row in MySQL, and starts a UUID-addressed Temporal workflow. A session keeps
download, conversion, report checks, ZIP creation and output uploads on one worker.
Worker loss starts a new workspace and redownloads the source. Success requires
clean reports, valid ZIP and all three output uploads before committing the
`ocfl-objects` key. Redis Pub/Sub carries transient update notifications; the UI
refetches after reconnect and polls active imports to recover missed events.

The application database is `provenant`, with one domain table. Temporal uses
separate `temporal` and `temporal_visibility` databases and its own `provenant`
namespace. UUIDs identify import requests; source BagIt identifiers remain intact.
The server fixes the UTC creation time once and passes it unchanged to retries.

Defaults are in `provenant.toml`. An optional ignored `provenant.local.toml` merges
over them; `--config` selects an explicit file instead. `PROVENANT_` environment
overrides apply last, for example `PROVENANT_DATABASE_DSN`,
`PROVENANT_TEMPORAL_ADDRESS` and `PROVENANT_EVENT_REDISADDRESS`. Tilt mounts the
selected TOML file. Checked-in Kubernetes credentials are disposable development
placeholders. Override them locally, and keep private values out of source files.

## Reset disposable application data

Use Tilt's **Reset local application data** button, or explicitly run:

```sh
bash hack/reset.sh --confirm-reset-provenant
```

Reset stops the application, terminates and confirms active import workflows,
recreates only the application database and `am-aips`/`ocfl-objects` buckets,
runs migrations, then restarts the application. Temporal databases and history
remain. Only the named previous reset Job is replaced. If reset fails, the
application stays stopped; inspect `kubectl --context k3d-provenant -n provenant
logs job/provenant-reset` before retrying. Do not trigger another application
rollout while resetting.

## Development checks

```sh
make fmt lint test-race vet workflowcheck build
python3 hack/check-generated.py
python3 hack/reuse-check.py /path/to/reuse
```

The license checker requires REUSE 6.2.0 and checks a source-only copy without
executing Git. `make gen-enums gen-ent gen-goa gen-ui-client gen-mock atlas-hash`
regenerates source-derived files. `make mod-tidy` maintains the module graph.
UI checks run with `npm run check` from `ui/`. Live integration tests use an admin
`PASSAGE_TEST_MYSQL_DSN` with no database name (they create/drop an isolated test
schema), and `PASSAGE_TEST_REDIS_URL` for distinct-instance event fan-out.

GitHub Actions pins the importer archive URL and SHA-256 directly in
[the build workflow](.github/workflows/ci.yml); no repository variables are needed.
CI verifies the digest, provisions the external CLI source in
`.cache/provenant-import`, builds both images locally, checks generation/licensing,
and enforces human DCO sign-off on main. To change the importer revision, update
the workflow and local setup values together after verifying the new archive and
reviewing its source changes against [the source inventory](docs/importer-source-sha256.json).
No image publication is configured. See [local evidence](docs/acceptance.md) for
validation results and remote CI status.

## Scope and limitations

Conversion retains the importer's profile rules and loss accounting. It does
not authenticate an archive's history or sign an attestation. Trust inputs and
verification policy remain external. This is a local prototype with no
authentication, production-readiness claim or configured upload-size ceiling.

## AI assistance

This project's implementation was prepared with OpenAI Codex, directed by
the human developer. AI assistance must be disclosed in future human commits.
An AI cannot provide DCO sign-off; the human signer takes responsibility for
origin and licensing. No human sign-off is claimed by these local edits.

## Licensing and contributions

Code, build files and configuration are Apache-2.0; prose is CC-BY-4.0.
See [LICENSES](LICENSES), [REUSE.toml](REUSE.toml), [DCO](DCO) and
[CONTRIBUTING.md](CONTRIBUTING.md). Upstream attribution is retained in
[source reuse](docs/enduro-reuse.md).

The selected repository home is
[jraddaoui/provenant-passage](https://github.com/jraddaoui/provenant-passage).
Local preparation does not create or publish that repository.
