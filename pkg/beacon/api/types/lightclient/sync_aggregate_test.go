package lightclient_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAggregateMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name          string
		syncAggregate lightclient.SyncAggregate
	}{
		{
			name: "Basic SyncAggregate",
			syncAggregate: lightclient.SyncAggregate{
				SyncCommitteeBits:      bitfield.Bitvector512{0, 1, 0, 1, 0},
				SyncCommitteeSignature: phase0.BLSSignature{0x03, 0x04},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal
			marshaled, err := json.Marshal(tc.syncAggregate)
			if err != nil {
				t.Fatalf("Failed to marshal SyncAggregate: %v", err)
			}

			// Unmarshal
			var unmarshaled lightclient.SyncAggregate
			err = json.Unmarshal(marshaled, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal SyncAggregate: %v", err)
			}

			// Compare
			if !reflect.DeepEqual(tc.syncAggregate, unmarshaled) {
				t.Errorf("Unmarshaled SyncAggregate does not match original. Got %+v, want %+v", unmarshaled, tc.syncAggregate)
			}
		})
	}
}

func TestSyncAggregateUnmarshalJSON(t *testing.T) {
	expectedBits := "0x01"
	expectedSignature := "0x1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505cc411d61252fb6cb3fa0017b679f8bb2305b26a285fa2737f175668d0dff91cc1b66ac1fb663c9bc59509846d6ec05345bd908eda73e670af888da41af171505"
	jsonStr := `
      {
		"sync_committee_bits": "` + expectedBits + `",
		"sync_committee_signature": "` + expectedSignature + `"
      }
    `

	var syncAggregate lightclient.SyncAggregate
	err := json.Unmarshal([]byte(jsonStr), &syncAggregate)
	require.NoError(t, err)

	assert.Equal(t, expectedBits, fmt.Sprintf("%#x", syncAggregate.SyncCommitteeBits.Bytes()))
	assert.Equal(t, expectedSignature, fmt.Sprintf("%#x", syncAggregate.SyncCommitteeSignature))

	// Test marshalling back to JSON
	marshaled, err := json.Marshal(syncAggregate)
	require.NoError(t, err)

	var unmarshaled lightclient.SyncAggregate
	err = json.Unmarshal(marshaled, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, expectedBits, fmt.Sprintf("%#x", unmarshaled.SyncCommitteeBits.Bytes()))
	assert.Equal(t, expectedSignature, fmt.Sprintf("%#x", unmarshaled.SyncCommitteeSignature))
}
