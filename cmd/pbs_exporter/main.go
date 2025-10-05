package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/0nebody/pbs_exporter/internal/collector"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
)

var (
	cgroupCollectorEnabled = kingpin.Flag("cgroup.enabled", "Enable cgroup collector.").Default("true").Bool()
	cgroupRoot             = kingpin.Flag("cgroup.root", "Root path of cgroup filesystem hierarchy.").Default("/sys/fs/cgroup").String()
	jobCollectorEnabled    = kingpin.Flag("job.enabled", "Enable job collector.").Default("true").Bool()
	listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9307").String()
	nodeCollectorEnabled   = kingpin.Flag("node.enabled", "Enable node collector.").Default("false").Bool()
	pbsHome                = kingpin.Flag("job.pbs_home", "PBS home directory.").Default("/var/spool/pbs").String()
	scrapeTimeout          = kingpin.Flag("scrape.timeout", "Per-scrape timeout in seconds.").Default("5").Int()
)

func redirectToMetrics(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/metrics", http.StatusFound)
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/", redirectToMetrics)
	mux.HandleFunc("/healthz", healthz)

	return mux
}

func main() {
	promslogConfig := &promslog.Config{}

	kingpin.CommandLine.UsageWriter(os.Stdout)
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print("pbs_exporter"))
	kingpin.Parse()

	logger := promslog.New(promslogConfig)
	logger.Info("Starting PBS Exporter")

	// Initialize collector configuration
	collectorConfig := collector.NewCollectorConfig(*cgroupRoot, logger)
	collectorConfig.PbsHome = *pbsHome
	collectorConfig.ScrapeTimeout = *scrapeTimeout
	collectorConfig.EnableCgroupCollector = *cgroupCollectorEnabled
	collectorConfig.EnableJobCollector = *jobCollectorEnabled
	collectorConfig.EnableNodeCollector = *nodeCollectorEnabled
	logger.Info("Using cgroup", "version", collectorConfig.CgroupVersion, "path", filepath.Join(collectorConfig.CgroupRoot, collectorConfig.CgroupPath))

	// ensure required directories exist
	if *cgroupCollectorEnabled && !utils.DirectoryExists(collectorConfig.CgroupRoot) {
		logger.Error("Cgroup root directory does not exist:", "path", collectorConfig.CgroupRoot)
		os.Exit(1)
	}
	if *jobCollectorEnabled && !utils.DirectoryExists(*pbsHome) {
		logger.Error("PBS home directory does not exist:", "path", *pbsHome)
		os.Exit(1)
	}

	// start pbs job watcher
	if *jobCollectorEnabled {
		err := collector.InitialiseJobCache(*pbsHome, logger)
		if err != nil {
			logger.Error("Failed to initialize job cache", "error", err)
		}
		go func() {
			err := collector.WatchPbsJobs(*pbsHome, logger)
			if err != nil {
				logger.Error("Failed to watch PBS jobs", "error", err)
			}
		}()
	}

	// start prometheus metrics collector
	multiCollector := collector.NewCollectors(collectorConfig)
	prometheus.MustRegister(multiCollector)
	httpHandler := newHTTPHandler()
	logger.Info("Serving metrics on", "address", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, httpHandler))
}
