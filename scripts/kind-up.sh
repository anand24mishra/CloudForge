#!/usr/bin/env bash
set -Eeuo pipefail

CLUSTER="${KIND_CLUSTER:-cloudforge}"

if kind get clusters | grep -qx "${CLUSTER}"; then
  kubectl config use-context "kind-${CLUSTER}" >/dev/null
  echo "kind cluster ${CLUSTER} already exists"
  exit 0
fi

kind create cluster --name "${CLUSTER}" --config=- <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
- role: worker
EOF

kubectl wait --for=condition=Ready nodes --all --timeout=180s
kubectl create namespace cloudforge --dry-run=client -o yaml | kubectl apply -f -
echo "kind cluster ${CLUSTER} is ready"
