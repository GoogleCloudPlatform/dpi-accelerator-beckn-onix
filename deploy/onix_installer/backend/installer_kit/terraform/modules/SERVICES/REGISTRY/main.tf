# Copyright 2025 Google LLC
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

# PSQL DB instance
module "registry_database_instance" {
  source             = "../../CLOUD_SQL/DATABASE_INSTANCE"
  db_instance_region = var.registry_db_instance_region
  db_instance_version = var.registry_db_instance_version
  db_instance_name   = var.registry_db_instance_name
  db_instance_tier   = var.registry_db_instance_tier
  db_instance_labels = var.registry_db_instance_labels
  db_instance_edition = var.registry_db_instance_edition
  db_aval_type       = var.registry_db_aval_type
  db_instance_disk_size = var.registry_db_instance_disk_size
  db_instance_disk_type = var.registry_db_instance_disk_type
  db_ipv4            = var.registry_db_ipv4
  db_instance_max_connections = var.registry_db_max_connections
  db_instance_cache = var.registry_db_instance_cache
  instance_network   = "projects/${var.project_id}/global/networks/${var.network_name}"
  depends_on = [
    module.registry_gsa
  ]
}

module "registry_database" {
  source        = "../../CLOUD_SQL/DATABASE"
  database_name = var.registry_database_name
  instance_name = var.registry_db_instance_name
  depends_on    = [module.registry_database_instance]
}

# Registry GCP Service Account
module "registry_gsa" {
  source       = "../../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.registry_gsa_account_id
  display_name = var.registry_gsa_display_name
  description  = var.registry_gsa_description
}

# IAM Roles for Registry GCP Service Account
module "IAM_for_registry_gsa" {
  source     = "../../IAM_ADMIN/IAM"
  for_each   = toset(var.registry_gsa_roles)
  project_id = var.project_id
  member_role = each.value
  member     = "serviceAccount:${module.registry_gsa.service_account_email}"
  depends_on = [module.registry_gsa]
}

# Registry Kubernetes Service Account
module "registry_ksa" {
  source      = "../../KUBERNETES_SA"
  ksa_name    = var.registry_ksa_name
  namespace   = var.app_namespace_name
  annotations = {
    "iam.gke.io/gcp-service-account" = module.registry_gsa.service_account_email
  }
  depends_on = [module.registry_gsa]
}

# Workload Identity Binding for Registry
resource "google_service_account_iam_binding" "registry_workload_identity_db" {
  service_account_id = module.registry_gsa.service_account_id
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[${var.app_namespace_name}/${var.registry_ksa_name}]"
  ]
  depends_on = [module.registry_ksa, module.registry_gsa]
}

# Cloud SQL DB User for Registry (using Cloud IAM for KSA authentication)
module "registry_db_user" {
  source        = "../../CLOUD_SQL/DB_USER"
  user_name     = "${var.registry_gsa_account_id}@${var.project_id}.iam"
  instance_name = var.registry_db_instance_name
  user_type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  depends_on    = [module.registry_database_instance, module.registry_gsa, module.registry_database]
}

module "registry_admin_gsa" {
  source       = "../../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.registry_admin_gsa_account_id
  display_name = var.registry_admin_gsa_display_name
  description  = var.registry_admin_gsa_description
}

module "IAM_for_registry_admin_gsa" {
  source      = "../../IAM_ADMIN/IAM"
  for_each    = toset(var.registry_admin_gsa_roles)
  project_id  = var.project_id
  member_role = each.value
  member      = "serviceAccount:${module.registry_admin_gsa.service_account_email}"
  depends_on  = [module.registry_admin_gsa]
}

module "registry_admin_ksa" {
  source      = "../../KUBERNETES_SA"
  ksa_name    = var.registry_admin_ksa_name
  namespace   = var.app_namespace_name
  annotations = {
    "iam.gke.io/gcp-service-account" = module.registry_admin_gsa.service_account_email
  }
  depends_on = [module.registry_admin_gsa]
}

resource "google_service_account_iam_binding" "registry_admin_workload_identity_db" {
  service_account_id = module.registry_admin_gsa.service_account_id
  role               = "roles/iam.workloadIdentityUser"

  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[${var.app_namespace_name}/${var.registry_admin_ksa_name}]"
  ]
  depends_on = [module.registry_admin_ksa, module.registry_admin_gsa]
}

module "registry_admin_db_user" {
  source        = "../../CLOUD_SQL/DB_USER"
  user_name     = "${var.registry_admin_gsa_account_id}@${var.project_id}.iam"
  instance_name = var.registry_db_instance_name
  user_type     = "CLOUD_IAM_SERVICE_ACCOUNT"
  depends_on    = [
    module.registry_database_instance,
    module.registry_admin_gsa,
    module.registry_database
  ]
}

