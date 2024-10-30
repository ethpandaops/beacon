package state_test

import (
	"testing"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/stretchr/testify/assert"
)

func TestForkEpochActive(t *testing.T) {
	tests := []struct {
		name     string
		fork     *state.ForkEpoch
		epoch    phase0.Epoch
		expected bool
	}{
		{
			name: "active when epoch is equal",
			fork: &state.ForkEpoch{
				Epoch: 100,
			},
			epoch:    100,
			expected: true,
		},
		{
			name: "active when epoch is greater",
			fork: &state.ForkEpoch{
				Epoch: 100,
			},
			epoch:    101,
			expected: true,
		},
		{
			name: "not active when epoch is less",
			fork: &state.ForkEpoch{
				Epoch: 100,
			},
			epoch:    99,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.fork.Active(test.epoch))
		})
	}
}

func TestForkEpochsActive(t *testing.T) {
	tests := []struct {
		name     string
		forks    state.ForkEpochs
		epoch    phase0.Epoch
		expected []*state.ForkEpoch
	}{
		{
			name: "returns active forks in order",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
			epoch: 201,
			expected: []*state.ForkEpoch{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
		},
		{
			name: "returns only active forks",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
			epoch: 150,
			expected: []*state.ForkEpoch{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.forks.Active(test.epoch))
		})
	}
}

func TestForkEpochsScheduled(t *testing.T) {
	tests := []struct {
		name     string
		forks    state.ForkEpochs
		epoch    phase0.Epoch
		expected []*state.ForkEpoch
	}{
		{
			name: "returns scheduled forks",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
			epoch: 150,
			expected: []*state.ForkEpoch{
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.forks.Scheduled(test.epoch))
		})
	}
}

func TestForkEpochsCurrentFork(t *testing.T) {
	tests := []struct {
		name        string
		forks       state.ForkEpochs
		epoch       phase0.Epoch
		expected    *state.ForkEpoch
		expectError bool
	}{
		{
			name: "returns current fork",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
			epoch: 201,
			expected: &state.ForkEpoch{
				Epoch: 200,
				Name:  spec.DataVersionAltair,
			},
		},
		{
			name:        "errors with no forks",
			forks:       state.ForkEpochs{},
			epoch:       100,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.forks.CurrentFork(test.epoch)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestForkEpochsPreviousFork(t *testing.T) {
	tests := []struct {
		name        string
		forks       state.ForkEpochs
		epoch       phase0.Epoch
		expected    *state.ForkEpoch
		expectError bool
	}{
		{
			name: "returns previous fork",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 200,
					Name:  spec.DataVersionAltair,
				},
			},
			epoch: 201,
			expected: &state.ForkEpoch{
				Epoch: 100,
				Name:  spec.DataVersionPhase0,
			},
		},
		{
			name: "returns current fork with single fork",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
			},
			epoch: 101,
			expected: &state.ForkEpoch{
				Epoch: 100,
				Name:  spec.DataVersionPhase0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.forks.PreviousFork(test.epoch)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestForkEpochsGetByName(t *testing.T) {
	tests := []struct {
		name        string
		forks       state.ForkEpochs
		forkName    string
		expected    *state.ForkEpoch
		expectError bool
	}{
		{
			name: "returns fork by name",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
			},
			forkName: "phase0",
			expected: &state.ForkEpoch{
				Epoch: 100,
				Name:  spec.DataVersionPhase0,
			},
		},
		{
			name: "errors with non-existent fork",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
			},
			forkName:    "invalid",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.forks.GetByName(test.forkName)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestForkEpochsLatestActiveFork(t *testing.T) {
	tests := []struct {
		name        string
		forks       state.ForkEpochs
		epoch       phase0.Epoch
		expected    *state.ForkEpoch
		expectError bool
	}{
		{
			name: "returns latest fork when multiple are active",
			forks: state.ForkEpochs{
				{
					Epoch: 100,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 100,
					Name:  spec.DataVersionAltair,
				},
				{
					Epoch: 100,
					Name:  spec.DataVersionBellatrix,
				},
			},
			epoch: 100,
			expected: &state.ForkEpoch{
				Epoch: 100,
				Name:  spec.DataVersionBellatrix,
			},
		},
		{
			name: "non phase0 genesis",
			forks: state.ForkEpochs{
				{
					Epoch: 0,
					Name:  spec.DataVersionPhase0,
				},
				{
					Epoch: 0,
					Name:  spec.DataVersionAltair,
				},
				{
					Epoch: 0,
					Name:  spec.DataVersionBellatrix,
				},
				{
					Epoch: 0,
					Name:  spec.DataVersionCapella,
				},
				{
					Epoch: 1000,
					Name:  spec.DataVersionDeneb,
				},
			},
			epoch: 100,
			expected: &state.ForkEpoch{
				Epoch: 0,
				Name:  spec.DataVersionCapella,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := test.forks.CurrentFork(test.epoch)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}
