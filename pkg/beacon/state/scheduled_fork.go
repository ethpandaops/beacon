package state

import (
	"fmt"
	"sort"
)

// ScheduledFork is an upcoming fork.
type ScheduledFork struct {
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
	PreviousVersion string `json:"previous_version"`
}

// ForkScheduleFromForkEpochs returns a fork schedule from a list of forks.
func ForkScheduleFromForkEpochs(forks ForkEpochs) ([]*ScheduledFork, error) {
	// Sort a copy by Epoch so we don't reorder the caller's backing array.
	sorted := make(ForkEpochs, len(forks))
	copy(sorted, forks)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Epoch < sorted[j].Epoch
	})

	forks = sorted

	scheduled := make([]*ScheduledFork, 0, len(forks))

	for i, fork := range forks {
		scheduledFork := &ScheduledFork{
			CurrentVersion:  fork.Version,
			Epoch:           fmt.Sprintf("%d", fork.Epoch),
			PreviousVersion: "0x00000000",
		}

		if i > 0 {
			scheduledFork.PreviousVersion = (forks)[i-1].Version
		}

		scheduled = append(scheduled, scheduledFork)
	}

	return scheduled, nil
}
