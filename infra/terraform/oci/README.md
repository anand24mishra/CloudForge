# OCI k3s infrastructure

This is intentionally a plan-first module. The image OCID is region-specific and must be looked up in the OCI console or image data source before use. Copy `terraform.tfvars.example` to a local `.tfvars` file, restrict SSH to your IP, configure the OCI provider credentials, and run `terraform plan` before any apply.

For a production-quality demo, configure an OCI Object Storage backend separately. Terraform backend configuration cannot safely interpolate normal variables, so keep the bucket/namespace/object names in a local backend config or CI secret rather than committing them.

The default A1 shape values are 2 OCPUs and 12 GB RAM. Verify the current Oracle Always Free limits and account availability before applying.
