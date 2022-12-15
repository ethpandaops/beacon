package beacon

import (
	"time"
)

// Health tracks the health status of the beacon node.
type Health struct {
	healthy bool

	failures  int
	successes int

	failThreshold    int
	successThreshold int

	lastCheck time.Time

	failTotal    uint64
	successTotal uint64
}

// NewHealth creates a new health tracker.
func NewHealth(successThreshold, failThreshold int) *Health {
	return &Health{
		failures:  0,
		successes: 0,

		failThreshold:    failThreshold,
		successThreshold: successThreshold,

		lastCheck: time.Time{},

		failTotal:    0,
		successTotal: 0,
	}
}

// RecordFail records a failure.
func (n *Health) RecordFail(err error) {
	n.failTotal++
	n.lastCheck = time.Now()
	n.failures++
	n.successes = 0

	if n.failures >= n.failThreshold {
		n.healthy = false
	}
}

// RecordSuccess records a success.
func (n *Health) RecordSuccess() {
	n.successTotal++
	n.lastCheck = time.Now()
	n.successes++
	n.failures = 0

	if n.successes >= n.successThreshold {
		n.healthy = true
	}
}

// Healthy returns true if the node is healthy.
func (n Health) Healthy() bool {
	return n.healthy
}

// FailedTotal returns the total number of failures.
func (n Health) FailedTotal() uint64 {
	return n.failTotal
}

// SuccessTotal returns the total number of successes.
func (n Health) SuccessTotal() uint64 {
	return n.successTotal
}
