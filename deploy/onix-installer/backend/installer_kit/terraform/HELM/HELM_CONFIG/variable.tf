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

variable "endpoint" {
    type = string
    description = "The endpoint of the cluster"
}

variable "access_token" {
    type = string
    description = "The access token for the cluster"
}

variable "ca_certificate" {
    type = string
    description = "The CA certificate for the cluster"
}

/**
variable "config_path" {
    type = any
    description = "The path to the kubeconfig file"
}
**/