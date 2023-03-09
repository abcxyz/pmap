output "oidc_service_account" {
  description = <<EOT
        Service Account used for generating the OIDC tokens. Required to enable request
        authentication when messages from Pub/Sub are delivered to push endpoints. If the
        endpoint is a Cloud Run service, this service account needs to be the run invoker.
    EOT
  value       = module.common_infra.oidc_service_account
}

output "oidc_service_account_name" {
  description = <<EOT
        Service Account name used for generating the OIDC tokens. Required to enable request
        authentication when messages from Pub/Sub are delivered to push endpoints. If the
        endpoint is a Cloud Run service, this service account needs to be the run invoker.
    EOT
  value       = module.common_infra.oidc_service_account_name
}

output "run_service_account" {
  description = "Service Account Cloud Run services to run as."
  value       = module.common_infra.run_service_account
}

output "run_service_account_name" {
  description = "Service Account name Cloud Run services to run as."
  value       = module.common_infra.run_service_account_name
}

output "gcs_notification_topics" {
  description = "A map of event to GCS notification Pub/Sub topics."
  value       = module.common_infra.gcs_notification_topics
}

output "bigquery_dataset" {
  description = "BigQuery dataset ID."
  value       = module.common_infra.bigquery_dataset
}

output "bigquery_topics" {
  description = "A map of event to Pub/Sub topics and BigQuery tables."
  value       = module.common_infra.bigquery_topics
}
