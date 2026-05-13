package state_test

import (
	"testing"

	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/go-eth2-client/spec"
)

func TestForkOrderIncludesAllSpecDataVersions(t *testing.T) {
	for i := 0; i < 1000; i++ {
		v := spec.DataVersion(i)
		if v.String() == "unknown" {
			continue
		}

		found := false
		for _, fv := range state.ForkOrder {
			if fv == v {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("ForkOrder missing version: %v", v)
		}
	}
}
