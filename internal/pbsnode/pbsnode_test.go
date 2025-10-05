package pbsnode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

var testNode = Nodes{
	PbsVersion: "2025.2.0.20250218043111",
	PbsServer:  "pbs",
	Nodes: map[string]Node{
		"gpu1n001": {
			Mom:        "gpu1n001.local.domain",
			Port:       15002,
			PbsVersion: "2025.2.0.20250218043111",
			Ntype:      "PBS",
			State:      "free",
			PCpus:      192,
			Jobs:       []string{"1000.pbs", "1001.pbs"},
			ResourcesAvailable: resourcesAvailable{
				Arch:   "linux",
				Host:   "gpu1n001",
				Hpmem:  17741905920,
				Mem:    1046853922816,
				Ncpus:  168,
				Ngpus:  4,
				Nfpgas: 0,
				Qlist:  "gpu_batch_exec,ded_gpu_b,eb_build",
				Vmem:   1046853922816,
				Vnode:  "gpu1n001",
			},
			ResourcesAssigned: resourcesAssigned{
				Hpmem: 0,
				Mem:   137438953472,
				Ncpus: 96,
				Ngpus: 4,
				Vmem:  0,
			},
			Comment:             "",
			ResvEnable:          "True",
			Sharing:             "default_shared",
			InMultivnodeHost:    0,
			License:             "l",
			Partition:           "gpu_batch",
			LastStateChangeTime: 1749687411,
			LastUsedTime:        1749878949,
			ServerInstanceId:    "login.local.domain:15001",
		},
	},
}

var testVNode = Nodes{
	PbsVersion: "2024.1.2.20241017100211",
	PbsServer:  "pbs",
	Nodes: map[string]Node{
		"cpu1n001": {
			Mom:        "cpu1n001.local.domain",
			Port:       15002,
			PbsVersion: "2024.1.2.20241017100211",
			Ntype:      "PBS",
			State:      "free",
			PCpus:      384,
			Jobs:       []string(nil),
			ResourcesAvailable: resourcesAvailable{
				Arch:   "linux",
				Host:   "cpu1n001",
				Hpmem:  0,
				Mem:    0,
				Ncpus:  0,
				Ngpus:  0,
				Nfpgas: 0,
				Qlist:  "",
				Vmem:   0,
				Vnode:  "cpu1n001",
			},
			ResourcesAssigned: resourcesAssigned{
				Hpmem: 0,
				Mem:   0,
				Ncpus: 0,
				Ngpus: 0,
				Vmem:  0,
			},
			Comment:             "",
			ResvEnable:          "True",
			Sharing:             "default_shared",
			InMultivnodeHost:    1,
			License:             "l",
			Partition:           "cpu_inter",
			LastStateChangeTime: 1744259096,
			LastUsedTime:        0,
			ServerInstanceId:    "login.local.domain:15001",
		},
		"cpu1n001[0]": {
			Mom:        "cpu1n001.local.domain",
			Port:       15002,
			PbsVersion: "2024.1.2.20241017100211",
			Ntype:      "PBS",
			State:      "free",
			PCpus:      188,
			Jobs:       []string{"1000.pbs"},
			ResourcesAvailable: resourcesAvailable{
				Arch:   "linux",
				Host:   "cpu1n001",
				Hpmem:  2955935744,
				Mem:    776358330368,
				Ncpus:  188,
				Ngpus:  0,
				Nfpgas: 0,
				Qlist:  "cpu_inter_exec,ded_cpu_i",
				Vmem:   776358330368,
				Vnode:  "cpu1n001[0]",
			},
			ResourcesAssigned: resourcesAssigned{
				Hpmem: 0,
				Mem:   188978561024,
				Ncpus: 42,
				Ngpus: 0,
				Vmem:  0,
			},
			Comment:             "",
			ResvEnable:          "True",
			Sharing:             "default_shared",
			InMultivnodeHost:    1,
			License:             "l",
			Partition:           "cpu_inter",
			LastStateChangeTime: 1744259096,
			LastUsedTime:        1745965766,
			ServerInstanceId:    "login.local.domain:15001"},
		"cpu1n001[1]": {
			Mom:        "cpu1n001.local.domain",
			Port:       15002,
			PbsVersion: "2024.1.2.20241017100211",
			Ntype:      "PBS",
			State:      "free",
			PCpus:      188,
			Jobs:       []string(nil),
			ResourcesAvailable: resourcesAvailable{
				Arch:   "linux",
				Host:   "cpu1n001",
				Hpmem:  2947547136,
				Mem:    777220259840,
				Ncpus:  188,
				Ngpus:  0,
				Nfpgas: 0,
				Qlist:  "cpu_inter_exec,ded_cpu_i",
				Vmem:   777220259840,
				Vnode:  "cpu1n001[1]",
			},
			ResourcesAssigned: resourcesAssigned{
				Hpmem: 0,
				Mem:   0,
				Ncpus: 0,
				Ngpus: 0,
				Vmem:  0,
			},
			Comment:             "",
			ResvEnable:          "True",
			Sharing:             "default_shared",
			InMultivnodeHost:    1,
			License:             "l",
			Partition:           "cpu_inter",
			LastStateChangeTime: 1744259096,
			LastUsedTime:        1744111433,
			ServerInstanceId:    "login.local.domain:15001",
		},
	},
}

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
	node := &Node{}
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

func TestIsLicensed(t *testing.T) {
	node := &Node{}
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"l", 1},
	}

	for _, test := range tests {
		node.License = test.input
		got := node.IsLicensed()
		if got != test.want {
			t.Errorf("IsLicensed() = %v, want %v", got, test.want)
		}
	}
}

func TestNodeState(t *testing.T) {
	node := &Node{}
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
		got := node.NodeState()
		if got != test.want {
			t.Errorf("NodeState() = %v, want %v", got, test.want)
		}
	}
}

func TestNodeStates(t *testing.T) {
	node := &Node{}
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
		got, err := node.NodeStates()
		if err != nil && !test.wantError {
			t.Errorf("NodeStates() error = %v", err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("NodeStates() = %v, want %v", got, test.want)
		}
	}
}

func TestIsAvailable(t *testing.T) {
	node := &Node{}
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
		got, err := node.IsAvailable()
		if err != nil && !test.wantError {
			t.Errorf("IsAvailable() error = %v", err)
		}
		if got != test.want {
			t.Errorf("IsAvailable() = %v, want %v", got, test.want)
		}
	}
}

func TestExecute(t *testing.T) {
	executor := &shellCmdExecutor{}
	tests := []struct {
		name     string
		command  []string
		stdout   string
		stderr   string
		exitCode int
		timeout  bool
	}{
		{
			name:     "Echo",
			command:  []string{"echo", "Hello, World!"},
			stdout:   "Hello, World!\n",
			stderr:   "",
			exitCode: 0,
			timeout:  false,
		},
		{
			name:     "Sleep",
			command:  []string{"sleep", "0.05"},
			stdout:   "",
			stderr:   "",
			exitCode: 0,
			timeout:  false,
		},
		{
			name:     "Error",
			command:  []string{"cat", "not_a_real_file"},
			stdout:   "",
			stderr:   "cat: not_a_real_file: No such file or directory\n",
			exitCode: 1,
			timeout:  false,
		},
		{
			name:     "Timeout",
			command:  []string{"sleep", "1"},
			stdout:   "",
			stderr:   "",
			exitCode: 1,
			timeout:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			stdout, stderr, err := executor.execute(ctx, test.command)
			if !test.timeout && ctx.Err() == context.DeadlineExceeded {
				t.Errorf("execute() error = %v", err)
			}
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
	nodes := &Nodes{}

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

func (m *mockCommandExecutor) execute(ctx context.Context, command []string) (bytes.Buffer, bytes.Buffer, error) {
	m.calledWith = command
	var stdout, stderr bytes.Buffer
	stdout.WriteString(m.stdoutData)
	stderr.WriteString(m.stderrData)
	return stdout, stderr, m.err
}

func TestGetPbsNodes(t *testing.T) {
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
		name      string
		executor  cmdExecutor
		want      *Nodes
		wantError bool
	}{
		{
			name: "Node collection",
			executor: &mockCommandExecutor{
				stdoutData: nodeOutput,
			},
			want:      &testNode,
			wantError: false,
		},
		{
			name: "vNode collection",
			executor: &mockCommandExecutor{
				stdoutData: vnodeOutput,
			},
			want:      &testVNode,
			wantError: false,
		},
		{
			name: "pbsnodes returns error",
			executor: &mockCommandExecutor{
				err: errors.New("command failed"),
			},
			want:      &Nodes{},
			wantError: true,
		},
		{
			name: "pbsnodes returns stderr",
			executor: &mockCommandExecutor{
				stderrData: "server error",
			},
			want:      &Nodes{},
			wantError: true,
		},
		{
			name: "Empty pbsnodes output",
			executor: &mockCommandExecutor{
				stdoutData: `{"nodes": {}}`,
			},
			want:      &Nodes{Nodes: map[string]Node{}},
			wantError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			executor = test.executor
			got, err := GetPbsNodes(context.Background())
			if err != nil {
				if test.wantError {
					return
				}
				tt.Fatalf("GetPbsNodes() failed: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				tt.Errorf("GetPbsNodes() = %v, want %v", got, test.want)
			}
		})
	}
}
