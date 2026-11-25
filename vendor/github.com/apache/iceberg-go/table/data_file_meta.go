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
	"errors"
	"fmt"
	"maps"

	"github.com/apache/arrow-go/v18/parquet/metadata"
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	tblutils "github.com/apache/iceberg-go/table/internal"
)

const referencedDataFilePathFieldID = 2147483546

// DataFileArgs collects everything needed to compute a fully populated
// [iceberg.DataFile] (data, equality-delete, or positional-delete) from
// a file's in-memory, format-specific metadata object without performing
// any filesystem reads.
type DataFileArgs struct {
	// Schema is the Iceberg schema the file's contents conform to.
	// Per-column statistics are matched by leaf-column path: each file
	// column's dotted path MUST equal the path derived from this schema's
	// field names. Embedded field-id metadata in the file (e.g. the Parquet
	// "PARQUET:field_id" key) is not consulted, so a column renamed relative
	// to the file will not resolve.
	Schema *iceberg.Schema

	// Spec is the partition spec the resulting DataFile is bound to —
	// typically tbl.Spec(). The zero value [iceberg.PartitionSpec]{} is
	// equivalent to *iceberg.UnpartitionedSpec and is accepted for
	// unpartitioned tables.
	Spec iceberg.PartitionSpec

	// Format identifies the on-disk file format and selects the internal
	// FileFormat implementation used to extract per-column statistics from
	// Metadata. It must match the concrete type of Metadata: the only supported
	// combination today is [iceberg.ParquetFile] with a *metadata.FileMetaData.
	Format iceberg.FileFormat

	// Metadata is the format-specific, in-memory file metadata object produced
	// by the external writer.
	//
	// Today only Format == [iceberg.ParquetFile] with *metadata.FileMetaData is supported
	Metadata any

	// FilePath is the fully qualified location the file bytes were
	// written to (the same path that will be recorded in the manifest entry).
	FilePath string

	// FileSize is the byte length of the object at FilePath. Must be
	// greater than zero.
	FileSize int64

	// Content selects which manifest-entry kind to produce — one of
	// [iceberg.EntryContentData], [iceberg.EntryContentEqDeletes] or
	// [iceberg.EntryContentPosDeletes].
	Content iceberg.ManifestEntryContent

	// PartitionValues maps spec field ID → partition value. Leave the map
	// nil/empty for unpartitioned tables, where any supplied value is rejected.
	//
	// Values must already be the transform results in the expected Go types.
	// A field is resolved as follows:
	//   - key present with a non-nil value: that value is recorded verbatim;
	//   - key present with a nil value, or key absent: treated as "not supplied"
	//     and, for order-preserving transforms (identity, truncate, the date/time
	//     transforms), inferred from the file's column statistics; for
	//     non-order-preserving transforms (bucket, void) it stays nil.
	//
	// Void-transform fields may be omitted (they always resolve to nil).
	//
	// This helper validates only that any supplied field IDs belong to the spec
	// (no stray IDs); it does not coerce or type-check the values themselves.
	// Callers are responsible for supplying values of the correct type.
	PartitionValues map[int]any

	// SortOrderID is the sort-order ID recorded on the manifest entry,
	// typically tbl.SortOrder().OrderID(). Zero means "unsorted".
	SortOrderID int

	// Properties are consulted only for write-side metrics-mode keys
	// (write.metadata.metrics.default and per-column
	// write.metadata.metrics.column.<name> overrides) that decide
	// which columns get full vs truncated vs no statistics. Pass
	// tbl.Properties() for table-default behavior; nil is safe.
	Properties iceberg.Properties

	// EqualityFieldIDs lists the schema field IDs that an equality
	// delete file matches on. Required when
	// Content == [iceberg.EntryContentEqDeletes] and MUST be empty
	// for every other Content value. Every listed ID must be present in Schema.
	EqualityFieldIDs []int

	// FormatVersion is the table's format version
	FormatVersion int

	// FirstRowID sets the data file's first_row_id (requires FormatVersion >= 3).
	// It must always be nil for delete files, and for v1/v2 data files (the
	// field is absent from those schemas).
	//
	// For v3 data files, whether it is required depends on which consuming API
	// the resulting DataFile is handed to:
	//   - [Transaction.AddDataFiles]: REQUIRED. These are externally-written
	//     files, so the library cannot fabricate row IDs for them; a nil
	//     first_row_id is rejected at commit time.
	//   - [Transaction.ReplaceDataFilesWithDataFiles]: REQUIRED for the added
	//     files, except when the caller opts into rewrite/compaction semantics
	//     (WithRewriteSemantics), where the manifest-list writer assigns it by
	//     inheritance and it may be left nil.
	//   - [Transaction.NewRowDelta] (via RowDelta.AddRows): NOT required — the
	//     value is assigned by inheritance from the manifest at commit time, so
	//     leaving it nil is correct.
	//
	// In short: set it explicitly whenever the file goes through AddDataFiles
	// (or a non-rewrite replace); otherwise leave it nil and let inheritance
	// assign it.
	FirstRowID *int64

	// ReferencedDataFile, when non-nil, records the single data file that a
	// file-scoped position-delete file applies to.
	//
	// It is only valid for position delete files. Leave it nil for data files, equality-delete files,
	// and partition-scoped position deletes.
	//
	// For partition-scoped position deletes whose file_path column references more
	// than one data file (cardinality > 1) — such a file has no single referenced data file.
	ReferencedDataFile *string
}

// DataFileFromMetadata builds a fully populated [iceberg.DataFile] from a
// file's in-memory, format-specific metadata object, extracting per-column
// statistics (column_sizes, value_counts, null_value_counts, lower_bounds,
// upper_bounds), split offsets, and the manifest-entry shape required for
// the requested content type. The format-specific extraction is dispatched
// through the internal FileFormat interface; no filesystem reads are
// performed.
//
// The returned DataFile is intended to be handed to
// [Transaction.AddDataFiles] (data files) or [Transaction.NewRowDelta]
// (delete files); it is NOT committed automatically. The file bytes
// described by args.Metadata must already have been uploaded to
// args.FilePath.
//
// Scope and limitations:
//   - Only Parquet is supported today (args.Format must be
//     [iceberg.ParquetFile] and args.Metadata a *metadata.FileMetaData).
//   - All three manifest-entry contents are supported end to end: data,
//     equality deletes, and positional deletes.
//   - Row lineage is supported: a data file's first_row_id can be set via
//     DataFileArgs.FirstRowID for format-version-3+ tables.
//   - Iceberg v3 deletion vectors (Puffin DVs) are not supported in this
//     helper function; it emits only Parquet position-delete files. A
//     file-scoped position delete may still record its target via
//     DataFileArgs.ReferencedDataFile; partition-scoped deletes leave it nil,
//     in which case engines may infer scoping from the file_path column bounds.
func DataFileFromMetadata(args DataFileArgs) (iceberg.DataFile, error) {
	if args.Schema == nil {
		return nil, errors.New("schema is required")
	}
	if args.Metadata == nil {
		return nil, errors.New("file metadata is required")
	}

	format := tblutils.GetFileFormat(args.Format)
	if format == nil {
		return nil, errors.New("unsupported file format: " + string(args.Format))
	}

	// Both nil checks are intentional:
	//   - args.Metadata == nil (above) catches an unset interface, while
	//   - pqMeta == nil below catches a typed-nil (*metadata.FileMetaData)(nil)
	pqMeta, ok := args.Metadata.(*metadata.FileMetaData)
	if !ok {
		return nil, fmt.Errorf(
			"unsupported metadata type for format %s: expected *metadata.FileMetaData, got %T",
			string(args.Format), args.Metadata)
	}
	if pqMeta == nil {
		return nil, errors.New("metadata pointer is nil")
	}
	if args.FilePath == "" {
		return nil, errors.New("file path is required")
	}
	if args.FileSize <= 0 {
		return nil, fmt.Errorf("file size must be greater than 0, got %d", args.FileSize)
	}

	switch args.Content {
	case iceberg.EntryContentData,
		iceberg.EntryContentEqDeletes,
		iceberg.EntryContentPosDeletes:
	default:
		return nil, fmt.Errorf("invalid content: %v", args.Content)
	}

	if args.Content == iceberg.EntryContentEqDeletes && len(args.EqualityFieldIDs) == 0 {
		return nil, errors.New("equality_ids is required for equality-delete files")
	}

	if args.Content != iceberg.EntryContentEqDeletes && len(args.EqualityFieldIDs) > 0 {
		return nil, fmt.Errorf("equality_ids must be empty for content %v", args.Content)
	}

	if args.Content != iceberg.EntryContentData && args.FirstRowID != nil {
		return nil, fmt.Errorf("first_row_id must be nil for delete files, got a non-nil value for content %v", args.Content)
	}

	if args.FirstRowID != nil && args.FormatVersion < 3 {
		return nil, fmt.Errorf(
			"first_row_id requires table format version >= 3, got v%d",
			args.FormatVersion)
	}

	if args.ReferencedDataFile != nil {
		if args.Content != iceberg.EntryContentPosDeletes {
			return nil, fmt.Errorf("referenced_data_file may only be set for position-delete files, got content %v", args.Content)
		}
		if *args.ReferencedDataFile == "" {
			return nil, errors.New("referenced_data_file must be non-empty when set")
		}
	}

	if err := validateContentSchema(args); err != nil {
		return nil, err
	}

	if err := validatePartitionValues(args.Spec, args.PartitionValues); err != nil {
		return nil, err
	}

	statsProps := args.Properties
	if args.ReferencedDataFile != nil {
		// A file-scoped position delete must carry exact (untruncated) file_path
		// The default truncate(16) metrics mode would clip long paths, thus, forcing
		// full metrics for that column.
		statsProps = maps.Clone(args.Properties)
		if statsProps == nil {
			statsProps = iceberg.Properties{}
		}
		if colName, ok := args.Schema.FindColumnName(referencedDataFilePathFieldID); ok {
			statsProps[MetricsModeColumnConfPrefix+"."+colName] = string(tblutils.MetricModeFull)
		}
	}

	statsPlan, err := computeStatsPlan(args.Schema, statsProps)
	if err != nil {
		return nil, fmt.Errorf("failed to build stats plan: %w", err)
	}

	pathMapping, err := format.PathToIDMapping(args.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to build path mapping: %w", err)
	}

	partitionValues := args.PartitionValues
	if partitionValues == nil {
		partitionValues = make(map[int]any)
	}

	var df iceberg.DataFile
	// DataFileStatsFromMeta and ToDataFile are the only calls wrapped in recover because they may panic on invalid input
	// (missing field IDs or builder validation failures)
	if err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				switch e := r.(type) {
				case error:
					err = fmt.Errorf("error encountered during extracting stats and building the data file: %w", e)
				default:
					err = fmt.Errorf("error encountered during extracting stats and building the data file: %v", e)
				}
			}
		}()

		stats := format.DataFileStatsFromMeta(
			args.Metadata,
			statsPlan,
			pathMapping,
			tblutils.VariantFieldIDsFromSchema(args.Schema),
			// Metadata-only path: no file is read here, so no arrow schema; AddFiles recomputes variant bounds in fileToDataFile.
			nil,
		)
		if stats == nil {
			return errors.New("failed to build data file stats from metadata")
		}

		if len(args.EqualityFieldIDs) > 0 {
			stats.EqualityFieldIDs = args.EqualityFieldIDs
		}

		df = stats.ToDataFile(tblutils.DataFileOpts{
			Schema:             args.Schema,
			Spec:               args.Spec,
			Path:               args.FilePath,
			Format:             args.Format,
			Content:            args.Content,
			FileSize:           args.FileSize,
			PartitionValues:    partitionValues,
			SortOrderID:        args.SortOrderID,
			FirstRowID:         args.FirstRowID,
			ReferencedDataFile: args.ReferencedDataFile,
		})

		return nil
	}(); err != nil {
		return nil, err
	}

	if args.ReferencedDataFile != nil {
		if err := verifyReferencedDataFile(df, *args.ReferencedDataFile); err != nil {
			return nil, err
		}
	}

	return df, nil
}

func verifyReferencedDataFile(df iceberg.DataFile, ref string) error {
	decode := func(bounds map[int][]byte, which string) (string, error) {
		raw, ok := bounds[referencedDataFilePathFieldID]
		if !ok {
			return "", fmt.Errorf(
				"referenced_data_file %q is set but the position-delete file_path column has no %s bound",
				ref, which)
		}
		lit, err := iceberg.LiteralFromBytes(iceberg.PrimitiveTypes.String, raw)
		if err != nil {
			return "", fmt.Errorf("failed to decode position-delete file_path %s bound: %w", which, err)
		}
		s, ok := lit.(iceberg.StringLiteral)
		if !ok {
			return "", fmt.Errorf("position-delete file_path %s bound decoded to unexpected type %T", which, lit)
		}

		return string(s), nil
	}

	lowerBounds, upperBounds := iceberginternal.BorrowedDataFileBounds(df)
	lower, err := decode(lowerBounds, "lower")
	if err != nil {
		return err
	}
	upper, err := decode(upperBounds, "upper")
	if err != nil {
		return err
	}

	if lower != ref || upper != ref {
		return fmt.Errorf(
			"referenced_data_file %q does not match the position-delete file_path bounds (lower=%q, upper=%q); a file-scoped position delete must reference exactly one data file",
			ref, lower, upper)
	}

	return nil
}

func validateContentSchema(args DataFileArgs) error {
	switch args.Content {
	case iceberg.EntryContentData:
		// Parquet field ids are validated implicitly by DataFileStatsFromMeta
		// via format.PathToIDMapping(args.Schema), which rejects columns absent
		// from the schema's field-id mapping and reports
		// them as an error through the scoped recover.
	case iceberg.EntryContentEqDeletes:
		if _, err := validateEqualityFieldIDs(args.Schema, args.EqualityFieldIDs); err != nil {
			return err
		}
	case iceberg.EntryContentPosDeletes:
		for _, want := range iceberg.PositionalDeleteSchema.Fields() {
			got, ok := args.Schema.FindFieldByID(want.ID)
			if !ok {
				return fmt.Errorf(
					"positional delete schema requires reserved field id %d (%s); schema is missing it", want.ID, want.Name)
			}
			if !got.Type.Equals(want.Type) {
				return fmt.Errorf(
					"positional delete schema field id %d (%s) must have type %s, got %s", want.ID, want.Name, want.Type, got.Type)
			}
		}
	}

	return nil
}

func validatePartitionValues(spec iceberg.PartitionSpec, values map[int]any) error {
	if spec.Equals(*iceberg.UnpartitionedSpec) {
		if len(values) > 0 {
			return fmt.Errorf("partition values supplied for an unpartitioned spec (id %d); expected none", spec.ID())
		}

		return nil
	}

	expected := make(map[int]string, spec.NumFields())
	for _, field := range spec.Fields() {
		expected[field.FieldID] = field.Name
	}

	for id := range values {
		if _, ok := expected[id]; !ok {
			return fmt.Errorf("partition value supplied for field id %d which is not part of the spec id %d",
				id, spec.ID())
		}
	}

	return nil
}
