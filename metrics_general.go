package beacon

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/samcm/beacon/api/types"
	"github.com/sirupsen/logrus"
)

// GeneralMetrics reports general information about the node.
type GeneralMetrics struct {
	beacon      Node
	log         logrus.FieldLogger
	NodeVersion prometheus.GaugeVec
	ClientName  prometheus.GaugeVec
	Peers       prometheus.GaugeVec
}

const (
	NameGeneral = "general"
)

// NewGeneral creates a new General instance.
func NewGeneralJob(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *GeneralMetrics {
	constLabels["module"] = NameGeneral

	g := &GeneralMetrics{
		beacon: beac,
		log:    log,
		NodeVersion: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "node_version",
				Help:        "The version of the running beacon node.",
				ConstLabels: constLabels,
			},
			[]string{
				"version",
			},
		),
		Peers: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "peers",
				Help:        "The count of peers connected to beacon node.",
				ConstLabels: constLabels,
			},
			[]string{
				"state",
				"direction",
			},
		),
	}

	prometheus.MustRegister(&g.NodeVersion)
	prometheus.MustRegister(&g.Peers)

	return g
}

// Name returns the name of the job.
func (g *GeneralMetrics) Name() string {
	return NameGeneral
}

// Start starts the job.
func (g *GeneralMetrics) Start(ctx context.Context) error {
	g.beacon.OnNodeVersionUpdated(ctx, func(ctx context.Context, event *NodeVersionUpdatedEvent) error {
		g.observeNodeVersion(ctx, event.Version)

		return nil
	})

	g.beacon.OnPeersUpdated(ctx, func(ctx context.Context, event *PeersUpdatedEvent) error {
		g.Peers.Reset()

		for _, state := range types.PeerStates {
			for _, direction := range types.PeerDirections {
				g.Peers.WithLabelValues(state, direction).Set(float64(len(event.Peers.ByStateAndDirection(state, direction))))
			}
		}

		return nil
	})

	if err := g.initialFetch(ctx); err != nil {
		return nil
	}

	return nil
}

func (g *GeneralMetrics) initialFetch(ctx context.Context) error {
	version, err := g.beacon.NodeVersion()
	if err != nil {
		return err
	}

	g.observeNodeVersion(ctx, version)

	return nil
}

func (g *GeneralMetrics) observeNodeVersion(ctx context.Context, version string) {
	g.NodeVersion.Reset()
	g.NodeVersion.WithLabelValues(version).Set(1)
}
