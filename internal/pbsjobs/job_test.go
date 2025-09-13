package pbsjobs

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
		{"(gpu1n001[111]:ncpus=4:ngpus=1:mem=1gb:nfpgas=0)", "111"},
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

func TestParseJobFiles(t *testing.T) {
	tmpdir := t.TempDir()
	logger := slog.Default()

	t.Run("Test with invalid path", func(t *testing.T) {
		_, err := ParseJobFiles("", logger)
		if err == nil {
			t.Fatalf("ParseJobFiles('', <logger>) did not return error for empty directory")
		}
	})

	job1, jb1, _ := generateJobFile("job1000", "1000.pbs", 0)
	job2, jb2, _ := generateJobFile("job1001", "1001.pbs", 0)
	t.Run("Test valid job files", func(t *testing.T) {
		tests := []struct {
			name    string
			content []byte
			want    *Job
			wantErr bool
		}{
			// valid job file
			{
				name:    "1000",
				content: jb1,
				want:    &job1,
				wantErr: false,
			},
			// test for race condition
			{
				name:    "1001",
				content: jb2,
				want:    &job2,
				wantErr: false,
			},
		}

		want := make(map[string]*Job)
		for _, test := range tests {
			os.WriteFile(filepath.Join(tmpdir, test.name+".pbs.JB"), test.content, 0644)
			if !test.wantErr {
				want[test.name] = test.want
			}
		}

		got, err := ParseJobFiles(tmpdir, logger)
		if err != nil {
			t.Fatalf("ParseJobFiles(%s, <logger>) returned error: %v", tmpdir, err)
		}

		for _, test := range tests {
			if !test.wantErr {
				if job, ok := got[test.name]; !ok {
					t.Errorf("ParseJobFiles(%s, <logger>) did not find job %s", tmpdir, test.name)
				} else if !reflect.DeepEqual(job.VariableList, test.want.VariableList) {
					t.Errorf("ParseJobFiles(%s, <logger>) = %v, want %v", tmpdir, job, test.want)
				}
			}
		}
	})
}

func TestPbsJobEvent(t *testing.T) {
	tmp := t.TempDir()
	logger := slog.Default()
	now := time.Now().Unix()
	jobCache := NewJobCache(logger, 60, 15*time.Second)
	jobFile := filepath.Join(tmp, "1000.pbs.JB")

	watcher, err := NewJobWatcher(tmp)
	if err != nil {
		t.Fatalf("NewJobWatcher() returned error: %v", err)
	}
	go PbsJobEvent(watcher, logger, jobCache)
	defer watcher.Close()

	t.Run("Create", func(t *testing.T) {
		j, jb, err := generateJobFile("job1000", "1000.pbs", now)
		if err != nil {
			t.Fatalf("Failed to generate mock job: %v", err)
		}

		if err := os.WriteFile(jobFile, jb, 0644); err != nil {
			t.Fatalf("Failed to create job file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		job, exists := jobCache.Get("1000")
		if !exists {
			t.Errorf("Expected job 1000 to exist in cache after file creation")
		}
		if job.JobName != "job1000" || job.Hashname != "1000.pbs" {
			t.Errorf("Expected JobName='%v' Hashname='%v', got %v, %v", j.JobName, j.Hashname, job.JobName, job.Hashname)
		}
	})

	t.Run("Update", func(t *testing.T) {
		j, jb, err := generateJobFile("updated_job", "1000.pbs", now)
		if err != nil {
			t.Fatalf("Failed to generate mock job: %v", err)
		}
		if err := os.WriteFile(jobFile, jb, 0644); err != nil {
			t.Fatalf("Failed to update job file: %v", err)
		}
		time.Sleep(100 * time.Millisecond)

		job, exists := jobCache.Get("1000")
		if !exists {
			t.Errorf("Expected job 1000 to exist in cache after file update")
		}
		if job.JobName != "updated_job" || job.Hashname != "1000.pbs" {
			t.Errorf("Expected JobName='%v' Hashname='%v', got %v, %v", j.JobName, j.Hashname, job.JobName, job.Hashname)
		}
	})

	t.Run("Delete", func(t *testing.T) {
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
