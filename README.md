# pmap

**This is not an official Google product.**

Privacy data mapping and related plans management.

## Architecture

![pmap architecture](./docs/assets/pmap-arch.png)

## End user workflow

1. Pmap service will be triggered when:

   - Data owner merges a PR that triggers github workflow to snapshot added/modifed resource/wipeout plan and upload them to GCS.
   - A cron job that snapshot all resource/wipeout plan and upload them to GCS.

2. Pubsub topic will subscribe to gcs object notifications, and push
the notification to cloud run service to trigger service (as cloud run service) to start enriching and validating data.
3. After service processing data, the result will be sent as notification to pubsub, and wrote to bigquery table.
4. Central privacy team create dashboards base on bigquery and governors review result through the dashboard.
