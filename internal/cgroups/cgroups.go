package cgroups

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/containerd/cgroups"
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

func listCgroups(cgroupRoot string, cgroupPath string) ([]string, error) {
	var cgroupPaths []string

	walkPath := filepath.Join(cgroupRoot, cgroupPath)
	maxDepth := strings.Count(walkPath, string(os.PathSeparator)) + 1

	err := filepath.WalkDir(walkPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path == walkPath {
			return nil
		}
		if strings.Count(path, string(os.PathSeparator)) > maxDepth {
			return fs.SkipDir
		}

		relPath := strings.TrimPrefix(path, cgroupRoot)
		cgroupPaths = append(cgroupPaths, relPath)

		return nil
	})

	return cgroupPaths, err
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
