# CloudForge

CloudForge is a small Go-based PaaS that turns a Git repository into a Kubernetes deployment. It detects the application stack, builds a container with Docker or Cloud Native Buildpacks, and deploys through Helm. The production path uses GitHub Actions, GHCR, and Argo CD.

## Highlights

- Stack detection for Dockerfile, Node.js, Python, Go, Java, and Buildpacks fallback
- One generic Helm chart with probes, resource limits, HPA, ingress, and rollback
- Multi-architecture CI images with Trivy scanning and GHCR publishing
- Terraform + cloud-init for a k3s cluster on an Oracle Cloud VM
- Prometheus, Grafana, Loki, cert-manager, and Sealed Secrets bootstrap
- Local `kind` workflow for a reproducible demo without cloud credentials

## Quick start

Requirements: Go, Docker, kind, kubectl, and Helm.

```sh
make kind-up
make build
./bin/cloudforge deploy examples/node-app --context kind-cloudforge
APP=node-app SERVICE=node-app ./scripts/smoke-test.sh
```

Inspect the generated deployment files without changing an app repository:

```sh
./bin/cloudforge detect examples/node-app
./bin/cloudforge init examples/node-app --output-dir /tmp/cloudforge-node
```

Useful commands:

```text
cloudforge init <repo>       Generate Dockerfile, workflow, and Helm values
cloudforge deploy <repo>     Build and deploy with Helm
cloudforge status <app>      Show Deployment and pod health
cloudforge logs <app>        Stream application logs
cloudforge rollback <app>    Restore the previous Deployment revision
cloudforge destroy <app>     Remove the Helm release
```

## Architecture

```text
Git repo → CloudForge detector/generator → GitHub Actions → GHCR
                                                    ↓
                                      deploy-config → Argo CD → Helm
                                                                  ↓
                                                   k3s/kind → Traefik → app
```

The implementation is organized around the CLI/API, detector and generators, generic chart, examples, GitOps config, and infrastructure modules. See [`docs/architecture.md`](docs/architecture.md) and [`docs/runbook.md`](docs/runbook.md) for the full design and demo steps.

## Safety

CloudForge never pushes to GitHub from the CLI or API. `init` writes generated files locally; the generated deploy-config push is an explicit GitHub Actions opt-in that requires repository secrets. No credentials or plain Kubernetes Secrets belong in this repository.

The OCI module is plan-first. Review the current Oracle Always Free limits, set a budget alert, and inspect `terraform plan` before applying it.

## License

MIT
