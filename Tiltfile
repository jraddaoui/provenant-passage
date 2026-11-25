# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
version_settings(constraint=">=0.35.0")
docker_prune_settings(disable=True)
ci_settings(timeout="20m", readiness_timeout="15m", k8s_grace_period="2m")
load("hack/tilt/dotenv/Tiltfile", "dotenv")
load("hack/tilt/uibutton/Tiltfile", "cmd_button")

if os.path.exists(".tilt.env"):
    dotenv(fn=".tilt.env")

context = os.environ.get("KUBE_CONTEXT", "k3d-provenant")
true_values = ["true", "1", "yes", "t", "y"]
ui_dev = os.environ.get("UI_DEV", "").lower() in true_values
allow_k8s_contexts(context)
if k8s_context() != context:
    fail("Use tilt --context " + context + "; do not change the global context.")

importer = os.environ.get("IMPORTER_SOURCE", ".cache/provenant-import")
custom_build(
    "provenant-server:dev",
    command=["hack/build_docker.sh", "provenant-server"],
    deps=["cmd", "internal", "go.mod", "go.sum", "Dockerfile", "provenant.toml", "LICENSES", "DCO", "NOTICE", "docs/importer-source-sha256.json", "hack/build_docker.sh", "hack/collect-go-notices.sh", importer + "/go.mod", importer + "/go.sum", importer + "/cmd", importer + "/internal", importer + "/LICENSES", importer + "/DCO"],
    env={"IMPORTER_SOURCE": importer},
)
ui_live_update = []
if ui_dev:
    ui_live_update = [
        fall_back_on(["ui/vite.config.ts", "ui/package.json", "ui/package-lock.json", "ui/.env"]),
        sync("ui/src", "/app/src"), sync("ui/public", "/app/public"), sync("ui/index.html", "/app/index.html"),
    ]
custom_build("provenant-ui:dev", command=["hack/build_docker.sh", "provenant-ui"],
    deps=["ui"], ignore=["ui/node_modules", "ui/dist", "ui/coverage", "ui/*.tsbuildinfo"],
    env={"UI_BUILD_TARGET": "builder" if ui_dev else "provenant-ui"},
    live_update=ui_live_update)

config_path = "provenant.local.toml" if os.path.exists("provenant.local.toml") else "provenant.toml"
configuration = encode_yaml({
    "apiVersion": "v1", "kind": "Secret",
    "metadata": {"name": "provenant-config", "namespace": "provenant"},
    "type": "Opaque", "stringData": {"provenant.toml": str(read_file(config_path))},
})
k8s_yaml([kustomize("hack/kube/base"), configuration])

trigger = TRIGGER_MODE_AUTO if os.environ.get("TRIGGER_MODE_AUTO", "").lower() in true_values else TRIGGER_MODE_MANUAL
k8s_resource("mysql", labels=["Infrastructure"], port_forwards=[os.environ.get("MYSQL_PORT", "3306") + ":3306"])
k8s_resource("redis", labels=["Infrastructure"])
k8s_resource("seaweedfs", labels=["Infrastructure"], port_forwards=[os.environ.get("SEAWEED_PORT", "23646") + ":23646"])
k8s_resource("temporal-schema-1-2-0-1", labels=["Temporal"], resource_deps=["mysql"])
for name in ["temporal-frontend", "temporal-history", "temporal-matching", "temporal-worker"]:
    k8s_resource(name, labels=["Temporal"], resource_deps=["temporal-schema-1-2-0-1"])
k8s_resource("temporal-admintools", labels=["Temporal"], resource_deps=["temporal-frontend"])
k8s_resource("temporal-namespace-1-2-0-1", labels=["Temporal"], resource_deps=["temporal-frontend"])
k8s_resource("temporal-web", labels=["Temporal"], resource_deps=["temporal-frontend"], port_forwards=[os.environ.get("TEMPORAL_UI_PORT", "7440") + ":8080"])
k8s_resource("provenant-server", labels=["Provenant"], trigger_mode=trigger,
    resource_deps=["mysql", "redis", "seaweedfs", "temporal-namespace-1-2-0-1"],
    port_forwards=[os.environ.get("API_PORT", "9000") + ":9000"])
cmd_button("reset-provenant", resource="provenant-server",
    argv=["bash", "hack/reset.sh", "--confirm-reset-provenant"],
    text="Reset local application data", requires_confirmation=True,
    env={"KUBE_CONTEXT": context})
k8s_resource("provenant-ui", labels=["Provenant"], trigger_mode=trigger,
    resource_deps=["provenant-server"], port_forwards=[os.environ.get("UI_PORT", "8080") + ":80"])
