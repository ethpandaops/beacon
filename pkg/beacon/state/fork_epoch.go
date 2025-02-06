package state

import (
	"errors"
	"sort"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

var (
	// ForkOrder is the canonical order of the forks.
	ForkOrder = []spec.DataVersion{
		spec.DataVersionPhase0,
		spec.DataVersionAltair,
		spec.DataVersionBellatrix,
		spec.DataVersionCapella,
		spec.DataVersionDeneb,
		spec.DataVersionElectra,
	}
)

// ForkEpoch is a beacon fork that activates at a specific epoch.
type ForkEpoch struct {
	Epoch   phase0.Epoch     `yaml:"epoch"`
	Version string           `json:"version"`
	Name    spec.DataVersion `json:"name"`
}

// Active returns true if the fork is active at the given epoch.
func (f *ForkEpoch) Active(epoch phase0.Epoch) bool {
	return epoch >= f.Epoch
}

// ForkEpochs is a list of forks that activate at specific epochs.
type ForkEpochs []*ForkEpoch

// Active returns a list of forks that are active at the given epoch.
func (f *ForkEpochs) Active(epoch phase0.Epoch) []*ForkEpoch {
	activated := []*ForkEpoch{}

	for _, fork := range *f {
		if fork.Active(epoch) {
			activated = append(activated, fork)
		}
	}

	// Sort them by our fork order since multiple forks can be activated on the same epoch.
	// For example, a non-phase0 genesis.
	sort.Slice(activated, func(i, j int) bool {
		return f.IndexOf(activated[i].Name) < f.IndexOf(activated[j].Name)
	})

	return activated
}

// Scheduled returns the scheduled forks at the given epoch.
func (f *ForkEpochs) Scheduled(epoch phase0.Epoch) []*ForkEpoch {
	scheduled := []*ForkEpoch{}

	for _, fork := range *f {
		if !fork.Active(epoch) {
			scheduled = append(scheduled, fork)
		}
	}

	return scheduled
}

// CurrentFork returns the current fork at the given epoch.
func (f *ForkEpochs) CurrentFork(epoch phase0.Epoch) (*ForkEpoch, error) {
	found := false

	largest := &ForkEpoch{
		Epoch: 0,
	}

	active := f.Active(epoch)
	for _, fork := range active {
		if fork.Epoch >= largest.Epoch {
			found = true

			largest = fork
		}
	}

	if !found {
		return &ForkEpoch{}, errors.New("no active fork")
	}

	return largest, nil
}

// PreviousFork returns the previous fork at the given epoch.
func (f *ForkEpochs) PreviousFork(epoch phase0.Epoch) (*ForkEpoch, error) {
	if len(*f) == 1 {
		return f.CurrentFork(epoch)
	}

	current, err := f.CurrentFork(epoch)
	if err != nil {
		return nil, err
	}

	found := false

	largest := &ForkEpoch{
		Epoch: 0,
	}

	for _, fork := range f.Active(epoch) {
		if fork.Active(epoch) && fork.Name != current.Name && fork.Epoch > largest.Epoch {
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
		if fork.Name.String() == name {
			return fork, nil
		}
	}

	return &ForkEpoch{}, errors.New("no fork at epoch")
}

// AsScheduledForks returns the forks as scheduled forks.
func (f *ForkEpochs) AsScheduledForks() ([]*ScheduledFork, error) {
	return ForkScheduleFromForkEpochs(*f)
}

// IndexOf returns the index of the given data version in the fork order.
func (f *ForkEpochs) IndexOf(name spec.DataVersion) int {
	for i, version := range ForkOrder {
		if version == name {
			return i
		}
	}
	return -1 // Return -1 if the name is not found
}
