<!-- SPDX-FileCopyrightText: 2026 Artefactual Systems Limited -->
<!-- SPDX-License-Identifier: CC-BY-4.0 -->

# OVH single-node demo deployment

Prepared 2026-11-26 for a human-operated deployment. The user selected public IP
access, public GHCR application images, and the same HTTP Basic Auth credentials
for Passage, Temporal Web and SeaweedFS UI. This is a restricted, single-node demo;
all admitted users share access to the data and administrative interfaces.

The files are prepared locally. No OVH deployment, image publication, public
certificate issuance, browser acceptance or backup restoration has been performed
by the agent. Commits, image pushes and cluster changes are human operations.

## Layout and exposure

- `hack/kube/base`: shared application and infrastructure manifests, no credentials.
- `hack/kube/overlays/local`: disposable credentials; Tilt uses this overlay.
- `hack/kube/overlays/ovh`: public TOML, resource/storage/network patches and settings.
- `hack/kube/overlays/ovh/edge`: pinned Traefik/cert-manager charts, values, TLS and routes.
- `hack/ovh-render.py`: offline renderer with ordered deployment phases.
- `hack/ovh-secrets.py`: private credential initialization and explicit Secret apply.

One OVH Octavia LoadBalancer fronts Traefik. Its IP is the public endpoint; do not
use a worker node IP in the certificate. The LoadBalancer and persistent volumes
are additional billable resources. Its four TCP listeners are:

| Public endpoint | Destination | Access |
| --- | --- | --- |
| `https://IP/` | Passage UI, including its `/api/` proxy | Shared Basic Auth |
| `https://IP:8443/` | Temporal Web | Shared Basic Auth |
| `https://IP:9443/` | SeaweedFS UI | Shared Basic Auth |
| `http://IP:80/` | ACME validation only | No application route |

The browser may ask for the same credentials again on another port. SeaweedFS's
own admin login, if requested, uses the same chosen username/password. S3, MySQL,
Redis, Temporal RPC, the Passage API and health port have no public Service.
Each UI remains at `/`, avoiding changes to frontend base paths.

Traefik terminates TLS and applies one `demo-auth` Middleware referencing one
private Secret. The IP certificate is also the default certificate for clients
without SNI. All application routes explicitly require TLS; Basic Auth is never
offered on port 80. A missing auth Secret must leave routes unavailable.

The dedicated application namespace permits traffic among its own pods. The
edge namespace can reach only the three UI ports. Egress remains allowed. This
is namespace isolation, not isolation between application components. Confirm
NetworkPolicy enforcement on the chosen cluster before declaring it verified.

## Prerequisites and public inputs

Requirements: kubectl, Helm, Python 3.12+, PyYAML 6.0.3, curl, Docker with BuildKit
and `htpasswd` from apache2-utils. Before installing or upgrading, check that
kubectl, Helm and the [pinned charts](../hack/kube/overlays/ovh/edge/charts.yaml)
support the cluster's Kubernetes version. Use the version commands below and
the official compatibility policies linked in the references.

Run commands from the repository root, one block at a time, and stop on errors.
Choose the explicit context from your downloaded OVH kubeconfig. Keep that
kubeconfig outside the checkout.

```sh
export OVH_CONTEXT='YOUR_OVH_CONTEXT'
export OVH_API_SERVER='https://YOUR_OVH_KUBERNETES_API_ENDPOINT'
kubectl --context "$OVH_CONTEXT" config view --minify -o jsonpath='{.clusters[0].cluster.server}'
kubectl --context "$OVH_CONTEXT" version
helm version --short
kubectl --context "$OVH_CONTEXT" get nodes
kubectl --context "$OVH_CONTEXT" get storageclass csi-cinder-high-speed -o yaml
```

Compare the context's API endpoint with OVH's endpoint. `provenant-retain` uses
`cinder.csi.openstack.org` and volume type `high-speed`; confirm that type exists
in your region before applying. Initial disk requests are MySQL 10 GiB, Redis
1 GiB and SeaweedFS 50 GiB. Edit them before the first creation if needed. The
working directory is a disposable 20 GiB `emptyDir` with a 25 GiB pod ephemeral
storage limit. Budget for decompressed content and ZIP creation, not just upload
size. CPU/memory settings are initial budgets, not measured capacity guarantees.

The versioned [settings](../hack/kube/overlays/ovh/settings.yaml) hold the public
LoadBalancer IP, ACME contact email/server and the two published image digests.
Empty values deliberately block rendering their dependent phase. Registry paths,
IPs and email addresses are not credential values; review what you wish to publish.

## Publish the two public images manually

Provision the pinned importer as described in [local setup](../README.md#run-locally).
Use a unique tag for this reviewed source state. Build for the worker architecture;
the existing build helper uses the Docker builder's architecture (normally amd64
on this workstation). These commands are for the human maintainer:

```sh
export IMAGE_TAG=ovh-demo-20261126-1
export SERVER_IMAGE="ghcr.io/jraddaoui/provenant-passage-server:$IMAGE_TAG"
export UI_IMAGE="ghcr.io/jraddaoui/provenant-passage-ui:$IMAGE_TAG"
EXPECTED_REF="$SERVER_IMAGE" bash hack/build_docker.sh provenant-server
EXPECTED_REF="$UI_IMAGE" UI_BUILD_TARGET=provenant-ui bash hack/build_docker.sh provenant-ui
docker login ghcr.io -u jraddaoui
docker push "$SERVER_IMAGE"
docker push "$UI_IMAGE"
docker image inspect "$SERVER_IMAGE" --format '{{index .RepoDigests 0}}'
docker image inspect "$UI_IMAGE" --format '{{index .RepoDigests 0}}'
```

For a manual GHCR login use a token with `write:packages` at the password prompt,
not in the command line or a repository file. In GitHub Packages, set both
packages to **Public** after first publication; repository visibility alone does
not do this. Test anonymous pulls. The Dockerfiles include the source-repository
label. Copy each `ghcr.io/...@sha256:...` result into settings. No imagePullSecret
is needed for public images. Third-party runtime images retain their existing
version tags; the application digests fix the images actually deployed.

## Install the edge and obtain its IP

Fetch the charts and verify the hashes in [charts.yaml](../hack/kube/overlays/ovh/edge/charts.yaml):

```sh
python3 hack/ovh-charts.py
helm upgrade --install cert-manager .cache/ovh-charts/cert-manager-v1.21.2.tgz \
  --kube-context "$OVH_CONTEXT" --namespace cert-manager --create-namespace \
  --values hack/kube/overlays/ovh/edge/cert-manager-values.yaml --wait --timeout 10m
helm upgrade --install passage-edge .cache/ovh-charts/traefik-42.4.1.tgz \
  --kube-context "$OVH_CONTEXT" --namespace passage-edge --create-namespace \
  --values hack/kube/overlays/ovh/edge/traefik-values.yaml --wait --timeout 10m
kubectl --context "$OVH_CONTEXT" -n passage-edge get service passage-edge
```

Record the Service's assigned public IPv4 in settings. Preserve this Service when
upgrading: deleting/recreating it can change the IP and require a new certificate.
There are no application routes yet. Do not install another default ingress
controller or publish individual backend Services for this setup.

## Create secrets and deploy the application in order

The helper creates a new directory outside the checkout, with mode 0700 and files
0600. It prompts for the shared demo password and generates independent random
MySQL, Redis and S3 credentials. Basic Auth stores a bcrypt hash. The SeaweedFS
admin environment also needs the chosen password. Back up this private directory
in your password manager or encrypted private storage; do not paste it in chat.

```sh
export PASSAGE_SECRETS_DIR="$HOME/.config/provenant-passage/ovh-secrets"
python3 hack/ovh-secrets.py init "$PASSAGE_SECRETS_DIR" --username demo
python3 hack/ovh-render.py foundation > /tmp/passage-foundation.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-foundation.yaml
python3 hack/ovh-secrets.py apply "$PASSAGE_SECRETS_DIR" \
  --context "$OVH_CONTEXT" --api-server "$OVH_API_SERVER"
for service in mysql redis seaweedfs; do
  kubectl --context "$OVH_CONTEXT" -n provenant rollout status "deployment/$service" --timeout=10m
done
```

Foundation pods wait for their Secrets until the apply completes. The secret
helper validates the context endpoint and local file permissions, renders all
five Secrets in memory, then applies them with server-side apply. Redis's password
is rendered from its env file; the config file is added to the same object in
memory because kubectl cannot combine `--from-env-file` with `--from-file`.
It does not print values or store a last-applied annotation. It refuses to overwrite an
existing credential directory. Treat a failed apply as potentially partial and
rerun with the same files after diagnosis.

If local rendering fails, no Secrets are applied. After fixing the error, rerun
`apply` with the existing directory, without rerunning `init`.

Tilt previously supplied ordering; `kubectl apply -k` does not reproduce it.
Apply and wait for these phases individually:

```sh
python3 hack/ovh-render.py schema > /tmp/passage-schema.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-schema.yaml
kubectl --context "$OVH_CONTEXT" -n provenant wait --for=condition=complete \
  job/temporal-schema-1-2-0-1 --timeout=10m

python3 hack/ovh-render.py temporal > /tmp/passage-temporal.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-temporal.yaml
for service in temporal-frontend temporal-history temporal-matching temporal-worker temporal-web; do
  kubectl --context "$OVH_CONTEXT" -n provenant rollout status "deployment/$service" --timeout=10m
done

python3 hack/ovh-render.py namespace > /tmp/passage-namespace.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-namespace.yaml
kubectl --context "$OVH_CONTEXT" -n provenant wait --for=condition=complete \
  job/temporal-namespace-1-2-0-1 --timeout=10m

python3 hack/ovh-render.py app > /tmp/passage-app.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-app.yaml
for service in provenant-server provenant-ui; do
  kubectl --context "$OVH_CONTEXT" -n provenant rollout status "deployment/$service" --timeout=10m
done
```

Application SQL migrations run at server startup. Temporal's Jobs retain their
existing versioned names; completed Jobs need not be recreated for an application
update. For a Temporal upgrade, review its migration procedure and use a new Job
name when its pod template changes. Do not delete/recreate databases to resolve
an immutable-Job error. Temporal admin-tools stays at zero replicas.

## Issue TLS before exposing the three interfaces

Let's Encrypt now issues certificates for public IPs. They require the
`shortlived` profile and expire after 160 hours. This setup requests automatic
renewal 48 hours before expiry. Start with the staging endpoint already in settings:

```sh
python3 hack/ovh-render.py tls > /tmp/passage-tls.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-tls.yaml
kubectl --context "$OVH_CONTEXT" -n passage-edge wait --for=condition=Ready \
  certificate/passage-ip --timeout=10m
```

Then change `acmeServer` to `https://acme-v02.api.letsencrypt.org/directory`, render
and apply TLS again. The renderer selects a distinct production Issuer/account;
changing the Certificate's issuer reference triggers replacement of the staging
certificate. Wait for issuance and check the actual served certificate. A staging
certificate is not browser trusted. Keep port 80 reachable for validation and
renewal. Do not attach Basic Auth to the ACME solver route.

Before adding application routes, this should validate TLS and return HTTP 404:

```sh
export PASSAGE_IP='YOUR_LOADBALANCER_IPV4'
curl --silent --show-error --output /dev/null --write-out '%{http_code}\n' "https://$PASSAGE_IP/"
```

Do not use `--insecure` to send credentials. After the production certificate is
served, enable the protected interfaces:

```sh
python3 hack/ovh-render.py routes > /tmp/passage-routes.yaml
kubectl --context "$OVH_CONTEXT" apply -f /tmp/passage-routes.yaml
for port in 443 8443 9443; do
  curl --silent --show-error --output /dev/null --write-out '%{http_code}\n' "https://$PASSAGE_IP:$port/"
done
```

All three unauthenticated responses must be 401. Test wrong credentials as well.
Use `curl --user demo` to prompt for the password, or visit each URL in a browser.
Verify Passage upload/status events/download, Temporal Web navigation and
SeaweedFS navigation. Verify `/api/aips` also returns 401 without authentication.
Confirm 80 does not serve a UI, and S3/RPC/database ports are not publicly reachable.
Observe successful certificate renewal and the new expiry before relying on it.

## Operations and limits

- Apply public configuration changes through the renderer, then explicit context
  commands. Update image digests in settings and reapply the `app` phase for an
  application release. Reapply `foundation` when ConfigMaps change. Retain old
  ConfigMaps needed by a previous Deployment revision until review.
- The UI's Nginx proxy limits uploads to `1g` in OVH, with streaming and SSE
  buffering still disabled. Local development retains its unlimited default.
  This is a public-proxy limit; the internal API still has no application-level
  size ceiling. Compressed archive expansion can exceed the upload size.
- MySQL, Redis and SeaweedFS use Recreate rollouts to avoid simultaneous writers
  to a single-node volume. Updates and node maintenance can cause downtime.
- Retain protects disks from automatic PVC deletion, not from data corruption,
  account deletion or operator actions. Save consistent MySQL backups (all three
  databases) and SeaweedFS data outside this node, and test restoration. Volume
  snapshots alone have not been verified as application-consistent backups here.
- MySQL currently retains the local design's shared `provenant` user for Passage
  and Temporal. This demo is not a multi-tenant authorization boundary. Splitting
  database principals is a separate hardening change.
- Changing a MySQL Secret does not change passwords in an initialized database.
  Coordinate database password changes with dependent Secrets and rollouts.
  Redis/SeaweedFS also need reload/restart after credential rotation. Update the
  Basic Auth hash and SeaweedFS admin password together when changing demo access.
- Private credentials, kubeconfigs, TLS/ACME private keys and backups remain
  outside Git. Certificates/account keys are maintained in Kubernetes Secrets by
  cert-manager. Public manifests reference their names only.
- Never run the local reset script against OVH. No remote reset/prune command is
  provided. Keep data and credential recovery separate from rebuilding manifests.

## Evidence and references

Offline checks: `PYTHONDONTWRITEBYTECODE=1 python3 hack/test_ovh.py`. These cover
secret-free rendering, mandatory runtime credentials, retained disks, rollout
strategy, three authenticated TLS routes, missing-input rejection and private
credential safeguards. They do not substitute for the OVH acceptance above.
See [acceptance](acceptance.md) for the recorded results and remaining checks.

Primary references checked 2026-11-26:

- [kubectl version-skew policy](https://kubernetes.io/releases/version-skew-policy/#kubectl) and [Helm compatibility policy](https://helm.sh/docs/topics/version_skew/).
- [OVH Octavia listeners and timeout annotations](https://docs.ovhcloud.com/en/guides/public-cloud/containers-orchestration/managed-kubernetes/expose-applications-using-load-balancer).
- [OVH Cinder volumes and reclaim policy](https://docs.ovhcloud.com/en/guides/public-cloud/containers-orchestration/managed-kubernetes/set-up-persistent-volume), updated 2026-10-26. Default classes delete backing volumes with PVCs; this overlay defines Retain explicitly.
- [Traefik BasicAuth](https://doc.traefik.io/traefik/reference/routing-configuration/http/middlewares/basicauth/) and [TLSStore](https://doc.traefik.io/traefik/reference/routing-configuration/kubernetes/crd/tls/tlsstore/).
- [Let's Encrypt IP certificate availability](https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability), 2026-01-15.
- [cert-manager IP support](https://cert-manager.io/docs/releases/release-notes/release-notes-1.1/#ip-addresses), [ACME profiles](https://cert-manager.io/docs/configuration/acme/#acme-certificate-profiles) (available since 1.18), and [supported Kubernetes versions](https://cert-manager.io/docs/releases/).
- [GHCR publication, visibility and repository linkage](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

Prepared with OpenAI Codex assistance; human review and sign-off remain required.
