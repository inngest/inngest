package avro

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// parseSchemaTree decodes a schema JSON string into the aschema parse tree in a
// single O(n) pass. A json.Unmarshaler on aschema/afield would instead make the
// stdlib decoder re-scan each nested subtree to delimit it, O(depth*size);
// decoding once into a generic tree and walking that is O(n).
//
// Scalar leaves (size, scale, precision, default) re-marshal their small
// decoded value and reuse the stdlib decode, so their accept/reject behavior
// matches the typed decode by construction. That re-marshal is O(leaf) and
// stays O(n) summed.
func parseSchemaTree(schema string) (*aschema, error) {
	// decodeSchemaAnyStrict preserves every number as its literal, which is
	// what the arms below re-marshal into o.Default / f.Default and what the
	// size / precision / scale reads decode; see that decoder for the two
	// silent failures a resolving decode produces here.
	v, err := decodeSchemaAnyStrict(schema)
	if err != nil {
		return nil, boundJSONErrorEcho(err)
	}
	var s aschema
	if err := aschemaFromAny(v, &s, nil); err != nil {
		return nil, err
	}
	return &s, nil
}

// aschemaFromAny populates s from a generic JSON value: a string is a
// primitive / name reference, an array is a union, an object is a complex
// type. Mirrors the dispatch the former aschema.UnmarshalJSON did on the
// first byte.
func aschemaFromAny(v any, s *aschema, memo strayShapeMemo) error {
	switch t := v.(type) {
	case string:
		s.primitive = t
		return nil
	case []any:
		s.union = make([]aschema, len(t))
		for i := range t {
			if err := aschemaFromAny(t[i], &s.union[i], memo); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		o, err := aobjectFromMap(t, memo)
		if err != nil {
			return err
		}
		s.object = o
		return nil
	default:
		return errors.New("invalid schema")
	}
}

// schemaTypeMismatch mirrors the error encoding/json produced when a
// schema key held the wrong JSON type (e.g. a non-string "type"), so
// downstream behavior and messages stay equivalent.
func schemaTypeMismatch(key, want string) error {
	return fmt.Errorf("invalid schema: %q must be a JSON %s", key, want)
}

func aobjectFromMap(m map[string]any, memo strayShapeMemo) (o *aobject, err error) {
	// memo is set ONLY on the metadata walker's stray-shape validity path
	// (strayBodyShapeOKMemo), never on a real parse. There the caller
	// discards the built aobject and only reads whether this subtree is a
	// valid schema shape, so a subtree already validated returns its cached
	// verdict without the O(subtree) rebuild — turning the walker's repeated
	// per-ancestor-level checks over a nested schema from O(depth^2) back
	// into one linear pass. The verdict is recorded by this same decode, so
	// it is identical to a fresh strayBodyShapeOK.
	if memo != nil {
		p := reflect.ValueOf(m).Pointer()
		if valid, ok := memo[p]; ok {
			if valid {
				return &aobject{}, nil
			}
			return nil, errStrayShapeCached
		}
		defer func() { memo[p] = err == nil }()
	}
	o = &aobject{}

	if v, ok := m["type"]; ok {
		ts, ok := v.(string)
		if !ok {
			return nil, schemaTypeMismatch("type", "string")
		}
		o.Type = ts
	}
	// Structural/naming keys shape-validate ONLY where the kind binds
	// them. On a non-binding kind a malformed body cannot define, scope,
	// or bind anything: the arm leaves the aobject field unset and the
	// extra loop below routes the raw value to props verbatim
	// (schemaReservedKeyForObject's shape-conditional arm) — the same
	// treatment unconsumed precision/scale already get, and the same
	// accept-and-ignore posture Java (SCHEMA_RESERVED skip,
	// Schema.java:175-176) and fastavro take. A shape-OK body still
	// parses into the aobject field even on a non-binding kind (the
	// documented as-written structural surfacing).
	nameIsString := false
	if v, ok := m["name"]; ok {
		if ns, ok := v.(string); ok {
			o.Name = ns
			nameIsString = true
		} else if strayKeyBinds(o.Type, "name") {
			return nil, schemaTypeMismatch("name", "string")
		}
	}
	if v, ok := m["namespace"]; ok {
		if ns, ok := v.(string); ok {
			o.Namespace = &ns
		} else if strayKeyBinds(o.Type, "namespace") {
			return nil, schemaTypeMismatch("namespace", "string")
		}
	}
	if ss, ok, err := stringSliceFrom(m, "symbols"); err != nil {
		if strayKeyBinds(o.Type, "symbols") {
			return nil, err
		}
	} else if ok {
		o.Symbols = ss
	}
	if ss, ok, err := stringSliceFrom(m, "aliases"); err != nil {
		if strayKeyBinds(o.Type, "aliases") {
			return nil, err
		}
	} else if ok {
		o.Aliases = ss
	}
	if v, ok := m["items"]; ok {
		it := &aschema{}
		if err := aschemaFromAny(v, it, memo); err != nil {
			if strayKeyBinds(o.Type, "items") {
				return nil, err
			}
		} else {
			o.Items = it
		}
	}
	if v, ok := m["values"]; ok {
		vs := &aschema{}
		if err := aschemaFromAny(v, vs, memo); err != nil {
			if strayKeyBinds(o.Type, "values") {
				return nil, err
			}
		} else {
			o.Values = vs
		}
	}
	if v, ok := m["size"]; ok {
		if l, err := decodeLaxInt("size", v); err == nil {
			o.Size = &l
		} else if strayKeyBinds(o.Type, "size") {
			return nil, err
		}
	}
	// logicalType is read only when string-typed. A non-string value can
	// never name a logical, so it is inert metadata riding to props
	// verbatim (schemaReservedKeyForObject's string-conditional arm) —
	// matching Java (only textual logicalType props are read;
	// LogicalTypes.fromSchemaImpl via Schema.getProp), fastavro, and
	// goavro, and matching the treatment an unknown STRING logical
	// already gets here (inert, surfaced as-written).
	if ls, ok := m["logicalType"].(string); ok {
		o.Logical = ls
	}
	// precision/scale are consumed (and shape-validated) only on a
	// recognized decimal carrier; everywhere else a malformed value is
	// inert and rides to props like any other unconsumed placement.
	if p, err := intPtrFrom(m, "scale"); err != nil {
		if decimalConsumesPrecisionScale(o.Type, o.Logical) {
			return nil, err
		}
	} else {
		o.Scale = p
	}
	if p, err := intPtrFrom(m, "precision"); err != nil {
		if decimalConsumesPrecisionScale(o.Type, o.Logical) {
			return nil, err
		}
	} else {
		o.Precision = p
	}
	if v, ok := m["default"]; ok {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		o.Default = json.RawMessage(raw)
	}
	if v, ok := m["fields"]; ok {
		if fs, ok := v.([]any); ok {
			fields := make([]afield, len(fs))
			ferr := error(nil)
			for i := range fs {
				if err := afieldFromAny(fs[i], &fields[i], memo); err != nil {
					ferr = err
					break
				}
			}
			if ferr == nil {
				o.Fields = fields
			} else if strayKeyBinds(o.Type, "fields") {
				return nil, ferr
			}
		} else if strayKeyBinds(o.Type, "fields") {
			return nil, schemaTypeMismatch("fields", "array")
		}
	}
	// Extra (non-reserved) properties, numerically normalized by
	// normalizeJSONValue. precision/scale are extra everywhere but a
	// recognized decimal carrier, so a CustomType callback's SchemaNode
	// surfaces a stray pair in Props exactly as Root() does.
	//
	// Route on the arms' RECORDED stray-body verdict, not a fresh check: the
	// arms above set o.Items/o.Values/o.Fields exactly when the body parsed as
	// the key's shape. A second decode re-enters aschemaFromAny, which routes
	// its own stray keys, so two decodes per level compound to O(2^depth) on a
	// nested-stray schema.
	shapeOK := o.strayShapeRecorded(nameIsString)
	for k, v := range m {
		if schemaReservedKeyForObject(k, v, o.Type, o.Logical, shapeOK) {
			continue
		}
		if o.extra == nil {
			o.extra = make(map[string]any)
		}
		o.extra[k] = normalizeJSONValue(v)
	}
	return o, nil
}

func afieldFromAny(v any, f *afield, memo strayShapeMemo) error {
	m, ok := v.(map[string]any)
	if !ok {
		return errors.New("invalid record field: must be a JSON object")
	}
	if v, ok := m["name"]; ok {
		ns, ok := v.(string)
		if !ok {
			return schemaTypeMismatch("name", "string")
		}
		f.Name = ns
	}
	if v, ok := m["order"]; ok {
		os, ok := v.(string)
		if !ok {
			return schemaTypeMismatch("order", "string")
		}
		f.Order = os
		f.orderSet = true
	}
	if ss, ok, err := stringSliceFrom(m, "aliases"); err != nil {
		return err
	} else if ok {
		f.Aliases = ss
	}
	// String-conditional like the type-object arm: a non-string
	// logicalType is an inert custom property (it lifts nothing onto the
	// field's type and rides in the field's metadata Props verbatim).
	if ls, ok := m["logicalType"].(string); ok {
		f.Logical = ls
	}
	// precision/scale shape verdicts are RECORDED here and decided after
	// the type parses: whether the pair is consumed depends on the
	// field-level logicalType and the lift target's kind (and a flat
	// field routes the raw pair into the lifted type object, where the
	// type-level gate rules) — none of which is known yet. An unconsumed
	// malformed body is inert field metadata riding to props verbatim; a
	// consumed one rejects loudly below, from these recorded errors.
	var scaleErr, precisionErr error
	if p, err := intPtrFrom(m, "scale"); err != nil {
		scaleErr = err
	} else {
		f.Scale = p
	}
	if p, err := intPtrFrom(m, "precision"); err != nil {
		precisionErr = err
	} else {
		f.Precision = p
	}
	if v, ok := m["default"]; ok {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		f.Default = json.RawMessage(raw)
	}
	if v, ok := m["type"]; ok {
		f.Type = &aschema{}
		if err := aschemaFromAny(v, f.Type, memo); err != nil {
			return err
		}
	}

	// Flat ("linkedin/goavro") field format and field-level logicalType
	// lift, mirroring the former afield.UnmarshalJSON. When "type" is a
	// string naming a complex kind and that kind's defining key (symbols
	// / items / values / fields / size) sits alongside the field's own
	// keys, lift those into a nested type object.
	if f.Type != nil && f.Type.primitive != "" && flatFieldNeedsLift(m, f.Type.primitive) {
		return f.liftFlatFieldType(m, f.Type.primitive)
	}
	// The decimal lift is the only field-level consumer of precision/
	// scale. Where it consumes (logicalType "decimal" with a bytes/fixed
	// lift target), a malformed body must reject LOUDLY rather than be
	// treated as absent: scale is optional (spec default 0), so silently
	// dropping a malformed scale beside a valid precision would parse as
	// decimal(p,0) — a silent wire-semantics change. Everywhere else the
	// pair is inert and the recorded error is deliberately dropped — the
	// key rides to the field's metadata props verbatim, like any custom
	// property (Java's FIELD_RESERVED excludes the pair, so its parser
	// never validates them; fastavro preserves them verbatim).
	if f.fieldDecimalLiftConsumesPrecisionScale() {
		if precisionErr != nil {
			return fmt.Errorf("invalid record field %q: %w", "precision", precisionErr)
		}
		if scaleErr != nil {
			return fmt.Errorf("invalid record field %q: %w", "scale", scaleErr)
		}
	}
	f.liftFieldLogicalIntoType()
	return nil
}

// flatFieldNeedsLift reports whether the field JSON object m, whose "type"
// attribute is the bare string tp, is written in the flat (goavro-style)
// field format: tp names a complex kind and that kind's defining key
// (symbols / items / values / fields / size) sits alongside the field's own
// keys. "error" is the record alias, defined by the "fields" key like
// "record".
//
// Shared by the wire parser (afieldFromAny) and, via walkNodeChildren
// (schema_walk.go), every JSON-map walker — the SchemaCache
// self-containment walkers and the Root() metadata tree: all must lift
// the SAME fields, or a walker would describe a different schema than
// the one that encodes — sharing the predicate makes the agreement
// structural.
func flatFieldNeedsLift(m map[string]any, tp string) bool {
	switch tp {
	case "enum", "array", "map", "record", "error", "fixed":
	default:
		return false
	}
	for key, forType := range afieldComplexKeys {
		if _, ok := m[key]; ok && (forType == tp || (key == "fields" && tp == "error")) {
			return true
		}
	}
	return false
}

// flatLiftTypeMap builds the nested type object's JSON map from a flat
// field's own keys: "default" and "order" are field-only and never
// propagate; "aliases" belongs to the field (flat-format aliases are field
// aliases); "name" and "namespace" propagate only for named kinds;
// everything else — the defining key, "type" itself, logicalType /
// precision / scale, doc, and custom properties — moves into the type.
//
// Shared by the wire parser (liftFlatFieldType) and the JSON-map walkers'
// flatField callbacks (collectTreeDefs, metadataField's callers) so the
// sides cannot drift on WHAT the lift routes; flatFieldNeedsLift is the
// shared WHEN.
func flatLiftTypeMap(m map[string]any, tp string) map[string]any {
	named := isNamedKind(tp)
	typeMap := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "default", "order":
			// Field-only keys, do not propagate.
		case "aliases":
			// Flat-format aliases belong to the field, not the type.
		case "name", "namespace":
			if named {
				typeMap[k] = v
			}
		default:
			typeMap[k] = v
		}
	}
	return typeMap
}

// liftFlatFieldType builds the field's nested type object from the field's
// own JSON keys (excluding field-only keys), for the flat field format.
// Mirrors the former afield.UnmarshalJSON flat-form branch. The key routing
// lives in flatLiftTypeMap (shared with the metadata walker): logicalType /
// precision / scale flow into the type object (they are not field-only),
// so the field-level copies are cleared afterward.
func (f *afield) liftFlatFieldType(m map[string]any, tp string) error {
	// The lifted type is a freshly constructed map (flatLiftTypeMap), not a
	// node of the caller's tree, so no shape memo applies — pass nil.
	o, err := aobjectFromMap(flatLiftTypeMap(m, tp), nil)
	if err != nil {
		return err
	}
	f.Type = &aschema{object: o}
	f.Logical, f.Scale, f.Precision = "", nil, nil
	return nil
}

// stringSliceFrom reads m[key] as a []string. The second return reports
// presence; a present non-array or non-string element is an error,
// matching encoding/json's []string decode.
func stringSliceFrom(m map[string]any, key string) ([]string, bool, error) {
	v, ok := m[key]
	if !ok {
		return nil, false, nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil, false, schemaTypeMismatch(key, "array")
	}
	out := make([]string, len(arr))
	for i := range arr {
		s, ok := arr[i].(string)
		if !ok {
			return nil, false, schemaTypeMismatch(key, "array of strings")
		}
		out[i] = s
	}
	return out, true, nil
}

// jsonNullBody reports whether a decoded JSON body is the null literal.
//
// null is the one body a typed decode accepts in silence: encoding/json
// documents that unmarshaling it into a non-pointer, non-interface, non-map
// destination "has no effect on the value and produces no error". A reader
// deciding PRESENCE by whether the decode failed therefore reads a
// present-but-unreadable attribute as ABSENT and keeps the zero.
//
// That zero is not neutral. A fixed of size 0 is a usable schema and a decimal
// of scale 0 changes what every wire value means, so coercing null into one
// substitutes a schema nobody wrote. Every typed body read asks this first and
// treats null as malformed, routing it like any wrong-typed body: hard reject
// where the kind binds the key, verbatim to props where it does not.
//
// Keys read by type ASSERTION need no such guard — a JSON null is a nil any,
// which satisfies no assertion.
func jsonNullBody(v any) bool {
	return v == nil
}

// decodeLaxInt re-marshals a decoded JSON value and reads it back through
// laxInt — the schema grammar's integer decode (plain integer syntax or
// the quoted [INTEGERS] form, length-capped). This is the ONE integer
// predicate for the "size" attribute, shared by the parse arm
// (aobjectFromMap), the metadata capture (nodeFromJSONObject), and the
// stray shape verdict (strayBodyShapeOK), so the surfaces cannot drift on
// what counts as a size: a value that fails here rides to props verbatim
// on every surface and never yields a coerced structural value.
//
// key names the attribute in the error, and is what makes the null verdict
// readable: a body that names no integer is reported against the key that
// was written, not as a decode of an anonymous value.
func decodeLaxInt(key string, v any) (laxInt, error) {
	if jsonNullBody(v) {
		return 0, schemaTypeMismatch(key, "integer")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return 0, err
	}
	var l laxInt
	if err := l.UnmarshalJSON(raw); err != nil {
		return 0, err
	}
	return l, nil
}

// intPtrFrom reads m[key] as a *int by re-marshaling the small value and
// reusing stdlib int decode, so the accept/reject behavior (rejecting
// floats, strings, overflow) is identical to the former *int struct field —
// with one shape the re-marshal cannot inherit. Unmarshaling into that
// struct field handled null by setting the POINTER to nil (absent); the
// re-marshal decodes into a plain int, where null is a no-op that would
// hand back a pointer to zero. [jsonNullBody] restores the distinction by
// rejecting the body outright, which is stricter than the struct field was
// and is the right verdict for both callers: precision and scale are
// consumed only on a decimal carrier, and a decimal whose parameter names
// no number is not a decimal.
func intPtrFrom(m map[string]any, key string) (*int, error) {
	v, ok := m[key]
	if !ok {
		return nil, nil
	}
	if jsonNullBody(v) {
		return nil, schemaTypeMismatch(key, "integer")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil, boundJSONErrorEcho(err)
	}
	return &n, nil
}

// strayRoutedKeys are the structural/naming keys whose STRAY placements
// (on a kind that does not bind them) get shape-conditional routing: a
// body that parses as the key's schema shape surfaces on the matching
// SchemaNode structural field (as-written), anything else rides in Props
// verbatim — the same route unconsumed precision/scale already take.
// Only the exact lowercase spelling is a reserved key; a case-variant
// spelling is an ordinary custom property with no routing of its own.
var strayRoutedKeys = [...]string{
	"items", "values", "fields", "symbols", "size", "name", "namespace", "aliases",
}

// canonicalStrayKey returns k when it is one of the stray-routed keys
// (exact lowercase spelling, like every reserved-key match), else "".
func canonicalStrayKey(k string) string {
	for _, key := range strayRoutedKeys {
		if k == key {
			return key
		}
	}
	return ""
}

// strayKeyBinds reports whether a node of the given kind BINDS key — the
// parser's kind-keyed grammar. A binding kind shape-validates the key's
// value and consumes it; on any other kind the key is a stray the parse
// never binds, so a malformed body there cannot be an attempt to define,
// scope, or reference anything.
//
// The keys whose binding also depends on the VALUE or on the logical type
// (logicalType, precision/scale) are answered by [schemaKeyBinds], which
// wraps this; every key whose binding is decided by the kind alone is
// answered here, so the two are one question with one kind-keyed table.
func strayKeyBinds(typ, key string) bool {
	switch key {
	case "type", "doc":
		// Read on every kind: "type" names the kind itself, and "doc" is
		// documentation the metadata surfaces on any node.
		return true
	case "items":
		return typ == "array"
	case "values":
		return typ == "map"
	case "fields":
		return isRecordKind(typ)
	case "symbols":
		return typ == "enum"
	case "size":
		return typ == "fixed"
	case "name", "namespace", "aliases":
		return isNamedKind(typ)
	case "default":
		// The enum evolution default is the ONLY type-level binding of
		// "default" (Java's ENUM_RESERVED is SCHEMA_RESERVED plus this one
		// key, Schema.java:178-180, applied at :1928). A record FIELD binds
		// it too, but a field object is read by afieldFromAny against the
		// field grammar, never by this kind-keyed one.
		return typ == "enum"
	case "order":
		// A field-only sort attribute: no type-level kind binds it, and
		// neither reference reserves it on a schema object (Java's
		// SCHEMA_RESERVED and ENUM_RESERVED both omit it, Schema.java:175-180).
		return false
	}
	return false
}

// strayBodyShapeOK reports whether v parses as key's schema shape. It
// runs the SAME decodes the parser's own arms run (aschemaFromAny,
// afieldFromAny, the string-slice and laxInt reads), so the wire parse
// and the metadata walker cannot disagree on a stray body's surfacing
// route: shape-OK bodies surface on the matching structural field,
// anything else stays a Props entry, and the accept/reject boundary for
// BINDING kinds is untouched (their arms still propagate the error).
func strayBodyShapeOK(key string, v any) bool {
	switch key {
	case "name", "namespace":
		_, ok := v.(string)
		return ok
	case "symbols", "aliases":
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for _, e := range arr {
			if _, ok := e.(string); !ok {
				return false
			}
		}
		return true
	case "size":
		_, err := decodeLaxInt(key, v)
		return err == nil
	case "items", "values":
		var s aschema
		return aschemaFromAny(v, &s, nil) == nil
	case "fields":
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for i := range arr {
			var f afield
			if afieldFromAny(arr[i], &f, nil) != nil {
				return false
			}
		}
		return true
	}
	return false
}

// strayShapeMemo caches, by subtree pointer, whether a schema-position
// subtree parses as a valid schema shape. The metadata walker's stray gates
// consult a stray body's shape once per node, and a nested-stray schema
// nests those bodies, so without memoization each body is re-validated once
// per enclosing level — O(depth^2). One memo shared across a Root() walk
// makes it linear: the FIRST validation of a subtree records it and every
// nested subtree it decodes (aobjectFromMap defers the record), so a later
// level's check of an inner subtree is a cache hit. The recorded verdict is
// produced by the SAME parser decodes strayBodyShapeOK runs, so it is
// identical to a fresh check; the map is used only by the metadata walker,
// and a nil memo (every other caller) takes the un-memoized path.
type strayShapeMemo map[uintptr]bool

// errStrayShapeCached is the sentinel aobjectFromMap returns for a subtree
// the memo already recorded as an invalid schema shape. It never escapes
// the shape-validity path (the only caller that passes a non-nil memo reads
// only whether the error is nil), so its text is never surfaced.
var errStrayShapeCached = errors.New("avro: stray shape (cached invalid)")

// strayBodyShapeOKMemo is strayBodyShapeOK with per-subtree memoization for
// the recursive keys (items/values/fields), whose bodies can nest and so
// drive the O(depth^2). A miss runs the ordinary decode threaded with the
// memo (recording this body and every subtree it walks); a later query of
// an already-walked subtree short-circuits at aobjectFromMap. The
// non-recursive keys carry no compounding cost and take the plain check.
func strayBodyShapeOKMemo(memo strayShapeMemo, key string, v any) bool {
	if memo == nil {
		return strayBodyShapeOK(key, v)
	}
	switch key {
	case "items", "values":
		var s aschema
		return aschemaFromAny(v, &s, memo) == nil
	case "fields":
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for i := range arr {
			var f afield
			if afieldFromAny(arr[i], &f, memo) != nil {
				return false
			}
		}
		return true
	default:
		return strayBodyShapeOK(key, v)
	}
}

// strayShapeVerdict reports whether a stray-routed reserved key's body
// (canonical key spelling, raw value v) parsed as that key's schema shape.
// A caller that already decoded the body once passes its RECORDED verdict
// here so the props-routing (schemaReservedKeyForObject) and the metadata
// child-surfacing never re-decode a subtree the caller already walked —
// the re-decode is what turns a nested-stray schema into O(2^depth) work.
type strayShapeVerdict func(canonKey string, v any) bool

// strayShapeRecorded returns the aobject's own arm verdicts as a
// strayShapeVerdict: a stray-routed body parsed as its schema shape iff the
// matching arm set the aobject field (o.Items for "items", o.Values for
// "values", o.Fields for "fields", and so on). The arms run aschemaFromAny/
// afieldFromAny/the string-slice and laxInt reads — the SAME decodes
// strayBodyShapeOK runs — so field-set is an exact mirror of the shape
// check, and consulting it lets the extra-property loop skip a second
// decode of every already-walked body. nameIsString carries the name arm's
// string-ness (o.Name is a bare string with no present/absent flag of its
// own; the empty short name "" is a valid name shape).
func (o *aobject) strayShapeRecorded(nameIsString bool) strayShapeVerdict {
	return func(canonKey string, _ any) bool {
		switch canonKey {
		case "items":
			return o.Items != nil
		case "values":
			return o.Values != nil
		case "fields":
			return o.Fields != nil
		case "symbols":
			return o.Symbols != nil
		case "size":
			return o.Size != nil
		case "aliases":
			return o.Aliases != nil
		case "namespace":
			return o.Namespace != nil
		case "name":
			return nameIsString
		}
		return false
	}
}
