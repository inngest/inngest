terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

# Este código é compatível com a versão 4.25.0 do Terraform e com as que têm compatibilidade com versões anteriores à 4.25.0.
# Para informações sobre como validar esse código do Terraform, consulte https://developer.hashicorp.com/terraform/tutorials/gcp-get-started/google-cloud-platform-build#format-and-validate-the-configuration

resource "google_compute_instance" "evolution_api" {
  boot_disk {
    auto_delete = true
    device_name = "evolution-api"

    initialize_params {
      image = "projects/ubuntu-os-cloud/global/images/ubuntu-2204-jammy-v20241218"
      size  = 15
      type  = "pd-ssd"
      resource_policies = ["projects/${var.project_id}/regions/${var.region}/resourcePolicies/weekly-snapshot-policy"]
    }

    mode = "READ_WRITE"
  }
  
  can_ip_forward      = false
  deletion_protection = false
  enable_display      = false

  labels = {
    goog-ec-src = "vm_add-tf"
  }

  machine_type = "e2-small"
  name         = "evolution-api"

  network_interface {
    access_config {
      network_tier = "PREMIUM"
    }

    queue_count = 0
    stack_type  = "IPV4_ONLY"
    subnetwork  = "projects/${var.project_id}/regions/${var.region}/subnetworks/vpc-${var.project_id}-${var.region}"
  }

  scheduling {
    automatic_restart   = true
    on_host_maintenance = "MIGRATE"
    preemptible         = false
    provisioning_model  = "STANDARD"
  }

  service_account {
    email  = var.service_account
    scopes = ["https://www.googleapis.com/auth/devstorage.read_only", "https://www.googleapis.com/auth/logging.write", "https://www.googleapis.com/auth/monitoring.write", "https://www.googleapis.com/auth/service.management.readonly", "https://www.googleapis.com/auth/servicecontrol", "https://www.googleapis.com/auth/trace.append"]
  }

  shielded_instance_config {
    enable_integrity_monitoring = true
    enable_secure_boot          = false
    enable_vtpm                 = true
  }

  metadata = {
    ssh-keys = "${var.ssh_user}:${file(var.ssh_public_key_path)}"
  }

  tags = ["http-server", "https-server"]
  zone = var.zone
}

# Cria o arquivo inventory do Ansible automaticamente
resource "local_file" "ansible_inventory" {
  filename = "${path.module}/../../ansible/inventory"
  content  = <<-EOT
[servers]
${google_compute_instance.n8n.network_interface[0].access_config[0].nat_ip} ansible_user=${var.ssh_user} ansible_ssh_private_key_file=${var.ssh_private_key_path}
EOT
}

output "vm_external_ip" {
  value = google_compute_instance.n8n.network_interface[0].access_config[0].nat_ip
}
