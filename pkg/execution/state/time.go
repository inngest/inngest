package state

import (
	"strconv"
	"time"
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
