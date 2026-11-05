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
	"context"
	"fmt"
	"io"
	"path"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/iceberg-go"
	iceio "github.com/apache/iceberg-go/io"
)

// GetFile opens the given file using the provided file system.
//
// The FileSource interface allows abstracting away the underlying file format
// while providing utilties to read the file as Arrow record batches.
func GetFile(ctx context.Context, fs iceio.IO, dataFile iceberg.DataFile, isPosDeletes bool) (FileSource, error) {
	switch dataFile.FileFormat() {
	case iceberg.ParquetFile:
		return &ParquetFileSource{
			mem:  compute.GetAllocator(ctx),
			fs:   fs,
			file: dataFile,
		}, nil
	default:
		return nil, fmt.Errorf("%w: only parquet format is implemented, got %s",
			iceberg.ErrNotImplemented, dataFile.FileFormat())
	}
}

type FileSource interface {
	GetReader(context.Context) (FileReader, error)
}

type Metadata any

type FileReader interface {
	io.Closer

	Metadata() Metadata
	SourceFileSize() int64
	Schema() (*arrow.Schema, error)
	// PrunedSchema takes in the list of projected field IDs and returns the arrow schema
	// that represents the underlying file schema with only the projected fields. It also
	// returns the indexes of the projected columns to allow reading *only* the needed
	// columns.
	PrunedSchema(projectedIDs map[int]struct{}, mapping iceberg.NameMapping) (*arrow.Schema, []int, error)
	// GetRecords returns a record reader for only the provided columns (using nil will read
	// all of the columns of the underlying file.) The `tester` is a function that can be used,
	// if non-nil, to filter aspects of the file such as skipping row groups in a parquet file.
	GetRecords(ctx context.Context, cols []int, tester any) (array.RecordReader, error)
	// ReadTable reads the entire file and returns it as an arrow table.
	ReadTable(context.Context) (arrow.Table, error)
}

// FileWriter is an incremental single-file writer with open/write/close
// lifecycle. It writes Arrow record batches and tracks bytes written for
// rolling file decisions.
type FileWriter interface {
	Write(arrow.RecordBatch) error
	BytesWritten() int64
	Close() (iceberg.DataFile, error)
	// Abort closes the underlying file handle without finalizing the
	// file format (e.g. Parquet footer). It is safe to call regardless
	// of how many rows have been written and should be used on error
	// paths where Close() may panic or produce an invalid file.
	Abort() error
}

type FileFormat interface {
	Open(context.Context, iceio.IO, string) (FileReader, error)
	PathToIDMapping(*iceberg.Schema) (map[string]int, error)
	DataFileStatsFromMeta(rdr Metadata, statsCols map[int]StatisticsCollector, colMapping map[string]int, variantFieldIDs map[int]struct{}, arrowSchema *arrow.Schema) *DataFileStatistics
	GetWriteProperties(iceberg.Properties) any
	WriteDataFile(ctx context.Context, fs iceio.WriteFileIO, partitionValues map[int]any, info WriteFileInfo, batches []arrow.RecordBatch) (iceberg.DataFile, error)
	NewFileWriter(ctx context.Context, fs iceio.WriteFileIO, partitionValues map[int]any, info WriteFileInfo, arrowSchema *arrow.Schema) (FileWriter, error)
}

func GetFileFormat(format iceberg.FileFormat) FileFormat {
	switch format {
	case iceberg.ParquetFile:
		return parquetFormat{}
	default:
		return nil
	}
}

func FormatFromFileName(fileName string) FileFormat {
	switch path.Ext(fileName) {
	case ".parquet":
		return parquetFormat{}
	default:
		return nil
	}
}

type WriteFileInfo struct {
	FileSchema       *iceberg.Schema
	Spec             iceberg.PartitionSpec
	FileName         string
	StatsCols        map[int]StatisticsCollector
	ColMapping       map[string]int
	VariantFieldIDs  map[int]struct{}
	WriteProps       any
	RowGroupBytes    int64
	Content          iceberg.ManifestEntryContent
	EqualityFieldIDs []int
	SortOrderID      int
}

type tablePropertiesContextKey struct{}

// WithTableProperties returns a new context with the given table properties attached.
// These properties are used by readers to configure read behavior (e.g. batch size).
func WithTableProperties(ctx context.Context, props iceberg.Properties) context.Context {
	return context.WithValue(ctx, tablePropertiesContextKey{}, props)
}

// TablePropertiesFromContext retrieves table properties from context.
// Returns nil if not set.
func TablePropertiesFromContext(ctx context.Context) iceberg.Properties {
	props, _ := ctx.Value(tablePropertiesContextKey{}).(iceberg.Properties)

	return props
}
