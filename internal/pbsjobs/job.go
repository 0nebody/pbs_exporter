package pbsjobs

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
	pbsVnodeRegexp = regexp.MustCompile(`[a-zA-Z0-9_.-]+\[(\d+)\]`)
)

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
	// ngpus is introduced in PBS 2025
	if j.ResourceList.Ngpus != 0 {
		return j.ResourceList.Ngpus, nil
	}

	for selectItem := range strings.SplitSeq(j.SchedSelect, ":") {
		if strings.HasPrefix(selectItem, "ngpus=") {
			ngpus := strings.Split(selectItem, "=")[1]
			ngpusInt, err := strconv.Atoi(ngpus)
			if err != nil {
				return 0, err
			}
			return ngpusInt, nil
		}
	}

	return 0, nil
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
	if len(vnodeMatch) > 1 {
		return vnodeMatch[1]
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
