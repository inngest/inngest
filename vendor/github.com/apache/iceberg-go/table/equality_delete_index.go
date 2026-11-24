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

package table

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"slices"
	"sort"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

// equalityDeleteIndex groups equality deletes by partition and sequence number.
// Unpartitioned equality deletes are global and apply across partition specs;
// partitioned deletes only apply to data files in the same spec and partition.
type equalityDeleteIndex struct {
	global      []equalityDeleteIndexEntry
	byPartition map[equalityDeletePartitionKey][]equalityDeleteIndexEntry
	schema      *iceberg.Schema
}

type equalityDeleteIndexEntry struct {
	entry  iceberg.ManifestEntry
	fields []equalityDeleteFieldMetrics
}

type equalityDeleteRangeKind uint8

const (
	equalityDeleteRangeNone equalityDeleteRangeKind = iota
	equalityDeleteRangeBool
	equalityDeleteRangeInt32
	equalityDeleteRangeInt64
	equalityDeleteRangeFloat32
	equalityDeleteRangeFloat64
	equalityDeleteRangeDate
	equalityDeleteRangeTime
	equalityDeleteRangeTimestamp
	equalityDeleteRangeTimestampNano
	equalityDeleteRangeString
	equalityDeleteRangeBytes
	equalityDeleteRangeUUID
	equalityDeleteRangeDecimal
)

// equalityDeleteMetricValue keeps decoded bounds in concrete fields so the
// candidate loop does not pay for interface comparator calls on every file.
// This is the same comparison work as Java's DeleteFileIndex, but avoids
// allocating or dispatching through a closure for each bound check.
type equalityDeleteMetricValue struct {
	kind         equalityDeleteRangeKind
	integerValue int64
	floatValue   float64
	reference    any
	uuidValue    uuid.UUID
	decimalValue iceberg.Decimal
}

func equalityDeleteMetricValueFromLiteral(
	kind equalityDeleteRangeKind,
	literal iceberg.Literal,
) (equalityDeleteMetricValue, bool) {
	value := equalityDeleteMetricValue{kind: kind}
	switch kind {
	case equalityDeleteRangeBool:
		v, ok := literal.(iceberg.TypedLiteral[bool])
		if ok {
			if v.Value() {
				value.integerValue = 1
			}
		}

		return value, ok
	case equalityDeleteRangeInt32:
		v, ok := literal.(iceberg.TypedLiteral[int32])
		if ok {
			value.integerValue = int64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeInt64:
		v, ok := literal.(iceberg.TypedLiteral[int64])
		if ok {
			value.integerValue = v.Value()
		}

		return value, ok
	case equalityDeleteRangeFloat32:
		v, ok := literal.(iceberg.TypedLiteral[float32])
		if ok {
			value.floatValue = float64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeFloat64:
		v, ok := literal.(iceberg.TypedLiteral[float64])
		if ok {
			value.floatValue = v.Value()
		}

		return value, ok
	case equalityDeleteRangeDate:
		v, ok := literal.(iceberg.TypedLiteral[iceberg.Date])
		if ok {
			value.integerValue = int64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeTime:
		v, ok := literal.(iceberg.TypedLiteral[iceberg.Time])
		if ok {
			value.integerValue = int64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeTimestamp:
		v, ok := literal.(iceberg.TypedLiteral[iceberg.Timestamp])
		if ok {
			value.integerValue = int64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeTimestampNano:
		v, ok := literal.(iceberg.TypedLiteral[iceberg.TimestampNano])
		if ok {
			value.integerValue = int64(v.Value())
		}

		return value, ok
	case equalityDeleteRangeString:
		v, ok := literal.(iceberg.TypedLiteral[string])
		if ok {
			value.reference = v.Value()
		}

		return value, ok
	case equalityDeleteRangeBytes:
		v, ok := literal.(iceberg.TypedLiteral[[]byte])
		if ok {
			value.reference = v.Value()
		}

		return value, ok
	case equalityDeleteRangeUUID:
		v, ok := literal.(iceberg.TypedLiteral[uuid.UUID])
		if ok {
			value.uuidValue = v.Value()
		}

		return value, ok
	case equalityDeleteRangeDecimal:
		v, ok := literal.(iceberg.TypedLiteral[iceberg.Decimal])
		if ok {
			value.decimalValue = v.Value()
		}

		return value, ok
	default:
		return equalityDeleteMetricValue{}, false
	}
}

func equalityDeleteMetricValueFromBytes(
	typ iceberg.Type,
	kind equalityDeleteRangeKind,
	data []byte,
	clone bool,
) (equalityDeleteMetricValue, bool) {
	if clone {
		// String, binary, and fixed literals may retain the input bytes. Clone
		// once so an index does not retain a borrowed metadata buffer.
		data = slices.Clone(data)
	}

	literal, err := iceberg.LiteralFromBytes(typ, data)
	if err != nil || equalityDeleteLiteralIsNaN(literal) {
		return equalityDeleteMetricValue{}, false
	}

	return equalityDeleteMetricValueFromLiteral(kind, literal)
}

func equalityDeleteMetricValueCompare(left, right *equalityDeleteMetricValue) int {
	switch left.kind {
	case equalityDeleteRangeBool:
		return cmp.Compare(left.integerValue, right.integerValue)
	case equalityDeleteRangeInt32, equalityDeleteRangeInt64,
		equalityDeleteRangeDate, equalityDeleteRangeTime,
		equalityDeleteRangeTimestamp, equalityDeleteRangeTimestampNano:
		return cmp.Compare(left.integerValue, right.integerValue)
	case equalityDeleteRangeFloat32, equalityDeleteRangeFloat64:
		return cmp.Compare(left.floatValue, right.floatValue)
	case equalityDeleteRangeString:
		return cmp.Compare(left.reference.(string), right.reference.(string))
	case equalityDeleteRangeBytes:
		return bytes.Compare(left.reference.([]byte), right.reference.([]byte))
	case equalityDeleteRangeUUID:
		return bytes.Compare(left.uuidValue[:], right.uuidValue[:])
	case equalityDeleteRangeDecimal:
		if left.decimalValue.Scale == right.decimalValue.Scale {
			return left.decimalValue.Val.Cmp(right.decimalValue.Val)
		}

		rescaled, err := right.decimalValue.Val.Rescale(
			int32(right.decimalValue.Scale), int32(left.decimalValue.Scale))
		if err != nil {
			return -1
		}

		return left.decimalValue.Val.Cmp(rescaled)
	default:
		return 0
	}
}

// equalityDeleteFieldMetrics contains the statistics for one equality field
// in one delete file. Bounds are copied because the DataFile statistics helper
// may return borrowed maps and byte slices.
type equalityDeleteFieldMetrics struct {
	fieldID     int
	typ         iceberg.Type
	required    bool
	primitive   bool
	canUseRange bool
	floatType   bool
	rangeKind   equalityDeleteRangeKind

	valueCount    int64
	nullCount     int64
	nanCount      int64
	hasValueCount bool
	hasNullCount  bool
	hasNaNCount   bool

	lowerValue       equalityDeleteMetricValue
	upperValue       equalityDeleteMetricValue
	hasDecodedBounds bool
}

type equalityDeleteDataFileStats struct {
	valueCounts   map[int]int64
	nullCounts    map[int]int64
	nanCounts     map[int]int64
	lowerBounds   map[int][]byte
	upperBounds   map[int][]byte
	decodedBounds map[int]*equalityDeleteDataFileBounds
}

type equalityDeleteDataFileBounds struct {
	lower  equalityDeleteMetricValue
	upper  equalityDeleteMetricValue
	usable bool
}

type partitionSpecLookup interface {
	PartitionSpecByID(int) *iceberg.PartitionSpec
}

type equalityDeletePartitionKey struct {
	specID  int32
	fieldID int
	value   any
	tuple   string
	single  bool
}

type (
	equalityDeleteIntegerPartitionValue int64
	equalityDeleteStringPartitionValue  string
	equalityDeleteEncodedPartitionValue string
	equalityDeleteBinaryPartitionValue  string
	equalityDeleteNilPartitionValue     struct{}
)

// Use a comparable value directly for the common single-field partition spec
// so each data-file lookup does not allocate an encoded tuple. Multi-field
// specs use the same normalized encoding as conflict validation.
func newEqualityDeletePartitionKey(
	specID int32,
	partition map[int]any,
) (equalityDeletePartitionKey, error) {
	if len(partition) == 1 {
		for fieldID, value := range partition {
			comparable, err := comparableEqualityDeletePartitionValue(value)
			if err != nil {
				return equalityDeletePartitionKey{}, fmt.Errorf("partition field %d: %w", fieldID, err)
			}

			return equalityDeletePartitionKey{
				specID: specID, fieldID: fieldID, value: comparable, single: true,
			}, nil
		}
	}

	tuple, err := canonicalPartitionKey(specID, partition)
	if err != nil {
		return equalityDeletePartitionKey{}, err
	}

	return equalityDeletePartitionKey{specID: specID, tuple: tuple}, nil
}

func comparableEqualityDeletePartitionValue(value any) (any, error) {
	switch value := value.(type) {
	case nil:
		return equalityDeleteNilPartitionValue{}, nil
	case bool:
		return value, nil
	case int:
		return equalityDeleteIntegerPartitionValue(value), nil
	case int32:
		return equalityDeleteIntegerPartitionValue(value), nil
	case int64:
		return equalityDeleteIntegerPartitionValue(value), nil
	case iceberg.Date:
		return equalityDeleteIntegerPartitionValue(value), nil
	case iceberg.Time:
		return equalityDeleteIntegerPartitionValue(value), nil
	case iceberg.Timestamp:
		return equalityDeleteIntegerPartitionValue(value), nil
	case iceberg.TimestampNano:
		return equalityDeleteIntegerPartitionValue(value), nil
	case float32, float64:
		// Keep equality-delete indexing aligned with writer partition keys:
		// canonicalize NaNs within each width and preserve signed zero.
		return comparablePartitionKey(value), nil
	case string:
		return equalityDeleteStringPartitionValue(value), nil
	case []byte:
		return equalityDeleteBinaryPartitionValue(value), nil
	case uuid.UUID:
		return equalityDeleteBinaryPartitionValue(value[:]), nil
	default:
		encoded, err := appendCanonicalPartitionValue(nil, value)
		if err != nil {
			return nil, err
		}

		return equalityDeleteEncodedPartitionValue(encoded), nil
	}
}

func newEqualityDeleteIndexEntry(
	entry iceberg.ManifestEntry,
	schema *iceberg.Schema,
) equalityDeleteIndexEntry {
	indexed := equalityDeleteIndexEntry{entry: entry}
	if schema == nil {
		return indexed
	}

	dataFile := entry.DataFile()
	_, _, _, fieldIDs := dataFileCollections(dataFile)
	if len(fieldIDs) == 0 {
		return indexed
	}

	indexed.fields = make([]equalityDeleteFieldMetrics, len(fieldIDs))
	knownFields := 0
	for i, fieldID := range fieldIDs {
		field := &indexed.fields[i]
		field.fieldID = fieldID

		schemaField, ok := schema.FindFieldByIDRef(fieldID, iceberginternal.SchemaRef{})
		if !ok {
			// A field can be absent from the current schema after it was
			// dropped. Without its type, the bounds cannot be decoded safely.
			continue
		}

		field.typ = schemaField.Type
		field.required = schemaField.Required && !schema.FieldHasOptionalParent(fieldID)
		_, field.primitive = field.typ.(iceberg.PrimitiveType)
		field.rangeKind = equalityDeleteRangeKindForType(field.typ)
		field.canUseRange = field.rangeKind != equalityDeleteRangeNone
		field.floatType = equalityDeleteFloatType(field.typ)
		knownFields++
	}

	if knownFields == 0 {
		return indexed
	}

	valueCounts, nullCounts, nanCounts, lowerBounds, upperBounds := dataFileStats(dataFile)
	for i := range indexed.fields {
		field := &indexed.fields[i]
		if field.typ == nil {
			continue
		}

		if value, ok := valueCounts[field.fieldID]; ok {
			field.valueCount = value
			field.hasValueCount = true
		}
		if value, ok := nullCounts[field.fieldID]; ok {
			field.nullCount = value
			field.hasNullCount = true
		}
		if value, ok := nanCounts[field.fieldID]; ok {
			field.nanCount = value
			field.hasNaNCount = true
		}
		if field.canUseRange {
			lowerBytes, hasLowerBytes := lowerBounds[field.fieldID]
			upperBytes, hasUpperBytes := upperBounds[field.fieldID]
			if !hasLowerBytes || !hasUpperBytes {
				continue
			}

			lowerValue, lowerOK := equalityDeleteMetricValueFromBytes(
				field.typ, field.rangeKind, lowerBytes, true)
			upperValue, upperOK := equalityDeleteMetricValueFromBytes(
				field.typ, field.rangeKind, upperBytes, true)
			if lowerOK && upperOK && equalityDeleteMetricValueCompare(&lowerValue, &upperValue) <= 0 {
				field.lowerValue = lowerValue
				field.upperValue = upperValue
				field.hasDecodedBounds = true
			}
		}
	}

	return indexed
}

func equalityDeleteDataFileStatsFor(file iceberg.DataFile) equalityDeleteDataFileStats {
	valueCounts, nullCounts, nanCounts, lowerBounds, upperBounds := dataFileStats(file)

	return equalityDeleteDataFileStats{
		valueCounts: valueCounts,
		nullCounts:  nullCounts,
		nanCounts:   nanCounts,
		lowerBounds: lowerBounds,
		upperBounds: upperBounds,
	}
}

func buildEqualityDeleteIndex(
	entries []iceberg.ManifestEntry,
	specs partitionSpecLookup,
	schema *iceberg.Schema,
) (*equalityDeleteIndex, error) {
	idx := &equalityDeleteIndex{schema: schema}
	unpartitionedBySpecID := make(map[int32]bool)
	for _, entry := range entries {
		df := entry.DataFile()
		indexedEntry := newEqualityDeleteIndexEntry(entry, schema)
		isUnpartitioned, ok := unpartitionedBySpecID[df.SpecID()]
		if !ok {
			spec := specs.PartitionSpecByID(int(df.SpecID()))
			if spec == nil {
				return nil, fmt.Errorf("indexing equality delete file %s: %w: id %d",
					df.FilePath(), ErrPartitionSpecNotFound, df.SpecID())
			}
			isUnpartitioned = spec.IsUnpartitioned()
			unpartitionedBySpecID[df.SpecID()] = isUnpartitioned
		}
		if isUnpartitioned {
			idx.global = append(idx.global, indexedEntry)

			continue
		}

		partition := dataFilePartition(df)
		key, err := newEqualityDeletePartitionKey(df.SpecID(), partition)
		if err != nil {
			return nil, fmt.Errorf("indexing equality delete file %s: %w", df.FilePath(), err)
		}
		if idx.byPartition == nil {
			idx.byPartition = make(map[equalityDeletePartitionKey][]equalityDeleteIndexEntry)
		}
		idx.byPartition[key] = append(idx.byPartition[key], indexedEntry)
	}

	sortBySequence := func(entries []equalityDeleteIndexEntry) {
		slices.SortStableFunc(entries, func(a, b equalityDeleteIndexEntry) int {
			return cmp.Compare(a.entry.SequenceNum(), b.entry.SequenceNum())
		})
	}
	sortBySequence(idx.global)
	for _, partitionEntries := range idx.byPartition {
		sortBySequence(partitionEntries)
	}

	return idx, nil
}

// forDataFile returns equality deletes with a strictly greater sequence number.
// The strict comparison keeps rows added in the same snapshot as an equality
// delete, matching the RowDelta semantics and Java's DeleteFileIndex.
func (idx *equalityDeleteIndex) forDataFile(dataEntry iceberg.ManifestEntry) ([]iceberg.DataFile, error) {
	if len(idx.global) == 0 && len(idx.byPartition) == 0 {
		return nil, nil
	}

	partitionEntries := []equalityDeleteIndexEntry(nil)
	if len(idx.byPartition) > 0 {
		dataFile := dataEntry.DataFile()
		partition := dataFilePartition(dataFile)
		if len(partition) > 0 {
			key, err := newEqualityDeletePartitionKey(dataFile.SpecID(), partition)
			if err != nil {
				return nil, fmt.Errorf("matching equality deletes to data file %s: %w", dataFile.FilePath(), err)
			}
			partitionEntries = idx.byPartition[key]
		}
	}

	dataSeqNum := dataEntry.SequenceNum()
	var dataStats *equalityDeleteDataFileStats
	if idx.schema != nil {
		stats := equalityDeleteDataFileStatsFor(dataEntry.DataFile())
		dataStats = &stats
	}

	out := appendEqualityDeletesAfter(nil, idx.global, dataSeqNum, dataStats)
	out = appendEqualityDeletesAfter(out, partitionEntries, dataSeqNum, dataStats)

	return out, nil
}

func appendEqualityDeletesAfter(
	out []iceberg.DataFile,
	entries []equalityDeleteIndexEntry,
	dataSeqNum int64,
	dataStats *equalityDeleteDataFileStats,
) []iceberg.DataFile {
	start := sort.Search(len(entries), func(i int) bool {
		return entries[i].entry.SequenceNum() > dataSeqNum
	})
	for _, entry := range entries[start:] {
		if !equalityDeleteCanContainData(dataStats, entry.fields) {
			continue
		}

		out = append(out, entry.entry.DataFile())
	}

	return out
}

func equalityDeleteCanContainData(
	dataStats *equalityDeleteDataFileStats,
	deleteFields []equalityDeleteFieldMetrics,
) bool {
	if dataStats == nil {
		return true
	}

	for i := range deleteFields {
		deleteField := &deleteFields[i]
		if deleteField.typ == nil {
			// The field may have been dropped from the current schema. Its
			// bounds cannot be decoded safely, so retain the delete file.
			continue
		}
		if !deleteField.primitive {
			// Metrics are not defined for nested fields.
			continue
		}

		if !deleteField.required {
			if equalityDeleteDataContainsNull(dataStats.nullCounts, deleteField) &&
				equalityDeleteFieldContainsNull(deleteField) {
				// Both files may contain null for this field. Ranges cannot
				// prove that a null equality key is absent.
				continue
			}

			if equalityDeleteDataAllNull(*dataStats, deleteField) &&
				equalityDeleteFieldAllNonNull(deleteField) {
				return false
			}

			if equalityDeleteFieldAllNull(deleteField) &&
				equalityDeleteDataAllNonNull(dataStats.nullCounts, deleteField) {
				return false
			}
		}

		if !deleteField.canUseRange ||
			(deleteField.floatType && !equalityDeleteFloatRangesAreKnown(*dataStats, deleteField)) {
			continue
		}
		dataBounds := equalityDeleteDataFileBoundsFor(dataStats, deleteField)
		if !dataBounds.usable || !deleteField.hasDecodedBounds {
			// Missing bounds are not evidence that the ranges are disjoint.
			continue
		}

		if equalityDeleteMetricValueCompare(&dataBounds.lower, &deleteField.upperValue) > 0 ||
			equalityDeleteMetricValueCompare(&deleteField.lowerValue, &dataBounds.upper) > 0 {
			return false
		}
	}

	return true
}

func equalityDeleteDataContainsNull(nullCounts map[int]int64, field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return false
	}
	if nullCounts == nil {
		return true
	}

	nullCount, ok := nullCounts[field.fieldID]

	return !ok || nullCount != 0
}

func equalityDeleteFieldContainsNull(field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return false
	}
	if !field.hasNullCount {
		return true
	}

	return field.nullCount != 0
}

func equalityDeleteDataAllNull(stats equalityDeleteDataFileStats, field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return false
	}

	nullCount, hasNullCount := stats.nullCounts[field.fieldID]
	valueCount, hasValueCount := stats.valueCounts[field.fieldID]

	return hasNullCount && hasValueCount && nullCount >= 0 && nullCount == valueCount
}

func equalityDeleteFieldAllNull(field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return false
	}

	return field.hasNullCount && field.hasValueCount && field.nullCount >= 0 && field.nullCount == field.valueCount
}

func equalityDeleteDataAllNonNull(nullCounts map[int]int64, field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return true
	}
	if nullCounts == nil {
		return false
	}

	nullCount, ok := nullCounts[field.fieldID]

	return ok && nullCount == 0
}

func equalityDeleteFieldAllNonNull(field *equalityDeleteFieldMetrics) bool {
	if field.required {
		return true
	}

	return field.hasNullCount && field.nullCount == 0
}

func equalityDeleteFloatRangesAreKnown(
	dataStats equalityDeleteDataFileStats,
	field *equalityDeleteFieldMetrics,
) bool {
	// Float bounds exclude NaN values. Only use them when both files
	// explicitly prove that the field contains no NaNs.
	dataNaNCount, dataHasNaNCount := dataStats.nanCounts[field.fieldID]

	return dataHasNaNCount && dataNaNCount == 0 && field.hasNaNCount && field.nanCount == 0
}

func equalityDeleteDataFileBoundsFor(
	dataStats *equalityDeleteDataFileStats,
	field *equalityDeleteFieldMetrics,
) *equalityDeleteDataFileBounds {
	if bounds, ok := dataStats.decodedBounds[field.fieldID]; ok {
		return bounds
	}

	bounds := &equalityDeleteDataFileBounds{}
	dataLower, hasDataLower := dataStats.lowerBounds[field.fieldID]
	dataUpper, hasDataUpper := dataStats.upperBounds[field.fieldID]
	if hasDataLower && hasDataUpper && field.canUseRange {
		lowerValue, lowerOK := equalityDeleteMetricValueFromBytes(
			field.typ, field.rangeKind, dataLower, false)
		upperValue, upperOK := equalityDeleteMetricValueFromBytes(
			field.typ, field.rangeKind, dataUpper, false)
		if lowerOK && upperOK && equalityDeleteMetricValueCompare(&lowerValue, &upperValue) <= 0 {
			*bounds = equalityDeleteDataFileBounds{
				lower:  lowerValue,
				upper:  upperValue,
				usable: true,
			}
		}
	}

	if dataStats.decodedBounds == nil {
		dataStats.decodedBounds = make(map[int]*equalityDeleteDataFileBounds)
	}
	dataStats.decodedBounds[field.fieldID] = bounds

	return bounds
}

func equalityDeleteRangeKindForType(typ iceberg.Type) equalityDeleteRangeKind {
	switch typ.(type) {
	case iceberg.BooleanType:
		return equalityDeleteRangeBool
	case iceberg.Int32Type:
		return equalityDeleteRangeInt32
	case iceberg.Int64Type:
		return equalityDeleteRangeInt64
	case iceberg.Float32Type:
		return equalityDeleteRangeFloat32
	case iceberg.Float64Type:
		return equalityDeleteRangeFloat64
	case iceberg.DateType:
		return equalityDeleteRangeDate
	case iceberg.TimeType:
		return equalityDeleteRangeTime
	case iceberg.TimestampType, iceberg.TimestampTzType:
		return equalityDeleteRangeTimestamp
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return equalityDeleteRangeTimestampNano
	case iceberg.StringType:
		return equalityDeleteRangeString
	case iceberg.BinaryType, iceberg.FixedType:
		return equalityDeleteRangeBytes
	case iceberg.UUIDType:
		return equalityDeleteRangeUUID
	case iceberg.DecimalType:
		return equalityDeleteRangeDecimal
	default:
		return equalityDeleteRangeNone
	}
}

func equalityDeleteFloatType(typ iceberg.Type) bool {
	switch typ.(type) {
	case iceberg.Float32Type, iceberg.Float64Type:
		return true
	default:
		return false
	}
}

func equalityDeleteLiteralIsNaN(literal iceberg.Literal) bool {
	switch value := literal.(type) {
	case iceberg.Float32Literal:
		return math.IsNaN(float64(value))
	case iceberg.Float64Literal:
		return math.IsNaN(float64(value))
	default:
		return false
	}
}
