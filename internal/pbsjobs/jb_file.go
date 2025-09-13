package pbsjobs

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

type JobMap map[JobMapKey]JobMapValue

type JobMapKey struct {
	Name     string
	Resource string
}

type JobMapValue struct {
	Index     []int
	Separator string
}

type JobHeader struct {
	Version  int32
	Flags    int32
	STime    int64
	OBitTime int64
}

type JobAttrHeader struct {
	Length   int32
	Name     int32
	Resource int32
	Value    int32
	Flags    int32
	RefCount int32
}

type JobAttr struct {
	Name     string
	Resource string
	Value    string
}

const (
	JobAttrEndFlag  = -711
	JobAttrPadding  = 80
	JobAttrSentinel = "Job_Name"
	JobAttrStartPos = 1120
)

var (
	jobMap, jobMapOrder = NewJobMapCache(reflect.TypeOf(Job{}))
	jobAttrHeaderSize   = binary.Size(JobAttrHeader{})
)

func NewJobMapCache(t reflect.Type) (JobMap, []JobMapKey) {
	cache := JobMap{}
	order := buildJobMapCache(t, cache, []string{}, []int{})

	return cache, order
}

func NewJobMapKey(path []string) (JobMapKey, error) {
	key := JobMapKey{}

	// PBS Job struct only supports 2 layers of depth
	if len(path) > 2 {
		return key, fmt.Errorf("invalid job map path depth: %v", path)
	}

	if len(path) > 0 {
		key.Name = path[0]
	}
	if len(path) > 1 {
		key.Resource = path[1]
	}

	return key, nil
}

func buildJobMapCache(t reflect.Type, cache JobMap, parent []string, path []int) []JobMapKey {
	order := []JobMapKey{}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	var name string
	var separator string
	for i := 0; i < t.NumField(); i++ {
		// get field tag
		field := t.Field(i)
		if tag, ok := field.Tag.Lookup("pbs"); ok && tag != "" {
			name = tag
		} else {
			name = t.Field(i).Name
		}

		// get field options
		if tag, ok := field.Tag.Lookup("sep"); ok && tag != "" {
			separator = tag
		} else {
			separator = ","
		}

		// recursively build job map
		path := append(path, i)
		parent := append(parent, name)
		if t.Field(i).Type.Kind() == reflect.Struct {
			order = append(order, buildJobMapCache(t.Field(i).Type, cache, parent, path)...)
			continue
		}

		if key, err := NewJobMapKey(parent); err == nil {
			// add leaf attributes to map
			cache[key] = JobMapValue{
				Index:     path,
				Separator: separator,
			}
			order = append(order, key)
		} else {
			panic(err)
		}
	}

	return order
}

func findJobAttrStartPos(contents []byte, sentinel []byte) (int64, error) {
	start := bytes.Index(contents, sentinel)
	if start < 0 {
		return 0, fmt.Errorf("job attribute sentinel '%s' not found", sentinel)
	}

	return int64(start), nil
}

func attrLength(length int, padding int) int {
	if length > 0 {
		return length + padding
	}
	return 0
}
