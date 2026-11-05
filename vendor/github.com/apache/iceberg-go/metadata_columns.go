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

import "math"

// MaxStructFieldID is the largest field ID a user-supplied schema may assign.
// Field IDs greater than this are reserved by the spec for metadata columns
// (_row_id, _last_updated_sequence_number, _file, _pos, _deleted, _spec_id,
// _partition, ...). Java: TypeUtil.MAX_STRUCT_FIELD_ID (Integer.MAX_VALUE - 200).
const MaxStructFieldID = math.MaxInt32 - 200

// IsReservedFieldID reports whether fieldID falls in the range the spec reserves
// for metadata columns. User schemas must not assign IDs in this range; table
// schema field IDs must not exceed MaxStructFieldID.
func IsReservedFieldID(fieldID int) bool {
	return fieldID > MaxStructFieldID
}

// Row lineage metadata column field IDs (v3+). Reserved IDs are Integer.MAX_VALUE - 107
// and Integer.MAX_VALUE - 108 per the Iceberg spec (Metadata Columns / Row Lineage).
const (
	// RowIDFieldID is the field ID for _row_id (optional long). A unique long identifier for every row.
	// Reserved as Integer.MAX_VALUE - 107.
	RowIDFieldID = 2147483540
	// LastUpdatedSequenceNumberFieldID is the field ID for _last_updated_sequence_number (optional long).
	// The sequence number of the commit that last updated the row. Reserved as Integer.MAX_VALUE - 108.
	LastUpdatedSequenceNumberFieldID = 2147483539
)

// Row lineage metadata column names (v3+).
const (
	RowIDColumnName                     = "_row_id"
	LastUpdatedSequenceNumberColumnName = "_last_updated_sequence_number"
)

// RowID returns a NestedField for _row_id (optional long) for use in schemas that include row lineage.
func RowID() NestedField {
	return NestedField{
		ID:       RowIDFieldID,
		Name:     RowIDColumnName,
		Required: false,
		Doc:      "Implicit row ID that is automatically assigned",
		Type:     Int64Type{},
	}
}

// LastUpdatedSequenceNumber returns a NestedField for _last_updated_sequence_number (optional long).
func LastUpdatedSequenceNumber() NestedField {
	return NestedField{
		ID:       LastUpdatedSequenceNumberFieldID,
		Name:     LastUpdatedSequenceNumberColumnName,
		Required: false,
		Doc:      "Sequence number when the row was last updated",
		Type:     Int64Type{},
	}
}

// IsMetadataColumn returns true if the field ID is a reserved metadata column (e.g. row lineage).
func IsMetadataColumn(fieldID int) bool {
	return fieldID == RowIDFieldID || fieldID == LastUpdatedSequenceNumberFieldID
}

// SchemaWithRowLineage appends both row-lineage metadata columns (_row_id,
// _last_updated_sequence_number) so a CoW rewrite or compaction preserves them.
func SchemaWithRowLineage(s *Schema) *Schema {
	return SchemaWithRowLineageColumns(s, true, true)
}

// SchemaWithRowLineageColumns appends the requested row-lineage columns: _row_id
// when rowID, _last_updated_sequence_number when lastUpdatedSeq. Request exactly
// the columns you will materialize so the read schema matches the produced batch.
//
// Idempotent by reserved field ID; always clones the field slice (no aliasing).
func SchemaWithRowLineageColumns(s *Schema, rowID, lastUpdatedSeq bool) *Schema {
	if s == nil {
		return nil
	}
	fields := s.Fields()

	hasRowID := false
	hasSeqNum := false
	for _, f := range fields {
		switch f.ID {
		case RowIDFieldID:
			hasRowID = true
		case LastUpdatedSequenceNumberFieldID:
			hasSeqNum = true
		}
	}

	if rowID && !hasRowID {
		fields = append(fields, RowID())
	}
	if lastUpdatedSeq && !hasSeqNum {
		fields = append(fields, LastUpdatedSequenceNumber())
	}

	return NewSchemaWithIdentifiers(s.ID, s.IdentifierFieldIDs, fields...)
}

// SchemaWithRowID returns a new schema with only the _row_id metadata column
// appended. _last_updated_sequence_number is intentionally omitted: leaving it
// absent in the written Parquet means readers synthesize it from the manifest
// entry's data_sequence_number, which is the new file's snapshot sequence
// number after the rewrite — exactly the value the spec requires for rewritten
// rows without an explicit override.
//
// Idempotent on RowIDFieldID; allocates a fresh field slice.
func SchemaWithRowID(s *Schema) *Schema {
	return SchemaWithRowLineageColumns(s, true, false)
}
