package pbsjob

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type attributeParsingTest = struct {
	name      string
	value     any
	string    string
	separator string
	valid     bool
}

var attributeParsingTests = []attributeParsingTest{
	{"Empty Slice", []string{}, "", "", true},
	{"Empty String", "", "", "", true},
	{"Bool", true, "true", "", true},
	{"Int", int(100), "100", "", true},
	{"Int8", int8(math.MaxInt8), "127", "", true},
	{"Int16", int16(math.MaxInt16), "32767", "", true},
	{"Int32", int32(math.MaxInt32), "2147483647", "", true},
	{"Int64", int64(math.MaxInt64), "9223372036854775807", "", true},
	{"Uint", uint(0), "0", "", true},
	{"Uint8", uint8(math.MaxUint8), "255", "", true},
	{"Uint16", uint16(math.MaxUint16), "65535", "", true},
	{"Uint32", uint32(math.MaxUint32), "4294967295", "", true},
	{"Uint64", uint64(math.MaxUint64), "18446744073709551615", "", true},
	{"String", "", "", "", true},
	{"Slice Int", []int{1, 2}, "1:2", ":", true},
	{"Slice String", []string{"a", "b"}, "a,b", ",", true},
	{"Invalid Slice ", []int{}, "1:a", ":", false},
	{"Invalid Type", make(map[string]int), "", "", false},
}

func loadJobJson(filename string, job *Job) error {
	testFile := filepath.Join("testdata", filename)
	contents, err := os.ReadFile(testFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(contents, job); err != nil {
		return err
	}
	return nil
}

func loadJobBinary(filename string) ([]byte, error) {
	testFile := filepath.Join("testdata", filename)
	contents, err := os.ReadFile(testFile)
	if err != nil {
		return []byte{}, err
	}
	return contents, nil
}

func generateJobFile(name string, jobid string, stime int64) (Job, []byte, error) {
	if stime == 0 {
		stime = time.Now().Unix()
	}

	job := Job{}
	if err := loadJobJson("job_1000.json", &job); err != nil {
		return job, nil, err
	}

	job.Hashname = jobid
	job.JobName = name
	job.Mtime = stime
	job.Stime = stime

	bytes, err := Marshal(&job)
	if err != nil {
		return job, bytes, err
	}

	return job, bytes, nil
}

func TestJobMapCache(t *testing.T) {
	type TestJob struct {
		JobName      string `pbs:"Job_Name"`
		JobOwner     string
		ResourceList struct {
			Mem int `pbs:"mem"`
		} `pbs:"Resource_List"`
		Binding []string `pbs:"binding" sep:":"`
	}

	wantCache := JobMap{}
	wantCache[JobMapKey{Name: "Job_Name", Resource: ""}] = JobMapValue{
		Index:     []int{0},
		Separator: ",",
	}
	wantCache[JobMapKey{Name: "JobOwner", Resource: ""}] = JobMapValue{
		Index:     []int{1},
		Separator: ",",
	}
	wantCache[JobMapKey{Name: "Resource_List", Resource: "mem"}] = JobMapValue{
		Index:     []int{2, 0},
		Separator: ",",
	}
	wantCache[JobMapKey{Name: "binding", Resource: ""}] = JobMapValue{
		Index:     []int{3},
		Separator: ":",
	}
	wantOrder := JobMapKey{Name: "Resource_List", Resource: "mem"}

	cache, order := NewJobMapCache(reflect.TypeOf(TestJob{}))

	if !reflect.DeepEqual(cache, wantCache) {
		t.Errorf("NewJobMapCache() = %+v, want %+v", cache, wantCache)
	}
	if !reflect.DeepEqual(order[2], wantOrder) {
		t.Errorf("NewJobMapCache() = %+v, want %+v", order[2], wantOrder)
	}
}

func TestJobAttrAssumptions(t *testing.T) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, _ := os.ReadDir(jobFileDir)

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}

		jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
		content, err := os.ReadFile(jobFilePath)
		if err != nil {
			t.Fatalf("Reading job file: %v", err)
		}

		var start int64
		t.Run("JobAttrStartPos", func(tt *testing.T) {
			start, err = findJobAttrStartPos(content, []byte(JobAttrSentinel))
			if err != nil {
				tt.Fatalf("findJobAttrStartPos([]byte, %v): %v", JobAttrSentinel, err)
			}
			if start != JobAttrStartPos {
				tt.Errorf("JobAttrStartPos: job file %s has %d, expected %d", jobFilePath, start, JobAttrStartPos)
			}
		})

		t.Run("JobAttrPadding", func(tt *testing.T) {
			jobAttr := content[start-int64(jobAttrHeaderSize) : start]
			buf := bytes.Buffer{}
			buf.Write(jobAttr)
			header := &JobAttrHeader{}
			if err := binary.Read(&buf, binary.LittleEndian, header); err != nil {
				tt.Fatalf("reading sentinel job attribute header: %v", err)
			}
			filePadding := header.Length - int32(jobAttrHeaderSize) - header.Name - header.Resource - header.Value
			if filePadding != JobAttrPadding {
				tt.Errorf("JobAttrPadding: job file %s has %d, expected %d", jobFilePath, filePadding, JobAttrPadding)
			}
		})

		t.Run("JobAttrEndFlag", func(tt *testing.T) {
			tail := content[len(content)-jobAttrHeaderSize:]
			buf := bytes.Buffer{}
			buf.Write(tail)
			var endFlag int32
			if err := binary.Read(&buf, binary.LittleEndian, &endFlag); err != nil {
				tt.Fatalf("unable to parse final attribute header of file %s: %v", jobFilePath, err)
			}
			if endFlag != JobAttrEndFlag {
				tt.Errorf("JobAttrEndFlag: job file %s missing %d final attribute header flag", jobFilePath, JobAttrEndFlag)
			}
		})
	}
}

func TestAttrSize(t *testing.T) {
	tests := []struct {
		length  int
		padding int
		want    int
	}{
		{0, 0, 0},
		{0, 1, 0},
		{1, 0, 1},
		{1, 1, 2},
	}

	for _, test := range tests {
		got := attrLength(test.length, test.padding)
		if got != test.want {
			t.Errorf("attrLength(%d, %d) = %d, want %d", test.length, test.padding, got, test.want)
		}
	}
}

func TestNewMapKey(t *testing.T) {
	tests := []struct {
		path      []string
		want      JobMapKey
		wantError bool
	}{
		{[]string{}, JobMapKey{}, false},
		{[]string{"n"}, JobMapKey{Name: "n"}, false},
		{[]string{"n", "r"}, JobMapKey{Name: "n", Resource: "r"}, false},
		{[]string{"n", "r", "x"}, JobMapKey{}, true},
	}

	for _, test := range tests {
		got, err := NewJobMapKey(test.path)
		if err != nil && !test.wantError {
			t.Fatalf("NewJobMapKey(%v) error: %v", test.path, err)
		}
		if got != test.want {
			t.Errorf("NewJobMapKey(%v) = %v, want %v", test.path, got, test.want)
		}
	}
}

func TestUnmarshalMarshal(t *testing.T) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, _ := os.ReadDir(jobFileDir)
	job1 := new(Job)
	job2 := new(Job)

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}
		jobFilepath := filepath.Join(jobFileDir, jobFile.Name())
		contents, err := os.ReadFile(jobFilepath)
		if err != nil {
			t.Fatalf("reading file: %v, %v", jobFile.Name(), err)
		}

		err = Unmarshal(contents, job1)
		if err != nil {
			t.Errorf("Unmarshal() failed for file %v: %v", jobFile.Name(), err)
		}
		jobBytes, err := Marshal(job1)
		if err != nil {
			t.Errorf("Marshal() failed for job: %v", err)
		}
		err = Unmarshal(jobBytes, job2)
		if err != nil {
			t.Errorf("Unmarshal() failed for job bytes: %v", err)
		}

		if !reflect.DeepEqual(job1, job2) {
			t.Errorf("Unmarshal(job) != Unmarshal(Marshal(Unmarshal(Job)))")
		}
	}
}
