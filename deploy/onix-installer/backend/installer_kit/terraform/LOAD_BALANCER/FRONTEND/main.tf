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

resource "google_compute_global_forwarding_rule" "forwarding_rule" {
  name = var.frontend_name  
  # Name of the forwarding rule
  
  description = var.frontend_description  
  # Description of the forwarding rule

  ip_address = var.frontend_ip  
  #Reserved Static or Interval #Enter the "global_address" link or IP of existing static IP
  
  target = var.target_proxy_id
  # Target HTTP Proxy for the forwarding rule

  port_range = var.frontend_port  #  "80"  Frontend IP for your LB
    # Port range for the forwarding rule
}