package cgroups

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
)

type MockCgroupV2 struct {
	MockProcsErr       error
	MockProcsVal       []uint64
	MockThreadsErr     error
	MockThreadsVal     []uint64
	MockStatErr        error
	MockStatVal        *v2.Metrics
	MockControllersVal []string
	MockControllersErr error
}

func (m *MockCgroupV2) Procs(bool) ([]uint64, error) {
	return m.MockProcsVal, m.MockProcsErr
}

func (m *MockCgroupV2) Threads(bool) ([]uint64, error) {
	return m.MockThreadsVal, m.MockThreadsErr
}

func (m *MockCgroupV2) Stat() (*v2.Metrics, error) {
	return m.MockStatVal, m.MockStatErr
}

func (m *MockCgroupV2) Controllers() ([]string, error) {
	return m.MockControllersVal, m.MockControllersErr
}

func TestLoadV2(t *testing.T) {
	t.Run("Load success", func(t *testing.T) {
		CgroupV2Manager := &CgroupV2Manager{
			root: "/sys/fs/cgroup",
		}
		_, err := CgroupV2Manager.Load("/user.slice")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Load error", func(t *testing.T) {
		CgroupV2Manager := &CgroupV2Manager{}
		_, err := CgroupV2Manager.Load("")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestStatV2(t *testing.T) {
	MockCgroupV2 := &MockCgroupV2{
		MockStatVal: &v2.Metrics{
			CPU: &v2.CPUStat{
				UsageUsec:  1000000,
				UserUsec:   1000000,
				SystemUsec: 1000000,
			},
			Hugetlb: []*v2.HugeTlbStat{
				{
					Current:  2,
					Max:      2,
					Pagesize: "2MB",
				},
			},
			Io: &v2.IOStat{
				Usage: []*v2.IOEntry{
					{
						Major:  253,
						Rbytes: 1000,
						Rios:   100,
						Wbytes: 2000,
						Wios:   200,
					},
				},
			},
			Memory: &v2.MemoryStat{
				ActiveAnon:   1.0,
				ActiveFile:   2.0,
				Anon:         5,
				FileMapped:   328,
				InactiveAnon: 4.0,
				InactiveFile: 5.0,
				Pgfault:      0.0,
				Pgmajfault:   0.0,
				Usage:        222,
				UsageLimit:   999,
			},
			Pids: &v2.PidsStat{
				Current: 100,
				Limit:   200,
			},
		},
		MockStatErr: nil,
	}
	cgroup := &CgroupV2{
		cgroup:      MockCgroupV2,
		controllers: []string{"cpu", "hugetlb", "io", "memory", "pids"},
	}
	testMetric.Controllers = cgroup.controllers
	got, err := cgroup.Stat()
	if err != nil {
		t.Fatalf("Stat() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, testMetric) {
		t.Errorf("Stat() = %+v, want %+v", got, testMetric)
	}
}

func TestProcsV2(t *testing.T) {
	want := []uint64{1234, 5678}
	MockCgroupV2 := &MockCgroupV2{
		MockProcsVal: want,
		MockProcsErr: nil,
	}
	cgroup := &CgroupV2{
		cgroup: MockCgroupV2,
	}
	got, err := cgroup.Procs()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Procs() = %v, want %v", got, want)
	}
}

func TestThreadsV2(t *testing.T) {
	want := []uint64{1234, 5678}
	MockCgroupV2 := &MockCgroupV2{
		MockThreadsVal: want,
		MockThreadsErr: nil,
	}
	cgroup := &CgroupV2{
		cgroup: MockCgroupV2,
	}
	got, err := cgroup.Threads()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Threads() = %v, want %v", got, want)
	}
}

func TestCpuCountV2(t *testing.T) {
	cgroupFs := t.TempDir()
	cpuset := []byte("0-1\n")
	want := 2

	cgroupV2 := &CgroupV2{
		root: cgroupFs,
		path: "",
	}
	if err := os.WriteFile(filepath.Join(cgroupFs, "cpuset.cpus"), cpuset, 0644); err != nil {
		t.Fatalf("Failed to write cpuset.cpus: %v", err)
	}

	got, err := cgroupV2.CpuCount()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got != want {
		t.Errorf("CpuCount() = %d, want %d", got, want)
	}
}
