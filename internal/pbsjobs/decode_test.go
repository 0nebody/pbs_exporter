package pbsjobs

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDecodeHeader(t *testing.T) {
	buf := bytes.Buffer{}
	dec := NewDecoder(&buf)
	hexString := "60090000210000005a6d4368000000000000000000000000343634353339325b315d2e6171756100"
	jobHeader, err := hex.DecodeString(hexString)
	if err != nil {
		t.Fatalf("decode hex string error: %v", err)
	}
	buf.Write(jobHeader)

	got := &JobHeader{}
	want := &JobHeader{
		Version:  2400,
		Flags:    33,
		STime:    1749249370,
		OBitTime: 0,
	}
	if err := dec.decodeHeader(got); err != nil {
		t.Fatalf("decodeHeader() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decodeHeader() = %+v, want %+v", got, want)
	}
}

func TestParseAttributeValue(t *testing.T) {
	var buf bytes.Buffer
	dec := NewDecoder(&buf)
	tests := append(
		attributeParsingTests,
		attributeParsingTest{"Empty Bool", false, "", "", true},
		attributeParsingTest{"Empty Int", int(0), "", "", true},
	)

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			want := test.value
			gotV := reflect.New(reflect.TypeOf(want)).Elem()
			if err := dec.parseAttributeValue(test.string, gotV, test.separator); err != nil {
				if test.valid {
					tt.Fatalf("parseAttributeValue(%v, %T, %s) returned error: %v", test.value, gotV, test.separator, err)
				}
				return
			}
			got := gotV.Interface()
			if !reflect.DeepEqual(got, want) {
				tt.Errorf("parseAttributeValue(%v, %T, %s) = %v, want %v", test.value, gotV, test.separator, got, want)
			}
		})
	}
}

func TestDecodeAttribute(t *testing.T) {
	b1, _ := hex.DecodeString("7700000009000000000000000600000000000000000000004a6f625f4e616d65006e616d650000")
	b2, _ := hex.DecodeString("800000000f000000040000000500000000000000000000007265736f75726365735f75736564006d656d003132330000")
	padding := make([]byte, JobAttrPadding)
	tests := []struct {
		bytes []byte
		want  JobAttr
	}{
		{append(b1, padding...), JobAttr{Name: "Job_Name", Resource: "", Value: "name"}},
		{append(b2, padding...), JobAttr{Name: "resources_used", Resource: "mem", Value: "123"}},
	}

	for _, test := range tests {
		job := &Job{}
		buf := bytes.Buffer{}
		dec := NewDecoder(&buf)
		buf.Write(test.bytes)

		header := JobAttrHeader{}
		got := JobAttr{
			Name:     test.want.Name,
			Resource: test.want.Resource,
		}

		if err := dec.decodeAttribute(job, &header, &got); err != nil {
			t.Fatalf("decodeAttribute() returned error: %v", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("decodeAttribute(&Job, &JobAttrHeader, &JobAttr) = %+v, want %+v", got, test.want)
		}
	}
}

func TestDecode(t *testing.T) {
	// test decode with mock binary file
	t.Run("Decode", func(tt *testing.T) {
		gotJob := new(Job)
		wantJob := new(Job)
		if err := loadJobJson("job_1000.json", wantJob); err != nil {
			t.Fatalf("load mock job from json: %v", err)
		}

		jobFile := "testdata/job_1000.bin"
		r, err := os.Open(jobFile)
		if err != nil {
			t.Fatalf("load mock binary job file: %s", jobFile)
		}
		dec := NewDecoder(r)

		if err := dec.Decode(gotJob); err != nil {
			t.Fatalf("Decode() returned error: %v", err)
		}
		if !reflect.DeepEqual(gotJob, wantJob) {
			t.Errorf("Decode(%s) = %+v, want %+v", jobFile, gotJob, wantJob)
		}
	})

	// check files in testdata for new PBS attributes
	t.Run("Strict", func(tt *testing.T) {
		jobFileDir := "./testdata/jobfiles"
		jobFiles, _ := os.ReadDir(jobFileDir)
		job := new(Job)
		for _, jobFile := range jobFiles {
			if !strings.HasSuffix(jobFile.Name(), ".JB") {
				continue
			}

			jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
			content, err := os.ReadFile(jobFilePath)
			if err != nil {
				tt.Fatalf("Failed to read job file: %v", err)
			}

			r := bytes.NewReader(content)
			dec := NewDecoder(r)
			dec.setStrict(true)
			if err := dec.Decode(job); err != nil {
				tt.Errorf("Failed to decode job file: %v", err)
			}
		}
	})
}

func BenchmarkDecode(b *testing.B) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, _ := os.ReadDir(jobFileDir)
	job := new(Job)

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}

		jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
		content, err := os.ReadFile(jobFilePath)
		if err != nil {
			b.Fatalf("Failed to read job file: %v", err)
		}

		r := bytes.NewReader(content)
		b.Run(jobFile.Name(), func(b *testing.B) {
			for b.Loop() {
				r.Reset(content)
				dec := NewDecoder(r)
				if err := dec.Decode(job); err != nil {
					b.Errorf("Failed to decode job file: %v", err)
				}
			}
		})
	}
}
