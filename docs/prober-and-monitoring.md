# Prober and monitoring

**PMAP is not a official Google product.**

Monitoring consists of two parts: prober and alerting.

## Prober

[Prober](../prober/) is a tool that constantly probes pmap services to check if
the services are up. It is built as a go binary and deployed to [cloud run
job](https://cloud.google.com/run/docs/overview/what-is-cloud-run#jobs), and is
triggered by [cloud scheduler](https://cloud.google.com/scheduler) to constantly
probing pmap services to check if the services are up.

In each prober job, the prober will cover two CUJ:

- Resource mapping (mapping service user journey)
- Wipeout planning (policy service user journey)

For each CUJ, prober will upload a yaml file to
[gcs](https://cloud.google.com/storage), and query the bigquery table to see if
a corrosponding entry exists in bigquery table. The job is considered as success
on if the following requirements for both mapping and policy service are met:

1. The object is uploaded.
2. There is an entry in bigquery table that matches the traceID.

## Monitoring and Alert Policies

We use [pubsub
metrics](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-pubsub) to
monitor pubsub subsrciption and sent out alerts. The metrics used are:

- subscription/oldest_unacked_message_age
- subscription/dead_letter_message_counts

We also use [cloud run
metircs](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-run) to monitor
and send alert base prober job execution result. The metric used is:

- job/completed_execution_count

## Installation

### Default setup

If you used the e2e module to set up pmap service, then you don't need to set up
prober separately. Otherwise, you can use the following terraform code to set up
your prober. All these variables are required.

```terraform
module "prober_and_monitoring" {
  source = "../modules/monitoring"

  project_id = var.project_id

  prober_bucket_id           = "GCS bucket id for where objects are uploaded to."
  prober_bigquery_dataset_id = "The ID of the bigquery dataset where prober run queries from."
  prober_mapping_table_id    = "The ID of the bigquery table which stores the resource mapping result."
  prober_policy_table_id     = "The ID of the bigquery table which stores the policy result."
  pmap_prober_image          = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/pmap-prober:0.0.4-amd64"
  notification_channel_email = "Email to which alert will be sent to"
  pmap_subscription_ids      = "The subscription ids used in pmap"
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
