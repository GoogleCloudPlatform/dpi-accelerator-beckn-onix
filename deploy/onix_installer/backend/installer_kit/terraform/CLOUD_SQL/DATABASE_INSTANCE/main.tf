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

resource "google_sql_database_instance" "instance" {
    region = var.db_instance_region  #The region the instance will sit in.
    database_version = var.db_instance_version
    # The MySQL, PostgreSQL or SQL Server version to use. Supported values include MYSQL_5_6, MYSQL_5_7, MYSQL_8_0, POSTGRES_9_6,POSTGRES_10, 
    # POSTGRES_11, POSTGRES_12, POSTGRES_13, POSTGRES_14, POSTGRES_15, SQLSERVER_2017_STANDARD, SQLSERVER_2017_ENTERPRISE, SQLSERVER_2017_EXPRESS, 
    # SQLSERVER_2017_WEB. SQLSERVER_2019_STANDARD, SQLSERVER_2019_ENTERPRISE, SQLSERVER_2019_EXPRESS, SQLSERVER_2019_WEB

    name = var.db_instance_name #The name of the instance. If the name is left blank, Terraform will randomly generate one when the instance is first created.
    
    deletion_protection = false #  Whether Terraform will be prevented from destroying the instance.

    settings {
        tier = var.db_instance_tier #  The machine type to use.
        edition = var.db_instance_edition # The edition of the instance, can be ENTERPRISE or ENTERPRISE_PLUS.
        # Data cache only for ENTERPRISE_PLUS editions.
        data_cache_config {
          data_cache_enabled = var.db_instance_cache
        }
        user_labels = var.db_instance_labels # Labels for your SQL Instance
        availability_type = var.db_aval_type # The availability type of the Cloud SQL instance, high availability (REGIONAL) or single zone (ZONAL).
        disk_size = var.db_instance_disk_size #  The size of data disk, in GB. Size of a running instance cannot be reduced but can be increased. The minimum value is 10GB.
        disk_type = var.db_instance_disk_type #  The type of data disk: PD_SSD or PD_HDD. Defaults to PD_SSD.

        ip_configuration {
          ipv4_enabled = var.db_ipv4  #  Whether this Cloud SQL instance should be assigned a public IPV4 address.
          # At least ipv4_enabled must be enabled or a private_network must be configured.
          private_network = var.instance_network #  The VPC network from which the Cloud SQL instance is accessible for private IP. 
          #  For example, projects/myProject/global/networks/default.
          
        }

        database_flags {
          name  = "cloudsql.iam_authentication"
          value = "on" # Enable IAM database authentication
        }
        database_flags {
          name  = "max_connections"
          value = var.db_instance_max_connections
        }

    }    


}