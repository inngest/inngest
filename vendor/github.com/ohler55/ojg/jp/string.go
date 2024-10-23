// Copyright (c) 2023, Peter Ohler, All rights reserved.

package jp

import (
	"unicode/utf8"
)

const hex = "0123456789abcdef"

var (
	//   0123456789abcdef0123456789abcdef
	jMap = "" +
		`........btn.fr..................` + // 0x00
		`oo"oooo'oooooooooooooooooooooooo` + // 0x20
		`oooooooooooooooooooooooooooo\ooo` + // 0x40
		"ooooooooooooooooooooooooooooooo." + // 0x60
		`88888888888888888888888888888888` + // 0x80
		`88888888888888888888888888888888` + // 0xa0
		`88888888888888888888888888888888` + // 0xc0
		`88888888888888888888888888888888` //  0xe0
)

// AppendString to a buffer while escaping characters as necessary.
func AppendString(buf []byte, s string, delim byte) []byte {
	buf = append(buf, delim)
	start := 0
	skip := 0
	for i, b := range []byte(s) {
		if i < skip {
			continue
		}
		c := jMap[b]
		switch c {
		case 'o':
			continue
		case '.':
			if start < i {
				buf = append(buf, s[start:i]...)
			}
			buf = append(buf, `\u00`...)
			buf = append(buf, hex[(b>>4)&0x0f])
			buf = append(buf, hex[b&0x0f])
			start = i + 1
		case '8':
			r, cnt := utf8.DecodeRuneInString(s[i:])
			switch r {
			case '\u2028':
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u2028`...)
				start = i + cnt
				skip = start
			case '\u2029':
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u2029`...)
				start = i + cnt
				skip = start
			case utf8.RuneError:
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\ufffd`...)
				start = i + cnt
				skip = start
			default:
				skip = i + cnt
			}
		default:
			if start < i {
				buf = append(buf, s[start:i]...)
			}
			if delim != '/' { // don't escape regexp as they are already escaped
				buf = append(buf, '\\')
			}
			buf = append(buf, c)
			start = i + 1
		}
	}
	if start < len(s) {
		buf = append(buf, s[start:]...)
	}
	return append(buf, delim)
}
