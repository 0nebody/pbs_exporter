package collector

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var procfsIo = `rchar: 0
wchar: 0
syscr: 0
syscw: 0
read_bytes: 200
write_bytes: 100
cancelled_write_bytes: 0
`

func TestGetCgroupIo(t *testing.T) {
	logger := slog.Default()
	pids := []uint64{1, 2, 3}
	wantRead := uint64(600)
	wantWrite := uint64(300)

	// Create mock procfs files for testing
	procFs := t.TempDir()
	for _, pid := range pids {
		pidDir := filepath.Join(procFs, strconv.FormatUint(pid, 10))
		if err := os.MkdirAll(pidDir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", pidDir, err)
		}
		ioFile := filepath.Join(pidDir, "io")
		if err := os.WriteFile(ioFile, []byte(procfsIo), 0644); err != nil {
			t.Fatalf("Failed to write io file %s: %v", ioFile, err)
		}
	}

	t.Run("Test empty procs", func(t *testing.T) {
		_, _, err := GetCgroupIo("", pids, logger)
		if err == nil {
			t.Fatalf("GetCgroupIo('', %v, <logger>) expected error for empty procfs, got nil", pids)
		}
	})

	t.Run("Test success with invalid proc ids", func(t *testing.T) {
		pids := []uint64{4, 5, 6}
		wantRead, wantWrite := uint64(0), uint64(0)
		ioRead, ioWrite, err := GetCgroupIo(procFs, pids, logger)
		if err != nil {
			t.Fatalf("GetCgroupIo(%s, %v, <logger>) unexpected error: %v", procFs, pids, err)
		}
		if ioRead != wantRead || ioWrite != wantWrite {
			t.Errorf("GetCgroupIo(%s, %v, <logger>) = %d %d, want %d %d", procFs, pids, ioRead, ioWrite, wantRead, wantWrite)
		}
	})

	t.Run("Test success", func(t *testing.T) {
		gotRead, gotWrite, err := GetCgroupIo(procFs, pids, logger)
		if err != nil {
			t.Fatalf("GetCgroupIo(%s, %v, <logger>) unexpected error: %v", procFs, pids, err)
		}
		if gotRead != wantRead || gotWrite != wantWrite {
			t.Errorf("GetCgroupIo(%s, %v, <logger>) = %v %v, want %v %v", procFs, pids, gotRead, gotWrite, wantRead, wantWrite)
		}
	})
}

func TestDescribeProcs(t *testing.T) {
	procCollector := NewProcCollector(configEnabled)
	ch := make(chan *prometheus.Desc)
	go func() {
		defer close(ch)
		procCollector.Describe(ch)
	}()

	got := 0
	want := reflect.TypeOf(*procCollector.metrics).NumField()
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

// This method is required by the interface and used only for the tests.
func (p *ProcCollector) Collect(ch chan<- prometheus.Metric) {
	p.CollectForCgroup(ch, "", []uint64{}, "")
}

func TestCollectProcs(t *testing.T) {
	procCollector := NewProcCollector(configEnabled)
	registry := prometheus.NewRegistry()
	registry.MustRegister(procCollector)

	got := testutil.CollectAndCount(registry)
	want := reflect.TypeOf(*procCollector.metrics).NumField()
	if got < want {
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
