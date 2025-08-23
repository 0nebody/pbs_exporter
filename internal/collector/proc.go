package collector

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

type ProcMetrics struct {
	logger *slog.Logger

	cgroupIoReadDesc  *prometheus.Desc
	cgroupIoWriteDesc *prometheus.Desc
}

func NewProcMetrics(config CollectorConfig) *ProcMetrics {
	return &ProcMetrics{
		logger: config.Logger,
		cgroupIoReadDesc: prometheus.NewDesc(
			"pbs_cgroup_io_read_bytes_total",
			"Total bytes read by the cgroup",
			[]string{"jobid", "runcount"},
			nil,
		),
		cgroupIoWriteDesc: prometheus.NewDesc(
			"pbs_cgroup_io_write_bytes_total",
			"Total bytes written by the cgroup",
			[]string{"jobid", "runcount"},
			nil,
		),
	}
}

func (p *ProcMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.cgroupIoReadDesc
	ch <- p.cgroupIoWriteDesc
}

func (p *ProcMetrics) CollectForCgroup(ch chan<- prometheus.Metric, cgroupPath string, procIds []uint64, jobRunCount string) {
	jobId := utils.GetCgroupJobId(cgroupPath)
	procRoot := procfs.DefaultMountPoint

	ioRead, ioWrite, err := GetCgroupIo(procRoot, procIds, p.logger)
	if err != nil {
		p.logger.Error("Unable to get IO from PIDs", "jobId", jobId, "err", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(p.cgroupIoReadDesc, prometheus.CounterValue, float64(ioRead), jobId, jobRunCount)
	ch <- prometheus.MustNewConstMetric(p.cgroupIoWriteDesc, prometheus.CounterValue, float64(ioWrite), jobId, jobRunCount)
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
