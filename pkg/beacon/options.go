package beacon

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ethpandaops/beacon/pkg/human"
	ehttp "github.com/ethpandaops/go-eth2-client/http"
)

// Options holds the options for a beacon node.
type Options struct {
	BeaconSubscription BeaconSubscriptionOptions
	HealthCheck        HealthCheckOptions
	PrometheusMetrics  bool
	DetectEmptySlots   bool
	GoEth2ClientParams []ehttp.Parameter
	APIClient          *APIClientOptions
}

// EnablePrometheusMetrics enables Prometheus metrics.
func (o *Options) EnablePrometheusMetrics() *Options {
	o.PrometheusMetrics = true

	return o
}

// DisablePrometheusMetrics disables Prometheus metrics.
func (o *Options) DisablePrometheusMetrics() *Options {
	o.PrometheusMetrics = false

	return o
}

// EnableEmptySlotDetection enables empty slot detection.
func (o *Options) EnableEmptySlotDetection() *Options {
	o.DetectEmptySlots = true

	return o
}

// DisableEmptySlotDetection disables empty slot detection.
func (o *Options) DisableEmptySlotDetection() *Options {
	o.DetectEmptySlots = false

	return o
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	apiClient := DefaultAPIClientOptions()

	return &Options{
		BeaconSubscription: DefaultDisabledBeaconSubscriptionOptions(),
		HealthCheck:        DefaultHealthCheckOptions(),
		PrometheusMetrics:  true,
		DetectEmptySlots:   false,
		APIClient:          &apiClient,
	}
}

// BeaconSubscriptionOptions holds the options for beacon subscription.
type BeaconSubscriptionOptions struct {
	Enabled bool
	Topics  EventTopics
}

// Disable disables the beacon subscription.
func (b *BeaconSubscriptionOptions) Disable() *BeaconSubscriptionOptions {
	b.Enabled = false

	return b
}

// Enable enables the beacon subscription.
func (b *BeaconSubscriptionOptions) Enable() *BeaconSubscriptionOptions {
	b.Enabled = true

	return b
}

// DefaultDisabledBeaconSubscriptionOptions returns the default options for a disabled beacon subscription.
func DefaultDisabledBeaconSubscriptionOptions() BeaconSubscriptionOptions {
	return BeaconSubscriptionOptions{
		Enabled: false,
		Topics:  []string{},
	}
}

// DefaultEnabledBeaconSubscriptionOptions returns the default options for an enabled beacon subscription.
//
// execution_payload, execution_payload_gossip, execution_payload_available,
// execution_payload_bid, payload_attestation_message, and proposer_preferences
// can be added once Gloas is live if you want them defaulting. Not adding for
// now due to connection thrashing, etc.
func DefaultEnabledBeaconSubscriptionOptions() BeaconSubscriptionOptions {
	return BeaconSubscriptionOptions{
		Enabled: true,
		Topics: []string{
			topicAttestation,
			topicSingleAttestation,
			topicBlock,
			topicBlockGossip,
			topicChainReorg,
			topicFinalizedCheckpoint,
			topicHead,
			topicVoluntaryExit,
			topicContributionAndProof,
			topicBlobSidecar,
		},
	}
}

// EnableDefaultBeaconSubscription enables the default beacon subscription.
func (o *Options) EnableDefaultBeaconSubscription() *Options {
	o.BeaconSubscription = DefaultEnabledBeaconSubscriptionOptions()

	return o
}

// HealthCheckOptions holds the options for the health check.
type HealthCheckOptions struct {
	// Interval is the interval at which the health check will be run.
	Interval human.Duration
	// SuccessThreshold is the number of consecutive successful health checks required before the node is considered healthy.
	SuccessfulResponses int
	// FailureThreshold is the number of consecutive failed health checks required before the node is considered unhealthy.
	FailedResponses int
}

// DefaultHealthCheckOptions returns the default health check options.
func DefaultHealthCheckOptions() HealthCheckOptions {
	return HealthCheckOptions{
		Interval:            human.Duration{Duration: 15 * time.Second},
		SuccessfulResponses: 3,
		FailedResponses:     3,
	}
}

// APIClientOptions configures buffered and streaming requests to the raw block,
// beacon state, and execution payload envelope endpoints.
//
// It does not configure the go-eth2-client used for typed endpoints or the
// internal JSON client used for peer, identity, and deposit snapshot requests.
// Use AddGoEth2ClientParams for the typed client. Use DefaultAPIClientOptions
// as the base when overriding individual fields.
type APIClientOptions struct {
	// AcceptEncoding requests a specific wire encoding for raw responses. Empty
	// leaves negotiation to net/http, which may transparently decompress gzip.
	AcceptEncoding string
	// Timeout bounds the entire request, including reading the response body.
	//
	// http.Client.Timeout keeps running after the response headers arrive, so it
	// also bounds any read the caller performs on a response body it owns. Set it
	// to zero to leave the body read governed by the request context.
	Timeout human.Duration
	// ResponseHeaderTimeout bounds the wait for response headers after the request
	// has been written. It never bounds reading the response body. Zero disables
	// it.
	ResponseHeaderTimeout human.Duration
	// DialTimeout bounds establishing the TCP connection. Zero disables it.
	DialTimeout human.Duration
	// TLSHandshakeTimeout bounds the TLS handshake. Zero disables it.
	TLSHandshakeTimeout human.Duration
	// MaxIdleConnsPerHost caps idle connections kept per host. Zero uses
	// http.DefaultMaxIdleConnsPerHost.
	MaxIdleConnsPerHost int
	// MaxConnsPerHost limits total connections per host, including connections
	// that are dialing, active, or idle. Excess HTTP/1 requests wait for a
	// connection; HTTP/2 streams can multiplex on one connection. Zero removes
	// the limit.
	MaxConnsPerHost int
	// MaxConcurrentRequests limits raw requests from transport start until the
	// response body is closed. Excess requests wait for a slot, including when
	// HTTP/2 multiplexes them on one connection. With no client timeout or
	// context deadline, that wait is unbounded. Zero removes the limit.
	MaxConcurrentRequests int
}

// DefaultAPIClientOptions returns bounded defaults for the raw API client.
func DefaultAPIClientOptions() APIClientOptions {
	return APIClientOptions{
		Timeout:               human.Duration{Duration: 10 * time.Minute},
		DialTimeout:           human.Duration{Duration: 30 * time.Second},
		TLSHandshakeTimeout:   human.Duration{Duration: 10 * time.Second},
		MaxConnsPerHost:       32,
		MaxConcurrentRequests: 32,
	}
}

// SetAPIClientOptions sets the raw API client options.
func (o *Options) SetAPIClientOptions(opts APIClientOptions) *Options {
	o.APIClient = &opts

	return o
}

// SetAPIClientAcceptEncoding sets the wire encoding requested by the raw API
// client while preserving its other options.
func (o *Options) SetAPIClientAcceptEncoding(encoding string) *Options {
	if o.APIClient == nil {
		opts := DefaultAPIClientOptions()
		o.APIClient = &opts
	}

	o.APIClient.AcceptEncoding = encoding

	return o
}

// SetAPIClientTimeout sets the total request timeout for the raw API client.
//
// Pass zero to remove the bound, which is what a caller streaming a large
// response body wants: the read is then governed by the request context rather
// than by a fixed ceiling it cannot see.
func (o *Options) SetAPIClientTimeout(timeout time.Duration) *Options {
	if o.APIClient == nil {
		opts := DefaultAPIClientOptions()
		o.APIClient = &opts
	}

	o.APIClient.Timeout = human.Duration{Duration: timeout}

	return o
}

func (o *Options) apiClientOptions() APIClientOptions {
	if o.APIClient == nil {
		return DefaultAPIClientOptions()
	}

	return *o.APIClient
}

func (a APIClientOptions) transport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   a.DialTimeout.Duration,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   a.MaxIdleConnsPerHost,
		MaxConnsPerHost:       a.MaxConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   a.TLSHandshakeTimeout.Duration,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: a.ResponseHeaderTimeout.Duration,
	}
}

func (a APIClientOptions) httpClient() *http.Client {
	transport := a.transport()

	var roundTripper http.RoundTripper = transport
	if a.AcceptEncoding != "" {
		roundTripper = &acceptEncodingRoundTripper{
			next:  roundTripper,
			value: a.AcceptEncoding,
		}
	}

	if a.MaxConcurrentRequests > 0 {
		roundTripper = newConcurrencyLimitedRoundTripper(roundTripper, a.MaxConcurrentRequests)
	}

	return &http.Client{
		Timeout:   a.Timeout.Duration,
		Transport: roundTripper,
	}
}

type acceptEncodingRoundTripper struct {
	next  http.RoundTripper
	value string
}

func (t *acceptEncodingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Accept-Encoding", t.value)

	return t.next.RoundTrip(request)
}

func (t *acceptEncodingRoundTripper) CloseIdleConnections() {
	closeIdleConnections(t.next)
}

type concurrencyLimitedRoundTripper struct {
	next  http.RoundTripper
	slots chan struct{}
}

func newConcurrencyLimitedRoundTripper(next http.RoundTripper, limit int) *concurrencyLimitedRoundTripper {
	return &concurrencyLimitedRoundTripper{
		next:  next,
		slots: make(chan struct{}, limit),
	}
}

func (t *concurrencyLimitedRoundTripper) RoundTrip(request *http.Request) (_ *http.Response, err error) {
	select {
	case t.slots <- struct{}{}:
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}

	transferred := false
	defer func() {
		if !transferred {
			<-t.slots
		}
	}()

	response, err := t.next.RoundTrip(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}

		return nil, err
	}

	if response == nil || response.Body == nil {
		return nil, errors.New("raw HTTP transport returned a nil response body")
	}

	response.Body = &concurrencyLimitedBody{
		ReadCloser: response.Body,
		release: func() {
			<-t.slots
		},
	}
	transferred = true

	return response, nil
}

func (t *concurrencyLimitedRoundTripper) CloseIdleConnections() {
	closeIdleConnections(t.next)
}

type concurrencyLimitedBody struct {
	io.ReadCloser
	once     sync.Once
	closeErr error
	release  func()
}

func (b *concurrencyLimitedBody) Close() error {
	b.once.Do(func() {
		defer b.release()

		b.closeErr = b.ReadCloser.Close()
	})

	return b.closeErr
}

func closeIdleConnections(roundTripper http.RoundTripper) {
	if transport, ok := roundTripper.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}

// AddGoEth2ClientParams adds the given parameters to the options.
func (o *Options) AddGoEth2ClientParams(params ...ehttp.Parameter) *Options {
	o.GoEth2ClientParams = append(o.GoEth2ClientParams, params...)

	return o
}

// GetGoEth2ClientParams returns the parameters for the go-eth2-client.
func (o *Options) GetGoEth2ClientParams() []ehttp.Parameter {
	return o.GoEth2ClientParams
}
