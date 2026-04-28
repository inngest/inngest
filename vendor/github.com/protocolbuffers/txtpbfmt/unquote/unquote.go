// Package unquote provides a function to unquote txtpb-formatted quoted string literals.
package unquote

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/protocolbuffers/txtpbfmt/ast"
)

// Unquote returns the value of the string node and the rune used to quote it.
// Calling Unquote on non-string node doesn't panic, but is otherwise undefined.
func Unquote(n *ast.Node) (string, rune, error) {
	return unquoteValues(n.Values, unquote)
}

// Raw returns the raw value of the string node and the rune used to quote it, with string escapes
// left in place.
// Calling UnquoteRaw on non-string node doesn't panic, but is otherwise undefined.
func Raw(n *ast.Node) (string, rune, error) {
	return unquoteValues(n.Values, unquoteRaw)
}

func unquoteValues(values []*ast.Value, unquoter func(string) (string, rune, error)) (string, rune, error) {
	var ret strings.Builder
	firstQuote := rune(0)
	for _, v := range values {
		uq, quote, err := unquoter(v.Value)
		if firstQuote == rune(0) {
			firstQuote = quote
		}
		if err != nil {
			return "", rune(0), err
		}
		ret.WriteString(uq)
	}
	return ret.String(), firstQuote, nil
}

// Returns the quote rune used in the given string (' or "). Returns an error if the string doesn't
// start and end with a matching pair of valid quotes.
func quoteRune(s string) (rune, error) {
	if len(s) < 2 {
		return 0, errors.New("not a quoted string")
	}
	quote := s[0]
	if quote != '"' && quote != '\'' {
		return 0, fmt.Errorf("invalid quote character %s", string(quote))
	}
	if s[len(s)-1] != quote {
		return 0, errors.New("unmatched quote")
	}
	return rune(quote), nil
}

func unquote(s string) (string, rune, error) {
	quote, err := quoteRune(s)
	if err != nil {
		return "", rune(0), err
	}
	unquoted, err := unquoteC(s[1:len(s)-1], quote)
	return unquoted, quote, err
}

func unquoteRaw(s string) (string, rune, error) {
	quote, err := quoteRune(s) // Trigger validation, which guarantees this is a quote-wrapped string.
	if err != nil {
		return "", rune(0), err
	}
	return s[1 : len(s)-1], quote, nil
}

var (
	errBadUTF8 = errors.New("bad UTF-8")
)

func unquoteC(s string, quote rune) (string, error) {
	// Copied from third_party/golang/protobuf/proto/text_parser.go

	// This is based on C++'s tokenizer.cc.
	// Despite its name, this is *not* parsing C syntax.
	// For instance, "\0" is an invalid quoted string.

	// Avoid allocation in trivial cases.
	simple := true
	for _, r := range s {
		if r == '\\' || r == quote {
			simple = false
			break
		}
	}
	if simple {
		return s, nil
	}

	buf := make([]byte, 0, 3*len(s)/2)
	for len(s) > 0 {
		r, n := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && n == 1 {
			return "", errBadUTF8
		}
		s = s[n:]
		if r != '\\' {
			buf = appendRune(buf, r)
			continue
		}

		ch, tail, err := unescape(s)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch...)
		s = tail
	}
	return string(buf), nil
}

func appendRune(buf []byte, r rune) []byte {
	if r < utf8.RuneSelf {
		return append(buf, byte(r))
	}
	return append(buf, string(r)...)
}

func unescape(s string) (ch string, tail string, err error) {
	// Copied from third_party/golang/protobuf/proto/text_parser.go

	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && n == 1 {
		return "", "", errBadUTF8
	}
	s = s[n:]
	switch r {
	case 'a':
		return "\a", s, nil
	case 'b':
		return "\b", s, nil
	case 'f':
		return "\f", s, nil
	case 'n':
		return "\n", s, nil
	case 'r':
		return "\r", s, nil
	case 't':
		return "\t", s, nil
	case 'v':
		return "\v", s, nil
	case '?':
		return "?", s, nil // trigraph workaround
	case '\'', '"', '\\':
		return string(r), s, nil
	case '0', '1', '2', '3', '4', '5', '6', '7':
		return unescapeOctal(r, s)
	case 'x', 'X':
		return unescapeHex(r, s, 2)
	case 'u':
		return unescapeHex(r, s, 4)
	case 'U':
		return unescapeHex(r, s, 8)
	}
	return "", "", fmt.Errorf(`unknown escape \%c`, r)
}

func unescapeOctal(r rune, s string) (string, string, error) {
	if len(s) < 2 {
		return "", "", fmt.Errorf(`\%c requires 2 following digits`, r)
	}
	ss := string(r) + s[:2]
	s = s[2:]
	i, err := strconv.ParseUint(ss, 8, 8)
	if err != nil {
		return "", "", fmt.Errorf(`\%s contains non-octal digits`, ss)
	}
	return string([]byte{byte(i)}), s, nil
}

func unescapeHex(r rune, s string, n int) (string, string, error) {
	if len(s) < n {
		return "", "", fmt.Errorf(`\%c requires %d following digits`, r, n)
	}
	ss := s[:n]
	s = s[n:]
	i, err := strconv.ParseUint(ss, 16, 64)
	if err != nil {
		return "", "", fmt.Errorf(`\%c%s contains non-hexadecimal digits`, r, ss)
	}
	if r == 'x' || r == 'X' {
		return string([]byte{byte(i)}), s, nil
	}
	if i > utf8.MaxRune {
		return "", "", fmt.Errorf(`\%c%s is not a valid Unicode code point`, r, ss)
	}
	return strconv.FormatUint(i, 10), s, nil
}
