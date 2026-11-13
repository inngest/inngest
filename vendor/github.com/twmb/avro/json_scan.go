package avro

import (
	"fmt"
	"math"
	"unicode/utf8"
	"unsafe"
)

// jsonScanner is a minimal zero-allocation JSON byte scanner.
// It operates on a []byte input and exposes a cursor-style API
// for the schema-guided decoder to extract tokens.
type jsonScanner struct {
	data []byte
	pos  int
}

func (s *jsonScanner) skipWhitespace() {
	for s.pos < len(s.data) {
		switch s.data[s.pos] {
		case ' ', '\t', '\n', '\r':
			s.pos++
		default:
			return
		}
	}
}

// peek returns the next non-whitespace byte, or 0 if at EOF.
func (s *jsonScanner) peek() byte {
	s.skipWhitespace()
	if s.pos >= len(s.data) {
		return 0
	}
	return s.data[s.pos]
}

// peekAt returns the byte `n` positions past the current
// non-whitespace cursor, without consuming. Used to disambiguate
// negative-number vs bare "-Infinity" at the start of a token.
func (s *jsonScanner) peekAt(n int) byte {
	s.skipWhitespace()
	if s.pos+n >= len(s.data) {
		return 0
	}
	return s.data[s.pos+n]
}

func (s *jsonScanner) expect(b byte) error {
	s.skipWhitespace()
	if s.pos >= len(s.data) {
		return fmt.Errorf("avro json: unexpected EOF, expected %q", b)
	}
	if s.data[s.pos] != b {
		return fmt.Errorf("avro json: expected %q, got %q at offset %d", b, s.data[s.pos], s.pos)
	}
	s.pos++
	return nil
}

func (s *jsonScanner) consumeNull() error {
	s.skipWhitespace()
	if s.pos+4 > len(s.data) || string(s.data[s.pos:s.pos+4]) != "null" {
		return fmt.Errorf("avro json: expected null at offset %d", s.pos)
	}
	s.pos += 4
	return nil
}

// consumeBareSpecialFloat consumes a bare JSON token shaped like an
// optional leading '-' followed by an alphabetic run. The token's
// content is returned verbatim for parseSpecialFloat to validate, so
// casing leniency (NaN/nan/NAN, Infinity/inf/Inf, etc.) is identical
// between this bare path and the quoted-string path in decodeFloat/
// decodeDouble — preventing a quoted-vs-bare casing-parity gap.
func (s *jsonScanner) consumeBareSpecialFloat() (string, error) {
	s.skipWhitespace()
	if s.pos >= len(s.data) {
		return "", fmt.Errorf("avro json: expected bare NaN/Infinity at offset %d", s.pos)
	}
	start := s.pos
	if s.data[s.pos] == '-' {
		s.pos++
	}
	tokenStart := s.pos
	for s.pos < len(s.data) {
		c := s.data[s.pos]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			s.pos++
			continue
		}
		break
	}
	if s.pos == tokenStart {
		s.pos = start
		return "", fmt.Errorf("avro json: expected bare NaN/Infinity at offset %d", start)
	}
	return string(s.data[start:s.pos]), nil
}

func (s *jsonScanner) consumeBool() (bool, error) {
	s.skipWhitespace()
	if s.pos+4 <= len(s.data) && string(s.data[s.pos:s.pos+4]) == "true" {
		s.pos += 4
		return true, nil
	}
	if s.pos+5 <= len(s.data) && string(s.data[s.pos:s.pos+5]) == "false" {
		s.pos += 5
		return false, nil
	}
	return false, fmt.Errorf("avro json: expected boolean at offset %d", s.pos)
}

// consumeStringRaw consumes a JSON string and returns the raw content
// between the quotes (with escape sequences unresolved). Returns start
// and end offsets into s.data. For strings without escapes (the common
// case), the caller can use s.data[start:end] directly.
func (s *jsonScanner) consumeStringRaw() (start, end int, hasEscapes bool, err error) {
	s.skipWhitespace()
	if s.pos >= len(s.data) || s.data[s.pos] != '"' {
		return 0, 0, false, fmt.Errorf("avro json: expected string at offset %d", s.pos)
	}
	s.pos++ // skip opening quote
	start = s.pos
	sawHighByte := false
	for s.pos < len(s.data) {
		b := s.data[s.pos]
		if b == '\\' {
			hasEscapes = true
			s.pos += 2 // skip escape + next char
			if s.pos > len(s.data) {
				return 0, 0, false, fmt.Errorf("avro json: unterminated escape at offset %d", s.pos-2)
			}
			continue
		}
		if b == '"' {
			end = s.pos
			s.pos++ // skip closing quote
			// RFC 8259: JSON text is UTF-8. Reject literal invalid byte
			// sequences. Gated by sawHighByte so pure-ASCII content (the
			// common case) skips the scan; \uXXXX escapes are ASCII here
			// and resolve to valid runes during walkJSONEscapes.
			if sawHighByte && !utf8.Valid(s.data[start:end]) {
				return 0, 0, false, fmt.Errorf("avro json: invalid UTF-8 in string at offset %d", start)
			}
			return start, end, hasEscapes, nil
		}
		// RFC 8259 §7: control characters U+0000–U+001F must be escaped.
		if b < 0x20 {
			return 0, 0, false, fmt.Errorf("avro json: unescaped control character %#x in string at offset %d", b, s.pos)
		}
		if b >= 0x80 {
			sawHighByte = true
		}
		s.pos++
	}
	return 0, 0, false, fmt.Errorf("avro json: unterminated string at offset %d", start-1)
}

// consumeStringZeroCopy consumes a JSON string and returns a zero-copy
// string backed by the scanner's input data. The returned string is only
// valid during the current DecodeJSON call — do NOT store it in output.
// Falls back to a copying path for strings with escape sequences.
func (s *jsonScanner) consumeStringZeroCopy() (string, error) {
	start, end, hasEscapes, err := s.consumeStringRaw()
	if err != nil {
		return "", err
	}
	if !hasEscapes {
		return unsafe.String(unsafe.SliceData(s.data[start:end]), end-start), nil
	}
	return resolveJSONEscapes(s.data[start:end])
}

// consumeNumberBytes consumes a JSON number and returns the raw bytes.
func (s *jsonScanner) consumeNumberBytes() ([]byte, error) {
	s.skipWhitespace()
	start := s.pos
	if s.pos < len(s.data) && s.data[s.pos] == '-' {
		s.pos++
	}
	if s.pos >= len(s.data) || s.data[s.pos] < '0' || s.data[s.pos] > '9' {
		return nil, fmt.Errorf("avro json: expected number at offset %d", start)
	}
	// Per RFC 8259: leading zeros are not permitted (except "0" itself
	// or "0.xxx"). If the first digit is '0', the next must be '.' or
	// end-of-number.
	if s.data[s.pos] == '0' && s.pos+1 < len(s.data) {
		next := s.data[s.pos+1]
		if next >= '0' && next <= '9' {
			return nil, fmt.Errorf("avro json: leading zeros not permitted at offset %d", start)
		}
	}
	for s.pos < len(s.data) {
		b := s.data[s.pos]
		if (b >= '0' && b <= '9') || b == '.' || b == 'e' || b == 'E' || b == '+' || b == '-' {
			s.pos++
			continue
		}
		break
	}
	return s.data[start:s.pos], nil
}

// skipValue skips an entire JSON value (for unknown record fields).
//
// Accepts the same bare special-float tokens decodeJSONFloat does — NaN,
// Infinity, -Infinity, INF, -INF, Inf, -Inf — so a fastavro-written record
// (json.dumps with allow_nan=True emits them bare, observed) decodes against a
// reader lacking that field. Java emits the quoted form, which the string arm
// already skips. Dispatches on the value path's [isBareSpecialFloatStart] and
// parseSpecialFloat's exact-match gate, so "Naive" and lowercase "nan" still
// error and the two paths cannot drift.
func (s *jsonScanner) skipValue() error {
	return s.skipValueDepth(0)
}

// skipValueDepth skips one JSON value while VALIDATING its full grammar, not
// merely delimiting it. Unknown record fields route here. The former
// delimit-only skip was a SECOND, lax JSON parser that accepted malformed
// input the value path (and Java/fastavro/encoding/json) reject: the number
// arm took 1.2.3/1e/5., the string arm skipped escapes blindly so "\q" passed,
// and skipCompound counted only bracket depth so [}] / {"a" 1} / [1,2,]
// "balanced". depth bounds recursion so a pathologically deep skipped value
// errors rather than overflowing the stack (the old skipCompound was
// iterative; this validator is recursive).
func (s *jsonScanner) skipValueDepth(depth int) error {
	// Use the value path's >= maxDepth (trips at the maxDepth-th level), not
	// >, so the two recursion guards agree on the bound. depth restarts at 0
	// per skipped value rather than threading the enclosing decode depth: a
	// skipped value is a self-contained sub-parse that discards its data, so a
	// fresh maxDepth budget is fine (the worst case, a deep value with a deep
	// skipped tail, is still ~2*maxDepth frames — bounded for stack/DoS).
	if depth >= maxDepth {
		return errTooDeep
	}
	s.skipWhitespace()
	if s.pos >= len(s.data) {
		return fmt.Errorf("avro json: unexpected EOF")
	}
	switch p := s.data[s.pos]; p {
	case '"':
		return s.skipStringStrict()
	case 't', 'f':
		_, err := s.consumeBool()
		return err
	case 'n':
		// Unambiguously the null literal: lowercase is not a bare-token
		// start on either path.
		return s.consumeNull()
	case '[':
		return s.skipArrayStrict(depth)
	case '{':
		return s.skipObjectStrict(depth)
	default:
		// A bare NaN/Infinity token and a number share this arm because
		// they share a first byte: '-' begins both -Infinity and every
		// negative number. isBareSpecialFloatStart is the value path's own
		// predicate (json_decode.go), so the skip path splits them exactly
		// where decodeJSONFloat does.
		if isBareSpecialFloatStart(s, p) {
			t, err := s.consumeBareSpecialFloat()
			if err != nil {
				return err
			}
			_, err = parseSpecialFloat(t)
			return err
		}
		return s.skipNumberStrict()
	}
}

// skipStringStrict consumes a JSON string and VALIDATES its escapes;
// consumeStringRaw checks control bytes and UTF-8 but delimits escapes blindly,
// so "\q"/"\x41" would otherwise pass.
func (s *jsonScanner) skipStringStrict() error {
	start, end, hasEscapes, err := s.consumeStringRaw()
	if err != nil {
		return err
	}
	if hasEscapes {
		return walkJSONEscapes(s.data[start:end], func(rune) error { return nil })
	}
	return nil
}

// skipNumberStrict consumes a JSON number and validates the RFC 8259 grammar
// through the SAME isJSONNumber gate the value path uses (parseJSONNumberAsFloat,
// json.Number) — so the skip path and value path cannot disagree on what a
// valid JSON number is. consumeNumberBytes only delimits a [0-9.eE+-] run, so
// without this gate 1.2.3/1e/5. would pass.
func (s *jsonScanner) skipNumberStrict() error {
	nb, err := s.consumeNumberBytes()
	if err != nil {
		return err
	}
	if !isJSONNumber(unsafe.String(unsafe.SliceData(nb), len(nb))) {
		return fmt.Errorf("avro json: invalid JSON number %q", truncForError(string(nb)))
	}
	return nil
}

func (s *jsonScanner) skipArrayStrict(depth int) error {
	s.pos++ // '['
	s.skipWhitespace()
	if s.pos < len(s.data) && s.data[s.pos] == ']' {
		s.pos++
		return nil
	}
	for {
		if err := s.skipValueDepth(depth + 1); err != nil {
			return err
		}
		s.skipWhitespace()
		if s.pos >= len(s.data) {
			return fmt.Errorf("avro json: unterminated array")
		}
		switch s.data[s.pos] {
		case ',':
			s.pos++
		case ']':
			s.pos++
			return nil
		default:
			return fmt.Errorf("avro json: expected ',' or ']' in array at offset %d", s.pos)
		}
	}
}

func (s *jsonScanner) skipObjectStrict(depth int) error {
	s.pos++ // '{'
	s.skipWhitespace()
	if s.pos < len(s.data) && s.data[s.pos] == '}' {
		s.pos++
		return nil
	}
	for {
		s.skipWhitespace()
		if s.pos >= len(s.data) || s.data[s.pos] != '"' {
			return fmt.Errorf("avro json: expected object key string at offset %d", s.pos)
		}
		if err := s.skipStringStrict(); err != nil {
			return err
		}
		s.skipWhitespace()
		if s.pos >= len(s.data) || s.data[s.pos] != ':' {
			return fmt.Errorf("avro json: expected ':' after object key at offset %d", s.pos)
		}
		s.pos++
		if err := s.skipValueDepth(depth + 1); err != nil {
			return err
		}
		s.skipWhitespace()
		if s.pos >= len(s.data) {
			return fmt.Errorf("avro json: unterminated object")
		}
		switch s.data[s.pos] {
		case ',':
			s.pos++
		case '}':
			s.pos++
			return nil
		default:
			return fmt.Errorf("avro json: expected ',' or '}' in object at offset %d", s.pos)
		}
	}
}

// parseJSONInt32 parses raw JSON number bytes directly as int32.
// Rejects fractional/exponent notation and out-of-range values.
func parseJSONInt32(b []byte) (int32, error) {
	n, err := parseJSONInt64(b)
	if err != nil {
		return 0, err
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("avro json: value %d overflows int32", n)
	}
	return int32(n), nil
}

// parseJSONInt64 parses raw JSON number bytes directly as int64.
// Rejects fractional/exponent notation.
func parseJSONInt64(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, fmt.Errorf("avro json: empty number")
	}
	neg := false
	i := 0
	if b[0] == '-' {
		neg = true
		i = 1
	}
	if i >= len(b) {
		return 0, fmt.Errorf("avro json: invalid number %q", truncBytesForError(b))
	}
	// Per-digit pre-multiply guard. The naive "n*10+d wrapped if it
	// went down" check has a gap once n ≈ 2^64/9: n*10+d can wrap
	// mod 2^64 to a value still ≥ prev (e.g. parsing
	// "20496382304121724020" lands at 2049638230412172404 with no
	// post-multiply wrap visible). Java's JsonParser.getLongValue
	// throws InputCoercionException; goavro uses strconv.ParseInt;
	// twmb/avro now does an equivalent pre-multiply bound.
	//
	// MaxInt64 = 9223372036854775807 → cutoff 922337203685477580, last digit 7.
	// |MinInt64| = 9223372036854775808 → same cutoff, last digit 8.
	const cutoff = uint64(math.MaxInt64) / 10
	maxDigit := uint64(7)
	if neg {
		maxDigit = 8
	}
	var n uint64
	for ; i < len(b); i++ {
		c := b[i]
		if c == '.' || c == 'e' || c == 'E' {
			// Fractional or exponent form: parse at arbitrary precision so
			// values near the int64 boundary are not truncated or rejected by
			// float64 rounding. "-9.2233720368547758e18" is the valid int64
			// -9223372036854775800 that float64 rounds to int64.Min, and its
			// positive twin rounds to int64.Max+1 and would be rejected.
			//
			// parseInt64Lenient and everything downstream treat s as read-only
			// and do not retain it, so alias b rather than copying.
			n, err := parseInt64Lenient(unsafe.String(unsafe.SliceData(b), len(b)))
			if err != nil {
				return 0, fmt.Errorf("avro json: %w", err)
			}
			return n, nil
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("avro json: invalid number %q", truncBytesForError(b))
		}
		d := uint64(c - '0')
		if n > cutoff || (n == cutoff && d > maxDigit) {
			return 0, fmt.Errorf("avro json: value %q overflows int64", truncBytesForError(b))
		}
		n = n*10 + d
	}
	if neg {
		// -int64(1<<63) wraps to MinInt64 in two's complement, which
		// is the correct value for the input "-9223372036854775808".
		return -int64(n), nil
	}
	return int64(n), nil
}

// walkJSONEscapes iterates raw JSON string content (between quotes),
// decoding escape sequences and UTF-8 multi-byte sequences into runes,
// then calling emit for each code point. Used by both resolveJSONEscapes
// (Avro string) and scanAvroJSONBytes (Avro bytes/fixed).
//
// The UTF-8 decoding is required for spec parity: "each character represents
// one byte", with code points 0-255 mapping to byte values, and both Java and
// fastavro decode to characters before mapping. A byte-by-byte walker emits
// literal "é" as the two bytes [0xC3, 0xA9] instead of the correct [0xE9].
func walkJSONEscapes(raw []byte, emit func(r rune) error) error {
	for i := 0; i < len(raw); {
		if raw[i] != '\\' {
			// Decode multi-byte UTF-8 as a single rune so the round trip
			// preserves the value. consumeStringRaw already rejected
			// invalid UTF-8, so DecodeRune always advances a full rune.
			r, size := utf8.DecodeRune(raw[i:])
			if err := emit(r); err != nil {
				return err
			}
			i += size
			continue
		}
		i++
		if i >= len(raw) {
			return fmt.Errorf("avro json: unterminated escape")
		}
		var r rune
		switch raw[i] {
		case '"', '\\', '/':
			r = rune(raw[i])
		case 'b':
			r = '\b'
		case 'f':
			r = '\f'
		case 'n':
			r = '\n'
		case 'r':
			r = '\r'
		case 't':
			r = '\t'
		case 'u':
			if i+4 >= len(raw) {
				return fmt.Errorf("avro json: short \\u escape")
			}
			var err error
			r, err = parseHex4(raw[i+1 : i+5])
			if err != nil {
				return err
			}
			i += 4
			if r >= 0xD800 && r <= 0xDBFF && i+2 < len(raw) && raw[i+1] == '\\' && raw[i+2] == 'u' {
				if i+6 < len(raw) {
					r2, err := parseHex4(raw[i+3 : i+7])
					if err == nil && r2 >= 0xDC00 && r2 <= 0xDFFF {
						r = 0x10000 + (r-0xD800)*0x400 + (r2 - 0xDC00)
						i += 6
					}
				}
			}
		default:
			// Unrecognized escape sequence. JSON defines exactly eight
			// (" \ / b f n r t) plus \uXXXX; anything else is malformed.
			// Reject rather than silently dropping the backslash (which
			// corrupts content — "C:\dir" would become "C:dir"). Matches
			// Java's JsonDecoder (Jackson, "Unrecognized character
			// escape") and fastavro (Python json, "Invalid \escape").
			return fmt.Errorf("avro json: invalid escape sequence \\%c", raw[i])
		}
		if err := emit(r); err != nil {
			return err
		}
		i++
	}
	return nil
}

// resolveJSONEscapes resolves JSON escape sequences in raw string content,
// producing a UTF-8 Go string.
func resolveJSONEscapes(raw []byte) (string, error) {
	var buf []byte
	err := walkJSONEscapes(raw, func(r rune) error {
		buf = utf8.AppendRune(buf, r)
		return nil
	})
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// scanAvroJSONBytes resolves a raw JSON string content into Avro bytes.
// In Avro's convention, each code point maps to a single byte (≤ 255).
func scanAvroJSONBytes(raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return []byte{}, nil
	}
	var buf []byte
	err := walkJSONEscapes(raw, func(r rune) error {
		if r > 255 {
			return fmt.Errorf("avro json: \\u%04X exceeds byte range in bytes field", r)
		}
		buf = append(buf, byte(r))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func parseHex4(b []byte) (rune, error) {
	if len(b) < 4 {
		return 0, fmt.Errorf("avro json: short hex escape")
	}
	var r rune
	for _, c := range b[:4] {
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			r |= rune(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("avro json: invalid hex digit %q", c)
		}
	}
	return r, nil
}
