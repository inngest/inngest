// Package avro encodes and decodes [Avro specification] data.
//
// Parse an Avro JSON schema with [Parse] (or [MustParse] for package-level
// vars), then call [Schema.Encode] / [Schema.Decode] for binary encoding,
// or [Schema.EncodeJSON] / [Schema.DecodeJSON] for JSON encoding.
// Use [SchemaFor] to infer a schema from a Go struct type, or
// [Schema.Root] to inspect a parsed schema's structure.
//
// # Basic usage
//
//	schema := avro.MustParse(`{
//	    "type": "record",
//	    "name": "User",
//	    "fields": [
//	        {"name": "name", "type": "string"},
//	        {"name": "age",  "type": "int"}
//	    ]
//	}`)
//
//	type User struct {
//	    Name string `avro:"name"`
//	    Age  int    `avro:"age"`
//	}
//
//	// Encode
//	data, err := schema.Encode(&User{Name: "Alice", Age: 30})
//
//	// Decode
//	var u User
//	_, err = schema.Decode(data, &u)
//
// # JSON encoding
//
// [Schema.EncodeJSON] is schema-aware and handles bytes, unions, and
// NaN/Infinity floats correctly — use it instead of a generic JSON encoder
// when serializing decoded Avro data to JSON. Options control the output
// format: [TaggedUnions] for Avro JSON union wrappers ({"type": value}),
// [TagLogicalTypes] for qualified branch names, and [LinkedinFloats] for
// the goavro NaN/Infinity convention.
//
// # Encoding from JSON input
//
// Generically-decoded JSON data (map[string]any with float64 numbers and
// string timestamps) can be encoded directly. Missing map keys are filled
// from schema defaults, [encoding/json.Number] is accepted for numeric Avro
// types only (string, bytes, fixed, and enum reject it — use a Go string or
// []byte for those), and timestamp fields accept RFC 3339 strings. String fields accept
// [encoding.TextAppender] and [encoding.TextMarshaler] implementations
// (with [encoding.TextUnmarshaler] on decode).
//
// # Schema evolution
//
// Avro data is always written with a specific schema — the "writer schema."
// When you read it later your application may expect a different one — the
// "reader schema" — having added a field, removed one, or widened an int to a
// long.
//
// [Resolve] bridges the two: given writer and reader, it returns a schema that
// decodes the old wire format into the reader's layout.
//
//   - Fields in the reader but not the writer are filled from defaults.
//   - Fields in the writer but not the reader are skipped.
//   - Fields that exist in both are matched by name (or alias) and decoded,
//     with type promotion applied where needed (e.g. int → long).
//
// You typically get the writer schema from the data itself: an OCF file
// header embeds it, and schema registries store it by ID or fingerprint.
//
// As a concrete example, suppose v1 of your application wrote User records
// with just a name:
//
//	var writerSchema = avro.MustParse(`{
//	    "type": "record", "name": "User",
//	    "fields": [
//	        {"name": "name", "type": "string"}
//	    ]
//	}`)
//
// In v2, you added an email field with a default:
//
//	var readerSchema = avro.MustParse(`{
//	    "type": "record", "name": "User",
//	    "fields": [
//	        {"name": "name",  "type": "string"},
//	        {"name": "email", "type": "string", "default": ""}
//	    ]
//	}`)
//
//	type User struct {
//	    Name  string `avro:"name"`
//	    Email string `avro:"email"`
//	}
//
// To read old v1 data with your v2 struct, resolve the two schemas:
//
//	resolved, err := avro.Resolve(writerSchema, readerSchema)
//
//	// Decode v1 data: "email" is absent in the old data, so it gets
//	// the reader default ("").
//	var u User
//	_, err = resolved.Decode(v1Data, &u)
//	// u == User{Name: "Alice", Email: ""}
//
// If you just want to check whether two schemas are compatible without
// building a resolved schema, use [CheckCompatibility].
//
// A null union branch decodes to the target's Go zero value, always replacing
// any prior value. Use *T to distinguish null from zero.
//
// The reader schema is your contract for precision. A lossy reader schema —
// float or double — silently IEEE-rounds on both encode and decode, and an
// out-of-range finite input becomes ±Inf on the wire. An exact one — int, long,
// bytes, string — requires the Go decode target to hold the wire value without
// loss: a value outside the target's range, or one it cannot represent exactly
// such as a long above 2^53 decoded into a float64, is an error. For exact
// round-trip of large integers choose a long reader schema with an int64
// target rather than relying on a float to round.
//
// # Struct tags
//
// Use the "avro" struct tag to control field mapping and schema inference.
// The format is avro:"[name][,option]..." where the name maps the Go field
// to the Avro field name (empty = use Go field name, "-" = exclude).
//
// Encoding/decoding options:
//
//	avro:"name"           // map to Avro field "name"
//	avro:"-"              // exclude field
//	avro:",inline"        // flatten nested struct fields into parent record
//	avro:",omitzero"      // encode a zero value as the field's default (or null)
//
// Schema inference options (used by [SchemaFor]):
//
//	avro:",default=value"         // set field default (must be last option; scalars only)
//	avro:",alias=old_name"        // field alias for evolution (repeatable, or alias=[a,b])
//	avro:",type-alias=old_name"   // named type alias (record/enum/fixed) for evolution (repeatable, or type-alias=[a,b])
//	avro:",timestamp-micros"      // override logical type (also: timestamp-nanos, date, time-millis, time-micros)
//	avro:",decimal(10,2)"         // decimal logical type with precision and scale
//	avro:",uuid"                  // UUID logical type
//
// The alias tag adds an alias to the field itself. The type-alias tag adds an
// alias to the named type (record, enum, or fixed) that the field references,
// walking through pointers, slices, and maps to find it. This is needed when a
// writer schema uses a different name for the same type — for example, a legacy
// schema naming a record "r508" instead of "FieldSummary".
//
// When encoding a map[string]any as a record, missing keys fill from the
// schema's defaults. A ["null", T] field declared without a default has an
// implicit null default, so a missing key there fills null rather than
// erroring. The omitzero tag applies the same fill to a struct's zero-valued
// fields, and to fields whose IsZero() reports true: a zero value encodes the
// field's default, or null for a nullable field with no default, or — for a
// non-nullable field with no default — the zero value itself, there being
// nothing to fill with. It differs from map fill in one case, a [T, "null"]
// union declared without a default: no null default can exist there, since a
// union default must match the first branch, so omitzero encodes null where map
// fill errors on the missing key.
//
// Embedded (anonymous) struct fields are inlined automatically; an explicit
// name tag prevents it. When several fields resolve to one name, a tagged field
// wins over an untagged one at any depth, and among equally tagged fields the
// shallowest wins. Two fields at the same depth with the same tagged status are
// an ambiguous collision, and this package errors rather than pick one:
// [SchemaFor] rejects the type, while encode and decode reject only when the
// schema actually resolves a field to that name, so a coincidental collision on
// a name the schema never references does not break the type.
//
// # Custom types
//
// [CustomType] registers custom Go type conversions for logical types,
// domain types, or to replace built-in behavior. A matching custom type
// replaces the built-in logical type deserializer — Decode callbacks
// receive raw Avro-native values, not enriched types like [time.Time].
// A [CustomType] with nil Decode suppresses the built-in handler with
// zero overhead, producing raw values directly. Use [NewCustomType] for
// type-safe primitive conversions, or the [CustomType] struct directly
// for complex cases (records, fixed types, property-based dispatch).
// Custom types are registered per-schema via [SchemaOpt].
//
// # Parsing options
//
// [Parse] and [SchemaCache.Parse] accept [WithLaxNames] to allow
// non-standard characters in type and field names.
//
// # Errors
//
// Encode and decode errors can be inspected with [errors.As]:
//
//   - [*SemanticError]: type mismatch (includes a dotted field path for nested records)
//   - [*ShortBufferError]: input truncated mid-value
//   - [*CompatibilityError]: schema evolution incompatibility
//
// # Other features
//
//   - Schema Cache: [SchemaCache] accumulates named types across Parse calls for schema registry workflows
//   - Schema Introspection: [Schema.Root] returns a [SchemaNode]; [Schema.String] returns the original JSON
//   - Single Object Encoding: [Schema.AppendSingleObject], [Schema.DecodeSingleObject]
//   - Fingerprinting: [Schema.Canonical], [Schema.Fingerprint], [NewRabin]
//   - Object Container Files: the [github.com/twmb/avro/ocf] sub-package
//
// The repository README's "Encode/decode behavior contract" section
// documents the intentional asymmetries between the encoder and
// decoder (lossy-by-design conversions, spec/interop choices, and
// decoder-only leniencies).
//
// [Avro specification]: https://avro.apache.org/docs/current/specification/
package avro
