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

variable "network_name" {
  description = "Name of the VPC"
  type        = string
}

variable "network_description" {
  description = "Description of the VPC"
  type        = string
}

variable "auto_create_subnetworks" {
  description = "Whether to create subnetworks automatically"
  type        = bool
  default     = false
}

variable "routing_mode" {
  description = "The network-wide routing mode to use"
  type        = string
  default     = "REGIONAL"
}

variable "mtu" {
  description = "Maximum Transmission Unit (MTU) of the network"
  type        = number
  default     = 1460
}

# Description: This below block contains the code to create a subnet in the VPC network

variable "subnet_name" {
  description = "Name of the subnet"
  type        = string
}

variable "subnet_description" {
  description = "Description of the subnet"
  type        = string
}

variable "ip_cidr_range" {
  description = "IP CIDR range of the subnet"
  type        = string
}

variable "range_name" {
  description = "Name of the secondary IP range"
  type        = string
}

variable "ip_range" {
  description = "IP CIDR range of the secondary IP range"
  type        = string
}

variable "range_name_1" {
  description = "Name of the secondary IP range"
  type        = string
}

variable "ip_range_1" {
  description = "IP CIDR range of the secondary IP range"
  type        = string
}

variable "private_ip_google_access" {
  description = "Whether VMs in this subnet can access Google services without a public IP address"
  type        = bool
  default     = true
}

variable "region" {
  description = "Region in which the subnet is created"
  type        = string
}

variable "metadata" {
  description = "Metadata logging configuration"
  type        = string
  default     = "INCLUDE_ALL_METADATA"
}