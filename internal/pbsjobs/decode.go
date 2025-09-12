package pbsjobs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/docker/go-units"
)

type ErrUnknownJobAttribute struct {
	Name     string
	Resource string
}

func (e *ErrUnknownJobAttribute) Error() string {
	return fmt.Sprintf("attribute '%s', resource: '%s' unknown", e.Name, e.Resource)
}

type Decoder struct {
	r      *bufio.Reader
	b      []byte
	strict bool
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:      bufio.NewReaderSize(r, 256),
		b:      make([]byte, 1024),
		strict: false,
	}
}

func (dec *Decoder) setStrict(b bool) {
	dec.strict = b
}

func (dec *Decoder) decodeHeader(header *JobHeader) error {
	if err := binary.Read(dec.r, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	return nil
}

func (dec *Decoder) parseAttributeValue(value string, attr reflect.Value, separator string) error {
	if attr.Kind() == reflect.Ptr {
		attr = attr.Elem()
	}

	if !attr.IsValid() || !attr.CanSet() {
		return fmt.Errorf("job attribute '%v' is unassignable", attr)
	}

	switch attr.Kind() {
	case reflect.Bool:
		if value == "" {
			value = "false"
		}
		bValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parsing bool job attribute '%v': %w", value, err)
		}
		attr.SetBool(bValue)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			attr.SetInt(0)
			return nil
		}
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			attr.SetInt(intValue)
		} else if intValue, err := units.RAMInBytes(value); err == nil {
			attr.SetInt(intValue)
		} else {
			return fmt.Errorf("parsing int job attribute '%v': %w", value, err)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == "" {
			attr.SetUint(0)
			return nil
		}
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parsing uint job attribute '%v': %w", value, err)
		}
		attr.SetUint(uintValue)

	case reflect.String:
		attr.SetString(value)

	case reflect.Slice:
		var values []string
		if value != "" {
			values = strings.Split(value, separator)
		}
		s := reflect.MakeSlice(attr.Type(), len(values), len(values))
		for i := range values {
			if err := dec.parseAttributeValue(values[i], s.Index(i), separator); err != nil {
				return fmt.Errorf("parsing %v slice: %w", attr.Type(), err)
			}
		}
		attr.Set(s)

	default:
		return fmt.Errorf("invalid data type %v", attr.Kind())
	}

	return nil
}

func (dec *Decoder) decodeAttributeValue(job *Job, attr *JobAttr) error {
	jobVal := reflect.ValueOf(job).Elem()

	if key, ok := jobMap[JobMapKey{Name: attr.Name, Resource: attr.Resource}]; ok {
		attrVal := jobVal.FieldByIndex(key.Index)
		if err := dec.parseAttributeValue(attr.Value, attrVal, key.Separator); err != nil {
			return fmt.Errorf("parsing key %v.%v: %w", attr.Name, attr.Resource, err)
		}
	} else if dec.strict {
		return &ErrUnknownJobAttribute{Name: attr.Name, Resource: attr.Resource}
	}

	return nil
}

func (dec *Decoder) decodeAttributeHeader(header *JobAttrHeader, attr *JobAttr) error {
	if err := binary.Read(dec.r, binary.LittleEndian, header); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("reading job attribute header: %w", err)
	}

	// dummy attribute header indicating end of attribute list
	if header.Length == JobAttrEndFlag {
		return io.EOF
	}

	// read into byte slice as attribute uses sizes over reliable delimiter
	requiredBuf := max(header.Name, header.Resource, header.Value)
	if cap(dec.b) < int(requiredBuf) {
		dec.b = make([]byte, requiredBuf)
	}

	dec.b = dec.b[:header.Name]
	if _, err := io.ReadFull(dec.r, dec.b); err != nil {
		return fmt.Errorf("reading job attribute name: %w", err)
	}
	attr.Name = string(bytes.Trim(dec.b, "\x00"))

	dec.b = dec.b[:header.Resource]
	if _, err := io.ReadFull(dec.r, dec.b); err != nil {
		return fmt.Errorf("reading job attribute resource: %w", err)
	}
	attr.Resource = string(bytes.Trim(dec.b, "\x00"))

	// value is suffixed with 2 bytes; the last may not be null
	dec.b = dec.b[:header.Value]
	if _, err := io.ReadFull(dec.r, dec.b); err != nil {
		return fmt.Errorf("reading job attribute value: %w", err)
	}
	attr.Value = string(bytes.Trim(dec.b[:max(0, len(dec.b)-1)], "\x00"))

	seek := header.Length - int32(jobAttrHeaderSize) - header.Name - header.Resource - header.Value
	if _, err := dec.r.Discard(int(seek)); err != nil {
		return fmt.Errorf("failed to seek to next attribute: %w", err)
	}

	return nil
}

func (dec *Decoder) decodeAttribute(job *Job, header *JobAttrHeader, attr *JobAttr) error {
	if err := dec.decodeAttributeHeader(header, attr); err != nil {
		return err
	}

	if err := dec.decodeAttributeValue(job, attr); err != nil {
		return err
	}

	return nil
}

func (dec *Decoder) Decode(job *Job) error {
	_, err := dec.r.Discard(JobAttrStartPos - jobAttrHeaderSize)
	if err != nil {
		return fmt.Errorf("seeking to start position %v: %w", JobAttrStartPos, err)
	}

	header := new(JobAttrHeader)
	attr := new(JobAttr)
	for {
		if err := dec.decodeAttribute(job, header, attr); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("attribute decode failure: %w", err)
		}
	}

	return nil
}

func Unmarshal(data []byte, job *Job) error {
	reader := bytes.NewReader(data)
	decoder := NewDecoder(reader)
	err := decoder.Decode(job)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
