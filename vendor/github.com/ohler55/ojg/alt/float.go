// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"strconv"
	"time"

	"github.com/ohler55/ojg/gen"
)

// Float convert the value provided to a float64. If conversion is not
// possible such as if the provided value is an array then the first option
// default value is returned or if not provided 0.0 is returned. If the type
// is not one of the float types and there is a second optional default then
// that second default value is returned. This approach keeps the return as a
// single value and gives the caller the choice of how to indicate a bad
// value.
func Float(v any, defaults ...float64) (f float64) {
	switch tf := v.(type) {
	case float64:
		f = tf
	case float32:
		f = float64(tf)
	case gen.Float:
		f = float64(tf)
	default:
		if 1 < len(defaults) {
			f = defaults[1]
		} else {
			switch tv := v.(type) {
			case int64:
				f = float64(tv)
			case int:
				f = float64(tv)
			case int8:
				f = float64(tv)
			case int16:
				f = float64(tv)
			case int32:
				f = float64(tv)
			case uint:
				f = float64(tv)
			case uint8:
				f = float64(tv)
			case uint16:
				f = float64(tv)
			case uint32:
				f = float64(tv)
			case uint64:
				f = float64(tv)
			case string:
				var err error
				if f, err = strconv.ParseFloat(tv, 64); err != nil {
					if 0 < len(defaults) {
						f = defaults[0]
					}
				}

			case time.Time:
				nano := tv.UnixNano()
				sec := nano / int64(time.Second)
				f = float64(sec) + float64(nano-sec*int64(time.Second))/float64(time.Second)

			case gen.Int:
				f = float64(tv)
			case gen.String:
				f = Float(string(tv), defaults...)
			case gen.Time:
				nano := time.Time(tv).UnixNano()
				sec := nano / int64(time.Second)
				f = float64(sec) + float64(nano-sec*int64(time.Second))/float64(time.Second)

			case gen.Big:
				return Float(string(tv), defaults...)

			default:
				if 0 < len(defaults) {
					f = defaults[0]
				}
			}
		}
	}
	return
}
