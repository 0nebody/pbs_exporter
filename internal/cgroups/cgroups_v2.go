package cgroups

import (
	"slices"

	"github.com/containerd/cgroups/v3/cgroup2"
	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
)

var (
	microsecPerSecond = uint64(1000000)
)

type CgroupV2Api interface {
	Procs(bool) ([]uint64, error)
	Controllers() ([]string, error)
	Stat() (*v2.Metrics, error)
	Threads(bool) ([]uint64, error)
}

type CgroupV2Manager struct {
	version string
	root    string
}

type CgroupV2 struct {
	root        string
	path        string
	controllers []string
	cgroup      CgroupV2Api
}

func (m *CgroupV2Manager) List(path string) ([]string, error) {
	return listCgroups(m.root, path)
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

func (c *CgroupV2Manager) Version() string {
	return c.version
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
		metrics.Cpu.System = statCPU.GetSystemUsec() / microsecPerSecond
		metrics.Cpu.Usage = statCPU.GetUsageUsec() / microsecPerSecond
		metrics.Cpu.User = statCPU.GetUserUsec() / microsecPerSecond
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
				Usage:    hugetlb.GetCurrent(),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "io") {
		statIO := stat.GetIo()
		statIoUsage := statIO.GetUsage()
		for _, ioUsage := range statIoUsage {
			metrics.Io.Usage = append(metrics.Io.Usage, IoUsage{
				Major:  ioUsage.GetMajor(),
				Rbytes: ioUsage.GetRbytes(),
				Rios:   ioUsage.GetRios(),
				Wbytes: ioUsage.GetWbytes(),
				Wios:   ioUsage.GetWios(),
			})
		}
	}

	if slices.Contains(metrics.Controllers, "memory") {
		statMemory := stat.GetMemory()

		metrics.Memory.ActiveAnon = statMemory.GetActiveAnon()
		metrics.Memory.ActiveFile = statMemory.GetActiveFile()
		metrics.Memory.FileMapped = statMemory.GetFileMapped()
		metrics.Memory.InactiveAnon = statMemory.GetInactiveAnon()
		metrics.Memory.InactiveFile = statMemory.GetInactiveFile()
		metrics.Memory.Limit = statMemory.GetUsageLimit()
		metrics.Memory.Rss = statMemory.GetAnon() + statMemory.GetFileMapped()
		metrics.Memory.Shmem = statMemory.GetShmem()
		metrics.Memory.Usage = statMemory.GetUsage()
		metrics.Memory.Wss = statMemory.GetUsage() - statMemory.GetInactiveFile()

		metrics.Memory.SwapUsage = statMemory.GetSwapUsage()
		metrics.Memory.SwapLimit = statMemory.GetSwapLimit()

		metrics.Memory.Pgfault = statMemory.GetPgfault()
		metrics.Memory.Pgmajfault = statMemory.GetPgmajfault()
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

	return &metrics, nil
}

func (c *CgroupV2) Procs() ([]uint64, error) {
	processIds, err := c.cgroup.Procs(true)
	if err != nil {
		return nil, err
	}

	return processIds, nil
}

func (c *CgroupV2) Threads() ([]uint64, error) {
	threadIds, err := c.cgroup.Threads(true)
	if err != nil {
		return nil, err
	}

	return threadIds, nil
}

func (c *CgroupV2) CpuCount() (int, error) {
	cgroupCPUs, err := GetCgroupCPUs(c.root, c.path)
	if err != nil {
		return 0, err
	}

	return len(cgroupCPUs), nil
}
