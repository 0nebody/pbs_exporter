package cgroups

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/containerd/cgroups"
	"github.com/containerd/cgroups/v3/cgroup1"
	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	"github.com/containerd/cgroups/v3/cgroup2"
	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
)

var (
	ErrCgroupUninitialised = errors.New("cgroup uninitialised")
)

type CgroupManager interface {
	List(path string) ([]string, error)
	Load(path string) (Cgroup, error)
	Version() string
}

type CgroupV1Api interface {
	Processes(cgroup1.Name, bool) ([]cgroup1.Process, error)
	Stat(...cgroup1.ErrorHandler) (*v1.Metrics, error)
	Subsystems() []cgroup1.Subsystem
	Tasks(cgroup1.Name, bool) ([]cgroup1.Task, error)
}

type CgroupV2Api interface {
	Procs(bool) ([]uint64, error)
	Controllers() ([]string, error)
	Stat() (*v2.Metrics, error)
	Threads(bool) ([]uint64, error)
}

type Cgroup interface {
	CpuCount() (int, error)
	Procs() ([]uint64, error)
	Stat() (*Metrics, error)
	Threads() ([]uint64, error)
}

type CgroupV1Manager struct {
	version string
	root    string
}

type CgroupV2Manager struct {
	version string
	root    string
}

type CgroupV1 struct {
	root       string
	path       string
	subsystems []string
	cgroup     CgroupV1Api
}

type CgroupV2 struct {
	root        string
	path        string
	controllers []string
	cgroup      CgroupV2Api
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
	AnonUsage       float64
	FileMappedUsage float64
	FileUsage       float64
	Limit           float64
	Pgfault         float64
	Pgmajfault      float64
	ShmemUsage      float64
	SwapLimit       float64
	SwapUsage       float64
	Usage           float64
}

type Tasks struct {
	PidLimit    float64
	Pids        []uint64
	PidUsage    float64
	ThreadLimit float64
	Threads     []uint64
	ThreadUsage float64
}

func (m *CgroupV1Manager) List(path string) ([]string, error) {
	return listCgroups(filepath.Join(m.root, "/cpu,cpuacct"), path)
}

func (m *CgroupV2Manager) List(path string) ([]string, error) {
	return listCgroups(m.root, path)
}

func (m *CgroupV1Manager) Load(path string) (Cgroup, error) {
	hierarchy := cgroup1.WithHierarchy(cgroupsV1Hierarchy(m.root))
	cgroup, err := cgroup1.Load(cgroup1.StaticPath(path), hierarchy)
	if err != nil {
		return nil, err
	}

	subsystems := []string{}
	for _, subsystem := range cgroup.Subsystems() {
		subsystems = append(subsystems, string(subsystem.Name()))
	}

	return &CgroupV1{
		root:       m.root,
		path:       path,
		cgroup:     cgroup,
		subsystems: subsystems,
	}, nil
}

func (m *CgroupV2Manager) Load(path string) (Cgroup, error) {
	opts := cgroup2.WithMountpoint(m.root)
	cgroup, err := cgroup2.Load(path, opts)
	if err != nil {
		return nil, err
	}

	controllers, err := cgroup.Controllers()
	if err != nil {
		return nil, err
	}

	return &CgroupV2{
		root:        m.root,
		path:        path,
		cgroup:      cgroup,
		controllers: controllers,
	}, nil
}

func (c *CgroupV1Manager) Version() string {
	return c.version
}

func (c *CgroupV2Manager) Version() string {
	return c.version
}

func (c *CgroupV1) Stat() (*Metrics, error) {
	metrics := &Metrics{
		Path:        c.path,
		Controllers: c.subsystems,
	}

	stat, err := c.cgroup.Stat()
	if err != nil {
		return nil, err
	}

	if slices.Contains(metrics.Controllers, "blkio") {
		statIO := stat.GetBlkio()
		ioMap := make(map[uint64]*IO)
		ioMapGet := func(major uint64) *IO {
			if _, ok := ioMap[major]; !ok {
				ioMap[major] = &IO{Major: major}
			}
			return ioMap[major]
		}

		for _, ioUsage := range statIO.GetIoServiceBytesRecursive() {
			ioStat := ioMapGet(ioUsage.GetMajor())
			switch operation := ioUsage.GetOp(); operation {
			case "Read":
				ioStat.Rbytes += float64(ioUsage.GetValue())
			case "Write":
				ioStat.Wbytes += float64(ioUsage.GetValue())
			}
		}

		for _, ioUsage := range statIO.GetIoServicedRecursive() {
			ioStat := ioMapGet(ioUsage.GetMajor())
			switch operation := ioUsage.GetOp(); operation {
			case "Read":
				ioStat.Rios += float64(ioUsage.GetValue())
			case "Write":
				ioStat.Wios += float64(ioUsage.GetValue())
			}
		}

		for _, ioStat := range ioMap {
			metrics.Io = append(metrics.Io, *ioStat)
		}
	}

	if slices.Contains(metrics.Controllers, "cpu") {
		statCPU := stat.GetCPU()
		statCPUUsage := statCPU.GetUsage()
		metrics.Cpu.System = float64(statCPUUsage.GetKernel()) / 1000000000.0
		metrics.Cpu.Usage = float64(statCPUUsage.GetTotal()) / 1000000000.0
		metrics.Cpu.User = float64(statCPUUsage.GetUser()) / 1000000000.0
	}

	if slices.Contains(metrics.Controllers, "cpuset") {
		cpuCount, err := c.CpuCount()
		if err != nil {
			return nil, err
		}
		metrics.Cpu.Count = cpuCount
	}

	if slices.Contains(metrics.Controllers, "hugetlb") {
		statHugetlb := stat.GetHugetlb()
		for _, hugetlb := range statHugetlb {
			metrics.Hugetlb = append(metrics.Hugetlb, Hugetlb{
				Max:      float64(hugetlb.GetMax()),
				Pagesize: hugetlb.GetPagesize(),
				Usage:    float64(hugetlb.GetUsage()),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "memory") {
		statMemory := stat.GetMemory()
		statMemoryUsage := statMemory.GetUsage()
		statMemorySwap := statMemory.GetSwap()
		metrics.Memory.AnonUsage = float64(statMemory.GetTotalRSS())
		metrics.Memory.FileMappedUsage = float64(statMemory.GetMappedFile())
		metrics.Memory.FileUsage = float64(statMemory.GetTotalCache())
		metrics.Memory.Limit = float64(statMemoryUsage.GetLimit())
		metrics.Memory.Pgfault = float64(statMemory.GetPgFault())
		metrics.Memory.Pgmajfault = float64(statMemory.GetPgMajFault())
		metrics.Memory.ShmemUsage = float64(0)
		metrics.Memory.SwapLimit = float64(statMemorySwap.GetLimit())
		metrics.Memory.SwapUsage = float64(statMemorySwap.GetUsage())
		metrics.Memory.Usage = float64(statMemoryUsage.GetUsage())

		if metrics.Memory.Limit > 8e+18 {
			return nil, ErrCgroupUninitialised
		}
	}

	if slices.Contains(metrics.Controllers, "pids") {
		pids := stat.GetPids()
		metrics.Tasks.PidLimit = float64(pids.GetLimit())
		metrics.Tasks.PidUsage = float64(pids.GetCurrent())
	} else {
		metrics.Tasks.PidLimit = float64(0)
	}

	pids, err := c.Procs()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Pids = pids
	metrics.Tasks.PidUsage = float64(len(pids))

	threads, err := c.Threads()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Threads = threads
	metrics.Tasks.ThreadUsage = float64(len(threads))

	return metrics, nil
}

func (c *CgroupV2) Stat() (*Metrics, error) {
	metrics := Metrics{
		Path:        c.path,
		Controllers: c.controllers,
	}

	stat, err := c.cgroup.Stat()
	if err != nil {
		return nil, err
	}

	if slices.Contains(metrics.Controllers, "cpu") {
		statCPU := stat.GetCPU()
		metrics.Cpu.System = float64(statCPU.GetSystemUsec()) / 1000000.0
		metrics.Cpu.Usage = float64(statCPU.GetUsageUsec()) / 1000000.0
		metrics.Cpu.User = float64(statCPU.GetUserUsec()) / 1000000.0
	}

	if slices.Contains(metrics.Controllers, "cpuset") {
		cpuCount, err := c.CpuCount()
		if err != nil {
			return nil, err
		}
		metrics.Cpu.Count = cpuCount
	}

	if slices.Contains(metrics.Controllers, "hugetlb") {
		statHugetlb := stat.GetHugetlb()
		for _, hugetlb := range statHugetlb {
			metrics.Hugetlb = append(metrics.Hugetlb, Hugetlb{
				Max:      float64(hugetlb.GetMax()),
				Pagesize: hugetlb.GetPagesize(),
				Usage:    float64(hugetlb.GetCurrent()),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "io") {
		statIO := stat.GetIo()
		statIoUsage := statIO.GetUsage()
		for _, ioUsage := range statIoUsage {
			metrics.Io = append(metrics.Io, IO{
				Major:  ioUsage.GetMajor(),
				Rbytes: float64(ioUsage.GetRbytes()),
				Rios:   float64(ioUsage.GetRios()),
				Wbytes: float64(ioUsage.GetWbytes()),
				Wios:   float64(ioUsage.GetWios()),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "memory") {
		statMemory := stat.GetMemory()
		metrics.Memory.AnonUsage = float64(statMemory.GetAnon())
		metrics.Memory.FileMappedUsage = float64(statMemory.GetFileMapped())
		metrics.Memory.FileUsage = float64(statMemory.GetFile())
		metrics.Memory.Limit = float64(statMemory.GetUsageLimit())
		metrics.Memory.Pgfault = float64(statMemory.GetPgfault())
		metrics.Memory.Pgmajfault = float64(statMemory.GetPgmajfault())
		metrics.Memory.ShmemUsage = float64(statMemory.GetShmem())
		metrics.Memory.SwapLimit = float64(statMemory.GetSwapLimit())
		metrics.Memory.SwapUsage = float64(statMemory.GetSwapUsage())
		metrics.Memory.Usage = float64(statMemory.GetUsage())
	}

	if slices.Contains(metrics.Controllers, "pids") {
		pids := stat.GetPids()
		metrics.Tasks.PidLimit = float64(pids.GetLimit())
		metrics.Tasks.PidUsage = float64(pids.GetCurrent())
	} else {
		metrics.Tasks.PidLimit = float64(0)
	}

	pids, err := c.Procs()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Pids = pids
	metrics.Tasks.PidUsage = float64(len(pids))

	threads, err := c.Threads()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Threads = threads
	metrics.Tasks.ThreadUsage = float64(len(threads))

	return &metrics, nil
}

func (c *CgroupV1) Procs() ([]uint64, error) {
	processes, err := c.cgroup.Processes(cgroup1.Cpuacct, true)
	if err != nil {
		return nil, err
	}

	var processIds []uint64
	for _, pid := range processes {
		processIds = append(processIds, uint64(pid.Pid))
	}

	return processIds, nil
}

func (c *CgroupV2) Procs() ([]uint64, error) {
	processes, err := c.cgroup.Procs(true)
	if err != nil {
		return nil, err
	}

	var processIds []uint64
	for _, pid := range processes {
		processIds = append(processIds, uint64(pid))
	}

	return processIds, nil
}

func (c *CgroupV1) Threads() ([]uint64, error) {
	threads, err := c.cgroup.Tasks(cgroup1.Cpuacct, true)
	if err != nil {
		return nil, err
	}

	var threadIds []uint64
	for _, pid := range threads {
		threadIds = append(threadIds, uint64(pid.Pid))
	}

	return threadIds, nil
}

func (c *CgroupV2) Threads() ([]uint64, error) {
	threads, err := c.cgroup.Threads(true)
	if err != nil {
		return nil, err
	}

	var threadIds []uint64
	for _, pid := range threads {
		threadIds = append(threadIds, uint64(pid))
	}

	return threadIds, nil
}

func (c *CgroupV1) CpuCount() (int, error) {
	cgroupCPUs, err := GetCgroupCPUs(filepath.Join(c.root, "cpuset"), c.path)
	if err != nil {
		return 0, err
	}

	return len(cgroupCPUs), nil
}

func (c *CgroupV2) CpuCount() (int, error) {
	cgroupCPUs, err := GetCgroupCPUs(c.root, c.path)
	if err != nil {
		return 0, err
	}

	return len(cgroupCPUs), nil
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

func cgroupsV1Hierarchy(root string) cgroup1.Hierarchy {
	return func() ([]cgroup1.Subsystem, error) {
		h, err := cgroup1.NewHugetlb(root)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		s := []cgroup1.Subsystem{
			cgroup1.NewNamed(root, "systemd"),
			cgroup1.NewFreezer(root),
			cgroup1.NewPids(root),
			cgroup1.NewNetCls(root),
			cgroup1.NewNetPrio(root),
			cgroup1.NewPerfEvent(root),
			cgroup1.NewCpuset(root),
			cgroup1.NewCpu(root),
			cgroup1.NewCpuacct(root),
			cgroup1.NewMemory(root),
			cgroup1.NewBlkio(root),
			cgroup1.NewRdma(root),
		}
		if err == nil {
			s = append(s, h)
		}

		return s, nil
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
