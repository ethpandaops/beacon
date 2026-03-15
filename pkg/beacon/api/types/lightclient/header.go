package lightclient

import (
	"encoding/json"
)

// LightClientHeader represents a light client header.
type LightClientHeader struct {
	Beacon BeaconBlockHeader `json:"beacon"`
}

type lightClientHeaderJSON struct {
	Beacon beaconBlockHeaderJSON `json:"beacon"`
}

func (h *LightClientHeader) ToJSON() lightClientHeaderJSON {
	return lightClientHeaderJSON{
		Beacon: h.Beacon.ToJSON(),
	}
}

func (h *LightClientHeader) FromJSON(data lightClientHeaderJSON) error {
	return h.Beacon.FromJSON(data.Beacon)
}

func (h LightClientHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.ToJSON())
}

func (h *LightClientHeader) UnmarshalJSON(data []byte) error {
	var jsonData lightClientHeaderJSON
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return err
	}
	return h.FromJSON(jsonData)
}
