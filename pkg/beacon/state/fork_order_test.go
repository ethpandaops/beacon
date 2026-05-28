package state_test

import (
	"slices"
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/go-eth2-client/spec"
)

func TestForkOrderIncludesAllSpecDataVersions(t *testing.T) {
	for i := range 1000 {
		v := spec.DataVersion(i)
		if v.String() == "unknown" {
			continue
		}

		found := slices.Contains(state.ForkOrder, v)

		if !found {
			t.Errorf("ForkOrder missing version: %v", v)
		}
	}
}
