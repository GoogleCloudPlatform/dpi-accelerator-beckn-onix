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

variable "frontend_name" {
    type = string
    description = "Name of the forwarding rule"
}

variable "frontend_description" {
    type = string
    description = "Description of the forwarding rule"
}

variable "frontend_ip" {
    type = string
    description = "Reserved Static or Interval #Enter the 'global_address' link or IP of existing static IP"
}

variable "target_proxy_id" {
    type = string
    description = "Target HTTP Proxy for the forwarding rule"
}

variable "frontend_port" {
    type = string
    description = "Port range for the forwarding rule"
}