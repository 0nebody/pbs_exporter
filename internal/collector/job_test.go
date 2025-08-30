package collector

import (
	"strings"
	"testing"
	"time"

	"github.com/0nebody/pbs_exporter/internal/pbsjobs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDescribeJobs(t *testing.T) {
	jobCollector := NewJobCollector(configEnabled)
	ch := make(chan *prometheus.Desc, 30)
	jobCollector.Describe(ch)
	close(ch)

	got := 0
	want := 12
	for desc := range ch {
		got++

		fqName := promDescFqname(desc.String())
		if !strings.HasPrefix(fqName, "pbs_job_") {
			t.Errorf("Describe() = %s, want: %s", fqName, "pbs_job_.*")
		}

		help := promDescHelp(desc.String())
		if len(help) == 0 {
			t.Errorf("Describe() expected help to be non-empty description of metric")
		}
	}

	if got != want {
		t.Errorf("Describe() = %d, want %d", got, want)
	}
}

func TestCollectJobs(t *testing.T) {
	hostname = "cpu1n001"
	jobCollector := NewJobCollector(configEnabled)
	jobCache = pbsjobs.NewJobCache(jobCollector.logger, 60, 15*time.Second)
	jobCache.Set("1000", &pbsjobs.Job{
		ExecHost:    "cpu1n001",
		JobName:     "test",
		JobOwner:    "test",
		Hashname:    "1000.pbs",
		JobState:    "R",
		Queue:       "batch",
		Interactive: 1,
		Mtime:       time.Now().Unix(),
		ResourceList: pbsjobs.ResourceList{
			Walltime: "00:00:01",
			Mem:      34359738368,
			Ncpus:    1,
			Nfpgas:   0,
			Ngpus:    1,
		},
		SchedSelect: "1:ncpus=4:ngpus=1:mem=32gb:nfpgas=0",
		Euser:       "user",
		Egroup:      "group",
		RunCount:    1,
		Project:     "project",
		RunVersion:  "1",
		Stime:       time.Now().Unix(),
	})
	registry := prometheus.NewRegistry()
	registry.MustRegister(jobCollector)

	got := testutil.CollectAndCount(registry)
	want := 11
	if got != want {
		t.Errorf("CollectAndCount() = %d, want %d", got, want)
	}

	lint, err := testutil.CollectAndLint(registry)
	if err != nil {
		t.Fatalf("CollectAndLint failed: %v", err)
	}
	if len(lint) > 0 {
		t.Errorf("CollectAndLint found issues: %v", lint)
	}
}
