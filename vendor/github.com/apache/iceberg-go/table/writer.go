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
	"fmt"
	"iter"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/config"
	"github.com/apache/iceberg-go/io"
	tblutils "github.com/apache/iceberg-go/table/internal"
	"github.com/google/uuid"
)

// WriteTask writes all its Batches into one file. The rolling writer bypasses
// it, naming files via dataFileName directly.
type WriteTask struct {
	Uuid        uuid.UUID
	ID          int
	PartitionID int // PartitionID is the partition identifier used in data file naming.
	FileCount   int // FileCount is a sequential counter for files written by this task.
	Schema      *iceberg.Schema
	Batches     []arrow.RecordBatch
	// SortOrderID claims Batches are globally sorted by that order; writeFile
	// records it unverified. Must stay zero for position deletes.
	SortOrderID int
}

func (w WriteTask) GenerateDataFileName(extension string) string {
	return dataFileName(w.Uuid, w.ID, w.PartitionID, w.FileCount, extension)
}

// dataFileName builds a data file name, mirroring the Java API:
// https://github.com/apache/iceberg/blob/03ed4ba9af4e47d32bdb22b7e3d033eb2a4b2c83/core/src/main/java/org/apache/iceberg/io/OutputFileFactory.java#L93
// Format: {partitionId:05d}-{taskId}-{operationId}-{fileCount:05d}.{extension}
func dataFileName(writeUUID uuid.UUID, taskID, partitionID, fileCount int, extension string) string {
	return fmt.Sprintf("%05d-%d-%s-%05d.%s", partitionID, taskID, writeUUID, fileCount, extension)
}

type defaultDataFileWriter struct {
	loc              LocationProvider
	fs               io.WriteFileIO
	fileSchema       *iceberg.Schema
	fileFormat       iceberg.FileFormat
	format           tblutils.FileFormat
	props            iceberg.Properties
	content          iceberg.ManifestEntryContent
	equalityFieldIDs []int
	invariants       dataWriterInvariants
}

// dataWriterInvariants contains the write configuration shared by every task
// handled by a defaultDataFileWriter.
type dataWriterInvariants struct {
	statsCols       map[int]tblutils.StatisticsCollector
	colMapping      map[string]int
	variantFieldIDs map[int]struct{}
	writeProps      any
	rowGroupBytes   int64
	spec            iceberg.PartitionSpec
	extension       string
	arrowSchema     *arrow.Schema
	schemaOpts      SchemaOptions
}

type dataFileWriterOption func(writer *defaultDataFileWriter)

func withFormat(format tblutils.FileFormat) dataFileWriterOption {
	return func(writer *defaultDataFileWriter) {
		writer.format = format
	}
}

func withFileSchema(schema *iceberg.Schema) dataFileWriterOption {
	return func(writer *defaultDataFileWriter) {
		writer.fileSchema = schema
	}
}

func withContent(content iceberg.ManifestEntryContent) dataFileWriterOption {
	return func(writer *defaultDataFileWriter) {
		writer.content = content
	}
}

func withEqualityFieldIDs(ids []int) dataFileWriterOption {
	return func(writer *defaultDataFileWriter) {
		writer.equalityFieldIDs = ids
	}
}

func newDataFileWriter(rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (*defaultDataFileWriter, error) {
	locProvider, err := LoadLocationProvider(rootLocation, props)
	if err != nil {
		return nil, err
	}

	fileFormat, err := iceberg.FileFormatFromString(
		props.Get(WriteFormatDefaultKey, WriteFormatDefaultDefault))
	if err != nil {
		return nil, err
	}
	if fileFormat == iceberg.ParquetFile {
		if err := tblutils.ValidateParquetWriteProperties(props); err != nil {
			return nil, err
		}
	}

	w := defaultDataFileWriter{
		loc:        locProvider,
		fs:         fs,
		fileSchema: meta.CurrentSchema(),
		fileFormat: fileFormat,
		format:     tblutils.GetFileFormat(fileFormat),
		content:    iceberg.EntryContentData,
		props:      props,
	}
	for _, apply := range opts {
		apply(&w)
	}

	statsCols, err := computeStatsPlan(w.fileSchema, meta.props)
	if err != nil {
		return nil, err
	}

	currentSpec, err := meta.CurrentSpec()
	if err != nil {
		return nil, err
	}

	var rowGroupBytes int64
	if w.fileFormat == iceberg.ParquetFile {
		rowGroupBytes, err = tblutils.ParquetRowGroupTargetSizeBytes(w.props)
		if err != nil {
			return nil, err
		}
	}
	arrowSchema, err := SchemaToArrowSchemaWithOptions(w.fileSchema, ArrowSchemaOptions{
		IncludeFieldIDs: true,
		TableProperties: meta.props,
	})
	if err != nil {
		return nil, err
	}

	var colMapping map[string]int
	var variantFieldIDs map[int]struct{}
	if w.fileFormat == iceberg.ParquetFile {
		colMapping, err = w.format.PathToIDMapping(w.fileSchema)
		if err != nil {
			return nil, err
		}
		variantFieldIDs = tblutils.VariantFieldIDsFromSchema(w.fileSchema)
	}

	w.invariants = dataWriterInvariants{
		statsCols:       statsCols,
		colMapping:      colMapping,
		variantFieldIDs: variantFieldIDs,
		writeProps:      w.format.GetWriteProperties(w.props),
		rowGroupBytes:   rowGroupBytes,
		spec:            *currentSpec,
		extension:       strings.ToLower(string(w.fileFormat)),
		arrowSchema:     arrowSchema,
		schemaOpts: SchemaOptions{
			DowncastTimestamp: true,
			IncludeFieldIDs:   true,
			UseWriteDefault:   true,
			TableProperties:   meta.props,
		},
	}

	return &w, nil
}

func (w *defaultDataFileWriter) writeFile(ctx context.Context, partitionValues map[int]any, task WriteTask) (iceberg.DataFile, error) {
	defer func() {
		for _, b := range task.Batches {
			b.Release()
		}
	}()

	// Reject here on the caller's goroutine, not deep in the writer's Close().
	if task.SortOrderID != UnsortedSortOrderID && w.content == iceberg.EntryContentPosDeletes {
		return nil, fmt.Errorf("position delete file claims sort order id %d; the spec requires a null sort order id for position deletes",
			task.SortOrderID)
	}

	batches := make([]arrow.RecordBatch, len(task.Batches))
	for i, b := range task.Batches {
		rec, err := toRequestedSchema(ctx, w.fileSchema,
			task.Schema, b, w.invariants.schemaOpts, w.invariants.arrowSchema)
		if err != nil {
			return nil, err
		}
		batches[i] = rec
		defer rec.Release()
	}

	filePath := w.loc.NewDataLocation(
		task.GenerateDataFileName(w.invariants.extension))

	return w.format.WriteDataFile(ctx, w.fs, partitionValues, tblutils.WriteFileInfo{
		FileSchema:       w.fileSchema,
		Content:          w.content,
		FileName:         filePath,
		StatsCols:        w.invariants.statsCols,
		ColMapping:       w.invariants.colMapping,
		VariantFieldIDs:  w.invariants.variantFieldIDs,
		WriteProps:       w.invariants.writeProps,
		RowGroupBytes:    w.invariants.rowGroupBytes,
		Spec:             w.invariants.spec,
		EqualityFieldIDs: w.equalityFieldIDs,
		SortOrderID:      task.SortOrderID,
	}, batches)
}

type dataFileWriter interface {
	writeFile(ctx context.Context, partitionValues map[int]any, task WriteTask) (iceberg.DataFile, error)
}

func newPositionDeleteWriter(rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (*defaultDataFileWriter, error) {
	// Always enforce the file schema to be the Positional Delete Schema by appending the option at the very end
	return newDataFileWriter(rootLocation, fs, meta, props, append(opts, withFileSchema(iceberg.PositionalDeleteSchema), withContent(iceberg.EntryContentPosDeletes))...)
}

func newEqualityDeleteWriter(rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (*defaultDataFileWriter, error) {
	return newDataFileWriter(rootLocation, fs, meta, props,
		append(opts, withContent(iceberg.EntryContentEqDeletes))...)
}

func newEqualityDeleteWriterMaker(deleteSchema *iceberg.Schema, equalityFieldIDs []int) dataFileWriterMaker {
	return func(rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (dataFileWriter, error) {
		return newEqualityDeleteWriter(rootLocation, fs, meta, props,
			append(opts, withFileSchema(deleteSchema), withEqualityFieldIDs(equalityFieldIDs))...)
	}
}

type dataFileWriterMaker func(rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, opts ...dataFileWriterOption) (dataFileWriter, error)

type concurrentDataFileWriter struct {
	newDataFileWriter dataFileWriterMaker
	sanitizeSchema    bool
}

type concurrentDataFileWriterOption func(w *concurrentDataFileWriter)

func withSchemaSanitization(enabled bool) concurrentDataFileWriterOption {
	return func(w *concurrentDataFileWriter) {
		w.sanitizeSchema = enabled
	}
}

func newConcurrentDataFileWriter(newDataFileWriter dataFileWriterMaker, opts ...concurrentDataFileWriterOption) *concurrentDataFileWriter {
	w := concurrentDataFileWriter{
		newDataFileWriter: newDataFileWriter,
		sanitizeSchema:    true,
	}
	for _, apply := range opts {
		apply(&w)
	}

	return &w
}

func (w *concurrentDataFileWriter) writeFiles(ctx context.Context, rootLocation string, fs io.WriteFileIO, meta *MetadataBuilder, props iceberg.Properties, partitionValues map[int]any, tasks iter.Seq[WriteTask]) iter.Seq2[iceberg.DataFile, error] {
	fileSchema := meta.CurrentSchema()
	if w.sanitizeSchema {
		sanitized, err := iceberg.SanitizeColumnNames(fileSchema)
		if err != nil {
			return func(yield func(iceberg.DataFile, error) bool) {
				yield(nil, err)
			}
		}

		// if the schema needs to be transformed, use the transformed schema
		// and adjust the arrow schema appropriately. otherwise we just
		// use the original schema.
		if !sanitized.Equals(fileSchema) {
			fileSchema = sanitized
		}
	}

	fw, err := w.newDataFileWriter(rootLocation, fs, meta, props, withFileSchema(fileSchema))
	if err != nil {
		return func(yield func(iceberg.DataFile, error) bool) {
			yield(nil, err)
		}
	}

	return tblutils.MapExec(ctx, config.EnvConfig.MaxWorkers, tasks, func(t WriteTask) (iceberg.DataFile, error) {
		return fw.writeFile(ctx, partitionValues, t)
	})
}
