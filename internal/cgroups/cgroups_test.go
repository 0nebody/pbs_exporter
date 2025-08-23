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
	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
)

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

type Name cgroup1.Name

func (s Name) Name() cgroup1.Name {
	return cgroup1.Name(string(s))
}

func (m *MockCgroupV1) Subsystems() []cgroup1.Subsystem {
	subsys := "cpu"
	return []cgroup1.Subsystem{Name(subsys)}
}

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

func TestLoad(t *testing.T) {
	t.Run("CgroupV1 load success", func(t *testing.T) {
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

	t.Run("CgroupV2 load success", func(t *testing.T) {
		CgroupV2Manager := &CgroupV2Manager{
			root: "/sys/fs/cgroup",
		}

		_, err := CgroupV2Manager.Load("/user.slice")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("CgroupV1 load error", func(t *testing.T) {
		CgroupV1Manager := &CgroupV1Manager{}

		_, err := CgroupV1Manager.Load("")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	})

	t.Run("CgroupV2 load error", func(t *testing.T) {
		CgroupV2Manager := &CgroupV2Manager{}

		_, err := CgroupV2Manager.Load("")
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	})
}

func TestStat(t *testing.T) {
	want := &Metrics{
		Path: "",
		Cpu: CPU{
			Count:  0,
			System: 1.0,
			Usage:  1.0,
			User:   1.0,
		},
		Hugetlb: []Hugetlb{
			{
				Max:      2.0,
				Pagesize: "2MB",
				Usage:    2.0,
			},
		},
		Io: []IO{
			{
				Major:  253,
				Rbytes: 1000.0,
				Rios:   100.0,
				Wbytes: 2000.0,
				Wios:   200.0,
			},
		},
		Memory: Memory{
			AnonUsage:       0.0,
			FileMappedUsage: 0.0,
			FileUsage:       0.0,
			Limit:           111.0,
			Pgfault:         0.0,
			Pgmajfault:      0.0,
			ShmemUsage:      0.0,
			SwapLimit:       0.0,
			SwapUsage:       0.0,
			Usage:           222.0,
		},
		Tasks: Tasks{
			PidLimit:    200.0,
			PidUsage:    0.0,
			ThreadLimit: 0.0,
			ThreadUsage: 0.0,
		},
	}

	t.Run("CgroupV1", func(t *testing.T) {
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
					Usage: &v1.MemoryEntry{
						Limit:   111,
						Usage:   222,
						Max:     333,
						Failcnt: 444,
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
		want.Controllers = cgroup.subsystems
		got, err := cgroup.Stat()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Stat() = %v, want %v", got, want)
		}
	})

	t.Run("CgroupV2", func(t *testing.T) {
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
					Usage:      222,
					UsageLimit: 111,
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
		want.Controllers = cgroup.controllers
		got, err := cgroup.Stat()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Stat() = %v, want %v", got, want)
		}
	})
}

func TestProcs(t *testing.T) {
	t.Run("CgroupV1", func(t *testing.T) {
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
	})

	t.Run("CgroupV2", func(t *testing.T) {
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
	})
}

func TestThreads(t *testing.T) {
	t.Run("CgroupV1", func(t *testing.T) {
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
	})

	t.Run("CgroupV2", func(t *testing.T) {
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
	})
}

func TestCpuCount(t *testing.T) {
	cgroupFs := t.TempDir()
	cpuset := []byte("0-1\n")
	want := 2

	t.Run("CgroupV1 CpuCount", func(t *testing.T) {
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
	})

	t.Run("CgroupV2 CpuCount", func(t *testing.T) {
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
	})
}

func TestNewCgroupManager(t *testing.T) {
	manager := NewCgroupManager("")

	if manager == nil {
		t.Fatal("Expected non-nil CgroupManager")
	}

	// assumes that host running test is using cgroup v2
	if manager.Version() != "v2" {
		t.Errorf("Expected version v2, got %s", manager.Version())
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

func TestListCgroups(t *testing.T) {
	cgroupRoot := t.TempDir()
	cgroupPath := "testcgroup"

	subdirs := []string{
		// cgroup v1 paths
		filepath.Join(cgroupRoot, "cpu,cpuacct", cgroupPath, "1000.pbs"),
		filepath.Join(cgroupRoot, "cpu,cpuacct", cgroupPath, "1002.pbs", "1"),
		// cgroup v2 paths
		filepath.Join(cgroupRoot, cgroupPath, "1000.pbs"),
		filepath.Join(cgroupRoot, cgroupPath, "1002.pbs", "1"),
	}
	for _, dir := range subdirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}
	want := []string{
		"/" + cgroupPath + "/1000.pbs",
		"/" + cgroupPath + "/1002.pbs",
	}

	t.Run("listCgroups success", func(t *testing.T) {
		got, err := listCgroups(cgroupRoot, cgroupPath)
		if err != nil {
			t.Fatalf("listCgroups returned error: %v", err)
		}

		if !slices.Equal(slices.Compact(got), slices.Compact(want)) {
			t.Errorf("listCgroups(%v, %v) = %v, want %v", cgroupRoot, cgroupPath, got, want)
		}
	})

	t.Run("cgroupV1 list success", func(t *testing.T) {
		cgroupV1Manager := &CgroupV1Manager{
			root: cgroupRoot,
		}

		got, err := cgroupV1Manager.List(cgroupPath)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !slices.Equal(slices.Compact(got), slices.Compact(want)) {
			t.Errorf("List(%v) = %v, want %v", cgroupPath, got, want)
		}
	})

	t.Run("cgroupV2 list success", func(t *testing.T) {
		cgroupV2Manager := &CgroupV2Manager{
			root: cgroupRoot,
		}

		got, err := cgroupV2Manager.List(cgroupPath)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !slices.Equal(slices.Compact(got), slices.Compact(want)) {
			t.Errorf("List(%v) = %v, want %v", cgroupPath, got, want)
		}
	})

	t.Run("NonexistentPath", func(t *testing.T) {
		_, err := listCgroups("/nonexistent", "path")
		if err == nil {
			t.Error("Expected error for nonexistent path, got nil")
		}
	})
}

func TestGetCgroupCPUs(t *testing.T) {
	cgroupFs := t.TempDir()
	tests := []struct {
		name   string
		cpuset string
		want   []int
	}{
		{
			name:   "Success single cpu",
			cpuset: "1\n",
			want:   []int{1},
		},
		{
			name:   "Success single range",
			cpuset: "1-2\n",
			want:   []int{1, 2},
		},
		{
			name:   "Success mixed",
			cpuset: "1-2,7,20-30\n",
			want:   []int{1, 2, 7, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30},
		},
		{
			name:   "failure empty cpuset",
			cpuset: "\n",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(cgroupFs, "cpuset.cpus"), []byte(tt.cpuset), 0644); err != nil {
				t.Fatalf("Failed to write cpus file: %v", err)
			}
			cpus, err := GetCgroupCPUs(cgroupFs, "")
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if !slices.Equal(cpus, tt.want) {
				t.Errorf("GetCgroupCPUs(%v, '') %v, want %v", cgroupFs, cpus, tt.want)
			}
		})
	}
}
