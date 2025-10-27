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
    type = string
    description = "Name of the cluster"
}

variable "node_pool_name" {
    type = string
    description = "Name for your node-pool"
}

variable "node_pool_location" {
    type = string
    description = "Enter the region you want to create the node pool. Same ad the cluster's region"
}

variable "project_id" {
    type = string
    description = "Project ID where the cluster is located"
}

variable "reg_node_location" {
    type = list(string)
    description = "If the cluster is regional enter the specified zone but if the cluster is zonal then omit the cluster's zone"
}

variable "max_pods_per_node" {
    type = number
    description = "The maximum number of pods per node in this node pool"
}


variable "disk_size" {
    type = number
    description = "Size of the disk attached to each node, specified in GB. The smallest allowed disk size is 10GB. Defaults to 100GB."
}

variable "disk_type" {
    type = string
    description = "Type of the disk attached to each node (e.g. 'pd-standard', 'pd-balanced' or 'pd-ssd'). If unspecified, the default disk type is 'pd-standard'"
}

variable "image_type" {
    type = string
    description = "The image type to use for this node. Note that changing the image type will delete and recreate all nodes in the node pool."
}

variable "enable_confidential_storage" {
    type = bool
    description = "Whether to enable Confidential VM boot disk"
    default = false
}

variable "pool_labels" {
    type = map(string)
    description = "Enter the label to add with the pool"
}

variable "machine_type" {
    type = string
    description = "Enter the type of machine"
}

variable "node_service_account" {
    type = string
    description = "Enter the service account"
}

variable "node_count" {
    type = string
    description = "Enter the initial number of nodes in each zone"
}

variable "min_node_count" {
    type = string
    description = "Enter the minimum number of nodes"
}

variable "max_node_count" {
    type = string
    description = "Enter the maximum node count"
}