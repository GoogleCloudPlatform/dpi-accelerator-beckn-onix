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

variable "global_ip_name" {
    type = string
    description = "The name of the global IP address"
}

variable "global_ip_description" {
    type = string
    description = "The description of the global IP address"
}

variable "global_ip_labels" {
    type = map(string)
    description = "The labels to apply to the global IP address"
}

variable "global_ip_version" {
    type = string
    description = "The IP version that will be used by the global IP address"   
    default = "IPV4"
}

variable "global_ip_address_type" {
    type = string
    description = "The type of address to reserve"
    default = "EXTERNAL"
}