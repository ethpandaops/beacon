package state

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// ForkEpoch is a beacon fork that activates at a specific epoch.
type ForkEpoch struct {
	Epoch   phase0.Epoch `yaml:"epoch"`
	Version string       `json:"version"`
	Name    string       `json:"name"`
}

// Active returns true if the fork is active at the given slot.
func (f *ForkEpoch) Active(slot, slotsPerEpoch phase0.Slot) bool {
	return phase0.Epoch(int(slot)/int(slotsPerEpoch)) >= f.Epoch
}

// ForkEpochs is a list of forks that activate at specific epochs.
type ForkEpochs []*ForkEpoch

// Active returns a list of forks that are active at the given slot.
func (f *ForkEpochs) Active(slot, slotsPerEpoch phase0.Slot) []*ForkEpoch {
	activated := []*ForkEpoch{}

	for _, fork := range *f {
		if fork.Active(slot, slotsPerEpoch) {
			activated = append(activated, fork)
		}
	}

	return activated
}

// CurrentFork returns the current fork at the given slot.
func (f *ForkEpochs) Scheduled(slot, slotsPerEpoch phase0.Slot) []*ForkEpoch {
	scheduled := []*ForkEpoch{}

	for _, fork := range *f {
		if !fork.Active(slot, slotsPerEpoch) {
			scheduled = append(scheduled, fork)
		}
	}

	return scheduled
}

// CurrentFork returns the current fork at the given slot.
func (f *ForkEpochs) CurrentFork(slot, slotsPerEpoch phase0.Slot) (*ForkEpoch, error) {
	found := false

	largest := &ForkEpoch{
		Epoch: 0,
	}

	for _, fork := range f.Active(slot, slotsPerEpoch) {
		if fork.Active(slot, slotsPerEpoch) && fork.Epoch >= largest.Epoch {
			found = true

			largest = fork
		}
	}

	if !found {
		return &ForkEpoch{}, errors.New("no active fork")
	}

	return largest, nil
}

// PreviousFork returns the previous fork at the given slot.
func (f *ForkEpochs) PreviousFork(slot, slotsPerEpoch phase0.Slot) (*ForkEpoch, error) {
	if len(*f) == 1 {
		return f.CurrentFork(slot, slotsPerEpoch)
	}

	current, err := f.CurrentFork(slot, slotsPerEpoch)
	if err != nil {
		return nil, err
	}

	found := false

	largest := &ForkEpoch{
		Epoch: 0,
	}

	for _, fork := range f.Active(slot, slotsPerEpoch) {
		if fork.Active(slot, slotsPerEpoch) && fork.Name != current.Name && fork.Epoch > largest.Epoch {
			found = true

			largest = fork
		}
	}

	if !found {
		return &ForkEpoch{}, errors.New("no previous fork")
	}

	return largest, nil
}

// GetByName returns the fork with the given name.
func (f *ForkEpochs) GetByName(name string) (*ForkEpoch, error) {
	for _, fork := range *f {
		if fork.Name == name {
			return fork, nil
		}
	}

	return &ForkEpoch{}, errors.New("no fork at epoch")
}

// AsScheduledForks returns the forks as scheduled forks.
func (f *ForkEpochs) AsScheduledForks() ([]*ScheduledFork, error) {
	return ForkScheduleFromForkEpochs(*f)
}
