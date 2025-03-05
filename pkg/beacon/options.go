package beacon

import (
	"time"

	ehttp "github.com/attestantio/go-eth2-client/http"
	"github.com/ethpandaops/beacon/pkg/human"
)

// Options holds the options for a beacon node.
type Options struct {
	BeaconSubscription BeaconSubscriptionOptions
	HealthCheck        HealthCheckOptions
	PrometheusMetrics  bool
	DetectEmptySlots   bool
	GoEth2ClientParams []ehttp.Parameter
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
	return &Options{
		BeaconSubscription: DefaultDisabledBeaconSubscriptionOptions(),
		HealthCheck:        DefaultHealthCheckOptions(),
		PrometheusMetrics:  true,
		DetectEmptySlots:   false,
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
func DefaultEnabledBeaconSubscriptionOptions() BeaconSubscriptionOptions {
	return BeaconSubscriptionOptions{
		Enabled: true,
		Topics: []string{
			topicAttestation,
			topicSingleAttestation,
			topicBlock,
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

// AddGoEth2ClientParams adds the given parameters to the options.
func (o *Options) AddGoEth2ClientParams(params ...ehttp.Parameter) *Options {
	o.GoEth2ClientParams = append(o.GoEth2ClientParams, params...)

	return o
}

// GetGoEth2ClientParams returns the parameters for the go-eth2-client.
func (o *Options) GetGoEth2ClientParams() []ehttp.Parameter {
	return o.GoEth2ClientParams
}
