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

output "agent_dlq_topic_name" {
  description = "The name of the Dead Letter Queue topic for the Agent Push Subscription"
  value       = length(google_pubsub_topic.agent_dlq_topic) > 0 ? google_pubsub_topic.agent_dlq_topic[0].name : null
}

output "agent_push_subscription_name" {
  description = "The name of the Push Subscription to the Agent Engine"
  value       = length(google_pubsub_subscription.agent_push_subscription) > 0 ? google_pubsub_subscription.agent_push_subscription[0].name : null
}
