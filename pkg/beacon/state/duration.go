package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// StringerDuration is a time.Duration that can be marshalled to JSON as a string.
type StringerDuration time.Duration

func (s StringerDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%v", time.Duration(s).Seconds()))
}

func (s *StringerDuration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*s = StringerDuration(time.Duration(value))

		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*s = StringerDuration(tmp)

		return nil
	default:
		return errors.New("invalid duration")
	}
}

// AsDuration returns the duration as a time.Duration.
func (s StringerDuration) AsDuration() time.Duration {
	return time.Duration(s)
}
