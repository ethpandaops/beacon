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

// FinalityUpdate represents a finality update for light clients.
type FinalityUpdate struct {
	AttestedHeader  LightClientHeader
	FinalizedHeader LightClientHeader
	FinalityBranch  []phase0.Root
	SyncAggregate   SyncAggregate
	SignatureSlot   phase0.Slot
}

type finalityUpdateJSON struct {
	AttestedHeader  lightClientHeaderJSON `json:"attested_header"`
	FinalizedHeader lightClientHeaderJSON `json:"finalized_header"`
	FinalityBranch  []string              `json:"finality_branch"`
	SyncAggregate   syncAggregateJSON     `json:"sync_aggregate"`
	SignatureSlot   string                `json:"signature_slot"`
}

func (f *FinalityUpdate) UnmarshalJSON(data []byte) error {
	var jsonData finalityUpdateJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	return f.FromJSON(jsonData)
}

func (f *FinalityUpdate) FromJSON(data finalityUpdateJSON) error {
	attestedHeader := LightClientHeader{}
	if err := attestedHeader.FromJSON(data.AttestedHeader); err != nil {
		return errors.Wrap(err, "failed to unmarshal attested header")
	}
	f.AttestedHeader = attestedHeader

	finalizedHeader := LightClientHeader{}
	if err := finalizedHeader.FromJSON(data.FinalizedHeader); err != nil {
		return errors.Wrap(err, "failed to unmarshal finalized header")
	}
	f.FinalizedHeader = finalizedHeader

	finalityBranch := make([]phase0.Root, len(data.FinalityBranch))
	for i, root := range data.FinalityBranch {
		decoded, err := hex.DecodeString(strings.TrimPrefix(root, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to unmarshal finality branch at index %d: %s", i, root))
		}
		finalityBranch[i] = phase0.Root(decoded)
	}
	f.FinalityBranch = finalityBranch

	syncAggregate := SyncAggregate{}
	if err := syncAggregate.FromJSON(data.SyncAggregate); err != nil {
		return errors.Wrap(err, "failed to unmarshal sync aggregate")
	}
	f.SyncAggregate = syncAggregate

	signatureSlot, err := strconv.ParseUint(data.SignatureSlot, 10, 64)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to unmarshal signature slot: %s", data.SignatureSlot))
	}
	f.SignatureSlot = phase0.Slot(signatureSlot)

	return nil
}

func (f FinalityUpdate) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.ToJSON())
}

func (f *FinalityUpdate) ToJSON() finalityUpdateJSON {
	finalityBranch := make([]string, len(f.FinalityBranch))
	for i, root := range f.FinalityBranch {
		finalityBranch[i] = fmt.Sprintf("%x", root)
	}

	return finalityUpdateJSON{
		AttestedHeader:  f.AttestedHeader.ToJSON(),
		FinalizedHeader: f.FinalizedHeader.ToJSON(),
		FinalityBranch:  finalityBranch,
		SyncAggregate:   f.SyncAggregate.ToJSON(),
		SignatureSlot:   fmt.Sprintf("%d", f.SignatureSlot),
	}
}
