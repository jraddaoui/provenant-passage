#!/usr/bin/env python3
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
"""Offline deployment contract checks. No cluster changes or image publication."""
import base64
import contextlib
import importlib.util
import io
import json
from pathlib import Path
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import yaml


def module(name):
    spec = importlib.util.spec_from_file_location(name, Path(__file__).with_name(name + ".py"))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


render = module("ovh-render")
credentials = module("ovh-secrets")


class DeploymentContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        result = subprocess.run(
            ["kubectl", "kustomize", str(render.OVERLAY)],
            capture_output=True, text=True, check=True,
        )
        cls.text = result.stdout
        cls.objects = [o for o in yaml.safe_load_all(result.stdout) if o]

    def test_remote_render_has_no_credentials_or_public_backend_services(self):
        self.assertFalse(any(o["kind"] == "Secret" for o in self.objects))
        for placeholder in ("provenant-dev", "admin123", "provenant-root-dev"):
            self.assertNotIn(placeholder, self.text)
        for obj in self.objects:
            if obj["kind"] == "Service":
                self.assertEqual(obj["spec"].get("type", "ClusterIP"), "ClusterIP")

    def test_configuration_is_public_but_credentials_are_mandatory(self):
        obj = next(o for o in self.objects if o["kind"] == "Deployment" and o["metadata"]["name"] == "provenant-server")
        pod = obj["spec"]["template"]["spec"]
        config = next(v for v in pod["volumes"] if v["name"] == "config")
        self.assertNotIn("secret", config)
        self.assertIn(config["configMap"]["name"], [o["metadata"]["name"] for o in self.objects if o["kind"] == "ConfigMap"])
        env = {e["name"]: e for e in pod["containers"][0]["env"]}
        for name in ("PROVENANT_DATABASE_DSN", "PROVENANT_EVENT_REDISADDRESS"):
            self.assertEqual(env[name]["valueFrom"]["secretKeyRef"]["name"], "provenant-runtime")
            self.assertFalse(env[name]["valueFrom"]["secretKeyRef"].get("optional", False))

    def test_persistent_writers_do_not_overlap_on_one_node(self):
        for obj in self.objects:
            if obj["kind"] == "Deployment" and obj["metadata"]["name"] in ("mysql", "redis", "seaweedfs"):
                self.assertEqual(obj["spec"]["strategy"]["type"], "Recreate")
            if obj["kind"] == "PersistentVolumeClaim":
                self.assertEqual(obj["spec"]["storageClassName"], "provenant-retain")
            if obj["kind"] == "StorageClass":
                self.assertEqual(obj["reclaimPolicy"], "Retain")

    def test_three_tls_routes_share_auth_with_no_cleartext_route(self):
        objects = render.render("routes", {})
        auth = next(o for o in objects if o["kind"] == "Middleware")
        self.assertEqual(auth["spec"]["basicAuth"]["secret"], "demo-basic-auth")
        routes = [o for o in objects if o["kind"] == "IngressRoute"]
        self.assertEqual(len(routes), 3)
        self.assertEqual({r["spec"]["entryPoints"][0] for r in routes}, {"websecure", "temporal", "seaweedfs"})
        for obj in routes:
            self.assertIn("tls", obj["spec"])
            self.assertEqual(obj["spec"]["routes"][0]["middlewares"], [{"name": "demo-auth"}])

    def test_unset_public_inputs_fail_before_any_output(self):
        for phase in ("app", "tls"):
            with self.subTest(phase=phase), self.assertRaises(ValueError):
                render.render(phase, {})
        with self.assertRaises(ValueError):
            render.image({"serverImage": "ghcr.io/example/server:latest"}, "serverImage")
        for value in ("127.0.0.1", "10.0.0.1"):
            with self.assertRaises(ValueError):
                render.render("tls", {"publicIP": value})

    def test_ip_certificate_uses_renewal_and_default_certificate_for_no_sni(self):
        objects = render.render("tls", {
            "publicIP": "8.8.8.8", "acmeEmail": "test@example.org",
            "acmeServer": "https://acme-staging-v02.api.letsencrypt.org/directory",
        })
        cert = next(o for o in objects if o["kind"] == "Certificate")
        self.assertEqual(cert["spec"]["ipAddresses"], ["8.8.8.8"])
        self.assertEqual(cert["spec"]["renewBefore"], "48h")
        issuer = next(o for o in objects if o["kind"] == "Issuer")
        self.assertEqual(issuer["spec"]["acme"]["profile"], "shortlived")
        store = next(o for o in objects if o["kind"] == "TLSStore")
        self.assertEqual(store["spec"]["defaultCertificate"]["secretName"], cert["spec"]["secretName"])

    def test_promoting_staging_changes_issuer_to_trigger_new_certificate(self):
        settings = {"publicIP": "8.8.8.8", "acmeEmail": "test@example.org"}
        refs = []
        for endpoint in ("acme-staging-v02", "acme-v02"):
            settings["acmeServer"] = f"https://{endpoint}.api.letsencrypt.org/directory"
            cert = next(o for o in render.render("tls", settings) if o["kind"] == "Certificate")
            refs.append(cert["spec"]["issuerRef"]["name"])
        self.assertNotEqual(*refs)

    def test_application_render_uses_both_published_digests(self):
        settings = {
            "serverImage": "ghcr.io/example/server@sha256:" + "a" * 64,
            "uiImage": "ghcr.io/example/ui@sha256:" + "b" * 64,
        }
        objects = render.render("app", settings)
        self.assertEqual(
            {o["spec"]["template"]["spec"]["containers"][0]["image"] for o in objects},
            set(settings.values()),
        )


class CredentialContract(unittest.TestCase):
    def test_checkout_location_is_rejected(self):
        with self.assertRaises(ValueError):
            credentials.private_directory(credentials.ROOT / ".secrets")

    def test_generated_files_are_private_consistent_and_never_overwritten(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp) / "credentials"
            password = "test-only-password-123456"
            output = io.StringIO()
            with patch.object(credentials.getpass, "getpass", return_value=password), contextlib.redirect_stdout(output):
                credentials.initialize(directory, "demo")
            self.assertNotIn(password, output.getvalue())
            self.assertEqual(stat.S_IMODE(directory.stat().st_mode), 0o700)
            for file in directory.iterdir():
                self.assertEqual(stat.S_IMODE(file.stat().st_mode), 0o600)
            self.assertTrue((directory / "htpasswd").read_text().startswith("demo:$2"))
            db = dict(line.split("=", 1) for line in (directory / "mysql.env").read_text().splitlines())
            self.assertIn(db["password"], (directory / "application.env").read_text())
            self.assertNotEqual(db["password"], db["root-password"])
            with self.assertRaises(ValueError):
                credentials.initialize(directory, "demo")

    def test_wrong_context_endpoint_is_rejected_before_any_mutation(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            for name in credentials.FILES:
                file = directory / name
                file.write_text("synthetic test data")
                file.chmod(0o600)
            fake = subprocess.CompletedProcess([], 0, json.dumps({
                "clusters": [{"cluster": {"server": "https://wrong.example.org"}}]
            }), "")
            with patch.object(credentials.subprocess, "run", return_value=fake) as run:
                with self.assertRaisesRegex(ValueError, "nothing applied"):
                    credentials.apply(directory, "ovh-demo", "https://expected.example.org")
                self.assertEqual(run.call_count, 1)
                self.assertIn("view", run.call_args.args[0])

    def test_real_kubectl_renders_all_credentials_before_one_apply(self):
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp) / "credentials"
            output = io.StringIO()
            with patch.object(credentials.getpass, "getpass", return_value="test-only-password-123456"), contextlib.redirect_stdout(output):
                credentials.initialize(directory, "demo")
            kubeconfig = Path(tmp) / "kubeconfig"
            kubeconfig.write_text(yaml.safe_dump({
                "apiVersion": "v1", "kind": "Config",
                "clusters": [{"name": "offline", "cluster": {"server": "https://unused.invalid"}}],
                "contexts": [{"name": "ovh-demo", "context": {"cluster": "offline", "user": "offline"}}],
                "users": [{"name": "offline", "user": {"token": "synthetic-offline-token"}}],
                "current-context": "ovh-demo",
            }))
            real_run = subprocess.run
            rendered = []
            applied = []

            def offline_run(command, **kwargs):
                action = command[3:]
                if action == ["config", "view", "--minify", "-o", "json"]:
                    return subprocess.CompletedProcess(command, 0, json.dumps({
                        "clusters": [{"cluster": {"server": "https://expected.example.org"}}]
                    }), "")
                if action[:4] == ["-n", "provenant", "create", "secret"]:
                    self.assertIn("--dry-run=client", command)
                    result = real_run([command[0], "--kubeconfig", str(kubeconfig), *command[1:]], **kwargs)
                    rendered.append(json.loads(result.stdout)["metadata"]["name"])
                    return result
                if action[:3] == ["-n", "provenant", "apply"]:
                    self.assertEqual(set(rendered), set(credentials.SPECS))
                    self.assertIn("--server-side", command)
                    applied.append(json.loads(kwargs["input"]))
                    return subprocess.CompletedProcess(command, 0, "", "")
                self.fail(f"unexpected subprocess: {command}")

            with patch.object(credentials.subprocess, "run", side_effect=offline_run), contextlib.redirect_stdout(output):
                credentials.apply(directory, "ovh-demo", "https://expected.example.org")
            self.assertEqual(len(applied), 1)
            objects = {obj["metadata"]["name"]: obj for obj in applied[0]["items"]}
            self.assertEqual(set(objects), set(credentials.SPECS))
            redis = objects["redis-secret"]["data"]
            self.assertEqual(set(redis), {"password", "redis.conf"})
            password = (directory / "redis.env").read_text().strip().split("=", 1)[1]
            self.assertEqual(base64.b64decode(redis["password"]).decode(), password)
            self.assertEqual(base64.b64decode(redis["redis.conf"]), (directory / "redis.conf").read_bytes())
            self.assertNotIn(password, output.getvalue())
            self.assertNotIn("test-only-password-123456", output.getvalue())


if __name__ == "__main__":
    unittest.main()
