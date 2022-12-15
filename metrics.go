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

// General returns the general metrics job.
func (m *Metrics) General() *GeneralMetrics {
	return m.jobs[NameGeneral].(*GeneralMetrics)
}

// Events returns the events metrics job.
func (m *Metrics) Events() *EventMetrics {
	return m.jobs[NameEvent].(*EventMetrics)
}

// Forks returns the forks metrics job.
func (m *Metrics) Forks() *ForkMetrics {
	return m.jobs[NameFork].(*ForkMetrics)
}

// Spec returns the spec metrics job.
func (m *Metrics) Spec() *SpecMetrics {
	return m.jobs[NameSpec].(*SpecMetrics)
}

// Sync returns the sync metrics job.
func (m *Metrics) Sync() *SyncMetrics {
	return m.jobs[NameSync].(*SyncMetrics)
}

// Health returns the health metrics job.
func (m *Metrics) Health() *HealthMetrics {
	return m.jobs[NameHealth].(*HealthMetrics)
}

// Beacon returns the beacon metrics job.
func (m *Metrics) Beacon() *BeaconMetrics {
	return m.jobs[NameBeacon].(*BeaconMetrics)
}
