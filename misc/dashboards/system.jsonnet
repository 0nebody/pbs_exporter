local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import './lib/config.libsonnet';
local panels = import './lib/panels.libsonnet';
local queries = import './lib/queries.libsonnet';
local variables = import './lib/variables.libsonnet';

local row = g.panel.row;

g.dashboard.new('PBS System')
+ g.dashboard.withUid('pbs-private-system')
+ g.dashboard.withDescription('PBS system summary of nodes, jobs, and users.')
+ g.dashboard.withTimezone(config.dashboard.timezone)
+ g.dashboard.withRefresh(config.dashboard.refresh)
+ g.dashboard.withTags(config.dashboard.commonTags + ['System'])
+ g.dashboard.graphTooltip.withSharedCrosshair()
+ g.dashboard.withVariables([
  variables.datasource,
  variables.partition,
  variables.queue,
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
              'Total nodes available',
              [queries.systemNodesAvailable]
            ),
            panels.stat.count(
              'CPUs',
              'Total CPU cores available for scheduling',
              [queries.systemCpuCoresAvailable]
            ),
            panels.stat.bytes(
              'Memory',
              'Total memory available for scheduling',
              [queries.systemMemoryAvailable]
            ),
            if config.pbs.gpus then
              panels.stat.count(
                'GPUs',
                'Total GPUs available for scheduling',
                [queries.systemGpusAvailable]
              ),
            if config.pbs.fpgas then
              panels.stat.count(
                'FPGAs',
                'Total FPGAs available for scheduling',
                [queries.systemFpgasAvailable]
              ),
            panels.stat.count(
              'Jobs',
              'Total jobs running',
              [queries.systemRunningJobs]
            ),
          ]), panelWidth=4, panelHeight=4
        ),
      ),
      row.new('Requests')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels(
          std.prune([
            panels.timeseries.base(
              'CPU Requests',
              'Total CPU cores requested by jobs',
              [queries.systemCpuCoresAvailable, queries.systemCpuCoresRequested]
            ),
            panels.timeseries.bytes(
              'Memory Requests',
              'Total memory requested by jobs',
              [queries.systemMemoryAvailable, queries.systemMemoryRequested]
            ),
            if config.pbs.gpus then
              panels.timeseries.base(
                'GPU Requests',
                'Total GPUs requested by jobs',
                [queries.systemGpusAvailable, queries.systemGpusRequested]
              ),
          ]), panelWidth=8, panelHeight=10
        ),
      ),
      row.new('Nodes')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels([
          panels.table.node(
            'Node List',
            'List of nodes belonging to partition and queue',
            std.prune([
              queries.systemJobsTable,
              queries.systemCpuCoresAvailableTable,
              queries.systemMemoryAvailableTable,
              if config.pbs.gpus then
                queries.systemGpusAvailableTable,
            ]),
          ),
        ], panelWidth=4, panelHeight=4),
      ),
      row.new('Users')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels([
          panels.table.user(
            'Users List',
            'List of users running jobs in partition and queue',
            std.prune([
              queries.systemUserTableJobs,
              queries.systemUserTableCpus,
              queries.systemUserTableMemory,
              if config.pbs.gpus then
                queries.systemUserTableGpus,
            ])
          ),
        ], panelWidth=12, panelHeight=10),
      ),
    ]), panelWidth=12, panelHeight=8
  ),
)
