local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import './lib/config.libsonnet';
local panels = import './lib/panels.libsonnet';
local queries = import './lib/queries.libsonnet';
local variables = import './lib/variables.libsonnet';

local row = g.panel.row;

g.dashboard.new('PBS Underutilised Jobs')
+ g.dashboard.withUid('pbs-private-util-jobs')
+ g.dashboard.withDescription('Jobs with low average requested resource utilisation.')
+ g.dashboard.withTimezone(config.dashboard.timezone)
+ g.dashboard.withRefresh(config.dashboard.refresh)
+ g.dashboard.withTags(config.dashboard.commonTags + ['Utilisation'])
+ g.dashboard.graphTooltip.withSharedCrosshair()
+ g.dashboard.withVariables([
  variables.datasource,
])
+ g.dashboard.withPanels(
  g.util.grid.wrapPanels(
    g.util.panel.resolveCollapsedFlagOnRows([
      row.new('User Overview')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels(
          std.prune([
            panels.pie.byUser(
              'User Memory Utilisation',
              'Number of jobs by user with low memory utilisation',
              [queries.jobLowMemoryByUser]
            ),
            panels.pie.byUser(
              'User CPU Utilisation',
              'Number of jobs by user with low CPU utilisation',
              [queries.jobLowCpuByUser]
            ),
            if config.pbs.gpus then
              panels.pie.byUser(
                'User GPU Utilisation',
                'Number of jobs by user with low GPU utilisation',
                [queries.jobLowGpuByUser]
              ),
            panels.pie.byUser(
              'User Single Core Jobs',
              'Number of jobs by user requesting more than one core with usage patterns of a single core job',
              [queries.jobLowCoreUtilByUser]
            ),
          ]), panelWidth=6, panelHeight=8
        ),
      ),
      row.new('Job List')
      + row.withCollapsed(false)
      + row.withPanels(
        g.util.grid.wrapPanels(
          std.prune([
            panels.table.badjob(
              'Memory Utilisation',
              'Jobs with low memory utilisation below the threshold of %d%%' % config.thresholds.mem_low,
              [queries.jobLowMemoryUtil, queries.jobLowMemoryRequested, queries.jobTableRunningStart]
            ),
            panels.table.badjob(
              'CPU Utilisation',
              'Jobs with low CPU utilisation below the threshold of %d%%' % config.thresholds.cpu_low,
              [queries.jobLowCpuUtil, queries.jobLowCpuRequested, queries.jobTableRunningStart]
            ),
            if config.pbs.gpus then
              panels.table.badjob(
                'GPU Utilisation',
                'Jobs with low GPU utilisation below the threshold of %d%%' % config.thresholds.gpu_low,
                [queries.jobLowGpuUtil, queries.jobLowGpuRequested, queries.jobTableRunningStart]
              ),
            panels.table.badjob(
              'Single Core Jobs',
              'Jobs requesting more than one core with usage patterns of a single core job',
              [queries.jobLowCoreUtil, queries.jobLowCpuRequested, queries.jobTableRunningStart]
            ),
          ]), panelWidth=24, panelHeight=10
        ),
      ),
    ]), panelWidth=12, panelHeight=8
  ),
)
