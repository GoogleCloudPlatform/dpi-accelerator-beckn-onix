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

variable "instance_name" {
    type = string
    description = "Name of the Redis instance"
}

variable "memory_size_gb" {
    type = string
    description = "Redis memory size in GiB. Minimum of 1GB for BASIC tier and 5GB for STANDARD_HA tier"
}

variable "instance_tier" {
    type = string
    description = "BASIC or STANDARD_HA"
}

variable "instance_region" {
    type = string
    description = "Region where the instance will be created"
}

variable "instance_location_id" {
    type = string
    description = "Zone where the instance will be created"
}

variable "instance_authorized_network" {
    type = string
    description = "Network that the instance will be connected to"
}

variable "instance_connect_mode" {
    type = string
    description = "Possible values are: DIRECT_PEERING, PRIVATE_SERVICE_ACCESS"
}