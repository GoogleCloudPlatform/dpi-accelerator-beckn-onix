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

output "cluster_name" {
  value = var.enable_onix ? module.gke[0].cluster_name : null
}

output "app_namespace_name" {
  value = var.enable_onix ? module.app_namespace[0].namespace_name : null
}

output "global_ip_address" {
  value = var.enable_onix ? module.lb_global_ip[0].global_ip_address : null
}

output "url_map" {
  value = var.enable_onix ? module.url_map[0].url_map : null
}

output "onix_topic_name" {
  value = var.enable_onix ? var.pubsub_topic_onix_name : null
}

output "adapter_ksa_name" {
  value = var.enable_onix && var.provision_adapter_infra ? module.adapter_service[0].adapter_ksa_name : null
}

output "adapter_topic_name" {
  value = var.enable_onix && var.provision_adapter_infra ? module.adapter_service[0].adapter_topic_name : null
}

output "adapter_topic_full_id" {
  value = var.enable_onix && var.provision_adapter_infra ? module.adapter_service[0].adapter_topic_full_id : null
}

output "registry_database_name" {
  value = var.enable_onix && var.provision_registry_infra ? var.registry_database_name : null
}

output "registry_ksa_name" {
  value = var.enable_onix && var.provision_registry_infra ? module.registry_service[0].registry_ksa_name : null
}

output "registry_admin_ksa_name" {
  value = var.enable_onix && var.provision_registry_infra ? module.registry_service[0].registry_admin_ksa_name : null
}

output "database_user_sa_email" {
  value = var.enable_onix && var.provision_registry_infra ? module.registry_service[0].registry_gsa_email : null
}

output "registry_admin_database_user_sa_email" {
  value = var.enable_onix && var.provision_registry_infra ? module.registry_service[0].registry_admin_gsa_email : null
}

output "gateway_ksa_name" {
  value = var.enable_onix && var.provision_gateway_infra ? module.gateway_service[0].gateway_ksa_name : null
}

output "subscription_ksa_name" {
  value = var.enable_onix && local.provision_subscription_infra ? module.subscription_service[0].subscription_ksa_name : null
}
