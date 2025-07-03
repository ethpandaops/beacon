package beacon

import (
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestEventMetrics_ConcurrentAccess tests that EventMetrics can handle concurrent calls to HandleEvent and
// tick without race conditions.
//
//nolint:promlinter // its a test.
func TestEventMetrics_ConcurrentAccess(t *testing.T) {
	// Unregister any existing metrics to avoid conflicts.
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create a simple EventMetrics instance without a full node.
	em := &EventMetrics{
		log:   log,
		crons: nil, // We won't use crons in the test.
		Count: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "test",
				Name:      "event_count",
				Help:      "Test event count",
			},
			[]string{"event"},
		),
		TimeSinceLastEvent: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "test",
				Name:      "time_since_last_event",
				Help:      "Test time since last event",
			},
		),
		LastEventTime: time.Now(),
	}

	prometheus.MustRegister(&em.Count)
	prometheus.MustRegister(em.TimeSinceLastEvent)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	const numGoroutines = 50
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup

	// Start goroutines that call HandleEvent
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < eventsPerGoroutine; j++ {
				event := &v1.Event{
					Topic: "test_event",
					Data:  []byte("test data"),
				}

				_ = em.HandleEvent(ctx, event)
			}
		}()
	}

	// Start goroutines that call tick to read LastEventTime
	for i := 0; i < 10; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := 0; j < 1000; j++ {
				em.tick(ctx)

				time.Sleep(time.Microsecond)
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify the counter was incremented correctly
	expectedCount := float64(numGoroutines * eventsPerGoroutine)

	// Get the metric value
	metricDTO := &dto.Metric{}

	metric, err := em.Count.GetMetricWithLabelValues("test_event")
	require.NoError(t, err)

	err = metric.Write(metricDTO)
	require.NoError(t, err)

	actualCount := metricDTO.Counter.GetValue()
	require.Equal(t, expectedCount, actualCount, "Event count mismatch")
}

// TestEventMetrics_LastEventTime specifically tests the LastEventTime field for race conditions between concurrent
// readers and writers.
//
//nolint:promlinter // its a test.
func TestEventMetrics_LastEventTime(t *testing.T) {
	// Unregister any existing metrics to avoid conflicts.
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	em := &EventMetrics{
		log: log,
		Count: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "test_race",
				Name:      "event_count",
				Help:      "Test event count",
			},
			[]string{"event"},
		),
		TimeSinceLastEvent: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "test_race",
				Name:      "time_since_last_event",
				Help:      "Test time since last event",
			},
		),
		LastEventTime: time.Now(),
	}

	ctx := context.Background()

	// Create a high contention scenario.
	done := make(chan bool)

	// Writer goroutine - rapidly updates LastEventTime.
	go func() {
		for i := 0; i < 10000; i++ {
			event := &v1.Event{
				Topic: "race_test",
				Data:  []byte("data"),
			}

			_ = em.HandleEvent(ctx, event)
		}

		done <- true
	}()

	// Reader goroutine - rapidly reads LastEventTime.
	go func() {
		for i := 0; i < 10000; i++ {
			em.tick(ctx)
		}

		done <- true
	}()

	// Wait for both to complete.
	<-done
	<-done
}
