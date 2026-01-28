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

resource "google_compute_url_map" "url_map" {

    # URL map is a resource that defines the mapping of URLs to backend services.
    # URL maps are used in external HTTP(S) load balancers and SSL proxy load balancers.

    name = var.url_map_name 
    # URL map name

    default_service = var.backend_service_id
    # The default backend service to which traffic is directed if none of the hostRules match.

    description = var.url_map_description
    # An optional description of this resource.
}