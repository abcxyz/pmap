output "prober_service_account" {
  description = "Service Account for prober"
  value       = google_service_account.prober_service_account.email
}

output "prober_service_account_name" {
  description = "Name of the Service Account for prober"
  value       = google_service_account.prober_service_account.name
}
