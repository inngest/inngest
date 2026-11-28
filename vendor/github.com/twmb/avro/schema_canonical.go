package avro

import (
	"bytes"
	"encoding/json"
	"strconv"
	"unicode/utf8"
)

// appendCompactJSON appends raw JSON with insignificant whitespace
// removed, matching json.Marshal(json.RawMessage). Only reached for the
// non-PCF "default" attribute, which the canon tree strips — present for
// faithfulness when the writer is used on an unstripped tree.
func appendCompactJSON(dst, raw []byte) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return append(dst, raw...)
	}
	return append(dst, buf.Bytes()...)
}

// canonicalBytes serializes the (already first-occurrence-rewritten and
// strip/order-canonicalized) schema tree to its Parsing Canonical Form
// bytes in a SINGLE pass.
//
// The former path encoded via nested aobject/aschema MarshalJSON methods,
// each returning its full subtree bytes which the parent then copied into
// its own buffer — O(depth*size) = O(n^2) over a nested schema. It also
// produced HTML escapes (< etc.) and U+2028/U+2029 escapes that an
// outer bytes.ReplaceAll then tried to undo, which was unsound for a
// string containing a literal backslash (the 6-byte \uXXXX target appears
// inside the \\uXXXX escape of such a string, so ReplaceAll collapsed it
// to invalid JSON and a corrupt fingerprint).
//
// This writer appends to ONE buffer (O(n)) and emits strings as raw UTF-8
// per the PCF [STRINGS] rule (only the mandatory JSON escapes — quote,
// backslash, controls — are escaped; <, >, &, U+2028, U+2029 and all
// other code points are written verbatim), matching Java's
// SchemaNormalization and eliminating the un-escape round trip.
func canonicalBytes(root aschema) []byte {
	dst := make([]byte, 0, 256)
	return appendCanonSchema(dst, &root)
}

func appendCanonSchema(dst []byte, s *aschema) []byte {
	switch {
	case s.primitive != "":
		return appendCanonString(dst, s.primitive)
	case s.object != nil:
		return appendCanonObject(dst, s.object)
	case s.union != nil:
		// Non-nil discriminates union-ness so a zero-branch union emits
		// `[]` rather than falling into the default arm.
		dst = append(dst, '[')
		for i := range s.union {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = appendCanonSchema(dst, &s.union[i])
		}
		return append(dst, ']')
	default:
		// An empty aschema cannot occur in a built canon tree; emit an
		// empty string so callers never see truncated JSON.
		return append(dst, '"', '"')
	}
}

// appendCanonObject writes an aobject in PCF key order with the
// required-empty-array rules (record/error always emit "fields"; enum always
// emits "symbols"). It also emits the non-PCF attributes (namespace, aliases,
// default, order, logicalType, precision, scale) when present, so it doubles
// as a general-purpose object writer usable on an UNSTRIPPED tree — exercised
// directly in that mode by schema_test.go. The Canonical() entry point feeds
// an already-stripped tree (canonicalFirstOccurrence), so on that path only
// name/type/the required arrays appear and the attribute branches are not
// reached.
func appendCanonObject(dst []byte, o *aobject) []byte {
	dst = append(dst, '{')
	first := true
	key := func(dst []byte, k string) []byte {
		if !first {
			dst = append(dst, ',')
		}
		first = false
		dst = appendCanonString(dst, k)
		return append(dst, ':')
	}

	// A named KIND always emits its name — including the empty fullname a
	// user WithLaxNames fn can accept ("name":""), matching fastavro's PCF
	// (executed, 1.12.2), the only other implementation known to parse the
	// shape; omitting it emitted a missing-name spelling instead. The
	// Name != "" arm keeps emission for hand-built objects that carry a
	// name on a non-named kind.
	if o.Name != "" || isNamedKind(o.Type) {
		dst = key(dst, "name")
		dst = appendCanonString(dst, o.Name)
	}
	dst = key(dst, "type")
	dst = appendCanonString(dst, o.Type)

	// One rule per required array: the kind that REQUIRES the key always
	// emits it (even empty — a record with no "fields" or an enum with no
	// "symbols" is unparseable), and any other kind emits it only when it
	// carries one. Stated as a single condition because both halves emit
	// identically; the metadata emitter states its "fields" rule the same
	// way (toJSONWalk, schema_node.go).
	if isRecordKind(o.Type) || len(o.Fields) > 0 {
		dst = key(dst, "fields")
		dst = append(dst, '[')
		for i := range o.Fields {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = appendCanonField(dst, &o.Fields[i])
		}
		dst = append(dst, ']')
	}

	if o.Type == "enum" || len(o.Symbols) > 0 {
		dst = key(dst, "symbols")
		dst = appendCanonStringArray(dst, o.Symbols)
	}
	if o.Items != nil {
		dst = key(dst, "items")
		dst = appendCanonSchema(dst, o.Items)
	}
	if o.Values != nil {
		dst = key(dst, "values")
		dst = appendCanonSchema(dst, o.Values)
	}
	if o.Size != nil {
		dst = key(dst, "size")
		dst = strconv.AppendInt(dst, int64(*o.Size), 10)
	}

	// Non-PCF attributes (stripped from the canon tree, so normally
	// absent; emitted faithfully when present).
	if o.Namespace != nil {
		dst = key(dst, "namespace")
		dst = appendCanonString(dst, *o.Namespace)
	}
	if len(o.Aliases) > 0 {
		dst = key(dst, "aliases")
		dst = appendCanonStringArray(dst, o.Aliases)
	}
	if len(o.Default) > 0 {
		dst = key(dst, "default")
		dst = appendCompactJSON(dst, o.Default)
	}
	if o.Logical != "" {
		dst = key(dst, "logicalType")
		dst = appendCanonString(dst, o.Logical)
	}
	if o.Precision != nil {
		dst = key(dst, "precision")
		dst = strconv.AppendInt(dst, int64(*o.Precision), 10)
	}
	if o.Scale != nil {
		dst = key(dst, "scale")
		dst = strconv.AppendInt(dst, int64(*o.Scale), 10)
	}

	return append(dst, '}')
}

// appendCanonField writes an afield: name and type always, then the field
// attributes (aliases, default, order, logicalType, precision, scale) when
// present. The Canonical() path feeds a stripped tree, so only name/type
// appear there; the attribute branches exist for the general-purpose writer
// mode (symmetric with appendCanonObject).
func appendCanonField(dst []byte, f *afield) []byte {
	dst = append(dst, '{')
	// "name" leads, so every later key is comma-preceded — no first-key
	// bookkeeping, unlike appendCanonObject's key closure.
	key := func(dst []byte, k string) []byte {
		dst = append(dst, ',')
		dst = appendCanonString(dst, k)
		return append(dst, ':')
	}
	dst = appendCanonString(dst, "name")
	dst = append(dst, ':')
	dst = appendCanonString(dst, f.Name)
	dst = key(dst, "type")
	if f.Type != nil {
		dst = appendCanonSchema(dst, f.Type)
	} else {
		dst = append(dst, '"', '"')
	}
	if len(f.Aliases) > 0 {
		dst = appendCanonStringArray(key(dst, "aliases"), f.Aliases)
	}
	if len(f.Default) > 0 {
		dst = appendCompactJSON(key(dst, "default"), f.Default)
	}
	if f.Order != "" {
		dst = appendCanonString(key(dst, "order"), f.Order)
	}
	if f.Logical != "" {
		dst = appendCanonString(key(dst, "logicalType"), f.Logical)
	}
	if f.Precision != nil {
		dst = strconv.AppendInt(key(dst, "precision"), int64(*f.Precision), 10)
	}
	if f.Scale != nil {
		dst = strconv.AppendInt(key(dst, "scale"), int64(*f.Scale), 10)
	}
	return append(dst, '}')
}

func appendCanonStringArray(dst []byte, ss []string) []byte {
	dst = append(dst, '[')
	for i, s := range ss {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendCanonString(dst, s)
	}
	return append(dst, ']')
}

const hexDigits = "0123456789abcdef"

// appendCanonString writes s as a JSON string with only the mandatory
// escapes (PCF [STRINGS] = raw UTF-8). Matches encoding/json with
// SetEscapeHTML(false) AND without U+2028/U+2029 escaping: <, >, &, the
// line/paragraph separators, and all other valid code points are emitted
// verbatim; only ", \, and control characters (< 0x20) are escaped, using
// the same forms encoding/json uses. Invalid UTF-8 is replaced with
// U+FFFD, matching encoding/json so the canonical bytes (and fingerprint)
// stay identical for every well-formed schema.
func appendCanonString(dst []byte, s string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if b >= 0x20 && b != '"' && b != '\\' {
				i++
				continue
			}
			if start < i {
				dst = append(dst, s[start:i]...)
			}
			switch b {
			case '"':
				dst = append(dst, '\\', '"')
			case '\\':
				dst = append(dst, '\\', '\\')
			case '\b':
				dst = append(dst, '\\', 'b')
			case '\f':
				dst = append(dst, '\\', 'f')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				dst = append(dst, '\\', 'u', '0', '0', hexDigits[b>>4], hexDigits[b&0xf])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			if start < i {
				dst = append(dst, s[start:i]...)
			}
			dst = append(dst, 0xef, 0xbf, 0xbd) // U+FFFD, as encoding/json emits
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		dst = append(dst, s[start:]...)
	}
	return append(dst, '"')
}
