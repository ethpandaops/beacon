package state

import (
	"testing"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

const testKeySlotDurationMs = "SLOT_DURATION_MS"

func TestSpecPrefersSlotDurationMs(t *testing.T) {
	spec := NewSpec(map[string]any{
		"SECONDS_PER_SLOT":    12 * time.Second,
		testKeySlotDurationMs: uint64(6000),
	})

	if got := spec.SecondsPerSlot.AsDuration(); got != 6*time.Second {
		t.Fatalf("got %v, want 6s", got)
	}
}

func TestSpecSlotDurationMsWithoutSecondsPerSlot(t *testing.T) {
	spec := NewSpec(map[string]any{testKeySlotDurationMs: "12000"})

	if got := spec.SecondsPerSlot.AsDuration(); got != 12*time.Second {
		t.Fatalf("got %v, want 12s", got)
	}
}

func TestSpecGetSlotDuration(t *testing.T) {
	spec := NewSpec(map[string]any{
		testKeySlotDurationMs:   uint64(12000),
		"SLOT_DURATION_MS_HEZE": uint64(9000),
		"HEZE_FORK_EPOCH":       uint64(5),
	})

	for epoch, want := range map[phase0.Epoch]time.Duration{0: 12 * time.Second, 4: 12 * time.Second, 5: 9 * time.Second, 600: 9 * time.Second} {
		if got := spec.GetSlotDuration(epoch); got != want {
			t.Fatalf("epoch %d: got %v, want %v", epoch, got, want)
		}
	}
}

func TestSpecGetSlotDurationWithoutHeze(t *testing.T) {
	spec := NewSpec(map[string]any{testKeySlotDurationMs: uint64(12000)})

	if got := spec.GetSlotDuration(1_000); got != 12*time.Second {
		t.Fatalf("got %v, want 12s", got)
	}
}
