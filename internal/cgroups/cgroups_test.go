package cgroups

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

var testMetric = &Metrics{
	Path: "",
	Cpu: CPU{
		Count:  0,
		System: 1,
		Usage:  1,
		User:   1,
	},
	Hugetlb: []Hugetlb{
		{
			Max:      2,
			Pagesize: "2MB",
			Usage:    2,
		},
	},
	Io: IO{
		Usage: []IoUsage{
			{
				Major:  253,
				Rbytes: 1000,
				Rios:   100,
				Wbytes: 2000,
				Wios:   200,
			},
		},
	},
	Memory: Memory{
		ActiveAnon:   1,
		ActiveFile:   2,
		FileMapped:   328,
		InactiveAnon: 4,
		InactiveFile: 5,
		Limit:        999,
		Pgfault:      0,
		Pgmajfault:   0,
		Rss:          333,
		Shmem:        0,
		SwapLimit:    0,
		SwapUsage:    0,
		Usage:        222,
		Wss:          217,
	},
	Tasks: Tasks{
		PidLimit:    200,
		PidUsage:    0,
		Pids:        []uint64(nil),
		ThreadLimit: 0,
		ThreadUsage: 0,
		Threads:     []uint64(nil),
	},
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
