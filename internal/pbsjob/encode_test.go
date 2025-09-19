package pbsjob

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEncodeHeader(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	header := &JobHeader{
		Version:  1,
		Flags:    1,
		STime:    1757232063,
		OBitTime: 1757232063,
	}
	enc.encodeHeader(header)
}

func TestEncodeAttributeValue(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	for _, test := range attributeParsingTests {
		t.Run(test.name, func(tt *testing.T) {
			got, err := enc.encodeAttributeValue(reflect.ValueOf(test.value), test.separator)
			if err != nil {
				if test.valid {
					tt.Fatalf("encodeAttributeValue(%v, %v) returned error: %v", test.value, test.separator, err)
				}
				return
			}
			want := test.string
			if got != want && test.valid {
				tt.Errorf("encodeAttributeValue(%v, %v) = %v, want %v", test.value, test.separator, got, want)
			}
		})
	}
}

func TestEncodeAttribute(t *testing.T) {
	want1, _ := hex.DecodeString("7700000009000000000000000600000000000000000000004a6f625f4e616d65006e616d650000")
	want2, _ := hex.DecodeString("800000000f000000040000000500000000000000000000007265736f75726365735f75736564006d656d003132330000")
	padding := make([]byte, JobAttrPadding)
	tests := []struct {
		name     string
		resource string
		value    any
		want     []byte
	}{
		{"Job_Name", "", "name", append(want1, padding...)},
		{"resources_used", "mem", 123, append(want2, padding...)},
	}

	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)
	for _, test := range tests {
		attr := &JobAttr{
			Name:     test.name,
			Resource: test.resource,
			Value:    "",
		}
		v := reflect.ValueOf(test.value)
		err := enc.encodeAttribute(attr, v, "")
		if err != nil {
			t.Fatalf("encodeAttribute(%+v, %v, '') returned error: %v", attr, test.value, err)
		}

		got := buf.Bytes()
		buf.Reset()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("encodeAttribute(%+v, %v, '') = %#v, want %#v", attr, test.value, got, test.want)
		}
	}
}

func TestEncode(t *testing.T) {
	job := &Job{}
	if err := loadJobJson("job_1000.json", job); err != nil {
		t.Fatalf("load mock job from json: %v", err)
	}

	wantBytes, err := loadJobBinary("job_1000.bin")
	if err != nil {
		t.Fatalf("load mock binary job file: %v", err)
	}

	buf := bytes.Buffer{}
	enc := NewEncoder(&buf)
	if err := enc.Encode(job); err != nil {
		t.Errorf("Encode() = returned error: %v", err)
	}
	gotBytes := buf.Bytes()
	if !reflect.DeepEqual(gotBytes, wantBytes) {
		t.Errorf("Encode() = %d, want %d", len(gotBytes), len(wantBytes))
	}
}

func BenchmarkEncode(b *testing.B) {
	jobFileDir := "./testdata/jobfiles"
	jobFiles, _ := os.ReadDir(jobFileDir)
	job := new(Job)

	for _, jobFile := range jobFiles {
		if !strings.HasSuffix(jobFile.Name(), ".JB") {
			continue
		}

		// create byte source from job files
		jobFilePath := filepath.Join(jobFileDir, jobFile.Name())
		content, err := os.ReadFile(jobFilePath)
		if err != nil {
			b.Fatalf("Failed to read job file: %v", err)
		}
		err = Unmarshal(content, job)
		if err != nil {
			b.Fatalf("Failed to unmarshal file %v: %v", jobFile.Name(), err)
		}

		b.ResetTimer()

		buf := bytes.Buffer{}
		b.Run(jobFile.Name(), func(b *testing.B) {
			for b.Loop() {
				enc := NewEncoder(&buf)
				if err := enc.Encode(job); err != nil {
					b.Errorf("Failed to encode job file: %v", err)
				}
				buf.Reset()
			}
		})
	}
}
