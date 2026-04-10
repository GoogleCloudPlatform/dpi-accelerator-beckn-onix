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
  type = string
}

variable "app_name" {
  description = "The application name (e.g., be-agent)"
  type        = string
}

variable "enable_adapter_integration" {
  description = "Feature flag to enable Pub/Sub integration between Beckn adapter and Agent Engine"
  type        = bool
  default     = false
}

variable "adapter_topic_id" {
  description = "The Pub/Sub topic ID for the Adapter, usually output from the ONIX module"
  type        = string
  default     = null
}

variable "agent_invoker_sa_email" {
  description = "The email of the Agent Invoker Service Account"
  type        = string
}

variable "agent_engine_endpoint_url" {
  description = "The fully qualified URL endpoint for the Agent Engine Reasoning App"
  type        = string
  default     = null
}
