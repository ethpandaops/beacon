package state

import (
	"math"
	"sort"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/spf13/cast"
)

// Spec keys of the EIP-8198 slot duration configuration.
const (
	specKeySlotDurationMs       = "SLOT_DURATION_MS"
	specKeySlotDurationSchedule = "SLOT_DURATION_SCHEDULE"
	specKeyEpoch                = "EPOCH"
)

// millisecondsToDuration converts a millisecond count from the spec into a
// time.Duration. It reports false for zero or values that overflow.
func millisecondsToDuration(ms uint64) (time.Duration, bool) {
	if ms == 0 || ms > uint64(math.MaxInt64/int64(time.Millisecond)) {
		return 0, false
	}

	return time.Duration(ms) * time.Millisecond, true
}

// SlotDurationScheduleEntry is one entry of the EIP-8198 SLOT_DURATION_SCHEDULE:
// slots from Epoch on last Duration.
type SlotDurationScheduleEntry struct {
	Epoch    phase0.Epoch
	Duration time.Duration
}

// SlotDurationSchedule is the EIP-8198 slot duration schedule, sorted by epoch
// and starting at epoch 0.
type SlotDurationSchedule []SlotDurationScheduleEntry

// GetSlotDuration returns the slot duration in effect at epoch.
func (s SlotDurationSchedule) GetSlotDuration(epoch phase0.Epoch) time.Duration {
	for i := len(s) - 1; i >= 0; i-- {
		if epoch >= s[i].Epoch {
			return s[i].Duration
		}
	}

	return 0
}

// parseSlotDurationSchedule parses SLOT_DURATION_SCHEDULE from the spec map
// (a list of {EPOCH, SLOT_DURATION_MS} objects). The result starts at epoch
// 0: if the spec has no schedule, or no genesis entry, one is derived from
// genesisDuration. It is empty only if neither is available.
func parseSlotDurationSchedule(raw any, genesisDuration time.Duration) SlotDurationSchedule {
	schedule := SlotDurationSchedule{}

	if entries, ok := raw.([]any); ok {
		for _, entry := range entries {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}

			duration, ok := millisecondsToDuration(cast.ToUint64(entryMap[specKeySlotDurationMs]))
			if !ok {
				continue
			}

			schedule = append(schedule, SlotDurationScheduleEntry{
				Epoch:    phase0.Epoch(cast.ToUint64(entryMap[specKeyEpoch])),
				Duration: duration,
			})
		}
	}

	sort.Slice(schedule, func(i, j int) bool {
		return schedule[i].Epoch < schedule[j].Epoch
	})

	if len(schedule) == 0 || schedule[0].Epoch != 0 {
		if genesisDuration <= 0 {
			return schedule
		}

		schedule = append(SlotDurationSchedule{{Epoch: 0, Duration: genesisDuration}}, schedule...)
	}

	return schedule
}
