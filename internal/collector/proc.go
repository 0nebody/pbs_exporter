package collector

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

type ProcCollector struct {
	logger  *slog.Logger
	metrics *ProcMetrics
}

type ProcMetrics struct {
	cgroupIoReadDesc  *prometheus.Desc
	cgroupIoWriteDesc *prometheus.Desc
}

func NewProcCollector(config CollectorConfig) *ProcCollector {
	procMetrics := &ProcMetrics{
		cgroupIoReadDesc: prometheus.NewDesc(
			"pbs_cgroup_io_read_bytes_total",
			"Total bytes read by the cgroup",
			defaultJobLabels,
			nil,
		),
		cgroupIoWriteDesc: prometheus.NewDesc(
			"pbs_cgroup_io_write_bytes_total",
			"Total bytes written by the cgroup",
			defaultJobLabels,
			nil,
		),
	}

	return &ProcCollector{
		logger:  config.Logger,
		metrics: procMetrics,
	}
}

func (p *ProcCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.metrics.cgroupIoReadDesc
	ch <- p.metrics.cgroupIoWriteDesc
}

func (p *ProcCollector) CollectForCgroup(ch chan<- prometheus.Metric, cgroupPath string, procIds []uint64, jobRunCount string) {
	jobId := utils.GetCgroupJobId(cgroupPath)
	procRoot := procfs.DefaultMountPoint

	ioRead, ioWrite, err := GetCgroupIo(procRoot, procIds, p.logger)
	if err != nil {
		p.logger.Error("Unable to get IO from PIDs", "jobId", jobId, "err", err)
		return
	}

	jobLabels := []string{jobId, jobRunCount}

	ch <- prometheus.MustNewConstMetric(
		p.metrics.cgroupIoReadDesc,
		prometheus.CounterValue,
		float64(ioRead),
		jobLabels...,
	)
	ch <- prometheus.MustNewConstMetric(
		p.metrics.cgroupIoWriteDesc,
		prometheus.CounterValue,
		float64(ioWrite),
		jobLabels...,
	)
}

func GetCgroupIo(procRoot string, pids []uint64, logger *slog.Logger) (uint64, uint64, error) {
	ioRead := uint64(0)
	ioWrite := uint64(0)

	fs, err := procfs.NewFS(procRoot)
	if err != nil {
		return ioRead, ioWrite, err
	}

	var wg sync.WaitGroup

	for _, pid := range pids {
		wg.Add(1)
		go func(pid uint64) {
			defer wg.Done()

			proc, err := fs.Proc(int(pid))
			if err != nil {
				logger.Debug("Unable to read PID", "pid", pid, "err", err)
				return
			}

			ioStats, err := proc.IO()
			if err != nil {
				logger.Debug("Unable to get IO for PID", "pid", pid, "err", err)
			} else {
				atomic.AddUint64(&ioRead, ioStats.ReadBytes)
				atomic.AddUint64(&ioWrite, ioStats.WriteBytes)
			}
		}(pid)
	}
	wg.Wait()

	return ioRead, ioWrite, nil
}
