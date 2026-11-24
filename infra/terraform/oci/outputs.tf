output "public_ip" {
  value = oci_core_instance.cloudforge.public_ip
}

output "ssh_command" {
  value = "ssh opc@${oci_core_instance.cloudforge.public_ip}"
}

output "k3s_kubeconfig_command" {
  value = "sudo cat /etc/rancher/k3s/k3s.yaml"
}
