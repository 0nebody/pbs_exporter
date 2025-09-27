package collector

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0nebody/pbs_exporter/internal/pbsjob"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDescribeCgroups(t *testing.T) {
	cgroupCollector := NewCgroupCollector(configEnabled)
	ch := make(chan *prometheus.Desc)
	go func() {
		defer close(ch)
		cgroupCollector.Describe(ch)
	}()

	got := 0
	want := reflect.TypeOf(*cgroupCollector.metrics).NumField()
	for desc := range ch {
		got++

		fqName := promDescFqname(desc.String())
		if !strings.HasPrefix(fqName, "pbs_cgroup_") {
			t.Errorf("Describe() = %s, want: %s", fqName, "pbs_cgroup_.*")
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

func TestCollectCgroups(t *testing.T) {
	config := configEnabled
	config.EnableJobCollector = false
	config.CgroupRoot = "/sys/fs/cgroup"
	config.CgroupPath = "user.slice"
	cgroupCollector := NewCgroupCollector(config)
	registry := prometheus.NewRegistry()
	registry.MustRegister(newCollectorContext(cgroupCollector))
	utils.PbsJobIdRegex = regexp.MustCompile(`/user.slice/user-(\d+).slice`)

	t.Run("CollectAndCount", func(t *testing.T) {
		got := testutil.CollectAndCount(registry)
		// assumes io and hugetlb disabled in test environment
		want := reflect.TypeOf(*cgroupCollector.metrics).NumField() - 6
		if got < want {
			t.Errorf("CollectAndCount() = %d, want %d", got, want)
		}
	})

	t.Run("CollectAndLint", func(t *testing.T) {
		lint, err := testutil.CollectAndLint(registry)
		if err != nil {
			t.Fatalf("CollectAndLint failed: %v", err)
		}
		if len(lint) > 0 {
			t.Errorf("CollectAndLint found issues: %v", lint)
		}
	})

	t.Run("CollectNoJobID", func(t *testing.T) {
		config.EnableJobCollector = true
		cgroupCollector := NewCgroupCollector(config)
		jobCache = pbsjob.NewJobCache(cgroupCollector.logger, 60, 15*time.Second)
		registry := prometheus.NewRegistry()
		registry.MustRegister(newCollectorContext(cgroupCollector))

		got := testutil.CollectAndCount(registry)
		want := 0
		if got != want {
			t.Errorf("CollectAndCount() = %d, want %d", got, want)
		}
	})
}
