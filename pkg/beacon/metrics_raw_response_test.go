package beacon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewRawResponseMetricsRegistersExpectedCollectors(t *testing.T) {
	registry := prometheus.NewRegistry()
	previousRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer = previousRegisterer
	})

	nodeName := "node-a"
	metrics := NewRawResponseMetrics("test", prometheus.Labels{metricLabelNode: nodeName})
	metrics.OpenStreams.Inc()
	metrics.Leaks.Inc()

	families, err := registry.Gather()
	require.NoError(t, err)

	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		byName[family.GetName()] = family
	}

	openStreams := byName["test_raw_response_open_streams"]
	require.NotNil(t, openStreams)
	require.Equal(t, dto.MetricType_GAUGE, openStreams.GetType())
	require.Equal(t, map[string]string{
		"module":        metricsJobNameRawResponse,
		metricLabelNode: nodeName,
	}, metricLabels(openStreams.Metric[0]))

	leaks := byName["test_raw_response_leaks_total"]
	require.NotNil(t, leaks)
	require.Equal(t, dto.MetricType_COUNTER, leaks.GetType())
	require.Equal(t, map[string]string{
		"module":        metricsJobNameRawResponse,
		metricLabelNode: nodeName,
	}, metricLabels(leaks.Metric[0]))
}

func TestNodeRawResponseObserverMetrics(t *testing.T) {
	rawResponses := &RawResponseMetrics{
		OpenStreams: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_raw_response_open_streams",
			Help: "Test open streams.",
		}),
		Leaks: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_raw_response_leaks_total",
			Help: "Test leaked streams.",
		}),
	}
	n := &node{
		metrics: &Metrics{
			jobs: map[string]MetricsJob{
				metricsJobNameRawResponse: rawResponses,
			},
		},
	}

	n.RawResponseOpened()
	n.RawResponseOpened()
	require.Equal(t, float64(2), metricValue(t, rawResponses.OpenStreams))
	require.Zero(t, metricValue(t, rawResponses.Leaks))

	n.RawResponseLeaked()
	require.Equal(t, float64(2), metricValue(t, rawResponses.OpenStreams))
	require.Equal(t, float64(1), metricValue(t, rawResponses.Leaks))

	n.RawResponseClosed()
	require.Equal(t, float64(1), metricValue(t, rawResponses.OpenStreams))

	n.RawResponseClosed()
	require.Zero(t, metricValue(t, rawResponses.OpenStreams))

	disabled := &node{}
	disabled.RawResponseOpened()
	disabled.RawResponseLeaked()
	disabled.RawResponseClosed()
}

func TestNewNodeCopiesRawHTTPClientOptions(t *testing.T) {
	apiClientOptions := APIClientOptions{MaxConnsPerHost: 4}
	options := Options{APIClient: &apiClientOptions}

	n, ok := NewNode(logrus.New(), &Config{}, "", options).(*node)
	require.True(t, ok)
	apiClientOptions.MaxConnsPerHost = 8

	require.Equal(t, 4, n.options.APIClient.MaxConnsPerHost)
}

func TestInstallClientsClosesUnpublishedRawClientAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	transport := newTrackingTransport(t)
	rawHTTPClient := &http.Client{Transport: transport}
	n := &node{config: &Config{}}

	err := n.installClients(ctx, nil, &http.Client{}, rawHTTPClient)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, n.client)
	require.Nil(t, n.api)
	require.Nil(t, n.rawHTTPClient)
	require.Equal(t, int64(1), transport.closeIdleCalls.Load())
}

func TestStopClosesPublishedRawClient(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	transport := newTrackingTransport(t)
	rawHTTPClient := &http.Client{Transport: transport}
	n := &node{
		config: &Config{},
		cancel: cancel,
	}

	require.NoError(t, n.installClients(ctx, nil, &http.Client{}, rawHTTPClient))
	require.Same(t, rawHTTPClient, n.rawHTTPClient)
	require.NoError(t, n.Stop(t.Context()))
	require.Equal(t, int64(1), transport.closeIdleCalls.Load())
	require.ErrorIs(t, ctx.Err(), context.Canceled)
}

func TestNodeStopClosesOnlyIdleRawHTTPConnections(t *testing.T) {
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseStream := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		_, _ = w.Write([]byte("before"))
		flusher.Flush()

		<-release

		_, _ = w.Write([]byte("after"))
	}))
	t.Cleanup(server.Close)
	t.Cleanup(releaseStream)

	transport := newTrackingTransport(t)
	client := &http.Client{Transport: transport}

	response, err := client.Get(server.URL)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = response.Body.Close()
	})

	prefix := make([]byte, len("before"))
	_, err = io.ReadFull(response.Body, prefix)
	require.NoError(t, err)
	require.Equal(t, "before", string(prefix))

	n := &node{
		options:       &Options{},
		rawHTTPClient: client,
	}
	require.NoError(t, n.Stop(context.Background()))
	require.Equal(t, int64(1), transport.closeIdleCalls.Load())

	releaseStream()

	suffix, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "after", string(suffix))
}

type trackingTransport struct {
	*http.Transport
	closeIdleCalls atomic.Int64
}

func newTrackingTransport(t *testing.T) *trackingTransport {
	t.Helper()

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)

	return &trackingTransport{Transport: defaultTransport.Clone()}
}

func (t *trackingTransport) CloseIdleConnections() {
	t.closeIdleCalls.Add(1)
	t.Transport.CloseIdleConnections()
}

func metricValue(t *testing.T, metric prometheus.Metric) float64 {
	t.Helper()

	value := &dto.Metric{}
	require.NoError(t, metric.Write(value))

	if value.Gauge != nil {
		return value.Gauge.GetValue()
	}

	return value.Counter.GetValue()
}

func metricLabels(metric *dto.Metric) map[string]string {
	labels := make(map[string]string, len(metric.Label))
	for _, label := range metric.Label {
		labels[label.GetName()] = label.GetValue()
	}

	return labels
}
