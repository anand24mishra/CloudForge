# CloudForge

CloudForge is a small, open-source PaaS for deploying an existing Git repository to Kubernetes. It detects common application stacks, chooses an existing Dockerfile or Cloud Native Buildpacks, generates deployment assets, and provides a GitHub Actions → GHCR → Argo CD GitOps path.

It is intentionally designed as a resume project: the local `kind` workflow is reproducible, while the OCI Terraform and k3s path is ready for a live demo without hiding the operational trade-offs.

## What is implemented

- Go CLI: `init`, `deploy`, `status`, `logs`, `rollback`, and `destroy`.
- Go HTTP API with deployment records, status, and an SSE log endpoint.
- Stack detection for Dockerfile, Node, Python, Go, Java, and an explicit Buildpacks fallback.
- Embedded generators for a Dockerfile, GitHub Actions workflow, and Helm values.
- Generic Helm chart with Deployment, Service, Ingress, probes, resource limits, and HPA.
- Example Node, Flask, and Go applications, including a Go app that demonstrates the Buildpacks path.
- CI and release workflows with tests, multi-architecture builds, Trivy, GHCR, and GoReleaser.
- Argo CD app-of-apps and deploy-config layout.
- Terraform scaffolding for a local kind cluster and an Oracle Cloud Always Free A1 k3s VM.
- Bootstrap manifests for Argo CD, cert-manager, kube-prometheus-stack, Loki, and Sealed Secrets.

The project never pushes to GitHub automatically. `cloudforge init` writes to a local working copy. The generated CI workflow only becomes active after you commit it and configure the relevant GitHub secrets and repository permissions yourself.

## Quick start on kind

Prerequisites: Go, Docker, kind, kubectl, and Helm.

```sh
make kind-up
make build
./bin/cloudforge init ./examples/node-app --output-dir /tmp/cloudforge-node
helm upgrade --install node-app charts/cloudforge-app \
  --namespace cloudforge --create-namespace \
  --set nameOverride=node-app \
  --set fullnameOverride=node-app \
  --set image.repository=cloudforge-node \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent \
  --set service.port=8080 \
  --set containerPort=8080
```

For a full local image flow:

```sh
docker build -t cloudforge-node:local examples/node-app
kind load docker-image cloudforge-node:local --name cloudforge
helm upgrade --install node-app charts/cloudforge-app \
  --namespace cloudforge --create-namespace \
  --set nameOverride=node-app --set fullnameOverride=node-app \
  --set image.repository=cloudforge-node --set image.tag=local \
  --set image.pullPolicy=IfNotPresent --set service.port=8080 --set containerPort=8080
kubectl -n cloudforge port-forward svc/node-app 8080:8080
curl http://127.0.0.1:8080/healthz
```

The generated files can be inspected before they are copied into an application repository:

```sh
./bin/cloudforge detect ./examples/node-app
./bin/cloudforge init ./examples/node-app --output-dir ./generated/node-app
./bin/cloudforge deploy ./examples/go-app --buildpacks --context kind-cloudforge
```

## CLI

```text
cloudforge init <repo>       Detect and generate deployment assets locally
cloudforge deploy <repo>     Build/load and deploy to the current Kubernetes context
cloudforge status [app]      Show Deployment and pod health
cloudforge logs <app>        Stream application logs
cloudforge rollback <app>    Roll back the Deployment to its previous revision
cloudforge destroy <app>     Delete the Helm release from the current cluster
```

`repo` may be a local path or a Git URL. Git URLs are cloned into a temporary directory and are never pushed by the CLI.

## GitOps path

1. Generated workflow runs tests and lint, builds `linux/amd64` and `linux/arm64`, scans with Trivy, and publishes to GHCR.
2. The workflow updates an image tag in a separately managed `deploy-config` repository when `DEPLOY_CONFIG_REPO` and its token are configured.
3. Argo CD watches `deploy-config/apps/` and syncs the generic chart into the cluster.

The deploy-config repository is represented locally under [`deploy-config/`](deploy-config/) as an example layout. Keep real Sealed Secrets and credentials outside this repository.

## Oracle Cloud demo path

The OCI files stay un-applied until you explicitly run Terraform. Review [`infra/terraform/oci/variables.tf`](infra/terraform/oci/variables.tf), configure a remote state bucket, and set a budget alert in the OCI console first. The included cloud-init keeps the A1 shape at 2 OCPUs / 12 GB by default and installs k3s plus the bootstrap script.

```sh
cd infra/terraform/oci
terraform init
terraform plan
# Apply only after reviewing the plan and confirming the tenancy/budget settings.
terraform apply
```

See [`docs/runbook.md`](docs/runbook.md) for the kind and OCI runbooks, and [`docs/architecture.md`](docs/architecture.md) for the system design.

## Safety and cost notes

- No GitHub push, PR, tag, or release is performed by this local project.
- No cloud resource is created by `make`, the CLI, or tests.
- Never commit `.env`, kubeconfigs, OCI keys, or plain Kubernetes Secrets.
- Oracle Always Free capacity and quotas can change; verify the current OCI Free Tier page and set a budget alert before provisioning.
