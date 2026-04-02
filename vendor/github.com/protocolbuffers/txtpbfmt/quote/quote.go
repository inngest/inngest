// Package quote provides functions for fixing and smart quotes.
package quote

import "strings"

// Fix fixes quotes.
func Fix(s string) string {
	res := make([]byte, 0, len(s))
	res = append(res, '"')
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			res = append(res, '\\')
		} else if s[i] == '\\' {
			res = append(res, s[i])
			i++
		}
		res = append(res, s[i])
	}
	res = append(res, '"')
	return string(res)
}

func unescapeQuotes(s string) string {
	res := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		// If we hit an escape sequence...
		if s[i] == '\\' {
			// ... keep the backslash unless it's in front of a quote ...
			if i == len(s)-1 || (s[i+1] != '"' && s[i+1] != '\'') {
				res = append(res, '\\')
			}
			// ... then point at the escaped character so it is output verbatim below.
			// Doing this within the loop (without "continue") ensures correct handling
			// of escaped backslashes.
			i++
		}
		if i < len(s) {
			res = append(res, s[i])
		}
	}
	return string(res)
}

// Smart wraps the string in double quotes, but will escape any
// double quotes that appear within the string.
func Smart(s string) string {
	s = unescapeQuotes(s)
	if strings.Contains(s, "\"") && !strings.Contains(s, "'") {
		// If we hit this branch, the string doesn't contain any single quotes, and
		// is being wrapped in single quotes, so no escaping is needed.
		return "'" + s + "'"
	}
	// Fix will wrap the string in double quotes, but will escape any
	// double quotes that appear within the string.
	return Fix(s)
}
