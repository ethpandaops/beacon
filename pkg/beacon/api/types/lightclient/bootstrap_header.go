package lightclient

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// BootstrapHeader is the header of a light client bootstrap.
type BootstrapHeader struct {
	Beacon BeaconBlockHeader `json:"beacon"`
}

// bootstrapHeaderJSON is the JSON representation of a bootstrap header.
type bootstrapHeaderJSON struct {
	Beacon beaconBlockHeaderJSON `json:"beacon"`
}

func (h *BootstrapHeader) ToJSON() bootstrapHeaderJSON {
	return bootstrapHeaderJSON{
		Beacon: h.Beacon.ToJSON(),
	}
}

func (h *BootstrapHeader) FromJSON(input bootstrapHeaderJSON) error {
	return h.Beacon.FromJSON(input.Beacon)
}

func (h BootstrapHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.ToJSON())
}

func (b *BootstrapHeader) UnmarshalJSON(input []byte) error {
	var err error

	var jsonData bootstrapHeaderJSON
	if err = json.Unmarshal(input, &jsonData); err != nil {
		return errors.Wrap(err, "invalid JSON")
	}

	return b.FromJSON(jsonData)
}
