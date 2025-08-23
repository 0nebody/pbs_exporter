# PBS Pro Exporter

A Prometheus exporter for realtime job monitoring of PBS Professional HPC clusters. Gathers metrics from PBS job cgroups along with job metadata and node metrics.

This exporter collects:
 - **Node Metrics:** Cluster-wide node status and attributes from `pbsnodes`.
 - **Job Metrics:** Job submission information for each PBS job.
 - **Cgroup Metrics:** CPU, memory, and I/O usage for each job via cgroups. Supports both V1 and V2.

## Usage

Configuration is managed with command-line flags. View command help:

```shell
pbs_exporter --help
usage: pbs_exporter [<flags>]

Flags:
  --[no-]help                      Show context-sensitive help (also try --help-long and --help-man).
  --[no-]cgroup.enabled            Enable cgroup collector.
  --cgroup.root="/sys/fs/cgroup"   Root path of cgroup filesystem hierarchy.
  --[no-]job.enabled               Enable job collector.
  --web.listen-address=":9307"     Address to listen on for web interface and telemetry.
  --[no-]node.enabled              Enable node collector.
  --job.pbs_home="/var/spool/pbs"  PBS home directory
  --[no-]proc.enabled              Enable proc collector.
  --log.level=info                 Only log messages with the given severity or above. One of: [debug, info, warn, error]
  --log.format=logfmt              Output format of log messages. One of: [logfmt, json]
  --[no-]version                   Show application version.
```

The exporter is designed to run in two modes: on compute nodes to gather job-specific data, and on a single node to gather cluster-wide metrics.

### Job Metrics (Compute Node)

Run the exporter on all compute nodes to collect job, and cgroup metrics.

```shell
pbs_exporter
```

### Cluster Metrics (Head/Login Node)

PBS node metrics will be the same from every node and should be collected once or deduplicated. Run the exporter for only node metrics:

```shell
pbs_exporter --node.enabled --no-cgroup.enabled --no-job.enabled --no-proc.enabled
```

## Installation

Binaries can be downloaded from the [Github releases](https://github.com/0nebody/pbs_exporter/releases) page.

## Build Instructions

Download source and build, requires `go` and `make`.

```bash
git clone https://github.com/0nebody/pbs_exporter.git
cd pbs_exporter
make pbs_exporter
```

## Monitoring

### Grafana Dashboards

Pre-built Grafana dashboards can be downloaded along with the exporter from the [Github releases](https://github.com/0nebody/pbs_exporter/releases) page. Basic dashboard modifications can be made with the [configuration file](misc/dashboards/lib/config.libsonnet) and building from Jsonnet.

```bash
make dashboards
```

Dashboards are split into public and private dashboards. Public dashboards filter metrics to display jobs launched by the logged in user. This assumes common usernames between HPC and Grafana users using a shared auth backend.

### GPU Metrics

GPU metrics are not collected by this exporter, but integrates with the [NVIDIA DCGM exporter](https://github.com/NVIDIA/dcgm-exporter). The DCGM exporter requires a PBS hook to map job IDs with assigned GPUs. Configuring DCGM exporter for HPC jobs is documented in the [NVIDIA DCGM repository](https://github.com/NVIDIA/dcgm-exporter?tab=readme-ov-file#how-to-include-hpc-jobs-in-metric-labels).

### Prometheus

An [example Prometheus configuration](misc/prometheus/prometheus.yaml) is available in the repository to help you get started with scraping the exporter.

## Common Issues

### Privilege Requirements

Access to `/proc` and `$PBS_HOME/mom_priv/jobs` requires elevated privileges. This is required when collecting with flags `--job.enabled` and `--proc.enabled` enabled.

```shell
msg Unable to get PID IO pid 1234 err open /proc/1234/io: permission denied
```

Use `setcap 'cap_sys_ptrace=ep cap_dac_read_search=ep' pbs_exporter` to run with minimal elevated privileges.
