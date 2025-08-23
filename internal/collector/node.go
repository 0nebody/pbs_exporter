package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"slices"
	"strings"

	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/docker/go-units"
	"github.com/prometheus/client_golang/prometheus"
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
			return fmt.Errorf("failed to parse human-readable bytes string '%s': %w", s, err)
		}
		*hrb = hbytes(value)
		return nil
	}

	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*hrb = hbytes(num)
		return nil
	}

	return fmt.Errorf("failed to unmarshal '%s' as hbytes", string(data))
}

type nodes struct {
	// PBS returns timestamp as int, but occasionally returns empty string.
	// Timestamp  int             `json:"timestamp"`
	PbsVersion string          `json:"pbs_version"`
	PbsServer  string          `json:"pbs_server"`
	Nodes      map[string]node `json:"nodes"`
}

type node struct {
	Mom                string   `json:"Mom"`
	Port               int      `json:"Port"`
	PbsVersion         string   `json:"pbs_version"`
	Ntype              string   `json:"ntype"`
	State              string   `json:"state"`
	PCpus              int      `json:"pcpus"`
	Jobs               []string `json:"jobs"`
	ResourcesAvailable struct {
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
	} `json:"resources_available"`
	ResourcesAssigned struct {
		Hpmem hbytes `json:"hpmem"`
		Mem   hbytes `json:"mem"`
		Ncpus int    `json:"ncpus"`
		Ngpus int    `json:"ngpus"`
		Vmem  hbytes `json:"vmem"`
	} `json:"resources_assigned"`
	Comment             string `json:"comment"`
	ResvEnable          string `json:"resv_enable"`
	Sharing             string `json:"sharing"`
	InMultivnodeHost    int    `json:"in_multivnode_host"`
	License             string `json:"license"`
	Partition           string `json:"partition"`
	LastStateChangeTime int    `json:"last_state_change_time"`
	LastUsedTime        int    `json:"last_used_time"`
	ServerInstanceId    string `json:"server_instance_id"`
}

func (n node) Vnode() string {
	vnodeMatch := pbsVnodeRegexp.FindStringSubmatch(n.ResourcesAvailable.Vnode)
	if len(vnodeMatch) > 1 {
		return vnodeMatch[1]
	}

	return ""
}

func (n node) getIsLicensed() int {
	if n.License == "l" {
		return 1
	}
	return 0
}

func (n node) getNodeState() int {
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

	nodeStates := strings.Split(n.State, ",")
	for _, state := range nodeStates {
		total += states[state]
	}

	return total
}

func (n node) getNodeStates() ([]string, error) {
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

func (n node) stateAvailable() (bool, error) {
	availableStates := []string{"free", "busy", "job-busy", "job-exclusive", "resv-exclusive"}
	unavailableStates := []string{"down", "maintenance", "offline", "provisioning", "stale", "state-unknown", "unresolvable", "wait-provisioning"}

	nodeStates, err := n.getNodeStates()
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

type NodeMetrics struct {
	logger   *slog.Logger
	executor commandExecutor

	hpmemDesc          *prometheus.Desc
	licenseDesc        *prometheus.Desc
	memDesc            *prometheus.Desc
	ncpusDesc          *prometheus.Desc
	nfpgasDesc         *prometheus.Desc
	ngpusDesc          *prometheus.Desc
	nodeInfoDesc       *prometheus.Desc
	nodeStateAvailable *prometheus.Desc
	stateDesc          *prometheus.Desc
}

func NewNodeMetrics(config CollectorConfig) *NodeMetrics {
	return &NodeMetrics{
		logger:   config.Logger,
		executor: &shellCommandExecutor{},
		hpmemDesc: prometheus.NewDesc(
			"pbs_node_hpmem_bytes",
			"Available huge page memory in bytes.",
			[]string{"node", "vnode"},
			nil,
		),
		licenseDesc: prometheus.NewDesc(
			"pbs_node_license_info",
			"Flag indicating if the node is licensed (1) or unlicensed (0).",
			[]string{"node", "vnode"},
			nil,
		),
		memDesc: prometheus.NewDesc(
			"pbs_node_mem_bytes",
			"Available memory in bytes.",
			[]string{"node", "vnode"},
			nil,
		),
		ncpusDesc: prometheus.NewDesc(
			"pbs_node_ncpus",
			"Available CPU cores.",
			[]string{"node", "vnode"},
			nil,
		),
		nfpgasDesc: prometheus.NewDesc(
			"pbs_node_nfpgas",
			"Available FPGAs.",
			[]string{"node", "vnode"},
			nil,
		),
		ngpusDesc: prometheus.NewDesc(
			"pbs_node_ngpus",
			"Available GPUs.",
			[]string{"node", "vnode"},
			nil,
		),
		nodeInfoDesc: prometheus.NewDesc(
			"pbs_node_info",
			"Node information.",
			[]string{"node", "vnode", "partition", "qlist"},
			nil,
		),
		nodeStateAvailable: prometheus.NewDesc(
			"pbs_node_state_available",
			"Node state availability; available (1) or unavailable (0).",
			[]string{"node", "vnode"},
			nil,
		),
		stateDesc: prometheus.NewDesc(
			"pbs_node_state_info",
			"Node state as bit field.",
			[]string{"node", "vnode"},
			nil,
		),
	}
}

type commandExecutor interface {
	execute(command []string) (stdout, stderr bytes.Buffer, err error)
}

type shellCommandExecutor struct{}

func (s *shellCommandExecutor) execute(command []string) (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout, stderr, err
}

func pbsNodeCommand(node string) []string {
	command := []string{"pbsnodes", "-H", node, "json"}
	if node == "" {
		command = []string{"pbsnodes", "-av", "-F", "json"}
	}
	return command
}

func parsePbsNodes(output []byte, nodes *nodes) error {
	err := json.Unmarshal(output, &nodes)
	if err != nil {
		return err
	}

	return nil
}

func (n *NodeMetrics) getPbsNodes() (nodes, error) {
	var nodeInfo nodes

	command := pbsNodeCommand("")
	stdout, stderr, err := n.executor.execute(command)
	if err != nil {
		return nodes{}, err
	}
	if stderr.Len() > 0 {
		return nodes{}, fmt.Errorf("pbsnodes command stderr output: %s", stderr.String())
	}

	err = parsePbsNodes(stdout.Bytes(), &nodeInfo)
	return nodeInfo, err
}

func (n *NodeMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- n.hpmemDesc
	ch <- n.licenseDesc
	ch <- n.memDesc
	ch <- n.ncpusDesc
	ch <- n.nfpgasDesc
	ch <- n.ngpusDesc
	ch <- n.nodeInfoDesc
	ch <- n.nodeStateAvailable
	ch <- n.stateDesc
}

func (n *NodeMetrics) Collect(ch chan<- prometheus.Metric) {
	nodeinfo, err := n.getPbsNodes()
	if err != nil {
		n.logger.Error("Error collecting node info from pbsnodes", "err", err)
		return
	}

	for host, v := range nodeinfo.Nodes {
		// skip natural node in multivnode host
		if v.InMultivnodeHost == 1 && host == v.ResourcesAvailable.Host {
			continue
		}

		// configuration error; skip
		vnode := v.Vnode()
		if vnode == "" && v.InMultivnodeHost == 1 {
			n.logger.Error("Vnode is empty for multi-vnode node", "host", host)
			continue
		}

		// export metrics regardless of the following values
		isAvailable, err := v.stateAvailable()
		if err != nil {
			n.logger.Warn("Error checking if node is available", "err", err)
		}

		commonLabels := []string{
			v.ResourcesAvailable.Host,
			vnode,
		}
		fullLabels := []string{
			v.ResourcesAvailable.Host,
			vnode,
			v.Partition,
			v.ResourcesAvailable.Qlist,
		}

		ch <- prometheus.MustNewConstMetric(n.hpmemDesc, prometheus.GaugeValue, float64(v.ResourcesAvailable.Hpmem), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.licenseDesc, prometheus.GaugeValue, float64(v.getIsLicensed()), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.memDesc, prometheus.GaugeValue, float64(v.ResourcesAvailable.Mem), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.ncpusDesc, prometheus.GaugeValue, float64(v.ResourcesAvailable.Ncpus), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.nfpgasDesc, prometheus.GaugeValue, float64(v.ResourcesAvailable.Nfpgas), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.ngpusDesc, prometheus.GaugeValue, float64(v.ResourcesAvailable.Ngpus), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.nodeInfoDesc, prometheus.GaugeValue, float64(1), fullLabels...)
		ch <- prometheus.MustNewConstMetric(n.nodeStateAvailable, prometheus.GaugeValue, float64(utils.BooleanToInt(isAvailable)), commonLabels...)
		ch <- prometheus.MustNewConstMetric(n.stateDesc, prometheus.GaugeValue, float64(v.getNodeState()), commonLabels...)
	}
}
