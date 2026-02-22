package pbsjob

import (
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/fsnotify/fsnotify"
)

var (
	pbsVnodeRegexp = regexp.MustCompile(`([a-zA-Z0-9_.-]+)\[(\d+)\]`)
)

type Vnode struct {
	Node  string
	Vnode string
}

type ExecVnode struct {
	Mem    int64 `pbs:"mem"`
	Ncpus  int64 `pbs:"ncpus"`
	Nfpgas int64 `pbs:"nfpgas"`
	Ngpus  int64 `pbs:"ngpus"`
}

// ResourcesUsed.Cpus can be json '{"host.domain": "1"}' or comma separated list '1,2,3,4'
// TODO: set ResourcesUsed.Cpus to int once PBS fixes issues with returning json.
type ResourcesUsed struct {
	Cpupercent int      `pbs:"cpupercent"`
	Cpus       []string `pbs:"cpus" sep:","`
	Cput       string   `pbs:"cput"`
	Mem        int64    `pbs:"mem"`
	Ncpus      int      `pbs:"ncpus"`
	Ngpus      int      `pbs:"ngpus"`
	Vmem       int64    `pbs:"vmem"`
	Walltime   string   `pbs:"walltime"`
}
type ResourceList struct {
	Mem      int64  `pbs:"mem"`
	Ncpus    int    `pbs:"ncpus"`
	Nfpgas   int    `pbs:"nfpgas"`
	Ngpus    int    `pbs:"ngpus"`
	Place    string `pbs:"place"`
	Walltime string `pbs:"walltime"`
}

type Job struct {
	JobName       string        `pbs:"Job_Name"`
	JobOwner      string        `pbs:"Job_Owner"`
	ResourcesUsed ResourcesUsed `pbs:"resources_used"`
	JobState      string        `pbs:"job_state"`
	Queue         string        `pbs:"queue"`
	Server        string        `pbs:"server"`
	Devices       string        `pbs:"devices"`
	AccountName   string        `pbs:"Account_Name"`
	Checkpoint    string        `pbs:"Checkpoint"`
	ErrorPath     string        `pbs:"Error_Path"`
	ExecHost      string        `pbs:"exec_host2" sep:"+"`
	ExecVnode     string        `pbs:"exec_vnode" sep:"+"`
	Interactive   int           `pbs:"interactive"`
	JoinPath      string        `pbs:"Join_Path"`
	KeepFiles     string        `pbs:"Keep_Files"`
	Mtime         int64         `pbs:"mtime"`
	OutputPath    string        `pbs:"Output_Path"`
	ResourceList  ResourceList  `pbs:"Resource_List"`
	SchedSelect   string        `pbs:"schedselect" sep:":"`
	Stime         int64         `pbs:"stime"`
	SessionID     string        `pbs:"session_id"`
	ShellPathList string        `pbs:"Shell_Path_List"`
	JobDir        string        `pbs:"jobdir"`
	Substate      string        `pbs:"substate"`
	VariableList  []string      `pbs:"Variable_List" sep:","`
	Euser         string        `pbs:"euser"`
	Egroup        string        `pbs:"egroup"`
	Hashname      string        `pbs:"hashname"`
	Cookie        string        `pbs:"cookie"`
	Umask         string        `pbs:"umask"`
	RunCount      int           `pbs:"run_count"`
	JobKillDelay  string        `pbs:"job_kill_delay"`
	ArrayId       string        `pbs:"array_id"`
	ArrayIndex    string        `pbs:"array_index"`
	Executable    string        `pbs:"executable"`
	ArgumentList  string        `pbs:"argument_list"`
	Project       string        `pbs:"project"`
	RunVersion    string        `pbs:"run_version"`
	SubmitHost    string        `pbs:"Submit_Host"`
	Binding       string        `pbs:"binding" sep:":"`
}

func (j *Job) JobId() string {
	return strings.Split(j.Hashname, ".")[0]
}

func (j *Job) JobUsername() string {
	return j.Euser
}

func (j *Job) JobUid() (string, error) {
	username := j.JobUsername()
	if username == "" {
		return "", fmt.Errorf("username is empty")
	}

	user, err := user.Lookup(username)
	if err != nil {
		return "", fmt.Errorf("lookup username %s: %v", username, err)
	}

	return user.Uid, nil
}

func (j *Job) Ngpus() (int, error) {
	// ngpus was introduced in PBS 2025
	if j.ResourceList.Ngpus != 0 {
		return j.ResourceList.Ngpus, nil
	}

	ngpus, err := j.ParseSelect("ngpus")
	if err != nil {
		return 0, err
	}

	return int(ngpus), nil
}

// Each Vnode in an ExecVnode is separated by "+" and wrapped in brackets.
// Each resource in the vnode is seperated by ":".
// single vnode:    (nodename[vnode_index]:ncpus=4:ngpus=2:mem=92gb:nfgpas=0)
// multiple vnodes: (nodename[vnode_index]:ncpus=4:mem=92gb)+(nodename2[vnode_index]:ncpus=4:mem=92gb)
func (j *Job) ParseExecVnode() (map[Vnode]ExecVnode, error) {
	execVnodes := make(map[Vnode]ExecVnode)

	for execVnodeEntry := range strings.SplitSeq(j.ExecVnode, "+") {
		vnodeResources := make(map[string]int64)

		execVnodeEntry, _ = strings.CutPrefix(execVnodeEntry, "(")
		execVnodeEntry, _ = strings.CutSuffix(execVnodeEntry, ")")
		hostVnode, resources, _ := strings.Cut(execVnodeEntry, ":")

		vnodeMatch := pbsVnodeRegexp.FindStringSubmatch(hostVnode)
		if len(vnodeMatch) < 3 {
			return nil, fmt.Errorf("unable to parse vnode from execVnode")
		}

		for resValue := range strings.SplitSeq(resources, ":") {
			resource, value, _ := strings.Cut(resValue, "=")
			if intValue, err := utils.ParseBytes(value); err == nil {
				vnodeResources[resource] = intValue
			}
		}

		vnode := Vnode{
			Node:  vnodeMatch[1],
			Vnode: vnodeMatch[2],
		}
		currentVnode := execVnodes[vnode]
		currentVnode.Mem += vnodeResources["mem"]
		currentVnode.Ncpus += vnodeResources["ncpus"]
		currentVnode.Nfpgas += vnodeResources["nfpgas"]
		currentVnode.Ngpus += vnodeResources["ngpus"]
		execVnodes[vnode] = currentVnode
	}

	return execVnodes, nil
}

func (j *Job) ParseSelect(resource string) (int64, error) {
	var total int64 = 0
	var nchunks int64 = 0

	for chunk := range strings.SplitSeq(j.SchedSelect, "+") {
		selectNodes, selectResources, found := strings.Cut(chunk, ":")
		if found {
			if nodeCount, err := strconv.ParseInt(selectNodes, 10, 64); err == nil {
				nchunks = nodeCount
			}
		}

		for selectResource := range strings.SplitSeq(selectResources, ":") {
			resourceKey, resourceValue, found := strings.Cut(selectResource, "=")
			if found && resourceKey == resource {
				if value, err := utils.ParseBytes(resourceValue); err == nil {
					total += value * nchunks
				} else {
					return 0, fmt.Errorf("parsing select %s for resource %s: %w", j.SchedSelect, resource, err)
				}
			}
		}
	}

	return total, nil
}

func (j *Job) RequestedWalltime() int64 {
	walltime := j.ResourceList.Walltime
	walltimeSeconds := utils.ParseWalltime(walltime)

	return walltimeSeconds
}

func (j *Job) NodeSelect() (int, error) {
	selectStatement := j.SchedSelect
	selectStatementParts := strings.Split(selectStatement, ":")
	nodeCount := selectStatementParts[0]

	numNodes, err := strconv.Atoi(nodeCount)
	if err != nil {
		return 0, err
	}

	return numNodes, nil
}

func (j *Job) IsInteractive() bool {
	return j.Interactive > 0
}

// The PBS Professional User's Guide indicates that "PBS assigns
// chunks to job processes in the order in which the chunks appear
// in the select statement. PBS takes the first chunk from the
// primary execution host; this is where the top task of the job
// runs." and "The job's primary execution host is the host that
// supplies the vnode to satisfy the first chunk requested by the job."
func (j *Job) IsPrimaryNode(hostname string) bool {
	primaryNode := strings.Split(j.ExecHost, "+")[0]
	isPrimaryNode := strings.HasPrefix(primaryNode, hostname)

	return isPrimaryNode
}

func (j *Job) IsRunning() bool {
	return j.JobState == "R"
}

func (j *Job) Vnode() string {
	primaryNode := strings.Split(j.ExecVnode, "+")[0]

	vnodeMatch := pbsVnodeRegexp.FindStringSubmatch(primaryNode)
	if len(vnodeMatch) > 2 {
		return vnodeMatch[2]
	}

	return ""
}

func getJobFiles(fileSystem fs.FS) ([]string, error) {
	files, err := fs.ReadDir(fileSystem, ".")
	if err != nil {
		return nil, err
	}

	var jobFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".JB") {
			jobFiles = append(jobFiles, file.Name())
		}
	}

	return jobFiles, nil
}

func ParseJobFiles(pbsJobPath string, logger *slog.Logger) (map[string]*Job, error) {
	jobFiles, err := getJobFiles(os.DirFS(pbsJobPath))
	if err != nil {
		logger.Error("Error reading job files", "error", err)
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	jobs := make(map[string]*Job)

	// Parse job files concurrently, logs and skips any that fail to parse
	for _, jobFile := range jobFiles {
		wg.Add(1)
		go func(jobFile string) {
			defer wg.Done()
			jobFilePath := filepath.Join(pbsJobPath, jobFile)
			content, err := os.ReadFile(jobFilePath)
			if err != nil {
				logger.Error("Error reading job file", "file", jobFilePath, "error", err)
				return
			}

			job := &Job{}
			if err := Unmarshal(content, job); err != nil {
				logger.Error("Error parsing job file", "file", jobFilePath, "error", err)
				return
			}

			mu.Lock()
			jobs[job.JobId()] = job
			mu.Unlock()
		}(jobFile)
	}
	wg.Wait()

	return jobs, nil
}

func NewJobWatcher(path string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	return watcher, nil
}

func PbsJobEvent(watcher *fsnotify.Watcher, logger *slog.Logger, pbsJobs *JobCache) error {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Only process events for job files (*.JB)
			if !strings.HasSuffix(event.Name, ".JB") {
				continue
			}

			switch op := event.Op; op {
			case fsnotify.Op(fsnotify.Create), fsnotify.Op(fsnotify.Write):
				// anonymous function to scope deferral of f.Close()
				func() {
					logger.Debug("PBS job file modified", "name", event.Name, "op", op)

					jobFile, err := os.Open(event.Name)
					if err != nil {
						logger.Error("Error opening file", "file", event.Name, "error", err)
						return
					}
					defer jobFile.Close()

					job := &Job{}
					dec := NewDecoder(jobFile)
					if err := dec.Decode(job); err != nil {
						logger.Error("Error parsing job file", "file", event.Name, "error", err)
						return
					}

					pbsJobs.Set(job.JobId(), job)
				}()
			case fsnotify.Op(fsnotify.Remove):
				logger.Debug("PBS Job file removed", "name", event.Name, "op", op)

				fileName := filepath.Base(event.Name)
				jobId := strings.Split(fileName, ".")[0]
				pbsJobs.Delete(jobId)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed with error: %v", err)
			}
		}
	}
}
