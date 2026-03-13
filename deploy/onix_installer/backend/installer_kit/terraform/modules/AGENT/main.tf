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
  length  = 16
  special = false
}

resource "google_secret_manager_secret" "db_password_secret" {
  secret_id = "${var.app_name}-agent-session-db-password"
  replication {
    auto {}
  }
  project = var.project_id
}

resource "google_secret_manager_secret_version" "db_password_secret_version" {
  secret      = google_secret_manager_secret.db_password_secret.id
  secret_data = random_password.db_password.result
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

# Grant the agent service account access to the secret
resource "google_secret_manager_secret_iam_member" "agent_secret_access" {
  secret_id = google_secret_manager_secret.db_password_secret.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${module.agent_service_account.service_account_email}"
}

#--------------------------------------------- Database Configuration ---------------------------------------------#

module "agent_database" {
  source        = "../CLOUD_SQL/DATABASE"
  database_name      = var.agent_db_name
  instance_name = var.db_instance_name
}

module "agent_db_user" {
  source        = "../CLOUD_SQL/DB_USER"
  user_name     = var.agent_db_user
  password = random_password.db_password.result
  instance_name = var.db_instance_name

  depends_on = [module.agent_database]
}

#--------------------------------------------- Component Modules ---------------------------------------------#

module "vertex_ai" {
  source   = "../VERTEX_AI"
  region   = var.region
  app_name = var.app_name
}

module "cloud_run" {
  source                = "../CLOUD_RUN"
  region                = var.region
  service_name          = "${var.app_name}-agent-app"
  image_url             = var.image_url
  network_name          = var.network_name
  subnet_name           = var.subnet_name
  service_account_email = module.agent_service_account.service_account_email

  # Compute and Networking
  cpu                   = var.cpu
  memory                = var.memory
  ingress               = var.ingress
  vpc_egress            = var.vpc_egress
  allow_unauthenticated = var.allow_unauthenticated
}
