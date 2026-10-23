package beacon

import (
	"testing"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
)

const testKeySlotsPerEpoch = "SLOTS_PER_EPOCH"

func TestNewWallclockFollowsHezeSlotDuration(t *testing.T) {
	genesis := time.Unix(1_792_577_902, 0)
	spec := state.NewSpec(map[string]any{
		"SECONDS_PER_SLOT":      12 * time.Second,
		"SLOT_DURATION_MS":      uint64(12000),
		"SLOT_DURATION_MS_HEZE": uint64(9000),
		"HEZE_FORK_EPOCH":       uint64(5),
		testKeySlotsPerEpoch:    uint64(32),
	})

	wallclock := newWallclock(genesis, &spec)
	defer wallclock.Stop()

	forkTime := genesis.Add(160 * 12 * time.Second)

	if slot := wallclock.Slots().FromNumber(161); !slot.TimeWindow().Start().Equal(forkTime.Add(9 * time.Second)) {
		t.Fatalf("slot 161 starts at %v, want %v", slot.TimeWindow().Start(), forkTime.Add(9*time.Second))
	}

	slot, epoch, err := wallclock.FromTime(forkTime.Add(18799 * 9 * time.Second))
	if err != nil {
		t.Fatal(err)
	}

	if slot.Number() != 18959 || epoch.Number() != 592 {
		t.Fatalf("got slot %d epoch %d, want slot 18959 epoch 592", slot.Number(), epoch.Number())
	}
}

func TestNewWallclockHezeWithoutSlotDurationChange(t *testing.T) {
	genesis := time.Unix(1_792_494_000, 0)
	spec := state.NewSpec(map[string]any{
		"SLOT_DURATION_MS":      uint64(12000),
		"SLOT_DURATION_MS_HEZE": uint64(12000),
		"HEZE_FORK_EPOCH":       uint64(5),
		testKeySlotsPerEpoch:    uint64(32),
	})

	wallclock := newWallclock(genesis, &spec)
	defer wallclock.Stop()

	if slot := wallclock.Slots().FromTime(genesis.Add(21900 * 12 * time.Second)); slot.Number() != 21900 {
		t.Fatalf("got slot %d, want 21900", slot.Number())
	}
}

func TestNewWallclockWithoutHeze(t *testing.T) {
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
