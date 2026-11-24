# CloudForge runbook

## Local kind demo

```sh
make kind-up
make build
docker build -t cloudforge/node-app:local examples/node-app
kind load docker-image cloudforge/node-app:local --name cloudforge
./bin/cloudforge deploy examples/node-app --context kind-cloudforge
./bin/cloudforge deploy examples/go-app --buildpacks --context kind-cloudforge
APP=node-app SERVICE=node-app ./scripts/smoke-test.sh
./bin/cloudforge status node-app --context kind-cloudforge
./bin/cloudforge logs node-app --context kind-cloudforge --follow=false
./bin/cloudforge rollback node-app --context kind-cloudforge
```

If you want to inspect the generated assets without changing the example:

```sh
./bin/cloudforge init examples/node-app --output-dir /tmp/cloudforge-generated-node
```

## Argo CD on kind

```sh
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update
helm upgrade --install argocd argo/argo-cd --namespace argocd --create-namespace
kubectl -n argocd port-forward svc/argocd-server 8081:443
```

The app-of-apps manifests contain placeholder repository URLs. Replace them in your own deploy-config repository before applying.

## OCI k3s demo

1. Create an OCI budget alert and confirm the current Always Free quota.
2. Copy `infra/terraform/oci/terraform.tfvars.example` to an ignored local tfvars file and fill in the region-specific ARM image OCID.
3. Run `terraform init` and `terraform plan` from `infra/terraform/oci`.
4. Review network ingress, A1 shape limits, public IP, boot volume, and the cloud-init repository URL.
5. Apply only after the plan is understood. Retrieve the k3s kubeconfig over SSH, replace `127.0.0.1` with the VM public IP, and protect the file locally.
6. Update the deploy-config repository URL and the cert-manager email before production-like use.

## Failure modes

- `ImagePullBackOff`: check GHCR visibility, image tag, and `imagePullSecrets`.
- `CrashLoopBackOff`: inspect `cloudforge logs`, then check the generated port and health path.
- Argo OutOfSync: validate the deploy-config path and the image tag, then inspect `argocd app get`.
- Let's Encrypt pending: DNS or port 80 is not reachable; use `sslip.io` only when the public IP resolves correctly.
- HPA has no metrics: install metrics-server or disable autoscaling for the local demo.
- OCI capacity error: A1 capacity is regional and can be unavailable; do not repeatedly apply without checking the quota.
