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

variable "region" {
  description = "The region"
  type        = string
}

variable "service_name" {
  description = "The name of the Cloud Run service"
  type        = string
}

variable "image_url" {
  description = "The URL of the container image to deploy"
  type        = string
}

variable "env_vars" {
  description = "Map of environment variables to inject into the container"
  type        = map(string)
  default     = {}
}

variable "network_name" {
  description = "The name of the VPC network for Direct VPC Egress"
  type        = string
}

variable "subnet_name" {
  description = "The name of the subnetwork for Direct VPC Egress"
  type        = string
}

variable "service_account_email" {
  description = "The email of the service account to run the service as"
  type        = string
}

variable "cpu" {
  description = "CPU limit for the Cloud Run container"
  type        = string
}

variable "memory" {
  description = "Memory limit for the Cloud Run container"
  type        = string
}

variable "ingress" {
  description = "Ingress traffic configuration for the Cloud Run service"
  type        = string
}

variable "vpc_egress" {
  description = "VPC egress traffic configuration for the Cloud Run service"
  type        = string
}

variable "allow_unauthenticated" {
  description = "Whether to allow unauthenticated access to the service"
  type        = bool
}
