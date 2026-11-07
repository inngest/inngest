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
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/config"
	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

var ErrEmptyEqualityFieldIDs = errors.New("equality field IDs must not be empty")

// equalityDeleteSchema projects a table schema to only the fields specified
// by the given field IDs, using Schema.Select for proper nested field handling.
func equalityDeleteSchema(tableSchema *iceberg.Schema, fieldIDs []int) (*iceberg.Schema, error) {
	names, err := validateEqualityFieldIDs(tableSchema, fieldIDs)
	if err != nil {
		return nil, err
	}

	return tableSchema.Select(true, names...)
}

func validateEqualityFieldIDs(tableSchema *iceberg.Schema, fieldIDs []int) ([]string, error) {
	if len(fieldIDs) == 0 {
		return nil, ErrEmptyEqualityFieldIDs
	}

	names := make([]string, 0, len(fieldIDs))
	for _, id := range fieldIDs {
		name, ok := tableSchema.FindColumnName(id)
		if !ok {
			return nil, fmt.Errorf("%w: field ID %d not found in table schema", iceberg.ErrInvalidSchema, id)
		}

		// Keep both lookups: FindColumnName returns the dotted path Schema.Select
		// needs for nested fields, while FindFieldByID exposes the field type.
		field, ok := tableSchema.FindFieldByID(id)
		if !ok {
			// Defensive only: FindColumnName already found this field ID.
			return nil, fmt.Errorf("%w: field ID %d not found in table schema", iceberg.ErrInvalidSchema, id)
		}
		if isFloatingPointType(field.Type) {
			return nil, fmt.Errorf("%w: equality field ID %d (%s) has unsupported floating-point type %s: floating-point columns cannot be used as equality delete keys",
				iceberg.ErrInvalidSchema, id, name, field.Type)
		}

		names = append(names, name)
	}

	return names, nil
}

func isFloatingPointType(typ iceberg.Type) bool {
	switch typ.(type) {
	case iceberg.Float32Type, iceberg.Float64Type:
		return true
	default:
		return false
	}
}

// WriteEqualityDeletes writes Arrow record batches as equality delete
// Parquet files and returns the resulting DataFiles. The returned files
// have ContentType == EntryContentEqDeletes and EqualityFieldIDs set,
// ready to be passed to [RowDelta.AddDeletes].
//
// The equalityFieldIDs identify which columns in the table schema form
// the delete key. The provided records must contain exactly those columns.
//
// The table must use format version 2 or higher.
//
// For partitioned tables, the provided records must include the partition
// source columns in addition to the equality key columns so that records
// can be routed to the correct partition directories. If the partition
// source columns overlap with the equality key columns, no extra columns
// are needed.
//
// Usage:
//
//	deleteFiles, err := tx.WriteEqualityDeletes(ctx, []int{1, 2}, records)
//	rd := tx.NewRowDelta(nil)
//	rd.AddDeletes(deleteFiles...)
//	err = rd.Commit(ctx)
func (t *Transaction) WriteEqualityDeletes(ctx context.Context, equalityFieldIDs []int, records iter.Seq2[arrow.RecordBatch, error]) ([]iceberg.DataFile, error) {
	meta, err := t.txnMeta()
	if err != nil {
		return nil, err
	}

	if meta.formatVersion < 2 {
		return nil, fmt.Errorf("equality deletes require table format version >= 2, got v%d",
			meta.formatVersion)
	}

	deleteSchema, err := equalityDeleteSchema(meta.CurrentSchema(), equalityFieldIDs)
	if err != nil {
		return nil, err
	}

	fs, err := t.tbl.fsF(ctx)
	if err != nil {
		return nil, err
	}

	wfs, err := requireWriteFileIO(fs)
	if err != nil {
		return nil, err
	}

	arrowSc, err := SchemaToArrowSchemaWithOptions(deleteSchema, ArrowSchemaOptions{
		IncludeFieldIDs: true,
		TableProperties: meta.props,
	})
	if err != nil {
		return nil, err
	}

	writeUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate write UUID: %w", err)
	}

	args := recordWritingArgs{
		sc:        arrowSc,
		itr:       records,
		fs:        wfs,
		writeUUID: &writeUUID,
		counter:   internal.Counter(0),
	}

	dataFiles, err := equalityDeleteRecordsToDataFiles(ctx, t.tbl.Location(), meta, deleteSchema, equalityFieldIDs, args)
	if err != nil {
		return nil, err
	}

	var result []iceberg.DataFile
	for df, err := range dataFiles {
		if err != nil {
			return nil, err
		}

		result = append(result, df)
	}

	return result, nil
}

// CommitEqualityDeletes is a convenience wrapper that writes the given
// records as equality delete files and stages a delete-only row delta in
// this transaction. It composes [Transaction.WriteEqualityDeletes] with
// [Transaction.NewRowDelta] + [RowDelta.AddDeletes] + [RowDelta.Commit],
// so it is the delete-only shortcut for CDC/upsert workloads whose caller
// already holds the equality-key values.
//
// Like the other row-level operations, the change is staged within the
// transaction; call [Transaction.Commit] to make it durable in the
// catalog. Conflict detection runs at that point.
//
// It adds no behavior of its own beyond that composition. In particular:
//   - the format-version >= 2 requirement is enforced by WriteEqualityDeletes;
//   - equality field IDs are validated by WriteEqualityDeletes and again by
//     RowDelta.Commit;
//   - concurrent-writer conflict detection is inherited from RowDelta, which
//     validates against write.delete.isolation-level (serializable by
//     default) when the transaction is committed.
//
// snapshotProps are recorded in the resulting snapshot summary. For update
// CDC events that must commit deletes and appends in one snapshot, use the
// lower-level RowDelta flow (AddRows + AddDeletes) directly.
//
// When records yields no rows, no delete files are produced and this is a
// no-op that returns nil rather than staging an empty snapshot.
//
// Usage:
//
//	if err := tx.CommitEqualityDeletes(ctx, []int{1, 2}, records, nil); err != nil {
//		return err
//	}
//	_, err := tx.Commit(ctx)
func (t *Transaction) CommitEqualityDeletes(ctx context.Context, equalityFieldIDs []int, records iter.Seq2[arrow.RecordBatch, error], snapshotProps iceberg.Properties) error {
	deleteFiles, err := t.WriteEqualityDeletes(ctx, equalityFieldIDs, records)
	if err != nil {
		return err
	}

	if len(deleteFiles) == 0 {
		return nil
	}

	return t.NewRowDelta(snapshotProps).AddDeletes(deleteFiles...).Commit(ctx)
}

func equalityDeleteRecordsToDataFiles(ctx context.Context, rootLocation string, meta *MetadataBuilder, deleteSchema *iceberg.Schema, equalityFieldIDs []int, args recordWritingArgs) (ret iter.Seq2[iceberg.DataFile, error], retErr error) {
	if args.counter == nil {
		args.counter = internal.Counter(0)
	}

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				retErr = fmt.Errorf("error encountered during equality delete file writing: %w", e)
			default:
				retErr = fmt.Errorf("error encountered during equality delete file writing: %v", e)
			}
		}
	}()

	if args.writeUUID == nil {
		u, err := uuid.NewRandom()
		if err != nil {
			return nil, fmt.Errorf("failed to generate write UUID: %w", err)
		}
		args.writeUUID = &u
	}

	targetFileSize := int64(meta.props.GetInt(WriteTargetFileSizeBytesKey,
		WriteTargetFileSizeBytesDefault))

	latestMetadata, err := meta.Build()
	if err != nil {
		return nil, err
	}

	if !latestMetadata.PartitionSpec().IsUnpartitioned() {
		tableSchema := meta.CurrentSchema()
		partitionSpec := latestMetadata.PartitionSpec()

		writeSchema, err := equalityDeleteWriteSchema(tableSchema, equalityFieldIDs, partitionSpec)
		if err != nil {
			return nil, err
		}

		factory, err := newWriterFactory(rootLocation, args, meta, writeSchema, targetFileSize,
			withContentType(iceberg.EntryContentEqDeletes),
			withFactoryFileSchema(deleteSchema),
			withFactoryEqualityFieldIDs(equalityFieldIDs))
		if err != nil {
			return nil, err
		}

		partitionWriter := newPartitionedFanoutWriter(
			partitionSpec, writeSchema, args.itr, factory)
		workers := config.EnvConfig.MaxWorkers

		return partitionWriter.Write(ctx, workers), nil
	}

	cw := newConcurrentDataFileWriter(newEqualityDeleteWriterMaker(deleteSchema, equalityFieldIDs),
		withSchemaSanitization(false))

	nextCount, stopCount := iter.Pull(args.counter)
	tasks := func(yield func(WriteTask) bool) {
		defer stopCount()

		fileCount := 0
		for batch := range binPackRecords(args.itr, defaultBinPackLookback, targetFileSize) {
			cnt, _ := nextCount()
			fileCount++
			t := WriteTask{
				Uuid:        *args.writeUUID,
				ID:          cnt,
				PartitionID: iceberg.UnpartitionedSpec.ID(),
				FileCount:   fileCount,
				Schema:      deleteSchema,
				Batches:     batch,
			}
			if !yield(t) {
				return
			}
		}
	}

	return cw.writeFiles(ctx, rootLocation, args.fs, meta, meta.props, nil, tasks), nil
}

// equalityDeleteWriteSchema returns a schema containing the union of equality
// key columns and partition source columns. This allows the partitioned fanout
// writer to extract partition values from the records while keeping the delete
// key as the equality identifier.
func equalityDeleteWriteSchema(tableSchema *iceberg.Schema, equalityFieldIDs []int, spec iceberg.PartitionSpec) (*iceberg.Schema, error) {
	names, err := validateEqualityFieldIDs(tableSchema, equalityFieldIDs)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]struct{}, len(equalityFieldIDs))

	// Equality key columns first (deterministic order).
	for _, id := range equalityFieldIDs {
		seen[id] = struct{}{}
	}

	// Partition source columns not already in the equality key.
	for _, f := range spec.Fields() {
		if _, ok := seen[f.SourceID()]; ok {
			continue
		}
		name, ok := tableSchema.FindColumnName(f.SourceID())
		if !ok {
			return nil, fmt.Errorf("%w: partition source field ID %d not found in table schema", iceberg.ErrInvalidSchema, f.SourceID())
		}
		seen[f.SourceID()] = struct{}{}
		names = append(names, name)
	}

	return tableSchema.Select(true, names...)
}
