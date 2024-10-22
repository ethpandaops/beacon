package lightclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
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

// BootstrapHeader is the header of a light client bootstrap.
type BootstrapHeader struct {
	Slot          phase0.Slot
	ProposerIndex phase0.ValidatorIndex
	ParentRoot    phase0.Root
	StateRoot     phase0.Root
	BodyRoot      phase0.Root
}

// bootstrapHeaderJSON is the JSON representation of a bootstrap header.
type bootstrapHeaderJSON struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
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
		Header: bootstrapHeaderJSON{
			Slot:          fmt.Sprintf("%d", b.Header.Slot),
			ProposerIndex: fmt.Sprintf("%d", b.Header.ProposerIndex),
			ParentRoot:    b.Header.ParentRoot.String(),
			StateRoot:     b.Header.StateRoot.String(),
			BodyRoot:      b.Header.BodyRoot.String(),
		},
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

	if jsonData.Header.Slot == "" {
		return errors.New("slot is required")
	}

	slot, err := strconv.ParseUint(jsonData.Header.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid slot: %s", jsonData.Header.Slot))
	}
	b.Header.Slot = phase0.Slot(slot)

	if jsonData.Header.ProposerIndex == "" {
		return errors.New("proposer index is required")
	}

	proposerIndex, err := strconv.ParseUint(jsonData.Header.ProposerIndex, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid proposer index: %s", jsonData.Header.ProposerIndex))
	}
	b.Header.ProposerIndex = phase0.ValidatorIndex(proposerIndex)

	if jsonData.Header.ParentRoot == "" {
		return errors.New("parent root is required")
	}

	parentRoot, err := hex.DecodeString(strings.TrimPrefix(jsonData.Header.ParentRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid parent root: %s", jsonData.Header.ParentRoot))
	}
	b.Header.ParentRoot = phase0.Root(parentRoot)

	if jsonData.Header.StateRoot == "" {
		return errors.New("state root is required")
	}

	stateRoot, err := hex.DecodeString(strings.TrimPrefix(jsonData.Header.StateRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid state root: %s", jsonData.Header.StateRoot))
	}
	b.Header.StateRoot = phase0.Root(stateRoot)

	if jsonData.Header.BodyRoot == "" {
		return errors.New("body root is required")
	}

	bodyRoot, err := hex.DecodeString(strings.TrimPrefix(jsonData.Header.BodyRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid body root: %s", jsonData.Header.BodyRoot))
	}
	b.Header.BodyRoot = phase0.Root(bodyRoot)

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

func (b *BootstrapHeader) UnmarshalJSON(input []byte) error {
	var err error

	var jsonData bootstrapHeaderJSON
	if err = json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	slot, err := strconv.ParseUint(jsonData.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid slot: %s", jsonData.Slot))
	}
	b.Slot = phase0.Slot(slot)

	proposerIndex, err := strconv.ParseUint(jsonData.ProposerIndex, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid proposer index: %s", jsonData.ProposerIndex))
	}
	b.ProposerIndex = phase0.ValidatorIndex(proposerIndex)

	parentRoot, err := hex.DecodeString(jsonData.ParentRoot)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid parent root: %s", jsonData.ParentRoot))
	}
	b.ParentRoot = phase0.Root(parentRoot)

	stateRoot, err := hex.DecodeString(jsonData.StateRoot)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid state root: %s", jsonData.StateRoot))
	}
	b.StateRoot = phase0.Root(stateRoot)

	bodyRoot, err := hex.DecodeString(jsonData.BodyRoot)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid body root: %s", jsonData.BodyRoot))
	}
	b.BodyRoot = phase0.Root(bodyRoot)

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
