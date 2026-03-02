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

resource "google_container_cluster" "primary_cluster" {

    name = var.cluster_name
    # Name of the cluster

    location = var.cluster_region
    # Enter zone for zonal cluster or region for regional cluster

    deletion_protection = var.terraform_deletion_protection
    # If "FALSE" then the cluster can be deleted using terraform destroy
    # Default is "TRUE"

    node_config {
        disk_size_gb = 20
    }

    addons_config {
        gcp_filestore_csi_driver_config {
            enabled = var.gcp_filestore_csi_driver_config
            # Enable or disable the GCP Filestore CSI driver
            # Default is false
        }
/**
        network_policy_config {
            enabled = var.network_policy_config
            # Enable or disable network policy
        }
**/
         
         gcs_fuse_csi_driver_config {
            enabled = var.gcs_fuse_csi_driver_config
            # Enable or disable the GCS Fuse CSI driver. Set enabled to true to use the GCS Fuse CSI driver.
            # Default is false
         }

         gce_persistent_disk_csi_driver_config {
            enabled = var.gce_persistent_disk_csi_driver_config
            # Enable or disable the GCE Persistent Disk CSI driver. Set enabled to true to use the GCE Persistent Disk CSI driver.
            # Default is false
         }

    }

    #cluster_ipv4_cidr = var.cluster_ipv4_cidr
    # The IP address range for the pods in this cluster in CIDR notation

    description = var.cluster_description
    # Description of the cluster

    #default_max_pods_per_node = var.default_max_pods_per_node
    # The maximum number of pods per node in this cluster

    initial_node_count = var.initial_node_count
    # If you're using google_container_node_pool objects with no default node pool, you'll need to set this to a value of at least 1, alongside setting remove_default_node_pool to true.

    ip_allocation_policy {
        cluster_secondary_range_name = var.cluster_secondary_range_name
        # The name of the secondary range to be used as for the cluster CIDR block. The secondary range will be used for pod IP addresses. This must be an existing secondary range associated with the cluster subnetwork.
        # Both range name and range cannot be defined. Only one of them can be defined.
        # Define the IP address in subnet module

        services_secondary_range_name = var.services_secondary_range_name
        # The name of the secondary range to be used as for the services CIDR block. The secondary range will be used for service ClusterIPs. This must be an existing secondary range associated with the cluster subnetwork.
        # Both range name and range cannot be defined. Only one of them can be defined.
        # Define the IP address in subnet module
    }

    networking_mode = var.networking_mode
    # Options are VPC_NATIVE or ROUTES. VPC_NATIVE enables IP aliasing. Newly created clusters will default to VPC_NATIVE.
    # Default is VPC_NATIVE


    logging_service = var.logging_service
    # The logging service that the cluster should write logs to. Available options include logging.googleapis.com, logging.googleapis.com/kubernetes (beta), and none
    # Default is logging.googleapis.com

    monitoring_service = var.monitoring_service
    # The monitoring service that the cluster should write metrics to. Available options include monitoring.googleapis.com, monitoring.googleapis.com/kubernetes (beta), and none
    # Default is monitoring.googleapis.com

    network = var.network
    # The name or self_link of the Google Compute Engine network to which the cluster is connected. For Shared VPC, set this to the self link of the shared VPC.
    # Define the network in network module

    subnetwork = var.subnetwork
    # The name or self_link of the Google Compute Engine subnetwork in which the cluster's instances are launched.

    workload_identity_config {
        workload_pool = var.workload_pool
        #workload_pool = "${data.google_project.project.project_id}.svc.id.goog"
        # Workload Identity allows Kubernetes service accounts to act as a user-managed Google IAM Service Account.
    }

    remove_default_node_pool = var.remove_default_node_pool
    # If set to true, the cluster's default node pool will be removed. This is useful when you want to manage the default node pool yourself.

    private_cluster_config {

        enable_private_nodes = true
        # If true, all nodes have private IP addresses only and can access the public internet through NAT gateway.

        enable_private_endpoint = false
        # If true, the cluster's private endpoint is used as the cluster endpoint.

        master_ipv4_cidr_block = var.master_ipv4_cidr_block
        # The IP range in CIDR notation to use for the hosted master network. This range will be used for assigning internal IP addresses to the master nodes.

        master_global_access_config {
            enabled = var.master_global_access_config
            # If enabled, master can be accessed from all regions
        }
    }

    master_authorized_networks_config {
      gcp_public_cidrs_access_enabled = var.public_cidrs_access_enabled #default value is false Whether Kubernetes master is accessible via Google Compute Engine Public IPs.
      cidr_blocks {
        cidr_block = var.master_access_cidr_block  #External network that can access Kubernetes master through HTTPS. Must be specified in CIDR notation.
        display_name = var.display_name #Field for users to identify CIDR blocks.
      }
    }
    
}