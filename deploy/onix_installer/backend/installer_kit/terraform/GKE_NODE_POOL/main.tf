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

resource "google_container_node_pool" "primary_nodes" {
    
    cluster = var.cluster_name 
    # Name of the cluster

    name = var.node_pool_name # Name for your node-pool
    
    
    location = var.node_pool_location  
    #  Enter the region you want to create the node pool. Same ad the cluster's region
    
    project = var.project_id 
    # Project ID where the cluster is located 
    
    node_locations = var.reg_node_location  
    #  If the cluster is regional enter the specified zone but if the cluster is zonal then omit the cluster's zone
    
    max_pods_per_node = var.max_pods_per_node 
    # The maximum number of pods per node in this node pool

    node_config {
      disk_size_gb = var.disk_size 
      #Size of the disk attached to each node, specified in GB. The smallest allowed disk size is 10GB. Defaults to 100GB.
      
      disk_type = var.disk_type
      # Type of the disk attached to each node (e.g. 'pd-standard', 'pd-balanced' or 'pd-ssd'). If unspecified, the default disk type is 'pd-standard'
      
      image_type = var.image_type 
      # Important The image type to use for this node. Note that changing the image type will delete and recreate all nodes in the node pool.
      
      enable_confidential_storage = var.enable_confidential_storage 
      # Whether to enable Confidential VM boot disk
      
      labels = var.pool_labels  
      # Enter the label to add with the pool
      
      metadata = {
        "serial_port_logging" = "false"
      }
      
      machine_type = var.machine_type 
      # Enter the type of machine
      
      service_account = var.node_service_account != "" ? var.node_service_account : null 
      # Enter the service account
      
      oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"] #Enter the api's you need to enable  It is recommended that you set service_account to a non-default service account and grant IAM roles to that service account for only the resources that it needs.
    
    }

    initial_node_count = var.node_count 
    # Enter the initial number of nodes in each zone

    autoscaling {

      min_node_count = var.min_node_count 
      # Enter the minimum number of nodes

      max_node_count = var.max_node_count 
      # Enter the maximum node count
    
    }

    lifecycle {
      ignore_changes = [ 
        node_config[0].metadata,
        node_config[0].labels,
        node_config[0].disk_size_gb,
        node_config[0].disk_type,
        node_config[0].image_type,
        node_config[0].machine_type,
        node_config[0].service_account,
        node_config[0].oauth_scopes,
        #node_config[0].enable_secure_boot,
        #node_config[0].enable_integrity_monitoring,
        #node_config[0].enable_confidential_storage,
        #node_config[0].shielded_instance_config,
        #node_config[0].boot_disk_kms_key,
        #node_config[0].boot_disk_type,
        #node_config[0].boot_disk_size_gb,
        #node_config[0].local_ssd_count,
        #node_config[0].local_ssd_interface,
        #node_config[0].tags,
        #node_config[0].preemptible,
        #node_config[0].accelerators,
        #node_config[0].sandbox_config,
        #node_config[0].linux_node_config,
        #node_config[0].windows_node_config,
        #node_config[0].workload_metadata_config,
        #node_config[0].taints,
        #node_config[0].shielded_instance_config,
        #node_config[0].boot_disk_kms_key,
       ]
    }
}