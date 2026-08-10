//go:build acceptance && acceptance_http2 && linux

package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAcceptanceConcurrentHTTP2BeaconStateStreams(t *testing.T) {
	const (
		transportHeapBudget uint64 = 128 << 20
		transportRSSBudget  uint64 = 128 << 20
	)

	pinGCSettings(t)

	ctx, cancel := context.WithTimeout(t.Context(), acceptanceTimeout)
	defer cancel()

	var requests atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ProtoMajor != 2 {
			t.Errorf("protocol = %s", req.Proto)
		}
		if req.URL.Path != beaconStatePath {
			http.NotFound(w, req)

			return
		}
		if got := req.Header.Get("Accept"); got != rawContentType {
			t.Errorf("Accept = %q", got)
		}

		requests.Add(1)
		w.Header().Set("Content-Type", rawContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(beaconStatePayloadBytes, 10))
		w.Header().Set("Eth-Consensus-Version", "gloas")
		w.WriteHeader(http.StatusOK)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush response headers: %v", err)

			return
		}

		_, _ = io.Copy(w, &generatedBody{remaining: beaconStatePayloadBytes})
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)

	httpClient := server.Client()
	httpClient.Timeout = 0
	t.Cleanup(httpClient.CloseIdleConnections)

	observer := &countingObserver{}
	log := logrus.New()
	log.SetOutput(io.Discard)
	client := NewConsensusClient(log, server.URL, httpClient, httpClient, nil, observer)

	report := measureConcurrentStreams(ctx, t, client, observer, streamConcurrency, beaconStatePayloadBytes)
	report.log(t, "50 concurrent HTTP/2 beacon state streams")

	if got := requests.Load(); got != int64(streamConcurrency) {
		t.Errorf("requests = %d, want %d", got, streamConcurrency)
	}
	if report.peakDelta() > transportHeapBudget {
		t.Errorf("peak heap delta %s exceeds budget %s", mib(report.peakDelta()), mib(transportHeapBudget))
	}
	if report.peakRSSDelta() > transportRSSBudget {
		t.Errorf("peak RSS delta %s exceeds budget %s", mib(report.peakRSSDelta()), mib(transportRSSBudget))
	}
}
