package lightclient_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLightClientHeaderMarshalUnmarshal(t *testing.T) {
	testCases := []struct {
		name   string
		header lightclient.LightClientHeader
	}{
		{
			name: "Basic LightClientHeader",
			header: lightclient.LightClientHeader{
				Beacon: lightclient.BeaconBlockHeader{
					Slot:          1234,
					ProposerIndex: 5678,
					ParentRoot:    phase0.Root{0x01, 0x02, 0x03},
					StateRoot:     phase0.Root{0x04, 0x05, 0x06},
					BodyRoot:      phase0.Root{0x07, 0x08, 0x09},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			marshaled, err := json.Marshal(tc.header)
			if err != nil {
				t.Fatalf("Failed to marshal LightClientHeader: %v", err)
			}

			var unmarshaled lightclient.LightClientHeader
			err = json.Unmarshal(marshaled, &unmarshaled)
			if err != nil {
				t.Fatalf("Failed to unmarshal LightClientHeader: %v", err)
			}

			if !reflect.DeepEqual(tc.header, unmarshaled) {
				t.Errorf("Unmarshaled LightClientHeader does not match original. Got %+v, want %+v", unmarshaled, tc.header)
			}
		})
	}
}

func TestLightClientHeaderUnmarshalJSON(t *testing.T) {
	expectedSlot := "1"
	expectedProposerIndex := "1"
	expectedParentRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	expectedStateRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"
	expectedBodyRoot := "0xcf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2"

	jsonStr := `{
		"beacon": {
          "slot": "` + expectedSlot + `",
          "proposer_index": "` + expectedProposerIndex + `",
          "parent_root": "` + expectedParentRoot + `",
          "state_root": "` + expectedStateRoot + `",
          "body_root": "` + expectedBodyRoot + `"
        }
      }`

	var header lightclient.LightClientHeader
	err := json.Unmarshal([]byte(jsonStr), &header)
	require.NoError(t, err)

	assert.Equal(t, expectedSlot, fmt.Sprintf("%d", header.Beacon.Slot))
	assert.Equal(t, expectedProposerIndex, fmt.Sprintf("%d", header.Beacon.ProposerIndex))
	assert.Equal(t, expectedParentRoot, header.Beacon.ParentRoot.String())
	assert.Equal(t, expectedStateRoot, header.Beacon.StateRoot.String())
	assert.Equal(t, expectedBodyRoot, header.Beacon.BodyRoot.String())
}
