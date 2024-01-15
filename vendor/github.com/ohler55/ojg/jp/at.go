// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

// At is the @ in a JSON path representation.
type At byte

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f At) Append(buf []byte, bracket, first bool) []byte {
	buf = append(buf, '@')
	return buf
}

func (f At) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	if 0 < len(rest) {
		locs = rest[0].locate(append(pp, f), data, rest[1:], max)
	}
	return
}
