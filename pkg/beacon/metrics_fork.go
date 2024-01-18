package beacon

import (
	"context"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
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
	metricsJobNameFork = "fork"
)

// NewForksJob returns a new Forks instance.
func NewForksJob(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *ForkMetrics {
	constLabels["module"] = metricsJobNameFork

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

// Name returns the name of the job.
func (f *ForkMetrics) Name() string {
	return metricsJobNameFork
}

// Start starts the job.
func (f *ForkMetrics) Start(ctx context.Context) error {
	// TODO(sam.calder-mason): Update this to use the wall clock instead.
	f.beacon.Wallclock().OnEpochChanged(func(epoch ethwallclock.Epoch) {
		f.calculateCurrent(ctx)
	})

	return nil
}

// Stop stops the job.
func (f *ForkMetrics) Stop() error {
	return nil
}

func (f *ForkMetrics) calculateCurrent(ctx context.Context) error {
	slot := f.beacon.Wallclock().Slots().Current()

	spec, err := f.beacon.Spec()
	if err != nil {
		return err
	}

	slotsPerEpoch := spec.SlotsPerEpoch

	f.Activated.Reset()
	f.Epochs.Reset()

	for _, fork := range spec.ForkEpochs {
		f.Epochs.WithLabelValues(fork.Name).Set(float64(fork.Epoch))

		if fork.Active(phase0.Slot(slot.Number()), slotsPerEpoch) {
			f.Activated.WithLabelValues(fork.Name).Set(1)
		} else {
			f.Activated.WithLabelValues(fork.Name).Set(0)
		}
	}

	current, err := spec.ForkEpochs.CurrentFork(phase0.Slot(slot.Number()), slotsPerEpoch)
	if err != nil {
		f.log.WithError(err).Error("Failed to set current fork")
	} else {
		f.Current.Reset()

		f.Current.WithLabelValues(current.Name).Set(1)
	}

	return nil
}
