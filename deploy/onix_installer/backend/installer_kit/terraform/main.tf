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

#--------------------------------------------- Provider Configuration ---------------------------------------------#

# The below block is to configure a provider. Here google is being used as the provider

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

#--------------------------------------------- Data Configuration for project ID ---------------------------------------------#

# The below block holds project_id as data

data "google_project" "project" {
  project_id = var.project_id
}

#--------------------------------------------- API/Service Configuration ---------------------------------------------#

locals {
  agent_apis = [
    "aiplatform.googleapis.com",
    "logging.googleapis.com",
    "cloudtrace.googleapis.com",
    "storage.googleapis.com",
    "serviceusage.googleapis.com",
    "discoveryengine.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "compute.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "servicenetworking.googleapis.com",
    "telemetry.googleapis.com",
    "secretmanager.googleapis.com"
  ]
}

resource "google_project_service" "apis" {
  for_each           = var.enable_agent ? toset(local.agent_apis) : []
  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

resource "time_sleep" "wait_for_apis" {
  depends_on      = [google_project_service.apis]
  create_duration = var.enable_agent ? "30s" : "0s"
}


#--------------------------------------------- Network Configuration ---------------------------------------------#

# VPC
module "network" {
  source = "./modules/VPC"

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

  depends_on = [time_sleep.wait_for_apis]
}

#--------------------------------------------- Router Configuration ---------------------------------------------#

# Below module is used to configure router to communicate with internet

module "router" {
  source            = "./modules/CLOUD_NAT/COMPUTE_ROUTER"
  router_name       = var.router_name
  network_name      = module.network.network_name
  router_description = var.router_description
  router_region     = var.region

  depends_on = [module.network]
}

#--------------------------------------------- Router NAT Configuration ---------------------------------------------#

module "router_nat" {
  source     = "./modules/CLOUD_NAT/COMPUTE_ROUTER_NAT"
  nat_name   = var.nat_name
  router_name = module.router.router_name
  nat_region = var.region
  project_id = data.google_project.project.project_id
  depends_on = [module.router, module.network]
}

#--------------------------------------------- Private VPC Access Configuration ---------------------------------------------#

module "global_address" {
  source                    = "./modules/VPC/GLOBAL_ADDRESS"
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
  
  depends_on = [module.global_address, module.network]
}

resource "time_sleep" "wait_for_ps_networking" {
  depends_on      = [google_service_networking_connection.private_vpc_connection]
  create_duration = "150s"
}

#--------------------------------------------- Cloud SQL Instance ---------------------------------------------#

# Controls whether the Cloud SQL Instance should be provisioned
locals {
  provision_db_instance = var.provision_registry_infra || (var.enable_agent && var.provision_agent_db)
}

module "db_instance" {
  count                         = local.provision_db_instance ? 1 : 0
  source                        = "./modules/CLOUD_SQL/DATABASE_INSTANCE"

  db_instance_name              = var.db_instance_name
  db_instance_region            = var.db_instance_region
  db_instance_version           = var.db_instance_version
  db_instance_tier              = var.db_instance_tier
  db_instance_labels            = var.db_instance_labels
  db_instance_edition           = var.db_instance_edition
  db_instance_availability_type = var.db_instance_availability_type
  db_instance_disk_size         = var.db_instance_disk_size
  db_instance_disk_type         = var.db_instance_disk_type
  db_instance_public_ipv4       = var.db_instance_public_ipv4
  db_instance_max_connections   = var.db_instance_max_connections
  db_instance_cache             = var.db_instance_cache
  db_instance_network           = "projects/${var.project_id}/global/networks/${module.network.network_name}"

  depends_on = [
    module.network,
    google_service_networking_connection.private_vpc_connection,
    time_sleep.wait_for_ps_networking
  ]
}

#--------------------------------------------- Redis Instance ---------------------------------------------#

module "redis" {
  source                = "./modules/REDIS/REDIS_INSTANCE"
  instance_name         = var.instance_name
  memory_size_gb        = var.memory_size_gb
  instance_tier         = var.instance_tier
  instance_region       = var.instance_region
  instance_location_id  = var.instance_location_id
  instance_authorized_network = "projects/${data.google_project.project.project_id}/global/networks/${module.network.network_name}"
  instance_connect_mode = var.instance_connect_mode
  depends_on            = [google_service_networking_connection.private_vpc_connection, module.global_address]
}

#--------------------------------------------- ONIX Orchestrator ---------------------------------------------#

module "onix" {
  source = "./modules/ONIX"
  enable_onix = var.enable_onix

  project_id = var.project_id
  region     = var.region
  app_name   = var.app_name

  network_name         = module.network.network_name
  subnet_name          = module.network.subnet_name
  network_range_name   = module.network.range_name
  network_range_name_1 = module.network.range_name_1
  db_instance_name = local.provision_db_instance ? module.db_instance[0].db_instance_name : null

  # GKE
  cluster_name               = var.cluster_name
  cluster_description        = var.cluster_description
  initial_node_count         = var.initial_node_count
  master_ipv4_cidr_block     = var.master_ipv4_cidr_block
  master_access_cidr_block   = var.master_access_cidr_block
  display_name               = var.display_name
  kubernetes_sa_account_id   = var.kubernetes_sa_account_id
  kubernetes_sa_display_name = var.kubernetes_sa_display_name
  kubernetes_sa_description  = var.kubernetes_sa_description
  kubernetes_sa_roles        = var.kubernetes_sa_roles

  # Node Pool
  node_pool_name    = var.node_pool_name
  reg_node_location = var.reg_node_location
  max_pods_per_node = var.max_pods_per_node
  disk_size         = var.disk_size
  disk_type         = var.disk_type
  image_type        = var.image_type
  pool_labels       = var.pool_labels
  machine_type      = var.machine_type
  node_count        = var.node_count
  min_node_count    = var.min_node_count
  max_node_count    = var.max_node_count

  # App Config
  app_namespace_name          = var.app_namespace_name

  # Nginx Ingress
  nginix_ingress_release_name = var.nginix_ingress_release_name
  nginix_ingress_repository   = var.nginix_ingress_repository
  nginix_namespace_name       = var.nginix_namespace_name
  nginix_ingress_chart        = var.nginix_ingress_chart

  # Load Balancer & Network
  health_check_name                = var.health_check_name
  health_check_description         = var.health_check_description
  enable_cloud_armor               = var.enable_cloud_armor
  rate_limit_count                 = var.rate_limit_count
  backend_service_name             = var.backend_service_name
  backend_service_description      = var.backend_service_description
  global_ip_name                   = var.global_ip_name
  global_ip_description            = var.global_ip_description
  global_ip_labels                 = var.global_ip_labels
  url_map_name                     = var.url_map_name
  url_map_description              = var.url_map_description

  # Firewalls
  http_firewall_name               = var.http_firewall_name
  http_firewall_description        = var.http_firewall_description
  http_firewall_direction          = var.http_firewall_direction
  http_allow_protocols             = var.http_allow_protocols
  http_allow_ports                 = var.http_allow_ports
  source_ranges                    = var.source_ranges
  allow_http_firewall_name         = var.allow_http_firewall_name
  allow_http_firewall_description  = var.allow_http_firewall_description
  allow_http_firewall_direction    = var.allow_http_firewall_direction
  allow_http_allow_protocols       = var.allow_http_allow_protocols
  allow_http_allow_ports           = var.allow_http_allow_ports
  http_source_ranges               = var.http_source_ranges
  allow_https_firewall_name        = var.allow_https_firewall_name
  allow_https_firewall_description = var.allow_https_firewall_description
  allow_https_firewall_direction   = var.allow_https_firewall_direction
  allow_https_allow_protocols      = var.allow_https_allow_protocols
  allow_https_allow_ports          = var.allow_https_allow_ports
  https_source_ranges              = var.https_source_ranges

  # Services & PubSub
  pubsub_topic_onix_name   = var.pubsub_topic_onix_name
  provision_adapter_infra  = var.provision_adapter_infra
  provision_gateway_infra  = var.provision_gateway_infra
  provision_registry_infra = var.provision_registry_infra

  # Service Vars
  adapter_ksa_name              = var.adapter_ksa_name
  adapter_gsa_account_id        = var.adapter_gsa_account_id
  adapter_gsa_display_name      = var.adapter_gsa_display_name
  adapter_gsa_description       = var.adapter_gsa_description
  adapter_gsa_roles             = var.adapter_gsa_roles
  adapter_topic_name            = var.adapter_topic_name

  gateway_ksa_name              = var.gateway_ksa_name
  gateway_gsa_account_id        = var.gateway_gsa_account_id
  gateway_gsa_display_name      = var.gateway_gsa_display_name
  gateway_gsa_description       = var.gateway_gsa_description
  gateway_gsa_roles             = var.gateway_gsa_roles

  registry_gsa_account_id         = var.registry_gsa_account_id
  registry_gsa_display_name       = var.registry_gsa_display_name
  registry_gsa_description        = var.registry_gsa_description
  registry_gsa_roles              = var.registry_gsa_roles
  registry_ksa_name               = var.registry_ksa_name
  registry_admin_gsa_account_id   = var.registry_admin_gsa_account_id
  registry_admin_gsa_display_name = var.registry_admin_gsa_display_name
  registry_admin_gsa_description  = var.registry_admin_gsa_description
  registry_admin_gsa_roles        = var.registry_admin_gsa_roles
  registry_admin_ksa_name         = var.registry_admin_ksa_name
  registry_database_name          = var.registry_database_name

  subscription_ksa_name         = var.subscription_ksa_name
  subscription_gsa_account_id   = var.subscription_gsa_account_id
  subscription_gsa_display_name = var.subscription_gsa_display_name
  subscription_gsa_description  = var.subscription_gsa_description
  subscription_gsa_roles        = var.subscription_gsa_roles

}


#--------------------------------------------- Agent Orchestrator ---------------------------------------------#

module "agent" {
  count = var.enable_agent ? 1 : 0

  source = "./modules/AGENT"

  # Core Inputs
  project_id   = var.project_id
  region       = var.region
  app_name     = var.app_name

  # Networking
  network_id                    = module.network.network_id
  subnet_name                   = module.network.subnet_name
  subnetwork_id                 = module.network.subnetwork_id
  agent_network_attachment_name = var.agent_network_attachment_name

  # Datastores
  datastore_imports = var.datastore_imports

  # Data
  provision_agent_db = var.provision_agent_db
  db_instance_name = local.provision_db_instance ? module.db_instance[0].db_instance_name : null
  agent_db_name    = var.agent_db_name
  agent_db_user    = var.agent_db_user

  # IAM
  agent_sa_account_id   = var.agent_sa_account_id
  agent_sa_display_name = var.agent_sa_display_name
  agent_sa_description  = var.agent_sa_description
  agent_sa_roles        = var.agent_sa_roles

  agent_invoker_sa_account_id   = var.agent_invoker_sa_account_id

  depends_on = [
    module.network,
    module.db_instance,
    time_sleep.wait_for_apis
  ]
}
