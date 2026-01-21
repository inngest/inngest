package strduration

import (
	"encoding/json"
	"strings"
	"time"

	str2duration "github.com/xhit/go-str2duration/v2"
)

// Duration wraps time.Duration to marshal to and from JSON using
// string representations such as "1d30m".
type Duration time.Duration

// MarshalJSON implements json.Marshaler, emitting the string version of the duration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(str2duration.String(time.Duration(d)))
}

// UnmarshalJSON implements json.Unmarshaler, accepting string durations.
func (d *Duration) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*d = 0
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	if strings.TrimSpace(s) == "" {
		*d = 0
		return nil
	}

	dur, err := str2duration.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(dur)
	return nil
}
