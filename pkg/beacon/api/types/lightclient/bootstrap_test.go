package lightclient_test

import (
	"encoding/json"
	"testing"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/stretchr/testify/require"
)

func TestBootstrap_MarshalJSON(t *testing.T) {
	bootstrap := &lightclient.Bootstrap{
		Header: lightclient.BootstrapHeader{
			Beacon: lightclient.BeaconBlockHeader{
				Slot:          123,
				ProposerIndex: 456,
				ParentRoot:    phase0.Root{0x01},
				StateRoot:     phase0.Root{0x02},
				BodyRoot:      phase0.Root{0x03},
			},
		},
		CurrentSyncCommittee: lightclient.BootstrapCurrentSyncCommittee{
			Pubkeys:         []phase0.BLSPubKey{{0x04}, {0x05}},
			AggregatePubkey: phase0.BLSPubKey{0x06},
		},
		CurrentSyncCommitteeBranch: []phase0.Root{{0x07}, {0x08}},
	}

	jsonData, err := json.Marshal(bootstrap)
	require.NoError(t, err)

	expectedJSON := `{
		"header": {
			"beacon": {
				"slot": "123",
				"proposer_index": "456",
				"parent_root": "0x0100000000000000000000000000000000000000000000000000000000000000",
				"state_root": "0x0200000000000000000000000000000000000000000000000000000000000000",
				"body_root": "0x0300000000000000000000000000000000000000000000000000000000000000"
			}
		},
		"current_sync_committee": {
			"pubkeys": [
				"0x040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"0x050000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
			],
			"aggregate_pubkey": "0x060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		},
		"current_sync_committee_branch": [
			"0x0700000000000000000000000000000000000000000000000000000000000000",
			"0x0800000000000000000000000000000000000000000000000000000000000000"
		]
	}`
	require.JSONEq(t, expectedJSON, string(jsonData))
}

func TestBootstrap_UnmarshalJSON(t *testing.T) {
	jsonData := []byte(`{
		"header": {
			"beacon": {
				"slot": "123",
				"proposer_index": "456",
				"parent_root": "0x0100000000000000000000000000000000000000000000000000000000000000",
				"state_root": "0x0200000000000000000000000000000000000000000000000000000000000000",
				"body_root": "0x0300000000000000000000000000000000000000000000000000000000000000"
			}
		},
		"current_sync_committee": {
			"pubkeys": [
				"0x040000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				"0x050000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
			],
			"aggregate_pubkey": "0x060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		},
		"current_sync_committee_branch": [
			"0x0700000000000000000000000000000000000000000000000000000000000000",
			"0x0800000000000000000000000000000000000000000000000000000000000000"
		]
	}`)

	var bootstrap lightclient.Bootstrap
	err := json.Unmarshal(jsonData, &bootstrap)
	require.NoError(t, err)

	require.Equal(t, phase0.Slot(123), bootstrap.Header.Beacon.Slot)
	require.Equal(t, phase0.ValidatorIndex(456), bootstrap.Header.Beacon.ProposerIndex)
	require.Equal(t, phase0.Root{0x01}, bootstrap.Header.Beacon.ParentRoot)
	require.Equal(t, phase0.Root{0x02}, bootstrap.Header.Beacon.StateRoot)
	require.Equal(t, phase0.Root{0x03}, bootstrap.Header.Beacon.BodyRoot)
	require.Equal(t, []phase0.BLSPubKey{{0x04}, {0x05}}, bootstrap.CurrentSyncCommittee.Pubkeys)
	require.Equal(t, phase0.BLSPubKey{0x06}, bootstrap.CurrentSyncCommittee.AggregatePubkey)
	require.Equal(t, []phase0.Root{{0x07}, {0x08}}, bootstrap.CurrentSyncCommitteeBranch)
}
