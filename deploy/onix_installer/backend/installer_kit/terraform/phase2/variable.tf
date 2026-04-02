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

variable "project_id" {
  type        = string
  description = "The project ID to deploy resources"
}

variable "region" {
  type        = string
}

variable "global_ip_address" {
  type = string
}

variable "url_map" {
  type        = string
  description = "The name of the URL map"
}


variable "dns_zone_name" {
  type = string
  default = ""
}

variable "subdomains" {
  type        = list(string)
  description = "List of domains which will be provisioned A records"
  default     = []
}

#--------------------------------------------- HTTPS Configuration ---------------------------------------------#

variable "enable_https" {
  type        = bool
  description = "Set to true to enable HTTPS and provision SSL certificate and HTTPS proxy."
  default     = false
}

variable "ssl_certificate_name" {
  type        = string
  description = "Name of the SSL certificate"
}

variable "ssl_certificate_description" {
  type        = string
  description = "Description for your SSL certificate"
}

variable "ssl_certificate_domains" {
  type        = list(string)
  description = "List of domains for your SSL certificate"
  default     = []
}

variable "https_proxy_name" {
  type        = string
  description = "Name of the HTTPS proxy"
}

variable "https_proxy_description" {
  type        = string
  description = "Description for your HTTPS proxy"
  default     = "HTTPS proxy for Beckn Platform"
}

variable "forwarding_rule_name" {
  type        = string
  description = "Name of the forwarding rule"
}

variable "forwarding_rule_description" {
  type        = string
  description = "Description for your forwarding rule"
}

variable "forwarding_rule_port_range" {
  type        = string
  description = "Port range for the forwarding rule"
  default     = "443"
}

variable "pubsub_topic_onix_name" {
  type        = string
}

variable "enable_subscriber" {
  type        = bool
}

variable "on_subscribe_handler_subscription_name" {
  description = "The name of subscription that that keeps polling registry to see whether the latest subcription is active or not."
  type        = string
  default     = ""
}

variable "on_subscribe_handler_push_url" {
  description = "The url where the on_subcriber_handler polls to know the status."
  type        = string
  default     = ""
}

variable "enable_auto_approver" {
  type        = bool
}

variable "auto_approver_subscription_name" {
  description = "The name of subscription that auto approves new and subscription requests for registry."
  type        = string
  default     = ""
}

variable "auto_approver_push_url" {
  description = "The url where the auto approver subcription send hits and approves requests"
  type        = string
  default     = ""
}

variable "onix-url-map-1-http-redirect" {
  description = "onix-url-map-1-http-redirect"
  type        = string
  default     = ""
}
