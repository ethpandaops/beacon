package beacon

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

const metricsJobNameRawResponse = "raw_response"

// RawResponseMetrics reports the lifecycle of streaming raw responses.
type RawResponseMetrics struct {
	OpenStreams prometheus.Gauge
	Leaks       prometheus.Counter
}

// NewRawResponseMetrics creates and registers raw response metrics.
func NewRawResponseMetrics(namespace string, constLabels map[string]string) *RawResponseMetrics {
	labels := make(prometheus.Labels, len(constLabels)+1)
	for name, value := range constLabels {
		labels[name] = value
	}

	labels["module"] = metricsJobNameRawResponse

	metrics := &RawResponseMetrics{
		OpenStreams: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Name:        "raw_response_open_streams",
			Help:        "Number of raw response streams currently open.",
			ConstLabels: labels,
		}),
		Leaks: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Name:        "raw_response_leaks_total",
			Help:        "Total number of raw response streams closed by cleanup.",
			ConstLabels: labels,
		}),
	}

	prometheus.MustRegister(metrics.OpenStreams)
	prometheus.MustRegister(metrics.Leaks)

	return metrics
}

// Name returns the name of the job.
func (m *RawResponseMetrics) Name() string {
	return metricsJobNameRawResponse
}

// Start starts the job.
func (m *RawResponseMetrics) Start(context.Context) error {
	return nil
}

// Stop stops the job.
func (m *RawResponseMetrics) Stop() error {
	return nil
}
