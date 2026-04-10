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


#--------------------------------------------- Secret Management ---------------------------------------------#

resource "random_password" "db_password" {
  count   = var.provision_agent_db ? 1 : 0
  length  = 16
  special = false
}

resource "google_secret_manager_secret" "db_password_secret" {
  count     = var.provision_agent_db ? 1 : 0
  secret_id = "${var.app_name}-agent-session-db-password"
  replication {
    auto {}
  }
  project = var.project_id
}

resource "google_secret_manager_secret_version" "db_password_secret_version" {
  count       = var.provision_agent_db ? 1 : 0
  secret      = google_secret_manager_secret.db_password_secret[0].id
  secret_data = random_password.db_password[0].result
}

#--------------------------------------------- IAM Configuration ---------------------------------------------#

module "agent_service_account" {
  source       = "../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.agent_sa_account_id
  display_name = var.agent_sa_display_name
  description  = var.agent_sa_description
}

module "agent_iam_binding" {
  source      = "../IAM_ADMIN/IAM"
  for_each    = toset(var.agent_sa_roles)
  project_id  = var.project_id
  member_role = each.value
  member      = "serviceAccount:${module.agent_service_account.service_account_email}"
  depends_on  = [module.agent_service_account]
}

module "agent_invoker_service_account" {
  source       = "../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.agent_invoker_sa_account_id
  display_name = "Agent Invoker SA - ${var.app_name}"
  description  = "Service Account for invoking the Vertex AI Agent Engine for app ${var.app_name}"
}

module "agent_invoker_iam_binding" {
  source      = "../IAM_ADMIN/IAM"
  project_id  = var.project_id
  member_role = "roles/aiplatform.user"
  member      = "serviceAccount:${module.agent_invoker_service_account.service_account_email}"
  depends_on  = [module.agent_invoker_service_account]
}

# Grant the agent service account access to the secret
resource "google_secret_manager_secret_iam_member" "agent_secret_access" {
  count     = var.provision_agent_db ? 1 : 0
  secret_id = google_secret_manager_secret.db_password_secret[0].id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${module.agent_service_account.service_account_email}"
}

# Google managed service account for Vertex AI
resource "google_project_service_identity" "ai_platform_identity" {
  provider = google-beta
  project  = var.project_id
  service  = "aiplatform.googleapis.com"
}

resource "time_sleep" "wait_for_ai_platform_service_identity" {
  depends_on      = [google_project_service_identity.ai_platform_identity]
  create_duration = "30s"
}

resource "google_project_iam_member" "vertex_permissions" {
  project = var.project_id
  role    = "roles/compute.networkAdmin"
  member  = "serviceAccount:${google_project_service_identity.ai_platform_identity.email}"
  depends_on = [time_sleep.wait_for_ai_platform_service_identity]
}

resource "time_sleep" "wait_for_iam_propagation" {
  depends_on      = [google_project_iam_member.vertex_permissions]
  create_duration = "40s"
}

#--------------------------------------------- Database Configuration ---------------------------------------------#

module "agent_database" {
  count         = var.provision_agent_db ? 1 : 0
  source        = "../CLOUD_SQL/DATABASE"
  database_name = var.agent_db_name
  instance_name = var.db_instance_name
}

module "agent_db_user" {
  count         = var.provision_agent_db ? 1 : 0
  source        = "../CLOUD_SQL/DB_USER"
  user_name     = var.agent_db_user
  password      = random_password.db_password[0].result
  instance_name = var.db_instance_name

  depends_on = [module.agent_database]
}

#--------------------------------------------- Component Modules ---------------------------------------------#

# NETWORK_ATTACHMENT submodule for PSC
module "network_attachment" {
  source                  = "../VPC/NETWORK_ATTACHMENT"
  network_attachment_name = var.agent_network_attachment_name
  region                  = var.region
  subnetworks             = [var.subnetwork_id]
}

# Proxy VM for outbound internet access
module "proxy_vm" {
  source        = "../COMPUTE_ENGINE/PROXY_VM"
  project_id    = var.project_id
  region        = var.region
  subnetwork_id = var.subnetwork_id
  app_name      = var.app_name
  network_id    = var.network_id
}

#--------------------------------------------- Datastore Configuration ---------------------------------------------#

module "discovery_engine" {
  source   = "../DISCOVERY_ENGINE"
  for_each = var.datastore_imports != null ? var.datastore_imports : {}

  # Replace dots with hyphens for the data_store_id (e.g., "agri.biochar_advice" -> "agri-biochar_advice")
  data_store_id = "${var.app_name}-${replace(lower(each.key), ".", "-")}-ds"
  display_name  = "${each.key} Knowledge Base - ${var.app_name}"
}
