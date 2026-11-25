package avro

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// maxJSONValueNesting bounds how deeply decodeSchemaAny descends. It is the
// depth the stdlib decoder this replaced accepted, kept exactly so a schema at
// the boundary is decided the same way it always was: 10000 nested values
// parse, 10001 do not.
//
// Distinct from maxSchemaJSONDepth, which bounds the BRACKET nesting of schema
// text in a pre-scan before any decode. That one exists to reject
// pathologically deep input up front and is far tighter; this one is the
// decoder's own stack guard, and it also covers the two callers that decode
// text the pre-scan never saw (a preserved default, a struct tag).
const maxJSONValueNesting = 10000

// decodeSchemaAny decodes ONE JSON value out of schema text into the generic
// tree every schema surface is written against: map[string]any for objects,
// []any for arrays, string, bool, nil, and json.Number for every number. It
// returns how many bytes the value consumed, so each caller decides for itself
// what trailing content means.
//
// This is the ONE decoder for schema JSON. Five sites used to hand-spell the
// same json.NewDecoder + UseNumber + Decode(&v any) — the parse tree, the
// metadata tree, a preserved field default, the cache's input normalization,
// and a struct-tag default — and three of them also hand-spelled their own
// trailing-content rule. A hand-spelled decode is a snapshot of the rule at the
// moment it was typed, and the sites had already begun to drift.
//
// [unmarshalAnyPreservePrecision] layers normalizeJSONValue over this for the
// metadata surfaces. Nothing else does, and that split is deliberate:
//
// # Numbers come back verbatim as json.Number, and must stay that way
//
// Resolving numbers here — to int64 / float64 via normalizeJSONNumber — would
// pay for itself on the metadata path, which normalizes every number anyway.
// It would also break the parse path in two ways that raise NO error and
// produce no diagnostic, only wrong output:
//
//   - A default is preserved as the bytes the author wrote: aobjectFromMap and
//     afieldFromAny re-marshal the decoded value into o.Default / f.Default,
//     and the wire encoder parses those bytes much later. Integer-syntax "-0"
//     resolved to a number loses its IEEE sign — an exact-integer rational has
//     no signed zero — so the re-marshal writes "0" and a float field whose
//     author wrote -0.0 silently encodes +0.0.
//
//   - A resolved number re-marshals to its SHORTEST form, which can be
//     drastically shorter than the literal. "1." followed by 1021 zeros and
//     "e3" is 1025 bytes, one past the float literal length cap that every
//     other arm enforces; resolved, it is the exact integer 1000, whose
//     re-marshal is four bytes. The cap then never sees an over-long literal
//     and the parse accepts a default that encode, decode, and metadata all
//     refuse.
//
// Keeping the literal is what makes both the sign and the cap a property of
// the INPUT rather than of how far the decode happened to resolve it.
//
// # Strings share the input's memory
//
// A key or string value carrying no escapes is returned as a SUBSTRING of
// schema rather than a copy, which is where most of the allocation saving comes
// from. Nothing can be written through a string, so no aliasing SEMANTICS
// follow — two [Schema.Root] trees stay mutually independent, and mutating one
// is still invisible to the other — but the LIFETIME does change: a Go string
// keeps its whole backing array alive, so any one of these substrings pins the
// entire schema text.
//
// Inside a [Schema] that costs nothing, since the text is already retained
// verbatim for [Schema.String]. It costs something wherever a piece outlives
// its schema: a [SchemaNode] kept after its *Schema is dropped, one short string
// pulled out of [SchemaNode.Props] and held, or a [SchemaCache], which retains
// the definitions of every schema it parsed. The bound is one copy of the text
// per schema, not per string.
func decodeSchemaAny(schema string) (any, int, error) {
	d := &schemaDecoder{in: schema}
	d.space()
	if d.at >= len(d.in) {
		// Text holding no value at all reports [io.EOF], and text that runs
		// out PART WAY through one reports [io.ErrUnexpectedEOF] (see
		// schemaDecoder.eof). Both sentinels reach a caller of Parse
		// unwrapped-through, because parse echoes a decode error rather than
		// replacing it, so errors.Is on either keeps answering what it
		// always did — the distinction is the only thing separating "you
		// handed me nothing" from "you handed me a truncated schema".
		return nil, 0, io.EOF
	}
	v, err := d.value(0)
	if err != nil {
		return nil, 0, err
	}
	return v, d.at, nil
}

// decodeSchemaAnyStrict is decodeSchemaAny plus the rule that the value must be
// the WHOLE input: trailing whitespace is consumed, anything else is content
// and rejects. Every schema-text caller wants this — a schema string with a
// second value after it is not a schema — and stating it once keeps the callers
// from spelling three different versions of "is there anything left".
func decodeSchemaAnyStrict(schema string) (any, error) {
	v, n, err := decodeSchemaAny(schema)
	if err != nil {
		return nil, err
	}
	for ; n < len(schema); n++ {
		switch schema[n] {
		case ' ', '\t', '\n', '\r':
		default:
			return nil, errors.New("invalid schema: unexpected trailing content")
		}
	}
	return v, nil
}

// schemaDecoder walks schema text with an index cursor. It never writes to in,
// so every string it hands back may share in's backing array.
type schemaDecoder struct {
	in  string
	at  int
	buf []byte // reused scratch for strings carrying escapes
}

func (d *schemaDecoder) errorf(format string, args ...any) error {
	return fmt.Errorf("invalid schema json: "+format, args...)
}

// eof reports text that ran out PART WAY through a value, which is a different
// answer from text holding no value at all: the first is a truncated schema,
// the second is an absent one. A decode into a generic tree has always drawn
// that line with [io.ErrUnexpectedEOF] against [io.EOF], and parse echoes the
// decode error rather than replacing it, so both sentinels are reachable by
// errors.Is on what Parse returns.
func (d *schemaDecoder) eof(what string) error {
	return fmt.Errorf("invalid schema json: %s at offset %d: %w", what, d.at, io.ErrUnexpectedEOF)
}

func (d *schemaDecoder) space() {
	for d.at < len(d.in) {
		switch d.in[d.at] {
		case ' ', '\t', '\n', '\r':
			d.at++
		default:
			return
		}
	}
}

// literal consumes an expected bare token (true / false / null).
func (d *schemaDecoder) literal(tok string, v any) (any, error) {
	if d.at+len(tok) <= len(d.in) && d.in[d.at:d.at+len(tok)] == tok {
		d.at += len(tok)
		return v, nil
	}
	if strings.HasPrefix(tok, d.in[d.at:]) {
		return nil, d.eof("truncated literal")
	}
	return nil, d.errorf("invalid literal at offset %d", d.at)
}

// value decodes one value at the cursor, which space() has already advanced to
// a non-space byte. depth counts enclosing containers.
//
// The nesting bound is charged on the two CONTAINER arms, not here, because a
// scalar costs no nesting: it is a leaf, it recurses no further, and a decode
// into a generic tree pushes state for containers alone. Charging every value
// spends the last unit of the budget on the innermost leaf, so a shape whose
// innermost container is NON-EMPTY refuses one level early — [ x10000 holding a
// number is 10000 containers and must parse, while [ x10000 holding nothing is
// the same 10000 and always did. The empty-innermost shapes agree either way,
// which is what makes the difference easy to miss.
func (d *schemaDecoder) value(depth int) (any, error) {
	if d.at >= len(d.in) {
		return nil, d.eof("unexpected end of input")
	}
	switch d.in[d.at] {
	case '{':
		if depth >= maxJSONValueNesting {
			return nil, d.errorf("exceeded max depth at offset %d", d.at)
		}
		return d.object(depth)
	case '[':
		if depth >= maxJSONValueNesting {
			return nil, d.errorf("exceeded max depth at offset %d", d.at)
		}
		return d.array(depth)
	case '"':
		return d.str()
	case 't':
		return d.literal("true", true)
	case 'f':
		return d.literal("false", false)
	case 'n':
		return d.literal("null", nil)
	default:
		return d.number()
	}
}

func (d *schemaDecoder) object(depth int) (any, error) {
	d.at++ // '{'
	m := make(map[string]any)
	d.space()
	if d.at < len(d.in) && d.in[d.at] == '}' {
		d.at++
		return m, nil
	}
	for {
		d.space()
		if d.at >= len(d.in) {
			return nil, d.eof("unexpected end of object")
		}
		if d.in[d.at] != '"' {
			return nil, d.errorf("expected object key at offset %d", d.at)
		}
		k, err := d.str()
		if err != nil {
			return nil, err
		}
		d.space()
		if d.at >= len(d.in) {
			return nil, d.eof("unexpected end of object")
		}
		if d.in[d.at] != ':' {
			return nil, d.errorf("expected ':' at offset %d", d.at)
		}
		d.at++
		d.space()
		v, err := d.value(depth + 1)
		if err != nil {
			return nil, err
		}
		// Last duplicate key wins, which is what a decode into a map does
		// and what every reference implementation lands on.
		m[k.(string)] = v
		d.space()
		if d.at >= len(d.in) {
			return nil, d.eof("unexpected end of object")
		}
		switch d.in[d.at] {
		case ',':
			d.at++
		case '}':
			d.at++
			return m, nil
		default:
			return nil, d.errorf("expected ',' or '}' at offset %d", d.at)
		}
	}
}

func (d *schemaDecoder) array(depth int) (any, error) {
	d.at++ // '['
	d.space()
	if d.at < len(d.in) && d.in[d.at] == ']' {
		d.at++
		// Empty and absent are different values to every caller that
		// range-checks a decoded array, so an empty array is a non-nil
		// empty slice.
		return []any{}, nil
	}
	var a []any
	for {
		d.space()
		v, err := d.value(depth + 1)
		if err != nil {
			return nil, err
		}
		a = append(a, v)
		d.space()
		if d.at >= len(d.in) {
			return nil, d.eof("unexpected end of array")
		}
		switch d.in[d.at] {
		case ',':
			d.at++
		case ']':
			d.at++
			return a, nil
		default:
			return nil, d.errorf("expected ',' or ']' at offset %d", d.at)
		}
	}
}

// str decodes a JSON string. It returns any so the value dispatch has one
// return shape; object() asserts the string back for a key.
//
// The common case — no escapes, valid UTF-8 — hands back a substring of the
// input, so a schema's keys and string values cost no allocation at all.
func (d *schemaDecoder) str() (any, error) {
	d.at++ // opening quote
	start := d.at
	highByte := false
	for d.at < len(d.in) {
		c := d.in[d.at]
		switch {
		case c == '"':
			s := d.in[start:d.at]
			d.at++
			if highByte && !utf8.ValidString(s) {
				s = replaceInvalidUTF8(s)
			}
			return s, nil
		case c == '\\':
			return d.strEscaped(start)
		case c < 0x20:
			// RFC 8259 section 7: the control characters must be escaped.
			return nil, d.errorf("unescaped control character %#x in string at offset %d", c, d.at)
		}
		if c >= utf8.RuneSelf {
			highByte = true
		}
		d.at++
	}
	return nil, d.eof("unterminated string")
}

// strEscaped finishes a string that carries at least one escape, copying into
// scratch. The cursor sits on the first backslash and start marks the opening
// quote's first content byte.
func (d *schemaDecoder) strEscaped(start int) (any, error) {
	buf := append(d.buf[:0], d.in[start:d.at]...)
	for d.at < len(d.in) {
		c := d.in[d.at]
		switch {
		case c == '"':
			d.at++
			d.buf = buf
			s := string(buf)
			if !utf8.ValidString(s) {
				s = replaceInvalidUTF8(s)
			}
			return s, nil
		case c == '\\':
			d.at++
			if d.at >= len(d.in) {
				return nil, d.eof("unterminated escape")
			}
			e := d.in[d.at]
			d.at++
			switch e {
			case '"', '\\', '/':
				buf = append(buf, e)
			case 'b':
				buf = append(buf, '\b')
			case 'f':
				buf = append(buf, '\f')
			case 'n':
				buf = append(buf, '\n')
			case 'r':
				buf = append(buf, '\r')
			case 't':
				buf = append(buf, '\t')
			case 'u':
				r, err := d.unicodeEscape()
				if err != nil {
					return nil, err
				}
				buf = utf8.AppendRune(buf, r)
			default:
				return nil, d.errorf("invalid escape %q at offset %d", e, d.at-1)
			}
		case c < 0x20:
			return nil, d.errorf("unescaped control character %#x in string at offset %d", c, d.at)
		default:
			buf = append(buf, c)
			d.at++
		}
	}
	return nil, d.eof("unterminated string")
}

// unicodeEscape reads the four hex digits after \u, pairing a leading
// surrogate with a trailing one when the next escape supplies it. An unpaired
// surrogate becomes the replacement rune, as a decode into a Go string does —
// UTF-8 cannot represent one.
func (d *schemaDecoder) unicodeEscape() (rune, error) {
	// Left to right, one digit at a time, so a non-hex character reports
	// itself and only genuinely running out of text reports truncation. The
	// order matters: checking the remaining LENGTH first would call "\uX"
	// truncated when what is wrong with it is the X.
	var r rune
	for i := 0; i < 4; i++ {
		if d.at >= len(d.in) {
			return 0, d.eof("truncated \\u escape")
		}
		h, ok := hexDigit(d.in[d.at])
		if !ok {
			return 0, d.errorf("invalid character %q in \\u escape at offset %d", d.in[d.at], d.at)
		}
		r = r<<4 | rune(h)
		d.at++
	}
	if !utf16.IsSurrogate(r) {
		return r, nil
	}
	// A leading surrogate pairs with a trailing one when the next escape
	// supplies it. This is a LOOKAHEAD, so anything else here is not an
	// error — it just leaves the surrogate unpaired.
	if d.at+6 <= len(d.in) && d.in[d.at] == '\\' && d.in[d.at+1] == 'u' {
		if lo, ok := hex4(d.in[d.at+2 : d.at+6]); ok {
			if paired := utf16.DecodeRune(r, lo); paired != utf8.RuneError {
				d.at += 6
				return paired, nil
			}
		}
	}
	// UTF-8 cannot represent a lone surrogate, so it becomes the
	// replacement rune, which is what decoding into a Go string produces.
	return utf8.RuneError, nil
}

func hexDigit(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func hex4(s string) (rune, bool) {
	var r rune
	for i := 0; i < 4; i++ {
		h, ok := hexDigit(s[i])
		if !ok {
			return 0, false
		}
		r = r<<4 | rune(h)
	}
	return r, true
}

// number consumes a JSON number and returns its literal. See the type doc for
// why the literal, and not a resolved value, is what leaves this decoder.
func (d *schemaDecoder) number() (any, error) {
	start := d.at
	if d.at < len(d.in) && d.in[d.at] == '-' {
		d.at++
	}
	switch {
	case d.at >= len(d.in):
		return nil, d.eof("truncated number")
	case d.in[d.at] == '0':
		// RFC 8259: no leading zeros. A second digit here ends the number
		// after the '0', which is what a decode does too — the leftover
		// digit becomes trailing content for the caller to rule on.
		d.at++
	case d.in[d.at] >= '1' && d.in[d.at] <= '9':
		for d.at < len(d.in) && isDigit(d.in[d.at]) {
			d.at++
		}
	default:
		return nil, d.errorf("expected number at offset %d", start)
	}
	if d.at < len(d.in) && d.in[d.at] == '.' {
		d.at++
		if d.at >= len(d.in) {
			return nil, d.eof("truncated number")
		}
		if !isDigit(d.in[d.at]) {
			return nil, d.errorf("expected digit after decimal point at offset %d", d.at)
		}
		for d.at < len(d.in) && isDigit(d.in[d.at]) {
			d.at++
		}
	}
	if d.at < len(d.in) && (d.in[d.at] == 'e' || d.in[d.at] == 'E') {
		d.at++
		if d.at < len(d.in) && (d.in[d.at] == '+' || d.in[d.at] == '-') {
			d.at++
		}
		if d.at >= len(d.in) {
			return nil, d.eof("truncated number")
		}
		if !isDigit(d.in[d.at]) {
			return nil, d.errorf("expected digit in exponent at offset %d", d.at)
		}
		for d.at < len(d.in) && isDigit(d.in[d.at]) {
			d.at++
		}
	}
	return json.Number(d.in[start:d.at]), nil
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// replaceInvalidUTF8 substitutes the replacement rune for each byte that is not
// part of a valid encoding — one per BYTE, which is what decoding into a Go
// string produces, and not one per invalid run.
func replaceInvalidUTF8(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(utf8.RuneError)
			i++
			continue
		}
		b.WriteString(s[i : i+size])
		i += size
	}
	return b.String()
}
