package collector

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestUnmarshalJSON(t *testing.T) {
	type testStruct struct {
		Mem hbytes `json:"mem"`
	}
	tests := []struct {
		name    string
		input   string
		output  *testStruct
		wantErr bool
	}{
		{
			name:    "Null memory",
			input:   `{"mem": null}`,
			output:  &testStruct{Mem: 0},
			wantErr: false,
		},
		{
			name:    "Human readable memory",
			input:   `{"mem": "1GiB"}`,
			output:  &testStruct{Mem: 1 * 1024 * 1024 * 1024},
			wantErr: false,
		},
		{
			name:    "Numeric memory as string",
			input:   `{"mem": "1024"}`,
			output:  &testStruct{Mem: 1024},
			wantErr: false,
		},
		{
			name:    "Numeric memory as integer",
			input:   `{"mem": 1024}`,
			output:  &testStruct{Mem: 1024},
			wantErr: false,
		},
		{
			name:    "Empty string",
			input:   `{"mem": ""}`,
			output:  &testStruct{},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := &testStruct{}
			err := json.Unmarshal([]byte(test.input), got)
			if (err != nil) != test.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(test.output, got) {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, test.output)
			}
		})
	}
}

func TestVnode(t *testing.T) {
	node := &node{}
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"cpu1n001", ""},
		{"cpu1n001[0]", "0"},
	}

	for _, test := range tests {
		node.ResourcesAvailable.Vnode = test.input
		got := node.Vnode()
		if got != test.want {
			t.Errorf("Vnode() = %v, want %v", got, test.want)
		}
	}
}

func TestGetIsLicensed(t *testing.T) {
	node := &node{}
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"l", 1},
	}

	for _, test := range tests {
		node.License = test.input
		got := node.getIsLicensed()
		if got != test.want {
			t.Errorf("getIsLicensed() = %v, want %v", got, test.want)
		}
	}
}

func TestGetNodeState(t *testing.T) {
	node := &node{}
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"free", 1},
		{"free,down,offline", 161},
	}

	for _, test := range tests {
		node.State = test.input
		got := node.getNodeState()
		if got != test.want {
			t.Errorf("getNodeState() = %v, want %v", got, test.want)
		}
	}
}

func TestGetNodeStates(t *testing.T) {
	node := &node{}
	tests := []struct {
		input     string
		want      []string
		wantError bool
	}{
		{"", nil, true},
		{"free", []string{"free"}, false},
		{"free,down,offline", []string{"free", "down", "offline"}, false},
		{"free,Down,Offline", []string{"free", "down", "offline"}, false},
	}
	for _, test := range tests {
		node.State = test.input
		got, err := node.getNodeStates()
		if err != nil && !test.wantError {
			t.Errorf("getNodeStates() error = %v", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("getNodeStates() = %v, want %v", got, test.want)
		}
	}
}

func TestStateAvailable(t *testing.T) {
	node := &node{}
	tests := []struct {
		input     string
		want      bool
		wantError bool
	}{
		{"", false, true},
		{"free", true, false},
		{"offline,free", false, false},
		{"free,offline", false, false},
		{"unknown-node-state", false, true},
	}

	for _, test := range tests {
		node.State = test.input
		got, err := node.stateAvailable()
		if err != nil && !test.wantError {
			t.Errorf("stateAvailable() error = %v", err)
		}
		if got != test.want {
			t.Errorf("stateAvailable() = %v, want %v", got, test.want)
		}
	}
}

func TestExecute(t *testing.T) {
	executor := &shellCommandExecutor{}
	tests := []struct {
		name     string
		command  []string
		stdout   string
		stderr   string
		exitCode int
	}{
		{
			name:     "Echo command",
			command:  []string{"echo", "Hello, World!"},
			stdout:   "Hello, World!\n",
			stderr:   "",
			exitCode: 0,
		},
		{
			name:     "Sleep command",
			command:  []string{"sleep", "1"},
			stdout:   "",
			stderr:   "",
			exitCode: 0,
		},
		{
			name:     "Error",
			command:  []string{"cat", "not_a_real_file"},
			stdout:   "",
			stderr:   "cat: not_a_real_file: No such file or directory\n",
			exitCode: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := executor.execute(test.command)
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != test.exitCode {
				t.Errorf("execute() error = %v, want exit code %d", err, test.exitCode)
			}
			if stdout.String() != test.stdout {
				t.Errorf("execute() stdout = %q, want %q", stdout.String(), test.stdout)
			}
			if stderr.String() != test.stderr {
				t.Errorf("execute() stderr = %q, want %q", stderr.String(), test.stderr)
			}
		})
	}
}

func TestPbsNodeCommand(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", []string{"pbsnodes", "-av", "-F", "json"}},
		{"cpu1n001", []string{"pbsnodes", "-H", "cpu1n001", "json"}},
	}

	for _, test := range tests {
		got := pbsNodeCommand(test.input)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("pbsNodeCommand(%q) = %v, want: %v", test.input, got, test.want)
		}
	}
}

func TestParsePbsNodes(t *testing.T) {
	nodes := &nodes{}

	t.Run("Success", func(t *testing.T) {
		nodeOutput, err := os.ReadFile("./testdata/node.json")
		if err != nil {
			t.Fatalf("Failed to read testdata: %v", err)
		}
		err = parsePbsNodes(nodeOutput, nodes)
		if err != nil {
			t.Errorf("Error parsing pbsnodes output: %v", err)
		}
		if len(nodes.Nodes) != 1 {
			t.Errorf("Expected nodes to be parsed, got empty nodes")
		}
		if nodes.Nodes["gpu1n001"].ResourcesAvailable.Arch != "linux" {
			t.Errorf("Expected node architecture to be 'linux', got '%s'", nodes.Nodes["gpu1n001"].ResourcesAvailable.Arch)
		}
	})

	t.Run("Success with unknown nodes", func(t *testing.T) {
		nodeOutput := []byte(`{"timestamp":0,"pbs_version":"2024.1.2.20241017100211","pbs_server":"server","nodes":{"pbs":{"Error":"Unknown node "}}}`)
		err := parsePbsNodes(nodeOutput, nodes)
		if err != nil {
			t.Errorf("Error parsing pbsnodes output: %v", err)
		}
	})

	t.Run("Success with invalid timestamp", func(t *testing.T) {
		nodeOutput := []byte(`{"timestamp": "","pbs_version":"2024.1.2.20241017100211","pbs_server":"server","nodes":{"pbs":{"Error":"Unknown node "}}}`)
		err := parsePbsNodes(nodeOutput, nodes)
		if err != nil {
			t.Errorf("Error parsing pbsnodes output: %v", err)
		}
	})

	t.Run("Fail with empty data", func(t *testing.T) {
		nodeOutput := []byte(``)
		err := parsePbsNodes(nodeOutput, nodes)
		if err == nil {
			t.Errorf("Expected error when parsing empty data, got nil")
		}
	})
}

type mockCommandExecutor struct {
	stdoutData string
	stderrData string
	err        error
	calledWith []string
}

func (m *mockCommandExecutor) execute(command []string) (bytes.Buffer, bytes.Buffer, error) {
	m.calledWith = command
	var stdout, stderr bytes.Buffer
	stdout.WriteString(m.stdoutData)
	stderr.WriteString(m.stderrData)
	return stdout, stderr, m.err
}

func TestDescribeNodes(t *testing.T) {
	nodeCollector := NewNodeCollector(configEnabled)
	ch := make(chan *prometheus.Desc, 30)
	nodeCollector.Describe(ch)
	close(ch)

	got := 0
	want := 9
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

func TestCollectNodes(t *testing.T) {
	content, err := os.ReadFile("./testdata/node.json")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	nodeOutput := string(content)

	content, err = os.ReadFile("./testdata/vnode.json")
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}
	vnodeOutput := string(content)

	tests := []struct {
		name                string
		mockExecutor        *mockCommandExecutor
		metricsCount        int
		expectedLogSnippets []string
	}{
		{
			name: "Node collection",
			mockExecutor: &mockCommandExecutor{
				stdoutData: nodeOutput,
			},
			metricsCount:        9,
			expectedLogSnippets: []string{},
		},
		{
			name: "vNode collection",
			mockExecutor: &mockCommandExecutor{
				stdoutData: vnodeOutput,
			},
			metricsCount:        18,
			expectedLogSnippets: []string{},
		},
		{
			name: "getPbsNodes returns error",
			mockExecutor: &mockCommandExecutor{
				err: errors.New("command failed"),
			},
			metricsCount: 0,
			expectedLogSnippets: []string{
				`command failed`,
			},
		},
		{
			name: "getPbsNodes returns stderr",
			mockExecutor: &mockCommandExecutor{
				stderrData: "server error",
			},
			metricsCount: 0,
			expectedLogSnippets: []string{
				`server error`,
			},
		},
		{
			name: "Empty pbsnodes output",
			mockExecutor: &mockCommandExecutor{
				stdoutData: `{"nodes": {}}`,
			},
			metricsCount:        0,
			expectedLogSnippets: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
			nodeCollector := NewNodeCollector(configEnabled)
			nodeCollector.executor = tt.mockExecutor
			nodeCollector.logger = testLogger

			registry := prometheus.NewRegistry()
			registry.MustRegister(nodeCollector)

			got := testutil.CollectAndCount(registry)
			if got != tt.metricsCount {
				t.Errorf("CollectAndCount() = %d, want %d", got, tt.metricsCount)
			}

			lint, err := testutil.CollectAndLint(registry)
			if err != nil {
				t.Fatalf("CollectAndLint failed: %v", err)
			}
			if len(lint) > 0 {
				t.Errorf("CollectAndLint found issues: %v", lint)
			}

			logOutput := logBuf.String()
			for _, snippet := range tt.expectedLogSnippets {
				if !strings.Contains(logOutput, snippet) {
					t.Errorf("Expected log to contain %q, but got:\n%s", snippet, logOutput)
				}
			}
		})
	}
}
