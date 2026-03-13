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

output "registry_gsa_email" {
  value       = module.registry_gsa.service_account_email
  description = "Email of the Registry GCP Service Account."
}

output "registry_ksa_name" {
  value       = module.registry_ksa.ksa_name
  description = "Name of the Registry Kubernetes Service Account."
}


output "registry_db_connection_name" {
  value = module.registry_database_instance.db_connection_name
  description = "The connection name of the Cloud SQL instance for Registry."
}

output "registry_admin_gsa_email" {
  value       = module.registry_admin_gsa.service_account_email
  description = "Email of the Registry GCP Service Account."
}

output "registry_admin_ksa_name" {
  value = module.registry_admin_ksa.ksa_name
}
