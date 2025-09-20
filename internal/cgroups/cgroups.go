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

	Cpu     CPU
	Hugetlb []Hugetlb
	Io      []IO
	Memory  Memory
	Tasks   Tasks
}

type CPU struct {
	Count  int
	System float64
	Usage  float64
	User   float64
}

type Hugetlb struct {
	Max      float64
	Pagesize string
	Usage    float64
}

type IO struct {
	Major  uint64
	Rbytes float64
	Rios   float64
	Wbytes float64
	Wios   float64
}

type Memory struct {
	ActiveAnon   float64
	ActiveFile   float64
	FileMapped   float64
	InactiveAnon float64
	InactiveFile float64
	Limit        float64
	Pgfault      float64
	Pgmajfault   float64
	Rss          float64
	Shmem        float64
	SwapLimit    float64
	SwapUsage    float64
	Usage        float64
	Wss          float64
}

type Tasks struct {
	PidLimit    float64
	PidUsage    float64
	Pids        []uint64
	ThreadLimit float64
	ThreadUsage float64
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
