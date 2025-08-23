local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

local variables = import 'lib/variables.libsonnet';

local dashboards = [
  { dashboard: (import 'job.jsonnet'), hasPublic: true },
  { dashboard: (import 'node.jsonnet'), hasPublic: false },
  { dashboard: (import 'system.jsonnet'), hasPublic: false },
  { dashboard: (import 'underutilised.jsonnet'), hasPublic: false },
  { dashboard: (import 'user.jsonnet'), hasPublic: true },
];

local makePublic(dashboard) =
  dashboard
  + g.dashboard.withTagsMixin(['Public'])
  + g.dashboard.withUid(std.strReplace(dashboard.uid, 'pbs-private', 'pbs-public'))
  + {
    // update variables to limit user to their own data
    templating+: {
      list: std.map(
        function(variable)
          local variableTransforms = {
            username: variables.loginUsername,
            jobid: variables.loginJob,
          };
          if std.objectHas(variableTransforms, variable.name)
          then variable + variableTransforms[variable.name]
          else variable,
        dashboard.templating.list
      ),
    },
  }
  + {
    // update links in panels between public dashboards
    panels: std.map(
      function(panel)
        if std.objectHas(panel, 'fieldConfig')
           && std.objectHas(panel.fieldConfig, 'defaults')
           && std.objectHas(panel.fieldConfig.defaults, 'links')
        then
          panel {
            fieldConfig+: {
              defaults+: {
                links: std.map(
                  function(link) {
                    title: link.title,
                    url: std.strReplace(link.url, 'pbs-private', 'pbs-public'),
                  },
                  panel.fieldConfig.defaults.links
                ),
              },
            },
          }
        else
          panel,
      dashboard.panels
    ),
  };

local makePrivate(dashboard) =
  dashboard + g.dashboard.withTagsMixin(['Private']);

local allDashboards = std.flattenArrays(
  std.map(
    function(d)
      local baseDashboard = d.dashboard;
      local private = makePrivate(baseDashboard);

      if d.hasPublic then
        [private, makePublic(baseDashboard)]
      else
        [private],
    dashboards
  )
);

local fileName(dashboard) = '%s.json' % dashboard.uid;

{ [fileName(dashboard)]: dashboard for dashboard in allDashboards }
