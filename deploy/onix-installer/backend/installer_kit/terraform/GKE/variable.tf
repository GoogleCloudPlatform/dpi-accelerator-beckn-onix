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

variable "cluster_name" {
  description = "The name of the GKE cluster"
  type        = string
}

variable "cluster_region" {
  description = "The region in which the GKE cluster will be created"
  type        = string
}

variable "terraform_deletion_protection" {
  description = "If 'FALSE' then the cluster can be deleted using terraform destroy"
  type        = bool
  default     = false
}

variable "gcp_filestore_csi_driver_config" {
  description = "Enable or disable the GCP Filestore CSI driver"
  type        = bool
  default     = false
}

variable "network_policy_config" {
  description = "Enable or disable network policy"
  type        = bool
  default     = false
}

variable "gcs_fuse_csi_driver_config" {
  description = "Enable or disable the GCS Fuse CSI driver"
  type        = bool
  default     = true
}

variable "gce_persistent_disk_csi_driver_config" {
  description = "Enable or disable the GCE Persistent Disk CSI driver"
  type        = bool
  default     = false
}

/**
variable "cluster_ipv4_cidr" {
  description = "The IP address range for the pods in this cluster in CIDR notation"
  type        = string
}
**/

variable "cluster_description" {
  description = "Description of the GKE cluster"
  type        = string
}

/**
variable "default_max_pods_per_node" {
  description = "The maximum number of pods per node in this cluster"
  type        = number
}
**/

variable "initial_node_count" {
  description = "The initial number of nodes in the GKE cluster"
  type        = number
}

variable "cluster_secondary_range_name" {
  description = "The name of the secondary range to be used as for the cluster CIDR block"
  type        = string
}

variable "services_secondary_range_name" {
  description = "The name of the secondary range to be used as for the services CIDR block"
  type        = string
}

variable "networking_mode" {
  description = "The networking mode for the GKE cluster"
  type        = string
  default     = "VPC_NATIVE"
}

variable "logging_service" {
  description = "The logging service that the GKE cluster should use"
  type        = string
  default     = "logging.googleapis.com/kubernetes"
}

variable "monitoring_config" {
  description = "The monitoring service that the GKE cluster should use"
  type        = string
  default     = "monitoring.googleapis.com/kubernetes"
  
}

variable "monitoring_service" {
  description = "The monitoring service that the GKE cluster should use"
  type        = string
  default     = "monitoring.googleapis.com/kubernetes"
}

variable "network" {
  description = "The network in which the GKE cluster will be created"
  type        = string
}

variable "subnetwork" {
  description = "The subnetwork in which the GKE cluster will be created"
  type        = string
  
}

variable "workload_pool" {
  description = "The workload pool"
  type        = string
}

variable "remove_default_node_pool" {
  description = "If 'TRUE' then the default node pool will be removed"
  type        = bool
  default     = true
}

variable "master_ipv4_cidr_block" {
  description = "The IP address range for the master in this cluster in CIDR notation"
  type        = string
}

variable "master_global_access_config" {
  description = "The master global access config"
  type        = bool
  default     = true
}

variable "public_cidrs_access_enabled" {
  description = "If 'TRUE' then the cluster will be public"
  type        = bool
  default     = true
}

variable "master_access_cidr_block" {
    description = "The IP address range for the master in this cluster in CIDR notation"
    type        = string
}

variable "display_name" {
    description = "The display name for master access cidr block"
    type        = string
}