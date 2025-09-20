package pbsjob

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type Encoder struct {
	w io.Writer
	b *bytes.Buffer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
		b: &bytes.Buffer{},
	}
}

func (enc *Encoder) encodeHeader(header *JobHeader) error {
	if err := binary.Write(enc.w, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("writing job file header: %w", err)
	}

	return nil
}

func (enc *Encoder) encodeAttributeValue(v reflect.Value, separator string) (string, error) {
	var err error

	switch v.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil

	case reflect.String:
		return v.String(), nil

	case reflect.Array, reflect.Slice:
		values := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			values[i], err = enc.encodeAttributeValue(v.Index(i), separator)
			if err != nil {
				return "", fmt.Errorf("parsing %v slice: %w", v.Type(), err)
			}
		}
		return strings.Join(values, separator), nil

	default:
		return "", fmt.Errorf("invalid data type %v", v.Kind())
	}
}

func (enc *Encoder) encodeAttribute(attr *JobAttr, v reflect.Value, separator string) error {
	// convert attribute value to string
	av, err := enc.encodeAttributeValue(v, separator)
	if err != nil {
		return fmt.Errorf("converting attribute %v: %w", attr, err)
	}
	attr.Value = av

	// write job attribute header
	header := JobAttrHeader{
		Length:   int32(jobAttrHeaderSize) + JobAttrPadding,
		Name:     int32(attrLength(len(attr.Name), 1)),
		Resource: int32(attrLength(len(attr.Resource), 1)),
		Value:    int32(attrLength(len(attr.Value), 2)),
		Flags:    0,
		RefCount: 0,
	}
	header.Length += header.Name + header.Resource + header.Value
	if err := binary.Write(enc.b, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("writing job attribute header: %w", err)
	}

	// write job attribute value
	if header.Name > 0 {
		enc.b.WriteString(attr.Name)
		enc.b.WriteByte('\x00')
	}
	if header.Resource > 0 {
		enc.b.WriteString(attr.Resource)
		enc.b.WriteByte('\x00')
	}
	if header.Value > 0 {
		enc.b.WriteString(attr.Value)
		enc.b.WriteByte('\x00')
		enc.b.WriteByte('\x00')
	}

	// write job attribute padding
	b := enc.b.AvailableBuffer()
	b = append(b, make([]byte, JobAttrPadding)...)
	enc.b.Write(b)

	// write job attribute from buffer to io.Writer
	if _, err := enc.b.WriteTo(enc.w); err != nil {
		return fmt.Errorf("writing job attribute values: %w", err)
	}

	return nil
}

func (enc *Encoder) Encode(job *Job) error {
	jobVal := reflect.ValueOf(job).Elem()

	// write job header
	if _, err := enc.w.Write(make([]byte, JobAttrStartPos-jobAttrHeaderSize)); err != nil {
		return fmt.Errorf("writing job header: %w", err)
	}

	// write job attributes
	for i := range jobMapOrder {
		v := jobMap[jobMapOrder[i]]
		attr := &JobAttr{
			Name:     jobMapOrder[i].Name,
			Resource: jobMapOrder[i].Resource,
		}
		value := jobVal.FieldByIndex(v.Index)

		if err := enc.encodeAttribute(attr, value, v.Separator); err != nil {
			return fmt.Errorf("encoding attribute: %w", err)
		}
	}

	// write PBS specific constant; end of attributes
	header := JobAttrHeader{Length: JobAttrEndFlag}
	if err := binary.Write(enc.w, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("writing end of pbs job attributes: %w", err)
	}

	return nil
}

func Marshal(job *Job) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewEncoder(&buf)
	if err := encoder.Encode(job); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
