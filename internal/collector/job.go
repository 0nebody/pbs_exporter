package collector

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"time"

	"github.com/0nebody/pbs_exporter/internal/pbsjobs"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	jobCache   *pbsjobs.JobCache
	pbsJobPath = "mom_priv/jobs"
)

type JobMetrics struct {
	logger  *slog.Logger
	pbsHome string

	infoDesc              *prometheus.Desc
	interactiveDesc       *prometheus.Desc
	requestedMemoryDesc   *prometheus.Desc
	requestedNcpusDesc    *prometheus.Desc
	requestedNfpgasDesc   *prometheus.Desc
	requestedNgpusDesc    *prometheus.Desc
	requestedNodesDesc    *prometheus.Desc
	requestedWalltimeDesc *prometheus.Desc
	requestsDesc          *prometheus.Desc
	runCountDesc          *prometheus.Desc
	startTimeDesc         *prometheus.Desc
	endTimeDesc           *prometheus.Desc
}

func InitialiseJobCache(pbsHome string, logger *slog.Logger) error {
	jobCache = pbsjobs.NewJobCache(logger, 60, 15*time.Second)

	jobPath := filepath.Join(pbsHome, pbsJobPath)
	if !utils.DirectoryExists(jobPath) {
		return fmt.Errorf("job directory does not exist: %s", jobPath)
	}

	// parse all existing job files
	jobFiles, err := pbsjobs.ParseJobFiles(jobPath, logger)
	if err != nil {
		return fmt.Errorf("failed to parse job files: %w", err)
	}

	// populate job cache
	for _, job := range jobFiles {
		jobId := job.JobId()
		jobCache.Set(jobId, job)
	}

	return nil
}

func WatchPbsJobs(pbsHome string, logger *slog.Logger) error {
	watchPath := filepath.Join(pbsHome, pbsJobPath)
	if !utils.DirectoryExists(watchPath) {
		return fmt.Errorf("PBS job directory does not exist: %s", watchPath)
	}

	watcher, err := pbsjobs.NewJobWatcher(watchPath)
	if err != nil {
		return fmt.Errorf("failed creating job watcher: %w", err)
	}
	defer watcher.Close()

	if err := pbsjobs.PbsJobEvent(watcher, logger, jobCache); err != nil {
		return fmt.Errorf("failed to watch PBS jobs: %w", err)
	}

	return nil
}

func NewJobMetrics(config CollectorConfig) *JobMetrics {
	return &JobMetrics{
		logger:  config.Logger,
		pbsHome: config.PbsHome,
		infoDesc: prometheus.NewDesc(
			"pbs_job_info",
			"Job information.",
			[]string{"jobid", "runcount", "interactive", "name", "node", "project", "queue", "state", "uid", "username", "vnode"},
			nil,
		),
		interactiveDesc: prometheus.NewDesc(
			"pbs_job_interactive",
			"Job interactive flag.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedMemoryDesc: prometheus.NewDesc(
			"pbs_job_requested_memory",
			"Requested memory for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedNcpusDesc: prometheus.NewDesc(
			"pbs_job_requested_ncpus",
			"Requested ncpus for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedNfpgasDesc: prometheus.NewDesc(
			"pbs_job_requested_nfpgas",
			"Requested nfpgas for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedNgpusDesc: prometheus.NewDesc(
			"pbs_job_requested_ngpus",
			"Requested ngpus for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedNodesDesc: prometheus.NewDesc(
			"pbs_job_requested_nodes",
			"Requested nodes for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestedWalltimeDesc: prometheus.NewDesc(
			"pbs_job_requested_walltime",
			"Requested walltime for the job.",
			[]string{"jobid", "runcount"},
			nil,
		),
		requestsDesc: prometheus.NewDesc(
			"pbs_job_requests_info",
			"Job requests information.",
			[]string{"jobid", "runcount", "mem", "ncpus", "nfpgas", "ngpus", "place", "walltime"},
			nil,
		),
		runCountDesc: prometheus.NewDesc(
			"pbs_job_run_count_total",
			"Number of times the job has been executed.",
			[]string{"jobid", "runcount"},
			nil,
		),
		startTimeDesc: prometheus.NewDesc(
			"pbs_job_start_time",
			"Start time of job as Unix timestamp (seconds since epoch).",
			[]string{"jobid", "runcount"},
			nil,
		),
		endTimeDesc: prometheus.NewDesc(
			"pbs_job_end_time",
			"End time of job as Unix timestamp (seconds since epoch).",
			[]string{"jobid", "runcount"},
			nil,
		),
	}
}

func (j *JobMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- j.infoDesc
	ch <- j.interactiveDesc
	ch <- j.requestedMemoryDesc
	ch <- j.requestedNcpusDesc
	ch <- j.requestedNfpgasDesc
	ch <- j.requestedNgpusDesc
	ch <- j.requestedNodesDesc
	ch <- j.requestedWalltimeDesc
	ch <- j.requestsDesc
	ch <- j.runCountDesc
	ch <- j.startTimeDesc
	ch <- j.endTimeDesc
}

func (j *JobMetrics) Collect(ch chan<- prometheus.Metric) {
	if jobCache == nil {
		j.logger.Error("Job cache is uninitialised")
		return
	}

	for _, job := range jobCache.List() {
		// export from primary node only.
		if !job.IsPrimaryNode(hostname) {
			continue
		}

		jobId := job.JobId()
		runCount := strconv.Itoa(job.RunCount)

		// export metrics regardless of user ID, ngpus, and node select
		jobUserId, err := job.JobUid()
		if err != nil {
			j.logger.Warn("Error getting job user ID", "jobid", jobId, "error", err)
		}
		nGpus, err := job.Ngpus()
		if err != nil {
			j.logger.Warn("Error getting job ngpus", "jobid", jobId, "error", err)
		}
		nodeSelect, err := job.NodeSelect()
		if err != nil {
			j.logger.Warn("Error getting job node select", "jobid", jobId, "error", err)
		}

		ch <- prometheus.MustNewConstMetric(j.infoDesc, prometheus.GaugeValue, 1, jobId, runCount, strconv.FormatBool(job.IsInteractive()), job.JobName, hostname, job.Project, job.Queue, job.JobState, jobUserId, job.JobUsername(), job.Vnode())
		ch <- prometheus.MustNewConstMetric(j.interactiveDesc, prometheus.GaugeValue, float64(utils.BooleanToInt(job.IsInteractive())), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedMemoryDesc, prometheus.GaugeValue, float64(job.ResourceList.Mem), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedNcpusDesc, prometheus.GaugeValue, float64(job.ResourceList.Ncpus), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedNfpgasDesc, prometheus.GaugeValue, float64(job.ResourceList.Nfpgas), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedNgpusDesc, prometheus.GaugeValue, float64(nGpus), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedNodesDesc, prometheus.GaugeValue, float64(nodeSelect), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestedWalltimeDesc, prometheus.GaugeValue, float64(job.RequestedWalltime()), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.requestsDesc, prometheus.GaugeValue, 1, jobId, runCount, strconv.FormatInt(job.ResourceList.Mem, 10), strconv.Itoa(job.ResourceList.Ncpus), strconv.Itoa(job.ResourceList.Nfpgas), strconv.Itoa(nGpus), job.ResourceList.Place, job.ResourceList.Walltime)
		ch <- prometheus.MustNewConstMetric(j.runCountDesc, prometheus.CounterValue, float64(job.RunCount), jobId, runCount)
		ch <- prometheus.MustNewConstMetric(j.startTimeDesc, prometheus.GaugeValue, float64(job.Stime), jobId, runCount)
		if !job.IsRunning() {
			ch <- prometheus.MustNewConstMetric(j.endTimeDesc, prometheus.GaugeValue, float64(job.Mtime), jobId, runCount)
		}
	}
}
