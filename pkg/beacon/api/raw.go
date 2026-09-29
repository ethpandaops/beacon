package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	maxErrorBodyBytes     = 8 << 10
	errorBodyReadTimeout  = 250 * time.Millisecond
	defaultRawContentType = "application/json"
)

// RawResponseObserver receives raw response body lifecycle events.
type RawResponseObserver interface {
	RawResponseOpened()
	RawResponseClosed()
	RawResponseLeaked()
}

// HTTPStatusError describes a non-200 response from a raw endpoint.
type HTTPStatusError struct {
	StatusCode  int
	RequestPath string
	RetryAfter  string
	BodyPrefix  string
}

func (e *HTTPStatusError) Error() string {
	if e.BodyPrefix == "" {
		return fmt.Sprintf("status code: %d for %s", e.StatusCode, e.RequestPath)
	}

	return fmt.Sprintf("status code: %d for %s: %q", e.StatusCode, e.RequestPath, e.BodyPrefix)
}

// Unwrap preserves ErrNotFound classification for HTTP 404 responses.
func (e *HTTPStatusError) Unwrap() error {
	if e.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	return nil
}

// RawResponse streams a successful raw endpoint response. Opening the response
// only validates its status; callers must treat any read error as fatal and
// avoid committing partial output. The caller must close the response,
// including when a read fails or the body is only partially read.
type RawResponse struct {
	// Body is the managed response stream.
	Body io.ReadCloser
	// ContentType is the response's actual Content-Type.
	ContentType string
	// ContentLength is -1 when the response length is unknown, so length-based
	// completeness cannot be verified.
	ContentLength int64
	// Uncompressed reports whether net/http transparently decompressed the body.
	// Use Options.SetAPIClientAcceptEncoding to receive an encoded body without
	// automatic decompression.
	Uncompressed bool
	// Header is a clone of the response headers.
	Header http.Header
}

// Read reads from the response body.
func (r *RawResponse) Read(p []byte) (int, error) {
	return r.Body.Read(p)
}

// Close releases the response body. Repeated calls return the first close
// result.
func (r *RawResponse) Close() error {
	return r.Body.Close()
}

type rawResponseCloseState struct {
	once     sync.Once
	body     io.ReadCloser
	log      logrus.FieldLogger
	observer RawResponseObserver
	path     string
	closeErr error
}

type managedRawBody struct {
	state   *rawResponseCloseState
	cleanup runtime.Cleanup
}

func (b *managedRawBody) Read(p []byte) (int, error) {
	n, err := b.state.body.Read(p)
	runtime.KeepAlive(b)

	return n, err
}

func (b *managedRawBody) Close() error {
	b.cleanup.Stop()
	err := b.state.close(false)
	runtime.KeepAlive(b)

	return err
}

func (s *rawResponseCloseState) close(leaked bool) error {
	s.once.Do(func() {
		s.closeErr = s.body.Close()

		if leaked {
			if s.log != nil {
				entry := s.log.WithField("request_path", s.path)
				if s.closeErr != nil {
					entry.WithError(s.closeErr).Error("raw response body was not closed")
				} else {
					entry.Error("raw response body was not closed")
				}
			}

			if s.observer != nil {
				s.observer.RawResponseLeaked()
			}
		}

		if s.observer != nil {
			s.observer.RawResponseClosed()
		}
	})

	return s.closeErr
}

func cleanupLeakedRawResponse(state *rawResponseCloseState) {
	_ = state.close(true)
}

func (c *consensusClient) doRaw(ctx context.Context, path string, contentType string) (*http.Response, error) {
	if contentType == "" {
		contentType = defaultRawContentType
	}

	u, err := url.Parse(c.url + path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Accept", contentType)

	rsp, err := c.rawClient.Do(req)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode == http.StatusOK {
		return rsp, nil
	}

	bodyPrefix := readAndCloseErrorBody(rsp.Body)

	return nil, &HTTPStatusError{
		StatusCode:  rsp.StatusCode,
		RequestPath: path,
		RetryAfter:  rsp.Header.Get("Retry-After"),
		BodyPrefix:  bodyPrefix,
	}
}

func readAndCloseErrorBody(body io.ReadCloser) string {
	timerDone := make(chan struct{})

	var closeOnce sync.Once

	closeBody := func() {
		closeOnce.Do(func() {
			_ = body.Close()
		})
	}
	timer := time.AfterFunc(errorBodyReadTimeout, func() {
		closeBody()

		close(timerDone)
	})

	data, _ := io.ReadAll(io.LimitReader(body, maxErrorBodyBytes+1))
	if len(data) > maxErrorBodyBytes {
		data = data[:maxErrorBodyBytes]
	}

	if timer.Stop() {
		close(timerDone)
	}

	<-timerDone

	closeBody()

	return string(data)
}

func (c *consensusClient) openRaw(ctx context.Context, path string, contentType string) (*RawResponse, error) {
	rsp, err := c.doRaw(ctx, path, contentType) //nolint:bodyclose // successful responses transfer body ownership
	if err != nil {
		return nil, err
	}

	logger := c.log
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	state := &rawResponseCloseState{
		body:     rsp.Body,
		log:      logger,
		observer: c.rawResponseObserver,
		path:     path,
	}
	body := &managedRawBody{state: state}
	body.cleanup = runtime.AddCleanup(body, cleanupLeakedRawResponse, state)

	transferred := false
	defer func() {
		if !transferred {
			_ = body.Close()
		}
	}()

	response := &RawResponse{
		Body:          body,
		ContentType:   rsp.Header.Get("Content-Type"),
		ContentLength: rsp.ContentLength,
		Uncompressed:  rsp.Uncompressed,
		Header:        rsp.Header.Clone(),
	}

	if c.rawResponseObserver != nil {
		c.rawResponseObserver.RawResponseOpened()
	}

	transferred = true

	runtime.KeepAlive(body)

	return response, nil
}

// OpenRawDebugBeaconState opens the beacon state response for streaming.
func (c *consensusClient) OpenRawDebugBeaconState(ctx context.Context, stateID string, contentType string) (*RawResponse, error) {
	return c.openRaw(ctx, fmt.Sprintf("/eth/v2/debug/beacon/states/%s", stateID), contentType)
}

// OpenRawBlock opens the block response for streaming.
func (c *consensusClient) OpenRawBlock(ctx context.Context, blockID string, contentType string) (*RawResponse, error) {
	return c.openRaw(ctx, fmt.Sprintf("/eth/v2/beacon/blocks/%s", blockID), contentType)
}

// OpenRawExecutionPayloadEnvelope opens the signed execution payload envelope
// response for streaming (gloas onwards).
func (c *consensusClient) OpenRawExecutionPayloadEnvelope(ctx context.Context, blockID string, contentType string) (*RawResponse, error) {
	return c.openRaw(ctx, fmt.Sprintf("/eth/v1/beacon/execution_payload_envelopes/%s", blockID), contentType)
}
