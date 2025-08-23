package utils

import (
	"reflect"
	"testing"
)

func TestBooleanToInt(t *testing.T) {
	tests := []struct {
		input bool
		want  int
	}{
		{true, 1},
		{false, 0},
	}

	for _, test := range tests {
		got := BooleanToInt(test.input)
		if got != test.want {
			t.Errorf("BooleanToInt(%v) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestGetCgroupJobId(t *testing.T) {
	// V1 path: pbs_jobs.service/jobid
	// V2 path: pbs_jobs.service/jobs
	jobTests := []struct {
		cgroupPath string
		want       string
	}{
		{"12345.pbs", ""},
		{"/pbs_jobs.service/jobid/12345.pbs", "12345"},
		{"/sys/fs/cgroup/cpu,cpuacct/pbs_jobs.service/jobid/12345.pbs", "12345"},
		{"/sys/fs/cgroup/cpu,cpuacct/pbs_jobs.service/jobid/12345.pbs/child", "12345"},
		{"/sys/fs/cgroup/cpu,cpuacct/pbs_jobs.service/jobid/12345[1].pbs", "12345[1]"},
		{"/sys/fs/cgroup/cpu,cpuacct/pbs_jobs.service/jobid/12345[2].pbs/child", "12345[2]"},
		{"/pbs_jobs.service/jobs/12345.pbs", "12345"},
		{"/sys/fs/cgroup/pbs_jobs.service/jobs/12345", "12345"},
		{"/sys/fs/cgroup/pbs_jobs.service/jobs/12345.1", "12345"},
		{"/sys/fs/cgroup/pbs_jobs.service/jobs/12345.1/child", "12345"},
		{"/sys/fs/cgroup/pbs_jobs.service/jobs/12345.2", "12345[1]"},
		{"/sys/fs/cgroup/pbs_jobs.service/jobs/12345.2/child", "12345[1]"},
	}

	for _, test := range jobTests {
		got := GetCgroupJobId(test.cgroupPath)
		if got != test.want {
			t.Errorf("GetCgroupJobId(%s) = %v, want %v", test.cgroupPath, got, test.want)
		}
	}
}

func TestMustHostname(t *testing.T) {
	got := MustHostname()
	if got == "" {
		t.Error("MustHostname() returned an empty string")
	}
}

func TestParseListFormat(t *testing.T) {
	tests := []struct {
		listFormat string
		want       []int
	}{
		{"", nil},
		{"0", []int{0}},
		{"0-4", []int{0, 1, 2, 3, 4}},
		{"0-4,9", []int{0, 1, 2, 3, 4, 9}},
		{"0-2,7,12-14", []int{0, 1, 2, 7, 12, 13, 14}},
	}

	for _, test := range tests {
		got, _ := ParseListFormat(test.listFormat)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("ParseListFormat(%s) = %v, want %v", test.listFormat, got, test.want)
		}
	}
}

func TestParseWalltime(t *testing.T) {
	tests := []struct {
		walltime string
		want     int64
	}{
		{"", 0},
		{"00:00:00", 0},
		{"00:00:01", 1},
		{"00:01:00", 60},
		{"01:00:00", 3600},
		{"10:10:10", 36610},
	}

	for _, test := range tests {
		got := ParseWalltime(test.walltime)
		if got != test.want {
			t.Errorf("getWalltime(%s) = %v, want %v", test.walltime, got, test.want)
		}
	}
}

func TestReadFileSingleLine(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"/dev/null", ""},
		{"nonexistent.go", ""},
		{"utils.go", "package utils"},
	}

	for _, test := range tests {
		got, _ := ReadFileSingleLine(test.filename)
		if got != test.want {
			t.Errorf("ReadFileSingleLine() = %v, want %v", got, test.want)
		}
	}
}

func TestDirectoryExists(t *testing.T) {
	tests := []struct {
		dirname string
		want    bool
	}{
		{"/", true},
		{"..", true},
		{"/notareadldir", false},
	}

	for _, test := range tests {
		got := DirectoryExists(test.dirname)
		if got != test.want {
			t.Errorf("DirectoryExists(%s) = %v, want %v", test.dirname, got, test.want)
		}
	}
}
