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
  description = "The project ID"
  type        = string
}

variable "region" {
  description = "The region"
  type        = string
}

variable "app_name" {
  description = "The name of the application (e.g., be-agent)"
  type        = string
}

variable "image_url" {
  description = "The URL of the pre-built Agent image"
  type        = string
}

variable "network_name" {
  description = "The name of the VPC network"
  type        = string
}

variable "subnet_name" {
  description = "The name of the subnetwork"
  type        = string
}


variable "db_instance_name" {
  description = "The name of the Cloud SQL instance"
  type        = string
}

variable "agent_db_name" {
  description = "The name of the database"
  type        = string
}

variable "agent_db_user" {
  description = "The database user"
  type        = string
}


# IAM Variables
variable "agent_sa_account_id" {
  description = "The account ID for the Agent Service Account"
  type        = string
}

variable "agent_sa_display_name" {
  description = "The display name for the Agent Service Account"
  type        = string
}

variable "agent_sa_description" {
  description = "The description for the Agent Service Account"
  type        = string
}

variable "agent_sa_roles" {
  description = "List of IAM roles to assign to the Agent Service Account"
  type        = list(string)
}

# Cloud Run Configuration
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
