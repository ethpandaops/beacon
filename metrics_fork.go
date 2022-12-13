package beacon

import (
	"context"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ForkMetrics reports the state of any forks (previous, active or upcoming).
type ForkMetrics struct {
	Epochs    prometheus.GaugeVec
	Activated prometheus.GaugeVec
	Current   prometheus.GaugeVec
	beacon    Node
	log       logrus.FieldLogger
}

const (
	NameFork = "fork"
)

// NewForksJob returns a new Forks instance.
func NewForksJob(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *ForkMetrics {
	constLabels["module"] = NameFork

	namespace += "_fork"

	f := &ForkMetrics{
		beacon: beac,
		log:    log,
		Epochs: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "epoch",
				Help:        "The epoch for the fork.",
				ConstLabels: constLabels,
			},
			[]string{
				"fork",
			},
		),
		Activated: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "activated",
				Help:        "The activation status of the fork (1 for activated).",
				ConstLabels: constLabels,
			},
			[]string{
				"fork",
			},
		),
		Current: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "current",
				Help:        "The current fork.",
				ConstLabels: constLabels,
			},
			[]string{
				"fork",
			},
		),
	}

	prometheus.MustRegister(f.Epochs)
	prometheus.MustRegister(f.Activated)
	prometheus.MustRegister(f.Current)

	return f
}

func (f *ForkMetrics) Name() string {
	return NameFork
}

func (f *ForkMetrics) Start(ctx context.Context) error {
	// TODO(sam.calder-mason): Update this to use the wall clock instead.
	f.beacon.OnBlock(ctx, func(ctx context.Context, event *v1.BlockEvent) error {
		return f.calculateCurrent(ctx, event.Slot)
	})

	return nil
}

func (f *ForkMetrics) calculateCurrent(ctx context.Context, slot phase0.Slot) error {
	spec, err := f.beacon.GetSpec(ctx)
	if err != nil {
		return err
	}

	slotsPerEpoch := spec.SlotsPerEpoch

	f.Activated.Reset()
	f.Epochs.Reset()

	for _, fork := range spec.ForkEpochs {
		f.Epochs.WithLabelValues(fork.Name).Set(float64(fork.Epoch))

		if fork.Active(slot, slotsPerEpoch) {
			f.Activated.WithLabelValues(fork.Name).Set(1)
		} else {
			f.Activated.WithLabelValues(fork.Name).Set(0)
		}
	}

	current, err := spec.ForkEpochs.CurrentFork(slot, slotsPerEpoch)
	if err != nil {
		f.log.WithError(err).Error("Failed to set current fork")
	} else {
		f.Current.Reset()

		f.Current.WithLabelValues(current.Name).Set(1)
	}

	return nil
}
