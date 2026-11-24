# deploy-config example

This directory models the separate GitOps repository Argo CD watches. In a real demo, copy it to a private or public repository and configure the generated workflow with an explicit token. CloudForge does not create that repository or push to it.

`apps/` contains one Argo Application per deployed app. Each application points at the generic CloudForge Helm chart and carries only non-secret values. Use Sealed Secrets for credentials.
