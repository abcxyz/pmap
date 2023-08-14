# pmap

**This is not an official Google product.**

## Background

Privacy data management is the process of collecting, storing, using, and
disposing of data in a way that protects the privacy of users. It
is a critical part of any organization that collects or uses user data, and
organizations are typically required to maintain compliance with policies set by
regulatory bodies.

To ensure that organizations maintain compliance with policies set by regulatory
bodies, they need to know the following:

* **The requirements for what teams
  must do, driven by legal requirements or external commitments (aka. 
  policy and compliance controls).** This includes translating the comprehensive
  external legal requirements into requirements that are tailored to products and
  services of the organizations.

* **Where the user data is stored or processed.** This includes understanding the
  different systems and databases that store or process user data, as well as the
  physical locations where user data is stored or processed.

* **Which policy or compliance control applies to
  the system that stores/processes user data (aka. data mapping).** This includes
  understanding how the organization's policies and compliance control are applied to
  different systems and databases.

* **The visibility of the privacy compliance.** This includes
  being able to track and monitor the organization's compliance with its policies/controls
  and applicable laws and regulations.
  PMAP provides a solution for the first three problems. We are working on a
  solution to provide visibility of privacy compliance in the near future.

## Architecture

![pmap architecture](./docs/assets/arch.png)

*   **Registration** - Data owners and policy owners will register data mappings and
    policies/controls in a central GitHub repository.
*   **GCS Snapshots** - Snapshot the data mappings and policies/controls from GitHub
    to GCS with Workload Identity Federation.
*   **Additional Processors** - Extension point of validation and enrichment for data mappings.
*   **Processing Service** - The service that is responsible for ingesting,
    validating and storing the data mappings and policies/controls .
*   **Storage and Analysis** - The data warehouse for processed data mappings and
    policies/controls , and UI for dashboarding.

### Why GitHub

We choose GitHub as it can preserve change history and enable multi-person
review and approval. Change history and review/approval process are crucial in
privacy data management.

### Why BigQuery

We choose BigQuery for its excellent analytics support:
* Be able to visualize
  data to reveal meaningful insights.
* Be able to join data from other
  data sources in the future to achieve the privacy compliance monitoring.

## Set Up

The central privacy/compliance eng team need to complete the steps below.

### Workload Identity Federation

Set up
[Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation),
and a service account with adequate condition and permission, see guide
[here](https://github.com/google-github-actions/auth#setting-up-workload-identity-federation).

```
-  Service account used in Authenticating via Workload Identity Federation
   needs [roles/storage.objectCreator]
   to snapshot the data mappings and policies/controls from GitHub to GCS.
```

### GitHub Central Repository

The central privacy/compliance eng team can determine how to group data
mappings and policies/controls as long as at least one level of group are needed (sub
folders in the root of the central GitHub repository are needed). Files
containing the data mappings or policies/controls can’t be stored directly in the
root of the central GitHub repository.

#### Data Mapping

*   Presubmit workflows for sanity checks, see example
    [here](docs/example/workflows/data_mapping_validation.yaml).

*   Postsubmit workflows to snapshot added_files and modified_files of
    data mappings to GCS, see example
    [here](docs/example/workflows/snapshot_data_mapping_change.yaml).

*   Cron Workflows to snapshot the all files of data mappings to GCS,
    see example [here](docs/example/workflows/snapshot_data_mapping_copy.yaml).

#### Policy and Control

*   Postsubmit workflows to snapshot added_files and modified_files of
    policies/controls to GCS, see example
    [here](docs/example/workflows/snapshot_policy_change.yaml).

*   Cron Workflows to snapshot the all files of policies/controls to GCS, see
    example [here](docs/example/workflows/snapshot_policy_copy.yaml)

### Infrastructure for pmap

* You can use the provided Terraform module to setup the basic infrastructure
needed for this service. Otherwise you can refer to the provided module to see
how to build your own Terraform from scratch.

```terraform
module "pmap" {
  source = "git::https://github.com/abcxyz/pmap.git//terraform/e2e?ref=main" # this should be pinned to the SHA desired

  project_id = "YOUR_PROJECT_ID"

  gcs_bucket_name                  = "pmap"
  pmap_container_image             = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/pmap:0.0.4-amd64"
  pmap_prober_image                = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/pmap-prober:0.0.4-amd64"
  bigquery_table_delete_protection = true
  # This is used when searching global Cloud Resources like GCS bucket.
  pmap_specific_envvars            = { "PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE" : "YOUR_DEFAULT_RESOURCE_SCOPE" }
  notification_channel_email       = "YOUR_NOTIFICATION_CHANNEL_EMAIL"
}
```

* Make sure the Service Account used in the Cloud Run service for
Data Mapping is granted the `roles/cloudasset.viewer` to the corresponding
scope `PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE` level 
following docs [here](https://cloud.google.com/iam/docs/granting-changing-revoking-access#grant-single-role).

```sh
# Grep the Service Account used in the Cloud Run service for Data Mapping 
gcloud run services describe <NAME_OF_DATA_MAPPING_CLOUD_RUN_SERVICE> 
```


## End User Workflows

### Policy/Control Owner

*    Create a policy/control (e.g. a wipeout plan) by opening a PR and add a `yaml`
     file under the sub folder where
     stores
     all the policies/controls. See example
     [here](docs/example/wipeout_plan.yaml).

### Data Owner

*   Register and annotate resources to associate the resources to its specific
    policies/controls by opening a PR and add a mapping `yaml` file under the sub folder where stores
    all the data mappings. The association of
    the resource to the corresponding policies/controls is achieved via `annotations` field.
    See example
    [here](docs/example/resource_mapping.yaml).

### Data Governor(TODO)