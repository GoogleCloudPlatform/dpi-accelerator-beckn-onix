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


# Registry GCP Service Account Variables
variable "registry_gsa_account_id" {
  description = "The ID of the Registry GCP Service Account."
  type        = string
}
variable "registry_gsa_display_name" {
  description = "The display name of the Registry GCP Service Account."
  type        = string
}
variable "registry_gsa_description" {
  description = "The description of the Registry GCP Service Account."
  type        = string
}
variable "registry_gsa_roles" {
  description = "List of IAM roles to grant to the Registry GCP Service Account."
  type        = list(string)
}

# Kubernetes Service Account Variables for Registry
variable "registry_ksa_name" {
  description = "Name for the Registry Kubernetes Service Account."
  type        = string
}

# Registry Admin GSA Variables
variable "registry_admin_gsa_account_id" {
  description = "The ID for the Registry Admin GCP Service Account."
  type        = string
}

variable "registry_admin_gsa_display_name" {
  description = "The display name for the Registry Admin GCP Service Account."
  type        = string
}

variable "registry_admin_gsa_description" {
  description = "The description for the Registry Admin GCP Service Account."
  type        = string
  default     = "GCP Service Account for Registry Admin service"
}

variable "registry_admin_gsa_roles" {
  description = "List of IAM roles to bind to the Registry Admin GCP Service Account."
  type        = list(string)
  default     = [] # Add appropriate roles, e.g., ["roles/cloudsql.client", "roles/logging.logWriter"]
}

# Registry Admin KSA Variable
variable "registry_admin_ksa_name" {
  description = "The name for the Registry Admin Kubernetes Service Account."
  type        = string
}

variable "project_id" {
  description = "The GCP project ID."
  type        = string
}

variable "network_name" {
  description = "The name of the VPC network."
  type        = string
}

variable "app_namespace_name" {
  description = "The common Kubernetes namespace name for all services."
  type        = string
}

variable "db_instance_name" {
  description = "The name of the Cloud SQL instance."
  type        = string
}

# Database Variable
variable "registry_database_name" {
  description = "The name of the database within the Cloud SQL instance for Registry."
  type        = string
}
