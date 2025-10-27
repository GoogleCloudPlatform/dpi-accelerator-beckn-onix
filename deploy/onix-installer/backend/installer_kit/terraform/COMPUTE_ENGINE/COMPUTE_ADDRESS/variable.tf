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

variable "ip_name" {
    type = string
    description = "Name allocated to your IP address"
}

variable "address_type" {
    type = string
    description = "The type of address to reserve. INTERNAL OR EXTERNAL"
    default = "EXTERNAL"
}

variable "description" {
    type = string
    description = "Description for this resource"
}

variable "ip_labels" {
    type = map(string)
    description = "Labels to apply to this address. A list of key->value pairs. Example: {name = \"webserver\"}"
}

variable "ip_network_tier" {
    type = string
    description = "The networking tier used for configuring this address. Should not be used when configuring Internal addresses"
    default = "PREMIUM"
}

variable "ip_version" {
    type = string
    description = "The IP Version that will be used by this address. Default IPv4, Possible Values IPv4, IPv6"
    default = "IPV4"
}

variable "ip_purpose" {
    type = string
    description = "Purpose of the IP"
    default = "GCE_ENDPOINT"
}

/**
variable "ip_region" {
    type = string
    description = "Region where the IP address will be created"
    default = ""
}
**/