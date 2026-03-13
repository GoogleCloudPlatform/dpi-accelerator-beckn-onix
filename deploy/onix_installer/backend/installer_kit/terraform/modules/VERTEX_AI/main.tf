resource "google_project_service" "vertex_ai_api" {
  service = "aiplatform.googleapis.com"
  disable_on_destroy = false
}

resource "google_vertex_ai_reasoning_engine" "agent_engine" {
  region       = var.region
  display_name = "${var.app_name}-reasoning-engine"
  description  = "Reasoning Engine for ${var.app_name}"

  depends_on = [google_project_service.vertex_ai_api]
}
