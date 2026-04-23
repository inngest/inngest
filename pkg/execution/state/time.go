package state

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xhit/go-str2duration/v2"
)

// Time defines a timestamp encoded the unix epoch in milliseconds.
type Time time.Time

// MarshalJSON is used to convert the timestamp to JSON
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(time.Time(t).UnixMilli(), 10)), nil
}

// UnmarshalJSON is used to convert the timestamp from JSON
func (t *Time) UnmarshalJSON(s []byte) (err error) {
	r := string(s)
	epoch, err := strconv.ParseInt(r, 10, 64)
	if err != nil {
		return err
	}
	*(*time.Time)(t) = time.UnixMilli(epoch).UTC()
	return nil
}

// Time returns the JSON time as a time.Time instance in UTC
func (t Time) Time() time.Time {
	return time.Time(t).UTC()
}

// String returns t as a formatted string
func (t Time) String() string {
	return t.Time().String()
}

// parseTimeout accepts either a string duration (3d) or a RFC 3339 time (2023-10-12T07:20:50.52Z).
func parseTimeout(timeout string, now func() time.Time) (time.Time, error) {
	dur, durErr := str2duration.ParseDuration(timeout)
	if durErr == nil {
		return now().Add(dur), nil
	}

	t, tErr := time.Parse(time.RFC3339, timeout)
	if tErr == nil {
		return t, nil
	}

	// When parsing as both a duration and a time fail, we try to distinguish
	// with the 'T' in RFC 3339 time (2023-10-12T07:20:50.52Z)
	if strings.ContainsRune(timeout, 'T') {
		return time.Time{}, fmt.Errorf("invalid RFC 3339 timestamp %q: %w", timeout, tErr)
	}
	return time.Time{}, fmt.Errorf("invalid duration %q: %w", timeout, durErr)
}
