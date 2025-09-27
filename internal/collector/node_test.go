package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/0nebody/pbs_exporter/internal/pbsnode"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDescribeNodes(t *testing.T) {
	nodeCollector := NewNodeCollector(configEnabled)
	ch := make(chan *prometheus.Desc)
	go func() {
		defer close(ch)
		nodeCollector.Describe(ch)
	}()

	got := 0
	want := reflect.TypeOf(*nodeCollector.metrics).NumField()
	for desc := range ch {
		got++

		fqName := promDescFqname(desc.String())
		if !strings.HasPrefix(fqName, "pbs_node_") {
			t.Errorf("Describe() = %s, want: %s", fqName, "pbs_node_.*")
		}

		help := promDescHelp(desc.String())
		if len(help) == 0 {
			t.Errorf("Describe() expected help to be non-empty description of metric")
		}
	}

	if got != want {
		t.Errorf("Describe() = %d, want %d", got, want)
	}
}

func mockPbsNodes(ctx context.Context) (*pbsnode.Nodes, error) {
	nodes := new(pbsnode.Nodes)
	content, err := os.ReadFile("./testdata/node.json")
	if err != nil {
		return nodes, fmt.Errorf("Failed to read testdata: %v", err)
	}
	err = json.Unmarshal(content, &nodes)
	return nodes, err
}

func TestCollectNodes(t *testing.T) {
	nodeCollector := NewNodeCollector(configEnabled)
	nodeCollector.pbsNodes = mockPbsNodes
	registry := prometheus.NewRegistry()
	registry.MustRegister(newCollectorContext(nodeCollector))

	got := testutil.CollectAndCount(registry)
	want := reflect.TypeOf(*nodeCollector.metrics).NumField()
	if got != want {
		t.Errorf("CollectAndCount() = %d, want %d", got, want)
	}

	lint, err := testutil.CollectAndLint(registry)
	if err != nil {
		t.Fatalf("CollectAndLint failed: %v", err)
	}
	if len(lint) > 0 {
		t.Errorf("CollectAndLint found issues: %v", lint)
	}
}
