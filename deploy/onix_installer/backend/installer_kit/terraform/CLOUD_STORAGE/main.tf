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

resource "google_storage_bucket" "bucket" {
    name = var.bucket_name   
    #Name of your bucket (Required)
    
    location = var.bucket_location  
    #Location of your bucket (Required)
    
    storage_class = var.bucket_storage_class  
    #(Optional, Default: 'STANDARD') The Storage Class of the new bucket. Supported values include: STANDARD, MULTI_REGIONAL, REGIONAL, NEARLINE, COLDLINE, ARCHIVE.
    
    force_destroy = var.destroy_value # (Optional, Default: false) When deleting a bucket, this boolean option will delete all contained objects.
                                      # If you try to delete a bucket that contains objects, Terraform will fail that run.

    versioning {
      enabled = var.versioning_value  #While set to true, versioning is fully enabled for this bucket.
    }

    labels = var.bucket_labels

    uniform_bucket_level_access = var.access_level

}