package lightclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-bitfield"
)

// SyncAggregate represents a sync aggregate.
type SyncAggregate struct {
	SyncCommitteeBits      bitfield.Bitvector512 `json:"sync_committee_bits"`
	SyncCommitteeSignature phase0.BLSSignature   `json:"sync_committee_signature"`
}

type syncAggregateJSON struct {
	SyncCommitteeBits      string `json:"sync_committee_bits"`
	SyncCommitteeSignature string `json:"sync_committee_signature"`
}

func (s *SyncAggregate) ToJSON() syncAggregateJSON {
	return syncAggregateJSON{
		SyncCommitteeBits:      fmt.Sprintf("%#x", s.SyncCommitteeBits.Bytes()),
		SyncCommitteeSignature: fmt.Sprintf("%#x", s.SyncCommitteeSignature),
	}
}

func (s *SyncAggregate) FromJSON(data syncAggregateJSON) error {
	if data.SyncCommitteeBits == "" {
		return errors.New("sync committee bits are required")
	}

	if data.SyncCommitteeSignature == "" {
		return errors.New("sync committee signature is required")
	}

	bits, err := hex.DecodeString(strings.TrimPrefix(data.SyncCommitteeBits, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid sync committee bits")
	}

	s.SyncCommitteeBits = bitfield.Bitvector512(bits)

	signature, err := hex.DecodeString(strings.TrimPrefix(data.SyncCommitteeSignature, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid sync committee signature")
	}
	s.SyncCommitteeSignature = phase0.BLSSignature(signature)

	return nil
}

func (s SyncAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToJSON())
}

func (s *SyncAggregate) UnmarshalJSON(input []byte) error {
	var data syncAggregateJSON
	if err := json.Unmarshal(input, &data); err != nil {
		return errors.Wrap(err, "failed to unmarshal sync aggregate")
	}

	return s.FromJSON(data)
}
