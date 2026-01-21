package lightclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
)

// SyncCommittee represents a sync committee.
type SyncCommittee struct {
	Pubkeys         []phase0.BLSPubKey `json:"pubkeys"`
	AggregatePubkey phase0.BLSPubKey   `json:"aggregate_pubkey"`
}

// syncCommitteeJSON is the JSON representation of a sync committee.
type syncCommitteeJSON struct {
	Pubkeys         []string `json:"pubkeys"`
	AggregatePubkey string   `json:"aggregate_pubkey"`
}

// ToJSON converts a SyncCommittee to its JSON representation.
func (s *SyncCommittee) ToJSON() syncCommitteeJSON {
	pubkeys := make([]string, len(s.Pubkeys))
	for i, pubkey := range s.Pubkeys {
		pubkeys[i] = fmt.Sprintf("%#x", pubkey)
	}
	return syncCommitteeJSON{
		Pubkeys:         pubkeys,
		AggregatePubkey: fmt.Sprintf("%#x", s.AggregatePubkey),
	}
}

// FromJSON converts a JSON representation of a SyncCommittee to a SyncCommittee.
func (s *SyncCommittee) FromJSON(data syncCommitteeJSON) error {
	s.Pubkeys = make([]phase0.BLSPubKey, len(data.Pubkeys))
	for i, pubkey := range data.Pubkeys {
		pk, err := hex.DecodeString(strings.TrimPrefix(pubkey, "0x"))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("invalid pubkey: %s", pubkey))
		}
		copy(s.Pubkeys[i][:], pk)
	}

	aggregatePubkey, err := hex.DecodeString(strings.TrimPrefix(data.AggregatePubkey, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid aggregate pubkey")
	}
	copy(s.AggregatePubkey[:], aggregatePubkey)

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s SyncCommittee) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToJSON())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *SyncCommittee) UnmarshalJSON(data []byte) error {
	var jsonData syncCommitteeJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	return s.FromJSON(jsonData)
}
