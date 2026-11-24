variable "tenancy_ocid" {
  type        = string
  description = "OCI tenancy OCID"
}

variable "user_ocid" {
  type        = string
  description = "OCI API user OCID"
}

variable "fingerprint" {
  type        = string
  description = "OCI API key fingerprint"
}

variable "private_key_path" {
  type        = string
  description = "Path to the OCI API private key"
}

variable "region" {
  type        = string
  description = "OCI region"
}

variable "compartment_ocid" {
  type        = string
  description = "Compartment for CloudForge resources"
}

variable "availability_domain" {
  type        = string
  default     = ""
  description = "Optional availability domain; the first AD is used when empty"
}

variable "oracle_linux_image_id" {
  type        = string
  description = "Oracle Linux ARM image OCID for the selected region"
}

variable "ssh_public_key" {
  type      = string
  sensitive = true
}

variable "ssh_ingress_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "Restrict to your IP in a real deployment"
}

variable "ocpus" {
  type        = number
  default     = 2
  description = "Keep within the current Always Free A1 allowance"
}

variable "memory_in_gbs" {
  type        = number
  default     = 12
  description = "Keep within the current Always Free A1 allowance"
}

variable "k3s_token" {
  type        = string
  sensitive   = true
  description = "Random long-lived k3s server token"
}

variable "cloudforge_repo_url" {
  type        = string
  default     = "https://github.com/OWNER/CloudForge.git"
  description = "Public URL used by cloud-init to fetch the bootstrap files"
}
