output "kubeconfig" {
  value     = kind_cluster.cloudforge.kubeconfig
  sensitive = true
}

output "cluster_name" {
  value = kind_cluster.cloudforge.name
}
