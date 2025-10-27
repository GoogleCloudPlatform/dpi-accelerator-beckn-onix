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

variable "nat_name" {
    type = string
    description = "Name of the NAT"
}

variable "source_subnetwork_ip_ranges_to_nat" {
    type = string
    description = "List of subnetwork ip ranges to NAT"
    default = "ALL_SUBNETWORKS_ALL_IP_RANGES"
}

variable "router_name" {
    type = string
    description = "Name of the router assigned to the NAT"
}

variable "nat_ip_allocate_option" {
    type = string
    description = "Option to allocate NAT IP"
    default = "AUTO_ONLY"
}

variable "log_enable" {
    type = bool
    description = "Enable logging"
    default = true
}

variable "log_filter" {
    type = string
    description = "Filter for logging"
    default = "ALL"
}

variable "endpoint_types" {
    type = list(string)
    description = "List of types of endpoints to which NAT applies"
    default = ["ENDPOINT_TYPE_VM"]
}

variable "nat_region" {
    type = string
    description = "Region of the NAT"
}

variable "project_id" {
    type = string
    description = "Project ID of the NAT"
}

