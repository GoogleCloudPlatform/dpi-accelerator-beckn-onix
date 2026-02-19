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

variable "backend_name" {
    type = string
    description = "Name of the backend"
}

variable "backend_description" {
    type = string
    description = "An optional description of this resource. Provide this property when you create the resource."
  
}

variable "backend_protocol" {
    type = string
    description = "The protocol this BackendService uses to communicate with backends. The default is HTTP. Possible values are: HTTP, HTTPS, HTTP2, TCP, SSL, GRPC, UNSPECIFIED."
    default = "HTTP"
}

variable "backend_timeout_sec" {
    type = string
    description = "How long to wait for the backend service to respond before considering it a failed request"
    default = "30"
}

variable "load_balancing_scheme" {
    type = string
    description = "Indicates whether the backend service will be used with internal or external load balancing. Default value is EXTERNAL. Possible values are: EXTERNAL, INTERNAL_SELF_MANAGED, INTERNAL_MANAGED, EXTERNAL_MANAGED."
    default = "EXTERNAL_MANAGED"
}

variable "security_policy" {
  type        = string
  default     = null
  description = "The ID of the Google Cloud Armor security policy to be attached to the backend service."
}

variable "group_1" {
    type = any
    description = "Referencing the NEG for each zone"
}

variable "group_2" {
    type = any
    description = "Referencing the NEG for each zone"
}

variable "group_3" {
    type = any
    description = "Referencing the NEG for each zone"
}

variable "backend_balancing_mode" {
    type = string
    description = "Specifies the balancing mode for this backend. Default value is UTILIZATION. Possible values are: UTILIZATION, RATE, CONNECTION. NA for Internet NEG"
    default = "RATE"
}


variable "max_rate_per_endpoint" {
    type = string
    description = "The max requests per second that a single backend network endpoint can handle. NA for Internet NEG"
    default = "300"
}


/**
variable "max_rate_per_instance" {
    type = string
    description = "The max number of simultaneous connections that a single backend endpoint can handle. This is used to calculate the capacity of the group. Can only be set if balancingMode is RATE."
    default = "300"
}
**/

variable "capacity_scaler" {
    type = string
    description = "A multiplier applied to the group's maximum servicing capacity (based on UTILIZATION, RATE or CONNECTION). Default value is 1, which means the group will serve up to 100% of its configured capacity (depending on balancingMode)."
    default = "1"
}

variable "health_check" {
    type = list(string)
    description = "Health check configuration for the backend service"
}

variable "log_config_enable" {
    type = bool
    description = "Whether to enable logging for the load balancer traffic served by this backend service."
    default = true
}

/**
variable "security_policy" {
    type = any
    description = "Security policy for the backend service"
}
**/

