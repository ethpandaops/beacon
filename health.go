package beacon

import (
	"time"
)

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

func (n *Health) RecordFail(err error) {
	n.failTotal++
	n.lastCheck = time.Now()
	n.failures++
	n.successes = 0

	if n.failures >= n.failThreshold {
		n.healthy = false
	}
}

func (n *Health) RecordSuccess() {
	n.successTotal++
	n.lastCheck = time.Now()
	n.successes++
	n.failures = 0

	if n.successes >= n.successThreshold {
		n.healthy = true
	}
}

func (n Health) Healthy() bool {
	return n.healthy
}

func (n Health) FailedTotal() uint64 {
	return n.failTotal
}

func (n Health) SuccessTotal() uint64 {
	return n.successTotal
}
