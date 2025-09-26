package collector

import (
	"log/slog"

	"github.com/0nebody/pbs_exporter/internal/cgroups"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	defaultJobLabels  = []string{"jobid", "runcount"}
	defaultNodeLabels = []string{"node", "vnode"}
	hostname          = utils.MustHostname()
	pbsCgroupPaths    = map[string]string{
		"v1": "pbs_jobs.service/jobid",
		"v2": "pbs_jobs.service/jobs",
	}
)

type Collectors struct {
	cgroupCollector *CgroupCollector
	jobCollector    *JobCollector
	nodeCollector   *NodeCollector
}

type CollectorConfig struct {
	CgroupPath    string
	CgroupRoot    string
	CgroupVersion string
	Logger        *slog.Logger
	PbsHome       string

	EnableCgroupCollector bool
	EnableJobCollector    bool
	EnableNodeCollector   bool
}

func NewCollectorConfig(cgroupRoot string, logger *slog.Logger) CollectorConfig {
	cgroupManager := cgroups.NewCgroupManager(cgroupRoot)
	cgroupVersion := cgroupManager.Version()

	return CollectorConfig{
		CgroupPath:    pbsCgroupPaths[cgroupVersion],
		CgroupRoot:    cgroupRoot,
		CgroupVersion: cgroupVersion,
		Logger:        logger,
	}
}

func NewCollectors(config CollectorConfig) *Collectors {
	collectors := &Collectors{}

	if config.EnableCgroupCollector {
		collectors.cgroupCollector = NewCgroupCollector(config)
	} else {
		config.Logger.Info("Cgroup collector is disabled")
	}

	if config.EnableJobCollector {
		collectors.jobCollector = NewJobCollector(config)
	} else {
		config.Logger.Info("PBS Job collector is disabled")
	}

	if config.EnableNodeCollector {
		collectors.nodeCollector = NewNodeCollector(config)
	} else {
		config.Logger.Info("PBS Node collector is disabled")
	}

	return collectors
}

func (m *Collectors) Describe(ch chan<- *prometheus.Desc) {
	if m.nodeCollector != nil {
		m.nodeCollector.Describe(ch)
	}
	if m.jobCollector != nil {
		m.jobCollector.Describe(ch)
	}
	if m.cgroupCollector != nil {
		m.cgroupCollector.Describe(ch)
	}
}

func (m *Collectors) Collect(ch chan<- prometheus.Metric) {
	if m.nodeCollector != nil {
		m.nodeCollector.Collect(ch)
	}
	if m.jobCollector != nil {
		m.jobCollector.Collect(ch)
	}
	if m.cgroupCollector != nil {
		m.cgroupCollector.Collect(ch)
	}
}
