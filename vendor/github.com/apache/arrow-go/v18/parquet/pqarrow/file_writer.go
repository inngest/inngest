// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !pqarrow_read_only

package pqarrow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/flight"
	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/metadata"
)

// normalizeFieldForParquet recursively normalizes an Arrow field so that its
// type matches the Parquet column structure that fieldToNode would produce.
// Specifically, list element field names are set to "element" because
// ListOfWithName (used by fieldToNode) always names the Parquet element group
// "element", regardless of the original Arrow element field name.
func normalizeFieldForParquet(f arrow.Field) (arrow.Field, error) {
	switch dt := f.Type.(type) {
	case *arrow.ListType:
		elem, err := normalizeFieldForParquet(dt.ElemField())
		if err != nil {
			return arrow.Field{}, err
		}
		elem.Name = "element"
		return arrow.Field{Name: f.Name, Type: arrow.ListOfField(elem), Nullable: f.Nullable, Metadata: f.Metadata}, nil
	case *arrow.LargeListType:
		elem, err := normalizeFieldForParquet(dt.ElemField())
		if err != nil {
			return arrow.Field{}, err
		}
		elem.Name = "element"
		return arrow.Field{Name: f.Name, Type: arrow.LargeListOfField(elem), Nullable: f.Nullable, Metadata: f.Metadata}, nil
	case *arrow.FixedSizeListType:
		elem, err := normalizeFieldForParquet(dt.ElemField())
		if err != nil {
			return arrow.Field{}, err
		}
		elem.Name = "element"
		return arrow.Field{Name: f.Name, Type: arrow.FixedSizeListOfField(dt.Len(), elem), Nullable: f.Nullable, Metadata: f.Metadata}, nil
	case *arrow.StructType:
		fields := make([]arrow.Field, dt.NumFields())
		for i := 0; i < dt.NumFields(); i++ {
			field, err := normalizeFieldForParquet(dt.Field(i))
			if err != nil {
				return arrow.Field{}, err
			}
			fields[i] = field
		}
		return arrow.Field{Name: f.Name, Type: arrow.StructOf(fields...), Nullable: f.Nullable, Metadata: f.Metadata}, nil
	case *arrow.MapType:
		key, err := normalizeFieldForParquet(dt.KeyField())
		if err != nil {
			return arrow.Field{}, err
		}
		item, err := normalizeFieldForParquet(dt.ItemField())
		if err != nil {
			return arrow.Field{}, err
		}
		return arrow.Field{Name: f.Name, Type: arrow.MapOfFields(key, item), Nullable: f.Nullable, Metadata: f.Metadata}, nil
	case *arrow.RunEndEncodedType:
		return arrow.Field{}, fmt.Errorf("RunEndEncoded types are not supported for writing to Parquet files: field %s", f.Name)
	case *arrow.ListViewType:
		return arrow.Field{}, fmt.Errorf("ListView types are not supported for writing to Parquet files: field %s", f.Name)
	case *arrow.LargeListViewType:
		return arrow.Field{}, fmt.Errorf("LargeListView types are not supported for writing to Parquet files: field %s", f.Name)
	}
	return f, nil
}

// normalizeSchemaForParquet returns a copy of the Arrow schema with list element
// field names updated to "element" to match the Parquet column paths produced by
// fieldToNode. This is used when storing the ARROW:schema metadata to ensure
// consistency between the stored schema and the actual Parquet column structure.
func normalizeSchemaForParquet(sc *arrow.Schema) (*arrow.Schema, error) {
	fields := make([]arrow.Field, sc.NumFields())
	for i, f := range sc.Fields() {
		field, err := normalizeFieldForParquet(f)
		if err != nil {
			return nil, err
		}
		fields[i] = field
	}
	meta := sc.Metadata()
	return arrow.NewSchema(fields, &meta), nil
}

// WriteTable is a convenience function to create and write a full array.Table to a parquet file. The schema
// and columns will be determined by the schema of the table, writing the file out to the provided writer.
// The chunksize will be utilized in order to determine the size of the row groups.
func WriteTable(tbl arrow.Table, w io.Writer, chunkSize int64, props *parquet.WriterProperties, arrprops ArrowWriterProperties) error {
	writer, err := NewFileWriter(tbl.Schema(), w, props, arrprops)
	if err != nil {
		return err
	}

	if err := writer.WriteTable(tbl, chunkSize); err != nil {
		return err
	}

	return writer.Close()
}

// FileWriter is an object for writing Arrow directly to a parquet file.
type FileWriter struct {
	wr         *file.Writer
	schema     *arrow.Schema
	manifest   *SchemaManifest
	rgw        file.RowGroupWriter
	arrowProps ArrowWriterProperties
	ctx        context.Context
	colIdx     int
	closed     bool
}

// NewFileWriter returns a writer for writing Arrow directly to a parquetfile, rather than
// the ArrowColumnWriter and WriteArrow functions which allow writing arrow to an existing
// file.Writer, this will create a new file.Writer based on the schema provided.
func NewFileWriter(arrschema *arrow.Schema, w io.Writer, props *parquet.WriterProperties, arrprops ArrowWriterProperties) (*FileWriter, error) {
	if props == nil {
		props = parquet.NewWriterProperties()
	}

	pqschema, err := ToParquet(arrschema, props, arrprops)
	if err != nil {
		return nil, err
	}

	meta := make(metadata.KeyValueMetadata, 0)
	for i := 0; i < arrschema.Metadata().Len(); i++ {
		meta.Append(arrschema.Metadata().Keys()[i], arrschema.Metadata().Values()[i])
	}

	if arrprops.storeSchema {
		// Normalize the Arrow schema so that list element field names match the
		// Parquet column group names. fieldToNode always uses "element" as the
		// Parquet group name for list element fields (via ListOfWithName), but
		// arrow.ListOf() uses "item" as the Arrow element field name. This
		// inconsistency causes readers (e.g. Snowflake) that map ARROW:schema field
		// names to Parquet column paths to fail to locate the correct columns.
		schemaToStore, err := normalizeSchemaForParquet(arrschema)
		if err != nil {
			return nil, err
		}
		serializedSchema := flight.SerializeSchema(schemaToStore, props.Allocator())
		meta.Append("ARROW:schema", base64.StdEncoding.EncodeToString(serializedSchema))
	}

	schemaNode := pqschema.Root()
	baseWriter, err := file.NewParquetWriterWithError(w, schemaNode, file.WithWriterProps(props), file.WithWriteMetadata(meta))
	if err != nil {
		return nil, err
	}

	manifest, err := NewSchemaManifest(pqschema, nil, &ArrowReadProperties{})
	if err != nil {
		return nil, err
	}

	return &FileWriter{wr: baseWriter, schema: arrschema, manifest: manifest, arrowProps: arrprops, ctx: NewArrowWriteContext(context.TODO(), &arrprops)}, nil
}

// NewRowGroup does what it says on the tin, creates a new row group in the underlying file.
// Equivalent to `AppendRowGroup` on a file.Writer
// It panics if closing the previous row group or starting the new row group fails.
// Use NewRowGroupChecked to handle those errors explicitly.
func (fw *FileWriter) NewRowGroup() {
	if err := fw.NewRowGroupChecked(); err != nil {
		panic(err)
	}
}

// NewRowGroupChecked is the error-returning form of NewRowGroup.
func (fw *FileWriter) NewRowGroupChecked() error {
	if fw.rgw != nil {
		if err := fw.rgw.Close(); err != nil {
			return err
		}
	}
	rgw, err := fw.wr.AppendRowGroupChecked()
	if err != nil {
		return err
	}
	fw.rgw = rgw
	fw.colIdx = 0
	return nil
}

// NewBufferedRowGroup starts a new memory Buffered Row Group to allow writing columns / records
// without immediately flushing them to disk. This allows using WriteBuffered to write records
// and decide where to break your row group based on the TotalBytesWritten rather than on the max
// row group len. If using Records, this should be paired with WriteBuffered, while
// Write will always write a new record as a row group in and of itself.
// It panics if closing the previous row group or starting the new row group fails.
// Use NewBufferedRowGroupChecked to handle those errors explicitly.
func (fw *FileWriter) NewBufferedRowGroup() {
	if err := fw.NewBufferedRowGroupChecked(); err != nil {
		panic(err)
	}
}

// NewBufferedRowGroupChecked is the error-returning form of NewBufferedRowGroup.
func (fw *FileWriter) NewBufferedRowGroupChecked() error {
	if fw.rgw != nil {
		if err := fw.rgw.Close(); err != nil {
			return err
		}
	}
	rgw, err := fw.wr.AppendBufferedRowGroupChecked()
	if err != nil {
		return err
	}
	fw.rgw = rgw
	fw.colIdx = 0
	return nil
}

// TotalCompressedBytes returns the total number of bytes after compression
// that have been written to the file so far. It includes all the closed row groups
// and the current row group.
func (fw *FileWriter) TotalCompressedBytes() int64 {
	if fw.wr != nil {
		return fw.wr.TotalCompressedBytes()
	}
	return 0
}

// TotalBytesWritten returns the total number of bytes
// that have been written to the file so far. It includes all the closed row groups
// and the current row group.
func (fw *FileWriter) TotalBytesWritten() int64 {
	if fw.wr != nil {
		return fw.wr.TotalBytesWritten()
	}
	return 0
}

// RowGroupTotalCompressedBytes returns the total number of bytes after compression
// that have been written to the current row group so far.
func (fw *FileWriter) RowGroupTotalCompressedBytes() int64 {
	if fw.rgw != nil {
		return fw.rgw.TotalCompressedBytes()
	}
	return 0
}

// RowGroupTotalBytesWritten returns the total number of bytes written and flushed out in
// the current row group.
func (fw *FileWriter) RowGroupTotalBytesWritten() int64 {
	if fw.rgw != nil {
		return fw.rgw.TotalBytesWritten()
	}
	return 0
}

// RowGroupNumRows returns the number of rows written to the current row group.
// Returns an error if they are unequal between columns that have been written so far.
func (fw *FileWriter) RowGroupNumRows() (int, error) {
	if fw.rgw != nil {
		return fw.rgw.NumRows()
	}
	return 0, nil
}

// NumRows returns the total number of rows that have been written so far.
func (fw *FileWriter) NumRows() int {
	if fw.wr != nil {
		return fw.wr.NumRows()
	}
	return 0
}

// WriteBuffered will either append to an existing row group or create a new one
// based on the record length and max row group length.
//
// Additionally, it allows to manually break your row group by
// checking RowGroupTotalBytesWritten and calling NewBufferedRowGroup,
// while Write will always create at least 1 row group for the record.
//
// Performance-wise WriteBuffered might be more favorable than Write if you're dealing with:
// * a loose memory environment (meaning you have a lot of memory to utilize)
// * records that have only a small (~<1K?) amount of rows
//
// More memory is utilized compared to Write as the whole row group data is kept in memory before it's written
// since Parquet files must have an entire column written before writing the next column.
func (fw *FileWriter) WriteBuffered(rec arrow.RecordBatch) error {
	if fw.closed {
		return fmt.Errorf("WriteBuffered called on already closed FileWriter")
	}
	if !rec.Schema().Equal(fw.schema) {
		return fmt.Errorf("record schema does not match writer's. \nrecord: %s\nwriter: %s", rec.Schema(), fw.schema)
	}

	var (
		maxRows = fw.wr.Properties().MaxRowGroupLength()
		curRows int
		err     error
	)
	if fw.rgw != nil {
		if curRows, err = fw.rgw.NumRows(); err != nil {
			return err
		}
	} else {
		if err := fw.NewBufferedRowGroupChecked(); err != nil {
			return err
		}
	}
	if int64(curRows)+rec.NumRows() <= maxRows {
		if err := fw.writeRecordBatchColumns(rec); err != nil {
			fw.Close()
			return err
		}
		fw.colIdx = 0
		return nil
	}

	columns := newChunkedRecordColumns(rec)
	defer releaseChunkedRecordColumns(columns)

	firstSize := maxRows - int64(curRows)

	for offset, size := int64(0), firstSize; ; {
		if err := fw.writeRecordBatchRange(columns, offset, size); err != nil {
			fw.Close()
			return err
		}
		if offset+size >= rec.NumRows() {
			break
		}
		if err := fw.NewBufferedRowGroupChecked(); err != nil {
			return err
		}
		offset += size
		size = utils.Min(maxRows, rec.NumRows()-offset)
	}
	fw.colIdx = 0
	return nil
}

// Write an arrow Record Batch to the file, respecting the MaxRowGroupLength in the writer
// properties to determine whether the record is broken up into more than one row group.
// At the very least a single row group is created per record,
// so calling Write always results in a new row group added.
//
// Performance-wise Write might be more favorable than WriteBuffered if you're dealing with:
// * a highly-restricted memory environment
// * very large records with lots of rows (potentially close to the max row group length)
func (fw *FileWriter) Write(rec arrow.RecordBatch) error {
	if fw.closed {
		return fmt.Errorf("invalid write call: FileWriter is already closed")
	}
	if !rec.Schema().Equal(fw.schema) {
		return fmt.Errorf("record schema does not match writer's. \nrecord: %s\nwriter: %s", rec.Schema(), fw.schema)
	}

	rowgroupLen := fw.wr.Properties().MaxRowGroupLength()
	if rec.NumRows() <= rowgroupLen {
		if err := fw.NewRowGroupChecked(); err != nil {
			return err
		}
		if err := fw.writeRecordBatchColumns(rec); err != nil {
			fw.Close()
			return err
		}
		fw.colIdx = 0
		return fw.rgw.Close()
	}

	columns := newChunkedRecordColumns(rec)
	defer releaseChunkedRecordColumns(columns)

	for offset := int64(0); offset < rec.NumRows(); offset += rowgroupLen {
		if err := fw.NewRowGroupChecked(); err != nil {
			return err
		}
		size := utils.Min(rowgroupLen, rec.NumRows()-offset)
		if err := fw.writeRecordBatchRange(columns, offset, size); err != nil {
			fw.Close()
			return err
		}
	}
	fw.colIdx = 0
	return fw.rgw.Close()
}

func newChunkedRecordColumns(rec arrow.RecordBatch) []*arrow.Chunked {
	columns := make([]*arrow.Chunked, int(rec.NumCols()))
	for i, column := range rec.Columns() {
		columns[i] = arrow.NewChunked(column.DataType(), []arrow.Array{column})
	}
	return columns
}

func releaseChunkedRecordColumns(columns []*arrow.Chunked) {
	for _, column := range columns {
		column.Release()
	}
}

func (fw *FileWriter) writeRecordBatchColumns(rec arrow.RecordBatch) error {
	for i := 0; i < int(rec.NumCols()); i++ {
		if err := fw.WriteColumnData(rec.Column(i)); err != nil {
			return err
		}
	}
	return nil
}

func (fw *FileWriter) writeRecordBatchRange(columns []*arrow.Chunked, offset, size int64) error {
	for _, column := range columns {
		if err := fw.WriteColumnChunked(column, offset, size); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable writes an arrow table to the underlying file using chunkSize to determine
// the size to break at for making row groups. Writing a table will always create a new
// row group for each chunk of chunkSize rows in the table. Calling this with 0 rows will
// still write a 0 length Row Group to the file.
func (fw *FileWriter) WriteTable(tbl arrow.Table, chunkSize int64) error {
	if fw.closed {
		return fmt.Errorf("invalid write call: FileWriter is already closed")
	}
	if chunkSize <= 0 && tbl.NumRows() > 0 {
		return errors.New("chunk size per row group must be greater than 0")
	} else if !tbl.Schema().Equal(fw.schema) {
		return fmt.Errorf("table schema does not match writer's. \nTable: %s\n writer: %s", tbl.Schema(), fw.schema)
	} else if chunkSize > fw.wr.Properties().MaxRowGroupLength() {
		chunkSize = fw.wr.Properties().MaxRowGroupLength()
	}

	writeRowGroup := func(offset, size int64) error {
		if err := fw.NewRowGroupChecked(); err != nil {
			return err
		}
		for i := 0; i < int(tbl.NumCols()); i++ {
			if err := fw.WriteColumnChunked(tbl.Column(i).Data(), offset, size); err != nil {
				return err
			}
		}
		return nil
	}

	if tbl.NumRows() == 0 {
		if err := writeRowGroup(0, 0); err != nil {
			fw.Close()
			return err
		}
		return nil
	}

	for offset := int64(0); offset < tbl.NumRows(); offset += chunkSize {
		if err := writeRowGroup(offset, utils.Min(chunkSize, tbl.NumRows()-offset)); err != nil {
			fw.Close()
			return err
		}
	}
	return nil
}

// AppendKeyValueMetadata appends a key/value pair to the existing key/value metadata
func (fw *FileWriter) AppendKeyValueMetadata(key string, value string) error {
	return fw.wr.AppendKeyValueMetadata(key, value)
}

// Close flushes out the data and closes the file. It can be called multiple times,
// subsequent calls after the first will have no effect. If the underlying writer implements
// [io.Closer], then this will also call Close on the underlying writer.
func (fw *FileWriter) Close() error {
	if !fw.closed {
		fw.closed = true
		if fw.rgw != nil {
			if err := fw.rgw.Close(); err != nil {
				return err
			}
		}

		writeCtx := arrowCtxFromContext(fw.ctx)
		if writeCtx.dataBuffer != nil {
			writeCtx.dataBuffer.Release()
			writeCtx.dataBuffer = nil
		}

		return fw.wr.Close()
	}
	return nil
}

// WriteColumnChunked will write the data provided to the underlying file, using the provided
// offset and size to allow writing subsets of data from the chunked column. It uses the current
// column in the underlying row group writer as the starting point, allowing progressive
// building of writing columns to a file via arrow data without needing to already have
// a record or table.
func (fw *FileWriter) WriteColumnChunked(data *arrow.Chunked, offset, size int64) error {
	if fw.closed {
		return fmt.Errorf("invalid write call: FileWriter is already closed")
	}
	acw, err := newArrowColumnWriter(data, offset, size, fw.manifest, fw.rgw, fw.colIdx)
	if err != nil {
		return err
	}
	fw.colIdx += acw.leafCount
	return acw.Write(fw.ctx)
}

// WriteColumnData writes the entire array to the file as the next columns. Like WriteColumnChunked
// it is based on the current column of the row group writer allowing progressive building
// of the file by columns without needing a full record or table to write.
func (fw *FileWriter) WriteColumnData(data arrow.Array) error {
	if fw.closed {
		return fmt.Errorf("invalid write call: FileWriter is already closed")
	}
	chunked := arrow.NewChunked(data.DataType(), []arrow.Array{data})
	defer chunked.Release()
	return fw.WriteColumnChunked(chunked, 0, int64(data.Len()))
}

// FileMetadata returns the current state of the FileMetadata that would be written
// if this file were to be closed. If the file has already been closed, then this
// will return the FileMetaData which was written to the file.
func (fw *FileWriter) FileMetadata() (*metadata.FileMetaData, error) {
	return fw.wr.FileMetadata()
}
