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

# Outputs for the AGENT module
output "agent_app_service_account_email" {
  description = "The email of the Agent service account"
  value       = module.agent_service_account.service_account_email
}

output "agent_invoker_service_account_email" {
  description = "The email of the Invoker service account"
  value       = module.agent_invoker_service_account.service_account_email
}

output "db_user" {
  description = "The database user for the Agent"
  value       = var.provision_agent_db ? var.agent_db_user : null
}

output "db_name" {
  description = "The database name for the Agent"
  value       = var.provision_agent_db ? var.agent_db_name : null
}

output "db_secret_name" {
  description = "The Secret Manager secret name for the DB password"
  value       = one(google_secret_manager_secret.db_password_secret[*].name)
}

output "db_password" {
  description = "The database password for the Agent"
  value       = one(random_password.db_password[*].result)
  sensitive   = true
}

output "agent_network_attachment_id" {
  value       = module.network_attachment.network_attachment_id
  description = "The ID of the Network Attachment for PSC"
}

output "datastore_ids" {
  value       = { for k, v in module.discovery_engine : k => v.datastore_id }
  description = "A mapping of keys to Datastore IDs for the agent"
}

output "agent_proxy_internal_ip" {
  value       = module.proxy_vm.network_ip
  description = "The internal IP address of the Proxy VM"
}
