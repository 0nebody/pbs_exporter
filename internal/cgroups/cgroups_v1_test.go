package cgroups

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/containerd/cgroups/v3/cgroup1"
	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
)

type Name cgroup1.Name

type MockCgroupV1 struct {
	MockProcsErr      error
	MockProcsVal      []uint64
	MockStatErr       error
	MockStatVal       *v1.Metrics
	MockSubsystemsVal []string
	MockTasksErr      error
	MockTasksVal      []uint64
}

func (m *MockCgroupV1) Processes(cgroup1.Name, bool) ([]cgroup1.Process, error) {
	Processes := make([]cgroup1.Process, len(m.MockProcsVal))
	for i, pid := range m.MockProcsVal {
		Processes[i] = cgroup1.Process{
			Pid: int(pid),
		}
	}
	return Processes, m.MockProcsErr
}

func (m *MockCgroupV1) Tasks(cgroup1.Name, bool) ([]cgroup1.Task, error) {
	Tasks := make([]cgroup1.Task, len(m.MockTasksVal))
	for i, pid := range m.MockTasksVal {
		Tasks[i] = cgroup1.Task{
			Pid: int(pid),
		}
	}
	return Tasks, m.MockTasksErr
}

func (m *MockCgroupV1) Stat(...cgroup1.ErrorHandler) (*v1.Metrics, error) {
	return m.MockStatVal, m.MockStatErr
}

func (s Name) Name() cgroup1.Name {
	return cgroup1.Name(string(s))
}

func (m *MockCgroupV1) Subsystems() []cgroup1.Subsystem {
	subsys := "cpu"
	return []cgroup1.Subsystem{Name(subsys)}
}

func TestLoadV1(t *testing.T) {
	t.Run("Load success", func(t *testing.T) {
		cgroupFs := t.TempDir()
		os.MkdirAll(filepath.Join(cgroupFs, "cpuset"), 0755)
		CgroupV1Manager := &CgroupV1Manager{
			root: cgroupFs,
		}
		_, err := CgroupV1Manager.Load("")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Load error", func(t *testing.T) {
		CgroupV1Manager := &CgroupV1Manager{}
		_, err := CgroupV1Manager.Load("")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestStatV1(t *testing.T) {
	MockCgroupV1 := &MockCgroupV1{
		MockStatVal: &v1.Metrics{
			Blkio: &v1.BlkIOStat{
				IoServiceBytesRecursive: []*v1.BlkIOEntry{
					{
						Op:     "Read",
						Device: "dm-0",
						Major:  253,
						Minor:  0,
						Value:  1000,
					},
					{
						Op:     "Write",
						Device: "dm-0",
						Major:  253,
						Minor:  0,
						Value:  2000,
					},
				},
				IoServicedRecursive: []*v1.BlkIOEntry{
					{
						Op:     "Read",
						Device: "dm-0",
						Major:  253,
						Minor:  0,
						Value:  100,
					},
					{
						Op:     "Write",
						Device: "dm-0",
						Major:  253,
						Minor:  0,
						Value:  200,
					},
				},
			},
			CPU: &v1.CPUStat{
				Usage: &v1.CPUUsage{
					Total:  1000000000,
					Kernel: 1000000000,
					User:   1000000000,
					PerCPU: []uint64{1000000000, 1000000000},
				},
			},
			Hugetlb: []*v1.HugetlbStat{
				{
					Failcnt:  2,
					Max:      2,
					Pagesize: "2MB",
					Usage:    2,
				},
			},
			Memory: &v1.MemoryStat{
				TotalActiveAnon:   1,
				TotalActiveFile:   2,
				TotalInactiveAnon: 4,
				TotalInactiveFile: 5,
				TotalPgFault:      0,
				TotalPgMajFault:   0,
				TotalRSS:          333,
				Usage: &v1.MemoryEntry{
					Limit:   999,
					Usage:   222,
					Max:     222,
					Failcnt: 444,
				},
				Swap: &v1.MemoryEntry{
					Limit:   0,
					Usage:   0,
					Max:     0,
					Failcnt: 0,
				},
			},
			Pids: &v1.PidsStat{
				Current: 100,
				Limit:   200,
			},
		},
		MockStatErr: nil,
	}
	cgroup := &CgroupV1{
		cgroup:     MockCgroupV1,
		subsystems: []string{"blkio", "cpu", "hugetlb", "memory", "pids"},
	}
	testMetric.Controllers = cgroup.subsystems
	got, err := cgroup.Stat()
	if err != nil {
		t.Fatalf("Stat() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, testMetric) {
		t.Errorf("Stat() = %+v, want %+v", got, testMetric)
	}
}

func TestProcsV1(t *testing.T) {
	want := []uint64{1234, 5678}
	MockCgroupV1 := &MockCgroupV1{
		MockProcsVal: want,
		MockProcsErr: nil,
	}
	cgroup := &CgroupV1{
		cgroup: MockCgroupV1,
	}
	got, err := cgroup.Procs()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Procs() = %v, want %v", got, want)
	}
}

func TestThreadsV1(t *testing.T) {
	want := []uint64{1234, 5678}
	MockCgroupV1 := &MockCgroupV1{
		MockTasksVal: want,
		MockTasksErr: nil,
	}
	cgroup := &CgroupV1{
		cgroup: MockCgroupV1,
	}
	got, err := cgroup.Threads()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("Threads() = %v, want %v", got, want)
	}
}

func TestCpuCountV1(t *testing.T) {
	cgroupFs := t.TempDir()
	cpuset := []byte("0-1\n")
	want := 2

	cgroupV1 := &CgroupV1{
		root: cgroupFs,
		path: "",
	}
	if err := os.MkdirAll(filepath.Join(cgroupFs, "cpuset"), 0755); err != nil {
		t.Fatalf("Failed to create cpuset directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cgroupFs, "cpuset", "cpuset.cpus"), cpuset, 0644); err != nil {
		t.Fatalf("Failed to write cpuset.cpus: %v", err)
	}

	got, err := cgroupV1.CpuCount()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got != want {
		t.Errorf("CpuCount() = %d, want %d", got, want)
	}
}

func TestCgroupsV1Hierarchy(t *testing.T) {
	cgroupFs := t.TempDir()
	hierarchy, err := cgroupsV1Hierarchy(cgroupFs)()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(hierarchy) == 0 {
		t.Fatal("Expected non-empty hierarchy")
	}
	got := []string{}
	want := []string{"cpu", "cpuacct", "cpuset", "memory", "pids", "hugetlb", "blkio", "systemd", "freezer", "net_cls", "net_prio", "perf_event", "rdma"}

	for _, subsystem := range hierarchy {
		got = append(got, string(subsystem.Name()))
	}
	sort.Strings(got)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Errorf("cgroupsV1Hierarchy(%s) = %v, want %v", cgroupFs, got, want)
	}
}
