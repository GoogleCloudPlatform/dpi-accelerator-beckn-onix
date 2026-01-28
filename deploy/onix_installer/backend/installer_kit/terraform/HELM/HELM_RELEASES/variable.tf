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

variable "helm_name" {
    type = string
    description = "The name of the Helm release"
}


variable "helm_repository" {
    type = string
    description = "The Helm repository to use"
    default = ""
}


variable "helm_namespace" {
    type = string
    description = "The namespace to deploy the Helm release to"
}

variable "helm_chart" {
    type = string
    description = "The Helm chart to deploy"
    default = ""
}

variable "helm_version" {
    type = string
    description = "The version of the Helm chart to deploy"
    default = "4.12.3"
}

variable "timeout" {
    type = string
    description = "The timeout for the Helm release"
    default = "600"
}

variable "helm_values" {
    type = any
    description = "The values to pass to the Helm chart"
    default = []
}