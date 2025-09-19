package collector

import (
	"log/slog"
	"strconv"
	"sync"

	"github.com/0nebody/pbs_exporter/internal/cgroups"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type CgroupCollector struct {
	cgroupPath          string
	cgroupRoot          string
	jobCollectorEnabled bool
	logger              *slog.Logger
	metrics             *CgroupMetrics
	procCollector       *ProcCollector
}

type CgroupMetrics struct {
	cpuCountDesc        *prometheus.Desc
	cpuSystemDesc       *prometheus.Desc
	cpuUsageDesc        *prometheus.Desc
	cpuUserDesc         *prometheus.Desc
	hugetlbMaxDesc      *prometheus.Desc
	hugetlbUsageDesc    *prometheus.Desc
	ioRbytesDesc        *prometheus.Desc
	ioRiosDesc          *prometheus.Desc
	ioWbytesDesc        *prometheus.Desc
	ioWiosDesc          *prometheus.Desc
	memActiveAnonDesc   *prometheus.Desc
	memActiveFileDesc   *prometheus.Desc
	memFileMappedDesc   *prometheus.Desc
	memInactiveAnonDesc *prometheus.Desc
	memInactiveFileDesc *prometheus.Desc
	memLimitDesc        *prometheus.Desc
	memPgfaultDesc      *prometheus.Desc
	memPgmajfaultDesc   *prometheus.Desc
	memRssDesc          *prometheus.Desc
	memShmemDesc        *prometheus.Desc
	memSwapLimitDesc    *prometheus.Desc
	memSwapUsageDesc    *prometheus.Desc
	memUsageDesc        *prometheus.Desc
	memWssDesc          *prometheus.Desc
	pidLimitDesc        *prometheus.Desc
	pidUsageDesc        *prometheus.Desc
	threadUsageDesc     *prometheus.Desc
}

func NewCgroupCollector(config CollectorConfig) *CgroupCollector {
	hugetlbJobLabels := append(defaultJobLabels, "hugetlb_pagesize")
	ioJobLabels := append(defaultJobLabels, "major")
	cgroupMetrics := &CgroupMetrics{
		cpuCountDesc: prometheus.NewDesc(
			"pbs_cgroup_cpus",
			"Number of CPUs allocated to the cgroup.",
			defaultJobLabels,
			nil,
		),
		cpuSystemDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_system_seconds_total",
			"Total system CPU time in seconds consumed by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		cpuUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_usage_seconds_total",
			"Total CPU time in seconds consumed by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		cpuUserDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_user_seconds_total",
			"Total user CPU time in seconds consumed by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		hugetlbMaxDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_max_bytes",
			"Maximum huge page memory usage of tasks in the cgroup.",
			hugetlbJobLabels,
			nil,
		),
		hugetlbUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_usage_bytes",
			"Current huge page memory usage of tasks in the cgroup.",
			hugetlbJobLabels,
			nil,
		),
		ioRbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rbytes_bytes",
			"Total bytes read by tasks in the cgroup.",
			ioJobLabels,
			nil,
		),
		ioRiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rios_total",
			"Total read IO operations performed by tasks in the cgroup.",
			ioJobLabels,
			nil,
		),
		ioWbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wbytes_bytes",
			"Total bytes written by tasks in the cgroup.",
			ioJobLabels,
			nil,
		),
		ioWiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wios_total",
			"Total write IO operations performed by tasks in the cgroup.",
			ioJobLabels,
			nil,
		),
		memActiveAnonDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_active_anon_bytes",
			"Amount of anonymouns and swap cache memory on active LRU list.",
			defaultJobLabels,
			nil,
		),
		memActiveFileDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_active_file_bytes",
			"Amount of file-backed memory on active LRU list.",
			defaultJobLabels,
			nil,
		),
		memFileMappedDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_file_mapped_bytes",
			"Amount of mapped file memory.",
			defaultJobLabels,
			nil,
		),
		memInactiveAnonDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_inactive_anon_bytes",
			"Amount of anonymouns and swap cache memory on inactive LRU list.",
			defaultJobLabels,
			nil,
		),
		memInactiveFileDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_inactive_file_bytes",
			"Amount of file-backed memory on inactive LRU list.",
			defaultJobLabels,
			nil,
		),
		memLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_limit_bytes",
			"Memory usage limit for the cgroup.",
			defaultJobLabels,
			nil,
		),
		memPgfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_pgfault_total",
			"Total number of page faults incurred (major and minor).",
			defaultJobLabels,
			nil,
		),
		memPgmajfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_pgmajfault_total",
			"Total number of major page faults incurred.",
			defaultJobLabels,
			nil,
		),
		memRssDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_rss_bytes",
			"Resident Set Size (RSS): memory required to run tasks in the cgroup",
			defaultJobLabels,
			nil,
		),
		memShmemDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_shmem_bytes",
			"Amount of cached filstem data that is swap-backed used by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		memSwapLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_swap_limit_bytes",
			"Swap memory usage limit for the cgroup.",
			defaultJobLabels,
			nil,
		),
		memSwapUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_swap_usage_bytes",
			"Total swap used by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		memUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_usage_bytes",
			"Total memory used by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		memWssDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_wss_bytes",
			"Working Set Size (WSS): active memory used by tasks in the cgroup.",
			defaultJobLabels,
			nil,
		),
		pidLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_pid_limit",
			"PID limit of cgroup.",
			defaultJobLabels,
			nil,
		),
		pidUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_pid_usage",
			"Number of PIDs used by the cgroup.",
			defaultJobLabels,
			nil,
		),
		threadUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_thread_usage",
			"Number of threads used by the cgroup.",
			defaultJobLabels,
			nil,
		),
	}

	return &CgroupCollector{
		cgroupPath:          config.CgroupPath,
		cgroupRoot:          config.CgroupRoot,
		jobCollectorEnabled: config.EnableJobCollector,
		logger:              config.Logger,
		metrics:             cgroupMetrics,
	}
}

func (c *CgroupCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.cpuCountDesc
	ch <- c.metrics.cpuSystemDesc
	ch <- c.metrics.cpuUsageDesc
	ch <- c.metrics.cpuUserDesc
	ch <- c.metrics.hugetlbMaxDesc
	ch <- c.metrics.hugetlbUsageDesc
	ch <- c.metrics.ioRbytesDesc
	ch <- c.metrics.ioRiosDesc
	ch <- c.metrics.ioWbytesDesc
	ch <- c.metrics.ioWiosDesc
	ch <- c.metrics.memActiveAnonDesc
	ch <- c.metrics.memActiveFileDesc
	ch <- c.metrics.memFileMappedDesc
	ch <- c.metrics.memInactiveAnonDesc
	ch <- c.metrics.memInactiveFileDesc
	ch <- c.metrics.memLimitDesc
	ch <- c.metrics.memPgfaultDesc
	ch <- c.metrics.memPgmajfaultDesc
	ch <- c.metrics.memRssDesc
	ch <- c.metrics.memShmemDesc
	ch <- c.metrics.memSwapLimitDesc
	ch <- c.metrics.memSwapUsageDesc
	ch <- c.metrics.memUsageDesc
	ch <- c.metrics.memWssDesc
	ch <- c.metrics.pidLimitDesc
	ch <- c.metrics.pidUsageDesc
	ch <- c.metrics.threadUsageDesc
}

func getCgroupStats(root string, path string, logger *slog.Logger) ([]*cgroups.Metrics, error) {
	var cgroupMetrics []*cgroups.Metrics

	manager := cgroups.NewCgroupManager(root)
	cgroupPaths, err := manager.List(path)
	if err != nil {
		logger.Error("Error listing cgroups", "err", err)
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, cgroupPath := range cgroupPaths {
		wg.Add(1)
		go func(cgroupPath string) {
			defer wg.Done()

			cgroup, err := manager.Load(cgroupPath)
			if err != nil {
				logger.Error("Error loading cgroup", "err", err, "cgroupPath", cgroupPath)
				return
			}

			metrics, err := cgroup.Stat()
			if err != nil {
				logger.Error("Error getting cgroup stats", "err", err, "cgroupPath", cgroupPath)
				return
			}

			mu.Lock()
			cgroupMetrics = append(cgroupMetrics, metrics)
			mu.Unlock()
		}(cgroupPath)
	}
	wg.Wait()

	return cgroupMetrics, nil
}

func (c *CgroupCollector) Collect(ch chan<- prometheus.Metric) {
	metrics, err := getCgroupStats(c.cgroupRoot, c.cgroupPath, c.logger)
	if err != nil {
		c.logger.Error("Failed to get cgroup metrics", "err", err)
		return
	}

	for _, metric := range metrics {
		// skip jobs with no id
		jobId := utils.GetCgroupJobId(metric.Path)
		if jobId == "" {
			c.logger.Error("Job ID empty", "cgroupPath", metric.Path)
			continue
		}

		// skip when job collector enabled but no job file for cgroup; cgroup is orphaned or being deleted.
		jobRunCount := ""
		if c.jobCollectorEnabled {
			if jobCache == nil {
				c.logger.Error("Job cache is uninitialised")
				return
			}
			if job, exists := jobCache.Get(jobId); exists {
				jobRunCount = strconv.Itoa(job.RunCount)
			} else {
				c.logger.Error("Job file not found", "jobId", jobId)
				continue
			}
		}

		jobLabels := []string{jobId, jobRunCount}

		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuCountDesc,
			prometheus.GaugeValue,
			float64(metric.Cpu.Count),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuSystemDesc,
			prometheus.CounterValue,
			metric.Cpu.System,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuUsageDesc,
			prometheus.CounterValue,
			metric.Cpu.Usage,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuUserDesc,
			prometheus.CounterValue,
			metric.Cpu.User,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memActiveAnonDesc,
			prometheus.GaugeValue,
			metric.Memory.ActiveAnon,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memActiveFileDesc,
			prometheus.GaugeValue,
			metric.Memory.ActiveFile,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memFileMappedDesc,
			prometheus.GaugeValue,
			metric.Memory.FileMapped,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memInactiveAnonDesc,
			prometheus.GaugeValue,
			metric.Memory.InactiveAnon,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memInactiveFileDesc,
			prometheus.GaugeValue,
			metric.Memory.InactiveFile,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memLimitDesc,
			prometheus.GaugeValue,
			metric.Memory.Limit,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memPgfaultDesc,
			prometheus.CounterValue,
			metric.Memory.Pgfault,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memPgmajfaultDesc,
			prometheus.CounterValue,
			metric.Memory.Pgmajfault,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memRssDesc,
			prometheus.GaugeValue,
			metric.Memory.Rss,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memShmemDesc,
			prometheus.GaugeValue,
			metric.Memory.Shmem,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memSwapLimitDesc,
			prometheus.GaugeValue,
			metric.Memory.SwapLimit,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memSwapUsageDesc,
			prometheus.GaugeValue,
			metric.Memory.SwapUsage,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memUsageDesc,
			prometheus.GaugeValue,
			metric.Memory.Usage,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memWssDesc,
			prometheus.GaugeValue,
			metric.Memory.Wss,
			jobLabels...,
		)
		for _, ioUsage := range metric.Io {
			major := strconv.FormatUint(ioUsage.Major, 10)
			ioLabels := append(jobLabels, major)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.ioRbytesDesc,
				prometheus.GaugeValue,
				ioUsage.Rbytes,
				ioLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.ioRiosDesc,
				prometheus.GaugeValue,
				ioUsage.Rios,
				ioLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.ioWbytesDesc,
				prometheus.GaugeValue,
				ioUsage.Wbytes,
				ioLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.ioWiosDesc,
				prometheus.GaugeValue,
				ioUsage.Wios,
				ioLabels...,
			)
		}
		ch <- prometheus.MustNewConstMetric(
			c.metrics.pidLimitDesc,
			prometheus.GaugeValue,
			metric.Tasks.PidLimit,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.pidUsageDesc,
			prometheus.GaugeValue,
			metric.Tasks.PidUsage,
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.metrics.threadUsageDesc,
			prometheus.GaugeValue,
			metric.Tasks.ThreadUsage,
			jobLabels...,
		)
		for _, hugetlb := range metric.Hugetlb {
			hugetlbLabels := append(jobLabels, hugetlb.Pagesize)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.hugetlbMaxDesc,
				prometheus.GaugeValue,
				hugetlb.Max,
				hugetlbLabels...,
			)
			ch <- prometheus.MustNewConstMetric(
				c.metrics.hugetlbUsageDesc,
				prometheus.GaugeValue,
				hugetlb.Usage,
				hugetlbLabels...,
			)
		}

		if c.procCollector != nil {
			c.procCollector.CollectForCgroup(ch, metric.Path, metric.Tasks.Pids, jobRunCount)
		}
	}
}
