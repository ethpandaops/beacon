package beacon

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
)

const testAcceptEncodingGzip = "gzip"

type optionsRoundTripFunc func(*http.Request) (*http.Response, error)

func (f optionsRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestDefaultAPIClientOptions(t *testing.T) {
	opts := DefaultAPIClientOptions()
	client := opts.httpClient()
	transport := opts.transport()

	if client.Timeout != 10*time.Minute {
		t.Fatalf("timeout = %s, want 10m", client.Timeout)
	}

	if transport.DialContext == nil {
		t.Fatal("DialContext is nil")
	}

	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Fatalf("TLSHandshakeTimeout = %s, want 10s", transport.TLSHandshakeTimeout)
	}

	if transport.MaxIdleConnsPerHost != 0 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 0", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 32 {
		t.Fatalf("MaxConnsPerHost = %d, want 32", transport.MaxConnsPerHost)
	}

	limited, ok := client.Transport.(*concurrencyLimitedRoundTripper)
	if !ok {
		t.Fatalf("Transport type = %T", client.Transport)
	}
	if cap(limited.slots) != 32 {
		t.Fatalf("MaxConcurrentRequests = %d, want 32", cap(limited.slots))
	}
}

func TestSetAPIClientTimeout(t *testing.T) {
	tests := []struct {
		name string
		set  time.Duration
		want time.Duration
	}{
		{name: "custom", set: 30 * time.Second, want: 30 * time.Second},
		{name: "zero removes the bound so the context governs the read", set: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := (&Options{}).SetAPIClientTimeout(tt.set)

			if got := opts.APIClient.httpClient().Timeout; got != tt.want {
				t.Fatalf("timeout = %s, want %s", got, tt.want)
			}

			if got := opts.APIClient.MaxConnsPerHost; got != 32 {
				t.Fatalf("MaxConnsPerHost = %d, want default 32", got)
			}
			if got := opts.APIClient.MaxConcurrentRequests; got != 32 {
				t.Fatalf("MaxConcurrentRequests = %d, want default 32", got)
			}
		})
	}
}

func TestSetAPIClientTimeoutPreservesExplicitOptions(t *testing.T) {
	opts := (&Options{}).
		SetAPIClientOptions(APIClientOptions{}).
		SetAPIClientTimeout(time.Second)

	if got := opts.APIClient.Timeout.Duration; got != time.Second {
		t.Fatalf("Timeout = %s, want 1s", got)
	}

	if got := opts.APIClient.MaxConnsPerHost; got != 0 {
		t.Fatalf("MaxConnsPerHost = %d, want explicit zero", got)
	}
	if got := opts.APIClient.MaxConcurrentRequests; got != 0 {
		t.Fatalf("MaxConcurrentRequests = %d, want explicit zero", got)
	}
}

func TestTransportKeepsHTTP2(t *testing.T) {
	transport := DefaultAPIClientOptions().transport()

	if !transport.ForceAttemptHTTP2 {
		t.Fatal("ForceAttemptHTTP2 is false; a custom DialContext disables HTTP/2 without it")
	}
}

func TestAPIClientAcceptEncoding(t *testing.T) {
	const encodedBody = "encoded"

	acceptEncoding := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		acceptEncoding <- req.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Encoding", testAcceptEncodingGzip)
		_, _ = io.WriteString(w, encodedBody)
	}))
	t.Cleanup(server.Close)

	client := (APIClientOptions{AcceptEncoding: testAcceptEncodingGzip}).httpClient()
	t.Cleanup(client.CloseIdleConnections)

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("get response: %v", err)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		t.Fatalf("read response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close response: %v", closeErr)
	}

	if got := <-acceptEncoding; got != testAcceptEncodingGzip {
		t.Fatalf("Accept-Encoding = %q", got)
	}
	if response.Uncompressed {
		t.Fatal("Uncompressed = true")
	}
	if got := response.Header.Get("Content-Encoding"); got != testAcceptEncodingGzip {
		t.Fatalf("Content-Encoding = %q", got)
	}
	if string(data) != encodedBody {
		t.Fatalf("body = %q", data)
	}
}

func TestSetAPIClientAcceptEncodingPreservesDefaults(t *testing.T) {
	opts := (&Options{}).SetAPIClientAcceptEncoding(testAcceptEncodingGzip)

	if got := opts.APIClient.AcceptEncoding; got != testAcceptEncodingGzip {
		t.Fatalf("AcceptEncoding = %q", got)
	}
	if got := opts.APIClient.Timeout.Duration; got != 10*time.Minute {
		t.Fatalf("Timeout = %s, want default 10m", got)
	}
	if got := opts.APIClient.MaxConnsPerHost; got != 32 {
		t.Fatalf("MaxConnsPerHost = %d, want default 32", got)
	}
	if got := opts.APIClient.MaxConcurrentRequests; got != 32 {
		t.Fatalf("MaxConcurrentRequests = %d, want default 32", got)
	}
}

func TestSetAPIClientAcceptEncodingPreservesExplicitOptions(t *testing.T) {
	opts := (&Options{}).
		SetAPIClientOptions(APIClientOptions{}).
		SetAPIClientAcceptEncoding(testAcceptEncodingGzip)

	if got := opts.APIClient.AcceptEncoding; got != testAcceptEncodingGzip {
		t.Fatalf("AcceptEncoding = %q", got)
	}
	if got := opts.APIClient.Timeout.Duration; got != 0 {
		t.Fatalf("Timeout = %s, want explicit zero", got)
	}
	if got := opts.APIClient.MaxConnsPerHost; got != 0 {
		t.Fatalf("MaxConnsPerHost = %d, want explicit zero", got)
	}
	if got := opts.APIClient.MaxConcurrentRequests; got != 0 {
		t.Fatalf("MaxConcurrentRequests = %d, want explicit zero", got)
	}
}

func TestTransportIsNotShared(t *testing.T) {
	opts := DefaultAPIClientOptions()

	if opts.httpClient().Transport == nil {
		t.Fatal("transport is nil, so the client would share http.DefaultTransport")
	}

	first, second := opts.transport(), opts.transport()
	if first == second {
		t.Fatal("transport() returned the same instance twice")
	}
}

func TestResponseHeaderTimeoutDoesNotBoundBodyRead(t *testing.T) {
	opts := DefaultAPIClientOptions()
	opts.ResponseHeaderTimeout = human.Duration{Duration: 15 * time.Second}
	opts.Timeout = human.Duration{}

	client := opts.httpClient()
	transport := opts.transport()

	if client.Timeout != 0 {
		t.Fatalf("client timeout = %s, want 0 so the body read is unbounded", client.Timeout)
	}

	if transport.ResponseHeaderTimeout != 15*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %s, want 15s", transport.ResponseHeaderTimeout)
	}
}

func TestAPIClientOptionsFieldPropagation(t *testing.T) {
	opts := APIClientOptions{
		Timeout:               human.Duration{Duration: time.Second},
		ResponseHeaderTimeout: human.Duration{Duration: 2 * time.Second},
		TLSHandshakeTimeout:   human.Duration{Duration: 3 * time.Second},
		MaxIdleConnsPerHost:   4,
		MaxConnsPerHost:       5,
		MaxConcurrentRequests: 6,
	}
	client := opts.httpClient()
	limited, ok := client.Transport.(*concurrencyLimitedRoundTripper)
	if !ok {
		t.Fatalf("Transport type = %T", client.Transport)
	}
	transport, ok := limited.next.(*http.Transport)
	if !ok {
		t.Fatalf("nested Transport type = %T", limited.next)
	}

	if client.Timeout != time.Second {
		t.Fatalf("Timeout = %s, want 1s", client.Timeout)
	}

	if transport.ResponseHeaderTimeout != 2*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %s, want 2s", transport.ResponseHeaderTimeout)
	}

	if transport.TLSHandshakeTimeout != 3*time.Second {
		t.Fatalf("TLSHandshakeTimeout = %s, want 3s", transport.TLSHandshakeTimeout)
	}

	if transport.MaxIdleConnsPerHost != 4 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 4", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 5 {
		t.Fatalf("MaxConnsPerHost = %d, want 5", transport.MaxConnsPerHost)
	}

	if got := cap(limited.slots); got != 6 {
		t.Fatalf("MaxConcurrentRequests = %d, want 6", got)
	}
}

func TestDefaultOptionsStoresAPIClientDefaults(t *testing.T) {
	opts := DefaultOptions()

	if opts.APIClient == nil {
		t.Fatal("APIClient is nil")
	}

	if *opts.APIClient != DefaultAPIClientOptions() {
		t.Fatalf("APIClient = %+v, want %+v", *opts.APIClient, DefaultAPIClientOptions())
	}
}

func TestSetAPIClientOptionsStoresCopy(t *testing.T) {
	apiOpts := APIClientOptions{
		AcceptEncoding:        testAcceptEncodingGzip,
		Timeout:               human.Duration{Duration: time.Second},
		ResponseHeaderTimeout: human.Duration{Duration: 2 * time.Second},
		DialTimeout:           human.Duration{Duration: 3 * time.Second},
		TLSHandshakeTimeout:   human.Duration{Duration: 4 * time.Second},
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       6,
		MaxConcurrentRequests: 7,
	}
	opts := (&Options{}).SetAPIClientOptions(apiOpts)
	apiOpts.AcceptEncoding = ""
	apiOpts.Timeout = human.Duration{}
	apiOpts.MaxConnsPerHost = 0
	apiOpts.MaxConcurrentRequests = 0

	if opts.APIClient == nil {
		t.Fatal("APIClient is nil")
	}

	if got := opts.APIClient.AcceptEncoding; got != testAcceptEncodingGzip {
		t.Fatalf("AcceptEncoding = %q", got)
	}

	if got := opts.APIClient.Timeout.Duration; got != time.Second {
		t.Fatalf("Timeout = %s, want 1s", got)
	}

	if got := opts.APIClient.ResponseHeaderTimeout.Duration; got != 2*time.Second {
		t.Fatalf("ResponseHeaderTimeout = %s, want 2s", got)
	}

	if got := opts.APIClient.DialTimeout.Duration; got != 3*time.Second {
		t.Fatalf("DialTimeout = %s, want 3s", got)
	}

	if got := opts.APIClient.TLSHandshakeTimeout.Duration; got != 4*time.Second {
		t.Fatalf("TLSHandshakeTimeout = %s, want 4s", got)
	}

	if got := opts.APIClient.MaxIdleConnsPerHost; got != 5 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 5", got)
	}

	if got := opts.APIClient.MaxConnsPerHost; got != 6 {
		t.Fatalf("MaxConnsPerHost = %d, want 6", got)
	}

	if got := opts.APIClient.MaxConcurrentRequests; got != 7 {
		t.Fatalf("MaxConcurrentRequests = %d, want 7", got)
	}
}

func TestNilAPIClientOptionsUseDefaults(t *testing.T) {
	opts := (&Options{}).apiClientOptions()

	if opts != DefaultAPIClientOptions() {
		t.Fatalf("options = %+v, want %+v", opts, DefaultAPIClientOptions())
	}
}

func TestExplicitZeroAPIClientOptionsSurvive(t *testing.T) {
	options := (&Options{}).SetAPIClientOptions(APIClientOptions{})
	opts := options.apiClientOptions()
	client := opts.httpClient()
	transport := opts.transport()

	if client.Timeout != 0 {
		t.Fatalf("Timeout = %s, want 0", client.Timeout)
	}

	if transport.DialContext == nil {
		t.Fatal("DialContext is nil")
	}

	if transport.TLSHandshakeTimeout != 0 {
		t.Fatalf("TLSHandshakeTimeout = %s, want 0", transport.TLSHandshakeTimeout)
	}

	if transport.MaxIdleConnsPerHost != 0 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 0", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 0 {
		t.Fatalf("MaxConnsPerHost = %d, want 0", transport.MaxConnsPerHost)
	}

	if opts.MaxConcurrentRequests != 0 {
		t.Fatalf("MaxConcurrentRequests = %d, want 0", opts.MaxConcurrentRequests)
	}

	if _, ok := client.Transport.(*http.Transport); !ok {
		t.Fatalf("Transport type = %T, want *http.Transport", client.Transport)
	}
}

func TestMaxConnsPerHostCapsHTTP1Connections(t *testing.T) {
	const maxConns = 2

	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	var releaseOnce sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	defer releaseOnce.Do(func() { close(release) })

	opts := APIClientOptions{MaxConnsPerHost: maxConns}
	client := opts.httpClient()
	defer client.CloseIdleConnections()

	results := make(chan error, 3)
	for range 3 {
		go func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			if err != nil {
				results <- err

				return
			}

			response, err := client.Do(req)
			if err == nil {
				err = response.Body.Close()
			}

			results <- err
		}()
	}

	for range maxConns {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for request to enter handler")
		}
	}

	select {
	case <-entered:
		t.Fatalf("more than %d requests entered before a connection was released", maxConns)
	case <-time.After(100 * time.Millisecond):
	}

	releaseOnce.Do(func() { close(release) })

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("queued request did not enter after connections were released")
	}

	for range 3 {
		select {
		case err := <-results:
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for request result")
		}
	}
}

func TestMaxConcurrentRequestsCapsHTTP2Streams(t *testing.T) {
	const (
		maxRequests   = 2
		totalRequests = 3
	)

	entered := make(chan struct{}, totalRequests)
	release := make(chan struct{})
	var releaseOnce sync.Once

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ProtoMajor != 2 {
			t.Errorf("protocol = %s", req.Proto)
		}

		w.WriteHeader(http.StatusOK)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush response: %v", err)

			return
		}

		entered <- struct{}{}
		<-release
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	transport := newConcurrencyLimitedRoundTripper(server.Client().Transport, maxRequests)
	client := &http.Client{Transport: transport}
	t.Cleanup(client.CloseIdleConnections)

	type result struct {
		response *http.Response
		err      error
	}
	results := make(chan result, totalRequests)

	for range totalRequests {
		go func() {
			response, err := client.Get(server.URL) //nolint:bodyclose // The receiving goroutine closes every successful response.
			results <- result{response: response, err: err}
		}()
	}

	for range maxRequests {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for request to enter handler")
		}
	}

	responses := make([]*http.Response, 0, totalRequests)
	trackResponse := func(response *http.Response) {
		responses = append(responses, response) //nolint:bodyclose // Registered for cleanup immediately below.
		t.Cleanup(func() {
			_ = response.Body.Close()
		})
	}

	for range maxRequests {
		select {
		case result := <-results:
			if result.err != nil {
				t.Fatalf("request failed: %v", result.err)
			}
			trackResponse(result.response)
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for response headers")
		}
	}

	select {
	case <-entered:
		t.Fatalf("more than %d HTTP/2 streams entered before a body was closed", maxRequests)
	case <-time.After(100 * time.Millisecond):
	}

	if err := responses[0].Body.Close(); err != nil {
		t.Fatalf("close first response: %v", err)
	}

	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("queued HTTP/2 request did not enter after a body was closed")
	}

	select {
	case result := <-results:
		if result.err != nil {
			t.Fatalf("queued request failed: %v", result.err)
		}
		trackResponse(result.response)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for queued response")
	}

	releaseOnce.Do(func() { close(release) })

	for _, response := range responses[1:] {
		if err := response.Body.Close(); err != nil {
			t.Fatalf("close response: %v", err)
		}
	}
}

func TestConfiguredClientTimeoutReleasesConcurrentRequestSlot(t *testing.T) {
	var requests atomic.Int32
	firstDone := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := req.Header.Get("Accept-Encoding"); got != "identity" {
			t.Errorf("Accept-Encoding = %q", got)
		}

		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusOK)
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Errorf("flush response: %v", err)

				return
			}

			<-req.Context().Done()
			close(firstDone)

			return
		}

		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)

	opts := DefaultAPIClientOptions()
	opts.AcceptEncoding = "identity"
	opts.Timeout = human.Duration{Duration: 100 * time.Millisecond}
	opts.MaxConcurrentRequests = 1

	client := opts.httpClient()
	t.Cleanup(client.CloseIdleConnections)

	limited, ok := client.Transport.(*concurrencyLimitedRoundTripper)
	if !ok {
		t.Fatalf("Transport type = %T", client.Transport)
	}

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("get first response: %v", err)
	}

	_, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if !errors.Is(readErr, context.DeadlineExceeded) {
		t.Fatalf("read error = %v, want context.DeadlineExceeded", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close first response: %v", closeErr)
	}
	if got := len(limited.slots); got != 0 {
		t.Fatalf("occupied slots after timeout = %d, want 0", got)
	}

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first handler did not observe timeout")
	}

	response, err = client.Get(server.URL)
	if err != nil {
		t.Fatalf("get second response: %v", err)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr = response.Body.Close()
	if readErr != nil {
		t.Fatalf("read second response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close second response: %v", closeErr)
	}
	if string(data) != "ok" {
		t.Fatalf("second response = %q", data)
	}
}

func TestConcurrentRequestLimitHonorsQueuedContext(t *testing.T) {
	transport := newConcurrencyLimitedRoundTripper(optionsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("nested transport was called")
	}), 1)
	transport.slots <- struct{}{}
	t.Cleanup(func() { <-transport.slots })

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://beacon.test", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := transport.RoundTrip(request) //nolint:bodyclose // A canceled queued request cannot return a response.
	if response != nil {
		t.Fatal("response is non-nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := len(transport.slots); got != 1 {
		t.Fatalf("occupied slots = %d, want 1", got)
	}
}

func TestConcurrentRequestLimitReleasesSlotOnTransportFailure(t *testing.T) {
	transportErr := errors.New("transport failed")
	transport := newConcurrencyLimitedRoundTripper(optionsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, transportErr
	}), 1)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://beacon.test", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := transport.RoundTrip(request) //nolint:bodyclose // The failing transport returns no response.
	if response != nil {
		t.Fatal("response is non-nil")
	}
	if !errors.Is(err, transportErr) {
		t.Fatalf("error = %v", err)
	}
	if got := len(transport.slots); got != 0 {
		t.Fatalf("occupied slots = %d, want 0", got)
	}
}

func TestConcurrentRequestLimitReleasesSlotOnTransportPanic(t *testing.T) {
	transport := newConcurrencyLimitedRoundTripper(optionsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		panic("transport panic")
	}), 1)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://beacon.test", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()

		_, _ = transport.RoundTrip(request) //nolint:bodyclose // The transport panics before returning a response.
	}()

	if recovered == nil {
		t.Fatal("transport panic was not propagated")
	}
	if got := len(transport.slots); got != 0 {
		t.Fatalf("occupied slots = %d, want 0", got)
	}
}
