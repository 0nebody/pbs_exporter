package collector

import (
	"context"
	"log/slog"
	"time"

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
	timeout         time.Duration
}

type CollectorConfig struct {
	CgroupPath    string
	CgroupRoot    string
	CgroupVersion string
	Logger        *slog.Logger
	PbsHome       string
	ScrapeTimeout int

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
	collectors := &Collectors{
		timeout: time.Duration(config.ScrapeTimeout) * time.Second,
	}

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

func (c *Collectors) Describe(ch chan<- *prometheus.Desc) {
	if c.nodeCollector != nil {
		c.nodeCollector.Describe(ch)
	}
	if c.jobCollector != nil {
		c.jobCollector.Describe(ch)
	}
	if c.cgroupCollector != nil {
		c.cgroupCollector.Describe(ch)
	}
}

func (c *Collectors) Collect(ch chan<- prometheus.Metric) {
	// https://github.com/prometheus/client_golang/issues/1538
	ctx, cancel := context.WithTimeout(context.TODO(), c.timeout)
	defer cancel()
	if c.nodeCollector != nil {
		c.nodeCollector.Collect(ctx, ch)
	}
	if c.jobCollector != nil {
		c.jobCollector.Collect(ctx, ch)
	}
	if c.cgroupCollector != nil {
		c.cgroupCollector.Collect(ctx, ch)
	}
}
