# Prober and monitoring

**PMAP is not a official Google product.**

Monitoring consists of two parts: prober and alerting.

## Prober

[Prober](../prober/) is a tool that constantly probes pmap services to check if
the services are up. It is built as a go binary and deployed as [cloud run
job](https://cloud.google.com/run/docs/overview/what-is-cloud-run#jobs), and
triggered by [cloud scheduler](https://cloud.google.com/scheduler) to constantly
probing pmap services to check if the services are up.

In each prober execution, the prober will cover two CUJs:

- Import resource mapping to BigQuery from GCS
- Import policies to BigQuery from GCS

For each CUJ, prober will upload a yaml file to
[gcs](https://cloud.google.com/storage), and query the bigquery table to see if
a corrosponding entry exists in bigquery table. The job is considered as success
on if the following requirements for both mapping and policy service are met:

1. The object is uploaded.
2. There is an entry in bigquery table that matches the traceID.

### Limitation

Ideally the prober should cover the full user journey which includes checking in
resource mappings and policies in a GitHub repository. However, there is no
effective way to automate PR authoring and bypass branch protection. As a result,
we seek the next closest journey - upload probing files to GCS bucket directly.

## Monitoring and Alert Policies

We also use [cloud run
metircs](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-run) to monitor
and send alert base prober job execution result. The metric used is:

- job/completed_execution_count

We also use [pubsub
metrics](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub) to
monitor pubsub subsrciption and sent out alerts. Pubsub messages plays a vital role in pmap services, as they are used to trigger mapping and policy
services, and write processing result into bigquery. The metrics used are:

- subscription/oldest_unacked_message_age
- subscription/dead_letter_message_counts

## Installation

### Default setup

If you used the e2e module to set up pmap service, then you don't need to set up
prober separately. Otherwise, you can use the following terraform code to set up
your prober. All these variables are required.

```terraform
module "prober_and_monitoring" {
  source = "../modules/monitoring"

  project_id = var.project_id

  prober_bucket_id           = "<YOUR_GCS_bucket_id_for_where_objects_are_uploaded to>"
  prober_bigquery_dataset_id = "<YOUR_BIGQUERY_DATASET_ID_where_prober_run_queries_from>"
  prober_mapping_table_id    = "<YOUR_BIGQUERY_TABLE_ID_which_stores_the_resource_mapping_result>"
  prober_policy_table_id     = "<YOUR_BIGQUERY_TABLE_ID_which_stores_the_policy_result>"
  pmap_prober_image          = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/pmap-prober:0.0.4-amd64" # change image version
  notification_channel_email = "<Email_ADDRESS_to_which_alert_will_be_sent_to>"
  pmap_subscription_ids      = "<YOUR_SUBSCRIPTION_ID_which_you_want_to_monitor_on>"
}
```

### Customization

By default, alerting is disabled, you can enable it by setting the following
variables:

```terraform
alert_enabled = true
```

You can also change threshold to your desired value. An example would be:

```terraform
prober_alert_threshold                           = 1
oldest_unacked_messages_age_threshold_in_seconds = 7200
num_of_undeliverable_messages_threshold          = 20
```

Prober's trigger frequency can be update by `prober_scheduler` variable. For
more information on how to set the frequency, you can refer to [cron job format
and time
zone](https://cloud.google.com/scheduler/docs/configuring/cron-job-schedules?&_ga=2.26495481.-578386315.1680561063#defining_the_job_schedule.).

```terraform
prober_scheduler = "*/10 * * * *"
```

To add more alerting policies for pmap service, you can do so by adding code to
[alert.tf](../terraform/modules/monitoring/alert.tf)
