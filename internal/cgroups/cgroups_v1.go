package cgroups

import (
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/containerd/cgroups/v3/cgroup1"
	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
)

var (
	nanosecPerSecond = uint64(1000000000)
)

type CgroupV1Api interface {
	Processes(cgroup1.Name, bool) ([]cgroup1.Process, error)
	Stat(...cgroup1.ErrorHandler) (*v1.Metrics, error)
	Subsystems() []cgroup1.Subsystem
	Tasks(cgroup1.Name, bool) ([]cgroup1.Task, error)
}

type CgroupV1Manager struct {
	version string
	root    string
}

type CgroupV1 struct {
	root       string
	path       string
	subsystems []string
	cgroup     CgroupV1Api
}

func (m *CgroupV1Manager) List(path string) ([]string, error) {
	return listCgroups(filepath.Join(m.root, "/cpu,cpuacct"), path)
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

func (c *CgroupV1Manager) Version() string {
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
		ioMap := make(map[uint64]*IoUsage)
		ioMapGet := func(major uint64) *IoUsage {
			if _, ok := ioMap[major]; !ok {
				ioMap[major] = &IoUsage{Major: major}
			}
			return ioMap[major]
		}

		for _, ioUsage := range statIO.GetIoServiceBytesRecursive() {
			ioStat := ioMapGet(ioUsage.GetMajor())
			switch operation := ioUsage.GetOp(); operation {
			case "Read":
				ioStat.Rbytes += ioUsage.GetValue()
			case "Write":
				ioStat.Wbytes += ioUsage.GetValue()
			}
		}

		for _, ioUsage := range statIO.GetIoServicedRecursive() {
			ioStat := ioMapGet(ioUsage.GetMajor())
			switch operation := ioUsage.GetOp(); operation {
			case "Read":
				ioStat.Rios += ioUsage.GetValue()
			case "Write":
				ioStat.Wios += ioUsage.GetValue()
			}
		}

		for _, ioStat := range ioMap {
			metrics.Io.Usage = append(metrics.Io.Usage, *ioStat)
		}
	}

	if slices.Contains(metrics.Controllers, "cpu") {
		statCPU := stat.GetCPU()
		statCPUUsage := statCPU.GetUsage()
		metrics.Cpu.System = statCPUUsage.GetKernel() / nanosecPerSecond
		metrics.Cpu.Usage = statCPUUsage.GetTotal() / nanosecPerSecond
		metrics.Cpu.User = statCPUUsage.GetUser() / nanosecPerSecond
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
				Max:      hugetlb.GetMax(),
				Pagesize: hugetlb.GetPagesize(),
				Usage:    hugetlb.GetUsage(),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "memory") {
		statMemory := stat.GetMemory()
		statMemoryUsage := statMemory.GetUsage()
		statMemorySwap := statMemory.GetSwap()

		metrics.Memory.ActiveAnon = statMemory.GetTotalActiveAnon()
		metrics.Memory.ActiveFile = statMemory.GetTotalActiveFile()
		metrics.Memory.FileMapped = statMemory.GetTotalRSS() - statMemory.GetTotalActiveAnon() - statMemory.GetTotalInactiveAnon()
		metrics.Memory.InactiveAnon = statMemory.GetTotalInactiveAnon()
		metrics.Memory.InactiveFile = statMemory.GetTotalInactiveFile()
		metrics.Memory.Limit = statMemoryUsage.GetLimit()
		metrics.Memory.Rss = statMemory.GetTotalRSS()
		// unavailable in cgroups v1
		metrics.Memory.Shmem = uint64(0)
		metrics.Memory.Usage = statMemoryUsage.GetUsage()
		metrics.Memory.Wss = statMemoryUsage.GetUsage() - statMemory.GetTotalInactiveFile()

		if statMemorySwap.GetUsage() >= statMemoryUsage.GetUsage() {
			metrics.Memory.SwapUsage = statMemorySwap.GetUsage() - statMemoryUsage.GetUsage()
		}
		if statMemorySwap.GetLimit() >= statMemoryUsage.GetLimit() {
			metrics.Memory.SwapLimit = statMemorySwap.GetLimit() - statMemoryUsage.GetLimit()
		}

		metrics.Memory.Pgfault = statMemory.PgFault
		metrics.Memory.Pgmajfault = statMemory.PgMajFault

		if metrics.Memory.Limit == math.MaxUint64 {
			return nil, ErrCgroupUninitialised
		}
	}

	if slices.Contains(metrics.Controllers, "pids") {
		pids := stat.GetPids()
		metrics.Tasks.PidLimit = pids.GetLimit()
		metrics.Tasks.PidUsage = pids.GetCurrent()
	} else {
		metrics.Tasks.PidLimit = uint64(0)
	}

	pids, err := c.Procs()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Pids = pids
	metrics.Tasks.PidUsage = uint64(len(pids))

	threads, err := c.Threads()
	if err != nil {
		return nil, err
	}
	metrics.Tasks.Threads = threads
	metrics.Tasks.ThreadUsage = uint64(len(threads))

	return metrics, nil
}

func (c *CgroupV1) Procs() ([]uint64, error) {
	processes, err := c.cgroup.Processes(cgroup1.Cpuacct, true)
	if err != nil {
		return nil, err
	}

	if len(processes) == 0 {
		return nil, nil
	}

	processIds := make([]uint64, len(processes))
	for i, pid := range processes {
		processIds[i] = uint64(pid.Pid)
	}

	return processIds, nil
}

func (c *CgroupV1) Threads() ([]uint64, error) {
	threads, err := c.cgroup.Tasks(cgroup1.Cpuacct, true)
	if err != nil {
		return nil, err
	}

	if len(threads) == 0 {
		return nil, nil
	}

	threadIds := make([]uint64, len(threads))
	for i, pid := range threads {
		threadIds[i] = uint64(pid.Pid)
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
