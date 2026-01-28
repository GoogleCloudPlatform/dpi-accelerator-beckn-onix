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

# Firewall rule resource definition
resource "google_compute_firewall" "first_rule" {

    name = var.firewall_name   
    # Name of the firewall rule (Required)

    description = var.firewall_description    
    # Description of the firewall rule (Optional, defaults to an empty string)

    network = var.vpc_network_name 
    # Name or self-link of the VPC network to which this rule applies (Required)

    priority = var.rule_priority  
    # Priority of the rule (Optional, defaults to 1000)
    # Lower values indicate higher priority (range: 0-65535)

    direction = var.firewall_direction    
    # Direction of traffic (Optional, defaults to "INGRESS")
    # Valid values: "INGRESS" (inbound) or "EGRESS" (outbound)

    # Logging configuration block (Optional)
    log_config {
        
        metadata = var.log_metadata         
        # Metadata to include in logs (Optional, defaults to "INCLUDE_ALL_METADATA")
        # Valid values: "EXCLUDE_ALL_METADATA" or "INCLUDE_ALL_METADATA"
    }

    # Allow rules block (Optional)
    allow {
        
        protocol = var.allow_protocols  
        # Protocol to allow (e.g., "tcp", "udp", "icmp", etc.) (Required if allow block is used)

        
        ports = var.allow_ports     
        # List of ports to allow (Optional, defaults to an empty list)
        # If empty, all ports for the specified protocol are allowed
    }
    
    /**
    # Deny rules block (Optional)
    deny {
    
        protocol = var.denied_protocols      
        # Protocol to deny (e.g., "tcp", "udp", "icmp", etc.) (Required if deny block is used)

    
        ports = var.denied_ports         
        # List of ports to deny (Optional, defaults to an empty list)
        # If empty, all ports for the specified protocol are denied
    }

    **/

/**

    target_service_accounts = var.target_service_accounts     
    # Target service accounts for the rule (Optional, defaults to an empty list)
    # The rule applies to instances associated with these service accounts

    
    target_tags = var.target_tags       
    # Target tags for the rule (Optional, defaults to an empty list)
    # The rule applies to instances with these tags
**/
    
    source_ranges = var.source_ranges     
    # Source IP ranges for the rule (Optional, defaults to an empty list)
    # For INGRESS rules, these are the source IPs that are allowed/denied

/**  
    source_tags = var.source_tags         
    # Source tags for the rule (Optional, defaults to an empty list)
    # For INGRESS rules, these are the source instance tags that are allowed/denied

  
    source_service_accounts = var.source_service_accounts     
    # Source service accounts for the rule (Optional, defaults to an empty list)
    # For INGRESS rules, these are the source service accounts that are allowed/denied

    
    destination_ranges = var.destination_ranges       
    # Destination IP ranges for the rule (Optional, defaults to an empty list)
    # For EGRESS rules, these are the destination IPs that are allowed/denied

**/
}
