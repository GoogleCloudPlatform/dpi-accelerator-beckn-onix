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

resource "google_compute_global_address" "vpc_peering_ip" {
    name = var.vpc_peering_ip_name
    purpose = var.vpc_peering_ip_purpose
    address_type = var.vpc_peering_ip_address_type
    prefix_length = var.vpc_peering_prefix_length
    network = var.vpc_peering_ip_network
}