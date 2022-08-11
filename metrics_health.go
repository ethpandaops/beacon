package beacon

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// HealthMetrics reports metrics on the health status of the node.
type HealthMetrics struct {
	beacon            Node
	log               logrus.FieldLogger
	CheckResultsTotal *prometheus.CounterVec
	Up                prometheus.Gauge
}

const (
	NameHealth = "health"
)

// NewHealthMetrics returns a new HealthMetrics instance.
func NewHealthMetrics(beac Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *HealthMetrics {
	constLabels["module"] = NameHealth

	namespace += "_health"

	h := &HealthMetrics{
		beacon: beac,
		log:    log,
		CheckResultsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "check_results_total",
				Help:        "Total of health checks results.",
				ConstLabels: constLabels,
			},
			[]string{"result"},
		),
		Up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "up",
				Help:        "Whether the node is up or not.",
				ConstLabels: constLabels,
			},
		),
	}

	prometheus.MustRegister(h.CheckResultsTotal)
	prometheus.MustRegister(h.Up)

	return h
}

func (h *HealthMetrics) Name() string {
	return NameSync
}

func (h *HealthMetrics) Start(ctx context.Context) error {
	h.beacon.OnHealthCheckFailed(ctx, func(ctx context.Context, event *HealthCheckFailedEvent) error {
		h.ObserveFailure()
		h.checkUp(ctx)

		return nil
	})

	h.beacon.OnHealthCheckSucceeded(ctx, func(ctx context.Context, event *HealthCheckSucceededEvent) error {
		h.ObserveSuccess()
		h.checkUp(ctx)

		return nil
	})

	return nil
}

func (h *HealthMetrics) ObserveFailure() {
	h.CheckResultsTotal.WithLabelValues("fai").Inc()
}

func (h *HealthMetrics) ObserveSuccess() {
	h.CheckResultsTotal.WithLabelValues("success").Inc()
}

func (h *HealthMetrics) checkUp(ctx context.Context) {
	status := h.beacon.GetStatus(ctx)

	if status.Healthy() {
		h.Up.Set(1)
	} else {
		h.Up.Set(0)
	}
}
