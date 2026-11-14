package avro

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Schema is a compiled Avro schema. Create one with [Parse] or [MustParse],
// then use [Schema.Encode] / [Schema.Decode] to convert between Go values and
// Avro binary. A Schema is safe for concurrent use.
type Schema struct {
	ser   serfn
	deser deserfn

	c    aschema     // canonical form, used for fingerprinting and schema comparison
	node *schemaNode // full metadata tree (aliases, defaults, etc.) for schema introspection and evolution
	full string      // original schema JSON, returned by String()

	// soe is the Single Object Encoding header: 2-byte magic (0xC3, 0x01)
	// + 8-byte LE CRC64-Avro fingerprint of the canonical form. Computed on
	// first use by soeHeader (soe.go), never at parse: hashing the canonical
	// form was 11-31% of Parse depending on schema shape, and only the three
	// SOE entry points ever read it. Read it ONLY through soeHeader — a bare
	// field read races with a concurrent first use, and Schema is documented
	// safe for concurrent use.
	//
	// The header is a pure function of c, so the two sites that adopt
	// another schema's canonical form (Resolve, SchemaCache's self-contained
	// splice) adopt its header by assigning c and leaving soe alone.
	soeHashed atomic.Bool
	soeOnce   sync.Once
	soe       [10]byte

	// resolveWriter is the writer schema, populated only by
	// Resolve(writer, reader) when the writer and reader differ (an identity
	// resolution returns the reader schema directly, leaving this nil). It
	// lets DecodeJSON apply writer→reader resolution to WRITER-shaped JSON,
	// matching Java's ResolvingDecoder-over-JsonDecoder (the JsonDecoder is
	// constructed with the writer schema). Binary Decode resolves via s.deser;
	// JSON resolution composes the writer's JSON decode + binary re-encode with
	// that same resolving s.deser. nil ⇒ not resolved; DecodeJSON decodes
	// directly against s.node.
	//
	// DecodeSingleObject reads it too, to accept wire bearing the WRITER's
	// fingerprint (see acceptsWriterSOE). That is a second question asked of
	// one field rather than a second copy of it: two fields that must always
	// hold the same schema can drift with nothing to notice, while one field
	// two callers ask cannot. The hazard the duplicate was guarding — a change
	// that repoints this at the raw view, or drops it, silently taking
	// single-object writer acceptance along — is caught instead by the net
	// asserting that acceptance follows the writer schema.
	resolveWriter *Schema

	// resolveWriterRaw is a custom-free view of the writer, used solely by
	// decodeJSONResolved for the JSON wire-shape round-trip (writer-JSON ->
	// writer-binary). That intermediate must hold RAW Avro-native values: if the
	// writer carries its own CustomType decoders, decoding writer-JSON through
	// resolveWriter would run them (producing Go-domain values) and the re-encode
	// would then need the writer's custom ENCODER to invert them -- which fails
	// for a Decode-only custom, or any custom whose Encode cannot reproduce the
	// decoded type. Binary Decode never re-encodes through the writer (it reads
	// raw writer bytes and applies the READER's custom), so it is unaffected;
	// this custom-free view makes the JSON path match it. Equal to resolveWriter
	// when the writer has no custom types (nothing to suppress); set alongside
	// resolveWriter by Resolve.
	resolveWriterRaw *Schema

	// Per-schema custom type overlay. Keyed by *schemaNode so the
	// shared node is not mutated — different schemas parsed with
	// different custom types get different overlays.
	custom map[*schemaNode]*customWiring

	// customBaked reports custom-conversion effects reachable through this
	// schema's node tree even when the custom overlay above is empty: a
	// reference to a SchemaCache-inherited named type whose DEFINING parse
	// wired custom types carries the binary callback wraps composed inside
	// the inherited ser/deser and the JSON wrap baked on the shared
	// node.decodeJSON, but no entry in THIS parse's overlay
	// (applyCustomTypes visits only newly built nodes). Resolve's
	// custom-free writer view must be built whenever either signal is set;
	// keying on len(custom) alone resurrected the decode-only re-encode
	// failure (3333e9b) for cache-parsed custom-typed writers.
	customBaked bool

	// slabFree marks a schema whose compiled deser provably never touches
	// the per-call *slab: a scalar leaf kind (see slabFreeKinds) with no
	// custom-decoder wiring anywhere (customBaked covers both this parse's
	// overlay and cache-inherited wraps). Decode skips the slab pool for
	// such schemas and passes a nil slab, so scalar decodes stay
	// allocation-free even when GC has drained the pool (issue #41). The
	// zero value false is the safe default: a construction path that does
	// not classify (Resolve's non-identity path, whose promote/skip
	// routines do use the slab) keeps the pool.
	slabFree bool
}

// customWiring bundles the per-node custom-type artifacts. Allocated
// once per node that matches at least one registered CustomType; the
// three slots are independently populated based on which callbacks
// the user provided.
type customWiring struct {
	// encode wraps the user's CustomType.Encode chain. Runs before
	// the built-in serializer. nil if no encoders matched, or if
	// every matching CustomType had Encode == nil.
	encode func(reflect.Value) (reflect.Value, error)
	// decoders is the CustomType.Decode callback chain. Run after the
	// built-in deserializer produces the raw Avro-native value. nil
	// if no decoders matched.
	decoders []func(any, *SchemaNode) (any, error)
	// sn is the public *SchemaNode passed to the encode and decoder
	// callbacks. Built once at parse time and reused across calls.
	// Always populated when the wiring is non-nil.
	sn *SchemaNode
	// suppressLogical mirrors the binary decoder-suppression decision
	// (hasMatchingCustomType) so the JSON decode wrapper feeds the custom
	// decoder the same raw-vs-enriched value the binary path does. False
	// for wildcard CustomTypes (empty LogicalType AND AvroType), which the
	// binary gate excludes — they must receive the enriched logical value.
	// Carried here so resolved nodes (resolve.go) reuse the parse-time
	// decision without re-running the gate.
	suppressLogical bool
	// encodeSuppresses mirrors the binary ENCODER-suppression decision
	// (hasMatchingCustomTypeWithEncode). The JSON encode arms (decimal /
	// big-decimal on bytes; all logicals on fixed) must skip the built-in
	// logical coercion arm iff the binary build replaced the logical
	// serializer with the base (raw-bytes) serializer — which is iff a
	// non-wildcard matching CustomType has an Encode. Gating the JSON arms
	// on the runtime proxy `custom[node].encode != nil` instead would
	// wrongly skip the arm for a WILDCARD-with-Encode (the binary keeps the
	// logical ser for wildcards), rejecting *big.Rat on JSON while binary
	// accepts it. Use this exact predicate, not the proxy.
	encodeSuppresses bool
}

// schemaNode preserves full schema metadata that canonical form strips:
// aliases, defaults, enum defaults, and links to compiled ser/deser.
type schemaNode struct {
	kind        string        // "null","boolean","int","long","float","double","bytes","string","record","enum","array","map","fixed","union"
	name        string        // fully-qualified name (named types only)
	aliases     []string      // named type aliases (fully qualified)
	bareAliases []string      // aliases declared without a dot, as written (short-name match tier; see bareAliasShorts)
	logical     string        // logical type
	fields      []fieldNode   // record fields
	symbols     []string      // enum symbols
	enumDef     string        // enum default symbol
	hasEnumDef  bool          // whether enum default is specified
	items       *schemaNode   // array item type
	values      *schemaNode   // map value type
	size        int           // fixed size
	precision   int           // decimal precision
	scale       int           // decimal scale
	branches    []*schemaNode // union branches
	ser         serfn
	deser       deserfn
	decodeJSON  jsonDecodeFn // non-nil only when custom decoders are wired
	serRecord   *serRecord
	deserRecord *deserRecord

	props    map[string]any // extra schema properties (for CustomType callbacks)
	fieldIdx map[string]int // record field name → index; built at parse time

	// tags is a union's name→branch lookup and symbolIdx is an enum's
	// symbol→ordinal lookup. Both live HERE, beside fieldIdx, rather than
	// on the serializer, because a per-value question has to be answerable
	// from whatever the asker holds — and the JSON codec holds a
	// *schemaNode, never a *serUnion. A table only one wire can reach is a
	// table the other wire re-derives per value by scanning the siblings,
	// which is linear in a count the schema's author chooses.
	tags      *unionTags     // union: see unionTags
	symbolIdx map[string]int // enum: symbol → ordinal; nil below enumIndexMin

	// unknownLogical preserves the schema's original logicalType value
	// when it failed validateLogical (no built-in handler matched AND
	// no registered CustomType matched at this Parse). The runtime
	// ignores it — built-in handlers were already chosen based on the
	// validated (possibly cleared) logical, and runtime dispatches use
	// `logical`. unknownLogical is consulted ONLY by the cache-
	// reference rejection check (rejectCachedRefIfCustomTypeWouldMatch)
	// so a later Parse that registers a CT for this logical can
	// detect the silent-drop scenario and error loudly.
	unknownLogical string
}

// jsonDecodeFn is the per-node JSON dispatch shape used when custom
// decoders are wired (mirrors deserfn for the binary path). nil node
// means the runtime falls back to kind dispatch.
type jsonDecodeFn func(*jsonDecoder, reflect.Value, *schemaNode) error

// fieldNode represents a record field with full metadata.
type fieldNode struct {
	name    string
	nameVal reflect.Value // pre-computed for map lookups without allocation
	// aliases: schema-evolution alternate field names. Consumers are
	// all decode/resolve side — JSON decode (via node.fieldIdx, which
	// is built from this slice at parse time), CheckCompatibility's
	// findWriterField (compat.go), and readerFieldLookup, which both
	// Resolve and CheckCompatibility match writer fields through
	// (resolve.go). Not consulted on encode — aliases are a reader-
	// side concept per the Avro 1.12 spec.
	aliases    []string
	node       *schemaNode
	defaultVal any
	hasDefault bool
}

type parseOptLax struct{ fn func(string) error }

func (parseOptLax) schemaOpt() {}

// WithLaxNames relaxes name validation in [Parse] and [SchemaCache.Parse],
// overriding the default requirement that names match the Avro strict name
// regex [A-Za-z_][A-Za-z0-9_]*. If fn is nil, only non-empty names are
// required. If fn is non-nil, it is called for each name component and
// should return an error for invalid names. Dot-separated fullnames are
// split before calling fn. Ignored by [SchemaFor].
func WithLaxNames(fn func(string) error) SchemaOpt { return parseOptLax{fn} }

// internalReparseNames is the name validator for the library's internal
// re-parses of schema text it produced itself: Resolve's custom-free writer
// view (resolve.go) and SchemaCache's self-contained splice rebuild
// (cache.go). Both sites re-parse text whose names the ORIGINAL parse
// already validated under the user's chosen validator (strict, or any
// WithLaxNames fn), so validation here has no safety role — it can only
// wrongly reject names the user accepted. It therefore accepts everything:
// WithLaxNames(nil), the previous validator at both sites, rejected empty
// name components (namespace "a..b") — the one class a user fn can accept
// that lax(nil) does not — hard-failing Resolve on an already-parsed,
// wire-valid writer and silently degrading the cache's metadata forms to a
// dangling reference. Validation never transforms names — they pass
// through verbatim — so the canonical/fingerprint/wire bytes match a
// standalone parse of the same schema under the user's validator.
var internalReparseNames = WithLaxNames(func(string) error { return nil })

// MustParse is like [Parse] but panics on error.
func MustParse(schema string, opts ...SchemaOpt) *Schema {
	s, err := Parse(schema, opts...)
	if err != nil {
		panic("avro: " + err.Error())
	}
	return s
}

// Parse parses an Avro JSON schema string and returns a compiled [*Schema].
// The input can be a primitive name (e.g. `"string"`), a JSON object
// (record, enum, array, map, fixed), or a JSON array (union). Named types
// may self-reference. The schema is fully validated: unknown types, duplicate
// names, invalid defaults, etc. all return errors.
//
// To parse schemas that reference named types from other schemas, use
// [SchemaCache].
func Parse(schema string, opts ...SchemaOpt) (*Schema, error) {
	b := &builder{
		named:      make(map[string]*namedType),
		building:   make(map[*schemaNode]struct{}),
		definedSet: make(map[*namedType]bool),
		minBytes:   newMinBytesWalk(),
	}
	applySchemaOpts(b, opts)
	return parse(schema, b)
}

func applySchemaOpts(b *builder, opts []SchemaOpt) {
	for _, o := range opts {
		switch o := o.(type) {
		case parseOptLax:
			if o.fn != nil {
				b.checkName = o.fn
			} else {
				b.checkName = func(s string) error {
					if s == "" {
						return errors.New("name must be non-empty")
					}
					return nil
				}
			}
		case CustomType:
			if o.needsAvroType && o.AvroType == "" {
				// Validated lazily: store now, error in parse.
				// We still append so the error is reported.
			}
			b.customTypes = append(b.customTypes, o)
		}
	}
	// The custom-match walk memos exist exactly when CustomTypes are
	// registered (their consumers are all gated on len(customTypes) > 0).
	// Allocated here — before any build work and before any nest(), which
	// shares them by reference — so the whole parse sees one memo.
	// b.customTypes is never appended to after this point; that is the
	// memo's correctness invariant (see customMatchInSubtree).
	if len(b.customTypes) > 0 {
		b.customMatch = make(map[*schemaNode]string)
		b.overlayDone = make(map[*schemaNode]bool)
	}
}

func parse(schema string, b *builder) (*Schema, error) {
	// Bound nesting depth with a single linear scan BEFORE building. The
	// build's maxDepth guard fires per schema node, but the JSON bracket
	// nesting can run deeper than the node depth (and json.Decode has its
	// own ~10000 limit); this O(input) pre-scan rejects pathologically
	// deep input up front. (Parse itself is O(n) via parseSchemaTree — a
	// single generic decode, no per-node subtree re-scan.)
	if err := checkSchemaNestingDepth(schema); err != nil {
		return nil, err
	}
	orig, err := parseSchemaTree(schema)
	if err != nil {
		return nil, err
	}
	if err := b.build("", orig); err != nil {
		return nil, boundErrorLen(err)
	}
	if err := b.finalize(); err != nil {
		return nil, boundErrorLen(err)
	}
	s := &Schema{
		ser:         b.ser,
		deser:       b.deser,
		c:           b.canon,
		node:        b.node,
		full:        schema,
		custom:      b.custom,
		customBaked: len(b.custom) > 0 || b.sawInheritedCustom,
	}
	s.slabFree = slabFreeKinds[b.node.kind] && !s.customBaked
	return s, nil
}

// maxSchemaJSONDepth bounds the raw JSON bracket nesting of a schema string.
// It is a coarse DoS backstop, NOT the semantic depth limit: the build's
// maxDepth caps SCHEMA-node nesting, and one schema level can carry up to
// three JSON brackets (a record's object + its "fields" array + a field
// object), so a build-acceptable schema (<= maxDepth nodes) reaches a JSON
// bracket depth of at most 3*maxDepth. The 4*maxDepth limit clears that
// provable ceiling by a full maxDepth, so the pre-scan never rejects a schema
// the builder would accept; it short-circuits input deeper than the builder
// accepts (and deeper than json's own ~10000 nesting cap) in one O(input)
// pass. Parse itself is O(n) — see parseSchemaTree and canonicalBytes — so
// this is a cheap early-reject rather than the defense the DoS bound rests
// on, as it was when the unmarshal and canonical marshal were quadratic.
const maxSchemaJSONDepth = maxDepth * 4

// checkSchemaNestingDepth reports an error if schema's JSON nests deeper than
// maxSchemaJSONDepth. It is a single linear pass that counts '{'/'[' nesting,
// skipping brackets inside JSON strings (honoring backslash escapes), so it
// runs in O(len(schema)) and constant space — cheap enough to gate every parse.
func checkSchemaNestingDepth(schema string) error {
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(schema); i++ {
		c := schema[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{', '[':
			depth++
			if depth > maxSchemaJSONDepth {
				return fmt.Errorf("avro: schema JSON nests deeper than the supported limit (%d brackets)", maxSchemaJSONDepth)
			}
		case '}', ']':
			depth--
		}
	}
	return nil
}

// canonicalFirstOccurrence rewrites the parse-time canon tree so each named
// type's full definition is emitted at its FIRST occurrence in field-walk
// order, bare fullname references afterward — Apache Avro's
// SchemaNormalization rule. The parse-time tree instead puts the body at the
// textual DEFINITION site. The two agree for the common
// define-before-reference shape, where the definition IS the first occurrence,
// and diverge only on a forward reference (which this package accepts): there
// the untransformed tree emits the body at the SECOND occurrence, producing a
// PCF — and so a fingerprint — that differs from Java's, breaking
// single-object and registry interop.
//
// References also normalize to the resolved fullname, since a bare forward
// reference stores the as-written short name. That matches PCF's fullname rule
// and keeps emission consistent across orderings.
func canonicalFirstOccurrence(s aschema) aschema {
	defs := map[string]*aobject{}
	shortCount := map[string]int{}
	shortToFull := map[string]string{}
	collectCanonDefs(s, defs, shortCount, shortToFull)
	if len(defs) == 0 {
		return s // no named types: nothing can be relocated
	}
	return rewriteCanonFirstOcc(s, "", defs, shortCount, shortToFull, map[string]struct{}{})
}

// collectCanonDefs records every named-type definition (the aobject whose
// body is present) in the canon tree, keyed by fullname, plus an
// unqualified-name index for resolving bare forward references into a
// namespaced scope.
func collectCanonDefs(s aschema, defs map[string]*aobject, shortCount map[string]int, shortToFull map[string]string) {
	switch {
	case s.object != nil:
		o := s.object
		if isNamedKind(o.Type) && o.Name != "" {
			if _, dup := defs[o.Name]; !dup {
				defs[o.Name] = o
				short := unqualified(o.Name)
				shortCount[short]++
				shortToFull[short] = o.Name
			}
		}
		for i := range o.Fields {
			if o.Fields[i].Type != nil {
				collectCanonDefs(*o.Fields[i].Type, defs, shortCount, shortToFull)
			}
		}
		if o.Items != nil {
			collectCanonDefs(*o.Items, defs, shortCount, shortToFull)
		}
		if o.Values != nil {
			collectCanonDefs(*o.Values, defs, shortCount, shortToFull)
		}
	case len(s.union) != 0:
		for i := range s.union {
			collectCanonDefs(s.union[i], defs, shortCount, shortToFull)
		}
	}
}

// lookupCanonDef resolves a reference name to its definition aobject, or
// nil when ref is a real Avro primitive (never a named ref) or names a
// type defined outside this schema (a SchemaCache cross-reference — left
// as a bare name, the pre-existing behavior). A bare reference resolves
// lexically in the enclosing namespace ns (mirroring parse-time
// resolveNamedRef), so a short name shared across namespaces still resolves
// to its in-scope fullname rather than being emitted verbatim.
func lookupCanonDef(ref, ns string, defs map[string]*aobject, shortCount map[string]int, shortToFull map[string]string) *aobject {
	if _, isPrim := serPrimitive[ref]; isPrim {
		return nil
	}
	var keys [2]string
	for _, k := range scopedRefKeys(&keys, ref, ns) {
		if o, ok := defs[k]; ok {
			return o
		}
	}
	// Globally-unique short-name fallback for bare references with no
	// other resolution.
	if !strings.Contains(ref, ".") && shortCount[ref] == 1 {
		return defs[shortToFull[ref]]
	}
	return nil
}

func rewriteCanonFirstOcc(s aschema, ns string, defs map[string]*aobject, shortCount map[string]int, shortToFull map[string]string, seen map[string]struct{}) aschema {
	switch {
	case s.primitive != "":
		def := lookupCanonDef(s.primitive, ns, defs, shortCount, shortToFull)
		if def == nil {
			return s // real primitive, or a cross-schema reference: emit as-is
		}
		if _, ok := seen[def.Name]; ok {
			return aschema{primitive: def.Name} // already emitted: bare fullname
		}
		seen[def.Name] = struct{}{}
		return aschema{object: rewriteCanonObj(def, ns, defs, shortCount, shortToFull, seen)}
	case s.object != nil:
		o := s.object
		if isNamedKind(o.Type) && o.Name != "" {
			if _, ok := seen[o.Name]; ok {
				return aschema{primitive: o.Name} // already emitted at an earlier occurrence
			}
			seen[o.Name] = struct{}{}
		}
		return aschema{object: rewriteCanonObj(o, ns, defs, shortCount, shortToFull, seen)}
	case s.union != nil:
		out := make([]aschema, len(s.union))
		for i := range s.union {
			out[i] = rewriteCanonFirstOcc(s.union[i], ns, defs, shortCount, shortToFull, seen)
		}
		return aschema{union: out}
	}
	return s
}

// rewriteCanonObj shallow-copies o (preserving every scalar/canonical
// attribute) and rebuilds only its recursive schema children
// (Fields[].Type / Items / Values) through rewriteCanonFirstOcc, so
// per-object PCF emission is unchanged and only the full-vs-reference
// placement of nested named types moves to first occurrence.
func rewriteCanonObj(o *aobject, ns string, defs map[string]*aobject, shortCount map[string]int, shortToFull map[string]string, seen map[string]struct{}) *aobject {
	// A named type (record/enum/fixed) establishes the namespace its children
	// resolve bare references in; array/map carry no name, so their children
	// inherit the enclosing namespace.
	childNS := ns
	if isNamedKind(o.Type) && o.Name != "" {
		childNS = namespaceOf(o.Name)
	}
	no := *o
	if len(o.Fields) > 0 {
		no.Fields = make([]afield, len(o.Fields))
		for i, f := range o.Fields {
			nf := f
			if f.Type != nil {
				t := rewriteCanonFirstOcc(*f.Type, childNS, defs, shortCount, shortToFull, seen)
				nf.Type = &t
			}
			no.Fields[i] = nf
		}
	}
	if o.Items != nil {
		t := rewriteCanonFirstOcc(*o.Items, childNS, defs, shortCount, shortToFull, seen)
		no.Items = &t
	}
	if o.Values != nil {
		t := rewriteCanonFirstOcc(*o.Values, childNS, defs, shortCount, shortToFull, seen)
		no.Values = &t
	}
	return &no
}

// Canonical returns the Parsing Canonical Form of the schema, stripping
// doc, aliases, defaults, and other non-essential attributes. The result
// is deterministic and matches Java's reference output byte-for-byte,
// so [Schema.Fingerprint] values are interoperable across implementations.
func (s *Schema) Canonical() []byte {
	// Single-pass writer emitting raw UTF-8 strings per the PCF [STRINGS]
	// rule (see canonicalBytes). O(n) over the schema, vs the former
	// nested-MarshalJSON path that re-copied each subtree at every level
	// (O(n^2)); and it never produces the HTML / U+2028 / U+2029 escapes
	// the former path had to un-escape with bytes.ReplaceAll — which
	// corrupted any name containing a literal backslash.
	return canonicalBytes(canonicalFirstOccurrence(s.c))
}

// Fingerprint hashes the schema's canonical form with h and returns h's digest.
// Use [NewRabin] for the spec's CRC-64-AVRO algorithm, or crypto/sha256 for its
// 256-bit recommendation.
//
// Byte order matters for CRC-64-AVRO. Go writes integer hashes high byte first,
// as crc32/crc64/adler32/fnv all do, so [NewRabin] returns the fingerprint
// BIG-ENDIAN; Java, fastavro and the single-object header write that same
// 64-bit value LITTLE-ENDIAN. Only the order differs — compare as a uint64, or
// reverse the bytes.
//
// A crypto/sha256 fingerprint is a byte string with no byte order and already
// matches Java and fastavro byte for byte; reversing it would break that.
//
// No call returns the little-endian CRC-64-AVRO form.
// [Schema.AppendSingleObject] writes it into the message header,
// [SingleObjectFingerprint] reads it back, [Schema.DecodeSingleObject] verifies.
func (s *Schema) Fingerprint(h hash.Hash) []byte {
	// Reset on the way IN, so the digest is a function of the schema and the
	// algorithm alone: neither a hash already used for an earlier fingerprint
	// nor one the caller wrote into can reach the answer. Resetting on the way
	// out would cover only the first of those, and it would clear the state
	// callers read back — taking Sum64 off the hash they passed is a
	// supported way to get the CRC-64-AVRO value as a number.
	h.Reset()
	h.Write(s.Canonical())
	return h.Sum(nil)
}

// String returns the original JSON passed to [Parse], preserving all
// attributes (doc, aliases, defaults, etc.) unlike [Schema.Canonical].
func (s *Schema) String() string {
	return s.full
}

type aschema struct {
	primitive string
	object    *aobject
	union     []aschema
}

// isNullBranch reports whether s is the "null" type in EITHER spelling the
// grammar admits: bare "null" or the wrapped {"type":"null"}. They are the same
// type — same union branch, same bytes — so every decision that matches a
// branch's written spelling routes through here rather than seeing only the
// bare form.
//
// Props and a logicalType on a wrapped null are inert and do NOT make it a
// non-null branch: Avro defines no null logical type, so nothing downstream can
// consume either key. Deciding otherwise would make [{"type":"null","x":1},T] a
// two-non-null-branch union whose first branch encodes zero bytes, a shape no
// other reader agrees with.
//
// The compiled tree answers the same question as schemaNode.kind, which
// normalizes both spellings; this is the as-written (pre-build) view of that
// same fact, so the two cannot disagree.
func (s *aschema) isNullBranch() bool {
	if s.primitive == "null" {
		return true
	}
	return s.object != nil && s.object.Type == "null"
}

// isNullableUnion reports whether s is a union whose first branch is "null".
// Per the Avro spec, such unions implicitly default to null.
func (s *aschema) isNullableUnion() bool {
	return len(s.union) >= 2 && s.union[0].isNullBranch()
}

// aschema, aobject, and afield are populated by parseSchemaTree
// (schema_parse.go) from a single generic decode — they are deliberately
// NOT json.Marshaler / json.Unmarshaler, so the stdlib decoder does not
// re-scan each nested node's subtree (which made Parse O(depth*size)).
// The canonical form is written by canonicalBytes (schema_canonical.go).

type afield struct {
	Name string   `json:"name"`
	Type *aschema `json:"type"`

	// In canonical form, the following are stripped.

	Aliases []string        `json:"aliases,omitempty"`
	Default json.RawMessage `json:"default,omitempty"`
	Order   string          `json:"order,omitempty"`

	// orderSet records that "order" was written, which Order alone cannot:
	// the empty string is its zero, so a validator reading `Order != ""` as
	// "the caller chose an order" skips exactly the one written value that
	// is not a legal order. Apache Avro has no such gap because it decides
	// on the NODE — `if (orderNode != null) Order.valueOf(...)`
	// (Schema.java:1895-1897) — where an empty string reaches valueOf and
	// throws. This restores that: presence and validity are one question.
	orderSet bool

	// Field-level logical type annotations: the Java/JDBC idiom putting
	// logicalType (and decimal's precision/scale) beside `type` on the field
	// object rather than inside the type definition. Confluent's code
	// generator, kafka-connect-avro-converter and most Debezium CDC sources
	// emit this shape. Captured here so the lift can move them into the type,
	// after which the parser sees only the canonical nested form.
	Logical   string `json:"logicalType,omitempty"`
	Scale     *int   `json:"scale,omitempty"`
	Precision *int   `json:"precision,omitempty"`

	// hasDefault is true if the field has a default value. This is set
	// in canonical afields (which strip Default) so that validateDefault
	// can check whether nested record fields have defaults.
	hasDefault bool
}

// afieldComplexKeys are the keys that signal a complex type definition at
// the field level (the "flat" field format accepted by linkedin/goavro).
var afieldComplexKeys = map[string]string{
	"symbols": "enum",
	"items":   "array",
	"values":  "map",
	"fields":  "record",
	"size":    "fixed",
}

// liftTarget returns the aschema a field-level logicalType annotation lands
// on, or nil when the field carries none or nothing can receive it.
//
// This is the ONE navigation for the field-level lift: liftFieldLogicalIntoType
// moves the annotation through it, and fieldDecimalLiftConsumesPrecisionScale
// reads through it. Keeping both on one function is what stops the verdict
// from validating parameters against a type the lift never addressed; the two
// drifted apart once already, on the wrapped-null branch.
func (f *afield) liftTarget() *aschema {
	if f.Logical == "" || f.Type == nil {
		return nil
	}
	switch {
	case f.Type.primitive != "":
		return f.Type
	case len(f.Type.union) > 0:
		for i := range f.Type.union {
			if f.Type.union[i].isNullBranch() {
				continue
			}
			return &f.Type.union[i] // first non-null branch only, like the lift
		}
		return nil
	case f.Type.object != nil:
		return f.Type
	}
	return nil
}

// liftEffectiveLogical reports the lift target's kind and the logical type in
// EFFECT there once the lift has run: the target's OWN annotation when it has
// one — closer to the type wins, so the field's is dropped — and otherwise the
// field's.
//
// Consumption is a question about what LANDS, not about where the lift points.
// A field-level "decimal" that never reaches its target cannot make that
// target read precision/scale, so the pair is inert metadata there and rides
// to Props like any custom property.
func (f *afield) liftEffectiveLogical() (kind, logical string, ok bool) {
	t := f.liftTarget()
	if t == nil {
		return "", "", false
	}
	switch {
	case t.primitive != "":
		// A bare primitive carries no annotation of its own, so the field's
		// always takes effect.
		return t.primitive, f.Logical, true
	case t.object != nil:
		if t.object.Logical != "" {
			return t.object.Type, t.object.Logical, true
		}
		return t.object.Type, f.Logical, true
	}
	return "", "", false
}

// liftFieldLogicalIntoType moves a field-level logicalType annotation (with
// optional precision/scale for the decimal case) into the field's type
// definition, so the rest of the parser sees the canonical nested form.
// The form
//
//	{"name":"ts","type":"long","logicalType":"timestamp-millis"}
//	{"name":"ts","type":["null","long"],"logicalType":"timestamp-millis"}
//
// is a documented common user error (AVRO-2015 / AVRO-3014). Apache Avro warns
// but does not lift; fastavro, hamba and goavro keep it as a field property
// applied to no branch. Hand-written .avsc files, older Java tooling and
// tutorial code emit it widely, so this package lifts it. The wire encoding is
// identical either way; only the JSON layout differs.
//
// An annotation already inside the type definition wins — closer to the type.
// The field-level copies are cleared after lifting so canonical re-emit does
// not duplicate them.
func (f *afield) liftFieldLogicalIntoType() {
	// The target comes from the SHARED navigation, so the lift and the
	// consume verdict can never address different types. It is the FIRST
	// non-null union branch, the type object, or the bare primitive — we do
	// NOT fall through to a later non-null branch (that would silently mutate
	// a different type than the spec-equivalent nested form would have
	// addressed, and on the `[null, T+logical, T]` shape would even
	// synthesize a duplicate union member).
	target := f.liftTarget()
	if target == nil {
		return
	}

	// ANNOTATION and PARAMETERS are separate questions, and conflating them
	// made the two spellings of one schema disagree. Closer-to-the-type wins
	// the ANNOTATION: a target that already declares its own logicalType
	// keeps it and the field's is dropped. The field still completes missing
	// PARAMETERS, but only where they mean something — where the EFFECTIVE
	// logical (the target's own if it has one, else the field's) is
	// "decimal". Anywhere else precision/scale annotate nothing, so copying
	// them in would write inert keys into the type.
	_, effLogical, _ := f.liftEffectiveLogical()
	fillParams := effLogical == "decimal"

	switch {
	case target.primitive != "":
		// A bare primitive, at the field's type position or as a union
		// branch: {"type":["null","long"], "logicalType":"x"} →
		//   {"type":["null",{"type":"long","logicalType":"x"}]}
		obj := &aobject{Type: target.primitive, Logical: f.Logical}
		if fillParams {
			obj.Scale = clonePtrInt(f.Scale)
			obj.Precision = clonePtrInt(f.Precision)
		}
		*target = aschema{object: obj}

	case target.object != nil:
		// {"type":{"type":"long"}, "logicalType":"x"} →
		//   {"type":{"type":"long","logicalType":"x"}}
		if target.object.Logical == "" {
			target.object.Logical = f.Logical
		}
		if fillParams {
			if target.object.Scale == nil {
				target.object.Scale = clonePtrInt(f.Scale)
			}
			if target.object.Precision == nil {
				target.object.Precision = clonePtrInt(f.Precision)
			}
		}
	}
}

// fieldDecimalLiftConsumesPrecisionScale reports whether the field-level
// logical lift consumes "precision"/"scale" as decimal parameters: the
// field declares logicalType "decimal" and the lift's target — the field's
// primitive type name, the first non-null union branch's kind, or the type
// object's kind — is a bytes/fixed carrier, matched as-written like the
// type level's decimalConsumesPrecisionScale (a named reference is not a
// carrier spelling, and the target's own logicalType annotation does not
// change where the lift points). Everywhere else the pair is inert field
// metadata and a malformed body rides to the field's props verbatim.
func (f *afield) fieldDecimalLiftConsumesPrecisionScale() bool {
	kind, logical, ok := f.liftEffectiveLogical()
	return ok && decimalConsumesPrecisionScale(kind, logical)
}

func clonePtrInt(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// maxParseErrorLen bounds the assembled length of a schema-parse error
// message. The build/finalize walkers wrap per nesting level (e.g.
// "invalid array: %v" at each of up to maxDepth levels), so a deeply
// nested schema can otherwise produce a multi-KB message from a small
// input — the same log/RPC/metric amplification the per-value trunc
// helpers prevent, but accumulated over the wrap chain rather than in one
// value. truncForError caps individual interpolated values; this caps the
// whole chain.
const maxParseErrorLen = 1024

// boundErrorLen returns err unchanged if its message fits maxParseErrorLen,
// otherwise a flattened error keeping the head (outer context) and the
// tail (the innermost cause — e.g. "recursion limit exceeded", which the
// wrap chain puts last) with the repeated middle elided.
func boundErrorLen(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if len(msg) <= maxParseErrorLen {
		return err
	}
	half := maxParseErrorLen / 2
	return errors.New(msg[:half] + " …[truncated]… " + msg[len(msg)-half:])
}

// boundJSONErrorEcho truncates user-controllable input echoed verbatim by
// stdlib json / strconv error types, so a hostile MiB-sized literal cannot
// produce a MiB-sized [Parse] error. Reaches *json.UnmarshalTypeError and
// *strconv.NumError.
//
// It walks the chain with [errors.As] and mutates IN PLACE, which must happen
// before the caller wraps with fmt.Errorf("%w"): that call caches its formatted
// message and locks in the pre-truncation content of every descendant.
func boundJSONErrorEcho(err error) error {
	if err == nil {
		return nil
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) && len(ute.Value) > 80 {
		ute.Value = truncForError(ute.Value)
	}
	var ne *strconv.NumError
	if errors.As(err, &ne) && len(ne.Num) > 80 {
		ne.Num = truncForError(ne.Num)
	}
	return err
}

type aobject struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`

	// A complex type can be one of many options. In canonical form, the
	// json fields are ordered "type", "name", and then one of the fields
	// below.

	Fields  []afield `json:"fields,omitempty"`  // record
	Symbols []string `json:"symbols,omitempty"` // enum
	Items   *aschema `json:"items,omitempty"`   // array
	Values  *aschema `json:"values,omitempty"`  // map
	Size    *laxInt  `json:"size,omitempty"`    // fixed

	// In canonical form, the following are stripped.

	Namespace *string         `json:"namespace,omitempty"`
	Aliases   []string        `json:"aliases,omitempty"`
	Default   json.RawMessage `json:"default,omitempty"`

	Logical   string `json:"logicalType,omitempty"`
	Scale     *int   `json:"scale,omitempty"`     // decimal logical type
	Precision *int   `json:"precision,omitempty"` // decimal logical type

	extra map[string]any // non-reserved properties, populated by aschema.UnmarshalJSON
}

// laxInt is an int that also accepts JSON strings containing integers,
// per the Avro spec's [INTEGERS] canonical form rule which acknowledges
// that "size" may appear as a quoted integer.
type laxInt int

// maxLaxIntDataLen caps the raw JSON bytes accepted by [laxInt.UnmarshalJSON].
// Legit int64 representations fit in 20 chars (-9223372036854775808); the
// quoted-string form (per the Avro [INTEGERS] rule) adds 2 chars; +2 chars
// of headroom covers the Go-style +sign that strconv.Atoi accepts. Hostile
// MiB-sized literals are rejected at entry so neither strconv.Atoi nor
// stdlib json.Unmarshal-into-int produces a multi-MB error string (both
// embed the failing input verbatim in *strconv.NumError.Num /
// *json.UnmarshalTypeError.Value). The string-arm's [fmt.Errorf] wrap
// caches the formatted message in *fmt.wrapError, defeating downstream
// truncation of the inner error — the only reliable defense is at entry.
const maxLaxIntDataLen = 24

func (l *laxInt) UnmarshalJSON(data []byte) error {
	if len(data) > maxLaxIntDataLen {
		return fmt.Errorf("integer value exceeds %d byte length cap", maxLaxIntDataLen)
	}
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid integer string: %w", err)
		}
		*l = laxInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*l = laxInt(n)
	return nil
}

// validName reports whether s matches [A-Za-z_][A-Za-z0-9_]*.
func validName(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
			continue
		}
		if i > 0 && c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

// Fixup types for forward references. Avro allows named types to be
// referenced before they are defined (e.g. a union branch or record field
// whose type hasn't been parsed yet). We record what needs patching and
// resolve everything in finalize() once all types are built.

// unionMissing patches a union's ser AND deser branch function tables
// (plus its branch nodes and name tables) when any branch type was a
// forward reference. One fixup record per union keeps the paired
// structures updated together by construction.
type unionMissing struct {
	ser        *serUnion
	deser      *deserUnion
	branches   []*schemaNode  // union node's branch slice; fwd-ref branch nodes are patched in finalize
	missing    map[int]string // branch index → type name
	parentName string         // enclosing scope, for the finalize namespace-qualified retry
}

// fieldMeta carries Avro-level type info for a record field, used by the
// unsafe fast path to select specialized ser/deser routines.
type fieldMeta struct {
	avroType      string
	logical       string // logical type (e.g. "timestamp-millis"), empty if none
	serRecord     *serRecord
	deserRecord   *deserRecord
	inner         *fieldMeta // for nullunion fields: the inner branch's metadata
	nullSecond    bool       // true for ["T","null"] unions (null is index 1)
	hasCustomType bool       // true if a CustomType was applied; disables unsafe fast path
	// minBytes is the minimum wire bytes required to encode one value
	// of this type. Set on the items' fieldMeta when an array is built;
	// used by the unsafe array deser to bound block counts. 0 means
	// items can be zero-byte (array<null>, array<EmptyRecord>).
	minBytes int
}

// metaFixup patches a fieldMeta's serRecord/deserRecord when the inner
// type of a null-union was a forward reference.
type metaFixup struct {
	meta       *fieldMeta
	name       string
	parentName string
}

// recordFieldFixup patches a record field's ser/deser function, avroType,
// meta, and schemaNode when the field's type was a forward reference.
type recordFieldFixup struct {
	// nd is the enclosing record's node, and the encode and decode tables
	// are reached through it: nd.serRecord and nd.deserRecord are the very
	// sr and dr the record case builds, assigned into the node in the same
	// literal. Carrying them again would be three names for one record,
	// which a fixup could then be built holding a node from one record and
	// tables from another.
	nd         *schemaNode
	idx        int
	name       string
	parentName string // enclosing record fullname, for the namespace-qualified retry
	defaultVal any    // parsed JSON default; valid only when hasDefault is true
	hasDefault bool   // whether the field had a "default" in the schema
}

// containerFixup patches an array or map container whose element type
// (items / values) was a forward reference. Used by both case "array"
// and case "map" in buildComplex so the two contexts share one fixup
// path; the only per-container variation is the min-bytes computation,
// which a closure carries.
type containerFixup struct {
	serItem     *serfn       // address of serArray.serItem / serMap.serItem
	deserItem   *deserfn     // address of deserArray.deserItem / deserMap.deserItem
	setMinBytes func(int)    // setter for minItemBytes (array) or 1+min (map)
	nodeChild   **schemaNode // address of arrayNode.items / mapNode.values
	name        string       // referenced named-type name
	parentName  string       // enclosing scope, for the namespace-qualified retry
	ctxLabel    string       // "array" or "map" for error messages
}

// defaultFixup defers a record field's default-value resolution + encoding
// to finalize, for a field whose OUTER type resolved at build time but whose
// type tree contains a forward-referenced descendant (e.g. array<fwd-ref>
// items, map<fwd-ref> values, or an inline record with a fwd-ref field).
// encodeDefault recurses into items/values/fields and dereferences each
// child's kind, so running it at build time against a not-yet-wired child
// node panics; deferring runs it after every container/field fixup has wired
// the descendants. The fwd-ref-OUTER case (the whole field type is a bare
// forward-ref name) is handled by recordFieldFixup instead, which also
// carries the default and resolves the node by name in finalize.
type defaultFixup struct {
	nd         *schemaNode // enclosing record; its serRecord/deserRecord are the field tables
	idx        int
	node       *schemaNode // the field's already-built outer node (children wired by other fixups)
	defaultVal any         // parsed-but-not-yet-coerced JSON default
}

// captureFwdRef is the shared boilerplate used by every site that might
// encounter a forward reference inside a nested build (record field,
// array items, map values). On success it returns (false, "", nil). On
// an unknownPrimitiveError it returns (true, name, nil) so the caller
// can queue a fixup. On any other error it wraps with ctxLabel and
// returns (false, "", err).
func captureFwdRef(err error, ctxLabel string) (isFwdRef bool, fwdName string, wrapped error) {
	if err == nil {
		return false, "", nil
	}
	if pe := (*unknownPrimitiveError)(nil); errors.As(err, &pe) {
		return true, pe.p, nil
	}
	return false, "", fmt.Errorf("invalid %s: %v", ctxLabel, err)
}

// namedType holds the compiled artifacts for a named Avro type (record,
// enum, fixed) so they can be looked up by name during schema building.
type namedType struct {
	// node carries the type's compiled artifacts as well as its metadata:
	// node.ser / node.deser are the wire functions, and node.serRecord /
	// node.deserRecord are the record tables (nil for enum and fixed). They
	// are read through here rather than copied beside it because a copy taken
	// at registration and a node read later are two answers to one question,
	// and only one of them can be re-pointed.
	//
	// The copy was safe — registration happens in the same statement that
	// builds the node, from the same expressions — but safe is not the bar: a
	// custom-typed reference wraps b.ser while deliberately leaving node.ser
	// bare (applyCustomTypes), so the two spellings ALREADY mean different
	// things elsewhere in this file, and a reader cannot tell from the name
	// which one a copy froze.
	node *schemaNode
	// hadCustomType is true when this named type was DEFINED by a parse
	// that wired at least one CustomType anywhere (deliberately coarse —
	// every definition of a custom-wired parse counts; stamped at
	// finalize, see registerNamed). The cache-reference boundary guard
	// compares it against the referencing parse's registrations to
	// allow consistent reuse and reject mismatches — the documented
	// remediation path "re-parse Inner with the CT first" depends on
	// this signal.
	hadCustomType bool
}

type builder struct {
	ser   serfn
	deser deserfn

	named        map[string]*namedType
	building     map[*schemaNode]struct{} // record/error nodes whose field loop is in progress (shared across nest, like named)
	definedNamed []*namedType             // named types DEFINED by this parse (vs inherited); stamped custom-affected at finalize
	// definedSet is the membership form of this-parse definitions. Unlike
	// definedNamed (a per-builder accumulator merged up only at unnest), it is
	// SHARED BY REFERENCE across nest() — like named/building/cachedNames — so a
	// guard running in a NESTED builder (e.g. while building a ["null","Self"]
	// field) can test whether a resolved reference points at a name THIS parse
	// defined, before unnest would have propagated definedNamed upward. A
	// re-registered cached name's fresh *namedType lands here; cachedNames cannot
	// make that distinction (it marks such a name as both inherited and redefined).
	definedSet      map[*namedType]bool
	missing         []unionMissing
	mfixups         []metaFixup
	fieldFixups     []recordFieldFixup
	containerFixups []containerFixup
	defaultFixups   []defaultFixup

	meta        fieldMeta
	canon       aschema
	node        *schemaNode
	checkName   func(string) error // nil means strict (default)
	customTypes []CustomType
	custom      map[*schemaNode]*customWiring
	// sawInheritedCustom is set when a reference resolves to a
	// SchemaCache-inherited named type stamped hadCustomType by its
	// defining parse: that type's subtree carries baked custom effects
	// this parse's own overlay (b.custom) knows nothing about. Feeds
	// Schema.customBaked.
	sawInheritedCustom bool
	// customMatch memoizes custom-match subtree verdicts per node for this
	// parse ("" = proven match-free, non-"" = a matched-type location; key
	// PRESENCE marks a computed verdict, so "" is a valid entry). Allocated
	// by applySchemaOpts only when CustomTypes are registered, and shared
	// by reference across nest() so the finalize stamping loop and the
	// per-reference cache walks collapse to one walk per node per parse.
	// See customMatchInSubtree for the soundness rules. Nil on white-box
	// test builders — every write is guarded, degrading to unshared walks.
	customMatch map[*schemaNode]string
	// overlayDone marks inherited subtrees overlayInheritedCustom has
	// completed, so N references to the same cached type overlay its nodes
	// once per parse instead of re-walking per reference. Sharing one set
	// across references is sound because the walk's effect is idempotent
	// and order-independent within a parse: buildCustomWiring is
	// deterministic (b.customTypes is fixed after applySchemaOpts) and
	// existing overlay entries are kept, so a second walk over the same
	// nodes could only rebuild identical wiring. Allocated alongside
	// customMatch; nil-tolerated the same way.
	overlayDone map[*schemaNode]bool
	cachedNames map[string]bool // names inherited from SchemaCache, not from this parse
	// allowReRegister permits re-DEFINING an inherited (cachedNames) type
	// instead of erroring "duplicate named type". Set by SchemaCache.Parse only
	// for parses that skip dedup and re-parse to get fresh CustomType wiring
	// (custom parses, and re-parses of a previously-custom-parsed schema).
	allowReRegister bool
	depth           int // current build recursion depth, bounded by maxDepth
	// minBytes is the ONE min-bytes walk shared across every container built in
	// this parse (shared across nests, like named/building). A backward name
	// reference resolves to the fully-built node, so a container whose element is
	// a cyclic type defined earlier pays a full walk at BUILD — and a schema can
	// point any number of containers at that one type, so a fresh walk per
	// container would make the build cost the container count times the per-walk
	// bound. Sharing caps it once. Kept SEPARATE from finalize's walk: build-time
	// calls can run over subtrees whose forward references are not yet wired
	// (provisional), so their memo must not leak into finalize, which recomputes
	// on the fully-wired graph.
	minBytes *minBytesWalk
}

// validNameErr validates a simple name using the builder's configured validator.
func (b *builder) validNameErr(s string) error {
	if b.checkName != nil {
		return b.checkName(s)
	}
	if !validName(s) {
		return fmt.Errorf("invalid name %q", truncForError(s))
	}
	return nil
}

// validFullnameErr validates a dot-separated fullname.
func (b *builder) validFullnameErr(s string) error {
	if s == "" {
		if b.checkName != nil {
			return b.checkName(s)
		}
		return fmt.Errorf("invalid name %q", truncForError(s))
	}
	for part := range strings.SplitSeq(s, ".") {
		if err := b.validNameErr(part); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) nest() *builder {
	return &builder{
		named:           b.named,
		building:        b.building,
		checkName:       b.checkName,
		customTypes:     b.customTypes,
		custom:          b.custom,
		customMatch:     b.customMatch,
		overlayDone:     b.overlayDone,
		cachedNames:     b.cachedNames,
		definedSet:      b.definedSet,
		allowReRegister: b.allowReRegister,
		depth:           b.depth,
		minBytes:        b.minBytes,
	}
}

func (b *builder) unnest(nest *builder) {
	b.missing = append(b.missing, nest.missing...)
	b.definedNamed = append(b.definedNamed, nest.definedNamed...)
	b.mfixups = append(b.mfixups, nest.mfixups...)
	b.fieldFixups = append(b.fieldFixups, nest.fieldFixups...)
	b.containerFixups = append(b.containerFixups, nest.containerFixups...)
	b.defaultFixups = append(b.defaultFixups, nest.defaultFixups...)
	if len(nest.custom) > 0 {
		if b.custom == nil {
			b.custom = make(map[*schemaNode]*customWiring, len(nest.custom))
		}
		maps.Copy(b.custom, nest.custom)
	}
	b.sawInheritedCustom = b.sawInheritedCustom || nest.sawInheritedCustom
}

// putCustomWiring stores the wiring under node, allocating b.custom on
// demand. Used by applyCustomTypes after building the per-node closures.
func (b *builder) putCustomWiring(node *schemaNode, w *customWiring) {
	if b.custom == nil {
		b.custom = make(map[*schemaNode]*customWiring)
	}
	b.custom[node] = w
}

// makeCustomSer wraps base with the custom-Encode function ce: apply ce to the
// value, then encode the converted value via base. SINGLE definition of the
// binary custom-Encode wrap, shared by applyCustomTypes (in-order references)
// and customWrappedSer (forward-ref finalize fixups) so the two paths cannot
// drift — a forward reference to a custom-encoded named type must emit the same
// wire as an in-order one.
func makeCustomSer(ce func(reflect.Value) (reflect.Value, error), base serfn) serfn {
	return func(dst []byte, v reflect.Value, depth int) ([]byte, error) {
		v, err := ce(v)
		if err != nil {
			return nil, err
		}
		// Pass depth unchanged: the custom wrapper annotates an existing
		// schema node, it is not a new nesting level. base does the node's
		// own depth accounting, and the decode wrapper + JSON path both
		// charge 0 here — re-entering at depth+1 would make a custom on a
		// recursive node trip errTooDeep a level shallower per recursion
		// than decode, breaking round-trips.
		return base(dst, v, depth)
	}
}

// customWrappedSer returns base wrapped with node's custom-Encode chain when
// one is registered (the same wrap applyCustomTypes installs for an in-order
// reference), else base unchanged. The forward-ref fixups in finalize call this
// so a forward reference to a custom-encoded named type applies the CustomType
// on the binary path — previously they used the unwrapped namedType.ser while
// the JSON encoder applied the custom (a silent binary↔JSON divergence: the
// forward-referenced field encoded raw on binary but converted on JSON).
func (b *builder) customWrappedSer(node *schemaNode, base serfn) serfn {
	if w := b.custom[node]; w != nil && w.encode != nil {
		return makeCustomSer(w.encode, base)
	}
	return base
}

// customWrappedDeser is the decode dual of customWrappedSer: it returns base
// wrapped with node's custom-Decode chain when one is registered (the same wrap
// applyCustomTypes installs for an in-order reference), else base unchanged.
// The forward-ref fixups in finalize call this so a forward reference to a
// custom-decoded named type applies the CustomType on the binary path —
// otherwise the forward-referenced field decodes raw on binary while the JSON
// decoder applies the custom via the patched node.decodeJSON (the decode twin
// of the encode divergence). Logical-suppression with no Decode callback needs
// no wrap here: the suppressed (raw) deser is already baked onto the shared
// leaf node, so a forward reference inherits it on both wire formats.
func (b *builder) customWrappedDeser(node *schemaNode, base deserfn) deserfn {
	if w := b.custom[node]; w != nil && len(w.decoders) > 0 {
		return wrapDeserWithCustomDecoders(base, w.decoders, w.sn)
	}
	return base
}

// primFastInfo holds per-primitive bindings for both the array and map
// fast paths. Indexed by the canonical primitive name; missing entries
// fall back to the generic (function-pointer) per-element path.
type primFastInfo struct {
	elemKind            reflect.Kind
	serArrayFn          func(*serArray) serfn
	serMapFn            func(*serMap) serfn
	deserArrayLoop      func(src []byte, sliceVal reflect.Value, start, count int, sl *slab) ([]byte, error)
	deserArrayIfaceLoop func(src []byte, slice []any, start, count int, sl *slab) ([]byte, error)
	deserMapBlock       func(src []byte, mapVal, keyVal, elemVal reflect.Value, count int, sl *slab) ([]byte, error)
	deserMapIfaceVal    deserIfaceFn
	deserArrayNative    func(sliceVal reflect.Value, src []byte, start, count int, sl *slab) (bool, []byte, error)
	deserMapNative      func(mapVal reflect.Value, src []byte, count int, sl *slab) (bool, []byte, error)
}

var primFast = map[string]primFastInfo{
	"string": {
		reflect.String,
		func(s *serArray) serfn { return s.serString }, func(s *serMap) serfn { return s.serString },
		deserArrayStringLoop, deserArrayStringIfaceLoop, deserMapStringBlock, deserStringIface,
		deserNativeArrayStringLoop, deserNativeMapStringBlock,
	},
	"boolean": {
		reflect.Bool,
		func(s *serArray) serfn { return s.serBoolean }, func(s *serMap) serfn { return s.serBoolean },
		deserArrayBooleanLoop, deserArrayBooleanIfaceLoop, deserMapBooleanBlock, deserBooleanIface,
		deserNativeArrayBooleanLoop, deserNativeMapBooleanBlock,
	},
	"int": {
		reflect.Int32,
		func(s *serArray) serfn { return s.serInt }, func(s *serMap) serfn { return s.serInt },
		deserArrayIntLoop, deserArrayIntIfaceLoop, deserMapIntBlock, deserIntIface,
		deserNativeArrayIntLoop, deserNativeMapIntBlock,
	},
	"long": {
		reflect.Int64,
		func(s *serArray) serfn { return s.serLong }, func(s *serMap) serfn { return s.serLong },
		deserArrayLongLoop, deserArrayLongIfaceLoop, deserMapLongBlock, deserLongIface,
		deserNativeArrayLongLoop, deserNativeMapLongBlock,
	},
	"float": {
		reflect.Float32,
		func(s *serArray) serfn { return s.serFloat }, func(s *serMap) serfn { return s.serFloat },
		deserArrayFloatLoop, deserArrayFloatIfaceLoop, deserMapFloatBlock, deserFloatIface,
		deserNativeArrayFloatLoop, deserNativeMapFloatBlock,
	},
	"double": {
		reflect.Float64,
		func(s *serArray) serfn { return s.serDouble }, func(s *serMap) serfn { return s.serDouble },
		deserArrayDoubleLoop, deserArrayDoubleIfaceLoop, deserMapDoubleBlock, deserDoubleIface,
		deserNativeArrayDoubleLoop, deserNativeMapDoubleBlock,
	},
}

// registerNamed stores nt under name and records it as a definition of
// this parse. The custom-affected flag (hadCustomType) is stamped for
// ALL of this parse's definitions at finalize, after applyCustomTypes
// has wired every node: registration happens early so self-references
// resolve mid-build, and a stamp taken here would predate the wiring it
// reports — permanently for a type whose OWN node matches the
// CustomType (fixed, enum), since no later per-arm re-stamp sees it.
func (b *builder) registerNamed(name string, nt *namedType) {
	b.named[name] = nt
	b.definedNamed = append(b.definedNamed, nt)
	// Mirror the append into the reference-shared membership set so the
	// cached-ref guard can recognize a this-parse definition from inside a
	// nested builder. Nil only for white-box test builders that construct a
	// builder literal without the top-level init; those degrade to the
	// cachedNames-only behavior (the guard's other early return).
	if b.definedSet != nil {
		b.definedSet[nt] = true
	}
}

// leadingDotName reports whether name spells the explicit
// null-namespace escape — a single leading dot with no other dot —
// and returns the fullname it denotes: ".x" is the null-namespace
// fullname "x", and "." is the bare empty name "". This is Java's Name
// constructor rule (Schema.java ~1455, release-1.12.0: lastDot split,
// then `if ("".equals(space)) space = null`), the same rule
// qualifyAliases applies to aliases; a name whose namespace part is
// non-empty (".a.b", "a.b") is a fullname verbatim. Shared by the
// definition build (aobject naming), reference resolution
// (scopedRefKeys), and the metadata fullname computation
// (nodeFullname) so the escape cannot drift between them.
func leadingDotName(name string) (string, bool) {
	if strings.LastIndexByte(name, '.') == 0 {
		return name[1:], true
	}
	return name, false
}

// scopedRefKeys writes the lookup keys for a name reference into dst in
// binding-precedence order and returns the filled prefix: a dotted
// reference is an exact fullname lookup; a bare reference tries the
// enclosing-namespace-qualified key first, then the bare key (the
// null-namespace type, when one exists). This IS the name-binding rule —
// Java's Names.get order (Schema.java: new Name(o, space) looked up
// before the null-space retry) and fastavro's schema_name qualification.
// Checking the bare key first would bind a null-namespace type in
// preference to the in-scope one whenever the two share a short name,
// silently changing the wire contract of every field using the
// reference. Every resolver — wire build/finalize (resolveNamedRef),
// canonical (lookupCanonDef), metadata (lookupNameRef) — derives its key
// order here so the precedence cannot drift between them.
func scopedRefKeys(dst *[2]string, ref, ns string) []string {
	if strings.Contains(ref, ".") {
		if short, ok := leadingDotName(ref); ok && short != "" {
			// ".x" is the null-namespace escape: an exact lookup of the
			// fullname "x", never qualified into the enclosing namespace.
			// A bare "." stays as-written and can only miss, since nothing
			// registers ".", keeping the empty-name type unreferenceable in
			// spelling (NOT_BUGS #60).
			dst[0] = short
			return dst[:1]
		}
		dst[0] = ref
		return dst[:1]
	}
	if ns != "" {
		dst[0], dst[1] = ns+"."+ref, ref
		return dst[:2]
	}
	dst[0] = ref
	return dst[:1]
}

// resolveNamedRef looks up a named-type reference by name, in
// scopedRefKeys precedence order against parentName's namespace. Returns
// the resolved fullname and the namedType, or ("", nil) when unresolved.
//
// Shared by tryAssignNamedRef (build-time, backward references) and
// finalize (forward-reference fixups) so the qualification rule lives in
// ONE place. finalize previously did a bare b.named[name] lookup with no
// qualified retry, so a forward reference into a namespaced scope failed
// ("unknown type") even though the byte-identical backward-ordered schema
// resolved through tryAssignNamedRef's retry — an order-dependent parse
// rejection of a valid schema.
func (b *builder) resolveNamedRef(name, parentName string) (string, *namedType) {
	var keys [2]string
	for _, k := range scopedRefKeys(&keys, name, namespaceOf(parentName)) {
		if nt := b.named[k]; nt != nil {
			return k, nt
		}
	}
	return "", nil
}

// tryAssignNamedRef resolves a named-type reference, possibly with
// namespace qualification against parentName. Returns true on hit (with
// b.ser / b.deser / b.meta / b.node populated and, when setCanon is
// true, b.canon set to the resolved name). Shared by buildPrimitive's
// bare-string named-ref path and buildComplex's wrapped-form
// {"type":"Name"} path so the rejectCachedRefIfCustomTypeWouldMatch
// gate and the namespace-qualified retry agree.
func (b *builder) tryAssignNamedRef(name, parentName string, setCanon bool) (bool, error) {
	resolved, nt := b.resolveNamedRef(name, parentName)
	if nt == nil {
		return false, nil
	}
	if err := b.rejectCachedRefIfCustomTypeWouldMatch(resolved, nt); err != nil {
		return true, err
	}
	// Only an INHERITED type can carry the stamp at reference time: local
	// definitions are stamped at this parse's own finalize, after build.
	if nt.hadCustomType {
		b.sawInheritedCustom = true
	}
	// Cross-parse inherited subtree (same condition family as the guard
	// above: not defined this parse, name inherited from the cache):
	// complete the overlay so resolve-time custom re-application sees the
	// inherited nodes. Locally defined types are wired by applyCustomTypes
	// at their own build. The visited set is the per-parse b.overlayDone,
	// so repeated references to the same inherited type walk its subtree
	// once per parse, not once per reference (see the field's soundness
	// note); the nil fallback covers white-box test builders only.
	if len(b.customTypes) > 0 && !b.definedSet[nt] && b.cachedNames[resolved] {
		if b.overlayDone == nil {
			b.overlayDone = make(map[*schemaNode]bool)
		}
		b.overlayInheritedCustom(nt.node, b.overlayDone)
	}
	if setCanon {
		b.canon = aschema{primitive: resolved}
	}
	b.ser = nt.node.ser
	b.deser = nt.node.deser
	if nt.node.serRecord != nil {
		b.meta = fieldMeta{avroType: "record", serRecord: nt.node.serRecord, deserRecord: nt.node.deserRecord}
	}
	b.node = nt.node
	return true, nil
}

func (b *builder) finalize() error {
	for _, m := range b.missing {
		for idx, name := range m.missing {
			_, nt := b.resolveNamedRef(name, m.parentName)
			if nt == nil {
				return fmt.Errorf("unknown type %q", truncForError(name))
			}
			m.ser.fns[idx] = b.customWrappedSer(nt.node, nt.node.ser)
			m.deser.fns[idx] = b.customWrappedDeser(nt.node, nt.node.deser)
			// The builder left branches[idx] nil for this forward-ref
			// branch (the named type wasn't built yet). The binary path
			// dispatches through the ser/deser fn tables (patched above),
			// but the JSON encode/decode, schema-resolution, and
			// union-default-validation paths walk node.branches directly
			// and would dereference the nil node.
			if m.branches != nil {
				m.branches[idx] = nt.node
			}
		}
		// With every branch node wired, re-derive the union's
		// name-dependent artifacts (duplicate-branch check, TaggedUnions
		// branch-name tables) from the RESOLVED names — buildUnion ran
		// them with the unresolved as-written names for fwd-ref branches.
		if m.branches != nil && m.deser != nil {
			if err := finalizeUnionNames(m.ser.tags, m.deser, m.branches); err != nil {
				return err
			}
		}
	}
	for _, m := range b.mfixups {
		_, nt := b.resolveNamedRef(m.name, m.parentName)
		if nt == nil {
			return fmt.Errorf("unknown type %q", truncForError(m.name))
		}
		m.meta.serRecord = nt.node.serRecord
		m.meta.deserRecord = nt.node.deserRecord
	}
	// Phase 1: wire every forward-referenced record-field node. Default
	// ENCODING is deferred to phase 2 below so it runs only after every
	// field AND container child node is wired — encodeDefault recurses into
	// a field's child nodes, and a not-yet-wired child would nil-panic.
	for _, m := range b.fieldFixups {
		_, nt := b.resolveNamedRef(m.name, m.parentName)
		if nt == nil {
			return fmt.Errorf("unknown type %q", truncForError(m.name))
		}
		m.nd.serRecord.fields[m.idx].fn = b.customWrappedSer(nt.node, nt.node.ser)
		m.nd.deserRecord.fields[m.idx].fn = b.customWrappedDeser(nt.node, nt.node.deser)
		if nt.node.serRecord != nil {
			// The encode and decode entries for one field were handed the
			// SAME *fieldMeta when the field was built, so naming the type
			// once updates both; the second write was a rewrite of the word
			// the first had just stored.
			m.nd.serRecord.fields[m.idx].meta.avroType = "record"
			m.nd.serRecord.fields[m.idx].meta.serRecord = nt.node.serRecord
			m.nd.deserRecord.fields[m.idx].meta.deserRecord = nt.node.deserRecord
		}
		m.nd.fields[m.idx].node = nt.node
	}
	// Phase 1b: wire every forward-referenced array/map container child.
	//
	// One walk is shared across the whole loop rather than one per fixup: a
	// schema can point any number of containers at one forward-referenced
	// subtree (N array fields all of items "L0"), and each is a distinct fixup
	// here. A fresh walk per fixup would recompute that subtree N times — for a
	// cyclic one, N times the full allowance — so the loop's cost would be the
	// container count times the per-walk bound while only the second factor was
	// capped. The graph is fully wired by this phase (phase 1a resolved every
	// field reference), so the shared memo caches exact values for the acyclic
	// case and the shared allowance bounds the cyclic residue once.
	mbw := newMinBytesWalk()
	for _, m := range b.containerFixups {
		_, nt := b.resolveNamedRef(m.name, m.parentName)
		if nt == nil {
			return fmt.Errorf("%s references unknown named type %q", m.ctxLabel, truncForError(m.name))
		}
		*m.serItem = b.customWrappedSer(nt.node, nt.node.ser)
		*m.deserItem = b.customWrappedDeser(nt.node, nt.node.deser)
		m.setMinBytes(mbw.minBytesOf(nt.node))
		*m.nodeChild = nt.node
	}
	// Phase 2 — deferred field defaults, in two passes. encodeDefault fills
	// an absent nested record field from its resolved f.defaultVal, so every
	// field's default VALUE must be recorded (phase 2a) before any default's
	// binary bytes are encoded (phase 2b); otherwise a field default that
	// nests into a sibling-defaulted record reads a nil f.defaultVal and
	// mis-encodes. Both deferral kinds participate: fwd-ref-OUTER fields
	// (recordFieldFixup, node resolved by name) and container/nested fields
	// whose outer type resolved but whose descendant was a fwd-ref
	// (defaultFixup, node already known).
	type pendingDefault struct {
		node      *schemaNode
		name      string
		converted any
		srf       *serRecordField
	}
	var pending []pendingDefault
	// Phase 2a: resolve + record every deferred default's value.
	for _, m := range b.fieldFixups {
		if !m.hasDefault {
			continue
		}
		_, nt := b.resolveNamedRef(m.name, m.parentName)
		if nt == nil {
			continue
		}
		node := nt.node
		if node == nil {
			continue
		}
		name := m.nd.serRecord.fields[m.idx].name
		converted, err := resolveFieldDefaultValue(
			coerceDefault(m.defaultVal, node), node, name,
			&m.nd.deserRecord.fields[m.idx], &m.nd.fields[m.idx],
		)
		if err != nil {
			return fmt.Errorf("type %q: %w", truncForError(m.name), err)
		}
		pending = append(pending, pendingDefault{node, name, converted, &m.nd.serRecord.fields[m.idx]})
	}
	for _, m := range b.defaultFixups {
		name := m.nd.serRecord.fields[m.idx].name
		converted, err := resolveFieldDefaultValue(
			coerceDefault(m.defaultVal, m.node), m.node, name,
			&m.nd.deserRecord.fields[m.idx], &m.nd.fields[m.idx],
		)
		if err != nil {
			return err
		}
		pending = append(pending, pendingDefault{m.node, name, converted, &m.nd.serRecord.fields[m.idx]})
	}
	// Phase 2b: encode binary default bytes now that every default value
	// (inline-built and deferred) is recorded on its field node.
	for _, p := range pending {
		if err := encodeFieldDefaultBytes(p.converted, p.node, p.name, p.srf); err != nil {
			return err
		}
	}
	// Stamp each named type THIS parse defined as custom-affected iff a
	// non-wildcard registration matches somewhere in ITS OWN subtree — exactly
	// when baked effects (logical suppression on shared leaf nodes, callback
	// wraps in the composed ser/deser) live inside the type. The predicate is
	// findCustomTypeMatchInSubtree, the same one the cache-boundary guard
	// applies on the reference side, so stamp and guard cannot disagree either
	// way. Two consequences of asking about the type's OWN subtree: the walk
	// crosses into inherited subtrees, so a wrapper around an inherited
	// custom-baked reference is stampable — a wired-this-parse test instead
	// makes the guard's "re-parse with the CustomType first" remediation
	// unsatisfiable for transitive chains — and a sibling whose subtree has no
	// match is not stamped merely because the parse wired something elsewhere,
	// which would false-reject later no-custom references to it.
	//
	// Taken at finalize, after the tree is built, because registration happens
	// early to support self-references. Inherited entries are not in
	// definedNamed and keep the flag from their defining parse.
	if len(b.customTypes) > 0 {
		for _, nt := range b.definedNamed {
			if b.customMatchInSubtree(nt.node) != "" {
				nt.hadCustomType = true
			}
		}
	}
	return nil
}

func (s *aschema) unionTypeName() (string, string, error) {
	if s.primitive != "" {
		return s.primitive, "", nil
	}
	// Non-nil (even when empty) means the JSON was an array: a nested
	// union, which a union may not immediately contain (Java: "Nested
	// union" fires for any union-typed member, zero branches included).
	if s.union != nil {
		return "union", "", errors.New("unions cannot immediately contain other unions")
	}
	if isNamedKind(s.object.Type) {
		return s.object.Type, s.object.Name, nil
	}
	return s.object.Type, "", nil
}

type unknownPrimitiveError struct{ p string }

func (e *unknownPrimitiveError) Error() string {
	return fmt.Sprintf("unknown primitive %q", truncForError(e.p))
}

func (b *builder) build(parentName string, s *aschema) error {
	// Discriminate union-ness by non-nil, not length: `[]` parses to a
	// non-nil zero-branch union (legal — Java's UnionSchema constructor,
	// fastavro, and avro-rs all accept it; no value can ever encode or
	// decode against it, but the schema itself is well-formed).
	if s == nil || s.primitive == "" && s.object == nil && s.union == nil {
		return errors.New("schema is not a primitive, complex, nor union")
	}
	if b.depth >= maxDepth {
		return fmt.Errorf("schema nests deeper than the supported limit (%d)", maxDepth)
	}
	// Seed the shared per-parse min-bytes walk at the root's first build, before
	// any nest() copies the pointer, so every container built below shares it.
	// Parse and SchemaCache.Parse seed it in their constructors; this covers a
	// builder constructed directly (white-box tests) so the build-path container
	// sites never dereference a nil walk. See b.minBytes.
	if b.minBytes == nil {
		b.minBytes = newMinBytesWalk()
	}
	b.depth++
	defer func() { b.depth-- }()

	var err error
	switch {
	case s.primitive != "":
		err = b.buildPrimitive(parentName, s)
	case s.union != nil:
		err = b.buildUnion(parentName, s)
	default:
		err = b.buildComplex(parentName, s)
	}
	if err != nil {
		return err
	}
	// Propagate extra schema properties to the node (for CustomType callbacks).
	if b.node != nil && s.object != nil && len(s.object.extra) > 0 {
		b.node.props = s.object.extra
	}
	// Apply custom types to newly built nodes (not unions — custom
	// types fire on individual branches, not the union container).
	if len(b.customTypes) > 0 && b.node != nil && b.node.kind != "union" {
		if err := b.applyCustomTypes(b.node); err != nil {
			return err
		}
	}
	return nil
}

// buildCustomSN builds a public SchemaNode from an internal schemaNode.
// Built once per node at parse time and cached for CustomType callbacks.
func buildCustomSN(node *schemaNode) *SchemaNode {
	sn := &SchemaNode{
		Type:        node.kind,
		LogicalType: node.logical,
		Name:        node.name,
		Size:        node.size,
		Precision:   node.precision,
		Scale:       node.scale,
		Symbols:     node.symbols,
	}
	if node.props != nil {
		sn.Props = node.props
	}
	return sn
}

// hasMatchingCustomType checks if any registered custom type would match
// a node with the given kind and logical type. Used to skip the built-in
// logical-type *decoder* when a custom type replaces it (the deser-side
// of the suppression contract — see [CustomType.Decode]'s docstring:
// "If nil, the built-in logical type handler is bypassed and the base
// Avro type decoder is used directly").
//
// The encode side has different semantics ([CustomType.Encode]: "If nil,
// the built-in logical type encoder is used"), so encoder-suppression
// uses [hasMatchingCustomTypeWithEncode] instead — only suppress the
// built-in encoder when the user actually provided an Encode callback
// to wrap it with.
func (b *builder) hasMatchingCustomType(kind, logical string) bool {
	return b.hasMatchingCustomTypeCond(kind, logical, false)
}

// hasMatchingCustomTypeWithEncode reports whether any matching CustomType
// has a non-nil Encode callback. Used to gate suppression of the
// built-in logical encoder: per [CustomType.Encode]'s docstring, an
// Encode==nil CustomType leaves the built-in encoder in place (so a
// user registering only Decode keeps the convenient time.Time /
// *big.Rat / avro.Duration encoder), while an Encode!=nil CustomType
// wraps the base (raw) encoder with the user's callback.
func (b *builder) hasMatchingCustomTypeWithEncode(kind, logical string) bool {
	return b.hasMatchingCustomTypeCond(kind, logical, true)
}

// hasMatchingCustomTypeCond is the shared body. When requireEncode is
// true, the predicate additionally requires ct.Encode != nil — used by
// the encoder-suppression gate. When false, the predicate matches any
// registered CustomType — used by the decoder-suppression gate (where
// Decode==nil still bypasses the built-in per the doc).
func (b *builder) hasMatchingCustomTypeCond(kind, logical string, requireEncode bool) bool {
	for _, ct := range b.customTypes {
		// Wildcards (both empty) should not suppress built-in
		// handlers — they use ErrSkipCustomType at runtime.
		if ct.LogicalType == "" && ct.AvroType == "" {
			continue
		}
		if ct.LogicalType != "" && ct.LogicalType != logical {
			continue
		}
		if ct.AvroType != "" && ct.AvroType != kind {
			continue
		}
		if requireEncode && ct.Encode == nil {
			continue
		}
		return true
	}
	return false
}

func (b *builder) applyCustomTypes(node *schemaNode) error {
	// Validate NewCustomType-created types with unsupported A type.
	for _, ct := range b.customTypes {
		if ct.needsAvroType && ct.AvroType == "" {
			return fmt.Errorf("avro: custom type %q: unsupported Avro native type for NewCustomType (use CustomType struct for non-primitive backing types)", ct.LogicalType)
		}
	}

	wiring := b.buildCustomWiring(node)
	if wiring == nil {
		return nil
	}

	if wiring.encode != nil {
		// Wrap the binary serializer. We update b.ser (which becomes the
		// Schema's ser) but NOT node.ser, so named types in the cache
		// keep their unwrapped ser/deser. The wrap closure is built by
		// makeCustomSer, shared with the forward-ref finalize fixups
		// (customWrappedSer) so an in-order reference and a forward reference
		// to the same custom-encoded named type apply the SAME wrap.
		b.ser = makeCustomSer(wiring.encode, node.ser)
	}

	// See buildCustomWiring for what this narrowing buys and the
	// suppression contract behind it.
	jsonAppliesLogical := wiring.suppressLogical && jsonDecodeAppliesLogical(node)
	if len(wiring.decoders) > 0 {
		b.deser = wrapDeserWithCustomDecoders(node.deser, wiring.decoders, wiring.sn)
		// JSON-side: wrap the node's per-decode dispatch with a
		// closure that captures the decoder chain. The JSON runtime
		// (decodeValue) checks node.decodeJSON first and falls back
		// to decodeKind otherwise — no per-call map lookup, no
		// recursion guard, no shared mutable state.
		node.decodeJSON = wrapDecodeJSONWithCustomDecoders(wiring.decoders, wiring.sn, wiring.suppressLogical)
	} else if jsonAppliesLogical {
		// No Decode callback (Encode-only, OR no callbacks at all) on a logical
		// node that the binary path suppresses (non-wildcard): the user
		// receives the RAW Avro-native value (CustomType.Decode docstring:
		// "If nil, the built-in logical type handler is bypassed ... the
		// base Avro type decoder is used directly, producing raw values").
		// decodeKind applies the logical transform unless suppressed, so
		// install the raw-decode wrapper with an empty decoder chain to
		// produce the same raw value through DecodeJSON. A wildcard
		// (suppressLogical false ⇒ jsonAppliesLogical false) skips this —
		// decodeKind keeps the logical transform, matching the binary wildcard.
		node.decodeJSON = wrapDecodeJSONWithCustomDecoders(nil, wiring.sn, wiring.suppressLogical)
	}

	b.putCustomWiring(node, wiring)
	b.meta.hasCustomType = true
	return nil
}

// buildCustomWiring collects the per-node custom-type wiring from this
// parse's registrations: the encode chain closure, the decoder chain, the
// callback SchemaNode, and the suppression flags. Returns nil when no
// registration matches or applies. PURE with respect to the builder and
// the node — no ser/deser wraps, no node mutation, no overlay insertion —
// so it is shared by applyCustomTypes (newly built nodes: wraps + overlay)
// and overlayInheritedCustom (SchemaCache-inherited nodes: overlay only;
// the wraps already live inside the inherited composition, baked by the
// type's own defining parse under the guard-enforced identical
// registrations).
func (b *builder) buildCustomWiring(node *schemaNode) *customWiring {
	// Collect all matching encoders and decoders for this node.
	type encoder struct {
		goType reflect.Type
		fn     func(any, *SchemaNode) (any, error)
	}
	var encoders []encoder
	var decoders []func(any, *SchemaNode) (any, error)

	for _, ct := range b.customTypes {
		if !ct.matches(node) {
			continue
		}
		if ct.Encode != nil {
			encoders = append(encoders, encoder{goType: ct.GoType, fn: ct.Encode})
		}
		if ct.Decode != nil {
			decoders = append(decoders, ct.Decode)
		}
	}

	// suppressLogical mirrors the binary decoder-suppression gate
	// (hasMatchingCustomType, the same predicate the primitive/decimal/fixed
	// builds use to decide between the raw and logical deserializer). The
	// binary build replaces the built-in logical deserializer with the raw one
	// whenever ANY non-wildcard CustomType matches — INCLUDING a matcher with
	// no Encode/Decode callbacks (per CustomType.Decode: "If nil, the built-in
	// logical type handler is bypassed ... producing raw Avro-native values").
	// Wildcards (empty LogicalType AND AvroType) are excluded by the gate.
	suppressLogical := b.hasMatchingCustomType(node.kind, node.logical)
	// jsonAppliesLogical narrows that to nodes whose JSON decoder actually
	// transforms the raw value (a logical decodeKind would apply): only those
	// need a JSON-side suppress-wrapper to mirror the binary raw decode.
	jsonAppliesLogical := suppressLogical && jsonDecodeAppliesLogical(node)

	// Nothing to wire when there are no callbacks AND the JSON path has no
	// suppression to mirror. A no-callback matcher on a LOGICAL node falls
	// THROUGH this guard (jsonAppliesLogical is true) so its JSON decode is
	// suppressed to raw, matching the binary path — without this, DecodeJSON
	// returns the enriched logical type (time.Time / *big.Rat) while Decode
	// returns the raw Avro-native value. A wildcard, or a no-callback matcher
	// on a non-logical node, has nothing to mirror and returns here.
	if len(encoders) == 0 && len(decoders) == 0 && !jsonAppliesLogical {
		return nil
	}

	// Build the cached SchemaNode for callbacks and the wiring entry.
	sn := buildCustomSN(node)
	wiring := &customWiring{sn: sn}

	if len(encoders) > 0 {
		customEncode := func(v reflect.Value) (reflect.Value, error) {
			// Dereference pointers and interface wrappers so GoType
			// matching compares against the concrete type. Check GoType
			// at each level so pointer-valued GoTypes (e.g. *url.URL)
			// match before the pointer is stripped. Capped at
			// maxIndirectDepth so a self-referential interface
			// (var p any; p = &p) can't spin forever here.
			for range maxIndirectDepth {
				if v.Kind() != reflect.Pointer && v.Kind() != reflect.Interface {
					break
				}
				if v.IsNil() {
					return v, nil
				}
				for _, enc := range encoders {
					if enc.goType != nil && v.Type() == enc.goType {
						result, err := enc.fn(v.Interface(), sn)
						if err != nil {
							if errors.Is(err, ErrSkipCustomType) {
								// Skip this encoder, try the next
								// chain entry — mirrors the
								// value-GoType scan below and the
								// decoder side (custom_type.go) so
								// the ErrSkipCustomType contract
								// holds regardless of GoType shape.
								continue
							}
							return reflect.Value{}, err
						}
						if result == nil {
							return reflect.Value{}, fmt.Errorf("avro: custom type encoder returned nil for %v", v.Type())
						}
						return reflect.ValueOf(result), nil
					}
				}
				v = v.Elem()
			}
			for _, enc := range encoders {
				if enc.goType != nil && v.Type() != enc.goType {
					continue
				}
				result, err := enc.fn(v.Interface(), sn)
				if err != nil {
					if errors.Is(err, ErrSkipCustomType) {
						continue
					}
					return reflect.Value{}, err
				}
				if result == nil {
					return reflect.Value{}, fmt.Errorf("avro: custom type encoder returned nil for %v", v.Type())
				}
				return reflect.ValueOf(result), nil
			}
			return v, nil // no encoder matched, pass through
		}

		// Store the customEncode in the wiring overlay (not on the
		// shared node) so it doesn't leak via the cache.
		wiring.encode = customEncode
	}

	wiring.suppressLogical = suppressLogical
	// encodeSuppresses mirrors the binary ENCODER-suppression gate exactly
	// (hasMatchingCustomTypeWithEncode — excludes wildcards), so the JSON
	// encode arms suppress the built-in logical coercion iff the binary build
	// did. See customWiring.encodeSuppresses.
	wiring.encodeSuppresses = b.hasMatchingCustomTypeWithEncode(node.kind, node.logical)

	if len(decoders) > 0 {
		wiring.decoders = decoders
	}
	return wiring
}

// overlayInheritedCustom completes this parse's custom overlay (b.custom) for a
// SchemaCache-inherited subtree. applyCustomTypes visits only newly built
// nodes, so a reference to an inherited type left its matching nodes without
// overlay entries. The callback wraps still fire on a DIRECT decode — the
// defining parse composed them into the inherited ser/deser — but every
// consumer that RE-APPLIES customs from the overlay skipped them silently:
// Resolve dropped the reader's custom, and the no-callback logical suppression
// gate with it, so a resolved decode returned raw values where a direct decode
// returned wrapped ones. This inserts pure wiring only, no ser/deser wraps and
// no node mutation, since the composition already carries them. Existing
// entries are kept, as a node can be reached through several references.
//
// visited is the PER-PARSE b.overlayDone set, shared across every reference.
// That is sound because the walk is idempotent and order-independent within a
// parse: b.customTypes is fixed after applySchemaOpts, so buildCustomWiring is
// deterministic and a re-visit could only rebuild identical wiring. Skipping
// it changes cost — one subtree walk per parse rather than per reference —
// never outcome.
func (b *builder) overlayInheritedCustom(node *schemaNode, visited map[*schemaNode]bool) {
	if node == nil || visited[node] {
		return
	}
	visited[node] = true
	if b.custom[node] == nil {
		if w := b.buildCustomWiring(node); w != nil {
			b.putCustomWiring(node, w)
		}
	}
	for _, f := range node.fields {
		b.overlayInheritedCustom(f.node, visited)
	}
	b.overlayInheritedCustom(node.items, visited)
	b.overlayInheritedCustom(node.values, visited)
	for _, br := range node.branches {
		b.overlayInheritedCustom(br, visited)
	}
}

func (b *builder) buildPrimitive(parentName string, s *aschema) error {
	b.canon = aschema{primitive: s.primitive}
	b.meta = fieldMeta{avroType: s.primitive}
	fn, exists := serPrimitive[s.primitive]
	if exists {
		b.ser = fn
		b.deser = deserPrimitive[s.primitive]
		b.node = &schemaNode{
			kind:  s.primitive,
			ser:   b.ser,
			deser: b.deser,
		}
		return nil
	}
	// A named type reference (record, enum, fixed). setCanon is passed for
	// both branches: only the namespace-qualified retry needs to rewrite
	// the canon, and the bare path overwrites it with the identical name
	// already set above.
	if found, err := b.tryAssignNamedRef(s.primitive, parentName, true); err != nil || found {
		return err
	}
	return &unknownPrimitiveError{s.primitive}
}

// rejectCachedRefIfCustomTypeWouldMatch errors when this Parse registered a
// CustomType that would match a node inside a cached named type's subtree. The
// cached node's handlers were baked at the original Parse with no knowledge of
// this one's customTypes, so reusing them silently drops the user's CustomType
// on the cached fields. Failing at Parse time — with instructions — beats
// handing back unwrapped raw bytes at runtime. Per the package's intentional
// asymmetries, CustomTypes are scoped to the resulting Schema, which includes
// its referenced types.
func (b *builder) rejectCachedRefIfCustomTypeWouldMatch(refName string, nt *namedType) error {
	if nt == nil || nt.node == nil {
		return nil
	}
	// This guard is ONLY about types inherited across Parses. A name DEFINED in
	// the current Parse — self- and forward references included — has this
	// Parse's CustomTypes in scope and applies them to its single definition,
	// so it is never a stale cross-parse cache.
	//
	// definedSet is the authoritative "defined this parse" test: it holds the
	// *namedType of every definition this parse registered, a cached name
	// re-registered under allowReRegister included. cachedNames cannot make the
	// distinction, since a re-registered name is in cachedNames AND defines a
	// fresh node here — keying the skip on it false-rejects a re-parse of a
	// cached self-referential schema the moment a matching CustomType is added.
	// definedSet is also shared by reference across nest(), so it is already
	// populated when the guard runs in the nested builder building a
	// self-referential field; definedNamed would not be, merging up only at
	// unnest. The cachedNames return below then handles only a name never
	// inherited at all, and a genuine cross-parse reference falls through both
	// returns to the match check.
	if b.definedSet[nt] {
		return nil
	}
	if !b.cachedNames[refName] {
		return nil
	}
	// A cached named type and the Parse referencing it MUST AGREE on whether a
	// matching CustomType is registered. The CustomType's effect is baked onto
	// the SHARED cached node — the binary logical-codec suppression bakes the
	// raw ser/deser onto node.ser/node.deser (build sites), and the JSON wrapper
	// sets node.decodeJSON — and a named-type reference resolves through those
	// node fields, with no per-Schema overlay. So a mismatch silently changes
	// what the referencing Schema decodes/encodes, on BOTH wire formats. Both
	// directions are rejected with the same "make them consistent" remediation.
	currentMatches := ""
	if len(b.customTypes) > 0 {
		currentMatches = b.customMatchInSubtree(nt.node)
	}
	switch {
	case currentMatches != "" && !nt.hadCustomType:
		// Forward: this Parse registers a CustomType matching the cached
		// subtree, but the cached node was built WITHOUT it — reusing it would
		// silently DROP this Parse's custom (the user gets raw/unwrapped values
		// on the cached fields). Re-parse the inner type with the CustomType.
		return fmt.Errorf("avro: cached type %q contains %q which would match a CustomType on this Parse; re-parse %q with the CustomType first", truncForError(refName), truncForError(currentMatches), truncForError(refName))
	case nt.hadCustomType && currentMatches == "":
		// Reverse: the cached node was built WITH a CustomType (its raw ser/deser
		// and JSON decodeJSON bake that conversion onto the shared node), but
		// this Parse registers no matching CustomType — reusing it would
		// silently APPLY the original conversion to a Schema that never opted in
		// (suppressed/raw values on BOTH wire formats). Register the same
		// CustomType in this Parse, or parse the inner type without one.
		return fmt.Errorf("avro: cached type %q was parsed with a CustomType affecting its subtree, but this Parse registers no matching CustomType; reusing it would apply that conversion here — register the CustomType in this Parse, or parse %q without one", truncForError(refName), truncForError(refName))
	}
	return nil
}

// customMatchInSubtree is the memoized entry point over
// findCustomTypeMatchInSubtree: one walk per node per parse, shared by the
// finalize stamping loop, the cache boundary guard, and the overlay
// completion's guard twin. Without the memo each caller re-walks from scratch
// — O(defs × reachable nodes) at finalize, quadratic on a backward-reference
// chain, and O(references × subtree) on a SchemaCache parse.
//
// Sound because b.customTypes is FIXED for the builder's lifetime and every
// queried subtree is fully built when walked. Two write rules keep verdicts
// EXACT on cyclic graphs:
//
//   - A clean completion writes "" for EVERY visited node: reachability is
//     transitive, so each one's reachable set is a subset of the root's, just
//     proved match-free. A "" that merely bubbled up mid-walk is NOT written
//     per-node — a completed child can still reach a match through a back-edge
//     to a node higher on the stack.
//   - A match writes the location for the nodes it unwinds through, each of
//     which reaches that match inside its own subtree.
//
// The location STRING is whichever match the walk order found first. Callers
// branch only on emptiness; the string just names a type in the guard's error.
func (b *builder) customMatchInSubtree(node *schemaNode) string {
	if m, ok := b.customMatch[node]; ok {
		return m
	}
	visited := make(map[*schemaNode]bool)
	m := b.findCustomTypeMatchInSubtree(node, visited)
	if b.customMatch != nil && m == "" {
		for n := range visited {
			b.customMatch[n] = ""
		}
	}
	return m
}

// findCustomTypeMatchInSubtree is the per-node recursion step: consult the
// per-parse memo, else walk via findCustomTypeMatchInSubtreeWalk, and record
// a found match for this node on the unwind (every node on the walk stack
// reaches the match, so the write is exact; see customMatchInSubtree for the
// full memo contract). White-box test builders may lack the memo map — reads
// of a nil map miss and writes are guarded, degrading to the unshared walk.
func (b *builder) findCustomTypeMatchInSubtree(node *schemaNode, visited map[*schemaNode]bool) string {
	if node == nil {
		return ""
	}
	if m, ok := b.customMatch[node]; ok {
		return m
	}
	m := b.findCustomTypeMatchInSubtreeWalk(node, visited)
	if m != "" && b.customMatch != nil {
		b.customMatch[node] = m
	}
	return m
}

// findCustomTypeMatchInSubtreeWalk walks node and its descendants,
// returning a short location string for the first node whose
// (kind, logical) would match any of b.customTypes. Returns "" if
// no descendant matches. Recursive types are handled via the
// visited set; node.fields, node.items, node.values, node.branches
// cover every container shape (record, array, map, union). Named-
// type recursion is rare in cached-reuse scenarios but the visited
// set keeps it safe.
func (b *builder) findCustomTypeMatchInSubtreeWalk(node *schemaNode, visited map[*schemaNode]bool) string {
	if node == nil || visited[node] {
		return ""
	}
	visited[node] = true
	// Effective logical: prefer the live logical, fall back to the
	// preserved-but-cleared unknownLogical. Built-in logicals are
	// always preserved; only unknown-at-original-Parse logicals end
	// up in unknownLogical. Either may match a CT registered now.
	effLogical := node.logical
	if effLogical == "" {
		effLogical = node.unknownLogical
	}
	for _, ct := range b.customTypes {
		// Wildcard CTs (both empty) opt into runtime ErrSkipCustomType
		// dispatch — they don't reliably suppress built-ins at parse
		// time, so they don't cause silent-drop on cached subtrees.
		// Skip them in this check; only explicitly-typed CTs would
		// silently fail to fire.
		if ct.LogicalType == "" && ct.AvroType == "" {
			continue
		}
		ltMatch := ct.LogicalType == "" || ct.LogicalType == effLogical
		atMatch := ct.AvroType == "" || ct.AvroType == node.kind
		if ltMatch && atMatch && (ct.LogicalType != "" || ct.AvroType != "") {
			if node.name != "" {
				return node.name
			}
			if effLogical != "" {
				return node.kind + "." + effLogical
			}
			return node.kind
		}
	}
	for _, f := range node.fields {
		if m := b.findCustomTypeMatchInSubtree(f.node, visited); m != "" {
			return m
		}
	}
	if m := b.findCustomTypeMatchInSubtree(node.items, visited); m != "" {
		return m
	}
	if m := b.findCustomTypeMatchInSubtree(node.values, visited); m != "" {
		return m
	}
	for _, br := range node.branches {
		if m := b.findCustomTypeMatchInSubtree(br, visited); m != "" {
			return m
		}
	}
	return ""
}

// Unions may not contain multiple schemas with the same type, except for
// record, fixed, and enum (of which we ensure unique names). Unions also
// cannot contain other immediate unions.
//
// If we see types we do not understand, it is possible they are referencing
// things that are not yet declared. We fixup at the very end.
func (b *builder) buildUnion(parentName string, s *aschema) error {
	var (
		// ONE table for the whole union, handed to the serializer and to the
		// node below so both wires ask the same allocation. finalizeUnionNames
		// refills it in place, so neither holder can go stale.
		tags        = new(unionTags)
		ser         = &serUnion{tags: tags}
		deser       = new(deserUnion)
		missing     = make(map[int]string)
		sawTypes    = make(map[string]bool)
		branchMetas = make([]fieldMeta, len(s.union))
		branchNodes = make([]*schemaNode, len(s.union))
		// Per-branch tag spellings, collected across the loop and turned
		// into the three tag tables in one place afterward — the collision
		// rule needs every branch's exact name, so it cannot be applied
		// branch-by-branch inside the loop.
		unionStd = make([]string, 0, len(s.union))
		unionLog = make([]string, 0, len(s.union))
	)
	// A zero-branch union appends nothing below; the canon tree still
	// needs a non-nil union so the canonical writer emits `[]` (union-ness
	// is discriminated by non-nil throughout the canon walk).
	if len(s.union) == 0 {
		b.canon.union = []aschema{}
	}

	for i, us := range s.union {
		u := b.nest()
		// captureFwdRef converts an unknownPrimitiveError into a
		// (true, name) signal so we can queue a missing-branch fixup
		// for finalize(); any other error is wrapped with the "union"
		// context label. pe.p inside captureFwdRef carries the
		// unresolved name from either the bare-string form (where
		// us.primitive is set) or the wrapped form {"type":"FwdName"}
		// (where us.object.Type is set).
		isFwdRef, fwdName, err := captureFwdRef(u.build(parentName, &us), "union")
		if err != nil {
			return err
		}
		if isFwdRef {
			missing[i] = fwdName
		}
		b.unnest(u)
		branchMetas[i] = u.meta
		branchNodes[i] = u.node

		typ, name, err := us.unionTypeName()
		if err != nil {
			return err
		}
		// Per Avro spec ("Unions"): "Names of named types must be
		// defined exactly once across all the schemas of the union."
		// Key by the resolved fullname when available so an inline
		// definition + a name reference to the same named type collide;
		// primitives still collide on kind. Forward-referenced branches
		// are NOT keyed here: their as-written name is not yet bound
		// (resolveNamedRef may qualify it in-scope or fall back to the
		// null namespace), so keying it would both miss real duplicates
		// ("Inner" vs the inline "n.Inner" it resolves to) and false-
		// reject valid unions (a fwd "Inner" destined for "n.Inner"
		// colliding with a null-namespace "Inner" branch). Every
		// fwd-ref-bearing union is re-checked over the RESOLVED names in
		// finalize via finalizeUnionNames.
		if missing[i] == "" {
			key := typ
			if u.node != nil && u.node.name != "" {
				key = u.node.name
			}
			if sawTypes[key] {
				return fmt.Errorf("duplicate union type %q", truncForError(key))
			}
			sawTypes[key] = true
		}

		b.canon.union = append(b.canon.union, u.canon)
		ser.fns = append(ser.fns, u.ser)
		deser.fns = append(deser.fns, u.deser)

		// Branch names for TaggedUnions wrapping.
		// u.node may be nil for forward-referenced types; use the
		// type name from the schema entry as fallback.
		var bn, ln string
		if u.node != nil {
			bn, ln = unionBranchNames(u.node)
		} else if name != "" {
			bn, ln = name, name
		} else {
			bn, ln = typ, typ
		}
		unionStd = append(unionStd, bn)
		unionLog = append(unionLog, ln)
	}
	fillUnionTagTables(tags, deser, branchNodes, unionStd, unionLog)
	// Populate byKind for type-name dispatch in serUnion.ser /
	// appendAvroJSONUnion / encodeDefault's union case (see
	// unionTypeNameForValue). Primitive kinds only — record/enum/fixed
	// branches go through tagged-union dispatch, and the spec
	// guarantees primitive kinds are unique within a union (so this
	// map is unambiguous by construction).
	//
	// Not rebuilt by finalizeUnionNames: a branch is nil there only while a
	// forward reference is unbound, and a forward reference always names a
	// NAMED type, which none of the kinds below can be.
	for i, branch := range branchNodes {
		if branch == nil {
			continue
		}
		switch branch.kind {
		case "null", "boolean", "int", "long", "float", "double", "string", "bytes":
			if tags.byKind == nil {
				tags.byKind = make(map[string]int, len(branchNodes))
			}
			if _, exists := tags.byKind[branch.kind]; !exists {
				tags.byKind[branch.kind] = i
			}
		}
	}

	switch {
	case len(s.union) == 2 && s.union[0].isNullBranch():
		b.ser = serNullUnion(ser)
		b.deser = deserNullUnion(deser)
		b.meta = b.buildNullUnionMeta(parentName, missing, branchMetas, 1, false)
	case len(s.union) == 2 && s.union[1].isNullBranch():
		b.ser = serNullSecondUnion(ser)
		b.deser = deserNullSecondUnion(deser)
		b.meta = b.buildNullUnionMeta(parentName, missing, branchMetas, 0, true)
	default:
		b.ser = ser.ser
		b.deser = deser.deser
		b.meta = fieldMeta{avroType: "union"}
	}
	if len(missing) > 0 {
		b.missing = append(b.missing, unionMissing{
			ser:        ser,
			deser:      deser,
			branches:   branchNodes,
			missing:    missing,
			parentName: parentName,
		})
	}
	b.node = &schemaNode{
		kind:     "union",
		branches: branchNodes,
		tags:     tags,
		ser:      b.ser,
		deser:    b.deser,
	}
	return nil
}

// unionTags is a union's parse-time name lookup: "which branch does this name
// select?", answered once and asked by BOTH wires.
//
// byName is findUnionBranch's accept-set, built by offering every branch to
// every unionTagTiers tier. byKind routes a value whose Go type names an Avro
// primitive kind, and routes nil to the null branch.
//
// Allocated ONCE per union and refilled IN PLACE, never reassigned:
// finalizeUnionNames rebuilds after forward references bind, and every holder
// must see that rebuild. Reassigning a field would leave whichever holder was
// wired first on the pre-finalize table.
//
// The methods tolerate a nil receiver so a table-less node — hand-assembled in
// a test — takes the scan without every caller repeating the check.
type unionTags struct {
	byName map[string]int // caller-written tag → branch index
	byKind map[string]int // primitive branch kind → branch index; first wins
}

func (t *unionTags) branchByName(name string) (int, bool) {
	if t == nil {
		return 0, false
	}
	i, ok := t.byName[name]
	return i, ok
}

func (t *unionTags) branchByKind(kind string) (int, bool) {
	if t == nil {
		return 0, false
	}
	i, ok := t.byKind[kind]
	return i, ok
}

// fillUnionTagTables builds a union's three tag tables: the two the DECODER
// EMITS (deser.branchNames / deser.logicalNames) and the one both ENCODERS and
// the JSON decoder RESOLVE a caller-written tag through (tags.byName).
//
// The emit tables carry one precedence rule — an exact branch name outranks a
// logical qualifier, the order findUnionBranch resolves in — so a branch never
// emits a tag the decoder would hand to a different branch. The accept table is
// built by asking unionTagTiers instead of restating any of it; see below.
func fillUnionTagTables(tags *unionTags, deser *deserUnion, branches []*schemaNode, standard, logical []string) {
	deser.branchNames = append(deser.branchNames[:0], standard...)
	deser.logicalNames = deser.logicalNames[:0]
	for i, ln := range logical {
		if ln != standard[i] && unionLogicalTagOwnedElsewhere(standard, i, ln) {
			ln = standard[i]
		}
		deser.logicalNames = append(deser.logicalNames, ln)
	}
	// The degrade above is the OPERATIVE guard for the EMIT tables:
	// deser.logicalNames is the tag the BINARY decoder wraps a value in, and
	// nothing else recomputes it.
	//
	// The ACCEPT table below is built by asking unionTagTiers rather than
	// restating it, offering every branch to every tier in order, so this
	// accept-set IS findUnionBranch's by construction. Two resolver rules ride
	// along: across tiers FIRST WRITE WINS, and within a guarded tier a name
	// two branches could claim is registered NOWHERE. That refusal must hold on
	// BOTH wire formats, or the caller gets a value on one and an error on the
	// other.
	//
	// A nil branch node is a forward reference buildUnion has not bound. Its
	// exact name registers from `standard` so the table works meanwhile, and
	// finalizeUnionNames rebuilds over the resolved nodes.
	if tags.byName == nil {
		tags.byName = make(map[string]int, len(standard))
	} else {
		clear(tags.byName)
	}
	// ONE scratch set for the whole walk, not one per tier. The branch count is
	// set by the schema TEXT — a twenty-thousand-branch union is legal Avro a
	// registry or OCF header can carry — so "does another branch claim this
	// name" must answer in constant time per branch. Scanning the branch slice
	// would be a scan inside a loop over that same slice, quadratic in a number
	// the caller picks; counting claims once per tier is linear. The count map
	// is allocated on first use and reused, so an ordinary union pays one
	// allocation and a union with no guarded tier pays none.
	claims := make([]string, len(branches))
	claimed := make([]bool, len(branches))
	var claimCount map[string]int
	for _, tier := range unionTagTiers {
		for i, b := range branches {
			switch {
			case b != nil:
				claims[i], claimed[i] = tierClaim(tier, b)
			case i == 0 || tier.name == unionTagTiers[0].name:
				// A forward reference has no node yet; its as-written name is
				// the exact-name tier's claim until finalizeUnionNames rebuilds.
				claims[i], claimed[i] = "", false
				if tier.name == unionTagTiers[0].name && i < len(standard) {
					claims[i], claimed[i] = standard[i], true
				}
			default:
				claims[i], claimed[i] = "", false
			}
		}
		if tier.guarded {
			if claimCount == nil {
				claimCount = make(map[string]int, len(branches))
			} else {
				clear(claimCount)
			}
			for i := range branches {
				if claimed[i] {
					claimCount[claims[i]]++
				}
			}
		}
		for i := range branches {
			if !claimed[i] {
				continue
			}
			// A name two branches could claim is registered NOWHERE: the
			// resolver refuses it rather than picking one, and the two sides
			// have to refuse the same set.
			if tier.guarded && claimCount[claims[i]] > 1 {
				continue
			}
			if _, taken := tags.byName[claims[i]]; !taken {
				tags.byName[claims[i]] = i
			}
		}
	}
}

// finalizeUnionNames re-derives a union's name-dependent artifacts after
// every forward-referenced branch node is wired. A named reference is
// position-independent, so nothing observable may depend on whether a branch
// was defined before or after it — but buildUnion runs before forward refs
// resolve and captured the UNRESOLVED as-written name in both the
// duplicate-branch key and the branch-name tables. This re-runs both:
//
//   - Duplicate detection (spec, "Unions": no named type twice), keyed by
//     resolved fullname for named branches and kind for unnamed ones, so a
//     short-name forward reference and an inline definition of the same type
//     now collide where their parse-time keys "Inner" and "n.Inner" did not.
//   - branchNames/logicalNames rebuild on the resolved full name, matching the
//     JSON side and making the tables identical to an in-order reference's.
func finalizeUnionNames(tags *unionTags, deser *deserUnion, branches []*schemaNode) error {
	saw := make(map[string]bool, len(branches))
	for _, n := range branches {
		key := n.kind
		if n.name != "" {
			key = n.name
		}
		if saw[key] {
			return fmt.Errorf("duplicate union type %q", truncForError(key))
		}
		saw[key] = true
	}
	std := unionStandardNames(branches)
	log := make([]string, len(branches))
	for i, n := range branches {
		_, log[i] = unionBranchNames(n)
	}
	fillUnionTagTables(tags, deser, branches, std, log)
	return nil
}

// buildNullUnionMeta returns the fieldMeta for the 2-branch null-union
// fast path. nonNullIdx is the index of the non-null branch (1 for
// ["null", T]; 0 for ["T", "null"]). When that branch is a forward
// reference, the inner meta is queued for finalize-time fixup;
// otherwise the inner meta is copied from branchMetas. nullSecond
// distinguishes the two orderings.
func (b *builder) buildNullUnionMeta(parentName string, missing map[int]string, branchMetas []fieldMeta, nonNullIdx int, nullSecond bool) fieldMeta {
	if name, isMissing := missing[nonNullIdx]; isMissing {
		inner := &fieldMeta{}
		b.mfixups = append(b.mfixups, metaFixup{meta: inner, name: name, parentName: parentName})
		return fieldMeta{avroType: "nullunion", nullSecond: nullSecond, inner: inner}
	}
	inner := new(fieldMeta)
	*inner = branchMetas[nonNullIdx]
	return fieldMeta{avroType: "nullunion", nullSecond: nullSecond, inner: inner}
}

func (b *builder) buildComplex(parentName string, s *aschema) error {
	// If this object is a primitive in the shape of a complex, we convert
	// this to a primitive.
	o := s.object

	// Save original logical type before validation clears unknown ones.
	origLogical := o.Logical
	if err := o.validateLogical(); err != nil {
		return err
	}
	// Restore unknown logical types if a registered CustomType matches.
	if o.Logical == "" && origLogical != "" {
		for _, ct := range b.customTypes {
			if ct.LogicalType == origLogical {
				o.Logical = origLogical
				break
			}
		}
	}

	if ser, isPrimitive := serPrimitive[o.Type]; isPrimitive {
		// "bytes" is the only primitive underlying that validateLogical
		// permits for decimal/big-decimal ("fixed" is built on the
		// named-type path below, never here). The o.Type=="bytes" gate
		// matters because the CustomType-resurrection above can restore a
		// dropped "decimal"/"big-decimal" logical onto ANY primitive
		// (validateLogical soft-drops decimal on a wrong underlying type
		// BEFORE its precision-required check, so o.Precision stays nil):
		// without the gate a `{"type":"int","logicalType":"decimal"}` +
		// decimal CustomType would enter this branch and dereference nil
		// o.Precision. A resurrected logical on a non-bytes primitive
		// instead falls through to the plain-primitive path, where the
		// base ser/deser carry the value and the CustomType wraps them.
		if o.Logical == "decimal" && o.Type == "bytes" {
			scale := 0
			if o.Scale != nil {
				scale = *o.Scale
			}
			// Per-direction suppression mirrors the timestamp/uuid path
			// below (line ~1660-1675): built-in encoder is preserved
			// whenever the user didn't provide an Encode callback (per
			// CustomType.Encode docstring "If nil, the built-in logical
			// type encoder is used"); built-in decoder is suppressed
			// whenever ANY matching CustomType exists (per
			// CustomType.Decode docstring "If nil, the built-in logical
			// type handler is bypassed and the base Avro type decoder
			// is used directly"). A single-gate suppression on any match
			// would break encode of *big.Rat with a Decode-only
			// CustomType.
			if b.hasMatchingCustomTypeWithEncode(o.Type, o.Logical) {
				b.ser = ser
			} else {
				b.ser = (&serBytesDecimal{precision: *o.Precision, scale: scale}).ser
			}
			if b.hasMatchingCustomType(o.Type, o.Logical) {
				b.deser = deserPrimitive[o.Type]
			} else {
				b.deser = (&deserBytesDecimal{scale: scale}).deser
			}
			b.canon = aschema{primitive: o.Type}
			b.meta = fieldMeta{avroType: o.Type, logical: o.Logical}
			nd := &schemaNode{
				kind:      o.Type,
				logical:   o.Logical,
				ser:       b.ser,
				deser:     b.deser,
				precision: *o.Precision,
				scale:     scale,
			}
			b.node = nd
			return nil
		}
		if o.Logical == "big-decimal" && o.Type == "bytes" {
			if b.hasMatchingCustomTypeWithEncode(o.Type, o.Logical) {
				b.ser = ser
			} else {
				b.ser = (&serBigDecimal{}).ser
			}
			if b.hasMatchingCustomType(o.Type, o.Logical) {
				b.deser = deserPrimitive[o.Type]
			} else {
				b.deser = (&deserBigDecimal{}).deser
			}
			b.canon = aschema{primitive: o.Type}
			b.meta = fieldMeta{avroType: o.Type, logical: o.Logical}
			nd := &schemaNode{
				kind:    o.Type,
				logical: o.Logical,
				ser:     b.ser,
				deser:   b.deser,
			}
			b.node = nd
			return nil
		}
		b.ser = ser
		b.deser = deserPrimitive[o.Type]
		// Use the logical serializer when the logical is spec-valid for this
		// underlying kind, where it is a strict superset of the base one
		// (accepting time.Time etc. as well as raw values).
		//
		// The kind gate matters because the CustomType resurrection near the top
		// of buildComplex can restore a soft-dropped logical onto a kind it is
		// NOT valid for (uuid on bytes, timestamp-millis on string).
		// logicalSer/logicalDeser key on the logical name alone, so without the
		// gate the binary codec would apply serUUID/serTimestamp* on the wrong
		// kind while the per-kind JSON paths stay raw — diverging binary from
		// JSON and, for the string case, emitting a wire this schema's own
		// decoder cannot read. logicalUnderlyingAcceptsObject is the predicate
		// validateLogical soft-drops with, so encode and decode gate alike.
		//
		// The deser ALSO suppresses on a matching CustomType, so a custom
		// decoder sees the raw Avro-native value. The two gates are independent:
		// a CustomType whose AvroType names a different kind resurrects the
		// logical (resurrection keys on LogicalType only) yet does not match for
		// suppression, so !hasMatchingCustomType alone would apply the
		// wrong-kind deser on binary while the kind-gated JSON path stays raw.
		if logSer := logicalSer(o.Logical); logSer != nil && logicalUnderlyingAcceptsObject(o) {
			b.ser = logSer
		}
		if !b.hasMatchingCustomType(o.Type, o.Logical) && logicalUnderlyingAcceptsObject(o) {
			if logDeser := logicalDeser(o.Logical); logDeser != nil {
				b.deser = logDeser
			}
		}
		b.canon = aschema{primitive: o.Type}
		b.meta = fieldMeta{avroType: o.Type, logical: o.Logical}
		nd := &schemaNode{
			kind:    o.Type,
			logical: o.Logical,
			ser:     b.ser,
			deser:   b.deser,
		}
		if o.Logical == "" && origLogical != "" {
			nd.unknownLogical = origLogical
		}
		// o.Precision/o.Scale are deliberately NOT copied here: the node
		// fields hold validated decimal parameters only (the bytes-decimal
		// branch above and the fixed build's decimal arm). On this path the
		// keys were never consumed — a soft-dropped/resurrected decimal on a
		// wrong carrier, or a stray placement — so their values are
		// unvalidated inert metadata, surfaced via extra→props instead.
		b.node = nd
		return nil
	}

	// Named-type reference wrapped in an object: {"type":"Node"} where
	// "Node" is a record/enum/fixed and no type-defining fields are
	// present. Java's parser accepts this form (see apache/avro
	// TestUnionSelfReference). The bare-string form "Node" is the
	// canonical reference shape; this branch handles the equivalent
	// wrapped form for interop with producers that emit it. Forward
	// references — names not yet declared — return unknownPrimitiveError
	// so the field/union/array/map dispatch can queue a fixup, mirroring
	// the bare-string forward-reference path in buildPrimitive.
	// Any of Fields/Symbols/Items/Values/Size/Name being set means the
	// caller is trying to *define* a new type — fall through and let the
	// regular dispatch handle (or error on) that case.
	if o.Name == "" &&
		len(o.Fields) == 0 && len(o.Symbols) == 0 &&
		o.Items == nil && o.Values == nil && o.Size == nil {
		if found, err := b.tryAssignNamedRef(o.Type, parentName, true); err != nil || found {
			return err
		}
		// Not a recognized base/complex type and not a declared named
		// type — treat as a forward reference. The caller (record-field
		// build, union dispatch, etc.) catches unknownPrimitiveError and
		// queues a fixup keyed on the name in the error.
		switch o.Type {
		case "record", "error", "enum", "fixed", "array", "map":
			// real complex-type-without-required-fields — fall through
			// to the existing switch which will surface the right error.
		default:
			if _, isPrim := serPrimitive[o.Type]; !isPrim {
				b.canon = aschema{primitive: o.Type}
				return &unknownPrimitiveError{o.Type}
			}
		}
	}

	// Preserve original aliases and enum default before canonical stripping.
	origAliases := s.object.Aliases
	origEnumDefault := s.object.Default
	origFieldAliases := make([][]string, len(s.object.Fields))
	for i, f := range s.object.Fields {
		origFieldAliases[i] = f.Aliases
	}

	// Canonical form: per the Avro spec's Parsing Canonical Form STRIP
	// rule, keep only: type, name, fields, symbols, items, values, size.
	// Strip all others (logicalType, precision, scale, doc, aliases, etc.).
	//
	// "error" normalizes to "record" so the canonical form, and every
	// Fingerprint from it, matches Java's SchemaNormalization and fastavro's
	// _to_parsing_canonical_form, both of which emit "record" for both. Without
	// it, error-typed schemas fingerprint differently and break Single Object
	// interop and registry fingerprint indexing.
	//
	// Root().Type, String() and the SchemaNode.Schema round trip still preserve
	// the as-written "error"; only the canonical surface normalizes.
	canonType := o.Type
	if canonType == "error" {
		canonType = "record"
	}
	canonObj := &aobject{
		Name: o.Name,
		Type: canonType,

		Fields:  o.Fields,
		Symbols: o.Symbols,
		Items:   o.Items,
		Values:  o.Values,
		Size:    o.Size,

		Namespace: o.Namespace,
	}
	b.canon = aschema{object: canonObj}

	if isNamedKind(o.Type) {
		if err := b.validFullnameErr(o.Name); err != nil {
			return fmt.Errorf("invalid %s name %q: %w", truncForError(o.Type), truncForError(o.Name), err)
		}
		// The namespace attribute is itself a dot-separated sequence of names
		// (spec §Names) and owes the same grammar. The check above saw only the
		// name attribute, so without this {"name":"R","namespace":"bad ns"}
		// skips validation entirely while the identical inline fullname
		// {"name":"bad ns.R"} is rejected — and since the canonical form
		// inlines the namespace into the fullname, the accepted schema's
		// Canonical() would then fail to re-parse in the same mode. Validating
		// here also routes namespace components through a WithLaxNames
		// validator, honoring its "called for each name component" contract. A
		// dotted name ignores the attribute per the spec, and "" is the
		// explicit null-namespace escape; both are exempt.
		if o.Namespace != nil && *o.Namespace != "" && !strings.Contains(o.Name, ".") {
			if err := b.validFullnameErr(*o.Namespace); err != nil {
				return fmt.Errorf("invalid %s namespace %q: %w", truncForError(o.Type), truncForError(*o.Namespace), err)
			}
		}
		// Aliases are NOT name-validated: the Avro spec (§Aliases) states
		// "any string is accepted as an alias", precisely so evolution can
		// alias a reader's valid name to a writer's illegal/legacy name.
		// fastavro does no alias validation (a "123 !bad" alias parses,
		// observed 1.12.2), and Java stores FIELD aliases as raw strings
		// (Field.addAlias, Schema.java:674-677) — though Java's default
		// parser DOES run its NameValidator over TYPE aliases
		// (parseAliases → NamedSchema.addAlias → new Name,
		// Schema.java:2000-2004/:847, rejecting e.g. digit-start), its
		// own divergence from the spec sentence. twmb follows the spec
		// and fastavro. qualifyAliases (below) still applies namespace
		// qualification and the leading-dot null-namespace escape;
		// resolution matches aliases as plain strings.
		ns := ""
		hasNS := false
		if o.Namespace != nil {
			ns = *o.Namespace
			hasNS = true
		}
		if strings.Contains(o.Name, ".") {
			// Fullname (dot-separated): ignore parent & our own namespace.
			parentName = ""
			hasNS = false
			if short, ok := leadingDotName(o.Name); ok {
				// ".x" is the null-namespace fullname "x" and "."
				// collapses to the bare empty name "" (leadingDotName).
				// Reachable only under a WithLaxNames fn accepting ""
				// (validFullnameErr already rejected the empty component
				// for strict parses, above), so strict acceptance is
				// unchanged. Without this the name registered VERBATIM
				// while child registration prefixed parentName[:dot+1]
				// and reference resolution used namespaceOf — three
				// rules that disagree exactly when the namespace part is
				// empty, so a bare sibling reference inside ".x" could
				// not resolve at all.
				o.Name = short
			}
		}
		if hasNS && ns != "" {
			o.Name = ns + "." + o.Name // have namespace: prefix our name
		} else if hasNS && ns == "" {
			// Explicit empty namespace: clear inherited namespace.
		} else if parentName != "" {
			if dot := strings.LastIndexByte(parentName, '.'); dot >= 0 {
				o.Name = parentName[:dot+1] + o.Name // no namespace: prefix our name with parent namespace if there is one
			}
		}
		// Per the Avro spec (Names): a primitive type name has no
		// namespace and may not name a record/enum/fixed. o.Name is now
		// the resolved fullname; serPrimitive's keys are exactly the 8
		// bare primitive names, so this matches only a null-namespace
		// bare primitive name (a namespaced "a.int" has fullname "a.int",
		// not a key). Matches Java's NamedSchema "may not be named after
		// primitives".
		if _, isPrim := serPrimitive[o.Name]; isPrim {
			return fmt.Errorf("%s may not be named after the primitive type %q", truncForError(o.Type), truncForError(o.Name))
		}
		o.Namespace = nil      // canonical form omits namespace
		canonObj.Name = o.Name // use fully-qualified name
		canonObj.Namespace = nil
		if _, exists := b.named[o.Name]; exists {
			if !(b.cachedNames[o.Name] && b.allowReRegister) {
				return fmt.Errorf("duplicate named type %q", truncForError(o.Name))
			}
			// Inherited name re-registered by a custom (re-)parse — allowed so
			// it gets fresh CustomType wiring.
		}
	} else {
		// A stray "namespace" on an unnamed kind is inert metadata: never
		// consumed (only the named branch above reads o.Namespace), never
		// scoping (children resolve in the enclosing scope on every path —
		// the walkers' nodeChildScope/nsForChildren are kind-keyed), and
		// surfaced as-written by the metadata tree, matching the primitive
		// type-object posture and both references (Java SCHEMA_RESERVED
		// ignores it on every schema object; fastavro accepts it and scopes
		// a named type under such an array in the ENCLOSING scope,
		// executed). A stray "name" on a CONTAINER kind still rejects: the
		// metadata walkers deliberately scope children by any non-empty
		// SchemaNode.Name (nsForChildren's hand-built posture), so a parsed
		// stray name on a kind with child positions would make Root() scope
		// named descendants differently than this parser — the same
		// walker-parity rationale as the structural-key exclusivity
		// rejects. Primitive objects keep accepting a stray name: they have
		// no child positions for that arm to act on.
		if o.Name != "" {
			return errors.New("only record, enum, and fixed can have a name")
		}
		// The inert attribute never reaches the canonical form (PCF has no
		// namespace key for unnamed kinds; fastavro's
		// to_parsing_canonical_form strips it, executed) — mirror the
		// primitive type-object collapse, which drops it the same way.
		o.Namespace = nil
		canonObj.Namespace = nil
	}

	switch o.Type {
	default:
		return fmt.Errorf("unknown complex type %q", truncForError(o.Type))

	case "record", "error":
		if len(o.Symbols) > 0 ||
			o.Items != nil ||
			o.Values != nil ||
			o.Size != nil {
			return errors.New("invalid record has schema for other types")
		}
		// The fields attribute is required (Java: "Record has no fields"),
		// while an EMPTY array is the legal empty record. Same
		// missing-vs-empty discrimination as enum symbols: the parser
		// materializes "fields":[] as a non-nil empty slice and leaves
		// the attribute's absence as nil.
		if o.Fields == nil {
			return errors.New("record is missing fields")
		}

		// Create record ser/deser and register early so
		// self-referencing fields (e.g. array items, map values)
		// can resolve the type by name during field building.
		sr := &serRecord{}
		dr := &deserRecord{}
		b.ser = sr.ser
		b.deser = dr.deser
		b.meta = fieldMeta{avroType: "record", serRecord: sr, deserRecord: dr}

		nd := &schemaNode{
			kind:        "record",
			name:        o.Name,
			logical:     o.Logical,
			aliases:     qualifyAliases(origAliases, o.Name),
			bareAliases: bareAliasShorts(origAliases),
			ser:         b.ser,
			deser:       b.deser,
			serRecord:   sr,
			deserRecord: dr,
		}
		b.registerNamed(o.Name, &namedType{node: nd})
		b.node = nd

		// Mark this record as under construction. A field default whose type
		// subtree references this record (a self- or mutual-recursive
		// reference) must defer its binary default-encode to finalize rather
		// than encode inline now: encodeDefault recurses into the referenced
		// record's fields, and nd.fields holds only the fields declared
		// before the current one until the loop below completes — encoding
		// inline against it emits truncated, non-decodable default bytes.
		// nodeAwaitsForwardRef consults this set so an in-construction record
		// is treated like a not-yet-wired forward-ref child. Cleared when the
		// build returns and nd is whole. Lazily created (the public entry
		// points seed it, but an internally-constructed builder may not) and
		// shared through nest, so a nested record built while this one is open
		// sees the same in-construction set.
		if b.building == nil {
			b.building = make(map[*schemaNode]struct{})
		}
		b.building[nd] = struct{}{}
		defer delete(b.building, nd)

		var names []string
		seenFields := make(map[string]bool, len(o.Fields))
		for i, of := range o.Fields {
			if err := b.validNameErr(of.Name); err != nil {
				return fmt.Errorf("invalid field name %q: %w", truncForError(of.Name), err)
			}
			// Field aliases are NOT name-validated — per the Avro spec any
			// string is accepted as an alias (so a reader can alias a writer's
			// illegal/legacy field name); matched as-is against writer field
			// names during resolution. Matches Java/fastavro.
			if seenFields[of.Name] {
				return fmt.Errorf("duplicate record field name %q", truncForError(of.Name))
			}
			seenFields[of.Name] = true
			// Written-ness, not non-emptiness, admits the value: "order":"" is
			// a written order and not one of the spec's three, so it fails
			// like any other non-spec spelling. The comparison is EXACT-case;
			// Apache Avro upper-cases before its lookup, but reserved
			// attribute VALUES match by exact spelling here.
			if of.orderSet && of.Order != "ascending" && of.Order != "descending" && of.Order != "ignore" {
				return fmt.Errorf("invalid field order %q for field %q", truncForError(of.Order), truncForError(of.Name))
			}
			bf := b.nest()
			// captureFwdRef converts unknownPrimitiveError from a nested
			// build into an "isFwdRef" signal so the caller can queue a
			// fixup in finalize(); other errors are wrapped with the
			// "record field" context label. Shared with array/map sites
			// so all three contexts handle fwd-refs uniformly.
			isFwdRef, fwdRefName, err := captureFwdRef(bf.build(o.Name, of.Type), "record field")
			if err != nil {
				return err
			}
			b.unnest(bf)
			if isFwdRef {
				bf.canon = aschema{primitive: fwdRefName}
			}
			o.Fields[i] = afield{
				Name:       of.Name,
				Type:       &bf.canon,
				hasDefault: len(of.Default) > 0,
			}
			meta := new(fieldMeta)
			*meta = bf.meta
			fieldIdx := len(sr.fields)
			sr.fields = append(sr.fields, serRecordField{
				name:    of.Name,
				nameVal: reflect.ValueOf(of.Name),
				fn:      bf.ser,
				meta:    meta,
			})
			drf := deserRecordField{
				name:    of.Name,
				nameVal: reflect.ValueOf(of.Name),
				fn:      bf.deser,
				fnIface: ifaceFnForPrimitive(meta),
				meta:    meta,
			}
			fn := fieldNode{
				name:    of.Name,
				nameVal: reflect.ValueOf(of.Name),
				aliases: origFieldAliases[i],
				node:    bf.node,
			}
			if isFwdRef {
				fix := recordFieldFixup{
					nd:         nd,
					idx:        fieldIdx,
					name:       fwdRefName,
					parentName: o.Name, // fields build under the record's fullname
				}
				if len(of.Default) > 0 {
					fix.defaultVal = unmarshalDefault(of.Default)
					fix.hasDefault = true
				}
				b.fieldFixups = append(b.fieldFixups, fix)
			}
			if len(of.Default) > 0 {
				if isFwdRef {
					// Forward-ref: signal hasDefault so the dispatch
					// knows a default exists. finalize() runs the full
					// pipeline against the resolved schemaNode and
					// overwrites defaultVal there.
					drf.hasDefault = true
					fn.hasDefault = true
				} else if b.nodeAwaitsForwardRef(bf.node) {
					// The field's outer type resolved, but a descendant
					// encodeDefault traverses is not whole — see
					// nodeAwaitsForwardRef. Defer the resolve+encode to
					// finalize, after the fixups wire the descendants and every
					// in-construction record completes. Signal hasDefault so
					// dispatch knows a default exists; the deferred pass fills
					// defaultVal/defaultBytes.
					drf.hasDefault = true
					fn.hasDefault = true
					b.defaultFixups = append(b.defaultFixups, defaultFixup{
						nd:         nd,
						idx:        fieldIdx,
						node:       bf.node,
						defaultVal: unmarshalDefault(of.Default),
					})
				} else {
					defaultVal := unmarshalDefault(of.Default)
					defaultVal = coerceDefault(defaultVal, bf.node)
					if err := applyResolvedDefault(
						defaultVal, bf.node, of.Name,
						&drf, &fn, &sr.fields[fieldIdx],
					); err != nil {
						return err
					}
				}
			} else if bf.canon.isNullableUnion() {
				// Per the Avro spec, a union whose first branch is "null"
				// implicitly defaults to null when no explicit default is given.
				// fn.defaultVal stays nil — the JSON encoder treats a nil
				// default as the null encoding.
				drf.hasDefault = true
				fn.hasDefault = true
				sr.fields[fieldIdx].defaultBytes = []byte{0} // varint 0 = null branch
				sr.fields[fieldIdx].hasDefault = true
			}
			dr.fields = append(dr.fields, drf)
			nd.fields = append(nd.fields, fn)
			names = append(names, of.Name)
		}
		sr.names = names
		dr.names = names
		// JSON DecodeJSON uses fieldIdx to route record-field keys to their
		// schema slot. Register every alias→idx mapping in addition to the
		// canonical name so JSON producers that emit using an alias name
		// route to the right field. The binary path's resolve.go does the
		// equivalent alias-aware lookup via readerFieldLookup.
		// Per Avro spec ("Aliases are alternative names, and thus subject
		// to the same uniqueness constraints as names"), a field name AND
		// alias share one namespace within a record. Reject symmetrically:
		// either a later name shadowing a prior alias, or a later alias
		// shadowing a prior name/alias, breaks uniqueness. Checking only the
		// alias side lets [{name:"a",aliases:["x"]},{name:"x"}] parse and then
		// route differently from Java's applyAliases, which rewrites a writer's
		// "x" to "a" before the literal-named field can claim it.
		nd.fieldIdx = make(map[string]int, len(nd.fields))
		for i, f := range nd.fields {
			if _, exists := nd.fieldIdx[f.name]; exists {
				return fmt.Errorf("record field name %q collides with another field name or alias", truncForError(f.name))
			}
			nd.fieldIdx[f.name] = i
			for _, a := range f.aliases {
				if _, exists := nd.fieldIdx[a]; exists {
					return fmt.Errorf("record field alias %q collides with another field name or alias", truncForError(a))
				}
				nd.fieldIdx[a] = i
			}
		}
	case "enum":
		if len(o.Fields) > 0 ||
			o.Items != nil ||
			o.Values != nil ||
			o.Size != nil {
			return errors.New("invalid enum has schema for other types")
		}

		// The symbols attribute is required (Java: "Enum has no symbols"), but
		// an EMPTY array is legal — the spec asks only for "a JSON array,
		// listing symbols", and Java, fastavro and avro-rs all accept zero.
		// Such an enum has no valid values, so every encode/decode errors, but
		// the schema parses, which matters for passing through foreign schemas
		// carrying a degenerate enum the data never uses.
		if o.Symbols == nil {
			return errors.New("enum is missing symbols")
		}
		seenSymbols := make(map[string]bool, len(o.Symbols))
		for _, e := range o.Symbols {
			if err := b.validNameErr(e); err != nil {
				return fmt.Errorf("invalid enum symbol %q: %w", truncForError(e), err)
			}
			if seenSymbols[e] {
				return fmt.Errorf("duplicate enum symbol %q", truncForError(e))
			}
			seenSymbols[e] = true
		}
		symbolIdx := enumSymbolIndex(o.Symbols)
		b.ser = newSerEnum(o.Symbols, symbolIdx).ser
		b.deser = (&deserEnum{symbols: o.Symbols}).deser
		b.meta = fieldMeta{avroType: "enum"}

		nd := &schemaNode{
			kind:        "enum",
			name:        o.Name,
			logical:     o.Logical,
			aliases:     qualifyAliases(origAliases, o.Name),
			bareAliases: bareAliasShorts(origAliases),
			symbols:     o.Symbols,
			symbolIdx:   symbolIdx,
			ser:         b.ser,
			deser:       b.deser,
		}
		if len(origEnumDefault) > 0 {
			// The default must be a JSON STRING token naming a symbol,
			// decided by token type BEFORE the membership check: on a
			// non-string body json.Unmarshal leaves the zero value ""
			// (for an explicit null it is even a no-error no-op), and ""
			// can be a legitimate MEMBER under a WithLaxNames validator
			// that accepts empty name components — membership alone would
			// silently bind such garbage to the "" symbol and schema
			// evolution would fill it. fastavro rejects every non-member
			// (hence every non-string) enum default at parse; Java binds
			// NO default for a non-text token (Schema.java:1921-1925,
			// textValue() → null skips EnumSchema's containment check) —
			// neither reference ever binds one.
			tok := bytes.TrimSpace(origEnumDefault)
			var defStr string
			if len(tok) == 0 || tok[0] != '"' || json.Unmarshal(tok, &defStr) != nil {
				return fmt.Errorf("enum default %s is not a string", truncForError(string(tok)))
			}
			if !seenSymbols[defStr] {
				return fmt.Errorf("enum default %q is not a member of symbols", truncForError(defStr))
			}
			nd.enumDef = defStr
			nd.hasEnumDef = true
		}
		b.registerNamed(o.Name, &namedType{node: nd})
		b.node = nd

	case "array":
		if len(o.Fields) > 0 ||
			len(o.Symbols) > 0 ||
			o.Values != nil ||
			o.Size != nil {
			return errors.New("invalid array has schema for other types")
		}
		if o.Items == nil {
			return errors.New("array is missing items schema")
		}
		af := b.nest()
		isFwdRef, fwdRefName, err := captureFwdRef(af.build(parentName, o.Items), "array")
		if err != nil {
			return err
		}
		b.unnest(af)
		if isFwdRef {
			af.canon = aschema{primitive: fwdRefName}
		}
		o.Items = &af.canon
		// canonObj captured o.Items by value before this recursion ran, so
		// it still points at the as-parsed (possibly {"type":"X"}-wrapped or
		// attribute-bearing) items schema. Repoint it at the canonicalized
		// child so the Parsing Canonical Form's [PRIMITIVES] and [STRIP]
		// rules apply inside array items, matching Java's
		// SchemaNormalization.build (which recurses into getElementType) and
		// every other top-level/field/branch site that already uses the
		// child's canon. Record fields stay correct via the o.Fields slice
		// alias; only the Items/Values pointer fields need the explicit sync.
		canonObj.Items = &af.canon
		sa := &serArray{serItem: af.ser}
		// One computation feeding both wire-side slots. They are the same
		// question asked by the safe and the unsafe array reader, and asking
		// it separately is how they came to hold different answers: only
		// da's was patched by the forward-ref fixup below.
		itemMin := b.minBytes.minBytesOf(af.node)
		da := &deserArray{deserItem: af.deser, minItemBytes: itemMin}
		// Specialized array ser/deser fast paths bypass the inner
		// schema's wrapped ser/deser functions. They are correct only
		// when no per-element conversion is needed: no custom type,
		// no logical type, AND no forward reference (the inner ser/
		// deser aren't wired until finalize() resolves the fwd-ref,
		// so the fast-path closure would capture nil fns at build
		// time).
		if isFwdRef || af.meta.hasCustomType || af.meta.logical != "" {
			b.ser = sa.ser
		} else if info, ok := primFast[af.canon.primitive]; ok {
			b.ser = info.serArrayFn(sa)
			da.fastLoop = info.deserArrayLoop
			da.fastElemKind = info.elemKind
			da.fastIfaceLoop = info.deserArrayIfaceLoop
			da.nativeLoop = info.deserArrayNative
		} else {
			b.ser = sa.ser
		}
		b.deser = da.deser
		inner := new(fieldMeta)
		*inner = af.meta
		inner.minBytes = itemMin
		b.meta = fieldMeta{avroType: "array", inner: inner}
		arrayNode := &schemaNode{
			kind:  "array",
			items: af.node,
			ser:   b.ser,
			deser: b.deser,
		}
		b.node = arrayNode
		if isFwdRef {
			// fwd-ref's resolved node is wired in finalize().
			// Capture pointers to all four wire-side slots that
			// depend on the resolved type so the fixup can patch
			// them once b.named[fwdRefName] becomes available.
			b.containerFixups = append(b.containerFixups, containerFixup{
				serItem:   &sa.serItem,
				deserItem: &da.deserItem,
				// Both slots: da backs the safe array path, inner.minBytes
				// the unsafe one. Patching one left the other holding the
				// build-time answer, computed while this child was still an
				// unwired forward reference.
				setMinBytes: func(n int) { da.minItemBytes = n; inner.minBytes = n },
				nodeChild:   &arrayNode.items,
				name:        fwdRefName,
				parentName:  parentName,
				ctxLabel:    "array",
			})
		}

	case "map":
		if len(o.Fields) > 0 ||
			len(o.Symbols) > 0 ||
			o.Items != nil ||
			o.Size != nil {
			return errors.New("invalid map has schema for other types")
		}
		if o.Values == nil {
			return errors.New("map is missing values schema")
		}
		mf := b.nest()
		isFwdRef, fwdRefName, err := captureFwdRef(mf.build(parentName, o.Values), "map")
		if err != nil {
			return err
		}
		b.unnest(mf)
		if isFwdRef {
			mf.canon = aschema{primitive: fwdRefName}
		}
		o.Values = &mf.canon
		// See the array case above: canonObj.Values still points at the
		// as-parsed values schema, so repoint it at the canonicalized child
		// or the canonical form (and fingerprint) diverges for any
		// map-of-wrapped-or-attribute-bearing-value schema.
		canonObj.Values = &mf.canon
		sm := &serMap{serItem: mf.ser}
		// minEntryBytes = 1 (empty-key length varint) + values' minimum
		// wire bytes. Matches deserArray.minItemBytes in spirit; bounds
		// block-count against remaining-buffer to prevent memory
		// amplification on hostile input.
		dm := &deserMap{deserItem: mf.deser, minEntryBytes: mapEntryMinBytes(b.minBytes.minBytesOf(mf.node))}
		// Same gate as the array case above: skip specialization when
		// values have a custom type, a logical type, OR a forward
		// reference (the fast-path closure can't capture an unresolved
		// inner ser/deser).
		if isFwdRef || mf.meta.hasCustomType || mf.meta.logical != "" {
			b.ser = sm.ser
		} else if info, ok := primFast[mf.canon.primitive]; ok {
			b.ser = info.serMapFn(sm)
			dm.fastBlock = info.deserMapBlock
			dm.fastElemKind = info.elemKind
			dm.fastIfaceVal = info.deserMapIfaceVal
			dm.nativeBlock = info.deserMapNative
		} else {
			b.ser = sm.ser
		}
		b.deser = dm.deser
		b.meta = fieldMeta{avroType: "map"}
		mapNode := &schemaNode{
			kind:   "map",
			values: mf.node,
			ser:    b.ser,
			deser:  b.deser,
		}
		b.node = mapNode
		if isFwdRef {
			b.containerFixups = append(b.containerFixups, containerFixup{
				serItem:     &sm.serItem,
				deserItem:   &dm.deserItem,
				setMinBytes: func(n int) { dm.minEntryBytes = mapEntryMinBytes(n) },
				nodeChild:   &mapNode.values,
				name:        fwdRefName,
				parentName:  parentName,
				ctxLabel:    "map",
			})
		}

	case "fixed":
		if len(o.Fields) > 0 ||
			len(o.Symbols) > 0 ||
			o.Items != nil ||
			o.Values != nil {
			return errors.New("invalid fixed has schema for other types")
		}
		if o.Size == nil {
			return errors.New("fixed is missing size")
		}
		size := int(*o.Size)
		// Size 0 is legal: the spec requires only "an integer", and Java,
		// fastavro, and avro-rs all accept it — every value of a size-0
		// fixed is the empty byte string. We reject only negatives. The upper
		// bound is intentionally unbounded at parse, matching fastavro and
		// avro-rs (which defer the size to value/wire time): a size beyond the
		// actual datum simply fails at encode/decode where it is value- and
		// buffer-bounded. (Java is stricter — FixedSchema's ctor caps size at
		// Integer.MAX_VALUE-8 via SystemLimitException.checkMaxBytesLength — but
		// we follow the lenient majority. No parse-time path may allocate
		// proportional to this size; see maxFixedLogicalLen in json_decode.go.)
		if size < 0 {
			return fmt.Errorf("invalid fixed size %v", size)
		}
		// Per-direction suppression: built-in encoder preserved when the
		// user didn't provide Encode (CustomType.Encode docstring: "If
		// nil, the built-in logical type encoder is used"); built-in
		// decoder suppressed whenever ANY matching CustomType exists
		// (CustomType.Decode docstring: "If nil, the built-in logical
		// type handler is bypassed and the base Avro type decoder is
		// used directly"). A single-gate suppression on any match would
		// route a Decode-only CustomType for fixed.decimal /
		// fixed.duration / fixed.uuid onto raw serSize which can't
		// accept *big.Rat / avro.Duration as input.
		hasEnc := b.hasMatchingCustomTypeWithEncode("fixed", s.object.Logical)
		hasAny := b.hasMatchingCustomType("fixed", s.object.Logical)
		switch s.object.Logical {
		case "duration":
			// serDuration always emits 12 bytes, so it is only correct for a
			// size-12 fixed. validateLogical soft-drops a duration on a wrong
			// size, but the CustomType resurrection near the top of buildComplex
			// can restore it; without the size gate serDuration would write 12
			// bytes into a size != 12 fixed — a wire this schema's own
			// deserFixed{size} reader (and the JSON arm) cannot read.
			// logicalUnderlyingAccept is the same size predicate validateLogical
			// uses to soft-drop, so a resurrected wrong-size logical keeps the
			// raw serSize, matching the suppressed decoder.
			if hasEnc || !logicalUnderlyingAccept["duration"](o) {
				b.ser = (&serSize{size}).ser
			} else {
				b.ser = serDuration
			}
			// deserDuration always reads 12 bytes, so it is only correct for a
			// size-12 fixed. Mirror the ser gate: a wrong-size duration (which
			// validateLogical soft-drops and a CustomType can resurrect) keeps
			// the raw deserFixed{size}, matching the plain fixed and the kind/
			// size-checked JSON decode. hasAny suppresses for a matching custom;
			// !logicalUnderlyingAccept covers a resurrection whose custom AvroType
			// names a different kind (so it does NOT match for suppression).
			if hasAny || !logicalUnderlyingAccept["duration"](o) {
				b.deser = (&deserFixed{size}).deser
			} else {
				b.deser = deserDuration
			}
		case "decimal":
			scale := 0
			if o.Scale != nil {
				scale = *o.Scale
			}
			if hasEnc {
				b.ser = (&serSize{size}).ser
			} else {
				b.ser = (&serFixedDecimal{size: size, precision: *o.Precision, scale: scale}).ser
			}
			if hasAny {
				b.deser = (&deserFixed{size}).deser
			} else {
				b.deser = (&deserFixedDecimal{size: size, scale: scale}).deser
			}
		case "uuid":
			// serFixedUUIDReflect always emits 16 bytes, so it is only correct for
			// a size-16 fixed. validateLogical soft-drops a uuid on a wrong size,
			// but the CustomType resurrection can restore it; without the size
			// gate serFixedUUIDReflect would write 16 bytes into a size != 16
			// fixed — a wire this schema's own deserFixed{size} reader (and the
			// JSON arm) cannot read. logicalUnderlyingAccept is the same size
			// predicate validateLogical uses to soft-drop.
			if hasEnc || !logicalUnderlyingAccept["uuid"](o) {
				b.ser = (&serSize{size}).ser
			} else {
				b.ser = serFixedUUIDReflect
			}
			// deserFixedUUIDReflect always reads 16 bytes; mirror the ser gate so a
			// wrong-size resurrected uuid keeps the raw deserFixed{size} (see the
			// duration case for the hasAny / !logicalUnderlyingAccept split).
			if hasAny || !logicalUnderlyingAccept["uuid"](o) {
				b.deser = (&deserFixed{size}).deser
			} else {
				b.deser = deserFixedUUIDReflect
			}
		default:
			b.ser = (&serSize{size}).ser
			b.deser = (&deserFixed{size}).deser
		}
		b.meta = fieldMeta{avroType: "fixed", logical: s.object.Logical}
		nd := &schemaNode{
			kind:        "fixed",
			name:        o.Name,
			aliases:     qualifyAliases(origAliases, o.Name),
			bareAliases: bareAliasShorts(origAliases),
			logical:     s.object.Logical,
			size:        size,
			ser:         b.ser,
			deser:       b.deser,
		}
		if s.object.Logical == "decimal" && s.object.Precision != nil {
			nd.precision = *s.object.Precision
			if s.object.Scale != nil {
				nd.scale = *s.object.Scale
			}
		}
		b.node = nd
		b.registerNamed(o.Name, &namedType{node: nd})
	}
	return nil
}

// bareAliasShorts collects the aliases declared WITHOUT any dot, as
// written. A dotted alias (including the ".Name" null-namespace escape) is
// an explicit fullname spelling and matches only exactly, via the
// qualifyAliases output; an alias declared bare ALSO short-name-matches a
// writer type in any namespace — fastavro's raw-string tier
// (match_schemas: `w_unqual_name in r_aliases`), the permissive side of
// the reference behaviors (Java's applyAliases map is fullname-keyed and
// has no short tier). Distinguishing bare from qualified requires the
// DECLARED spelling: reconstructing it from the qualified form would
// over-widen an explicitly-written same-namespace alias ("a.Old" declared
// on a type in namespace a is indistinguishable from bare "Old" after
// qualification).
func bareAliasShorts(aliases []string) []string {
	var shorts []string
	for _, a := range aliases {
		if !strings.Contains(a, ".") {
			shorts = append(shorts, a)
		}
	}
	return shorts
}

// qualifyAliases fully qualifies alias names using the parent name's namespace.
func qualifyAliases(aliases []string, fullname string) []string {
	if len(aliases) == 0 {
		return nil
	}
	ns := ""
	if dot := strings.LastIndexByte(fullname, '.'); dot >= 0 {
		ns = fullname[:dot+1]
	}
	out := make([]string, len(aliases))
	for i, a := range aliases {
		switch {
		case strings.ContainsRune(a, '.'):
			// Dotted aliases follow the names' dot rule, via the SAME
			// helper (leadingDotName): a single leading dot with a dotless
			// remainder is the explicit null-namespace escape (".x" is the
			// fullname "x", "." the empty name), never qualified into the
			// type's own namespace; any other dotted spelling is a fullname
			// VERBATIM. Java's Name constructor nulls the space only when
			// it is EMPTY (Schema.java ~1455: lastDot split, then
			// `if ("".equals(space)) space = null`), so ".a.b" keeps its
			// non-empty space ".a" and denotes ".a.b" as written — a name
			// only a lax-parsed writer can carry. Stripping any leading dot
			// here made alias ".a.b" match writer "a.b", a match neither
			// Java (space kept) nor fastavro (raw-string comparison) makes.
			short, _ := leadingDotName(a)
			out[i] = short
		default:
			out[i] = ns + a
		}
	}
	return out
}

// logicalUnderlyingAccept maps known logical types to the predicate
// that decides whether the carrier's Avro type is permitted. Mismatches
// soft-drop the logical (returning the bare underlying type) per spec
// and Java/fastavro/hamba parity — see the soft-drop comment in
// validateLogical for the rationale.
//
// "decimal" is handled inline in validateLogical because its precision/
// scale validation is too involved to fit a one-line predicate.
var logicalUnderlyingAccept = map[string]func(o *aobject) bool{
	"uuid": func(o *aobject) bool {
		return o.Type == "string" || (o.Type == "fixed" && o.Size != nil && int(*o.Size) == 16)
	},
	"date":                   func(o *aobject) bool { return o.Type == "int" },
	"time-millis":            func(o *aobject) bool { return o.Type == "int" },
	"time-micros":            func(o *aobject) bool { return o.Type == "long" },
	"timestamp-millis":       func(o *aobject) bool { return o.Type == "long" },
	"timestamp-micros":       func(o *aobject) bool { return o.Type == "long" },
	"timestamp-nanos":        func(o *aobject) bool { return o.Type == "long" },
	"local-timestamp-millis": func(o *aobject) bool { return o.Type == "long" },
	"local-timestamp-micros": func(o *aobject) bool { return o.Type == "long" },
	"local-timestamp-nanos":  func(o *aobject) bool { return o.Type == "long" },
	"big-decimal":            func(o *aobject) bool { return o.Type == "bytes" },
	// Duration on non-fixed, or fixed with size != 12, soft-drops.
	// Java's Duration.validate at LogicalTypes.java:323-327 throws
	// IllegalArgumentException for `type != FIXED || size != 12`;
	// fromSchemaIgnoreInvalid catches and drops. hamba's
	// parseFixedLogicalType at schema_parse.go:517 only matches
	// `ltyp == Duration && size == 12` and drops everything else.
	"duration": func(o *aobject) bool {
		return o.Type == "fixed" && o.Size != nil && int(*o.Size) == 12
	},
}

// logicalUnderlyingAcceptsObject reports whether o.Logical is spec-valid on
// o's Avro underlying (type + size) — the single predicate the primitive
// build's logical ser/deser selection gates on, identical for both directions.
// Returns false for "decimal"/"big-decimal" (their underlying validity is
// handled inline by validateLogical / the dedicated bytes-decimal build, not
// this table) and for any logical with no specialized name-keyed codec, which
// is correct: logicalSer/logicalDeser are nil there, so the gate's result is
// moot. A CustomType-resurrected wrong-kind logical returns false here, so the
// raw base ser/deser are kept — matching validateLogical's soft-drop.
func logicalUnderlyingAcceptsObject(o *aobject) bool {
	accept := logicalUnderlyingAccept[o.Logical]
	return accept != nil && accept(o)
}

func (o *aobject) validateLogical() error {
	switch o.Logical {
	case "":
		// No logical type. Stray precision/scale are inert metadata —
		// see the note below the switch.

	case "decimal":
		// Wrong underlying type is the one fall-back-on-mismatch case
		// the spec implies: an unknown logical on the wrong primitive should
		// not block the parse. Precision/scale constraints are explicit Avro
		// 1.12 rules, so violating them is malformed and hard-rejects here.
		// fastavro raises the same way (its truthiness guards skip 0-or-missing
		// values, observed 1.12.2). Java's LogicalTypes.Decimal.validate throws
		// too, but fromSchemaIgnoreInvalid catches it at parse and soft-drops
		// to bare bytes/fixed. Rejecting beats that: a producer-declared
		// decimal quietly becoming plain bytes is a silent interop divergence.
		if o.Type != "bytes" && o.Type != "fixed" {
			o.Logical = ""
			return nil
		}
		if o.Precision == nil {
			return fmt.Errorf("decimal logical type requires precision")
		}
		if *o.Precision <= 0 {
			return fmt.Errorf("decimal precision %d must be positive", *o.Precision)
		}
		scale := 0
		if o.Scale != nil {
			scale = *o.Scale
		}
		if scale < 0 {
			return fmt.Errorf("decimal scale %d must not be negative", scale)
		}
		if scale > *o.Precision {
			return fmt.Errorf("decimal scale %d exceeds precision %d", scale, *o.Precision)
		}
		// DoS bound: precision/scale drive 10^scale allocations in
		// bytesToRat / ratToUnscaled at every decode/encode.
		if *o.Precision > decimalScaleLimit {
			return fmt.Errorf("decimal precision %d exceeds %d limit", *o.Precision, decimalScaleLimit)
		}
		if scale > decimalScaleLimit {
			return fmt.Errorf("decimal scale %d exceeds %d limit", scale, decimalScaleLimit)
		}
		if o.Type == "fixed" && o.Size != nil {
			maxDigits := maxDecimalDigits(int(*o.Size))
			if *o.Precision > maxDigits {
				return fmt.Errorf("decimal precision %d exceeds fixed(%d) capacity %d", *o.Precision, *o.Size, maxDigits)
			}
		}
		return nil

	// Wrong-underlying-type soft-drop, mirroring the decimal arm and the spec:
	// "If a logical type is invalid, …then implementations should ignore the
	// logical type and use the underlying Avro type." Java catches the
	// validate() throw in fromSchemaIgnoreInvalid and drops silently; fastavro
	// returns None for an unknown (rt, lt) and falls through to bare; hamba
	// returns nil for any combo outside its switch. Three impls and the spec
	// agree, so hard-rejecting would break interop with producers emitting
	// evolution or legacy combos.
	default:
		if accept, known := logicalUnderlyingAccept[o.Logical]; known {
			if !accept(o) {
				o.Logical = ""
			}
		} else {
			// Per the Avro spec, unknown logical types are ignored and the
			// underlying type is used as-is.
			o.Logical = ""
		}
	}

	// Leftover precision/scale — any placement but the decimal arm above, which
	// consumes and validates them — are inert metadata, NOT a parse error. The
	// spec permits undefined attributes as metadata, Java never consults
	// precision without a logicalType, and fastavro accepts every such
	// placement (executed 1.12.2). They surface as custom properties and no
	// wire codec reads them. Rejecting here would also make the package
	// disagree with itself: the same stray keys parse when an unknown logical
	// or wrong-carrier decimal soft-drops above, and the FIELD level always
	// keeps them as inert props.
	return nil
}

// maxDecimalDigits returns the maximum number of decimal digits that fit in
// a two's-complement signed integer of the given byte size:
// floor(log10(2^(8*size-1) - 1)).
func maxDecimalDigits(size int) int {
	if size <= 0 {
		return 0
	}
	// Saturate before the bit multiply. A fixed size can exceed 2^60 on a
	// 64-bit build (this package accepts sizes Java's int32 cannot), where
	// 8*size-1 wraps the platform int negative and the capacity comes back
	// negative, falsely rejecting a valid precision. The ceiling is
	// maxSchemaMagnitude — shared with every other consumer of a
	// schema-declared magnitude and chosen against the multiply below — rather
	// than a second ceiling reasoned out here. The verdict is unchanged either
	// way: validateLogical rejects a precision above decimalScaleLimit before
	// calling in, and the saturated capacity is far larger than that.
	size = saturateSchemaMagnitude(size)
	bits := magnitudeWidestMultiplier*size - 1 // sign bit excluded
	// log10(2^bits - 1) ≈ bits * log10(2)
	return int(math.Floor(float64(bits) * math.Log10(2)))
}

// logicalSer / logicalDeser look up the time-aware encoder / decoder
// for a logical type, or return nil if the logical has no specialized
// codec. Both encode and decode tables in one place so a new logical
// only needs to be wired in once.
var (
	logicalSers = map[string]serfn{
		"timestamp-millis":       serTimestampMillis,
		"timestamp-micros":       serTimestampMicros,
		"timestamp-nanos":        serTimestampNanos,
		"local-timestamp-millis": serLocalTimestampMillis,
		"local-timestamp-micros": serLocalTimestampMicros,
		"local-timestamp-nanos":  serLocalTimestampNanos,
		"date":                   serDate,
		"time-millis":            serTimeMillis,
		"time-micros":            serTimeMicros,
		"uuid":                   serUUID,
	}
	// Decode collapses local-timestamp-* with timestamp-* because both
	// resolve to the same UTC time.Time (the wire long is interpreted
	// identically; see logical.go for the encode-side rationale).
	logicalDesers = map[string]deserfn{
		"timestamp-millis":       deserTimestampMillis,
		"local-timestamp-millis": deserTimestampMillis,
		"timestamp-micros":       deserTimestampMicros,
		"local-timestamp-micros": deserTimestampMicros,
		"timestamp-nanos":        deserTimestampNanos,
		"local-timestamp-nanos":  deserTimestampNanos,
		"date":                   deserDate,
		"time-millis":            deserTimeMillis,
		"time-micros":            deserTimeMicros,
		"uuid":                   deserUUID,
	}
)

func logicalSer(logical string) serfn     { return logicalSers[logical] }
func logicalDeser(logical string) deserfn { return logicalDesers[logical] }

// unmarshalDefault parses a field's raw JSON default. Numeric literals stay
// json.Number rather than rounding through float64 — long defaults above 2^53
// would otherwise silently lose precision (9007199254740993 → …92) — which is
// what the shared decoder returns for every number anyway.
func unmarshalDefault(raw json.RawMessage) any {
	// Cannot fail: raw is preserved from the initial parse and is valid JSON.
	// Lenient rather than strict for the same reason — whatever the parse
	// accepted into these bytes is what comes back out.
	dv, _, _ := decodeSchemaAny(string(raw))
	return dv
}

// nodeAwaitsForwardRef reports whether node has any child encodeDefault would
// traverse that is not yet whole. encodeDefault recurses into items / values /
// fields / branches and dereferences each child's kind, so two child shapes
// must defer the default pipeline to finalize:
//
//   - a nil child: an unwired forward reference to a type declared later.
//     Dereferencing it panics.
//   - a partial record/error still in b.building: a self- or mutually
//     recursive reference back into a record whose field loop is unfinished.
//     Its fields slice holds only the fields declared so far, so encoding
//     inline drops the rest and emits truncated, non-decodable default bytes.
//
// True here means the caller defers the whole resolve+encode-default pipeline
// to finalize rather than running it inline.
//
// Cycle-safe via a seen set: a back-edge to an already-built node (a recursive
// schema whose referenced record finished building) is a wired pointer not in
// b.building, so it is correctly not treated as pending.
func (b *builder) nodeAwaitsForwardRef(node *schemaNode) bool {
	return nodeAwaitsForwardRefSeen(node, b.building, map[*schemaNode]struct{}{})
}

func nodeAwaitsForwardRefSeen(node *schemaNode, building, seen map[*schemaNode]struct{}) bool {
	if node == nil {
		return true
	}
	if _, ok := seen[node]; ok {
		return false
	}
	seen[node] = struct{}{}
	switch node.kind {
	case "array":
		return nodeAwaitsForwardRefSeen(node.items, building, seen)
	case "map":
		return nodeAwaitsForwardRefSeen(node.values, building, seen)
	case "record", "error":
		if _, ok := building[node]; ok {
			return true
		}
		for i := range node.fields {
			if nodeAwaitsForwardRefSeen(node.fields[i].node, building, seen) {
				return true
			}
		}
	case "union":
		for _, b := range node.branches {
			if nodeAwaitsForwardRefSeen(b, building, seen) {
				return true
			}
		}
	}
	return false
}

// resolveFieldDefaultValue runs the validate + convertDefaultBytes half of
// the default pipeline against a coerced default value and its resolved
// schemaNode, recording the (converted) default on the deser/field metadata.
// It deliberately does NOT encode the binary defaultBytes — that is
// [encodeFieldDefaultBytes], which must run only after every field's default
// VALUE is recorded, because encodeDefault fills absent nested record fields
// from their f.defaultVal. Returns the converted value for the caller to hand
// to encodeFieldDefaultBytes.
//
// convertDefaultBytes maps bytes/fixed string defaults to []byte so the JSON
// encoder sees the wire form directly and its logical-type-aware arms can't
// misinterpret the string as decimal / UUID / etc. Walks the resolved
// schemaNode tree (not the aschema canon) so name-references — forward and
// backward — follow into the real type.
func resolveFieldDefaultValue(defaultVal any, node *schemaNode, fieldName string,
	drf *deserRecordField, fn *fieldNode,
) (any, error) {
	if err := validateDefault(defaultVal, node); err != nil {
		return nil, fmt.Errorf("record field %q: invalid default: %v", truncForError(fieldName), err)
	}
	defaultVal = convertDefaultBytes(defaultVal, node)
	drf.defaultVal = defaultVal
	drf.hasDefault = true
	fn.defaultVal = defaultVal
	fn.hasDefault = true
	return defaultVal, nil
}

// encodeFieldDefaultBytes encodes the (already-resolved) default value into
// the field's pre-encoded binary defaultBytes. Split from
// resolveFieldDefaultValue so deferred defaults can resolve every field's
// VALUE first (encodeDefault reads sibling/nested f.defaultVal for absent
// fields).
func encodeFieldDefaultBytes(defaultVal any, node *schemaNode, fieldName string, srf *serRecordField) error {
	defaultBytes, deferred, err := encodeDefaultCharged(defaultVal, node)
	if err != nil {
		return fmt.Errorf("record field %q: encoding default: %v", truncForError(fieldName), err)
	}
	srf.defaultBytes = defaultBytes
	srf.hasDefault = true
	// Recorded, not returned: a default that cannot be WRITTEN must not stop
	// the schema PARSING, because a reader that drops this field never writes
	// it and reads such data correctly today. The encode-side consumers of
	// defaultBytes surface this at the moment the default would reach the wire.
	// The verdict comes from INSIDE the walk, where each leaf asked the same
	// predicate its serializer asks — asking here instead would answer for the
	// field's kind and miss every cap nested inside a container.
	srf.defaultErr = deferred
	return nil
}

// applyResolvedDefault runs the full validate + convertDefaultBytes +
// encodeDefault pipeline for a coerced default value against its resolved
// schemaNode, writing the result into the three field-slot triple
// (deserRecordField, fieldNode, serRecordField). fieldName is used for error
// context.
//
// Used by the build-time path for fields whose type tree is fully resolved
// (no pending forward reference — [nodeAwaitsForwardRef] is false). Fields
// with an unresolved forward-referenced descendant defer to finalize via the
// split resolveFieldDefaultValue / encodeFieldDefaultBytes pair so encodeDefault
// never dereferences a not-yet-wired child node.
func applyResolvedDefault(defaultVal any, node *schemaNode, fieldName string,
	drf *deserRecordField, fn *fieldNode, srf *serRecordField,
) error {
	converted, err := resolveFieldDefaultValue(defaultVal, node, fieldName, drf, fn)
	if err != nil {
		return err
	}
	return encodeFieldDefaultBytes(converted, node, fieldName, srf)
}

// unmarshalAnyPreservePrecision parses raw JSON into a Go value with the
// same shape as encoding/json's default any decode (map[string]any, []any,
// string, bool, nil for the structural pieces) BUT preserves integer
// precision: integer-valued JSON numbers materialize as int64 instead of
// float64, lifting the silent 2^53 round-down that bare
// json.Unmarshal(&v any) applies. Fractional / exponent-form numbers
// stay float64 since their natural domain is float64-precision anyway.
// Integers that overflow int64 are returned as json.Number so the
// caller still has arbitrary-precision access via .String() / .Int().
//
// Used by the metadata surfaces — record-level extras feeding
// SchemaNode.Props, and Root()'s re-parse — where a bare Unmarshal silently
// rounds JSON ints above 2^53. unmarshalDefault already gives the encode/decode
// path the same guarantee.
func unmarshalAnyPreservePrecision(raw string) (any, error) {
	v, err := decodeSchemaAnyStrict(raw)
	if err != nil {
		return nil, err
	}
	return normalizeJSONValue(v), nil
}

// normalizeJSONValue recursively walks a value parsed via UseNumber and
// converts json.Number to int64 / float64 / json.Number per
// normalizeJSONNumber. Maps and slices are walked in place; other types
// pass through.
func normalizeJSONValue(v any) any {
	switch tv := v.(type) {
	case json.Number:
		return normalizeJSONNumber(tv)
	case map[string]any:
		for k, val := range tv {
			tv[k] = normalizeJSONValue(val)
		}
		return tv
	case []any:
		for i, val := range tv {
			tv[i] = normalizeJSONValue(val)
		}
		return tv
	}
	return v
}

// normalizeJSONNumber resolves a UseNumber-preserved json.Number to the
// idiomatic Go type by VALUE, not by literal syntax:
//
//   - Exact integer fitting int64 → int64. Covers 42, 1.5e1 (= 15) and
//     9.5e17 alike; the literal's syntax does not matter.
//   - Exact integer exceeding int64 in pure-digit syntax → json.Number,
//     preserving arbitrary precision.
//   - Anything else → float64, with ±Inf for magnitudes past float64's
//     exponent range ("1e1000" → +Inf).
//
// Going by VALUE rather than syntax is what removes a metadata-vs-wire
// divergence at the int64 boundary: syntax-based dispatch gave
// "9.2233720368547758e18" a wire value of int64(9223372036854775800) and a
// metadata value of float64 rounded to 2^63. Both are now the int64.
//
// The overflow path routes through [parseFloatAcceptOverflow] so the metadata
// surface agrees with the encode/decode/parse arms on ErrRange-with-Inf, as
// Java's Jackson and fastavro's float() both do.
func normalizeJSONNumber(n json.Number) any {
	s := string(n)
	// Integer-syntax fast path: no decimal point, no exponent — strconv
	// alone is enough, no need to spin up a big.Rat.
	if !strings.ContainsAny(s, ".eE") {
		if i, err := n.Int64(); err == nil {
			return i
		}
		// Overflows int64; preserve as json.Number for arbitrary precision.
		return n
	}
	// Fractional or exponent syntax. Value-based dispatch: parse with
	// arbitrary precision and check if the value is an exact integer.
	// Without this, a literal like "1.5e1" (= 15) or "9.5e17"
	// (= 950000000000000000) surfaces as float64 — silently rounding for
	// values exceeding float64's 53-bit mantissa and diverging from the
	// wire-encode pipeline's exact-integer parse for integer-defaultable
	// schemas. Going through boundedRatFromString lets metadata and wire
	// agree on the same int64 value regardless of how the user wrote the
	// literal.
	if r, ok, err := boundedRatFromString(s); err == nil && ok && r.IsInt() {
		// Negative zero in float syntax ("-0.0", "-0e5") is the one exact
		// integer whose IEEE sign the int64 collapse would erase (a big.Rat
		// has no signed zero). The wire encoder parses it via ParseFloat and
		// preserves the sign, and Java's Jackson surfaces a DoubleNode(-0.0);
		// keep the sign by falling through to parseFloatAcceptOverflow below
		// (→ -0.0) so the metadata Default matches the wire and re-parses
		// sign-stable. Integer syntax ("-0") stays integer 0 (no sign) above.
		negZero := r.Sign() == 0 && s != "" && s[0] == '-'
		if !negZero {
			if bi := r.Num(); bi.IsInt64() {
				return bi.Int64()
			}
		}
		// Exact integer beyond int64: surface as float64 when the magnitude
		// fits float64's exponent, ±Inf when it does not. Both match what an
		// encode against a float/double schema puts on the wire.
	}
	if f, err := parseFloatAcceptOverflow(s, 64); err == nil {
		return f
	}
	return n
}

// numericDefault extracts a typed integer default, the shared body of
// defaultAsInt32 / defaultAsInt64. A JSON number arrives from unmarshalDefault
// as json.Number at full precision; callers may also pass float64, e.g.
// round-tripped through coerceDefault.
//
// Whole numbers written in fractional or exponent form ("1.0", "4e1") are
// accepted, matching the runtime's existing "whole-number floats encode
// against int/long" divergence from Java's isIntegralNumber gate and
// fastavro's isinstance check, which both reject "1.0". Tightening only at
// parse would create a within-package encode-vs-parse asymmetry.
//
// The precision guard rejects exactly the subset where the metadata path
// (normalizeJSONNumber, which surfaces such literals as float64) would round
// to a DIFFERENT integer than the wire fill (parseInt{32,64}Lenient, exact via
// big.Rat). "1.0" and "4e1" represent exactly in float64, so both surfaces
// agree and it accepts. "9.2233720368547758e18" does not: wire gives
// 9223372036854775800, float64 rounds to 9223372036854775808+. Rejecting keeps
// a schema from carrying a default whose metadata and wire values disagree,
// while accepted defaults still produce Java/fastavro-identical wire bytes.
func numericDefault[T int32 | int64](val any, parse func(string) (T, error), fromFloat func(float64) (T, error), fromInt64 func(int64) (T, error)) (T, error) {
	switch v := val.(type) {
	case json.Number:
		return parse(string(v))
	case float64:
		return fromFloat(v)
	case int64:
		return fromInt64(v)
	case int32:
		return fromInt64(int64(v))
	}
	var z T
	return z, fmt.Errorf("expected number, got %T", val)
}

// int64FitsInt32 narrows n to int32 with bounds check. Shared by
// numericDefault's int64/int32 arms (for defaultAsInt32 callers) and
// keeps the bounds rule in one place.
func int64FitsInt32(n int64) (int32, error) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("integer %d overflows int32", n)
	}
	return int32(n), nil
}

// int64Identity is numericDefault's fromInt64 for the int64 (long)
// target — pass through unchanged.
func int64Identity(n int64) (int64, error) { return n, nil }

func defaultAsInt32(val any) (int32, error) {
	return numericDefault(val, parseInt32Lenient, floatFitsInt32, int64FitsInt32)
}

func defaultAsInt64(val any) (int64, error) {
	return numericDefault(val, parseInt64Lenient, floatFitsInt64, int64Identity)
}

// floatMantissaLimit returns the largest integer magnitude exactly
// representable in float32 (bitSize=32) or float64 (bitSize=64) —
// the mantissa bound used for float→int whole-number precision-loss
// checks at [floatFitsInt32From] and [floatFitsInt64From]. The reverse
// direction (int→float) is lossy by destination per Java/fastavro parity;
// see [appendAvroFloat32] / [appendAvroFloat64].
func floatMantissaLimit(bitSize int) int64 {
	if bitSize == 32 {
		return 1 << 24
	}
	return 1 << 53
}

// intFitsFloat reports whether an int64 value of magnitude n can be
// represented exactly in the target float (float32 or float64). Used
// by decode-time arms that write a long-wire value into a Go float
// target: the user explicitly chose a smaller-precision Go type, so we
// surface the precision loss rather than silently rounding. Encode-time
// arms use the lossy-destination policy and silently round; see
// [appendAvroFloat32] / [appendAvroFloat64].
func intFitsFloat(n int64, bitSize int) (float64, error) {
	lim := floatMantissaLimit(bitSize)
	if n < -lim || n > lim {
		return 0, fmt.Errorf("integer %d overflows float%d exact precision", n, bitSize)
	}
	return float64(n), nil
}

// parseFloatAcceptOverflow is [strconv.ParseFloat] with one twist:
// ErrRange-with-±Inf counts as success, since Java/fastavro return the Inf and
// the wire format permits it. Other parse errors propagate.
//
// It is the single source of truth for every ParseFloat-on-user-input caller —
// binary encode, JSON encode, schema parse, the metadata API, and JSON decode
// — so the length cap covers them all. ParseFloat is O(n) at ~30-50ms/MiB, and
// schema parse calls this twice per float field, so a 1 MiB hostile literal
// would cost ~130ms, past the 100ms per-entry-point bound. Legitimate float64
// literals, hex-float and max-exponent forms included, fit well under 350
// chars, so the 1024-byte cap rejects hostile input in O(1) without touching
// real ones. Same pattern as boundedRatFromString and parseInt64Lenient.
//
// bitSize is 64 everywhere except the JSON decode of a "float" schema, which
// passes 32 to parse at float32 precision directly and avoid a float64→float32
// double rounding.
func parseFloatAcceptOverflow(s string, bitSize int) (float64, error) {
	if len(s) > maxParseFloatLen {
		return 0, fmt.Errorf("float literal exceeds %d byte length cap", maxParseFloatLen)
	}
	f, err := strconv.ParseFloat(s, bitSize)
	if err == nil {
		return f, nil
	}
	if errors.Is(err, strconv.ErrRange) && math.IsInf(f, 0) {
		return f, nil
	}
	return 0, err
}

// maxParseFloatLen caps what parseFloatAcceptOverflow forwards to
// [strconv.ParseFloat]; see there for why. The longest legitimate float64
// literal fits in ~320 chars, so this leaves comfortable headroom.
const maxParseFloatLen = 1024

// defaultAsFloat extracts a numeric default for a float or double field.
// Accepts json.Number / float64 / int64 / int32 — i.e., the Go types
// produced by UseNumber-parsed JSON literals plus any post-coerce
// numerics. Does NOT accept Go string: the spec 1.12 §"Record" default-
// values table requires the JSON type of a float/double default to be
// `number`, never a JSON string. The narrow Java-deployed exception —
// Schema.java:1899-1902's parseField text→DoubleNode coercion for outer
// FLOAT/DOUBLE field types — is handled UPSTREAM in [coerceDefault] so
// the string never reaches this validator. Union branches and any
// downstream caller (encodeDefault, validateLeaf, the metadata-side
// branchAcceptsDefault path) see only post-coerce typed values; a
// string here is invalid by construction and rejected via the default
// `expected number, got %T` error.
//
// Encoding into a float/double field is lossy by destination — int64/
// int32 inputs exceeding the mantissa precision silently IEEE-round
// (matches Java's Schema.parseField text→DoubleNode coercion and
// fastavro's float()). The float32 narrowing to ±Inf happens at the
// caller's float64 → float32 cast.
func defaultAsFloat(val any) (float64, error) {
	switch v := val.(type) {
	case json.Number:
		return parseJSONNumberAsFloat(v.String(), 64)
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case float32:
		// A float32 reaches this predicate only on the METADATA side: the
		// union-branch selector branchAcceptsDefault coerces a container's
		// nested float/double child through coerceMetadataDefault before the
		// accept-check, and coerceMetadataDefault narrows a "float" schema to
		// float32 (schema_node.go). float32→float64 widening is exact, so the
		// value is a valid float default. The wire path never reaches here with
		// a float32 (coerceDefault yields float64 and the parsed default tree
		// has no float32), so this arm does not change wire or parse behavior.
		return float64(v), nil
	}
	return 0, fmt.Errorf("expected number, got %T", val)
}

// firstUnionBranchAcceptingDefault returns the first union branch whose
// validateDefault accepts val, or nil if none match. Shared by
// coerceDefault and walkDefault's union arms — both implement Avro's
// "first matching branch wins" default-resolution rule (1.12 relaxed
// from "first branch" to "any branch," with deterministic first-match
// tie-break). Keeping the iteration in one place ensures coerceDefault
// and walkDefault stay in lockstep if validateDefault's semantics
// change. coerceMetadataDefault (schema_node.go) uses the analogous
// branchAcceptsDefault predicate on the *SchemaNode public type — the
// pattern is the same but the type split prevents direct reuse.
//
// String defaults are matched ONLY by branches whose Avro type's
// permitted JSON type is `string` per spec 1.12 §"Record" default-
// values table (string, bytes, enum, fixed). Numeric branches
// (int, long, float, double) reject string defaults at this layer
// — [defaultAsFloat] has no string-acceptance arm; Java's
// parseField text→DoubleNode coercion fires only for the OUTER
// FLOAT/DOUBLE field type (handled in [coerceDefault] below) and
// never for union branches.
func firstUnionBranchAcceptingDefault(val any, node *schemaNode) *schemaNode {
	for _, branch := range node.branches {
		// validateDefault coerces in place: validateLeaf's record/array/map
		// arms rewrite fields via coerceDefault (e.g. a string "5" -> float64
		// against a double field, the documented outer-float carveout). Validate
		// a COPY so a FAILED branch's partial coercion cannot leak into the next
		// branch's check — otherwise acceptance is order-dependent (a later
		// string-typed branch sees a float64 a prior failed branch left behind),
		// violating Avro 1.12's order-independent "default matches any branch"
		// (Java isValidDefault is anyMatch over an immutable node). The caller
		// re-coerces the original val against the returned branch, so the
		// selected branch and its coerced value are unchanged for any default
		// that already parsed.
		if validateDefault(deepCopyTree(val), branch) == nil {
			return branch
		}
	}
	return nil
}

// coerceDefault converts a string default to float64 when the field type is
// literally float or double. The spec's default-values table calls a JSON
// string invalid there; this is an interop carveout for legacy Java-generated
// schemas, modeled on parseField's TextNode → DoubleNode special case
// (Schema.java:1899-1902). avro-rs and goavro do not implement it.
//
// A DIRECT scalar float/double UNION branch does NOT coerce: only a node whose
// kind is literally float/double transforms, and a union recurses into its
// first validateDefault-accepting branch, where the scalar leaf has no string
// arm. So ["double"] default "5" rejects at parse and ["double","string"]
// picks string, matching Java/avro-rs/goavro (NOT_BUGS #10).
//
// A float/double field NESTED in a record/array/map DOES coerce, container-in-
// a-union included, because validateLeaf's container arms call coerceDefault
// on each child. That is deliberate leniency beyond Java, whose coercion is
// outer-field-only. It is safe because the metadata selector applies the same
// coercion via coerceMetadataDefault, so Root().Default reports the branch the
// wire auto-fills on both formats — no metadata/wire divergence. Do not re-file
// this as a Java mismatch; revisit only on a real interop breakage with
// evidence. Pinned by
// TestMatrix_UnionContainerNestedFloatDefaultSelectionMatchesWire.
//
// Walks *schemaNode, the resolved type tree, so name-referenced nested fields
// coerce too — name-refs lose their type info on the canon side.
func coerceDefault(val any, node *schemaNode) any {
	if node == nil {
		return val
	}
	if node.kind == "union" {
		// First validateDefault-accepting branch wins; recurse so the coerced
		// value matches that branch's natural Go type. No numeric branch
		// accepts a string (defaultAsFloat has no string arm), so a string
		// default picks string/bytes/enum/fixed or nothing, and parse then
		// fails via validateDefault.
		if branch := firstUnionBranchAcceptingDefault(val, node); branch != nil {
			return coerceDefault(val, branch)
		}
		return val
	}
	if node.kind != "float" && node.kind != "double" {
		return val
	}
	s, ok := val.(string)
	if !ok {
		return val
	}
	// parseFloatAcceptOverflow directly, not defaultAsFloat: that one is the
	// strict validator union branches and the encode arms share, and this
	// leniency is scoped to this call site alone. On a parse failure keep the
	// original string so validateDefault produces the canonical error.
	if f, err := parseFloatAcceptOverflow(s, 64); err == nil {
		return f
	}
	return val
}

// walkDefault drives the (val, node) recursion shared by the
// default-tree walkers. visit is called once per non-union node and
// may mutate val; for union nodes walkDefault picks the first
// validateDefault-accepting branch (skipping visit at the union
// itself) and recurses into the matched branch. If no union branch
// matches, walkDefault returns the canonical "default does not match
// any union branch" error so callers that don't care (the mutator
// walker convertDefaultBytes) can discard it while validateDefault
// surfaces it.
//
// Container arms wrap nested errors with "field %q:", "array element
// %d:", or "map key %q:" so the per-element error path is identical
// across walkers.
//
// Caller contract: visit MUST be idempotent. The union arm calls
// validateDefault to pick a branch and then re-invokes visit at every
// node of the matched branch — a non-idempotent visit (e.g. one that
// increments a counter) would double-fire at every union depth.
//
// Walks *schemaNode (the resolved type tree) so name-references —
// forward and backward — follow into the real type. Returns
// immediately for a nil node so fwd-ref-deferred validation is a
// no-op.
func walkDefault(val any, node *schemaNode, visit func(any, *schemaNode) (any, error)) (any, error) {
	if node == nil {
		return val, nil
	}
	if node.kind == "union" {
		// Per Avro 1.12 a default may match any branch, not only the first
		// (AVRO-3649). The matcher is validateDefault, shared with
		// coerceDefault: a structural-only check like "is val a string?" would
		// pick a fixed:N branch for a string whose rune count does not fit,
		// mutate it into a length-N []byte no branch can encode, and fail at
		// encodeDefault on a schema validateDefault accepted. Re-running
		// validateDefault here is safe — it is idempotent.
		if branch := firstUnionBranchAcceptingDefault(val, node); branch != nil {
			return walkDefault(val, branch, visit)
		}
		return val, fmt.Errorf("default does not match any union branch: %T(%s)", val, truncValueForError(val))
	}
	val, err := visit(val, node)
	if err != nil {
		return val, err
	}
	switch node.kind {
	case "record":
		if m, ok := val.(map[string]any); ok {
			for _, f := range node.fields {
				fv, exists := m[f.name]
				if !exists {
					continue
				}
				fv2, err := walkDefault(fv, f.node, visit)
				if err != nil {
					return val, fmt.Errorf("field %q: %w", truncForError(f.name), err)
				}
				m[f.name] = fv2
			}
		}
	case "array":
		if arr, ok := val.([]any); ok && node.items != nil {
			for i, item := range arr {
				item2, err := walkDefault(item, node.items, visit)
				if err != nil {
					return val, fmt.Errorf("array element %d: %w", i, err)
				}
				arr[i] = item2
			}
		}
	case "map":
		if m, ok := val.(map[string]any); ok && node.values != nil {
			for k, v := range m {
				v2, err := walkDefault(v, node.values, visit)
				if err != nil {
					return val, fmt.Errorf("map key %q: %w", truncForError(k), err)
				}
				m[k] = v2
			}
		}
	}
	return val, nil
}

// convertDefaultBytes walks a parsed-then-validated default value and
// converts string defaults to []byte for bytes/fixed schema nodes,
// recursively descending into records/arrays/maps/unions. The Avro
// JSON spec specifies that bytes/fixed defaults are codepoint-mapped
// strings; binary encodeDefault already takes the codepoint route via
// avroJSONBytesToBytes, while the JSON encoder's appendAvroJSON
// logical-type-aware arms (decimal, big-decimal, uuid) would otherwise
// misinterpret the string semantically. Storing the wire-form []byte
// up front makes both encode paths agree without requiring per-arm
// special cases.
//
// Called after validateDefault has succeeded for non-fwd-ref fields;
// for fwd-ref fields validation is deferred and the conversion is
// best-effort. The walkDefault union-no-match error is discarded —
// validateDefault would have caught it for non-fwd-ref defaults, and
// fwd-ref defaults shouldn't surface conversion-time errors.
func convertDefaultBytes(val any, node *schemaNode) any {
	out, _ := walkDefault(val, node, func(val any, node *schemaNode) (any, error) {
		if node.kind != "bytes" && node.kind != "fixed" {
			return val, nil
		}
		if str, ok := val.(string); ok {
			if b, err := avroJSONBytesToBytes(str); err == nil {
				return b, nil
			}
		}
		return val, nil
	})
	return out
}

// validateAvroByteString reports an error when s contains a code point
// > 0xFF — the Avro JSON-bytes / JSON-fixed default form maps each
// codepoint to one byte, so values outside that range are not
// representable. fieldType is "bytes" or "fixed" for the message.
func validateAvroByteString(s, fieldType string) error {
	for _, r := range s {
		if r > 255 {
			return fmt.Errorf("%s default contains code point U+%04X, max allowed is U+00FF", fieldType, r)
		}
	}
	return nil
}

// validateDefault checks a parsed JSON default against the schema, driving
// walkDefault with the validateLeaf visit. The recursion, union branch-matching
// and per-element error wrapping stay in walkDefault so the validate / convert
// / coerce walkers cannot drift on them.
//
// Mutates record/array/map structures in place via coerceDefault, propagating
// float-from-string coercions into nested fields reached through name-refs.
// Returns nil for a nil node; fwd-refs defer validation to finalize.
func validateDefault(val any, node *schemaNode) error {
	_, err := walkDefault(val, node, validateLeaf)
	return err
}

// defaultObjectShape asserts val is a non-null JSON object for an Avro record
// or map default, returning the canonical "expected object for <kind> default"
// error. Shared by the parse-time validator (validateLeaf) and the wire encoder
// (encodeDefault, resolve.go) — two cross-path sites for one shape rule — so
// the user-visible error cannot drift between them. Only the shape assertion is
// shared; each caller keeps its own post-assertion work (validateLeaf coerces
// fields in place, encodeDefault emits wire bytes).
func defaultObjectShape(val any, kind string) (map[string]any, error) {
	if val == nil {
		return nil, fmt.Errorf("expected object for %s default, got null", kind)
	}
	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected object for %s default, got %T", kind, val)
	}
	return m, nil
}

// defaultArrayShape is defaultObjectShape's array counterpart.
func defaultArrayShape(val any) ([]any, error) {
	if val == nil {
		return nil, fmt.Errorf("expected array for array default, got null")
	}
	arr, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array for array default, got %T", val)
	}
	return arr, nil
}

// validateLeaf is the per-node visit for validateDefault: primitive
// kind validation, plus container-shape checks + per-field coercion
// (walkDefault handles the actual recursion).
func validateLeaf(val any, node *schemaNode) (any, error) {
	switch node.kind {
	case "null":
		if val != nil {
			return val, fmt.Errorf("expected null, got %T", val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return val, fmt.Errorf("expected boolean, got %T", val)
		}
	case "int":
		if _, err := defaultAsInt32(val); err != nil {
			return val, fmt.Errorf("int default: %w", err)
		}
	case "long":
		if _, err := defaultAsInt64(val); err != nil {
			return val, fmt.Errorf("long default: %w", err)
		}
	case "float", "double":
		if _, err := defaultAsFloat(val); err != nil {
			return val, fmt.Errorf("%s default: %w", node.kind, err)
		}
	case "string":
		if _, ok := val.(string); !ok {
			return val, fmt.Errorf("expected string, got %T", val)
		}
	case "bytes":
		s, ok := val.(string)
		if !ok {
			return val, fmt.Errorf("expected string for bytes, got %T", val)
		}
		return val, validateAvroByteString(s, "bytes")
	case "enum":
		sym, ok := val.(string)
		if !ok {
			return val, fmt.Errorf("expected string for enum default, got %T", val)
		}
		// Unconditional membership: a non-nil enum node always carries its
		// final symbols, so an empty enum rejects every default. STRICTER than
		// both references — Java checks containment only for the enum-level
		// "default" attribute and accepts any textual FIELD default
		// (Schema.java:1755-1759), and fastavro 1.12.2 parses a non-member
		// outright (observed). Failing at parse is right because a non-member
		// default can never encode, and it makes union-default selection skip
		// an empty or non-member enum branch so a later one can accept.
		if !slices.Contains(node.symbols, sym) {
			return val, fmt.Errorf("enum default %q is not a member of symbols", truncForError(sym))
		}
	case "fixed":
		s, ok := val.(string)
		if !ok {
			return val, fmt.Errorf("expected string for fixed default, got %T", val)
		}
		if err := validateAvroByteString(s, "fixed"); err != nil {
			return val, err
		}
		if len([]rune(s)) != node.size {
			return val, fmt.Errorf("fixed default length %d does not match size %d", len([]rune(s)), node.size)
		}
	case "record":
		// null is not a record (Java/fastavro/hamba all reject). Without
		// this, ["Record","null"] with default null would match the
		// Record branch (synthesizing an empty map + relying on per-
		// field defaults) instead of falling through to null —
		// encodeDefault would emit Record(field-defaults) wire bytes
		// where null was intended.
		m, err := defaultObjectShape(val, "record")
		if err != nil {
			return val, err
		}
		// Required-field presence check before coercion: a missing
		// no-default field is an error regardless of per-field types.
		for _, f := range node.fields {
			if _, exists := m[f.name]; !exists && !f.hasDefault {
				return val, fmt.Errorf("record default missing field %q with no default", truncForError(f.name))
			}
		}
		// Coerce each present field in-place; walkDefault then recurses
		// to validate the coerced value at each child node.
		for _, f := range node.fields {
			if fv, exists := m[f.name]; exists {
				m[f.name] = coerceDefault(fv, f.node)
			}
		}
	case "array":
		arr, err := defaultArrayShape(val)
		if err != nil {
			return val, err
		}
		for i, item := range arr {
			arr[i] = coerceDefault(item, node.items)
		}
	case "map":
		m, err := defaultObjectShape(val, "map")
		if err != nil {
			return val, err
		}
		for k, v := range m {
			m[k] = coerceDefault(v, node.values)
		}
	}
	return val, nil
}
