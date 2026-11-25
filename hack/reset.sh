#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
if [[ "${1:-}" != --confirm-reset-provenant ]]; then
  echo 'Usage: hack/reset.sh --confirm-reset-provenant (deletes local application data)' >&2
  exit 1
fi
context="${KUBE_CONTEXT:-k3d-provenant}"
if [[ "$context" != k3d-provenant ]]; then
  echo 'Reset is restricted to the verified k3d-provenant context.' >&2
  exit 1
fi
k=(kubectl --context "$context" --namespace provenant)
image_ref="$("${k[@]}" get deployment provenant-server -o jsonpath='{.spec.template.spec.containers[0].image}')"
"${k[@]}" scale deployment provenant-server --replicas=0
"${k[@]}" wait --for=delete pod -l app=provenant-server --timeout=150s
# This named Job is the only previous resource removed.
"${k[@]}" delete job provenant-reset --ignore-not-found --wait=true
"${k[@]}" create -f - <<YAML
apiVersion: batch/v1
kind: Job
metadata:
  name: provenant-reset
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 660
  template:
    spec:
      serviceAccountName: provenant
      restartPolicy: Never
      containers:
        - name: reset
          image: ${image_ref}
          command: [provenant-reset, --confirm-reset-provenant]
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef: {name: mysql-secret, key: root-password}
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef: {name: seaweedfs-secret, key: access_key_id}
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef: {name: seaweedfs-secret, key: secret_access_key}
            - name: AWS_EC2_METADATA_DISABLED
              value: "true"
          volumeMounts:
            - {name: config, mountPath: /etc/provenant, readOnly: true}
      volumes:
        - name: config
          secret: {secretName: provenant-config}
YAML
if ! "${k[@]}" wait --for=condition=complete job/provenant-reset --timeout=11m; then
  "${k[@]}" logs job/provenant-reset
  echo 'Reset failed; application remains stopped. Inspect the named reset Job before retrying.' >&2
  exit 1
fi
"${k[@]}" logs job/provenant-reset
"${k[@]}" scale deployment provenant-server --replicas=1
"${k[@]}" rollout status deployment/provenant-server --timeout=180s
