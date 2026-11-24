#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-cloudforge}"
APP="${APP:-node-app}"
SERVICE="${SERVICE:-node-app}"

kubectl -n "${NAMESPACE}" rollout status "deployment/${APP}" --timeout=180s
kubectl -n "${NAMESPACE}" port-forward "service/${SERVICE}" 18080:8080 >/tmp/cloudforge-port-forward.log 2>&1 &
FORWARD_PID=$!
trap 'kill "${FORWARD_PID}" 2>/dev/null || true' EXIT
for _ in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:18080/healthz >/dev/null; then
    curl -fsS http://127.0.0.1:18080/
    exit 0
  fi
  sleep 1
done
cat /tmp/cloudforge-port-forward.log
exit 1
