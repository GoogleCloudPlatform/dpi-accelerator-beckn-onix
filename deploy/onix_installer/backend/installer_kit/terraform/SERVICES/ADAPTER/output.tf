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

output "adapter_ksa_name" {
  value       = module.adapter_ksa.ksa_name
  description = "Name of the Adapter Kubernetes Service Account."
}

output "adapter_gsa_email" {
  value       = module.adapter_gsa.service_account_email
  description = "Email of the Adapter GCP Service Account."
}

output "adapter_topic_name" {
  value       = var.adapter_topic_name
  description = "Name of the Adapter Pub/Sub Topic."
}