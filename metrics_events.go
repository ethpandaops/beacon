package beacon

import (
	"context"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// EventMetrics reports event counts.
type EventMetrics struct {
	log                logrus.FieldLogger
	Count              prometheus.CounterVec
	TimeSinceLastEvent prometheus.Gauge

	beacon Node

	LastEventTime time.Time
}

const (
	NameEvent = "event"
)

// NewEvent creates a new Event instance.
func NewEventJob(bc Node, log logrus.FieldLogger, namespace string, constLabels map[string]string) *EventMetrics {
	constLabels["module"] = NameEvent
	namespace += "_event"

	e := &EventMetrics{
		log:    log,
		beacon: bc,
		Count: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace:   namespace,
				Name:        "count",
				Help:        "The count of beacon events.",
				ConstLabels: constLabels,
			},
			[]string{
				"event",
			},
		),
		TimeSinceLastEvent: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace:   namespace,
				Name:        "time_since_last_subscription_event_ms",
				Help:        "The amount of time since the last subscription event (in milliseconds).",
				ConstLabels: constLabels,
			},
		),
		LastEventTime: time.Now(),
	}

	prometheus.MustRegister(&e.Count)
	prometheus.MustRegister(e.TimeSinceLastEvent)

	return e
}

// Name returns the name of the job.
func (e *EventMetrics) Name() string {
	return NameEvent
}

// Start starts the job.
func (e *EventMetrics) Start(ctx context.Context) error {
	e.beacon.OnEvent(ctx, e.HandleEvent)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second * 1):
				e.tick(ctx)
			}
		}
	}()

	return nil
}

//nolint:unparam // ctx will probably be used in the future
func (e *EventMetrics) tick(ctx context.Context) {
	e.TimeSinceLastEvent.Set(float64(time.Since(e.LastEventTime).Milliseconds()))
}

// HandleEvent handles all beacon events
func (e *EventMetrics) HandleEvent(ctx context.Context, event *v1.Event) error {
	e.Count.WithLabelValues(event.Topic).Inc()
	e.LastEventTime = time.Now()
	e.TimeSinceLastEvent.Set(0)

	return nil
}
