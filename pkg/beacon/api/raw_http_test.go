package api

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testHTTP1Protocol = "HTTP/1.1"
	testHTTP2Name     = "HTTP/2"
	testHTTP2Protocol = "HTTP/2.0"
	testBodyABC       = "abc"
	testGzipEncoding  = "gzip"
)

type protocolServer struct {
	url    string
	client *http.Client
}

func newProtocolServer(t *testing.T, http2 bool, handler http.Handler) protocolServer {
	t.Helper()

	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = http2
	if http2 {
		server.StartTLS()
	} else {
		server.Start()
	}
	t.Cleanup(server.Close)

	client := server.Client()
	client.Timeout = 0
	t.Cleanup(client.CloseIdleConnections)

	return protocolServer{
		url:    server.URL,
		client: client,
	}
}

func TestOpenRawReturnsAfterHeadersAndContextCancelsRead(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		name := testHTTP1Protocol
		if http2 {
			name = testHTTP2Name
		}

		t.Run(name, func(t *testing.T) {
			headersSent := make(chan struct{})
			handlerDone := make(chan struct{})
			server := newProtocolServer(t, http2, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer close(handlerDone)
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("X-Protocol", req.Proto)
				w.WriteHeader(http.StatusOK)
				if err := http.NewResponseController(w).Flush(); err != nil {
					t.Errorf("flush response headers: %v", err)

					return
				}
				close(headersSent)
				<-req.Context().Done()
			}))
			client := &consensusClient{
				url:       server.url,
				client:    server.client,
				rawClient: server.client,
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			type openResult struct {
				response *RawResponse
				err      error
			}
			opened := make(chan openResult, 1)
			go func() {
				response, err := client.OpenRawDebugBeaconState(ctx, "head", "application/octet-stream")
				opened <- openResult{response: response, err: err}
			}()

			select {
			case <-headersSent:
			case <-time.After(2 * time.Second):
				t.Fatal("server did not send response headers")
			}

			var result openResult
			select {
			case result = <-opened:
			case <-time.After(2 * time.Second):
				t.Fatal("open did not return while the response body was unavailable")
			}
			if result.err != nil {
				t.Fatalf("open response: %v", result.err)
			}
			if result.response == nil {
				t.Fatal("open response returned nil")
			}
			defer result.response.Close()

			wantProtocol := testHTTP1Protocol
			if http2 {
				wantProtocol = testHTTP2Protocol
			}
			if got := result.response.Header.Get("X-Protocol"); got != wantProtocol {
				t.Fatalf("protocol = %q, want %q", got, wantProtocol)
			}

			cancel()
			_, err := io.ReadAll(result.response)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("read error = %v, want context.Canceled", err)
			}

			select {
			case <-handlerDone:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not observe context cancellation")
			}
		})
	}
}

func TestRawClientTimeoutDuringBodyRead(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		name := testHTTP1Protocol
		if http2 {
			name = testHTTP2Name
		}

		t.Run(name, func(t *testing.T) {
			handlerDone := make(chan struct{})
			server := newProtocolServer(t, http2, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				defer close(handlerDone)
				w.Header().Set("X-Protocol", req.Proto)
				w.WriteHeader(http.StatusOK)
				if err := http.NewResponseController(w).Flush(); err != nil {
					t.Errorf("flush response headers: %v", err)

					return
				}
				<-req.Context().Done()
			}))
			server.client.Timeout = 100 * time.Millisecond

			client := &consensusClient{
				url:       server.url,
				client:    server.client,
				rawClient: server.client,
			}
			response, err := client.OpenRawBlock(t.Context(), "head", "")
			if err != nil {
				t.Fatalf("open response: %v", err)
			}
			defer response.Close()

			type readResult struct {
				err error
			}
			readDone := make(chan readResult, 1)
			go func() {
				_, readErr := io.ReadAll(response)
				readDone <- readResult{err: readErr}
			}()

			select {
			case result := <-readDone:
				if !errors.Is(result.err, context.DeadlineExceeded) {
					t.Fatalf("read error = %v, want context.DeadlineExceeded", result.err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("client timeout did not abort body read")
			}

			select {
			case <-handlerDone:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not observe client timeout")
			}
		})
	}
}

func TestRawResponseHeaderTimeout(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		name := testHTTP1Protocol
		if http2 {
			name = testHTTP2Name
		}

		t.Run(name, func(t *testing.T) {
			handlerStarted := make(chan struct{})
			handlerDone := make(chan struct{})
			server := newProtocolServer(t, http2, http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
				close(handlerStarted)
				<-req.Context().Done()
				close(handlerDone)
			}))

			baseTransport, ok := server.client.Transport.(*http.Transport)
			if !ok {
				t.Fatalf("transport type = %T", server.client.Transport)
			}
			transport := baseTransport.Clone()
			transport.ResponseHeaderTimeout = 100 * time.Millisecond
			timeoutClient := &http.Client{Transport: transport}
			t.Cleanup(timeoutClient.CloseIdleConnections)
			client := &consensusClient{
				url:       server.url,
				client:    timeoutClient,
				rawClient: timeoutClient,
			}

			type openResult struct {
				response *RawResponse
				err      error
			}
			opened := make(chan openResult, 1)
			go func() {
				response, err := client.OpenRawBlock(t.Context(), "head", "")
				opened <- openResult{response: response, err: err}
			}()

			select {
			case <-handlerStarted:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not receive request")
			}

			select {
			case result := <-opened:
				if result.response != nil {
					t.Fatal("header timeout returned a response")
				}
				if result.err == nil {
					t.Fatal("header timeout returned nil error")
				}
				var netErr net.Error
				if !errors.As(result.err, &netErr) || !netErr.Timeout() {
					t.Fatalf("error = %v, want timeout", result.err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("response header timeout did not fire")
			}

			select {
			case <-handlerDone:
			case <-time.After(2 * time.Second):
				t.Fatal("handler did not observe response header timeout")
			}
		})
	}
}

type rawHTTPServer struct {
	url  string
	done <-chan struct{}
}

func newRawHTTPServer(t *testing.T, rawResponse string) rawHTTPServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer connection.Close()

		request, readErr := http.ReadRequest(bufio.NewReader(connection))
		if readErr != nil {
			return
		}
		_ = request.Body.Close()
		_, _ = io.WriteString(connection, rawResponse)
	}()

	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("raw HTTP server did not stop")
		}
	})

	return rawHTTPServer{
		url:  "http://" + listener.Addr().String(),
		done: done,
	}
}

func TestHTTP1ResponseFramingReadErrors(t *testing.T) {
	tests := []struct {
		name              string
		rawResponse       string
		wantContentLength int64
		wantBody          string
		wantUnexpectedEOF bool
	}{
		{
			name: "declared length truncation",
			rawResponse: testHTTP1Protocol + " 200 OK\r\n" +
				"Content-Length: 10\r\n" +
				"Connection: close\r\n\r\n" +
				testBodyABC,
			wantContentLength: 10,
			wantBody:          testBodyABC,
			wantUnexpectedEOF: true,
		},
		{
			name: "truncated chunked response",
			rawResponse: testHTTP1Protocol + " 200 OK\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"Connection: close\r\n\r\n" +
				"3\r\n" + testBodyABC + "\r\n5\r\nxy",
			wantContentLength: -1,
			wantBody:          testBodyABC + "xy",
			wantUnexpectedEOF: true,
		},
		{
			name: "close delimited response",
			rawResponse: testHTTP1Protocol + " 200 OK\r\n" +
				"Connection: close\r\n\r\n" +
				testBodyABC,
			wantContentLength: -1,
			wantBody:          testBodyABC,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newRawHTTPServer(t, test.rawResponse)
			transport := &http.Transport{DisableKeepAlives: true}
			httpClient := &http.Client{Transport: transport}
			t.Cleanup(httpClient.CloseIdleConnections)
			client := &consensusClient{
				url:       server.url,
				client:    httpClient,
				rawClient: httpClient,
			}

			response, err := client.OpenRawBlock(t.Context(), "head", "")
			if err != nil {
				t.Fatalf("open response: %v", err)
			}
			data, readErr := io.ReadAll(response)
			closeErr := response.Close()
			if closeErr != nil {
				t.Fatalf("close response: %v", closeErr)
			}
			if string(data) != test.wantBody {
				t.Fatalf("body = %q, want %q", data, test.wantBody)
			}
			if response.ContentLength != test.wantContentLength {
				t.Fatalf("ContentLength = %d, want %d", response.ContentLength, test.wantContentLength)
			}
			if test.wantUnexpectedEOF {
				if !errors.Is(readErr, io.ErrUnexpectedEOF) {
					t.Fatalf("read error = %v, want io.ErrUnexpectedEOF", readErr)
				}
			} else if readErr != nil {
				t.Fatalf("read error = %v, want nil", readErr)
			}

			select {
			case <-server.done:
			case <-time.After(2 * time.Second):
				t.Fatal("raw HTTP server did not finish")
			}
		})
	}
}

func TestHTTP2DeclaredLengthTruncation(t *testing.T) {
	server := newProtocolServer(t, true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Header().Set("X-Protocol", req.Proto)
		_, _ = io.WriteString(w, testBodyABC)
	}))
	client := &consensusClient{
		url:       server.url,
		client:    server.client,
		rawClient: server.client,
	}

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	defer response.Close()

	data, err := io.ReadAll(response)
	if string(data) != testBodyABC {
		t.Fatalf("body = %q", data)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("read error = %v, want io.ErrUnexpectedEOF", err)
	}
	if got := response.Header.Get("X-Protocol"); got != testHTTP2Protocol {
		t.Fatalf("protocol = %q, want %s", got, testHTTP2Protocol)
	}
	if response.ContentLength != 10 {
		t.Fatalf("ContentLength = %d, want 10", response.ContentLength)
	}
}

func TestHTTP2ResetDuringBodyRead(t *testing.T) {
	server := newProtocolServer(t, true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", "10")
		w.Header().Set("X-Protocol", req.Proto)
		_, _ = io.WriteString(w, testBodyABC)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush response body: %v", err)

			return
		}

		panic(http.ErrAbortHandler)
	}))
	client := &consensusClient{
		url:       server.url,
		client:    server.client,
		rawClient: server.client,
	}

	response, err := client.OpenRawBlock(t.Context(), "head", "")
	if err != nil {
		t.Fatalf("open response: %v", err)
	}
	defer response.Close()

	data, readErr := io.ReadAll(response)
	if string(data) != testBodyABC {
		t.Fatalf("body = %q, want %q", data, testBodyABC)
	}
	if readErr == nil {
		t.Fatal("HTTP/2 reset returned a nil read error")
	}
	if errors.Is(readErr, io.ErrUnexpectedEOF) {
		t.Fatalf("HTTP/2 reset error = %v, unexpectedly classified as io.ErrUnexpectedEOF", readErr)
	}
	if got := response.Header.Get("X-Protocol"); got != testHTTP2Protocol {
		t.Fatalf("protocol = %q, want %s", got, testHTTP2Protocol)
	}
}

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	return compressed.Bytes()
}

func TestRawGzipTransportSemantics(t *testing.T) {
	const contentType = "application/octet-stream;charset=utf-8"
	plain := []byte(strings.Repeat("beacon-state-", 32))
	compressed := gzipBytes(t, plain)

	tests := []struct {
		name              string
		headers           map[string]string
		wantUncompressed  bool
		wantContentLength int64
		wantEncoding      string
		wantBody          []byte
	}{
		{
			name:              "transparent gzip",
			wantUncompressed:  true,
			wantContentLength: -1,
			wantBody:          plain,
		},
		{
			name:              "explicit gzip",
			headers:           map[string]string{"Accept-Encoding": testGzipEncoding},
			wantContentLength: int64(len(compressed)),
			wantEncoding:      testGzipEncoding,
			wantBody:          compressed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			acceptEncoding := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				acceptEncoding <- req.Header.Get("Accept-Encoding")
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Content-Encoding", testGzipEncoding)
				w.Header().Set("Content-Length", strconv.Itoa(len(compressed)))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(compressed)
			}))
			t.Cleanup(server.Close)

			transport := &http.Transport{}
			httpClient := &http.Client{Transport: transport}
			t.Cleanup(httpClient.CloseIdleConnections)
			client := &consensusClient{
				url:       server.URL,
				client:    httpClient,
				rawClient: httpClient,
				headers:   test.headers,
			}

			response, err := client.OpenRawDebugBeaconState(t.Context(), "head", "application/octet-stream")
			if err != nil {
				t.Fatalf("open response: %v", err)
			}
			data, readErr := io.ReadAll(response)
			closeErr := response.Close()
			if readErr != nil {
				t.Fatalf("read response: %v", readErr)
			}
			if closeErr != nil {
				t.Fatalf("close response: %v", closeErr)
			}
			if got := <-acceptEncoding; got != testGzipEncoding {
				t.Fatalf("Accept-Encoding = %q, want %s", got, testGzipEncoding)
			}
			if response.Uncompressed != test.wantUncompressed {
				t.Fatalf("Uncompressed = %v, want %v", response.Uncompressed, test.wantUncompressed)
			}
			if response.ContentLength != test.wantContentLength {
				t.Fatalf("ContentLength = %d, want %d", response.ContentLength, test.wantContentLength)
			}
			if response.ContentType != contentType {
				t.Fatalf("ContentType = %q", response.ContentType)
			}
			if got := response.Header.Get("Content-Encoding"); got != test.wantEncoding {
				t.Fatalf("Content-Encoding = %q, want %q", got, test.wantEncoding)
			}
			if !bytes.Equal(data, test.wantBody) {
				t.Fatalf("body differs: got %d bytes, want %d", len(data), len(test.wantBody))
			}
		})
	}
}

func contextWithReuseTrace(ctx context.Context) (context.Context, *atomic.Bool) {
	reused := new(atomic.Bool)
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			reused.Store(info.Reused)
		},
	}

	return httptrace.WithClientTrace(ctx, trace), reused
}

func TestHTTP1ConnectionReuseAfterFullSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "7")
		_, _ = io.WriteString(w, "payload")
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawBlock(ctx, "head", "")
		if err != nil {
			t.Fatalf("request %d: open response: %v", requestNumber, err)
		}
		if _, err := io.Copy(io.Discard, response); err != nil {
			t.Fatalf("request %d: drain response: %v", requestNumber, err)
		}
		if err := response.Close(); err != nil {
			t.Fatalf("request %d: close response: %v", requestNumber, err)
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d: reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestHTTP1ConnectionReuseAfterSmallStatusError(t *testing.T) {
	const errorBody = `{"message":"not found"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(errorBody)))
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, errorBody)
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawExecutionPayloadEnvelope(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("request %d error = %v, want ErrNotFound", requestNumber, err)
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d: reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestHTTP1ConnectionReuseAfterRateLimit(t *testing.T) {
	const (
		errorBody  = `{"message":"rate limited"}`
		retryAfter = "12"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(errorBody)))
		w.Header().Set("Retry-After", retryAfter)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, errorBody)
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawBlock(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}

		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("request %d error = %v", requestNumber, err)
		}
		if statusErr.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("request %d status = %d", requestNumber, statusErr.StatusCode)
		}
		if statusErr.RetryAfter != retryAfter {
			t.Fatalf("request %d Retry-After = %q", requestNumber, statusErr.RetryAfter)
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestHTTP1ConnectionReuseAfterExactCapStatusError(t *testing.T) {
	errorBody := strings.Repeat("x", maxErrorBodyBytes)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush status headers: %v", err)

			return
		}

		_, _ = io.WriteString(w, errorBody)
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawBlock(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("request %d error = %v", requestNumber, err)
		}
		if len(statusErr.BodyPrefix) != maxErrorBodyBytes {
			t.Fatalf("request %d prefix length = %d", requestNumber, len(statusErr.BodyPrefix))
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestHTTPConnectionLifecycleLeavesNoOpenConnections(t *testing.T) {
	var openConnections atomic.Int64
	stateChanged := make(chan struct{}, 16)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case strings.HasSuffix(req.URL.Path, "/missing"):
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, "missing")
		case strings.HasSuffix(req.URL.Path, "/partial"):
			w.Header().Set("Content-Length", "1024")
			_, _ = io.WriteString(w, "x")
		default:
			_, _ = io.WriteString(w, "complete")
		}
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			openConnections.Add(1)
		case http.StateHijacked, http.StateClosed:
			openConnections.Add(-1)
		default:
		}

		select {
		case stateChanged <- struct{}{}:
		default:
		}
	}
	server.Start()
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	response, err := client.OpenRawBlock(t.Context(), "complete", "")
	if err != nil {
		t.Fatalf("open complete response: %v", err)
	}
	if _, err = io.Copy(io.Discard, response); err != nil {
		t.Fatalf("read complete response: %v", err)
	}
	if err = response.Close(); err != nil {
		t.Fatalf("close complete response: %v", err)
	}

	response, err = client.OpenRawBlock(t.Context(), "missing", "")
	if response != nil {
		t.Fatal("missing response returned a body")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing response error = %v", err)
	}

	response, err = client.OpenRawBlock(t.Context(), "partial", "")
	if err != nil {
		t.Fatalf("open partial response: %v", err)
	}
	if _, err := io.Copy(io.Discard, response); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial response read error = %v", err)
	}
	if err := response.Close(); err != nil {
		t.Fatalf("close partial response: %v", err)
	}

	httpClient.CloseIdleConnections()

	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()

	for openConnections.Load() != 0 {
		select {
		case <-stateChanged:
		case <-timer.C:
			t.Fatalf("%d HTTP connections remain open", openConnections.Load())
		}
	}
}

func TestHTTP1ConnectionNotReusedAfterPartialSuccess(t *testing.T) {
	var requests atomic.Int32
	firstWritten := make(chan struct{})
	firstDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(1<<20))
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "x")
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Errorf("flush partial response: %v", err)

				return
			}
			close(firstWritten)
			<-req.Context().Done()
			close(firstDone)

			return
		}

		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	ctx, firstReused := contextWithReuseTrace(t.Context())
	response, err := client.OpenRawBlock(ctx, "head", "")
	if err != nil {
		t.Fatalf("open first response: %v", err)
	}
	select {
	case <-firstWritten:
	case <-time.After(2 * time.Second):
		t.Fatal("first handler did not write body prefix")
	}
	buffer := make([]byte, 1)
	if _, readErr := io.ReadFull(response, buffer); readErr != nil {
		t.Fatalf("read first response prefix: %v", readErr)
	}
	if closeErr := response.Close(); closeErr != nil {
		t.Fatalf("close first response: %v", closeErr)
	}
	if firstReused.Load() {
		t.Fatal("first request unexpectedly reused a connection")
	}

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("first handler did not observe the abandoned response")
	}

	ctx, secondReused := contextWithReuseTrace(t.Context())
	response, err = client.OpenRawBlock(ctx, "head", "")
	if err != nil {
		t.Fatalf("open second response: %v", err)
	}
	_, readErr := io.Copy(io.Discard, response)
	closeErr := response.Close()
	if readErr != nil {
		t.Fatalf("read second response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close second response: %v", closeErr)
	}
	if secondReused.Load() {
		t.Fatal("connection was reused after a partially consumed HTTP/1.1 response")
	}
}

func TestHTTP1ConnectionNotReusedAfterOverCapStatusError(t *testing.T) {
	errorBody := strings.Repeat("x", maxErrorBodyBytes+2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(errorBody)))
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, errorBody)
	}))
	t.Cleanup(server.Close)

	transport := &http.Transport{MaxIdleConnsPerHost: 1}
	httpClient := &http.Client{Transport: transport}
	t.Cleanup(httpClient.CloseIdleConnections)
	client := &consensusClient{url: server.URL, client: httpClient, rawClient: httpClient}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawBlock(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("request %d error = %v", requestNumber, err)
		}
		if len(statusErr.BodyPrefix) != maxErrorBodyBytes {
			t.Fatalf("request %d prefix length = %d", requestNumber, len(statusErr.BodyPrefix))
		}
		if reused.Load() {
			t.Fatalf("request %d reused a connection after an over-cap response", requestNumber)
		}
	}
}

func TestHTTP2ConnectionReusedAfterPartialSuccess(t *testing.T) {
	var requests atomic.Int32
	firstDone := make(chan struct{})
	server := newProtocolServer(t, true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(1<<20))
			w.Header().Set("X-Protocol", req.Proto)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "x")
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Errorf("flush partial response: %v", err)

				return
			}
			<-req.Context().Done()
			close(firstDone)

			return
		}

		w.Header().Set("X-Protocol", req.Proto)
		_, _ = io.WriteString(w, "ok")
	}))
	client := &consensusClient{url: server.url, client: server.client, rawClient: server.client}

	ctx, firstReused := contextWithReuseTrace(t.Context())
	response, err := client.OpenRawBlock(ctx, "head", "")
	if err != nil {
		t.Fatalf("open first response: %v", err)
	}
	buffer := make([]byte, 1)
	if _, readErr := io.ReadFull(response, buffer); readErr != nil {
		t.Fatalf("read first response prefix: %v", readErr)
	}
	if closeErr := response.Close(); closeErr != nil {
		t.Fatalf("close first response: %v", closeErr)
	}
	if firstReused.Load() {
		t.Fatal("first request unexpectedly reused a connection")
	}

	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP/2 handler did not observe stream cancellation")
	}

	ctx, secondReused := contextWithReuseTrace(t.Context())
	response, err = client.OpenRawBlock(ctx, "head", "")
	if err != nil {
		t.Fatalf("open second response: %v", err)
	}
	data, readErr := io.ReadAll(response)
	closeErr := response.Close()
	if readErr != nil {
		t.Fatalf("read second response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close second response: %v", closeErr)
	}
	if string(data) != "ok" {
		t.Fatalf("second body = %q", data)
	}
	if got := response.Header.Get("X-Protocol"); got != testHTTP2Protocol {
		t.Fatalf("protocol = %q, want %s", got, testHTTP2Protocol)
	}
	if !secondReused.Load() {
		t.Fatal("HTTP/2 connection was not reused after cancelling one stream")
	}
}

func TestHTTP2ConnectionReuseAfterStatusErrors(t *testing.T) {
	const errorBody = `{"message":"not found"}`
	server := newProtocolServer(t, true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(errorBody)))
		w.Header().Set("X-Protocol", req.Proto)
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, errorBody)
	}))
	client := &consensusClient{url: server.url, client: server.client, rawClient: server.client}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawExecutionPayloadEnvelope(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("request %d error = %v", requestNumber, err)
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestHTTP2ConnectionReuseAfterOverCapStatusError(t *testing.T) {
	errorBody := strings.Repeat("x", maxErrorBodyBytes+2)
	server := newProtocolServer(t, true, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(errorBody)))
		w.Header().Set("X-Protocol", req.Proto)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, errorBody)
	}))
	client := &consensusClient{url: server.url, client: server.client, rawClient: server.client}

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		ctx, reused := contextWithReuseTrace(t.Context())
		response, err := client.OpenRawBlock(ctx, "head", "")
		if response != nil {
			t.Fatalf("request %d returned a response", requestNumber)
		}
		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) {
			t.Fatalf("request %d error = %v", requestNumber, err)
		}
		if len(statusErr.BodyPrefix) != maxErrorBodyBytes {
			t.Fatalf("request %d prefix length = %d", requestNumber, len(statusErr.BodyPrefix))
		}

		wantReused := requestNumber == 2
		if got := reused.Load(); got != wantReused {
			t.Fatalf("request %d reused = %v, want %v", requestNumber, got, wantReused)
		}
	}
}

func TestRawStatusErrorFormattingWithoutBody(t *testing.T) {
	t.Parallel()

	statusErr := &HTTPStatusError{
		StatusCode:  http.StatusBadGateway,
		RequestPath: "/eth/v2/beacon/blocks/head",
	}
	want := fmt.Sprintf(
		"status code: %d for /eth/v2/beacon/blocks/head",
		http.StatusBadGateway,
	)
	if got := statusErr.Error(); got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
	if statusErr.Unwrap() != nil {
		t.Fatalf("Unwrap() = %v, want nil", statusErr.Unwrap())
	}
}
