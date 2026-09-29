package beacon

import (
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

const (
	testKeySlotDurationMs = "SLOT_DURATION_MS"
	testKeySlotsPerEpoch  = "SLOTS_PER_EPOCH"
)

func TestNewWallclockFollowsSlotDurationSchedule(t *testing.T) {
	genesis := time.Unix(1_700_000_000, 0)
	spec := state.NewSpec(map[string]any{
		testKeySlotDurationMs: uint64(12000),
		testKeySlotsPerEpoch:  uint64(32),
		"SLOT_DURATION_SCHEDULE": []any{
			map[string]any{"EPOCH": uint64(0), testKeySlotDurationMs: uint64(12000)},
			map[string]any{"EPOCH": uint64(2), testKeySlotDurationMs: uint64(11000)},
		},
	})

	wallclock := newWallclock(genesis, &spec)
	defer wallclock.Stop()

	slot := wallclock.Slots().FromNumber(65)
	if want := genesis.Add(64*12*time.Second + 11*time.Second); !slot.TimeWindow().Start().Equal(want) {
		t.Fatalf("slot 65 starts at %v, want %v", slot.TimeWindow().Start(), want)
	}

	slot, epoch, _ := wallclock.FromTime(genesis.Add(64*12*time.Second + 32*11*time.Second))
	if slot.Number() != 96 || epoch.Number() != 3 {
		t.Fatalf("got slot %d epoch %d, want slot 96 epoch 3", slot.Number(), epoch.Number())
	}
}

func TestNewWallclockWithoutSchedule(t *testing.T) {
	genesis := time.Unix(1_700_000_000, 0)
	spec := state.NewSpec(map[string]any{
		"SECONDS_PER_SLOT":   12 * time.Second,
		testKeySlotsPerEpoch: uint64(32),
	})

	wallclock := newWallclock(genesis, &spec)
	defer wallclock.Stop()

	if slot := wallclock.Slots().FromTime(genesis.Add(100 * 12 * time.Second)); slot.Number() != 100 {
		t.Fatalf("got slot %d, want 100", slot.Number())
	}
}
