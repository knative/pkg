# Knative Devstats Site

[Devstats](https://github.com/cncf/devstats) is an open source project used for
collecting metrics for GitHub project(s). Knative has a site running on GKE
which is used for viewing these metrics. The directory contains manifests for
running Devstats on a Kubernetes Cluster.

## Components

- **NFS mount**: We use an NFS mount for storing so that the back and cron jobs
  have a consistent location to pull from.

- **Postgres**: Postgres is the backend database used for storing knative
  metrics. Devstats creates a database called `knative` (the name of the GitHub
  project), which stores all metrics. Consult the
  [devstats documentation](https://github.com/cncf/devstats/blob/master/USAGE.md#database-structure)
  for database schema information.

- **Backfill Job**: A backfill job is ran once on creation to populate the
  database initially. This script can take a few hours. After it runs initially,
  the cron job populates the devstats going forward.

- **Cron Job**: We run devstats hourly through a cron job. This will poll
  metrics for all Knative repos and update the database accordingly.

- **Grafana**: The front end for displaying dashboards. Grafana reads from the
  Postgres database. Dashboards are defined in the
  [grafana/dashboards](grafana/dashboards/knative/) directory.

- **Home Pod**: We allocate a home pod for initial installation. This home pod
  is responsible for configuring the mounted NFS volume and setting up the
  database properly. See [the installation instructions](INSTALL.md) for more
  information.
