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

# Gateway Kubernetes Service Account
module "gateway_ksa" {
  source      = "../../KUBERNETES_SA"
  ksa_name    = var.gateway_ksa_name
  namespace   = var.app_namespace_name
  annotations = {
    "iam.gke.io/gcp-service-account" = module.gateway_gsa.service_account_email
  }
  depends_on = [module.gateway_gsa]
}

# Gateway GCP Service Account
module "gateway_gsa" {
  source       = "../../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.gateway_gsa_account_id
  display_name = var.gateway_gsa_display_name
  description  = var.gateway_gsa_description
}

# IAM Roles for Gateway GCP Service Account
module "IAM_for_gateway_gsa" {
  source     = "../../IAM_ADMIN/IAM"
  for_each   = toset(var.gateway_gsa_roles)
  project_id = var.project_id
  member_role = each.value
  member     = "serviceAccount:${module.gateway_gsa.service_account_email}"
  depends_on = [module.gateway_gsa]
}

# Workload Identity Binding for Gateway
resource "google_service_account_iam_binding" "gateway_workload_identity" {
  service_account_id = module.gateway_gsa.service_account_id
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.project_id}.svc.id.goog[${var.app_namespace_name}/${var.gateway_ksa_name}]"
  ]
  depends_on = [module.gateway_ksa, module.gateway_gsa]
}
