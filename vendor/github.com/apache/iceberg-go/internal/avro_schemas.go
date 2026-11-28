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

package internal

import (
	"fmt"
	"slices"

	"github.com/twmb/avro"
	"github.com/twmb/avro/atype"
)

func NullableNode(node avro.SchemaNode) avro.SchemaNode {
	return avro.SchemaNode{
		Type:     "union",
		Branches: []avro.SchemaNode{{Type: "null"}, node},
	}
}

// mustSchema compiles a SchemaNode into a Schema, panicking on error.
// This avoids the pointer-receiver addressability issue with struct literals.
func mustSchema(n avro.SchemaNode) *avro.Schema {
	return Must(n.Schema())
}

func NullableSchema(schema *avro.Schema) *avro.Schema {
	return mustSchema(NullableNode(*schema.Root()))
}

var requiredLength = [...]int{
	1, 1, 1, 2, 2, 3, 3, 4, 4, 4, 5, 5,
	6, 6, 6, 7, 7, 8, 8, 9, 9, 9, 10, 10, 11, 11, 11, 12, 12,
	13, 13, 13, 14, 14, 15, 15, 16, 16, 16, 17,
}

// DecimalRequiredBytes returns the required number of bytes to store a
// decimal value of the given precision. If the precision is outside
// the range (0, 40], this returns -1 as it is invalid.
func DecimalRequiredBytes(precision int) int {
	if precision <= 0 || precision >= 40 {
		return -1
	}

	return requiredLength[precision]
}

func DecimalNode(precision, scale int) avro.SchemaNode {
	return avro.SchemaNode{
		Type:        atype.Fixed,
		Name:        "fixed",
		Size:        DecimalRequiredBytes(precision),
		LogicalType: atype.Decimal,
		Precision:   precision,
		Scale:       scale,
	}
}

func DecimalSchema(precision, scale int) *avro.Schema {
	return mustSchema(DecimalNode(precision, scale))
}

func FixedNode(size int) avro.SchemaNode {
	return avro.SchemaNode{
		Type: atype.Fixed,
		Name: fmt.Sprintf("fixed_%d", size),
		Size: size,
	}
}

func FixedSchema(size int) *avro.Schema {
	return mustSchema(FixedNode(size))
}

// Schema nodes for reuse in composition.
var (
	NullNode   = avro.SchemaNode{Type: atype.Null}
	BoolNode   = avro.SchemaNode{Type: atype.Boolean}
	IntNode    = avro.SchemaNode{Type: atype.Int}
	LongNode   = avro.SchemaNode{Type: atype.Long}
	FloatNode  = avro.SchemaNode{Type: atype.Float}
	DoubleNode = avro.SchemaNode{Type: atype.Double}
	StringNode = avro.SchemaNode{Type: atype.String}
	BytesNode  = avro.SchemaNode{Type: atype.Bytes}

	DateNode        = avro.SchemaNode{Type: atype.Int, LogicalType: atype.Date}
	TimeNode        = avro.SchemaNode{Type: atype.Long, LogicalType: atype.TimeMicros}
	TimestampNode   = avro.SchemaNode{Type: atype.Long, LogicalType: atype.TimestampMicros, Props: map[string]any{"adjust-to-utc": false}}
	TimestampTzNode = avro.SchemaNode{Type: atype.Long, LogicalType: atype.TimestampMicros, Props: map[string]any{"adjust-to-utc": true}}

	TimestampNsNode   = avro.SchemaNode{Type: atype.Long, LogicalType: atype.TimestampNanos, Props: map[string]any{"adjust-to-utc": false}}
	TimestampTzNsNode = avro.SchemaNode{Type: atype.Long, LogicalType: atype.TimestampNanos, Props: map[string]any{"adjust-to-utc": true}}

	UUIDNode = avro.SchemaNode{Type: atype.Fixed, Name: "uuid_fixed", Size: 16, LogicalType: atype.UUID}
)

// Compiled schemas for direct encoding/decoding use.
var (
	NullSchema           = mustSchema(NullNode)
	BoolSchema           = mustSchema(BoolNode)
	NullableBoolSchema   = NullableSchema(BoolSchema)
	BinarySchema         = mustSchema(BytesNode)
	NullableBinarySchema = NullableSchema(BinarySchema)
	StringSchema         = mustSchema(StringNode)
	IntSchema            = mustSchema(IntNode)
	NullableIntSchema    = NullableSchema(IntSchema)
	LongSchema           = mustSchema(LongNode)
	NullableLongSchema   = NullableSchema(LongSchema)
	FloatSchema          = mustSchema(FloatNode)
	DoubleSchema         = mustSchema(DoubleNode)
	DateSchema           = mustSchema(DateNode)
	TimeSchema           = mustSchema(TimeNode)
	TimestampSchema      = mustSchema(TimestampNode)
	TimestampTzSchema    = mustSchema(TimestampTzNode)
	UUIDSchema           = mustSchema(UUIDNode)
)

func fieldNode(name string, typ avro.SchemaNode, fieldID int, opts ...func(*avro.SchemaField)) avro.SchemaField {
	f := avro.SchemaField{
		Name:  name,
		Type:  typ,
		Props: map[string]any{"field-id": fieldID},
	}
	for _, opt := range opts {
		opt(&f)
	}

	return f
}

func withDoc(doc string) func(*avro.SchemaField) {
	return func(f *avro.SchemaField) {
		f.Doc = doc
	}
}

func withDefault(val any) func(*avro.SchemaField) {
	return func(f *avro.SchemaField) {
		f.Default = val
		f.HasDefault = true
	}
}

func newMapNode(name string, keyNode, valueNode avro.SchemaNode, keyFieldID, valueFieldID int) avro.SchemaNode {
	return avro.SchemaNode{
		Type: "array",
		Items: &avro.SchemaNode{
			Type: "record",
			Name: name,
			Fields: []avro.SchemaField{
				fieldNode("key", keyNode, keyFieldID),
				fieldNode("value", valueNode, valueFieldID),
			},
		},
		Props: map[string]any{"logicalType": "map"},
	}
}

// AvroSchemaMap stores prebuilt schemas by name for lookup.
var AvroSchemaMap = make(map[string]*avro.Schema)

// fieldSummaryNode is reused in manifest list schemas.
var fieldSummaryNode = avro.SchemaNode{
	Type: "record",
	Name: "r508",
	Fields: []avro.SchemaField{
		fieldNode("contains_null", BoolNode, 509,
			withDoc("true if the field contains null values")),
		fieldNode("contains_nan", NullableNode(BoolNode), 518,
			withDoc("true if the field contains NaN values")),
		fieldNode("lower_bound", NullableNode(BytesNode), 510,
			withDoc("serialized lower bound")),
		fieldNode("upper_bound", NullableNode(BytesNode), 511,
			withDoc("serialized upper bound")),
	},
}

func init() {
	AvroSchemaMap["field_summary"] = mustSchema(fieldSummaryNode)

	partitionsNode := NullableNode(avro.SchemaNode{
		Type:  "array",
		Items: &fieldSummaryNode,
		Props: map[string]any{"element-id": 508},
	})

	AvroSchemaMap["manifest_list_file_v1"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_file",
		Fields: []avro.SchemaField{
			fieldNode("manifest_path", StringNode, 500, withDoc("Location URI with FS scheme")),
			fieldNode("manifest_length", LongNode, 501, withDoc("Total file size in bytes")),
			fieldNode("partition_spec_id", IntNode, 502, withDoc("Spec ID used to write")),
			fieldNode("added_snapshot_id", LongNode, 503, withDoc("Snapshot ID that added the manifest")),
			fieldNode("added_files_count", NullableNode(IntNode), 504, withDoc("Added entry count")),
			fieldNode("existing_files_count", NullableNode(IntNode), 505, withDoc("Existing entry count")),
			fieldNode("deleted_files_count", NullableNode(IntNode), 506, withDoc("Deleted entry count")),
			fieldNode("partitions", partitionsNode, 507, withDoc("Partition field summaries")),
			fieldNode("added_rows_count", NullableNode(LongNode), 512, withDoc("Added row count")),
			fieldNode("existing_rows_count", NullableNode(LongNode), 513, withDoc("Existing row count")),
			fieldNode("deleted_rows_count", NullableNode(LongNode), 514, withDoc("Deleted row count")),
			fieldNode("key_metadata", NullableNode(BytesNode), 519, withDoc("Key metadata")),
		},
	})

	AvroSchemaMap["manifest_list_file_v2"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_file",
		Fields: []avro.SchemaField{
			fieldNode("manifest_path", StringNode, 500, withDoc("Location URI with FS scheme")),
			fieldNode("manifest_length", LongNode, 501, withDoc("Total file size in bytes")),
			fieldNode("partition_spec_id", IntNode, 502, withDoc("Spec ID used to write")),
			fieldNode("content", IntNode, 517, withDoc("Content type"), withDefault(0)),
			fieldNode("sequence_number", LongNode, 515, withDoc("Sequence number"), withDefault(int64(0))),
			fieldNode("min_sequence_number", LongNode, 516, withDoc("Minimum sequence number"), withDefault(int64(0))),
			fieldNode("added_snapshot_id", LongNode, 503, withDoc("Snapshot ID that added the manifest")),
			fieldNode("added_files_count", IntNode, 504, withDoc("Added entry count")),
			fieldNode("existing_files_count", IntNode, 505, withDoc("Existing entry count")),
			fieldNode("deleted_files_count", IntNode, 506, withDoc("Deleted entry count")),
			fieldNode("partitions", partitionsNode, 507, withDoc("Partition field summaries")),
			fieldNode("added_rows_count", LongNode, 512, withDoc("Added row count")),
			fieldNode("existing_rows_count", LongNode, 513, withDoc("Existing row count")),
			fieldNode("deleted_rows_count", LongNode, 514, withDoc("Deleted row count")),
			fieldNode("key_metadata", NullableNode(BytesNode), 519, withDoc("Key metadata")),
		},
	})

	AvroSchemaMap["data_file_v1"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "r2",
		Fields: []avro.SchemaField{
			fieldNode("file_path", StringNode, 100, withDoc("Location URI with FS scheme")),
			fieldNode("file_format", StringNode, 101, withDoc("File format name: avro, orc, parquet")),
			// skip partition field, we'll add that dynamically as needed
			fieldNode("record_count", LongNode, 103, withDoc("Number of records in the file")),
			fieldNode("file_size_in_bytes", LongNode, 104, withDoc("Size of the file in bytes")),
			fieldNode("block_size_in_bytes", LongNode, 105,
				withDoc("Deprecated. Always write default in v1. Do not write in v2."),
				withDefault(int64(64*1024*1024))),
			fieldNode("column_sizes",
				NullableNode(newMapNode("k117_v118", IntNode, LongNode, 117, 118)),
				108, withDoc("map of column id to total size on disk")),
			fieldNode("value_counts",
				NullableNode(newMapNode("k119_v120", IntNode, LongNode, 119, 120)),
				109, withDoc("map of value to count")),
			fieldNode("null_value_counts",
				NullableNode(newMapNode("k121_v122", IntNode, LongNode, 121, 122)),
				110, withDoc("map of value to count")),
			fieldNode("nan_value_counts",
				NullableNode(newMapNode("k138_v139", IntNode, LongNode, 138, 139)),
				137, withDoc("map of value to count")),
			fieldNode("lower_bounds",
				NullableNode(newMapNode("k126_v127", IntNode, BytesNode, 126, 127)),
				125, withDoc("map of column id to lower bound")),
			fieldNode("upper_bounds",
				NullableNode(newMapNode("k129_v130", IntNode, BytesNode, 129, 130)),
				128, withDoc("map of column id to upper bound")),
			fieldNode("key_metadata", NullableNode(BytesNode), 131, withDoc("Encryption Key Metadata Blob")),
			fieldNode("split_offsets",
				NullableNode(avro.SchemaNode{Type: "array", Items: &LongNode, Props: map[string]any{"element-id": 133}}),
				132, withDoc("splitable offsets")),
			fieldNode("sort_order_id", NullableNode(IntNode), 140, withDoc("Sort order ID")),
		},
	})

	AvroSchemaMap["data_file_v2"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "r2",
		Fields: []avro.SchemaField{
			fieldNode("content", IntNode, 134, withDoc("Content type"), withDefault(0)),
			fieldNode("file_path", StringNode, 100, withDoc("Location URI with FS scheme")),
			fieldNode("file_format", StringNode, 101, withDoc("File format name: avro, orc, parquet")),
			// skip partition field, we'll add that dynamically as needed
			fieldNode("record_count", LongNode, 103, withDoc("Number of records in the file")),
			fieldNode("file_size_in_bytes", LongNode, 104, withDoc("Size of the file in bytes")),
			fieldNode("column_sizes",
				NullableNode(newMapNode("k117_v118", IntNode, LongNode, 117, 118)),
				108, withDoc("map of column id to total size on disk")),
			fieldNode("value_counts",
				NullableNode(newMapNode("k119_v120", IntNode, LongNode, 119, 120)),
				109, withDoc("map of value to count")),
			fieldNode("null_value_counts",
				NullableNode(newMapNode("k121_v122", IntNode, LongNode, 121, 122)),
				110, withDoc("map of value to count")),
			fieldNode("nan_value_counts",
				NullableNode(newMapNode("k138_v139", IntNode, LongNode, 138, 139)),
				137, withDoc("map of value to count")),
			fieldNode("lower_bounds",
				NullableNode(newMapNode("k126_v127", IntNode, BytesNode, 126, 127)),
				125, withDoc("map of column id to lower bound")),
			fieldNode("upper_bounds",
				NullableNode(newMapNode("k129_v130", IntNode, BytesNode, 129, 130)),
				128, withDoc("map of column id to upper bound")),
			fieldNode("key_metadata", NullableNode(BytesNode), 131, withDoc("Encryption Key Metadata Blob")),
			fieldNode("split_offsets",
				NullableNode(avro.SchemaNode{Type: "array", Items: &LongNode, Props: map[string]any{"element-id": 133}}),
				132, withDoc("splitable offsets")),
			fieldNode("equality_ids",
				NullableNode(avro.SchemaNode{Type: "array", Items: &IntNode, Props: map[string]any{"element-id": 136}}),
				135, withDoc("field ids used to determine row equality in equality delete files")),
			fieldNode("sort_order_id", NullableNode(IntNode), 140, withDoc("Sort order ID")),
		},
	})

	AvroSchemaMap["manifest_entry_v1"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_entry",
		Fields: []avro.SchemaField{
			fieldNode("status", IntNode, 0),
			fieldNode("snapshot_id", LongNode, 1),
			// leave data_file for dynamic generation
		},
	})

	AvroSchemaMap["manifest_entry_v2"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_entry",
		Fields: []avro.SchemaField{
			fieldNode("status", IntNode, 0),
			fieldNode("snapshot_id", NullableNode(LongNode), 1),
			fieldNode("sequence_number", NullableNode(LongNode), 3),
			fieldNode("file_sequence_number", NullableNode(LongNode), 4),
			// leave data_file for dynamic generation
		},
	})

	AvroSchemaMap["manifest_list_file_v3"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_file",
		Fields: []avro.SchemaField{
			fieldNode("manifest_path", StringNode, 500, withDoc("Location URI with FS scheme")),
			fieldNode("manifest_length", LongNode, 501, withDoc("Total file size in bytes")),
			fieldNode("partition_spec_id", IntNode, 502, withDoc("Spec ID used to write")),
			fieldNode("content", IntNode, 517, withDoc("Content type"), withDefault(0)),
			fieldNode("sequence_number", LongNode, 515, withDoc("Sequence number"), withDefault(int64(0))),
			fieldNode("min_sequence_number", LongNode, 516, withDoc("Minimum sequence number"), withDefault(int64(0))),
			fieldNode("added_snapshot_id", LongNode, 503, withDoc("Snapshot ID that added the manifest")),
			fieldNode("added_files_count", IntNode, 504, withDoc("Added entry count")),
			fieldNode("existing_files_count", IntNode, 505, withDoc("Existing entry count")),
			fieldNode("deleted_files_count", IntNode, 506, withDoc("Deleted entry count")),
			fieldNode("partitions", partitionsNode, 507, withDoc("Partition field summaries")),
			fieldNode("added_rows_count", LongNode, 512, withDoc("Added row count")),
			fieldNode("existing_rows_count", LongNode, 513, withDoc("Existing row count")),
			fieldNode("deleted_rows_count", LongNode, 514, withDoc("Deleted row count")),
			fieldNode("key_metadata", NullableNode(BytesNode), 519, withDoc("Key metadata")),
			fieldNode("first_row_id", NullableNode(LongNode), 520, withDoc("First row ID")),
		},
	})

	AvroSchemaMap["data_file_v3"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "r2",
		Fields: []avro.SchemaField{
			fieldNode("content", IntNode, 134, withDoc("Content type"), withDefault(0)),
			fieldNode("file_path", StringNode, 100, withDoc("Location URI with FS scheme")),
			fieldNode("file_format", StringNode, 101, withDoc("File format name: avro, orc, parquet")),
			// skip partition field, we'll add that dynamically as needed
			fieldNode("record_count", LongNode, 103, withDoc("Number of records in the file")),
			fieldNode("file_size_in_bytes", LongNode, 104, withDoc("Size of the file in bytes")),
			fieldNode("column_sizes",
				NullableNode(newMapNode("k117_v118", IntNode, LongNode, 117, 118)),
				108, withDoc("map of column id to total size on disk")),
			fieldNode("value_counts",
				NullableNode(newMapNode("k119_v120", IntNode, LongNode, 119, 120)),
				109, withDoc("map of value to count")),
			fieldNode("null_value_counts",
				NullableNode(newMapNode("k121_v122", IntNode, LongNode, 121, 122)),
				110, withDoc("map of value to count")),
			fieldNode("nan_value_counts",
				NullableNode(newMapNode("k138_v139", IntNode, LongNode, 138, 139)),
				137, withDoc("map of value to count")),
			fieldNode("lower_bounds",
				NullableNode(newMapNode("k126_v127", IntNode, BytesNode, 126, 127)),
				125, withDoc("map of column id to lower bound")),
			fieldNode("upper_bounds",
				NullableNode(newMapNode("k129_v130", IntNode, BytesNode, 129, 130)),
				128, withDoc("map of column id to upper bound")),
			fieldNode("key_metadata", NullableNode(BytesNode), 131, withDoc("Encryption Key Metadata Blob")),
			fieldNode("split_offsets",
				NullableNode(avro.SchemaNode{Type: "array", Items: &LongNode, Props: map[string]any{"element-id": 133}}),
				132, withDoc("splitable offsets")),
			fieldNode("equality_ids",
				NullableNode(avro.SchemaNode{Type: "array", Items: &IntNode, Props: map[string]any{"element-id": 136}}),
				135, withDoc("field ids used to determine row equality in equality delete files")),
			fieldNode("sort_order_id", NullableNode(IntNode), 140, withDoc("Sort order ID")),
			fieldNode("first_row_id", NullableNode(LongNode), 142, withDoc("The _row_id for the first row in the data file")),
			fieldNode("referenced_data_file", NullableNode(StringNode), 143,
				withDoc("Fully qualified location of a data file that all deletes reference")),
			fieldNode("content_offset", NullableNode(LongNode), 144,
				withDoc("The offset in the file where the content starts")),
			fieldNode("content_size_in_bytes", NullableNode(LongNode), 145,
				withDoc("The length of the referenced content stored in the file")),
		},
	})

	AvroSchemaMap["manifest_entry_v3"] = mustSchema(avro.SchemaNode{
		Type: "record",
		Name: "manifest_entry",
		Fields: []avro.SchemaField{
			fieldNode("status", IntNode, 0),
			fieldNode("snapshot_id", NullableNode(LongNode), 1),
			fieldNode("sequence_number", NullableNode(LongNode), 3),
			fieldNode("file_sequence_number", NullableNode(LongNode), 4),
			// leave data_file for dynamic generation
		},
	})
}

func newDataFileSchema(partitionType *avro.Schema, version int) (*avro.Schema, error) {
	key := fmt.Sprintf("data_file_v%d", version)
	schema := AvroSchemaMap[key]

	node := schema.Root()
	node.Fields = append(slices.Clone(node.Fields), avro.SchemaField{
		Name:  "partition",
		Type:  *partitionType.Root(),
		Props: map[string]any{"field-id": 102},
	})

	return node.Schema()
}

func NewManifestFileSchema(version int) (*avro.Schema, error) {
	switch version {
	case 1, 2, 3:
	default:
		return nil, fmt.Errorf("unsupported iceberg spec version: %d", version)
	}

	key := fmt.Sprintf("manifest_list_file_v%d", version)

	return AvroSchemaMap[key], nil
}

func NewManifestEntrySchema(partitionType *avro.Schema, version int) (*avro.Schema, error) {
	switch version {
	case 1, 2, 3:
	default:
		return nil, fmt.Errorf("unsupported iceberg spec version: %d", version)
	}

	dfschema, err := newDataFileSchema(partitionType, version)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("manifest_entry_v%d", version)
	schema := AvroSchemaMap[key]

	node := schema.Root()
	node.Fields = append(slices.Clone(node.Fields), avro.SchemaField{
		Name:  "data_file",
		Type:  *dfschema.Root(),
		Props: map[string]any{"field-id": 2},
	})

	return node.Schema()
}

// WithFieldID returns a props map for use with SchemaField for Iceberg field IDs.
func WithFieldID(id int) map[string]any {
	return map[string]any{"field-id": id}
}
