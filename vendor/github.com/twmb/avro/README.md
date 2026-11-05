# avro

[![Go Reference](https://pkg.go.dev/badge/github.com/twmb/avro.svg)](https://pkg.go.dev/github.com/twmb/avro)

Encode and decode [Avro](https://avro.apache.org/docs/current/specification/) binary data.

This project aims to be the "best" Avro encoder/decoder in the Go ecosystem by:
* Keeping the API tight but comprehensive
* Combining features that exist in only one of linkedin/goavro or hamba/avro
* Being safe above all, while being fast with `unsafe` specialization functions internally
* Running round after round of AI audits to shake out _any_ bug / DoS / huge alloc that exists
* Being documented thoroughly for both human and AI users

For the "why" of this project, see the [Why](#why) section.

## Index

- [Quick Start](#quick-start)
- [Type Mapping](#type-mapping)
- [Struct Tags](#struct-tags)
- [Schema Inference](#schema-inference)
- [Schema Introspection](#schema-introspection)
- [Logical Types](#logical-types)
- [Schema Evolution](#schema-evolution)
- [Schema Cache](#schema-cache)
- [Custom Types](#custom-types)
- [Type name constants](#type-name-constants)
- [Object Container Files](#object-container-files)
- [JSON Encoding](#json-encoding)
- [Single Object Encoding](#single-object-encoding)
- [Fingerprinting](#fingerprinting)
- [Errors](#errors)
- [Performance](#performance)
- [Encode/decode behavior contract](#encodedecode-behavior-contract)
- [Why](#why)

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/twmb/avro"
)

var schema = avro.MustParse(`{
    "type": "record",
    "name": "User",
    "fields": [
        {"name": "name", "type": "string"},
        {"name": "age",  "type": "int"}
    ]
}`)

type User struct {
	Name string `avro:"name"`
	Age  int    `avro:"age"`
}

func main() {
	data, err := schema.Encode(&User{Name: "Alice", Age: 30})
	if err != nil {
		log.Fatal(err)
	}

	var u User
	_, err = schema.Decode(data, &u)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u) // {Alice 30}
}
```

## Type Mapping

All types can decode into `any`.

| Avro Type | Encode | Decode |
|-----------|--------|--------|
| null      | anything nillable | zeroes the target |
| boolean   | `bool` | `bool` |
| int, long, float, double | `numeric` † | `numeric` † |
| string    | `string []byte TextAppender TextMarshaler` | `string []byte TextUnmarshaler` |
| bytes     | `bytes-like` | `bytes-like` ‡ |
| enum      | `string integer TextAppender TextMarshaler` | `string integer TextUnmarshaler` |
| fixed     | `bytes-like` ‡ | `bytes-like` ‡ |
| array     | slice or `[N]array` | slice or `[N]array` ‡ |
| map       | `map[string]T` | `map[string]T` |
| union     | `*T`, tagged-union map, or the matched branch | `*T` or the matched branch |
| record    | struct or `map[string]any` | struct or `map[string]any` |

Shorthands used above:
`numeric` = `int int8–int64 uint uint8–uint64 float32 float64 json.Number`.
`integer` = `int int8–int64 uint uint8–uint64`.
`bytes-like` = `[]byte [N]byte string`.
`TextAppender` / `TextMarshaler` / `TextUnmarshaler` are the standard `encoding` interfaces.

† Numeric types accept any `numeric` Go type, but coercion has precision rules —
  whole-number/range checks into integers, silent IEEE rounding into floats. See
  [Encode/decode behavior contract](#encodedecode-behavior-contract).
‡ Length must match: decoding bytes into `[N]byte`, or encoding/decoding any value
  as fixed, requires the byte length to equal N; `[N]array` requires exactly N elements.

Decoding into `any` yields the natural Go type — `int32`/`int64` for int/long,
`float32`/`float64` for float/double, `[]any` for arrays, `map[string]any` for maps
and records; logical types use the Go types in [Logical Types](#logical-types). A
null (e.g. a union's null branch) decodes to the target's zero value, replacing any
prior contents — use `*T` to tell null from zero.

## Struct Tags

Struct fields are matched to Avro record fields by name. Use the `avro` struct
tag to control the mapping:

```go
type Example struct {
    Name    string  `avro:"name"`          // maps to Avro field "name"
    Ignored int     `avro:"-"`             // excluded from encoding/decoding
    Inner   Nested  `avro:",inline"`       // inline Nested's fields into this record
    Value   int     `avro:"val,omitzero"`  // encode zero value as Avro default
}
```

The tag format is:

```
avro:"[name][,option][,option]..."
```

The name portion maps the struct field to the Avro field with that name. If
empty, the Go field name is used as-is. A tag of `"-"` excludes the field
entirely.

Supported options:

- **inline**: flatten a nested struct's fields into the parent record, as if
  they were declared directly on the parent. The field must be a struct or
  pointer to struct. This works like anonymous (embedded) struct fields, but
  for named fields. When using inline, the name portion of the tag must be
  empty and no other tag options are allowed — the flattened embed has no
  field of its own for `default=`, `alias=`, logical-type tags, etc. to
  apply to. Put those options on the embed's child fields directly.

- **omitzero**: when encoding, a zero value (or a field whose `IsZero() bool`
  method returns true) is encoded as if its key were omitted from a
  `map[string]any` — the field's schema default is used. A nullable field with
  no default encodes `null`; a non-nullable field with no default keeps its zero
  value (there is nothing to fill with). Useful for optional fields in
  `["null", T]` / `["T", "null"]` unions or fields with explicit defaults.

Embedded (anonymous) struct fields are automatically inlined — their fields are
promoted into the parent as if declared directly. To prevent inlining an
embedded struct, give it an explicit name tag:

```go
type Parent struct {
    Nested                    // inlined: Nested's fields are promoted
    Other  Aux `avro:"other"` // not inlined: treated as a single field
}
```

When multiple fields resolve to the same Avro field name, a tagged field wins
over an untagged one at any depth; among fields with the same tagged status,
the shallowest field wins. Two fields that resolve to the same name at the same depth
with the same tagged status are an ambiguous collision (Go itself makes such a
field reference a compile error). twmb errors rather than silently selecting
one: `SchemaFor` rejects the type, while encode and decode reject only when the
schema actually resolves a field to the ambiguous name — a coincidental
collision on a name the schema never references does not break the type.

## Schema Inference

`SchemaFor` infers an Avro schema from a Go struct type, using the same struct
tags as encoding/decoding:

```go
type User struct {
    Name      string     `avro:"name"`
    Age       int32      `avro:"age,default=18"`
    Email     *string    `avro:"email"`
    CreatedAt time.Time  `avro:"created_at"`
}

schema := avro.MustSchemaFor[User](avro.WithNamespace("com.example"))
```

This produces the equivalent of:

```json
{
  "type": "record",
  "name": "User",
  "namespace": "com.example",
  "fields": [
    {"name": "name", "type": "string"},
    {"name": "age", "type": "int", "default": 18},
    {"name": "email", "type": ["null", "string"]},
    {"name": "created_at", "type": {"type": "long", "logicalType": "timestamp-millis"}}
  ]
}
```

Go types map to Avro types automatically: `*T` becomes a `["null", T]` union
with `"default": null` (backward-compatible by default), `[N]byte` becomes a
fixed type named after the Go type (or `fixed_N` for unnamed arrays),
`time.Time` becomes `timestamp-millis`, and so on (see [Type Mapping](#type-mapping)).

Additional tag options for schema inference:

| Tag | Example | Description |
|-----|---------|-------------|
| `default=` | `avro:",default=0"` | Default value (must be last; scalars only) |
| `alias=` | `avro:",alias=old"` | Field alias for schema evolution (repeatable, or `alias=[a,b]`) |
| `type-alias=` | `avro:",type-alias=old"` | Alias for the field's named type — record, enum, or fixed (repeatable, or `type-alias=[a,b]`) |
| `timestamp-micros` | `avro:",timestamp-micros"` | Override logical type |
| `decimal(p,s)` | `avro:",decimal(10,2)"` | Decimal logical type (requires `*big.Rat` or `big.Rat`) |
| `uuid` | `avro:",uuid"` | UUID logical type (requires Go string, `[16]byte`, or a text marshaler type) |
| `date` | `avro:",date"` | Date logical type (requires `time.Time`, `time.Duration`, or an int8/16/32/uint8/uint16) |

`alias` and `type-alias` serve different purposes in schema evolution. `alias`
adds an alias to the **field** — it lets a writer field with a different name
match this reader field. `type-alias` adds an alias to the **named type**
(record, enum, or fixed) that the field references — it lets a writer type with
a different name match the reader type. The alias is applied to the innermost
named type, walking through pointers, slices, and maps:

```go
type FieldSummary struct {
    ContainsNull bool    `avro:"contains_null"`
    ContainsNaN  *bool   `avro:"contains_nan"`
}

type ManifestFile struct {
    // alias=old_partitions: match writer field named "old_partitions"
    // type-alias=r508:      match writer record type named "r508" for FieldSummary
    Partitions *[]FieldSummary `avro:"partitions,type-alias=r508"`
}
```

Options:

- `WithNamespace(ns)` sets the Avro namespace for the record.
- `WithName(name)` overrides the record name (defaults to the Go struct name).

## Schema Introspection

`Schema.Root()` returns a `SchemaNode` representing the parsed schema. This
provides read access to all schema metadata including field types, logical
types, doc strings, and custom properties:

```go
schema, _ := avro.Parse(schemaJSON)
root := schema.Root()

for _, f := range root.Fields {
    fmt.Printf("field %s: type=%s\n", f.Name, f.Type.Type)
    if cn, ok := f.Props["connect.name"].(string); ok {
        fmt.Printf("  kafka connect type: %s\n", cn)
    }
}
```

`SchemaNode` can also be used to build schemas programmatically:

```go
node := &avro.SchemaNode{
    Type: "record",
    Name: "User",
    Fields: []avro.SchemaField{
        {Name: "name", Type: avro.SchemaNode{Type: "string"}},
        {Name: "age", Type: avro.SchemaNode{Type: "int"}, Default: 18},
    },
}
schema, err := node.Schema()
```

Reserved attribute names (`type`, `name`, `items`, ...) match only their
exact lowercase spelling, as in the Java, Python (fastavro), and goavro
implementations. A key differing from a reserved name only by letter case
(`"ITEMS"`, `"Namespace"`) is an ordinary custom property, preserved in
`Props`. One migration note: hamba/avro matches these keys
case-insensitively, so a schema that parsed there but fails here has a
miscased reserved key. A miscased *structural* key fails loudly at parse
time — `{"type":"array","Items":"int"}` errors with "array is missing
items schema"; fix the casing — while a miscased non-structural key
(`"Doc"`, `"Aliases"`) simply becomes a harmless custom property instead
of binding the attribute.

## Logical Types

Logical types decode to their natural Go equivalents:

| Logical Type | Avro Type | Encode | Decode |
|---|---|---|---|
| date | int | time.Time, RFC 3339 or YYYY-MM-DD string, or int | time.Time (UTC) |
| time-millis | int | time.Duration, time.Time (lossy †), or int | time.Duration or time.Time (lossy †) |
| time-micros | long | time.Duration, time.Time (lossy †), or int | time.Duration or time.Time (lossy †) |
| timestamp-millis | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| timestamp-micros | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| timestamp-nanos | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| local-timestamp-millis | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| local-timestamp-micros | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| local-timestamp-nanos | long | time.Time, RFC 3339 string, or int | time.Time (UTC) |
| uuid (string) | string | [16]byte or string | string into any; [16]byte or string into typed target |
| uuid (fixed(16)) | fixed(16) | [16]byte, []byte, or hex-dash string | [16]byte into any or [16]byte target; string into string target |
| decimal | bytes or fixed | *big.Rat, float64, numeric string, json.Number, or underlying type | *big.Rat, float64/float32, numeric string, json.Number, or underlying type |
| big-decimal | bytes | *big.Rat, float64, numeric string, json.Number | *big.Rat, float64/float32, numeric string, json.Number |
| duration | fixed(12) | avro.Duration or underlying type | avro.Duration or underlying type |

When encoding, timestamp and date fields accept RFC 3339 strings, and decimal
fields accept float64 and numeric strings (e.g. "3.14"). Values that don't
match the expected format fall through to the underlying type's encoder, which
will return an error.

† **time-millis / time-micros with `time.Time`** is a convenience escape hatch
for users whose Go data already lives in `time.Time`. Avro's `time-millis` and
`time-micros` logical types are time-of-day only — the wire bytes physically
cannot represent a date or zone. On encode, only the wall-clock fields
(hour/minute/second/nanosecond) are written; year/month/day/location are
silently discarded. On decode into a `time.Time` target, the wire value is
materialized at the Unix epoch (`1970-01-01 UTC`) plus the time-of-day. A
round-trip through `time.Time` therefore preserves the time-of-day but
resets the date and zone: `2024-01-15 12:34:56 PST` → wire → `1970-01-01
12:34:56 UTC`. If round-trip date fidelity matters, use `timestamp-millis` /
`timestamp-micros` (which preserve the full instant) or convert to and from
`time.Duration` explicitly. `time.Duration` round-trips exactly when its
nanosecond component is a whole multiple of the schema's resolution unit
(millisecond for `time-millis`, microsecond for `time-micros`); sub-resolution
nanoseconds are silently truncated toward zero on encode (integer division
by the resolution unit, dropping the remainder).

big-decimal carries no schema-level precision or scale; scale is derived
per value, and rationals with no finite decimal expansion (e.g.
`big.NewRat(1, 3)`) return an error.

Unknown logical types are silently ignored per the Avro spec, and the
underlying type is used as-is.

## Schema Evolution

Avro data is always written with a specific schema — the **writer schema**.
When you read that data later, your application may expect a different schema —
the **reader schema**. You may have added a field, removed one, or widened a
type from int to long.

`Resolve` bridges this gap. Given the writer and reader schemas, it returns a
new schema that decodes data in the old wire format and produces values in the
reader's layout:

- Fields in the reader but not the writer are filled from **defaults**.
- Fields in the writer but not the reader are **skipped**.
- Fields that exist in both are matched by **name** (or **alias**) and decoded,
  with type promotion applied where needed (e.g. int → long).

### Example

Suppose v1 of your application wrote User records with just a name:

```go
var writerSchema = avro.MustParse(`{
    "type": "record", "name": "User",
    "fields": [
        {"name": "name", "type": "string"}
    ]
}`)
```

In v2 you added an email field with a default:

```go
var readerSchema = avro.MustParse(`{
    "type": "record", "name": "User",
    "fields": [
        {"name": "name",  "type": "string"},
        {"name": "email", "type": "string", "default": ""}
    ]
}`)

type User struct {
    Name  string `avro:"name"`
    Email string `avro:"email"`
}
```

To read old v1 data with your v2 struct, resolve the two schemas:

```go
resolved, err := avro.Resolve(writerSchema, readerSchema)

var u User
_, err = resolved.Decode(v1Data, &u)
// u == User{Name: "Alice", Email: ""}
```

The following type promotions are supported:

| Writer → Reader |
|---|
| int → long, float, double |
| long → float, double |
| float → double |
| string ↔ bytes |

`CheckCompatibility` checks whether two schemas are compatible without
building a resolved schema. The direction you check depends on the guarantee
you need:

```go
// Backward: new schema can read old data.
avro.CheckCompatibility(oldSchema, newSchema)

// Forward: old schema can read new data.
avro.CheckCompatibility(newSchema, oldSchema)

// Full: check both directions.
avro.CheckCompatibility(oldSchema, newSchema)
avro.CheckCompatibility(newSchema, oldSchema)
```

## Schema Cache

When working with a schema registry, schemas often reference types defined in
other schemas. `SchemaCache` accumulates named types across multiple Parse
calls so they can be resolved:

```go
var cache avro.SchemaCache

// Parse referenced schema first — order matters.
_, err := cache.Parse(`{
    "type": "record",
    "name": "Address",
    "fields": [{"name": "city", "type": "string"}]
}`)

// Now parse a schema that references Address.
schema, err := cache.Parse(`{
    "type": "record",
    "name": "User",
    "fields": [
        {"name": "name",    "type": "string"},
        {"name": "address", "type": "Address"}
    ]
}`)
```

Parsing the same schema string multiple times returns the cached result,
handling diamond dependencies without caller-side deduplication. The returned
`*Schema` is independent of the cache and safe to use concurrently.

## Custom Types

Register custom Go type conversions with `NewCustomType` for type-safe
primitive conversions, or `CustomType` for advanced cases:

```go
type Money struct {
    Cents    int64
    Currency string
}

moneyType := avro.NewCustomType[Money, int64]("money",
    func(m Money, _ *avro.SchemaNode) (int64, error) { return m.Cents, nil },
    func(c int64, _ *avro.SchemaNode) (Money, error) {
        return Money{Cents: c, Currency: "USD"}, nil
    },
)

schema := avro.MustParse(moneySchema, moneyType)

// Encode and decode — Money fields are automatically converted.
data, _ := schema.Encode(&order)
var out Order
schema.Decode(data, &out) // out.Price is Money{Cents: 500, ...}

// Works with SchemaFor too.
schema = avro.MustSchemaFor[Order](moneyType)
```

A matching custom type replaces the built-in logical type deserializer.
Decode callbacks receive raw Avro-native values (int64 for long, int32
for int, etc.). A nil Decode suppresses the built-in handler with zero
overhead, producing raw values directly:

```go
// Decode timestamps as raw int64 instead of time.Time.
schema := avro.MustParse(raw, avro.CustomType{
    LogicalType: "timestamp-millis",
    AvroType:    "long",
})
```

For property-based dispatch (e.g., Kafka Connect / Debezium types), use
an empty matching criteria with `ErrSkipCustomType`:

```go
avro.CustomType{
    Decode: func(v any, node *avro.SchemaNode) (any, error) {
        name, _ := node.Props["connect.name"].(string)
        switch name {
        case "io.debezium.time.Timestamp":
            return time.UnixMilli(v.(int64)).UTC(), nil
        default:
            return nil, avro.ErrSkipCustomType
        }
    },
}
```

## Type name constants

The `atype` sub-package exports constants for Avro primitive type names,
complex type names, logical type names, and field sort orders — the string
values used in `SchemaNode`, `SchemaField`, and `CustomType`:

```go
import (
    "github.com/twmb/avro"
    "github.com/twmb/avro/atype"
)

node := avro.SchemaNode{
    Type:        atype.Long,
    LogicalType: atype.TimestampMicros,
}

ct := avro.CustomType{LogicalType: atype.Decimal}
```

They are untyped string constants and can be used anywhere a string is
expected. Helps catch typos (`atype.TimestampMicros` vs
`"timestam-micros"`) at compile time.

## Object Container Files

The `ocf` sub-package reads and writes [Avro Object Container Files](https://avro.apache.org/docs/current/specification/#object-container-files) —
self-describing binary files that embed the schema in the header and store
data in compressed blocks.

### Writing

```go
var schema = avro.MustParse(`{
    "type": "record",
    "name": "User",
    "fields": [
        {"name": "name", "type": "string"},
        {"name": "age",  "type": "int"}
    ]
}`)

f, _ := os.Create("users.avro")
w, err := ocf.NewWriter(f, schema, ocf.WithCodec(ocf.SnappyCodec()))
if err != nil {
    log.Fatal(err)
}
w.Encode(&User{Name: "Alice", Age: 30})
w.Encode(&User{Name: "Bob", Age: 25})
w.Close()
f.Close()
```

### Reading

```go
f, _ := os.Open("users.avro")
r, err := ocf.NewReader(f)
if err != nil {
    log.Fatal(err)
}
defer r.Close()
for {
    var u User
    err := r.Decode(&u)
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(u)
}
```

The reader's `Schema()` method returns the schema parsed from the file header,
which you can pass as the writer schema to `Resolve`.

### Codecs

Built-in codecs: **null** (default, no compression), **deflate**
(`DeflateCodec`), **snappy** (`SnappyCodec`), and **zstandard** (`ZstdCodec`).
Custom codecs can be provided via the `Codec` interface.

**Memory bounds.** `WithMaxBlockBytes` (default 64 MiB) caps the *compressed*
block size, not the *decompressed* size — snappy, deflate, and zstd all
allocate the declared decompressed length before validating the payload, so
a malicious block can drive a larger allocation than the cap allows. For
OCF read from untrusted sources, bound memory at the transport or process
layer (request size cap, cgroup limit).

### Appending

`NewAppendWriter` opens an existing OCF for appending — it reads the header to
recover the schema, codec, and sync marker, then seeks to the end.

## JSON Encoding

`EncodeJSON` is a schema-aware JSON serializer. By default it produces standard
JSON with bare union values and `\uXXXX`-encoded bytes:

```go
// Standard JSON (default): bare unions
jsonBytes, err := schema.EncodeJSON(&user)
// {"name":"Alice","email":"a@b.com"}

// Avro JSON: unions wrapped as {"type_name": value}
jsonBytes, err = schema.EncodeJSON(&user, avro.TaggedUnions())
// {"name":"Alice","email":{"string":"a@b.com"}}
```

The tagged form is the Avro JSON spec's representation and what Java and other Avro
tools require; the bare default is plainer JSON for non-Avro consumers.

`DecodeJSON` accepts both formats (tagged and bare unions) and all NaN/Infinity
conventions:

```go
var user User
err = schema.DecodeJSON(jsonBytes, &user)
```

`Decode` and `DecodeJSON` also accept `TaggedUnions()` to wrap union values
when decoding into `*any`:

```go
var native any
schema.Decode(binary, &native, avro.TaggedUnions())
// native["email"] is map[string]any{"string": "a@b.com"}
```

`Encode` and `DecodeJSON` accept both tagged and bare union input, so
tagged union output from `Decode` can round-trip through `Encode` directly.

Pass `TagLogicalTypes()` with `TaggedUnions()` to qualify union branch names
with their logical type (e.g. `"long.timestamp-millis"` instead of `"long"`),
matching the linkedin/goavro naming convention.

NaN and Infinity float values are encoded as `"NaN"`, `"Infinity"`, `"-Infinity"`
strings by default (Java Avro convention). Pass `LinkedinFloats()` for
the linkedin/goavro convention (`null` for NaN, `±1e999` for Infinity).

## Single Object Encoding

For sending self-describing values over the wire (as opposed to files, where
OCF is preferred), use Single Object Encoding. Each message is a 2-byte magic
header, an 8-byte CRC-64-AVRO fingerprint, and the Avro binary payload.

```go
// Encode with fingerprint header
data, err := schema.AppendSingleObject(nil, &user)

// Decode (schema known)
_, err = schema.DecodeSingleObject(data, &user)

// Decode (schema unknown): extract fingerprint, look up schema
fp, payload, err := avro.SingleObjectFingerprint(data)
schema := registry.Lookup(fp) // your schema registry
_, err = schema.Decode(payload, &user)
```

## Fingerprinting

`Canonical` returns the [Parsing Canonical Form](https://avro.apache.org/docs/current/specification/#parsing-canonical-form-for-schemas)
of a schema — a deterministic JSON representation stripped of doc, aliases,
defaults, and other non-essential attributes. Use it for schema comparison and
fingerprinting.

```go
canonical := schema.Canonical() // []byte

// CRC-64-AVRO (Rabin) — the Avro-standard fingerprint
fp := schema.Fingerprint(avro.NewRabin())

// SHA-256 — common for cross-language registries
fp256 := schema.Fingerprint(sha256.New())
```

## Errors

Encode and decode errors can be inspected with `errors.As`:

- **`*SemanticError`**: type mismatch between Go and Avro (includes a dotted
  field path for nested records, e.g. `"address.zip"`).
- **`*ShortBufferError`**: input truncated mid-value.
- **`*CompatibilityError`**: schema evolution incompatibility (from `Resolve`
  or `CheckCompatibility`).

## Performance

Struct field access uses `unsafe` pointer arithmetic (similar to
`encoding/json` v2) to avoid `reflect.Value` overhead on every encode/decode.
All schemas, type mappings, and codec state are cached after first use so
repeated operations pay no extra allocation cost.

## Encode/decode behavior contract

The encoder and decoder are mostly symmetric: any Go shape the encoder accepts
as input is a Go shape the decoder accepts as a target, and an encode→decode
round-trip through the same Go type yields the same value. The cases below are
deliberate exceptions.

### Round-trips that lose data

- **`time.Time` → `time-millis` / `time-micros`** keeps only the time-of-day;
  date and zone are dropped. The decoded `time.Time` sits at the Unix epoch with
  the original time-of-day. Use `timestamp-*` to keep the full instant.
- **`time.Time` → `date`** keeps only the UTC date; time-of-day and zone are dropped.
- **`big-decimal` trailing-zero scale** isn't preserved — scale is derived per
  value, and `big.Rat` can't carry the distinction.

### Numbers

- **Precision follows the reader schema.** A `float`/`double` schema rounds
  silently into a Go float (overflowing to ±Inf on encode), but decoding one into
  a Go *integer* still requires a whole number in range. An `int`/`long` schema
  never loses precision silently — decoding into a Go type that can't hold the
  value exactly errors. For exact large-integer round-trips keep the schema `long`
  with an `int64` target; evolving to `double` opts into rounding.
- **Union branch is chosen by the Go type, not the value.** A Go `int` always
  selects a union's `long` branch, never `int`, even when the value would fit —
  dispatch keys off the static type, keeping wire size deterministic. Force the
  `int` branch with an `int32` value or `map[string]any{"int": v}`.
- **Encoding a `json.Number` into an `int`/`long` works even in fractional or
  exponent form, as long as the value is whole** — `json.Number("9.5e17")`
  succeeds (it's exactly 950000000000000000). Caveat: the same literal as a schema
  *default* (`"default":9.5e17`) won't load in Java; use the plain integer form if
  you publish schemas to Java consumers.

## Why

I had efficient encode-side code lying around for two years. I'd written it one
way, started rewriting it another, got most of the way through, and decided it
wasn't worth the effort — hamba/avro was around by then and really good, so I
stopped.

When I started using LLMs in January 2026, I figured I'd touch up all the old
projects I had lying around. Did it for this one — and coincidentally, hamba/avro
got archived right around then. I wanted one library that did it all: hamba/avro
missed things linkedin/goavro had, and vice versa. Here we are.
