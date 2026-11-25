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

package rest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"

	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/table"
	"github.com/twmb/avro/atype"
)

// The exported REST wire types below are intentional: the low-level planning
// methods expose ScanTasks directly, while DecodeScanTasks is the boundary for
// callers that want table.FileScanTask domain objects.

// RESTCountMap is the REST column-id-to-count representation. Keys and values
// are parallel arrays because JSON object keys cannot preserve integer IDs.
type RESTCountMap struct {
	Keys   []int   `json:"keys"`
	Values []int64 `json:"values"`
}

// RESTValueMap is the REST column-id-to-binary representation used for lower
// and upper bounds. Java's ContentFileParser encodes each raw Iceberg binary
// value as a hexadecimal string; these are not typed JSON partition values.
type RESTValueMap struct {
	Keys   []int    `json:"keys"`
	Values []string `json:"values"`
}

// RESTContentFile contains the fields common to REST data and delete files.
// Partition values remain raw until DecodeScanTasks resolves spec-id and the
// result type of each partition transform.
type RESTContentFile struct {
	SpecID          int               `json:"spec-id"`
	Partition       []json.RawMessage `json:"partition"`
	Content         string            `json:"content"`
	FilePath        string            `json:"file-path"`
	FileFormat      string            `json:"file-format"`
	FileSizeInBytes int64             `json:"file-size-in-bytes"`
	RecordCount     int64             `json:"record-count"`
	KeyMetadata     *string           `json:"key-metadata,omitempty"`
	SplitOffsets    []int64           `json:"split-offsets,omitempty"`
	SortOrderID     *int              `json:"sort-order-id,omitempty"`
}

// RESTDataFile is the data-file arm of the REST ContentFile union.
type RESTDataFile struct {
	RESTContentFile

	FirstRowID      *int64        `json:"first-row-id,omitempty"`
	ColumnSizes     *RESTCountMap `json:"column-sizes,omitempty"`
	ValueCounts     *RESTCountMap `json:"value-counts,omitempty"`
	NullValueCounts *RESTCountMap `json:"null-value-counts,omitempty"`
	NaNValueCounts  *RESTCountMap `json:"nan-value-counts,omitempty"`
	LowerBounds     *RESTValueMap `json:"lower-bounds,omitempty"`
	UpperBounds     *RESTValueMap `json:"upper-bounds,omitempty"`
}

// RESTFileScanTask is the REST FileScanTask wire payload. Delete references
// are zero-based indices into the DeleteFiles slice of the same ScanTasks
// envelope; they must be resolved before responses are combined.
type RESTFileScanTask struct {
	DataFile             *RESTDataFile   `json:"data-file"`
	DeleteFileReferences []int           `json:"delete-file-references,omitempty"`
	ResidualFilter       json.RawMessage `json:"residual-filter,omitempty"`
}

// RESTDeleteFile is the position/equality arm of the REST ContentFile union.
// Content discriminates the variant. Puffin position deletes are decoded as
// deletion vectors and associated with the data file that references them.
type RESTDeleteFile struct {
	RESTContentFile

	EqualityIDs []int `json:"equality-ids,omitempty"`
	// ReferencedDataFile is emitted by Java's ContentFileParser for
	// file-scoped position deletes and deletion vectors. It is not currently
	// declared by the REST OpenAPI DeleteFile schema, so clients must also
	// support deriving a DV's target from its FileScanTask association.
	ReferencedDataFile *string `json:"referenced-data-file,omitempty"`
	ContentOffset      *int64  `json:"content-offset,omitempty"`
	ContentSizeInBytes *int64  `json:"content-size-in-bytes,omitempty"`
}

// DecodeScanTasks converts one REST ScanTasks envelope into domain scan tasks.
// Delete-file references are scoped to one envelope, so callers must invoke
// this before combining inline and fetchScanTasks responses.
//
// schema is the schema selected for the scan and is used to decode residual
// literals. When nil, metadata.CurrentSchema is used.
// fallbackResidual is used when a task omits residual-filter, as required by
// the REST specification; it is commonly the original scan filter.
func DecodeScanTasks(
	wire ScanTasks,
	metadata table.ScanPlanningMetadata,
	schema *iceberg.Schema,
	fallbackResidual iceberg.BooleanExpression,
) ([]table.FileScanTask, error) {
	return decodeScanTasks(wire, metadata, schema, fallbackResidual, newPartitionDecodePlanCache())
}

func decodeScanTasks(
	wire ScanTasks,
	metadata table.ScanPlanningMetadata,
	schema *iceberg.Schema,
	fallbackResidual iceberg.BooleanExpression,
	plans *partitionDecodePlanCache,
) ([]table.FileScanTask, error) {
	if metadata == nil {
		return nil, fmt.Errorf("%w: decoding scan tasks: table metadata is required", ErrRESTError)
	}
	if schema == nil {
		schema = metadata.CurrentSchema()
	}
	if schema == nil {
		return nil, fmt.Errorf("%w: decoding scan tasks: scan schema is required", ErrRESTError)
	}
	if len(wire.FileScanTasks) == 0 {
		if len(wire.DeleteFiles) != 0 {
			return nil, fmt.Errorf("%w: decoding scan tasks: %d delete files have no file scan tasks", ErrRESTError, len(wire.DeleteFiles))
		}

		return nil, nil
	}

	// Decode every delete before resolving references. This validates all wire
	// entries, including an unreferenced entry that will be rejected below.
	deletes := make([]iceberg.DataFile, len(wire.DeleteFiles))
	for i := range wire.DeleteFiles {
		decoded, err := decodeRESTDeleteFile(wire.DeleteFiles[i], metadata, "", plans)
		if err != nil {
			return nil, fmt.Errorf("%w: decoding scan tasks: delete-files[%d]: %w", ErrRESTError, i, err)
		}
		deletes[i] = decoded
	}

	out := make([]table.FileScanTask, 0, len(wire.FileScanTasks))
	dvOwners := make(map[int]string)
	referencedDeletes := make([]bool, len(deletes))
	for i := range wire.FileScanTasks {
		wireTask := wire.FileScanTasks[i]
		if wireTask.DataFile == nil {
			return nil, fmt.Errorf("%w: decoding scan tasks: file-scan-tasks[%d] is missing data-file", ErrRESTError, i)
		}

		dataFile, err := decodeRESTDataFile(*wireTask.DataFile, metadata, plans)
		if err != nil {
			return nil, fmt.Errorf("%w: decoding scan tasks: file-scan-tasks[%d].data-file: %w", ErrRESTError, i, err)
		}

		residual, err := decodeTaskResidual(wireTask.ResidualFilter, schema, fallbackResidual)
		if err != nil {
			return nil, fmt.Errorf("%w: decoding scan tasks: file-scan-tasks[%d].residual-filter: %w", ErrRESTError, i, err)
		}

		task := table.FileScanTask{
			File:       dataFile,
			Start:      0,
			Length:     dataFile.FileSizeBytes(),
			Residual:   residual,
			FirstRowID: dataFile.FirstRowID(),
		}
		// The REST FileScanTask schema does not carry manifest data sequence
		// numbers, so DataSequenceNumber intentionally remains nil.

		seenRefs := make(map[int]struct{}, len(wireTask.DeleteFileReferences))
		for j, ref := range wireTask.DeleteFileReferences {
			if ref < 0 || ref >= len(deletes) {
				return nil, fmt.Errorf(
					"%w: decoding scan tasks: file-scan-tasks[%d].delete-file-references[%d] is %d, want 0 <= index < %d",
					ErrRESTError, i, j, ref, len(deletes))
			}
			if _, ok := seenRefs[ref]; ok {
				return nil, fmt.Errorf(
					"%w: decoding scan tasks: file-scan-tasks[%d] repeats delete-file reference %d",
					ErrRESTError, i, ref)
			}
			seenRefs[ref] = struct{}{}
			referencedDeletes[ref] = true

			wireDelete := wire.DeleteFiles[ref]
			if wireDelete.ReferencedDataFile != nil && *wireDelete.ReferencedDataFile != dataFile.FilePath() {
				return nil, fmt.Errorf(
					"%w: decoding scan tasks: delete-files[%d].referenced-data-file is %q, task data-file is %q",
					ErrRESTError, ref, *wireDelete.ReferencedDataFile, dataFile.FilePath())
			}

			deleteFile := deletes[ref]
			if deleteFile.FileFormat() == iceberg.PuffinFile {
				if owner, ok := dvOwners[ref]; ok && owner != dataFile.FilePath() {
					return nil, fmt.Errorf(
						"%w: decoding scan tasks: deletion vector %d is referenced by both %q and %q",
						ErrRESTError, ref, owner, dataFile.FilePath())
				}
				dvOwners[ref] = dataFile.FilePath()
				// Derive referenced-data-file from the FileScanTask association
				// when the server omits Java's optional extension field.
				if wireDelete.ReferencedDataFile == nil {
					deleteFile, err = decodeRESTDeleteFile(wireDelete, metadata, dataFile.FilePath(), plans)
					if err != nil {
						return nil, fmt.Errorf("%w: decoding scan tasks: delete-files[%d]: %w", ErrRESTError, ref, err)
					}
				}
			}

			switch deleteFile.ContentType() {
			case iceberg.EntryContentEqDeletes:
				task.EqualityDeleteFiles = append(task.EqualityDeleteFiles, deleteFile)
			case iceberg.EntryContentPosDeletes:
				if deleteFile.FileFormat() == iceberg.PuffinFile {
					task.DeletionVectorFiles = append(task.DeletionVectorFiles, deleteFile)
				} else {
					task.DeleteFiles = append(task.DeleteFiles, deleteFile)
				}
			default:
				return nil, fmt.Errorf("%w: decoding scan tasks: delete-files[%d] has non-delete content", ErrRESTError, ref)
			}
		}

		out = append(out, task)
	}

	for i, referenced := range referencedDeletes {
		if !referenced {
			return nil, fmt.Errorf(
				"%w: decoding scan tasks: delete-files[%d] is not referenced by any file scan task",
				ErrRESTError, i)
		}
	}

	return out, nil
}

func decodeRESTDataFile(
	wire RESTDataFile,
	metadata table.ScanPlanningMetadata,
	plans *partitionDecodePlanCache,
) (iceberg.DataFile, error) {
	builder, _, err := contentFileBuilder(wire.RESTContentFile, iceberg.EntryContentData, metadata, plans)
	if err != nil {
		return nil, err
	}

	columnSizes, err := decodeCountMap("column-sizes", wire.ColumnSizes)
	if err != nil {
		return nil, err
	}
	valueCounts, err := decodeCountMap("value-counts", wire.ValueCounts)
	if err != nil {
		return nil, err
	}
	nullCounts, err := decodeCountMap("null-value-counts", wire.NullValueCounts)
	if err != nil {
		return nil, err
	}
	nanCounts, err := decodeCountMap("nan-value-counts", wire.NaNValueCounts)
	if err != nil {
		return nil, err
	}
	lowerBounds, err := decodeValueMap("lower-bounds", wire.LowerBounds)
	if err != nil {
		return nil, err
	}
	upperBounds, err := decodeValueMap("upper-bounds", wire.UpperBounds)
	if err != nil {
		return nil, err
	}

	if wire.ColumnSizes != nil {
		builder.ColumnSizes(columnSizes)
	}
	if wire.ValueCounts != nil {
		builder.ValueCounts(valueCounts)
	}
	if wire.NullValueCounts != nil {
		builder.NullValueCounts(nullCounts)
	}
	if wire.NaNValueCounts != nil {
		builder.NaNValueCounts(nanCounts)
	}
	if wire.LowerBounds != nil {
		builder.LowerBoundValues(lowerBounds)
	}
	if wire.UpperBounds != nil {
		builder.UpperBoundValues(upperBounds)
	}
	if wire.FirstRowID != nil {
		if *wire.FirstRowID < 0 {
			return nil, fmt.Errorf("first-row-id must be non-negative: %d", *wire.FirstRowID)
		}
		builder.FirstRowID(*wire.FirstRowID)
	}

	return builder.Build(), nil
}

func decodeRESTDeleteFile(
	wire RESTDeleteFile,
	metadata table.ScanPlanningMetadata,
	referencedDataFile string,
	plans *partitionDecodePlanCache,
) (iceberg.DataFile, error) {
	content, ok := parseRESTFileContent(wire.Content)
	if !ok || content == iceberg.EntryContentData {
		return nil, fmt.Errorf("unknown delete content %q", wire.Content)
	}

	builder, format, err := contentFileBuilder(wire.RESTContentFile, content, metadata, plans)
	if err != nil {
		return nil, err
	}

	switch content {
	case iceberg.EntryContentEqDeletes:
		if wire.ReferencedDataFile != nil || wire.ContentOffset != nil || wire.ContentSizeInBytes != nil {
			return nil, errors.New("equality-deletes file must not carry position-delete reference or blob offsets")
		}
		if len(wire.EqualityIDs) == 0 {
			return nil, errors.New("equality-deletes file is missing equality-ids")
		}
		seen := make(map[int]struct{}, len(wire.EqualityIDs))
		for i, id := range wire.EqualityIDs {
			if id <= 0 {
				return nil, fmt.Errorf("equality-ids[%d] must be positive: %d", i, id)
			}
			if _, ok := seen[id]; ok {
				return nil, fmt.Errorf("equality-ids contains duplicate field ID %d", id)
			}
			seen[id] = struct{}{}
		}
		builder.EqualityFieldIDs(slices.Clone(wire.EqualityIDs))
	case iceberg.EntryContentPosDeletes:
		if len(wire.EqualityIDs) != 0 {
			return nil, errors.New("position-deletes file must not carry equality-ids")
		}
		if wire.ContentOffset != nil {
			if *wire.ContentOffset < 0 {
				return nil, fmt.Errorf("content-offset must be non-negative: %d", *wire.ContentOffset)
			}
			if wire.ContentSizeInBytes == nil {
				return nil, errors.New("content-size-in-bytes is required when content-offset is present")
			}
			builder.ContentOffset(*wire.ContentOffset)
		}
		if wire.ContentSizeInBytes != nil {
			if *wire.ContentSizeInBytes < 0 {
				return nil, fmt.Errorf("content-size-in-bytes must be non-negative: %d", *wire.ContentSizeInBytes)
			}
			builder.ContentSizeInBytes(*wire.ContentSizeInBytes)
		}
		if wire.ReferencedDataFile != nil {
			if *wire.ReferencedDataFile == "" {
				return nil, errors.New("referenced-data-file cannot be empty")
			}
			if referencedDataFile != "" && *wire.ReferencedDataFile != referencedDataFile {
				return nil, fmt.Errorf(
					"referenced-data-file is %q, task data-file is %q",
					*wire.ReferencedDataFile, referencedDataFile)
			}
			builder.ReferencedDataFile(*wire.ReferencedDataFile)
		}
		if format == iceberg.PuffinFile {
			if wire.ContentOffset == nil || wire.ContentSizeInBytes == nil {
				return nil, errors.New("puffin deletion vector requires content-offset and content-size-in-bytes")
			}
			if wire.ReferencedDataFile == nil && referencedDataFile != "" {
				builder.ReferencedDataFile(referencedDataFile)
			}
		}
	}

	return builder.Build(), nil
}

func contentFileBuilder(
	wire RESTContentFile,
	expectedContent iceberg.ManifestEntryContent,
	metadata table.ScanPlanningMetadata,
	plans *partitionDecodePlanCache,
) (*iceberg.DataFileBuilder, iceberg.FileFormat, error) {
	actualContent, ok := parseRESTFileContent(wire.Content)
	wantContent := map[iceberg.ManifestEntryContent]string{
		iceberg.EntryContentData:       "data",
		iceberg.EntryContentPosDeletes: "position-deletes",
		iceberg.EntryContentEqDeletes:  "equality-deletes",
	}[expectedContent]
	if !ok || actualContent != expectedContent {
		return nil, "", fmt.Errorf("content is %q, want %q", wire.Content, wantContent)
	}
	if wire.RecordCount < 0 {
		return nil, "", fmt.Errorf("record-count must be non-negative: %d", wire.RecordCount)
	}
	if wire.FileSizeInBytes < 0 {
		return nil, "", fmt.Errorf("file-size-in-bytes must be non-negative: %d", wire.FileSizeInBytes)
	}

	plan, err := plans.planFor(wire.SpecID, metadata)
	if err != nil {
		return nil, "", err
	}
	partition, logicalTypes, fixedSizes, err := decodePartition(wire.Partition, plan)
	if err != nil {
		return nil, "", err
	}

	format, err := iceberg.FileFormatFromString(wire.FileFormat)
	if err != nil {
		return nil, "", err
	}
	builder, err := iceberg.NewDataFileBuilder(
		*plan.spec,
		expectedContent,
		wire.FilePath,
		format,
		partition,
		logicalTypes,
		fixedSizes,
		wire.RecordCount,
		wire.FileSizeInBytes,
	)
	if err != nil {
		return nil, "", err
	}

	if wire.KeyMetadata != nil {
		key, err := hex.DecodeString(*wire.KeyMetadata)
		if err != nil {
			return nil, "", fmt.Errorf("invalid key-metadata hexadecimal value: %w", err)
		}
		builder.KeyMetadata(key)
	}
	if wire.SplitOffsets != nil {
		for i, offset := range wire.SplitOffsets {
			if offset < 0 {
				return nil, "", fmt.Errorf("split-offsets[%d] must be non-negative: %d", i, offset)
			}
			if i > 0 && offset <= wire.SplitOffsets[i-1] {
				return nil, "", errors.New("split-offsets must be strictly increasing")
			}
		}
		builder.SplitOffsets(slices.Clone(wire.SplitOffsets))
	}
	if wire.SortOrderID != nil {
		if *wire.SortOrderID < 0 {
			return nil, "", fmt.Errorf("sort-order-id must be non-negative: %d", *wire.SortOrderID)
		}
		builder.SortOrderID(*wire.SortOrderID)
	}

	return builder, format, nil
}

func parseRESTFileContent(value string) (iceberg.ManifestEntryContent, bool) {
	switch value {
	case "data", "DATA":
		return iceberg.EntryContentData, true
	case "position-deletes", "POSITION_DELETES":
		return iceberg.EntryContentPosDeletes, true
	case "equality-deletes", "EQUALITY_DELETES":
		return iceberg.EntryContentEqDeletes, true
	default:
		return 0, false
	}
}

// partitionDecodePlanCache is scoped to one remote scan operation. Metadata is
// immutable for the operation, so plans can safely retain resolved types and
// the partition spec while decoded values remain per-file allocations.
type partitionDecodePlanCache struct {
	plans map[int]*partitionDecodePlan
}

type partitionDecodePlan struct {
	spec   *iceberg.PartitionSpec
	fields []partitionDecodeField
}

type partitionDecodeField struct {
	fieldID         int
	fieldName       string
	sourceID        int
	sourceTypeFound bool
	resultType      iceberg.Type
	logicalType     string
	fixedSize       int
	hasFixedSize    bool
}

func newPartitionDecodePlanCache() *partitionDecodePlanCache {
	return &partitionDecodePlanCache{plans: make(map[int]*partitionDecodePlan)}
}

func (c *partitionDecodePlanCache) planFor(
	specID int,
	metadata table.ScanPlanningMetadata,
) (*partitionDecodePlan, error) {
	if plan, ok := c.plans[specID]; ok {
		return plan, nil
	}

	spec := metadata.PartitionSpecByID(specID)
	if spec == nil {
		return nil, fmt.Errorf("unknown partition spec ID %d", specID)
	}

	plan, err := newPartitionDecodePlan(spec, metadata)
	if err != nil {
		return nil, err
	}
	c.plans[specID] = plan

	return plan, nil
}

func newPartitionDecodePlan(spec *iceberg.PartitionSpec, metadata table.ScanPlanningMetadata) (*partitionDecodePlan, error) {
	plan := &partitionDecodePlan{
		spec:   spec,
		fields: make([]partitionDecodeField, spec.NumFields()),
	}
	for i, field := range spec.Fields() {
		transform := reflect.ValueOf(field.Transform)
		if !transform.IsValid() || (transform.Kind() == reflect.Pointer && transform.IsNil()) {
			return nil, fmt.Errorf("%w: partition field %q (ID %d) has no transform",
				iceberg.ErrInvalidTransform, field.Name, field.FieldID)
		}

		fieldPlan := partitionDecodeField{
			fieldID:   field.FieldID,
			fieldName: field.Name,
			sourceID:  field.SourceID(),
		}
		sourceType, ok := findFieldType(fieldPlan.sourceID, metadata)
		fieldPlan.sourceTypeFound = ok
		if ok {
			fieldPlan.resultType = field.Transform.ResultType(sourceType)
			fieldPlan.logicalType, fieldPlan.fixedSize, fieldPlan.hasFixedSize = partitionLogicalType(fieldPlan.resultType)
		}
		plan.fields[i] = fieldPlan
	}

	return plan, nil
}

func decodePartition(
	values []json.RawMessage,
	plan *partitionDecodePlan,
) (map[int]any, map[int]string, map[int]int, error) {
	if len(values) != len(plan.fields) {
		return nil, nil, nil, fmt.Errorf(
			"partition for spec ID %d has %d values, want %d", plan.spec.ID(), len(values), len(plan.fields))
	}

	partition := make(map[int]any, len(values))
	logicalTypes := make(map[int]string)
	fixedSizes := make(map[int]int)
	for i, field := range plan.fields {
		raw := values[i]
		if isJSONNull(raw) {
			partition[field.fieldID] = nil

			continue
		}

		if !field.sourceTypeFound {
			return nil, nil, nil, fmt.Errorf(
				"partition field %q (ID %d) has unknown source field ID %d",
				field.fieldName, field.fieldID, field.sourceID)
		}
		literal, err := decodePartitionLiteral(raw, field.resultType)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("partition[%d] for field %q: %w", i, field.fieldName, err)
		}
		partition[field.fieldID] = literal.Any()
		if field.logicalType != "" {
			logicalTypes[field.fieldID] = field.logicalType
		}
		if field.hasFixedSize {
			fixedSizes[field.fieldID] = field.fixedSize
		}
	}

	return partition, logicalTypes, fixedSizes, nil
}

func findFieldType(fieldID int, metadata table.ScanPlanningMetadata) (iceberg.Type, bool) {
	if current := metadata.CurrentSchema(); current != nil {
		if typ, ok := current.FindTypeByID(fieldID); ok {
			return typ, true
		}
	}
	for _, schema := range metadata.Schemas() {
		if schema == nil {
			continue
		}
		if typ, ok := schema.FindTypeByID(fieldID); ok {
			return typ, true
		}
	}

	return nil, false
}

func partitionLogicalType(typ iceberg.Type) (string, int, bool) {
	switch typ := typ.(type) {
	case iceberg.DateType:
		return atype.Date, 0, false
	case iceberg.TimeType:
		return atype.TimeMicros, 0, false
	case iceberg.TimestampType, iceberg.TimestampTzType:
		return atype.TimestampMicros, 0, false
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return atype.TimestampNanos, 0, false
	case iceberg.DecimalType:
		return atype.Decimal, typ.Scale(), true
	case iceberg.UUIDType:
		return atype.UUID, 0, false
	}

	return "", 0, false
}

func decodeCountMap(name string, wire *RESTCountMap) (map[int]int64, error) {
	if wire == nil {
		return nil, nil
	}
	if len(wire.Keys) != len(wire.Values) {
		return nil, fmt.Errorf("%s has %d keys and %d values", name, len(wire.Keys), len(wire.Values))
	}

	out := make(map[int]int64, len(wire.Keys))
	for i, key := range wire.Keys {
		if key <= 0 {
			return nil, fmt.Errorf("%s contains invalid field ID %d", name, key)
		}
		if _, ok := out[key]; ok {
			return nil, fmt.Errorf("%s contains duplicate field ID %d", name, key)
		}
		if wire.Values[i] < 0 {
			return nil, fmt.Errorf("%s[%d] for field ID %d is negative", name, i, key)
		}
		out[key] = wire.Values[i]
	}

	return out, nil
}

func decodeValueMap(name string, wire *RESTValueMap) (map[int][]byte, error) {
	if wire == nil {
		return nil, nil
	}
	if len(wire.Keys) != len(wire.Values) {
		return nil, fmt.Errorf("%s has %d keys and %d values", name, len(wire.Keys), len(wire.Values))
	}

	out := make(map[int][]byte, len(wire.Keys))
	for i, key := range wire.Keys {
		if key <= 0 {
			return nil, fmt.Errorf("%s contains invalid field ID %d", name, key)
		}
		if _, ok := out[key]; ok {
			return nil, fmt.Errorf("%s contains duplicate field ID %d", name, key)
		}
		decoded, err := hex.DecodeString(wire.Values[i])
		if err != nil {
			return nil, fmt.Errorf("%s[%d] for field ID %d is not hexadecimal: %w", name, i, key, err)
		}
		out[key] = decoded
	}

	return out, nil
}

func decodePartitionLiteral(raw json.RawMessage, typ iceberg.Type) (iceberg.Literal, error) {
	if isJSONNull(raw) {
		return nil, fmt.Errorf("null is not valid for %s", typ)
	}

	var literal iceberg.Literal
	switch typ := typ.(type) {
	case iceberg.BooleanType:
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid boolean value: %w", err)
		}
		literal = iceberg.BoolLiteral(value)
	case iceberg.Int32Type:
		value, err := decodeJSONInteger(raw, 32)
		if err != nil {
			return nil, err
		}
		literal = iceberg.Int32Literal(int32(value))
	case iceberg.Int64Type:
		value, err := decodeJSONInteger(raw, 64)
		if err != nil {
			return nil, err
		}
		literal = iceberg.Int64Literal(value)
	case iceberg.Float32Type:
		value, err := decodeJSONFloat(raw, 32)
		if err != nil {
			return nil, err
		}
		literal = iceberg.Float32Literal(float32(value))
	case iceberg.Float64Type:
		value, err := decodeJSONFloat(raw, 64)
		if err != nil {
			return nil, err
		}
		literal = iceberg.Float64Literal(value)
	case iceberg.BinaryType, iceberg.FixedType:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid %s hexadecimal value: %w", typ, err)
		}
		decoded, err := hex.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("invalid %s hexadecimal value: %w", typ, err)
		}
		literal, err = iceberg.BinaryLiteral(decoded).To(typ)
		if err != nil {
			return nil, err
		}
	case iceberg.StringType, iceberg.UUIDType, iceberg.DateType, iceberg.TimeType,
		iceberg.TimestampType, iceberg.TimestampTzType, iceberg.TimestampNsType,
		iceberg.TimestampTzNsType, iceberg.DecimalType:
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid %s value: %w", typ, err)
		}
		var err error
		literal, err = iceberg.StringLiteral(value).To(typ)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported REST primitive type %s", typ)
	}

	return literal, nil
}

func decodeJSONInteger(raw json.RawMessage, bitSize int) (int64, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value json.Number
	if err := dec.Decode(&value); err != nil {
		return 0, fmt.Errorf("invalid integer value: %w", err)
	}
	parsed, err := value.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid integer value %q: %w", value, err)
	}
	if bitSize == 32 && (parsed < math.MinInt32 || parsed > math.MaxInt32) {
		return 0, fmt.Errorf("integer value %d is outside int32 range", parsed)
	}

	return parsed, nil
}

func decodeJSONFloat(raw json.RawMessage, bitSize int) (float64, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value json.Number
	if err := dec.Decode(&value); err != nil {
		return 0, fmt.Errorf("invalid floating-point value: %w", err)
	}
	parsed, err := value.Float64()
	if err != nil {
		return 0, fmt.Errorf("invalid floating-point value %q: %w", value, err)
	}
	if math.IsInf(parsed, 0) || math.IsNaN(parsed) {
		return 0, errors.New("floating-point value must be finite")
	}
	if bitSize == 32 && (parsed > math.MaxFloat32 || parsed < -math.MaxFloat32) {
		return 0, fmt.Errorf("floating-point value %v is outside float32 range", parsed)
	}

	return parsed, nil
}

func decodeTaskResidual(
	raw json.RawMessage,
	schema *iceberg.Schema,
	fallback iceberg.BooleanExpression,
) (iceberg.BooleanExpression, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fallback, nil
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return nil, errors.New("explicit null residual-filter is invalid")
	}

	// Accept the object spelling from the OpenAPI schema in addition to the
	// bare booleans emitted by Java's ExpressionParser.
	var constant struct {
		Type string `json:"type"`
	}
	if trimmed[0] == '{' && json.Unmarshal(trimmed, &constant) == nil {
		switch constant.Type {
		case "true":
			return iceberg.AlwaysTrue{}, nil
		case "false":
			return iceberg.AlwaysFalse{}, nil
		}
	}

	return iceberg.ParseExpr(trimmed, schema)
}

func isJSONNull(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)

	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}
