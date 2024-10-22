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

// BeaconBlockHeader represents a beacon block header.
type BeaconBlockHeader struct {
	Slot          phase0.Slot           `json:"slot"`
	ProposerIndex phase0.ValidatorIndex `json:"proposer_index"`
	ParentRoot    phase0.Root           `json:"parent_root"`
	StateRoot     phase0.Root           `json:"state_root"`
	BodyRoot      phase0.Root           `json:"body_root"`
}

type beaconBlockHeaderJSON struct {
	Slot          string `json:"slot"`
	ProposerIndex string `json:"proposer_index"`
	ParentRoot    string `json:"parent_root"`
	StateRoot     string `json:"state_root"`
	BodyRoot      string `json:"body_root"`
}

func (h *BeaconBlockHeader) ToJSON() beaconBlockHeaderJSON {
	return beaconBlockHeaderJSON{
		Slot:          fmt.Sprintf("%d", h.Slot),
		ProposerIndex: fmt.Sprintf("%d", h.ProposerIndex),
		ParentRoot:    h.ParentRoot.String(),
		StateRoot:     h.StateRoot.String(),
		BodyRoot:      h.BodyRoot.String(),
	}
}

func (h *BeaconBlockHeader) FromJSON(data beaconBlockHeaderJSON) error {
	slot, err := strconv.ParseUint(data.Slot, 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid slot")
	}
	h.Slot = phase0.Slot(slot)

	proposerIndex, err := strconv.ParseUint(data.ProposerIndex, 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid proposer index")
	}
	h.ProposerIndex = phase0.ValidatorIndex(proposerIndex)

	parentRoot, err := hex.DecodeString(strings.TrimPrefix(data.ParentRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid parent root")
	}
	h.ParentRoot = phase0.Root(parentRoot)

	stateRoot, err := hex.DecodeString(strings.TrimPrefix(data.StateRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid state root")
	}
	h.StateRoot = phase0.Root(stateRoot)

	bodyRoot, err := hex.DecodeString(strings.TrimPrefix(data.BodyRoot, "0x"))
	if err != nil {
		return errors.Wrap(err, "invalid body root")
	}
	h.BodyRoot = phase0.Root(bodyRoot)

	return nil
}

func (h BeaconBlockHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.ToJSON())
}

func (h *BeaconBlockHeader) UnmarshalJSON(data []byte) error {
	var jsonData beaconBlockHeaderJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	return h.FromJSON(jsonData)
}
