package beacon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	beaconapi "github.com/ethpandaops/beacon/pkg/beacon/api"
	ehttp "github.com/ethpandaops/go-eth2-client/http"
	"github.com/sirupsen/logrus"
)

func TestNodeOpenRawEndpointsThroughBootstrapClient(t *testing.T) {
	const (
		stateBody    = "state-response"
		blockBody    = "block-response"
		envelopeBody = "envelope-response"
	)

	responses := map[string]string{
		"/eth/v2/debug/beacon/states/finalized":                 stateBody,
		"/eth/v2/beacon/blocks/head":                            blockBody,
		"/eth/v1/beacon/execution_payload_envelopes/0x12345678": envelopeBody,
	}
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, ok := responses[req.URL.Path]
		if !ok {
			http.NotFound(w, req)

			return
		}
		if got := req.Header.Get("Accept"); got != "application/octet-stream" {
			t.Errorf("Accept = %q", got)
		}
		if got := req.Header.Get("X-Test-Header"); got != "present" {
			t.Errorf("X-Test-Header = %q", got)
		}

		requests.Add(1)
		w.Header().Set("Content-Type", "application/octet-stream;charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Eth-Consensus-Version", "gloas")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	log := logrus.New()
	log.SetOutput(io.Discard)
	publicNode := NewNode(log, &Config{
		Addr: server.URL,
		Headers: map[string]string{
			"X-Test-Header": "present",
		},
	}, "", Options{
		GoEth2ClientParams: []ehttp.Parameter{ehttp.WithAllowDelayedStart(true)},
	})
	implementation, ok := publicNode.(*node)
	if !ok {
		t.Fatalf("node type = %T", publicNode)
	}

	if err := implementation.ensureClients(t.Context()); err != nil {
		t.Fatalf("ensure clients: %v", err)
	}
	t.Cleanup(func() {
		if err := implementation.Stop(t.Context()); err != nil {
			t.Errorf("stop node: %v", err)
		}
	})

	tests := []struct {
		name string
		body string
		open func(context.Context, Node) (*beaconapi.RawResponse, error)
	}{
		{
			name: "beacon state",
			body: stateBody,
			open: func(ctx context.Context, node Node) (*beaconapi.RawResponse, error) {
				return node.OpenRawBeaconState(ctx, "finalized", "application/octet-stream")
			},
		},
		{
			name: "block",
			body: blockBody,
			open: func(ctx context.Context, node Node) (*beaconapi.RawResponse, error) {
				return node.OpenRawBlock(ctx, "head", "application/octet-stream")
			},
		},
		{
			name: "execution payload envelope",
			body: envelopeBody,
			open: func(ctx context.Context, node Node) (*beaconapi.RawResponse, error) {
				return node.OpenRawExecutionPayloadEnvelope(ctx, "0x12345678", "application/octet-stream")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := test.open(t.Context(), publicNode)
			if err != nil {
				t.Fatalf("open response: %v", err)
			}

			body, readErr := io.ReadAll(response)
			closeErr := response.Close()
			if readErr != nil {
				t.Fatalf("read response: %v", readErr)
			}
			if closeErr != nil {
				t.Fatalf("close response: %v", closeErr)
			}
			if string(body) != test.body {
				t.Fatalf("body = %q, want %q", body, test.body)
			}
			if response.ContentLength != int64(len(test.body)) {
				t.Fatalf("ContentLength = %d", response.ContentLength)
			}
			if got := response.Header.Get("Eth-Consensus-Version"); got != "gloas" {
				t.Fatalf("Eth-Consensus-Version = %q", got)
			}
		})
	}

	if got := requests.Load(); got != int32(len(tests)) {
		t.Fatalf("requests = %d, want %d", got, len(tests))
	}
}
