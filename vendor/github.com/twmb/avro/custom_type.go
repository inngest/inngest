package avro

import (
	"errors"
	"fmt"
	"reflect"
)

// ErrSkipCustomType is returned from a [CustomType] Encode or Decode
// function to indicate the value is not handled by this custom type.
// The library falls through to the next matching custom type or to
// built-in behavior.
var ErrSkipCustomType = errors.New("avro: skip custom type")

// CustomType defines a custom conversion between a Go type and an Avro
// type. [NewCustomType] covers the primitive-backing-type case with the
// wiring inferred from its type parameters; this struct is the general
// form, and the only one that reaches records, fixed types, and
// property-based dispatch.
//
// Pass to [Parse] or [SchemaFor] as a [SchemaOpt].
//
// Matching at parse time: LogicalType and AvroType are checked against
// schema nodes. All non-empty criteria must match.
//   - LogicalType only: matches any schema node with that logicalType
//   - LogicalType + AvroType: matches that logicalType on that Avro type
//   - AvroType only: matches all nodes of that Avro type
//   - Neither: matches ALL schema nodes (use with [ErrSkipCustomType]
//     for property-based dispatch like Kafka Connect types)
//
// At encode time, GoType is also checked: the Encode function only
// fires when the value's type matches GoType. This prevents the codec
// from intercepting native values (e.g. a raw int64 passes through
// without conversion for a custom-typed long field).
//
// A matching CustomType replaces the built-in logical type
// deserializer. Among user registrations, first match wins.
//
// Backed by a complex Avro type, the Encode function returns
// map[string]any, []any, and so on.
type CustomType struct {
	// LogicalType narrows matching to schema nodes with this logicalType.
	LogicalType string

	// AvroType narrows matching to schema nodes of this Avro type
	// (e.g. "long", "bytes", "record"). Also used by SchemaFor to
	// infer the underlying Avro type.
	AvroType string

	// GoType adds an encode-time filter: when set, the Encode function
	// only fires when the value's concrete type matches GoType. Values
	// of other types pass through to the underlying serializer unchanged.
	// If nil, Encode fires for all values on matched schema nodes
	// (those matching LogicalType/AvroType).
	//
	// [SchemaFor] uses GoType to match struct fields: when a field's Go
	// type equals GoType, SchemaFor emits AvroType + LogicalType (or
	// Schema) instead of the default type mapping. Because the custom
	// supplies the whole field schema, a logical-type tag on a matched
	// field has no effect and is rejected — set LogicalType (or Schema)
	// here instead. If nil, the custom type does not affect schema
	// generation, but is still wired into the returned [*Schema] for
	// encode/decode.
	GoType reflect.Type

	// Schema is the full schema to emit in SchemaFor. Only needed for
	// types requiring extra metadata (fixed needs name+size, decimal
	// needs precision+scale, records need fields). If nil, SchemaFor
	// infers from AvroType + LogicalType.
	//
	// SchemaFor preserves every fullname the schema declares: a
	// namespaced type keeps its namespace, and a null-namespace type
	// embedded under [WithNamespace] keeps its null namespace (the
	// emitted definition carries the "namespace":"" inheritance escape).
	// One combination is unrepresentable and errors: a null-namespace
	// type used on two or more fields under WithNamespace, because Avro
	// has no reference spelling that reaches the null namespace from
	// inside another namespace.
	//
	// SchemaFor composes a private copy of the rendered schema, so the
	// SchemaNode and everything reachable from it (including Props
	// container values) are never mutated by a build, and it fails the
	// build with the walk's named error when the schema exceeds the
	// schema-tree budgets or contains an unnamed pointer cycle.
	//
	// A union branch's type may be written in either spelling Avro
	// admits — the bare name ("null") or the wrapped object
	// ({"type":"null"}) — and composition treats the two as the one type
	// they are. A null branch is recognized in both spellings, and in the
	// wrapped form regardless of any properties or logicalType it carries
	// (Avro defines no null logical type, so both are inert): a nullable
	// union collapses through a pointer field and receives its null
	// default identically either way.
	Schema *SchemaNode

	// Encode converts a caller-provided Go value to an Avro-native
	// value, called before serialization. The callback receives the
	// value as passed to [Schema.Encode] (e.g. a custom Money type),
	// and should return the corresponding Avro-native value (e.g.
	// int64 cents). Return [ErrSkipCustomType] to fall through to the
	// next matching custom type or built-in behavior. Any other
	// non-nil error is fatal.
	//
	// If nil, the built-in logical type encoder is used, which accepts
	// both enriched types ([time.Time], [time.Duration]) and raw
	// values (int64, int32, etc.).
	//
	// The schema argument is built once at Parse and shared across all
	// concurrent invocations. Treat it as read-only; in particular, do
	// not mutate schema.Props or schema.Symbols — those slices/maps
	// alias the parser's internal state and concurrent writes from
	// multiple goroutines decoding the same [*Schema] will race.
	Encode func(v any, schema *SchemaNode) (any, error)

	// Decode converts a raw Avro-native value to a custom Go value,
	// called after deserialization. The callback receives the raw
	// Avro-native value (int32 for int, int64 for long, []byte for
	// bytes/fixed, etc.) and should return the desired Go type.
	// Return [ErrSkipCustomType] to fall through. Any other non-nil
	// error is fatal.
	//
	// When all matching decoders skip at a node, the wire is re-decoded
	// into the target faithfully (identical to a no-custom decode); a
	// wildcard custom (empty LogicalType and AvroType) that matches leaf
	// nodes but skips containers therefore makes decoding into a
	// deeply-nested TYPED target (struct/slice/map) cost O(depth^2) — for
	// untrusted deeply-nested data decode into an interface / map[string]any
	// (single-pass) or register against a specific LogicalType/AvroType.
	//
	// If nil, the built-in logical type handler is bypassed and the
	// base Avro type decoder is used directly, producing raw
	// Avro-native values (int32, int64, etc.) rather than enriched
	// types ([time.Time], [time.Duration], etc.).
	//
	// The schema argument is shared across concurrent callback invocations;
	// see [CustomType.Encode] for the read-only contract.
	Decode func(v any, schema *SchemaNode) (any, error)

	// Set by NewCustomType; if true and AvroType is "", Parse returns
	// an error ("unsupported Avro native type").
	needsAvroType bool
}

func (CustomType) schemaOpt() {}

// WithCustomType registers a custom type conversion for use with
// [Parse], [SchemaCache.Parse], or [SchemaFor]. [CustomType] and
// [NewCustomType] both satisfy [SchemaOpt] directly, so this wrapper
// is optional.
func WithCustomType(ct CustomType) SchemaOpt { return ct }

// matches returns true if ct's criteria match the given schema node.
func (ct CustomType) matches(node *schemaNode) bool {
	if ct.LogicalType != "" && ct.LogicalType != node.logical {
		return false
	}
	if ct.AvroType != "" && ct.AvroType != node.kind {
		return false
	}
	return true
}

// NewCustomType returns a type-safe [CustomType] for the common case of
// mapping a custom Go type to/from a primitive Avro type.
//
// G is the custom Go type (e.g. Money). A is the Avro-native Go type:
// int32 for int, int64 for long, float32 for float, float64 for double,
// string for string, []byte for bytes, bool for boolean. A may also be a
// named type whose underlying kind is one of these (e.g. type Cents int64);
// the Avro type is inferred from A's kind and the decoded value is converted
// to A.
//
// GoType and AvroType are inferred from the type parameters. If A is
// not a supported Avro-native type, [Parse] or [SchemaFor] returns an
// error.
//
// Note: AvroType is inferred from A's Go kind, which may not match
// the Avro schema's type for logical types backed by smaller types.
// For example, time-millis uses Avro "int" but time.Duration is int64
// (which infers "long"). Use int32 as A, or use the [CustomType]
// struct directly with an explicit AvroType.
//
// For fixed, records, or types needing extra schema metadata, use the
// [CustomType] struct directly.
func NewCustomType[G, A any](
	logicalType string,
	encode func(G, *SchemaNode) (A, error),
	decode func(A, *SchemaNode) (G, error),
) CustomType {
	goType := reflect.TypeFor[G]()
	avroType := inferAvroType(reflect.TypeFor[A]())

	var encFn func(any, *SchemaNode) (any, error)
	if encode != nil {
		encFn = func(v any, sn *SchemaNode) (any, error) {
			return encode(v.(G), sn)
		}
	}
	var decFn func(any, *SchemaNode) (any, error)
	if decode != nil {
		aType := reflect.TypeFor[A]()
		decFn = func(v any, sn *SchemaNode) (any, error) {
			a, ok := v.(A)
			if !ok {
				// The base deserializer produces the CANONICAL Go value for A's
				// Avro kind (int32 for int, []byte for bytes, ...). When A is a
				// NAMED type over that kind (type UnixMillis int64), the value's
				// dynamic type is the base kind, not A, so a bare v.(A) panics.
				// inferAvroType keys on A's reflect.Kind, so the canonical value
				// is always convertible to A; convert rather than assert.
				rv := reflect.ValueOf(v)
				if !rv.IsValid() || !rv.Type().ConvertibleTo(aType) {
					return nil, fmt.Errorf("avro: custom decode: cannot convert %T to %s", v, aType)
				}
				a = rv.Convert(aType).Interface().(A)
			}
			return decode(a, sn)
		}
	}

	return CustomType{
		LogicalType:   logicalType,
		AvroType:      avroType,
		GoType:        goType,
		Encode:        encFn,
		Decode:        decFn,
		needsAvroType: true,
	}
}

// inferAvroType maps a Go reflect.Type to an Avro type name.
// Returns "" for unsupported types (validated at Parse time).
func inferAvroType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int32:
		return "int"
	case reflect.Int64:
		return "long"
	case reflect.Float32:
		return "float"
	case reflect.Float64:
		return "double"
	case reflect.String:
		return "string"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "bytes"
		}
	}
	return ""
}

// setCustomResult sets a custom type conversion result into the target
// reflect.Value, allocating pointees along the way. Returns a
// SemanticError if the result type is not assignable to the final target
// (rather than letting reflect.Value.Set panic).
func setCustomResult(v reflect.Value, result any, avroType string) error {
	if result == nil {
		setZero(v)
		return nil
	}
	rv := reflect.ValueOf(result)
	// Walk pointers, allocating as needed, stopping early once the result is
	// directly assignable. Bounded by maxIndirectDepth, the same ceiling the
	// non-custom path applies: a cyclic target type (type P *P) would otherwise
	// allocate a level per iteration forever.
	for i := 0; v.Kind() == reflect.Pointer; i++ {
		if rv.Type().AssignableTo(v.Type()) {
			v.Set(rv)
			return nil
		}
		if i >= maxIndirectDepth {
			return &SemanticError{GoType: v.Type(), AvroType: avroType}
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	if !rv.Type().AssignableTo(v.Type()) {
		return &SemanticError{GoType: v.Type(), AvroType: avroType}
	}
	v.Set(rv)
	return nil
}

// wrapDeserWithCustomDecoders wraps a deserfn with custom decode functions.
// Used both at parse time and during schema resolution to re-apply custom
// decoders to promoted/resolved nodes.
//
// When every decoder returns ErrSkipCustomType, the wire is RE-DECODED into the
// real target through inner, byte-for-byte the decode a no-custom schema
// performs. Placing the probe-decoded any instead would lose what only a real
// decode reproduces: a reused map keeping its existing keys, a logical node
// landing in a base typed target, an overlapping union recovering its exact
// wire branch.
//
// Cost: a naive re-decode re-runs nested wrappers, which re-probe, giving
// O(depth^2). The probe counts customMatches over the subtree; if none matched,
// bypassCustom makes the re-decode one O(subtree) pass. Only when a nested
// custom did match does the re-decode run with customs active, at O(depth^2)
// bounded by maxDepth.
//
// A fresh interface target needs no probe: inner already produces the canonical
// no-custom value, so it decodes straight into v, which doubles as the chain
// input. That keeps a parent's probe — all-`any` element targets — single-pass.
func wrapDeserWithCustomDecoders(inner deserfn, decoders []func(any, *SchemaNode) (any, error), sn *SchemaNode) deserfn {
	return func(src []byte, v reflect.Value, sl *slab) ([]byte, error) {
		// A no-match ancestor set this so the whole subtree decodes through inner
		// once: no custom matched anywhere in it, so skipping the chain is
		// identical to running it all-skip.
		if sl.bypassCustom {
			return inner(src, v, sl)
		}
		// Fresh interface target: inner's interface output IS the canonical value
		// a no-custom decode yields (tagged per the caller's option), so decode
		// straight into v and read it back for the chain — keeping a parent's
		// probe (whose element targets are all fresh `any`) to a single pass. A
		// NON-nil interface is excluded: inner would reuse the held value in place
		// (e.g. decode into a reused *T the custom is about to replace), so it
		// takes the probe + re-decode path below, just like a typed target.
		if v.Kind() == reflect.Interface && v.IsNil() {
			rest, err := inner(src, v, sl)
			if err != nil {
				return rest, err
			}
			chainVal := v.Interface()
			for _, dec := range decoders {
				result, err := dec(chainVal, sn)
				if err != nil {
					if errors.Is(err, ErrSkipCustomType) {
						continue
					}
					return nil, err
				}
				sl.customMatches++
				return rest, setCustomResult(v, result, sn.Type)
			}
			return rest, nil // all-skip: v already holds the no-custom value
		}
		// Typed target: probe into a throwaway any for the chain, then re-decode
		// faithfully into v on the all-skip fall-through.
		var tmp any
		savedMatches := sl.customMatches
		rest, err := inner(src, reflect.ValueOf(&tmp).Elem(), sl)
		if err != nil {
			return rest, err
		}
		for _, dec := range decoders {
			result, err := dec(tmp, sn)
			if err != nil {
				if errors.Is(err, ErrSkipCustomType) {
					continue
				}
				return nil, err
			}
			sl.customMatches++
			return rest, setCustomResult(v, result, sn.Type)
		}
		// Every decoder skipped: re-decode the original wire into the typed
		// target. No nested custom matched ⇒ bypass for a single pass; otherwise
		// re-decode with customs active to reproduce the nested match.
		if sl.customMatches == savedMatches {
			sl.bypassCustom = true
			_, err = inner(src, v, sl)
			sl.bypassCustom = false
			return rest, err
		}
		_, err = inner(src, v, sl)
		return rest, err
	}
}
