# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

data "google_compute_zones" "available" {
  project = var.project_id
  region  = var.region
}

locals {
  proxy_vm_sa_roles = [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
  ]
}

# Create the Service Account
resource "google_service_account" "proxy_vm_sa" {
  account_id   = "${var.app_name}-proxy-vm-sa"
  display_name = "Proxy VM Service Account"
  project      = var.project_id
}

# Bind the roles to the SA
resource "google_project_iam_member" "proxy_role_bindings" {
  for_each = toset(local.proxy_vm_sa_roles)
  project  = var.project_id
  role     = each.value
  member   = "serviceAccount:${google_service_account.proxy_vm_sa.email}"
}

resource "google_compute_instance" "proxy_vm" {
  name         = "${var.app_name}-proxy-vm"
  machine_type = "e2-micro"
  zone         = data.google_compute_zones.available.names[0]
  project      = var.project_id

  allow_stopping_for_update = true

  tags = ["${var.app_name}-agent-proxy"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = var.subnetwork_id
  }

  service_account {
    email  = google_service_account.proxy_vm_sa.email
    scopes = ["cloud-platform"]
  }

  metadata_startup_script = <<-EOF
    #!/bin/bash
    set -e
    sudo apt-get update
    sudo DEBIAN_FRONTEND=noninteractive apt-get install tinyproxy -y
    sudo sed -i 's/^Listen 127.0.0.1/#Listen 127.0.0.1/' /etc/tinyproxy/tinyproxy.conf
    sudo sed -i 's/^Allow 127.0.0.1/#Allow 127.0.0.1\nAllow 10.0.0.0\/8\nAllow 172.16.0.0\/12\nAllow 192.168.0.0\/16/' /etc/tinyproxy/tinyproxy.conf
    sudo systemctl restart tinyproxy
  EOF
}

resource "google_compute_firewall" "proxy_fw" {
  name    = "${var.app_name}-allow-proxy"
  network = var.network_id
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["8888"]
  }

  source_ranges = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]
  target_tags   = ["${var.app_name}-agent-proxy"]
}
