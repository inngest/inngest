package avro

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	textAppenderType    = reflect.TypeFor[encoding.TextAppender]()
	textMarshalerType   = reflect.TypeFor[encoding.TextMarshaler]()
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	isZeroerType        = reflect.TypeFor[interface{ IsZero() bool }]()
	byteType            = reflect.TypeFor[byte]()
)

// copyBytesToArray writes b into the byte-array v: reflect.Copy's memmove for
// an exact-uint8 element, element-wise SetUint for a NAMED byte type. Callers
// have established that v is an Array of Uint8-kind elements with
// v.Len() == len(b).
//
// reflect.Copy panics on [N]B where type B byte — a panic reaching the public
// Decode on a value the encoder accepts, since the byte encoders iterate via
// Uint. SetUint writes through the Kind and restores round-trip parity.
func copyBytesToArray(v reflect.Value, b []byte) {
	if v.Type().Elem() == byteType {
		reflect.Copy(v, reflect.ValueOf(b))
		return
	}
	for i := range b {
		v.Index(i).SetUint(uint64(b[i]))
	}
}

// byteArrayToSlice reads the byte-array v into a fresh []byte. It is the
// encode-side counterpart of [copyBytesToArray]: the exact-uint8 element uses
// reflect.Copy's memmove, a named byte element ([N]B, type B byte) falls back to
// element-wise reads, where reflect.Copy would panic on the exact-type
// mismatch. The caller has established that v is an Array whose element Kind is
// Uint8.
func byteArrayToSlice(v reflect.Value) []byte {
	b := make([]byte, v.Len())
	if v.Type().Elem() == byteType {
		reflect.Copy(reflect.ValueOf(b), v)
		return b
	}
	for i := range b {
		b[i] = byte(v.Index(i).Uint())
	}
	return b
}

// reuseOrMakeStringAnyMap reuses v's existing map[string]any backing
// when v is an interface wrapping one (the streaming-decode pattern
// pinned by TestDecodeReuseAnyTargetStaleKeys); otherwise allocates a
// fresh map sized to hint. Shared by deserRecord's interface arm and
// decodeRecordAny so the two record-into-*any paths agree on reuse.
//
// Reuse retains keys not present in the schema; callers that need a
// fresh decode should clear or replace the map before each call.
func reuseOrMakeStringAnyMap(v reflect.Value, hint int) map[string]any {
	if inner := v.Elem(); inner.IsValid() && inner.Type() == mapStringAnyType {
		return inner.Interface().(map[string]any)
	}
	return make(map[string]any, hint)
}

// tryTextUnmarshal calls (*v).UnmarshalText(b) when v is addressable and its
// address implements [encoding.TextUnmarshaler]. Returns (true, err) when
// invoked, (false, nil) when v cannot accept the text. Caller owns b; this does
// not copy. Used at every text-shaped decode site, binary and JSON.
//
// TextUnmarshaler stands alone: the decoder does not also require TextMarshaler,
// so the one-way parse-only idiom is supported.
func tryTextUnmarshal(v reflect.Value, b []byte) (bool, error) {
	if !v.CanAddr() || !v.Addr().Type().Implements(textUnmarshalerType) {
		return false, nil
	}
	return true, v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText(b)
}

// textOutFor returns the strongest text-out method on v: TextAppender
// is preferred (alloc-free), TextMarshaler is the fallback. Checks
// both v.Interface() (value method set) and v.Addr().Interface()
// (pointer method set on addressable values) so pointer-receiver
// MarshalText/AppendText on an addressable struct field is reachable
// — mirroring tryTextUnmarshal's TextUnmarshaler discovery via v.Addr().
//
// TextMarshaler / TextAppender stand on their own — the encoder does
// not require the type to also implement TextUnmarshaler.
//
// Fast-out: text-out methods are tried BEFORE the reflect.String / enum
// int-ordinal arms at every encode site, so this runs on every plain
// string / enum encode. The alloc-free type check short-circuits before
// the v.Interface() boxing for types that can't implement a text-out
// method — keeping the common plain-string/enum path allocation-free.
// If implementsTextMarshaler is false, neither method set has the method,
// so the body below would return (nil, nil) anyway.
func textOutFor(v reflect.Value) (encoding.TextAppender, encoding.TextMarshaler) {
	if !implementsTextMarshaler(v.Type()) {
		return nil, nil
	}
	var appender encoding.TextAppender
	var marshaler encoding.TextMarshaler
	if v.CanInterface() {
		i := v.Interface()
		if a, ok := i.(encoding.TextAppender); ok {
			appender = a
		}
		if m, ok := i.(encoding.TextMarshaler); ok {
			marshaler = m
		}
	}
	if (appender == nil || marshaler == nil) && v.CanAddr() {
		i := v.Addr().Interface()
		if appender == nil {
			if a, ok := i.(encoding.TextAppender); ok {
				appender = a
			}
		}
		if marshaler == nil {
			if m, ok := i.(encoding.TextMarshaler); ok {
				marshaler = m
			}
		}
	}
	return appender, marshaler
}

// textValue materializes v's TextAppender or TextMarshaler output as a string.
// Returns (text, true, nil) on success, ("", false, nil) when v has no text-out
// method so the caller falls through, ("", false, SemanticError) when the
// method itself errored. Every text-shaped encode site uses it, so all share
// one wrap shape and one AppendText-over-MarshalText preference.
func textValue(v reflect.Value, avroType string) (string, bool, error) {
	a, m := textOutFor(v)
	if a == nil && m == nil {
		return "", false, nil
	}
	var text []byte
	var err error
	if a != nil {
		text, err = a.AppendText(nil)
	} else {
		text, err = m.MarshalText()
	}
	if err != nil {
		return "", false, &SemanticError{GoType: v.Type(), AvroType: avroType, Err: err}
	}
	return string(text), true, nil
}

// implementsTextMarshaler reports whether t's value or pointer method set
// implements TextMarshaler or TextAppender. The fast string-encode paths read
// the underlying string directly and bypass appendAvroString's text-out arm, so
// keeping text-method types off them is what makes such a type encode its
// marshaled form in a struct field exactly as it does as a scalar. Evaluated
// once per type at compile time, never per value.
func implementsTextMarshaler(t reflect.Type) bool {
	// The pointer method set is a superset of the value one, so an empty
	// pointer set means no methods at all. One NumMethod read beats four
	// Implements scans on the per-element encode path, where zero-method
	// string and enum types dominate.
	pt := reflect.PointerTo(t)
	if pt.NumMethod() == 0 {
		return false
	}
	return t.Implements(textMarshalerType) || t.Implements(textAppenderType) ||
		pt.Implements(textMarshalerType) || pt.Implements(textAppenderType)
}

// implementsTextUnmarshaler reports whether *t implements TextUnmarshaler. The
// fast string-decode paths and the array/map loops write the wire string
// directly and bypass setStringValue's UnmarshalText arm, so keeping such types
// off them makes a struct field or container decode as the scalar does.
func implementsTextUnmarshaler(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(textUnmarshalerType)
}

// stringFastPathEligibleEncode reports whether a reflect.String-kind Go type
// may use an unsafe/fast string-encode path (usString, usFixedUUIDString,
// the container loops). It must take the slow reflect path when it is
// json.Number (appendAvroString's RFC 8259 reject) or implements a text-out
// method (appendAvroString's text-out arm) — both of which the fast paths
// bypass. Callers have already established Kind()==String.
//
// The SINGLE source of truth for which string-kind types are
// fast-path-ineligible on encode. Every encode gate consults it, so a new
// slow-path-only concern is added once rather than re-swept across all of them.
func stringFastPathEligibleEncode(t reflect.Type) bool {
	return t != jsonNumberType && !implementsTextMarshaler(t)
}

// stringFastPathEligibleDecode is the decode-side counterpart: a
// reflect.String-kind target is fast-path-ineligible when it is json.Number
// (setStringValue's RFC 8259 guard) or implements TextUnmarshaler
// (setStringValue's UnmarshalText arm). Single source of truth for the
// decode gates (udStringDeser, udFixedUUIDString, fastPathSafeForElem).
func stringFastPathEligibleDecode(t reflect.Type) bool {
	return t != jsonNumberType && !implementsTextUnmarshaler(t)
}

var (
	errIndirectNil  = errors.New("invalid nil in non-union, non-null")
	errIndirectDeep = errors.New("avro: pointer/interface chain on input is cyclic or nests deeper than supported")
)

// maxIndirectDepth bounds indirect/indirectAlloc unwrap loops. A self-
// referential interface (e.g. `var p any; p = &p`) creates a real cycle in
// Go that would otherwise spin forever in reflect.Value.Elem(). Five levels
// of pointer/interface wrapping is more than any realistic user value.
const maxIndirectDepth = 5

func indirect(v reflect.Value) (reflect.Value, error) {
	for range maxIndirectDepth {
		switch v.Kind() {
		case reflect.Invalid:
			// Defensive: an invalid Value (e.g. reflect.ValueOf(nil)
			// somewhere internally) reaches this guard rather than
			// panicking on a subsequent v.Type() call. Treat as nil.
			return v, errIndirectNil
		case reflect.Pointer, reflect.Interface:
			if v.IsNil() {
				return v, errIndirectNil
			}
			v = v.Elem()
		default:
			return v, nil
		}
	}
	// After maxIndirectDepth unwraps, the value may already be a non-pointer
	// base reached at exactly the cap — accept it, matching indirectAlloc,
	// isNilValue, and serNull (which peel in the loop body and inspect the
	// base afterward, so they accept a chain of maxIndirectDepth levels).
	// Inspecting only inside the loop's default arm would reject such a base
	// because confirming it costs one further iteration the loop has already
	// spent — an encode/decode off-by-one where a maxIndirectDepth-deep
	// pointer value decodes but fails to encode. Only a STILL-indirect value,
	// including a cyclic interface (var p any; p = &p), is genuinely too deep.
	switch v.Kind() {
	case reflect.Invalid:
		return v, errIndirectNil
	case reflect.Pointer, reflect.Interface:
		return v, errIndirectDeep
	default:
		return v, nil
	}
}

func indirectAlloc(v reflect.Value) reflect.Value {
	for range maxIndirectDepth {
		switch v.Kind() {
		case reflect.Pointer:
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		case reflect.Interface:
			if v.IsNil() {
				return v
			}
			// Non-nil interface: unwrap only if the inner is a
			// non-nil pointer (write through the pointer is
			// addressable). For ANY other concrete — primitives,
			// structs, slices, maps, nil pointers — v.Elem() is
			// not addressable. Some decoders reach for v.Set(...)
			// on the unwrapped value (e.g. decodeNull zeros it,
			// decodeArray replaces the slice), which panics. Keep
			// the interface itself as the destination so those
			// decoders write via Set on the settable interface
			// Value.
			inner := v.Elem()
			if inner.Kind() != reflect.Pointer || inner.IsNil() {
				return v
			}
			v = inner
		default:
			return v
		}
	}
	return v
}

// setIface assigns rv to an interface-kind v, checking assignability first.
// Without the check reflect.Value.Set panics; the case that reaches it is a
// user passing *interface{Foo()} as a decode target.
//
// Caller contract: v.Kind() must be reflect.Interface. A concrete v gets a
// SemanticError, so concrete-target paths split the dispatch at the call site —
// see deserFixedUUIDReflect for the pattern.
//
// Cold paths only. On the HOT primitive paths, do NOT use setIface: passing rv
// across a function boundary loses escape analysis and heap-allocates every
// reflect.ValueOf(primitive), ~+2 allocs / +330 B per record decode in the
// bench. Inline the check instead, fast path first so rv exists only on the
// slow branch:
//
//	if v.Type().NumMethod() == 0 {        // empty interface (any) — common
//	    v.Set(reflect.ValueOf(b))
//	    return nil
//	}
//	rv := reflect.ValueOf(b)              // slow path: typed interface
//	if !rv.Type().AssignableTo(v.Type()) {
//	    return &SemanticError{GoType: v.Type(), AvroType: "boolean"}
//	}
//	v.Set(rv)
//	return nil
func setIface(v, rv reflect.Value, avroType string) error {
	if v.Kind() != reflect.Interface {
		return &SemanticError{GoType: v.Type(), AvroType: avroType}
	}
	if v.Type().NumMethod() == 0 || rv.Type().AssignableTo(v.Type()) {
		v.Set(rv)
		return nil
	}
	return &SemanticError{GoType: v.Type(), AvroType: avroType}
}

// mapKeyAs returns key as a reflect.Value typed to match mapType.Key().
// Fast-path identity when the types already match (the common
// map[string]V case); slow-path Convert() for named-string-key maps
// (e.g. `type UserID string; map[UserID]V`). Without this conversion,
// reflect.MapIndex / SetMapIndex panics with "value of type string is
// not assignable to type X". Used by every record-as-map encode and
// every map-decode site that builds an Avro map with string keys.
func mapKeyAs(mapType reflect.Type, key reflect.Value) reflect.Value {
	if mapType.Key() == key.Type() {
		return key
	}
	return key.Convert(mapType.Key())
}

// fieldByIndex is like reflect.Value.FieldByIndex but allocates nil embedded
// pointer structs along the path, which is needed during deserialization.
func fieldByIndex(v reflect.Value, index []int) (reflect.Value, error) {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				// A nil embedded pointer reached through an UNEXPORTED
				// embedded field name is unsettable — Go reflection
				// cannot allocate it. Error cleanly rather than panic in
				// Set (matching encoding/json, which likewise cannot
				// write through an unexported embedded pointer).
				if !v.CanSet() {
					return reflect.Value{}, fmt.Errorf("avro: cannot decode into a field promoted through a nil unexported embedded pointer")
				}
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v, nil
}

// fieldByIndexZero is the read-only (encode) counterpart of fieldByIndex:
// it walks index from v, returning the ZERO value of the target field
// when the path crosses a nil embedded pointer instead of panicking. A
// nil embedded *struct thus encodes its promoted fields as zero —
// symmetric with fieldByIndex allocating them on decode.
func fieldByIndexZero(v reflect.Value, index []int) reflect.Value {
	for n, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return reflect.Zero(fieldTypeByIndex(v.Type(), index[n:]))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// fieldTypeByIndex resolves the type reached by walking index from t,
// dereferencing pointer types along the way.
func fieldTypeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, i := range index {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		t = t.Field(i).Type
	}
	return t
}

// cachedMapping holds the results of typeFieldMapping, cached per Go type.
type cachedMapping struct {
	indices  [][]int
	omitzero []bool
}

// typeFieldMapping returns the field index paths for each schema field in the
// given Go type. It handles embedded (anonymous) structs and inline-tagged
// fields by recursing into them. Avro-tagged fields take priority over
// name-matched fields, and shallower fields take priority over deeper ones.
//
// The result is cached in the provided sync.Map for subsequent calls with the
// same type.
func typeFieldMapping(fieldNames []string, cache *sync.Map, t reflect.Type) (*cachedMapping, error) {
	if cache != nil {
		if v, ok := cache.Load(t); ok {
			return v.(*cachedMapping), nil
		}
	}

	type fieldInfo struct {
		name     string
		index    []int
		tagged   bool
		omitzero bool
	}

	// collect walks the struct tree depth-first, recording fields in
	// encounter order. Shallower fields are seen first, which matters
	// for the priority logic below.
	var fields []fieldInfo
	var collect func(t reflect.Type, index []int, visited map[reflect.Type]bool)
	collect = func(t reflect.Type, index []int, visited map[reflect.Type]bool) {
		if visited[t] {
			return // prevent infinite recursion on embedded struct cycles
		}
		// PER-PATH marking: a cycle revisits a type while it is still ON the
		// current path, so the on-path check above terminates it; but the
		// SAME type reachable through two SIBLING embed paths is not a cycle
		// and must be collected at each occurrence so the shallower one
		// reaches the shallowest-wins dedup below. Marking-forever pruned the
		// sibling occurrence, silently selecting the deeper field and
		// violating doc.go's promotion contract. Same idiom as toJSONWalk
		// (schema_node.go).
		visited[t] = true
		defer delete(visited, t)
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
				// Recurse into embedded structs (even unexported
				// ones, since they can have exported fields).
				if ft.Kind() == reflect.Struct {
					tag := sf.Tag.Get("avro")
					if tag == "-" {
						continue
					}
					// If the embedded struct has an explicit avro
					// tag, treat it as a named field rather than
					// inlining its fields.
					parts := splitFieldTag(tag)
					name := parts[0]
					if name != "" {
						_, oz := parseTagOptions(parts[1:])
						fields = append(fields, fieldInfo{
							name:     name,
							index:    idx,
							tagged:   true,
							omitzero: oz,
						})
						continue
					}
					collect(ft, idx, visited)
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
			parts := splitFieldTag(tag)
			name := parts[0]
			tagged := name != ""
			inline, oz := parseTagOptions(parts[1:])

			if inline {
				ft := sf.Type
				if ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}
				if ft.Kind() == reflect.Struct {
					collect(ft, idx, visited)
					continue
				}
			}

			if name == "" {
				name = sf.Name
			}
			fields = append(fields, fieldInfo{
				name:     name,
				index:    idx,
				tagged:   tagged,
				omitzero: oz,
			})
		}
	}
	collect(t, nil, make(map[reflect.Type]bool))

	type entry struct {
		index    []int
		tagged   bool
		omitzero bool
	}
	m := make(map[string]entry, len(fields))
	ambiguous := make(map[string][2]string) // name -> the two colliding Go field names
	for _, f := range fields {
		if existing, ok := m[f.name]; ok {
			if f.tagged && !existing.tagged {
				m[f.name] = entry{f.index, f.tagged, f.omitzero}
				delete(ambiguous, f.name)
				continue
			}
			if !f.tagged && existing.tagged {
				continue
			}
			if len(f.index) < len(existing.index) {
				m[f.name] = entry{f.index, f.tagged, f.omitzero}
				delete(ambiguous, f.name)
				continue
			}
			if len(f.index) == len(existing.index) {
				// Equal depth, same tagged status, no tiebreaker: AMBIGUOUS.
				// encoding/json silently drops such a field; this DEFERS the
				// error to lookup, so a coincidental collision on a name the
				// schema never references does not break the whole struct
				// while one that does resolve errors loudly. SchemaFor's
				// collectFields rejects eagerly because it must emit every
				// field; the runtime is schema-driven.
				ambiguous[f.name] = [2]string{t.FieldByIndex(existing.index).Name, t.FieldByIndex(f.index).Name}
			}
			continue
		}
		m[f.name] = entry{f.index, f.tagged, f.omitzero}
	}

	ats := make([][]int, 0, len(fieldNames))
	ozs := make([]bool, 0, len(fieldNames))
	for _, name := range fieldNames {
		if names, amb := ambiguous[name]; amb {
			return nil, &SemanticError{GoType: t, AvroType: "record", Err: fmt.Errorf(
				"duplicate field name %q in type %s (fields %q and %q both map to the same Avro name)",
				truncForError(name), t.String(), truncForError(names[0]), truncForError(names[1]))}
		}
		e, exists := m[name]
		if !exists {
			// truncForError: name is a schema field name with no length cap
			// (validName grammar / WithLaxNames). It rides in .Err, not the
			// render-truncated .Field, so SemanticError.Error() echoes it raw
			// — bound it at construction like the sibling duplicate-name error
			// above and the composed-sentence echoes in json_codec/json_decode.
			return nil, &SemanticError{GoType: t, AvroType: "record", Err: fmt.Errorf("missing field %s", truncForError(name))}
		}
		ats = append(ats, e.index)
		ozs = append(ozs, e.omitzero)
	}

	result := &cachedMapping{indices: ats, omitzero: ozs}
	if cache != nil {
		cache.Store(t, result)
	}
	return result, nil
}

// splitFieldTag tokenizes an avro struct tag with the SAME grammar SchemaFor
// uses ([splitTag]): top-level commas separate options, default= takes the rest
// verbatim, and bracketed alias=[...] / decimal(...) values do not split on
// their internal commas. A naive strings.Split would read `default=a,omitzero`
// or `alias=[x,inline,y]` as separate options and fire omitzero/inline that
// SchemaFor never sees. A malformed tag falls back to the naive split, so the
// runtime never newly errors on a tag a hand-written-schema user relies on.
func splitFieldTag(tag string) []string {
	parts, err := splitTag(tag)
	if err != nil {
		// The tag is malformed (unbalanced brackets/parens), so splitTag's
		// grammar cannot tokenize it. A naive strings.Split here would surface
		// interior tokens as options — firing inline/omitzero out of an
		// alias=[…] / decimal(…) value or any other comma-bearing fragment —
		// the exact spurious firing splitTag was adopted to prevent, observable
		// as a bracket typo silently flipping a field between nested-record and
		// inline-flattened. Stay lenient (a hand-written-schema user's malformed
		// tag must not newly error) but fire NO options: map the field under its
		// name (the tag text up to the first comma; an Avro field name contains
		// no comma) and drop the unparseable option fragments.
		name, _, _ := strings.Cut(tag, ",")
		return []string{name}
	}
	return parts
}

// parseTagOptions parses tag options after the field name. It returns whether
// "inline" and "omitzero" were found. Inputs come from [splitFieldTag], so an
// option carrying a value (default=…, alias=…) arrives as a single part that
// can never equal a bare keyword.
func parseTagOptions(opts []string) (inline, omitzero bool) {
	for _, o := range opts {
		switch o {
		case "inline":
			inline = true
		case "omitzero":
			omitzero = true
		}
	}
	return
}

// valueIsZero reports whether v is the zero value for its type, or implements
// an IsZero() bool method that returns true.
func valueIsZero(v reflect.Value) bool {
	// A nil pointer or interface IS the zero value. Short-circuit before the
	// IsZero assertion: a promoted value-receiver IsZero on a nil *time.Time
	// dereferences the nil and panics.
	if k := v.Kind(); (k == reflect.Pointer || k == reflect.Interface) && v.IsNil() {
		return true
	}
	if v.CanInterface() {
		if z, ok := v.Interface().(interface{ IsZero() bool }); ok {
			return z.IsZero()
		}
		// Pointer-receiver IsZero lives in *T's method set, which the assertion
		// above misses. Box a non-addressable value into an addressable temp so
		// Encode(v) and Encode(&v) agree, as they already do for a
		// value-receiver IsZero. The nil short-circuit above ran, so v is
		// non-nil and *T IsZero is safe to invoke.
		if reflect.PointerTo(v.Type()).Implements(isZeroerType) {
			if !v.CanAddr() {
				box := reflect.New(v.Type())
				box.Elem().Set(v)
				v = box.Elem()
			}
			return v.Addr().Interface().(interface{ IsZero() bool }).IsZero()
		}
	}
	return v.IsZero()
}

// setZero sets v to its type's zero value. A null must clear the target,
// concrete primitives included, or a pre-populated field retains stale data
// across a reused decode and breaks the documented promise that null clears.
func setZero(v reflect.Value) {
	v.Set(reflect.Zero(v.Type()))
}
