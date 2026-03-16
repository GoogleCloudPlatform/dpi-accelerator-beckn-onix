output "service_url" {
  description = "The URL of the deployed Agent service"
  value       = module.cloud_run.service_url
}

output "service_name" {
  description = "The name of the deployed Agent service"
  value       = module.cloud_run.service_name
}

output "agent_app_service_account_email" {
  description = "The email of the Agent service account"
  value       = module.agent_service_account.service_account_email
}

output "db_user" {
  description = "The database user for the Agent"
  value       = var.agent_db_user
}

output "db_name" {
  description = "The database name for the Agent"
  value       = var.agent_db_name
}

output "db_secret_name" {
  description = "The Secret Manager secret name for the DB password"
  value       = google_secret_manager_secret.db_password_secret.name
}

output "reasoning_engine_name" {
  description = "The Name of the Reasoning Engine"
  value       = module.vertex_ai.reasoning_engine_name
}

output "reasoning_engine_id" {
  description = "The Resource ID of the Reasoning Engine"
  value       = module.vertex_ai.reasoning_engine_resource_id
}

output "db_password" {
  description = "The database password for the Agent"
  value       = random_password.db_password.result
  sensitive   = true
}

