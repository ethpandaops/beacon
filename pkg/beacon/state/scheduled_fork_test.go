package state

import (
	"sync"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
)

func TestForkScheduleFromForkEpochs_DoesNotMutateInput(t *testing.T) {
	forks := ForkEpochs{
		{Epoch: 74240, Name: spec.DataVersionAltair, Version: "0x01000000"},
		{Epoch: 0, Name: spec.DataVersionPhase0, Version: "0x00000000"},
		{Epoch: 194048, Name: spec.DataVersionBellatrix, Version: "0x02000000"},
	}

	before := make([]spec.DataVersion, len(forks))
	for i, f := range forks {
		before[i] = f.Name
	}

	scheduled, err := ForkScheduleFromForkEpochs(forks)
	assert.NoError(t, err)
	assert.Len(t, scheduled, 3)

	after := make([]spec.DataVersion, len(forks))
	for i, f := range forks {
		after[i] = f.Name
	}

	assert.Equal(t, before, after, "ForkScheduleFromForkEpochs must not reorder the caller's slice")

	// The returned schedule should still be sorted ascending by epoch,
	// independent of the input order.
	assert.Equal(t, "0", scheduled[0].Epoch)
	assert.Equal(t, "74240", scheduled[1].Epoch)
	assert.Equal(t, "194048", scheduled[2].Epoch)
}

func TestForkScheduleFromForkEpochs_ConcurrentWithReader(t *testing.T) {
	forks := ForkEpochs{}
	for i := range 50 {
		forks = append(forks, &ForkEpoch{Epoch: phase0.Epoch(i * 1000), Name: spec.DataVersionPhase0})
	}

	var wg sync.WaitGroup

	// Writer: repeatedly builds a schedule from the shared slice, the same
	// way a consumer rendering a fork schedule page would.
	for range 4 {
		wg.Go(func() {
			for range 50 {
				_, err := ForkScheduleFromForkEpochs(forks)
				assert.NoError(t, err)
			}
		})
	}

	// Reader: the same access pattern ForkMetrics.calculateCurrent uses.
	for range 4 {
		wg.Go(func() {
			for range 50 {
				for _, fork := range forks {
					_ = fork.Name.String()
				}
			}
		})
	}

	wg.Wait()
}
