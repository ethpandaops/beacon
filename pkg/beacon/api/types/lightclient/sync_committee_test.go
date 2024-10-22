package lightclient

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncCommitteeMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name          string
		syncCommittee SyncCommittee
	}{
		{
			name: "Basic SyncCommittee",
			syncCommittee: SyncCommittee{
				Pubkeys: []phase0.BLSPubKey{
					{0x01, 0x23, 0x45},
					{0x67, 0x89, 0xab},
				},
				AggregatePubkey: phase0.BLSPubKey{0xcd, 0xef, 0x01},
			},
		},
		{
			name: "Empty SyncCommittee",
			syncCommittee: SyncCommittee{
				Pubkeys:         []phase0.BLSPubKey{},
				AggregatePubkey: phase0.BLSPubKey{},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			marshaled, err := json.Marshal(tc.syncCommittee)
			if err != nil {
				t.Fatalf("Failed to marshal SyncCommittee: %v", err)
			}

			var unmarshaled SyncCommittee
			err = json.Unmarshal(marshaled, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal SyncCommittee: %v", err)
			}

			if !reflect.DeepEqual(tc.syncCommittee, unmarshaled) {
				t.Errorf("Unmarshaled SyncCommittee does not match original. Got %+v, want %+v", unmarshaled, tc.syncCommittee)
			}
		})
	}
}

func TestSyncCommitteeUnmarshalJSON(t *testing.T) {
	expectedPubkey := "0x93247f2209abcacf57b75a51dafae777f9dd38bc7053d1af526f220a7489a6d3a2753e5f3e8b1cfe39b56f43611df74a"

	jsonStr := `
      {
        "pubkeys": [
          "` + expectedPubkey + `",
          "` + expectedPubkey + `"
        ],
        "aggregate_pubkey": "` + expectedPubkey + `"
      }
    `

	var syncCommittee SyncCommittee
	err := json.Unmarshal([]byte(jsonStr), &syncCommittee)
	require.NoError(t, err)

	assert.Equal(t, expectedPubkey, syncCommittee.AggregatePubkey.String())
	for _, pubkey := range syncCommittee.Pubkeys {
		assert.Equal(t, expectedPubkey, pubkey.String())
	}

	// Test marshalling back to JSON
	marshaled, err := json.Marshal(syncCommittee)
	require.NoError(t, err)

	var unmarshaled SyncCommittee
	err = json.Unmarshal(marshaled, &unmarshaled)
	require.NoError(t, err)
}
