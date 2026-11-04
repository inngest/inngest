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
	"strconv"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

// WriteRecordOption configures the behavior of WriteRecords.
type WriteRecordOption func(*writeRecordConfig)

type writeRecordConfig struct {
	targetFileSize        int64
	writeUUID             *uuid.UUID
	maxWriteWorkers       int
	clustered             bool
	fileSchema            *iceberg.Schema
	preserveRowLineage    bool
	recordBatchBufferSize int
	parquetRowGroupLimit  int
}

// WithTargetFileSize overrides the table's default target file size.
func WithTargetFileSize(size int64) WriteRecordOption {
	return func(c *writeRecordConfig) {
		c.targetFileSize = size
	}
}

// WithWriteUUID sets a specific UUID for file naming.
func WithWriteUUID(id uuid.UUID) WriteRecordOption {
	return func(c *writeRecordConfig) {
		c.writeUUID = &id
	}
}

// WithMaxWriteWorkers overrides the default number of fanout workers
// used for partitioned writes. Each worker processes record batches,
// partitions them, and writes to the appropriate partition files.
// Fewer workers means fewer concurrent parquet writers compressing
// pages simultaneously, which reduces peak memory. A value of 0
// (the default) uses [config.EnvConfig.MaxWorkers].
//
// Combining this option with [WithClusteredWrite] is rejected by
// [WriteRecords]: the clustered write path is single-threaded by
// design, so the two options have no meaningful interaction.
func WithMaxWriteWorkers(n int) WriteRecordOption {
	return func(c *writeRecordConfig) {
		c.maxWriteWorkers = n
	}
}

// WithClusteredWrite enables the memory-efficient clustered write
// path for partitioned tables. It keeps at most one partition writer
// open at a time: when a record arrives for a new partition, the
// current writer is flushed and closed before a new one is opened.
//
// The input must be clustered by partition across batches: once a
// partition's writer has been closed, encountering further records
// for that partition returns an error. Within a single batch the
// writer reclusters rows by partition, so interleaved values like
// [a,b,a,b] are accepted; the strict check fires only across batch
// boundaries. This is the natural order for compaction, where each
// source data file typically belongs to a single partition. If the
// input is not clustered across batches, use the fanout writer (the
// default) instead.
//
// Combining this option with [WithMaxWriteWorkers] is rejected by
// [WriteRecords]: the clustered path is single-threaded by design.
func WithClusteredWrite() WriteRecordOption {
	return func(c *writeRecordConfig) {
		c.clustered = true
	}
}

// WithRecordBatchBufferSize sets the capacity, in record batches, of
// each rolling data writer's input channel. Every buffered batch is
// retained in memory until its writer consumes it, so this bound times
// the batch row count caps the memory a stalled writer can hold. The
// default is 64 batches. A non-positive value is ignored.
func WithRecordBatchBufferSize(n int) WriteRecordOption {
	return func(c *writeRecordConfig) {
		if n > 0 {
			c.recordBatchBufferSize = n
		}
	}
}

// WithParquetRowGroupLimit overrides the table's
// write.parquet.row-group-limit property for this write, capping the
// number of rows per Parquet row group in the output files. Smaller row
// groups bound the writer's buffered memory before each flush. A
// non-positive value is ignored.
func WithParquetRowGroupLimit(n int) WriteRecordOption {
	return func(c *writeRecordConfig) {
		if n > 0 {
			c.parquetRowGroupLimit = n
		}
	}
}

// WithPreserveRowLineage sets the output file schema to include the v3 row-
// lineage metadata columns (_row_id, _last_updated_sequence_number) so that
// row identity is preserved through rewrites and compactions. The input
// records must already carry _row_id (e.g. from a scan that projected the
// lineage columns).
//
// [WriteRecords] validates this option against the table state and the input
// Arrow schema: it errors when applied to a v1/v2 table or when the input
// records don't include the _row_id column. The schema parameter is the
// projected Iceberg schema (typically [iceberg.SchemaWithRowLineage]) and is
// used to write the output Parquet's field IDs.
func WithPreserveRowLineage(schema *iceberg.Schema) WriteRecordOption {
	return func(c *writeRecordConfig) {
		c.fileSchema = schema
		c.preserveRowLineage = true
	}
}

// WriteRecords writes Arrow record batches to Parquet data files for the given
// table, returning an iterator of the resulting DataFile objects.
//
// The provided Arrow schema must be compatible with the table's current Iceberg
// schema: each field in the Arrow schema is matched to the table schema by
// field ID (or by name via the table's name mapping if field IDs are absent).
// The Arrow schema may be a subset of the table schema (projection), but every
// field present must have a type that is promotable to the corresponding table
// field type. When the table uses microsecond timestamps, Arrow nanosecond
// timestamps are also accepted and are downcast during the write path.
//
// WriteRecords releases each RecordBatch it consumes. If the caller needs a
// batch to remain valid after it has been yielded, it must call Retain before
// yielding and is then responsible for the corresponding Release.
func WriteRecords(ctx context.Context, tbl *Table,
	schema *arrow.Schema,
	records iter.Seq2[arrow.RecordBatch, error],
	opts ...WriteRecordOption,
) iter.Seq2[iceberg.DataFile, error] {
	cfg := writeRecordConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.clustered && cfg.maxWriteWorkers > 0 {
		return internal.SingleErrorIter[iceberg.DataFile](
			errors.New("WithClusteredWrite and WithMaxWriteWorkers are incompatible: the clustered write path is single-threaded"))
	}

	if cfg.preserveRowLineage {
		if v := tbl.metadata.Version(); v < 3 {
			return internal.SingleErrorIter[iceberg.DataFile](
				fmt.Errorf("WithPreserveRowLineage requires a v3+ table, got v%d", v))
		}
		if len(schema.FieldIndices(iceberg.RowIDColumnName)) == 0 {
			return internal.SingleErrorIter[iceberg.DataFile](
				fmt.Errorf("WithPreserveRowLineage requires input records to include the %q column", iceberg.RowIDColumnName))
		}
	}

	// Validate the input arrow schema against the projected schema. When
	// row lineage is being preserved, the projected schema includes the
	// reserved metadata columns; using tbl.Schema() instead would reject
	// the lineage columns since they're not declared in the user schema.
	checkSchema := tbl.Schema()
	if cfg.fileSchema != nil {
		checkSchema = cfg.fileSchema
	}
	err := checkArrowSchemaCompatWithProperties(checkSchema, schema, false, tbl.Metadata().Properties())
	if err != nil {
		if downcastErr := checkArrowSchemaCompatWithProperties(checkSchema, schema, true, tbl.Metadata().Properties()); downcastErr == nil {
			err = nil
		}
	}
	if err != nil {
		return internal.SingleErrorIter[iceberg.DataFile](
			fmt.Errorf("arrow schema is not compatible with the table schema: %w", err))
	}

	fs, err := tbl.fsF(ctx)
	if err != nil {
		return internal.SingleErrorIter[iceberg.DataFile](err)
	}

	writeFS, err := requireWriteFileIO(fs)
	if err != nil {
		return internal.SingleErrorIter[iceberg.DataFile](err)
	}

	meta, err := MetadataBuilderFromBase(tbl.metadata, tbl.metadataLocation)
	if err != nil {
		return internal.SingleErrorIter[iceberg.DataFile](fmt.Errorf("failed to build metadata: %w", err))
	}

	if cfg.targetFileSize > 0 {
		if meta.props == nil {
			meta.props = make(iceberg.Properties)
		}
		meta.props[WriteTargetFileSizeBytesKey] = strconv.FormatInt(cfg.targetFileSize, 10)
	}

	if cfg.parquetRowGroupLimit > 0 {
		if meta.props == nil {
			meta.props = make(iceberg.Properties)
		}
		meta.props[ParquetRowGroupLimitKey] = strconv.Itoa(cfg.parquetRowGroupLimit)
	}

	releasing := func(yield func(arrow.RecordBatch, error) bool) {
		for rec, err := range records {
			if err != nil {
				yield(nil, err)

				return
			}
			if !yield(rec, nil) {
				rec.Release()

				return
			}
			rec.Release()
		}
	}

	args := recordWritingArgs{
		sc:                    schema,
		itr:                   releasing,
		fs:                    writeFS,
		writeUUID:             cfg.writeUUID,
		maxWriteWorkers:       cfg.maxWriteWorkers,
		clustered:             cfg.clustered,
		recordBatchBufferSize: cfg.recordBatchBufferSize,
	}

	if cfg.fileSchema != nil {
		args.factoryOpts = append(args.factoryOpts, withFactoryFileSchema(cfg.fileSchema))
	}

	return recordsToDataFiles(ctx, tbl.Location(), meta, args)
}
