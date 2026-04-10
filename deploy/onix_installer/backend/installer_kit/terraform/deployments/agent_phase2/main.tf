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

data "google_project" "project" {
  project_id = var.project_id
}

# ---------------------------------------------------------
# DLQ Topic for the Push Subscription
# ---------------------------------------------------------
resource "google_pubsub_topic" "agent_dlq_topic" {
  count   = var.enable_adapter_integration ? 1 : 0
  name    = "${var.app_name}-adapter-agent-dlq"
  project = var.project_id
}

# Grant Pub/Sub Service Agent publishing rights to the DLQ
resource "google_pubsub_topic_iam_member" "pubsub_dlq_publisher" {
  count   = var.enable_adapter_integration ? 1 : 0
  topic   = google_pubsub_topic.agent_dlq_topic[0].id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# Grant Pub/Sub Service Agent rights to mint OIDC tokens as the Agent Invoker SA
resource "google_service_account_iam_member" "pubsub_token_creator" {
  count              = var.enable_adapter_integration ? 1 : 0
  service_account_id = "projects/${var.project_id}/serviceAccounts/${var.agent_invoker_sa_email}"
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:service-${data.google_project.project.number}@gcp-sa-pubsub.iam.gserviceaccount.com"
}

# ---------------------------------------------------------
# Push Subscription with Transform
# ---------------------------------------------------------
resource "google_pubsub_subscription" "agent_push_subscription" {
  count   = var.enable_adapter_integration ? 1 : 0
  name    = "${var.app_name}-adapter-agent-sub"
  topic   = var.adapter_topic_id
  project = var.project_id

  push_config {
    push_endpoint = var.agent_engine_endpoint_url

    no_wrapper {
      write_metadata = true
    }

    oidc_token {
      service_account_email = var.agent_invoker_sa_email
    }
  }

  message_transforms {
    disabled = false

    javascript_udf {
      function_name = "transformBecknToAgentEngine"
      code          = <<EOF
function transformBecknToAgentEngine(message, metadata) {
  try {
    const data = JSON.parse(message.data);
    const action = data?.context?.action;

    if (!action || !action.startsWith('on_')) {
      return null; 
    }

    const transformedBody = {
      class_method: action,
      input: {
        request: data 
      }
    };

    message.data = JSON.stringify(transformedBody);
    return message;
  } catch (err) {
    return message; 
  }
}
EOF
    }
  }

  dead_letter_policy {
    dead_letter_topic = google_pubsub_topic.agent_dlq_topic[0].id
    max_delivery_attempts = 5
  }

  retry_policy {
    minimum_backoff = "1s"
    maximum_backoff = "15s"
  }

  depends_on = [
    google_pubsub_topic_iam_member.pubsub_dlq_publisher,
    google_service_account_iam_member.pubsub_token_creator
  ]
}
