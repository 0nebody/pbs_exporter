package cgroups

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/containerd/cgroups/v3"
)

var (
	ErrCgroupUninitialised = errors.New("cgroup uninitialised")
)

type CgroupManager interface {
	List(path string) ([]string, error)
	Load(path string) (Cgroup, error)
	Version() string
}

type Cgroup interface {
	CpuCount() (int, error)
	Procs() ([]uint64, error)
	Stat() (*Metrics, error)
	Threads() ([]uint64, error)
}

type Metrics struct {
	Path        string
	Controllers []string
	Io          IO
	Cpu         CPU
	Hugetlb     []Hugetlb
	Memory      Memory
	Tasks       Tasks
}

type CPU struct {
	Count  int
	System uint64
	Usage  uint64
	User   uint64
}

type Hugetlb struct {
	Max      uint64
	Pagesize string
	Usage    uint64
}

type IO struct {
	Usage []IoUsage
}

type IoUsage struct {
	Major  uint64
	Rbytes uint64
	Rios   uint64
	Wbytes uint64
	Wios   uint64
}

type Memory struct {
	ActiveAnon   uint64
	ActiveFile   uint64
	FileMapped   uint64
	InactiveAnon uint64
	InactiveFile uint64
	Limit        uint64
	Pgfault      uint64
	Pgmajfault   uint64
	Rss          uint64
	Shmem        uint64
	SwapLimit    uint64
	SwapUsage    uint64
	Usage        uint64
	Wss          uint64
}

type Tasks struct {
	PidLimit    uint64
	PidUsage    uint64
	Pids        []uint64
	ThreadLimit uint64
	ThreadUsage uint64
	Threads     []uint64
}

func NewCgroupManager(root string) CgroupManager {
	if cgroups.Mode() == cgroups.Unified {
		return &CgroupV2Manager{
			version: "v2",
			root:    root,
		}
	} else {
		return &CgroupV1Manager{
			version: "v1",
			root:    root,
		}
	}
}

func listCgroups(root string, path string) ([]string, error) {
	cgroupPath := filepath.Join(root, path)

	entries, err := os.ReadDir(cgroupPath)
	if err != nil {
		return nil, err
	}

	var cgroupPaths []string
	for _, d := range entries {
		if d.IsDir() {
			relPath := filepath.Join("/", path, d.Name())
			cgroupPaths = append(cgroupPaths, relPath)
		}
	}

	return cgroupPaths, nil
}

func GetCgroupCPUs(cgroupRoot string, cgroupPath string) ([]int, error) {
	cpuSetPath := filepath.Join(cgroupRoot, cgroupPath, "cpuset.cpus")
	cpuSetList, err := utils.ReadFileSingleLine(cpuSetPath)
	if err != nil {
		return nil, err
	}

	cpuSet, err := utils.ParseListFormat(cpuSetList)
	if err != nil {
		return nil, err
	}

	return cpuSet, nil
}
