package avro

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type serfn func([]byte, reflect.Value, int) ([]byte, error)

// maxDepth bounds recursion in both the encoder and decoder. On the
// encoder side this protects against cyclic Go input against recursive
// schemas (stack overflow is fatal in Go). On the decoder side it
// protects against malicious wire data driving unbounded recursion via
// recursive schemas (e.g. linked-list "Node" with all "next" fields
// non-null). 1000 is well below Go's stack growth limit and far above
// any legitimate Avro depth.
const maxDepth = 1000

var errTooDeep = errors.New("avro: recursion limit exceeded (cyclic or pathologically deep input)")

// AppendEncode appends the Avro binary encoding of v to dst. See
// [Schema.Decode] for the Go-to-Avro type mapping. In addition to the types
// listed there, encoding also accepts:
//   - [encoding/json.Number] for any numeric Avro type (int, long, float, double)
//   - RFC 3339 strings for timestamp and date logical types
//   - [*big.Rat], [big.Rat], float32, float64, [encoding/json.Number], and numeric strings for decimal logical types
//   - [encoding.TextAppender], [encoding.TextMarshaler], and []byte for string types (and vice versa for [encoding.TextUnmarshaler])
//   - string (hex-dash UUID format) for fixed(16) UUID logical types
//   - Tagged union maps (map[string]any{"typeName": value}) for union types,
//     as produced by [Schema.Decode] with [TaggedUnions]
func (s *Schema) AppendEncode(dst []byte, v any, opts ...Opt) ([]byte, error) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		// nil is valid for null schemas and unions (null branch).
		switch s.node.kind {
		case "null":
			return dst, nil
		case "union":
			return s.ser(dst, rv, 0)
		default:
			return nil, &SemanticError{AvroType: s.node.kind, Err: errors.New("cannot encode nil")}
		}
	}
	return s.ser(dst, rv, 0)
}

// Encode encodes v as Avro binary. It is shorthand for AppendEncode(nil, v).
func (s *Schema) Encode(v any, opts ...Opt) ([]byte, error) {
	return s.AppendEncode(nil, v, opts...)
}

///////////
// UNION //
///////////

type serUnion struct {
	fns []serfn
	// tags is the union's name lookup, shared with the union's schemaNode
	// (schema.go) so the binary and JSON encoders and the JSON decoder all
	// resolve a caller-written tag and a Go-type name through ONE table.
	// byName unwraps a tagged union map; byKind prefers a type-name match
	// over try-each — mirroring Java's GenericData.resolveUnion and
	// hamba/fastavro's name-based dispatch — and routes nil to null.
	tags *unionTags
}

// tryUnwrapTagged checks if v is a single-key map whose key matches a
// branch name. Returns the branch index and unwrapped value on match.
//
// Routes Pointer/Interface chains through [indirect] so &m / any(&m)
// reach the tagged-map check, mirroring appendAvroJSON's entry peel
// (json_codec.go). Without the peel, AppendEncode(&taggedMap, union)
// silently fell through to try-each while AppendEncodeJSON(&taggedMap,
// union) accepted via the JSON entry-peel, producing a binary↔JSON
// parity gap at top level, inside arrays of unions, and inside record
// fields. indirect's errIndirectNil / errIndirectDeep both surface as
// "no match" so the caller's nil-first dispatch picks the null branch
// (TestMatrix_TaggedUnionEncodeIndirection pins both arms).
func (s *serUnion) tryUnwrapTagged(v reflect.Value) (int, reflect.Value, bool) {
	v, err := indirect(v)
	if err != nil {
		return 0, v, false
	}
	if v.Kind() != reflect.Map || v.Type().Key().Kind() != reflect.String || v.Len() != 1 {
		return 0, v, false
	}
	iter := v.MapRange()
	iter.Next()
	if idx, ok := s.tags.branchByName(iter.Key().String()); ok {
		return idx, iter.Value(), true
	}
	return 0, v, false
}

// ser encodes a union value. Tagged union maps are tried first; if
// that fails or v is not a tagged map, the value's Go-type canonical
// Avro name is dispatched directly (Java/fastavro/hamba parity — see
// branchKinds doc); if no name match exists, each branch is tried in
// order, preserving the documented whole-number-float and
// json.Number-into-int promotion paths.
func (s *serUnion) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	if idx, inner, ok := s.tryUnwrapTagged(v); ok {
		attempt := appendVarint(dst, int32(idx))
		if result, err := s.fns[idx](attempt, inner, depth+1); err == nil {
			return result, nil
		} else if errors.Is(err, errTooDeep) {
			return nil, err
		}
	}

	// Nil-first dispatch: if v is nil-equivalent and the union has a
	// null branch, pick null regardless of arity. Mirrors the 2-branch
	// optimization serNullUnionAt and generalizes the "Go nil = absent
	// → null branch" semantic uniformly across all union arities. Pre-
	// fix, only the 2-branch optimization did this; the generic
	// dispatcher used type-name dispatch first, so nil []byte against
	// ["null","int","bytes"] routed to "bytes" (empty bytes) while the
	// 2-branch sibling ["null","bytes"] routed to null. The two
	// behaviors now agree.
	if nullIdx, ok := s.tags.branchByKind("null"); ok && isNilValue(v) {
		return appendVarint(dst, int32(nullIdx)), nil
	}

	base := dst
	if name := unionTypeNameForValue(v); name != "" {
		if idx, ok := s.tags.branchByKind(name); ok {
			attempt := appendVarint(base, int32(idx))
			if result, err := s.fns[idx](attempt, v, depth+1); err == nil {
				return result, nil
			} else if errors.Is(err, errTooDeep) {
				return nil, err
			}
		}
	}

	// Try every branch; keep the last concrete error so callers see why
	// the closest match failed instead of a generic "no matching branch".
	var lastErr error
	for i, fn := range s.fns {
		attempt := appendVarint(base, int32(i))
		out, err := fn(attempt, v, depth+1)
		if err == nil {
			return out, nil
		}
		// Propagate too-deep immediately; trial loop would mask it.
		if errors.Is(err, errTooDeep) {
			return nil, err
		}
		lastErr = err
	}
	e := &SemanticError{AvroType: "union"}
	if v.IsValid() {
		e.GoType = v.Type()
	}
	if lastErr != nil {
		e.Err = fmt.Errorf("no matching branch: %w", lastErr)
	} else {
		e.Err = errors.New("no matching branch")
	}
	return nil, e
}

// unionTypeNameForValue returns the Avro primitive kind name that
// matches v's Go type directly, or "" if v should fall through to
// try-each (json.Number, time.Time, time.Duration, *big.Rat, etc. —
// the encoder's lenient/coercion paths handle these via try-each).
// Used by serUnion.ser, appendAvroJSONUnion, and encodeDefault's union
// case for consistent first-pass branch selection.
//
// Pointer/interface chains are unwrapped up to maxIndirectDepth (same
// guard as indirect/indirectAlloc) to avoid stack overflow on cyclic
// inputs like `var p any; p = &p` that the fuzz harness produces.
// Cycles beyond the cap return "" — try-each then handles the eventual
// rejection from indirect().
func unionTypeNameForValue(v reflect.Value) string {
	for range maxIndirectDepth {
		if !v.IsValid() {
			return ""
		}
		if v.Type() == jsonNumberType {
			// A json.Number can flow into any of int/long/float/double
			// via numeric coercion — let try-each find the right branch
			// rather than locking it to one. See jsonNumberType.
			return ""
		}
		switch v.Kind() {
		case reflect.Pointer, reflect.Interface:
			if v.IsNil() {
				return "null"
			}
			v = v.Elem()
			continue
		case reflect.Bool:
			return "boolean"
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16:
			return "int"
		case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
			return "long"
		case reflect.Float32:
			return "float"
		case reflect.Float64:
			return "double"
		case reflect.String:
			return "string"
		case reflect.Slice, reflect.Array:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				return "bytes"
			}
		}
		return ""
	}
	return ""
}

// Avro encodes the union branch index as a varint before the value.
// Varint 0 encodes to byte 0x00, varint 1 encodes to byte 0x02
// (zigzag: 1 << 1 = 2). These two-branch null-union helpers inline
// the single-byte varints directly.

// serNullUnion handles ["null", T] unions: null is index 0 (byte 0),
// T is index 1 (byte 2).
func serNullUnion(u *serUnion) serfn { return serNullUnionAt(u, 1, 0, 2) }

// serNullSecondUnion handles ["T", "null"] unions: T is index 0 (byte 0),
// null is index 1 (byte 2).
func serNullSecondUnion(u *serUnion) serfn { return serNullUnionAt(u, 0, 2, 0) }

// serNullUnionAt is the shared implementation. valIdx is the index of T
// in the union; nullByte and valByte are the wire-format bytes for the
// null and value branches respectively.
func serNullUnionAt(u *serUnion, valIdx int, nullByte, valByte byte) serfn {
	return func(dst []byte, v reflect.Value, depth int) ([]byte, error) {
		// The union is a schema node: guard at it exactly like the general
		// serUnion.ser and the decode-side deserNullUnionAt (which both
		// guards AND bumps at the union node). The value branch is entered
		// at depth+1 below, charging the union→branch edge; this guard
		// charges the union node itself. Without it the union edge is
		// counted (via depth+1) but the node is unguarded, so a
		// record{f:["null", container<Self>]} chain trips errTooDeep one
		// level deeper on encode than on every decode/JSON path. See the
		// depth-uniformity invariant in deserNullUnionAt.
		if depth >= maxDepth {
			return nil, errTooDeep
		}
		if isNilValue(v) {
			return append(dst, nullByte), nil
		}
		if idx, inner, ok := u.tryUnwrapTagged(v); ok {
			if idx != valIdx && isNilValue(inner) {
				return append(dst, nullByte), nil
			}
			if idx == valIdx {
				if result, err := u.fns[valIdx](append(dst, valByte), inner, depth+1); err == nil {
					return result, nil
				} else if errors.Is(err, errTooDeep) {
					return nil, err
				}
			}
		}
		return u.fns[valIdx](append(dst, valByte), v, depth+1)
	}
}

// isNilValue reports whether v is nil-equivalent for the 2-branch [null,T]
// union optimization. It peels Pointer / Interface layers — &nilPtr, a **T
// whose outer pointer wraps a nil *T, included — and treats nil Map / Slice /
// Chan / Func as nil. The accept set matches serNull and appendAvroJSON's null
// arm exactly, so all five dispatch sites agree on what counts as nil: both
// binary paths, both JSON paths, and the unsafe struct fast path.
//
// The unsafe site cannot call this — it holds an unsafe.Pointer, not a
// reflect.Value — so it tests the outer pointer alone, which equals isNilValue
// exactly when the inner kind is not itself nilable. Its tryCompileFieldSer
// gate therefore declines every isNilableKind inner to the reflect path,
// staying in lockstep without re-implementing the peel.
//
// Capped at maxIndirectDepth so a self-referential interface (var p any; p =
// &p) terminates. The over-deep case reports not-nil, so the encoder raises a
// real error downstream instead of silently encoding null.
func isNilValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	// Peel Pointer/Interface in one loop, then inspect the final kind
	// in a separate switch — matches serNull's shape so a depth-cap
	// chain bottoming at a nil Map/Slice/Chan/Func is correctly
	// identified. Combining the peel and the Map/Slice/Chan/Func
	// nil-check inside one loop loses the bottom-value check at
	// exactly the depth-cap boundary: the loop would peel the last
	// Pointer to a Map but terminate before the next iteration could
	// inspect Map.IsNil.
	for range maxIndirectDepth {
		if v.Kind() != reflect.Pointer && v.Kind() != reflect.Interface {
			break
		}
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	// The bottom kind set is shared with the unsafe fast-path gate via
	// isNilableKind, so the two can't drift on which kinds count as nil.
	if isNilableKind(v.Kind()) {
		return v.IsNil()
	}
	return false
}

// isNilableKind reports whether k is one of the kinds isNilValue treats as
// nil-equivalent at the bottom of its pointer/interface peel. It is the
// gatekeeper for the unsafe null-union fast paths: usNullUnionEnter /
// usArrayNullUnionPtr decide null-vs-value by testing ONLY the outer pointer
// (*(*unsafe.Pointer)(p) == nil), which equals isNilValue exactly when the
// pointed-to inner is NOT itself nilable. When the inner kind IS nilable — a
// non-nil *T pointing at a nil slice/map/interface/pointer — isNilValue peels
// further and reports null where the bare outer-pointer test reports value, so
// the null-union field and array-element gates decline such inners to the
// reflect path (which consults isNilValue). Mirrors isNilValue's bottom switch
// so the two cannot drift.
func isNilableKind(k reflect.Kind) bool {
	switch k {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return true
	}
	return false
}

////////////////
// PRIMITIVES //
////////////////

var serPrimitive = map[string]serfn{
	"null":    serNull,
	"boolean": serBoolean,
	"int":     serInt,
	"long":    serLong,
	"float":   serFloat,
	"double":  serDouble,
	"bytes":   serBytes,
	"string":  serString,
}

// For unions, we try encoding across all values until one works, and often we
// hit "null" at the start with an error. This error is saved to avoid allocs.
var errNonNil = errors.New("cannot encode non-nil value as null")

// errAppendTextShrunk reports an encoding.TextAppender implementation that
// violated its append contract by returning a slice shorter than its input.
// Saved as a var so a union try-each that hits the same violating value on
// multiple branches doesn't allocate the message per attempt.
var errAppendTextShrunk = errors.New("AppendText returned a slice shorter than its input")

func serNull(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	if !v.IsValid() {
		return dst, nil
	}
	// Peel pointer + interface layers so a typed-nil arriving inside an any
	// wrapper, or as **T with a nil inner (&p where var p *int = nil, the
	// shape AppendEncode(&nilPtr) produces), is recognized as nil. Without the
	// peel both look non-nil: any((*int)(nil)) is an Interface with
	// IsNil()==false because the interface holds type info, and &nilPtr is a
	// Pointer with IsNil()==false because the OUTER pointer is non-nil. The
	// kind switch below would return errNonNil for both. Mirrors
	// appendAvroJSON's indirect loop and isNilValue's. Bounded by
	// maxIndirectDepth so a self-referential interface terminates.
	for range maxIndirectDepth {
		if v.Kind() != reflect.Interface && v.Kind() != reflect.Pointer {
			break
		}
		if v.IsNil() {
			return dst, nil
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		if v.IsNil() {
			return dst, nil
		}
	}
	return dst, errNonNil
}

func appendAvroBool(dst []byte, v reflect.Value) ([]byte, error) {
	if v.Kind() != reflect.Bool {
		return nil, semErr(v, "boolean")
	}
	if v.Bool() {
		return append(dst, 1), nil
	}
	return append(dst, 0), nil
}

// serPrim returns the standard ser primitive wrapper: indirect-then-
// dispatch to appendFn. Used to wire the six primitive serializers
// (serBoolean/Int/Long/Float/Double/String) to their appendAvro*
// helpers. Keeps the indirect+nil-check shape in one place.
func serPrim(appendFn func([]byte, reflect.Value) ([]byte, error)) serfn {
	return func(dst []byte, v reflect.Value, _ int) ([]byte, error) {
		v, err := indirect(v)
		if err != nil {
			return nil, err
		}
		return appendFn(dst, v)
	}
}

var serBoolean = serPrim(appendAvroBool)

// jsonNumberType is compared by type, never by Kind: json.Number's Kind is
// reflect.String, but its stdlib contract is an RFC 8259 number literal, so
// the package treats it as a NUMERIC carrier and accepts it only for the
// numeric Avro types (doc.go, "Encoding from JSON input"). Every string-Kind
// gate on either wire — string, bytes, fixed, enum, the RFC 3339 timestamp
// probe, the union type-name dispatcher, and SchemaFor's Kind switch — must
// therefore exclude this type before it reads the Kind, or a json.Number
// lands on the one Avro type the other wire is guaranteed to reject for it.
var jsonNumberType = reflect.TypeFor[json.Number]()
var mapStringAnyType = reflect.TypeFor[map[string]any]()

// These builtin (unnamed) numeric/bool types are the exact natural Go
// representations for their Avro primitive — e.g. int32 for "int", int64
// for "long". When an array's element type is exactly one of these, the
// encode is a direct read+emit with NO coercion, bounds, or overflow logic
// (the value provably fits the wire type), so the per-element dispatch in
// appendAvroInt / appendAvroFloat / appendAvroBool can be hoisted out of
// the loop. Named or other-width types (e.g. `type Celsius int32`, int8,
// uint32) are NOT matched here — they take the general per-element path,
// which applies the correct coercion/bounds for them.
var (
	boolType    = reflect.TypeFor[bool]()
	int32Type   = reflect.TypeFor[int32]()
	int64Type   = reflect.TypeFor[int64]()
	intType     = reflect.TypeFor[int]()
	float32Type = reflect.TypeFor[float32]()
	float64Type = reflect.TypeFor[float64]()
)

// stringType is the builtin (unnamed) string type. It is the
// overwhelming-common Go type at string/enum encode sites, and being
// unnamed it can carry no methods — so it definitively cannot implement a
// text-out interface. A `v.Type() == stringType` pointer comparison is the
// zero-reflection fast path that lets the common case skip the text-method
// probe (textOutFor) entirely, now that text-out is tried before the
// reflect.String / enum-ordinal arms. Named string types fall through to
// the text-aware path.
var stringType = reflect.TypeFor[string]()

// floatFitsInt32 returns f truncated to int32 and a nil error iff f is a
// whole number within [MinInt32, MaxInt32]. Callers wrap the returned
// error with their SemanticError/context.
func floatFitsInt32(f float64) (int32, error) {
	n := math.Trunc(f)
	if f != n {
		return 0, fmt.Errorf("value %v is not a whole number", f)
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("value %v overflows int32", f)
	}
	return int32(n), nil
}

// floatFitsInt64 returns f truncated to int64 and a nil error iff f is a
// whole number within int64 range. Uses inclusive MinInt64 / exclusive
// 1<<63 since the next representable float64 above MaxInt64 is 1<<63.
func floatFitsInt64(f float64) (int64, error) {
	n := math.Trunc(f)
	if f != n {
		return 0, fmt.Errorf("value %v is not a whole number", f)
	}
	if n < -(1<<63) || n >= 1<<63 {
		return 0, fmt.Errorf("value %v overflows int64", f)
	}
	return int64(n), nil
}

// floatFitsInt32From is floatFitsInt32 with an additional source-float
// mantissa-precision check. When the source is float32 (bits == 32),
// values exceeding ±(1<<24) are rejected — those are values the matching
// decoder's float32-target arm in setIntValue would reject, so accepting
// them on encode breaks the same-type round-trip. bits == 64 needs no
// additional bound: int32 fits within float64's 1<<53 mantissa exactly.
// Source-bit-aware mantissa rule lives here (and floatFitsInt64From) so
// every encode arm that takes Go float input — serInt, serArray.serInt,
// serMap.serInt, jsonCoerceToInt32 — agrees with setIntValue's decode
// arm on the symmetric round-trip boundary.
func floatFitsInt32From(f float64, bits int) (int32, error) {
	n, err := floatFitsInt32(f)
	if err != nil {
		return 0, err
	}
	// Only float32 sources can lose precision inside the int32 range
	// (1<<24 mantissa bound); a float64 source has enough mantissa to
	// represent every int32 exactly.
	if bits == 32 {
		lim := int32(floatMantissaLimit(32))
		if n < -lim || n > lim {
			return 0, fmt.Errorf("value %v exceeds float32 exact-precision range", f)
		}
	}
	return n, nil
}

// floatFitsInt64From is floatFitsInt64 with an additional source-float
// mantissa-precision check. Mirrors setLongValue's float-target precLimit:
// 1<<24 for a float32 source, 1<<53 for float64. The bound lives at
// [floatMantissaLimit] and is also consulted by the decode-side
// [intFitsFloat] (long-wire → smaller Go float-target precision check).
// Encode-side int → float coercion is lossy by destination per Java /
// fastavro parity and does NOT consult this bound; see
// [appendAvroFloat32] / [appendAvroFloat64].
func floatFitsInt64From(f float64, bits int) (int64, error) {
	n, err := floatFitsInt64(f)
	if err != nil {
		return 0, err
	}
	bound := floatMantissaLimit(bits)
	if n < -bound || n > bound {
		return 0, fmt.Errorf("value %v exceeds float%d exact-precision range", f, bits)
	}
	return n, nil
}

// jsonNumberToFloat converts a json.Number reflect.Value to a float64
// reflect.Value suitable for the float encode arms. Returns:
//   - (float64 Value, true, nil) — accepted via parseJSONNumberAsFloat
//     (±Inf from overflow is accepted, matching Java/fastavro/decode).
//   - (v, false, nil) — not a json.Number; caller falls through.
//   - (v, true, err) — IS json.Number but JSON-grammar-invalid (hex
//     float, underscore, exceeds length cap). Java/fastavro reject.
func jsonNumberToFloat(v reflect.Value) (reflect.Value, bool, error) {
	if v.Type() != jsonNumberType {
		return v, false, nil
	}
	f, err := parseJSONNumberAsFloat(string(v.Interface().(json.Number)), 64)
	if err != nil {
		return v, true, err
	}
	return reflect.ValueOf(f), true, nil
}

// parseJSONNumberAsFloat is the shared "json.Number → float64" pipeline:
// gate via [isJSONNumber] (JSON-grammar strict — rejects hex floats,
// underscores, the forms strconv.ParseFloat would accept but JSON does
// not), then parse via [parseFloatAcceptOverflow] (±Inf from ErrRange
// counts as success per the wire-format lossy-destination policy).
//
// Single source of truth for every site turning a JSON-number string into a
// float64 — binary encode, JSON encode, schema-parse default validation, and
// JSON decode — so a future tightening lands once.
//
// bitSize is 64 for every caller but decodeJSONFloat against a "float" schema,
// which passes 32 to parse at float32 precision and avoid a double rounding.
// The isJSONNumber gate is bitSize-independent, the same grammar check the
// int/long arms and goavro's numberLength apply, so a trailing-dot "5." is
// rejected uniformly.
//
// Input routes through [truncForError] so a 1 MiB hostile literal cannot
// produce a 1 MiB error string.
func parseJSONNumberAsFloat(s string, bitSize int) (float64, error) {
	if !isJSONNumber(s) {
		return 0, fmt.Errorf("invalid JSON number %q", truncForError(s))
	}
	f, err := parseFloatAcceptOverflow(s, bitSize)
	if err != nil {
		return 0, fmt.Errorf("invalid JSON number %q: %w", truncForError(s), err)
	}
	return f, nil
}

// truncForError caps a user-controllable string at 80 chars for inclusion
// in error messages, preventing 1 MiB hostile inputs from producing 1 MiB
// error strings. Mirrors the maxParseFloatLen DoS posture at the message
// layer.
func truncForError(s string) string {
	const max = 80
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// truncRatForError renders r for an error message without materializing a
// huge decimal string. big.Rat.RatString on a megabit-scale rational builds
// a multi-megabyte string (superlinear base conversion) before truncForError
// could trim it — a CPU/alloc amplification on hostile or pathological
// big-decimal input. When either component is too large to format cheaply,
// report bit sizes instead of the value. 512 bits ≈ 154 decimal digits each,
// comfortably above anything truncForError keeps but cheap to stringify.
func truncRatForError(r *big.Rat) string {
	if r.Num().BitLen() <= 512 && r.Denom().BitLen() <= 512 {
		return truncForError(r.RatString())
	}
	return fmt.Sprintf("(num %d bits / denom %d bits)", r.Num().BitLen(), r.Denom().BitLen())
}

// truncBytesForError caps a user-controllable byte slice at 40 chars
// before string conversion for error interpolation. 40 chars fits a
// MaxInt64 representation (20 chars) and a canonical hex-dash UUID
// (36 chars) with headroom, while keeping the error message bounded
// on hostile multi-MB inputs. Lower than [truncForError]'s 80-char
// cap because every caller of truncBytesForError today (parseJSONInt32 /
// parseJSONInt64 in json_scan.go, parseUUIDBytes in deser.go) operates
// on a fixed-format value whose useful diagnostic prefix fits in
// 40 chars; truncForError's 80-char cap is sized for arbitrary
// string defaults / decimal literals where the useful prefix is wider.
func truncBytesForError(b []byte) string {
	const max = 40
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// truncValueForError returns a "%v"-style string representation of v
// bounded by [truncForError]. Use when interpolating a user-controllable
// arbitrary-typed default value into an error message (the `%T(%v)` shape
// at walkDefault's union arm and encodeDefault's union arm). For string /
// []byte / json.Number inputs — the common ways user-controllable bytes
// reach a default — the input is truncated WITHOUT first allocating the
// unbounded "%v" representation. Other types format via fmt.Sprintf and
// then truncate; container types (map / slice) are still bounded by
// schema-parse-time JSON validation upstream so the intermediate
// allocation is bounded by the on-the-wire JSON size.
func truncValueForError(v any) string {
	switch tv := v.(type) {
	case string:
		return truncForError(tv)
	case []byte:
		return truncBytesForError(tv)
	case json.Number:
		return truncForError(string(tv))
	}
	return truncForError(fmt.Sprintf("%v", v))
}

// parseInt64Lenient parses s as a decimal integer, accepting pure-integer,
// exponent ("1e3"), and zero-fractional-part ("1.0", "1.5e1"=15) forms.
// Rejects invalid grammar, out-of-int64 values, non-zero fractional parts,
// and exponents beyond decimalScaleLimit (DoS bound).
//
// The slow path uses [boundedRatFromString], not
// strconv.ParseFloat+[floatFitsInt64]: float64 cannot distinguish int64.Min
// from int64.Min-1024 and rounds valid exponent-form int64s across the
// boundary. Java's BigDecimal and fastavro's long64 check are also
// arbitrary-precision.
//
// Shared by [jsonNumberToInt64], [defaultAsInt64], [jsonCoerceToInt64],
// and [parseJSONInt64]'s exponent/fractional branch.
func parseInt64Lenient(s string) (int64, error) {
	// Length cap at the entry. Bounds every downstream walk
	// (isJSONNumber, strconv.ParseInt, boundedRatFromString) in O(1)
	// before any per-byte work happens; also bounds the size of any
	// error message that echoes the input. Legit int64 inputs (decimal
	// max 20 chars; exponent form max ~24 chars) fit easily under the
	// cap.
	if len(s) > maxInt64LenientLen {
		return 0, fmt.Errorf("integer literal exceeds %d-byte length cap", maxInt64LenientLen)
	}
	// JSON-spec grammar gate: strconv.ParseInt(s,10,64) accepts forms
	// the JSON spec rejects — leading '+' ("+5" → 5), leading-zero
	// multi-digit ("01" → 1). Validate first so the fast path agrees
	// with the slow path on grammar (boundedRatFromString applies the
	// same gate). Java's JsonParser rejects "+5"/"01" at JSON parse;
	// fastavro's JSON layer is Python's json, whose grammar rejects
	// both too (json.loads("+5") "Expecting value", json.loads("01")
	// "Extra data" — observed on 3.14; Python's int() itself is
	// lenient and accepts both, but never sees them). Both match
	// strict JSON.
	if !isJSONNumber(s) {
		return 0, fmt.Errorf("invalid JSON number %q", s)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		return n, nil
	}
	// ParseInt failed. For pure-integer-form overflow (ErrRange, no .eE),
	// reject directly — value is definitionally outside int64 range and
	// the boundedRatFromString fallback would only confirm what ErrRange
	// already proved. ErrSyntax (anything else: exponent/fractional/
	// non-numeric) falls through to the arbitrary-precision path for a
	// precise IsInt+IsInt64 check and an accurate error message.
	if errors.Is(err, strconv.ErrRange) && !strings.ContainsAny(s, ".eE") {
		return 0, fmt.Errorf("integer literal overflows int64: %q", s)
	}
	r, ok, perr := boundedRatFromString(s)
	if perr != nil {
		return 0, perr
	}
	if !ok {
		return 0, fmt.Errorf("invalid number %s", s)
	}
	if !r.IsInt() {
		return 0, fmt.Errorf("value %s is not a whole number", s)
	}
	bi := r.Num()
	if !bi.IsInt64() {
		return 0, fmt.Errorf("value %s overflows int64", s)
	}
	return bi.Int64(), nil
}

// maxInt64LenientLen caps the input length parseInt64Lenient accepts.
// Fires at the entry of parseInt64Lenient so every downstream walk
// (isJSONNumber, strconv.ParseInt for pure-integer overflow,
// boundedRatFromString for slow-path exponent/fractional) bounds in O(1)
// instead of O(n). Also bounds the size of error messages that echo
// the input. The longest legit int64 input in exponent form
// ("-9.223372036854775808e18" = 24 chars) plus generous padding fits
// in 64.
const maxInt64LenientLen = 64

// parseInt32Lenient is [parseInt64Lenient] with int32 range narrowing.
// Shares the same arbitrary-precision parsing so fractional-part-lost-
// to-float64-rounding inputs like "1.0000000000000001" are correctly
// rejected as non-whole rather than silently truncating to 1 via float64.
// Used by [defaultAsInt32] (schema int default validate) and
// [jsonCoerceToInt32] (JSON encode of json.Number against int). The
// pure-integer-form fast path goes through strconv.ParseInt → int64 →
// int32 narrowing; only fractional/exponent form pays the big.Rat cost.
func parseInt32Lenient(s string) (int32, error) {
	n, err := parseInt64Lenient(s)
	if err != nil {
		return 0, err
	}
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("value %s overflows int32", s)
	}
	return int32(n), nil
}

// jsonNumberToInt64 converts a json.Number reflect.Value to a validated int64,
// checking that the value is a whole number within int64 range. Routes
// through [parseInt64Lenient] for precision-preserving parsing. The returned
// error is bare; callers wrap it in their SemanticError.
func jsonNumberToInt64(v reflect.Value) (int64, bool, error) {
	if v.Type() != jsonNumberType {
		return 0, false, nil
	}
	n, err := parseInt64Lenient(string(v.Interface().(json.Number)))
	if err != nil {
		return 0, true, err
	}
	return n, true, nil
}

// appendAvroInt appends v as an Avro int (zigzag varint, narrowed to int32).
// Accepts reflect.Int*/Uint*/Float*/json.Number with overflow + precision
// checks per type. Direct-call helper — compiler inlines at every site.
func appendAvroInt(dst []byte, v reflect.Value) ([]byte, error) {
	if v.CanInt() {
		n := v.Int()
		if n < math.MinInt32 || n > math.MaxInt32 {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "int", Err: fmt.Errorf("value %d overflows int32", n)}
		}
		return appendVarint(dst, int32(n)), nil
	}
	if v.CanUint() {
		n := v.Uint()
		if n > math.MaxInt32 {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "int", Err: fmt.Errorf("value %d overflows int32", n)}
		}
		return appendVarint(dst, int32(n)), nil
	}
	if v.CanFloat() {
		n, err := floatFitsInt32From(v.Float(), v.Type().Bits())
		if err != nil {
			return nil, semErrW(v, "int", err)
		}
		return appendVarint(dst, n), nil
	}
	if n, ok, err := jsonNumberToInt64(v); ok {
		if err != nil {
			return nil, semErrW(v, "int", err)
		}
		if n < math.MinInt32 || n > math.MaxInt32 {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "int", Err: fmt.Errorf("value %d overflows int32", n)}
		}
		return appendVarint(dst, int32(n)), nil
	}
	return nil, semErr(v, "int")
}

// appendAvroLong is appendAvroInt with the int64 bound + varlong emit.
func appendAvroLong(dst []byte, v reflect.Value) ([]byte, error) {
	if v.CanInt() {
		return appendVarlong(dst, v.Int()), nil
	}
	if v.CanUint() {
		n := v.Uint()
		if n > math.MaxInt64 {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "long", Err: fmt.Errorf("value %d overflows int64", n)}
		}
		return appendVarlong(dst, int64(n)), nil
	}
	if v.CanFloat() {
		n, err := floatFitsInt64From(v.Float(), v.Type().Bits())
		if err != nil {
			return nil, semErrW(v, "long", err)
		}
		return appendVarlong(dst, n), nil
	}
	if n, ok, err := jsonNumberToInt64(v); ok {
		if err != nil {
			return nil, semErrW(v, "long", err)
		}
		return appendVarlong(dst, n), nil
	}
	return nil, semErr(v, "long")
}

var (
	serInt    = serPrim(appendAvroInt)
	serLong   = serPrim(appendAvroLong)
	serFloat  = serPrim(appendAvroFloat32)
	serDouble = serPrim(appendAvroFloat64)
)

// finiteFloat32Overflows reports whether f is a finite float64 whose
// float32(f) narrowing is ±Inf. ±Inf and NaN inputs return false: those
// have valid float32 forms and shouldn't be rejected by callers that
// otherwise accept finite-only. Encode-side narrowing accepts ±Inf
// silently per the lossy-destination policy; this helper is used only
// on the decode side to surface precision loss when the user picked a
// smaller Go target (deserDouble setFloatValue with Float32 target,
// udDouble Float32 target).
func finiteFloat32Overflows(f float64) bool {
	return !math.IsInf(f, 0) && !math.IsNaN(f) && math.IsInf(float64(float32(f)), 0)
}

// float32WireBits returns f's exact 32-bit pattern, matching Java's
// Float.floatToRawIntBits and the unsafe path (usFloat). reflect.Value.Float()
// would widen to float64 and narrow back, quieting signaling-NaN payloads;
// this avoids that detour so float32 encodes identically on every path. v
// must be Kind Float32.
func float32WireBits(v reflect.Value) uint32 {
	// Fast path: float32→float64→float32 (reflect.Value.Float() then narrow)
	// is bit-exact for every non-NaN value and for ±Inf, so its bits equal
	// the raw bits — and it's several times cheaper than reading raw. Only a
	// NaN survives the round-trip differently (signaling NaNs get quieted),
	// so only a NaN needs the raw read to preserve its payload (match Java
	// floatToRawIntBits). This keeps normal float32 encoding regression-free
	// while still preserving sNaN.
	f := float32(v.Float())
	if f == f { // not NaN
		return math.Float32bits(f)
	}
	if v.CanAddr() {
		return *(*uint32)(unsafe.Pointer(v.UnsafeAddr()))
	}
	if v.Type() == float32Type {
		// Non-addressable builtin float32 (e.g. Encode(f32)): Interface()
		// preserves the bits and is alloc-free here (the box is unpacked
		// immediately, so it stays on the stack).
		return math.Float32bits(v.Interface().(float32))
	}
	// Named float32, non-addressable (rare): bit-copy into an addressable
	// temp via Set (a typedmemmove, not a numeric conversion), then read raw.
	tmp := reflect.New(v.Type()).Elem()
	tmp.Set(v)
	return *(*uint32)(unsafe.Pointer(tmp.UnsafeAddr()))
}

// appendAvroFloat32 appends v's Avro-float (4-byte) encoding to dst.
// Encoding into a float schema is lossy by destination: int/uint inputs
// exceeding float32's 24-bit mantissa silently IEEE-round, and finite
// float64 inputs that overflow float32's range silently narrow to ±Inf.
// Matches Java's GenericDatumWriter (Number.floatValue()) and fastavro
// (struct.pack("<f", v)). Users wanting precise round-trip for large
// integers should use "long", not "float".
//
// Used by serFloat (top-level), serArray.serFloat / serMap.serFloat
// (specialized container paths), and any other site that encodes a
// reflect-typed value as Avro float.
func appendAvroFloat32(dst []byte, v reflect.Value) ([]byte, error) {
	if v.Kind() == reflect.Float32 {
		// Same-width: emit exact bits (preserve sNaN), matching Java + unsafe.
		return appendUint32(dst, float32WireBits(v)), nil
	}
	if v.CanFloat() {
		// float64 source → genuine narrowing to float32 (lossy by design).
		return appendUint32(dst, math.Float32bits(float32(v.Float()))), nil
	}
	if v.CanInt() {
		return appendUint32(dst, math.Float32bits(float32(v.Int()))), nil
	}
	if v.CanUint() {
		return appendUint32(dst, math.Float32bits(float32(v.Uint()))), nil
	}
	if fv, ok, err := jsonNumberToFloat(v); ok {
		if err != nil {
			return nil, semErrW(v, "float", err)
		}
		return appendAvroFloat32(dst, fv)
	}
	return nil, semErr(v, "float")
}

// appendAvroFloat64 is the parallel helper for Avro double. Same lossy-
// destination policy: int/uint inputs exceeding float64's 53-bit mantissa
// silently IEEE-round.
func appendAvroFloat64(dst []byte, v reflect.Value) ([]byte, error) {
	if v.CanFloat() {
		return appendUint64(dst, math.Float64bits(v.Float())), nil
	}
	if v.CanInt() {
		return appendUint64(dst, math.Float64bits(float64(v.Int()))), nil
	}
	if v.CanUint() {
		return appendUint64(dst, math.Float64bits(float64(v.Uint()))), nil
	}
	if fv, ok, err := jsonNumberToFloat(v); ok {
		if err != nil {
			return nil, semErrW(v, "double", err)
		}
		return appendAvroFloat64(dst, fv)
	}
	return nil, semErr(v, "double")
}

// rejectJSONNumberRawTarget reports a SemanticError when v is a json.Number
// being encoded against a non-numeric Avro type (bytes, fixed, or enum).
// json.Number is a numeric carrier — its stdlib contract is an RFC 8259 number
// literal — so it is valid only for numeric Avro types, the same rule
// appendAvroString applies for the string type and rejectJSONNumberStringTarget
// applies on decode. Plain strings stay accepted at these sites for
// json.Unmarshal pipelines that carry Avro bytes/fixed/enum content as strings;
// only json.Number is turned away. Callers invoke this inside a
// v.Kind()==reflect.String branch.
func rejectJSONNumberRawTarget(v reflect.Value, avroType string) error {
	if v.Type() == jsonNumberType {
		return semErr(v, avroType)
	}
	return nil
}

func serBytes(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	// Accept plain strings for json.Unmarshal pipelines where JSON strings
	// may represent Avro bytes fields, but reject json.Number: it is a numeric
	// carrier (valid only for numeric Avro types), so a bytes target is a type
	// mismatch — symmetric with the decoder, which rejects a json.Number bytes
	// target. See rejectJSONNumberRawTarget.
	if v.Kind() == reflect.String {
		if err := rejectJSONNumberRawTarget(v, "bytes"); err != nil {
			return nil, err
		}
		return doSerString(dst, v.String()), nil
	}
	if (v.Kind() != reflect.Array && v.Kind() != reflect.Slice) || v.Type().Elem().Kind() != reflect.Uint8 {
		return nil, semErr(v, "bytes")
	}
	return doSerBytes(dst, v, depth), nil
}

var serString = serPrim(appendAvroString)

// appendAvroString appends v as an Avro string. The resolution order is
// the canonical contract for any Avro-string-typed encode site:
//
//  1. json.Number is rejected (Kind==String but numeric semantics; let
//     union dispatch route it to a numeric branch).
//  2. encoding.TextAppender (preferred over TextMarshaler when both
//     are implemented; appends directly into dst, saving one alloc).
//  3. encoding.TextMarshaler → MarshalText then write.
//  4. reflect.String → write the underlying string.
//  5. []byte slice → write bytes.
//  6. Anything else → SemanticError.
//
// Text interfaces come BEFORE the reflect.String fast path so a string-kind
// type implementing TextMarshaler uses its marshaled form, as encoding/json
// does. The JSON encoder's avroStringValue must stay in lockstep on this
// precedence; it exists separately only because JSON always materializes the
// string for escaping.
func appendAvroString(dst []byte, v reflect.Value) ([]byte, error) {
	// One Type() read serves both discriminators:
	// json.Number is rejected, and the builtin (unnamed) string — the common
	// case, which can carry no text-out method — is fast-pathed past the
	// textOutFor probe. Named string types fall through to the text-aware
	// arms below.
	t := v.Type()
	if t == jsonNumberType {
		return nil, semErr(v, "string")
	}
	if t == stringType {
		return doSerString(dst, v.String()), nil
	}
	if a, m := textOutFor(v); a != nil {
		// AppendText is preferred for the alloc-free inline write:
		// reserve a single-byte length placeholder, let AppendText
		// write directly into dst, then backfill the real header
		// (shifting text iff the header grew past 1 byte).
		mark := len(dst)
		dst = appendVarlong(dst, 0)
		hdrLen := len(dst) - mark
		var err error
		dst, err = a.AppendText(dst)
		if err != nil {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "string", Err: err}
		}
		// A contract-violating AppendText returning a slice SHORTER than its
		// input — typically `return []byte(s), nil`, a fresh slice instead of
		// an append — drives the backfill arithmetic below out of bounds and
		// panics through Encode. Name the violation instead.
		//
		// A fresh return at or above the input length is undetectable without
		// a per-string memcmp of everything encoded so far, which is not paid
		// for a caller's own contract violation, so it silently replaces
		// earlier output. encoding/json/v2's jsontext.AppendRaw is equally
		// trusting (executed, go1.26.2: identical panic on the short shape).
		// The undetectable shapes are pinned in
		// text_appender_contract_test.go.
		if len(dst) < mark+hdrLen {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "string", Err: errAppendTextShrunk}
		}
		textLen := len(dst) - mark - hdrLen
		var buf [10]byte
		hdr := appendVarlong(buf[:0], int64(textLen))
		if len(hdr) == hdrLen {
			copy(dst[mark:], hdr)
		} else {
			dst = append(dst, make([]byte, len(hdr)-hdrLen)...)
			copy(dst[mark+len(hdr):], dst[mark+hdrLen:mark+hdrLen+textLen])
			copy(dst[mark:], hdr)
		}
		return dst, nil
	} else if m != nil {
		text, err := m.MarshalText()
		if err != nil {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "string", Err: err}
		}
		return doSerString(dst, string(text)), nil
	}
	if v.Kind() == reflect.String {
		return doSerString(dst, v.String()), nil
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		// doSerString does `append(dst, s...)` and doesn't retain s, so
		// alias v.Bytes() instead of copying.
		b := v.Bytes()
		return doSerString(dst, unsafe.String(unsafe.SliceData(b), len(b))), nil
	}
	return nil, semErr(v, "string")
}

// avroStringValue resolves v to its canonical Avro-string textual form
// as a Go string. It is the JSON-encoder's counterpart to appendAvroString
// and must keep the same precedence (json.Number rejected; then
// encoding.TextAppender / encoding.TextMarshaler; then reflect.String;
// then []byte slice). The JSON encoder always materializes the string to
// apply JSON quoting/escapes, so the alloc-free TextAppender-into-buffer
// optimization in appendAvroString does not apply here.
func avroStringValue(v reflect.Value) (string, error) {
	// One Type() read for both discriminators (see appendAvroString).
	t := v.Type()
	if t == jsonNumberType {
		return "", semErr(v, "string")
	}
	if t == stringType {
		return v.String(), nil
	}
	if text, ok, err := textValue(v, "string"); err != nil {
		return "", err
	} else if ok {
		return text, nil
	}
	if v.Kind() == reflect.String {
		return v.String(), nil
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		return string(v.Bytes()), nil
	}
	return "", semErr(v, "string")
}

////////////////////
// STRING & BYTES //
////////////////////

func doSerBytes(dst []byte, v reflect.Value, _ int) []byte {
	l := v.Len()
	dst = appendVarlong(dst, int64(l))
	if l == 0 {
		return dst
	}
	if v.CanAddr() {
		return append(dst, v.Slice(0, l).Bytes()...)
	}
	for i := range l {
		dst = append(dst, byte(v.Index(i).Uint()))
	}
	return dst
}

func doSerString(dst []byte, s string) []byte {
	dst = appendVarlong(dst, int64(len(s)))
	return append(dst, s...)
}

/////////////
// COMPLEX //
/////////////

type serRecordField struct {
	name         string
	nameVal      reflect.Value // pre-computed reflect.ValueOf(name); avoids alloc per map lookup
	fn           serfn
	meta         *fieldMeta
	defaultBytes []byte // pre-encoded Avro binary for the field's default value
	hasDefault   bool
	// defaultErr defers a verdict about defaultBytes from PARSE to ENCODE.
	// A field default is pre-encoded once at parse, which is the wrong moment
	// to refuse it: a schema whose default cannot be WRITTEN is still a schema
	// a reader must be able to PARSE, because a reader that drops the field
	// never writes that default and decodes such data correctly. So the parse
	// records the reason here and every consumer of defaultBytes surfaces it
	// at the moment the default would actually reach the wire.
	defaultErr error
}

// avroType names the field's Avro type, or "" when the field carries no
// metadata to name it with.
//
// The type is asked of the metadata rather than copied beside it: the two
// were always written from the same expression, and a copy that only ever
// equals its source is a second thing to keep in step. Every dispatch below
// switches on this, so a field whose meta is missing — which only a
// hand-assembled field can be, since the build hands every field a fieldMeta
// — matches no arm and declines to the reflect path, which is what the
// per-arm nil checks used to spell out one arm at a time.
func (f *serRecordField) avroType() string {
	if f.meta == nil {
		return ""
	}
	return f.meta.avroType
}

// appendDefault appends the field's pre-encoded default, or returns the
// verdict recorded when it was built. One accessor rather than a check beside
// each append, so a later consumer of defaultBytes cannot pick up the bytes
// without the verdict that governs them.
func (f *serRecordField) appendDefault(dst []byte) ([]byte, error) {
	if f.defaultErr != nil {
		return nil, f.defaultErr
	}
	return append(dst, f.defaultBytes...), nil
}

// ozAction is what an omitzero-tagged field does when its Go value is zero (or
// IsZero() reports true). It is the SINGLE source of truth for the omitzero
// contract, shared by the binary, JSON, and unsafe encode paths so they cannot
// drift apart (the doc-vs-impl divergence that this consolidates). See doc.go's
// "# Struct tags".
type ozAction uint8

const (
	ozNoop    ozAction = iota // no-op: encode the zero value normally
	ozDefault                 // emit the field's schema default (== map default-fill)
	ozNull                    // emit the null branch (a nullable field with no default)
)

// omitzeroAction reports what omitzero does for a zero value of this field:
// fill the schema default if it has one, else null if the field is a nullable
// union, else nothing. This mirrors map[string]any default-fill, except a
// nullable field with no default yields null here rather than the "missing
// key" error map-fill raises (a zero value of a nullable field maps naturally
// to its null branch).
func (f *serRecordField) omitzeroAction() ozAction {
	switch {
	case f.hasDefault:
		return ozDefault
	case f.avroType() == "nullunion":
		return ozNull
	default:
		return ozNoop
	}
}

type serRecord struct {
	fields []serRecordField
	names  []string
	cache  sync.Map // map[reflect.Type]*cachedMapping
	fast   sync.Map // map[reflect.Type]*fastRecordSer — per-Go-type compiled unsafe path
}

// fastFor returns the compiled unsafe fast path for t, or nil if not
// yet compiled. Read-only; used by nested-record sites that don't
// trigger compilation themselves (the outer record's slow-path entry
// is responsible for that).
func (s *serRecord) fastFor(t reflect.Type) *fastRecordSer {
	if v, ok := s.fast.Load(t); ok {
		return v.(*fastRecordSer)
	}
	return nil
}

// loadOrCompileFast returns the compiled fast path for t, compiling
// and storing it on first call. Returns nil when compilation fails
// (e.g. typeFieldMapping rejects t); callers fall back to the reflect
// path. Multiple goroutines compiling the same type concurrently each
// build their own *fastRecordSer; LoadOrStore picks one winner so all
// callers end up with the same pointer.
func (s *serRecord) loadOrCompileFast(t reflect.Type) *fastRecordSer {
	if fast := s.fastFor(t); fast != nil {
		return fast
	}
	fast := compileFastSer(s.fields, s.names, &s.cache, t)
	if fast == nil {
		return nil
	}
	actual, _ := s.fast.LoadOrStore(t, fast)
	return actual.(*fastRecordSer)
}

func (s *serRecord) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	k := v.Kind()
	t := v.Type()
	if k != reflect.Struct && (k != reflect.Map || t.Key().Kind() != reflect.String) {
		return nil, &SemanticError{GoType: t, AvroType: "record"}
	}
	if k == reflect.Map {
		// map[string]any fast path: reflect.Value.MapIndex copies the value
		// through reflect.copyVal which allocates per field for interface{}
		// element maps. Direct map access skips that. Input keys must
		// match the schema's canonical field names — aliases are a
		// reader-side / decode concept, not relevant on encode (we are
		// the writer and our output uses our schema's canonical names).
		if t == mapStringAnyType {
			m := v.Interface().(map[string]any)
			for _, f := range s.fields {
				value, exists := m[f.name]
				if !exists {
					if !f.hasDefault {
						return nil, &SemanticError{GoType: t, AvroType: "record", Field: f.name, Err: errors.New("missing key")}
					}
					if dst, err = f.appendDefault(dst); err != nil {
						return nil, recordFieldError(t, f.name, err)
					}
					continue
				}
				// reflect.ValueOf(nil) returns the invalid zero Value,
				// which the field fn would have to special-case via
				// .IsValid() before any Type/Kind call. reflect.Zero(any)
				// produces a valid zero `any` Value that flows through
				// indirect()/serUnion's IsNil checks naturally — they
				// recognize a nil interface and route to the union's
				// null branch (or surface errIndirectNil on a non-union).
				var rv reflect.Value
				if value != nil {
					rv = reflect.ValueOf(value)
				} else {
					rv = reflect.Zero(anyType)
				}
				if dst, err = f.fn(dst, rv, depth+1); err != nil {
					return nil, recordFieldError(t, f.name, err)
				}
			}
			return dst, nil
		}
		keyType := t.Key()
		for _, f := range s.fields {
			if err := validateJSONNumberMapKey(f.name, keyType, "record"); err != nil {
				return nil, err
			}
			value := v.MapIndex(mapKeyAs(t, f.nameVal))
			if !value.IsValid() {
				if !f.hasDefault {
					return nil, &SemanticError{GoType: t, AvroType: "record", Field: f.name, Err: errors.New("missing key")}
				}
				if dst, err = f.appendDefault(dst); err != nil {
					return nil, recordFieldError(t, f.name, err)
				}
				continue
			}
			if dst, err = f.fn(dst, value, depth+1); err != nil {
				return nil, recordFieldError(t, f.name, err)
			}
		}
		return dst, nil
	}
	// Struct: try precompiled unsafe fast path. Requires addressable
	// value so we can take a pointer for unsafe field access.
	//
	// Dispatch at the SAME depth: serRecordFast is the fast body for this
	// one record node, not a nested level. serRecordFast passes its fields
	// at depth+1 exactly as the reflect path below does, so the record→
	// field edge costs one depth unit on both paths. Passing depth+1 here
	// would double-count the record node (once for the dispatch hop, once
	// for the field pass), halving the effective bound for struct-fast
	// records vs the reflect/map path and breaking depth uniformity.
	if v.CanAddr() {
		if fast := s.loadOrCompileFast(t); fast != nil {
			return serRecordFast(dst, fast, v, depth)
		}
	}
	// Slow path: reflect-based field access.
	mapping, err := typeFieldMapping(s.names, &s.cache, t)
	if err != nil {
		return nil, err
	}
	for i, f := range s.fields {
		fv := fieldByIndexZero(v, mapping.indices[i])
		// omitzero: a zero/IsZero value encodes as if the field were
		// absent — the schema default, else null for a nullable field,
		// else the zero itself (no-op). The decision is shared via
		// omitzeroAction so binary/JSON/unsafe agree. The null byte's
		// position depends on the union: ["null",T] → 0x00 (index 0),
		// ["T","null"] → 0x02 (zigzag index 1), from nullUnionBytes.
		if mapping.omitzero[i] && valueIsZero(fv) {
			switch f.omitzeroAction() {
			case ozDefault:
				if dst, err = f.appendDefault(dst); err != nil {
					return nil, recordFieldError(t, f.name, err)
				}
				continue
			case ozNull:
				nullByte, _ := nullUnionBytes(f.meta != nil && f.meta.nullSecond)
				dst = append(dst, nullByte)
				continue
			}
			// ozNoop: fall through to the normal field encoder.
		}
		if dst, err = f.fn(dst, fv, depth+1); err != nil {
			return nil, recordFieldError(t, f.name, err)
		}
	}
	return dst, nil
}

type serEnum struct {
	symbols []string
	// symbolIdx is the shared symbol → ordinal lookup; see enumSymbolIndex
	// for when it is nil. The enum's schemaNode holds the same map, so the
	// JSON arms resolve a symbol through this table too rather than
	// scanning the symbol slice once per value.
	symbolIdx map[string]int
}

// enumIndexMin is the symbol count above which an enum gets a lookup map.
// Below it the linear scan beats a map lookup and the scan's cost is capped
// by the constant itself, so both wires stay O(1) in the symbol count either
// way. ONE threshold, applied here and read by every consumer — a second
// spelling of it is how the two wires would come to disagree about which
// enums are cheap to search.
const enumIndexMin = 8

// enumSymbolIndex builds the symbol → ordinal lookup an enum's consumers
// share, or nil when the enum is small enough to scan. Built once at
// schema-build time to avoid lock-or-race choices on the hot path.
func enumSymbolIndex(symbols []string) map[string]int {
	if len(symbols) <= enumIndexMin {
		return nil
	}
	idx := make(map[string]int, len(symbols))
	for i, sym := range symbols {
		if _, dup := idx[sym]; !dup {
			idx[sym] = i
		}
	}
	return idx
}

// newSerEnum constructs a serEnum sharing idx with the enum's schemaNode.
func newSerEnum(symbols []string, idx map[string]int) *serEnum {
	return &serEnum{symbols: symbols, symbolIdx: idx}
}

// indexOfSymbol resolves a symbol name to its ordinal.
func (s *serEnum) indexOfSymbol(needle string) (int, bool) {
	return lookupEnumSymbol(s.symbols, s.symbolIdx, needle)
}

// lookupEnumSymbol resolves a symbol name to its ordinal against an enum's
// symbols and its (possibly nil) index. Both wires ask through here, so the
// binary encoder cannot be answering from a table while a JSON arm scans.
func lookupEnumSymbol(symbols []string, idx map[string]int, needle string) (int, bool) {
	if idx != nil {
		i, ok := idx[needle]
		return i, ok
	}
	for i, symbol := range symbols {
		if symbol == needle {
			return i, true
		}
	}
	return 0, false
}

// symbolIndex resolves a symbol name to its ordinal on an enum node.
func (n *schemaNode) symbolIndex(needle string) (int, bool) {
	return lookupEnumSymbol(n.symbols, n.symbolIdx, needle)
}

// enumOrdinalIndex validates an integer-kind enum carrier as an ordinal in
// [0, nSymbols) and returns it as an int. The range check is done in the
// carrier's OWN width (int64 / uint64) BEFORE narrowing to int — narrowing
// first (int(v.Uint()) / int(v.Int())) truncates a value ≥ 2^32 to its low
// bits on a 32-bit build, so an out-of-range ordinal like uint64(1<<32+5)
// would wrap to 5 and pass `n < len(symbols)`, silently encoding the wrong
// symbol (and diverging from the same program's 64-bit behavior). Comparing
// wide first rejects it on every platform. Shared by serEnum.ser (binary) and
// appendAvroJSON's enum case (JSON) so the bound and the truncation guard
// can't drift between the two encoders; each caller wraps the returned error
// in its own SemanticError / "avro json:" shape and does its own emit.
func enumOrdinalIndex(v reflect.Value, nSymbols int) (int, error) {
	if v.CanInt() {
		n := v.Int()
		if n < 0 || n >= int64(nSymbols) {
			return 0, fmt.Errorf("index %d out of range [0, %d)", n, nSymbols)
		}
		return int(n), nil
	}
	n := v.Uint()
	if n >= uint64(nSymbols) {
		return 0, fmt.Errorf("index %d out of range [0, %d)", n, nSymbols)
	}
	return int(n), nil
}

func (s *serEnum) ser(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	// Builtin string fast path: unnamed string can carry no text-out method,
	// so it IS the symbol — skip the textValue probe. (This is only a
	// shortcut for the provably-text-less builtin; named string types fall
	// through to textValue below, so uniformity holds — a named string with
	// MarshalText still uses it.)
	if v.Type() == stringType {
		needle := v.String()
		if i, ok := s.indexOfSymbol(needle); ok {
			return appendVarint(dst, int32(i)), nil
		}
		return nil, &SemanticError{GoType: v.Type(), AvroType: "enum", Err: fmt.Errorf("unknown symbol %q", truncForError(needle))}
	}
	// Text-out methods first (uniformity): a carrier's MarshalText / AppendText,
	// if any, names its symbol — robust to a Go int whose value doesn't match
	// the Avro symbol order (Java's getEnumOrdinal(datum.toString())). Named
	// string types without a text method, and plain ints, fall through.
	if needle, ok, err := textValue(v, "enum"); err != nil {
		return nil, err
	} else if ok {
		if i, idxOk := s.indexOfSymbol(needle); idxOk {
			return appendVarint(dst, int32(i)), nil
		}
		return nil, &SemanticError{GoType: v.Type(), AvroType: "enum", Err: fmt.Errorf("unknown symbol %q", truncForError(needle))}
	}
	if v.Kind() == reflect.String {
		// The jsonNumberType exclusion, at the enum gate: without it the
		// number's content is looked up as a symbol name and silently
		// encodes an ordinal the decoder (setEnumTarget → setStringTarget
		// → rejectJSONNumberStringTarget) can never read back.
		if err := rejectJSONNumberRawTarget(v, "enum"); err != nil {
			return nil, err
		}
		needle := v.String()
		if i, ok := s.indexOfSymbol(needle); ok {
			return appendVarint(dst, int32(i)), nil
		}
		return nil, &SemanticError{GoType: v.Type(), AvroType: "enum", Err: fmt.Errorf("unknown symbol %q", truncForError(needle))}
	}
	if v.CanInt() || v.CanUint() {
		n, err := enumOrdinalIndex(v, len(s.symbols))
		if err != nil {
			return nil, &SemanticError{GoType: v.Type(), AvroType: "enum", Err: err}
		}
		return appendVarint(dst, int32(n)), nil
	}
	return nil, semErr(v, "enum")
}

type serArray struct {
	serItem serfn
}

// arrayZeroByteEncodeCompliance enforces producer-side compliance with the
// decoder's maxZeroByteItems cap, shared by EVERY array encoder — the reflect
// serArray.ser and the unsafe usArrayRecord/usArrayPtrRecord/usArrayDirect.
// If the encoded body is empty, every item wrote zero bytes
// (array<null>/array<EmptyRecord>/array<size-0-fixed>), and the decoder
// (checkArrayBlockBounds) rejects a cumulative count above maxZeroByteItems —
// so a larger array would be a wire we cannot read back. Reject at encode
// instead of emitting a self-incompatible wire (the OCF shouldFlush
// discipline). Non-zero-byte items grow the buffer, so this fires only for
// genuinely zero-byte element types; the >=1-byte primitive fast paths never
// reach it. Every array encoder MUST route through this one helper so the
// reflect and unsafe paths cannot drift.
func arrayZeroByteEncodeCompliance(emptyBody bool, n int) error {
	if emptyBody && n > maxZeroByteItems {
		return &SemanticError{AvroType: "array", Err: fmt.Errorf(
			"array of %d zero-byte items exceeds the decoder's %d-element limit", n, maxZeroByteItems)}
	}
	return nil
}

func (s *serArray) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	dst, v, l, err := serArrayPreamble(dst, v)
	if err != nil || l == 0 {
		return dst, err
	}
	bodyStart := len(dst)
	for i := range l {
		if dst, err = s.serItem(dst, v.Index(i), depth+1); err != nil {
			return nil, err
		}
	}
	if err := arrayZeroByteEncodeCompliance(len(dst) == bodyStart, l); err != nil {
		return nil, err
	}
	return append(dst, 0), nil
}

// unwrapElemPtr unwraps a pointer/interface element for the
// serArray/serMap primitive specializations. A single-level element ([]*T /
// map[string]*T) is unwrapped inline; ≥2 levels dispatch to indirect().
// Callers inline the Kind gate at each site so direct elements pay nothing.
//
// The dispatch hands indirect the ORIGINAL element, not the once-peeled value:
// indirect's maxIndirectDepth budget must count EVERY level so a container
// element accepts exactly the cap the leaf encoder (serPrim → indirect) and
// every other context accept — no more. Peeling one level first and then
// giving indirect a fresh full budget would accept a chain one level deeper
// (1+maxIndirectDepth) than the wire's own reader can decode and than the JSON
// encoder accepts: an encode-produces-unreadable-wire / binary↔JSON-encode
// divergence. (Same single-peel discipline the union target keeps so it
// indirects once, not twice.)
func unwrapElemPtr(v reflect.Value) (reflect.Value, error) {
	if v.IsNil() {
		return v, errIndirectNil
	}
	if e := v.Elem(); e.Kind() != reflect.Interface && e.Kind() != reflect.Pointer {
		return e, nil
	}
	return indirect(v)
}

// peelElem unwraps one Pointer/Interface layer from an array/map element,
// tagging the unwrap error with avroType. Direct-call helper used by the
// primitive serArray/serMap specializations.
func peelElem(v reflect.Value, avroType string) (reflect.Value, error) {
	if v.Kind() != reflect.Interface && v.Kind() != reflect.Pointer {
		return v, nil
	}
	out, err := unwrapElemPtr(v)
	if err != nil {
		return v, &SemanticError{AvroType: avroType, Err: err}
	}
	return out, nil
}

// appendArrayPrimitive runs the shared (preamble + per-element peel +
// appendFn + terminator) sequence for the primitive serArray
// specializations. appendFn is a typed function-pointer parameter (NOT a
// closure capture), so the compiler emits one indirect call per element
// — matching the inlined direct-call shape.
func appendArrayPrimitive(
	dst []byte, v reflect.Value, avroType string,
	appendFn func([]byte, reflect.Value) ([]byte, error),
) ([]byte, error) {
	dst, v, l, err := serArrayPreamble(dst, v)
	if err != nil || l == 0 {
		return dst, err
	}
	// Hoist the per-element type dispatch out of the loop when the element type
	// is the exact natural Go type for this Avro primitive. The element type is
	// uniform across the slice, so this resolves once per encode rather than
	// per element, and each fast loop is a direct read+emit with no coercion or
	// bounds logic — the exact type provably fits the wire type. Named,
	// other-width, pointer, text and json.Number elements fall to the general
	// appendFn loop below, which applies the right coercion. Each case first
	// asserts the unnamed []V and ranges it natively; the hoist loop below each
	// assertion handles [N]T fixed arrays (where the []V assertion fails) and
	// non-interfaceable slices; a named element type misses the exact-type
	// case entirely and uses the general appendFn loop. float32 emits raw bits
	// (math.Float32bits(x) equals float32WireBits for every value: non-NaN
	// round-trips exactly, NaN is read raw — so it matches the reflect path).
	switch et := v.Type().Elem(); {
	case avroType == "string" && et == stringType:
		if v.CanInterface() {
			if s, ok := v.Interface().([]string); ok {
				for _, x := range s {
					dst = doSerString(dst, x)
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			dst = doSerString(dst, v.Index(i).String())
		}
		return append(dst, 0), nil
	case avroType == "boolean" && et == boolType:
		if v.CanInterface() {
			if s, ok := v.Interface().([]bool); ok {
				for _, x := range s {
					if x {
						dst = append(dst, 1)
					} else {
						dst = append(dst, 0)
					}
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			if v.Index(i).Bool() {
				dst = append(dst, 1)
			} else {
				dst = append(dst, 0)
			}
		}
		return append(dst, 0), nil
	case avroType == "int" && et == int32Type:
		if v.CanInterface() {
			if s, ok := v.Interface().([]int32); ok {
				for _, x := range s {
					dst = appendVarint(dst, x)
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			dst = appendVarint(dst, int32(v.Index(i).Int()))
		}
		return append(dst, 0), nil
	case avroType == "long" && (et == int64Type || et == intType):
		if v.CanInterface() {
			if s, ok := v.Interface().([]int64); ok {
				for _, x := range s {
					dst = appendVarlong(dst, x)
				}
				return append(dst, 0), nil
			}
			if s, ok := v.Interface().([]int); ok {
				for _, x := range s {
					dst = appendVarlong(dst, int64(x))
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			dst = appendVarlong(dst, v.Index(i).Int())
		}
		return append(dst, 0), nil
	case avroType == "float" && et == float32Type:
		if v.CanInterface() {
			if s, ok := v.Interface().([]float32); ok {
				for _, x := range s {
					dst = appendUint32(dst, math.Float32bits(x))
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			dst = appendUint32(dst, float32WireBits(v.Index(i)))
		}
		return append(dst, 0), nil
	case avroType == "double" && et == float64Type:
		if v.CanInterface() {
			if s, ok := v.Interface().([]float64); ok {
				for _, x := range s {
					dst = appendUint64(dst, math.Float64bits(x))
				}
				return append(dst, 0), nil
			}
		}
		for i := range l {
			dst = appendUint64(dst, math.Float64bits(v.Index(i).Float()))
		}
		return append(dst, 0), nil
	}
	for i := range l {
		elem, err := peelElem(v.Index(i), avroType)
		if err != nil {
			return nil, err
		}
		if dst, err = appendFn(dst, elem); err != nil {
			return nil, err
		}
	}
	return append(dst, 0), nil
}

// appendMapPrimitive encodes a map whose values are an Avro primitive.
//
// Fast path: an exactly-string key and an exactly-natural value type make the
// whole map a known concrete type, so it type-asserts and ranges natively with
// no reflect at all. Gated on CanInterface, since a map read from an unexported
// field is not interfaceable.
//
// Reflect fallback: non-string keys, named / other-width / pointer / text value
// types, and non-interfaceable maps. SetIterKey/SetIterValue reuse two
// addressable Values, costing 2 heap allocs per encode instead of the 2 per
// ENTRY that iter.Key()/iter.Value() would.
func appendMapPrimitive(
	dst []byte, v reflect.Value, avroType string,
	appendFn func([]byte, reflect.Value) ([]byte, error),
) ([]byte, error) {
	dst, v, l, err := serMapPreamble(dst, v)
	if err != nil || l == 0 {
		return dst, err
	}
	keyType := v.Type().Key()
	// Native concrete fast path: an exactly-string key plus an exact-natural
	// value type means the whole map has a known unnamed type, so assert it
	// and range natively — no reflect.MapRange, no per-entry Value. The
	// comma-ok assertion also rejects named map types (type M map[string]T,
	// whose Key/Elem match but whose dynamic type does not), which then take
	// the reflect path. A string key never needs json.Number validation.
	if keyType == stringType && v.CanInterface() {
		switch et := v.Type().Elem(); {
		case avroType == "string" && et == stringType:
			if m, ok := v.Interface().(map[string]string); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					dst = doSerString(dst, val)
				}
				return append(dst, 0), nil
			}
		case avroType == "int" && et == int32Type:
			if m, ok := v.Interface().(map[string]int32); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					dst = appendVarint(dst, val)
				}
				return append(dst, 0), nil
			}
		case avroType == "long" && et == int64Type:
			if m, ok := v.Interface().(map[string]int64); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					dst = appendVarlong(dst, val)
				}
				return append(dst, 0), nil
			}
		case avroType == "long" && et == intType:
			if m, ok := v.Interface().(map[string]int); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					dst = appendVarlong(dst, int64(val))
				}
				return append(dst, 0), nil
			}
		case avroType == "float" && et == float32Type:
			if m, ok := v.Interface().(map[string]float32); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					// Native float32: emit exact bits (preserve sNaN), matching
					// Java floatToRawIntBits, the unsafe path, and (now) the
					// reflect path via float32WireBits.
					dst = appendUint32(dst, math.Float32bits(val))
				}
				return append(dst, 0), nil
			}
		case avroType == "double" && et == float64Type:
			if m, ok := v.Interface().(map[string]float64); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					dst = appendUint64(dst, math.Float64bits(val))
				}
				return append(dst, 0), nil
			}
		case avroType == "boolean" && et == boolType:
			if m, ok := v.Interface().(map[string]bool); ok {
				for k, val := range m {
					dst = doSerString(dst, k)
					if val {
						dst = append(dst, 1)
					} else {
						dst = append(dst, 0)
					}
				}
				return append(dst, 0), nil
			}
		}
	}
	// Reflect fallback: non-string keys, named / other-width / pointer / text
	// value types, named map types, and non-interfaceable maps. SetIterKey/
	// SetIterValue reuse two addressable Values so iteration costs 2 heap
	// allocs per encode, not 2 per entry (iter.Key()/iter.Value()).
	keyNeedsJSONNumberCheck := keyType == jsonNumberType
	keyV := reflect.New(keyType).Elem()
	valV := reflect.New(v.Type().Elem()).Elem()
	iter := v.MapRange()
	for iter.Next() {
		keyV.SetIterKey(iter)
		key := keyV.String()
		if keyNeedsJSONNumberCheck {
			if err := validateJSONNumberMapKey(key, keyType, "map"); err != nil {
				return nil, err
			}
		}
		dst = doSerString(dst, key)
		valV.SetIterValue(iter)
		elem, err := peelElem(valV, avroType)
		if err != nil {
			return nil, err
		}
		if dst, err = appendFn(dst, elem); err != nil {
			return nil, err
		}
	}
	return append(dst, 0), nil
}

// serArrayPreamble handles the shared preamble for all serArray methods:
// indirect, kind check, length encoding, and empty-return. Called once
// per encode — no performance impact.
func serArrayPreamble(dst []byte, v reflect.Value) ([]byte, reflect.Value, int, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, v, 0, err
	}
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		return nil, v, 0, semErr(v, "array")
	}
	l := v.Len()
	dst = appendVarlong(dst, int64(l))
	return dst, v, l, nil
}

// The following serArray methods serialize array items by encoding
// primitive values directly from v.Index(i), avoiding reflect.Value
// escapes through serfn function pointers. Each is selected at schema
// build time based on the array's item type.
//
// Factoring constraint: appendArrayPrimitive must take the appendFn
// as a typed function-pointer parameter, with each per-method site
// passing a direct symbol (appendAvroInt, appendAvroLong, …). Two
// alternative factorings regress benchstat on BenchmarkLargeArrayEncode
// + BenchmarkMapEncode: a closure-based factory forces element escape
// (~25%); a generic-with-empty-struct GCShape dispatch adds a runtime
// dictionary lookup (+34-62%). The direct-symbol indirect call matches
// the inlined per-method call shape.

func (s *serArray) serString(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "string", appendAvroString)
}
func (s *serArray) serBoolean(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "boolean", appendAvroBool)
}
func (s *serArray) serInt(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "int", appendAvroInt)
}
func (s *serArray) serLong(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "long", appendAvroLong)
}
func (s *serArray) serFloat(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "float", appendAvroFloat32)
}
func (s *serArray) serDouble(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendArrayPrimitive(dst, v, "double", appendAvroFloat64)
}

type serMap struct {
	serItem serfn
}

func (s *serMap) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	if depth >= maxDepth {
		return nil, errTooDeep
	}
	dst, v, l, err := serMapPreamble(dst, v)
	if err != nil || l == 0 {
		return dst, err
	}
	keyType := v.Type().Key()
	// Reused addressable Values: see appendMapPrimitive. valV is addressable
	// (iter.Value() is not), so a struct-valued map now reaches serRecord's
	// unsafe fast path — byte-identical to the reflect path, just faster.
	keyV := reflect.New(keyType).Elem()
	valV := reflect.New(v.Type().Elem()).Elem()
	iter := v.MapRange()
	for iter.Next() {
		keyV.SetIterKey(iter)
		key := keyV.String()
		if err := validateJSONNumberMapKey(key, keyType, "map"); err != nil {
			return nil, err
		}
		dst = doSerString(dst, key)
		valV.SetIterValue(iter)
		if dst, err = s.serItem(dst, valV, depth+1); err != nil {
			return nil, err
		}
	}
	return append(dst, 0), nil
}

// serMapPreamble handles the shared preamble for all serMap methods:
// indirect, map+key check, length encoding, and empty-return. Called
// once per encode — no performance impact.
func serMapPreamble(dst []byte, v reflect.Value) ([]byte, reflect.Value, int, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, v, 0, err
	}
	t := v.Type()
	if t.Kind() != reflect.Map || t.Key().Kind() != reflect.String {
		return nil, v, 0, &SemanticError{GoType: t, AvroType: "map"}
	}
	l := v.Len()
	dst = appendVarlong(dst, int64(l))
	return dst, v, l, nil
}

// The following serMap methods serialize map values by extracting
// primitive values directly from iter.Value(), avoiding reflect.Value
// escapes through serfn function pointers. Each is selected at schema
// build time based on the map's value type. See serArray.serString for
// the factoring history.

func (s *serMap) serString(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "string", appendAvroString)
}
func (s *serMap) serBoolean(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "boolean", appendAvroBool)
}
func (s *serMap) serInt(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "int", appendAvroInt)
}
func (s *serMap) serLong(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "long", appendAvroLong)
}
func (s *serMap) serFloat(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "float", appendAvroFloat32)
}
func (s *serMap) serDouble(dst []byte, v reflect.Value, _ int) ([]byte, error) {
	return appendMapPrimitive(dst, v, "double", appendAvroFloat64)
}

type serSize struct {
	n int
}

func (s *serSize) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	t := v.Type()
	// Accept [N]byte arrays, []byte slices, and plain strings of the correct
	// length (json.Unmarshal pipelines). json.Number (Kind=reflect.String) is
	// rejected: it is a numeric carrier, valid only for numeric Avro types,
	// symmetric with the decoder which rejects a json.Number fixed target.
	switch t.Kind() {
	case reflect.Array:
		if t.Elem().Kind() != reflect.Uint8 || t.Len() != s.n {
			return nil, &SemanticError{GoType: t, AvroType: "fixed"}
		}
	case reflect.Slice:
		if t.Elem().Kind() != reflect.Uint8 || v.Len() != s.n {
			return nil, &SemanticError{GoType: t, AvroType: "fixed"}
		}
		return append(dst, v.Bytes()...), nil
	case reflect.String:
		if err := rejectJSONNumberRawTarget(v, "fixed"); err != nil {
			return nil, err
		}
		str := v.String()
		if len(str) != s.n {
			return nil, &SemanticError{GoType: t, AvroType: "fixed"}
		}
		return append(dst, str...), nil
	default:
		return nil, &SemanticError{GoType: t, AvroType: "fixed"}
	}
	// Fixed is written as raw bytes with no length prefix.
	if v.CanAddr() {
		return append(dst, v.Slice(0, s.n).Bytes()...), nil
	}
	for i := 0; i < s.n; i++ {
		dst = append(dst, byte(v.Index(i).Uint()))
	}
	return dst, nil
}

/////////////////////////////
// LOGICAL TYPE SERIALIZERS //
/////////////////////////////

var (
	timeType         = reflect.TypeFor[time.Time]()
	durationType     = reflect.TypeFor[time.Duration]()
	avroDurationType = reflect.TypeFor[Duration]()
	bigRatType       = reflect.TypeFor[big.Rat]()
)

// Duration represents the Avro duration logical type: a 12-byte fixed
// value containing three little-endian unsigned 32-bit integers
// representing months, days, and milliseconds.
type Duration struct {
	Months       uint32
	Days         uint32
	Milliseconds uint32
}

// Bytes encodes the Duration as a 12-byte little-endian fixed value,
// matching the Avro duration wire format.
func (d Duration) Bytes() [12]byte {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[0:4], d.Months)
	binary.LittleEndian.PutUint32(b[4:8], d.Days)
	binary.LittleEndian.PutUint32(b[8:12], d.Milliseconds)
	return b
}

// DurationFromBytes decodes a 12-byte little-endian fixed value into a
// Duration. Returns zero Duration if b is shorter than 12 bytes. This is
// useful in [CustomType] Decode callbacks that override the default duration
// handling: the callback receives raw []byte and can use this function to
// interpret the value before converting to a custom Go type.
func DurationFromBytes(b []byte) Duration {
	if len(b) < 12 {
		return Duration{}
	}
	return Duration{
		Months:       binary.LittleEndian.Uint32(b[0:4]),
		Days:         binary.LittleEndian.Uint32(b[4:8]),
		Milliseconds: binary.LittleEndian.Uint32(b[8:12]),
	}
}

// String returns an ISO 8601 duration string. Zero components are omitted.
// Examples: "P1Y3M15DT1H30M0.500S", "P30D", "PT1H".
func (d Duration) String() string {
	if d.Months == 0 && d.Days == 0 && d.Milliseconds == 0 {
		return "P0D"
	}
	buf := []byte{'P'}
	if y := d.Months / 12; y > 0 {
		buf = append(buf, fmt.Sprintf("%dY", y)...)
	}
	if m := d.Months % 12; m > 0 {
		buf = append(buf, fmt.Sprintf("%dM", m)...)
	}
	if d.Days > 0 {
		buf = append(buf, fmt.Sprintf("%dD", d.Days)...)
	}
	if d.Milliseconds > 0 {
		ms := d.Milliseconds
		h := ms / 3600000
		ms %= 3600000
		m := ms / 60000
		ms %= 60000
		s := ms / 1000
		frac := ms % 1000
		buf = append(buf, 'T')
		if h > 0 {
			buf = append(buf, fmt.Sprintf("%dH", h)...)
		}
		if m > 0 {
			buf = append(buf, fmt.Sprintf("%dM", m)...)
		}
		if frac > 0 {
			buf = append(buf, fmt.Sprintf("%d.%03dS", s, frac)...)
		} else if s > 0 {
			buf = append(buf, fmt.Sprintf("%dS", s)...)
		}
	}
	return string(buf)
}

// tryParseTimeString attempts to parse a string value as RFC 3339.
//
// The jsonNumberType exclusion, at the timestamp probe: a json.Number must
// fall through to the numeric encode arm (serLong / jsonCoerceToInt64),
// which validates its content as a number, rather than being reinterpreted
// here as a timestamp string. Mirrors the decode side
// (formatToStringKindTarget).
func tryParseTimeString(v reflect.Value) (time.Time, bool) {
	if v.Kind() != reflect.String || v.Type() == jsonNumberType {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, v.String())
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// extractTime returns v's time.Time content when v is a time.Time
// directly or a string parseable via RFC 3339. Mirrors the
// timeType-arm/tryParseTimeString-arm pattern used at every
// time-logical encode site (binary ser + JSON ser) so a future change
// to the accepted-input set lands in one place.
func extractTime(v reflect.Value) (time.Time, bool) {
	if v.Type() == timeType {
		return v.Interface().(time.Time), true
	}
	return tryParseTimeString(v)
}

// tryParseDateString attempts to parse a string value as either RFC 3339 or
// ISO 8601 date-only ("2006-01-02"). json.Number is excluded for the same
// reason as [tryParseTimeString]: a numeric carrier must fall through to the
// numeric encode arm, not be reinterpreted as a date string.
func tryParseDateString(v reflect.Value) (time.Time, bool) {
	if v.Kind() != reflect.String || v.Type() == jsonNumberType {
		return time.Time{}, false
	}
	s := v.String()
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.DateOnly, s)
	}
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// serTimeAsLong is the shared body of the six time-logical "long"
// serializers (timestamp / local-timestamp at millis, micros, nanos).
// Each per-logical wrapper passes the corresponding timeTo<Logical>
// converter. The pattern mirrors deserTimeAsLong on the decode side.
func serTimeAsLong(dst []byte, v reflect.Value, depth int, conv func(time.Time) (int64, error)) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if t, ok := extractTime(v); ok {
		n, err := conv(t)
		if err != nil {
			return nil, semErrW(v, "long", err)
		}
		return appendVarlong(dst, n), nil
	}
	return serLong(dst, v, depth)
}

func serTimestampMillis(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToTimestampMillis)
}

func serTimestampMicros(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToTimestampMicros)
}

func serTimestampNanos(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToTimestampNanos)
}

// Local-timestamp serializers encode wall-clock fields as if UTC, matching
// Java's TimeConversions.LocalTimestampMillisConversion (toInstant(ZoneOffset.UTC))
// and fastavro. See timeToLocalTimestampMillis in logical.go for rationale.

func serLocalTimestampMillis(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToLocalTimestampMillis)
}

func serLocalTimestampMicros(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToLocalTimestampMicros)
}

func serLocalTimestampNanos(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	return serTimeAsLong(dst, v, depth, timeToLocalTimestampNanos)
}

func serDate(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if v.Type() == timeType {
		d, err := timeToDate(v.Interface().(time.Time))
		if err != nil {
			return nil, &SemanticError{GoType: timeType, AvroType: "date", Err: err}
		}
		return appendVarint(dst, d), nil
	}
	if t, ok := tryParseDateString(v); ok {
		d, err := timeToDate(t)
		if err != nil {
			return nil, semErrW(v, "date", err)
		}
		return appendVarint(dst, d), nil
	}
	return serInt(dst, v, depth)
}

// serTimeMillis encodes a time-millis (time-of-day milliseconds) value.
// Accepts time.Duration (canonical) and time.Time as a convenience
// escape hatch; the time.Time arm silently discards the date and zone
// since the wire format physically can't represent them. Documented
// in README §Logical Types. time.Duration round-trips exactly only when
// its nanosecond component is a whole multiple of one millisecond;
// sub-millisecond nanoseconds are silently truncated toward zero (integer
// division by 1ms, dropping the remainder) since the wire format physically
// can't represent them. time.Time round-trips preserve time-of-day only
// (date and zone are dropped).
func serTimeMillis(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if v.Type() == durationType {
		ms, err := durationToTimeMillis(time.Duration(v.Int()))
		if err != nil {
			return nil, &SemanticError{GoType: durationType, AvroType: "time-millis", Err: err}
		}
		return appendVarint(dst, ms), nil
	}
	if v.Type() == timeType {
		ms, err := durationToTimeMillis(timeOfDay(v.Interface().(time.Time)))
		if err != nil {
			return nil, &SemanticError{GoType: timeType, AvroType: "time-millis", Err: err}
		}
		return appendVarint(dst, ms), nil
	}
	return serInt(dst, v, depth)
}

// serTimeMicros mirrors serTimeMillis at microsecond resolution —
// see that function's doc for the time.Time-escape-hatch lossiness.
func serTimeMicros(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if v.Type() == durationType {
		return appendVarlong(dst, time.Duration(v.Int()).Microseconds()), nil
	}
	if v.Type() == timeType {
		return appendVarlong(dst, timeOfDay(v.Interface().(time.Time)).Microseconds()), nil
	}
	return serLong(dst, v, depth)
}

func serDuration(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if v.Type() == avroDurationType {
		b := v.Interface().(Duration).Bytes()
		return append(dst, b[:]...), nil
	}
	return (&serSize{12}).ser(dst, v, depth)
}

// coerceDecimalRat is decimalRatFor with the indirect + SemanticError-
// wrap preamble factored out. Returns (peeled v, rat, ok, err):
//   - err != nil: caller surfaces it (indirect-nil or a wrapped tryCoerceToRat
//     failure that names avroType)
//   - ok == true: caller calls its serRat helper with rat
//   - ok == false, err == nil: caller falls through to its bytes/fixed
//     opaque-bytes path
//
// Shared by serBytesDecimal/serFixedDecimal/serBigDecimal so the three
// agree on the indirect / bigRat-fast-path / tryCoerceToRat / err-wrap
// chain. Returning the peeled v lets serBigDecimal pass v.Type() into
// its serRat for SemanticError context.
func coerceDecimalRat(v reflect.Value, avroType string) (reflect.Value, *big.Rat, bool, error) {
	v, err := indirect(v)
	if err != nil {
		return v, nil, false, err
	}
	r, ok, err := decimalRatFor(v)
	if err != nil {
		return v, nil, false, &SemanticError{GoType: v.Type(), AvroType: avroType, Err: err}
	}
	return v, r, ok, nil
}

// decimalRatFor extracts a *big.Rat from v for decimal-logical-type
// encoding. Tries the direct big.Rat type first (the canonical input),
// then falls through to tryCoerceToRat which handles float, json.Number,
// and numeric strings. Used by both Avro-binary and Avro-JSON encoders
// so the accepted-input set stays in lockstep across the two paths.
//
// Three-valued return: (rat, true, nil) on success; (nil, false, nil)
// when v was not a recognized number-form (caller may fall back to
// raw-bytes encoding paths); (nil, false, err) when v was clearly a
// number-form but rejected for safety (e.g. bounded exponent) — caller
// must propagate err rather than fall through, so a hostile
// json.Number isn't silently re-encoded as raw bytes.
func decimalRatFor(v reflect.Value) (*big.Rat, bool, error) {
	if v.Type() == bigRatType {
		tmp := v.Interface().(big.Rat)
		return &tmp, true, nil
	}
	return tryCoerceToRat(v)
}

// tryCoerceToRat attempts to convert a value to *big.Rat for decimal logical
// types. Accepts float64, json.Number, and numeric strings (e.g. "3.14").
//
// Floats use shortest-decimal formatting (strconv.FormatFloat, prec=-1) rather
// than (*big.Rat).SetFloat64, which exposes the exact binary mantissa:
// float64(0.33) becomes 5944751508129055/18014398509481984, a natural decimal
// scale of ~52 digits, and ratToUnscaled would then reject every
// non-power-of-2 float against any finite schema scale. Java's
// BigDecimal.valueOf(double) takes the same route via Double.toString, so 0.33
// becomes 33/100 and rounds exactly at schema scale 2.
//
// This package is the only Go impl accepting native float input for decimal —
// fastavro wants decimal.Decimal, hamba and goavro want *big.Rat. The float arm
// bypasses boundedRatFromString's isJSONNumber and magnitude gates because they
// would pass anyway: FormatFloat's 'f' output is JSON-valid by construction and
// float64's ~±308 exponent stays far under decimalScaleLimit. Skipping them
// avoids a per-call intermediate parse.
//
// Returns (nil, false, err) when the input was clearly a number form
// (json.Number, or a reflect.String that parses as a number) but its
// magnitude exceeds decimalScaleLimit — see boundedRatFromString.
func tryCoerceToRat(v reflect.Value) (*big.Rat, bool, error) {
	if v.CanFloat() {
		f := v.Float()
		// Reject non-finite values: NaN/±Inf are not in the decimal value
		// set. Java's BigDecimal.valueOf(double) rejects with
		// NumberFormatException; fastavro's prepare_bytes_decimal also
		// errors (observed 1.12.2: a TypeError from Decimal("nan")'s
		// non-integer as_tuple exponent).
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false, nil
		}
		// Float magnitudes are bounded by float64's ~310-digit FormatFloat
		// output; no decimalScaleLimit guard needed on this arm.
		//
		// bitSize=v.Type().Bits() so a float32 input uses float32's
		// shortest-decimal rule. reflect.Value.Float() widens float32 →
		// float64 losslessly but the IEEE-754 binary mantissa carries
		// trailing noise visible at float64 precision (float32(0.33) →
		// float64(0.33000001311302185)). Formatting at the source's
		// natural precision avoids parsing that noise into a fraction
		// with non-terminating-at-the-schema-scale denominator. Mirrors
		// Java's `new BigDecimal(Float.toString(f))` convention.
		if r, ok := new(big.Rat).SetString(strconv.FormatFloat(f, 'f', -1, v.Type().Bits())); ok {
			return r, true, nil
		}
		return nil, false, nil
	}
	if v.Type() == jsonNumberType {
		s := v.Interface().(json.Number).String()
		r, ok, err := boundedRatFromString(s)
		if err != nil {
			return nil, false, fmt.Errorf("json.Number %q: %w", truncForError(s), err)
		}
		if ok {
			return r, true, nil
		}
		// json.Number's type guarantees the input was meant as a number;
		// a parse failure (e.g. malformed exponent) is fatal too.
		return nil, false, fmt.Errorf("invalid decimal number %q", truncForError(s))
	}
	if v.Kind() == reflect.String {
		s := v.String()
		r, ok, err := boundedRatFromString(s)
		if err != nil {
			return nil, false, fmt.Errorf("decimal string %q: %w", truncForError(s), err)
		}
		if ok {
			return r, true, nil
		}
		// Plain reflect.String values that don't parse as a number may
		// be legitimately destined for the raw-bytes fallback (a
		// non-numeric string field), so silence is appropriate here.
	}
	return nil, false, nil
}

// decimalUnscaledBytes runs the shared (r → unscaled → validate precision →
// big-endian two's-complement bytes) pipeline. avroType labels the
// SemanticError ("bytes" or "fixed"); goType identifies the source type.
// One pipeline for the four decimal-emit sites (binary serBytesDecimal /
// serFixedDecimal, JSON appendAvroJSON's bytes+decimal and fixed+decimal
// arms) so precision/scale handling can't drift across them.
func decimalUnscaledBytes(r *big.Rat, scale, precision int, avroType string, goType reflect.Type) ([]byte, error) {
	unscaled, err := ratToUnscaled(r, scale)
	if err != nil {
		return nil, &SemanticError{GoType: goType, AvroType: avroType, Err: err}
	}
	if err := checkDecimalPrecision(unscaled, precision); err != nil {
		return nil, &SemanticError{GoType: goType, AvroType: avroType, Err: err}
	}
	b := bigIntToBytes(unscaled)
	// Charge the emitted payload against the bound the DECODER will apply to
	// these same bytes. Declared precision already keeps a parse-valid decimal
	// well inside it, so this arm cannot fire today; it is here because the
	// bound belongs to the format, not to whichever gate happens to imply it
	// — a later precision change must not silently reopen an unreadable emit.
	if err := checkDecimalUnscaledLen(b); err != nil {
		return nil, &SemanticError{GoType: goType, AvroType: avroType, Err: err}
	}
	return b, nil
}

// appendDecimalFixed pads/sign-extends b into exactly size bytes and
// appends the result to dst. Returns a SemanticError when b exceeds
// size (decimal value too wide for the fixed schema). Shared by
// serFixedDecimal.serRat (binary) and the JSON fixed+decimal arm so
// both agree on the high-bit-pad rule and the oversize-reject shape.
func appendDecimalFixed(dst, b []byte, size int, goType reflect.Type) ([]byte, error) {
	if len(b) > size {
		return nil, &SemanticError{GoType: goType, AvroType: "fixed",
			Err: fmt.Errorf("decimal value requires %d bytes, exceeds fixed size %d", len(b), size)}
	}
	// The padding below widens the payload to the schema's size, and SIZE is
	// what the decoder charges — not len(b). A fixed wider than the bound can
	// therefore never carry a readable decimal, whatever the value.
	if err := checkDecimalUnscaledSize(size); err != nil {
		return nil, &SemanticError{GoType: goType, AvroType: "fixed", Err: err}
	}
	pad := byte(0)
	if len(b) > 0 && b[0]&0x80 != 0 {
		pad = 0xff
	}
	for i := len(b); i < size; i++ {
		dst = append(dst, pad)
	}
	return append(dst, b...), nil
}

// rejectNonNumericStructuredString rejects a reflect.String carrier reaching a
// decimal / big-decimal opaque fall-through. decimalRatFor consumes numeric
// strings first, so anything arriving here is non-numeric — while the string
// DECODE target of both logicals reads the wire as numeric text whenever it
// can. Encoding such a string opaquely therefore emits bytes the decoder reads
// back as a NUMBER, and a crafted string whose raw bytes happen to be valid
// framing decodes to a different value, silently breaking the round trip. For
// both logicals the string carrier is the numeric-text form ONLY; []byte is the
// opaque escape hatch (Kind Slice, unmatched here) and round-trips
// symmetrically, its decode target reading raw bytes unconditionally.
//
// Deliberately stricter than the plain bytes/fixed string leniency, where a
// string round-trips opaquely because that decode target also reads raw bytes.
// Neither Java nor fastavro accepts a native string for these schemas at all,
// so there is no interop cost. logical labels the SemanticError message
// ("decimal" or "big-decimal"); avroType is the underlying Avro type ("bytes" /
// "fixed").
func rejectNonNumericStructuredString(v reflect.Value, avroType, logical string) error {
	if v.Kind() == reflect.String {
		return &SemanticError{GoType: v.Type(), AvroType: avroType, Err: fmt.Errorf("invalid %s string %q", logical, truncForError(v.String()))}
	}
	return nil
}

// chargeOpaqueDecimalBytes charges the opaque []byte escape hatch — the carrier
// for a caller who assembles the payload themselves — against the bound the
// DECODER applies to those same bytes. The numeric carriers are charged inside
// the shared builders (decimalUnscaledBytes / appendDecimalFixed /
// buildBigDecimalPayload); this is the arm that reaches the wire without them.
//
// logical decides WHICH bytes the decoder will charge, because the two wire
// shapes differ: for "decimal" the payload IS the unscaled value, while for
// "big-decimal" the payload is a framing whose length-prefixed INNER unscaled
// value is what parseBigDecimalPayload charges. A framing this cannot read is
// left alone deliberately — the decoder then fails on the framing itself, and
// judging framing here would be a different rule than this bound.
func chargeOpaqueDecimalBytes(v reflect.Value, avroType, logical string) error {
	if k := v.Kind(); k != reflect.Slice && k != reflect.Array {
		return nil // not a byte carrier; the base serializer names its own error
	}
	if v.Type().Elem().Kind() != reflect.Uint8 {
		return nil
	}
	// Read a bounded prefix rather than materializing the payload, which may
	// be an unaddressable array: only the leading varlong is ever needed.
	const varlongMax = 10 // a zigzag varlong is 1-10 bytes (appendVarlong)
	prefix := make([]byte, 0, varlongMax)
	for i := 0; i < v.Len() && i < varlongMax; i++ {
		prefix = append(prefix, byte(v.Index(i).Uint()))
	}
	n, ok := decimalChargeLen(prefix, v.Len(), logical)
	if !ok {
		return nil
	}
	if err := checkDecimalUnscaledSize(n); err != nil {
		return &SemanticError{GoType: v.Type(), AvroType: avroType, Err: err}
	}
	return nil
}

// decimalChargeLen answers the one question every decimal emit path has to ask
// before it writes: how many bytes will the DECODER hand to
// checkDecimalUnscaledLen for a payload this long whose leading bytes are
// prefix. It is one function because the answer differs only by wire shape and
// nothing else, and each caller having its own copy is how the two shapes drift.
//
// On "decimal" the payload IS the unscaled value. On "big-decimal" the payload
// is a framing and the charged slice is the length-prefixed INNER value, so the
// leading varlong is read to find it — and ok is false when that varlong cannot
// be read, because a payload whose framing is unreadable fails on the framing,
// which is a different rule than this bound.
func decimalChargeLen(prefix []byte, totalLen int, logical string) (int, bool) {
	if logical != "big-decimal" {
		return totalLen, true
	}
	uLen, _, err := readVarlong(prefix)
	if err != nil || uLen < 0 {
		return 0, false
	}
	// Clamp so the int conversion cannot overflow on a 32-bit build; anything
	// at or past the bound rejects identically.
	return int(min(uLen, int64(maxDecimalUnscaledBytes)+1)), true
}

// chargeDecimalLeaf is the producer-compliance charge for a decimal payload
// emitted by the DEFAULT walk. It is called at the LEAF — the bytes/fixed arm
// that actually writes the payload — because that is the only place the
// carrier's own kind and logical are in hand; asking at the field's node
// answers for a container whenever the decimal is nested inside one.
//
// It asks the same authority the serializers ask, through the same two
// functions, so it refuses exactly the payloads decode refuses.
func chargeDecimalLeaf(payload []byte, logical string) error {
	if logical != "decimal" && logical != "big-decimal" {
		return nil
	}
	n, ok := decimalChargeLen(payload, len(payload), logical)
	if !ok {
		return nil
	}
	return checkDecimalUnscaledSize(n)
}

type serBytesDecimal struct {
	precision int
	scale     int
}

func (s *serBytesDecimal) serRat(dst []byte, r *big.Rat) ([]byte, error) {
	b, err := decimalUnscaledBytes(r, s.scale, s.precision, "bytes", bigRatType)
	if err != nil {
		return nil, err
	}
	dst = appendVarlong(dst, int64(len(b)))
	return append(dst, b...), nil
}

func (s *serBytesDecimal) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, r, ok, err := coerceDecimalRat(v, "bytes")
	if err != nil {
		return nil, err
	}
	if ok {
		return s.serRat(dst, r)
	}
	if err := rejectNonNumericStructuredString(v, "bytes", "decimal"); err != nil {
		return nil, err
	}
	if err := chargeOpaqueDecimalBytes(v, "bytes", "decimal"); err != nil {
		return nil, err
	}
	return serBytes(dst, v, depth)
}

type serFixedDecimal struct {
	size      int
	precision int
	scale     int
}

func (s *serFixedDecimal) serRat(dst []byte, r *big.Rat) ([]byte, error) {
	b, err := decimalUnscaledBytes(r, s.scale, s.precision, "fixed", bigRatType)
	if err != nil {
		return nil, err
	}
	return appendDecimalFixed(dst, b, s.size, bigRatType)
}

func (s *serFixedDecimal) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, r, ok, err := coerceDecimalRat(v, "fixed")
	if err != nil {
		return nil, err
	}
	if ok {
		return s.serRat(dst, r)
	}
	if err := rejectNonNumericStructuredString(v, "fixed", "decimal"); err != nil {
		return nil, err
	}
	// The opaque arm writes exactly s.size bytes, which is what the decoder
	// charges — the same condition appendDecimalFixed applies to the numeric
	// arm, so the two carriers agree on which schemas can emit at all.
	if err := checkDecimalUnscaledSize(s.size); err != nil {
		return nil, &SemanticError{GoType: v.Type(), AvroType: "fixed", Err: err}
	}
	return (&serSize{s.size}).ser(dst, v, depth)
}

type serBigDecimal struct{}

func (s *serBigDecimal) ser(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, r, ok, err := coerceDecimalRat(v, "bytes")
	if err != nil {
		return nil, err
	}
	if ok {
		return s.serRat(dst, r, v.Type())
	}
	// A non-numeric string carrier is not a valid big-decimal; reject it
	// (numeric-text-only, symmetric with decode) rather than fall through to
	// the opaque raw-bytes path below. A big-decimal string DECODE target reads
	// numeric text whenever the wire parses as valid framing
	// (applyBigDecimalPayload → setDecimalRat, deser.go), so a crafted string
	// whose bytes ARE a valid framing would decode to a different value —
	// mirrors serBytesDecimal / serFixedDecimal.
	if err := rejectNonNumericStructuredString(v, "bytes", "big-decimal"); err != nil {
		return nil, err
	}
	if err := chargeOpaqueDecimalBytes(v, "bytes", "big-decimal"); err != nil {
		return nil, err
	}
	// Fall back to plain bytes for a []byte carrier: preserves opaque-bytes
	// pass-through for users who construct the wire payload manually. A []byte
	// decode target reads raw bytes unconditionally, so []byte is symmetric.
	return serBytes(dst, v, depth)
}

func (s *serBigDecimal) serRat(dst []byte, r *big.Rat, srcType reflect.Type) ([]byte, error) {
	inner, err := buildBigDecimalPayload(r)
	if err != nil {
		return nil, &SemanticError{GoType: srcType, AvroType: "bytes", Err: err}
	}
	// Outer bytes framing: zigzag-len(inner) || inner.
	dst = appendVarlong(dst, int64(len(inner)))
	return append(dst, inner...), nil
}

// buildBigDecimalPayload returns the big-decimal inner payload bytes
// (length-prefixed unscaled || zigzag scale). Errors on rationals
// with no finite decimal expansion.
func buildBigDecimalPayload(r *big.Rat) ([]byte, error) {
	scale, ok := finiteScale(r)
	if !ok {
		return nil, fmt.Errorf("big.Rat %s has no finite decimal expansion; big-decimal cannot encode this value", truncRatForError(r))
	}
	num := new(big.Int).Mul(r.Num(), pow10(scale))
	unscaled, _ := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	// Remainder is provably 0 since finiteScale chose s to make
	// 10^s / Denom an integer.
	uBytes := bigIntToBytes(unscaled)
	// big-decimal carries no precision attribute, so nothing upstream bounds
	// the magnitude: this is the one numeric-carrier arm where an ordinary
	// *big.Rat reaches the wire unbounded. Charge the INNER unscaled bytes —
	// the exact slice parseBigDecimalPayload hands to the same function.
	if err := checkDecimalUnscaledLen(uBytes); err != nil {
		return nil, err
	}
	// Inner payload: zigzag-len(uBytes) || uBytes || zigzag(scale).
	inner := appendVarlong(nil, int64(len(uBytes)))
	inner = append(inner, uBytes...)
	inner = appendVarlong(inner, int64(scale))
	return inner, nil
}

// log2of5 is log2(5), used to estimate a denominator's factor-of-5 count
// from its bit length without an O(scale) division loop.
const log2of5 = 2.321928094887362

// finiteScale returns the smallest s >= 0 such that r * 10^s is an
// integer, or (0, false) if r has no finite decimal expansion (or
// would require a scale beyond decimalScaleLimit — same outcome
// from the caller's perspective).
// For a reduced denominator d = 2^a * 5^b returns max(a, b).
//
// The factor-of-2 count a is one TrailingZeroBits() call. The denominator's odd
// part must be a pure power of 5 for the decimal to terminate. Dividing it by 5
// one factor at a time is O(scale^2) on the shrinking big.Int — ~1.4 CPU
// seconds for a 6-byte wire value at the cap, an attacker amplification on the
// decode→re-encode path — so instead estimate b from the bit length (5^b has
// floor(b·log2 5)+1 bits) and verify with a single 5^b == d comparison plus a
// ≤1-step climb absorbing the float rounding. 5^b is strictly increasing, so at
// most one b can equal d, and a miss means d has a prime factor other than 5
// and the value is non-terminating. The derivation is O(M(scale)), matching the
// regular-decimal encode path.
// (Java/fastavro/avro-rs pay even less because their decimal types store the
// scale and never factorize; big.Rat is a reduced fraction with no scale, so
// the value-derived scale must be computed here — O(M) is the floor for that
// input, and is the same order as the unscaled-value computation that
// follows in buildBigDecimalPayload regardless.)
func finiteScale(r *big.Rat) (int, bool) {
	d := new(big.Int).Set(r.Denom())
	a := int(d.TrailingZeroBits())
	if a > decimalScaleLimit {
		return 0, false
	}
	if a > 0 {
		d.Rsh(d, uint(a))
	}
	b := 0
	one := big.NewInt(1)
	if d.Cmp(one) != 0 {
		// Reject a denominator too large to be a permitted power of 5
		// before materializing 5^b. Compare in float64 BEFORE the int()
		// conversion so a multi-gigabit denominator can't overflow int on a
		// 32-bit build. 5^b == d implies b < BitLen(d)/log2 5; the +1 margin
		// leaves the exact b == cap case for the final check below.
		bEstF := float64(d.BitLen()-1) / log2of5
		if bEstF > float64(decimalScaleLimit)+1 {
			return 0, false
		}
		five := big.NewInt(5)
		b = int(bEstF)
		pow := new(big.Int).Exp(five, big.NewInt(int64(b)), nil)
		for pow.Cmp(d) < 0 { // estimate low: climb (≤ ~2 steps)
			pow.Mul(pow, five)
			b++
		}
		for pow.Cmp(d) > 0 && b > 0 { // estimate high: descend
			pow.Quo(pow, five)
			b--
		}
		if pow.Cmp(d) != 0 {
			return 0, false // d is not a pure power of 5 → non-terminating
		}
	}
	if b > decimalScaleLimit {
		return 0, false
	}
	if a > b {
		return a, true
	}
	return b, true
}

// pow10 returns 10^n as a *big.Int. n must be non-negative.
// Shared chokepoint for every decimal encode/decode site that materializes
// a power of ten — keeps the (single) DoS-bound update site bound to
// decimalScaleLimit if a future tightening is needed.
func pow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

// scaledRat returns unscaled * 10^(-scale) as a fresh *big.Rat.
// Positive scale divides (the standard Avro decimal interpretation;
// schema scale=2 of unscaled=33 → 0.33); negative scale multiplies
// (the Avro big-decimal form where the wire encodes a left-shifted
// integer, e.g. scale=-3 of unscaled=42 → 42000).
func scaledRat(unscaled *big.Int, scale int) *big.Rat {
	if scale < 0 {
		num := new(big.Int).Mul(unscaled, pow10(-scale))
		return new(big.Rat).SetFrac(num, big.NewInt(1))
	}
	return new(big.Rat).SetFrac(unscaled, pow10(scale))
}

// ratToUnscaled returns the unscaled big.Int (rat * 10^scale / denom) when
// the value is exactly representable at the requested scale, or an error
// when the conversion would require rounding. Used by both
// serBytesDecimal/serFixedDecimal (which separately need the unscaled
// value to validate against precision) and the JSON encoder.
//
// Java's DecimalConversion.validate uses RoundingMode.UNNECESSARY,
// throwing AvroTypeException when the value's scale exceeds the schema's
// (Conversions.java:151). fastavro's prepare_bytes_decimal raises
// ValueError when delta = exp + scale < 0 (_logical_writers_py.py:131).
// big.NewRat(1, 3) at scale=2 has remainder 1 after multiplying by 100,
// matching Java's "scale=infinite > schema scale=2" rejection.
func ratToUnscaled(r *big.Rat, scale int) (*big.Int, error) {
	num := new(big.Int).Mul(r.Num(), pow10(scale))
	unscaled, rem := new(big.Int).QuoRem(num, r.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		return nil, fmt.Errorf("decimal value %s cannot be represented at scale %d without rounding", truncRatForError(r), scale)
	}
	return unscaled, nil
}

// checkDecimalPrecision rejects unscaled values whose decimal-digit count
// exceeds precision. Per Avro 1.12 spec: precision is "the (maximum)
// precision of decimals stored in this type". Java's Conversions.DecimalConversion,
// fastavro, and hamba/avro all enforce this on encode.
func checkDecimalPrecision(unscaled *big.Int, precision int) error {
	if precision <= 0 {
		return nil
	}
	digits := decimalDigitCount(unscaled)
	if digits > precision {
		return fmt.Errorf("decimal value has %d digits, exceeds schema precision %d", digits, precision)
	}
	return nil
}

// decimalDigitCount returns the number of decimal digits in |i|.
// Uses big.Int.BitLen as a fast lower-bound check, then String() for the
// exact count when ambiguous.
func decimalDigitCount(i *big.Int) int {
	if i.Sign() == 0 {
		return 1
	}
	abs := new(big.Int).Abs(i)
	return len(abs.String())
}

// bigIntToBytes encodes i as big-endian two's complement, using the
// minimum number of bytes needed to represent the value with correct sign.
func bigIntToBytes(i *big.Int) []byte {
	switch i.Sign() {
	case 0:
		return []byte{0}
	case 1:
		b := i.Bytes() // big-endian unsigned
		if b[0]&0x80 != 0 {
			// High bit set would look negative in two's complement;
			// prepend a zero byte to keep it positive.
			b = append([]byte{0}, b...)
		}
		return b
	default:
		// Two's complement for negative: flip bits of (|i| - 1).
		// This works because -n in two's complement is ^(n-1).
		abs := new(big.Int).Neg(i)
		abs.Sub(abs, big.NewInt(1))
		b := abs.Bytes()
		if len(b) == 0 {
			return []byte{0xff} // -1
		}
		for j := range b {
			b[j] = ^b[j]
		}
		if b[0]&0x80 == 0 {
			// High bit clear would look positive; prepend 0xff
			// to preserve the negative sign.
			b = append([]byte{0xff}, b...)
		}
		return b
	}
}

// serFixedUUIDReflect serializes a fixed(16) UUID. Accepts [16]byte (raw),
// string (hex-dash UUID parsed to bytes), or []byte of length 16.
func serFixedUUIDReflect(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	// [16]byte trusts its bytes: the raw 16 bytes ARE the UUID wire form,
	// so they are written directly without a MarshalText→parseUUID round
	// trip (which would be redundant for a canonical type and could
	// spuriously fail parseUUID on an otherwise-valid [16]byte). This is
	// the uuidBytes-first rule the JSON encoder mirrors.
	if u, ok := uuidBytes(v); ok {
		return append(dst, u[:]...), nil
	}
	// TextMarshaler / AppendText before the reflect.String arm (parity with
	// the string encoders): a struct or string-kind type implementing a
	// text method derives its UUID text that way. The text must be a
	// parseable UUID; parseUUID validates and yields the 16 wire bytes.
	if text, ok, err := textValue(v, "fixed"); err != nil {
		return nil, err
	} else if ok {
		u, err := parseUUID(text)
		if err != nil {
			return nil, err
		}
		return append(dst, u[:]...), nil
	}
	if v.Kind() == reflect.String {
		// A plain string with UUID-shaped content is accepted (parsed to the
		// 16 wire bytes). json.Number is rejected up front: it is a numeric
		// carrier, valid only for numeric Avro types, symmetric with the
		// decoder which rejects a json.Number fixed target. (It would also
		// fail parseUUID, but the explicit reject gives a clear error.)
		if err := rejectJSONNumberRawTarget(v, "fixed"); err != nil {
			return nil, err
		}
		u, err := parseUUID(v.String())
		if err != nil {
			return nil, err
		}
		return append(dst, u[:]...), nil
	}
	return (&serSize{16}).ser(dst, v, depth)
}

// isUUIDType returns true when t is an array of 16 uint8 bytes (e.g. [16]byte
// or any type whose underlying type is [16]byte).
func isUUIDType(t reflect.Type) bool {
	return t.Kind() == reflect.Array && t.Len() == 16 && t.Elem().Kind() == reflect.Uint8
}

// uuidBytes copies v's [16]byte payload out into a stack-allocated array
// when v is a UUID-typed array, returning (u, true). For non-UUID v it
// returns (zero, false). Shared by the three encode sites that need to
// materialize a [16]byte from a reflect.Value (serFixedUUIDReflect,
// serUUID, the JSON "string"+uuid arm).
func uuidBytes(v reflect.Value) ([16]byte, bool) {
	if !isUUIDType(v.Type()) {
		return [16]byte{}, false
	}
	var u [16]byte
	// Exact-uint8 element: reflect.Copy's memmove (zero-alloc). A named byte
	// element ([16]B, type B byte) is Kind Uint8 but not exactly uint8, so
	// reflect.Copy panics; read it out element-wise instead (the same shape as
	// byteArrayToSlice / copyBytesToArray, kept inline here to stay zero-alloc).
	if v.Type().Elem() == byteType {
		reflect.Copy(reflect.ValueOf(&u).Elem(), v)
	} else {
		for i := range u {
			u[i] = byte(v.Index(i).Uint())
		}
	}
	return u, true
}

// uuidToString formats a [16]byte as the RFC 4122 hex-dash string
// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.
func uuidToString(u [16]byte) string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

func serUUID(dst []byte, v reflect.Value, depth int) ([]byte, error) {
	v, err := indirect(v)
	if err != nil {
		return nil, err
	}
	if u, ok := uuidBytes(v); ok {
		return doSerString(dst, uuidToString(u)), nil
	}
	return serString(dst, v, depth)
}
