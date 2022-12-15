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
	// Sort them by Epoch.
	sort.Slice(forks, func(i, j int) bool {
		return (forks)[i].Epoch < (forks)[j].Epoch
	})

	scheduled := []*ScheduledFork{}
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
