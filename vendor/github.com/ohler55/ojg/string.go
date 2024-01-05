// Copyright (c) 2021, Peter Ohler, All rights reserved.

package ojg

import (
	"unicode/utf8"
)

const hex = "0123456789abcdef"

var (
	maxTokenLen = 64

	// Copied from sen/maps.go

	//   0123456789abcdef0123456789abcdef
	jMap = "" +
		`........btn.fr..................` + // 0x00
		`oo"ooohooooooooooooooooooooohoho` + // 0x20
		`oooooooooooooooooooooooooooo\ooo` + // 0x40
		"ooooooooooooooooooooooooooooooo." + // 0x60
		`88888888888888888888888888888888` + // 0x80
		`88888888888888888888888888888888` + // 0xa0
		`88888888888888888888888888888888` + // 0xc0
		`88888888888888888888888888888888` //  0xe0

	//   0123456789abcdef0123456789abcdef
	senMap = "" +
		`........bxx.fr..................` + // 0x00
		`xx"xoxhxxxooxoox0000000000xxhxho` + // 0x20
		`ooooooooooooooooooooooooooox\xoo` + // 0x40
		"oooooooooooooooooooooooooooxoxo." + // 0x60
		`88888888888888888888888888888888` + // 0x80
		`88888888888888888888888888888888` + // 0xa0
		`88888888888888888888888888888888` + // 0xc0
		`88888888888888888888888888888888` //  0xe0
)

// AppendJSONString appends a JSON encoding of a string to the provided byte
// slice.
func AppendJSONString(buf []byte, s string, htmlSafe bool) []byte {
	buf = append(buf, '"')
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
		case 'h':
			if htmlSafe {
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u00`...)
				buf = append(buf, hex[(b>>4)&0x0f])
				buf = append(buf, hex[b&0x0f])
				start = i + 1
			}
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
			buf = append(buf, '\\')
			buf = append(buf, c)
			start = i + 1
		}
	}
	if start < len(s) {
		buf = append(buf, s[start:]...)
	}
	return append(buf, '"')
}

// AppendSENString appends a SEN encoding of a string to the provided byte
// slice.
func AppendSENString(buf []byte, s string, htmlSafe bool) []byte {
	if len(s) == 0 {
		return append(buf, `""`...)
	}
	b0 := len(buf)
	m := senMap[s[0]]
	quote := maxTokenLen < len(s) || (m != 'o' && m != '8' && !(!htmlSafe && m == 'h'))
	buf = append(buf, '"')
	start := 0
	skip := 0
	for i, b := range []byte(s) {
		if i < skip {
			continue
		}
		c := senMap[b]
		switch c {
		case 'o', '0':
			continue
		case 'x':
			quote = true
		case '.':
			quote = true
			if start < i {
				buf = append(buf, s[start:i]...)
			}
			buf = append(buf, `\u00`...)
			buf = append(buf, hex[(b>>4)&0x0f])
			buf = append(buf, hex[b&0x0f])
			start = i + 1
		case 'h':
			if htmlSafe {
				quote = true
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u00`...)
				buf = append(buf, hex[(b>>4)&0x0f])
				buf = append(buf, hex[b&0x0f])
				start = i + 1
			}
		case '8':
			r, cnt := utf8.DecodeRuneInString(s[i:])
			switch r {
			case '\u2028':
				quote = true
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u2028`...)
				start = i + cnt
				skip = start
			case '\u2029':
				quote = true
				if start < i {
					buf = append(buf, s[start:i]...)
				}
				buf = append(buf, `\u2029`...)
				start = i + cnt
				skip = start
			case utf8.RuneError:
				quote = true
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
			buf = append(buf, '\\')
			buf = append(buf, c)
			start = i + 1
			quote = true
		}
	}
	if start < len(s) {
		buf = append(buf, s[start:]...)
	}
	if quote {
		return append(buf, '"')
	}
	copy(buf[b0:], buf[b0+1:])

	return buf[:len(buf)-1]
}
