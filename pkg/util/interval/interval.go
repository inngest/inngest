package interval

import (
	"fmt"
	"time"
)

func New(start, end time.Time) Interval {
	return Interval{
		A: start.UnixNano(),
		B: end.Sub(start).Nanoseconds(),
	}
}

// Interval represents an interval between a start and end time.
//
// In order to minimize space, the start time is represented as UnixNano(), and the duration
// is represented as the number of nanoseconds after the start.
// the nanosecond
type Interval struct {
	// A represents the start of the interval, taken as the nanoseconds after the unix epoch
	// (eg. via time.Now().UnixNano())
	A int64 `json:"a"`
	// B represents the duration, as nanoseconds.
	B int64 `json:"b"`
}

func (i Interval) String() string {
	return fmt.Sprintf(
		"%s-%s (%d)",
		i.Start().UTC().Format(time.RFC3339Nano),
		i.End().UTC().Format(time.RFC3339Nano),
		time.Duration(i.B).Microseconds(),
	)
}

func (i Interval) Start() time.Time {
	return time.Unix(0, i.A)
}

func (i Interval) End() time.Time {
	dur := time.Nanosecond * time.Duration(i.B)
	if dur < time.Millisecond {
		dur = time.Millisecond
	}
	return i.Start().Add(dur)
}
