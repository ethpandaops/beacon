package lightclient_test

import (
	"fmt"
	"testing"

	"encoding/json"
	"reflect"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptimisticUpdateMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name   string
		update lightclient.OptimisticUpdate
	}{
		{
			name: "Basic Update",
			update: lightclient.OptimisticUpdate{
				AttestedHeader: lightclient.LightClientHeader{
					Beacon: lightclient.BeaconBlockHeader{
						Slot:          1234,
						ProposerIndex: 5678,
						ParentRoot:    phase0.Root{0x01},
						StateRoot:     phase0.Root{0x02},
						BodyRoot:      phase0.Root{0x03},
					},
				},
				SyncAggregate: lightclient.SyncAggregate{
					SyncCommitteeBits:      bitfield.Bitvector512{0, 1},
					SyncCommitteeSignature: [96]byte{0x0c},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal
			marshaled, err := json.Marshal(tc.update)
			if err != nil {
				t.Fatalf("Failed to marshal Update: %v", err)
			}

			// Unmarshal
			var unmarshaled lightclient.OptimisticUpdate
			err = json.Unmarshal(marshaled, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal Update: %v", err)
			}

			// Compare
			if !reflect.DeepEqual(tc.update, unmarshaled) {
				t.Errorf("Unmarshaled Update does not match original. Got %+v, want %+v", unmarshaled, tc.update)
			}
		})
	}
}

func TestOptimisticUpdateUnmarshalJSON(t *testing.T) {
	expectedRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	expectedSignature := "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
	expectedBits := "0x01"

	jsonStr := `{
      "attested_header": {
        "beacon": {
          "slot": "1",
          "proposer_index": "1",
          "parent_root": "` + expectedRoot + `",
          "state_root": "` + expectedRoot + `",
          "body_root": "` + expectedRoot + `"
        }
      },
      "sync_aggregate": {
        "sync_committee_bits": "` + expectedBits + `",
        "sync_committee_signature": "` + expectedSignature + `"
      }
    }`

	var update lightclient.OptimisticUpdate
	err := json.Unmarshal([]byte(jsonStr), &update)
	require.NoError(t, err)

	// Check all fields manually
	assert.Equal(t, phase0.Slot(1), update.AttestedHeader.Beacon.Slot)
	assert.Equal(t, phase0.ValidatorIndex(1), update.AttestedHeader.Beacon.ProposerIndex)
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.ParentRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.StateRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.BodyRoot.String())

	assert.Equal(t, expectedBits, fmt.Sprintf("%#x", update.SyncAggregate.SyncCommitteeBits.Bytes()))

	assert.Equal(t, expectedSignature, update.SyncAggregate.SyncCommitteeSignature.String())

	// Marshal back to JSON
	marshaledJSON, err := json.Marshal(update)
	require.NoError(t, err)

	// Unmarshal both JSONs to interfaces for comparison
	var originalData, remarshaledData interface{}
	err = json.Unmarshal([]byte(jsonStr), &originalData)
	require.NoError(t, err)
	err = json.Unmarshal(marshaledJSON, &remarshaledData)
	require.NoError(t, err)

	// Compare the unmarshaled data
	assert.Equal(t, originalData, remarshaledData, "Remarshaled JSON does not match the original")
}
