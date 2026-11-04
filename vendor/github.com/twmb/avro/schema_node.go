package avro

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// SchemaNode is a read-write representation of an Avro schema. It can be
// obtained from a parsed schema via [Schema.Root], or constructed directly
// and converted to a [*Schema] via the [SchemaNode.Schema] method.
//
// The Type field determines which other fields are relevant:
//
//   - Primitives (null, boolean, int, long, float, double, string, bytes):
//     LogicalType, Precision, Scale, Props optional; other fields ignored.
//   - record/error: Name, Fields required; Namespace, Doc, Props optional.
//   - enum: Name, Symbols required; Namespace, Doc, Props optional.
//   - array: Items required.
//   - map: Values required.
//   - fixed: Name, Size required; LogicalType, Precision, Scale, Namespace,
//     Props optional.
//   - union: Branches lists the member schemas.
//
// A named type (record, enum, fixed) already defined elsewhere in the schema
// can be referenced by setting Type to its full name (e.g.
// com.example.Address) with no other fields. In a [Schema.Root] tree,
// references also resolve outward: converting ANY node with
// [SchemaNode.Schema] resolves names against the schema the tree came from,
// so a field type, union branch, or deeper node converts even when the
// definition lives outside the extracted node. A hand-built tree has no
// enclosing schema, so there every referenced name must be defined within the
// tree being converted, or Schema returns an error.
type SchemaNode struct {
	Type        string // Avro type or named type reference
	LogicalType string // e.g. date, timestamp-millis, decimal, uuid; empty if none (or if the attribute's value is not a string — see Props)

	Name string // name for record, enum, fixed

	// Namespace is the named type's RESOLVED namespace: [Schema.Root] fills it
	// for every named type, a child inheriting its enclosing namespace
	// surfaces that namespace here, and "" always means the null namespace,
	// never "inherit". [SchemaNode.Schema] emits a "namespace":"" escape when
	// a null-namespace type sits inside a namespaced scope, so the distinction
	// survives the round trip. A dotted Name takes precedence over this field.
	Namespace string

	Aliases []string // alternate names for named types (record, enum, fixed)
	Doc     string   // documentation string

	Fields   []SchemaField // record fields
	Items    *SchemaNode   // array element schema
	Values   *SchemaNode   // map value schema
	Branches []SchemaNode  // union member schemas
	Symbols  []string      // enum symbols
	Size     int           // fixed byte size

	EnumDefault    string // default symbol for enum schema evolution
	HasEnumDefault bool   // true if an enum default is defined

	// Precision and Scale are the decimal logical type's parameters, set and
	// validated exactly when LogicalType is "decimal" on a bytes or fixed
	// carrier. Anywhere else — no logical type, an unknown or non-decimal one,
	// or a decimal on a carrier it soft-drops from — the attributes are inert
	// metadata surfaced in Props, matching the field level.
	Precision int // decimal precision
	Scale     int // decimal scale

	// Props holds custom (non-reserved) schema attributes — anything in the
	// schema JSON that is not a standard Avro field (e.g. "com.example.tag").
	// It is also the ONLY surface for a reserved structural key on a kind that
	// does not bind it whose body does not parse as that key's schema shape (a
	// stray "items":3 on an "int"): the matching structural field stays zero.
	// A schema-shaped stray body instead surfaces as-written on Items / Values
	// / Fields. A non-string logicalType is likewise inert and appears here
	// verbatim, since nothing but a string can name a logical.
	//
	// Values use the natural Go types from JSON: string, bool, nil, []any,
	// map[string]any, int64 for whole numbers, float64 for fractional. A
	// number stays json.Number when neither fits — a whole number too large
	// for int64, or a fractional literal over 1024 bytes, whose digits are
	// kept verbatim rather than rounded. Whole-valued exponents collapse to
	// int64 (1e3 reads as int64(1000)); exponents overflowing float64 give
	// ±Inf. math.NaN() re-reads as the string "NaN" after Schema()/Root(),
	// because JSON has no NaN literal; ±Inf round-trips as float64(±Inf).
	Props map[string]any

	// refTarget is set only by [Schema.Root], on name-reference nodes, and
	// points at the referenced definition inside the same tree.
	// [SchemaNode.Schema] emits that definition when the tree being converted
	// does not define the name itself — which is what lets a node extracted at
	// ANY depth convert to a working schema. Invisible otherwise: hand-built
	// nodes leave it nil (a dangling reference stays a loud parse error),
	// copies and slice extractions carry it, and a node rebuilt field-by-field
	// drops it and thereafter behaves hand-built.
	refTarget *SchemaNode

	// present records which attributes were WRITTEN, which the fields
	// themselves cannot: an attribute whose body is the field's zero — "", [],
	// 0 — is indistinguishable from one nobody wrote, so `Doc != ""` /
	// `len(Aliases) > 0` / `Size != 0` each mean two things and drop exactly
	// the value written as the zero. Unexported because a hand-composed node
	// has no empty-doc to express (the distinction exists only for parsed
	// text) and an exported companion would be new API.
	//
	// Presence is recorded only where the attribute is CONSUMED, and consulted
	// per attribute rather than as one rule, because the authority differs by
	// placement. Where Apache Avro has the placement its emission condition
	// governs, and those conditions differ: doc emits when non-null
	// (Schema.java:1039/:1154/:1367/:1062) so an empty doc survives, aliases
	// when non-EMPTY (:886, :1070) so an empty list is dropped even on a
	// binding kind. Where neither reference has the placement — a structural
	// key on a kind that does not bind it — this package's stray-routing
	// posture governs: as-written is the key's ONLY surface, so it must
	// survive the rebuild rather than reaching neither surface.
	present presenceSet

	// refNS is the namespace scope refTarget was resolved in, recorded
	// alongside the stamp because the two are only meaningful together:
	// whether Type still names the target is a question about the scope the
	// reference was WRITTEN in, and an extracted node is re-rooted at the
	// null namespace, losing it. Set and read only with refTarget
	// (nodeRefTargetAgrees); nil stamp means the value is unused.
	refNS string
}

// SchemaField represents a field in an Avro record schema.
type SchemaField struct {
	Name string     // field name
	Type SchemaNode // field schema

	// Default is the field's default value, present when HasDefault is
	// true. The Go type matches the schema:
	//
	//   - int schemas give int32, long schemas give int64. Out-of-range
	//     defaults are rejected at parse.
	//   - float schemas give float32, double float64. Overflows narrow to
	//     ±Inf; NaN, ±Inf, and a float-syntax "-0.0" round-trip. An
	//     integer-syntax "-0" is the sign-less integer 0 and surfaces as +0.0
	//     (matching Java/fastavro), though the wire encoder writes -0.0 for it.
	//   - string and enum schemas give string.
	//   - bytes and fixed give []byte, already decoded from the JSON spec's
	//     codepoint-per-byte form.
	//   - record, array, and map give map[string]any or []any, each leaf
	//     following these same rules.
	//
	// Union defaults pick the first branch that accepts the value, and the Go
	// type tells you which: ["float","int"] with default 42 gives float32(42).
	//
	// Unlike Props, a numeric Default is never json.Number — parse rejects
	// defaults that do not fit the declared type.
	Default any

	HasDefault bool     // true if a default value is defined in the schema
	Aliases    []string // field aliases for schema evolution
	Order      string   // sort order: "ascending" (default), "descending", or "ignore"
	Doc        string   // documentation string

	// docSet is the field-level twin of [SchemaNode]'s: a field's "doc"
	// key written as the empty string is a written doc, and Apache Avro
	// emits it (Schema.java:1062 asks f.doc() != null). See the SchemaNode
	// field for why the state is hidden.
	docSet bool

	// Props holds custom (non-reserved) field properties; numbers decode as in
	// [SchemaNode.Props]. Field-level "logicalType", "precision", and "scale"
	// appear here as written — the wire-side lift is a codec concession that
	// never removes them from this surface. An unconsumed precision/scale (no
	// field logicalType, a non-decimal one, or a decimal whose lift target is
	// not a bytes/fixed carrier) is an ordinary property whatever its JSON
	// shape; only a consumed placement shape-validates the pair at parse.
	Props map[string]any
}

// Schema parses the SchemaNode into a [*Schema] that can be used for
// encoding and decoding. Returns an error if the node is invalid.
//
// Named types appearing multiple times are deduplicated by FULLNAME: the
// first occurrence emits the definition, later ones emit the fullname as a
// reference. Two types sharing a short name across namespaces are distinct
// and both emit definitions.
//
// A node extracted from a [Schema.Root] tree may reference definitions living
// elsewhere in the enclosing schema — an earlier field, a prior [SchemaCache]
// parse, or the enclosing type itself for a recursive schema. Those resolve
// automatically: the definition is emitted at the reference's first
// occurrence, so the result needs neither the enclosing schema nor any cache.
// A name the tree defines itself wins over the enclosing schema's definition,
// and custom properties on a wrapped reference ride onto the emitted
// definition (reserved usage-site attributes do not survive, matching the
// SchemaCache splice). Hand-built nodes carry no enclosing schema, so there a
// reference the tree does not define is an error.
//
// opts pass through to the internal [Parse]. A schema originally parsed with
// [SchemaOpt]s that change what Parse accepts or wires — [WithLaxNames],
// [CustomType] registrations — needs the same opts here, or the rebuilt schema
// fails to parse or silently lacks the custom wiring.
func (n *SchemaNode) Schema(opts ...SchemaOpt) (*Schema, error) {
	d := &deduper{
		defined: make(map[string]*SchemaNode),
		visited: make(map[*SchemaNode]struct{}),
	}
	tree := n.toJSONDedup(d)
	if d.err != nil {
		return nil, d.err
	}
	b, err := marshalSchemaTree(tree)
	if err != nil {
		return nil, fmt.Errorf("avro: marshaling schema node: %w", err)
	}
	return Parse(string(b), opts...)
}

// deduper tracks named type definitions during toJSONDedup and records
// conflicting redefinitions. It also detects cycles introduced via
// *SchemaNode Items/Values pointers (which are the only way a SchemaNode
// tree can have true cycles — Fields and Branches are value slices).
type deduper struct {
	defined map[string]*SchemaNode   // fullname → first definition's node
	visited map[*SchemaNode]struct{} // seen *SchemaNode pointers (cycle detection)
	err     error                    // first conflict or cycle encountered

	// localNames holds the fullname of every named type DEFINED anywhere in
	// the tree, collected up front (collectLocalNames). The refTarget splice
	// consults it so a reference whose definition is present — before OR after
	// the reference, forward references included — stays as-written and binds
	// locally on re-parse. Only a reference to a name the tree nowhere defines
	// splices the stamped target in.
	localNames map[string]bool
}

// Root returns a SchemaNode tree describing the parsed schema. All
// metadata is preserved (doc strings, namespaces, custom properties,
// numeric defaults). See [SchemaNode.Props] and [SchemaField.Default]
// for how values decode.
//
// Reserved Avro attribute names ("type", "name", "namespace", "doc",
// "aliases", …) match only by their exact lowercase spelling, as in the Avro
// reference implementations. A case variant such as "Aliases" is an ordinary
// custom property: it never binds the attribute and is reported verbatim in
// [SchemaNode.Props]. Parsing applies the same rule, so a schema whose only
// spelling of a structural key is a case variant ("ITEMS" on an array) fails
// Parse — the structural attribute is absent.
//
// A field in the flat (goavro-style) format — a bare-string complex-kind type
// with the kind's defining key (symbols, items, values, fields, size)
// alongside the field's own keys — is described post-lift, exactly as it
// parses: the field's type is the lifted nested definition (named after the
// field for record/error/enum/fixed), and the keys the lift routed into the
// type appear on the type node rather than in [SchemaField.Props].
// [SchemaNode.Schema] rebuilds the nested form, which parses identically.
//
// Every node converts back to a usable [*Schema] via [SchemaNode.Schema],
// name-reference nodes included: the tree carries the schema's named-type
// definitions, so any extracted subtree is self-contained.
//
// Root re-parses the JSON on each call. Cache the result if you access it
// repeatedly (e.g. in a per-message loop).
func (s *Schema) Root() *SchemaNode {
	raw, err := unmarshalAnyPreservePrecision(s.full)
	if err != nil {
		panic("avro: Schema.Root: invalid stored JSON: " + err.Error())
	}
	// One shape memo for the whole walk: stray bodies nest, so re-validating
	// each subtree once per enclosing level would be O(depth^2).
	n := nodeFromJSON(raw, "", make(strayShapeMemo))
	table := fixupNameRefDefaults(&n)
	// Stamp each name-reference node with its resolved target so an extracted
	// subtree converts even when the definition lives outside it. Same table
	// the default fixup resolved through, so the two surfaces cannot bind a
	// reference differently.
	stampNameRefs(&n, table, "")
	return &n
}

// toJSONDedup is like toJSON but deduplicates named types. The first
// occurrence of a named type (record, enum, fixed) emits the full
// definition; subsequent occurrences emit the name as a reference.
func (n *SchemaNode) toJSONDedup(d *deduper) any {
	b := newWalkBudget()
	d.localNames = make(map[string]bool)
	collectLocalNames(n, d.localNames, make(map[*SchemaNode]struct{}), 0)
	return n.toJSONWalk(d.visited, d, "", 0, &b, false)
}

// jsonSerializableValue returns v with the Avro-JSON shape fixups applied,
// directly or under map[string]any / []any layers; applyJSONFixup documents
// what they are. Containers are deep-copied only when a descendant needs
// converting, so the common case allocates nothing and the user's SchemaNode
// storage is never mutated.
func jsonSerializableValue(v any) any {
	if !needsJSONFixup(v) {
		return v
	}
	return applyJSONFixup(v)
}

// isNegativeZero reports whether f is IEEE-754 negative zero (−0.0). Distinct
// from +0.0 only by the sign bit; both compare == 0.
func isNegativeZero(f float64) bool {
	return f == 0 && math.Signbit(f)
}

func needsJSONFixup(v any) bool {
	switch tv := v.(type) {
	case float64:
		return math.IsInf(tv, 0) || math.IsNaN(tv) || isNegativeZero(tv)
	case float32:
		return math.IsInf(float64(tv), 0) || math.IsNaN(float64(tv)) || isNegativeZero(float64(tv))
	case []byte:
		return true
	case map[string]any:
		for _, val := range tv {
			if needsJSONFixup(val) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(tv, needsJSONFixup)
	case nil, string, bool, json.Number, int, int32, int64:
	default:
		return needsJSONFixupKind(v)
	}
	return false
}

var jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

// treeValueMarshalOpaque reports whether v defines its own JSON form — a
// MarshalJSON/MarshalText method, or json.Number, which encoding/json
// special-cases as a number rather than a string. Such values keep their
// marshal semantics untouched: the fixups and the canonicalizing render copy
// leave them alone, and the composition walkers treat them as opaque leaves.
// The assertions use the value's own method set, matching what encoding/json
// consults for an interface-carried (unaddressable) value.
func treeValueMarshalOpaque(v any) bool {
	switch v.(type) {
	case json.Number, json.Marshaler, encoding.TextMarshaler:
		return true
	}
	return false
}

// canonicalByteSliceKind reports whether t marshals as a raw byte string: a
// uint8-kind slice whose element supplies no marshal of its own. Mirrors
// encoding/json's byte-slice rule, which consults the element's POINTER method
// set because slice elements are addressable.
func canonicalByteSliceKind(t reflect.Type) bool {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return false
	}
	p := reflect.PointerTo(t.Elem())
	return !p.Implements(jsonMarshalerType) && !p.Implements(textMarshalerType)
}

// sliceElemMarshalPositionDependent reports whether boxing a t-typed
// slice/array element into an interface would CHANGE its marshal: a
// pointer-receiver-only marshaler is reachable from an addressable element in
// place but not from an interface-carried copy. Containers of such elements
// stay opaque rather than canonicalizing into a different []any.
func sliceElemMarshalPositionDependent(t reflect.Type) bool {
	p := reflect.PointerTo(t)
	if !p.Implements(jsonMarshalerType) && !p.Implements(textMarshalerType) {
		return false
	}
	return !t.Implements(jsonMarshalerType) && !t.Implements(textMarshalerType)
}

// canonicalStringKeyMap reports whether t's keys canonicalize to their plain
// string value: every string-KIND key does. encoding/json checks the string
// kind FIRST, so a string-kind key marshals raw and its MarshalText is never
// consulted (executed; jsonv2 flips that precedence, so pinning the raw string
// keeps the composed schema identical across toolchains). json.Marshaler is
// never consulted for keys either way. NON-string-kind keys are the opposite —
// their MarshalText output IS the key under both — so those maps stay
// marshal-opaque image-owners.
func canonicalStringKeyMap(t reflect.Type) bool {
	return t.Key().Kind() == reflect.String
}

// needsJSONFixupKind extends fixup detection to caller-typed values by reflect
// kind, so a named `type B []byte` or named float behaves like the canonical
// twin its marshal is indistinguishable from. Marshal-opaque values are exempt
// — their marshal wins. One deliberate asymmetry: the value-PRESERVING fixups
// (±Inf, -0.0) apply to named float kinds, but the type-CHANGING NaN→"NaN"
// stays canonical-only, so a named float NaN keeps json.Marshal's loud
// unsupported-value error instead of being silently stringified.
func needsJSONFixupKind(v any) bool {
	if treeValueMarshalOpaque(v) {
		return false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float64, reflect.Float32:
		f := rv.Float()
		return math.IsInf(f, 0) || isNegativeZero(f)
	case reflect.Slice:
		if canonicalByteSliceKind(rv.Type()) {
			return true
		}
		fallthrough
	case reflect.Array:
		if sliceElemMarshalPositionDependent(rv.Type().Elem()) {
			return false
		}
		for i := range rv.Len() {
			if needsJSONFixup(rv.Index(i).Interface()) {
				return true
			}
		}
	case reflect.Map:
		if !canonicalStringKeyMap(rv.Type()) {
			return false
		}
		for it := rv.MapRange(); it.Next(); {
			if needsJSONFixup(it.Value().Interface()) {
				return true
			}
		}
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return false
		}
		return needsJSONFixup(rv.Elem().Interface())
	}
	return false
}

// applyJSONFixup converts the four values encoding/json cannot round-trip
// through an Avro schema. In each case the naive marshal is silently wrong:
//
//   - ±Inf → json.Number("±1e1000"). Go's encoder rejects ±Inf outright, so a
//     Root() whose Default overflowed could not re-marshal at all; the literal
//     re-parses to ±Inf via parseFloatAcceptOverflow.
//   - NaN → the string "NaN". RFC 8259 has no NaN literal and no numeric trick
//     recovers one. Re-parse restores NaN only for a float/double Default,
//     where the schema type drives the coercion; in Props the string stays a
//     string, since coercing it would reinterpret a user's intentional "NaN"
//     (Parse never puts NaN there).
//   - -0.0 → json.Number("-0.0"). Marshal renders "-0", integer syntax that
//     re-parses as +0 and flips the rebuilt default's sign.
//   - []byte → codepoint-per-byte string. Marshal base64s it and the Avro
//     parser reads that base64 text back as raw bytes, so {1,2,3} returns as
//     the four bytes of "AQID". Props []byte follow the same Avro convention;
//     users wanting base64 must pre-encode to a string.
func applyJSONFixup(v any) any {
	switch tv := v.(type) {
	case float64:
		if math.IsInf(tv, 1) {
			return json.Number("1e1000")
		}
		if math.IsInf(tv, -1) {
			return json.Number("-1e1000")
		}
		if math.IsNaN(tv) {
			return "NaN"
		}
		if isNegativeZero(tv) {
			return json.Number("-0.0")
		}
		return tv
	case float32:
		if math.IsInf(float64(tv), 1) {
			return json.Number("1e1000")
		}
		if math.IsInf(float64(tv), -1) {
			return json.Number("-1e1000")
		}
		if math.IsNaN(float64(tv)) {
			return "NaN"
		}
		if isNegativeZero(float64(tv)) {
			return json.Number("-0.0")
		}
		return tv
	case []byte:
		return bytesToAvroJSONString(tv)
	case map[string]any:
		out := make(map[string]any, len(tv))
		for k, val := range tv {
			out[k] = applyJSONFixup(val)
		}
		return out
	case []any:
		out := make([]any, len(tv))
		for i, val := range tv {
			out[i] = applyJSONFixup(val)
		}
		return out
	}
	return applyJSONFixupKind(v)
}

// applyJSONFixupKind is needsJSONFixupKind's conversion twin: it rebuilds
// the caller-typed value in canonical shape with the same fixups the
// exact-type arms apply, leaving marshal-opaque values and the
// no-canonical-twin residuals untouched. A named float NaN deliberately
// falls through un-fixed (needsJSONFixupKind never selects it) so the
// marshal error stays loud.
func applyJSONFixupKind(v any) any {
	if v == nil || treeValueMarshalOpaque(v) {
		return v
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Float64, reflect.Float32:
		f := rv.Float()
		switch {
		case math.IsInf(f, 1):
			return json.Number("1e1000")
		case math.IsInf(f, -1):
			return json.Number("-1e1000")
		case isNegativeZero(f):
			return json.Number("-0.0")
		}
		return v
	case reflect.Slice:
		if canonicalByteSliceKind(rv.Type()) {
			b := make([]byte, rv.Len())
			for i := range b {
				b[i] = byte(rv.Index(i).Uint())
			}
			return bytesToAvroJSONString(b)
		}
		fallthrough
	case reflect.Array:
		if sliceElemMarshalPositionDependent(rv.Type().Elem()) {
			return v
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = applyJSONFixup(rv.Index(i).Interface())
		}
		return out
	case reflect.Map:
		if !canonicalStringKeyMap(rv.Type()) {
			return v
		}
		out := make(map[string]any, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			out[it.Key().String()] = applyJSONFixup(it.Value().Interface())
		}
		return out
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return applyJSONFixup(rv.Elem().Interface())
	}
	return v
}

// maxSchemaJSONNodes bounds the COUNT of JSON nodes one SchemaNode→JSON walk
// emits; walkBudget explains why counting is necessary. Far above any real
// schema's node count, so a usable tree is never rejected.
const maxSchemaJSONNodes = 1 << 20

// maxSchemaJSONBytes bounds the total SIZE of scalar payload one walk emits;
// see walkBudget. Far above any real schema's serialized size.
const maxSchemaJSONBytes = 1 << 26

// walkBudget is the per-walk resource budget threaded through toJSONWalk and
// valueWalkLimit. Both axes decrement across the WHOLE walk — structural
// nodes, every Props value and SchemaField.Default, and the dedup
// conflict-comparison marshals — so no single channel can blow either.
// maxSchemaJSONDepth guards a third axis, and neither of these is redundant
// with it:
//
//   - nodes counts emitted JSON nodes (objects plus array elements, enum
//     symbols and aliases included). Depth caps the longest container PATH but
//     cannot see a shared-reference DAG: one *SchemaNode reached through both
//     Items and Values, or one sub-value under two map keys, is tiny in memory
//     yet fans out into an exponential TREE when serialized, because neither
//     toJSONWalk nor json.Marshal memoizes shared references. A ~40-node DAG
//     demands 2^40 emitted nodes and OOMs before Schema's eventual Parse —
//     whose depth pre-scan would have rejected the JSON — ever runs.
//   - bytes counts emitted scalar payload. The node count cannot see a leaf's
//     SIZE: the tree holds strings and []string BY REFERENCE, so assigning a
//     multi-megabyte Doc charges exactly one node while json.Marshal re-expands
//     it. K nodes sharing one L-byte string are O(K+L) in memory, K*L in the
//     output.
//
// An over-budget walk stops with a clean error (dedup path) or a truncated
// payload (bare path) rather than hanging, matching the depth bound's posture.
type walkBudget struct {
	nodes int
	bytes int
}

// newWalkBudget returns a fresh full budget for one SchemaNode→JSON walk.
func newWalkBudget() walkBudget {
	return walkBudget{nodes: maxSchemaJSONNodes, bytes: maxSchemaJSONBytes}
}

// takeNode charges one emitted node, reporting false when the node budget is
// already exhausted (this path never drives it below zero).
func (b *walkBudget) takeNode() bool {
	if b.nodes <= 0 {
		return false
	}
	b.nodes--
	return true
}

// takeNodes charges n emitted nodes (a slice's elements). When n exceeds the
// remainder the budget is driven to zero and false is reported, so the caller
// truncates rather than handing json.Marshal a giant array.
func (b *walkBudget) takeNodes(n int) bool {
	if n > b.nodes {
		b.nodes = 0
		return false
	}
	b.nodes -= n
	return true
}

// takeBytes charges n emitted payload bytes. Over-budget drives it NEGATIVE —
// so toJSONWalk's top-of-call check and valueWalkLimit both observe exhaustion
// — and reports false, keeping the payload away from json.Marshal.
func (b *walkBudget) takeBytes(n int) bool {
	if n > b.bytes {
		b.bytes = -1
		return false
	}
	b.bytes -= n
	return true
}

// emitString charges a structural scalar string's bytes, returning it for
// emission, or "" (recording the over-budget error) when the byte budget is
// exhausted — so json.Marshal never copies a payload past the bound.
func (b *walkBudget) emitString(d *deduper, s string) string {
	if b.takeBytes(len(s)) {
		return s
	}
	d.fail(errSchemaTreeBytes())
	return ""
}

// emitStrings charges a structural []string payload — element COUNT against
// the node budget, content bytes against the byte budget — returning it, or an
// empty slice (recording the over-budget error) when either is exhausted. The
// truncation is deterministic so the dedup conflict comparison stays
// meaningful: asymmetric truncation would make identical bodies compare
// unequal, the hazard toJSONShared also addresses. Exhaustion is reported by
// the post-comparison check, not as a spurious body conflict.
func (b *walkBudget) emitStrings(d *deduper, ss []string) []string {
	if !b.takeNodes(len(ss)) {
		d.fail(errSchemaTreeNodes())
		return []string{}
	}
	total := 0
	for _, s := range ss {
		total += len(s)
	}
	if !b.takeBytes(total) {
		d.fail(errSchemaTreeBytes())
		return []string{}
	}
	return ss
}

// fail records err as the deduper's first error. It is a no-op on the bare
// (d == nil) walk, which truncates over-budget output silently, and never
// overwrites an earlier error.
func (d *deduper) fail(err error) {
	if d != nil && d.err == nil {
		d.err = err
	}
}

func errSchemaTreeNodes() error {
	return fmt.Errorf("avro: SchemaNode tree expands to more than the supported %d nodes", maxSchemaJSONNodes)
}

func errSchemaTreeBytes() error {
	return fmt.Errorf("avro: SchemaNode tree expands to more than the supported %d bytes", maxSchemaJSONBytes)
}

// valueWalkLimit result codes.
const (
	valueWalkOK        = iota // safe to hand to jsonSerializableValue / json.Marshal
	valueWalkTooDeep          // nests past the depth budget (stack-overflow risk)
	valueWalkTooWide          // expands to too many nodes (fan-out / json.Marshal cost)
	valueWalkTooLarge         // expands to too many payload bytes (json.Marshal output size)
	valueWalkBadMapKey        // a map key json.Marshal's key resolver cannot name
)

// marshalEmitLen reports how many bytes json.Marshal will emit for a value
// that defines its own JSON form, and whether it is such a value at all.
// json.Marshal consults json.Marshaler first, then encoding.TextMarshaler,
// so this checks them in that order. json.Number is deliberately NOT here:
// it is a string-KIND value the String arm already charges by content.
//
// Measuring costs one call to the caller's own method, charged and dropped.
// The walk stops at the first value that busts the budget, so a tree of N
// over-budget marshalers materializes one image, not N, and retains none.
//
// A method returning an error is left uncharged and unhandled: json.Marshal
// will surface that same error, and inventing a budget verdict for a value
// that will never be emitted would reject the tree for the wrong reason.
func marshalEmitLen(rv reflect.Value, limit int) (int, bool) {
	if !rv.IsValid() || !rv.CanInterface() {
		return 0, false
	}
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return 0, false // json.Marshal emits "null" without calling the method
		}
	}
	switch m := rv.Interface().(type) {
	case json.Marshaler:
		out, err := m.MarshalJSON()
		if err != nil {
			return 0, false
		}
		return compactedEmitLen(out, limit), true
	case encoding.TextMarshaler:
		out, err := m.MarshalText()
		if err != nil {
			return 0, false
		}
		// Emitted as a quoted JSON string, through the same escaper.
		return jsonEscapedLenBytes(out, limit) + 2, true
	}
	return 0, false
}

// marshalSchemaTree is the ONE call that turns a rendered schema tree into
// bytes. The walk budget charges against exactly this emitter's escaping and
// the census differential derives its expectation from it, so a change here
// — an Encoder with SetEscapeHTML(false), say — moves the charge and the
// test that proves the charge together, instead of silently parting them.
func marshalSchemaTree(tree any) ([]byte, error) { return json.Marshal(tree) }

// asciiEscapedLen is the emitted length of one byte below utf8.RuneSelf.
// Escaping is byte-LOCAL below RuneSelf — a byte's cost never depends on its
// neighbours — so this table plus the multi-byte arms below is a complete
// description of the emitter's string output, and testing all 256 values is
// a domain proof rather than a sample.
func asciiEscapedLen(b byte) int {
	switch b {
	case '\\', '"', '\b', '\f', '\n', '\r', '\t':
		return 2 // two-character escape
	case '<', '>', '&':
		return 6 // < — the emitter escapes HTML
	}
	if b < 0x20 {
		return 6 // \u00XX
	}
	return 1
}

// jsonEscapedLen reports how many bytes the emitter writes for s's CONTENT
// (what lands between the quotes), and stops once the running total passes
// limit, returning a value greater than it.
//
// It COUNTS; it never builds — measuring by emitting would allocate the image
// the budget exists to prevent. That is the one reason the escape rules are
// restated here instead of delegated to marshalSchemaTree, and the restatement
// is held to it by an executed differential over the emitter's complete
// single-byte domain plus every multi-byte case (census Q9).
//
// The early exit bounds the scan by the BUDGET rather than the input: escaping
// never shrinks a string, so the total passes limit within limit+1 input
// bytes. A hostile 1 GiB string is abandoned after ~64 MiB.
func jsonEscapedLen(s string, limit int) int {
	n := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			n += asciiEscapedLen(b)
			i++
		} else {
			c, size := utf8.DecodeRuneInString(s[i:])
			switch {
			case c == utf8.RuneError && size == 1:
				n += 6 // invalid UTF-8 is emitted as �
			case c == ' ' || c == ' ':
				n += 6 // escaped unconditionally, JSONP safety
			default:
				n += size // emitted verbatim
			}
			i += size
		}
		if n > limit {
			return n
		}
	}
	return n
}

// jsonEscapedLenBytes is jsonEscapedLen over a byte slice, for text a
// TextMarshaler returned (the emitter runs it through the same escaper).
func jsonEscapedLenBytes(s []byte, limit int) int {
	n := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			n += asciiEscapedLen(b)
			i++
		} else {
			c, size := utf8.DecodeRune(s[i:])
			switch {
			case c == utf8.RuneError && size == 1:
				n += 6
			case c == ' ' || c == ' ':
				n += 6
			default:
				n += size
			}
			i += size
		}
		if n > limit {
			return n
		}
	}
	return n
}

// avroCodepointEscapedLen is the emitted length of a byte slice that the JSON
// fixup renders as the Avro codepoint string: byte v becomes U+00v, so a byte
// at or above 0x80 costs the two bytes of its UTF-8 form and everything below
// costs what the ASCII table says. Charging the value's json-FACING image
// rather than its Go shape is the rule: a []byte never reaches the emitter as
// a byte slice.
func avroCodepointEscapedLen(rv reflect.Value, limit int) int {
	n := 0
	for i := range rv.Len() {
		if b := byte(rv.Index(i).Uint()); b < utf8.RuneSelf {
			n += asciiEscapedLen(b)
		} else {
			n += 2
		}
		if n > limit {
			return n
		}
	}
	return n
}

// compactedEmitLen upper-bounds what the emitter writes for JSON a
// json.Marshaler returned. That output is re-scanned by encoding/json's
// compactor, which escapes <, > and & (one byte becoming six) and U+2028 /
// U+2029 (three becoming six), and drops insignificant whitespace. Only the
// GROWTH is counted: ignoring the shrinkage over-charges slightly, which is
// the safe direction for a cap, and it avoids re-running a JSON scanner.
func compactedEmitLen(out []byte, limit int) int {
	n := len(out)
	for i := 0; i < len(out); i++ {
		switch c := out[i]; {
		case c == '<' || c == '>' || c == '&':
			n += 5
		case c == 0xE2 && i+2 < len(out) && out[i+1] == 0x80 && out[i+2]&^1 == 0xA8:
			n += 3
		}
		if n > limit {
			return n
		}
	}
	return n
}

// mapKeyEmitLen reports the bytes json.Marshal emits for one map key, and
// whether its key resolver can name the key at all.
//
// The arms mirror encoding/json's resolveKeyName IN ORDER, GUARDS INCLUDED —
// string KIND first (marshals raw; its MarshalText is not consulted), then
// encoding.TextMarshaler, then integer formatting. Modeling only the
// convenient arms would leave the rest free of charge.
//
// The nil-pointer guard is the one that matters: for a nil pointer key whose
// type has a pointer-receiver MarshalText, json.Marshal resolves "" WITHOUT
// calling the method. Calling it dereferences nil — a panic raised inside the
// very walk that exists to make an arbitrary tree safe to marshal.
//
// The final arm is resolveKeyName's `panic("unexpected map key type")`,
// reachable via a nil interface key in a map[encoding.TextMarshaler]V, which
// json's encoder-construction check admits (the interface implements itself)
// and its resolver then cannot name. The walk returns a named error instead,
// as it does for the key kinds json rejects at construction (float, array, a
// struct with no text method): every key is accounted for, so "json cannot
// emit this key" is a verdict the walk owns rather than a panic to forward.
func mapKeyEmitLen(k reflect.Value, limit int) (int, bool) {
	if k.Kind() == reflect.String {
		return jsonEscapedLen(k.String(), limit), true
	}
	if k.CanInterface() {
		if tm, ok := k.Interface().(encoding.TextMarshaler); ok {
			if k.Kind() == reflect.Pointer && k.IsNil() {
				return 0, true // resolved to "" without calling the method
			}
			out, err := tm.MarshalText()
			if err != nil {
				// Left uncharged, like marshalEmitLen: the eventual
				// json.Marshal surfaces this same error by its own name.
				return 0, true
			}
			return jsonEscapedLenBytes(out, limit), true
		}
	}
	switch k.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		return 20, true // a signed 64-bit decimal is at most 20 bytes
	}
	return 0, false
}

// valueWalkLimit walks an arbitrary user-supplied Props value or
// SchemaField.Default the way json.Marshal will, returning a non-OK code when
// it is unsafe to serialize. Neither jsonSerializableValue nor json.Marshal
// bounds anything, so three orthogonal limits are enforced here:
//
//   - DEPTH (depthLeft): the longest container PATH. Nested far enough, the
//     fixup walk or json.Marshal overflows the stack uncatchably — recover
//     cannot catch that — before Schema's eventual Parse can reject it.
//     depthLeft charges the structural nesting already accrued, so the total
//     marshaled nesting stays within toJSONWalk's one ceiling.
//   - EXPANSION (b.nodes, shared with toJSONWalk): the nodes json.Marshal will
//     emit. A value sharing a sub-value across sibling paths is shallow yet
//     fans out into a 2^depth tree (see walkBudget), invisible to the depth
//     bound. Decrementing on EVERY node also terminates the walk itself, so it
//     can neither overflow its own stack nor hang on a shared-reference DAG or
//     a cyclic Go type (type P *P).
//   - PAYLOAD SIZE (b.bytes, shared likewise): every emitted scalar — string
//     and json.Number content, []byte content, map keys, struct field names.
//     Huge or widely-shared leaves are small in memory yet expand past it in
//     the output, invisible to the node count.
//
// The walk mirrors what json.Marshal recurses into — maps, slices, arrays,
// structs, pointer/interface indirection — not just the map[string]any / []any
// shapes [Schema.Root] produces, since a hand-built node or a SchemaFor
// CustomType.Schema can store ANY Go value. []byte/[N]byte is a scalar,
// charged by length rather than walked as a nested array.
func valueWalkLimit(rv reflect.Value, depthLeft int, b *walkBudget) int {
	if depthLeft < 0 {
		return valueWalkTooDeep
	}
	if b.bytes < 0 {
		return valueWalkTooLarge
	}
	if !b.takeNode() {
		return valueWalkTooWide
	}
	// json.Marshal never walks a value carrying its own MarshalJSON /
	// MarshalText — the method's return IS the emission. Recursing here would
	// charge the Go shape while json emits something else entirely (an empty
	// struct whose MarshalJSON returns a megabyte charges one node, no bytes).
	// Charge what the method emits and stop, mirroring json's own dispatch.
	// Charging reads the output and discards it, so rendering is unchanged.
	if n, ok := marshalEmitLen(rv, b.bytes); ok {
		if !b.takeBytes(n) {
			return valueWalkTooLarge
		}
		return valueWalkOK
	}
	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer:
		if rv.IsNil() {
			return valueWalkOK
		}
		return valueWalkLimit(rv.Elem(), depthLeft-1, b)
	case reflect.Map:
		for iter := rv.MapRange(); iter.Next(); {
			// json emits EVERY map key, whatever its Kind — string-kind raw,
			// anything else via MarshalText or integer formatting — so all of
			// them must be charged. See mapKeyEmitLen.
			n, ok := mapKeyEmitLen(iter.Key(), b.bytes)
			if !ok {
				return valueWalkBadMapKey
			}
			if !b.takeBytes(n) {
				return valueWalkTooLarge
			}
			if r := valueWalkLimit(iter.Value(), depthLeft-1, b); r != valueWalkOK {
				return r
			}
		}
	case reflect.Slice, reflect.Array:
		if canonicalByteSliceKind(rv.Type()) || (rv.Kind() == reflect.Array && rv.Type().Elem().Kind() == reflect.Uint8) {
			// The JSON fixup renders these as the Avro codepoint STRING, so
			// that is the image to charge — not the Go length. The gate is
			// the fixup's OWN predicate, so a byte slice whose element type
			// carries a marshaler (which the fixup declines to rewrite, and
			// json emits as an ARRAY) falls through to the walk below
			// instead of being charged as a scalar it never becomes.
			if !b.takeBytes(avroCodepointEscapedLen(rv, b.bytes)) {
				return valueWalkTooLarge
			}
			return valueWalkOK
		}
		for i := 0; i < rv.Len(); i++ {
			if r := valueWalkLimit(rv.Index(i), depthLeft-1, b); r != valueWalkOK {
				return r
			}
		}
	case reflect.Struct:
		t := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue // json.Marshal skips unexported fields
			}
			// json.Marshal emits the field name (or json tag) as an object key.
			if !b.takeBytes(len(t.Field(i).Name)) {
				return valueWalkTooLarge
			}
			if r := valueWalkLimit(rv.Field(i), depthLeft-1, b); r != valueWalkOK {
				return r
			}
		}
	case reflect.String:
		// string AND json.Number (type Number string) — charge what the
		// emitter WRITES, which is the escaped form: a control byte costs
		// six output bytes, and the emitter escapes HTML.
		if !b.takeBytes(jsonEscapedLen(rv.String(), b.bytes)) {
			return valueWalkTooLarge
		}
	}
	return valueWalkOK
}

// boundedSerializableValue applies jsonSerializableValue to a Props value or
// SchemaField.Default after bounding it through valueWalkLimit. depth is the
// structural nesting toJSONWalk has already accrued, so the value may add at
// most maxSchemaJSONDepth-depth further levels. Exceeding any bound records the
// error on the dedup path (so [SchemaNode.Schema] returns it) and truncates to
// nil on the bare path, matching toJSONWalk's own over-limit handling.
func boundedSerializableValue(d *deduper, depth int, b *walkBudget, v any) any {
	switch valueWalkLimit(reflect.ValueOf(v), maxSchemaJSONDepth-depth, b) {
	case valueWalkTooDeep:
		d.fail(fmt.Errorf("avro: SchemaNode default/property value nests deeper than the supported limit (%d)", maxSchemaJSONDepth))
		return nil
	case valueWalkTooWide:
		d.fail(fmt.Errorf("avro: SchemaNode default/property value expands to more than the supported %d nodes", maxSchemaJSONNodes))
		return nil
	case valueWalkTooLarge:
		d.fail(fmt.Errorf("avro: SchemaNode default/property value expands to more than the supported %d bytes", maxSchemaJSONBytes))
		return nil
	case valueWalkBadMapKey:
		d.fail(errors.New("avro: SchemaNode default/property value contains a map whose key type has no JSON object-key form (not a string kind, an integer kind, or a usable encoding.TextMarshaler)"))
		return nil
	}
	return jsonSerializableValue(v)
}

// toJSONShared snapshots n's full JSON body (no dedup) for toJSONWalk's
// conflict comparison, charging the SHARED budget rather than a fresh one. A
// named type re-occurring as a DISTINCT pointer with an identical body costs a
// 2x re-marshal of its subtree — O(k*w) for k copies of a w-node body — while
// the dedup walk charges only 1 node per re-occurrence (it emits a reference,
// not the body). The outer budget alone would leave k*w unbounded even though
// the emitted schema is tiny. Sharing caps the comparison work too. On
// exhaustion the walk returns truncated output and the caller reports
// over-budget rather than a spurious conflict, since asymmetric truncation of
// n vs prev could make identical bodies compare unequal.
func (n *SchemaNode) toJSONShared(b *walkBudget) any {
	return n.toJSONWalk(make(map[*SchemaNode]struct{}), nil, "", 0, b, false)
}

// toJSONWalk is the cycle-aware walker shared by toJSON and toJSONDedup.
// visited threads through every recursive call so Items/Values pointer cycles
// terminate. A non-nil d tracks named-type definitions and reports conflicting
// redefinitions; nil just emits the tree. enclosingNS is the scope at this
// node's position: named types emit their namespace relative to it — omitted
// when inherited, "namespace":"" for a null-namespace type inside a namespaced
// scope — and name references emit the fullname so they re-bind
// position-independently.
//
// depth is the structural nesting level. visited terminates only POINTER
// cycles, and a distinct-node-per-level chain (a hand-built array<array<…>> a
// million deep) repeats none, so without this bound the walk overflows the
// stack uncatchably before Parse's own bracket-nesting bound ever runs. The
// same maxSchemaJSONDepth ceiling applies here; any tree shallow enough to
// encode or decode sits far below it, since the codec's maxDepth is 4x smaller.
//
// stray is true when n was reached through a structural key its parent's kind
// does not bind (a stray "items" on an "int", surfaced as-written by the
// metadata walker), and transitively for everything below it. The wire parser
// binds no names at those positions, so the walk renders them verbatim while
// the dedup consult skips them entirely — no registration, no
// second-definition→reference rewrite, no conflict comparison. Otherwise a
// definition-shaped stray body would stand in for, or spuriously conflict
// with, the real definition of that fullname, and the rebuilt text would
// either fail to re-parse or silently rewrite the as-written stray content.
func (n *SchemaNode) toJSONWalk(visited map[*SchemaNode]struct{}, d *deduper, enclosingNS string, depth int, b *walkBudget, stray bool) any {
	if depth > maxSchemaJSONDepth {
		d.fail(fmt.Errorf("avro: SchemaNode tree nests deeper than the supported limit (%d)", maxSchemaJSONDepth))
		return nil
	}
	// Charge this node against the shared budget; walkBudget explains why the
	// depth bound alone is not enough. Once either axis is exhausted every
	// further node returns early without descending, pruning at the frontier.
	if b.bytes < 0 {
		d.fail(errSchemaTreeBytes())
		return nil
	}
	if !b.takeNode() {
		d.fail(errSchemaTreeNodes())
		return nil
	}
	// Charge type / name / namespace BEFORE they are hashed into the dedup map,
	// scanned, or emitted as a reference, so a huge shared Name/Namespace
	// cannot amplify per occurrence. The type switches below short-circuit on
	// length, so charging once here covers every later emission of the three.
	if !b.takeBytes(len(n.Type) + len(n.Name) + len(n.Namespace)) {
		d.fail(errSchemaTreeBytes())
		return nil
	}
	if _, cycle := visited[n]; cycle {
		// Cycle through Items/Values back to n. Named types emit the fullname
		// as a reference (the canonical Avro recursive shape); unnamed cycles
		// are an error in the dedup walker and nil-stable JSON in the bare one,
		// so two equal cyclic subtrees still compare equal. Keyed on the
		// FULLNAME being expressible, not the short name: fullname "ns." (an
		// empty short name with a namespace) is a valid reference target, while
		// fullname "" has no spelling and stays the cycle error. A
		// stray-reached name registers nothing, so it takes the error path too.
		if isNamedKind(n.Type) && nodeFullname(n) != "" && !stray {
			return nodeFullname(n)
		}
		if d != nil && d.err == nil {
			d.err = fmt.Errorf("avro: cyclic SchemaNode detected")
		}
		return nil
	}
	visited[n] = struct{}{}
	defer delete(visited, n)

	// Dedup: an already-emitted named type becomes a name ref; a redefinition
	// with a different body is a conflict. Keyed by FULLNAME, since name
	// equality is defined on the fullname (spec, "Names") — two types sharing a
	// short name across namespaces are not redefinitions. The reference is the
	// fullname too: dotted re-binds exactly anywhere, and a null-namespace
	// type's bare fullname re-binds via the parser's null-namespace fallback.
	// A bare reference from inside a namespaced scope that also collides with
	// an in-scope short name re-binds in-scope — the same ambiguity Java's
	// getQualified/Names.get pair has, since references have no "namespace":""
	// escape.
	if d != nil && !stray && isNamedKind(n.Type) && nodeFullname(n) != "" {
		if prev, exists := d.defined[nodeFullname(n)]; exists {
			// Marshal-compare the bodies only for DISTINCT nodes: a named type
			// referenced repeatedly resolves to the same *SchemaNode and is
			// definitionally equal. Deferring to an actual collision keeps the
			// common case O(n) instead of eagerly marshaling every named type's
			// subtree. The comparison uses toJSONShared, not toJSON, so many
			// identical-bodied duplicates stay inside the budget.
			if prev != n && d.err == nil {
				cur, _ := json.Marshal(n.toJSONShared(b))
				prevB, _ := json.Marshal(prev.toJSONShared(b))
				switch {
				case b.nodes <= 0:
					// Budget exhausted mid-comparison: the bodies are
					// truncated, so comparing them is meaningless. Report
					// over-budget rather than risk a spurious conflict from
					// asymmetric truncation.
					d.err = errSchemaTreeNodes()
				case b.bytes < 0:
					d.err = errSchemaTreeBytes()
				case string(cur) != string(prevB):
					d.err = fmt.Errorf("avro: conflicting definitions for named type %q", truncForError(nodeFullname(n)))
				}
			}
			return nodeFullname(n)
		}
	}

	// The namespace scope inside this node: a named type opens its own.
	childNS := nsForChildren(n, enclosingNS)

	// Name-reference resolution (the hidden Root stamp): a reference to a name
	// this tree does not define emits the stamped definition at its first
	// occurrence — through this same budgeted, cycle-checked, deduped
	// recursion — and the fullname thereafter, so an extracted subtree is
	// self-contained. Gated on d != nil, so conflict snapshots stay
	// splice-free on both sides, and on !stray, since the wire parser binds no
	// names at stray positions. A reference the tree DOES define locally,
	// before or after this position, stays as-written and re-binds locally.
	refType := n.Type
	if d != nil && !stray && nodeRefTargetAgrees(n) && nodeIsNameRefShape(n) {
		if fn := nodeFullname(n.refTarget); fn != "" && !d.localNames[fn] {
			if _, emitted := d.defined[fn]; !emitted {
				// FRESH visited map: a recursive definition reaches back
				// through the extraction point (splicing Node re-enters
				// the union the outer walk is still inside), which the
				// shared map's cycle arm would misread as an unnamed
				// cycle. The revisit is finite — the target registers in
				// d.defined before walking its children, so each name
				// splices once and interior revisits end at the fullname
				// arm. True cycles inside the target are still caught by
				// the fresh map, and the shared depth and node/byte
				// budgets bound the emission either way.
				spliced := n.refTarget.toJSONWalk(make(map[*SchemaNode]struct{}), d, enclosingNS, depth, b, false)
				// A wrapped reference's custom properties ride onto the
				// spliced definition, definition-wins and reserved keys
				// dropped — same as the SchemaCache splice's wrapper arm.
				if m2, ok := spliced.(map[string]any); ok && len(n.Props) > 0 {
					defTyp, _ := m2["type"].(string)
					defLogical, _ := m2["logicalType"].(string)
					for k, v := range n.Props {
						if !b.takeBytes(len(k)) {
							d.fail(errSchemaTreeBytes())
							continue
						}
						pv := boundedSerializableValue(d, depth, b, v)
						if schemaReservedKeyForObject(k, pv, defTyp, defLogical, nil) {
							continue
						}
						if _, has := m2[k]; has {
							continue
						}
						m2[k] = pv
					}
				}
				return spliced
			}
			// Already emitted (an earlier splice, or the walk passed the
			// definition): reference it by fullname — the spelling that
			// re-binds exactly regardless of the standalone parse's
			// namespace scope at this position.
			refType = fn
		}
	}

	switch n.Type {
	case "null", "boolean", "int", "long", "float", "double", "string", "bytes":
		// Bare-string emission is only lossless when the node carries
		// NOTHING but its Type — see nodeCarriesOnlyType, which derives that
		// from the field set rather than listing the fields it remembers.
		if nodeCarriesOnlyType(n) {
			return n.Type
		}
	case "union":
		branches := make([]any, len(n.Branches))
		for i := range n.Branches {
			branches[i] = n.Branches[i].toJSONWalk(visited, d, childNS, depth+1, b, stray)
		}
		return branches
	}

	// The same losslessness question as the primitive arm above, for a NAME
	// REFERENCE: it may collapse to the bare name only when the node carries
	// nothing else. Both sites ask nodeCarriesOnlyType rather than repeating
	// a field list: two hand-written copies drift, and one that misses a
	// field silently drops the stray attribute stored there.
	if n.Type != "array" && n.Type != "map" && !isNamedKind(n.Type) &&
		n.Type != "union" && nodeCarriesOnlyType(n) {
		return refType
	}

	// Remember this named type's node for the next occurrence's conflict
	// check. Store the node, not its marshaled body: eager marshaling is
	// O(depth*subtree), and the body is needed only if a duplicate appears.
	if d != nil && !stray {
		// Fullname-keyed like the duplicate check above. Fullname "" has no
		// reference spelling so it stays un-deduped, and stray-reached names
		// register nothing — the wire parser binds neither.
		if isNamedKind(n.Type) && nodeFullname(n) != "" {
			d.defined[nodeFullname(n)] = n
		}
	}

	m := map[string]any{"type": refType}
	// A named KIND always emits its name, the empty short name a WithLaxNames
	// fn can accept included — matching appendCanonObject and the parser, for
	// which a missing and an empty name are the same fullname. The Name != ""
	// arm keeps emission for hand-built names on non-named kinds.
	if n.Name != "" || isNamedKind(n.Type) || n.present.has(presName) {
		m["name"] = n.Name
	}
	if isNamedKind(n.Type) && !strings.Contains(n.Name, ".") {
		// Namespace relative to the enclosing scope, mirroring Java's
		// Name.writeName: omitted when equal (re-parse inherits it),
		// "namespace":"" to escape inheritance for a null-namespace type in a
		// namespaced scope, the value otherwise. A dotted Name carries its own
		// namespace and gets no attribute — the spec ignores it there.
		switch eff := n.Namespace; {
		case eff == enclosingNS:
			// inherited (or both null): omit
		case eff == "":
			m["namespace"] = ""
		default:
			m["namespace"] = eff
		}
	} else if (n.Namespace != "" || n.present.has(presNamespace)) && !isNamedKind(n.Type) {
		// Unnamed node with a namespace attribute: preserve as-written
		// (the parser ignores it; fidelity only). Presence carries the
		// explicit-empty form, which the value alone cannot show.
		m["namespace"] = n.Namespace
	}
	// Where a kind BINDS aliases, Apache Avro's condition is non-EMPTY
	// (Schema.java:886) and an empty list is deliberately dropped. Where it
	// does not bind, there is no such condition and the stray-routing posture
	// applies: as-written is the key's only surface, so presence decides.
	if len(n.Aliases) > 0 || (n.present.has(presAliases) && !strayKeyBinds(n.Type, "aliases")) {
		m["aliases"] = b.emitStrings(d, n.Aliases)
	}
	// doc emits when WRITTEN, not when non-empty: an empty doc is a doc, and
	// Apache Avro emits it (Schema.java:1039/:1154/:1367 ask getDoc() != nil).
	// That per-attribute difference from aliases is why presence is asked here.
	if n.Doc != "" || n.present.has(presDoc) {
		m["doc"] = b.emitString(d, n.Doc)
	}
	if n.HasEnumDefault {
		m["default"] = b.emitString(d, n.EnumDefault)
	}
	// logicalType is not in Java's reserved set, so it survives as an
	// ordinary schema property whatever its content, empty string included
	// (parseProperties :1983, writeProps).
	if n.LogicalType != "" || n.present.has(presLogicalType) {
		m["logicalType"] = b.emitString(d, n.LogicalType)
	}
	if n.Precision != 0 {
		m["precision"] = n.Precision
	}
	if n.Scale != 0 {
		m["scale"] = n.Scale
	}
	// size is required on fixed and 0 is a legal size, so omitting the zero
	// would make the re-emitted schema unparseable. On any other kind it is a
	// stray, surfaced as-written — the same required-or-as-written shape the
	// symbols and fields rules below use.
	if n.Type == "fixed" || n.Size != 0 || n.present.has(presSize) {
		m["size"] = n.Size
	}
	// symbols is required on enum (spec, Complex Types > Enums), so it emits
	// even when empty — and as [] rather than null, which is unparseable.
	if n.Type == "enum" {
		if n.Symbols == nil {
			m["symbols"] = []string{}
		} else {
			m["symbols"] = b.emitStrings(d, n.Symbols)
		}
	} else if len(n.Symbols) > 0 || n.present.has(presSymbols) {
		m["symbols"] = b.emitStrings(d, n.Symbols)
	}
	if n.Items != nil {
		m["items"] = n.Items.toJSONWalk(visited, d, childNS, depth+1, b, stray || n.Type != "array")
	}
	if n.Values != nil {
		m["values"] = n.Values.toJSONWalk(visited, d, childNS, depth+1, b, stray || n.Type != "map")
	}
	// fields is required on record/error (spec, Complex Types > Records), so
	// it emits even when empty.
	if isRecordKind(n.Type) || len(n.Fields) > 0 || n.present.has(presFields) {
		fieldStray := stray || !isRecordKind(n.Type)
		fields := make([]map[string]any, len(n.Fields))
		for i, f := range n.Fields {
			fd := map[string]any{
				"name": b.emitString(d, f.Name),
				"type": f.Type.toJSONWalk(visited, d, childNS, depth+1, b, fieldStray),
			}
			if f.HasDefault || f.Default != nil {
				// Inverse of the metadata-API normalization: a Root() of
				// "default":1e1000 holds ±Inf, which json.Marshal rejects,
				// so applyJSONFixup puts the literal back.
				fd["default"] = boundedSerializableValue(d, depth, b, f.Default)
			}
			if len(f.Aliases) > 0 {
				fd["aliases"] = b.emitStrings(d, f.Aliases)
			}
			if f.Order != "" {
				fd["order"] = b.emitString(d, f.Order)
			}
			if f.Doc != "" || f.docSet {
				fd["doc"] = b.emitString(d, f.Doc)
			}
			for k, v := range f.Props {
				// The Props KEY is emitted as a JSON object key; charge it, then
				// the value through the depth+node+byte-bounded value walk.
				if !b.takeBytes(len(k)) {
					d.fail(errSchemaTreeBytes())
					continue
				}
				fd[k] = boundedSerializableValue(d, depth, b, v)
			}
			fields[i] = fd
		}
		m["fields"] = fields
	}
	for k, v := range n.Props {
		if !b.takeBytes(len(k)) {
			d.fail(errSchemaTreeBytes())
			continue
		}
		m[k] = boundedSerializableValue(d, depth, b, v)
	}
	return m
}

// nodeFromJSON converts a parsed JSON value into a SchemaNode. parentNS
// is the enclosing namespace scope; named types without an explicit
// "namespace" attribute resolve into it (see [SchemaNode].Namespace).
func nodeFromJSON(v any, parentNS string, memo strayShapeMemo) SchemaNode {
	switch s := v.(type) {
	case string:
		return SchemaNode{Type: s}
	case []any:
		branches := make([]SchemaNode, len(s))
		for i, b := range s {
			branches[i] = nodeFromJSON(b, parentNS, memo)
		}
		return SchemaNode{Type: "union", Branches: branches}
	case map[string]any:
		return nodeFromJSONObject(s, parentNS, memo)
	default:
		return SchemaNode{}
	}
}

// Known schema keys that are NOT custom properties.
var schemaReservedKeys = map[string]bool{
	"type": true, "name": true, "namespace": true, "doc": true,
	"fields": true, "symbols": true, "items": true, "values": true,
	"size": true, "logicalType": true, "precision": true, "scale": true,
	"aliases": true, "default": true, "order": true,
}

// Known field keys that are NOT custom properties.
var fieldReservedKeys = map[string]bool{
	"name": true, "type": true, "default": true, "doc": true,
	"aliases": true, "order": true,
}

// jsonNumericInt accepts a value parsed via unmarshalAnyPreservePrecision
// (int64 for integer literals) and falls through to float64 / json.Number
// for compatibility with values originating from a bare encoding/json
// Unmarshal — primarily SchemaNode trees constructed programmatically
// and round-tripped through Schema().Root().
func jsonNumericInt(v any) (int, bool) {
	switch t := v.(type) {
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i), true
		}
	case string:
		// The Avro [INTEGERS] rule allows quoted-string integers; kept
		// for symmetry with laxInt even though the current callers
		// (precision/scale on a validated decimal carrier — size reads
		// through decodeLaxInt) never see the quoted form post-parse.
		if len(t) <= maxLaxIntDataLen {
			if i, err := strconv.Atoi(t); err == nil {
				return i, true
			}
		}
	}
	return 0, false
}

// getString reads m[key] into dst when the body is a JSON string, and
// reports whether it did. The bool is the attribute's PRESENCE as distinct
// from its value: an empty string sets dst to the field's own zero, so the
// field alone can no longer say whether the key was written.
//
// The lookup is by exact name: reserved attribute names match ONLY their
// exact lowercase spelling, and a case-variant key is an ordinary custom
// property (Java's reserved sets are exact-lowercase HashSets,
// Schema.java:175-176; fastavro and goavro read exact names too).
func getString(m map[string]any, key string, dst *string) bool {
	s, ok := m[key].(string)
	if ok {
		*dst = s
	}
	return ok
}

// getInt assigns *dst to m[key] when present and parseable via
// jsonNumericInt (precision/scale/size).
func getInt(m map[string]any, key string, dst *int) {
	if v, ok := m[key]; ok {
		if p, ok := jsonNumericInt(v); ok {
			*dst = p
		}
	}
}

// isRecordKind reports whether typ names the Avro record kind in
// [SchemaNode.Type]. "record" and "error" are both JSON literals for the same
// on-wire kind (the RPC convention names error-record types "error"), and the
// builder normalizes both to node.kind=="record". SchemaNode.Type keeps the
// name as written, so every metadata dispatcher branching on the record kind
// must accept either alias; centralizing it here stops the set drifting.
func isRecordKind(typ string) bool {
	return typ == "record" || typ == "error"
}

// isNamedKind reports whether typ is one of the four Avro named-type kinds
// (record / error / enum / fixed) — the set that carries a Name and can be
// referenced, deduped, and aliased. "error" is the record alias and always
// travels with "record". The named-type analogue of [isRecordKind].
func isNamedKind(typ string) bool {
	return typ == "record" || typ == "error" || typ == "enum" || typ == "fixed"
}

// coerceMetadataDefault is the metadata-API parallel of [coerceDefault]
// (schema.go): it puts a parsed-JSON default into the Go form the wire-encode
// pipeline materializes, so SchemaField.Default's type matches the wire bytes
// rather than the raw JSON shape.
//
//   - int/long/float/double → the schema-width Go type (int32/int64/float32/
//     float64), matching Java's JacksonUtils.toObject. Parse rejects
//     out-of-range integer defaults so the integer narrowing is lossless;
//     float32 narrowing of a finite overflow surfaces ±Inf, as the wire does.
//   - a string default on bytes/fixed → []byte via Avro's codepoint-per-byte
//     mapping, mirroring [convertDefaultBytes]. Without it, the natural
//     consumer pattern `defs[f.Name] = f.Default; s.Encode(defs)` works for
//     every bytes/fixed default EXCEPT fixed+uuid, whose encoder arm
//     hard-fails parseUUID on the 16-codepoint wire-form string.
//
// Unions are walked (Avro 1.12 lets a default match any branch), as are nested
// record/array/map types. A string-form numeric default coerces to float only
// on an outer float/double FIELD — Java parity with parseField's text→Double
// coercion — never on a union branch, matching Java, avro-rs, and goavro;
// union+numeric-string defaults are rejected at parse and never arrive here.
// Everything else passes through unchanged.
func coerceMetadataDefault(val any, t *SchemaNode, table map[string]*SchemaNode, ns string) any {
	if t == nil {
		return val
	}
	// Name-ref resolution: with a non-nil table, a bare name-reference Type
	// resolves to the named node and recurses inside the TARGET's own
	// namespace scope. table == nil is the best-effort inline pass during
	// nodeFromJSON, where the full tree — and so the name table — does not
	// exist yet.
	if resolved := lookupNameRef(t, table, ns); resolved != nil {
		return coerceMetadataDefault(val, resolved, table, nodeEffNS(resolved))
	}
	if t.Type == "union" {
		// On the best-effort first pass the name-referenced branches cannot be
		// resolved, so a greedy earlier branch — bytes accepting a string —
		// would destructively coerce string to []byte and lock out the enum
		// branch the table-populated pass would pick, since the enum arm takes
		// only a string. Defer all branch selection to coerceTreeDefaults.
		if table == nil {
			return val
		}
		// FIRST branch that accepts val's Go type, matching coerceDefault's
		// validateDefault selection and Java's parseField. "First branch that
		// TRANSFORMS" would diverge on ["string","float"] default "1.5": the
		// wire picks string, while a transform test picks float because
		// string→string looks like a no-op.
		if branch := firstMetadataBranchAcceptingDefault(t, val, table, ns); branch != nil {
			return coerceMetadataDefault(val, branch, table, nsForChildren(branch, ns))
		}
		return val
	}
	if val == nil {
		return val
	}
	if t.Type == "int" {
		// int defaults surface as int32 so Default's Go type matches both the
		// wire width and the user's natural field type.
		//
		// Every numeric form routes through the range-checked defaultAsInt32.
		// A top-level out-of-int32 default is rejected at parse, but during
		// union-branch SELECTION a wider sibling makes the schema parse-valid,
		// so this can run on a value parse never rejected. A blind int64→int32
		// cast would WRAP it (3000000000 → -1294967296) and
		// branchAcceptsDefault would then take the int branch the wire
		// rejects, picking a different branch than the wire auto-fill and
		// corrupting Root().Default and the rebuild with it. Leaving the
		// out-of-range value alone lets defaultAsInt32 reject it so selection
		// falls to the wider sibling, as the wire does.
		switch val := val.(type) {
		case int32:
			return val
		case int64, json.Number, string, float64:
			if n, err := defaultAsInt32(val); err == nil {
				return n
			}
			return val
		}
		return val
	}
	if t.Type == "long" {
		// Coerce non-int64 numeric inputs (json.Number, string,
		// float64-whole-number) to int64. int64 inputs pass through.
		// Schema parse rejects out-of-int64 via defaultAsInt64.
		switch val := val.(type) {
		case int64:
			return val
		case int32:
			return int64(val)
		case json.Number, string, float64:
			if n, err := defaultAsInt64(val); err == nil {
				return n
			}
			return val
		}
		return val
	}
	if t.Type == "float" || t.Type == "double" {
		// Width-faithful narrowing per Java's JacksonUtils.toObject: float
		// schema → float32, double → float64. Default is then exactly what the
		// wire encoder emits, finite overflows included ({"default":1e100,
		// "type":"float"} gives float32(+Inf), matching the wire bits) as well
		// as integer forms past the mantissa, which IEEE-round silently.
		//
		// Strings go through parseFloatAcceptOverflow inline rather than
		// [defaultAsFloat], which is strict (no string arm) so it stays
		// reusable at union-branch matching and encode. This arm is
		// [coerceDefault]'s parseField-style text→float coercion, outer
		// float/double only.
		var f float64
		switch val := val.(type) {
		case float64:
			f = val
		case float32:
			f = float64(val)
		case string:
			var err error
			if f, err = parseFloatAcceptOverflow(val, 64); err != nil {
				return val
			}
		case int64, int32, json.Number:
			var err error
			if f, err = defaultAsFloat(val); err != nil {
				return val
			}
		default:
			return val
		}
		if t.Type == "float" {
			return float32(f)
		}
		return f
	}
	if t.Type == "bytes" || t.Type == "fixed" {
		if s, ok := val.(string); ok {
			if b, err := avroJSONBytesToBytes(s); err == nil {
				return b
			}
		}
		return val
	}
	if isRecordKind(t.Type) {
		if m, ok := val.(map[string]any); ok {
			childNS := nsForChildren(t, ns)
			out := make(map[string]any, len(m))
			for k, v := range m {
				inner := v
				for i := range t.Fields {
					if t.Fields[i].Name == k {
						inner = coerceMetadataDefault(v, &t.Fields[i].Type, table, childNS)
						break
					}
				}
				out[k] = inner
			}
			return out
		}
		return val
	}
	if t.Type == "array" && t.Items != nil {
		if a, ok := val.([]any); ok {
			out := make([]any, len(a))
			for i, v := range a {
				out[i] = coerceMetadataDefault(v, t.Items, table, ns)
			}
			return out
		}
		return val
	}
	if t.Type == "map" && t.Values != nil {
		if m, ok := val.(map[string]any); ok {
			out := make(map[string]any, len(m))
			for k, v := range m {
				out[k] = coerceMetadataDefault(v, t.Values, table, ns)
			}
			return out
		}
		return val
	}
	return val
}

// nodeEffNS returns n's effective namespace: a dotted Name carries its
// own namespace (taking precedence per the spec); otherwise Namespace,
// which is already resolved ("" means the null namespace).
func nodeEffNS(n *SchemaNode) string {
	if i := strings.LastIndexByte(n.Name, '.'); i >= 0 {
		return n.Name[:i]
	}
	return n.Namespace
}

// nodeFullname returns n's fullname: the dotted Name verbatim, or the resolved
// namespace joined to the name. A single LEADING dot collapses per the
// null-namespace escape the parser normalizes at build (leadingDotName), so
// ".x" is the fullname "x" and "." is the bare empty name.
func nodeFullname(n *SchemaNode) string {
	if strings.Contains(n.Name, ".") {
		if short, ok := leadingDotName(n.Name); ok {
			return short
		}
		return n.Name
	}
	if n.Namespace != "" {
		return n.Namespace + "." + n.Name
	}
	return n.Name
}

// nsForChildren returns the namespace scope in effect inside n: a named
// type opens its own scope; unnamed nodes pass the enclosing scope
// through.
func nsForChildren(n *SchemaNode, enclosing string) string {
	// Named KINDS open their own scope even with an empty short name (a
	// user WithLaxNames fn can accept ""; nodeEffNS carries the resolved
	// namespace either way); the Name != "" arm keeps hand-built names on
	// non-named kinds scoping as before.
	if n != nil && (n.Name != "" || isNamedKind(n.Type)) {
		return nodeEffNS(n)
	}
	return enclosing
}

// lookupNameRef returns the named target of t if t.Type is a name
// reference (not a structural or primitive kind) AND table has it, else
// nil. A nil table always returns nil (synchronous-build callers disable
// name-ref resolution because the tree isn't fully walked yet). ns is
// the enclosing namespace scope at the reference site; the key order
// comes from scopedRefKeys (schema.go) so the metadata binding cannot
// drift from the wire's.
func lookupNameRef(t *SchemaNode, table map[string]*SchemaNode, ns string) *SchemaNode {
	if t == nil || table == nil {
		return nil
	}
	// Structural kinds (primitives, "record"/"error", "enum", "fixed",
	// "array", "map", "union") are schema definitions, not name-ref
	// targets. "error" is in this list per [isRecordKind] — without it,
	// a SchemaNode with t.Type=="error" would fall through to
	// table["error"] and wrongly resolve to any record literally named
	// "error" in the schema.
	switch t.Type {
	case "null", "boolean", "int", "long", "float", "double",
		"bytes", "string", "record", "error", "enum", "fixed", "array", "map", "union":
		return nil
	}
	var keys [2]string
	for _, k := range scopedRefKeys(&keys, t.Type, ns) {
		if r, ok := table[k]; ok {
			return r
		}
	}
	return nil
}

// fixupNameRefDefaults walks the SchemaNode tree once to populate a
// name-table of every reachable record/enum/fixed, then re-coerces
// HasDefault fields with the table so name-referenced defaults (and
// defaults whose union contains a name-ref branch) materialize the
// way inline-typed siblings already do via the synchronous coerce.
// The table is returned for Root's reference stamping, so both surfaces
// resolve through the identical name set.
func fixupNameRefDefaults(root *SchemaNode) map[string]*SchemaNode {
	table := map[string]*SchemaNode{}
	collectNamedTypes(root, table)
	if len(table) == 0 {
		return table
	}
	coerceTreeDefaults(root, table, "")
	return table
}

// stampNameRefs records, on every node whose Type is a name reference
// that resolves in table, the referenced definition (SchemaNode.refTarget).
// Resolution is lookupNameRef — the same scopedRefKeys precedence every
// other resolver derives from — at the reference's enclosing namespace
// scope, so the stamp cannot bind differently than the wire or the
// default coercion did. Descent is kind-bound like collectNamedTypes:
// a stray-surfaced body (an "items" on an "int") neither defines nor
// references, so nothing inside one is stamped. Root trees are
// JSON-derived and acyclic, and their depth is bounded by the parse's
// own nesting cap, so the plain recursion terminates.
func stampNameRefs(n *SchemaNode, table map[string]*SchemaNode, ns string) {
	if n == nil || len(table) == 0 {
		return
	}
	if t := lookupNameRef(n, table, ns); t != nil {
		n.refTarget = t
		n.refNS = ns
	}
	child := nsForChildren(n, ns)
	if n.Type == "array" {
		stampNameRefs(n.Items, table, child)
	}
	if n.Type == "map" {
		stampNameRefs(n.Values, table, child)
	}
	if isRecordKind(n.Type) {
		for i := range n.Fields {
			stampNameRefs(&n.Fields[i].Type, table, child)
		}
	}
	for i := range n.Branches {
		stampNameRefs(&n.Branches[i], table, child)
	}
}

// collectLocalNames gathers the fullnames of every named type defined in
// n's tree, descending the same kind-bound structure the emission walk
// treats as non-stray. Unlike collectNamedTypes it must survive arbitrary
// hand-built input, because it runs at the START of [SchemaNode.Schema],
// before the emission walk's own cycle and depth guards: visited
// terminates Items/Values pointer cycles, and depth stops chains past the
// emission walk's own ceiling (names below it sit in a region the walk
// rejects before any splice could consult them).
func collectLocalNames(n *SchemaNode, names map[string]bool, visited map[*SchemaNode]struct{}, depth int) {
	if n == nil || depth > maxSchemaJSONDepth {
		return
	}
	if _, ok := visited[n]; ok {
		return
	}
	visited[n] = struct{}{}
	if isNamedKind(n.Type) {
		if fn := nodeFullname(n); fn != "" {
			names[fn] = true
		}
	}
	if n.Type == "array" {
		collectLocalNames(n.Items, names, visited, depth+1)
	}
	if n.Type == "map" {
		collectLocalNames(n.Values, names, visited, depth+1)
	}
	if isRecordKind(n.Type) {
		for i := range n.Fields {
			collectLocalNames(&n.Fields[i].Type, names, visited, depth+1)
		}
	}
	for i := range n.Branches {
		collectLocalNames(&n.Branches[i], names, visited, depth+1)
	}
}

// nodeCarriesNothingBut walks n's exported fields and reports whether every
// one of them other than Type is zero, skipping the fields exempt reports as
// carrying no loss.
//
// The walk is DERIVED from the struct's field set rather than written as a
// remembered list, which is the whole point for both callers: a hand-written
// subset can always be missing a member, and a value in a forgotten field then
// vanishes. Asking the field set cannot be. One walk rather than two, so a
// later field cannot be seen by one question and overlooked by the other.
//
// Unexported state is skipped deliberately: it is derived bookkeeping — the
// name-reference stamp and its scope — not as-written content, so it must not
// force a different emission.
func nodeCarriesNothingBut(n *SchemaNode, exempt func(*SchemaNode, string) bool) bool {
	rv := reflect.ValueOf(n).Elem()
	t := rv.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() || f.Name == "Type" || exempt(n, f.Name) {
			continue
		}
		if !rv.Field(i).IsZero() || nodePresenceSet(n, f.Name) {
			return false
		}
	}
	return true
}

// presenceSet is a bit per attribute that can be written with a body its
// destination stores as that destination's own zero. One field rather than
// one bool each keeps the hidden state on the public struct at a single
// member, and keeps every consumer asking the same question.
type presenceSet uint16

const (
	presDoc presenceSet = 1 << iota
	presLogicalType
	presName
	presNamespace
	presAliases
	presSymbols
	presSize
	presFields
)

func (p presenceSet) has(b presenceSet) bool { return p&b != 0 }

func (p *presenceSet) setIf(cond bool, b presenceSet) {
	if cond {
		*p |= b
	}
}

// presenceBitFor maps an exported SchemaNode field name to its bit, so the
// emptiness walk, the emitter and the guards all key on one vocabulary.
func presenceBitFor(field string) (presenceSet, bool) {
	switch field {
	case "Doc":
		return presDoc, true
	case "LogicalType":
		return presLogicalType, true
	case "Name":
		return presName, true
	case "Namespace":
		return presNamespace, true
	case "Aliases":
		return presAliases, true
	case "Symbols":
		return presSymbols, true
	case "Size":
		return presSize, true
	case "Fields":
		return presFields, true
	}
	return 0, false
}

// nodePresenceSet reports whether the named exported field carries content
// its VALUE cannot show — an attribute written as the field's own zero.
//
// Without it the emptiness walk above and the emitter disagree by
// construction: the walk asks IsZero, so a doc written as "" reads as an
// empty node and the shortcut collapses it to a bare type name, discarding
// the very attribute the emitter was taught to write. The two questions
// ("does this node carry anything" and "does this node emit anything") must
// read the SAME state or the answer depends on which one runs first.
//
// It is keyed by field NAME so it composes with the exemption sets rather
// than overriding them: at a name-reference splice Doc and LogicalType are
// exempt usage-site attributes (NOT_BUGS #25), so their presence is ignored
// there exactly as their value is.
func nodePresenceSet(n *SchemaNode, field string) bool {
	b, ok := presenceBitFor(field)
	return ok && n.present.has(b)
}

// bareEmissionExempt classifies the fields whose value cannot be lost by
// collapsing a node to its bare type name, because they have no emitted form
// there at all. An exemption is a CLAIM, and
// TestInvariant_BareEmissionCoversEverySchemaNodeField checks both halves of
// it: an exempt field must not block, and a non-exempt field must both block
// and survive the emit → re-parse round trip.
func bareEmissionExempt(n *SchemaNode, field string) bool {
	switch field {
	case "Branches":
		// No JSON key routes to Branches outside a union — the union arm
		// returns before reaching here — so a hand-built value on another
		// kind is inert and collapsing cannot lose it.
		return n.Type != "union"
	case "EnumDefault":
		// HasEnumDefault is the carrier: the "default" key is emitted from
		// it, and a node with EnumDefault set but HasEnumDefault false
		// declares no default at all, so there is nothing to carry. The
		// carrier itself is NOT exempt.
		return !n.HasEnumDefault
	}
	return false
}

// nameRefUsageSiteExempt classifies the fields a node may carry and still be
// emitted as a pure NAME REFERENCE — that is, the fields whose loss at a
// reference's usage site is already adjudicated, so blocking on them would
// convert a documented silent drop into a hard parse error.
//
//   - Doc, Aliases, Namespace and LogicalType are reserved USAGE-SITE
//     attributes on a wrapped reference ({"type":"Inner","doc":"x"}). A
//     definition cannot carry a second name, namespace or doc per usage site,
//     so the splice drops them by design.
//   - Props is the wrapper's custom properties, which the splice MERGES onto
//     the definition (definition-wins, reserved keys dropped).
//
// Every other field blocks, so the node renders as-written instead of splicing
// and the re-parse judges the hybrid loudly. Precision and Scale are NOT
// exempt despite also being usage-site attributes: the parse routes an
// unconsumed precision/scale to Props, so a non-zero value here can only come
// from a caller writing the field directly, and that write must not vanish.
func nameRefUsageSiteExempt(_ *SchemaNode, field string) bool {
	switch field {
	case "Doc", "Aliases", "Namespace", "LogicalType", "Props":
		return true
	}
	return false
}

// nodeCarriesOnlyType reports whether n holds no information beyond its
// Type, so collapsing it to the bare type name (`"int"` rather than
// `{"type":"int"}`) loses nothing.
func nodeCarriesOnlyType(n *SchemaNode) bool {
	return nodeCarriesNothingBut(n, bareEmissionExempt)
}

// nodeIsNameRefShape reports whether n can be emitted as a pure name
// reference (bare, or wrapped with custom properties): no structural,
// naming, or kind-specific keys of its own beyond the usage-site attributes
// a splice is already documented to drop or merge. A stamped node that fails
// this (a caller grafted content onto an extracted reference) renders
// as-written instead of splicing, so nothing it carries is silently
// discarded — the re-parse then judges the hybrid loudly.
func nodeIsNameRefShape(n *SchemaNode) bool {
	return nodeCarriesNothingBut(n, nameRefUsageSiteExempt)
}

// nodeRefTargetAgrees reports whether n's stamped refTarget is still the type
// n's exported Type NAMES. The stamp is hidden state that survives a struct
// copy — which is exactly how a caller extracts a sub-node — so a caller who
// then edits Type would otherwise get the ORIGINAL spelling's definition
// spliced in, hidden state silently beating the exported field they just set.
//
// Agreement is decided by ASKING THE RESOLVER — lookupNameRef against a
// one-entry table holding the stamped target — never by restating which
// spellings it binds. That inherits every form scopedRefKeys admits and any
// later change to it; a hand-written list under-accepts the day the resolver
// grows a form.
//
// The scope asked at is the stamp's own (refNS), not the walk's current one.
// An extracted node is re-rooted at the null namespace, so asking at the walk's
// scope would call a short-name reference stale purely for having been lifted
// out of its namespace.
//
// Anything else — a primitive, a different name — means the node was edited
// after Root() stamped it. The stamp is stale and ignored, and the node behaves
// like a hand-built reference: binding to a definition the converted tree
// provides, or dangling loudly.
func nodeRefTargetAgrees(n *SchemaNode) bool {
	t := n.refTarget
	if t == nil {
		return false
	}
	fn := nodeFullname(t)
	if fn == "" {
		return false
	}
	return lookupNameRef(n, map[string]*SchemaNode{fn: t}, n.refNS) == t
}

func collectNamedTypes(n *SchemaNode, table map[string]*SchemaNode) {
	if n == nil {
		return
	}
	// Empty-short-name named kinds register too when their fullname is
	// expressible (fullname "ns."), matching the wire builder's
	// registration; fullname "" has no reference spelling to serve.
	if n.Name != "" || (isNamedKind(n.Type) && nodeFullname(n) != "") { // record / enum / fixed
		// Namespace is resolved at construction (see [SchemaNode]), so
		// the fullname is direct — no inheritance walk needed here.
		// Register exactly what the wire builder registers: every type
		// under its fullname ONLY. A null-namespace type's fullname IS
		// its bare name, so it owns the bare key; also registering
		// namespaced types under their short name would make the bare
		// key last-walked-wins, binding a bare reference at
		// null-namespace scope to a different type than the wire bound
		// whenever short names collide.
		table[nodeFullname(n)] = n
	}
	// Descend only the structural fields the node's KIND binds. The
	// metadata walker surfaces stray container keys as-written (a stray
	// "items" on an "int" populates Items), so an unconditional descent
	// would register a definition-shaped stray body under its fullname —
	// and the map is last-write-wins, so a stray walked after the real
	// definition would silently become the table's answer for that name,
	// coercing name-referenced defaults through a body the wire never
	// bound. Branches stay unconditional: only genuine union parsing
	// populates them (no JSON key routes a stray there).
	if n.Type == "array" && n.Items != nil {
		collectNamedTypes(n.Items, table)
	}
	if n.Type == "map" && n.Values != nil {
		collectNamedTypes(n.Values, table)
	}
	if isRecordKind(n.Type) {
		for i := range n.Fields {
			collectNamedTypes(&n.Fields[i].Type, table)
		}
	}
	for i := range n.Branches {
		collectNamedTypes(&n.Branches[i], table)
	}
}

func coerceTreeDefaults(n *SchemaNode, table map[string]*SchemaNode, ns string) {
	if n == nil {
		return
	}
	childNS := nsForChildren(n, ns)
	for i := range n.Fields {
		f := &n.Fields[i]
		if f.HasDefault {
			f.Default = coerceMetadataDefault(f.Default, &f.Type, table, childNS)
		}
		coerceTreeDefaults(&f.Type, table, childNS)
	}
	if n.Items != nil {
		coerceTreeDefaults(n.Items, table, childNS)
	}
	if n.Values != nil {
		coerceTreeDefaults(n.Values, table, childNS)
	}
	for i := range n.Branches {
		coerceTreeDefaults(&n.Branches[i], table, childNS)
	}
}

// defaultMatchesBytesOrFixedKind mirrors the wire-encode pipeline's
// validateLeaf for "bytes" / "fixed" branches: codepoint range
// (validateAvroByteString) and, for "fixed", the exact-rune-count
// size match. The metadata-side branch selection must enforce the
// same codepoint range and fixed-size constraint as the wire path
// or [fixed:8,"string"] with a 4-char default would metadata-match
// fixed (string-kind only) while wire matches string (size check
// rejects fixed) — pattern 14a sibling of the convertDefaultBytes/
// validateDefault delegation at
// TestRegression_UnionBytesFixedDefaultMisroutedToWrongBranch.
func defaultMatchesBytesOrFixedKind(t *SchemaNode, val any) bool {
	switch v := val.(type) {
	case string:
		if err := validateAvroByteString(v, t.Type); err != nil {
			return false
		}
		if t.Type == "fixed" && len([]rune(v)) != t.Size {
			return false
		}
		return true
	case []byte:
		if t.Type == "fixed" && len(v) != t.Size {
			return false
		}
		return true
	}
	return false
}

// branchAcceptsDefault reports whether the Avro type t natively accepts val as
// a default, using the same Go→Avro compatibility validateDefault enforces on
// the wire side. coerceMetadataDefault's union selection walks branches in
// order and takes the first that accepts, as Java's parseField does.
//
// The numeric arms delegate to defaultAsInt32 / defaultAsInt64 /
// defaultAsFloat, so both selectors apply identical predicates. float/double
// accept any numeric input per the lossy-destination policy, so ["float","int"]
// default 42 picks float on both surfaces, and REJECT strings, since Java's
// text→Double coercion fires only for an outer field type. bytes/fixed accept a
// codepoint-mapped string or []byte.
//
// The structural arms recurse so per-element validity mirrors walkDefault —
// record enforces required-field presence, array/map require every element to
// accept. Without that, [{record needing X}, {record needing nothing}] with
// default {} metadata-matches the FIRST branch on type alone while the wire
// picks the second, and a user type-switching on Default sees a Go type
// contradicting the decoded auto-fill.
func branchAcceptsDefault(t *SchemaNode, val any, table map[string]*SchemaNode, ns string) bool {
	// Resolve a bare name-reference if the caller supplied a name-table.
	if resolved := lookupNameRef(t, table, ns); resolved != nil {
		return branchAcceptsDefault(resolved, val, table, nodeEffNS(resolved))
	}
	switch t.Type {
	case "null":
		return val == nil
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "int":
		_, err := defaultAsInt32(val)
		return err == nil
	case "long":
		_, err := defaultAsInt64(val)
		return err == nil
	case "float", "double":
		_, err := defaultAsFloat(val)
		return err == nil
	case "string":
		_, ok := val.(string)
		return ok
	case "enum":
		sym, ok := val.(string)
		if !ok {
			return false
		}
		// The wire's validateLeaf rejects non-member symbols, so enum needs
		// its own arm: the string arm would accept any string, and a union
		// [enum:{A,B}, bytes] with default "Z" would metadata-match enum
		// while the wire rejects it and picks bytes. Membership is
		// unconditional, so an empty enum accepts nothing and the walk falls
		// through exactly as the wire does. A name-referenced enum arrives
		// here only after lookupNameRef, with its symbols final.
		return slices.Contains(t.Symbols, sym)
	case "bytes", "fixed":
		return defaultMatchesBytesOrFixedKind(t, val)
	case "record", "error":
		m, ok := val.(map[string]any)
		if !ok {
			return false
		}
		childNS := nsForChildren(t, ns)
		for i := range t.Fields {
			f := &t.Fields[i]
			fv, present := m[f.Name]
			if !present {
				if !f.HasDefault {
					return false
				}
				continue
			}
			// Coerce the child before the accept-check, as the wire selector
			// does: validateLeaf's container arms rewrite each child via
			// coerceDefault, so a string in a nested float/double field becomes
			// a float and the wire selects the container branch. Without the
			// twin coercion here, this selector would reject that field and
			// pick a later branch. coerceMetadataDefault returns fresh
			// containers, so a rejected sibling's value is never mutated, and
			// the coerced value is what Default surfaces — selection and
			// surfacing agree. Deliberately NOT applied to the scalar
			// float/double arm, so a DIRECT scalar branch (["double","string"]
			// default "5") still rejects the numeric one (NOT_BUGS #10).
			fv = coerceMetadataDefault(fv, &f.Type, table, childNS)
			if !branchAcceptsDefault(&f.Type, fv, table, childNS) {
				return false
			}
		}
		return true
	case "array":
		arr, ok := val.([]any)
		if !ok {
			return false
		}
		if t.Items == nil {
			return true
		}
		for _, item := range arr {
			// Coerce each element to mirror the wire selector — see the
			// record arm above.
			item = coerceMetadataDefault(item, t.Items, table, ns)
			if !branchAcceptsDefault(t.Items, item, table, ns) {
				return false
			}
		}
		return true
	case "map":
		m, ok := val.(map[string]any)
		if !ok {
			return false
		}
		if t.Values == nil {
			return true
		}
		for _, v := range m {
			// Coerce each value to mirror the wire selector — see the
			// record arm above.
			v = coerceMetadataDefault(v, t.Values, table, ns)
			if !branchAcceptsDefault(t.Values, v, table, ns) {
				return false
			}
		}
		return true
	case "union":
		return firstMetadataBranchAcceptingDefault(t, val, table, ns) != nil
	}
	return false
}

// firstMetadataBranchAcceptingDefault returns the first branch of union t whose
// [branchAcceptsDefault] accepts val, name-refs resolved, or nil. The
// *SchemaNode twin of [firstUnionBranchAcceptingDefault]: both implement Avro's
// "first matching branch wins" rule (1.12 relaxed "first branch" to "any
// branch", with a deterministic first-match tie-break) on their own side of the
// dual-namespace boundary. [coerceMetadataDefault] uses the returned branch to
// recurse; [branchAcceptsDefault]'s union arm only needs nil vs non-nil.
func firstMetadataBranchAcceptingDefault(t *SchemaNode, val any, table map[string]*SchemaNode, ns string) *SchemaNode {
	for i := range t.Branches {
		branch := &t.Branches[i]
		if resolved := lookupNameRef(branch, table, ns); resolved != nil {
			branch = resolved
		}
		if branchAcceptsDefault(branch, val, table, nsForChildren(branch, ns)) {
			return branch
		}
	}
	return nil
}

func nodeFromJSONObject(m map[string]any, parentNS string, memo strayShapeMemo) SchemaNode {
	n := SchemaNode{}

	getString(m, "type", &n.Type)
	n.present.setIf(getString(m, "name", &n.Name), presName)
	// Namespace resolves at build: an explicit attribute wins (including
	// the explicit-empty "namespace":"" null-namespace form — a DIFFERENT
	// type than one inheriting the enclosing namespace, per the spec's
	// fullname rules); an undotted named type without the attribute
	// inherits the enclosing scope. A dotted name carries its own
	// namespace; any attribute alongside it is preserved as-written for
	// fidelity but ignored, exactly as the parser ignores it.
	explicitNS, hasExplicitNS := "", false
	if s, ok := m["namespace"].(string); ok {
		explicitNS, hasExplicitNS = s, true
	}
	n.present.setIf(hasExplicitNS, presNamespace)
	switch {
	// Named kinds with an empty short name inherit/take-explicit exactly
	// like any undotted name (the parser resolves "name":"" under an
	// enclosing namespace to fullname "ns.").
	case (n.Name != "" || isNamedKind(n.Type)) && !strings.Contains(n.Name, "."):
		if hasExplicitNS {
			n.Namespace = explicitNS
		} else {
			n.Namespace = parentNS
		}
	case hasExplicitNS:
		n.Namespace = explicitNS
	}
	childNS := nsForChildren(&n, parentNS)
	// getString consumes only a STRING body, so recording presence off the same
	// read keeps the two in step: a non-string doc/logicalType routes elsewhere
	// (dropped, and Props respectively) and must not count as written.
	//
	// doc is recorded on EVERY kind, not only the ones Apache Avro reads it on.
	// The authority for a placement is whichever reference HAS that placement,
	// and it governs empty and non-empty bodies alike. Apache Avro has no doc
	// slot on a primitive or container so it cannot rule there, and this
	// package already sides with fastavro by preserving
	// {"type":"int","doc":"d"}. Taking the empty twin from Apache Avro's
	// ABSENCE while the non-empty twin follows fastavro's PRESENCE would split
	// one placement between two authorities.
	n.present.setIf(getString(m, "doc", &n.Doc), presDoc)
	n.present.setIf(getString(m, "logicalType", &n.LogicalType), presLogicalType)
	// Precision/Scale hold validated decimal parameters only: consumption
	// happens exactly on a recognized decimal carrier, and every other
	// placement leaves the keys to the Props loop below, mirroring the wire
	// parser's extra routing.
	if decimalConsumesPrecisionScale(n.Type, n.LogicalType) {
		getInt(m, "precision", &n.Precision)
		getInt(m, "scale", &n.Scale)
	}
	// Size/aliases/symbols capture through the SAME decodes the parser's arms
	// run, so a structural field is set exactly when the parse consumes the key
	// out of props. A malformed stray body — mixed-type array, non-integral
	// number — rides to Props verbatim as its ONLY surface; capturing a coerced
	// image here would fabricate metadata appearing nowhere in the input. At
	// BOUND positions the parse already validated the value, so these gates
	// never decline there.
	sizeOK := false
	if v, ok := m["size"]; ok {
		if l, err := decodeLaxInt("size", v); err == nil {
			n.Size = int(l)
			sizeOK = true
			n.present |= presSize
		}
	}
	if ss, ok, err := stringSliceFrom(m, "aliases"); err == nil && ok {
		n.Aliases = ss
		n.present |= presAliases
	}
	if ss, ok, err := stringSliceFrom(m, "symbols"); err == nil && ok {
		n.Symbols = ss
		n.present |= presSymbols
	}

	if n.Type == "enum" {
		if d, ok := m["default"].(string); ok {
			n.EnumDefault = d
			n.HasEnumDefault = true
		}
	}

	// Child schemas come from walkNodeChildren, so the child set, the lift
	// decision and key routing, and each child's namespace scope cannot drift
	// from the wire parser or the SchemaCache walkers. At a BOUND position a
	// field with no type key never parses, but inside a STRAY "fields" the
	// record build never runs, so a typeless element is parseable and fires
	// fieldNoType. Every element fires exactly one callback, leaving no
	// pre-sized zero SchemaField behind.
	//
	// strayKeys: this walker alone also enumerates container keys the node's
	// kind does not bind — a stray "items" on an "int" — because SchemaNode's
	// contract is to SURFACE them as-written on the matching structural field.
	// Surfacing is read-only: nothing here registers a name or mutates the
	// tree, which is why the stray positions are safe here and gated off for
	// every other walker (see nodeChildVisitor.strayKeys).
	walkNodeChildren(m, parentNS, childNS, nodeChildVisitor{
		strayKeys:      true,
		strayShapeMemo: memo,
		fields:         func(arr []any) { n.Fields = make([]SchemaField, len(arr)); n.present |= presFields },
		field: func(i int, fm map[string]any, typeKey, scope string) {
			n.Fields[i] = metadataField(fm, nodeFromJSON(fm[typeKey], scope, memo), nil)
		},
		fieldNoType: func(i int, fm map[string]any) {
			// Typeless element inside a stray "fields": surface the
			// written attributes (name / doc / aliases / order / default
			// / props) on the field with a zero Type — as-written, never
			// a fabricated zero element.
			n.Fields[i] = metadataField(fm, SchemaNode{}, nil)
		},
		flatField: func(i int, fm map[string]any, kind, scope string) {
			// Flat (goavro-style) field format: the wire parser lifts the
			// field's defining keys into a nested type definition, named
			// after the field. The metadata tree must describe that same
			// post-lift schema. Otherwise the type node is an empty shell,
			// Root().Schema() cannot rebuild it — the rebuild emits a
			// nested type OBJECT, which the lift's bare-string gate
			// ignores — and the lifted named type stays invisible to
			// name-reference default coercion, which keys on Name.
			flatType := flatLiftTypeMap(fm, kind)
			n.Fields[i] = metadataField(fm, nodeFromJSONObject(flatType, scope, memo), flatType)
		},
		items: func(key, scope string) {
			node := nodeFromJSON(m[key], scope, memo)
			n.Items = &node
		},
		values: func(key, scope string) {
			node := nodeFromJSON(m[key], scope, memo)
			n.Values = &node
		},
	})

	// Collect custom properties — anything not reserved, with precision/scale
	// reserved only on a recognized decimal carrier. The structural keys
	// already recorded their verdicts above (the gate that fired each
	// walkNodeChildren callback IS the shape check, and the size/aliases/
	// symbols captures ran the parser's own decodes), so route on those rather
	// than decoding a second time. name/namespace are single string asserts
	// with no compounding cost, so a fresh check is fine.
	shapeOK := func(key string, v any) bool {
		switch key {
		case "items":
			return n.Items != nil
		case "values":
			return n.Values != nil
		case "fields":
			return n.Fields != nil
		case "symbols":
			return n.Symbols != nil
		case "aliases":
			return n.Aliases != nil
		case "size":
			return sizeOK
		}
		return strayBodyShapeOK(key, v)
	}
	for k, v := range m {
		if schemaReservedKeyForObject(k, v, n.Type, n.LogicalType, shapeOK) {
			continue
		}
		if n.Props == nil {
			n.Props = make(map[string]any)
		}
		n.Props[k] = v
	}

	return n
}

// metadataField builds one SchemaField from its raw field object and
// already-built type node. flatType, non-nil for a flat-form field, is the
// lifted key set: keys the lift routed into the type are excluded from Props,
// and the routed doc belongs to the lifted type rather than the field, exactly
// as the wire lift routes it.
func metadataField(fm map[string]any, typ SchemaNode, flatType map[string]any) SchemaField {
	sf := SchemaField{Type: typ}
	getString(fm, "name", &sf.Name)
	if d, ok := fm["default"]; ok {
		// Coerce string defaults to typed float64 for float/double
		// fields (and recurse through nested record/array/map/union
		// types), matching Java's Schema.parseField text→DoubleNode
		// coercion and the wire-encode pipeline's coerceDefault — so
		// SchemaField.Default reflects the materialized wire form
		// instead of the raw JSON string. nil name-table: best-effort
		// inline coercion only; fixupNameRefDefaults (called at the end
		// of Root) re-coerces with a populated table to resolve
		// name-references that aren't visible during this per-field
		// construction.
		sf.Default = coerceMetadataDefault(d, &sf.Type, nil, "")
		sf.HasDefault = true
	}
	if flatType == nil {
		sf.docSet = getString(fm, "doc", &sf.Doc)
	}
	// Field aliases read through the parser's own decode (stringSliceFrom):
	// bound fields are parse-validated and stray-fields elements are
	// shape-checked before this runs, so the gate never declines here — it
	// exists so this surface structurally cannot coerce a malformed body.
	if ss, ok, err := stringSliceFrom(fm, "aliases"); err == nil && ok {
		sf.Aliases = ss
	}
	getString(fm, "order", &sf.Order)
	// The exact-lowercase field reserved keys are consumed into the
	// SchemaField attributes above; every other key — including a
	// case-variant spelling of a reserved key, which is an ordinary
	// custom property — is preserved verbatim. Same rule as the
	// type-object Props routing (schemaReservedKeyForObject).
	for k, v := range fm {
		if fieldReservedKeys[k] {
			continue
		}
		if _, routed := flatType[k]; routed {
			continue
		}
		if sf.Props == nil {
			sf.Props = make(map[string]any)
		}
		sf.Props[k] = v
	}
	return sf
}

// decimalConsumesPrecisionScale reports whether a type object with the
// given type and logicalType (values as-written; matched exactly, like the
// parser's own logical dispatch) is a recognized decimal carrier — the one
// placement where the parser consumes "precision"/"scale" as decimal
// parameters and validates them (validateLogical's decimal arm). On every
// other placement the two keys are inert metadata surfaced as custom
// properties, matching the field level: the spec permits attributes it
// does not define as metadata, and no wire codec reads an unconsumed
// precision/scale.
func decimalConsumesPrecisionScale(typ, logical string) bool {
	return logical == "decimal" && (typ == "bytes" || typ == "fixed")
}

// schemaKeyBinds reports whether a type object of the given kind/logical BINDS
// reserved key k (raw value v): the whole grammar in one place — [strayKeyBinds]
// for the keys the kind alone decides, plus the two that also depend on the
// value or the logical type.
//
// It exists so [schemaReservedKeyForObject] asks the binding question once
// rather than enumerating consumed keys. Such an enumeration is a hand-written
// subset, and a subset can always be missing a member: type-level "default"
// and "order" fell through a previous list and were dropped, reaching neither a
// structural field nor Props.
func schemaKeyBinds(k string, v any, typ, logical string) bool {
	switch k {
	case "precision", "scale":
		// Decimal parameters only on a recognized carrier; anywhere else
		// the pair is inert metadata (#71).
		return decimalConsumesPrecisionScale(typ, logical)
	case "logicalType":
		// Consumed only when string-typed: a non-string value can never
		// name a logical, so it is an ordinary custom property (Java reads
		// only textual logicalType props; fastavro and goavro treat any
		// non-matching value as inert). Mirrors the parse arm's
		// string-conditional read.
		_, isString := v.(string)
		return isString
	}
	return strayKeyBinds(typ, k)
}

// schemaReservedKeyForObject reports whether key k (value v) on a type object
// of the given kind/logical is consumed or structurally surfaced, and so kept
// OUT of Props. Props is all raw keys minus these.
//
// Reserved names match ONLY their exact lowercase spelling; every other key,
// case variants of reserved names included, is an ordinary custom property
// that rides to Props verbatim whatever its body.
//
// Exactly two ways a reserved key stays out of Props: the kind BINDS it, or the
// kind does not bind it but SURFACES it as-written on a structural field, which
// needs both a field to land on and a body parsing as that key's shape. A key
// that is neither has Props as its only surface — where type-level "default"
// and "order" land, since only enum binds "default", no kind binds "order", and
// neither has a SchemaNode field on a non-binding kind. Java keeps both as
// schema properties (SCHEMA_RESERVED omits both, Schema.java:175-176;
// ENUM_RESERVED adds "default" alone), as does fastavro 1.12.2 (executed).
//
// shapeOK answers the stray-body shape question from a verdict the caller
// ALREADY computed, so it always describes the queried body. That is what
// keeps this routing from re-decoding a subtree the caller walked: a fresh
// strayBodyShapeOK would re-enter aschemaFromAny, which itself routes stray
// keys, so two decodes per level compound to O(2^depth) on a nested-stray
// schema. A nil shapeOK takes a fresh decode, for the one caller — the cache
// splice merge — with no recorded verdict and no nested strays.
func schemaReservedKeyForObject(k string, v any, typ, logical string, shapeOK strayShapeVerdict) bool {
	if !schemaReservedKeys[k] {
		return false
	}
	if schemaKeyBinds(k, v, typ, logical) {
		return true
	}
	if canonicalStrayKey(k) == "" {
		return false
	}
	if shapeOK != nil {
		return shapeOK(k, v)
	}
	return strayBodyShapeOK(k, v)
}
