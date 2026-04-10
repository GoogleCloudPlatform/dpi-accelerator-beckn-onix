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

output "project_id" {
  value = var.project_id
}

output "project_number" {
  value = data.google_project.project.number
}

output "region" {
  value = var.region
}

# Values for "db_instance" module outputs

output "db_instance_name" {
  value = local.provision_db_instance ? var.db_instance_name : null
}

output "db_instance_connection_name" {
  value = local.provision_db_instance ? module.db_instance[0].db_connection_name : null
}

output "db_instance_private_ip_address" {
  value = local.provision_db_instance ? module.db_instance[0].private_ip_address : null
}

# Values for "redis" module outputs

output "redis_instance_ip" {
  value       = module.redis.redis_instance_ip
  description = "The IP address of the created Redis instance"
}

# Values for "onix" module outputs

output "cluster_name" {
  value = var.enable_onix ? module.onix.cluster_name : null
}

output "app_namespace_name" {
  value = var.enable_onix ? module.onix.app_namespace_name : null
}

output "global_ip_address" {
  value = var.enable_onix ? module.onix.global_ip_address : null
}

output "url_map" {
  value = var.enable_onix ? module.onix.url_map : null
}

output "onix_topic_name" {
  value = var.enable_onix ? module.onix.onix_topic_name : null
}

output "adapter_ksa_name" {
  value = var.enable_onix ? module.onix.adapter_ksa_name : null
}

output "adapter_topic_name" {
  value = var.enable_onix ? module.onix.adapter_topic_name : null
}

output "adapter_topic_full_id" {
  value = var.enable_onix ? module.onix.adapter_topic_full_id : null
}

output "registry_database_name" {
  value = var.enable_onix ? module.onix.registry_database_name : null
}

output "registry_ksa_name" {
  value = var.enable_onix ? module.onix.registry_ksa_name : null
}

output "registry_admin_ksa_name" {
  value = var.enable_onix ? module.onix.registry_admin_ksa_name : null
}

output "database_user_sa_email" {
  value = var.enable_onix ? module.onix.database_user_sa_email : null
}

output "registry_admin_database_user_sa_email" {
  value = var.enable_onix ? module.onix.registry_admin_database_user_sa_email : null
}

output "gateway_ksa_name" {
  value = var.enable_onix ? module.onix.gateway_ksa_name : null
}

output "subscription_ksa_name" {
  value = var.enable_onix ? module.onix.subscription_ksa_name : null
}


# Values for "agent" module outputs

output "agent_db_user" {
  description = "The database user for the Agent"
  value       = var.enable_agent ? module.agent[0].db_user : null
}

output "agent_db_name" {
  description = "The database name for the Agent"
  value       = var.enable_agent ? module.agent[0].db_name : null
}

output "agent_db_secret_name" {
  description = "The Secret Manager secret name for the DB password"
  value       = var.enable_agent ? module.agent[0].db_secret_name : null
}

output "agent_db_password" {
  description = "The database password for the Agent"
  value       = var.enable_agent ? module.agent[0].db_password : null
  sensitive   = true
}

output "agent_app_service_account_email" {
  description = "The email of the Agent service account"
  value       = var.enable_agent ? module.agent[0].agent_app_service_account_email : null
}

output "agent_invoker_service_account_email" {
  description = "The email of the Invoker service account"
  value       = var.enable_agent ? module.agent[0].agent_invoker_service_account_email : null
}

output "agent_network_attachment_id" {
  description = "The ID of the Network Attachment for PSC"
  value       = var.enable_agent ? module.agent[0].agent_network_attachment_id : null
}

output "gcs_bucket" {
  description = "The GCS bucket used for Terraform state and configs"
  value       = "${var.project_id}-dpi-${var.app_name}-bucket"
}

output "agent_datastore_ids" {
  description = "A mapping of keys to Datastore IDs for the agent"
  value       = var.enable_agent ? module.agent[0].datastore_ids : {}
}

output "agent_proxy_internal_ip" {
  description = "The internal IP address of the Proxy VM"
  value       = var.enable_agent ? module.agent[0].agent_proxy_internal_ip : null
}
