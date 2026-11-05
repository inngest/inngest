package icebergcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// tableMetadata is Iceberg table metadata (format v2/v3, see
// https://iceberg.apache.org/spec/#table-metadata-fields). It's the catalog's
// whole model of a table: commits apply updates to it, it's validated as-is,
// and it's what gets serialized into `<n>-<uuid>.metadata.json`, so what we
// check is exactly what readers see.
//
// Schemas, partition specs and sort orders are kept as (duplicate-key free)
// JSON objects. We only read/assign their ids, everything else passes
// through untouched.
type tableMetadata struct {
	FormatVersion       int                       `json:"format-version"`
	TableUUID           string                    `json:"table-uuid"`
	Location            string                    `json:"location"`
	LastSequenceNumber  int64                     `json:"last-sequence-number"`
	LastUpdatedMS       int64                     `json:"last-updated-ms"`
	LastColumnID        int                       `json:"last-column-id"`
	Schemas             []jsonObject              `json:"schemas"`
	CurrentSchemaID     int                       `json:"current-schema-id"`
	PartitionSpecs      []jsonObject              `json:"partition-specs"`
	DefaultSpecID       int                       `json:"default-spec-id"`
	LastPartitionID     int                       `json:"last-partition-id"`
	Properties          map[string]string         `json:"properties,omitempty"`
	CurrentSnapshotID   *int64                    `json:"current-snapshot-id,omitempty"`
	Snapshots           []snapshot                `json:"snapshots,omitempty"`
	SnapshotLog         []snapshotLogEntry        `json:"snapshot-log,omitempty"`
	MetadataLog         []metadataLogEntry        `json:"metadata-log,omitempty"`
	SortOrders          []jsonObject              `json:"sort-orders"`
	DefaultSortOrderID  int                       `json:"default-sort-order-id"`
	Refs                map[string]snapshotRef    `json:"refs,omitempty"`
	Statistics          []statisticsFile          `json:"statistics,omitempty"`
	PartitionStatistics []partitionStatisticsFile `json:"partition-statistics,omitempty"`
	// NextRowID is v3 row lineage, nil before v3.
	NextRowID *int64 `json:"next-row-id,omitempty"`
}

type snapshot struct {
	SnapshotID       int64             `json:"snapshot-id"`
	ParentSnapshotID *int64            `json:"parent-snapshot-id,omitempty"`
	SequenceNumber   int64             `json:"sequence-number"`
	TimestampMS      int64             `json:"timestamp-ms"`
	ManifestList     string            `json:"manifest-list,omitempty"`
	Manifests        []string          `json:"manifests,omitempty"` // v1 only
	Summary          map[string]string `json:"summary,omitempty"`
	SchemaID         *int              `json:"schema-id,omitempty"`
	FirstRowID       *int64            `json:"first-row-id,omitempty"` // v3
	AddedRows        *int64            `json:"added-rows,omitempty"`   // v3
	KeyID            *string           `json:"key-id,omitempty"`       // v3
}

type snapshotLogEntry struct {
	SnapshotID  int64 `json:"snapshot-id"`
	TimestampMS int64 `json:"timestamp-ms"`
}

type metadataLogEntry struct {
	MetadataFile string `json:"metadata-file"`
	TimestampMS  int64  `json:"timestamp-ms"`
}

type snapshotRef struct {
	SnapshotID         int64  `json:"snapshot-id"`
	Type               string `json:"type"`
	MinSnapshotsToKeep *int   `json:"min-snapshots-to-keep,omitempty"`
	MaxSnapshotAgeMS   *int64 `json:"max-snapshot-age-ms,omitempty"`
	MaxRefAgeMS        *int64 `json:"max-ref-age-ms,omitempty"`
}

type statisticsFile struct {
	SnapshotID            int64           `json:"snapshot-id"`
	StatisticsPath        string          `json:"statistics-path"`
	FileSizeInBytes       int64           `json:"file-size-in-bytes"`
	FileFooterSizeInBytes int64           `json:"file-footer-size-in-bytes"`
	KeyMetadata           *string         `json:"key-metadata,omitempty"`
	BlobMetadata          json.RawMessage `json:"blob-metadata"`
}

type partitionStatisticsFile struct {
	SnapshotID      int64  `json:"snapshot-id"`
	StatisticsPath  string `json:"statistics-path"`
	FileSizeInBytes int64  `json:"file-size-in-bytes"`
}

const (
	mainBranch = "main"
	// partitionDataIDStart is where partition field ids start (spec), so an
	// unpartitioned table's last-partition-id is 999.
	partitionDataIDStart = 1000
	// minRowLineageVersion is the first format version w/ row lineage.
	minRowLineageVersion = 3
	maxFormatVersion     = 3
)

// parseMetadata decodes a metadata.json and normalizes the few things other
// writers spell differently (Java writes `current-snapshot-id: -1`, v1-era
// files may lack refs).
func parseMetadata(b []byte) (*tableMetadata, error) {
	var md tableMetadata
	if err := json.Unmarshal(b, &md); err != nil {
		return nil, err
	}
	if md.CurrentSnapshotID != nil && *md.CurrentSnapshotID == -1 {
		md.CurrentSnapshotID = nil
	}
	if md.Refs == nil {
		md.Refs = map[string]snapshotRef{}
	}
	if _, ok := md.Refs[mainBranch]; !ok && md.CurrentSnapshotID != nil {
		md.Refs[mainBranch] = snapshotRef{SnapshotID: *md.CurrentSnapshotID, Type: "branch"}
	}
	if md.Properties == nil {
		md.Properties = map[string]string{}
	}
	return &md, nil
}

// clone deep-copies md via JSON, which is plenty fast at catalog commit
// rates and can't miss a field.
func (md *tableMetadata) clone() *tableMetadata {
	b, err := json.Marshal(md)
	if err != nil {
		panic(err) // only plain data in here
	}
	out, err := parseMetadata(b)
	if err != nil {
		panic(err)
	}
	return out
}

func (md *tableMetadata) snapshot(id int64) *snapshot {
	for i := range md.Snapshots {
		if md.Snapshots[i].SnapshotID == id {
			return &md.Snapshots[i]
		}
	}
	return nil
}

func findByID(objs []jsonObject, key string, id int) bool {
	for _, o := range objs {
		if v, _ := o.int(key); v == id {
			return true
		}
	}
	return false
}

// ---- jsonObject ----

// jsonObject is a JSON object we mostly pass through. Unmarshal rejects
// duplicate keys, so the members we read are the same ones every other
// (case-sensitive) parser of the written file sees.
type jsonObject map[string]json.RawMessage

func (o *jsonObject) UnmarshalJSON(b []byte) error {
	m, err := decodeObject(b)
	if err != nil {
		return err
	}
	*o = m
	return nil
}

// decodeObject decodes a JSON object into its members, rejecting duplicate
// keys (which encoding/json would resolve last-wins) and trailing data.
func decodeObject(raw []byte) (jsonObject, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("expected a JSON object")
	}
	out := jsonObject{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := tok.(string)
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		out[key] = v
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("trailing data after object")
	}
	return out, nil
}

func (o jsonObject) int(key string) (int, bool) {
	raw, ok := o[key]
	if !ok {
		return 0, false
	}
	var v int
	if err := json.Unmarshal(raw, &v); err != nil {
		return 0, false
	}
	return v, true
}

func (o jsonObject) setInt(key string, v int) {
	o[key] = json.RawMessage(strconv.Itoa(v))
}

func (o jsonObject) objects(key string) ([]jsonObject, error) {
	raw, ok := o[key]
	if !ok {
		return nil, nil
	}
	var out []jsonObject
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("%q: %w", key, err)
	}
	return out, nil
}

func (o jsonObject) setObjects(key string, objs []jsonObject) {
	if objs == nil {
		objs = []jsonObject{}
	}
	b, err := json.Marshal(objs)
	if err != nil {
		panic(err)
	}
	o[key] = b
}

func (o jsonObject) clone() jsonObject {
	out := make(jsonObject, len(o))
	for k, v := range o {
		out[k] = v
	}
	return out
}

// canonical renders o w/o the given keys, w/ sorted keys and no whitespace,
// for "is this the same schema/spec/order" comparisons.
func (o jsonObject) canonical(without ...string) string {
	tmp := o.clone()
	for _, k := range without {
		delete(tmp, k)
	}
	b, _ := json.Marshal(tmp)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return string(b)
	}
	out, _ := json.Marshal(v)
	return string(out)
}

// ---- schema field ids ----

// walkSchemaIDs visits every field id in a schema (struct field ids, list
// element ids, map key/value ids) in the order Iceberg's AssignFreshIds
// assigns them: all of a struct's fields first, then each field's type.
// fn returns the id to store (return the input to leave it alone).
func walkSchemaIDs(schema jsonObject, fn func(int) int) (jsonObject, error) {
	out, err := walkType(json.RawMessage(mustMarshal(schema)), fn)
	if err != nil {
		return nil, err
	}
	return decodeObject(out)
}

func walkType(raw json.RawMessage, fn func(int) int) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		return raw, nil // primitive
	}
	obj, err := decodeObject(raw)
	if err != nil {
		return nil, err
	}
	var typ string
	if err := json.Unmarshal(obj["type"], &typ); err != nil {
		return nil, fmt.Errorf("invalid type: %s", raw)
	}
	remap := func(key string) error {
		id, ok := obj.int(key)
		if !ok {
			return fmt.Errorf("%s type missing %q", typ, key)
		}
		obj.setInt(key, fn(id))
		return nil
	}
	walk := func(key string) error {
		t, ok := obj[key]
		if !ok {
			return fmt.Errorf("%s type missing %q", typ, key)
		}
		t, err := walkType(t, fn)
		if err != nil {
			return err
		}
		obj[key] = t
		return nil
	}
	switch typ {
	case "struct":
		fields, err := obj.objects("fields")
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			id, ok := f.int("id")
			if !ok {
				return nil, fmt.Errorf("struct field missing \"id\"")
			}
			f.setInt("id", fn(id))
		}
		for _, f := range fields {
			t, err := walkType(f["type"], fn)
			if err != nil {
				return nil, err
			}
			f["type"] = t
		}
		obj.setObjects("fields", fields)
	case "list":
		if err := remap("element-id"); err != nil {
			return nil, err
		}
		if err := walk("element"); err != nil {
			return nil, err
		}
	case "map":
		if err := remap("key-id"); err != nil {
			return nil, err
		}
		if err := remap("value-id"); err != nil {
			return nil, err
		}
		if err := walk("key"); err != nil {
			return nil, err
		}
		if err := walk("value"); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown nested type %q", typ)
	}
	return mustMarshal(obj), nil
}

func highestFieldID(schema jsonObject) (int, error) {
	maxID := 0
	_, err := walkSchemaIDs(schema, func(id int) int {
		maxID = max(maxID, id)
		return id
	})
	return maxID, err
}

// v3Types are primitive types only format-version 3 allows.
var v3Types = map[string]bool{
	"variant": true, "unknown": true, "timestamp_ns": true, "timestamptz_ns": true,
	"geometry": true, "geography": true,
}

// schemaTypes lists every primitive type name in a schema (geometry(...)
// and geography(...) w/ params trimmed to the bare name).
func schemaTypes(raw json.RawMessage) []string {
	var out []string
	var walk func(json.RawMessage)
	walk = func(raw json.RawMessage) {
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 && raw[0] == '"' {
			var s string
			_ = json.Unmarshal(raw, &s)
			if i := bytes.IndexByte([]byte(s), '('); i > 0 {
				s = s[:i]
			}
			out = append(out, s)
			return
		}
		obj, err := decodeObject(raw)
		if err != nil {
			return
		}
		fields, _ := obj.objects("fields")
		for _, f := range fields {
			walk(f["type"])
		}
		for _, k := range []string{"element", "key", "value"} {
			if t, ok := obj[k]; ok {
				walk(t)
			}
		}
	}
	walk(raw)
	return out
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
