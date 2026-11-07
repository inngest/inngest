// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iceberg

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

// NormalizeVariantPath renders member names as the spec's RFC-9535 normalized JSON path. Exported for table/internal; not part of the stable public API.
func NormalizeVariantPath(fields []string) string {
	if len(fields) == 0 {
		return "$"
	}

	var b strings.Builder
	b.WriteByte('$')
	for _, f := range fields {
		b.WriteString("['")
		b.WriteString(rfc9535Escape(f))
		b.WriteString("']")
	}

	return b.String()
}

func rfc9535Escape(name string) string {
	if strings.IndexFunc(name, func(r rune) bool {
		return r < 0x20 || r == '\'' || r == '\\'
	}) < 0 {
		return name
	}

	var b strings.Builder
	b.Grow(len(name) + 4)
	for _, r := range name {
		switch r {
		case '\b':
			b.WriteString(`\b`)
		case '\t':
			b.WriteString(`\t`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\'':
			b.WriteString(`\'`)
		case '\\':
			b.WriteString(`\\`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// parseVariantPath parses a variant path into member names, accepting dot ($.a.b) and bracket ($['a']) notation.
func parseVariantPath(path string) ([]string, error) {
	if !strings.HasPrefix(path, "$") {
		return nil, fmt.Errorf("%w: invalid variant path, does not start with $: %q", ErrInvalidArgument, path)
	}

	rest := path[len("$"):]
	names := []string{}
	for len(rest) > 0 {
		switch rest[0] {
		case '.':
			if strings.HasPrefix(rest, "..") {
				return nil, fmt.Errorf("%w: unsupported variant path, contains recursive descent: %q", ErrInvalidArgument, path)
			}
			rest = rest[1:]
			name := rest
			if end := strings.IndexAny(rest, ".["); end >= 0 {
				name, rest = rest[:end], rest[end:]
			} else {
				rest = ""
			}
			if !isRFC9535MemberName(name) {
				return nil, fmt.Errorf("%w: invalid variant path %q (%q has invalid characters)", ErrInvalidArgument, path, name)
			}
			names = append(names, name)
		case '[':
			name, consumed, err := parseBracketSelector(rest, path)
			if err != nil {
				return nil, err
			}
			names = append(names, name)
			rest = rest[consumed:]
		default:
			return nil, fmt.Errorf("%w: invalid variant path %q (expected '.' or '[' near %q)", ErrInvalidArgument, path, rest)
		}
	}

	return names, nil
}

// parseBracketSelector parses one `['name']` selector at the start of s (the RFC-9535
// inverse of rfc9535Escape), returning the unescaped member and bytes consumed.
func parseBracketSelector(s, path string) (string, int, error) {
	if len(s) < 2 || s[1] != '\'' {
		return "", 0, fmt.Errorf("%w: unsupported variant path %q (only quoted member selectors like ['name'] are supported)", ErrInvalidArgument, path)
	}

	var b strings.Builder
	for i := 2; i < len(s); {
		c := s[i]
		switch c {
		case '\'':
			if i+1 >= len(s) || s[i+1] != ']' {
				return "", 0, fmt.Errorf("%w: invalid variant path %q (expected ']' after quoted member)", ErrInvalidArgument, path)
			}

			return b.String(), i + 2, nil
		case '\\':
			if i+1 >= len(s) {
				return "", 0, fmt.Errorf("%w: invalid variant path %q (dangling escape)", ErrInvalidArgument, path)
			}
			switch e := s[i+1]; e {
			case 'b':
				b.WriteByte('\b')
			case 't':
				b.WriteByte('\t')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case '\'', '\\', '/':
				b.WriteByte(e)
			case 'u':
				r, consumed, uerr := decodeUnicodeEscape(s, i)
				if uerr != nil {
					return "", 0, fmt.Errorf("%w: invalid variant path %q (%s)", ErrInvalidArgument, path, uerr)
				}
				b.WriteRune(r)
				i += consumed

				continue
			default:
				return "", 0, fmt.Errorf("%w: invalid variant path %q (unknown escape \\%c)", ErrInvalidArgument, path, e)
			}
			i += 2
		default:
			b.WriteByte(c)
			i++
		}
	}

	return "", 0, fmt.Errorf("%w: invalid variant path %q (unterminated member selector)", ErrInvalidArgument, path)
}

// decodeUnicodeEscape decodes the \uXXXX escape at s[i:], pairing a UTF-16 surrogate pair, and returns the rune and bytes consumed.
func decodeUnicodeEscape(s string, i int) (rune, int, error) {
	if i+6 > len(s) {
		return 0, 0, errors.New("truncated \\u escape")
	}
	hi, err := strconv.ParseUint(s[i+2:i+6], 16, 32)
	if err != nil {
		return 0, 0, errors.New("bad \\u escape")
	}
	if !utf16.IsSurrogate(rune(hi)) {
		return rune(hi), 6, nil
	}

	// A surrogate must be a high surrogate followed by a low-surrogate \u escape.
	if i+12 > len(s) || s[i+6] != '\\' || s[i+7] != 'u' {
		return 0, 0, errors.New("unpaired surrogate \\u escape")
	}
	lo, err := strconv.ParseUint(s[i+8:i+12], 16, 32)
	if err != nil {
		return 0, 0, errors.New("bad \\u escape")
	}
	r := utf16.DecodeRune(rune(hi), rune(lo))
	if r == unicode.ReplacementChar {
		return 0, 0, errors.New("invalid surrogate pair")
	}

	return r, 12, nil
}

// isRFC9535MemberName reports whether name is a valid RFC-9535 member-name shorthand.
func isRFC9535MemberName(name string) bool {
	for i, r := range name {
		if i == 0 {
			if !isRFC9535NameFirst(r) {
				return false
			}

			continue
		}
		if isRFC9535NameFirst(r) || (r >= '0' && r <= '9') {
			continue
		}

		return false
	}

	return name != ""
}

func isRFC9535NameFirst(r rune) bool {
	return r == '_' ||
		(r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= 0x80 && r <= 0xD7FF) ||
		(r >= 0xE000 && r <= 0x10FFFF)
}
