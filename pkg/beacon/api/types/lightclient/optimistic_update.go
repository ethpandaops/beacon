package lightclient

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// OptimisticUpdate represents a light client optimistic update.
type OptimisticUpdate struct {
	AttestedHeader LightClientHeader `json:"attested_header"`
	SyncAggregate  SyncAggregate     `json:"sync_aggregate"`
}

// optimisticUpdateJSON is the JSON representation of an optimistic update
type optimisticUpdateJSON struct {
	AttestedHeader lightClientHeaderJSON `json:"attested_header"`
	SyncAggregate  syncAggregateJSON     `json:"sync_aggregate"`
}

func (u OptimisticUpdate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&optimisticUpdateJSON{
		AttestedHeader: u.AttestedHeader.ToJSON(),
		SyncAggregate:  u.SyncAggregate.ToJSON(),
	})
}

func (u *OptimisticUpdate) UnmarshalJSON(input []byte) error {
	var jsonData optimisticUpdateJSON
	if err := json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	if err := u.AttestedHeader.FromJSON(jsonData.AttestedHeader); err != nil {
		return errors.Wrap(err, "invalid attested header")
	}

	if err := u.SyncAggregate.FromJSON(jsonData.SyncAggregate); err != nil {
		return errors.Wrap(err, "invalid sync aggregate")
	}

	return nil
}
