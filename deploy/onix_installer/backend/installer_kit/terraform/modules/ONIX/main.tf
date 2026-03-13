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

#--------------------------------------------- Data Configuration for project ID ---------------------------------------------#

data "google_project" "project" {
  project_id = var.project_id
}

data "google_client_config" "default" {}

#--------------------------------------------- Kubernetes Provider Configuration ---------------------------------------------#

provider "kubernetes" {
  host = "https://${module.gke.cluster_endpoint}"
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  token = data.google_client_config.default.access_token
}

provider "helm" {
  kubernetes = {
  host = "https://${module.gke.cluster_endpoint}"
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  token = data.google_client_config.default.access_token
  }
}

#--------------------------------------------- GKE Configuration ---------------------------------------------#

module "gke" {
  source = "../../GKE"

  cluster_name        = var.cluster_name
  cluster_region      = var.region
  cluster_description = var.cluster_description
  initial_node_count  = var.initial_node_count

  network    = "projects/${data.google_project.project.project_id}/global/networks/${var.network_name}"
  subnetwork = "projects/${data.google_project.project.project_id}/regions/${var.region}/subnetworks/${var.subnet_name}"

  workload_pool = "${data.google_project.project.project_id}.svc.id.goog"

  cluster_secondary_range_name = var.network_range_name
  services_secondary_range_name = var.network_range_name_1

  master_ipv4_cidr_block   = var.master_ipv4_cidr_block
  master_access_cidr_block = var.master_access_cidr_block
  display_name             = var.display_name
}

#--------------------------------------------- GKE Node Pool Configuration ---------------------------------------------#

module "kubernetes_service_account" {
  source       = "../../IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.kubernetes_sa_account_id
  display_name = var.kubernetes_sa_display_name
  description  = var.kubernetes_sa_description
}

module "IAM_for_kubernetes_sa" {
  source     = "../../IAM_ADMIN/IAM"
  for_each   = toset(var.kubernetes_sa_roles)
  project_id = var.project_id
  member_role = each.value
  member     = "serviceAccount:${module.kubernetes_service_account.service_account_email}"
  depends_on = [module.kubernetes_service_account]
}

module "gke_node_pool" {
  source = "../../GKE_NODE_POOL"

  cluster_name         = module.gke.cluster_name
  node_pool_name       = var.node_pool_name
  node_pool_location   = var.region
  project_id           = data.google_project.project.project_id
  reg_node_location    = var.reg_node_location
  max_pods_per_node    = var.max_pods_per_node
  disk_size            = var.disk_size
  disk_type            = var.disk_type
  image_type           = var.image_type
  pool_labels          = var.pool_labels
  machine_type         = var.machine_type
  node_service_account = module.kubernetes_service_account.service_account_email
  node_count           = var.node_count
  min_node_count       = var.min_node_count
  max_node_count       = var.max_node_count

  depends_on = [module.gke]
}

#--------------------------------------------- Helm Configuration (Module) ---------------------------------------------#

module "helm_config" {
  source         = "../../HELM/HELM_CONFIG"
  endpoint       = "https://${module.gke.cluster_endpoint}"
  ca_certificate = base64decode(module.gke.ca_certificate)
  access_token   = data.google_client_config.default.access_token
}

#--------------------------------------------- Application Namespace Configuration ---------------------------------------------#

module "nginx_namepsace"{
  source = "../../NAMESPACE"
  namespace_name = var.nginix_namespace_name
  depends_on = [ module.gke, module.gke_node_pool]
}

module "app_namespace" {
  source         = "../../NAMESPACE"
  namespace_name = var.app_namespace_name
  depends_on     = [module.gke, module.gke_node_pool]
}

#--------------------------------------------- Nginx Ingress Configuration ---------------------------------------------#

resource "random_id" "suffix" {
  byte_length = 4
}

locals {
  neg_name = "${var.nginix_ingress_release_name}-neg-${random_id.suffix.hex}"
}

module "nginx_ingress" {
  source          = "../../HELM/HELM_RELEASES"
  helm_name       = var.nginix_ingress_release_name
  helm_repository = var.nginix_ingress_repository
  helm_namespace  = var.nginix_namespace_name
  helm_chart      = var.nginix_ingress_chart
  helm_values = [templatefile("./CONFIG_FILES/nginx.conf.tpl", { neg_name = local.neg_name })]
  depends_on      = [module.gke, module.gke_node_pool, module.helm_config, module.http_rule, module.http_firewall_rule, module.https-firewall-rule]
}

#--------------------------------------------- Health Check Configuration ---------------------------------------------#

module "health_check" {
  source               = "../../HEALTH_CHECK"
  health_check_name    = var.health_check_name
  health_check_description = var.health_check_description
}

#--------------------------------------------- Backend Service Configuration ---------------------------------------------#

module "security_policy" {
  count            = var.enable_cloud_armor ? 1 : 0
  source           = "../../LOAD_BALANCER/SECURITY_POLICY"
  app_name         = var.app_name
  allowed_regions  = var.allowed_regions
  rate_limit_count = var.rate_limit_count
}

module "backend_service" {
  source              = "../../LOAD_BALANCER/BACKEND"
  backend_name        = var.backend_service_name
  backend_description = var.backend_service_description
  group_1             = "projects/${data.google_project.project.project_id}/zones/${var.region}-a/networkEndpointGroups/${local.neg_name}"
  group_2             = "projects/${data.google_project.project.project_id}/zones/${var.region}-b/networkEndpointGroups/${local.neg_name}"
  group_3             = "projects/${data.google_project.project.project_id}/zones/${var.region}-c/networkEndpointGroups/${local.neg_name}"
  health_check        = ["projects/${data.google_project.project.project_id}/global/healthChecks/${module.health_check.health_check_name}"]
  depends_on          = [module.gke, module.health_check, module.gke_node_pool, module.nginx_ingress]
  security_policy = var.enable_cloud_armor ? module.security_policy[0].policy_id : null
}

#--------------------------------------------- Firewall Configuration ---------------------------------------------#

module "http_rule" {
  source             = "../../VPC/FIREWALL_ALLOW"
  firewall_name      = var.http_firewall_name
  firewall_description = var.http_firewall_description
  vpc_network_name   = var.network_name
  firewall_direction = var.http_firewall_direction
  allow_protocols    = var.http_allow_protocols
  allow_ports        = var.http_allow_ports
  source_ranges      = var.source_ranges
}

module "http_firewall_rule" {
  source             = "../../VPC/FIREWALL_ALLOW"
  firewall_name      = var.allow_http_firewall_name
  firewall_description = var.allow_http_firewall_description
  vpc_network_name   = var.network_name
  firewall_direction = var.allow_http_firewall_direction
  allow_protocols    = var.allow_http_allow_protocols
  allow_ports        = var.allow_http_allow_ports
  source_ranges      = var.http_source_ranges
}

module "https-firewall-rule" {
  source             = "../../VPC/FIREWALL_ALLOW"
  firewall_name      = var.allow_https_firewall_name
  firewall_description = var.allow_https_firewall_description
  vpc_network_name   = var.network_name
  firewall_direction = var.allow_https_firewall_direction
  allow_protocols    = var.allow_https_allow_protocols
  allow_ports        = var.allow_https_allow_ports
  source_ranges      = var.https_source_ranges
}

#--------------------------------------------- Global IP Configuration ---------------------------------------------#

module "lb_global_ip" {
  source = "../../COMPUTE_ENGINE/GLOBAL_ADDRESS"
  global_ip_name     = var.global_ip_name
  global_ip_description = var.global_ip_description
  global_ip_labels   = var.global_ip_labels
}

#--------------------------------------------- URL Map Configuration ---------------------------------------------#

module "url_map" {
  source           = "../../LOAD_BALANCER/URL_MAP"
  url_map_name     = var.url_map_name
  backend_service_id = module.backend_service.backend_id
  url_map_description = var.url_map_description
  depends_on       = [module.backend_service]
}

#--------------------------------------------- Service Specific ---------------------------------------------#

module "pubsub_topic_onix" {
  source     = "../../PUB_SUB_TOPIC"
  topic_name = var.pubsub_topic_onix_name
}

locals {
  provision_subscription_infra = var.provision_adapter_infra || var.provision_gateway_infra
}

module "adapter_service" {
  count = var.provision_adapter_infra ? 1 : 0
  source = "../../SERVICES/ADAPTER"

  project_id = data.google_project.project.project_id
  app_namespace_name = module.app_namespace.namespace_name
  adapter_ksa_name = var.adapter_ksa_name
  adapter_gsa_account_id = var.adapter_gsa_account_id
  adapter_gsa_display_name = var.adapter_gsa_display_name
  adapter_gsa_description = var.adapter_gsa_description
  adapter_gsa_roles = var.adapter_gsa_roles
  adapter_topic_name = var.adapter_topic_name

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace,
  ]
}

module "registry_service" {
  count = var.provision_registry_infra ? 1 : 0
  source = "../../SERVICES/REGISTRY"

  project_id = data.google_project.project.project_id
  network_name = var.network_name
  app_namespace_name = module.app_namespace.namespace_name

  # Helper SQL Instance (input)
  db_instance_name = var.db_instance_name

  # DB variables
  registry_database_name         = var.registry_database_name
  
  # GSA and KSA related variables
  registry_gsa_account_id = var.registry_gsa_account_id
  registry_gsa_display_name = var.registry_gsa_display_name
  registry_gsa_description = var.registry_gsa_description
  registry_gsa_roles = var.registry_gsa_roles
  registry_ksa_name = var.registry_ksa_name

  # GSA and KSA related variables for Registry Admin
  registry_admin_gsa_account_id = var.registry_admin_gsa_account_id
  registry_admin_gsa_display_name = var.registry_admin_gsa_display_name
  registry_admin_gsa_description = var.registry_admin_gsa_description
  registry_admin_gsa_roles = var.registry_admin_gsa_roles
  registry_admin_ksa_name = var.registry_admin_ksa_name

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace
  ]
}

module "gateway_service" {
  count = var.provision_gateway_infra ? 1 : 0
  source = "../../SERVICES/GATEWAY"

  project_id = data.google_project.project.project_id
  app_namespace_name = module.app_namespace.namespace_name
  gateway_ksa_name = var.gateway_ksa_name
  gateway_gsa_account_id = var.gateway_gsa_account_id
  gateway_gsa_display_name = var.gateway_gsa_display_name
  gateway_gsa_description = var.gateway_gsa_description
  gateway_gsa_roles = var.gateway_gsa_roles

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace
  ]
}

module "subscription_service" {
  count = local.provision_subscription_infra ? 1 : 0
  source = "../../SERVICES/SUBSCRIPTION"

  project_id = data.google_project.project.project_id
  app_namespace_name = module.app_namespace.namespace_name
  subscription_ksa_name = var.subscription_ksa_name
  subscription_gsa_account_id = var.subscription_gsa_account_id
  subscription_gsa_display_name = var.subscription_gsa_display_name
  subscription_gsa_description = var.subscription_gsa_description
  subscription_gsa_roles = var.subscription_gsa_roles

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace
  ]
}
