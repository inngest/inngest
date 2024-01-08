// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TimeFormat defines how time is encoded. Options are to use a time. layout
// string format such as time.RFC3339Nano, "second" for a decimal
// representation, "nano" for a an integer.
var TimeFormat = ""

// TimeWrap if not empty encoded time as an object with a single member. For
// example if set to "@" then and TimeFormat is RFC3339Nano then the encoded
// time will look like '{"@":"2020-04-12T16:34:04.123456789Z"}'
var TimeWrap = ""

// Time is a time.Time Node.
type Time time.Time

// String returns a string representation of the Node.
func (n Time) String() string {
	var b strings.Builder

	n.buildString(&b)

	return b.String()
}

// Alter returns the backing time.Time value of the Node.
func (n Time) Alter() any {
	return time.Time(n)
}

// Simplify returns the backing time.Time value of the Node.
func (n Time) Simplify() any {
	return time.Time(n)
}

// Dup returns the backing time.Time value of the Node.
func (n Time) Dup() Node {
	return n
}

// Empty returns false.
func (n Time) Empty() bool {
	return false
}

func (n Time) buildString(b *strings.Builder) {
	if 0 < len(TimeWrap) {
		b.WriteString(`{"`)
		b.WriteString(TimeWrap)
		b.WriteString(`":`)
	}
	switch TimeFormat {
	case "", "nano":
		b.WriteString(strconv.FormatInt(time.Time(n).UnixNano(), 10))
	case "second":
		// Decimal format but float is not accurate enough so build the output
		// in two parts.
		nano := time.Time(n).UnixNano()
		secs := nano / int64(time.Second)
		if 0 < nano {
			_, _ = fmt.Fprintf(b, "%d.%09d", secs, nano-(secs*int64(time.Second)))
		} else {
			_, _ = fmt.Fprintf(b, "%d.%09d", secs, -nano+(secs*int64(time.Second)))
		}
	default:
		b.WriteString(`"`)
		b.WriteString(time.Time(n).Format(TimeFormat))
		b.WriteString(`"`)
	}
	if 0 < len(TimeWrap) {
		b.WriteString("}")
	}
}
