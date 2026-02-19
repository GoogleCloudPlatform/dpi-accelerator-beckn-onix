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

#--------------------------------------------- Project & Region Configuration ---------------------------------------------#

variable "project_id" {
  type        = string
  description = "The project ID to deploy resources"
}

variable "region" {
  type        = string
  description = "The region to deploy resources"
}

variable "app_name" {
  type        = string
  description = "The application name"
}

#--------------------------------------------- Kubernetes Service Account for GKE Nodes ---------------------------------------------#

variable "kubernetes_sa_account_id" {
  type        = string
  description = "ID for your kubernetes account ID"
}

variable "kubernetes_sa_display_name" {
  type        = string
  description = "Display name for your kubernetes account"
}

variable "kubernetes_sa_description" {
  type        = string
  description = "Description for your kubernetes account"
}

variable "kubernetes_sa_roles" {
  type        = list(string)
  description = "Roles for your kubernetes account"
}

#--------------------------------------------- Network Configuration ---------------------------------------------#

variable "network_name" {
  type        = string
  description = "The name of the network"
}

variable "network_description" {
  type        = string
  description = "The description of the network"
}

variable "subnet_name" {
  type        = string
  description = "The name of the subnet"
}

variable "subnet_description" {
  type        = string
  description = "The description of the subnet"
}

variable "ip_cidr_range" {
  type        = string
  description = "The IP CIDR range of the subnet"
}

variable "range_name" {
  type        = string
  description = "The name of the secondary range"
}

variable "ip_range" {
  type        = string
  description = "The IP CIDR range of the secondary range"
}

variable "range_name_1" {
  type        = string
  description = "The name of the secondary range"
}

variable "ip_range_1" {
  type        = string
  description = "The IP CIDR range of the secondary range"
}

#--------------------------------------------- GKE Cluster Configuration ---------------------------------------------#

variable "cluster_name" {
  type        = string
  description = "The name of the GKE cluster"
}

variable "cluster_description" {
  type        = string
  description = "The description of the GKE cluster"
}

variable "initial_node_count" {
  type        = string
  description = "The initial number of nodes in the GKE cluster"
}

variable "master_ipv4_cidr_block" {
  type        = string
  description = "The IP CIDR range of the master"
}

variable "master_access_cidr_block" {
  type        = string
  description = "The IP CIDR range of the master"
}

variable "display_name" {
  type        = string
  description = "The display name of the GKE cluster"
}

#--------------------------------------------- GKE Node Pool Configuration ---------------------------------------------#

variable "node_pool_name" {
  type        = string
  description = "The name of the node pool"
}

variable "reg_node_location" {
  type        = list(string)
  description = "The region of the node pool"
}

variable "max_pods_per_node" {
  type        = string
  description = "The maximum number of pods per node"
}

variable "disk_size" {
  type        = string
  description = "The disk size of the node pool"
}

variable "disk_type" {
  type        = string
  description = "The disk type of the node pool"
}

variable "image_type" {
  type        = string
  description = "The image type of the node pool"
}

variable "pool_labels" {
  type        = map(string)
  description = "The labels of the node pool"
  default     = {}
}

variable "machine_type" {
  type        = string
  description = "The machine type of the node pool"
}

variable "node_count" {
  type        = string
  description = "The number of nodes in the node pool"
  default     = null
}

variable "min_node_count" {
  type        = string
  description = "The minimum number of nodes in the node pool"
  default     = 1
}

variable "max_node_count" {
  type        = string
  description = "The maximum number of nodes in the node pool"
  default     = 3
}

#--------------------------------------------- Router & NAT Configuration ---------------------------------------------#

variable "router_name" {
  type        = string
  description = "The name of the router"
}

variable "router_description" {
  type        = string
  description = "The description of the router"
}

variable "nat_name" {
  type        = string
  description = "The name of the NAT"
}

#--------------------------------------------- Application Namespace Configuration ---------------------------------------------#

variable "app_namespace_name" {
  type        = string
  description = "The common Kubernetes namespace name for all services."
  default     = "beckn-services"
}

#--------------------------------------------- Nginx Ingress Controller Configuration ---------------------------------------------#

variable "nginix_ingress_release_name" {
  type        = string
  description = "The name of the helm release"
}

variable "nginix_ingress_repository" {
  type        = string
  description = "The repository of the helm release"
}

variable "nginix_namespace_name" {
  type        = string
  description = "The name of the namespace"
}

variable "nginix_ingress_chart" {
  type        = string
  description = "The chart of the helm release"
}

#--------------------------------------------- Health Check Configuration ---------------------------------------------#

variable "health_check_name" {
  type        = string
  description = "The name of the health check"
}

variable "health_check_description" {
  type        = string
  description = "The description of the health check"
}

#--------------------------------------------- Backend Service Configuration ---------------------------------------------#

variable "enable_cloud_armor" {
  type    = bool
  default = false
}

variable "allowed_regions" {
  type    = list(string)
  default = ["IN"]
}

variable "rate_limit_count" {
  type    = number
  default = 100
}

variable "backend_service_name" {
  type        = string
  description = "The name of the backend service"
}

variable "backend_service_description" {
  type        = string
  description = "The description of the backend service"
}

#--------------------------------------------- Firewall Rules Configuration ---------------------------------------------#

variable "http_firewall_name" {
  type        = string
  description = "The name of the firewall rule"
}

variable "http_firewall_description" {
  type        = string
  description = "The description of the firewall rule"
}

variable "http_firewall_direction" {
  type        = string
  description = "The direction of the firewall rule"
}

variable "http_allow_protocols" {
  type        = string
  description = "The protocol to allow"
}

variable "http_allow_ports" {
  type        = list(number)
  description = "The list of ports to allow"
  default     = []
}

variable "source_ranges" {
  type        = list(string)
  description = "The list of IP ranges in CIDR format that the rule applies to"
  default     = []
}

variable "allow_http_firewall_name" {
  type        = string
  description = "Name of the firewall rule"
}

variable "allow_http_firewall_description" {
  type        = string
  description = "Description for your firewall rule"
}

variable "allow_http_firewall_direction" {
  type        = string
  description = "Traffic Direction of the firewall rule"
}

variable "allow_http_allow_protocols" {
  type        = string
  description = "The protocol to allow"
}

variable "allow_http_allow_ports" {
  type        = list(number)
  description = "The list of ports to allow"
}

variable "http_source_ranges" {
  type        = list(string)
  description = "The list of IP ranges in CIDR format that the rule applies to "
}

variable "allow_https_firewall_name" {
  type        = string
  description = "Name of the firewall rule"
}

variable "allow_https_firewall_description" {
  type        = string
  description = "Description for your firewall rule"
}

variable "allow_https_firewall_direction" {
  type        = string
  description = "Traffic Direction of the firewall rule"
}

variable "allow_https_allow_protocols" {
  type        = string
  description = "The protocol to allow"
}

variable "allow_https_allow_ports" {
  type        = list(number)
  description = "The list of ports to allow"
}

variable "https_source_ranges" {
  type        = list(string)
  description = "The list of IP ranges in CIDR format that the rule applies to "
}

#--------------------------------------------- Global IP & URL Map Configuration ---------------------------------------------#

variable "global_ip_name" {
  type        = string
  description = "The name of the global IP"
}

variable "global_ip_description" {
  type        = string
  description = "The description of the global IP"
}

variable "global_ip_labels" {
  type        = map(string)
  description = "The labels of the global IP"
  default     = {}
}

variable "url_map_name" {
  type        = string
  description = "The name of the URL map"
}

variable "url_map_description" {
  type        = string
  description = "The description of the URL map"
}

#--------------------------------------------- Private VPC Access Configuration ---------------------------------------------#

variable "vpc_peering_ip_name" {
  type        = string
  description = "Name of the VPC peering IP"
}

variable "vpc_peering_ip_purpose" {
  type        = string
  description = "Purpose of the VPC peering IP"
}

variable "vpc_peering_ip_address_type" {
  type        = string
  description = "Address type of the VPC peering IP"
}

variable "vpc_peering_prefix_length" {
  type        = string
  description = "Prefix length for the VPC peering IP"
}

#--------------------------------------------- Redis Instance Configuration ---------------------------------------------#

variable "instance_name" {
  type        = string
  description = "Name of the Redis instance"
}

variable "memory_size_gb" {
  type        = string
  description = "Memory size for the Redis instance"
}

variable "instance_tier" {
  type        = string
  description = "Tier for the Redis instance"
}

variable "instance_region" {
  type        = string
  description = "Region for the Redis instance"
}

variable "instance_location_id" {
  type        = string
  description = "Location ID for the Redis instance"
}

variable "instance_connect_mode" {
  type        = string
  description = "Connect mode for the Redis instance"
}

#--------------------------------------------- Configuration Bucket ---------------------------------------------#

variable "config_bucket_name" {
  type        = string
  description = "The name of the bucket to store your configs"
}

#--------------------------------------------- Pub/Sub Topic Configuration ---------------------------------------------#

variable "pubsub_topic_onix_name" {
  type        = string
  description = "The name of the Pub/Sub topic for onix events."
}

#--------------------------------------------- Service-Specific Infrastructure Provisioning Flags ---------------------------------------------#

variable "provision_adapter_infra" {
  type        = bool
  description = "Set to true to provision infrastructure for the Adapter service."
  default     = true
}

variable "provision_gateway_infra" {
  type        = bool
  description = "Set to true to provision infrastructure for the Gateway service."
  default     = true
}

variable "provision_registry_infra" {
  type        = bool
  description = "Set to true to provision infrastructure for the Registry service."
  default     = true
}

#--------------------------------------------- Adapter Service Configuration ---------------------------------------------#

variable "adapter_ksa_name" {
  type        = string
  description = "Name of the service account for the adapter"
}

variable "adapter_gsa_account_id" {
  type        = string
  description = "The ID of the Adapter GCP Service Account."
}

variable "adapter_gsa_display_name" {
  type        = string
  description = "The display name of the Adapter GCP Service Account."
}

variable "adapter_gsa_description" {
  type        = string
  description = "The description of the Adapter GCP Service Account."
}

variable "adapter_gsa_roles" {
  type        = list(string)
  description = "List of IAM roles to grant to the Adapter GCP Service Account."
}

variable "adapter_topic_name" {
  type        = string
  description = "Name of the Pub/Sub topic for adapter"
}

#--------------------------------------------- Gateway Service Configuration ---------------------------------------------#

variable "gateway_ksa_name" {
  type        = string
  description = "Name for the Gateway Kubernetes Service Account."
}

variable "gateway_gsa_account_id" {
  type        = string
  description = "The ID of the Gateway GCP Service Account."
}

variable "gateway_gsa_display_name" {
  type        = string
  description = "The display name of the Gateway GCP Service Account."
}

variable "gateway_gsa_description" {
  type        = string
  description = "The description of the Gateway GCP Service Account."
}

variable "gateway_gsa_roles" {
  type        = list(string)
  description = "List of IAM roles to grant to the Gateway GCP Service Account."
  default     = []
}

#--------------------------------------------- Registry Service Configuration ---------------------------------------------#

variable "registry_gsa_account_id" {
  type        = string
  description = "The ID of the Registry GCP Service Account."
}
variable "registry_gsa_display_name" {
  type        = string
  description = "The display name of the Registry GCP Service Account."
}
variable "registry_gsa_description" {
  type        = string
  description = "The description of the Registry GCP Service Account."
}
variable "registry_gsa_roles" {
  type        = list(string)
  description = "List of IAM roles to grant to the Registry GCP Service Account."
}

variable "registry_ksa_name" {
  type        = string
  description = "Name of the service account for the registry"
}

# Registry Admin GSA and KSA related variables
variable "registry_admin_gsa_account_id" {
  type        = string
  description = "The ID of the Registry Admin GCP Service Account."
}
variable "registry_admin_gsa_display_name" {
  type        = string
  description = "The display name of the Registry Admin GCP Service Account."
}
variable "registry_admin_gsa_description" {
  type        = string
  description = "The description of the Registry Admin GCP Service Account."
}
variable "registry_admin_gsa_roles" {
  type        = list(string)
  description = "List of IAM roles to grant to the Registry Admin GCP Service Account."
}
variable "registry_admin_ksa_name" {
  type        = string
  description = "Name for the Registry Admin Kubernetes Service Account."
}

# Registry Database Instance Variables (updated with registry_db prefix)
variable "registry_db_instance_region" {
  type        = string
  description = "Region for the DB instance"
}
variable "registry_db_instance_version" {
  type        = string
  description = "Version for the DB instance"
}
variable "registry_db_instance_name" {
  type        = string
  description = "Name of the DB instance"
}
variable "registry_db_instance_tier" {
  type        = string
  description = "Tier for the DB instance"
}
variable "registry_db_instance_labels" {
  type        = map(string)
  description = "Labels for the DB instance"
  default     = {}
}
variable "registry_db_instance_edition" {
  type        = string
  description = "Edition for the DB instance"
}
variable "registry_db_aval_type" {
  type        = string
  description = "Availability type for the DB instance"
}
variable "registry_db_instance_disk_size" {
  type        = string
  description = "Disk size for the DB instance"
}
variable "registry_db_instance_disk_type" {
  type        = string
  description = "Disk type for the DB instance"
}
variable "registry_db_ipv4" {
  type        = bool
  description = "Whether public IPv4 should be enabled for the DB instance"
}

variable "registry_db_max_connections" {
    type = number
    description = "Max Connections Cloud SQL can have for Registry or Admin."
}

variable "registry_db_instance_cache" {
    type = bool
    description = "Whether data cache is to be enabled for registry db instance."
}

variable "registry_database_name" {
  type        = string
  description = "Name of the registry database"
}

variable "auto_approver_subscription_name" {
  description = "The name of subscription that auto approves new and subscription requests for registry."
  type        = string
  default     = null
}

variable "auto_approver_push_url" {
  description = "The url where the auto approver subcription send hits and approves requests"
  type        = string
  default     = ""
}

#--------------------------------------------- Subscription Service Configuration ---------------------------------------------#

variable "subscription_ksa_name" {
  type        = string
  description = "Name for the Subscription Kubernetes Service Account."
}
variable "subscription_gsa_account_id" {
  type        = string
  description = "The ID of the Subscription GCP Service Account."
}
variable "subscription_gsa_display_name" {
  type        = string
  description = "The display name of the Subscription GCP Service Account."
}
variable "subscription_gsa_description" {
  type        = string
  description = "The description of the Subscription GCP Service Account."
}
variable "subscription_gsa_roles" {
  type        = list(string)
  description = "List of IAM roles to grant to the Subscription GCP Service Account."
  default     = []
}

variable "on_subscribe_handler_subscription_name" {
  description = "The name of subscription that that keeps polling registry to see whether the latest subcription is active or not."
  type        = string
  default     = null
}

variable "on_subscribe_handler_push_url" {
  description = "The url where the on_subcriber_handler polls to know the status."
  type        = string
  default     = ""
}

#--------------------------------------------- HTTPS Configuration ---------------------------------------------#

variable "enable_https" {
  type        = bool
  description = "Set to true to enable HTTPS and provision SSL certificate and HTTPS proxy."
  default     = false
}

variable "ssl_certificate_name" {
  type        = string
  description = "Name of the SSL certificate"
  default     = "beckn-managed-ssl-cert"
}

variable "ssl_certificate_description" {
  type        = string
  description = "Description for your SSL certificate"
  default     = "Managed SSL certificate for Beckn Platform"
}

variable "ssl_certificate_domains" {
  type        = list(string)
  description = "List of domains for your SSL certificate"
  default     = []
}

variable "https_proxy_name" {
  type        = string
  description = "Name of the HTTPS proxy"
  default     = "beckn-https-proxy"
}

variable "https_proxy_description" {
  type        = string
  description = "Description for your HTTPS proxy"
  default     = "HTTPS proxy for Beckn Platform"
}

variable "forwarding_rule_name" {
  type        = string
  description = "Name of the forwarding rule"
  default     = "beckn-https-forwarding-rule"
}

variable "forwarding_rule_description" {
  type        = string
  description = "Description for your forwarding rule"
  default     = "Global forwarding rule for HTTPS traffic to Beckn Platform"
}

variable "forwarding_rule_port_range" {
  type        = string
  description = "Port range for the forwarding rule"
  default     = "443"
}
