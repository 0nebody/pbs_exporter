package collector

import (
	"context"
	"log/slog"

	"github.com/0nebody/pbs_exporter/internal/pbsnode"
	"github.com/0nebody/pbs_exporter/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
)

type NodeCollector struct {
	logger   *slog.Logger
	metrics  *NodeMetrics
	pbsNodes func(ctx context.Context) (*pbsnode.Nodes, error)
}

type NodeMetrics struct {
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

func NewNodeCollector(config CollectorConfig) *NodeCollector {
	nodeMetrics := &NodeMetrics{
		hpmemDesc: prometheus.NewDesc(
			"pbs_node_hpmem_bytes",
			"Available huge page memory in bytes.",
			defaultNodeLabels,
			nil,
		),
		licenseDesc: prometheus.NewDesc(
			"pbs_node_license_info",
			"Flag indicating if the node is licensed (1) or unlicensed (0).",
			defaultNodeLabels,
			nil,
		),
		memDesc: prometheus.NewDesc(
			"pbs_node_mem_bytes",
			"Available memory in bytes.",
			defaultNodeLabels,
			nil,
		),
		ncpusDesc: prometheus.NewDesc(
			"pbs_node_ncpus",
			"Available CPU cores.",
			defaultNodeLabels,
			nil,
		),
		nfpgasDesc: prometheus.NewDesc(
			"pbs_node_nfpgas",
			"Available FPGAs.",
			defaultNodeLabels,
			nil,
		),
		ngpusDesc: prometheus.NewDesc(
			"pbs_node_ngpus",
			"Available GPUs.",
			defaultNodeLabels,
			nil,
		),
		nodeInfoDesc: prometheus.NewDesc(
			"pbs_node_info",
			"Node information.",
			append(defaultNodeLabels, "partition", "qlist"),
			nil,
		),
		nodeStateAvailable: prometheus.NewDesc(
			"pbs_node_state_available",
			"Node state availability; available (1) or unavailable (0).",
			defaultNodeLabels,
			nil,
		),
		stateDesc: prometheus.NewDesc(
			"pbs_node_state_info",
			"Node state as bit field.",
			defaultNodeLabels,
			nil,
		),
	}

	return &NodeCollector{
		logger:   config.Logger,
		metrics:  nodeMetrics,
		pbsNodes: pbsnode.GetPbsNodes,
	}
}

func (n *NodeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- n.metrics.hpmemDesc
	ch <- n.metrics.licenseDesc
	ch <- n.metrics.memDesc
	ch <- n.metrics.ncpusDesc
	ch <- n.metrics.nfpgasDesc
	ch <- n.metrics.ngpusDesc
	ch <- n.metrics.nodeInfoDesc
	ch <- n.metrics.nodeStateAvailable
	ch <- n.metrics.stateDesc
}

func (n *NodeCollector) Collect(ctx context.Context, ch chan<- prometheus.Metric) {
	nodeinfo, err := n.pbsNodes(ctx)
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
		isAvailable, err := v.IsAvailable()
		if err != nil {
			n.logger.Warn("Error checking if node is available", "err", err)
		}

		nodeLabels := []string{v.ResourcesAvailable.Host, vnode}
		infoLabels := append(nodeLabels, v.Partition, v.ResourcesAvailable.Qlist)

		ch <- prometheus.MustNewConstMetric(
			n.metrics.hpmemDesc,
			prometheus.GaugeValue,
			float64(v.ResourcesAvailable.Hpmem),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.licenseDesc,
			prometheus.GaugeValue,
			float64(v.IsLicensed()),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.memDesc,
			prometheus.GaugeValue,
			float64(v.ResourcesAvailable.Mem),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.ncpusDesc,
			prometheus.GaugeValue,
			float64(v.ResourcesAvailable.Ncpus),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.nfpgasDesc,
			prometheus.GaugeValue,
			float64(v.ResourcesAvailable.Nfpgas),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.ngpusDesc,
			prometheus.GaugeValue,
			float64(v.ResourcesAvailable.Ngpus),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.nodeInfoDesc,
			prometheus.GaugeValue,
			float64(1),
			infoLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.nodeStateAvailable,
			prometheus.GaugeValue,
			float64(utils.BooleanToInt(isAvailable)),
			nodeLabels...,
		)
		ch <- prometheus.MustNewConstMetric(
			n.metrics.stateDesc,
			prometheus.GaugeValue,
			float64(v.NodeState()),
			nodeLabels...,
		)
	}
}
