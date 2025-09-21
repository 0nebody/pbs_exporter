package collector

import (
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	promDescFqNameRegex = regexp.MustCompile(`fqName: "([^"]+)"`)
	promDescHelpRegex   = regexp.MustCompile(`help: "([^"]+)"`)
)

func promDescFqname(description string) string {
	matches := promDescFqNameRegex.FindStringSubmatch(description)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func promDescHelp(description string) string {
	matches := promDescHelpRegex.FindStringSubmatch(description)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

var configEnabled = CollectorConfig{
	CgroupPath:            "cgroupv2/pbs_jobs.service/jobs",
	CgroupRoot:            "./testdata",
	Logger:                slog.New(slog.NewTextHandler(os.Stderr, nil)),
	PbsHome:               "./testdata",
	EnableCgroupCollector: true,
	EnableJobCollector:    true,
	EnableNodeCollector:   true,
}

var configDisabled = CollectorConfig{
	Logger:                slog.New(slog.NewTextHandler(os.Stderr, nil)),
	EnableCgroupCollector: false,
	EnableJobCollector:    false,
	EnableNodeCollector:   false,
}

func TestNewCollectors(t *testing.T) {
	t.Run("Collectors enabled", func(t *testing.T) {
		collectors := NewCollectors(configEnabled)
		rCollectors := reflect.ValueOf(collectors).Elem()

		for i := 0; i < rCollectors.NumField(); i++ {
			if rCollectors.Field(i).IsNil() {
				fieldName := rCollectors.Type().Field(i).Name
				t.Errorf("Expected collectors.%s to be set, got nil", fieldName)
			}
		}
	})

	t.Run("Collectors disabled", func(t *testing.T) {
		collectors := NewCollectors(configDisabled)
		rCollectors := reflect.ValueOf(collectors).Elem()

		for i := 0; i < rCollectors.NumField(); i++ {
			if !rCollectors.Field(i).IsNil() {
				fieldName := rCollectors.Type().Field(i).Name
				t.Errorf("Expected collectors.%s to be nil, got %v", fieldName, rCollectors.Field(i))
			}
		}
	})
}

func TestDescribe(t *testing.T) {
	t.Run("Collectors Enabled", func(t *testing.T) {
		collectors := NewCollectors(configEnabled)
		ch := make(chan *prometheus.Desc)
		go func() {
			defer close(ch)
			collectors.Describe(ch)
		}()

		got := 0
		want := reflect.TypeOf(*collectors.cgroupCollector.metrics).NumField()
		want += reflect.TypeOf(*collectors.jobCollector.metrics).NumField()
		want += reflect.TypeOf(*collectors.nodeCollector.metrics).NumField()

		for desc := range ch {
			got++

			fqName := promDescFqname(desc.String())
			if !strings.HasPrefix(fqName, "pbs_") {
				t.Errorf("Describe() = %s, want: %s", fqName, "pbs_.*")
			}

			help := promDescHelp(desc.String())
			if len(help) == 0 {
				t.Errorf("Describe() expected help to be non-empty description of metric")
			}
		}

		if got != want {
			t.Errorf("Expected %d descriptors, got %d", want, got)
		}
	})

	t.Run("Collectors Disabled", func(t *testing.T) {
		collectors := NewCollectors(configDisabled)
		ch := make(chan *prometheus.Desc)
		go func() {
			defer close(ch)
			collectors.Describe(ch)
		}()

		got := 0
		want := 0
		for range ch {
			got++
		}
		if got != want {
			t.Errorf("Expected %d descriptors, got %d", want, got)
		}
	})
}

func TestCollect(t *testing.T) {
	t.Run("Collectors Enabled", func(t *testing.T) {
		collectors := NewCollectors(configEnabled)
		ch := make(chan *prometheus.Desc)
		go func() {
			defer close(ch)
			collectors.Describe(ch)
		}()
		// won't check actual metrics, as they are tested elsewhere
	})

	t.Run("Collectors Disabled", func(t *testing.T) {
		collectors := NewCollectors(configDisabled)
		ch := make(chan *prometheus.Desc)
		go func() {
			defer close(ch)
			collectors.Describe(ch)
		}()

		got := 0
		want := 0
		for range ch {
			got++
		}
		if got != want {
			t.Errorf("Expected %d metrics, got %d", want, got)
		}
	})
}
