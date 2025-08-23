local g = import 'github.com/grafana/grafonnet/gen/grafonnet-latest/main.libsonnet';

{
  gauge: {
    local gauge = g.panel.gauge,
    local options = gauge.options,
    local panelOptions = gauge.panelOptions,
    local standardOptions = gauge.standardOptions,

    base(title, description, targets):
      gauge.new(title)
      + panelOptions.withDescription(description)
      + standardOptions.withDecimals(2)
      + gauge.queryOptions.withTargets(targets),

    utilisation(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('percentunit')
      + standardOptions.thresholds.withMode('percentage')
      + standardOptions.thresholds.withSteps([
        {
          color: 'red',
          value: 0,
        },
        {
          color: 'yellow',
          value: 30,
        },
        {
          color: 'orange',
          value: 60,
        },
        {
          color: 'green',
          value: 90,
        },
      ])
      + options.reduceOptions.withCalcs('mean'),
  },

  pie: {
    local pie = g.panel.pieChart,
    local options = pie.options,
    local panelOptions = pie.panelOptions,
    local standardOptions = pie.standardOptions,

    base(title, description, targets):
      pie.new(title)
      + panelOptions.withDescription(description)
      + pie.queryOptions.withTargets(targets),

    byUser(title, description, targets):
      self.base(title, description, targets)
      + options.legend.withShowLegend(false)
      + standardOptions.withLinks([{
        title: 'User Jobs',
        url: '/d/pbs-private-user/pbs-user?var-username=${__field.labels.username}',
      }]),
  },

  stat: {
    local stat = g.panel.stat,
    local options = stat.options,
    local panelOptions = stat.panelOptions,
    local standardOptions = stat.standardOptions,

    base(title, description, targets):
      stat.new(title)
      + panelOptions.withDescription(description)
      + options.withGraphMode('none')
      + standardOptions.withDecimals(0)
      + standardOptions.thresholds.withSteps([
        {
          color: 'green',
        },
      ])
      + stat.queryOptions.withTargets(targets),

    count(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('none'),

    bytes(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('bytes'),

    seconds(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withDecimals(2)
      + standardOptions.withUnit('s'),
  },

  table: {
    local table = g.panel.table,
    local fieldConfig = table.fieldConfig,
    local options = table.options,
    local panelOptions = table.panelOptions,
    local queryOptions = table.queryOptions,
    local standardOptions = table.standardOptions,

    base(title, description, targets):
      table.new(title)
      + fieldConfig.defaults.custom.withFilterable(true)
      + options.withShowHeader('auto')
      + panelOptions.withDescription(description)
      + panelOptions.withGridPos(h=12, w=24)
      + standardOptions.withFilterable(true)
      + standardOptions.withOverrides([
        // Hide all name fields
        standardOptions.override.byRegexp.new('/__name__.*/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
        // Hide all time fields
        standardOptions.override.byRegexp.new('/Time.*/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
      ] + [
        // Set the display name to the query legend when available
        standardOptions.override.byName.new('Value #' + t.refId)
        + standardOptions.override.byName.withProperty('displayName', t.legendFormat)
        for t in targets
        if std.objectHas(t, 'refId') && std.objectHas(t, 'legendFormat')
      ] + [
        // Set memory columns to bytes
        standardOptions.override.byRegexp.new('/.*[m|M]emory.*/')
        + standardOptions.override.byRegexp.withProperty('unit', 'bytes'),
      ])
      + standardOptions.withUnit('none')
      + table.queryOptions.withTargets(targets),

    node(title, description, targets):
      self.base(title, description, targets)
      + options.footer.withShow(true)
      + queryOptions.withTransformations([
        {
          id: 'joinByField',
          options: {
            byField: 'node',
            mode: 'outer',
          },
        },
      ])
      + options.withSortBy([
        {
          displayName: 'node',
          desc: false,
        },
      ])
      + standardOptions.withLinks([
        {
          title: 'Node Details',
          url: '/d/pbs-private-node/pbs-node-new?var-node=${__data.fields.node}',
        },
      ])
      + standardOptions.withUnit('sishort'),

    user(title, description, targets):
      self.base(title, description, targets)
      + options.footer.withShow(true)
      + queryOptions.withTransformations([
        {
          id: 'joinByField',
          options: {
            byField: 'username',
            mode: 'outer',
          },
        },
      ])
      + standardOptions.withLinks([
        {
          title: 'User Details',
          url: '/d/pbs-private-user/pbs-user-new?var-username=${__data.fields.username}',
        },
      ])
      + standardOptions.withUnit('sishort'),

    job(title, description, targets, links):
      self.base(title, description, targets)
      + options.footer.withShow(false)
      + standardOptions.withLinks(links)
      + queryOptions.withTransformations([
        {
          id: 'joinByField',
          options: {
            byField: 'jobid',
            mode: 'outer',
          },
        },
        {
          id: 'filterFieldsByName',
          options: {
            include: {
              names: [
                'name',
                'jobid',
                'project',
                'queue',
                'username',
                'Value #start',
                'Value #end',
              ],
            },
          },
        },
      ])
      + standardOptions.withOverridesMixin([
        standardOptions.override.byRegexp.new('/([s|S]tart).*/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
        standardOptions.override.byRegexp.new('/([e|E]nd).*/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
        standardOptions.override.byRegexp.new('/username/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
      ]),

    badjob(title, description, targets):
      self.base(title, description, targets)
      + options.footer.withShow(false)
      + standardOptions.withLinks([
        {
          title: 'Job Details',
          url: '/d/pbs-private-job/pbs-job?var-jobid=${__data.fields["jobid"]}&var-username=${__data.fields.username}&from=${__data.fields["Value #start"]}&to=now',
        },
      ])
      + queryOptions.withTransformations([
        {
          id: 'joinByField',
          options: {
            byField: 'jobid',
            mode: 'inner',
          },
        },
        {
          id: 'filterFieldsByName',
          options: {
            include: {
              names: [
                'jobid',
                'name',
                'username',
                'Value #used',
                'Value #requested',
                'Value #start',
                'Value #end',
              ],
            },
          },
        },
      ])
      + standardOptions.withOverridesMixin([
        standardOptions.override.byRegexp.new('/([s|S]tart).*/')
        + standardOptions.override.byRegexp.withProperty('custom.hidden', true),
        standardOptions.override.byRegexp.new('/.*[u|U]sed.*/')
        + standardOptions.override.byRegexp.withProperty('decimals', '2')
        + standardOptions.override.byRegexp.withProperty('unit', 'percent'),
      ]),
  },

  timeseries: {
    local timeSeries = g.panel.timeSeries,
    local options = timeSeries.options,
    local panelOptions = timeSeries.panelOptions,
    local standardOptions = timeSeries.standardOptions,

    base(title, description, targets):
      timeSeries.new(title)
      + timeSeries.queryOptions.withTargets(targets)
      + options.legend.withDisplayMode('table')
      + standardOptions.withUnit('sishort')
      + panelOptions.withDescription(description)
      + options.legend.withCalcs([
        'mean',
        'max',
        'lastNotNull',
      ]),

    bytes(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('bytes'),

    megabytes(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('decmbytes'),

    percent(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withMax(100)
      + standardOptions.withMin(0)
      + standardOptions.withUnit('percent'),

    percentunit(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withMax(1)
      + standardOptions.withMin(0)
      + standardOptions.withUnit('percentunit'),

    power(title, description, targets):
      self.base(title, description, targets)
      + standardOptions.withUnit('watt'),
  },
}
