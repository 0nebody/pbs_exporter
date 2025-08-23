package collector

import (
	"log/slog"
	"strconv"
	"sync"

	"github.com/0nebody/pbs_exporter/internal/cgroups"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type CgroupMetrics struct {
	cgroupPath          string
	cgroupRoot          string
	jobCollectorEnabled bool
	logger              *slog.Logger
	procCollector       *ProcMetrics

	cpuCountDesc           *prometheus.Desc
	cpuSystemDesc          *prometheus.Desc
	cpuUsageDesc           *prometheus.Desc
	cpuUserDesc            *prometheus.Desc
	hugetlbMaxDesc         *prometheus.Desc
	hugetlbUsageDesc       *prometheus.Desc
	ioRbytesDesc           *prometheus.Desc
	ioRiosDesc             *prometheus.Desc
	ioWbytesDesc           *prometheus.Desc
	ioWiosDesc             *prometheus.Desc
	memAnonUsageDesc       *prometheus.Desc
	memFileMappedUsageDesc *prometheus.Desc
	memFileUsageDesc       *prometheus.Desc
	memLimitDesc           *prometheus.Desc
	memPgfaultDesc         *prometheus.Desc
	memPgmajfaultDesc      *prometheus.Desc
	memShmemUsageDesc      *prometheus.Desc
	memSwapLimitDesc       *prometheus.Desc
	memSwapUsageDesc       *prometheus.Desc
	memUsageDesc           *prometheus.Desc
	pidLimitDesc           *prometheus.Desc
	pidUsageDesc           *prometheus.Desc
	threadUsageDesc        *prometheus.Desc
}

func NewCgroupMetrics(config CollectorConfig) *CgroupMetrics {
	return &CgroupMetrics{
		cgroupPath:          config.CgroupPath,
		cgroupRoot:          config.CgroupRoot,
		jobCollectorEnabled: config.EnableJobCollector,
		logger:              config.Logger,

		cpuCountDesc: prometheus.NewDesc(
			"pbs_cgroup_cpus",
			"Number of CPUs allocated to cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuSystemDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_system_seconds_total",
			"System CPU time of cgroup in seconds.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_usage_seconds_total",
			"Current CPU usage of cgroup in seconds.",
			[]string{"jobid", "runcount"},
			nil,
		),
		cpuUserDesc: prometheus.NewDesc(
			"pbs_cgroup_cpu_user_seconds_total",
			"User CPU time of cgroup in seconds.",
			[]string{"jobid", "runcount"},
			nil,
		),
		hugetlbMaxDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_max_bytes",
			"Maximum hugetlb usage of cgroup in bytes.",
			[]string{"jobid", "runcount", "hugetlb_pagesize"},
			nil,
		),
		hugetlbUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_hugetlb_usage_bytes",
			"Current hugetlb usage of cgroup in bytes.",
			[]string{"jobid", "runcount", "hugetlb_pagesize"},
			nil,
		),
		ioRbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rbytes_bytes",
			"Bytes read by cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioRiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_rios_total",
			"IO read operations by cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioWbytesDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wbytes_bytes",
			"Bytes written by cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		ioWiosDesc: prometheus.NewDesc(
			"pbs_cgroup_io_wios_total",
			"IO write operations by cgroup.",
			[]string{"jobid", "runcount", "major"},
			nil,
		),
		memAnonUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_anon_bytes",
			"Anonymous memory usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memFileUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_file_bytes",
			"File cache usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memShmemUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_shmem_bytes",
			"Shared memory usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memFileMappedUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_file_mapped_bytes",
			"File mapped memory usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memPgfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_pgfault_total",
			"Total number of page faults in cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memPgmajfaultDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_pgmajfault_total",
			"Total number of major page faults in cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_usage_bytes",
			"Current memory usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_limit_bytes",
			"Memory limit of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memSwapUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_swap_usage_bytes",
			"Current swap usage of cgroup in bytes.",
			[]string{"jobid", "runcount"},
			nil,
		),
		memSwapLimitDesc: prometheus.NewDesc(
			"pbs_cgroup_memory_swap_limit_bytes",
			"Swap limit of cgroup in bytes.",
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
			"PID usage of cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
		threadUsageDesc: prometheus.NewDesc(
			"pbs_cgroup_thread_usage",
			"Thread usage of cgroup.",
			[]string{"jobid", "runcount"},
			nil,
		),
	}
}

func (c *CgroupMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuCountDesc
	ch <- c.cpuSystemDesc
	ch <- c.cpuUsageDesc
	ch <- c.cpuUserDesc
	ch <- c.hugetlbMaxDesc
	ch <- c.hugetlbUsageDesc
	ch <- c.ioRbytesDesc
	ch <- c.ioRiosDesc
	ch <- c.ioWbytesDesc
	ch <- c.ioWiosDesc
	ch <- c.memAnonUsageDesc
	ch <- c.memFileMappedUsageDesc
	ch <- c.memFileUsageDesc
	ch <- c.memLimitDesc
	ch <- c.memPgfaultDesc
	ch <- c.memPgmajfaultDesc
	ch <- c.memShmemUsageDesc
	ch <- c.memSwapLimitDesc
	ch <- c.memSwapUsageDesc
	ch <- c.memUsageDesc
	ch <- c.pidLimitDesc
	ch <- c.pidUsageDesc
	ch <- c.threadUsageDesc
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

func (c *CgroupMetrics) Collect(ch chan<- prometheus.Metric) {
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

		ch <- prometheus.MustNewConstMetric(c.cpuCountDesc, prometheus.GaugeValue, float64(metric.Cpu.Count), jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.cpuSystemDesc, prometheus.CounterValue, metric.Cpu.System, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.cpuUsageDesc, prometheus.CounterValue, metric.Cpu.Usage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.cpuUserDesc, prometheus.CounterValue, metric.Cpu.User, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memAnonUsageDesc, prometheus.GaugeValue, metric.Memory.AnonUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memFileMappedUsageDesc, prometheus.GaugeValue, metric.Memory.FileMappedUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memFileUsageDesc, prometheus.GaugeValue, metric.Memory.FileUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memLimitDesc, prometheus.GaugeValue, metric.Memory.Limit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memPgfaultDesc, prometheus.CounterValue, metric.Memory.Pgfault, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memPgmajfaultDesc, prometheus.CounterValue, metric.Memory.Pgmajfault, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memShmemUsageDesc, prometheus.GaugeValue, metric.Memory.ShmemUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memSwapLimitDesc, prometheus.GaugeValue, metric.Memory.SwapLimit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memSwapUsageDesc, prometheus.GaugeValue, metric.Memory.SwapUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.memUsageDesc, prometheus.GaugeValue, metric.Memory.Usage, jobId, jobRunCount)
		for _, ioUsage := range metric.Io {
			major := strconv.FormatUint(ioUsage.Major, 10)
			ch <- prometheus.MustNewConstMetric(c.ioRbytesDesc, prometheus.GaugeValue, ioUsage.Rbytes, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.ioRiosDesc, prometheus.GaugeValue, ioUsage.Rios, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.ioWbytesDesc, prometheus.GaugeValue, ioUsage.Wbytes, jobId, jobRunCount, major)
			ch <- prometheus.MustNewConstMetric(c.ioWiosDesc, prometheus.GaugeValue, ioUsage.Wios, jobId, jobRunCount, major)
		}
		ch <- prometheus.MustNewConstMetric(c.pidLimitDesc, prometheus.GaugeValue, metric.Tasks.PidLimit, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.pidUsageDesc, prometheus.GaugeValue, metric.Tasks.PidUsage, jobId, jobRunCount)
		ch <- prometheus.MustNewConstMetric(c.threadUsageDesc, prometheus.GaugeValue, metric.Tasks.ThreadUsage, jobId, jobRunCount)
		for _, hugetlb := range metric.Hugetlb {
			ch <- prometheus.MustNewConstMetric(c.hugetlbMaxDesc, prometheus.GaugeValue, hugetlb.Max, jobId, jobRunCount, hugetlb.Pagesize)
			ch <- prometheus.MustNewConstMetric(c.hugetlbUsageDesc, prometheus.GaugeValue, hugetlb.Usage, jobId, jobRunCount, hugetlb.Pagesize)
		}

		if c.procCollector != nil {
			c.procCollector.CollectForCgroup(ch, metric.Path, metric.Tasks.Pids, jobRunCount)
		}
	}
}
