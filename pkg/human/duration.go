package human

import (
	"encoding/json"
	"time"
)

// Duration is a time.Duration that can be marshalled to JSON as a string.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	return d.Unmarshal(string(text))
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	return d.Unmarshal(s)
}

func (d *Duration) Unmarshal(s string) (err error) {
	d.Duration, err = time.ParseDuration(s)

	return
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.Duration.String()), nil //nolint:staticcheck // existing.
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String()) //nolint:staticcheck // existing.
}
