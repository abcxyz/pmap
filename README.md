# pmap

**This is not an official Google product.**

Privacy data mapping and related plans management.

## Set Up

The central privacy team need to complete the steps below.

### Workload Identity Federation

Set up [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation),
and a service account with adequate condition and permission, see guide
[here](https://github.com/google-github-actions/auth#setting-up-workload-identity-federation).

    -  Service account used in Authenticating via Workload Identity Federation
       needs [roles/storage.objectCreator] 
       to snapshot the privacy mapping data/retention plans from GitHub to GCS.

### GitHub Central Repository

The GitHub central repository is the source of truth
for privacy data mappings/wipeout plans.
We rely on GitHub to preserve change history and enable multi-person review.

The central privacy team can determine how to group privacy data mappings/wipeout plans as long as
at least one level of group are needed
(Sub folders in the root of the central GitHub repository are needed).
Files containing privacy data mappings/wipeout plans can’t be stored
directly in the root of the central GitHub repository.

#### Privacy Data Mapping

* Presubmit workflows for sanity checks

```yaml
name: 'Privacy Mapping Data Validation'

on:
  push:
    branches:
      - 'master'
  pull_request:
    branches:
      - 'master'
  workflow_dispatch:

concurrency:
  group: '${{ github.workflow }}-${{ github.head_ref || github.ref }}-privacy-mapping-data-validation'
  cancel-in-progress: true

jobs:
  snapshot:
    uses: 'abcxyz/pmap/.github/workflows/resource-mapping-check.yml@ref=main' #this should be pinned to the SHA desired
    with:
      resource_mapping_directory: 'YOUR_PRIVACY_DATA_MAPPING_SUBFOLDER'
      go_version: '>=1.20.0'
```

* Postsubmit workflows to snapshot added_files and modified_files of privacy data mappings to GCS

```yaml
name: 'snapshot-privacy-mapping-data-change'

on:
  push:
    branches:
      - 'master'
  workflow_dispatch:

# Don't cancel in progress since we don't want to have half-baked file change snapshot.
concurrency: '${{ github.workflow }}-${{ github.head_ref || github.ref }}-snapshot-privacy-mapping-data-change'

jobs:
  snapshot:
    permissions:
      contents: 'read'
      id-token: 'write'
    uses: 'abcxyz/pmap/.github/workflows/snapshot-file-change.yml@ref=main' #this should be pinned to the SHA desired
    with:
      workload_identity_provider: 'YOUR_WORKLOAD_IDENTITY_PROVIDER'
      service_account: 'YOUR_SERVICE_ACCOUNT'
      destination_prefix: 'YOUR_GCS_DESTINATION_PREFIX_FOR_PRIVACY_MAPPING_DATA'
      path: 'YOUR_PRIVACY_DATA_MAPPING_SUBFOLDER'
```

* Cron Workflows to snapshot the all files of privacy data mappings to GCS

```yaml
name: 'snapshot-privacy-mapping-data-copy'

on:
  schedule:
    - cron: 'YOUR_CRON_JOB_FREQUENCY'
  workflow_dispatch:

# Don't cancel in progress since we don't want to have half-baked file change snapshot.
concurrency: '${{ github.workflow }}-${{ github.head_ref || github.ref }}-snapshot-privacy-mapping-data-copy'

jobs:
  snapshot:
    permissions:
      contents: 'read'
      id-token: 'write'
    uses: 'abcxyz/pmap/.github/workflows/snapshot-file-copy.yml@ref=main' #this should be pinned to the SHA desired
    with:
      workload_identity_provider: 'YOUR_WORKLOAD_IDENTITY_PROVIDER'
      service_account: 'YOUR_SERVICE_ACCOUNT'
      destination_prefix: 'YOUR_GCS_DESTINATION_PREFIX_FOR_PRIVACY_MAPPING_DATA'
      path: 'YOUR_PRIVACY_DATA_MAPPING_SUBFOLDER'
```

#### Retention Plan

* Postsubmit workflows to snapshot added_files and modified_files of retention plans to GCS,

```yaml
name: 'snapshot-retention-plan-data-change'

on:
  push:
    branches:
      - 'master'
  workflow_dispatch:

# Don't cancel in progress since we don't want to have half-baked file change snapshot.
concurrency: '${{ github.workflow }}-${{ github.head_ref || github.ref }}-snapshot-retention-plan-data-change'

jobs:
  snapshot:
    permissions:
      contents: 'read'
      id-token: 'write'
    uses: 'abcxyz/pmap/.github/workflows/snapshot-file-change.yml@ref=main' #this should be pinned to the SHA desired
    with:
      workload_identity_provider: 'YOUR_WORKLOAD_IDENTITY_PROVIDER'
      service_account: 'YOUR_SERVICE_ACCOUNT'
      destination_prefix: 'YOUR_GCS_DESTINATION_PREFIX_FOR_RETENTION_PLAN'
      path: 'YOUR_RETENTION_PLAN_SUBFOLDER'
```

* Cron Workflows to snapshot the all files of retention plans to GCS

```yaml
name: 'snapshot-retention-plan-data-copy'

on:
  schedule:
    - cron: 'YOUR_CRON_JOB_FREQUENCY'
  workflow_dispatch:

# Don't cancel in progress since we don't want to have half-baked file change snapshot.
concurrency: '${{ github.workflow }}-${{ github.head_ref || github.ref }}-snapshot-retention-plan-data-copy'

jobs:
  snapshot:
    permissions:
      contents: 'read'
      id-token: 'write'
    uses: 'abcxyz/pmap/.github/workflows/snapshot-file-copy.yml@ref=main' #this should be pinned to the SHA desired
    with:
      workload_identity_provider: 'YOUR_WORKLOAD_IDENTITY_PROVIDER'
      service_account: 'YOUR_SERVICE_ACCOUNT'
      destination_prefix: 'YOUR_GCS_DESTINATION_PREFIX_FOR_RETENTION_PLAN_DATA'
      path: 'YOUR_RETENTION_PLAN_SUBFOLDER'
```

### Infrastructure for pmap

You can use the provided Terraform module to setup the basic infrastructure
needed for this service. Otherwise you can refer to the provided module to see
how to build your own Terraform from scratch.

```terraform
module "pmap" {
  source = "git::https://github.com/abcxyz/pmap.git//terraform/e2e?ref=main" # this should be pinned to the SHA desired

  project_id = "YOUR_PROJECT_ID"

  gcs_bucket_name                  = "pmap"
  pmap_container_image             = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/pmap:0.0.3-amd64"
  bigquery_table_delete_protection = true
  # This is used when searching global Cloud Resources like GCS bucket.
  pmap_specific_envvars            = { "PMAP_MAPPING_DEFAULT_RESOURCE_SCOPE" : "YOUR_DEFAULT_RESOURCE_SCOPE" }
}
```
## End User Workflows

### Data Owners
1. Create a wipeout plan by opening a PR in the yaml format file
   (file has  `.yaml` filename suffix) under the sub folder where stores  
   all the wipeout plans. See example [here](./docs/example/wipeout_plan.yaml).
2. Register and annotate resources to associate the resources to its specific wipeout plan
   by opening a PR in the yaml format file
   (file has  `.yaml` filename suffix) under the sub folder where stores  
   all the privacy data mappings. See example [here](./docs/example/resource_mapping.yaml).
   **NOTE:** The association of the resource to the wipeout plan is achieved via `annotations` field. 
