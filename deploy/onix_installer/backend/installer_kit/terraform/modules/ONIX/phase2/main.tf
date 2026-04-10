# Copyright 2026 Google LLC
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

provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_dns_record_set" "subdomain_records" {
  for_each = toset(var.subdomains)

  name         = "${each.value}."
  managed_zone = var.dns_zone_name
  type         = "A"
  ttl          = 300

  rrdatas = [var.global_ip_address]
}

data "google_project" "project" {
  project_id = var.project_id
}

#--------------------------------------------- HTTPS Configuration ---------------------------------------------#

resource "google_compute_managed_ssl_certificate" "ssl_certificate" {
  count       = var.enable_https ? 1 : 0
  name        = var.ssl_certificate_name
  project     = var.project_id
  description = var.ssl_certificate_description
  managed {
    domains = var.ssl_certificate_domains
  }
  depends_on = [google_dns_record_set.subdomain_records]
}

resource "google_compute_target_https_proxy" "https_proxy" {
  count        = var.enable_https ? 1 : 0
  name         = var.https_proxy_name
  url_map      = var.url_map
  description  = var.https_proxy_description
  ssl_certificates = var.enable_https ? [google_compute_managed_ssl_certificate.ssl_certificate[0].id] : []
  depends_on   = [google_compute_managed_ssl_certificate.ssl_certificate]
}

resource "google_compute_global_forwarding_rule" "https_forwarding_rule" {
  count                 = var.enable_https ? 1 : 0
  name                  = var.forwarding_rule_name
  description           = var.forwarding_rule_description
  ip_address            = var.global_ip_address
  port_range            = var.forwarding_rule_port_range
  target                = var.enable_https ? google_compute_target_https_proxy.https_proxy[0].id : null
  load_balancing_scheme = "EXTERNAL_MANAGED"
}

#--------------------------------------------- HTTP-HTTPS Redirect Configuration ---------------------------------------------#

# URL Map for HTTP redirect
resource "google_compute_url_map" "http_redirect_url_map" {
  name             = var.onix-url-map-1-http-redirect
  description      = "URL map to redirect HTTP to HTTPS"
  default_url_redirect {
    https_redirect       = true
    strip_query          = false
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
  }

}

# HTTP Target Proxy
resource "google_compute_target_http_proxy" "http_proxy" {
  name        = "${var.https_proxy_name}-http"
  url_map     = google_compute_url_map.http_redirect_url_map.id
  description = "HTTP proxy for redirecting to HTTPS"
  depends_on  = [google_compute_url_map.http_redirect_url_map]
}

# Global Forwarding Rule for HTTP (port 80)
resource "google_compute_global_forwarding_rule" "http_forwarding_rule" {
  name                  = "${var.forwarding_rule_name}-http"
  description           = "Forwarding rule for HTTP to HTTPS redirect"
  ip_address            = var.global_ip_address
  port_range            = "80"
  target                = google_compute_target_http_proxy.http_proxy.id
  load_balancing_scheme = "EXTERNAL_MANAGED"
  depends_on            = [google_compute_target_http_proxy.http_proxy]
}

# ---------------------------------------------------------
# IDENTITY 1: For invoking Subscriber service
# ---------------------------------------------------------
resource "google_service_account" "sa_subscriber_invoker" {
  count        = var.enable_inbound_auth && var.enable_subscriber ? 1 : 0
  account_id   = var.subscriber_invoker_sa
  display_name = "Subscriber Invoker Service Account"
  project      = var.project_id
}

resource "google_service_account_iam_member" "pubsub_token_creator_on_subscribe" {
  count              = var.enable_inbound_auth && var.enable_subscriber ? 1 : 0
  service_account_id = google_service_account.sa_subscriber_invoker[0].name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# ---------------------------------------------------------
# IDENTITY 2: For invoking Admin service
# ---------------------------------------------------------
resource "google_service_account" "sa_admin_invoker" {
  count        = var.enable_inbound_auth && var.enable_auto_approver ? 1 : 0
  account_id   = var.admin_invoker_sa
  display_name = "Admin Invoker Service Account"
  project      = var.project_id
}

resource "google_service_account_iam_member" "pubsub_token_creator_auto_approver" {
  count              = var.enable_inbound_auth && var.enable_auto_approver ? 1 : 0
  service_account_id = google_service_account.sa_admin_invoker[0].name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# ---------------------------------------------------------
# IDENTITY 3: For invoking Adapter service
# ---------------------------------------------------------
resource "google_service_account" "sa_adapter_invoker" {
  count        = var.enable_inbound_auth ? 1 : 0
  account_id   = var.adapter_invoker_sa
  display_name = "Adapter Invoker Service Account"
  project      = var.project_id
}

resource "google_pubsub_subscription" "on_subscribe_subscription" {
  count   = var.enable_subscriber ? 1 : 0
  name    = var.on_subscribe_handler_subscription_name
  topic   = var.pubsub_topic_onix_name
  project = var.project_id

  push_config {
    push_endpoint = var.on_subscribe_handler_push_url
    no_wrapper {
      write_metadata = true
    }

    # Conditionally add OIDC Token
    dynamic "oidc_token" {
      for_each = var.enable_inbound_auth ? [1] : []
      content {
        service_account_email = google_service_account.sa_subscriber_invoker[0].email
        audience              = var.subscriber_audience
      }
    }
  }

  filter = "attributes.event_type=\"ON_SUBSCRIBE_RECIEVED\""

  ack_deadline_seconds = "30"

  expiration_policy {
    ttl = ""
  }

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  # Ensure IAM propagation finishes before attaching the SA to the subscription
  depends_on = [
    google_service_account_iam_member.pubsub_token_creator_on_subscribe
  ]
}

resource "google_pubsub_subscription" "auto_approver_subscription" {
  count   = var.enable_auto_approver ? 1 : 0
  name    = var.auto_approver_subscription_name
  topic   = var.pubsub_topic_onix_name
  project = var.project_id

  push_config {
    push_endpoint = var.auto_approver_push_url
    no_wrapper {
      write_metadata = true
    }

    # Conditionally add OIDC Token
    dynamic "oidc_token" {
      for_each = var.enable_inbound_auth ? [1] : []
      content {
        service_account_email = google_service_account.sa_admin_invoker[0].email
        audience              = var.admin_audience
      }
    }
  }

  filter = "attributes.event_type = \"NEW_SUBSCRIPTION_REQUEST\" OR attributes.event_type = \"UPDATE_SUBSCRIPTION_REQUEST\""

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  # Ensure IAM propagation finishes before attaching the SA to the subscription
  depends_on = [
    google_service_account_iam_member.pubsub_token_creator_auto_approver
  ]

  message_transforms {
    disabled = false

    javascript_udf {
      function_name = "approver"
      code          = <<EOF
function approver(message, metadata) {
  const data = JSON.parse(message.data);
  if (!data["message_id"]) {
    return null;
  }
  message.data = JSON.stringify({action: "APPROVE_SUBSCRIPTION", operation_id: data["message_id"]});
  return message;
}
EOF
    }
  }
}

resource "google_iam_workload_identity_pool" "main" {
  count = var.enable_inbound_auth ? 1 : 0
  workload_identity_pool_id = var.pool_id
  display_name               = "Inbound Auth Pool"
}

resource "time_sleep" "wait_for_wi_pool" {
  count = var.enable_inbound_auth ? 1 : 0
  depends_on = [google_iam_workload_identity_pool.main]
  create_duration = "30s"
}

resource "google_iam_workload_identity_pool_provider" "oidc_provider" {
  count = var.enable_inbound_auth ? 1 : 0
  workload_identity_pool_id = google_iam_workload_identity_pool.main[0].workload_identity_pool_id
  workload_identity_pool_provider_id = var.provider_id

  depends_on = [time_sleep.wait_for_wi_pool]

  attribute_condition = "google.subject in [${join(", ", [for v in var.allowed_values : "'${v}'"])}]"
  attribute_mapping = {
    "google.subject" = "assertion.${var.idclaim}"
  }

  oidc {
    issuer_uri = var.issuer_url
    allowed_audiences = var.allowed_audiences
    jwks_json = var.jwks_json != "" ? var.jwks_json : null
    }
}
