package ociauth

import (
	"net/http"
	"strings"
)

// Octet types from RFC 2616.
type octetType byte

var octetTypes [256]octetType

const (
	isToken octetType = 1 << iota
	isSpace
)

func init() {
	// OCTET      = <any 8-bit sequence of data>
	// CHAR       = <any US-ASCII character (octets 0 - 127)>
	// CTL        = <any US-ASCII control character (octets 0 - 31) and DEL (127)>
	// CR         = <US-ASCII CR, carriage return (13)>
	// LF         = <US-ASCII LF, linefeed (10)>
	// SP         = <US-ASCII SP, space (32)>
	// HT         = <US-ASCII HT, horizontal-tab (9)>
	// <">        = <US-ASCII double-quote mark (34)>
	// CRLF       = CR LF
	// LWS        = [CRLF] 1*( SP | HT )
	// TEXT       = <any OCTET except CTLs, but including LWS>
	// separators = "(" | ")" | "<" | ">" | "@" | "," | ";" | ":" | "\" | <">
	//              | "/" | "[" | "]" | "?" | "=" | "{" | "}" | SP | HT
	// token      = 1*<any CHAR except CTLs or separators>
	// qdtext     = <any TEXT except <">>

	for c := range 256 {
		var t octetType
		isCtl := c <= 31 || c == 127
		isChar := 0 <= c && c <= 127
		isSeparator := strings.ContainsRune(" \t\"(),/:;<=>?@[]\\{}", rune(c))
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			t |= isSpace
		}
		if isChar && !isCtl && !isSeparator {
			t |= isToken
		}
		octetTypes[c] = t
	}
}

// authHeader holds the parsed contents of a Www-Authenticate HTTP header.
type authHeader struct {
	scheme string
	params map[string]string
}

func challengeFromResponse(resp *http.Response) *authHeader {
	var h *authHeader
	for _, chalStr := range resp.Header["Www-Authenticate"] {
		h1 := parseWWWAuthenticate(chalStr)
		if h1 == nil {
			continue
		}
		if h1.scheme != "basic" && h1.scheme != "bearer" {
			continue
		}
		if h == nil {
			h = h1
		} else if h1.scheme == "basic" && h.scheme == "bearer" {
			// We prefer basic auth to bearer auth.
			h = h1
		}
	}
	return h
}

// parseWWWAuthenticate parses the contents of a Www-Authenticate HTTP header.
// It returns nil if the parsing fails.
func parseWWWAuthenticate(header string) *authHeader {
	var h authHeader
	h.params = make(map[string]string)

	scheme, s := expectToken(header)
	if scheme == "" {
		return nil
	}
	h.scheme = strings.ToLower(scheme)
	s = skipSpace(s)
	for len(s) > 0 {
		var pkey, pvalue string
		pkey, s = expectToken(skipSpace(s))
		if pkey == "" {
			return nil
		}
		if !strings.HasPrefix(s, "=") {
			return nil
		}
		pvalue, s = expectTokenOrQuoted(s[1:])
		if pvalue == "" {
			return nil
		}
		h.params[strings.ToLower(pkey)] = pvalue
		s = skipSpace(s)
		if !strings.HasPrefix(s, ",") {
			break
		}
		s = s[1:]
	}
	if len(s) > 0 {
		return nil
	}
	return &h
}

func skipSpace(s string) (rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isSpace == 0 {
			break
		}
	}
	return s[i:]
}

func expectToken(s string) (token, rest string) {
	i := 0
	for ; i < len(s); i++ {
		if octetTypes[s[i]]&isToken == 0 {
			break
		}
	}
	return s[:i], s[i:]
}

func expectTokenOrQuoted(s string) (value string, rest string) {
	if !strings.HasPrefix(s, "\"") {
		return expectToken(s)
	}
	s = s[1:]
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			return s[:i], s[i+1:]
		case '\\':
			p := make([]byte, len(s)-1)
			j := copy(p, s[:i])
			escape := true
			for i = i + 1; i < len(s); i++ {
				b := s[i]
				switch {
				case escape:
					escape = false
					p[j] = b
					j++
				case b == '\\':
					escape = true
				case b == '"':
					return string(p[:j]), s[i+1:]
				default:
					p[j] = b
					j++
				}
			}
			return "", ""
		}
	}
	return "", ""
}
