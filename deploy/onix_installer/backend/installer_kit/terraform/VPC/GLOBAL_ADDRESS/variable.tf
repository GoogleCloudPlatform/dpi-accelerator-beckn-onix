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

variable "vpc_peering_ip_name" {
    type = string
    description = "The name of the global address resource."
}

variable "vpc_peering_ip_purpose" {
    type = string
    description = "The purpose of the global address resource."
}

variable "vpc_peering_ip_address_type" {
    type = string
    description = "The type of address to reserve."
}

variable "vpc_peering_ip_network" {
    type = string
    description = "The network that the global address is reserved for."
}

variable "vpc_peering_prefix_length" {
    type = string
    description = "The prefix length if the purpose is VPC_PEERING."
}
