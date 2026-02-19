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

resource "google_compute_backend_service" "backend_service" {

    name = var.backend_name 
    # Name of the backend
    
    protocol = var.backend_protocol 
    #  "HTTP" #  The protocol this BackendService uses to communicate with backends. The default is HTTP. Possible values are: HTTP, HTTPS, HTTP2, TCP, SSL, GRPC, UNSPECIFIED.
    
    description = var.backend_description
    # "Backend service for beckn" # An optional description of this resource. Provide this property when you create the resource.

    # port_name = var.backend_port_name
    
    timeout_sec = var.backend_timeout_sec  
    #  "30"  # How long to wait for the backend service to respond before considering it a failed request
    
    #ip_address_selection_policy = "IPV4_ONLY" IP address selection policy is not allowed for GLOBAL HTTP load balancers with EXTERNAL load balancing scheme.
    
    load_balancing_scheme = var.load_balancing_scheme 
    #  "EXTERNAL"  # Indicates whether the backend service will be used with internal or external load balancing.
    # Default value is EXTERNAL. Possible values are: EXTERNAL, INTERNAL_SELF_MANAGED, INTERNAL_MANAGED, EXTERNAL_MANAGED.

    #for_each = google_compute_network_endpoint_group.neg

    backend {
      
      group = var.group_1 #"projects/$Project_Name/zones/$Zone/networkEndpointGroups/$NEG_Name"  # Referencing the NEG for each zone
      #group = "projects/${var.project_id}/global/networkEndpointGroups/test-neg"

      balancing_mode = var.backend_balancing_mode
      #"RATE" # Specifies the balancing mode for this backend.  Default value is UTILIZATION. Possible values are: UTILIZATION, RATE, CONNECTION. NA for Internet NEG

      #max_connections_per_endpoint = "300" 
      # The max number of simultaneous connections that a single backend network endpoint can handle. NA for Internet NEG

      max_rate_per_endpoint = var.max_rate_per_endpoint
      # The max number of simultaneous connections that a single backend network endpoint can handle. NA for Internet NEG

      capacity_scaler = var.capacity_scaler
      #"1" # A multiplier applied to the group's maximum servicing capacity (based on UTILIZATION, RATE or CONNECTION). Default value is 1, which means the group will serve up to 100% of its configured capacity (depending on balancingMode). 

    }

    backend {
      
      group = var.group_2 #"projects/$Project_Name/zones/$Zone/networkEndpointGroups/$NEG_Name"  # Referencing the NEG for each zone
      #group = "projects/${var.project_id}/global/networkEndpointGroups/test-neg"

      balancing_mode = var.backend_balancing_mode
      #"RATE" # Specifies the balancing mode for this backend.  Default value is UTILIZATION. Possible values are: UTILIZATION, RATE, CONNECTION. NA for Internet NEG

      #max_connections_per_endpoint = "300" 
      # The max number of simultaneous connections that a single backend network endpoint can handle. NA for Internet NEG

      max_rate_per_endpoint = var.max_rate_per_endpoint
      # The max number of simultaneous connections that a single backend network endpoint can handle. NA for Internet NEG

      capacity_scaler = var.capacity_scaler
      #"1" # A multiplier applied to the group's maximum servicing capacity (based on UTILIZATION, RATE or CONNECTION). Default value is 1, which means the group will serve up to 100% of its configured capacity (depending on balancingMode). 

    }

    backend {
      
      group = var.group_3 #"projects/$Project_Name/zones/$Zone/networkEndpointGroups/$NEG_Name"  # Referencing the NEG for each zone
      #group = "projects/${var.project_id}/global/networkEndpointGroups/test-neg"

      balancing_mode = var.backend_balancing_mode
      #"RATE" # Specifies the balancing mode for this backend.  Default value is UTILIZATION. Possible values are: UTILIZATION, RATE, CONNECTION. NA for Internet NEG

      max_rate_per_endpoint = var.max_rate_per_endpoint
      # The max number of simultaneous connections that a single backend network endpoint can handle. NA for Internet NEG

      # max_rate_per_endpoint = var.backend_mrp_endpoint  
      # "300" # The max requests per second that a single backend network endpoint can handle. NA for Internet NEG

      capacity_scaler = var.capacity_scaler
      #"1" # A multiplier applied to the group's maximum servicing capacity (based on UTILIZATION, RATE or CONNECTION). Default value is 1, which means the group will serve up to 100% of its configured capacity (depending on balancingMode). 

    } 

    health_checks = var.health_check

    log_config {
      enable = var.log_config_enable 
      #Whether to enable logging for the load balancer traffic served by this backend service.
    }

    #security_policy = var.security_policy #"projects/{$var.project_id}/global/securityPolicies/$Security_Policy_Name"

    # The ID of the Google Cloud Armor security policy to be attached to the backend service.
    # This is optional and enables WAF, Geo-filtering, and Rate Limiting when configured.
    security_policy = var.security_policy

}