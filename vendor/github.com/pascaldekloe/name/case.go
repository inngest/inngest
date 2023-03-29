// Package name implements naming conventions.
// The functions offer flexible parsing and strict formatting for label
// techniques such as snake_case, Lisp-case, CamelCase and (Java) property keys.
package name

import (
	"strings"
	"unicode"
)

// CamelCase returns the medial capitals form of word sequence s.
// The input can be any case or even just a bunch of words.
// Upper case sequences (abbreviations) are preserved.
// Argument upper forces the letter case for the first rune. Use
// true for UpperCamelCase and false for lowerCamelCase.
func CamelCase(s string, upper bool) string {
	var b strings.Builder
	b.Grow(len(s))

	for i, r := range s {
		if i == 0 {
			if upper {
				b.WriteRune(unicode.ToUpper(r))
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			upper = false
			continue
		}

		switch {
		case unicode.IsLetter(r):
			if upper {
				r = unicode.ToUpper(r)
			}

			fallthrough
		case unicode.IsNumber(r):
			upper = false
			b.WriteRune(r)

		default:
			upper = true
		}
	}

	return b.String()
}

// SnakeCase is an alias for Delimit(s, '_').
func SnakeCase(s string) string {
	return Delimit(s, '_')
}

// DotSeparated is an alias for Delimit(s, '.').
func DotSeparated(s string) string {
	return Delimit(s, '.')
}

// Delimit returns word sequence s delimited with separator sep.
// The input can be any case or even just a bunch of words.
// Upper case sequences (abbreviations) are preserved. Use
// strings.ToLower and strings.ToUpper to enforce a letter case.
func Delimit(s string, sep rune) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)

	var last rune // previous rune; pending write
	sepDist := 1  // distance between a sep and the current rune r
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			if unicode.IsLower(last) {
				if b.Len() == 0 {
					last = unicode.ToUpper(last)
				} else {
					b.WriteRune(last)
					last = sep
					sepDist = 1
				}
			}

		case unicode.IsLetter(r): // lower-case
			if unicode.IsUpper(last) {
				if sepDist > 2 {
					b.WriteRune(sep)
				}
				last = unicode.ToLower(last)
			}

		case !unicode.IsNumber(r):
			if last == 0 || last == sep {
				continue
			}
			r = sep
			sepDist = 0
		}

		if last != 0 {
			b.WriteRune(last)
		}
		last = r
		sepDist++
	}

	if last != 0 && last != sep {
		if b.Len() == 0 {
			last = unicode.ToUpper(last)
		}
		b.WriteRune(last)
	}

	return b.String()
}
