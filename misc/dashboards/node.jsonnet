local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import './lib/config.libsonnet';
local panels = import './lib/panels.libsonnet';
local queries = import './lib/queries.libsonnet';
local variables = import './lib/variables.libsonnet';

local row = g.panel.row;

g.dashboard.new('PBS Node')
+ g.dashboard.withUid('pbs-private-node')
+ g.dashboard.withDescription('Overview of nodes in the PBS cluster with requested resources and utilisation.')
+ g.dashboard.withTimezone(config.dashboard.timezone)
+ g.dashboard.withRefresh(config.dashboard.refresh)
+ g.dashboard.withTags(config.dashboard.commonTags + ['Node'])
+ g.dashboard.graphTooltip.withSharedCrosshair()
+ g.dashboard.withVariables([
  variables.datasource,
  variables.node,
])
+ g.dashboard.withPanels(
  g.util.grid.wrapPanels(
    g.util.panel.resolveCollapsedFlagOnRows([
      row.new('Overview')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels(
          std.prune([
            panels.stat.count(
              'Nodes',
              'Total number of selected nodes',
              [queries.nodeNodesAvailable]
            ),
            panels.stat.count(
              'CPUs',
              'Total number of CPUs available to PBS from selected nodes',
              [queries.nodeCpuCoresAvailable]
            ),
            panels.stat.bytes(
              'Memory',
              'Total amount of memory available to PBS from selected nodes',
              [queries.nodeMemoryAvailable]
            ),
            if config.pbs.gpus then
              panels.stat.count(
                'GPUs',
                'Total number of GPUs available to PBS from selected nodes',
                [queries.nodeGpusAvailable]
              ),
            if config.pbs.fpgas then
              panels.stat.count(
                'FPGAs',
                'Total number of FPGAs available to PBS from selected nodes',
                [queries.nodeFpgasAvailable]
              ),
            panels.stat.seconds(
              'Walltime',
              'Remaining walltime of jobs running on selected nodes until nodes are drained',
              [queries.nodeRequestedWalltime]
            ),
          ]), panelWidth=4, panelHeight=4
        ),
      ),
      row.new('Job List')
      + row.withCollapsed(true)
      + row.withPanels(
        g.util.grid.wrapPanels([
          panels.table.job(
            'Running Jobs',
            'List of all running jobs on selected nodes',
            [
              queries.nodeJobTableRunning,
              queries.nodeJobTableRunningStart,
            ],
            [
              {
                title: 'Job Details',
                url: '/d/pbs-private-job/pbs-job?var-jobid=${__data.fields["jobid"]}&var-username=${__data.fields.username}&from=${__data.fields["Value #start"]}&to=now',
              },
            ]
          ),
        ], panelWidth=24, panelHeight=10),
      ),
      row.new('Utilisation')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels(
          std.prune([
            panels.timeseries.base(
              'CPU Requests',
              'Total CPU cores requested by jobs running on selected nodes',
              [queries.nodeCpuCoresAvailable, queries.nodeCpuCoresRequested]
            ),
            panels.timeseries.percentunit(
              'CPU Usage',
              'Total utilisation of requested cores for jobs running on selected nodes',
              [queries.nodeCpuUsageTotal, queries.nodeCpuUsageUser, queries.nodeCpuUsageSystem]
            ),
            panels.timeseries.bytes(
              'Memory Requests',
              'Total memory requested by jobs running on selected nodes',
              [queries.nodeMemoryAvailable, queries.nodeMemoryRequested]
            ),
            panels.timeseries.percentunit(
              'Memory Usage',
              'Total utilisation of requested memory for jobs running on selected nodes',
              [queries.nodeMemoryUsageUsed, queries.nodeMemoryWss, queries.nodeMemoryRss, queries.nodeMemoryCache]
            ),
            if config.pbs.gpus then
              panels.timeseries.base(
                'GPU Requests',
                'Total number of GPUs requested by jobs running on selected nodes',
                [queries.nodeGpusAvailable, queries.nodeGpusRequested]
              ),
            if config.pbs.gpus then
              panels.timeseries.percentunit(
                'GPU Usage',
                'Total utilisation of requested GPUs for jobs running on selected nodes',
                [queries.nodeGpuComputeUtilisation, queries.nodeGpuFramebufferUtilisation]
              ),
            panels.timeseries.base(
              'Processes',
              'Total number of processes running in PBS jobs on selected nodes',
              queries.nodeProcessCount
            ),
            panels.timeseries.base(
              'Threads',
              'Total number of threads running in PBS jobs on selected nodes',
              queries.nodeThreadCount
            ),
          ]), panelWidth=12, panelHeight=10
        ),
      ),
    ]), panelWidth=12, panelHeight=8
  ),
)
