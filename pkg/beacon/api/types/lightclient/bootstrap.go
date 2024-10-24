package lightclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

// Bootstrap is a light client bootstrap.
type Bootstrap struct {
	Header                     BootstrapHeader               `json:"header"`
	CurrentSyncCommittee       BootstrapCurrentSyncCommittee `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch []phase0.Root                 `json:"current_sync_committee_branch"`
}

// bootstrapJSON is the JSON representation of a bootstrap
type bootstrapJSON struct {
	Header                     bootstrapHeaderJSON                     `json:"header"`
	CurrentSyncCommittee       bootstrapCurrentSyncCommitteeJSON       `json:"current_sync_committee"`
	CurrentSyncCommitteeBranch bootstrapCurrentSyncCommitteeBranchJSON `json:"current_sync_committee_branch"`
}

// BootstrapCurrentSyncCommittee is the current sync committee of a light client bootstrap.
type BootstrapCurrentSyncCommittee struct {
	Pubkeys         []phase0.BLSPubKey
	AggregatePubkey phase0.BLSPubKey
}

// bootstrapCurrentSyncCommitteeJSON is the JSON representation of a bootstrap current sync committee.
type bootstrapCurrentSyncCommitteeJSON struct {
	Pubkeys         []string `json:"pubkeys"`
	AggregatePubkey string   `json:"aggregate_pubkey"`
}

// bootstrapCurrentSyncCommitteeBranchJSON is the JSON representation of a bootstrap current sync committee branch.
type bootstrapCurrentSyncCommitteeBranchJSON []string

func (b Bootstrap) MarshalJSON() ([]byte, error) {
	pubkeys := make([]string, len(b.CurrentSyncCommittee.Pubkeys))
	for i, pubkey := range b.CurrentSyncCommittee.Pubkeys {
		pubkeys[i] = pubkey.String()
	}

	branch := make([]string, len(b.CurrentSyncCommitteeBranch))
	for i, root := range b.CurrentSyncCommitteeBranch {
		branch[i] = root.String()
	}

	return json.Marshal(&bootstrapJSON{
		Header: b.Header.ToJSON(),
		CurrentSyncCommittee: bootstrapCurrentSyncCommitteeJSON{
			Pubkeys:         pubkeys,
			AggregatePubkey: b.CurrentSyncCommittee.AggregatePubkey.String(),
		},
		CurrentSyncCommitteeBranch: branch,
	})
}

func (b *Bootstrap) UnmarshalJSON(input []byte) error {
	var err error

	var jsonData bootstrapJSON
	if err = json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	if err = b.Header.Beacon.FromJSON(jsonData.Header.Beacon); err != nil {
		return errors.Wrap(err, "invalid header")
	}

	if len(jsonData.CurrentSyncCommitteeBranch) == 0 {
		return errors.New("current sync committee branch is required")
	}

	if len(jsonData.CurrentSyncCommittee.Pubkeys) == 0 {
		return errors.New("current sync committee pubkeys are required")
	}

	pubkeys := make([]phase0.BLSPubKey, len(jsonData.CurrentSyncCommittee.Pubkeys))
	for i, pubkey := range jsonData.CurrentSyncCommittee.Pubkeys {
		pubkeyBytes, err := hex.DecodeString(strings.TrimPrefix(pubkey, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid pubkey: %s", pubkey))
		}

		pubkeys[i] = phase0.BLSPubKey(pubkeyBytes)
	}
	b.CurrentSyncCommittee.Pubkeys = pubkeys

	if jsonData.CurrentSyncCommittee.AggregatePubkey == "" {
		return errors.New("current sync committee aggregate pubkey is required")
	}

	aggregatePubkeyBytes, err := hex.DecodeString(strings.TrimPrefix(jsonData.CurrentSyncCommittee.AggregatePubkey, "0x"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid aggregate pubkey: %s", jsonData.CurrentSyncCommittee.AggregatePubkey))
	}
	b.CurrentSyncCommittee.AggregatePubkey = phase0.BLSPubKey(aggregatePubkeyBytes)

	branch := make([]phase0.Root, len(jsonData.CurrentSyncCommitteeBranch))
	for i, root := range jsonData.CurrentSyncCommitteeBranch {
		r, err := hex.DecodeString(strings.TrimPrefix(root, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid root: %s", root))
		}

		branch[i] = phase0.Root(r)
	}

	b.CurrentSyncCommitteeBranch = branch

	return nil
}

func (b *BootstrapCurrentSyncCommittee) UnmarshalJSON(input []byte) error {
	var err error

	var jsonData bootstrapCurrentSyncCommitteeJSON
	if err = json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	b.Pubkeys = make([]phase0.BLSPubKey, len(jsonData.Pubkeys))
	for i, pubkey := range jsonData.Pubkeys {
		pubkeyBytes, err := hex.DecodeString(pubkey)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid pubkey: %s", pubkey))
		}

		b.Pubkeys[i] = phase0.BLSPubKey(pubkeyBytes)
	}

	aggregatePubkeyBytes, err := hex.DecodeString(jsonData.AggregatePubkey)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid aggregate pubkey: %s", jsonData.AggregatePubkey))
	}
	b.AggregatePubkey = phase0.BLSPubKey(aggregatePubkeyBytes)

	return nil
}
