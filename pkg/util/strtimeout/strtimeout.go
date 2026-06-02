package strtimeout

import (
	"fmt"
	"strings"
	"time"

	"github.com/xhit/go-str2duration/v2"
)

// ParseTimeout accepts either a string duration (3d) or a RFC 3339 time (2023-10-12T07:20:50.52Z).
func ParseTimeout(timeout string, now func() time.Time) (time.Time, error) {
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
