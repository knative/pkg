# Prow config

This directory contains the config for our
[Prow](https://github.com/kubernetes/test-infra/tree/master/prow) instance.

- `Makefile` Commands to interact with the Prow instance regarding configs and
  updates.
- `boskos` Configuration for the Boskos instance.
- `cluster.yaml` Configuration of the Prow cluster.
- `config.yaml` Generated configuration of the Prow jobs.
- `config_knative.yaml` Input configuration for `make_config.go` to generate
  `config.yaml`.
- `config_start.yaml` Initial, empty configuration for Prow.
- `make_config.go` `periodic_config.go` `testgrid_config.go` Tool that generates
  `config.yaml` from `config_knative.yaml`.
- `plugins.yaml` Configuration of the Prow plugins.
- `run_job.sh` Convenience script to start a Prow job from command-line.
