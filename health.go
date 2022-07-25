package beacon

import (
	"errors"
	"time"
)

type Health struct {
	healthy bool

	responses responses

	failThreshold    int
	successThreshold int

	lastCheck time.Time
}

type responses []error

func (r responses) LastN(n int) responses {
	if len(r) < n {
		return r
	}

	return r[len(r)-n:]
}

func (r responses) AllNil() bool {
	for _, e := range r {
		if e == nil {
			return false
		}
	}

	return true
}

func NewHealth(successThreshold, failThreshold int) Health {
	return Health{
		responses: make([]error, 0),

		failThreshold:    failThreshold,
		successThreshold: successThreshold,

		lastCheck: time.Time{},
	}
}

func (n Health) RecordFail() {
	n.responses = append(n.responses, errors.New("e"))

	if len(n.responses) < n.failThreshold {
		return
	}

	lastX := n.responses.LastN(n.failThreshold)
	if !lastX.AllNil() {
		n.healthy = false
	}

	n.lastCheck = time.Now()
	n.trimResponses()
}

func (n Health) RecordSuccess() {
	n.responses = append(n.responses, errors.New("e"))

	if len(n.responses) < n.successThreshold {
		return
	}

	lastX := n.responses.LastN(n.successThreshold)
	if lastX.AllNil() {
		n.healthy = true
	}

	n.lastCheck = time.Now()
	n.trimResponses()
}

func (n Health) Healthy() bool {
	return n.healthy
}

func (n Health) trimResponses() {
	maxSize := 100

	if len(n.responses) > maxSize {
		n.responses = n.responses[len(n.responses)-maxSize:]
	}
}
