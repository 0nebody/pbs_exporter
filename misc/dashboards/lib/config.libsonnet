{
  pbs: {
    fpgas: true,
    gpus: true,
    hugetlb: false,
    swap: false,
  },
  datasourceName: 'Prometheus',
  datasourceFilterRegex: '',
  dashboard: {
    refresh: '1m',
    timezone: 'browser',
    commonTags: ['PBS'],
  },
  thresholds: {
    cpu_low: 10,
    gpu_low: 10,
    mem_low: 10,
    runtime: 15 * 60,
  },
}
