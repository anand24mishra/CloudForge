variable "cluster_name" {
  type    = string
  default = "cloudforge"
}

resource "kind_cluster" "cloudforge" {
  name            = var.cluster_name
  wait_for_ready  = true
  node_image      = "kindest/node:v1.35.0"
  kubeconfig_path = "${path.root}/kubeconfig"

  kind_config {
    kind        = "Cluster"
    api_version = "kind.x-k8s.io/v1alpha4"

    node {
      role = "control-plane"
      extra_port_mappings {
        container_port = 80
        host_port      = 8080
      }
      extra_port_mappings {
        container_port = 443
        host_port      = 8443
      }
    }
    node {
      role = "worker"
    }
  }
}
