package state

import (
	"testing"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

func TestNewSpecSlotDurationSchedule(t *testing.T) {
	// Values as go-eth2-client parses /eth/v1/config/spec.
	spec := NewSpec(map[string]any{
		"SECONDS_PER_SLOT": 12 * time.Second,
		"SLOT_DURATION_MS": uint64(12000),
		"SLOTS_PER_EPOCH":  uint64(32),
		"SLOT_DURATION_SCHEDULE": []any{
			map[string]any{"EPOCH": uint64(4), "SLOT_DURATION_MS": uint64(8000)},
			map[string]any{"EPOCH": uint64(0), "SLOT_DURATION_MS": uint64(12000)},
			map[string]any{"EPOCH": uint64(2), "SLOT_DURATION_MS": uint64(11000)},
		},
	})

	want := SlotDurationSchedule{
		{Epoch: 0, Duration: 12 * time.Second},
		{Epoch: 2, Duration: 11 * time.Second},
		{Epoch: 4, Duration: 8 * time.Second},
	}

	if len(spec.SlotDurationSchedule) != len(want) {
		t.Fatalf("got %+v, want %+v", spec.SlotDurationSchedule, want)
	}

	for i := range want {
		if spec.SlotDurationSchedule[i] != want[i] {
			t.Fatalf("entry %d: got %+v, want %+v", i, spec.SlotDurationSchedule[i], want[i])
		}
	}

	if got := spec.SlotDurationSchedule.GetSlotDuration(phase0.Epoch(3)); got != 11*time.Second {
		t.Errorf("GetSlotDuration(3) = %v", got)
	}

	if got := spec.SecondsPerSlot.AsDuration(); got != 12*time.Second {
		t.Errorf("SecondsPerSlot = %v", got)
	}
}

func TestNewSpecWithoutSlotDurationSchedule(t *testing.T) {
	spec := NewSpec(map[string]any{
		"SECONDS_PER_SLOT": 12 * time.Second,
		"SLOTS_PER_EPOCH":  uint64(32),
	})

	if len(spec.SlotDurationSchedule) != 1 || spec.SlotDurationSchedule[0] != (SlotDurationScheduleEntry{Epoch: 0, Duration: 12 * time.Second}) {
		t.Fatalf("got %+v", spec.SlotDurationSchedule)
	}
}

func TestNewSpecSlotDurationMsOnly(t *testing.T) {
	// SECONDS_PER_SLOT is deprecated; nodes may only serve SLOT_DURATION_MS.
	spec := NewSpec(map[string]any{
		"SLOT_DURATION_MS": uint64(6000),
		"SLOTS_PER_EPOCH":  uint64(8),
	})

	if got := spec.SecondsPerSlot.AsDuration(); got != 6*time.Second {
		t.Fatalf("SecondsPerSlot = %v", got)
	}

	if len(spec.SlotDurationSchedule) != 1 || spec.SlotDurationSchedule[0].Duration != 6*time.Second {
		t.Fatalf("got %+v", spec.SlotDurationSchedule)
	}
}

func TestNewSpecScheduleWithoutGenesisEntry(t *testing.T) {
	spec := NewSpec(map[string]any{
		"SLOT_DURATION_MS": uint64(12000),
		"SLOT_DURATION_SCHEDULE": []any{
			map[string]any{"EPOCH": uint64(2), "SLOT_DURATION_MS": uint64(11000)},
		},
	})

	if len(spec.SlotDurationSchedule) != 2 || spec.SlotDurationSchedule[0] != (SlotDurationScheduleEntry{Epoch: 0, Duration: 12 * time.Second}) {
		t.Fatalf("got %+v", spec.SlotDurationSchedule)
	}
}
