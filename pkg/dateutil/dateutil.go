package dateutil

import (
	"errors"
	"time"
)

var (
	ErrUnknownFormat = errors.New("unknown date format")

	strFormats = []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		time.RFC1123,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RubyDate,
		time.UnixDate,
		time.ANSIC,
		time.Stamp,
		time.StampMilli,
		"2006-01-02",
	}
)

// Parse attempts to parse an incoming stringa s a date via multiple common formats.
func ParseString(input string) (time.Time, error) {
	for _, format := range strFormats {
		if t, err := time.Parse(format, input); err == nil {
			return t, nil
		}
	}
	return time.Time{}, ErrUnknownFormat
}

// ParseInt attempts to parse the given input int as a date, if it matches
// common int formats.  These are unix time, unix time in milliseconds, and
// unix time in nanoseconds.
func ParseInt(input int64) (time.Time, error) {
	if input < 946_684_800 {
		// We don't automatically parse times before 2000
		return time.Time{}, ErrUnknownFormat
	}
	// Is this unix time?  Unix time is typically
	if input < 9999999999 {
		return time.Unix(input, 0), nil
	}
	if input < 9999999999999 {
		// unix time, milliseconds.  a JS value.
		return time.Unix(0, input*1_000_000), nil
	}
	return time.Unix(0, input), nil
}

func Parse(input interface{}) (time.Time, error) {
	switch val := input.(type) {
	case string:
		return ParseString(val)
	case int64:
		return ParseInt(val)
	case uint64:
		return ParseInt(int64(val))
	case float64:
		return ParseInt(int64(val))
	}
	return time.Time{}, ErrUnknownFormat
}
