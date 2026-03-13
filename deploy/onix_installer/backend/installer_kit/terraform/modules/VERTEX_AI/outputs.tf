output "reasoning_engine_name" {
  description = "The ID of the Reasoning Engine"
  value       = google_vertex_ai_reasoning_engine.agent_engine.name
}

output "reasoning_engine_resource_id" {
  description = "The Resource ID of the Reasoning Engine"
  value       = google_vertex_ai_reasoning_engine.agent_engine.id
}
