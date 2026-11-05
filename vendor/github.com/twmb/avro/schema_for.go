package avro

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"
)

// SchemaOpt configures schema construction via [Parse], [SchemaCache.Parse],
// or [SchemaFor]. Inapplicable options are silently ignored.
type SchemaOpt interface{ schemaOpt() }

type schemaOpts struct {
	namespace string
	name      string
}

type withNamespace string

func (withNamespace) schemaOpt() {}

// WithNamespace sets the Avro namespace for the top-level record in
// [SchemaFor]. Ignored by [Parse].
func WithNamespace(ns string) SchemaOpt { return withNamespace(ns) }

type withName string

func (withName) schemaOpt() {}

// WithName overrides the Avro record name in [SchemaFor]. By default
// the Go struct name is used. Ignored by [Parse].
func WithName(name string) SchemaOpt { return withName(name) }

// SchemaFor infers an Avro schema from the Go type T. T must be a struct.
//
// Field names are taken from the avro struct tag, falling back to the Go
// field name. The following tag options are supported:
//
//   - avro:"-" excludes the field
//   - avro:",inline" flattens a nested struct's fields into the parent
//   - avro:",omitzero" is recorded but does not affect the schema
//   - avro:",alias=old_name" adds a field alias (repeatable)
//   - avro:",type-alias=old_name" adds an alias to the field's named type (record, enum, fixed; repeatable)
//   - avro:",default=value" sets the field's default value (must be last option; scalars only)
//   - avro:",timestamp-millis" overrides the logical type (also: timestamp-micros,
//     timestamp-nanos, date, time-millis, time-micros)
//   - avro:",decimal(precision,scale)" sets the decimal logical type
//   - avro:",uuid" sets the uuid logical type
//
// Type inference:
//   - bool → boolean
//   - int8, int16, int32 → int
//   - int, int64, uint32 → long
//   - uint8, uint16 → int
//   - float32 → float
//   - float64 → double
//   - string → string
//   - []byte → bytes
//   - [N]byte → fixed (size N, name from Go type name or "fixed_N")
//   - *T → ["null", T] union with default null (a pointer chain of any
//     depth — **T, ***T — collapses to the same single nullable union)
//   - []T → array
//   - map[string]T → map
//   - struct → record (recursive)
//   - time.Time → long with timestamp-millis (override with tag)
//   - time.Duration → int with time-millis (override with tag; a Duration is a
//     span of time, so it is only meaningful with the time-millis/time-micros
//     logicals — overriding it onto date or a timestamp-* logical maps a
//     duration onto a point in time, and a large Duration overflows the
//     narrower wire type)
//   - avro.Duration → fixed(12) with the duration logical type (the dedicated
//     Go type for the Avro duration logical — little-endian months/days/
//     milliseconds; recognized by type, takes no tag, and does not accept one)
//   - *big.Rat → requires explicit decimal(p,s) tag
//   - [16]byte with uuid tag → fixed(16) with uuid logical type
//   - string (or text marshaler type) with uuid tag → string with uuid logical type
func SchemaFor[T any](opts ...SchemaOpt) (*Schema, error) {
	var o schemaOpts
	var customTypes []CustomType
	for _, opt := range opts {
		switch v := opt.(type) {
		case withNamespace:
			o.namespace = string(v)
		case withName:
			o.name = string(v)
		case CustomType:
			customTypes = append(customTypes, v)
		}
	}
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("avro: SchemaFor requires a struct type, got %s", t)
	}
	name := o.name
	if name == "" {
		name = t.Name()
	}
	seen := make(map[reflect.Type]seenForm)
	// applied is threaded globally alongside seen. type-alias dedup keys on a
	// named type's fullname, and seen guarantees exactly one definition per type
	// across the whole inference, so the applied state must span the whole call
	// too. A per-record map made cross-record identical aliases on a shared named
	// type spuriously reject: the type is defined (alias recorded) in one record
	// but referenced from another record whose fresh map is empty, so the
	// reference fell into the "defined without type-alias" branch. applied is only
	// SET at a definition (addTypeAliases applied==true) and only READ at a
	// reference, so global scope never false-accepts — same-name distinct-type
	// collisions are still caught independently by dedupNamedTypes.
	applied := make(appliedTypeAliases)
	s, err := inferRecord(t, name, o.namespace, seen, customTypes, applied)
	if err != nil {
		return nil, err
	}
	s, err = dedupNamedTypes(s, make(map[string]string), "")
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("avro: marshaling inferred schema: %w", err)
	}
	return Parse(string(b), opts...)
}

// resolveNameScope resolves a named-kind node's identity at its position,
// following the parser's rules (spec, "Names"): a dotted name is a fullname
// whose namespace attribute is ignored; an explicit namespace attribute
// (including the "" inheritance escape) is authoritative; otherwise the
// name inherits the enclosing namespace. Returns the resolved fullname and
// the namespace scope the node opens for its children (record fields
// resolve inside it). Shared by dedupNamedTypes and normalizeSchemaScope so
// the keying walk and the equality walk cannot drift on scope rules.
//
// Reserved keys are read by exact name: the tree this walk keys is the
// tree Parse will consume, and Parse matches reserved attribute names
// only by their exact lowercase spelling (a case-variant key is an
// ordinary custom property — see [Schema.Root]), so exact reads here key
// definitions exactly as Parse binds them.
func resolveNameScope(v map[string]any, enclosingNS string) (full, ns string) {
	name, _ := v["name"].(string)
	short := name
	ns = enclosingNS
	if i := strings.LastIndex(name, "."); i >= 0 {
		short, ns = name[i+1:], name[:i]
	} else if attr, ok := v["namespace"].(string); ok {
		ns = attr
	}
	return avroFullName(ns, short), ns
}

// normalizeSchemaScope returns a copy of a schema tree with every name
// resolved against its position — named definitions carry their fullname in
// "name" with no separate namespace attribute, and bare name references are
// qualified by the enclosing scope — so two renderings of one definition
// compare equal exactly when they denote the same types. The raw relative
// JSON of one definition can differ by position (an explicit namespace
// attribute at one site, inheritance at another; dotted vs split
// spellings), so dedupNamedTypes compares this normalized form.
func normalizeSchemaScope(v any, enclosingNS string) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		childNS := enclosingNS
		var typ, full string
		named := false
		if tv, ok := v["type"]; ok {
			typ, _ = tv.(string)
			if isNamedKind(typ) {
				named = true
				full, childNS = resolveNameScope(v, enclosingNS)
			}
		}
		// Key classification is by exact name, matching the Parse this
		// tree feeds (see resolveNameScope): only the exact lowercase
		// spelling is the reserved attribute, so only it normalizes — a
		// case-variant key is an ordinary custom property and compares
		// verbatim like any other.
		//
		// Structural keys normalize only on the kind that BINDS them
		// (fields on record/error, items on array, values on map),
		// mirroring the parser's kind-keyed grammar: on any other kind
		// the key is inert as-written metadata (never name-bound), so it
		// compares VERBATIM — two occurrences are one definition exactly
		// when their inert content is byte-identical, not merely
		// spelling-equivalent under a scope the parser never applies.
		for k, val := range v {
			switch {
			case named && k == "name":
				out[k] = full
			case named && k == "namespace":
				// Folded into the fullname.
			case isRecordKind(typ) && k == "fields":
				fields, ok := val.([]map[string]any)
				if !ok {
					out[k] = val
					continue
				}
				nf := make([]map[string]any, len(fields))
				for i, f := range fields {
					cf := make(map[string]any, len(f))
					for fk, fv := range f {
						if fk == "type" {
							cf[fk] = normalizeSchemaScope(fv, childNS)
						} else {
							cf[fk] = fv
						}
					}
					nf[i] = cf
				}
				out[k] = nf
			case typ == "array" && k == "items",
				typ == "map" && k == "values":
				out[k] = normalizeSchemaScope(val, childNS)
			default:
				out[k] = val
			}
		}
		return out
	case []any: // union branches
		out := make([]any, len(v))
		for i, b := range v {
			out[i] = normalizeSchemaScope(b, enclosingNS)
		}
		return out
	case string:
		// A primitive stays itself; a dotted reference is already a
		// fullname; a bare reference resolves in the enclosing namespace.
		if avroPrimitives[v] || strings.Contains(v, ".") || enclosingNS == "" {
			return v
		}
		return enclosingNS + "." + v
	}
	return v
}

// pinCustomSchemaScope pins the namespace scope of a CustomType.Schema
// subtree that is about to be embedded inside a namespaced SchemaFor tree.
// toJSON renders the subtree relative to the null namespace, so a named
// node that neither carries a dotted name nor a namespace attribute
// declares the NULL namespace — but at a namespaced embedding position the
// parser's inheritance would capture it into the surrounding namespace,
// silently renaming the user's declared type. Inject the "namespace":""
// inheritance escape on each such node — the same escape toJSONWalk emits
// for a null-namespace type inside a namespaced scope. The walk stops at
// the first named node on every path: once a node pins its scope (dotted
// name, explicit attribute, or this injection), everything below it renders
// relative to that node and is position-independent already.
func pinCustomSchemaScope(v any) {
	switch v := v.(type) {
	case map[string]any:
		typ, _ := v["type"].(string)
		if isNamedKind(typ) {
			name, _ := v["name"].(string)
			if !strings.Contains(name, ".") {
				// Only the exact "namespace" key is the namespace
				// attribute (a case-variant spelling is an ordinary
				// custom property Parse never reads), so the node pins
				// its scope exactly when the exact key is present.
				if _, has := v["namespace"]; !has {
					v["namespace"] = ""
				}
			}
			return
		}
		// Unnamed containers pass the enclosing scope through; descend to
		// the named frontier — only through the key the node's kind BINDS
		// (items on array, values on map), mirroring the parser's
		// kind-keyed grammar. On any other kind the key is inert
		// as-written metadata: a named-kind-shaped value inside it is
		// never name-bound by Parse, so injecting the inheritance escape
		// there would alter caller metadata, not pin a scope.
		if typ == "array" {
			if items, ok := v["items"]; ok {
				pinCustomSchemaScope(items)
			}
		}
		if typ == "map" {
			if values, ok := v["values"]; ok {
				pinCustomSchemaScope(values)
			}
		}
	case []any: // union branches
		for _, b := range v {
			pinCustomSchemaScope(b)
		}
	}
}

// renderCustomSchemaTree renders a CustomType.Schema subtree for embedding into
// a SchemaFor tree. It is a metadata walk over CALLER-owned storage, which
// gives it two boundary duties:
//
//   - The error-reporting walk, not the bare one, so an over-budget or
//     unnamed-cyclic subtree fails the build by name. The bare walk truncates
//     to nil — correct for the error-LESS surfaces, where the alternative is a
//     panic — but a truncated Props VALUE parses cleanly as a null prop, so
//     nothing downstream would catch the alteration.
//
//   - Deep-copy AND canonicalize before returning. The walk hands Props
//     containers over BY REFERENCE when they need no JSON fixup, and the
//     composition walkers write into the tree they are given —
//     pinCustomSchemaScope injects "namespace":"", dedupNamedTypes rewrites
//     slots into references — so without the copy those writes land in the
//     caller's SchemaNode. Without canonicalizeTreeValue, a caller-typed value
//     like `type M map[string]any` passes every walker type-switch untouched
//     while Parse binds its marshal as real structure. This render is the only
//     way caller-owned containers enter the pre-Parse tree.
func renderCustomSchemaTree(n *SchemaNode) (any, error) {
	d := &deduper{
		defined: make(map[string]*SchemaNode),
		visited: make(map[*SchemaNode]struct{}),
	}
	tree := n.toJSONDedup(d)
	if d.err != nil {
		return nil, fmt.Errorf("avro: SchemaFor: CustomType.Schema: %w", d.err)
	}
	return deepCopyJSONTree(tree), nil
}

// deepCopyJSONTree copies AND CANONICALIZES every container level of a rendered
// tree, so the composition walkers see every value Parse will bind and no
// mutating walker reaches storage shared with the source SchemaNode.
//
// The copy: []string aliases/symbols arrive by reference (emitStrings returns
// the caller's slice), and addTypeAliases appends to a type's "aliases" — an
// append into a caller slice with spare capacity writes their backing array
// past its length, which no deep-equal of the caller's tree can detect.
//
// The canonicalization: the tree's semantics are its json.Marshal output, and a
// caller-typed `type M map[string]any` marshals identically to its canonical
// twin, so left alone it threads through every walker type-switch untouched
// while Parse binds its marshal as real structure.
//
// Nil-ness is part of the marshal image — a nil map marshals null, an empty one
// {} — so every arm preserves it exactly. Scalar leaves stay shared; []byte
// never survives a render.
func deepCopyJSONTree(v any) any {
	switch v := v.(type) {
	case map[string]any:
		if v == nil {
			return v
		}
		out := make(map[string]any, len(v))
		for k, val := range v {
			out[k] = deepCopyJSONTree(val)
		}
		return out
	case []any:
		if v == nil {
			return v
		}
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = deepCopyJSONTree(e)
		}
		return out
	case []map[string]any: // record fields
		if v == nil {
			return v
		}
		out := make([]map[string]any, len(v))
		for i, m := range v {
			out[i] = deepCopyJSONTree(m).(map[string]any)
		}
		return out
	case []string: // aliases, symbols
		if v == nil {
			return v
		}
		out := make([]string, len(v))
		copy(out, v)
		return out
	case nil, string, bool, float64, float32, int, int32, int64, json.Number:
		return v
	}
	return canonicalizeTreeValue(v)
}

// canonicalizeTreeValue rewrites a caller-typed tree value into the
// canonical Go shape whose json.Marshal output is identical: a named
// string-keyed map into map[string]any, a named slice/array into []any
// (or []string when every element is a plain string kind, matching the
// aliases/symbols form), a byte-kinded slice into []byte, named leaves
// into their predeclared types, with pointers and interfaces unwrapped.
// A nil named map/slice canonicalizes to nil — its marshal image is null,
// exactly like its canonical twin's. Values whose marshal is self-defined
// (own MarshalJSON/MarshalText, json.Number — treeValueMarshalOpaque) and
// shapes with no stable same-marshal canonical twin (structs; maps with
// non-string-kind keys, whose MarshalText output is the key on every
// toolchain; slices whose elements' marshal is position-dependent) stay as
// they are: opaque leaves the walkers pass through untouched and Parse
// reads from the marshal — the documented residual posture. Cyclic
// values cannot reach here: the render's budgeted walk (valueWalkLimit)
// errors on them before the copy runs.
func canonicalizeTreeValue(v any) any {
	if v == nil || treeValueMarshalOpaque(v) {
		return v
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return deepCopyJSONTree(rv.Elem().Interface())
	case reflect.Map:
		if !canonicalStringKeyMap(rv.Type()) {
			return v
		}
		if rv.IsNil() {
			return nil
		}
		out := make(map[string]any, rv.Len())
		for it := rv.MapRange(); it.Next(); {
			out[it.Key().String()] = deepCopyJSONTree(it.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return nil
		}
		if rv.Kind() == reflect.Slice && canonicalByteSliceKind(rv.Type()) {
			b := make([]byte, rv.Len())
			for i := range b {
				b[i] = byte(rv.Index(i).Uint())
			}
			return b
		}
		elem := rv.Type().Elem()
		if sliceElemMarshalPositionDependent(elem) {
			return v
		}
		if elem.Kind() == reflect.String && elem != jsonNumberType &&
			!elem.Implements(jsonMarshalerType) && !elem.Implements(textMarshalerType) {
			out := make([]string, rv.Len())
			for i := range out {
				out[i] = rv.Index(i).String()
			}
			return out
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = deepCopyJSONTree(rv.Index(i).Interface())
		}
		return out
	case reflect.String:
		return rv.String()
	case reflect.Bool:
		return rv.Bool()
	case reflect.Float64:
		return rv.Float()
	case reflect.Float32:
		return float32(rv.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()
	}
	return v
}

// dedupNamedTypes walks a JSON-like schema tree (maps, slices, strings) and
// replaces a repeated, IDENTICAL named-type definition (record/enum/fixed)
// with a name reference. It tracks the enclosing namespace exactly as the
// parser does (resolveNameScope: a named definition opens its own scope),
// so:
//
//   - definitions key on their RESOLVED FULLNAME, since name equality is
//     defined on the fullname (spec, "Names"), so a.X and X coexist;
//   - a repeat dedups to a DOTTED fullname reference, which re-binds
//     position-independently. A null-namespace type has no dotted spelling, so
//     its bare reference is emitted only in a null enclosing scope — at a
//     namespaced position a bare name binds in the enclosing namespace and
//     references have no "namespace":"" escape, so that corner returns a named
//     error rather than a wrong-binding reference;
//   - two occurrences compare SCOPE-NORMALIZED, since one definition's relative
//     JSON differs by position.
//
// It also enforces one definition per fullname. Two DIFFERENT definitions
// claiming one name — two Go types, or two forms of one type, like a [16]byte
// named "uuid" used both ,uuid and plain — error rather than emit an
// unrepresentable schema. This is the single general collision check; the
// fixed/record/enum arms above need not detect it.
func dedupNamedTypes(v any, defined map[string]string, enclosingNS string) (any, error) {
	switch v := v.(type) {
	case map[string]any:
		childNS := enclosingNS
		// Is this a named type definition? An expressible fullname is the
		// registration key; fullname "" (an empty lax name with no
		// namespace) has no reference spelling, so it stays inline and
		// un-deduped, mirroring the metadata rebuild's dedup walker.
		typ, _ := v["type"].(string)
		if isNamedKind(typ) {
			full, ns := resolveNameScope(v, enclosingNS)
			childNS = ns
			if full != "" {
				cur, err := json.Marshal(normalizeSchemaScope(v, enclosingNS))
				if err != nil {
					return nil, fmt.Errorf("avro: SchemaFor: marshaling %q for the duplicate-definition check: %w", full, err)
				}
				if prev, exists := defined[full]; exists {
					if string(cur) != prev {
						return nil, fmt.Errorf("avro: SchemaFor: the Avro name %q is produced by two different "+
							"definitions (two Go types, or a logical and a plain form of one type, mapping to one "+
							"fixed/record/enum fullname); each Avro named type must be unique — rename a Go type so the "+
							"names are distinct", full)
					}
					// Identical — emit a reference. A dotted fullname
					// re-binds position-independently; a null-namespace
					// fullname is spellable only from a null enclosing
					// scope.
					if strings.Contains(full, ".") || enclosingNS == "" {
						return full, nil
					}
					return nil, fmt.Errorf("avro: SchemaFor: the null-namespace type %q recurs inside namespace %q, "+
						"where no reference can denote it (a bare name binds in the enclosing namespace, and "+
						"references have no \"namespace\":\"\" escape); give the type a namespace so a dotted "+
						"reference can name it, or build without WithNamespace", full, enclosingNS)
				}
				defined[full] = string(cur)
			}
		}
		// Recurse into children that can hold schemas. Record fields
		// resolve in the record's own namespace scope; items and values
		// belong to unnamed array/map nodes, which pass the scope through
		// (childNS == enclosingNS there). Reads and rewrites use the
		// exact reserved spelling — the only spelling Parse binds (see
		// resolveNameScope).
		//
		// Each descent is gated on the kind that BINDS the key (fields on
		// record/error, items on array, values on map), mirroring the
		// parser's kind-keyed grammar: on any other kind the key is inert
		// as-written metadata. Walking it would register definitions
		// Parse never binds — a later genuine definition of the same
		// fullname would then dedup into a dangling reference or report a
		// false duplicate — so the stray passes through untouched.
		if isRecordKind(typ) {
			if fields, ok := v["fields"].([]map[string]any); ok {
				for i := range fields {
					if _, ok := fields[i]["type"]; !ok {
						continue
					}
					nt, err := dedupNamedTypes(fields[i]["type"], defined, childNS)
					if err != nil {
						return nil, err
					}
					fields[i]["type"] = nt
				}
			}
		}
		if typ == "array" {
			if _, ok := v["items"]; ok {
				nt, err := dedupNamedTypes(v["items"], defined, childNS)
				if err != nil {
					return nil, err
				}
				v["items"] = nt
			}
		}
		if typ == "map" {
			if _, ok := v["values"]; ok {
				nt, err := dedupNamedTypes(v["values"], defined, childNS)
				if err != nil {
					return nil, err
				}
				v["values"] = nt
			}
		}
		return v, nil
	case []any: // union branches
		for i, elem := range v {
			nt, err := dedupNamedTypes(elem, defined, enclosingNS)
			if err != nil {
				return nil, err
			}
			v[i] = nt
		}
		return v, nil
	}
	return v, nil
}

// MustSchemaFor is like [SchemaFor] but panics on error.
func MustSchemaFor[T any](opts ...SchemaOpt) *Schema {
	s, err := SchemaFor[T](opts...)
	if err != nil {
		panic(err)
	}
	return s
}

// avroFullName returns the Avro fullname for a named type: the namespace
// and name joined by a dot, or the bare name when there is no namespace.
// This is the single identity a named type is registered under (seen[t])
// and referenced by, so every site that needs a named type's identity —
// the record definition, a later name reference, and the type-alias dedup
// in inferField — derives it the same way and cannot drift. (The drift it
// prevents: registering under the bare name while referencing by fullname,
// so a same-type/identical-alias pair is wrongly seen as a fresh
// definition once a namespace is configured.)
func avroFullName(namespace, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "." + name
}

// seenForm records, per visited type, the Avro name a type was registered
// under and — for a [16]byte fixed — which form emitted it. The form bit lets
// a [16]byte type whose name equals the uuid logical name ("uuid") be caught
// as a name collision when used as both a ,uuid logical and a plain fixed,
// rather than silently merged.
type seenForm struct {
	name     string
	uuidForm bool // registered as the uuid-logical fixed form
}

// inferRecord builds a schema map for a struct type. The seen map tracks
// types that have been visited so repeat references (both recursive and
// shared) emit a named reference instead of a duplicate definition.
func inferRecord(t reflect.Type, name, namespace string, seen map[reflect.Type]seenForm, customTypes []CustomType, applied appliedTypeAliases) (any, error) {
	if sf, ok := seen[t]; ok {
		return sf.name, nil
	}

	fullName := avroFullName(namespace, name)
	// Register the name before processing fields so that recursive
	// references resolve to a name reference rather than re-entering here.
	seen[t] = seenForm{name: fullName}

	fields, err := collectFields(t, make(map[reflect.Type]bool))
	if err != nil {
		return nil, err
	}

	avroFields := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		af, err := inferField(f, namespace, seen, customTypes, applied)
		if err != nil {
			return nil, fmt.Errorf("avro: field %q: %w", f.name, err)
		}
		avroFields = append(avroFields, af)
	}

	record := map[string]any{
		"type":   "record",
		"name":   name,
		"fields": avroFields,
	}
	if namespace != "" {
		record["namespace"] = namespace
	}
	return record, nil
}

type schemaField struct {
	name      string
	index     []int
	goType    reflect.Type
	tagged    bool
	aliases   []string
	typeAlias []string // aliases for the field's named type (record, enum, fixed)
	dflt      *string  // nil = no default; pointer to raw value string
	logical   string   // e.g. "timestamp-millis", "date", "uuid"
	decimal   [2]int   // [precision, scale]; zero if not decimal
}

// collectFields returns root's Avro fields: the full promoted set, then
// resolved (tagged wins over untagged, shallower wins over deeper, and a tie
// at the winning depth is an ambiguity to report).
//
// Resolving HERE rather than inside the walk is the contract, not a layout
// choice. The rule ranges over the whole collected set, since a shallower field
// declared anywhere above an embedded struct takes the name. Per recursion
// level it would decide on a partial set — calling a collision ambiguous before
// the level that resolves it is read, rejecting a type Go's own promotion
// accepts — and would read root-accumulated index paths against whatever nested
// type that level was visiting.
//
// typeFieldMapping answers the same question for encode and decode and keeps
// its resolution outside its recursion for the same reason. Agreeing on the
// rule is not enough; they must agree on where it runs, which is why
// collectFields takes no index parameter.
func collectFields(root reflect.Type, visited map[reflect.Type]bool) ([]schemaField, error) {
	raw, err := collectFieldsRaw(root, nil, visited)
	if err != nil {
		return nil, err
	}
	return resolvePromotedFields(root, raw)
}

// collectFieldsRaw walks a struct type depth-first, handling embedded structs
// and inline tags, and returns every promoted field it finds in encounter
// order — shallower first, which is what the resolution downstream relies on.
// It deliberately does NOT resolve name collisions; see collectFields.
func collectFieldsRaw(t reflect.Type, index []int, visited map[reflect.Type]bool) ([]schemaField, error) {
	if visited[t] {
		return nil, nil
	}
	// PER-PATH marking, in lockstep with typeFieldMapping's collect
	// (reflect.go): the on-path check terminates embed CYCLES, but a type
	// reachable through two SIBLING embed paths must be collected at each
	// occurrence — so the shallower one reaches the shallowest-wins dedup,
	// and a type genuinely inlined twice surfaces as the duplicate-field
	// collision it is (rather than being silently pruned). The two walkers
	// must agree, so this marking discipline is kept identical.
	visited[t] = true
	defer delete(visited, t)

	var raw []schemaField
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		idx := make([]int, len(index)+1)
		copy(idx, index)
		idx[len(index)] = i

		if sf.Anonymous {
			ft := sf.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				tag := sf.Tag.Get("avro")
				if tag == "-" {
					continue
				}
				// Same guard as the named-field path below, in the same
				// position relative to the exact-match skip: an embedded
				// struct is where "-,opt" is likeliest to be written, and
				// deferring to Avro's name grammar is not a substitute —
				// WithLaxNames can accept "-", and then the embed silently
				// becomes a field the tag asked to skip.
				if err := checkSkipDirectiveExact(sf.Name, tag); err != nil {
					return nil, err
				}
				parts, err := splitTag(tag)
				if err != nil {
					return nil, err
				}
				if parts[0] != "" {
					// Explicit name on embedded struct: treat as named field.
					// inline is incompatible with an explicit name — the name
					// says "make this a field"; inline says "flatten, no field."
					for _, p := range parts[1:] {
						if p == "inline" {
							return nil, fmt.Errorf("avro: field %s has tag %q: inline is incompatible with an explicit field name (inline flattens the embed; there is no field at this position to name)",
								sf.Name, truncForError(tag))
						}
					}
					f, err := parseSchemaTag(sf, parts, idx)
					if err != nil {
						return nil, err
					}
					raw = append(raw, f)
					continue
				}
				// Anonymous embed with empty name flattens. The embed has
				// no Avro field of its own at this position, so options
				// that apply to a field (default=, alias=, type-alias=,
				// omitzero, logical-type tags) have no target. Reject
				// rather than silently drop.
				for _, p := range parts[1:] {
					if p != "inline" {
						return nil, fmt.Errorf("avro: field %s has tag %q: inline is incompatible with option %q (the anonymous embed flattens; there is no field at this position for the option to apply to)",
							sf.Name, truncForError(tag), truncForError(p))
					}
				}
				nested, err := collectFieldsRaw(ft, idx, visited)
				if err != nil {
					return nil, err
				}
				raw = append(raw, nested...)
				continue
			}
			if !sf.IsExported() {
				continue
			}
		} else if !sf.IsExported() {
			continue
		}

		tag := sf.Tag.Get("avro")
		if tag == "-" {
			continue
		}
		// Same guard as the anonymous-embed path above; rationale lives on
		// checkSkipDirectiveExact.
		if err := checkSkipDirectiveExact(sf.Name, tag); err != nil {
			return nil, err
		}
		parts, err := splitTag(tag)
		if err != nil {
			return nil, err
		}

		// Check for inline.
		hasInline := false
		for _, p := range parts[1:] {
			if p == "inline" {
				hasInline = true
				break
			}
		}
		if hasInline {
			// inline flattens the embedded struct into the parent — the
			// embed has no field of its own at this position. Other tag
			// options apply to a field (name, default=, alias=,
			// type-alias=, omitzero, logical-type tags) and have no
			// target with inline. Reject rather than silently drop so
			// typos surface here instead of producing a schema that
			// quietly ignores the user's tag.
			if parts[0] != "" {
				return nil, fmt.Errorf("avro: field %s has tag %q: inline is incompatible with an explicit field name (inline flattens the embed; there is no field at this position to name)",
					sf.Name, truncForError(tag))
			}
			for _, p := range parts[1:] {
				if p != "inline" {
					return nil, fmt.Errorf("avro: field %s has tag %q: inline is incompatible with option %q (inline flattens the embed; there is no field at this position for the option to apply to)",
						sf.Name, truncForError(tag), truncForError(p))
				}
			}
			ft := sf.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() != reflect.Struct {
				return nil, fmt.Errorf("avro: field %s has tag %q: inline requires a struct or pointer-to-struct field type; got %s (inline flattens the embed; there is no struct here to flatten)",
					sf.Name, truncForError(tag), ft)
			}
			nested, err := collectFieldsRaw(ft, idx, visited)
			if err != nil {
				return nil, err
			}
			raw = append(raw, nested...)
			goto next
		}

		{
			f, err := parseSchemaTag(sf, parts, idx)
			if err != nil {
				return nil, err
			}
			raw = append(raw, f)
		}
	next:
	}
	return raw, nil
}

// resolvePromotedFields decides which promoted field owns each Avro name,
// over the COMPLETE set collected from t. See collectFields: the rule ranges
// over the whole set, and t is the type raw's index paths are rooted at.
func resolvePromotedFields(t reflect.Type, raw []schemaField) ([]schemaField, error) {
	// Deduplicate. Must agree with reflect.go's typeFieldMapping so
	// SchemaFor's inferred schema and the runtime field mapping pick the
	// same Go field for each Avro name. The precedence rules (documented on
	// the encode/decode field-mapping contract: "a tagged field wins over an
	// untagged one at any depth; among fields with the same tagged status,
	// the shallowest wins"):
	//   1. Tagged beats untagged at ANY depth. A tiebreaker, so not ambiguous.
	//      Runs first.
	//   2. Among same-tagged-status fields the shallower wins. Without this,
	//      dedup keeps first-seen — the DEEPER field, since nested-struct
	//      fields are appended to raw before outer ones.
	//   3. Only a same-depth, same-tagged-status collision AT THE WINNING DEPTH
	//      is ambiguous; picking one silently loses data at encode. Java's
	//      setFields rejects a duplicate (Schema.java:981) and hamba likewise.
	//      The decision is DEFERRED: a shallower field declared LATER — the
	//      common "embeds first, own fields after" layout — resolves a
	//      same-depth deep collision, so erroring the instant two deep fields
	//      collide would reject a struct whose name a shallower field
	//      unambiguously owns. typeFieldMapping and Go's promotion defer too.
	type entry struct {
		idx int
		schemaField
	}
	m := make(map[string]entry, len(raw))
	ambiguous := make(map[string][2]string) // name -> the two colliding Go field names
	for i, f := range raw {
		if existing, ok := m[f.name]; ok {
			// Tag tiebreaker first: tagged beats untagged regardless of
			// depth, so a tagged/untagged pair is resolved, never ambiguous.
			if f.tagged && !existing.tagged {
				m[f.name] = entry{i, f}
				delete(ambiguous, f.name)
				continue
			}
			if !f.tagged && existing.tagged {
				continue
			}
			// Same tagged status: the shallower field wins and clears any
			// ambiguity a deeper collision recorded; only a collision that
			// survives at the winning (shallowest) depth is genuinely
			// ambiguous, and that is decided after the full walk below.
			if len(f.index) < len(existing.index) {
				m[f.name] = entry{i, f}
				delete(ambiguous, f.name)
				continue
			}
			if len(f.index) == len(existing.index) {
				ambiguous[f.name] = [2]string{t.FieldByIndex(existing.index).Name, t.FieldByIndex(f.index).Name}
			}
			continue
		}
		m[f.name] = entry{i, f}
	}

	// A name still marked ambiguous after the full walk has two fields at its
	// winning depth with the same tagged status and no shallower resolver.
	// SchemaFor must emit every field, so it rejects here (the schema-driven
	// runtime mapper instead defers the error to a lookup of the specific
	// name). Report deterministically by raw encounter order.
	for _, f := range raw {
		if names, amb := ambiguous[f.name]; amb {
			return nil, fmt.Errorf("avro: duplicate field name %q in type %s (fields %q and %q both map to the same Avro name)",
				truncForError(f.name), t.String(), truncForError(names[0]), truncForError(names[1]))
		}
	}

	// Preserve encounter order.
	result := make([]schemaField, 0, len(m))
	for _, f := range raw {
		if e, ok := m[f.name]; ok && e.idx >= 0 {
			result = append(result, e.schemaField)
			// Mark as consumed by setting idx to -1.
			e.idx = -1
			m[f.name] = e
		}
	}
	return result, nil
}

// checkSkipDirectiveExact rejects an avro tag that begins with the "-" skip
// directive without being exactly "-" ("-,omitzero", "-foo"). The directive
// is exact-match only, so anything else is a typo: the user meant to skip but
// left options attached, or means a field literally named "-", which Avro's
// naming rules reject anyway — and which [WithLaxNames] would let through as
// a real field the tag asked to skip.
//
// Both tag-reading paths in collectFields call this immediately after their
// own exact `tag == "-"` skip: the named-field path and the anonymous
// embedded-struct path, which handles its own tag and never reaches the
// named path's checks. Sharing one implementation is what keeps the two from
// drifting on the directive's grammar, and the position after the exact-match
// skip is what keeps plain "-" skipping on both.
func checkSkipDirectiveExact(fieldName, tag string) error {
	if !strings.HasPrefix(tag, "-") {
		return nil
	}
	return fmt.Errorf("avro: field %s has tag %q: the skip directive %q is exact-match only; remove the suffix or rename the field",
		fieldName, truncForError(tag), "-")
}

// splitTag splits a struct tag value on commas, but respects parentheses
// and brackets. For example, "name,decimal(10,2),alias=[a,b]" splits into
// ["name", "decimal(10,2)", "alias=[a,b]"].
func splitTag(tag string) ([]string, error) {
	var parts []string
	var stack []rune
	start := 0
	for i, c := range tag {
		switch c {
		case '(', '[':
			stack = append(stack, c)
		case ')':
			if len(stack) == 0 || stack[len(stack)-1] != '(' {
				return nil, fmt.Errorf("unexpected %q in avro tag %q", c, tag)
			}
			stack = stack[:len(stack)-1]
		case ']':
			if len(stack) == 0 || stack[len(stack)-1] != '[' {
				return nil, fmt.Errorf("unexpected %q in avro tag %q", c, tag)
			}
			stack = stack[:len(stack)-1]
		case ',':
			if len(stack) == 0 {
				parts = append(parts, tag[start:i])
				start = i + 1
				// default= consumes the rest of the tag verbatim — its
				// value may be an arbitrary string containing unbalanced
				// brackets/parens (e.g. `default=note (a`) or commas. Stop
				// splitting and bracket-checking at this boundary so the
				// value is preserved rather than rejected as an "unclosed
				// (". parseSchemaTag's default= arm already documents that
				// it takes the rest of the tag as the last option.
				if strings.HasPrefix(tag[start:], "default=") {
					parts = append(parts, tag[start:])
					return parts, nil
				}
			}
		}
	}
	if len(stack) > 0 {
		return nil, fmt.Errorf("unclosed %q in avro tag %q", string(stack[len(stack)-1]), tag)
	}
	parts = append(parts, tag[start:])
	return parts, nil
}

// parseBracketedValues parses a tag value that is either a single value or
// a bracket-delimited list: "foo" returns ["foo"], "[foo,bar]" returns
// ["foo", "bar"]. Returns an error for empty values or empty brackets.
func parseBracketedValues(s string) ([]string, error) {
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := s[1 : len(s)-1]
		if inner == "" {
			return nil, fmt.Errorf("empty brackets in %q", s)
		}
		// Simple comma split is safe: Avro names are [A-Za-z_][A-Za-z0-9_]*
		// and cannot contain commas or brackets.
		vals := strings.Split(inner, ",")
		if slices.Contains(vals, "") {
			return nil, fmt.Errorf("empty element in %q", s)
		}
		return vals, nil
	}
	if s == "" {
		return nil, fmt.Errorf("empty value")
	}
	return []string{s}, nil
}

// parseSchemaTag parses the avro struct tag parts into a schemaField.
func parseSchemaTag(sf reflect.StructField, parts []string, index []int) (schemaField, error) {
	f := schemaField{
		name:   parts[0],
		index:  index,
		goType: sf.Type,
		tagged: parts[0] != "",
	}
	if f.name == "" {
		f.name = sf.Name
	}

	for i, opt := range parts[1:] {
		switch {
		case opt == "inline" || opt == "omitzero":
			// Already handled or recorded elsewhere.
			continue
		case strings.HasPrefix(opt, "alias="):
			vals, err := parseBracketedValues(opt[len("alias="):])
			if err != nil {
				return f, fmt.Errorf("alias: %w", err)
			}
			f.aliases = append(f.aliases, vals...)
		case strings.HasPrefix(opt, "type-alias="):
			vals, err := parseBracketedValues(opt[len("type-alias="):])
			if err != nil {
				return f, fmt.Errorf("type-alias: %w", err)
			}
			f.typeAlias = append(f.typeAlias, vals...)
		case strings.HasPrefix(opt, "default="):
			// default= must be the last option; take the rest of the tag.
			rest := strings.Join(append([]string{opt[len("default="):]}, parts[i+2:]...), ",")
			f.dflt = &rest
			return f, nil
		case strings.HasPrefix(opt, "decimal(") && strings.HasSuffix(opt, ")"):
			inner := opt[len("decimal(") : len(opt)-1]
			var p, s int
			// The trailing %s captures any content after the two integers so
			// decimal(9,2,3) / decimal(9,2x) / decimal(9,2e1) are rejected
			// rather than silently truncated to decimal(9,2): "%d,%d" alone
			// stops as soon as two integers match and ignores the rest.
			var extra string
			if n, _ := fmt.Sscanf(inner, "%d,%d%s", &p, &s, &extra); n < 2 || extra != "" {
				return f, fmt.Errorf("invalid decimal tag %q: want decimal(precision,scale)", opt)
			}
			f.decimal = [2]int{p, s}
			f.logical = "decimal"
		case opt == "uuid":
			f.logical = "uuid"
		case opt == "timestamp-millis" || opt == "timestamp-micros" || opt == "timestamp-nanos" ||
			opt == "date" ||
			opt == "time-millis" || opt == "time-micros" ||
			opt == "local-timestamp-millis" || opt == "local-timestamp-micros" || opt == "local-timestamp-nanos":
			f.logical = opt
		default:
			return f, fmt.Errorf("unknown avro tag option %q", opt)
		}
	}
	return f, nil
}

var (
	bigRatPtrType   = reflect.TypeFor[*big.Rat]()
	bigRatValueType = reflect.TypeFor[big.Rat]()
)

// appliedTypeAliases tracks type-alias values applied to each named type,
// keyed by type name. Used to accept identical aliases on later fields
// referencing the same type, while rejecting contradictory ones.
type appliedTypeAliases map[string][]string

// tagDefaultValue reads a `default=` struct tag's raw text as a JSON value,
// falling back to the text verbatim as a string.
//
// The fallback is what makes an unquoted tag (`default=hello`) usable, and it
// takes the WHOLE text: a decode that stopped after the first JSON value would
// turn `42 oops` into the number 42, silently dropping what the author wrote.
// Trailing content therefore means "this was never JSON", not "this is JSON
// plus junk" — the same verdict decimal()'s trailing-junk guard reaches.
//
// Numbers come back as their literal, which is what keeps a long default like
// 9223372036854775807 from round-tripping through float64 into
// 9.223372036854776e+18 and being rejected by the parse it feeds.
func tagDefaultValue(raw string) any {
	v, n, err := decodeSchemaAny(raw)
	if err != nil {
		return raw
	}
	for ; n < len(raw); n++ {
		switch raw[n] {
		case ' ', '\t', '\n', '\r':
		default:
			return raw
		}
	}
	return v
}

// inferField builds the Avro field definition for a single struct field.
func inferField(f schemaField, namespace string, seen map[reflect.Type]seenForm, customTypes []CustomType, applied appliedTypeAliases) (map[string]any, error) {
	fieldDef := map[string]any{
		"name": f.name,
	}

	schema, err := inferType(f.goType, f.logical, f.decimal, namespace, seen, customTypes, applied, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(f.typeAlias) > 0 {
		r := addTypeAliases(schema, f.typeAlias)
		switch {
		case r.applied:
			applied[r.refName] = f.typeAlias
		case r.refName != "":
			if prev, ok := applied[r.refName]; ok && slices.Equal(prev, f.typeAlias) {
				// Identical aliases — accept silently.
			} else if ok {
				return nil, fmt.Errorf("type-alias on field %q conflicts with type-alias already applied to type %q on an earlier field", f.name, r.refName)
			} else {
				return nil, fmt.Errorf("type-alias on field %q has no effect: type %q was already defined on an earlier field without type-alias (move the type-alias there)", f.name, r.refName)
			}
		default:
			return nil, fmt.Errorf("type-alias on field %q: type is not a named type (record, enum, or fixed)", f.name)
		}
	}
	fieldDef["type"] = schema

	if len(f.aliases) > 0 {
		fieldDef["aliases"] = f.aliases
	}
	if f.dflt != nil {
		v := tagDefaultValue(*f.dflt)
		fieldDef["default"] = v
		// A narrow Go integer kind maps to a WIDER Avro type (int8/16 and
		// uint8/16 → int; uint32 / uint → long), so a default that is a
		// valid Avro int/long but exceeds the Go field's range builds a
		// schema whose own default cannot be materialized back into the
		// field at decode-fill time. Reject it here for consistency with
		// the other Go-type/tag compatibility checks (uuid-on-wrong-kind,
		// etc.) rather than deferring to a decode-time error far from the
		// SchemaFor call.
		if err := checkIntDefaultFitsGoKind(v, f.goType); err != nil {
			return nil, fmt.Errorf("default for field %q: %w", f.name, err)
		}
	} else if union, ok := schema.([]any); ok && len(union) > 0 && isNullBranchTree(union[0]) {
		// Null-first unions (from *T or CustomType) default to null so
		// the field is backward-compatible (readers can read data written
		// before this field existed). Explicit default= overrides this.
		fieldDef["default"] = nil
	}
	return fieldDef, nil
}

// checkIntDefaultFitsGoKind verifies a numeric default fits the Go field's
// integer kind. It runs only when the Go field (after peeling pointers) is
// an integer kind; non-integer defaults and non-integer fields return nil
// and leave type compatibility to Parse's Avro-type validation.
//
// The default's integer value is extracted via defaultAsInt64 — the SAME
// lenient parser the wire default-fill path uses — so exponent / whole-
// number-float literals (e.g. "4e3") are caught here exactly as the wire
// path would catch them at decode time, instead of only the plain-integer
// forms. A value defaultAsInt64 can't read as an integer (a string,
// fractional, or > int64 default) is left to Parse, which validates it
// against the Avro type.
func checkIntDefaultFitsGoKind(v any, t reflect.Type) error {
	// Bound the peel so a cyclic pointer type (`type P *P`, whose Elem is
	// itself) terminates instead of looping forever. This is reached only
	// when a CustomType matched the field — so inferType returned before its
	// own (now-bounded) recursion — and a default is present. Past the cap the
	// type is still a pointer, so the switch below treats it as non-integer
	// and defers to Parse. Mirrors indirect/indirectAlloc's maxIndirectDepth.
	for i := 0; i < maxIndirectDepth && t.Kind() == reflect.Pointer; i++ {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := defaultAsInt64(v)
		if err != nil {
			return nil
		}
		if reflect.New(t).Elem().OverflowInt(n) {
			return fmt.Errorf("value %d overflows %s", n, t)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := defaultAsInt64(v)
		if err != nil {
			return nil
		}
		if n < 0 {
			return fmt.Errorf("value %d cannot be stored in unsigned %s", n, t)
		}
		if reflect.New(t).Elem().OverflowUint(uint64(n)) {
			return fmt.Errorf("value %d overflows %s", n, t)
		}
	}
	return nil
}

var avroPrimitives = map[string]bool{
	"null": true, "boolean": true, "int": true, "long": true,
	"float": true, "double": true, "string": true, "bytes": true,
}

// isNullBranchTree reports whether a union branch in a PRE-PARSE schema tree
// is the "null" type, in either spelling Avro admits: the bare primitive
// string "null", or an object whose "type" is "null". Props and a logicalType
// on a wrapped null are inert — Avro defines no null logical type, so nothing
// can consume either key and the branch's type and wire form are unchanged —
// so a carrier-bearing wrapped null is still a null branch.
//
// The `any`-tree mirror of [aschema.isNullBranch]. The two must agree, because
// this tree is handed straight to that parser: a branch this calls non-null and
// the parser calls null emits a shape contradicting its parsed meaning. One
// asker per representation — isNullBranch for the parsed tree, the normalized
// node kind for the compiled and metadata trees, this for pre-Parse — so a new
// asker here belongs in this function, not in a fresh bare-spelling compare.
func isNullBranchTree(v any) bool {
	switch v := v.(type) {
	case string:
		return v == "null"
	case map[string]any:
		typ, _ := v["type"].(string)
		return typ == "null"
	}
	return false
}

// typeAliasResult describes what addTypeAliases found.
type typeAliasResult struct {
	applied bool   // alias was added to a type definition
	refName string // non-empty if schema was a named type reference (definition is elsewhere)
}

// addTypeAliases walks through unions, arrays, and maps to find the
// innermost named type (record, enum, fixed) and adds aliases to it.
// For unions, aliases are added to the first named-type branch found
// (typically the only one in a ["null", T] union produced by *T).
// This supports the type-alias struct tag, which sets aliases on the
// named type referenced by a field (as opposed to alias= which sets
// aliases on the field itself).
//
// Like the composition walkers, this reads reserved keys by exact name, as the
// Parse consuming the tree binds them: only the exact lowercase spelling is
// reserved, and a Props case-variant is an ordinary custom property — so the
// walk
// descends and writes exactly the keys Parse will bind. The one consumer
// with no Parse counterpart is refName: a per-build bookkeeping key
// (applied[]), which only needs to be CONSISTENT across occurrences
// within one build — the shared exact reads make every occurrence
// resolve it identically.
func addTypeAliases(schema any, aliases []string) typeAliasResult {
	switch s := schema.(type) {
	case map[string]any:
		typ, _ := s["type"].(string)
		switch {
		case isNamedKind(typ):
			appendTypeAliasValues(s, aliases)
			// refName must be the type's fullname (namespace + name) — the
			// same identity inferRecord registers in seen[t] and a later
			// field's name reference resolves to. The definition carries
			// name and namespace as separate keys; join them so the dedup
			// in inferField keys both the defining and referencing fields
			// by one identity. (A namespace-less type's fullname is its
			// bare name, so fixed types and no-namespace records are
			// unchanged.)
			name, _ := s["name"].(string)
			ns, _ := s["namespace"].(string)
			return typeAliasResult{applied: true, refName: avroFullName(ns, name)}
		case typ == "array":
			if items, ok := s["items"]; ok {
				return addTypeAliases(items, aliases)
			}
		case typ == "map":
			if values, ok := s["values"]; ok {
				return addTypeAliases(values, aliases)
			}
		}
	case []any: // union
		var best typeAliasResult
		for _, branch := range s {
			r := addTypeAliases(branch, aliases)
			if r.applied {
				return r
			}
			if r.refName != "" {
				best = r
			}
		}
		return best
	case string:
		// A string is either a primitive type name ("int", "long", ...)
		// or a named type reference ("Inner"). Named references mean the
		// type was already defined on an earlier field.
		if !avroPrimitives[s] {
			return typeAliasResult{refName: s}
		}
	}
	return typeAliasResult{}
}

// appendTypeAliasValues merges the tag's aliases into the type's existing
// exact-key "aliases" attribute — the only spelling Parse binds (a
// Props-carried case-variant is an ordinary custom property, left
// untouched beside the real attribute). The existing value is []string or
// []any on every route: freshly-inferred literals build []string, and the
// render boundary canonicalizes every caller-typed array shape into
// []string/[]any fresh copies (deepCopyJSONTree/canonicalizeTreeValue —
// so the appends below never write into a caller-owned backing array).
// The one shape that reaches here outside those two is a marshal-opaque
// value (its own MarshalJSON/MarshalText): left untouched for Parse to
// read from its marshal, the documented opacity residual — a merge would
// require marshaling it early, and its output is its author's contract.
func appendTypeAliasValues(s map[string]any, aliases []string) {
	v, ok := s["aliases"]
	if !ok {
		s["aliases"] = append([]string(nil), aliases...)
		return
	}
	switch existing := v.(type) {
	case []string:
		s["aliases"] = append(existing, aliases...)
	case []any:
		merged := make([]any, 0, len(existing)+len(aliases))
		merged = append(merged, existing...)
		for _, a := range aliases {
			merged = append(merged, a)
		}
		s["aliases"] = merged
	}
}

// baseTypeForLogical returns the underlying Avro type required by the
// given logical type per the Avro 1.12 spec. Used by SchemaFor's
// inferType to produce schemas that validateLogical (schema.go) will
// accept regardless of the Go source type's natural Avro mapping —
// e.g. time-millis MUST annotate int even when the Go field is
// time.Time (whose default mapping is long).
func baseTypeForLogical(logical, fallback string) string {
	switch logical {
	case "date", "time-millis":
		return "int"
	case "time-micros",
		"timestamp-millis", "timestamp-micros", "timestamp-nanos",
		"local-timestamp-millis", "local-timestamp-micros", "local-timestamp-nanos":
		return "long"
	}
	return fallback
}

// inferType returns the Avro schema for a Go type.
//
// ptrChain is the number of CONSECUTIVE pointer levels already unwrapped to
// reach t, reset to 0 at every record-field / array-item / map-value boundary
// (the codec calls indirect/indirectAlloc fresh on each such leaf value). The
// pointer arm caps it: the codec's indirect/indirectAlloc (reflect.go) unwrap
// at most maxIndirectDepth pointer levels, so SchemaFor must refuse a deeper
// chain at BUILD time rather than emit a ["null",T] the codec then rejects with
// errIndirectDeep — a build-accepts/encode-rejects asymmetry. This also
// terminates a cyclic non-struct pointer type (type P *P) at the cap instead
// of recursing to the maxDepth ceiling.
func inferType(t reflect.Type, logical string, decimal [2]int, namespace string, seen map[reflect.Type]seenForm, customTypes []CustomType, applied appliedTypeAliases, depth, ptrChain int) (any, error) {
	// A recursive non-struct Go type — `type S []S`, `type P *P`,
	// `type M map[string]M`, or a long-enough pointer/slice/map chain — has
	// a cyclic type graph, and the pointer/slice/map arms below recurse on
	// the element type. Struct cycles terminate via seen[t] (a struct
	// registers its name before recursing into its fields), but a non-struct
	// type registers no name, so the recursion would run until the goroutine
	// stack overflows and the process dies. Bound it at maxDepth — the same
	// ceiling the wire pipeline enforces — so such a type returns a clean
	// error. depth resets to 0 at each record-field boundary (inferField),
	// so a chain of distinct nested structs stays bounded by seen, not depth,
	// and only an unbroken non-struct chain accrues depth here.
	if depth >= maxDepth {
		return nil, fmt.Errorf("avro: type %s nests too deeply or is recursive (exceeds depth %d); a recursive non-struct type such as `type T []T`, `*T`, or `map[string]T` has no Avro schema representation", t, maxDepth)
	}
	// Check custom types before anything else (including pointer unwrapping).
	for _, ct := range customTypes {
		if ct.GoType != nil && ct.GoType == t {
			// The custom supplies the field's schema, so a logical-type tag
			// on the field has nothing to apply to. Accepting it would
			// silently drop the user's tag — the lying-schema outcome the
			// logical-tag strictness rejects everywhere else (the
			// avro.Duration and uuid/decimal wrong-kind arms) — so reject
			// with the remedy: the logical type belongs on the CustomType.
			if logical != "" {
				return nil, fmt.Errorf("avro: a CustomType is registered for %s and supplies the schema; the field's logical-type tag %q has no effect — remove the tag, or set LogicalType/Schema on the CustomType", t, logical)
			}
			if ct.Schema != nil {
				tree, err := renderCustomSchemaTree(ct.Schema)
				if err != nil {
					return nil, err
				}
				// The subtree is rendered relative to the null namespace;
				// embedding it inside a namespaced tree must not let
				// namespace inheritance capture its null-namespace types
				// (see pinCustomSchemaScope). Only a namespaced SchemaFor
				// scope can capture, so the null-scope build leaves the
				// tree exactly as rendered.
				if namespace != "" {
					pinCustomSchemaScope(tree)
				}
				return tree, nil
			}
			if ct.AvroType == "" {
				return nil, fmt.Errorf("avro: CustomType for %s has no AvroType or Schema; cannot infer schema (set AvroType or Schema for SchemaFor)", t)
			}
			schema := map[string]any{"type": ct.AvroType}
			if ct.LogicalType != "" {
				schema["logicalType"] = ct.LogicalType
			}
			return schema, nil
		}
	}

	// Pointer → nullable union. A pointer means "nullable", and the codecs
	// treat a pointer chain as a single nullable level (indirect /
	// indirectAlloc). Recurse one level — so an intermediate pointer type
	// can still match a registered CustomType — then collapse: if the
	// inner already inferred to a null-first union (a deeper pointer like
	// **T, or a CustomType whose schema is ["null", …]), return it
	// unwrapped rather than nesting. Avro forbids a union immediately
	// inside a union, so wrapping each level would emit an unparseable
	// ["null", ["null", T]].
	if t.Kind() == reflect.Pointer {
		// The codec unwraps at most maxIndirectDepth consecutive pointer levels
		// (indirect/indirectAlloc both accept a chain bottoming at a non-pointer
		// base within that cap). A deeper chain would build a valid ["null",T]
		// here but fail Encode of a non-nil value with errIndirectDeep, so refuse
		// it at build time. Mirrors checkIntDefaultFitsGoKind's maxIndirectDepth
		// pointer-peel bound.
		if ptrChain >= maxIndirectDepth {
			return nil, fmt.Errorf("avro: %s: pointer chain nests deeper than the codec supports (it unwraps at most %d consecutive pointer levels); flatten the indirection, register a CustomType, or define the schema explicitly", t, maxIndirectDepth)
		}
		inner, err := inferType(t.Elem(), logical, decimal, namespace, seen, customTypes, applied, depth+1, ptrChain+1)
		if err != nil {
			return nil, err
		}
		if u, ok := inner.([]any); ok && len(u) > 0 && isNullBranchTree(u[0]) {
			return u, nil
		}
		return []any{"null", inner}, nil
	}

	// Logical types for known Go types. The base type is determined by
	// the logical's spec-required underlying Avro type (NOT by the Go
	// source type), so e.g. `time.Time` tagged time-millis correctly
	// emits {int, time-millis} and `time.Duration` tagged
	// timestamp-millis correctly emits {long, timestamp-millis}.
	// Reject non-time logicals (uuid, decimal) on time types — they
	// would emit an invalid {long, uuid} or {long, decimal} schema
	// which Parse soft-drops, silently losing the user's tag.
	inferTimeLike := func(defaultLogical string) (any, error) {
		lt := logical
		if lt == "" {
			lt = defaultLogical
		}
		switch lt {
		case "date", "time-millis", "time-micros",
			"timestamp-millis", "timestamp-micros", "timestamp-nanos",
			"local-timestamp-millis", "local-timestamp-micros", "local-timestamp-nanos":
			// time-related logical, OK.
		default:
			return nil, fmt.Errorf("avro: %s does not support logical type %q; use a time or date logical type", t, lt)
		}
		return map[string]any{"type": baseTypeForLogical(lt, "long"), "logicalType": lt}, nil
	}
	switch t {
	case timeType:
		return inferTimeLike("timestamp-millis")
	case durationType:
		return inferTimeLike("time-millis")

	case avroDurationType:
		// avro.Duration is the dedicated Go type for the Avro duration logical:
		// a fixed(12) whose bytes are little-endian months/days/milliseconds.
		// Recognition is BY TYPE (avro.Duration is a struct, so this case must
		// fire before the struct→record path below, which would otherwise
		// decompose its exported uint32 fields into a record) — there is no
		// "duration" tag option and none is needed: unlike *big.Rat→decimal,
		// the duration logical carries no parameters, so the bare type
		// suffices. A non-empty logical here is therefore always a tag the
		// user mis-attached (uuid / decimal / a time logical); reject it
		// rather than silently emitting the duration schema and dropping the
		// tag, matching the strict-reject posture of the time/uuid/decimal
		// arms. The fixed name "duration" is safe even when a plain
		// `type duration [12]byte` is also present: dedupNamedTypes rejects any
		// Avro name claimed by two different definitions.
		if logical != "" {
			return nil, fmt.Errorf("avro: avro.Duration maps to the duration logical type (a fixed(12)); it does not support logical type %q — remove the tag", logical)
		}
		return map[string]any{
			"type":        "fixed",
			"name":        "duration",
			"size":        12,
			"logicalType": "duration",
		}, nil

	case jsonNumberType:
		// The jsonNumberType exclusion, at schema inference: the Kind switch
		// below would emit an Avro "string", the one type the codec is
		// guaranteed to reject for this Go type — a build-accepts/
		// encode-rejects deferred failure. Reject up front instead,
		// matching the uuid/decimal/time strictness. (A registered CustomType
		// for json.Number still works — the loop above runs first; a NAMED
		// alias `type N json.Number` is a distinct reflect.Type that the
		// codec treats as a plain string and is unaffected.)
		return nil, fmt.Errorf("avro: json.Number has no single Avro type for SchemaFor; use a concrete Go numeric type (int32/int64/float64), string, or a CustomType")

	case bigRatPtrType, bigRatValueType:
		if logical == "" || (logical == "decimal" && decimal == [2]int{}) {
			return nil, fmt.Errorf("*big.Rat requires explicit decimal(precision,scale) tag")
		}
		if logical != "decimal" {
			return nil, fmt.Errorf("avro: *big.Rat / big.Rat does not support logical type %q; use decimal(precision,scale)", logical)
		}
		return map[string]any{
			"type":        "bytes",
			"logicalType": "decimal",
			"precision":   decimal[0],
			"scale":       decimal[1],
		}, nil
	}

	// decimal logical type requires *big.Rat or big.Rat (handled above).
	// Any other Go type carrying ",decimal(p,s)" produces a schema that
	// wouldn't reflect the user's intent — the encoder for decimal-on-bytes
	// expects *big.Rat input, not raw bytes / int / string — so reject at
	// SchemaFor time rather than silently dropping the tag.
	if logical == "decimal" {
		return nil, fmt.Errorf("avro: decimal logical type requires *big.Rat or big.Rat; got %s", t)
	}

	// UUID: spec wire form is either string or fixed(16). Accept Go
	// string kind and text-marshaler types as string; [16]byte goes
	// through the Array case below for fixed(16). Reject other Go
	// types — they would produce a schema that lies about the field's
	// Go type and cause Encode to fail at runtime far from here.
	if logical == "uuid" {
		isArr16 := t.Kind() == reflect.Array && t.Elem().Kind() == reflect.Uint8 && t.Len() == 16
		if !isArr16 {
			enc, dec := implementsTextMarshaler(t), implementsTextUnmarshaler(t)
			stringKind := t.Kind() == reflect.String
			byteSlice := t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
			switch {
			case stringKind, byteSlice && (enc || dec), enc && dec:
				// Round-trips as a uuid string: a string kind; a []byte slice
				// carrying a text method; or any type implementing BOTH text
				// directions. (The same round-trip rule as the plain string arm.)
				return map[string]any{"type": "string", "logicalType": "uuid"}, nil
			case enc:
				return nil, fmt.Errorf("avro: uuid logical type on %s: it implements TextMarshaler/AppendText but not TextUnmarshaler, so a uuid string schema could encode it but not decode into it — implement both text directions or use Go string / [16]byte", t)
			case dec:
				return nil, fmt.Errorf("avro: uuid logical type on %s: it implements TextUnmarshaler but not TextMarshaler/AppendText, so a uuid string schema could decode into it but not encode it — implement both text directions or use Go string / [16]byte", t)
			default:
				return nil, fmt.Errorf("avro: uuid logical type requires Go string, [16]byte, or a text marshaler type; got %s", t)
			}
		}
	}

	// Integer-wire logical types (date / time-millis on int wire;
	// time-micros / timestamp-* / local-timestamp-* on long wire)
	// attached to a plain Go integer field. The user opted into the
	// logical with the tag; honor it by emitting the wire+logical
	// schema. The Go field must have a natural Avro wire type that
	// matches the logical's required wire — otherwise the encoder
	// would silently widen or narrow, hiding the user's intent and
	// producing a schema whose annotation may be soft-dropped at
	// Parse if the wire doesn't match.
	if logical != "" {
		wire := baseTypeForLogical(logical, "")
		if wire != "" {
			compat := false
			switch wire {
			case "int":
				switch t.Kind() {
				case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16:
					compat = true
				}
			case "long":
				switch t.Kind() {
				case reflect.Int, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Uint:
					compat = true
				}
			}
			if !compat {
				return nil, fmt.Errorf("avro: logical type %q requires a Go integer field whose natural Avro wire type is %q; got %s", logical, wire, t)
			}
			return map[string]any{"type": wire, "logicalType": logical}, nil
		}
	}

	// A type implementing text interfaces is inferred as string — but only when
	// a string schema ROUND-TRIPS for it. The codec encodes a string from
	// TextMarshaler/AppendText (or a string / []byte kind) and decodes one via
	// TextUnmarshaler (or a string / []byte kind). A string-kind or []byte-slice
	// type round-trips regardless of which text methods it has (the kind itself
	// covers the missing direction); any OTHER type round-trips only if it
	// implements BOTH an encode-side method AND TextUnmarshaler. A non-string
	// type implementing exactly one direction would yield a one-directional
	// "string" schema whose unsupported direction fails at Encode/Decode far
	// from here, and SchemaFor cannot reliably guess which direction the caller
	// wants — so it refuses, the same strict-reject posture as the logical-type
	// tags above (never emit a schema that lies about the Go type).
	//
	// EXCEPTION: a ,uuid-tagged [16]byte is a fixed(16) uuid handled by the
	// Array case below — the codec trusts its raw bytes and never consults a
	// text method for a uuid-on-fixed value (see [Schema.Decode]'s uuid-on-fixed
	// contract); downgrading it to "string" would silently drop the fixed(16)
	// shape and the uuid logical type.
	uuidArr16 := logical == "uuid" && t.Kind() == reflect.Array &&
		t.Elem().Kind() == reflect.Uint8 && t.Len() == 16
	if !uuidArr16 {
		enc, dec := implementsTextMarshaler(t), implementsTextUnmarshaler(t)
		if enc || dec {
			kindFallback := t.Kind() == reflect.String ||
				(t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8)
			switch {
			case kindFallback || (enc && dec):
				return "string", nil
			case enc:
				return nil, fmt.Errorf("avro: type %s implements TextMarshaler/AppendText but not TextUnmarshaler: a string schema could encode it but not decode into it — implement TextUnmarshaler too, use a string/[]byte-based Go type, or define the schema explicitly", t)
			default:
				return nil, fmt.Errorf("avro: type %s implements TextUnmarshaler but not TextMarshaler/AppendText: a string schema could decode into it but not encode it — implement an encode-side text method (TextMarshaler/AppendText) too, use a string/[]byte-based Go type, or define the schema explicitly", t)
			}
		}
	}

	// inferArray emits the array schema for a non-byte slice/array element
	// type. Shared by the Slice and Array arms below so the items-recursion
	// + wrapping shape stays in one place.
	inferArray := func(elem reflect.Type) (any, error) {
		// Array/slice element: the codec encodes each element with a fresh
		// indirect call, so the pointer chain resets (ptrChain=0).
		items, err := inferType(elem, "", [2]int{}, namespace, seen, customTypes, applied, depth+1, 0)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "array", "items": items}, nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return "boolean", nil
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return "int", nil
	case reflect.Uint8, reflect.Uint16:
		return "int", nil
	case reflect.Int, reflect.Int64, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return "long", nil
	case reflect.Float32:
		return "float", nil
	case reflect.Float64:
		return "double", nil
	case reflect.String:
		return "string", nil

	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "bytes", nil
		}
		return inferArray(t.Elem())

	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			// The Avro name a []byte array gets depends on the uuid tag:
			// "uuid" (with the logical type) when ,uuid-tagged, otherwise
			// t.Name() / "fixed_N". So the SAME Go type can legitimately
			// appear under two different Avro names — e.g. used once
			// ,uuid-tagged and once plain — which are distinct Avro types
			// (they differ by logicalType). seen[t] records the name an
			// earlier occurrence emitted; emit a name reference ONLY when
			// this occurrence's name matches it. A different-named form of
			// the same type emits its own full definition. The name-match
			// guard keeps the same-form dedup (so a later field can
			// reference an earlier definition that a type-alias= mutated)
			// while letting the two forms each define their own fixed.
			isUUIDForm := logical == "uuid" && t.Len() == 16
			var name string
			var def map[string]any
			if isUUIDForm {
				name = "uuid"
				def = map[string]any{
					"type":        "fixed",
					"name":        "uuid",
					"size":        16,
					"logicalType": "uuid",
				}
			} else {
				name = t.Name()
				if name == "" {
					name = fmt.Sprintf("fixed_%d", t.Len())
				}
				def = map[string]any{
					"type": "fixed",
					"name": name,
					"size": t.Len(),
				}
			}
			// Reference an earlier definition only for the SAME type in the SAME
			// form (same Avro name AND same logical-vs-plain form). When the form
			// differs but the name coincides (a [16]byte named exactly "uuid"
			// used both ,uuid and plain), emit this form's own definition; the
			// name collision is then caught uniformly by dedupNamedTypes, which
			// rejects any Avro name claimed by two different definitions.
			if prev, ok := seen[t]; ok && prev.name == name && prev.uuidForm == isUUIDForm {
				return name, nil
			}
			seen[t] = seenForm{name: name, uuidForm: isUUIDForm}
			return def, nil
		}
		return inferArray(t.Elem())

	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string, got %s", t.Key())
		}
		// Map value: encoded with a fresh indirect call per entry, so the
		// pointer chain resets (ptrChain=0).
		values, err := inferType(t.Elem(), "", [2]int{}, namespace, seen, customTypes, applied, depth+1, 0)
		if err != nil {
			return nil, err
		}
		return map[string]any{"type": "map", "values": values}, nil

	case reflect.Struct:
		name := t.Name()
		if name == "" {
			return nil, fmt.Errorf("anonymous struct types are not supported; use a named type")
		}
		return inferRecord(t, name, namespace, seen, customTypes, applied)

	default:
		return nil, fmt.Errorf("unsupported Go type %s", t)
	}
}
