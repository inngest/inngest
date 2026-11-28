package avro

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)

// decodeLogicalInt applies logical type conversion for int-backed logical types
// when decoding to *any targets.
func decodeLogicalInt(val int32, node *schemaNode) any {
	switch node.logical {
	case "date":
		return dateToTime(val)
	case "time-millis":
		return timeMillisToDuration(val)
	}
	return val
}

// timestampToTimeConv returns the wire-int64 → time.Time converter for the
// six long-typed timestamp logical names (local-* shares its non-local
// converter), or (nil, false) for any other logical. Single source for the
// name→converter mapping that decodeLogicalLong and decodeLong's three target
// arms (any / time.Time / string) all share, so adding or changing a timestamp
// logical can't silently update only some of them.
func timestampToTimeConv(logical string) (func(int64) time.Time, bool) {
	switch logical {
	case "timestamp-millis", "local-timestamp-millis":
		return timestampMillisToTime, true
	case "timestamp-micros", "local-timestamp-micros":
		return timestampMicrosToTime, true
	case "timestamp-nanos", "local-timestamp-nanos":
		return timestampNanosToTime, true
	}
	return nil, false
}

// decodeLogicalLong applies logical type conversion for long-backed logical types
// when decoding to *any targets. Returns an error only for time-micros when
// val * time.Microsecond would wrap; the timestamp conversions are total.
func decodeLogicalLong(val int64, node *schemaNode) (any, error) {
	if conv, ok := timestampToTimeConv(node.logical); ok {
		return conv(val), nil
	}
	if node.logical == "time-micros" {
		return timeMicrosToDuration(val)
	}
	return val, nil
}

// decodeLogicalBytes applies logical type conversion for bytes-backed
// logical types when decoding to *any targets. Errors on malformed
// payloads.
func decodeLogicalBytes(b []byte, node *schemaNode) (any, error) {
	if node.logical == "decimal" {
		// Bound the unscaled length before bytesToRat materializes/converts —
		// the into-any path bypasses setDecimalValue (see maxDecimalUnscaledBytes).
		if err := checkDecimalUnscaledLen(b); err != nil {
			return nil, err
		}
		return bytesToRat(b, node.scale), nil
	}
	if node.logical == "big-decimal" {
		r, _, err := parseBigDecimalPayload(b)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	return b, nil
}

// jsonDecodeAppliesLogical reports whether decodeKind would transform the raw
// Avro-native value into an enriched Go type for this node's logical type —
// the JSON parallel of the binary logical deserializer.
//
// It DERIVES the answer by probing the decodeLogical{Int,Long,Bytes,Fixed}
// functions decodeKind itself uses and checking whether the result is still the
// raw Avro-native type. No second list to keep in sync: a logical added to or
// removed from a decodeLogical* switch shows up here automatically, so the
// suppression gate cannot drift from what decode does.
//
// Consulted only by applyCustomTypes at PARSE time. The placeholder boxing
// costs a handful of allocs per custom-typed logical node, once per schema, and
// never touches the hot path.
// TestMatrix_JSONDecodeAppliesLogicalMatchesDecode pins the result for
// every logical against the human-known expected set.
func jsonDecodeAppliesLogical(node *schemaNode) bool {
	if node.logical == "" {
		return false
	}
	switch node.kind {
	case "int":
		_, raw := decodeLogicalInt(0, node).(int32)
		return !raw
	case "long":
		v, err := decodeLogicalLong(0, node)
		if err != nil { // only a transforming arm (time-micros) can error
			return true
		}
		_, raw := v.(int64)
		return !raw
	case "bytes":
		v, err := decodeLogicalBytes(nil, node)
		if err != nil { // only a transforming arm (big-decimal) can error on empty input
			return true
		}
		_, raw := v.([]byte)
		return !raw
	case "fixed":
		// decodeLogicalFixed's uuid / duration arms convert only at len 16 / 12;
		// decimal converts at any len; an unknown logical never converts. So the
		// probe's answer depends only on whether node.size is exactly 12 or 16 —
		// no fixed logical inspects a length above 16. Cap the probe buffer just
		// above that bound: node.size is schema-controlled and only validated as
		// non-negative (a fixed size has no upper bound, matching fastavro), so a
		// hostile {"type":"fixed","size":<huge>,"logicalType":...} with a matching
		// CustomType would otherwise drive a multi-GB / panic-inducing make() here
		// at parse time. A capped length >16 is neither 12 nor 16, so it yields
		// the same answer the true oversized length would.
		probeLen := node.size
		if probeLen > maxFixedLogicalLen {
			probeLen = maxFixedLogicalLen + 1
		}
		_, raw := decodeLogicalFixed(make([]byte, probeLen), node).([]byte)
		return !raw
	case "string":
		// uuid-on-string has a TYPED-target transform — decodeString parses the
		// hex-dash string into a [16]byte / UUID-typed target — that the *any
		// probe above can't see (into *any / string it IS identity). Report it
		// as transforming so a no-Decode CustomType on uuid-string installs the
		// suppression wrapper, and the raw decode (decodeString with raw=true)
		// then errors on a [16]byte target exactly as the binary deserString
		// does. Other string logicals have no typed-target transform.
		return node.logical == "uuid"
	}
	return false
}

// maxFixedLogicalLen is the largest fixed byte-length that any decodeLogicalFixed
// arm inspects (uuid at 16; duration at 12; decimal converts at any length). It
// bounds the jsonDecodeAppliesLogical parse-time probe buffer so a hostile fixed
// size can't drive a huge allocation. If a future fixed-backed logical converts
// at a longer length, raise this to match its len check.
const maxFixedLogicalLen = 16

// decodeLogicalFixed applies logical type conversion for fixed-backed logical types
// when decoding to *any targets.
func decodeLogicalFixed(b []byte, node *schemaNode) any {
	switch node.logical {
	case "decimal":
		return bytesToRat(b, node.scale)
	case "duration":
		if len(b) == 12 {
			return DurationFromBytes(b)
		}
	case "uuid":
		if len(b) == 16 {
			return [16]byte(b)
		}
	}
	return b
}

// assignAny sets a native Go value on a reflect.Value target.
// For nil val + nilable v (interface, pointer, map, slice), zeros v.
// For non-nil val, returns a SemanticError if val's type isn't assignable
// to v's type — guarding against decode targets like *interface{Foo()}
// that the produced value doesn't satisfy. Skips the AssignableTo lookup
// for the empty-interface (any) target — the hot decode-into-*any path.
func assignAny(v reflect.Value, val any, avroType string) error {
	if val == nil {
		setZero(v)
		return nil
	}
	rv := reflect.ValueOf(val)
	if v.Kind() == reflect.Interface && v.Type().NumMethod() != 0 && !rv.Type().AssignableTo(v.Type()) {
		return &SemanticError{GoType: v.Type(), AvroType: avroType}
	}
	v.Set(rv)
	return nil
}

// consumeSlabString consumes a JSON string and copies it into the slab
// allocator, batching small string allocations into one backing buffer.
func (ctx *jsonDecoder) consumeSlabString() (string, error) {
	start, end, hasEscapes, err := ctx.scanner.consumeStringRaw()
	if err != nil {
		return "", err
	}
	if hasEscapes {
		return resolveJSONEscapes(ctx.scanner.data[start:end])
	}
	return ctx.slab.string(ctx.scanner.data[start:], end-start), nil
}

// jsonDecoder is the state for schema-guided JSON decoding.
type jsonDecoder struct {
	scanner *jsonScanner
	// slab carries the decode options as well as the string arena. A record
	// field filled from its schema default routes through the BINARY deser
	// fn, which reads taggedUnions / tagLogicalTypes off the slab, so the
	// slab has to hold them whatever this struct does. Holding them twice
	// would let a present union field and a default-filled one answer the
	// same option differently — the exact inconsistency the slab assignment
	// exists to prevent.
	slab *slab
	// suppressLogical, when set, makes the next decodeKind hand the RAW
	// Avro-native value (int32/int64/[]byte) to its leaf decoder instead
	// of the logical-transformed Go value (time.Time/time.Duration/
	// *big.Rat). Set by wrapDecodeJSONWithCustomDecoders so a custom
	// decoder chain receives the raw value, mirroring the binary path's
	// logical-deser suppression. decodeKind captures and clears it on
	// entry so it scopes to exactly one node.
	suppressLogical bool
}

// decodeValue is the core recursive decoder. It reads the next JSON
// value from the scanner, guided by the schema node, and assigns to v.
//
// For interface (any) targets, it produces JSON-native or enriched Go
// values. For typed targets (struct, int, string, etc.), it assigns
// directly.
//
// When the node carries a custom-decoder wrapper (decodeJSON), the
// wrapper handles the dispatch — it captures the decoder chain at
// schema build, calls decodeKind for the inner value, then applies
// each custom decoder in turn. No runtime map lookup, no recursion
// guard. Concurrency safety is structural: the schema graph is
// read-only at decode time and the jsonDecoder is per-call.
func (ctx *jsonDecoder) decodeValue(v reflect.Value, node *schemaNode) error {
	if ctx.slab.depth >= maxDepth {
		return errTooDeep
	}
	ctx.slab.depth++
	defer func() { ctx.slab.depth-- }()
	if node.decodeJSON != nil {
		return node.decodeJSON(ctx, v, node)
	}
	return ctx.decodeKind(v, node)
}

// decodeKind is decodeValue minus the depth guard and the
// custom-decoder dispatch — just the kind switch. Called directly by
// decodeValue for nodes without custom decoders, and by the
// custom-decoder closure to produce the inner *any value before
// applying the decoder chain.
func (ctx *jsonDecoder) decodeKind(v reflect.Value, node *schemaNode) error {
	// Capture and clear suppressLogical so it applies to exactly this
	// node's leaf decode and never leaks into children decoded during
	// recursion (e.g. a custom type whose AvroType is "record" — its
	// fields must still get their own logical conversions).
	raw := ctx.suppressLogical
	ctx.suppressLogical = false

	// Unions handle pointer/nil targets specially (before indirectAlloc),
	// so dispatch early.
	if node.kind == "union" {
		return ctx.decodeUnion(v, node)
	}

	v = indirectAlloc(v)
	toAny := v.Kind() == reflect.Interface

	switch node.kind {
	case "null":
		return ctx.decodeNull(v, toAny)
	case "boolean":
		return ctx.decodeBool(v, toAny)
	case "int":
		return ctx.decodeInt(v, node, toAny, raw)
	case "long":
		return ctx.decodeLong(v, node, toAny, raw)
	case "float":
		return ctx.decodeFloat(v)
	case "double":
		return ctx.decodeDouble(v)
	case "string":
		return ctx.decodeString(v, node, toAny, raw)
	case "enum":
		return ctx.decodeEnum(v, node)
	case "bytes":
		return ctx.decodeBytes(v, node, toAny, raw)
	case "fixed":
		return ctx.decodeFixed(v, node, toAny, raw)
	case "array":
		return ctx.decodeArray(v, node, toAny)
	case "map":
		return ctx.decodeMap(v, node, toAny)
	case "record":
		return ctx.decodeRecord(v, node, toAny)
	default:
		return fmt.Errorf("avro json: unsupported schema kind %q", node.kind)
	}
}

// wrapDecodeJSONWithCustomDecoders builds a per-node JSON decode
// closure that captures the custom decoder chain — the JSON parallel
// of wrapDeserWithCustomDecoders (custom_type.go). The inner value is
// produced via decodeKind so we don't re-enter the wrapper for the
// same node.
func wrapDecodeJSONWithCustomDecoders(decoders []func(any, *SchemaNode) (any, error), sn *SchemaNode, suppressLogical bool) jsonDecodeFn {
	return func(ctx *jsonDecoder, v reflect.Value, node *schemaNode) error {
		// A no-match ancestor set this: decode the subtree raw through the kind
		// switch. This node's own suppression still applies to its leaf decode.
		if ctx.slab.bypassCustom {
			ctx.suppressLogical = suppressLogical
			return ctx.decodeKind(v, node)
		}
		// suppressLogical decodes the RAW Avro-native value (int32/int64/[]byte)
		// for this node exactly when the binary path also suppresses the logical
		// deserializer (hasMatchingCustomType). A wildcard CustomType is excluded,
		// so the logical transform is kept and binary↔JSON parity holds. decodeKind
		// captures and clears the flag, so it applies only to this node's leaf.
		if len(decoders) == 0 {
			// Pure suppression (no Decode callback): decode straight into the
			// target through the raw arms — DRY parity with the binary raw deser
			// (a box-into-any could not land a []byte into a [N]byte array the way
			// decodeKind's deserFixed reflect.Copy does).
			ctx.suppressLogical = suppressLogical
			return ctx.decodeKind(v, node)
		}
		// Fresh interface target: decodeKind's interface output IS the canonical
		// value a no-custom decode yields, so decode straight into v and read it
		// back for the chain — keeping a parent probe (whose elements are all
		// fresh `any`) to a single pass, mirroring the binary wrapper. A NON-nil
		// interface is excluded (it would reuse the held value in place) and takes
		// the probe + re-decode path below.
		if v.Kind() == reflect.Interface && v.IsNil() {
			ctx.suppressLogical = suppressLogical
			if err := ctx.decodeKind(v, node); err != nil {
				return err
			}
			chainVal := v.Interface()
			for _, dec := range decoders {
				out, err := dec(chainVal, sn)
				if err != nil {
					if errors.Is(err, ErrSkipCustomType) {
						continue
					}
					return err
				}
				ctx.slab.customMatches++
				return setCustomResult(v, out, node.kind)
			}
			return nil // all-skip: v already holds the no-custom value
		}
		// Typed target: probe into a throwaway any for the chain; on the all-skip
		// fall-through rewind the scanner and RE-DECODE faithfully into v — the
		// same decode a no-custom schema performs (a reused map keeps its keys, a
		// logical node lands in a base typed target, an overlapping union recovers
		// its exact wire branch), none of which placing the any value reproduces.
		var tmp any
		tmpV := reflect.ValueOf(&tmp).Elem()
		savedPos := ctx.scanner.pos
		savedMatches := ctx.slab.customMatches
		ctx.suppressLogical = suppressLogical
		if err := ctx.decodeKind(tmpV, node); err != nil {
			return err
		}
		for _, dec := range decoders {
			out, err := dec(tmp, sn)
			if err != nil {
				if errors.Is(err, ErrSkipCustomType) {
					continue
				}
				return err
			}
			// setCustomResult (not assignAny): a result not assignable to a
			// concrete target returns a SemanticError instead of panicking, and
			// the un-indirected v lets a *T result land in a *T target — matching
			// the binary path (wrapDeserWithCustomDecoders).
			ctx.slab.customMatches++
			return setCustomResult(v, out, node.kind)
		}
		// Every decoder skipped: rewind and re-decode into the typed target. No
		// nested custom matched ⇒ bypass for a single pass; otherwise re-decode
		// with customs active to reproduce the nested match (bounded by maxDepth).
		ctx.scanner.pos = savedPos
		ctx.suppressLogical = suppressLogical
		if ctx.slab.customMatches == savedMatches {
			ctx.slab.bypassCustom = true
			err := ctx.decodeKind(v, node)
			ctx.slab.bypassCustom = false
			return err
		}
		return ctx.decodeKind(v, node)
	}
}

func (ctx *jsonDecoder) decodeNull(v reflect.Value, _ bool) error {
	if err := ctx.scanner.consumeNull(); err != nil {
		return err
	}
	setZero(v)
	return nil
}

func (ctx *jsonDecoder) decodeBool(v reflect.Value, toAny bool) error {
	b, err := ctx.scanner.consumeBool()
	if err != nil {
		return err
	}
	if toAny {
		return setIface(v, reflect.ValueOf(b), "boolean")
	}
	if v.Kind() == reflect.Bool {
		v.SetBool(b)
		return nil
	}
	return semErr(v, "boolean")
}

func (ctx *jsonDecoder) decodeInt(v reflect.Value, node *schemaNode, toAny, raw bool) error {
	nb, err := ctx.scanner.consumeNumberBytes()
	if err != nil {
		return err
	}
	val, err := parseJSONInt32(nb)
	if err != nil {
		return err
	}
	if toAny {
		if raw {
			return setIface(v, reflect.ValueOf(val), "int")
		}
		return setIface(v, reflect.ValueOf(decodeLogicalInt(val, node)), "int")
	}
	if raw {
		// Suppressed by a matching no-Decode CustomType: assign the raw int32,
		// skipping the date/time-millis typed-target arms below — mirrors
		// binary's raw deserInt, which builds no logical deser under
		// suppression (so a time.Time / time.Duration / string target is
		// rejected or filled raw exactly as on the binary path). Without this,
		// a suppressed date decoded into time.Time succeeded on JSON (enriched)
		// while binary rejected it, and time-millis into time.Duration silently
		// produced a different value (raw ns vs the logical conversion).
		return setIntValue(v, val)
	}
	// All DecodeJSON entry points produce addressable values
	// (Schema.DecodeJSON requires a pointer; recursive paths use
	// reflect.New().Elem() or addressable struct fields).
	if v.Type() == timeType {
		switch node.logical {
		case "date":
			*(*time.Time)(v.Addr().UnsafePointer()) = dateToTime(val)
			return nil
		case "time-millis":
			// Mirrors serTimeMillis's timeType arm.
			*(*time.Time)(v.Addr().UnsafePointer()) = timeOfDayToTime(timeMillisToDuration(val))
			return nil
		}
	}
	if v.Type() == durationType && node.logical == "time-millis" {
		*(*time.Duration)(v.Addr().UnsafePointer()) = timeMillisToDuration(val)
		return nil
	}
	// String target for date: mirrors json_codec.go's "int" date arm
	// which accepts a date-string on encode (tryParseDateString).
	// Parity with deserDate on the binary side. json.Number is excluded
	// by formatToStringKindTarget — falls through to setIntValue's
	// json.Number arm for the raw integer wire value.
	if node.logical == "date" {
		if wrote, err := formatToStringKindTarget(v, dateToTime(val).Format(time.DateOnly), "int"); wrote {
			return err
		}
	}
	return setIntValue(v, val)
}

func (ctx *jsonDecoder) decodeLong(v reflect.Value, node *schemaNode, toAny, raw bool) error {
	nb, err := ctx.scanner.consumeNumberBytes()
	if err != nil {
		return err
	}
	val, err := parseJSONInt64(nb)
	if err != nil {
		return err
	}
	if toAny {
		if raw {
			return setIface(v, reflect.ValueOf(val), "long")
		}
		logical, err := decodeLogicalLong(val, node)
		if err != nil {
			return err
		}
		return setIface(v, reflect.ValueOf(logical), "long")
	}
	if raw {
		// Suppressed by a matching no-Decode CustomType: assign the raw int64,
		// skipping the timestamp/time-micros typed-target arms below — mirrors
		// binary's raw deserLong (see decodeInt for the full rationale and the
		// silent time.Duration value-divergence this prevents).
		return setLongValue(v, val)
	}
	// All DecodeJSON entry points produce addressable values (see decodeInt).
	if v.Type() == timeType {
		p := (*time.Time)(v.Addr().UnsafePointer())
		if conv, ok := timestampToTimeConv(node.logical); ok {
			*p = conv(val)
			return nil
		}
		if node.logical == "time-micros" {
			// Mirrors serTimeMicros's timeType arm.
			d, err := timeMicrosToDuration(val)
			if err != nil {
				return err
			}
			*p = timeOfDayToTime(d)
			return nil
		}
	}
	if v.Type() == durationType && node.logical == "time-micros" {
		d, err := timeMicrosToDuration(val)
		if err != nil {
			return err
		}
		*(*time.Duration)(v.Addr().UnsafePointer()) = d
		return nil
	}
	// String target for the six long-typed time logicals: mirrors the
	// JSON encoder's "long" arm (json_codec.go), which accepts an RFC
	// 3339 string via extractTime. Parity with deserTimeAsLong on the
	// binary side. json.Number is excluded by formatToStringKindTarget;
	// falls through to setLongValue's json.Number arm for the raw integer
	// wire value (same routing as the time-micros / time-millis logicals).
	conv, ok := timestampToTimeConv(node.logical)
	if !ok {
		return setLongValue(v, val)
	}
	if wrote, err := formatToStringKindTarget(v, conv(val).Format(time.RFC3339Nano), "long"); wrote {
		return err
	}
	return setLongValue(v, val)
}

// isJSONNullStart reports whether the next token is the JSON "null"
// literal. The peeked first byte 'n' is ambiguous (could also start a
// bare lowercase "nan"); we disambiguate by checking the second byte
// — null is the only token starting with "nu".
func isJSONNullStart(s *jsonScanner, p byte) bool {
	return p == 'n' && s.peekAt(1) == 'u'
}

// isBareSpecialFloatStart reports whether the next token could begin
// a bare NaN / Infinity / -Infinity / Inf / INF token in the canonical
// Java/fastavro casings (uppercase first letter; parseSpecialFloat
// applies the exact-match gate after consumption). Lowercase first
// letters ('n', 'i') are rejected — Java's JsonDecoder, fastavro's
// Python json, and goavro all reject lowercase, and the lowercase 'n'
// in particular would collide with the JSON null literal in the union
// dispatcher.
func isBareSpecialFloatStart(s *jsonScanner, p byte) bool {
	switch p {
	case 'N', 'I':
		return true
	case '-':
		return s.peekAt(1) == 'I'
	}
	return false
}

// decodeJSONFloat decodes the next JSON token into a float64,
// dispatching across the four producer conventions twmb accepts:
//
//   - quoted-string "NaN"/"Infinity"/"-Infinity"/"INF"/"-INF"/"Inf"/"-Inf"
//     (Java JsonEncoder form, twmb's default emit form). parseSpecialFloat
//     gates exact-match (Java parity — see its docstring for the per-impl
//     accept sets; fastavro reads only the bare-token forms).
//   - bare null → NaN (goavro convention). isJSONNullStart disambiguates
//     from bare special-float tokens whose first byte is unambiguously
//     uppercase post-tightening.
//   - bare NaN/Infinity/-Infinity/INF/-INF/Inf/-Inf (fastavro / Python json.dumps
//     with allow_nan=True). Routed through parseSpecialFloat for consistency
//     with the quoted-string arm — same exact-match acceptance set.
//   - numeric literal, with ±Inf accept on overflow (goavro's 1e999 / -1e999
//     convention, and any over-range literal that strconv.ParseFloat
//     produces as Inf with ErrRange).
//
// Shared by decodeFloat (bitSize=32) and decodeDouble (bitSize=64);
// the per-target narrowing is applied via setFloatValue downstream.
// typ is "float" or "double" for the syntax-error message.
func (ctx *jsonDecoder) decodeJSONFloat(bitSize int, typ string) (float64, error) {
	p := ctx.scanner.peek()
	switch {
	case p == '"':
		// Zero-copy: parseSpecialFloat reads the token (NaN / Infinity /
		// ...) and returns a float without retaining the string, so the
		// transient scanner-backed string is safe here.
		s, err := ctx.scanner.consumeStringZeroCopy()
		if err != nil {
			return 0, err
		}
		return parseSpecialFloat(s)
	case isJSONNullStart(ctx.scanner, p):
		if err := ctx.scanner.consumeNull(); err != nil {
			return 0, err
		}
		return math.NaN(), nil
	case isBareSpecialFloatStart(ctx.scanner, p):
		tok, err := ctx.scanner.consumeBareSpecialFloat()
		if err != nil {
			return 0, err
		}
		return parseSpecialFloat(tok)
	default:
		nb, err := ctx.scanner.consumeNumberBytes()
		if err != nil {
			return 0, err
		}
		// Same shared gate + parse the int/long arms and the encode side
		// use: parseJSONNumberAsFloat applies the isJSONNumber grammar gate
		// (rejecting non-JSON forms like the trailing-dot "5." / "5.e3"
		// that strconv.ParseFloat would otherwise accept), caps the length
		// for DoS, and accepts ±Inf from overflow (1e999). bitSize is
		// threaded so a "float" schema parses at float32 precision (single
		// rounding). nb aliases the scanner buffer; the helper is read-only
		// and its error path copies via truncForError.
		f, err := parseJSONNumberAsFloat(unsafe.String(unsafe.SliceData(nb), len(nb)), bitSize)
		if err != nil {
			return 0, fmt.Errorf("avro json: %s: %w", typ, err)
		}
		return f, nil
	}
}

func (ctx *jsonDecoder) decodeFloat(v reflect.Value) error {
	f, err := ctx.decodeJSONFloat(32, "float")
	if err != nil {
		return err
	}
	// setFloatValue's interface arm subsumes what the toAny branch
	// would otherwise do — single point of truth for the float-target
	// matrix shared with deserFloat. The float32 narrowing happens
	// inside setFloatValue for typed float32 targets.
	return setFloatValue(v, f, "float", 32)
}

func (ctx *jsonDecoder) decodeDouble(v reflect.Value) error {
	f, err := ctx.decodeJSONFloat(64, "double")
	if err != nil {
		return err
	}
	return setFloatValue(v, f, "double", 64)
}

func (ctx *jsonDecoder) decodeString(v reflect.Value, node *schemaNode, toAny, raw bool) error {
	s, err := ctx.consumeSlabString()
	if err != nil {
		return err
	}
	// UUID logical type: [16]byte target parses the hex-dash string into raw
	// bytes, matching deserUUID on the binary side. Skipped when raw (a custom
	// type suppresses the logical with no Decode callback): the binary path then
	// uses deserString, which has no [16]byte arm and errors — so producing the
	// raw string here keeps DecodeJSON in parity with Decode.
	if node.logical == "uuid" && !toAny && !raw && isUUIDType(v.Type()) {
		u, err := parseUUID(s)
		if err != nil {
			return err
		}
		copyBytesToArray(v, u[:])
		return nil
	}
	if toAny {
		return setIface(v, reflect.ValueOf(s), "string")
	}
	// TextUnmarshaler before the reflect.String fast path: a string-kind
	// type implementing TextUnmarshaler uses its text parsing, mirroring
	// the encoder (avroStringValue tries text before reflect.String) and
	// setStringValue on the binary side. Also covers named []byte subtypes
	// like net.IP. The implements-check gates the []byte(s) allocation so
	// the common plain-string path stays alloc-free.
	if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
		_, err := tryTextUnmarshal(v, []byte(s))
		return err
	}
	if v.Kind() == reflect.String {
		return setStringTarget(v, s, "string")
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		v.SetBytes([]byte(s))
		return nil
	}
	return semErr(v, "string")
}

func (ctx *jsonDecoder) decodeEnum(v reflect.Value, node *schemaNode) error {
	s, err := ctx.consumeSlabString()
	if err != nil {
		return err
	}
	// Through the node's shared symbol table, not a scan of the symbol
	// slice: an enum's symbol count is set by the schema text and this runs
	// once per value, so a scan multiplies two caller-chosen numbers. The
	// binary encoder resolves the same question through the same table
	// (serEnum.indexOfSymbol).
	idx, ok := node.symbolIndex(s)
	if !ok {
		return fmt.Errorf("avro json: unknown enum symbol %q", truncForError(s))
	}
	// Mirrors deserEnum's target dispatch: Interface→symbol; String→symbol;
	// Int/Uint→ordinal (Java's JsonDecoder.readEnum and fastavro's read_enum
	// both return the index).
	return setEnumTarget(v, idx, s)
}

func (ctx *jsonDecoder) decodeBytes(v reflect.Value, node *schemaNode, toAny, raw bool) error {
	// Decimal / big-decimal logical types: accept JSON numbers (e.g. 0.33
	// or 1.5) in addition to Avro JSON byte strings, for round-trip with
	// EncodeJSON output and convenience for hand-edited JSON. Big-decimal
	// is bytes-only per spec (rejected in decodeFixed). Skipped in raw
	// (custom-decoder) mode: the callback receives the raw Avro-native
	// []byte, and a bare JSON number has no raw-bytes form — matching the
	// binary path, which has no bare-number form at all.
	if !raw && hasDecimalBareNumberArm(node) {
		if handled, err := ctx.decodeBareDecimal(v, node, toAny); handled {
			return err
		}
	}
	start, end, _, err := ctx.scanner.consumeStringRaw()
	if err != nil {
		return err
	}
	b, err := scanAvroJSONBytes(ctx.scanner.data[start:end])
	if err != nil {
		return err
	}
	if toAny {
		if raw {
			return setIface(v, reflect.ValueOf(b), "bytes")
		}
		val, err := decodeLogicalBytes(b, node)
		if err != nil {
			return err
		}
		return setIface(v, reflect.ValueOf(val), "bytes")
	}
	return assignBytes(v, b, node, raw)
}

func (ctx *jsonDecoder) decodeFixed(v reflect.Value, node *schemaNode, toAny, raw bool) error {
	// Decimal logical type: accept JSON numbers, same as decodeBytes.
	// Big-decimal is bytes-only per spec; hasDecimalBareNumberArm enforces
	// that (returns false for big-decimal on a fixed node, even one a
	// CustomType resurrected), so the bare-number arm never fires here for it.
	// Skipped in raw (custom-decoder) mode — see decodeBytes.
	if !raw && hasDecimalBareNumberArm(node) {
		if handled, err := ctx.decodeBareDecimal(v, node, toAny); handled {
			return err
		}
	}
	start, end, _, err := ctx.scanner.consumeStringRaw()
	if err != nil {
		return err
	}
	b, err := scanAvroJSONBytes(ctx.scanner.data[start:end])
	if err != nil {
		return err
	}
	// Per spec, the JSON string for a fixed value must have exactly
	// node.size code points (= bytes after code-point semantics).
	// Java's JsonDecoder.readFixed enforces this; the JSON encoder
	// produces exactly that length. Reject mismatches symmetrically.
	if len(b) != node.size {
		return fmt.Errorf("avro json: fixed value has %d bytes, schema declares %d", len(b), node.size)
	}
	if !raw && node.logical == "decimal" {
		// The fixed-decimal into-any path goes through decodeLogicalFixed (no
		// error return), bypassing setDecimalValue's bound — cap the unscaled
		// length here so a huge fixed-decimal can't drive the base conversion
		// (see maxDecimalUnscaledBytes).
		if err := checkDecimalUnscaledLen(b); err != nil {
			return err
		}
	}
	if toAny {
		if raw {
			return setIface(v, reflect.ValueOf(b), "fixed")
		}
		return setIface(v, reflect.ValueOf(decodeLogicalFixed(b, node)), "fixed")
	}
	return assignBytes(v, b, node, raw)
}

// assignBytes assigns decoded bytes to a typed target, handling decimal,
// duration, and uuid logical types. Logical-arm fall-through (the arm
// fires but doesn't return) lands on the generic byte/string/array
// targets below.
//
// raw=true means a matching no-Decode CustomType suppressed the logical
// codec: skip every logical arm and assign the raw bytes, mirroring the
// binary path's raw deserBytes/deserFixed (which build no logical deser
// when suppressLogical fires). Without this, a suppressed bytes/fixed node
// with a decimal/uuid/duration logicalType still transformed on the JSON
// side (e.g. "decimal" → *big.Rat) while binary handed back raw bytes.
func assignBytes(v reflect.Value, b []byte, node *schemaNode, raw bool) error {
	if raw {
		return setBytesValue(v, b, node.kind)
	}
	// Each arm fires only on the kind its logical is spec-valid on, so the
	// typed-target transform set matches the *any path and
	// jsonDecodeAppliesLogical's probe for the same (kind, logical). decimal is
	// valid on bytes AND fixed, big-decimal on bytes only, duration and uuid on
	// fixed only (uuid-on-string goes through decodeString).
	//
	// A logical on the WRONG kind arises only when a CustomType resurrects a
	// soft-dropped placement, and that match also SUPPRESSES the codec, so the
	// contract is the raw value — which the kind-gated fall-through produces.
	// Without the gate, JSON transformed (uuid→hex-dash, duration→avro.Duration)
	// while binary returned raw bytes.
	switch node.logical {
	case "decimal":
		// Share setDecimalValue with the binary path so JSON accepts
		// the same target types (*big.Rat, big.Rat, json.Number,
		// *float32, *float64, *string) with the same overflow guards.
		if ok, err := setDecimalValue(v, b, node.scale); ok {
			return err
		}
	case "big-decimal":
		// b is the inner big-decimal payload (length-prefixed unscaled
		// + zigzag scale); the outer codepoint-string decode has
		// already stripped the JSON quoting. applyBigDecimalPayload
		// encapsulates the binary-side opaque-bytes fall-through; when
		// it returns (false, _) we drop into setBytesValue below.
		if node.kind == "bytes" {
			if done, err := applyBigDecimalPayload(v, b); done {
				return err
			}
		}
	case "duration":
		// len(b)==12 mirrors decodeLogicalFixed's duration arm (the *any path)
		// and the uuid arm below (len==16): DurationFromBytes reads a fixed 12
		// bytes, so it is only correct for a size-12 fixed. A CustomType-
		// resurrected wrong-size duration (decodeFixed enforces len(b)==node.size)
		// falls through to the raw setBytesValue, matching the suppressed binary
		// deserFixed{size} and the plain (soft-dropped) fixed.
		if node.kind == "fixed" && len(b) == 12 && v.Type() == avroDurationType {
			v.Set(reflect.ValueOf(DurationFromBytes(b)))
			return nil
		}
	case "uuid":
		if node.kind == "fixed" && len(b) == 16 {
			var u [16]byte
			copy(u[:], b)
			// [16]byte trusts the raw bytes (isUUIDType-first, matching
			// deserFixedUUIDReflect): no UnmarshalText round trip. Without
			// this, a [16]byte type that also implements TextUnmarshaler
			// (e.g. google/uuid.UUID) diverged from the binary path.
			if isUUIDType(v.Type()) {
				copyBytesToArray(v, u[:])
				return nil
			}
			// TextUnmarshaler before the reflect.String arm (parity with the
			// binary side): the canonical hex-dash form is fed to UnmarshalText.
			if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
				_, err := tryTextUnmarshal(v, []byte(uuidToString(u)))
				return err
			}
			// String target: format as RFC 4122 hex-dash.
			if v.Kind() == reflect.String {
				return setStringTarget(v, uuidToString(u), "fixed")
			}
		}
	}
	// Generic byte-target fall-through shared with the binary path.
	// setBytesValue handles slice/array/string (and would handle the
	// interface arm too; the toAny branch upstream already covered
	// interface targets for this call site).
	return setBytesValue(v, b, node.kind)
}

// hasDecimalBareNumberArm reports whether node is a logical-typed bytes/
// fixed schema that accepts the bare-number JSON form on decode (the
// lenient convenience that lets a hand-edited producer write 0.33 instead
// of the spec codepoint-string form). decimal qualifies on bytes AND fixed;
// big-decimal is bytes-only per spec, so it qualifies ONLY on bytes — on a
// fixed node the big-decimal logical is non-standard (resurrected solely by
// a CustomType, which suppresses the codec so the contract is the raw
// value), and transforming a bare number there would diverge from the
// suppressed binary path. Kind-gating big-decimal here keeps the predicate
// in lockstep with assignBytes's kind gate and makes the call-site comments
// ("big-decimal ... never reaches here" / "not eligible on a fixed branch")
// true by construction. The union-dispatch sibling jsonTokenMatchesBranch
// uses the same rule.
func hasDecimalBareNumberArm(node *schemaNode) bool {
	switch node.logical {
	case "decimal":
		return true
	case "big-decimal":
		return node.kind == "bytes"
	}
	return false
}

// decodeBareDecimal handles the bare-number JSON arm for decimal-like
// logical types (decimal, big-decimal). Returns handled=true when the
// next token was a bare number (and the value was assigned or an error
// produced); handled=false when the next token is a quoted string and
// the caller should fall through to the spec-form path. Shared by
// decodeBytes (decimal + big-decimal) and decodeFixed (decimal only;
// big-decimal is bytes-only per spec) so all three sites agree on
// scale derivation and target-set dispatch.
func (ctx *jsonDecoder) decodeBareDecimal(v reflect.Value, node *schemaNode, toAny bool) (handled bool, err error) {
	c := ctx.scanner.peek()
	if c == '"' || c == 0 {
		return false, nil
	}
	nb, perr := ctx.scanner.consumeNumberBytes()
	if perr != nil {
		return true, perr
	}
	// boundedRatFromString and its callees (json.Valid, strconv.ParseInt,
	// big.Rat.SetString, fmt.Errorf) treat the string read-only and don't
	// retain it past the call, so alias nb instead of copying.
	r, ok, perr := boundedRatFromString(unsafe.String(unsafe.SliceData(nb), len(nb)))
	if perr != nil {
		return true, fmt.Errorf("avro json: %s %q: %w", node.logical, truncBytesForError(nb), perr)
	}
	if !ok {
		return true, fmt.Errorf("avro json: invalid %s number %q", node.logical, truncBytesForError(nb))
	}
	if toAny {
		return true, setIface(v, reflect.ValueOf(r), node.kind)
	}
	// Decimal uses the schema-declared node.scale; big-decimal has no
	// schema-level scale (it's encoded inline on the wire), so derive
	// the natural scale from the rat. Scale is consulted only by
	// setDecimalRat's json.Number / string target arms; for big.Rat,
	// float, and interface targets the value is unchanged.
	scale := node.scale
	if node.logical == "big-decimal" {
		s, ok := finiteScale(r)
		if !ok {
			return true, fmt.Errorf("avro json: big-decimal value %s has no finite decimal expansion", truncRatForError(r))
		}
		scale = s
	}
	if applied, err := setDecimalRat(v, r, scale); applied {
		return true, err
	}
	return true, &SemanticError{GoType: v.Type(), AvroType: node.kind}
}

func (ctx *jsonDecoder) decodeArray(v reflect.Value, node *schemaNode, toAny bool) error {
	if err := ctx.scanner.expect('['); err != nil {
		return err
	}
	if toAny {
		var arr []any
		if ctx.scanner.peek() != ']' {
			var elem any
			elemV := reflect.ValueOf(&elem).Elem()
			for {
				if err := ctx.decodeValue(elemV, node.items); err != nil {
					return err
				}
				arr = append(arr, elem)
				elem = nil
				if ctx.scanner.peek() != ',' {
					break
				}
				ctx.scanner.pos++ // consume comma
			}
		}
		if arr == nil {
			arr = []any{}
		}
		if err := ctx.scanner.expect(']'); err != nil {
			return err
		}
		return setIface(v, reflect.ValueOf(arr), "array")
	}
	// Typed array target ([N]T): decode each JSON element into v.Index(i),
	// require exactly len(v) elements. Mirrors deserArray.deserFixedArray
	// on the binary side (deser.go); the JSON encoder accepts the same
	// target via appendAvroJSON case "array" (json_codec.go), so without
	// this branch [N]T round-trips bin→JSON but not JSON→JSON.
	if v.Kind() == reflect.Array {
		arrLen := v.Len()
		idx := 0
		if ctx.scanner.peek() != ']' {
			for {
				if idx >= arrLen {
					return &SemanticError{GoType: v.Type(), AvroType: "array", Err: fmt.Errorf("expected %d elements, got more", arrLen)}
				}
				if err := ctx.decodeValue(v.Index(idx), node.items); err != nil {
					return err
				}
				idx++
				if ctx.scanner.peek() != ',' {
					break
				}
				ctx.scanner.pos++
			}
		}
		if idx != arrLen {
			return &SemanticError{GoType: v.Type(), AvroType: "array", Err: fmt.Errorf("expected %d elements, got %d", arrLen, idx)}
		}
		return ctx.scanner.expect(']')
	}
	// Typed slice target.
	if v.Kind() != reflect.Slice {
		return semErr(v, "array")
	}
	// Native concrete fast path: plain primitive item + unnamed []V. Drops the
	// per-element reflect.Append + reflect parse. Logical items / named slice /
	// named elem fall through.
	if node.items.logical == "" && node.items.decodeJSON == nil {
		if handled, err := decodeJSONNativeSliceDispatch(ctx, v, node.items); handled {
			return err
		}
	}
	v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	if ctx.scanner.peek() != ']' {
		elem := reflect.New(v.Type().Elem()).Elem()
		for {
			if err := ctx.decodeValue(elem, node.items); err != nil {
				return err
			}
			v.Set(reflect.Append(v, elem))
			elem.SetZero()
			if ctx.scanner.peek() != ',' {
				break
			}
			ctx.scanner.pos++
		}
	}
	return ctx.scanner.expect(']')
}

func (ctx *jsonDecoder) decodeMap(v reflect.Value, node *schemaNode, toAny bool) error {
	if err := ctx.scanner.expect('{'); err != nil {
		return err
	}
	if toAny {
		m := make(map[string]any)
		if ctx.scanner.peek() != '}' {
			var val any
			valV := reflect.ValueOf(&val).Elem()
			for {
				key, err := ctx.consumeSlabString()
				if err != nil {
					return err
				}
				if err := ctx.scanner.expect(':'); err != nil {
					return err
				}
				if err := ctx.decodeValue(valV, node.values); err != nil {
					return err
				}
				m[key] = val
				val = nil
				if ctx.scanner.peek() != ',' {
					break
				}
				ctx.scanner.pos++
			}
		}
		if err := ctx.scanner.expect('}'); err != nil {
			return err
		}
		return setIface(v, reflect.ValueOf(m), "map")
	}
	// Typed map target.
	if v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String {
		return semErr(v, "map")
	}
	if v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}
	keyType := v.Type().Key()
	valType := v.Type().Elem()
	// Native concrete fast path: plain primitive value + exactly-string key.
	// Drops the per-entry reflect SetMapIndex (m[k]=v instead). The reflect
	// parse into a reused elem stays; logical / named / non-interfaceable
	// fall through. (Array JSON decode has no equivalent — it already parses
	// in place via Index(i), so there's no SetMapIndex to remove.)
	if node.values.logical == "" && node.values.decodeJSON == nil && keyType == stringType && v.CanInterface() {
		if handled, err := decodeJSONNativeMap(ctx, v, node.values); handled {
			return err
		}
	}
	if ctx.scanner.peek() != '}' {
		elem := reflect.New(valType).Elem()
		// Reusable key Value typed to match the user's map key type
		// (handles `type UserID string; map[UserID]V` without panic).
		keyVal := reflect.New(keyType).Elem()
		for {
			key, err := ctx.consumeSlabString()
			if err != nil {
				return err
			}
			if err := validateJSONNumberMapKey(key, keyType, "map"); err != nil {
				return err
			}
			if err := ctx.scanner.expect(':'); err != nil {
				return err
			}
			if err := ctx.decodeValue(elem, node.values); err != nil {
				return err
			}
			keyVal.SetString(key)
			v.SetMapIndex(keyVal, elem)
			elem.SetZero()
			if ctx.scanner.peek() != ',' {
				break
			}
			ctx.scanner.pos++
		}
	}
	return ctx.scanner.expect('}')
}

// JSON parse-to-native leaves: scan + parse one JSON token straight into the
// Go value, no reflect.Value. decodeInt/decodeFloat/decodeBool/decodeString
// are each one of these leaves plus a setXValue, so the native map/slice loops
// below reuse the exact same token parsing — the leaf is the shared point.
func jsonReadString(c *jsonDecoder) (string, error) { return c.consumeSlabString() }
func jsonReadBool(c *jsonDecoder) (bool, error)     { return c.scanner.consumeBool() }
func jsonReadInt32(c *jsonDecoder) (int32, error) {
	nb, err := c.scanner.consumeNumberBytes()
	if err != nil {
		return 0, err
	}
	return parseJSONInt32(nb)
}
func jsonReadInt64(c *jsonDecoder) (int64, error) {
	nb, err := c.scanner.consumeNumberBytes()
	if err != nil {
		return 0, err
	}
	return parseJSONInt64(nb)
}

// jsonReadInt is only reached when int is 64-bit (its callers gate on
// strconv.IntSize == 64), so int(n) is a lossless identity there. On 32-bit
// the long→int native arms fall back to the overflow-checked reflect path.
func jsonReadInt(c *jsonDecoder) (int, error) { n, err := jsonReadInt64(c); return int(n), err }
func jsonReadFloat32(c *jsonDecoder) (float32, error) {
	f, err := c.decodeJSONFloat(32, "float")
	return float32(f), err
}
func jsonReadFloat64(c *jsonDecoder) (float64, error) { return c.decodeJSONFloat(64, "double") }

// decodeJSONNativeStringMap stores each parsed value straight into m — no
// reflect SetMapIndex, no reflect parse (readOne yields a native V).
func decodeJSONNativeStringMap[V any](ctx *jsonDecoder, m map[string]V, readOne func(*jsonDecoder) (V, error)) error {
	if ctx.scanner.peek() != '}' {
		for {
			key, err := ctx.consumeSlabString()
			if err != nil {
				return err
			}
			if err := ctx.scanner.expect(':'); err != nil {
				return err
			}
			val, err := readOne(ctx)
			if err != nil {
				return err
			}
			m[key] = val
			if ctx.scanner.peek() != ',' {
				break
			}
			ctx.scanner.pos++
		}
	}
	return ctx.scanner.expect('}')
}

// decodeJSONNativeSlice builds a native []V via append and sets it once,
// dropping the generic path's per-element reflect.Append AND reflect parse.
func decodeJSONNativeSlice[V any](ctx *jsonDecoder, v reflect.Value, readOne func(*jsonDecoder) (V, error)) error {
	var s []V
	if ctx.scanner.peek() != ']' {
		for {
			val, err := readOne(ctx)
			if err != nil {
				return err
			}
			s = append(s, val)
			if ctx.scanner.peek() != ',' {
				break
			}
			ctx.scanner.pos++
		}
	}
	if s == nil {
		s = []V{}
	}
	v.Set(reflect.ValueOf(s))
	return ctx.scanner.expect(']')
}

// decodeJSONNativeMap routes an unnamed map[string]V of a plain primitive to
// the native loop. handled=false (scanner untouched — the assertion fails
// before any read) for named map/value types, which fall back to reflect.
func decodeJSONNativeMap(ctx *jsonDecoder, v reflect.Value, valNode *schemaNode) (bool, error) {
	switch et := v.Type().Elem(); {
	case valNode.kind == "string" && et == stringType:
		if m, ok := v.Interface().(map[string]string); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadString)
		}
	case valNode.kind == "int" && et == int32Type:
		if m, ok := v.Interface().(map[string]int32); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadInt32)
		}
	case valNode.kind == "long" && et == int64Type:
		if m, ok := v.Interface().(map[string]int64); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadInt64)
		}
	// long → int: int(int64) narrows on 32-bit platforms (int is 32-bit
	// there), silently truncating an out-of-int32 wire value where the reflect
	// path errors. Gate on a 64-bit int (compile-time constant) so 32-bit
	// falls back to the overflow-checked reflect path. (The int32/int64 arms
	// are safe — parseJSONInt32/parseJSONInt64 range-check.)
	case valNode.kind == "long" && et == intType && strconv.IntSize == 64:
		if m, ok := v.Interface().(map[string]int); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadInt)
		}
	case valNode.kind == "float" && et == float32Type:
		if m, ok := v.Interface().(map[string]float32); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadFloat32)
		}
	case valNode.kind == "double" && et == float64Type:
		if m, ok := v.Interface().(map[string]float64); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadFloat64)
		}
	case valNode.kind == "boolean" && et == boolType:
		if m, ok := v.Interface().(map[string]bool); ok {
			return true, decodeJSONNativeStringMap(ctx, m, jsonReadBool)
		}
	}
	return false, nil
}

// decodeJSONNativeSliceDispatch routes an unnamed []V of a plain primitive to
// the native loop. A named slice type (Name() != "") can't take v.Set([]V),
// and a named element type misses the exact-type case; both fall back.
func decodeJSONNativeSliceDispatch(ctx *jsonDecoder, v reflect.Value, itemNode *schemaNode) (bool, error) {
	if v.Type().Name() != "" {
		return false, nil
	}
	switch et := v.Type().Elem(); {
	case itemNode.kind == "string" && et == stringType:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadString)
	case itemNode.kind == "int" && et == int32Type:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadInt32)
	case itemNode.kind == "long" && et == int64Type:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadInt64)
	case itemNode.kind == "long" && et == intType && strconv.IntSize == 64:
		// See decodeJSONNativeMap: 32-bit int narrows; gate to 64-bit so
		// 32-bit uses the overflow-checked reflect path.
		return true, decodeJSONNativeSlice(ctx, v, jsonReadInt)
	case itemNode.kind == "float" && et == float32Type:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadFloat32)
	case itemNode.kind == "double" && et == float64Type:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadFloat64)
	case itemNode.kind == "boolean" && et == boolType:
		return true, decodeJSONNativeSlice(ctx, v, jsonReadBool)
	}
	return false, nil
}

func (ctx *jsonDecoder) decodeRecord(v reflect.Value, node *schemaNode, toAny bool) error {
	if err := ctx.scanner.expect('{'); err != nil {
		return err
	}
	if toAny {
		return ctx.decodeRecordAny(v, node)
	}
	// Typed target: struct or map[string]T.
	if v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String {
		return ctx.decodeRecordMap(v, node)
	}
	if v.Kind() == reflect.Struct {
		return ctx.decodeRecordStruct(v, node)
	}
	return semErr(v, "record")
}

// iterateRecordFields drives the JSON object field loop for records:
// dispatch each key to handle, skip unknown keys, and after the loop
// invoke fillDefault for any absent field that has a schema default
// (errors otherwise). fillDefault may be nil.
func (ctx *jsonDecoder) iterateRecordFields(node *schemaNode, handle func(idx int, key string) error, fillDefault func(idx int) error) error {
	seen := make([]bool, len(node.fields))
	// Track WHICH JSON key claimed each reader slot, so a second key
	// that resolves to the same field-index (the canonical name plus
	// an alias both appearing in the same JSON object) produces an
	// error rather than silently overwriting. The schema parse already
	// rejects within-schema alias/name collisions at schema.go:1999, so
	// fieldIdx only has multiple keys per index for the legitimate
	// "renamed-with-alias" case — and a single JSON object emitting
	// both forms is the producer-side ambiguity this guard catches.
	seenKey := make([]string, len(node.fields))
	if ctx.scanner.peek() != '}' {
		for {
			// Zero-copy: key is used only for fieldIdx lookup and
			// error messages, never stored in output.
			key, err := ctx.scanner.consumeStringZeroCopy()
			if err != nil {
				return err
			}
			if err := ctx.scanner.expect(':'); err != nil {
				return err
			}
			idx := -1
			if node.fieldIdx != nil {
				if i, ok := node.fieldIdx[key]; ok {
					idx = i
				}
			}
			if idx < 0 {
				if err := ctx.scanner.skipValue(); err != nil {
					return err
				}
			} else {
				// Reject ONLY when two DIFFERENT JSON keys resolve to the
				// same idx (the alias-collision case). The same canonical
				// key appearing twice falls through to last-wins (handle
				// is called again, decoding the second value and
				// overwriting the first), matching Java's Jackson,
				// fastavro's Python json.loads, and Go's encoding/json on
				// duplicate keys.
				if seen[idx] && seenKey[idx] != key {
					return fmt.Errorf("avro json: record %q field %q resolved from both %q and %q in the same JSON object",
						truncForError(node.name), truncForError(node.fields[idx].name), truncForError(seenKey[idx]), truncForError(key))
				}
				seen[idx] = true
				seenKey[idx] = key
				if err := handle(idx, key); err != nil {
					return err
				}
			}
			if ctx.scanner.peek() != ',' {
				break
			}
			ctx.scanner.pos++
		}
	}
	if err := ctx.scanner.expect('}'); err != nil {
		return err
	}
	for i, f := range node.fields {
		if seen[i] {
			continue
		}
		if !f.hasDefault {
			return fmt.Errorf("avro json: record %q missing required field %q", truncForError(node.name), truncForError(f.name))
		}
		if fillDefault != nil {
			if err := fillDefault(i); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ctx *jsonDecoder) decodeRecordAny(v reflect.Value, node *schemaNode) error {
	// Fail fast on a target that can't hold the result, mirroring
	// deserRecord.deser. The caller (decodeRecord) has already
	// consumed the leading '{', so this advances at most one byte
	// before erroring — but it avoids the much larger waste of
	// iterating each field, allocating a map, and decoding values
	// only to throw them away on assignment.
	if v.Type().NumMethod() != 0 && !mapStringAnyType.AssignableTo(v.Type()) {
		return semErr(v, "record")
	}
	// Reuse the existing map[string]any if v already wraps one — the
	// streaming pattern (DecodeJSON repeatedly into the same *any).
	// See [reuseOrMakeStringAnyMap].
	m := reuseOrMakeStringAnyMap(v, len(node.fields))
	var val any
	valV := reflect.ValueOf(&val).Elem()
	err := ctx.iterateRecordFields(node,
		func(idx int, key string) error {
			f := &node.fields[idx]
			if err := ctx.decodeValue(valV, f.node); err != nil {
				return recordFieldError(v.Type(), f.name, err)
			}
			m[f.name] = val
			val = nil
			return nil
		},
		func(idx int) error {
			f := &node.fields[idx]
			var defVal any
			defValV := reflect.ValueOf(&defVal).Elem()
			if err := ctx.applyFieldDefault(defValV, node, idx); err != nil {
				return fmt.Errorf("field %q default: %w", truncForError(f.name), err)
			}
			m[f.name] = defVal
			return nil
		},
	)
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(m))
	return nil
}

func (ctx *jsonDecoder) decodeRecordMap(v reflect.Value, node *schemaNode) error {
	if v.IsNil() {
		v.Set(reflect.MakeMapWithSize(v.Type(), len(node.fields)))
	}
	elem := reflect.New(v.Type().Elem()).Elem()
	mapType := v.Type()
	keyType := mapType.Key()
	return ctx.iterateRecordFields(node,
		func(idx int, key string) error {
			f := &node.fields[idx]
			if err := validateJSONNumberMapKey(f.name, keyType, "record"); err != nil {
				return err
			}
			if err := ctx.decodeValue(elem, f.node); err != nil {
				return recordFieldError(v.Type(), f.name, err)
			}
			v.SetMapIndex(mapKeyAs(mapType, f.nameVal), elem)
			elem.SetZero()
			return nil
		},
		func(idx int) error {
			f := &node.fields[idx]
			if err := validateJSONNumberMapKey(f.name, keyType, "record"); err != nil {
				return err
			}
			if err := ctx.applyFieldDefault(elem, node, idx); err != nil {
				return fmt.Errorf("field %q default: %w", truncForError(f.name), err)
			}
			v.SetMapIndex(mapKeyAs(mapType, f.nameVal), elem)
			elem.SetZero()
			return nil
		},
	)
}

func (ctx *jsonDecoder) decodeRecordStruct(v reflect.Value, node *schemaNode) error {
	dr := node.deserRecord
	if dr == nil {
		return &SemanticError{GoType: v.Type(), AvroType: "record", Err: errors.New("no record metadata")}
	}
	mapping, err := typeFieldMapping(dr.names, &dr.cache, v.Type())
	if err != nil {
		return err
	}
	return ctx.iterateRecordFields(node,
		func(idx int, key string) error {
			f := &node.fields[idx]
			fv, err := fieldByIndex(v, mapping.indices[idx])
			if err != nil {
				return recordFieldError(v.Type(), f.name, err)
			}
			if err := ctx.decodeValue(fv, f.node); err != nil {
				return recordFieldError(v.Type(), f.name, err)
			}
			return nil
		},
		func(idx int) error {
			f := &node.fields[idx]
			if len(mapping.indices[idx]) == 0 {
				// Struct has no field for this Avro field — nothing to fill.
				// Mirrors decodeRecord's tolerance of struct-field omission.
				return nil
			}
			fv, err := fieldByIndex(v, mapping.indices[idx])
			if err != nil {
				return fmt.Errorf("field %q default: %w", truncForError(f.name), err)
			}
			if err := ctx.applyFieldDefault(fv, node, idx); err != nil {
				return fmt.Errorf("field %q default: %w", truncForError(f.name), err)
			}
			return nil
		},
	)
}

// applyFieldDefault decodes the field's pre-encoded binary default into target
// via the record's WRAPPED binary deserfn, the same one a present field uses.
// That is what makes a registered CustomType.Decode fire for default-filled
// fields. node.fields[idx].node.deser is the unwrapped primitive, built before
// applyCustomTypes installed the chain, so calling it directly surfaces the raw
// Avro-native value into a target expecting the custom domain type.
//
// A zero-length defaultBytes is a VALID default for any field whose wire
// encoding is naturally 0 bytes: null-typed fields, empty records, records of
// all-null fields. The caller already gated on f.hasDefault, so the check below
// only guards a malformed schema missing serRecord — built in lockstep with
// deserRecord, so it covers both.
func (ctx *jsonDecoder) applyFieldDefault(target reflect.Value, node *schemaNode, idx int) error {
	if node.serRecord == nil || idx >= len(node.serRecord.fields) {
		return fmt.Errorf("record has no pre-encoded default for field %d", idx)
	}
	enc := node.serRecord.fields[idx].defaultBytes
	// Copy the encoded bytes — deserfns may slab-substring into src
	// and we don't want them to reach into the schema's shared default.
	src := append([]byte(nil), enc...)
	_, err := node.deserRecord.fields[idx].fn(src, target, ctx.slab)
	return err
}

// unionBranchRecurses reports whether a union branch kind decodes a nested
// value that can recurse back into the union (record/array/map). The bare and
// tagged JSON union decoders commit to the first such branch instead of
// re-decoding the subtree as a later container branch: backtracking across
// recursive container branches is 2^depth (a hostile-input DoS), and the Avro
// JSON spec's tagged {"branch":value} form — which Java/fastavro/goavro require
// — never branch-guesses. Scalar branches cannot recurse, so they keep their
// bounded backtrack.
func unionBranchRecurses(kind string) bool {
	return kind == "record" || kind == "array" || kind == "map"
}

func (ctx *jsonDecoder) decodeUnion(v reflect.Value, node *schemaNode) error {
	p := ctx.scanner.peek()

	// JSON null → null branch, if the union has one. Handled before
	// indirectAlloc so *T targets stay nil. Java and fastavro both reject null
	// when no "null" label is in the union; this matches.
	//
	// isJSONNullStart disambiguates from bare special-float tokens. A bare 'n'
	// is unambiguous today, since parseSpecialFloat rejects lowercase, but the
	// helper stays so a future leniency re-accepting lowercase nan cannot be
	// hijacked into the null arm. decodeFloat and decodeDouble use it likewise.
	if isJSONNullStart(ctx.scanner, p) {
		hasNull := false
		for _, br := range node.branches {
			if br.kind == "null" {
				hasNull = true
				break
			}
		}
		if !hasNull {
			return fmt.Errorf("avro json: union has no null branch but value is null at offset %d", ctx.scanner.pos)
		}
		if err := ctx.scanner.consumeNull(); err != nil {
			return err
		}
		setZero(v)
		return nil
	}

	// Branch indirection is PER-BRANCH (see decodeBranchInto / decodeUnionObject),
	// mirroring the binary deserUnion.deser path: a custom-decode branch decodes
	// against the un-indirected target so a Decode returning a pointer lands in an
	// interface/pointer via setCustomResult (as binary's wrapDeserWithCustomDecoders
	// does), while a non-custom branch indirects in decodeKind (in-place reuse of a
	// *T held in an interface; value boxing for a nil/value interface).
	// Pre-indirecting the union target here would dereference a reused *T held in
	// an interface and reject a custom pointer result — a binary↔JSON divergence
	// on the target-reuse contract.

	// JSON object → could be tagged union {"type": value} or bare record/map.
	if p == '{' {
		return ctx.decodeUnionObject(v, node)
	}

	// Bare non-object value — match by JSON token type.
	return ctx.decodeUnionBare(v, node, p)
}

func (ctx *jsonDecoder) decodeUnionObject(v reflect.Value, node *schemaNode) error {
	savedPos := ctx.scanner.pos
	// Preserve the deepest concrete decode error from a matched branch
	// so a failed tagged interpretation surfaces the real reason (e.g.
	// "cannot assign float to map[string]any") rather than being masked
	// by the bare-fallback's generic "no union branch matched".
	var taggedErr error

	// Try tagged union: {"branchName": value}.
	ctx.scanner.pos++ // consume '{'
	ctx.scanner.skipWhitespace()
	if ctx.scanner.peek() != '}' {
		// Zero-copy: key is used only for branch name lookup, never stored.
		key, err := ctx.scanner.consumeStringZeroCopy()
		if err == nil {
			if branch := findUnionBranch(node, key); branch != nil {
				if err := ctx.scanner.expect(':'); err == nil {
					target, toAny := unionTarget(v, branch)
					if toAny {
						// Decode into a tmp `any` first so the target stays
						// untouched until the close-brace arrives — a malformed
						// tagged payload like `{"long": 42,` would otherwise write
						// it and THEN backtrack to the bare-fallback, leaving it
						// dirty on the final err. For a custom branch into an
						// interface, unionTarget returns the raw interface so
						// assignAny sets the (possibly pointer) custom result into
						// it rather than a pre-dereferenced pointee.
						var val any
						err := ctx.decodeValue(reflect.ValueOf(&val).Elem(), branch)
						if err == nil {
							if ctx.scanner.peek() == '}' {
								ctx.scanner.pos++
								return assignAny(target, ctx.wrapUnion(target, val, node, branch), branch.kind)
							}
						} else if errors.Is(err, errTooDeep) {
							// Don't fall through to bare-union retry; the recursion
							// limit applies regardless of how the branch is matched.
							return err
						} else if unionBranchRecurses(branch.kind) {
							// Commit to the tagged interpretation for a CONTAINER
							// branch: do NOT fall back to the bare retry below. The
							// bare retry re-decodes the whole subtree, and when a
							// record field name collides with a branch name the tagged
							// decode and the bare retry BOTH recurse → 2^depth (the
							// same DoS the decodeUnionBare commit-to-first prevents).
							// {"branch":value} is the spec's tagged form; a key
							// matching a container branch name commits to it. Scalar
							// branches can't recurse, so they keep the bare fallback.
							return err
						} else {
							taggedErr = err
						}
					} else {
						// Typed path: decodeValue writes target directly.
						// Backtracking after a partial write is acceptable — the
						// only trigger is a missing close brace on otherwise-valid
						// JSON, and the bare fallback overwrites if it matches.
						err := ctx.decodeValue(target, branch)
						if err == nil {
							if ctx.scanner.peek() == '}' {
								ctx.scanner.pos++
								return nil
							}
						} else if errors.Is(err, errTooDeep) {
							return err
						} else if unionBranchRecurses(branch.kind) {
							// Commit to the tagged container interpretation; see the
							// toAny arm above (the bare retry would double the
							// recursion → 2^depth on a field/branch name collision).
							return err
						} else {
							taggedErr = err
						}
					}
				}
			}
		}
	}

	// Tagged interpretation failed — backtrack and try bare. Pass the
	// tagged-side concrete error so it can be surfaced if bare also
	// fails to match.
	ctx.scanner.pos = savedPos
	if err := ctx.decodeUnionBare(v, node, '{'); err != nil {
		if taggedErr != nil {
			return fmt.Errorf("%w (tagged-form: %v)", err, taggedErr)
		}
		return err
	}
	return nil
}

// decodeBranchInto decodes the next JSON value as the given union branch
// and writes the result into v. Used by decodeUnionBare where the entire
// branch interpretation either fully succeeds (return nil) or fully
// fails (caller backtracks and tries the next branch). decodeUnionObject
// uses an inline tmp `any` instead since it must hold the decoded value
// pending a close-brace check before committing to v — see the comment
// on its tagged-path arm.
func (ctx *jsonDecoder) decodeBranchInto(rawV reflect.Value, union, branch *schemaNode) error {
	v, toAny := unionTarget(rawV, branch)
	if toAny {
		var val any
		if err := ctx.decodeValue(reflect.ValueOf(&val).Elem(), branch); err != nil {
			return err
		}
		// wrapUnion returns nil for null branches; reflect.ValueOf(nil)
		// is the invalid zero Value, so use assignAny which sets a typed
		// nil for interface targets.
		return assignAny(v, ctx.wrapUnion(v, val, union, branch), branch.kind)
	}
	return ctx.decodeValue(v, branch)
}

// unionTarget selects the decode target and toAny flag for a matched union
// branch, mirroring the binary deserUnion.deser per-branch indirection: the
// binary union passes the branch fn the un-dereferenced target. A non-custom
// branch indirects (reusing a *T held in an interface IN PLACE, or boxing a
// value); a custom branch keeps the raw target so its wrapper's setCustomResult
// can land a pointer result into a reused interface or a concrete *T field.
func unionTarget(rawV reflect.Value, branch *schemaNode) (reflect.Value, bool) {
	if branch.decodeJSON != nil {
		// Custom-decode branch: decode against the UN-indirected target (any
		// kind) so the wrapper's setCustomResult lands a pointer result into a
		// reused *T held in an interface OR a concrete *T field — exactly as the
		// binary deserUnion.deser passes the un-dereferenced target. Pre-
		// dereferencing here rejected a Decode that returns a pointer. toAny
		// routes the interface case through the wrap path.
		return rawV, rawV.Kind() == reflect.Interface
	}
	iv := indirectAlloc(rawV)
	if iv.Kind() == reflect.Interface {
		// Interface target: the toAny path assigns the decoded value into this
		// peeled interface directly (assignAny), never re-decoding into it, so
		// there is no second indirection — return the peeled interface.
		return iv, true
	}
	// Concrete target: return the UN-peeled rawV. The branch decode runs its own
	// single indirectAlloc (decodeKind), which peels rawV from the top, capping
	// at maxIndirectDepth — matching binary's single peel in the leaf decoder.
	// Returning the already-peeled iv would make that second indirectAlloc peel
	// a FURTHER maxIndirectDepth levels, so a union concrete-pointer target
	// accepted up to 2*maxIndirectDepth levels where binary (and a non-union
	// target) rejects past maxIndirectDepth — a binary↔JSON decode divergence.
	// (indirectAlloc above already allocated the in-cap chain, so the re-peel
	// reuses it; its only purpose here is to settle toAny.)
	return rawV, false
}

func (ctx *jsonDecoder) decodeUnionBare(v reflect.Value, node *schemaNode, p byte) error {
	// Match by JSON token type against branch kinds. The last branch's
	// decode error (if any) is preserved so the final message names the
	// concrete reason — typically a target-type mismatch like the binary
	// path reports ("cannot use map[string]any with Avro type float").
	// Without this, callers saw a generic "no union branch matched at
	// offset N" that hid the actual root cause.
	var lastErr error
	for _, branch := range node.branches {
		// Skip null: decodeUnion's upstream isJSONNullStart filter
		// pre-routes JSON null literals before this loop runs, so any
		// peek byte reaching here is guaranteed NOT to start a null
		// token. The skip avoids jsonTokenMatchesBranch's default arm
		// from matching peek byte 'n' (a bare-special-float start like
		// "nan") against the null branch — decodeJSONFloat will reject
		// the lowercase form downstream, but routing through null
		// first would emit a misleading error. If isJSONNullStart's
		// accept set ever broadens (e.g. lowercase 'nan' handling
		// changes), re-verify this skip can't drop a now-reachable
		// null branch.
		if branch.kind == "null" {
			continue
		}
		if !jsonTokenMatchesBranch(p, branch) {
			continue
		}
		savedPos := ctx.scanner.pos
		if err := ctx.decodeBranchInto(v, node, branch); err == nil {
			return nil
		} else if errors.Is(err, errTooDeep) {
			return err
		} else {
			lastErr = err
			// Commit to the FIRST token-class-matching CONTAINER branch; do not
			// backtrack. Backtracking re-decodes the subtree per branch, 2^depth
			// on a recursive union of records/arrays/maps — a ~120-byte bare
			// nested object then rejects in seconds. The spec's tagged form
			// names the branch, and the tagged path already commits
			// deterministically, so a caller needing a later container branch
			// uses it. Container tokens match only container branches, so this
			// never skips a scalar one. Scalar branches cannot recurse, so their
			// bounded backtrack stays and ["int","long"] still falls through to
			// long for an int-overflowing value at O(1) per node.
			if unionBranchRecurses(branch.kind) {
				break
			}
			ctx.scanner.pos = savedPos
		}
	}
	if lastErr != nil {
		return fmt.Errorf("avro json: no union branch matched at offset %d: %w", ctx.scanner.pos, lastErr)
	}
	return fmt.Errorf("avro json: no union branch matched at offset %d", ctx.scanner.pos)
}

func (ctx *jsonDecoder) wrapUnion(v reflect.Value, val any, union, branch *schemaNode) any {
	if !ctx.slab.taggedUnions || val == nil {
		return val
	}
	// Mirror the binary wrap (deserUnion.maybeWrap): the {branch: value}
	// envelope applies only to interface targets that map[string]any is
	// assignable to. Any other interface target (non-empty interfaces)
	// receives the bare branch value — the wrap is skipped silently,
	// never turned into an assignment error.
	if v.Kind() == reflect.Interface && !mapStringAnyType.AssignableTo(v.Type()) {
		return val
	}
	// Reuse unionEmitTag so the tagged-map key produced on decode is
	// byte-identical to what the encode side emits — in particular, a
	// named fixed carrying a logical type wraps under its NAME, not
	// "fixed.<logicalType>" (see unionBranchNames for the goavro/Java
	// references this mirrors), and a logical qualifier another branch
	// owns as its exact name degrades to the unqualified name on BOTH
	// sides rather than only one.
	return map[string]any{unionEmitTag(union, branch, ctx.slab.tagLogicalTypes): val}
}

// jsonTokenMatchesBranch returns true if a JSON token type could
// potentially match a given schema branch kind.
func jsonTokenMatchesBranch(p byte, branch *schemaNode) bool {
	switch p {
	case '"':
		switch branch.kind {
		case "string", "enum", "bytes", "fixed":
			return true
		case "float", "double":
			return true // NaN/Infinity strings
		}
	case 't', 'f':
		return branch.kind == "boolean"
	case '[':
		return branch.kind == "array"
	case '{':
		return branch.kind == "record" || branch.kind == "map"
	default: // digit or '-'
		switch branch.kind {
		case "int", "long", "float", "double":
			return true
		case "bytes", "fixed":
			// Lenient decode: a hand-edited or alternate-tool JSON
			// producer may emit a decimal-like-typed bytes/fixed
			// branch as a bare number. decodeBytes / decodeFixed
			// accept both forms via decodeBareDecimal; dispatch must
			// offer the branch so the number-form reaches them.
			// Big-decimal is bytes-only per spec, hence not eligible
			// on a fixed branch.
			return hasDecimalBareNumberArm(branch)
		}
	}
	return false
}
