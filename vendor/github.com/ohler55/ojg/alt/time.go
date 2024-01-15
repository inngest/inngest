// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"time"

	"github.com/ohler55/ojg/gen"
)

// Time convert the value provided to a time.Time. If conversion is not
// possible such as if the provided value is an array then the first option
// default value is returned or if not provided zero time is returned. If the
// type is not one of the int or uint types and there is a second optional
// default then that second default value is returned. This approach keeps the
// return as a single value and gives the caller the choice of how to indicate
// a bad value.
func Time(v any, defaults ...time.Time) (t time.Time) {
	switch tt := v.(type) {
	case time.Time:
		t = tt
	case gen.Time:
		t = time.Time(tt)
	default:
		if 1 < len(defaults) {
			t = defaults[1]
		} else {
			switch tv := v.(type) {
			case int64:
				t = time.Unix(0, tv).UTC()
			case int:
				t = time.Unix(0, int64(tv)).UTC()
			case uint:
				t = time.Unix(0, int64(tv)).UTC()
			case uint64:
				t = time.Unix(0, int64(tv)).UTC()
			case float32:
				// Only good to minutes.
				secs := int64(tv) / 60 * 60
				t = time.Unix(secs, 0).UTC()
			case float64:
				secs := int64(tv)
				// Only good to microseconds, not nanoseconds.
				nano := int64((tv-float64(secs))*float64(time.Second)) / 1000 * 1000
				t = time.Unix(secs, nano).UTC()
			case string:
				var err error
				if t, err = time.Parse(time.RFC3339Nano, tv); err != nil {
					if 0 < len(defaults) {
						t = defaults[0]
					}
				}

			case gen.Int:
				t = time.Unix(0, int64(tv)).UTC()
			case gen.Float:
				secs := int64(tv)
				// Only good to useconds, not nanoseconds.
				nano := int64((float64(tv)-float64(secs))*float64(time.Second)) / 1000 * 1000
				t = time.Unix(secs, nano).UTC()
			case gen.String:
				var err error
				if t, err = time.Parse(time.RFC3339Nano, string(tv)); err != nil {
					if 0 < len(defaults) {
						t = defaults[0]
					}
				}
			default:
				if 0 < len(defaults) {
					t = defaults[0]
				}
			}
		}
	}
	return
}
