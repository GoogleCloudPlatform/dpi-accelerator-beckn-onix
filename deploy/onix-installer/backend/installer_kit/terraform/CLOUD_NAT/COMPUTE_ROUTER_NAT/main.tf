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

resource "google_compute_router_nat" "router_nat" {

    name = var.nat_name
    # Name of the NAT

    source_subnetwork_ip_ranges_to_nat = var.source_subnetwork_ip_ranges_to_nat
    # List of subnetwork ip ranges to NAT

    router = var.router_name
    # Name of the router assigned to the NAT

    nat_ip_allocate_option = var.nat_ip_allocate_option
    # Option to allocate NAT IP

    log_config {
        enable = var.log_enable
        # Enable logging

        filter = var.log_filter
        # Specifies the desired filtering of logs on this NAT. 
        # Possible values are: ERRORS_ONLY, TRANSLATIONS_ONLY, ALL.
    }

    endpoint_types = var.endpoint_types
    # List of types of endpoints to which NAT applies

    region = var.nat_region
    # Region of the NAT

    project = var.project_id
    # Project ID of the NAT

}