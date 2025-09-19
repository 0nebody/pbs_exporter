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

type JobCollector struct {
	logger  *slog.Logger
	metrics *JobMetrics
	pbsHome string
}

type JobMetrics struct {
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

func NewJobCollector(config CollectorConfig) *JobCollector {
	jobMetrics := &JobMetrics{
		infoDesc: prometheus.NewDesc(
			"pbs_job_info",
			"Job information.",
			append(defaultJobLabels,
				"interactive", "name", "node", "project", "queue", "state", "uid", "username", "vnode"),
			nil,
		),
		interactiveDesc: prometheus.NewDesc(
			"pbs_job_interactive",
			"Job interactive flag.",
			defaultJobLabels,
			nil,
		),
		requestedMemoryDesc: prometheus.NewDesc(
			"pbs_job_requested_memory",
			"Requested memory for the job.",
			defaultJobLabels,
			nil,
		),
		requestedNcpusDesc: prometheus.NewDesc(
			"pbs_job_requested_ncpus",
			"Requested ncpus for the job.",
			defaultJobLabels,
			nil,
		),
		requestedNfpgasDesc: prometheus.NewDesc(
			"pbs_job_requested_nfpgas",
			"Requested nfpgas for the job.",
			defaultJobLabels,
			nil,
		),
		requestedNgpusDesc: prometheus.NewDesc(
			"pbs_job_requested_ngpus",
			"Requested ngpus for the job.",
			defaultJobLabels,
			nil,
		),
		requestedNodesDesc: prometheus.NewDesc(
			"pbs_job_requested_nodes",
			"Requested nodes for the job.",
			defaultJobLabels,
			nil,
		),
		requestedWalltimeDesc: prometheus.NewDesc(
			"pbs_job_requested_walltime",
			"Requested walltime for the job.",
			defaultJobLabels,
			nil,
		),
		requestsDesc: prometheus.NewDesc(
			"pbs_job_requests_info",
			"Job requests information.",
			append(defaultJobLabels, "mem", "ncpus", "nfpgas", "ngpus", "place", "walltime"),
			nil,
		),
		runCountDesc: prometheus.NewDesc(
			"pbs_job_run_count_total",
			"Number of times the job has been executed.",
			defaultJobLabels,
			nil,
		),
		startTimeDesc: prometheus.NewDesc(
			"pbs_job_start_time",
			"Start time of job as Unix timestamp (seconds since epoch).",
			defaultJobLabels,
			nil,
		),
		endTimeDesc: prometheus.NewDesc(
			"pbs_job_end_time",
			"End time of job as Unix timestamp (seconds since epoch).",
			defaultJobLabels,
			nil,
		),
	}

	return &JobCollector{
		logger:  config.Logger,
		pbsHome: config.PbsHome,
		metrics: jobMetrics,
	}
}

func (j *JobCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- j.metrics.infoDesc
	ch <- j.metrics.interactiveDesc
	ch <- j.metrics.requestedMemoryDesc
	ch <- j.metrics.requestedNcpusDesc
	ch <- j.metrics.requestedNfpgasDesc
	ch <- j.metrics.requestedNgpusDesc
	ch <- j.metrics.requestedNodesDesc
	ch <- j.metrics.requestedWalltimeDesc
	ch <- j.metrics.requestsDesc
	ch <- j.metrics.runCountDesc
	ch <- j.metrics.startTimeDesc
	ch <- j.metrics.endTimeDesc
}

func (j *JobCollector) Collect(ch chan<- prometheus.Metric) {
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

		jobLabels := []string{jobId, runCount}
		infoLabels := append(
			jobLabels,
			strconv.FormatBool(job.IsInteractive()),
			job.JobName,
			hostname,
			job.Project,
			job.Queue,
			job.JobState,
			jobUserId,
			job.JobUsername(),
			job.Vnode(),
		)
		reqLabels := append(
			jobLabels,
			strconv.FormatInt(job.ResourceList.Mem, 10),
			strconv.Itoa(job.ResourceList.Ncpus),
			strconv.Itoa(job.ResourceList.Nfpgas),
			strconv.Itoa(nGpus),
			job.ResourceList.Place,
			job.ResourceList.Walltime,
		)

		ch <- prometheus.MustNewConstMetric(
			j.metrics.infoDesc,
			prometheus.GaugeValue,
			1,
			infoLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.interactiveDesc,
			prometheus.GaugeValue,
			float64(utils.BooleanToInt(job.IsInteractive())),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedMemoryDesc,
			prometheus.GaugeValue,
			float64(job.ResourceList.Mem),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedNcpusDesc,
			prometheus.GaugeValue,
			float64(job.ResourceList.Ncpus),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedNfpgasDesc,
			prometheus.GaugeValue,
			float64(job.ResourceList.Nfpgas),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedNgpusDesc,
			prometheus.GaugeValue,
			float64(nGpus),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedNodesDesc,
			prometheus.GaugeValue,
			float64(nodeSelect),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestedWalltimeDesc,
			prometheus.GaugeValue,
			float64(job.RequestedWalltime()),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.requestsDesc,
			prometheus.GaugeValue,
			1,
			reqLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.runCountDesc,
			prometheus.CounterValue,
			float64(job.RunCount),
			jobLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			j.metrics.startTimeDesc,
			prometheus.GaugeValue,
			float64(job.Stime),
			jobLabels...,
		)
		if !job.IsRunning() {
			ch <- prometheus.MustNewConstMetric(
				j.metrics.endTimeDesc,
				prometheus.GaugeValue,
				float64(job.Mtime),
				jobLabels...,
			)
		}
	}
}
