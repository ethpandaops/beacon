package beacon

import (
	"time"

	"github.com/samcm/beacon/human"
)

// Options holds the options for a beacon node.
type Options struct {
	BeaconSubscription  BeaconSubscriptionOptions
	HealthCheck         HealthCheckOptions
	FetchProposerDuties bool
	PrometheusMetrics   bool
	DetectEmptySlots    bool
}

// EnableFetchingProposerDuties enables fetching proposer duties.
func (o *Options) EnableFetchingProposerDuties() *Options {
	o.FetchProposerDuties = true

	return o
}

// DisableFetchingProposerDuties disables fetching proposer duties.
func (o *Options) DisableFetchingProposerDuties() *Options {
	o.FetchProposerDuties = false

	return o
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
		BeaconSubscription:  DefaultDisabledBeaconSubscriptionOptions(),
		HealthCheck:         DefaultHealthCheckOptions(),
		FetchProposerDuties: true,
		PrometheusMetrics:   true,
		DetectEmptySlots:    false,
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
			topicBlock,
			topicChainReorg,
			topicFinalizedCheckpoint,
			topicHead,
			topicVoluntaryExit,
			topicContributionAndProof,
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
