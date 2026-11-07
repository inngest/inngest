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
	"cmp"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/iceberg-go"
)

// InspectTable exposes a table's metadata (snapshots, history, manifests, and
// so on) as queryable Arrow tables. It mirrors the metadata tables offered by
// the Java, PyIceberg, and Rust clients.
//
// Obtain one via Table.Inspect. Each method returns an array.RecordReader that
// the caller is responsible for releasing.
type InspectTable struct {
	tbl   Table
	alloc memory.Allocator
}

// Inspect returns an InspectTable for reading this table's metadata tables,
// allocating Arrow buffers from opts, if provided, otherwise the default
// allocator.
func (t Table) Inspect(opts ...InspectOption) InspectTable {
	i := InspectTable{tbl: t, alloc: memory.DefaultAllocator}
	for _, opt := range opts {
		opt(&i)
	}

	return i
}

// InspectOption configures an InspectTable.
type InspectOption func(*InspectTable)

// WithInspectAllocator sets the Arrow memory allocator used to build
// metadata-table records. Tests can pass a memory.CheckedAllocator to detect
// leaks; callers with memory accounting can inject their own pool.
func WithInspectAllocator(alloc memory.Allocator) InspectOption {
	return func(i *InspectTable) {
		if alloc != nil {
			i.alloc = alloc
		}
	}
}

// History returns the chronological log of every snapshot that was ever the
// table's current snapshot, one row per snapshot-log entry. Rolled-back
// snapshots remain visible but are flagged via is_current_ancestor.
//
// Columns:
//   - made_current_at (timestamptz, required): when the snapshot became current
//   - snapshot_id (long, required): the snapshot that became current
//   - parent_id (long, optional): the snapshot's parent, null when the snapshot
//     has no parent or has since been expired
//   - is_current_ancestor (boolean, required): whether the snapshot is an
//     ancestor of the current snapshot; false for rolled-back snapshots
//
// is_current_ancestor is derived by walking the current snapshot's parent
// chain. If an intermediate ancestor has been expired (removed from the
// snapshot list), that walk is truncated and snapshots below the gap are
// reported as non-ancestors. Well-formed tables never hit this: ExpireSnapshots
// keeps the current snapshot and its full ancestry intact. This matches the
// silent-truncation behavior of the Java, PyIceberg, and Rust clients.
//
// The returned reader holds a single record batch. The caller must Release it.
func (i InspectTable) History(ctx context.Context) (array.RecordReader, error) {
	arrowSchema, err := SchemaToArrowSchema(HistorySchema(), nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect history: build arrow schema: %w", err)
	}

	// Walk the current snapshot's parent chain once. Membership in this set
	// is what tells the live lineage apart from rolled-back history entries.
	// AncestorsOf already guards against cycles in malformed metadata, so a
	// corrupt parent chain cannot hang the scan.
	ancestors := make(map[int64]struct{})
	if current := i.tbl.metadata.CurrentSnapshot(); current != nil {
		for _, snap := range AncestorsOf(current.SnapshotID, i.tbl.metadata.SnapshotByID) {
			ancestors[snap.SnapshotID] = struct{}{}
		}
	}

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()

	// Field positions and concrete builder types follow HistorySchema; the
	// assertions are safe as long as the two stay in sync.
	madeCurrentAt := bldr.Field(0).(*array.TimestampBuilder)
	snapshotID := bldr.Field(1).(*array.Int64Builder)
	parentID := bldr.Field(2).(*array.Int64Builder)
	isCurrentAncestor := bldr.Field(3).(*array.BooleanBuilder)

	for entry := range i.tbl.metadata.SnapshotLogs() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Iceberg stores timestamps as epoch milliseconds; Arrow uses
		// microseconds for the timestamptz type.
		madeCurrentAt.Append(arrow.Timestamp(entry.TimestampMs * 1000))
		snapshotID.Append(entry.SnapshotID)

		// parent_id resolves through the live snapshot table, so an entry
		// referencing an expired snapshot renders a null parent.
		if snap := i.tbl.metadata.SnapshotByID(entry.SnapshotID); snap != nil && snap.ParentSnapshotID != nil {
			parentID.Append(*snap.ParentSnapshotID)
		} else {
			parentID.AppendNull()
		}

		_, ok := ancestors[entry.SnapshotID]
		isCurrentAncestor.Append(ok)
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect history: %w", err)
	}

	return rr, nil
}

// singleBatchReader finishes bldr into one record batch and wraps it in a
// RecordReader. NewRecordReader retains the batch, so we release our own
// reference unconditionally: on success the reader owns it, and on error
// NewRecordReader has already released its own retain. This avoids a deferred
// double-release.
func singleBatchReader(arrowSchema *arrow.Schema, bldr *array.RecordBuilder) (array.RecordReader, error) {
	rec := bldr.NewRecordBatch()
	rr, err := array.NewRecordReader(arrowSchema, []arrow.RecordBatch{rec})
	rec.Release()
	if err != nil {
		return nil, fmt.Errorf("new record reader: %w", err)
	}

	return rr, nil
}

// Snapshots returns one row per snapshot known to the table, in the order they
// are stored in metadata.
//
// Columns:
//   - committed_at (timestamptz, required): when the snapshot was committed
//   - snapshot_id (long, required): the snapshot id
//   - parent_id (long, optional): the parent snapshot id, null for a root
//   - operation (string, optional): the snapshot summary operation, null when
//     the snapshot carries no summary
//   - manifest_list (string, optional): path to the snapshot's manifest list
//   - summary (map<string,string>, optional): the stored snapshot summary,
//     including the "operation" key alongside its additional properties; null
//     when the snapshot carries no summary
//
// The returned reader holds a single record batch. The caller must Release it.
func (i InspectTable) Snapshots(ctx context.Context) (array.RecordReader, error) {
	arrowSchema, err := SchemaToArrowSchema(SnapshotsSchema(), nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect snapshots: build arrow schema: %w", err)
	}

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()

	// Field positions and concrete builder types follow SnapshotsSchema; the
	// assertions are safe as long as the two stay in sync.
	committedAt := bldr.Field(0).(*array.TimestampBuilder)
	snapshotID := bldr.Field(1).(*array.Int64Builder)
	parentID := bldr.Field(2).(*array.Int64Builder)
	operation := bldr.Field(3).(*array.StringBuilder)
	manifestList := bldr.Field(4).(*array.StringBuilder)
	summary := bldr.Field(5).(*array.MapBuilder)
	summaryKeys := summary.KeyBuilder().(*array.StringBuilder)
	summaryValues := summary.ItemBuilder().(*array.StringBuilder)

	for _, snap := range i.tbl.metadata.Snapshots() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Iceberg stores timestamps as epoch milliseconds; Arrow uses
		// microseconds for the timestamptz type.
		committedAt.Append(arrow.Timestamp(snap.TimestampMs * 1000))
		snapshotID.Append(snap.SnapshotID)

		if snap.ParentSnapshotID != nil {
			parentID.Append(*snap.ParentSnapshotID)
		} else {
			parentID.AppendNull()
		}

		// operation and summary come from the snapshot summary, which is
		// optional: a snapshot without one renders both as null.
		if snap.Summary != nil {
			operation.Append(string(snap.Summary.Operation))
			summary.Append(true)
			// The summary map mirrors the stored/serialized summary, which
			// folds the operation back in under the "operation" key (see
			// Summary.MarshalJSON). Emit it alongside the extra properties so
			// the table faithfully reflects the on-disk summary.
			if snap.Summary.Operation != "" {
				summaryKeys.Append(operationKey)
				summaryValues.Append(string(snap.Summary.Operation))
			}
			for k, v := range snap.Summary.Properties {
				summaryKeys.Append(k)
				summaryValues.Append(v)
			}
		} else {
			operation.AppendNull()
			summary.AppendNull()
		}

		// manifest_list is optional: a snapshot without a manifest-list path
		// (e.g. some V1 snapshots) renders null rather than an empty string.
		if snap.ManifestList != "" {
			manifestList.Append(snap.ManifestList)
		} else {
			manifestList.AppendNull()
		}
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect snapshots: %w", err)
	}

	return rr, nil
}

// Manifests returns one row for each manifest in the current snapshot.
// Partition summaries are exposed as the human-readable values used by the
// other Iceberg clients.
func (i InspectTable) Manifests(ctx context.Context) (array.RecordReader, error) {
	schema := ManifestsSchema()
	arrowSchema, err := SchemaToArrowSchema(schema, nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect manifests: build arrow schema: %w", err)
	}

	manifests, err := i.currentSnapshotManifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("inspect manifests: %w", err)
	}

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()

	if err := i.appendManifestRows(ctx, bldr, manifests, nil); err != nil {
		return nil, fmt.Errorf("inspect manifests: %w", err)
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect manifests: %w", err)
	}

	return rr, nil
}

func (i InspectTable) appendManifestRows(
	ctx context.Context,
	bldr *array.RecordBuilder,
	manifests []iceberg.ManifestFile,
	referenceSnapshotID *int64,
) error {
	content := bldr.Field(0).(*array.Int32Builder)
	path := bldr.Field(1).(*array.StringBuilder)
	length := bldr.Field(2).(*array.Int64Builder)
	partitionSpecID := bldr.Field(3).(*array.Int32Builder)
	addedSnapshotID := bldr.Field(4).(*array.Int64Builder)
	addedDataFiles := bldr.Field(5).(*array.Int32Builder)
	existingDataFiles := bldr.Field(6).(*array.Int32Builder)
	deletedDataFiles := bldr.Field(7).(*array.Int32Builder)
	addedDeleteFiles := bldr.Field(8).(*array.Int32Builder)
	existingDeleteFiles := bldr.Field(9).(*array.Int32Builder)
	deletedDeleteFiles := bldr.Field(10).(*array.Int32Builder)
	partitionSummaries := bldr.Field(11).(*array.ListBuilder)
	summaryStruct := partitionSummaries.ValueBuilder().(*array.StructBuilder)
	summaryContainsNull := summaryStruct.FieldBuilder(0).(*array.BooleanBuilder)
	summaryContainsNaN := summaryStruct.FieldBuilder(1).(*array.BooleanBuilder)
	summaryLower := summaryStruct.FieldBuilder(2).(*array.StringBuilder)
	summaryUpper := summaryStruct.FieldBuilder(3).(*array.StringBuilder)
	var referenceSnapshot *array.Int64Builder
	var keyMetadata *array.BinaryBuilder
	if referenceSnapshotID != nil {
		referenceSnapshot = bldr.Field(12).(*array.Int64Builder)
		keyMetadata = bldr.Field(13).(*array.BinaryBuilder)
	}

	for _, manifest := range manifests {
		if err := ctx.Err(); err != nil {
			return err
		}

		manifestContent := manifest.ManifestContent()
		switch manifestContent {
		case iceberg.ManifestContentData, iceberg.ManifestContentDeletes:
		default:
			return fmt.Errorf("manifest %s has unknown content %d", manifest.FilePath(), manifestContent)
		}
		snapshotID := manifest.SnapshotID()
		if snapshotID < 0 {
			return fmt.Errorf("manifest %s has negative added_snapshot_id %d", manifest.FilePath(), snapshotID)
		}

		content.Append(int32(manifestContent))
		path.Append(manifest.FilePath())
		length.Append(manifest.Length())
		partitionSpecID.Append(manifest.PartitionSpecID())
		addedSnapshotID.Append(snapshotID)
		appendCount := func(builder *array.Int32Builder, name string, count int32) error {
			if err := appendManifestCount(builder, manifest.Version(), name, count); err != nil {
				return fmt.Errorf("manifest %s: %w", manifest.FilePath(), err)
			}

			return nil
		}

		switch manifestContent {
		case iceberg.ManifestContentData:
			if err := appendCount(addedDataFiles, "added_data_files", manifest.AddedDataFiles()); err != nil {
				return err
			}
			if err := appendCount(existingDataFiles, "existing_data_files", manifest.ExistingDataFiles()); err != nil {
				return err
			}
			if err := appendCount(deletedDataFiles, "deleted_data_files", manifest.DeletedDataFiles()); err != nil {
				return err
			}
			addedDeleteFiles.Append(0)
			existingDeleteFiles.Append(0)
			deletedDeleteFiles.Append(0)
		case iceberg.ManifestContentDeletes:
			addedDataFiles.Append(0)
			existingDataFiles.Append(0)
			deletedDataFiles.Append(0)
			// ManifestFile's data-file accessors expose the generic manifest-list
			// file counts, which represent delete files for delete manifests.
			if err := appendCount(addedDeleteFiles, "added_delete_files", manifest.AddedDataFiles()); err != nil {
				return err
			}
			if err := appendCount(existingDeleteFiles, "existing_delete_files", manifest.ExistingDataFiles()); err != nil {
				return err
			}
			if err := appendCount(deletedDeleteFiles, "deleted_delete_files", manifest.DeletedDataFiles()); err != nil {
				return err
			}
		}
		if referenceSnapshotID != nil {
			referenceSnapshot.Append(*referenceSnapshotID)
			if metadata := manifest.KeyMetadata(); metadata != nil {
				keyMetadata.Append(metadata)
			} else {
				keyMetadata.AppendNull()
			}
		}

		partitions := manifest.Partitions()
		if partitions == nil {
			partitionSummaries.Append(true)

			continue
		}

		spec := i.tbl.metadata.PartitionSpecByID(int(manifest.PartitionSpecID()))
		if spec == nil {
			return fmt.Errorf("manifest %s references missing partition spec %d",
				manifest.FilePath(), manifest.PartitionSpecID())
		}
		partType := spec.PartitionType(i.tbl.metadata.CurrentSchema())
		if len(partitions) > spec.NumFields() {
			return fmt.Errorf("manifest %s has %d partition summaries for partition spec %d with %d fields",
				manifest.FilePath(), len(partitions), manifest.PartitionSpecID(), spec.NumFields())
		}

		partitionSummaries.Append(true)
		for idx, summary := range partitions {
			summaryStruct.Append(true)
			summaryContainsNull.Append(summary.ContainsNull)
			if summary.ContainsNaN == nil {
				summaryContainsNaN.AppendNull()
			} else {
				summaryContainsNaN.Append(*summary.ContainsNaN)
			}

			fieldType := partType.FieldList[idx].Type
			transform := spec.Field(idx).Transform
			if err := appendManifestBound(summaryLower, fieldType, transform, summary.LowerBound); err != nil {
				return fmt.Errorf("manifest %s partition field %q lower bound: %w",
					manifest.FilePath(), partType.FieldList[idx].Name, err)
			}
			if err := appendManifestBound(summaryUpper, fieldType, transform, summary.UpperBound); err != nil {
				return fmt.Errorf("manifest %s partition field %q upper bound: %w",
					manifest.FilePath(), partType.FieldList[idx].Name, err)
			}
		}
	}

	return nil
}

func appendManifestCount(builder *array.Int32Builder, version int, name string, count int32) error {
	if count < 0 {
		if version == 1 && count == -1 {
			// V1 counts are optional, and an absent count means unknown rather
			// than zero. The V1 decoder represents that absent value as -1.
			builder.AppendNull()

			return nil
		}

		return fmt.Errorf("negative %s count %d in manifest list version %d", name, count, version)
	}

	builder.Append(count)

	return nil
}

func (i InspectTable) currentSnapshotManifests(ctx context.Context) ([]iceberg.ManifestFile, error) {
	snapshot := i.tbl.metadata.CurrentSnapshot()
	if snapshot == nil {
		return nil, nil
	}
	if i.tbl.fsF == nil {
		return nil, errors.New("table file IO is not configured")
	}

	manifestSet, err := i.tbl.manifestSet(ctx, *snapshot)
	if err != nil {
		return nil, err
	}

	return manifestSet.allManifests(), nil
}

func appendManifestBound(builder *array.StringBuilder, typ iceberg.Type, transform iceberg.Transform, bound *[]byte) error {
	if bound == nil {
		builder.AppendNull()

		return nil
	}

	literal, err := manifestBoundLiteral(typ, *bound)
	if err != nil {
		return err
	}
	if literal == nil {
		builder.AppendNull()

		return nil
	}

	builder.Append(transform.ToHumanStrType(typ, literal.Any()))

	return nil
}

// Refs returns one row per snapshot reference known to the table. Reference
// names are sorted to make the result deterministic even though table metadata
// stores refs in a map.
//
// Columns:
//   - name (string, required): the branch or tag name
//   - type (string, required): BRANCH or TAG
//   - snapshot_id (long, required): the referenced snapshot
//   - max_reference_age_in_ms (long, optional): tag/branch reference retention
//   - min_snapshots_to_keep (int, optional): branch snapshot retention
//   - max_snapshot_age_in_ms (long, optional): branch snapshot retention
//
// The returned reader holds a single record batch. The caller must Release it.
func (i InspectTable) Refs(ctx context.Context) (array.RecordReader, error) {
	arrowSchema, err := SchemaToArrowSchema(RefsSchema(), nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect refs: build arrow schema: %w", err)
	}

	type refRow struct {
		name string
		ref  SnapshotRef
	}
	var refs []refRow
	for name, ref := range i.tbl.metadata.Refs() {
		refs = append(refs, refRow{name: name, ref: ref})
	}
	slices.SortFunc(refs, func(a, b refRow) int {
		return cmp.Compare(a.name, b.name)
	})

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()

	name := bldr.Field(0).(*array.StringBuilder)
	refType := bldr.Field(1).(*array.StringBuilder)
	snapshotID := bldr.Field(2).(*array.Int64Builder)
	maxReferenceAge := bldr.Field(3).(*array.Int64Builder)
	minSnapshotsToKeep := bldr.Field(4).(*array.Int32Builder)
	maxSnapshotAge := bldr.Field(5).(*array.Int64Builder)

	for _, row := range refs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		name.Append(row.name)
		refType.Append(strings.ToUpper(string(row.ref.SnapshotRefType)))
		snapshotID.Append(row.ref.SnapshotID)

		if row.ref.MaxRefAgeMs != nil {
			maxReferenceAge.Append(*row.ref.MaxRefAgeMs)
		} else {
			maxReferenceAge.AppendNull()
		}
		if row.ref.MinSnapshotsToKeep != nil {
			value := *row.ref.MinSnapshotsToKeep
			if value < math.MinInt32 || value > math.MaxInt32 {
				return nil, fmt.Errorf(
					"inspect refs: min snapshots to keep %d is outside int32 range",
					value,
				)
			}
			minSnapshotsToKeep.Append(int32(value))
		} else {
			minSnapshotsToKeep.AppendNull()
		}
		if row.ref.MaxSnapshotAgeMs != nil {
			maxSnapshotAge.Append(*row.ref.MaxSnapshotAgeMs)
		} else {
			maxSnapshotAge.AppendNull()
		}
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect refs: %w", err)
	}

	return rr, nil
}

// MetadataLogEntries returns one row for every metadata file in the table's
// metadata log, plus the current metadata file when its location is set.
// Snapshot information is resolved from the snapshot log at each metadata
// file's timestamp.
//
// Columns:
//   - timestamp (timestamptz, required): when the metadata file was written
//   - file (string, required): metadata file location
//   - latest_snapshot_id (long, optional): latest snapshot visible then
//   - latest_schema_id (int, optional): schema used by that snapshot
//   - latest_sequence_number (long, optional): sequence number of that snapshot
//
// The returned reader holds a single record batch. The caller must Release it.
func (i InspectTable) MetadataLogEntries(ctx context.Context) (array.RecordReader, error) {
	arrowSchema, err := SchemaToArrowSchema(MetadataLogEntriesSchema(), nil, true, false)
	if err != nil {
		return nil, fmt.Errorf("inspect metadata log entries: build arrow schema: %w", err)
	}

	entries := slices.Collect(i.tbl.metadata.PreviousFiles())
	if i.tbl.metadataLocation != "" {
		entries = append(entries, MetadataLogEntry{
			MetadataFile: i.tbl.metadataLocation,
			TimestampMs:  i.tbl.metadata.LastUpdatedMillis(),
		})
	}

	bldr := array.NewRecordBuilder(i.alloc, arrowSchema)
	defer bldr.Release()

	timestamp := bldr.Field(0).(*array.TimestampBuilder)
	file := bldr.Field(1).(*array.StringBuilder)
	latestSnapshotID := bldr.Field(2).(*array.Int64Builder)
	latestSchemaID := bldr.Field(3).(*array.Int32Builder)
	latestSequenceNumber := bldr.Field(4).(*array.Int64Builder)

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		timestamp.Append(arrow.Timestamp(entry.TimestampMs * 1000))
		file.Append(entry.MetadataFile)

		snapshotID, snapshot, found := latestSnapshotAt(i.tbl.metadata, entry.TimestampMs)
		if !found {
			latestSnapshotID.AppendNull()
			latestSchemaID.AppendNull()
			latestSequenceNumber.AppendNull()

			continue
		}

		latestSnapshotID.Append(snapshotID)
		if snapshot == nil {
			latestSchemaID.AppendNull()
			latestSequenceNumber.AppendNull()

			continue
		}
		if snapshot.SchemaID != nil {
			// Iceberg schema IDs are bounded to int32 by the specification.
			//nolint:gosec // schema IDs are spec-bounded to int32
			latestSchemaID.Append(int32(*snapshot.SchemaID))
		} else {
			latestSchemaID.AppendNull()
		}
		latestSequenceNumber.Append(snapshot.SequenceNumber)
	}

	rr, err := singleBatchReader(arrowSchema, bldr)
	if err != nil {
		return nil, fmt.Errorf("inspect metadata log entries: %w", err)
	}

	return rr, nil
}

// latestSnapshotAt follows the metadata-table behavior used by Java's
// MetadataLogEntriesTable: it resolves against the current snapshot log, not
// a point-in-time copy of that log. Trimming old entries can therefore shift
// the result to a later eligible entry or make it unavailable. Equal
// timestamps keep the first entry encountered, matching Java's
// SnapshotUtil.snapshotIdAsOfTime; PyIceberg keeps the last entry instead.
// The snapshot ID remains available when the snapshot itself has expired from
// metadata.
func latestSnapshotAt(metadata Metadata, timestampMs int64) (int64, *Snapshot, bool) {
	entry, found := snapshotLogEntryAsOf(metadata.SnapshotLogs(), timestampMs, true)
	if !found {
		return 0, nil, false
	}

	return entry.SnapshotID, metadata.SnapshotByID(entry.SnapshotID), true
}

func manifestBoundLiteral(typ iceberg.Type, bound []byte) (iceberg.Literal, error) {
	switch t := typ.(type) {
	case iceberg.UnknownType:
		return nil, nil
	case iceberg.Int64Type:
		if len(bound) == 4 {
			literal, err := iceberg.LiteralFromBytes(iceberg.PrimitiveTypes.Int32, bound)
			if err != nil {
				return nil, err
			}

			return literal.To(t)
		}
	case iceberg.Float64Type:
		if len(bound) == 4 {
			literal, err := iceberg.LiteralFromBytes(iceberg.PrimitiveTypes.Float32, bound)
			if err != nil {
				return nil, err
			}

			return literal.To(t)
		}
	}

	return iceberg.LiteralFromBytes(typ, bound)
}

// HistorySchema returns the Iceberg schema of the history metadata table. The
// field IDs are fixed by the Iceberg metadata-tables spec and match the Java,
// PyIceberg, and Rust clients for cross-client parity. A fresh schema value is
// returned on each call; callers must not mutate it or rely on pointer identity.
func HistorySchema() *iceberg.Schema {
	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "made_current_at", Type: iceberg.PrimitiveTypes.TimestampTz, Required: true},
		iceberg.NestedField{ID: 2, Name: "snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 3, Name: "parent_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 4, Name: "is_current_ancestor", Type: iceberg.PrimitiveTypes.Bool, Required: true},
	)
}

// SnapshotsSchema returns the Iceberg schema of the snapshots metadata table.
// The field IDs are fixed by the Iceberg metadata-tables spec and match the
// Java, PyIceberg, and Rust clients for cross-client parity. A fresh schema
// value is returned on each call; callers must not mutate it or rely on pointer
// identity.
//
// The summary map's values are optional, matching PyIceberg and Rust. Java
// declares them required; the choice is not observable in practice because
// snapshot-summary values are always non-null strings.
func SnapshotsSchema() *iceberg.Schema {
	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "committed_at", Type: iceberg.PrimitiveTypes.TimestampTz, Required: true},
		iceberg.NestedField{ID: 2, Name: "snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 3, Name: "parent_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 4, Name: "operation", Type: iceberg.PrimitiveTypes.String, Required: false},
		iceberg.NestedField{ID: 5, Name: "manifest_list", Type: iceberg.PrimitiveTypes.String, Required: false},
		iceberg.NestedField{ID: 6, Name: "summary", Required: false, Type: &iceberg.MapType{
			KeyID:         7,
			KeyType:       iceberg.PrimitiveTypes.String,
			ValueID:       8,
			ValueType:     iceberg.PrimitiveTypes.String,
			ValueRequired: false,
		}},
	)
}

// RefsSchema returns a fresh Iceberg schema for the refs metadata table. The
// field IDs and names match Java's RefsTable for cross-client parity; callers
// should not rely on pointer identity.
func RefsSchema() *iceberg.Schema {
	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "name", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 2, Name: "type", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 3, Name: "snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 4, Name: "max_reference_age_in_ms", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 5, Name: "min_snapshots_to_keep", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 6, Name: "max_snapshot_age_in_ms", Type: iceberg.PrimitiveTypes.Int64, Required: false},
	)
}

// MetadataLogEntriesSchema returns a fresh Iceberg schema for the
// metadata-log-entries metadata table. The field IDs and names match Java's
// implementation; callers should not rely on pointer identity.
func MetadataLogEntriesSchema() *iceberg.Schema {
	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 1, Name: "timestamp", Type: iceberg.PrimitiveTypes.TimestampTz, Required: true},
		iceberg.NestedField{ID: 2, Name: "file", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 3, Name: "latest_snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: false},
		iceberg.NestedField{ID: 4, Name: "latest_schema_id", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 5, Name: "latest_sequence_number", Type: iceberg.PrimitiveTypes.Int64, Required: false},
	)
}

// ManifestsSchema returns the Iceberg schema of the manifests metadata table.
func ManifestsSchema() *iceberg.Schema {
	return iceberg.NewSchema(0,
		iceberg.NestedField{ID: 14, Name: "content", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 1, Name: "path", Type: iceberg.PrimitiveTypes.String, Required: true},
		iceberg.NestedField{ID: 2, Name: "length", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 3, Name: "partition_spec_id", Type: iceberg.PrimitiveTypes.Int32, Required: true},
		iceberg.NestedField{ID: 4, Name: "added_snapshot_id", Type: iceberg.PrimitiveTypes.Int64, Required: true},
		iceberg.NestedField{ID: 5, Name: "added_data_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 6, Name: "existing_data_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 7, Name: "deleted_data_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 15, Name: "added_delete_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 16, Name: "existing_delete_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 17, Name: "deleted_delete_files_count", Type: iceberg.PrimitiveTypes.Int32, Required: false},
		iceberg.NestedField{ID: 8, Name: "partition_summaries", Required: true, Type: &iceberg.ListType{
			ElementID:       9,
			ElementRequired: true,
			Element: &iceberg.StructType{FieldList: []iceberg.NestedField{
				{ID: 10, Name: "contains_null", Type: iceberg.PrimitiveTypes.Bool, Required: true},
				{ID: 11, Name: "contains_nan", Type: iceberg.PrimitiveTypes.Bool, Required: false},
				{ID: 12, Name: "lower_bound", Type: iceberg.PrimitiveTypes.String, Required: false},
				{ID: 13, Name: "upper_bound", Type: iceberg.PrimitiveTypes.String, Required: false},
			}},
		}},
	)
}
