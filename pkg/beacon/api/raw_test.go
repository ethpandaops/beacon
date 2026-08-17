package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	observerEventOpened = "opened"
	observerEventClosed = "closed"
	observerEventLeaked = "leaked"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingBody struct {
	reader   io.Reader
	closeErr error
	reads    atomic.Int32
	closes   atomic.Int32
}

func (b *trackingBody) Read(p []byte) (int, error) {
	b.reads.Add(1)

	return b.reader.Read(p)
}

func (b *trackingBody) Close() error {
	b.closes.Add(1)

	return b.closeErr
}

type panicBody struct {
	closes atomic.Int32
}

func (*panicBody) Read([]byte) (int, error) {
	panic("read panic")
}

func (b *panicBody) Close() error {
	b.closes.Add(1)

	return nil
}

type blockingBody struct {
	closed    chan struct{}
	closeOnce sync.Once
	closes    atomic.Int32
}

func newBlockingBody() *blockingBody {
	return &blockingBody{closed: make(chan struct{})}
}

func (b *blockingBody) Read([]byte) (int, error) {
	<-b.closed

	return 0, errors.New("body closed")
}

func (b *blockingBody) Close() error {
	b.closes.Add(1)
	b.closeOnce.Do(func() {
		close(b.closed)
	})

	return nil
}

type recordingObserver struct {
	mu     sync.Mutex
	events []string
}

type panicOpenedObserver struct {
	closed atomic.Int32
}

func (*panicOpenedObserver) RawResponseOpened() {
	panic("opened callback failed")
}

func (o *panicOpenedObserver) RawResponseClosed() {
	o.closed.Add(1)
}

func (*panicOpenedObserver) RawResponseLeaked() {}

func (o *recordingObserver) RawResponseOpened() {
	o.record(observerEventOpened)
}

func (o *recordingObserver) RawResponseClosed() {
	o.record(observerEventClosed)
}

func (o *recordingObserver) RawResponseLeaked() {
	o.record(observerEventLeaked)
}

func (o *recordingObserver) record(event string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.events = append(o.events, event)
}

func (o *recordingObserver) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()

	return append([]string(nil), o.events...)
}

func newRawTestClient(rawClient *http.Client) *consensusClient {
	return &consensusClient{
		url:       "http://beacon.test",
		client:    rawClient,
		rawClient: rawClient,
	}
}

func TestNewConsensusClientRequiresHTTPClients(t *testing.T) {
	t.Parallel()

	validClient := &http.Client{}
	tests := []struct {
		name      string
		client    *http.Client
		rawClient *http.Client
		wantPanic string
	}{
		{
			name:      "regular client",
			rawClient: validClient,
			wantPanic: "api: nil HTTP client",
		},
		{
			name:      "raw client",
			client:    validClient,
			wantPanic: "api: nil raw HTTP client",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != test.wantPanic {
					t.Fatalf("panic = %v, want %q", recovered, test.wantPanic)
				}
			}()

			NewConsensusClient(nil, "", test.client, test.rawClient, nil, nil)
		})
	}
}

func TestOpenRawEndpointsRequestAndMetadataWithoutEagerRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		open func(*consensusClient) (*RawResponse, error)
	}{
		{
			name: "debug beacon state",
			path: "/eth/v2/debug/beacon/states/finalized",
			open: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawDebugBeaconState(t.Context(), "finalized", "application/octet-stream")
			},
		},
		{
			name: "block",
			path: "/eth/v2/beacon/blocks/0x1234",
			open: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawBlock(t.Context(), "0x1234", "application/octet-stream")
			},
		},
		{
			name: "execution payload envelope",
			path: "/eth/v1/beacon/execution_payload_envelopes/head",
			open: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawExecutionPayloadEnvelope(t.Context(), "head", "application/octet-stream")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			body := &trackingBody{reader: strings.NewReader("payload")}
			responseHeader := http.Header{
				"Content-Type":          {"application/octet-stream;charset=utf-8"},
				"Eth-Consensus-Version": {"gloas"},
				"X-Mutable":             {"before"},
			}

			var request *http.Request
			transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				request = req.Clone(req.Context())

				return &http.Response{
					StatusCode:    http.StatusOK,
					Header:        responseHeader,
					Body:          body,
					ContentLength: 7,
				}, nil
			})
			client := newRawTestClient(&http.Client{Transport: transport})
			client.headers = map[string]string{
				"Authorization": "Bearer token",
				"X-Request-ID":  "request-id",
			}

			response, err := test.open(client)
			if err != nil {
				t.Fatalf("open response: %v", err)
			}
			if response == nil {
				t.Fatal("open response returned nil")
			}
			t.Cleanup(func() {
				_ = response.Close()
			})

			if request.Method != http.MethodGet {
				t.Fatalf("method = %q, want GET", request.Method)
			}
			if request.URL.Path != test.path {
				t.Fatalf("path = %q, want %q", request.URL.Path, test.path)
			}
			if got := request.Header.Get("Accept"); got != "application/octet-stream" {
				t.Fatalf("Accept = %q", got)
			}
			if got := request.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("Authorization = %q", got)
			}
			if got := request.Header.Get("X-Request-ID"); got != "request-id" {
				t.Fatalf("X-Request-ID = %q", got)
			}
			if got := body.reads.Load(); got != 0 {
				t.Fatalf("body reads before return = %d, want 0", got)
			}
			if response.ContentType != "application/octet-stream;charset=utf-8" {
				t.Fatalf("ContentType = %q", response.ContentType)
			}
			if response.ContentLength != 7 {
				t.Fatalf("ContentLength = %d", response.ContentLength)
			}
			if response.Uncompressed {
				t.Fatal("Uncompressed = true")
			}
			if got := response.Header.Get("Eth-Consensus-Version"); got != "gloas" {
				t.Fatalf("Eth-Consensus-Version = %q", got)
			}

			responseHeader.Set("X-Mutable", "after")
			if got := response.Header.Get("X-Mutable"); got != "before" {
				t.Fatalf("response header was not cloned: %q", got)
			}
		})
	}
}

func TestOpenRawDefaultsAcceptToJSON(t *testing.T) {
	t.Parallel()

	var accept string
	body := &trackingBody{reader: http.NoBody}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		accept = req.Header.Get("Accept")

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	defer response.Close()

	if accept != defaultRawContentType {
		t.Fatalf("Accept = %q, want application/json", accept)
	}
	if got := body.reads.Load(); got != 0 {
		t.Fatalf("body reads before return = %d, want 0", got)
	}
}

func TestBufferedRawDefaultsAcceptToJSON(t *testing.T) {
	t.Parallel()

	var accept string
	body := &trackingBody{reader: http.NoBody}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		accept = req.Header.Get("Accept")

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	data, err := client.RawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("fetch response: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("body length = %d", len(data))
	}
	if accept != defaultRawContentType {
		t.Fatalf("Accept = %q, want application/json", accept)
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("body closes = %d, want 1", got)
	}
}

func TestRawBufferedAndStreamingParity(t *testing.T) {
	t.Parallel()

	const payload = "raw-payload-\x00\x01"

	tests := []struct {
		name     string
		path     string
		buffered func(*consensusClient) ([]byte, error)
		streamed func(*consensusClient) (*RawResponse, error)
	}{
		{
			name: "debug beacon state",
			path: "/eth/v2/debug/beacon/states/finalized",
			buffered: func(client *consensusClient) ([]byte, error) {
				return client.RawDebugBeaconState(t.Context(), "finalized", "application/octet-stream")
			},
			streamed: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawDebugBeaconState(t.Context(), "finalized", "application/octet-stream")
			},
		},
		{
			name: "block",
			path: "/eth/v2/beacon/blocks/head",
			buffered: func(client *consensusClient) ([]byte, error) {
				return client.RawBlock(t.Context(), "head", "application/octet-stream")
			},
			streamed: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawBlock(t.Context(), "head", "application/octet-stream")
			},
		},
		{
			name: "execution payload envelope",
			path: "/eth/v1/beacon/execution_payload_envelopes/0xabcd",
			buffered: func(client *consensusClient) ([]byte, error) {
				return client.RawExecutionPayloadEnvelope(t.Context(), "0xabcd", "application/octet-stream")
			},
			streamed: func(client *consensusClient) (*RawResponse, error) {
				return client.OpenRawExecutionPayloadEnvelope(t.Context(), "0xabcd", "application/octet-stream")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var calls atomic.Int32
			client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != test.path {
					t.Errorf("path = %q, want %q", req.URL.Path, test.path)
				}
				calls.Add(1)

				return &http.Response{
					StatusCode:    http.StatusOK,
					Header:        http.Header{"Content-Type": {"application/octet-stream"}},
					Body:          io.NopCloser(strings.NewReader(payload)),
					ContentLength: int64(len(payload)),
				}, nil
			})})

			buffered, err := test.buffered(client)
			if err != nil {
				t.Fatalf("buffered response: %v", err)
			}
			response, err := test.streamed(client)
			if err != nil {
				t.Fatalf("streamed response: %v", err)
			}
			streamed, readErr := io.ReadAll(response)
			closeErr := response.Close()
			if readErr != nil {
				t.Fatalf("read streamed response: %v", readErr)
			}
			if closeErr != nil {
				t.Fatalf("close streamed response: %v", closeErr)
			}
			if !bytes.Equal(buffered, streamed) {
				t.Fatalf("buffered %q != streamed %q", buffered, streamed)
			}
			if string(streamed) != payload {
				t.Fatalf("payload = %q", streamed)
			}
			if got := calls.Load(); got != 2 {
				t.Fatalf("calls = %d, want 2", got)
			}
		})
	}
}

func TestDoRawRequiresExactStatusOK(t *testing.T) {
	t.Parallel()

	for _, status := range []int{
		http.StatusContinue,
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusPartialContent,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			body := &trackingBody{reader: strings.NewReader("body")}
			client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: status,
					Header:     make(http.Header),
					Body:       body,
				}, nil
			})})

			response, err := client.OpenRawBlock(t.Context(), "head", "")
			if status == http.StatusOK {
				if err != nil {
					t.Fatalf("status 200: %v", err)
				}
				if response == nil {
					t.Fatal("status 200 returned nil response")
				}
				if got := body.closes.Load(); got != 0 {
					t.Fatalf("status 200 close count before ownership transfer = %d", got)
				}
				if closeErr := response.Close(); closeErr != nil {
					t.Fatalf("close status 200 response: %v", closeErr)
				}

				return
			}

			if response != nil {
				t.Fatal("non-200 returned a response")
			}
			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("error %T = %v, want *HTTPStatusError", err, err)
			}
			if statusErr.StatusCode != status {
				t.Fatalf("status = %d, want %d", statusErr.StatusCode, status)
			}
			if got := body.closes.Load(); got != 1 {
				t.Fatalf("close count = %d, want 1", got)
			}
		})
	}
}

func TestRawHTTPStatusErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status       int
		retryAfter   string
		body         string
		wantNotFound bool
	}{
		{
			status:       http.StatusNotFound,
			body:         `{"message":"missing"}`,
			wantNotFound: true,
		},
		{
			status:     http.StatusTooManyRequests,
			retryAfter: "17",
			body:       "slow\ndown\tplease",
		},
		{
			status: http.StatusInternalServerError,
			body:   `"disk failed"`,
		},
	}

	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			t.Parallel()

			body := &trackingBody{reader: strings.NewReader(test.body)}
			client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: test.status,
					Header:     http.Header{"Retry-After": {test.retryAfter}},
					Body:       body,
				}, nil
			})})

			response, err := client.OpenRawBlock(t.Context(), "0xfeed", "application/octet-stream")
			if response != nil {
				t.Fatal("error returned a non-nil response")
			}

			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("error %T = %v, want *HTTPStatusError", err, err)
			}
			if statusErr.StatusCode != test.status {
				t.Fatalf("StatusCode = %d", statusErr.StatusCode)
			}
			if statusErr.RequestPath != "/eth/v2/beacon/blocks/0xfeed" {
				t.Fatalf("RequestPath = %q", statusErr.RequestPath)
			}
			if statusErr.RetryAfter != test.retryAfter {
				t.Fatalf("RetryAfter = %q", statusErr.RetryAfter)
			}
			if statusErr.BodyPrefix != test.body {
				t.Fatalf("BodyPrefix = %q", statusErr.BodyPrefix)
			}
			if got := err.Error(); !strings.Contains(got, strconv.Quote(test.body)) {
				t.Fatalf("error %q does not contain quoted body %q", got, strconv.Quote(test.body))
			}
			if got := errors.Is(err, ErrNotFound); got != test.wantNotFound {
				t.Fatalf("errors.Is(ErrNotFound) = %v", got)
			}
			if got := body.closes.Load(); got != 1 {
				t.Fatalf("close count = %d, want 1", got)
			}
		})
	}
}

func TestRawHTTPStatusErrorBodyPrefixBounded(t *testing.T) {
	t.Parallel()

	const suffix = "must-not-be-read"
	payload := strings.Repeat("x", maxErrorBodyBytes) + suffix
	body := &trackingBody{reader: strings.NewReader(payload)}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if response != nil {
		t.Fatal("error returned a non-nil response")
	}
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error %T = %v, want *HTTPStatusError", err, err)
	}
	if len(statusErr.BodyPrefix) != maxErrorBodyBytes {
		t.Fatalf("BodyPrefix length = %d, want %d", len(statusErr.BodyPrefix), maxErrorBodyBytes)
	}
	if statusErr.BodyPrefix != strings.Repeat("x", maxErrorBodyBytes) {
		t.Fatal("BodyPrefix was not the first 8 KiB")
	}
	if strings.Contains(err.Error(), suffix) {
		t.Fatal("error contains bytes beyond the prefix bound")
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("close count = %d, want 1", got)
	}
}

func TestRawHTTPStatusErrorReadTimeBound(t *testing.T) {
	t.Parallel()

	body := newBlockingBody()
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	type result struct {
		response *RawResponse
		err      error
		elapsed  time.Duration
	}
	resultCh := make(chan result, 1)
	go func() {
		start := time.Now()
		response, err := client.OpenRawBlock(t.Context(), "head", "")
		resultCh <- result{response: response, err: err, elapsed: time.Since(start)}
	}()

	select {
	case got := <-resultCh:
		if got.response != nil {
			t.Fatal("error returned a non-nil response")
		}
		var statusErr *HTTPStatusError
		if !errors.As(got.err, &statusErr) {
			t.Fatalf("error %T = %v, want *HTTPStatusError", got.err, got.err)
		}
		if got.elapsed < errorBodyReadTimeout-(50*time.Millisecond) {
			t.Fatalf("returned too early after %s", got.elapsed)
		}
		if got.elapsed > time.Second {
			t.Fatalf("read cap took %s, want approximately %s", got.elapsed, errorBodyReadTimeout)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stalled error body to be closed")
	}

	if got := body.closes.Load(); got != 1 {
		t.Fatalf("close count = %d, want 1", got)
	}
}

func TestRawTransportErrors(t *testing.T) {
	t.Parallel()

	transportErr := errors.New("transport failed")
	tests := []struct {
		name      string
		transport roundTripFunc
		wantIs    error
	}{
		{
			name: "round trip error",
			transport: func(*http.Request) (*http.Response, error) {
				return nil, transportErr
			},
			wantIs: transportErr,
		},
		{
			name: "nil response",
			transport: func(*http.Request) (*http.Response, error) {
				return nil, nil //nolint:nilnil // verifies net/http rejects an invalid transport result
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := newRawTestClient(&http.Client{Transport: test.transport})
			response, err := client.OpenRawBlock(t.Context(), "head", "")
			if response != nil {
				t.Fatal("transport error returned a non-nil response")
			}
			if err == nil {
				t.Fatal("transport error returned nil error")
			}
			if test.wantIs != nil && !errors.Is(err, test.wantIs) {
				t.Fatalf("error %v does not wrap %v", err, test.wantIs)
			}
		})
	}
}

func TestBufferedRawReadFailureReturnsNilAndCloses(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	body := &trackingBody{reader: io.MultiReader(strings.NewReader("partial"), errorReader{err: readErr})}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	data, err := client.RawBlock(t.Context(), "head", "")
	if data != nil {
		t.Fatalf("data = %q, want nil", data)
	}
	if !errors.Is(err, readErr) {
		t.Fatalf("error = %v, want %v", err, readErr)
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("close count = %d, want 1", got)
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestBufferedRawClosesBodyWhenReadPanics(t *testing.T) {
	t.Parallel()

	body := &panicBody{}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	func() {
		defer func() {
			if recovered := recover(); recovered != "read panic" {
				t.Fatalf("recovered = %v", recovered)
			}
		}()

		_, _ = client.RawBlock(t.Context(), "head", "")
	}()

	if got := body.closes.Load(); got != 1 {
		t.Fatalf("close count = %d, want 1", got)
	}
}

func TestRawResponseCloseIsIdempotentAndStable(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	body := &trackingBody{reader: strings.NewReader("payload"), closeErr: closeErr}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	if err := response.Body.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Body.Close() = %v", err)
	}
	if err := response.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("RawResponse.Close() = %v", err)
	}
	if err := response.Body.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("second Body.Close() = %v", err)
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count = %d, want 1", got)
	}
}

func TestRawResponseDelegatesToBody(t *testing.T) {
	t.Parallel()

	body := &trackingBody{reader: strings.NewReader("payload")}
	response := &RawResponse{Body: body}

	data, err := io.ReadAll(response)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("body = %q, want payload", data)
	}
	if err := response.Close(); err != nil {
		t.Fatalf("close response: %v", err)
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count = %d, want 1", got)
	}
}

func TestRawResponseConcurrentClose(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("close failed")
	body := &trackingBody{reader: strings.NewReader("payload"), closeErr: closeErr}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}

	const goroutines = 32
	start := make(chan struct{})
	errs := make(chan error, goroutines)
	var waitGroup sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		waitGroup.Add(1)
		go func(useBody bool) {
			defer waitGroup.Done()
			<-start
			if useBody {
				errs <- response.Body.Close()
			} else {
				errs <- response.Close()
			}
		}(i%2 == 0)
	}
	close(start)
	waitGroup.Wait()
	close(errs)

	for err := range errs {
		if !errors.Is(err, closeErr) {
			t.Errorf("Close() = %v, want %v", err, closeErr)
		}
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count = %d, want 1", got)
	}
}

func TestRawResponseObserverNormalClose(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	body := &trackingBody{reader: strings.NewReader("payload")}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})
	client.rawResponseObserver = observer

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	if got := observer.snapshot(); !stringSlicesEqual(got, []string{observerEventOpened}) {
		t.Fatalf("events after open = %v", got)
	}
	if err := response.Close(); err != nil {
		t.Fatalf("close response: %v", err)
	}
	if got := observer.snapshot(); !stringSlicesEqual(got, []string{observerEventOpened, observerEventClosed}) {
		t.Fatalf("events after close = %v", got)
	}
}

func TestRawResponseObserverPanicClosesBodyBeforeTransfer(t *testing.T) {
	t.Parallel()

	body := &trackingBody{reader: strings.NewReader("payload")}
	observer := &panicOpenedObserver{}
	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})
	client.rawResponseObserver = observer

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()

		_, _ = client.OpenRawBlock(t.Context(), "head", "")
	}()

	if recovered == nil {
		t.Fatal("observer panic was not propagated")
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count = %d, want 1", got)
	}
	if got := observer.closed.Load(); got != 1 {
		t.Fatalf("closed callbacks = %d, want 1", got)
	}
}

func TestRawResponseCleanupCallback(t *testing.T) {
	t.Parallel()

	closeErr := errors.New("socket close failed")
	body := &trackingBody{reader: strings.NewReader("payload"), closeErr: closeErr}
	observer := &recordingObserver{}
	var logOutput bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logOutput)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	client := newRawTestClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})})
	client.log = logger
	client.rawResponseObserver = observer

	response, err := client.OpenRawExecutionPayloadEnvelope(t.Context(), "0xfeed", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	managedBody, ok := response.Body.(*managedRawBody)
	if !ok {
		t.Fatalf("body type = %T", response.Body)
	}

	cleanupLeakedRawResponse(managedBody.state)

	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count = %d, want 1", got)
	}
	if got := observer.snapshot(); !stringSlicesEqual(got, []string{
		observerEventOpened,
		observerEventLeaked,
		observerEventClosed,
	}) {
		t.Fatalf("events = %v", got)
	}
	logged := logOutput.String()
	for _, want := range []string{
		`level=error`,
		`msg="raw response body was not closed"`,
		`request_path=/eth/v1/beacon/execution_payload_envelopes/0xfeed`,
		`error="socket close failed"`,
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log %q does not contain %q", logged, want)
		}
	}
	if err := response.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("close after cleanup = %v", err)
	}
	if got := body.closes.Load(); got != 1 {
		t.Fatalf("underlying close count after explicit close = %d, want 1", got)
	}
	if got := observer.snapshot(); !stringSlicesEqual(got, []string{
		observerEventOpened,
		observerEventLeaked,
		observerEventClosed,
	}) {
		t.Fatalf("events after explicit close = %v", got)
	}
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
