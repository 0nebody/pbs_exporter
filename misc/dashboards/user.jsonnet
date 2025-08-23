local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import './lib/config.libsonnet';
local panels = import './lib/panels.libsonnet';
local queries = import './lib/queries.libsonnet';
local variables = import './lib/variables.libsonnet';

local row = g.panel.row;

g.dashboard.new('PBS User')
+ g.dashboard.withUid('pbs-private-user')
+ g.dashboard.withDescription('Summary of requested resources and jobs for a PBS user.')
+ g.dashboard.withTimezone(config.dashboard.timezone)
+ g.dashboard.withRefresh(config.dashboard.refresh)
+ g.dashboard.withTags(config.dashboard.commonTags + ['User'])
+ g.dashboard.graphTooltip.withSharedCrosshair()
+ g.dashboard.withVariables([
  variables.datasource,
  variables.username,
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
              'Total number of nodes requested by user',
              [queries.userRequestedNodes]
            ),
            panels.stat.count(
              'CPUs',
              'Total number of CPUs requested by user',
              [queries.userRequestedCores]
            ),
            panels.stat.bytes(
              'Memory',
              'Total amount of memory requested by user',
              [queries.userRequestedMemory]
            ),
            if config.pbs.gpus then
              panels.stat.count(
                'GPUs',
                'Total number of GPUs requested by user',
                [queries.userRequestedNgpus]
              ),
            if config.pbs.fpgas then
              panels.stat.count(
                'FPGAs',
                'Total number of FPGAs requested by user',
                [queries.userRequestedNfpgas]
              ),
            panels.stat.seconds(
              'Requested Walltime',
              'Total requested walltime by user',
              [queries.userRequestedWalltime]
            ),
            panels.gauge.utilisation(
              'Node Usage',
              'Total utilised nodes by user',
              [queries.userUtilisedNodes]
            ),
            panels.gauge.utilisation(
              'CPU Usage',
              'Total average CPUs utilisation by user',
              [queries.userUtilisedCores]
            ),
            panels.gauge.utilisation(
              'Memory Usage',
              'Total average memory utilisation by user',
              [queries.userUtilisedMemory]
            ),
            if config.pbs.gpus then
              panels.gauge.utilisation(
                'GPU Usage',
                'Total average GPU utilisation by user',
                [queries.userUtilisedNgpus]
              ),
            if config.pbs.fpgas then
              panels.gauge.utilisation(
                'FPGA Usage',
                'Total average FPGA utilisation by user',
                [queries.userUtilisedNfpgas]
              ),
            panels.stat.seconds(
              'Remaining Walltime',
              'Total remaining walltime by user',
              [queries.userUtilisedWalltime]
            ),
          ]), panelWidth=4, panelHeight=4
        ),
      ),
      row.new('Job List')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels([
          panels.table.job(
            'Running Jobs',
            'List of all running jobs for user',
            [
              queries.userJobTableRunning,
              queries.userJobTableRunningStart,
            ],
            [
              {
                title: 'Job Details',
                url: '/d/pbs-private-job/pbs-job?var-jobid=${__data.fields["jobid"]}&var-username=${__data.fields.username}&from=${__data.fields["Value #start"]}&to=now',
              },
            ]
          ),
          panels.table.job(
            'Completed Jobs',
            'List of all completed jobs for user',
            [
              queries.userJobTableCompleted,
              queries.userJobTableCompletedStart,
              queries.userJobTableCompletedEnd,
            ],
            [
              {
                title: 'Job Details',
                url: '/d/pbs-private-job/pbs-job?var-jobid=${__data.fields["jobid"]}&var-username=${__data.fields.username}&from=${__data.fields["Value #start"]}&to=${__data.fields["Value #end"]}',
              },
            ]
          ),
        ], panelWidth=24, panelHeight=10),
      ),
    ]), panelWidth=12, panelHeight=8
  ),
)
