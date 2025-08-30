local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import './lib/config.libsonnet';
local panels = import './lib/panels.libsonnet';
local queries = import './lib/queries.libsonnet';
local variables = import './lib/variables.libsonnet';

local row = g.panel.row;

g.dashboard.new('PBS Job')
+ g.dashboard.graphTooltip.withSharedCrosshair()
+ g.dashboard.withDescription('Performance metrics for a PBS Job.')
+ g.dashboard.withTags(config.dashboard.commonTags + ['Job'])
+ g.dashboard.withTimezone(config.dashboard.timezone)
+ g.dashboard.withRefresh(config.dashboard.refresh)
+ g.dashboard.withUid('pbs-private-job')
+ g.dashboard.withVariables([
  variables.datasource,
  variables.username,
  variables.job,
  variables.jobNode,
])
+ g.dashboard.withPanels(
  g.util.grid.wrapPanels(
    g.util.panel.resolveCollapsedFlagOnRows(
      std.prune([
        row.new('Overview')
        + row.withCollapsed(false)
        + row.withPanels(
          g.util.grid.wrapPanels(
            std.prune([
              panels.stat.count(
                'Nodes',
                'Nodes requested in job select statement',
                [queries.requestedNodes]
              ),
              panels.stat.count(
                'CPUs',
                'CPU cores requested in job select statement',
                [queries.requestedCores]
              ),
              panels.stat.bytes(
                'Memory',
                'Memory requested in job select statement',
                [queries.requestedMemory]
              ),
              if config.pbs.gpus then
                panels.stat.count(
                  'GPUs',
                  'GPUs requested in job select statement',
                  [queries.requestedNgpus]
                ),
              if config.pbs.fpgas then
                panels.stat.count(
                  'FPGAs',
                  'FPGAs requested in job select statement',
                  [queries.requestedNfpgas]
                ),
              panels.stat.seconds(
                'Requested Walltime',
                'Requested walltime in job select statement',
                [queries.requestedWalltime]
              ),
              panels.gauge.utilisation(
                'Node Usage',
                'Percentage of nodes actively running tasks for the job',
                [queries.cgroupNodeUsage]
              ),
              panels.gauge.utilisation(
                'CPU Usage',
                'Average utilisation of requested CPUs',
                [queries.cgroupCpuEfficiency]
              ),
              panels.gauge.utilisation(
                'Memory Usage',
                'Average utilisation of requested memory',
                [queries.cgroupMemoryUsageEfficiency]
              ),
              if config.pbs.gpus then
                panels.gauge.utilisation(
                  'GPU Usage',
                  'Average utilisation of requested GPUs',
                  [queries.cgroupGpuUtilisation]
                ),
              if config.pbs.fpgas then
                panels.gauge.utilisation(
                  'FPGA Usage',
                  'Average utilisation of requested FPGAs',
                  [queries.cgroupFpgaUsage]
                ),
              panels.stat.seconds(
                'Remaining Walltime',
                'Remaining walltime for the job',
                [queries.walltimeRemaining]
              ),
            ]), panelWidth=4, panelHeight=4
          ),
        ),
        row.new('Utilisation')
        + row.withCollapsed(false)
        + row.withPanels(
          g.util.grid.wrapPanels(
            std.prune([
              panels.timeseries.percentunit(
                'CPU Usage',
                'Total cpu utilisation of cgroup',
                [queries.cpuUsageUser, queries.cpuUsageSystem, queries.cpuUsageTotal]
              ),
              panels.timeseries.bytes(
                'Memory Usage',
                'Total memory utilisation of cgroup',
                [
                  queries.cgroupMemoryRequested,
                  queries.cgroupMemoryUsed,
                  queries.cgroupMemoryWss,
                  queries.cgroupMemoryRss,
                  queries.cgroupMemoryCache,
                ],
              ),
              if config.pbs.swap then
                panels.timeseries.bytes(
                  'Swap Usage',
                  'Total swap utilisation of cgroup',
                  [queries.cgroupSwapRequested, queries.cgroupSwapUsed]
                ),
              panels.timeseries.base(
                'Memory Page Fault',
                'Total memory page faults of cgroup',
                [queries.cgroupMemoryPgFault, queries.cgroupMemoryPgMajFault]
              ),
              panels.timeseries.bytes(
                'Disk IO',
                'Total disk IO of cgroup',
                [queries.cgroupIoRead, queries.cgroupIoWrite]
              ),
              panels.timeseries.base(
                'Processes',
                'Number of processes in cgroup by node',
                [queries.cgroupProcessCount]
              ),
              panels.timeseries.base(
                'Threads',
                'Number of threads in cgroup by node',
                [queries.cgroupThreadCount]
              ),
            ]), panelWidth=12, panelHeight=10
          ),
        ),
        if config.pbs.gpus then
          row.new('GPU Utilisation')
          + row.withCollapsed(false)
          + row.withPanels([
            panels.timeseries.percent(
              'Compute Utilisation',
              'Percentage utilisation of the GPU for compute tasks',
              [queries.gpuUtilisation]
            ),
            panels.timeseries.percentunit(
              'Tensor Core Utilisation',
              'Percentage utilisation of the GPU for tensor core tasks',
              [queries.gpuTensorCoreUtilisation]
            ),
            panels.timeseries.megabytes(
              'Framebuffer Mem Used',
              'Percentage of framebuffer memory used',
              [queries.gpuFramebufferMemoryUsed]
            ),
            panels.timeseries.power(
              'Power Usage',
              'Power usage of the GPU',
              [queries.gpuPowerUsage]
            ),
          ]),
      ])
    ), panelWidth=12, panelHeight=8
  ),
)
