package beacon

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// SyncMetrics reports metrics on the sync status of the node.
type SyncMetrics struct {
	beacon               Node
	log                  logrus.FieldLogger
	Percentage           prometheus.Gauge
	EstimatedHighestSlot prometheus.Gauge
	HeadSlot             prometheus.Gauge
	Distance             prometheus.Gauge
	IsSyncing            prometheus.Gauge
	IsOptimistic         prometheus.Gauge
}

const (
	metricsJobNameSync = "sync"
)

// NewSyncMetrics returns a new Sync metrics instance.
func NewSyncMetrics(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *SyncMetrics {
	constLabels["module"] = metricsJobNameSync

	namespace += "_sync"

	s := &SyncMetrics{
		beacon: beac,
		log:    log,
		Percentage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "percentage",
				Help:        "How synced the node is with the network (0-100%).",
				ConstLabels: constLabels,
			},
		),
		EstimatedHighestSlot: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "estimated_highest_slot",
				Help:        "The estimated highest slot of the network.",
				ConstLabels: constLabels,
			},
		),
		HeadSlot: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "head_slot",
				Help:        "The current slot of the node.",
				ConstLabels: constLabels,
			},
		),
		Distance: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "distance",
				Help:        "The sync distance of the node.",
				ConstLabels: constLabels,
			},
		),
		IsSyncing: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "is_syncing",
				Help:        "1 if the node is in syncing state.",
				ConstLabels: constLabels,
			},
		),
		IsOptimistic: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "is_optimistic",
				Help:        "1 if the node is optimistically syncing.",
				ConstLabels: constLabels,
			},
		),
	}

	prometheus.MustRegister(s.Percentage)
	prometheus.MustRegister(s.EstimatedHighestSlot)
	prometheus.MustRegister(s.HeadSlot)
	prometheus.MustRegister(s.Distance)
	prometheus.MustRegister(s.IsSyncing)
	prometheus.MustRegister(s.IsOptimistic)

	return s
}

// Name returns the name of the job.
func (s *SyncMetrics) Name() string {
	return metricsJobNameSync
}

// Start starts the job.
func (s *SyncMetrics) Start(ctx context.Context) error {
	s.beacon.OnSyncStatus(ctx, func(ctx context.Context, event *SyncStatusEvent) error {
		status := event.State

		s.Distance.Set(float64(status.SyncDistance))
		s.HeadSlot.Set(float64(status.HeadSlot))
		s.observeSyncIsSyncing(status.IsSyncing)
		s.observeSyncIsOptimistic(status.IsOptimistic)

		estimatedHighestHeadSlot := status.SyncDistance + status.HeadSlot
		s.EstimatedHighestSlot.Set(float64(estimatedHighestHeadSlot))

		percent := (float64(status.HeadSlot) / float64(estimatedHighestHeadSlot) * 100)
		if !status.IsSyncing {
			percent = 100
		}

		s.Percentage.Set(percent)

		return nil
	})

	return nil
}

// Stop stops the job.
func (s *SyncMetrics) Stop() error {
	return nil
}

func observeBoolGauge(dst prometheus.Gauge, b bool) {
	if b {
		dst.Set(1)

		return
	}

	dst.Set(0)
}

func (s *SyncMetrics) observeSyncIsSyncing(syncing bool) {
	observeBoolGauge(s.IsSyncing, syncing)
}

func (s *SyncMetrics) observeSyncIsOptimistic(optimistic bool) {
	observeBoolGauge(s.IsOptimistic, optimistic)
}
