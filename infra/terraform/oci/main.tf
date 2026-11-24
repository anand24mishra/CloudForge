data "oci_identity_availability_domains" "available" {
  compartment_id = var.tenancy_ocid
}

locals {
  availability_domain = var.availability_domain != "" ? var.availability_domain : data.oci_identity_availability_domains.available.availability_domains[0].name
}

resource "oci_core_vcn" "cloudforge" {
  compartment_id = var.compartment_ocid
  display_name   = "cloudforge-vcn"
  cidr_blocks    = ["10.0.0.0/16"]
  dns_label      = "cloudforge"
}

resource "oci_core_internet_gateway" "cloudforge" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.cloudforge.id
  display_name   = "cloudforge-igw"
}

resource "oci_core_route_table" "cloudforge" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.cloudforge.id
  display_name   = "cloudforge-public-routes"
  route_rules {
    destination       = "0.0.0.0/0"
    destination_type  = "CIDR_BLOCK"
    network_entity_id = oci_core_internet_gateway.cloudforge.id
  }
}

resource "oci_core_security_list" "cloudforge" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_vcn.cloudforge.id
  display_name   = "cloudforge-security"

  egress_security_rules {
    protocol    = "all"
    destination = "0.0.0.0/0"
  }
  ingress_security_rules {
    protocol = "6"
    source   = var.ssh_ingress_cidr
    tcp_options {
      min = 22
      max = 22
    }
  }
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 80
      max = 80
    }
  }
  ingress_security_rules {
    protocol = "6"
    source   = "0.0.0.0/0"
    tcp_options {
      min = 443
      max = 443
    }
  }
}

resource "oci_core_subnet" "cloudforge" {
  compartment_id             = var.compartment_ocid
  vcn_id                     = oci_core_vcn.cloudforge.id
  display_name               = "cloudforge-public-subnet"
  cidr_block                 = "10.0.1.0/24"
  route_table_id             = oci_core_route_table.cloudforge.id
  security_list_ids          = [oci_core_security_list.cloudforge.id]
  dhcp_options_id            = oci_core_vcn.cloudforge.default_dhcp_options_id
  dns_label                  = "public"
  prohibit_public_ip_on_vnic = false
}

resource "oci_core_instance" "cloudforge" {
  availability_domain = local.availability_domain
  compartment_id      = var.compartment_ocid
  display_name        = "cloudforge-k3s"
  shape               = "VM.Standard.A1.Flex"

  shape_config {
    ocpus         = var.ocpus
    memory_in_gbs = var.memory_in_gbs
  }

  source_details {
    source_type             = "image"
    source_id               = var.oracle_linux_image_id
    boot_volume_size_in_gbs = 50
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.cloudforge.id
    assign_public_ip = true
    hostname_label   = "cloudforge"
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
    user_data           = base64encode(templatefile("${path.module}/../../cloud-init/k3s.yaml", { k3s_token = var.k3s_token, cloudforge_repo_url = var.cloudforge_repo_url }))
  }
}
