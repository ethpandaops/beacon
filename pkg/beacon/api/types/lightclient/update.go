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

// Update represents a light client update.
type Update struct {
	AttestedHeader          LightClientHeader `json:"attested_header"`
	NextSyncCommittee       SyncCommittee     `json:"next_sync_committee"`
	NextSyncCommitteeBranch []phase0.Root     `json:"next_sync_committee_branch"`
	FinalizedHeader         LightClientHeader `json:"finalized_header"`
	FinalityBranch          []phase0.Root     `json:"finality_branch"`
	SyncAggregate           SyncAggregate     `json:"sync_aggregate"`
	SignatureSlot           phase0.Slot       `json:"signature_slot"`
}

// updateJSON is the JSON representation of an update
type updateJSON struct {
	AttestedHeader          lightClientHeaderJSON `json:"attested_header"`
	NextSyncCommittee       syncCommitteeJSON     `json:"next_sync_committee"`
	NextSyncCommitteeBranch []string              `json:"next_sync_committee_branch"`
	FinalizedHeader         lightClientHeaderJSON `json:"finalized_header"`
	FinalityBranch          []string              `json:"finality_branch"`
	SyncAggregate           syncAggregateJSON     `json:"sync_aggregate"`
	SignatureSlot           string                `json:"signature_slot"`
}

func (u Update) MarshalJSON() ([]byte, error) {
	nextSyncCommitteeBranch := make([]string, len(u.NextSyncCommitteeBranch))
	for i, root := range u.NextSyncCommitteeBranch {
		nextSyncCommitteeBranch[i] = root.String()
	}

	finalityBranch := make([]string, len(u.FinalityBranch))
	for i, root := range u.FinalityBranch {
		finalityBranch[i] = root.String()
	}

	return json.Marshal(&updateJSON{
		AttestedHeader:          u.AttestedHeader.ToJSON(),
		NextSyncCommittee:       u.NextSyncCommittee.ToJSON(),
		NextSyncCommitteeBranch: nextSyncCommitteeBranch,
		FinalizedHeader:         u.FinalizedHeader.ToJSON(),
		FinalityBranch:          finalityBranch,
		SyncAggregate:           u.SyncAggregate.ToJSON(),
		SignatureSlot:           fmt.Sprintf("%d", u.SignatureSlot),
	})
}

func (u *Update) UnmarshalJSON(input []byte) error {
	var jsonData updateJSON
	if err := json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	if err := u.AttestedHeader.FromJSON(jsonData.AttestedHeader); err != nil {
		return errors.Wrap(err, "invalid attested header")
	}

	if err := u.NextSyncCommittee.FromJSON(jsonData.NextSyncCommittee); err != nil {
		return errors.Wrap(err, "invalid next sync committee")
	}

	u.NextSyncCommitteeBranch = make([]phase0.Root, len(jsonData.NextSyncCommitteeBranch))
	for i, root := range jsonData.NextSyncCommitteeBranch {
		r, err := hex.DecodeString(strings.TrimPrefix(root, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid next sync committee branch root: %s", root))
		}
		u.NextSyncCommitteeBranch[i] = phase0.Root(r)
	}

	if err := u.FinalizedHeader.FromJSON(jsonData.FinalizedHeader); err != nil {
		return errors.Wrap(err, "invalid finalized header")
	}

	u.FinalityBranch = make([]phase0.Root, len(jsonData.FinalityBranch))
	for i, root := range jsonData.FinalityBranch {
		r, err := hex.DecodeString(strings.TrimPrefix(root, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid finality branch root: %s", root))
		}
		u.FinalityBranch[i] = phase0.Root(r)
	}

	if err := u.SyncAggregate.FromJSON(jsonData.SyncAggregate); err != nil {
		return errors.Wrap(err, "invalid sync aggregate")
	}

	slot, err := strconv.ParseUint(jsonData.SignatureSlot, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("invalid signature slot: %s", jsonData.SignatureSlot))
	}
	u.SignatureSlot = phase0.Slot(slot)

	return nil
}
