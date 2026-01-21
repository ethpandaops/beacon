package lightclient

import (
	"encoding/json"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinalityUpdateMarshalUnmarshal(t *testing.T) {
	originalUpdate := &FinalityUpdate{
		AttestedHeader: LightClientHeader{
			Beacon: BeaconBlockHeader{
				Slot:          123,
				ProposerIndex: 456,
				ParentRoot:    phase0.Root{0x01},
				StateRoot:     phase0.Root{0x02},
				BodyRoot:      phase0.Root{0x03},
			},
		},
		FinalizedHeader: LightClientHeader{
			Beacon: BeaconBlockHeader{
				Slot:          789,
				ProposerIndex: 101,
				ParentRoot:    phase0.Root{0x04},
				StateRoot:     phase0.Root{0x05},
				BodyRoot:      phase0.Root{0x06},
			},
		},
		FinalityBranch: []phase0.Root{{01, 02}, {03, 04}},
		SyncAggregate: SyncAggregate{
			SyncCommitteeBits:      bitfield.Bitvector512{1, 1, 1, 0, 0, 1},
			SyncCommitteeSignature: [96]byte{0x0a},
		},
		SignatureSlot: 1234,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(originalUpdate)
	require.NoError(t, err)

	// Unmarshal from JSON
	var unmarshaledUpdate FinalityUpdate
	err = json.Unmarshal(jsonData, &unmarshaledUpdate)
	require.NoError(t, err)

	// Compare original and unmarshaled data
	assert.Equal(t, originalUpdate.AttestedHeader, unmarshaledUpdate.AttestedHeader)
	assert.Equal(t, originalUpdate.FinalizedHeader, unmarshaledUpdate.FinalizedHeader)
	assert.Equal(t, originalUpdate.FinalityBranch, unmarshaledUpdate.FinalityBranch)
	assert.Equal(t, originalUpdate.SyncAggregate, unmarshaledUpdate.SyncAggregate)
	assert.Equal(t, originalUpdate.SignatureSlot, unmarshaledUpdate.SignatureSlot)
}

func TestFinalityUpdateUnmarshalPhase0(t *testing.T) {
	expectedRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	expectedSignature := "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"

	jsonData := []byte(`
 		{
			"attested_header": {
				"beacon": {
					"slot": "1",
					"proposer_index": "1",
					"parent_root": "` + expectedRoot + `",
					"state_root": "` + expectedRoot + `",
					"body_root": "` + expectedRoot + `"
				}
			},
			"finalized_header": {
				"beacon": {
					"slot": "1",
					"proposer_index": "1",
					"parent_root": "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2",
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
		}`)

	var update FinalityUpdate
	err := json.Unmarshal(jsonData, &update)
	require.NoError(t, err)

	assert.Equal(t, phase0.Slot(1), update.AttestedHeader.Beacon.Slot)
	assert.Equal(t, phase0.ValidatorIndex(1), update.AttestedHeader.Beacon.ProposerIndex)
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.ParentRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.StateRoot.String())
	assert.Equal(t, expectedRoot, update.AttestedHeader.Beacon.BodyRoot.String())

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

	// Test marshalling back to JSON
	marshaled, err := json.Marshal(update)
	require.NoError(t, err)

	var unmarshaled FinalityUpdate
	err = json.Unmarshal(marshaled, &unmarshaled)
	require.NoError(t, err)
}
