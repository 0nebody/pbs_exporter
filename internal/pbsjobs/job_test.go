package pbsjobs

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestJobId(t *testing.T) {
	job := &Job{}
	tests := []struct {
		hashname string
		want     string
	}{
		{"", ""},
		{"12345", "12345"},
		{"12345.pbs", "12345"},
		{"12345.p.b.s", "12345"},
		{"12345[0].pbs", "12345[0]"},
	}
	for _, test := range tests {
		job.Hashname = test.hashname
		got := job.JobId()
		if got != test.want {
			t.Errorf("JobId() = %v, want %v", got, test.want)
		}
	}
}

func TestJobUsername(t *testing.T) {
	job := &Job{}
	job.Euser = "username"
	got := job.JobUsername()
	want := "username"
	if got != want {
		t.Errorf("JobUsername() = %v, want %v", got, want)
	}
}

func TestJobUid(t *testing.T) {
	job := &Job{}

	t.Run("ValidUser", func(t *testing.T) {
		tests := []struct {
			euser string
			uids  []string
		}{
			{"root", []string{"0"}},
			{"bin", []string{"1", "2"}},
		}

		for _, test := range tests {
			job.Euser = test.euser
			got, err := job.JobUid()
			if err != nil {
				t.Fatalf("JobUid() = %v, want %v", err, nil)
			}
			if !slices.Contains(test.uids, got) {
				t.Errorf("JobUid() = %v, want %v", got, test.uids)
			}
		}
	})

	t.Run("InvalidUser", func(t *testing.T) {
		job.Euser = "notauser"
		got, err := job.JobUid()
		if err == nil {
			t.Fatalf("JobUid() expected error, got %v", got)
		}
		if got != "" {
			t.Errorf("JobUid() = %v, want empty string", got)
		}
	})
}

func TestNgpusResource(t *testing.T) {
	job := &Job{}
	tests := []struct {
		name        string
		ngpus       int
		schedSelect string
		want        int
	}{
		{"NoGpus", 0, "", 0},
		{"GpusFromResources", 2, "1:ncpus=4:mem=32gb:nfpgas=0", 2},
		{"GpusFromSelect", 0, "1:ncpus=4:ngpus=1:mem=32gb:nfpgas=0", 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job.ResourceList.Ngpus = test.ngpus
			job.SchedSelect = test.schedSelect
			got, err := job.Ngpus()
			if err != nil {
				t.Fatalf("Ngpus() returned error: %v", err)
			}
			if int(got) != test.want {
				t.Errorf("Ngpus() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRequestedWalltime(t *testing.T) {
	job := &Job{}
	tests := []struct {
		walltime string
		want     int64
	}{
		{"", 0},
		{"00:00:00", 0},
		{"00:00:01", 1},
		{"00:01:00", 60},
		{"01:00:00", 3600},
	}

	for _, test := range tests {
		t.Run(test.walltime, func(t *testing.T) {
			job.ResourceList.Walltime = test.walltime
			got := job.RequestedWalltime()
			if got != test.want {
				t.Errorf("RequestedWalltime() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNodeSelect(t *testing.T) {
	job := &Job{}
	tests := []struct {
		schedSelect string
		want        int
		wantErr     bool
	}{
		{"", 0, true},
		{"5", 5, false},
		{"a:ncpus=4:ngpus=1:mem=32gb:nfpgas=0", 0, true},
		{"1:ncpus=4:ngpus=1:mem=32gb:nfpgas=0", 1, false},
	}

	for _, test := range tests {
		t.Run(test.schedSelect, func(t *testing.T) {
			job.SchedSelect = test.schedSelect
			got, err := job.NodeSelect()
			if !test.wantErr && err != nil {
				t.Fatalf("NodeSelect() returned error: %v", err)
			}
			if got != test.want {
				t.Errorf("NodeSelect() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsInteractive(t *testing.T) {
	job := &Job{}
	tests := []struct {
		interactive int
		want        bool
	}{
		{0, false},
		{12345, true},
	}
	for _, test := range tests {
		job.Interactive = test.interactive
		got := job.IsInteractive()
		if got != test.want {
			t.Errorf("IsInteractive() = %v, want %v", got, test.want)
		}
	}
}

func TestIsPrimaryNode(t *testing.T) {
	job := &Job{}
	tests := []struct {
		execHost string
		want     bool
	}{
		{"", false},
		{"cpu1n001.local.domain:15002/3", true},
		{"cpu1n002.local.domain:15002/3+cpu1n001.local.domain:15002/8", false},
	}
	for _, test := range tests {
		job.ExecHost = test.execHost
		got := job.IsPrimaryNode("cpu1n001")
		if got != test.want {
			t.Errorf("IsPrimaryNode(cpu1n001) = %v, want %v", got, test.want)
		}
	}
}

func TestIsRunning(t *testing.T) {
	job := &Job{}
	tests := []struct {
		state string
		want  bool
	}{
		{"", false},
		{"5", false},
		{"E", false},
		{"R", true},
	}
	for _, test := range tests {
		job.JobState = test.state
		got := job.IsRunning()
		if got != test.want {
			t.Errorf("IsRunning() = %v, want %v", got, test.want)
		}
	}
}

func TestVnode(t *testing.T) {
	job := &Job{}
	tests := []struct {
		execVnode string
		want      string
	}{
		{"", ""},
		{"(gpu1n001:ncpus=4:ngpus=1:mem=1gb:nfpgas=0)", ""},
		{"(gpu1n001[a]:ncpus=4:ngpus=1:mem=1gb:nfpgas=0)", ""},
		{"(gpu1n001[1]:ncpus=4:ngpus=1:mem=1gb:nfpgas=0)", "1"},
		{"(gpu-1_01.local.domain[1]:ncpus=4:ngpus=1:mem=1gb:nfpgas=0)", "1"},
	}

	for _, test := range tests {
		job.ExecVnode = test.execVnode
		vnode := job.Vnode()
		if vnode != test.want {
			t.Errorf("Vnode() = %v, want %v", vnode, test.want)
		}
	}
}

func TestGetFieldName(t *testing.T) {
	type Test struct {
		a string
		b string `pbs:"t"`
		c string `pbs:""`
	}
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"Field with no tag", "a", "a"},
		{"Field with pbs tag", "b", "t"},
		{"Field with empty pbs tag", "c", "c"},
	}

	typeof := reflect.TypeOf(Test{})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, found := typeof.FieldByName(test.field)
			if !found {
				t.Fatalf("Field %s not found in type %s", test.field, typeof.Name())
			}
			got := getFieldName(field)
			if got != test.want {
				t.Errorf("getFieldName(%s) = %v, want %v", test.field, got, test.want)
			}
		})
	}
}

func TestCreateJobMap(t *testing.T) {
	type TestJob struct {
		JobName      string `pbs:"Job_Name"`
		ResourceList struct {
			Mem int `pbs:"mem"`
		} `pbs:"Resource_List"`
	}
	expectedPaths := map[string]bool{
		"Job_Name":             true,
		"Resource_List\x00mem": true,
	}

	job := TestJob{}
	rvJob := reflect.ValueOf(&job).Elem()
	jobMap := createJobMap(rvJob, []string{})

	for _, field := range jobMap {
		sentinel := strings.Join(field.path, "\x00")
		if _, ok := expectedPaths[sentinel]; !ok {
			t.Errorf("structBranch() produced unexpected field path: %s", sentinel)
		}
	}
}

func TestParseJobMap(t *testing.T) {
	type JobName struct {
		JobName      string `pbs:"Job_Name"`
		ResourceList struct {
			Ncpus int `pbs:"ncpus"`
		} `pbs:"Resource_List"`
	}
	tests := []struct {
		name       string
		binaryData []byte
		want       []JobMap
	}{
		{
			"Empty data",
			[]byte(""),
			[]JobMap{{data: "", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Success single key-value pair",
			[]byte("\x00Job_Name\x00job_queue_name\x00"),
			[]JobMap{{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Success with dictionary structure",
			[]byte("\x00Job_Name\x00job_queue_name\x00Resource_List\x00ncpus\x004\x00"),
			[]JobMap{
				{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"},
				{data: "4", path: []string{"Resource_List", "ncpus"}, sentinel: "Resource_List\x00ncpus"},
			},
		},
		{
			"Success with no \x00 prefix",
			[]byte("Job_Name\x00job_queue_name\x00"),
			[]JobMap{{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Success with key in value partial match",
			[]byte("Not_Job_Name\x00Job_Name\x00job_queue_name\x00"),
			[]JobMap{{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Success with key in value partial match empty value",
			[]byte("Job_Name\x00job_queue_name\x00Not_Job_Name\x00\x00"),
			[]JobMap{{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Success with full match",
			[]byte("\x00Job_Name\x00job_queue_name\x00Not_Job_Name\x00abc\x00"),
			[]JobMap{{data: "job_queue_name", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Fail with no value for key",
			[]byte("Job_Name"),
			[]JobMap{{data: "", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Fail with no \x00 suffix (key)",
			[]byte("\x00Job_Namejob_queue_name\x00"),
			[]JobMap{{data: "", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
		{
			"Fail with no \x00 suffix (value)",
			[]byte("\x00Job_Name\x00job_queue_name"),
			[]JobMap{{data: "", path: []string{"Job_Name"}, sentinel: "Job_Name"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobMap := createJobMap(reflect.ValueOf(&JobName{}).Elem(), []string{})
			parseJobMap(tt.binaryData, jobMap)
			for i := range tt.want {
				if jobMap[i].data != tt.want[i].data {
					t.Errorf("parseJobMap() data = %v, want %v", jobMap[i].data, tt.want[i].data)
				}
				if !slices.Equal(jobMap[i].path, tt.want[i].path) {
					t.Errorf("parseJobMap() path = %v, want %v", jobMap[i].path, tt.want[i].path)
				}
				if jobMap[i].sentinel != tt.want[i].sentinel {
					t.Errorf("parseJobMap() sentinel = %v, want %v", jobMap[i].sentinel, tt.want[i].sentinel)
				}
				if !jobMap[i].rvField.CanSet() {
					t.Errorf("parseJobMap() rvField is not settable: %v", jobMap[i].rvField)
				}
			}
		})
	}
}

func TestParseJobFile(t *testing.T) {
	t.Run("Parse string", func(t *testing.T) {
		job, errors := parseJobFile([]byte("\x00Job_Name\x00job_queue_name\x00"))
		if len(errors) > 0 {
			t.Fatalf("Failed to parse job file: %v", errors)
		}
		if job.JobName != "job_queue_name" {
			t.Errorf("parseJobFile() = %v, want %v", job.JobName, "job_queue_name")
		}
	})

	t.Run("Parse int", func(t *testing.T) {
		job, errors := parseJobFile([]byte("\x00interactive\x002147483647\x00"))
		if len(errors) > 0 {
			t.Fatalf("Failed to parse job file: %v", errors)
		}
		if job.Interactive != 2147483647 {
			t.Errorf("parseJobFile() = %v, want %v", job.Interactive, 2147483647)
		}
	})

	t.Run("Parse int64", func(t *testing.T) {
		job, errors := parseJobFile([]byte("\x00stime\x009223372036854775807\x00"))
		if len(errors) > 0 {
			t.Fatalf("Failed to parse job file: %v", errors)
		}
		if job.Stime != 9223372036854775807 {
			t.Errorf("parseJobFile() = %v, want %v", job.Stime, 9223372036854775807)
		}
	})

	t.Run("Parse human readable memory", func(t *testing.T) {
		want := int64(64 * 1024 * 1024 * 1024)
		job, errors := parseJobFile([]byte("\x00Resource_List\x00mem\x0064GiB\x00"))
		if len(errors) > 0 {
			t.Fatalf("Failed to parse job file: %v", errors)
		}
		if job.ResourceList.Mem != want {
			t.Errorf("parseJobFile() = %v, want %v", job.ResourceList.Mem, want)
		}
	})

	t.Run("Parse string array", func(t *testing.T) {
		job, errors := parseJobFile([]byte("\x00Variable_List\x00PBS_O_HOME=/home/user,PBS_O_INTERACTIVE_AUTH_METHOD=resvport,PBS_O_SYSTEM=Linux,PBS_O_QUEUE=cpu_batch\x00"))
		if len(errors) > 0 {
			t.Fatalf("Failed to parse job file: %v", errors)
		}
		if !slices.Equal(job.VariableList, []string{"PBS_O_HOME=/home/user", "PBS_O_INTERACTIVE_AUTH_METHOD=resvport", "PBS_O_SYSTEM=Linux", "PBS_O_QUEUE=cpu_batch"}) {
			t.Errorf("parseJobFile() = %v, want %v", job.VariableList, []string{"PBS_O_HOME=/home/user", "PBS_O_INTERACTIVE_AUTH_METHOD=resvport", "PBS_O_SYSTEM=Linux", "PBS_O_QUEUE=cpu_batch"})
		}
	})
}

func TestParseJobFiles(t *testing.T) {
	tmpdir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Test with invalid path", func(t *testing.T) {
		_, err := ParseJobFiles("", logger)
		if err == nil {
			t.Fatalf("ParseJobFiles('', <logger>) did not return error for empty directory")
		}
	})

	t.Run("Test valid job files", func(t *testing.T) {
		tests := []struct {
			name    string
			content string
			want    *Job
			wantErr bool
		}{
			// valid job file
			{
				name:    "1000",
				content: "\x00hashname\x001000.pbs\x00Job_Name\x00test_job\x00",
				want: &Job{
					JobName:  "test_job",
					Hashname: "1000.pbs",
				},
				wantErr: false,
			},
			// test for race condition
			{
				name:    "1001",
				content: "\x00hashname\x001001.pbs\x00Job_Name\x00test_job2\x00",
				want: &Job{
					JobName:  "test_job2",
					Hashname: "1001.pbs",
				},
				wantErr: false,
			},
		}

		want := make(map[string]*Job)
		for _, tt := range tests {
			os.WriteFile(filepath.Join(tmpdir, tt.name+".pbs.JB"), []byte(tt.content), 0644)
			if !tt.wantErr {
				want[tt.name] = tt.want
			}
		}

		got, err := ParseJobFiles(tmpdir, logger)
		if err != nil {
			t.Fatalf("ParseJobFiles(%s, <logger>) returned error: %v", tmpdir, err)
		}

		for _, tt := range tests {
			if !tt.wantErr {
				if job, ok := got[tt.name]; !ok {
					t.Errorf("ParseJobFiles(%s, <logger>) did not find job %s", tmpdir, tt.name)
				} else if !reflect.DeepEqual(job.VariableList, tt.want.VariableList) {
					t.Errorf("ParseJobFiles(%s, <logger>) = %v, want %v", tmpdir, job, tt.want)
				}
			}
		}
	})
}

func TestJobFile(t *testing.T) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, err := os.ReadDir(jobFileDir)
	if err != nil {
		t.Fatalf("Failed to read job files: %v", err)
	}

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}

		jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
		content, err := os.ReadFile(jobFilePath)
		if err != nil {
			t.Fatalf("Failed to read job file: %v", err)
		}
		t.Run(jobFile.Name(), func(t *testing.T) {
			job, errors := parseJobFile(content)
			if len(errors) > 0 {
				t.Fatalf("Failed to parse job file: %v", errors)
			}
			if job.Hashname == "" {
				t.Errorf("Parsed job has empty hashname from file %s", jobFile.Name())
			}
		})
	}
}

func BenchmarkParseJobFile(b *testing.B) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, _ := os.ReadDir(jobFileDir)

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}

		jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
		content, err := os.ReadFile(jobFilePath)
		if err != nil {
			b.Fatalf("Failed to read job file: %v", err)
		}
		b.Run(jobFilePath, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, errors := parseJobFile(content)
				if len(errors) > 0 {
					b.Fatalf("Failed to parse job file: %v", errors)
				}
			}
		})
	}
}

func TestGetJobFiles(t *testing.T) {
	want := []string{"10001.pbs.JB"}
	fs := fstest.MapFS{
		"10000.pbs":    {Data: []byte("")},
		"10001.pbs.JB": {Data: []byte("")},
		"10002.pbs.TK": {Data: []byte("")},
		"10003.pbs.SC": {Data: []byte("")},
	}
	got, err := getJobFiles(fs)

	if err != nil {
		t.Fatalf("getJobFiles() returned error: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("getJobFiles() = %v, want %v", got, want)
	}
}

func TestNewJobWatcher(t *testing.T) {
	jobWatcher, err := NewJobWatcher("")
	if err != nil {
		t.Fatalf("NewJobWatcher('') returned error: %v", err)
	}
	if jobWatcher == nil {
		t.Fatal("NewJobWatcher('') returned nil")
	}
}

func TestPbsJobEvent(t *testing.T) {
	tmp := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	jobCache := NewJobCache(logger, 60, 15*time.Second)
	now := time.Now().Unix()

	watcher, err := NewJobWatcher(tmp)
	if err != nil {
		t.Fatalf("NewJobWatcher() returned error: %v", err)
	}
	go PbsJobEvent(watcher, logger, jobCache)
	defer watcher.Close()

	jobFile := filepath.Join(tmp, "1000.pbs.JB")
	if err := os.WriteFile(jobFile, []byte(fmt.Sprintf("\x00Job_Name\x00test_job\x00hashname\x001000.pbs\x00stime\x00%d\x00", now)), 0644); err != nil {
		t.Fatalf("Failed to create job file: %v", err)
	}
	// Wait for the watcher to pick up the update
	time.Sleep(100 * time.Millisecond)

	t.Run("Test job file creation", func(t *testing.T) {
		jobFile := filepath.Join(tmp, "1001.pbs.JB")
		if err := os.WriteFile(jobFile, []byte(fmt.Sprintf("\x00Job_Name\x00test_job\x00hashname\x001001.pbs\x00stime\x00%d\x00", now)), 0644); err != nil {
			t.Fatalf("Failed to create job file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		job, exists := jobCache.Get("1001")
		if !exists {
			t.Errorf("Expected job 1001 to exist in cache after file creation")
		}
		if job.JobName != "test_job" {
			t.Errorf("Expected job name 'test_job', got %v", job.JobName)
		}
	})

	t.Run("Test job file update", func(t *testing.T) {
		jobFile := filepath.Join(tmp, "1000.pbs.JB")
		if err := os.WriteFile(jobFile, []byte(fmt.Sprintf("\x00Job_Name\x00updated_job\x00hashname\x001000.pbs\x00stime\x00%d\x00", now)), 0644); err != nil {
			t.Fatalf("Failed to update job file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		job, exists := jobCache.Get("1000")
		if !exists {
			t.Errorf("Expected job 1000 to exist in cache after file update")
		}
		if job.JobName != "updated_job" {
			t.Errorf("Expected updated job name 'updated_job', got %v", job.JobName)
		}
	})

	t.Run("Test job file deletion", func(t *testing.T) {
		jobFile := filepath.Join(tmp, "1000.pbs.JB")
		if err := os.Remove(jobFile); err != nil {
			t.Fatalf("Failed to delete job file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		// job will exist in cache until expiration
		isRunning := jobCache.IsRunning("1000")
		if isRunning {
			t.Errorf("Expected job 1000 to not be running after deletion, but it is still running")
		}
	})
}
