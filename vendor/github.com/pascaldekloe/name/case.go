// Package name implements naming conventions.
// The functions offer flexible parsing and strict formatting for label
// techniques such as snake_case, Lisp-case, CamelCase and (Java) property keys.
package name

import "unicode"

// CamelCase returns the medial capitals form of word sequence s.
// The input can be any case or even just a bunch of words.
// Upper case sequences (abbreviations) are preserved.
// Argument upper sets the letter case for the first rune. Use true for
// UpperCamelCase and false for lowerCamelCase.
func CamelCase(s string, upper bool) string {
	if s == "" {
		return ""
	}

	out := make([]rune, 1, len(s)+5)
	for i, r := range s {
		if i == 0 {
			if upper {
				r = unicode.ToUpper(r)
			}
			out[0] = r
			continue
		}

		if i == 1 {
			if !upper && unicode.Is(unicode.Lower, r) {
				out[0] = unicode.ToLower(out[0])
			}

			upper = false
		}

		switch {
		case unicode.IsLetter(r):
			if upper {
				r = unicode.ToUpper(r)
			}

			fallthrough
		case unicode.IsNumber(r):
			upper = false
			out = append(out, r)

		default:
			upper = true

		}
	}

	return string(out)
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
	out := make([]rune, 0, len(s)+5)

	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			if last := len(out) - 1; last >= 0 && unicode.IsLower(out[last]) {
				out = append(out, sep)
			}

		case unicode.IsLetter(r):
			if i := len(out) - 1; i >= 0 {
				if last := out[i]; unicode.IsUpper(last) {
					out = out[:i]
					if i > 0 && out[i-1] != sep {
						out = append(out, sep)
					}
					out = append(out, unicode.ToLower(last))
				}
			}

		case !unicode.IsNumber(r):
			if i := len(out); i != 0 && out[i-1] != sep {
				out = append(out, sep)
			}
			continue

		}
		out = append(out, r)
	}

	if len(out) == 0 {
		return ""
	}

	// trim tailing separator
	if i := len(out) - 1; out[i] == sep {
		out = out[:i]
	}

	if len(out) == 1 {
		out[0] = unicode.ToLower(out[0])
	}

	return string(out)
}
