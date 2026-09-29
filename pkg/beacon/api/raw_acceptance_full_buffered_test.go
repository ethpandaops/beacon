//go:build acceptance && acceptance_full_buffered && linux

package api

import (
	"context"
	"testing"
)

// Run this manual baseline on a host with at least 10 GiB available:
//
//	go test -tags "acceptance acceptance_full_buffered" ./pkg/beacon/api -run TestAcceptanceFullBufferedBaseline -count=1 -timeout=20m
//
// It is excluded from CI because all fifty payloads remain live simultaneously.
func TestAcceptanceFullBufferedBaseline(t *testing.T) {
	pinGCSettings(t)

	ctx, cancel := context.WithTimeout(t.Context(), acceptanceTimeout)
	defer cancel()

	report := measureBuffered(ctx, t, streamConcurrency, beaconStatePayloadBytes)
	report.log(t, "50 concurrent buffered beacon states")

	liveFloor := uint64(beaconStatePayloadBytes) * streamConcurrency * 9 / 10
	if report.peakDelta() < liveFloor {
		t.Errorf(
			"buffered peak heap delta %s is below live payload floor %s",
			mib(report.peakDelta()), mib(liveFloor),
		)
	}
}
