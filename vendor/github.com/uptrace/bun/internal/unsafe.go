//go:build !appengine
// +build !appengine

package internal

import "unsafe"

// String converts byte slice to string.
func String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}

// Bytes converts string to byte slice.
func Bytes(s string) []byte {
	if s == "" {
		return []byte{}
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
