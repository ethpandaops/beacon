package beacon

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type Metrics struct {
	jobs map[string]MetricsJob
}

type MetricsJob interface {
	Start(ctx context.Context) error
	Name() string
}

func NewMetrics(log logrus.FieldLogger, namespace, nodeName string, beacon Node) *Metrics {
	constLabels := prometheus.Labels{
		"name": nodeName,
	}

	general := NewGeneralJob(beacon, log, namespace, constLabels)
	event := NewEventJob(beacon, log, namespace, constLabels)
	forks := NewForksJob(beacon, log, namespace, constLabels)
	spec := NewSpecJob(beacon, log, namespace, constLabels)
	sync := NewSyncJob(beacon, log, namespace, constLabels)

	jobs := map[string]MetricsJob{
		general.Name(): general,
		event.Name():   event,
		forks.Name():   forks,
		spec.Name():    spec,
		sync.Name():    sync,
	}

	m := &Metrics{jobs}

	return m
}

func (m *Metrics) Start(ctx context.Context) error {
	for _, job := range m.jobs {
		if err := job.Start(ctx); err != nil {
			return fmt.Errorf("failed to start job %s: %v", job.Name(), err)
		}
	}

	return nil
}

func (m *Metrics) General() *GeneralMetrics {
	return m.jobs[NameGeneral].(*GeneralMetrics)
}

func (m *Metrics) Events() *EventMetrics {
	return m.jobs[NameEvent].(*EventMetrics)
}

func (m *Metrics) Forks() *ForkMetrics {
	return m.jobs[NameFork].(*ForkMetrics)
}

func (m *Metrics) Spec() *SpecMetrics {
	return m.jobs[NameSpec].(*SpecMetrics)
}

func (m *Metrics) Sync() *SyncMetrics {
	return m.jobs[NameSync].(*SyncMetrics)
}

func (m *Metrics) Health() *HealthMetrics {
	return m.jobs[NameHealth].(*HealthMetrics)
}
