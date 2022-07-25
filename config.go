package beacon

import "github.com/samcm/beacon/human"

type Config struct {
	// Name is the human-readable name of the node.
	Name string `yaml:"name"`
	// Address is the address of the node.
	Addr string `yaml:"addr"`
	// EventTopics contains the list of topics to subscribe to for events.
	EventTopics EventTopics `yaml:"event_topics"`
	// HealthCheckConfig is the health check configuration.
	HealthCheckConfig HealthCheckConfig `yaml:"health_check"`
}

type EventTopics []string

type HealthCheckConfig struct {
	// Interval is the interval at which the health check will be run.
	Interval human.Duration `yaml:"interval"`
	// SuccessThreshold is the number of consecutive successful health checks required before the node is considered healthy.
	SuccessfulResponses int `yaml:"successful_responses"`
	// FailureThreshold is the number of consecutive failed health checks required before the node is considered unhealthy.
	FailedResponses int `yaml:"failed_responses"`
}

func (e EventTopics) Exists(topic string) bool {
	for _, t := range e {
		if t == topic {
			return true
		}
	}

	return false
}
