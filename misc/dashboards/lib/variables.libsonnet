local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local config = import 'config.libsonnet';

local var = g.dashboard.variable;

{
  datasource:
    var.datasource.new('datasource', 'prometheus')
    + var.datasource.withRegex(config.datasourceFilterRegex)
    + var.datasource.generalOptions.showOnDashboard.withLabelAndValue()
    + var.datasource.generalOptions.withLabel('Data source')
    + {
      allowCustomValue: false,
      current: {
        selected: true,
        text: config.datasourceName,
        value: config.datasourceName,
      },
    },

  loginUsername:
    var.custom.new('username', [{ key: 'username', value: '${__user.login}' }])
    + var.custom.generalOptions.withLabel('Username')
    + var.custom.generalOptions.showOnDashboard.withNothing()
    + {
      allowCustomValue: false,
    },

  loginJob:
    g.dashboard.variable.query.new('jobid', 'query_result(present_over_time(pbs_job_info{username="${__user.login}"}[$__range]))')
    + var.query.generalOptions.withLabel('Job ID')
    + var.query.refresh.onTime()
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.withRegex('.*jobid="((\\d+)(\\[\\d+\\])?)".*')
    + var.query.withSort(i=0)
    + { allowCustomValue: false },

  username:
    var.query.new('username', 'label_values(pbs_job_info{}, username)')
    + var.query.generalOptions.withLabel('Username')
    + var.query.refresh.onTime()
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.withSort(i=0),

  job:
    g.dashboard.variable.query.new('jobid', 'query_result(present_over_time(pbs_job_info{username="$username"}[$__range]))')
    + var.query.generalOptions.withLabel('Job ID')
    + var.query.refresh.onTime()
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.withRegex('.*jobid="((\\d+)(\\[\\d+\\])?)".*')
    + var.query.withSort(i=0),

  jobNode:
    var.query.new('node', 'label_values(pbs_cgroup_cpus{jobid="$jobid"}, instance)')
    + var.query.generalOptions.withLabel('Node')
    + var.query.refresh.onTime()
    + var.query.selectionOptions.withIncludeAll()
    + var.query.withDatasourceFromVariable(self.datasource),

  node:
    var.query.new('node', 'label_values(pbs_node_state_available{}, node)')
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.generalOptions.withLabel('Node')
    + var.query.refresh.onTime()
    + var.query.selectionOptions.withIncludeAll(),

  partition:
    var.query.new('partition', 'label_values(pbs_node_info{}, partition)')
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.generalOptions.withLabel('Partition')
    + var.query.refresh.onTime()
    + var.query.selectionOptions.withIncludeAll(),

  queue:
    var.query.new('queue', 'label_values(pbs_node_info{}, qlist)')
    + var.query.withDatasourceFromVariable(self.datasource)
    + var.query.withRegex('/([^,]+)/g')
    + var.query.generalOptions.withLabel('Queue')
    + var.query.refresh.onTime()
    + var.query.selectionOptions.withIncludeAll(),
}
