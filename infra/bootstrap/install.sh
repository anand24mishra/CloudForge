#!/usr/bin/env bash
set -Eeuo pipefail

export KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

helm repo add argo https://argoproj.github.io/argo-helm
helm repo add jetstack https://charts.jetstack.io
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add sealed-secrets https://bitnami-labs.github.io/sealed-secrets
helm repo update

helm upgrade --install argocd argo/argo-cd \
  --namespace argocd --create-namespace \
  --set server.insecure=true \
  --set configs.params."server\.insecure"=true

helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true

helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace \
  --set grafana.service.type=ClusterIP \
  --set prometheus.prometheusSpec.retention=3d \
  --set prometheus.prometheusSpec.resources.requests.cpu=100m \
  --set prometheus.prometheusSpec.resources.requests.memory=256Mi \
  --set prometheus.prometheusSpec.resources.limits.memory=768Mi

helm upgrade --install loki grafana/loki \
  --namespace monitoring \
  --set deploymentMode=SingleBinary \
  --set singleBinary.replicas=1 \
  --set loki.auth_enabled=false \
  --set loki.commonConfig.replication_factor=1 \
  --set backend.replicas=0 \
  --set read.replicas=0 \
  --set write.replicas=0

helm upgrade --install sealed-secrets sealed-secrets/sealed-secrets \
  --namespace kube-system --create-namespace

kubectl apply -f "${ROOT}/cluster-issuer.yaml"
kubectl apply -f "${ROOT}/argocd-root-application.yaml"
kubectl -n argocd rollout status deployment/argocd-server --timeout=10m
echo "CloudForge bootstrap complete"
