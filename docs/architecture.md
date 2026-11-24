# CloudForge architecture

```text
Developer
  │ cloudforge deploy <repo> / REST API
  ▼
CloudForge API (Go)
  ├─ detect stack: Dockerfile, Node, Python, Go, Java, Buildpacks fallback
  ├─ generate local Dockerfile, workflow, and Helm values
  └─ inspect Kubernetes status/logs or request rollback

GitHub Actions
  ├─ test + lint
  ├─ docker/buildx or pack build
  ├─ Trivy scan
  └─ push multi-arch image to GHCR
        │ optional, explicitly configured deploy-config token
        ▼
deploy-config Git repository
        │ watched by Argo CD
        ▼
Helm chart → Deployment → Service → Traefik Ingress → app.<domain>

Terraform → OCI VCN/subnet/A1 VM → cloud-init → k3s
                                      ├─ Argo CD
                                      ├─ cert-manager / Let's Encrypt
                                      ├─ kube-prometheus-stack
                                      ├─ Loki
                                      └─ Sealed Secrets
```

## Design choices

- The generic chart is the stable contract between CI and the cluster. App-specific behavior lives in values rather than copied charts.
- GitHub Actions owns build and image security. Argo CD owns cluster reconciliation. Neither needs the other system's long-lived cluster credentials.
- The CLI uses client-go for read paths and a narrowly scoped `kubectl rollout undo` command for rollback because Kubernetes stores rollout revisions as controller-managed ReplicaSets.
- The local path works without a cloud account. OCI provisioning is isolated under `infra/terraform/oci` and is never run by tests or Make targets.
- Every generated file is reviewable. The local generator intentionally does not open a PR or push to a remote repository.
