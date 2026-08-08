package driver

import (
	"github.com/gowebpki/jcs"
)

// canonicalize transforms marshalled JSON into its RFC 8785 canonical form.
//
// jcs rejects JSON containing unpaired UTF-16 surrogate escapes (eg. `\ud83d`
// not followed by a low surrogate) with "Unexpected non-ASCII character" or
// "Missing surrogate". These escapes are valid JSON, and JavaScript's
// JSON.stringify produces them whenever a string was truncated mid surrogate
// pair (emoji, astral-plane CJK, etc.), so they show up in real event and
// step data. Because the transform runs on every step invocation, a single
// such value poisons the run: every attempt fails identically and the run
// dead-letters.
//
// When the transform fails, unpaired surrogate escapes are replaced with
// `\ufffd` (the escaped Unicode replacement character) and the transform is retried
// once. The repair only runs on the failure path, so well-formed payloads pay
// no extra cost, and the output stays deterministic: the same input always
// produces the same canonical bytes, which is what request signing requires.
func canonicalize(j []byte) ([]byte, error) {
	b, err := jcs.Transform(j)
	if err == nil {
		return b, nil
	}
	repaired := repairUnpairedSurrogateEscapes(j)
	if repaired == nil {
		// Nothing to repair; the input is broken in some other way.
		return nil, err
	}
	return jcs.Transform(repaired)
}

// repairUnpairedSurrogateEscapes replaces unpaired `\uXXXX` surrogate escapes
// inside JSON string literals with the escaped Unicode replacement character
// `\ufffd`. Paired surrogates and all other content are left untouched.
// Returns nil when the input needs no repairs.
func repairUnpairedSurrogateEscapes(j []byte) []byte {
	out := make([]byte, 0, len(j))
	repaired := false
	inString := false

	for i := 0; i < len(j); i++ {
		c := j[i]

		if !inString {
			if c == '"' {
				inString = true
			}
			out = append(out, c)
			continue
		}

		switch {
		case c == '"':
			inString = false
			out = append(out, c)
		case c == '\\':
			if i+1 < len(j) && j[i+1] != 'u' {
				// A simple escape such as \\ or \" — copy both bytes so the
				// escaped character can't be misread as a string delimiter.
				out = append(out, c, j[i+1])
				i++
				continue
			}
			first, ok := parseUEscape(j, i)
			if !ok {
				// Truncated or malformed escape; copy it through and let
				// jcs report it.
				out = append(out, c)
				continue
			}
			if isHighSurrogate(first) {
				if second, ok := parseUEscape(j, i+6); ok && isLowSurrogate(second) {
					// A valid pair; copy both escapes untouched.
					out = append(out, j[i:i+12]...)
					i += 11
					continue
				}
			}
			if isHighSurrogate(first) || isLowSurrogate(first) {
				out = append(out, '\\', 'u', 'f', 'f', 'f', 'd')
				repaired = true
				i += 5
				continue
			}
			// An ordinary \uXXXX escape.
			out = append(out, j[i:i+6]...)
			i += 5
		default:
			out = append(out, c)
		}
	}

	if !repaired {
		return nil
	}
	return out
}

// parseUEscape parses a `\uXXXX` escape starting at offset i and returns its
// UTF-16 code unit. ok is false when the bytes at i do not form a full escape.
func parseUEscape(j []byte, i int) (uint16, bool) {
	if i+5 >= len(j) || j[i] != '\\' || j[i+1] != 'u' {
		return 0, false
	}
	var v uint16
	for _, c := range j[i+2 : i+6] {
		var d uint16
		switch {
		case c >= '0' && c <= '9':
			d = uint16(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint16(c-'A') + 10
		default:
			return 0, false
		}
		v = v<<4 | d
	}
	return v, true
}

func isHighSurrogate(u uint16) bool { return u >= 0xD800 && u <= 0xDBFF }

func isLowSurrogate(u uint16) bool { return u >= 0xDC00 && u <= 0xDFFF }
