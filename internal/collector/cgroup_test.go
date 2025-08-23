package collector

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0nebody/pbs_exporter/internal/pbsjobs"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDescribeCgroups(t *testing.T) {
	cm := NewCgroupMetrics(configEnabled)
	ch := make(chan *prometheus.Desc, 30)
	cm.Describe(ch)
	close(ch)

	got := 0
	want := 23
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
	cmConfig := configEnabled
	cmConfig.EnableJobCollector = false
	cmConfig.CgroupRoot = "/sys/fs/cgroup"
	cmConfig.CgroupPath = "user.slice"
	cm := NewCgroupMetrics(cmConfig)
	registry := prometheus.NewRegistry()
	registry.MustRegister(cm)
	utils.PbsJobIdRegex = regexp.MustCompile(`/user.slice/user-(\d+).slice`)

	t.Run("CollectAndCount", func(t *testing.T) {
		got := testutil.CollectAndCount(registry)
		want := 17
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
		cmConfig.EnableJobCollector = true
		jobCache = pbsjobs.NewJobCache(cm.logger, 60, 15*time.Second)
		cm := NewCgroupMetrics(cmConfig)
		registry := prometheus.NewRegistry()
		registry.MustRegister(cm)
		got := testutil.CollectAndCount(registry)
		want := 0
		if got != want {
			t.Errorf("CollectAndCount() = %d, want %d", got, want)
		}
	})
}
