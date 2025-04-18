package beacon

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Metrics contains all the metrics jobs.
type Metrics struct {
	jobs map[string]MetricsJob
	log  logrus.FieldLogger
}

// MetricsJob is a job that reports metrics.
type MetricsJob interface {
	Start(ctx context.Context) error
	Stop() error
	Name() string
}

// NewMetrics returns a new Metrics instance.
func NewMetrics(log logrus.FieldLogger, namespace, nodeName string, beacon Node) *Metrics {
	constLabels := prometheus.Labels{
		"node": nodeName,
	}

	beac := NewBeaconMetrics(beacon, log, namespace, constLabels)
	general := NewGeneralJob(beacon, log, namespace, constLabels)
	event := NewEventJob(beacon, log, namespace, constLabels)
	forks := NewForksJob(beacon, log, namespace, constLabels)
	spec := NewSpecJob(beacon, log, namespace, constLabels)
	sync := NewSyncMetrics(beacon, log, namespace, constLabels)
	health := NewHealthMetrics(beacon, log, namespace, constLabels)

	jobs := map[string]MetricsJob{
		sync.Name():    sync,
		general.Name(): general,
		event.Name():   event,
		forks.Name():   forks,
		spec.Name():    spec,
		health.Name():  health,
		beac.Name():    beac,
	}

	m := &Metrics{
		jobs,
		log,
	}

	return m
}

// Start starts all the jobs.
func (m *Metrics) Start(ctx context.Context) error {
	for _, job := range m.jobs {
		if err := job.Start(ctx); err != nil {
			return fmt.Errorf("failed to start job %s: %v", job.Name(), err)
		}
	}

	return nil
}

// Stop stops all the metrics jobs.
func (m *Metrics) Stop() error {
	for _, job := range m.jobs {
		if err := job.Stop(); err != nil {
			return fmt.Errorf("failed to stop job %s: %v", job.Name(), err)
		}
	}

	return nil
}

// General returns the general metrics job.
func (m *Metrics) General() *GeneralMetrics {
	return m.jobs[metricsJobNameGeneral].(*GeneralMetrics) //nolint:errcheck // existing.
}

// Events returns the events metrics job.
func (m *Metrics) Events() *EventMetrics {
	return m.jobs[metricsJobNameEvent].(*EventMetrics) //nolint:errcheck // existing.
}

// Forks returns the forks metrics job.
func (m *Metrics) Forks() *ForkMetrics {
	return m.jobs[metricsJobNameFork].(*ForkMetrics) //nolint:errcheck // existing.
}

// Spec returns the spec metrics job.
func (m *Metrics) Spec() *SpecMetrics {
	return m.jobs[metricsJobNameSpec].(*SpecMetrics) //nolint:errcheck // existing.
}

// Sync returns the sync metrics job.
func (m *Metrics) Sync() *SyncMetrics {
	return m.jobs[metricsJobNameSync].(*SyncMetrics) //nolint:errcheck // existing.
}

// Health returns the health metrics job.
func (m *Metrics) Health() *HealthMetrics {
	return m.jobs[metricsJobNameHealth].(*HealthMetrics) //nolint:errcheck // existing.
}

// Beacon returns the beacon metrics job.
func (m *Metrics) Beacon() *BeaconMetrics {
	return m.jobs[metricsJobNameBeacon].(*BeaconMetrics) //nolint:errcheck // existing.
}
