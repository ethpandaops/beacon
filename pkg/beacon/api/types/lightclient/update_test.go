package lightclient_test

import (
	"testing"

	"encoding/json"
	"reflect"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name   string
		update lightclient.Update
	}{
		{
			name: "Basic Update",
			update: lightclient.Update{
				AttestedHeader: lightclient.LightClientHeader{
					Beacon: lightclient.BeaconBlockHeader{
						Slot:          1234,
						ProposerIndex: 5678,
						ParentRoot:    phase0.Root{0x01},
						StateRoot:     phase0.Root{0x02},
						BodyRoot:      phase0.Root{0x03},
					},
				},
				NextSyncCommittee: lightclient.SyncCommittee{
					Pubkeys:         []phase0.BLSPubKey{{0x04}},
					AggregatePubkey: phase0.BLSPubKey{0x05},
				},
				NextSyncCommitteeBranch: []phase0.Root{{0x06}},
				FinalizedHeader: lightclient.LightClientHeader{
					Beacon: lightclient.BeaconBlockHeader{
						Slot:          5678,
						ProposerIndex: 1234,
						ParentRoot:    phase0.Root{0x07},
						StateRoot:     phase0.Root{0x08},
						BodyRoot:      phase0.Root{0x09},
					},
				},
				FinalityBranch: []phase0.Root{{0x0a}},
				SyncAggregate: lightclient.SyncAggregate{
					SyncCommitteeBits:      bitfield.Bitvector512{0, 1},
					SyncCommitteeSignature: [96]byte{0x0c},
				},
				SignatureSlot: 9876,
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
			var unmarshaled lightclient.Update
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

func TestUpdateUnmarshalJSON(t *testing.T) {
	expectedRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	expectedSignature := "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
	expectedPubkey := "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"

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
      "next_sync_committee": {
        "pubkeys": [
          "` + expectedPubkey + `",
          "` + expectedPubkey + `"
        ],
        "aggregate_pubkey": "` + expectedPubkey + `"
      },
      "next_sync_committee_branch": [
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `"
      ],
      "finalized_header": {
        "beacon": {
          "slot": "1",
          "proposer_index": "1",
          "parent_root": "` + expectedRoot + `",
          "state_root": "` + expectedRoot + `",
          "body_root": "` + expectedRoot + `"
        }
      },
      "finality_branch": [
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `",
        "` + expectedRoot + `"
      ],
      "sync_aggregate": {
        "sync_committee_bits": "0x01",
        "sync_committee_signature": "` + expectedSignature + `"
      },
      "signature_slot": "1"
    }`

	var update lightclient.Update
	err := json.Unmarshal([]byte(jsonStr), &update)
	require.NoError(t, err)

	// Check all fields manually
	assert.Equal(t, phase0.Slot(1), update.AttestedHeader.Beacon.Slot)
	assert.Equal(t, phase0.ValidatorIndex(1), update.AttestedHeader.Beacon.ProposerIndex)
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.ParentRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.StateRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.BodyRoot.String())

	assert.Len(t, update.NextSyncCommittee.Pubkeys, 2)
	for _, pubkey := range update.NextSyncCommittee.Pubkeys {
		assert.Equal(t, expectedPubkey, pubkey.String())
	}
	assert.Equal(t, expectedPubkey, update.NextSyncCommittee.AggregatePubkey.String())

	assert.Len(t, update.NextSyncCommitteeBranch, 5)
	for _, root := range update.NextSyncCommitteeBranch {
		assert.Equal(t, expectedRoot, root.String())
	}

	assert.Equal(t, phase0.Slot(1), update.FinalizedHeader.Beacon.Slot)
	assert.Equal(t, phase0.ValidatorIndex(1), update.FinalizedHeader.Beacon.ProposerIndex)
	assert.Equal(t, expectedRoot, update.FinalizedHeader.Beacon.ParentRoot.String())
	assert.Equal(t, expectedRoot, update.FinalizedHeader.Beacon.StateRoot.String())
	assert.Equal(t, expectedRoot, update.FinalizedHeader.Beacon.BodyRoot.String())

	assert.Len(t, update.FinalityBranch, 6)
	for _, root := range update.FinalityBranch {
		assert.Equal(t, expectedRoot, root.String())
	}

	assert.Equal(t, bitfield.Bitvector512{1}, update.SyncAggregate.SyncCommitteeBits)
	assert.Equal(t, expectedSignature, update.SyncAggregate.SyncCommitteeSignature.String())

	assert.Equal(t, phase0.Slot(1), update.SignatureSlot)

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
