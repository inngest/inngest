// Copyright (c) 2020, Peter Ohler, All rights reserved.

package alt

import (
	"strings"

	"github.com/ohler55/ojg/gen"
)

// Bool convert the value provided to a bool. If conversion is not possible
// such as if the provided value is an array then the first option default
// value is returned or if not provided false is returned. If the type is not
// a bool nor a gen.Bool and there is a second optional default then that
// second default value is returned. This approach keeps the return as a
// single value and gives the caller the choice of how to indicate a bad
// value.
func Bool(v any, defaults ...bool) (b bool) {
	switch tv := v.(type) {
	case nil:
		if 1 < len(defaults) {
			b = defaults[1]
		}
	case bool:
		b = tv
	case string:
		switch {
		case 1 < len(defaults):
			b = defaults[1]
		case strings.EqualFold(tv, "true"):
			b = true
		case strings.EqualFold(tv, "false"):
			b = false
		case 0 < len(defaults):
			b = defaults[0]
		}
	case gen.Bool:
		b = bool(tv)
	case gen.String:
		switch {
		case 1 < len(defaults):
			b = defaults[1]
		case strings.EqualFold(string(tv), "true"):
			b = true
		case strings.EqualFold(string(tv), "false"):
			b = false
		case 0 < len(defaults):
			b = defaults[0]
		}
	default:
		if 0 < len(defaults) {
			b = defaults[0]
		}
	}
	return
}
