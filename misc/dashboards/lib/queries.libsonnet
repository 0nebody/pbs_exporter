local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local variables = import './variables.libsonnet';
local config = import 'config.libsonnet';

local prometheusQuery = g.query.prometheus;

{
  cpuUsageUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_cpu_user_seconds_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
        / sum by (jobid) (
          pbs_cgroup_cpus{instance=~"$node",jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('User'),

  cpuUsageSystem:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_cpu_system_seconds_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
        / sum by (jobid) (
          pbs_cgroup_cpus{instance=~"$node",jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('System'),

  cpuUsageTotal:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_cpu_usage_seconds_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
        / sum by (jobid) (
          pbs_cgroup_cpus{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Total'),

  cgroupNodeUsage:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          pbs_cgroup_pid_usage{jobid="$jobid"} > 0
        )
        / count(
          pbs_cgroup_pid_usage{jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Nodes Utilised'),

  cgroupCpuEfficiency:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_cpu_usage_seconds_total{jobid="$jobid"}[$__rate_interval]
          )
        )
        / sum by (jobid) (
          pbs_cgroup_cpus{jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('CPU Efficiency'),

  cgroupMemoryRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_limit_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested'),

  cgroupMemoryUsed:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_usage_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Used'),

  cgroupMemoryAnon:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_anon_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Anonymous'),

  cgroupMemoryFile:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_file_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('File'),

  cgroupMemoryShmem:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_shmem_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Shared'),

  cgroupMemoryFileMapped:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_file_mapped_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('File Mapped'),

  cgroupMemoryPgFault:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          rate(
            pbs_cgroup_memory_pgfault_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Page Faults'),

  cgroupMemoryPgMajFault:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          rate(
            pbs_cgroup_memory_pgmajfault_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Major Page Faults'),

  cgroupSwapRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_swap_limit_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested'),

  cgroupSwapUsed:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_swap_usage_bytes{instance=~"$node", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Used'),

  cgroupMemoryUsageEfficiency:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_cgroup_memory_usage_bytes{jobid="$jobid"}
          and on (jobid, runcount)
          pbs_job_info{username="$username", jobid="$jobid"}
        )
        / sum by (jobid) (
          pbs_cgroup_memory_limit_bytes{jobid="$jobid"} > 0
          and on (jobid, runcount)
          pbs_job_info{username="$username", jobid="$jobid"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Memory Usage Efficiency'),

  cgroupIoRead:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_io_read_bytes_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('IO Read'),

  cgroupIoWrite:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          irate(
            pbs_cgroup_io_write_bytes_total{instance=~"$node", jobid="$jobid"}[$__rate_interval]
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('IO Write'),

  cgroupProcessCount:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_cgroup_pid_usage{instance=~"$node", jobid="$jobid"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('{{instance}}'),

  cgroupThreadCount:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_cgroup_thread_usage{instance=~"$node", jobid="$jobid"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('{{instance}}'),

  cgroupGpuUtilisation:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(
          label_replace(
            DCGM_FI_DEV_GPU_UTIL{hpc_job=~"^$jobid.*", instance=~"$node"}, "jobid", "$1", "hpc_job", "([^.]+).*"
          ) and on (jobid, instance)
          pbs_job_info{username="$username"}
        ) / 100
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('GPU Utilization'),

  gpuUtilisation:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        DCGM_FI_DEV_GPU_UTIL{instance=~"$node", hpc_job=~"^$jobid.*"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('GPU Utilization'),

  gpuTensorCoreUtilisation:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        DCGM_FI_PROF_PIPE_TENSOR_ACTIVE{instance=~"$node", hpc_job=~"^$jobid.*"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Tensor Core Utilization'),

  gpuFramebufferMemoryUsed:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        DCGM_FI_DEV_FB_USED{instance=~"$node", hpc_job=~"^$jobid.*"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('GPU Framebuffer Memory Used'),

  gpuPowerUsage:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        DCGM_FI_DEV_POWER_USAGE{instance=~"$node", hpc_job=~"^$jobid.*"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('GPU Power Usage'),

  requestedNodes:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_nodes{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Nodes'),

  requestedCores:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_ncpus{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Cores'),

  requestedMemory:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_memory{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Memory'),

  requestedNfpgas:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_nfpgas{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested FPGAs'),

  requestedNgpus:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_ngpus{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested GPUs'),

  requestedWalltime:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(pbs_job_requested_walltime{jobid="$jobid"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Walltime'),

  walltimeRemaining:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        min(
          pbs_job_requested_walltime{jobid="$jobid"}
          - (
            time()
            - pbs_job_start_time{jobid="$jobid"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Walltime Remaining'),

  cgroupFpgaUsage:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_cgroup_fpga_usage{jobid="$jobid"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('FPGA Utilisation'),

  systemNodesAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          sum by (node) (
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
              and on(node, vnode)
              (pbs_node_state_available{} == 1)
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available Nodes'),

  systemCpuCoresAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_ncpus
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available Cores'),

  systemCpuCoresRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_ncpus{}
            * on (jobid, runcount) group_left(node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Cores'),

  systemMemoryAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_mem_bytes
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available Memory'),

  systemMemoryRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_memory{}
            * on (jobid, runcount) group_left(node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Memory'),

  systemGpusAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_ngpus
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available GPUs'),

  systemGpusRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_ngpus{}
            * on (jobid, runcount) group_left(node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested GPUs'),

  systemFpgasAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_nfpgas
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available FPGAs'),

  systemRunningJobs:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          pbs_job_info{state="R"}
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode) group_left()
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Running Jobs'),

  systemJobsTable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (node) (
          pbs_job_requested_ncpus{}
            * on (jobid, runcount) group_left(node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Jobs')
    + prometheusQuery.withRefId('jobs'),

  systemCpuCoresAvailableTable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (node) (
          pbs_node_ncpus{}
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('CPUs')
    + prometheusQuery.withRefId('cpus'),

  systemMemoryAvailableTable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (node) (
          pbs_node_mem_bytes
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Memory')
    + prometheusQuery.withRefId('memory'),

  systemGpusAvailableTable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (node) (
          pbs_node_ngpus
            * on(node, vnode) group_left(partition, qlist)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
            * on(node, vnode)
            pbs_node_state_available{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('GPUs')
    + prometheusQuery.withRefId('gpus'),

  systemUserTableJobs:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (username) (
          pbs_job_requested_ncpus{}
            * on (jobid, runcount) group_left(username, node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Jobs')
    + prometheusQuery.withRefId('jobs'),

  systemUserTableCpus:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (username) (
          pbs_job_requested_ncpus{}
            * on (jobid, runcount) group_left(username, node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('CPUs')
    + prometheusQuery.withRefId('cpus'),

  systemUserTableMemory:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (username) (
          pbs_job_requested_memory{}
            * on (jobid, runcount) group_left(username, node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Memory')
    + prometheusQuery.withRefId('memory'),

  systemUserTableGpus:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (username) (
          pbs_job_requested_ngpus{}
            * on (jobid, runcount) group_left(username, node, vnode)
            pbs_job_info{queue=~"$queue", state="R"}
            and on (node, vnode)
            pbs_node_info{partition=~"$partition", qlist=~".*$queue.*"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('GPUs')
    + prometheusQuery.withRefId('gpus'),

  userRequestedNodes:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_nodes{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Nodes'),

  userRequestedCores:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_ncpus{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Cores'),

  userRequestedMemory:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_memory{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Memory'),

  userRequestedNfpgas:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_nfpgas{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested FPGAs'),

  userRequestedNgpus:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_ngpus{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested GPUs'),

  userRequestedWalltime:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_walltime{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Walltime'),

  userUtilisedNodes:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          pbs_cgroup_pid_usage{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"} > 0
        ) / count(
          pbs_cgroup_pid_usage{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Nodes'),

  userUtilisedCores:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          irate(
            pbs_cgroup_cpu_usage_seconds_total{}[$__rate_interval]
          ) and on (jobid, runcount)
          pbs_job_info{username="$username"}
        ) / sum(
          pbs_job_requested_ncpus{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Cores'),

  userUtilisedMemory:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        (
          sum(
            pbs_cgroup_memory_usage_bytes{}
            and on(jobid)
            pbs_job_info{username="$username"}
          )
        /
          sum(
            pbs_job_requested_memory{}
            and on(jobid)
            pbs_job_info{username="$username"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Memory'),

  userUtilisedNfpgas:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_job_requested_nfpgas{}
          and on (jobid, runcount)
          pbs_job_info{username="$username"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested FPGAs'),

  userUtilisedNgpus:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(
          label_replace(DCGM_FI_DEV_GPU_UTIL{}, "jobid", "$1", "hpc_job", "([^.]+).*")
          and on (jobid)
          pbs_job_info{username="$username"}
        ) / 100
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested GPUs'),

  userUtilisedWalltime:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          (
            pbs_job_requested_walltime{}
            + on (jobid, runcount)
            pbs_job_start_time{}
            and on (jobid, runcount)
            pbs_job_info{username="$username", state="R"}
          ) - time()
        )
      |||
    )
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested Walltime'),

  userJobTableRunning:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_job_info{username="$username"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Jobs')
    + prometheusQuery.withRefId('jobs'),

  userJobTableRunningStart:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_job_start_time{} * 1000
        and on (jobid, runcount)
        pbs_job_info{username="$username"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Start')
    + prometheusQuery.withRefId('start'),

  userJobTableCompleted:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        last_over_time(
          pbs_job_info{username="$username"}[$__range]
        ) unless on (jobid)
        pbs_job_info{username="$username"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Jobs')
    + prometheusQuery.withRefId('jobs'),

  userJobTableCompletedStart:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        last_over_time(
          pbs_job_start_time{}[$__range]
        ) * 1000 and on (jobid)
        last_over_time(
          pbs_job_info{username="$username"}[$__range]
        ) unless on (jobid)
        pbs_job_info{username="$username"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Start')
    + prometheusQuery.withRefId('start'),

  userJobTableCompletedEnd:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        last_over_time(
          pbs_job_end_time{}[$__range]
        ) * 1000 and on (jobid)
        last_over_time(
          pbs_job_info{username="$username"}[$__range]
        ) unless on (jobid)
        pbs_job_info{username="$username"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('End')
    + prometheusQuery.withRefId('end'),

  nodeJobTableRunning:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_job_info{}
        and on (jobid, runcount)
        pbs_cgroup_cpus{instance=~"$node"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Jobs')
    + prometheusQuery.withRefId('jobs'),

  nodeJobTableRunningStart:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_job_start_time{} * 1000
        and on (jobid, runcount)
        pbs_cgroup_cpus{instance=~"$node"}
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Start')
    + prometheusQuery.withRefId('start'),

  nodeNodesAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          count by (node)(
            pbs_node_state_available{node=~"$node"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available'),

  nodeCpuCoresAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(pbs_node_ncpus{node=~"$node"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available'),

  nodeCpuCoresRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(pbs_cgroup_cpus{instance=~"$node"})
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested'),

  nodeCpuUsageTotal:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          irate(
            pbs_cgroup_cpu_usage_seconds_total{instance=~"$node"}[$__rate_interval]
          )
        ) / scalar(
          sum(
            pbs_cgroup_cpus{instance=~"$node"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Total'),

  nodeCpuUsageUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          irate(
            pbs_cgroup_cpu_user_seconds_total{instance=~"$node"}[$__rate_interval]
          )
        ) / scalar(
          sum(
            pbs_cgroup_cpus{instance=~"$node"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('User'),

  nodeCpuUsageSystem:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          irate(
            pbs_cgroup_cpu_system_seconds_total{instance=~"$node"}[$__rate_interval]
          )
        ) / scalar(
          sum(
            pbs_cgroup_cpus{instance=~"$node"}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('System'),

  nodeMemoryAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_mem_bytes{node=~"$node"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available'),

  nodeMemoryRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1e+18
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested'),

  nodeMemoryUsageTotal:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_usage_bytes{instance=~"$node"}
        ) / scalar(
          sum(
            pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1.0e18
          ) 
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Total'),

  nodeMemoryAnon:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_anon_bytes{instance=~"$node"}
        ) / scalar(
          sum(
            pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1.0e18
          ) 
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Anonymous'),

  nodeMemoryFile:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_file_bytes{instance=~"$node"}
        ) / scalar(
          sum(
            pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1.0e18
          ) 
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('File'),

  nodeMemoryShmem:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_shmem_bytes{instance=~"$node"}
        ) / scalar(
          sum(
            pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1.0e18
          ) 
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Shared'),

  nodeMemoryFileMapped:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_memory_file_mapped_bytes{instance=~"$node"}
        ) / scalar(
          sum(
            pbs_cgroup_memory_limit_bytes{instance=~"$node"} < 1.0e18
          ) 
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('File Mapped'),

  nodeGpusAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_ngpus{node=~"$node"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available'),

  nodeGpusRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count(
          DCGM_FI_DEV_GPU_UTIL{Hostname=~"$node", hpc_job!=""}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Requested'),

  nodeGpuComputeUtilisation:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(
          DCGM_FI_DEV_GPU_UTIL{Hostname=~"$node", hpc_job!=""}
        ) / 100
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Compute'),

  nodeGpuFramebufferUtilisation:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        avg(
          DCGM_FI_DEV_FB_USED{Hostname=~"$node", hpc_job!=""}
          / (
            DCGM_FI_DEV_FB_USED{Hostname=~"$node", hpc_job!=""}
            +
            DCGM_FI_DEV_FB_FREE{Hostname=~"$node", hpc_job!=""}
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Memory'),

  nodeFpgasAvailable:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_node_nfpgas{node=~"$node"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Available'),

  nodeProcessCount:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_pid_usage{instance=~"$node"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Processes'),

  nodeThreadCount:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum(
          pbs_cgroup_thread_usage{instance=~"$node"}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Threads'),

  nodeRequestedWalltime:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        max(
          (
            pbs_job_start_time{}
            + pbs_job_requested_walltime{}
            - time()
          ) and on (jobid, runcount) (
            pbs_job_info{state="R"}
            and on (jobid, runcount) (
              pbs_cgroup_cpus{instance=~"$node"} > 0
            )
          )
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withLegendFormat('Walltime Remaining'),

  local lowMemUtilJobs = |||
    avg by (jobid, username, runcount) (
      avg_over_time(
        (
          pbs_cgroup_memory_usage_bytes{}
          /
          (pbs_cgroup_memory_limit_bytes{} > 0)
          * 100
        )[3h:5m]
      ) < %(mem_low)s
    )
    * on(jobid, runcount) group_left(name, username)
    pbs_job_info{}
    and on (jobid, runcount)
    time() - pbs_job_start_time > %(runtime)s
  ||| % config.thresholds,

  jobLowMemoryUtil:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      lowMemUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Used')
    + prometheusQuery.withRefId('used'),

  jobLowMemoryByUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (username) (
          %(lowMemUtilJobs)s
        )
      ||| % lowMemUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('{{ username }}'),

  jobLowMemoryRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_job_requested_memory{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Req Memory')
    + prometheusQuery.withRefId('requested'),

  local lowCpuUtilJobs = |||
    avg by (jobid, runcount) (
      avg_over_time(
        (
          rate(pbs_cgroup_cpu_usage_seconds_total[$__rate_interval])
          /
          (pbs_cgroup_cpus > 0)
          * 100
        )[3h:5m]
      ) < %(cpu_low)s
    )
    * on(jobid, runcount) group_left(name, username)
    pbs_job_info{}
    and on (jobid, runcount)
    time() - pbs_job_start_time > %(runtime)s
  ||| % config.thresholds,

  jobLowCpuUtil:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      lowCpuUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Used')
    + prometheusQuery.withRefId('used'),

  jobLowCpuByUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (username) (
          %(lowCpuUtilJobs)s
        )
      ||| % lowCpuUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('{{ username }}'),

  jobLowCpuRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_job_requested_ncpus{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Req CPUs')
    + prometheusQuery.withRefId('requested'),

  local lowGpuUtilJobs = |||
    avg by (jobid, instance) (
      label_replace(
        avg_over_time(DCGM_FI_DEV_GPU_UTIL{hpc_job!=""}[3h:5m]) < %(gpu_low)s, "jobid", "$1", "hpc_job", "([^.]+).*"
      )
    )
    * on (jobid, instance) group_left(username, name, queue, runcount)
    pbs_job_info{state="R"}
    and on (jobid, runcount)
    time() - pbs_job_start_time > %(runtime)s
  ||| % config.thresholds,

  jobLowGpuUtil:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      lowGpuUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Used')
    + prometheusQuery.withRefId('used'),

  jobLowGpuByUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (username) (
          %(lowGpuUtilJobs)s
        )
      ||| % lowGpuUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('{{ username }}'),

  jobLowGpuRequested:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        sum by (jobid) (
          pbs_job_requested_ngpus{}
        )
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Req GPUs')
    + prometheusQuery.withRefId('requested'),

  local lowCoreUtilJobs = |||
    (
      sum by (jobid, runcount) (
        rate(pbs_cgroup_cpu_usage_seconds_total[$__rate_interval])
      ) >= 0.90 < 1.01
    )
    * on(jobid, runcount) group_left(username, name, queue)
    pbs_job_info{}
    and on (jobid, runcount)
    (
      pbs_job_requested_ncpus >= 2
    )
    and on (jobid, runcount)
    time() - pbs_job_start_time > %(runtime)s
  ||| % config.thresholds,

  jobLowCoreUtil:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      lowCoreUtilJobs,
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Used')
    + prometheusQuery.withRefId('used'),

  jobLowCoreUtilByUser:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        count by (username) (
          %(lowCoreUtilJobs)s
        )
      ||| % lowCoreUtilJobs
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('{{ username }}'),

  jobTableRunningStart:
    prometheusQuery.new(
      '$' + variables.datasource.name,
      |||
        pbs_job_start_time{} * 1000
      |||
    )
    + prometheusQuery.withEditorMode('code')
    + prometheusQuery.withFormat('table')
    + prometheusQuery.withInstant(true)
    + prometheusQuery.withLegendFormat('Start')
    + prometheusQuery.withRefId('start'),
}
