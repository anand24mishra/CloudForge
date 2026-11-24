#!/usr/bin/env bash
set -Eeuo pipefail

go test ./...
go vet ./...
test -z "$(gofmt -l .)"
helm lint charts/cloudforge-app
helm template demo charts/cloudforge-app --namespace cloudforge >/tmp/cloudforge-rendered.yaml
kubectl apply --dry-run=client -f /tmp/cloudforge-rendered.yaml >/dev/null
echo "CloudForge validation passed"
