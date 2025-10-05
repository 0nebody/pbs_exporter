package pbsnode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/docker/go-units"
)

var (
	executor       cmdExecutor = &shellCmdExecutor{}
	pbsVnodeRegexp             = regexp.MustCompile(`[a-zA-Z0-9_.-]+\[(\d)\]`)
)

type hbytes int64

func (hrb *hbytes) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*hrb = 0
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		value, err := units.RAMInBytes(s)
		if err != nil {
			return fmt.Errorf("parse human-readable bytes string '%s': %w", s, err)
		}
		*hrb = hbytes(value)
		return nil
	}

	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*hrb = hbytes(num)
		return nil
	}

	return fmt.Errorf("unmarshalling '%s' as hbytes", string(data))
}

type Nodes struct {
	// PBS returns timestamp as int, but occasionally returns empty string.
	// Timestamp  int             `json:"timestamp"`
	PbsVersion string          `json:"pbs_version"`
	PbsServer  string          `json:"pbs_server"`
	Nodes      map[string]Node `json:"nodes"`
}

type Node struct {
	Mom                 string             `json:"Mom"`
	Port                int                `json:"Port"`
	PbsVersion          string             `json:"pbs_version"`
	Ntype               string             `json:"ntype"`
	State               string             `json:"state"`
	PCpus               int                `json:"pcpus"`
	Jobs                []string           `json:"jobs"`
	ResourcesAvailable  resourcesAvailable `json:"resources_available"`
	ResourcesAssigned   resourcesAssigned  `json:"resources_assigned"`
	Comment             string             `json:"comment"`
	ResvEnable          string             `json:"resv_enable"`
	Sharing             string             `json:"sharing"`
	InMultivnodeHost    int                `json:"in_multivnode_host"`
	License             string             `json:"license"`
	Partition           string             `json:"partition"`
	LastStateChangeTime int                `json:"last_state_change_time"`
	LastUsedTime        int                `json:"last_used_time"`
	ServerInstanceId    string             `json:"server_instance_id"`
}

type resourcesAvailable struct {
	Arch   string `json:"arch"`
	Host   string `json:"host"`
	Hpmem  hbytes `json:"hpmem"`
	Mem    hbytes `json:"mem"`
	Ncpus  int    `json:"ncpus"`
	Ngpus  int    `json:"ngpus"`
	Nfpgas int    `json:"nfpgas"`
	Qlist  string `json:"qlist"`
	Vmem   hbytes `json:"vmem"`
	Vnode  string `json:"vnode"`
}

type resourcesAssigned struct {
	Hpmem hbytes `json:"hpmem"`
	Mem   hbytes `json:"mem"`
	Ncpus int    `json:"ncpus"`
	Ngpus int    `json:"ngpus"`
	Vmem  hbytes `json:"vmem"`
}

func (n Node) Vnode() string {
	vnodeMatch := pbsVnodeRegexp.FindStringSubmatch(n.ResourcesAvailable.Vnode)
	if len(vnodeMatch) > 1 {
		return vnodeMatch[1]
	}

	return ""
}

func (n Node) IsLicensed() int {
	if n.License == "l" {
		return 1
	}
	return 0
}

func (n Node) NodeState() int {
	total := 0
	states := map[string]int{
		"free":              1,
		"busy":              2,
		"job-busy":          4,
		"job-exclusive":     8,
		"resv-exclusive":    16,
		"offline":           32,
		"maintenance":       64,
		"down":              128,
		"provisioning":      256,
		"stale":             512,
		"state-unknown":     1024,
		"unresolvable":      2048,
		"wait-provisioning": 4096,
		"sleep":             8192,
		"powered-off":       16384,
		"powering-down":     32768,
		"powering-on":       65536,
	}

	for state := range strings.SplitSeq(n.State, ",") {
		total += states[state]
	}

	return total
}

func (n Node) NodeStates() ([]string, error) {
	nodeState := strings.Split(n.State, ",")
	validStates := []string{
		"busy",
		"down",
		"free",
		"job-busy",
		"job-exclusive",
		"maintenance",
		"offline",
		"powered-off",
		"powering-down",
		"powering-on",
		"provisioning",
		"resv-exclusive",
		"sleep",
		"stale",
		"state-unknown",
		"unresolvable",
		"wait-provisioning",
	}

	for i, state := range nodeState {
		lState := strings.ToLower(state)
		if slices.Contains(validStates, lState) {
			nodeState[i] = lState
		} else {
			return nil, fmt.Errorf("unknown node state: %s", lState)
		}
	}

	return nodeState, nil
}

func (n Node) IsAvailable() (bool, error) {
	availableStates := []string{
		"busy",
		"free",
		"job-busy",
		"job-exclusive",
		"resv-exclusive",
	}
	unavailableStates := []string{
		"down",
		"maintenance",
		"offline",
		"provisioning",
		"stale",
		"state-unknown",
		"unresolvable",
		"wait-provisioning",
	}

	nodeStates, err := n.NodeStates()
	if err != nil {
		return false, err
	}

	isAvailable := slices.ContainsFunc(nodeStates, func(s string) bool {
		return slices.Contains(availableStates, s)
	})
	isUnavailable := slices.ContainsFunc(nodeStates, func(s string) bool {
		return slices.Contains(unavailableStates, s)
	})

	return isAvailable && !isUnavailable, nil
}

type cmdExecutor interface {
	execute(ctx context.Context, command []string) (stdout, stderr bytes.Buffer, err error)
}

type shellCmdExecutor struct{}

func (s *shellCmdExecutor) execute(ctx context.Context, command []string) (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return stdout, stderr, fmt.Errorf("context deadline exceeded: %v", err)
		}
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

func pbsNodeCommand(node string) []string {
	command := []string{"pbsnodes", "-H", node, "json"}
	if node == "" {
		command = []string{"pbsnodes", "-av", "-F", "json"}
	}
	return command
}

func parsePbsNodes(output []byte, nodes *Nodes) error {
	if err := json.Unmarshal(output, &nodes); err != nil {
		return err
	}

	return nil
}

func GetPbsNodes(ctx context.Context) (*Nodes, error) {
	nodeInfo := new(Nodes)
	command := pbsNodeCommand("")
	stdout, stderr, err := executor.execute(ctx, command)
	if err != nil {
		return nodeInfo, err
	}
	if stderr.Len() > 0 {
		return nodeInfo, fmt.Errorf("pbsnodes command stderr: %s", stderr.String())
	}

	err = parsePbsNodes(stdout.Bytes(), nodeInfo)
	return nodeInfo, err
}
