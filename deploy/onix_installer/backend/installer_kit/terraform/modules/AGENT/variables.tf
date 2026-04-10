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

variable "network_id" {
  description = "The id of the network"
  type        = string
}

variable "subnet_name" {
  description = "The name of the subnetwork"
  type        = string
}

variable "subnetwork_id" {
  description = "The self-link of the subnetwork"
  type        = string
}

variable "provision_agent_db" {
  description = "Whether to provision the agent database"
  type        = bool
  default     = true
}

variable "agent_network_attachment_name" {
  description = "The name of the network attachment for the agent"
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

variable "agent_invoker_sa_account_id" {
  description = "The account ID for the Agent Invoker Service Account"
  type        = string
}

variable "datastore_imports" {
  description = "A map of datastore keys to their GCS URI for importing."
  type        = map(string)
  default     = {}
}
