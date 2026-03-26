# Copyright 2026 Google LLC
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

variable "project_id" {
  type        = string
  description = "The ID of the project in which to provision resources."
}

variable "region" {
  type        = string
  description = "The region in which to provision resources."
}

variable "subnetwork_id" {
  type        = string
  description = "The self-link of the subnetwork where the proxy VM will be deployed."
}

variable "app_name" {
  type        = string
  description = "The application name, used as a prefix for named resources."
}

variable "network_id" {
  type        = string
  description = "The id of the network where the proxy VM will be deployed."
}
