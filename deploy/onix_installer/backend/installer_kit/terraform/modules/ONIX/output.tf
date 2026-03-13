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

output "cluster_name" {
  value = module.gke.cluster_name
}

output "app_namespace_name" {
  value = module.app_namespace.namespace_name
}

output "global_ip_address" {
  value = module.lb_global_ip.global_ip_address
}

output "url_map" {
  value = module.url_map.url_map
}

output "onix_topic_name" {
  value = var.pubsub_topic_onix_name
}

output "adapter_ksa_name" {
  value = var.provision_adapter_infra ? module.adapter_service[0].adapter_ksa_name : null
}

output "adapter_topic_name" {
  value = var.provision_adapter_infra ? module.adapter_service[0].adapter_topic_name : null
}

output "registry_database_name" {
  value = var.provision_registry_infra ? var.registry_database_name : null
}

output "registry_ksa_name" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_ksa_name : null
}

output "registry_admin_ksa_name" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_admin_ksa_name : null
}

output "database_user_sa_email" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_gsa_email : null
}

output "registry_admin_database_user_sa_email" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_admin_gsa_email : null
}

output "gateway_ksa_name" {
  value = var.provision_gateway_infra ? module.gateway_service[0].gateway_ksa_name : null
}

output "subscription_ksa_name" {
  value = local.provision_subscription_infra ? module.subscription_service[0].subscription_ksa_name : null
}
