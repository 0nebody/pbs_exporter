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
	cgroupMetrics := &CgroupMetrics{
		cpuCountDesc: prometheus.NewDesc(
			"pbs_cgroup_cpus",
			"Number of CPUs allocated to the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuSystemDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_system_seconds_total",
			"Total system CPU time in seconds consumed by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_usage_seconds_total",
			"Total CPU time in seconds consumed by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuUserDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_user_seconds_total",
			"Total user CPU time in seconds consumed by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		hugetlbMaxDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_max_bytes",
			"Maximum huge page memory usage of tasks in the cgroup.",
			[]string{"jobid", "runcount", "hugetlb_pagesize"},
			nil,
		),
		hugetlbUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_usage_bytes",
			"Current huge page memory usage of tasks in the cgroup.",
			[]string{"jobid", "runcount", "hugetlb_pagesize"},
			nil,
		),
		ioRbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rbytes_bytes",
			"Total bytes read by tasks in the cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioRiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rios_total",
			"Total read IO operations performed by tasks in the cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioWbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wbytes_bytes",
			"Total bytes written by tasks in the cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioWiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wios_total",
			"Total write IO operations performed by tasks in the cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		memActiveAnonDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_active_anon_bytes",
			"Amount of anonymouns and swap cache memory on active LRU list.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memActiveFileDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_active_file_bytes",
			"Amount of file-backed memory on active LRU list.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memFileMappedDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_file_mapped_bytes",
			"Amount of mapped file memory.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memInactiveAnonDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_inactive_anon_bytes",
			"Amount of anonymouns and swap cache memory on inactive LRU list.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memInactiveFileDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_inactive_file_bytes",
			"Amount of file-backed memory on inactive LRU list.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_limit_bytes",
			"Memory usage limit for the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memPgfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_pgfault_total",
			"Total number of page faults incurred (major and minor).",
			[]string{"jobid", "runcount"},
			nil,
		),
		memPgmajfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_pgmajfault_total",
			"Total number of major page faults incurred.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memRssDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_rss_bytes",
			"Resident Set Size (RSS): memory required to run tasks in the cgroup",
			[]string{"jobid", "runcount"},
			nil,
		),
		memShmemDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_shmem_bytes",
			"Amount of cached filstem data that is swap-backed used by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memSwapLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_swap_limit_bytes",
			"Swap memory usage limit for the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memSwapUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_swap_usage_bytes",
			"Total swap used by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_usage_bytes",
			"Total memory used by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memWssDesc: prometheus.NewDesc(
			"pbs_cgroup_mem_wss_bytes",
			"Working Set Size (WSS): active memory used by tasks in the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		pidLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_pid_limit",
			"PID limit of cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		pidUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_pid_usage",
			"Number of PIDs used by the cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		threadUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_thread_usage",
			"Number of threads used by the cgroup.",
			[]string{"jobid", "runcount"},
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

		ch <- prometheus.MustNewConstMetric(c.metrics.cpuCountDesc, prometheus.GaugeValue, float64(metric.Cpu.Count), jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.cpuSystemDesc, prometheus.CounterValue, metric.Cpu.System, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.cpuUsageDesc, prometheus.CounterValue, metric.Cpu.Usage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.cpuUserDesc, prometheus.CounterValue, metric.Cpu.User, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memActiveAnonDesc, prometheus.GaugeValue, metric.Memory.ActiveAnon, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memActiveFileDesc, prometheus.GaugeValue, metric.Memory.ActiveFile, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memFileMappedDesc, prometheus.GaugeValue, metric.Memory.FileMapped, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memInactiveAnonDesc, prometheus.GaugeValue, metric.Memory.InactiveAnon, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memInactiveFileDesc, prometheus.GaugeValue, metric.Memory.InactiveFile, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memLimitDesc, prometheus.GaugeValue, metric.Memory.Limit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memPgfaultDesc, prometheus.CounterValue, metric.Memory.Pgfault, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memPgmajfaultDesc, prometheus.CounterValue, metric.Memory.Pgmajfault, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memRssDesc, prometheus.GaugeValue, metric.Memory.Rss, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memShmemDesc, prometheus.GaugeValue, metric.Memory.Shmem, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memSwapLimitDesc, prometheus.GaugeValue, metric.Memory.SwapLimit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memSwapUsageDesc, prometheus.GaugeValue, metric.Memory.SwapUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memUsageDesc, prometheus.GaugeValue, metric.Memory.Usage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.memWssDesc, prometheus.GaugeValue, metric.Memory.Wss, jobId, jobRunCount)
		for _, ioUsage := range metric.Io {
			major := strconv.FormatUint(ioUsage.Major, 10)
			ch <- prometheus.MustNewConstMetric(c.metrics.ioRbytesDesc, prometheus.GaugeValue, ioUsage.Rbytes, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.metrics.ioRiosDesc, prometheus.GaugeValue, ioUsage.Rios, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.metrics.ioWbytesDesc, prometheus.GaugeValue, ioUsage.Wbytes, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.metrics.ioWiosDesc, prometheus.GaugeValue, ioUsage.Wios, jobId, jobRunCount, major)
		}
		ch <- prometheus.MustNewConstMetric(c.metrics.pidLimitDesc, prometheus.GaugeValue, metric.Tasks.PidLimit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.pidUsageDesc, prometheus.GaugeValue, metric.Tasks.PidUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.metrics.threadUsageDesc, prometheus.GaugeValue, metric.Tasks.ThreadUsage, jobId, jobRunCount)
		for _, hugetlb := range metric.Hugetlb {
			ch <- prometheus.MustNewConstMetric(c.metrics.hugetlbMaxDesc, prometheus.GaugeValue, hugetlb.Max, jobId, jobRunCount, hugetlb.Pagesize)
			ch <- prometheus.MustNewConstMetric(c.metrics.hugetlbUsageDesc, prometheus.GaugeValue, hugetlb.Usage, jobId, jobRunCount, hugetlb.Pagesize)
		}

		if c.procCollector != nil {
			c.procCollector.CollectForCgroup(ch, metric.Path, metric.Tasks.Pids, jobRunCount)
		}
	}
}
