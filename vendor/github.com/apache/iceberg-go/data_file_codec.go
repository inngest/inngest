// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package iceberg

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/internal/datafileavro"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/twmb/avro"
)

// defaultSchemaCacheSize is the initial capacity of dataFileSchemaCache.
// 8192 covers a few thousand active tables (each contributing 1–3 entries
// for its partition spec × format version) which is what a long-running
// consumer like a server-side compaction service typically sees.
// SetSchemaCacheSize tunes this at runtime.
const defaultSchemaCacheSize = 8192

func init() {
	if datafileavro.Unmarshal != nil {
		panic("iceberg: datafileavro.Unmarshal already set")
	}
	datafileavro.Unmarshal = func(data []byte, spec, schema any, version int) (any, error) {
		s, ok := spec.(PartitionSpec)
		if !ok {
			return nil, fmt.Errorf("iceberg: datafileavro.Unmarshal: expected PartitionSpec, got %T", spec)
		}
		sc, ok := schema.(*Schema)
		if !ok {
			return nil, fmt.Errorf("iceberg: datafileavro.Unmarshal: expected *Schema, got %T", schema)
		}

		return unmarshalAvroDataFileEntry(data, s, sc, version)
	}
}

// AvroEntryMarshaler is implemented by DataFile values that can be
// encoded using the manifest-entry Avro encoding. The iceberg
// package's built-in DataFile implementation satisfies it; external
// implementations can also satisfy it to participate in the
// [github.com/apache/iceberg-go/codec] DataFile codec.
//
// The encoded bytes are the same bytes a manifest carries for this
// data file. Implementations must produce output that the iceberg
// package's manifest-entry Avro decoder accepts.
// The built-in DataFile implementation rejects projected or stat-stripped files
// with ErrInvalidArgument because omitted metadata cannot be recovered.
type AvroEntryMarshaler interface {
	MarshalAvroEntry(spec PartitionSpec, schema *Schema, version int) ([]byte, error)
}

// MarshalAvroEntry encodes this DataFile as Avro bytes using the
// manifest-entry encoding for the given partition spec, table schema
// and format version (1, 2, or 3). The wire format is the same one a
// manifest carries for this data file, so adding a field to the
// underlying struct (and its avro tags) automatically extends what
// MarshalAvroEntry transports — there is no separate wire-mirror
// struct to keep in sync.
//
// MarshalAvroEntry is the iceberg-package side of the
// [github.com/apache/iceberg-go/codec] DataFile codec; callers
// performing cross-process transport should prefer that package's
// high-level API.
//
// MarshalAvroEntry is safe to call concurrently with any other
// reader or encoder of the same DataFile: a fresh *dataFile is
// cloned (avro-tagged fields only) and the avro encoder reads, but
// does not mutate, the cloned values. Pointer-typed avro fields like
// ColSizes share their backing storage with the source; the
// thread-safety guarantee relies on the avro encoder being
// non-mutating.
//
// v1 note: the v1 manifest-entry schema has a non-nullable snapshot_id
// field. MarshalAvroEntry writes 0 there, so v1 bytes are not usable
// as a standalone manifest entry — they only round-trip via the
// matching decoder.
//
// distinct_counts (field 111) is deprecated in the spec for every
// version (apache/iceberg#12182). MarshalAvroEntry drops the field
// on encode for v1, v2, and v3 alike — values populated on the
// source DataFile are not transported. The Avro tag on the dataFile
// struct is intentionally retained so legacy manifests that already
// carry the field on the wire still decode through the matching
// reader. New DataFiles should not set distinct counts.
func (d *dataFile) MarshalAvroEntry(spec PartitionSpec, schema *Schema, version int) ([]byte, error) {
	if err := d.validateForWrite(); err != nil {
		return nil, err
	}
	if version < 1 || version > 3 {
		return nil, fmt.Errorf("iceberg: MarshalAvroEntry: unsupported format version %d", version)
	}
	s, maps, err := manifestEntrySchemaFor(spec, schema, version)
	if err != nil {
		return nil, err
	}
	clone := cloneDataFileAvroFields(d)
	partitionData, err := avroEncodePartitionData(
		d.DataFilePartitionRef(internal.DataFileRef{}), maps)
	if err != nil {
		return nil, err
	}
	clone.PartitionData = partitionData

	return s.Encode(newEncodeEntry(version, clone))
}

// unmarshalAvroDataFileEntry decodes Avro bytes produced by
// [(*dataFile).MarshalAvroEntry] back into a DataFile. The
// (spec, schema, version) triple must match the encoder; passing a
// different spec or version yields a decode error or silently
// mis-typed partition values.
//
// The returned DataFile carries the partition spec id and the field-id
// lookup tables, so Partition() and the stats accessors return id-keyed
// maps as if the file had been read from a manifest.
//
// It is reachable from the [github.com/apache/iceberg-go/codec]
// package through the [datafileavro] bridge.
func unmarshalAvroDataFileEntry(data []byte, spec PartitionSpec, schema *Schema, version int) (DataFile, error) {
	if version < 1 || version > 3 {
		return nil, fmt.Errorf("iceberg: unmarshalAvroDataFileEntry: unsupported format version %d", version)
	}
	s, maps, err := manifestEntrySchemaFor(spec, schema, version)
	if err != nil {
		return nil, err
	}
	entry, df := newDecodeEntry(version)
	if _, err := s.Decode(data, entry); err != nil {
		return nil, fmt.Errorf("iceberg: unmarshalAvroDataFileEntry: %w", err)
	}
	df.specID = int32(spec.ID())
	df.fieldNameToID = maps.nameToID
	df.fieldIDToLogicalType = maps.idToType
	df.fieldIDToDecimalScale = maps.idToDecimalScale

	return df, nil
}

// newEncodeEntry returns the right manifest-entry shape for the schema
// version: v1's manifest_entry has a non-nullable snapshot_id and uses
// [fallbackManifestEntry], v2/v3 use [manifestEntry] with nullable
// pointers.
func newEncodeEntry(version int, df *dataFile) any {
	if version == 1 {
		return &fallbackManifestEntry{
			manifestEntry: manifestEntry{EntryStatus: EntryStatusADDED, Data: df},
		}
	}

	return &manifestEntry{EntryStatus: EntryStatusADDED, Data: df}
}

// newDecodeEntry mirrors [newEncodeEntry] for the read side: it returns
// the pointer to pass to avro.Schema.Decode along with the pre-allocated
// *dataFile that will be populated.
func newDecodeEntry(version int) (any, *dataFile) {
	df := &dataFile{}
	if version == 1 {
		return &fallbackManifestEntry{manifestEntry: manifestEntry{Data: df}}, df
	}

	return &manifestEntry{Data: df}, df
}

var dataFileAvroFieldIndexes = avroFieldIndexes(reflect.TypeOf(dataFile{}))

func avroFieldIndexes(t reflect.Type) []int {
	indexes := make([]int, 0, t.NumField())
	for i := range t.NumField() {
		if _, hasAvroTag := t.Field(i).Tag.Lookup("avro"); hasAvroTag {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// cloneDataFileAvroFields returns a fresh *dataFile populated with src's
// avro-tagged fields. Internal state (sync.Once, lazy-init caches,
// specID, the field-id lookup maps) is intentionally left at zero
// values because the avro encoder reads only the avro-tagged fields.
//
// The avro-tagged field indexes are discovered once when the package is
// initialized. This keeps dataFile as the single source of truth for the
// wire shape while avoiding a reflect.Type and StructTag lookup for every
// encoded manifest entry. It also sidesteps the go-vet copies-lock warning
// that would fire on a struct-literal copy of *dataFile (it embeds sync.Once).
//
// Note: this is a shallow copy. Pointer-typed avro fields (ColSizes,
// LowerBounds, etc.) share their backing storage with the source.
// The no-mutation guarantee of MarshalAvroEntry depends on the avro
// encoder being read-only on the values it walks; TestMarshalAvroEntry
// DoesNotMutate asserts this end-to-end across every avro-tagged
// field, so a future regression in the encoder surfaces in tests.
func cloneDataFileAvroFields(src *dataFile) *dataFile {
	out := &dataFile{}
	srcVal := reflect.ValueOf(src).Elem()
	outVal := reflect.ValueOf(out).Elem()
	for _, i := range dataFileAvroFieldIndexes {
		outVal.Field(i).Set(srcVal.Field(i))
	}

	return out
}

// avroEncodePartitionData converts an id-keyed partition tuple (carrying
// iceberg-typed values like Date or Decimal) into the name-keyed
// avro-friendly map the manifest-entry schema expects. Idempotent:
// values already in primitive form pass through unchanged.
func avroEncodePartitionData(idKeyed map[int]any, fields dataFileFieldMaps) (map[string]any, error) {
	out := make(map[string]any, len(fields.nameToID))
	for name, id := range fields.nameToID {
		if _, unknown := fields.unknownFieldIDs[id]; unknown {
			// UnknownType has no value representation and is encoded as Avro
			// null. Normalize historical values whose source type was dropped.
			out[name] = nil

			continue
		}
		v, ok := idKeyed[id]
		if !ok {
			continue
		}
		if logical, ok := fields.idToType[id]; ok {
			converted, err := convertLogicalTypeValue(v, logical, fields.idToFixedSize[id])
			if err != nil {
				return nil, fmt.Errorf("failed to convert partition field %d: %w", id, err)
			}
			out[name] = converted
		} else {
			out[name] = v
		}
	}

	return out, nil
}

type dataFileFieldMaps struct {
	nameToID         map[string]int
	idToType         map[int]string
	idToFixedSize    map[int]int
	idToDecimalScale map[int]int
	unknownFieldIDs  map[int]struct{}
}

// dataFileSchemaCacheKey identifies a cached Avro schema by the ordered
// partition field IDs, names, result types, and format version. These determine
// the partition Avro shape without including docs or table-local schema/spec IDs.
type dataFileSchemaCacheKey struct {
	partAvroFingerprint string
	version             int
}

type dataFileSchemaEntry struct {
	schema *avro.Schema
	maps   dataFileFieldMaps
}

var dataFileSchemaCache = mustNewSchemaCache(defaultSchemaCacheSize)

func mustNewSchemaCache(size int) *lru.Cache[dataFileSchemaCacheKey, *dataFileSchemaEntry] {
	c, err := lru.New[dataFileSchemaCacheKey, *dataFileSchemaEntry](size)
	if err != nil {
		panic(fmt.Sprintf("iceberg: schema cache size %d invalid: %v", size, err))
	}

	return c
}

// SetSchemaCacheSize resizes the manifest-entry schema cache used by
// the DataFile codec. The default capacity is sized for a few thousand
// active partition specs; long-running consumers with larger working
// sets (e.g. a compaction service touching many tables) should raise
// it. Existing entries are preserved on grow; on shrink, least-recently
// used entries are evicted down to the new size. Safe to call concurrently
// with codec operations: the underlying golang-lru/v2 cache serializes Resize
// against Get/Add through the same mutex.
func SetSchemaCacheSize(size int) error {
	if size <= 0 {
		return fmt.Errorf("iceberg: SetSchemaCacheSize: size must be positive, got %d", size)
	}
	dataFileSchemaCache.Resize(size)

	return nil
}

// manifestEntrySchemaFor returns the cached avro schema and partition
// field-id lookups for the given partition type and format version.
// The cache key fingerprints the partition Avro shape, so specs that
// differ only in field documentation share a single entry.
func manifestEntrySchemaFor(spec PartitionSpec, schema *Schema, version int) (*avro.Schema, dataFileFieldMaps, error) {
	fingerprint, err := partitionSchemaFingerprint(spec, schema)
	if err != nil {
		return nil, dataFileFieldMaps{}, err
	}
	key := dataFileSchemaCacheKey{partAvroFingerprint: fingerprint, version: version}
	if cached, ok := dataFileSchemaCache.Get(key); ok {
		return cached.schema, cached.maps, nil
	}
	partSchema, err := partitionTypeToAvroSchema(spec.PartitionType(schema))
	if err != nil {
		return nil, dataFileFieldMaps{}, err
	}
	fullSchema, err := internal.NewManifestEntrySchema(partSchema, version)
	if err != nil {
		return nil, dataFileFieldMaps{}, err
	}
	fieldMaps := getFieldIDMap(fullSchema)
	entry := &dataFileSchemaEntry{
		schema: fullSchema,
		maps:   fieldMaps,
	}
	dataFileSchemaCache.Add(key, entry)

	return entry.schema, entry.maps, nil
}

func partitionSchemaFingerprint(spec PartitionSpec, schema *Schema) (string, error) {
	var key strings.Builder
	for _, field := range spec.fields {
		sourceType := Type(UnknownType{})
		// ResultType only inspects the source type here; borrow it to avoid
		// cloning nested source types on every schema-cache hit.
		if sourceField, ok := schema.FindFieldByIDRef(field.SourceID(), internal.SchemaRef{}); ok {
			sourceType = sourceField.Type
		}
		resultType := field.Transform.ResultType(sourceType)
		// Reserve room for the ID, name length, type tag, and typical parameters.
		key.Grow(len(field.Name) + 32)

		// Length prefixes keep names unambiguous. The compact type tag also
		// includes fixed lengths and decimal precision/scale without formatting
		// a temporary type string on every cache hit.
		writePartitionFingerprintInt(&key, field.FieldID)
		writePartitionFingerprintInt(&key, len(field.Name))
		key.WriteString(field.Name)
		if err := writePartitionTypeFingerprint(&key, resultType); err != nil {
			return "", err
		}
	}

	return key.String(), nil
}

// Keep the supported types and parameters in sync with partitionTypeToAvroSchema.
// TestPartitionTypeFingerprintMatchesAvroConversion checks both paths together.
func writePartitionTypeFingerprint(key *strings.Builder, typ Type) error {
	// These tags are internal to the cache key. Parameterized types append their
	// parameters so equal Avro partition shapes still produce equal keys.
	switch t := typ.(type) {
	case Int32Type:
		key.WriteByte('i')
	case Int64Type:
		key.WriteByte('j')
	case Float32Type:
		key.WriteByte('k')
	case Float64Type:
		key.WriteByte('d')
	case StringType:
		key.WriteByte('S')
	case DateType:
		key.WriteByte('D')
	case TimeType:
		key.WriteByte('T')
	case TimestampType:
		key.WriteByte('t')
	case TimestampTzType:
		key.WriteByte('z')
	case TimestampNsType:
		key.WriteByte('n')
	case TimestampTzNsType:
		key.WriteByte('N')
	case UUIDType:
		key.WriteByte('u')
	case BooleanType:
		key.WriteByte('b')
	case BinaryType:
		key.WriteByte('B')
	case FixedType:
		key.WriteByte('f')
		writePartitionFingerprintInt(key, t.Len())
	case DecimalType:
		key.WriteByte('q')
		writePartitionFingerprintInt(key, t.Precision())
		writePartitionFingerprintInt(key, t.Scale())
	case UnknownType:
		key.WriteByte('U')
	default:
		return fmt.Errorf("unsupported partition type: %s", typ.String())
	}

	return nil
}

func writePartitionFingerprintInt(key *strings.Builder, value int) {
	var buf [20]byte
	key.Write(strconv.AppendInt(buf[:0], int64(value), 10))
	key.WriteByte(':')
}
