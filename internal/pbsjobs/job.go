package pbsjobs

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/docker/go-units"
	"github.com/fsnotify/fsnotify"
)

var (
	pbsVnodeRegexp = regexp.MustCompile(`[a-zA-Z0-9_.-]+\[(\d)\]`)
)

type JobMap struct {
	data     string
	path     []string
	rvField  reflect.Value
	sentinel string
}

// ResourcesUsed.Cpus can be json '{"host.domain": "1"}' or comma separated list '1,2,3,4'
// TODO: set ResourcesUsed.Cpus to int once PBS fixes issues with returning json.
type ResourcesUsed struct {
	Cpupercent int      `pbs:"cpupercent"`
	Cpus       []string `pbs:"cpus"`
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
	Checkpoint    string        `pbs:"Checkpoint"`
	ErrorPath     string        `pbs:"Error_Path"`
	ExecHost      string        `pbs:"exec_host2"`
	ExecVnode     string        `pbs:"exec_vnode"`
	Interactive   int           `pbs:"interactive"`
	JoinPath      string        `pbs:"Join_Path"`
	KeepFiles     string        `pbs:"Keep_Files"`
	Mtime         int64         `pbs:"mtime"`
	OutputPath    string        `pbs:"Output_Path"`
	ResourceList  ResourceList  `pbs:"Resource_List"`
	SchedSelect   string        `pbs:"schedselect"`
	Stime         int64         `pbs:"stime"`
	SessionID     string        `pbs:"session_id"`
	ShellPathList string        `pbs:"Shell_Path_List"`
	JobDir        string        `pbs:"jobdir"`
	Substate      string        `pbs:"substate"`
	VariableList  []string      `pbs:"Variable_List"`
	Euser         string        `pbs:"euser"`
	Egroup        string        `pbs:"egroup"`
	Hashname      string        `pbs:"hashname"`
	Cookie        string        `pbs:"cookie"`
	Umask         string        `pbs:"umask"`
	RunCount      int           `pbs:"run_count"`
	JobKillDelay  string        `pbs:"job_kill_delay"`
	Executable    string        `pbs:"executable"`
	ArgumentList  string        `pbs:"argument_list"`
	Project       string        `pbs:"project"`
	RunVersion    string        `pbs:"run_version"`
	SubmitHost    string        `pbs:"Submit_Host"`
	Binding       string        `pbs:"binding"`
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

	for _, selectItem := range strings.Split(j.SchedSelect, ":") {
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

func getFieldName(field reflect.StructField) string {
	name := field.Name
	tag := field.Tag.Get("pbs")

	if tag != "" {
		return tag
	} else {
		return name
	}
}

func createJobMap(rvJob reflect.Value, path []string) []JobMap {
	jm := []JobMap{}
	rtJob := rvJob.Type()

	for i := 0; i < rvJob.NumField(); i++ {
		field := rvJob.Field(i)
		name := getFieldName(rtJob.Field(i))

		if field.Type().Kind() == reflect.Struct {
			jm = append(jm, createJobMap(field, append(path, name))...)
		} else {
			full_path := append(path, name)
			jm = append(jm, JobMap{
				path:     full_path,
				sentinel: strings.Join(full_path, "\x00"),
				rvField:  field,
			})
		}
	}

	return jm
}

// Assumptions:
//   - Data is a sequence of key-value pairs, where each key and value is suffixed with \x00.
//   - A complete match is found when the sentinel and value are prefixed with \x00.
//   - Keys are unique, if a complete match is found, search stops for that key.
//   - Partial matches will continue to search for a full match.
func parseJobMap(data []byte, jobMap []JobMap) {
	for i, search := range jobMap {
		searchData := data
		term := []byte(search.sentinel)
		for x, d := bytes.Index(searchData, term), 0; x > -1; x, d = bytes.Index(searchData, term), d+x+1 {
			// exit if the sentinel terminates the data, it will not have a value
			if x+len(term) >= len(searchData) {
				searchData = searchData[x+len(term):]
				continue
			}

			hasNullPrefix := searchData[0] == 0x00

			// key must end in null byte, key exists as string in value otherwise
			if searchData[x+len(term)] != 0x00 {
				searchData = searchData[x+len(term):]
				continue
			}

			// value starts after the key + null byte
			// maintain byte in searchData for next iteration
			searchData = searchData[x+len(term):]

			// find end of value, which is the next null byte
			if valueEnd := bytes.IndexByte(searchData[1:], 0x00); valueEnd > -1 {
				// don't overwrite with empty value
				if valueEnd > 0 {
					jobMap[i].data = string(searchData[1 : valueEnd+1])
				}

				// empty search data if full match found
				if hasNullPrefix {
					searchData = searchData[:0]
				}
			}
		}
	}
}

func parseJobFile(content []byte) (*Job, []error) {
	job := &Job{}
	errors := []error{}
	rvJob := reflect.ValueOf(job).Elem()
	jobMap := createJobMap(rvJob, []string{})

	parseJobMap(content, jobMap)

	for _, field := range jobMap {
		switch field.rvField.Kind() {
		case reflect.Int, reflect.Int64:
			if field.data == "" {
				field.rvField.SetInt(0)
			} else {
				if intValue, err := strconv.ParseInt(field.data, 10, 64); err == nil {
					field.rvField.SetInt(intValue)
				} else if intValue, err := units.RAMInBytes(field.data); err == nil {
					field.rvField.SetInt(intValue)
				} else {
					errors = append(errors, fmt.Errorf("error parsing job file int: %v %v", field.data, err))
				}
			}
		case reflect.String:
			field.rvField.SetString(field.data)
		case reflect.Slice:
			if field.data != "" {
				// TODO: make the separator configurable
				separator := ","
				values := strings.Split(field.data, separator)

				switch field.rvField.Type().Elem().Kind() {
				case reflect.String:
					field.rvField.Set(reflect.ValueOf(values))
				case reflect.Int:
					intValues := make([]int, len(values))
					for i, v := range values {
						if intValue, err := strconv.Atoi(v); err == nil {
							intValues[i] = intValue
						} else {
							errors = append(errors, fmt.Errorf("error parsing job file int slice: %v %v", v, err))
						}
					}
					field.rvField.Set(reflect.ValueOf(intValues))
				}
			}
		}
	}

	return job, errors
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

			job, errors := parseJobFile(content)
			if len(errors) > 0 {
				for _, err := range errors {
					logger.Error("Error parsing job file", "file", jobFilePath, "error", err)
				}
				// return
			}

			mu.Lock()
			jobs[job.JobId()] = job
			mu.Unlock()
		}(jobFile)
	}
	wg.Wait()

	return jobs, nil
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
				logger.Debug("PBS job file modified", "name", event.Name, "op", op)

				content, err := os.ReadFile(event.Name)
				if err != nil {
					logger.Error("Error reading file", "file", event.Name, "error", err)
					continue
				}

				job, errors := parseJobFile(content)
				if len(errors) > 0 {
					for _, err := range errors {
						logger.Error("Error parsing job file", "file", event.Name, "error", err)
					}
					continue
				}

				pbsJobs.Set(job.JobId(), job)

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
