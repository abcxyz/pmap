resource "google_project_service" "serviceusage" {
  project = var.project_id

  service            = "serviceusage.googleapis.com"
  disable_on_destroy = false
}
