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

#--------------------------------------------- Provider Configuration ---------------------------------------------#

# The below block is to configure a provider. Here google is being used as the provider

provider "google" {
  project = var.project_id
  region  = var.region
}

#--------------------------------------------- Data Configuration for project ID ---------------------------------------------#

# The below block holds project_id as data

data "google_project" "project" {
  project_id = var.project_id
}


output "project_id" {
  value = var.project_id
}

output "cluster_region" {
  value = var.region
}

# Below module is used for setting up the Service Account for the Kubernetes Engine
module "kubernetes_service_account" {
  source       = "./IAM_ADMIN/SERVICE_ACCOUNT"
  account_id   = var.kubernetes_sa_account_id
  display_name = var.kubernetes_sa_display_name
  description  = var.kubernetes_sa_description
}

# Below module is used to bind the kubernetes service account with required IAM roles
module "IAM_for_kubernetes_sa" {
  source     = "./IAM_ADMIN/IAM"
  for_each   = toset(var.kubernetes_sa_roles)
  project_id = var.project_id
  member_role = each.value
  member     = "serviceAccount:${module.kubernetes_service_account.service_account_email}"
  depends_on = [module.kubernetes_service_account]
}

#--------------------------------------------- Network Configuration ---------------------------------------------#

# Below module is used to create network, subnetwork, ip range of subnetwork, secondary subnets and ranges for pods and services

module "network" {
  source = "./VPC"

  network_name        = var.network_name
  network_description = var.network_description

  subnet_name        = var.subnet_name
  subnet_description = var.subnet_description
  ip_cidr_range      = var.ip_cidr_range
  range_name         = var.range_name
  ip_range           = var.ip_range
  range_name_1       = var.range_name_1
  ip_range_1         = var.ip_range_1
  region             = var.region
}

#--------------------------------------------- GKE Configuration ---------------------------------------------#

# Below module will set up a gke cluster
# master_ipv4_cidr_block must be unique within the project
# master_access_cidr_block is the cidr block allowed to access the master

module "gke" {
  source = "./GKE"

  cluster_name        = var.cluster_name
  cluster_region      = var.region
  cluster_description = var.cluster_description
  initial_node_count  = var.initial_node_count

  network    = "projects/${data.google_project.project.project_id}/global/networks/${module.network.network_name}"
  subnetwork = "projects/${data.google_project.project.project_id}/regions/${var.region}/subnetworks/${module.network.subnet_name}"

  workload_pool = "${data.google_project.project.project_id}.svc.id.goog"

  cluster_secondary_range_name = module.network.range_name
  services_secondary_range_name = module.network.range_name_1

  master_ipv4_cidr_block   = var.master_ipv4_cidr_block
  master_access_cidr_block = var.master_access_cidr_block
  display_name             = var.display_name

  depends_on = [module.network]
}

output "cluster_name" {
  value = module.gke.cluster_name
}

#--------------------------------------------- GKE Node Pool Configuration ---------------------------------------------#

# Below module is used for setting up node pool inside the cluster
# Since the cluster is regional, if node_count = 1, then there will be 3 nodes, 1 per zone

module "gke_node_pool" {
  source = "./GKE_NODE_POOL"

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

  depends_on = [module.gke, module.network]
}

#--------------------------------------------- Helm Configuration ---------------------------------------------#

# Below data block is used to dynamically retrieve the authentication token for interacting with the Kubernetes cluster deployed on GKE.

data "google_client_config" "default" {}

#--------------------------------------------- Kubernetes Provider Configuration ---------------------------------------------#

provider "kubernetes" {
  host = "https://${module.gke.cluster_endpoint}"
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  token = data.google_client_config.default.access_token
}

#--------------------------------------------- Router Configuration ---------------------------------------------#

# Below module is used to configure router for nodes to communicate with internet

module "router" {
  source            = "./CLOUD_NAT/COMPUTE_ROUTER"
  router_name       = var.router_name
  network_name      = module.network.network_name
  router_description = var.router_description
  router_region     = var.region

  depends_on = [module.network]
}

#--------------------------------------------- Router NAT Configuration ---------------------------------------------#

# Below module is used to NAT nodes private IP with public. Since the cluster is private without router and NAT the cluster cannot communicate with the internet

module "router_nat" {
  source     = "./CLOUD_NAT/COMPUTE_ROUTER_NAT"
  nat_name   = var.nat_name
  router_name = module.router.router_name
  nat_region = var.region
  project_id = data.google_project.project.project_id
  depends_on = [module.router, module.network]
}

#--------------------------------------------- Helm Configuration (Provider) ---------------------------------------------#

provider "helm" {
  kubernetes = {
  host = "https://${module.gke.cluster_endpoint}"
  cluster_ca_certificate = base64decode(module.gke.ca_certificate)
  token = data.google_client_config.default.access_token
  }
}

#--------------------------------------------- Helm Configuration (Module) ---------------------------------------------#

module "helm_config" {
  source         = "./HELM/HELM_CONFIG"
  endpoint       = "https://${module.gke.cluster_endpoint}"
  ca_certificate = base64decode(module.gke.ca_certificate)
  access_token   = data.google_client_config.default.access_token
}

#--------------------------------------------- Application Namespace Configuration ---------------------------------------------#

# Module for creating namespace

module "nginx_namepsace"{
  source = "./NAMESPACE"
  namespace_name = var.nginix_namespace_name
  depends_on = [ module.gke, module.gke_node_pool]
}

# Module for creating the common Kubernetes namespace for all services
module "app_namespace" {
  source         = "./NAMESPACE"
  namespace_name = var.app_namespace_name
  depends_on     = [module.gke, module.gke_node_pool]
}

output "app_namespace_name" {
  value = module.app_namespace.namespace_name
}

#--------------------------------------------- Nginx Ingress Configuration ---------------------------------------------#

# The below resource block is used to bind NEGs with unique ID to avoid conflicts
resource "random_id" "suffix" {
  byte_length = 4
}

# Define the NEG name dynamically
locals {
  neg_name = "${var.nginix_ingress_release_name}-neg-${random_id.suffix.hex}"
}

# Below module is used to deploy nginix-ingress controller
# By default, Kubernetes services are only accessible within the cluster.
# The Ingress Controller allows external users to access services via an Ingress resource.

module "nginx_ingress" {
  source          = "./HELM/HELM_RELEASES"
  helm_name       = var.nginix_ingress_release_name
  helm_repository = var.nginix_ingress_repository
  helm_namespace  = var.nginix_namespace_name
  helm_chart      = var.nginix_ingress_chart
  helm_values = [templatefile("./CONFIG_FILES/nginx.conf.tpl", { neg_name = local.neg_name })]
  depends_on      = [module.gke, module.gke_node_pool, module.helm_config, module.http_rule, module.http_firewall_rule, module.https-firewall-rule]
}

#--------------------------------------------- Health Check Configuration ---------------------------------------------#

# Below module is to create and have health checks on the backends

module "health_check" {
  source               = "./HEALTH_CHECK"
  health_check_name    = var.health_check_name
  health_check_description = var.health_check_description
  depends_on           = [module.network]
}

#--------------------------------------------- Backend Service Configuration ---------------------------------------------#

# The below module is used for creating a backend for the load balancer

module "backend_service" {
  source              = "./LOAD_BALANCER/BACKEND"
  backend_name        = var.backend_service_name
  backend_description = var.backend_service_description
  group_1             = "projects/${data.google_project.project.project_id}/zones/${var.region}-a/networkEndpointGroups/${local.neg_name}"
  group_2             = "projects/${data.google_project.project.project_id}/zones/${var.region}-b/networkEndpointGroups/${local.neg_name}"
  group_3             = "projects/${data.google_project.project.project_id}/zones/${var.region}-c/networkEndpointGroups/${local.neg_name}"
  health_check        = ["projects/${data.google_project.project.project_id}/global/healthChecks/${module.health_check.health_check_name}"]
  depends_on          = [module.gke, module.health_check, module.gke_node_pool, module.nginx_ingress]
}

#--------------------------------------------- Firewall Configuration ---------------------------------------------#

# The below module is used for configuring a firewall for health check

module "http_rule" {
  source             = "./VPC/FIREWALL_ALLOW"
  firewall_name      = var.http_firewall_name
  firewall_description = var.http_firewall_description
  vpc_network_name   = module.network.network_name
  firewall_direction = var.http_firewall_direction
  allow_protocols    = var.http_allow_protocols
  allow_ports        = var.http_allow_ports
  source_ranges      = var.source_ranges
  depends_on         = [module.network]
}

# The below module is used for configuring a firewall to allow http traffic
# Note Nginx Ingress will not be deployed since it is necessary to pull the image from internet

module "http_firewall_rule" {
  source             = "./VPC/FIREWALL_ALLOW"
  firewall_name      = var.allow_http_firewall_name
  firewall_description = var.allow_http_firewall_description
  vpc_network_name   = module.network.network_name
  firewall_direction = var.allow_http_firewall_direction
  allow_protocols    = var.allow_http_allow_protocols
  allow_ports        = var.allow_http_allow_ports
  source_ranges      = var.http_source_ranges
  depends_on         = [module.network]
}

# The below module is used for configuring a firewall to allow https traffic
# Note Nginx Ingress will not be deployed since it is necessary to pull the image from internet

module "https-firewall-rule" {
  source             = "./VPC/FIREWALL_ALLOW"
  firewall_name      = var.allow_https_firewall_name
  firewall_description = var.allow_https_firewall_description
  vpc_network_name   = module.network.network_name
  firewall_direction = var.allow_https_firewall_direction
  allow_protocols    = var.allow_https_allow_protocols
  allow_ports        = var.allow_https_allow_ports
  source_ranges      = var.https_source_ranges
  depends_on         = [module.network]
}

#--------------------------------------------- Global IP Configuration ---------------------------------------------#

# Configure a global IP for your Load Balancer

module "lb_global_ip" {
  source             = "./COMPUTE_ENGINE/GLOBAL_ADDRESS"
  global_ip_name     = var.global_ip_name
  global_ip_description = var.global_ip_description
  global_ip_labels   = var.global_ip_labels
}

output "global_ip_address" {
  value = module.lb_global_ip.global_ip_address
}

#--------------------------------------------- URL Map Configuration ---------------------------------------------#

# URL MAP for your Load Balancer

module "url_map" {
  source           = "./LOAD_BALANCER/URL_MAP"
  url_map_name     = var.url_map_name
  backend_service_id = module.backend_service.backend_id
  url_map_description = var.url_map_description
  depends_on       = [module.backend_service]
}

output "url_map" {
  value = module.url_map.url_map
}

#--------------------------------------------- Private VPC Access Configuration ---------------------------------------------#

module "global_address" {
  source                    = "./VPC/GLOBAL_ADDRESS"
  vpc_peering_ip_name       = var.vpc_peering_ip_name
  vpc_peering_ip_purpose    = var.vpc_peering_ip_purpose
  vpc_peering_ip_address_type = var.vpc_peering_ip_address_type
  vpc_peering_ip_network    = "projects/${data.google_project.project.project_id}/global/networks/${module.network.network_name}"
  vpc_peering_prefix_length = var.vpc_peering_prefix_length
  depends_on                = [module.network]
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network             = "projects/${data.google_project.project.project_id}/global/networks/${module.network.network_name}"
  service             = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [module.global_address.reserved_ip_range]
  deletion_policy = "ABANDON"
  
  depends_on = [module.global_address, module.network] # Removed redundant global_address dependency
}

resource "time_sleep" "wait_for_ps_networking" {
  depends_on      = [google_service_networking_connection.private_vpc_connection]
  create_duration = "150s"
}

#--------------------------------------------- Redis Configuration ---------------------------------------------#

# The redis instance will be spanned if the networking configuration is private access and a private access module is not being used

module "redis" {
  source                = "./REDIS/REDIS_INSTANCE"
  instance_name         = var.instance_name
  memory_size_gb        = var.memory_size_gb
  instance_tier         = var.instance_tier
  instance_region       = var.instance_region
  instance_location_id  = var.instance_location_id
  instance_authorized_network = "projects/${data.google_project.project.project_id}/global/networks/${module.network.network_name}"
  instance_connect_mode = var.instance_connect_mode
  depends_on            = [google_service_networking_connection.private_vpc_connection, module.global_address]
}

output "redis_instance_ip" {
  value       = module.redis.redis_instance_ip
  description = "The IP address of the created Redis instance"
}

#--------------------------------------------- Configuration Bucket ---------------------------------------------#
# This bucket stores configurations for various services.
module "config_bucket" {
  source          = "./CLOUD_STORAGE"
  bucket_name     = var.config_bucket_name
  bucket_location = var.region
}

output "config_bucket_name" {
  value = module.config_bucket.bucket_name
}

#--------------------------------------------- Service Specific ---------------------------------------------#


# Pub Sub Topic Configuration for onix events
module "pubsub_topic_onix" {
  source     = "./PUB_SUB_TOPIC"
  topic_name = var.pubsub_topic_onix_name
}

output "onix_topic_name" {
  value=var.pubsub_topic_onix_name
}


# Controls whether the Subscription service should be provisioned implicitly
locals {
  provision_subscription_infra = var.provision_adapter_infra || var.provision_gateway_infra
}


# Module for Adapter Service
module "adapter_service" {
  count = var.provision_adapter_infra ? 1 : 0
  source = "./SERVICES/ADAPTER"

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
    module.config_bucket
  ]
}

output "adapter_ksa_name" {
  value = var.provision_adapter_infra ? module.adapter_service[0].adapter_ksa_name : null
}

output "adapter_topic_name" {
  value = var.provision_adapter_infra ? module.adapter_service[0].adapter_topic_name : null
}

module "registry_service" {
  count = var.provision_registry_infra ? 1 : 0
  source = "./SERVICES/REGISTRY"

  project_id = data.google_project.project.project_id
  network_name = module.network.network_name
  app_namespace_name = module.app_namespace.namespace_name

  # DB related variables
  registry_db_instance_region = var.registry_db_instance_region
  registry_db_instance_version = var.registry_db_instance_version
  registry_db_instance_name = var.registry_db_instance_name
  registry_db_instance_tier = var.registry_db_instance_tier
  registry_db_instance_labels = var.registry_db_instance_labels
  registry_db_instance_edition = var.registry_db_instance_edition
  registry_db_aval_type = var.registry_db_aval_type
  registry_db_instance_disk_size = var.registry_db_instance_disk_size
  registry_db_instance_disk_type = var.registry_db_instance_disk_type
  registry_db_ipv4 = var.registry_db_ipv4
  registry_db_max_connections = var.registry_db_max_connections
  registry_db_instance_cache = var.registry_db_instance_cache
  registry_database_name = var.registry_database_name
  
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
    module.network,
    google_service_networking_connection.private_vpc_connection,
    module.global_address,
    time_sleep.wait_for_ps_networking,
    module.app_namespace,
    module.config_bucket
  ]
}

output "registry_db_instance_name" {
  value = var.provision_registry_infra ? var.registry_db_instance_name : null
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

output "registry_db_connection_name" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_db_connection_name : null
}

output "database_user_sa_email" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_gsa_email : null
}

output "registry_admin_database_user_sa_email" {
  value = var.provision_registry_infra ? module.registry_service[0].registry_admin_gsa_email : null
}

# Module for Gateway Service
module "gateway_service" {
  count = var.provision_gateway_infra ? 1 : 0
  source = "./SERVICES/GATEWAY"

  project_id = data.google_project.project.project_id
  app_namespace_name = module.app_namespace.namespace_name # Pass common namespace
  gateway_ksa_name = var.gateway_ksa_name
  gateway_gsa_account_id = var.gateway_gsa_account_id
  gateway_gsa_display_name = var.gateway_gsa_display_name
  gateway_gsa_description = var.gateway_gsa_description
  gateway_gsa_roles = var.gateway_gsa_roles # New variable for Gateway GSA roles

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace # Ensure namespace exists
  ]
}

output "gateway_ksa_name" {
  value = var.provision_gateway_infra ? module.gateway_service[0].gateway_ksa_name : null
}


# Module for Subscription Service
module "subscription_service" {
  count = local.provision_subscription_infra ? 1 : 0
  source = "./SERVICES/SUBSCRIPTION"

  project_id = data.google_project.project.project_id
  app_namespace_name = module.app_namespace.namespace_name # Pass common namespace
  subscription_ksa_name = var.subscription_ksa_name
  subscription_gsa_account_id = var.subscription_gsa_account_id
  subscription_gsa_display_name = var.subscription_gsa_display_name
  subscription_gsa_description = var.subscription_gsa_description
  subscription_gsa_roles = var.subscription_gsa_roles # New variable for Subscription GSA roles

  depends_on = [
    module.gke,
    module.gke_node_pool,
    module.app_namespace # Ensure namespace exists
  ]
}

output "subscription_ksa_name" {
  value = local.provision_subscription_infra ? module.subscription_service[0].subscription_ksa_name : null
}