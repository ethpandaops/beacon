package beacon

import (
	"time"

	"github.com/samcm/beacon/human"
)

type Options struct {
	BeaconSubscription BeaconSubscriptionOptions
	HealthCheck        HealthCheckOptions
}

func DefaultOptions() *Options {
	return &Options{
		BeaconSubscription: DefaultDisabledBeaconSubscriptionOptions(),
		HealthCheck:        DefaultHealthCheckOptions(),
	}
}

type BeaconSubscriptionOptions struct {
	Enabled                       bool
	InactivityResubscribeInterval human.Duration
	Topics                        EventTopics
}

func (b *BeaconSubscriptionOptions) Disable() *BeaconSubscriptionOptions {
	b.Enabled = false

	return b
}

func (b *BeaconSubscriptionOptions) Enable() *BeaconSubscriptionOptions {
	b.Enabled = true

	return b
}

func DefaultDisabledBeaconSubscriptionOptions() BeaconSubscriptionOptions {
	return BeaconSubscriptionOptions{
		Enabled:                       false,
		InactivityResubscribeInterval: human.Duration{Duration: 9999 * time.Hour},
		Topics:                        []string{},
	}
}

func DefaultEnabledBeaconSubscriptionOptions() BeaconSubscriptionOptions {
	return BeaconSubscriptionOptions{
		Enabled:                       false,
		InactivityResubscribeInterval: human.Duration{Duration: 15 * time.Minute},
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

func (o *Options) EnableDefaultBeaconSubscription() *Options {
	o.BeaconSubscription = DefaultEnabledBeaconSubscriptionOptions()

	return o
}

type HealthCheckOptions struct {
	// Interval is the interval at which the health check will be run.
	Interval human.Duration
	// SuccessThreshold is the number of consecutive successful health checks required before the node is considered healthy.
	SuccessfulResponses int
	// FailureThreshold is the number of consecutive failed health checks required before the node is considered unhealthy.
	FailedResponses int
}

func DefaultHealthCheckOptions() HealthCheckOptions {
	return HealthCheckOptions{
		Interval:            human.Duration{Duration: 15 * time.Second},
		SuccessfulResponses: 3,
		FailedResponses:     3,
	}
}
