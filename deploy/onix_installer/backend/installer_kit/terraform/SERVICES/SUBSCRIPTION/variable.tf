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

variable "project_id" {
  description = "The GCP project ID."
  type        = string
}

variable "app_namespace_name" {
  description = "The common Kubernetes namespace name for all services."
  type        = string
}

variable "subscription_ksa_name" {
  description = "Name for the Subscription Kubernetes Service Account."
  type        = string
}

variable "subscription_gsa_account_id" {
  description = "The ID of the Subscription GCP Service Account."
  type        = string
}

variable "subscription_gsa_display_name" {
  description = "The display name of the Subscription GCP Service Account."
  type        = string
}

variable "subscription_gsa_description" {
  description = "The description of the Subscription GCP Service Account."
  type        = string
}

variable "subscription_gsa_roles" {
  description = "List of IAM roles to grant to the Subscription GCP Service Account."
  type        = list(string)
  default     = [] # Default to empty if no specific roles are needed initially
}