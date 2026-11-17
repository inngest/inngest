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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/decimal128"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/metadata"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
	"github.com/apache/iceberg-go"
	"github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/geoarrow/geoarrow-go"
	"github.com/google/uuid"
)

const (
	ParquetRowGroupSizeBytesKey       = "write.parquet.row-group-size-bytes"
	ParquetRowGroupSizeBytesDefault   = 128 * 1024 * 1024 // 128 MB
	ParquetRowGroupLimitKey           = "write.parquet.row-group-limit"
	ParquetRowGroupLimitDefault       = 1048576
	ParquetPageSizeBytesKey           = "write.parquet.page-size-bytes"
	ParquetPageSizeBytesDefault       = 1024 * 1024 // 1 MB
	ParquetPageRowLimitKey            = "write.parquet.page-row-limit"
	ParquetPageRowLimitDefault        = 20000
	ParquetDictSizeBytesKey           = "write.parquet.dict-size-bytes"
	ParquetDictSizeBytesDefault       = 2 * 1024 * 1024 // 2 MB
	ParquetPageVersionKey             = "write.parquet.page-version"
	ParquetPageVersionDefault         = "2"
	ParquetCompressionKey             = "write.parquet.compression-codec"
	ParquetCompressionDefault         = "zstd"
	ParquetCompressionLevelKey        = "write.parquet.compression-level"
	ParquetCompressionLevelDefault    = -1
	ParquetBloomFilterMaxBytesKey     = "write.parquet.bloom-filter-max-bytes"
	ParquetBloomFilterMaxBytesDefault = 1024 * 1024
	// These bounds mirror Arrow's public bloom-filter writer-property constraint.
	parquetBloomFilterMaxBytesMin            = 32
	parquetBloomFilterMaxBytesMax            = 128 * 1024 * 1024
	ParquetBloomFilterColumnEnabledKeyPrefix = "write.parquet.bloom-filter-enabled.column"

	// These bounds mirror the compression libraries used by Arrow's Parquet
	// writers. -1 is the shared default level and is included in each range.
	parquetGzipCompressionLevelMin   = -3
	parquetGzipCompressionLevelMax   = 9
	parquetZstdCompressionLevelMin   = -5
	parquetZstdCompressionLevelMax   = 22
	parquetBrotliCompressionLevelMin = 0
	parquetBrotliCompressionLevelMax = 11

	// Deliberately not namespaced under write.parquet: this is the parquet-mr key
	// Iceberg Java reads straight from the table properties, so a table carrying
	// it gets the same dictionary behavior from either implementation.
	ParquetDictEnabledKey                     = "parquet.enable.dictionary"
	ParquetDictEnabledDefault                 = true
	ParquetDictEncodingColumnEnabledKeyPrefix = "write.parquet.dict-encoding-enabled.column"

	ParquetShredVariantsKey     = "write.parquet.shred-variants"
	ParquetShredVariantsDefault = false
	// Rows buffered per file to infer shredding (held per open partition writer).
	ParquetVariantBufferSizeKey     = "write.parquet.variant-inference-buffer-size"
	ParquetVariantBufferSizeDefault = 100

	ParquetBatchSizeKey     = "read.parquet.batch-size"
	ParquetBatchSizeDefault = 1 << 17 // 131072 rows
)

type parquetFormat struct{}

func (parquetFormat) Open(ctx context.Context, fs iceio.IO, path string) (_ FileReader, err error) {
	inputfile, err := fs.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			if closeErr := inputfile.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	rdr, err := file.NewParquetReader(inputfile)
	if err != nil {
		return nil, err
	}

	alloc := compute.GetAllocator(ctx)
	arrRdr, err := pqarrow.NewFileReader(rdr, pqarrow.ArrowReadProperties{}, alloc)
	if err != nil {
		return nil, err
	}

	return wrapPqArrowReader{
		FileReader:       arrRdr,
		dictionarySource: inputfile,
		dictionaryMem:    alloc,
	}, nil
}

func (parquetFormat) PathToIDMapping(sc *iceberg.Schema) (map[string]int, error) {
	result := make(map[string]int)

	paths, err := iceberg.PreOrderVisit(sc, &id2ParquetPathVisitor{})
	if err != nil {
		return nil, err
	}

	for _, entry := range paths {
		result[entry.path] = entry.fieldID
	}

	return result, nil
}

func VariantFieldIDsFromSchema(sc *iceberg.Schema) map[int]struct{} {
	if sc == nil {
		return nil
	}

	result := make(map[int]struct{})
	collectVariantFields(sc.Fields(), result)

	return result
}

func collectVariantFields(fields []iceberg.NestedField, result map[int]struct{}) {
	for _, field := range fields {
		if _, ok := field.Type.(iceberg.VariantType); ok {
			result[field.ID] = struct{}{}
		} else if nested, ok := field.Type.(iceberg.NestedType); ok {
			collectVariantFields(nested.Fields(), result)
		}
	}
}

func (p parquetFormat) createStatsAgg(typ iceberg.PrimitiveType, physicalTypeStr string, truncLen int) (StatsAgg, error) {
	// Switch on Iceberg logical type first, then handle physical representations.
	// This matches iceberg-java's approach in ParquetConversions.java.
	switch typ.(type) {
	case iceberg.BooleanType:
		return newStatAgg[bool](typ, truncLen), nil

	case iceberg.Int32Type, iceberg.DateType:
		return newStatAgg[int32](typ, truncLen), nil

	case iceberg.Int64Type, iceberg.TimeType, iceberg.TimestampType, iceberg.TimestampTzType,
		iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		// Allow INT32 physical for INT64 logical (promotion)
		if physicalTypeStr == "INT32" {
			return newStatAgg[int32](typ, truncLen), nil
		}

		return newStatAgg[int64](typ, truncLen), nil

	case iceberg.Float32Type:
		return newStatAgg[float32](typ, truncLen), nil

	case iceberg.Float64Type:
		// Allow FLOAT physical for DOUBLE logical (promotion)
		if physicalTypeStr == "FLOAT" {
			return newStatAgg[float32](typ, truncLen), nil
		}

		return newStatAgg[float64](typ, truncLen), nil

	case iceberg.StringType:
		return newStatAgg[string](typ, truncLen), nil

	case iceberg.BinaryType:
		return newStatAgg[[]byte](typ, truncLen), nil

	case iceberg.UUIDType:
		return newStatAgg[uuid.UUID](typ, truncLen), nil

	case iceberg.FixedType:
		return newStatAgg[[]byte](typ, truncLen), nil

	case iceberg.DecimalType:
		// Decimals can be stored as INT32 (precision <= 9), INT64 (precision <= 18),
		// FIXED_LEN_BYTE_ARRAY, or BYTE_ARRAY per Parquet spec.
		switch physicalTypeStr {
		case "INT32":
			return &decAsIntAgg[int32]{
				newStatAgg[int32](typ, truncLen).(*statsAggregator[int32]),
			}, nil
		case "INT64":
			return &decAsIntAgg[int64]{
				newStatAgg[int64](typ, truncLen).(*statsAggregator[int64]),
			}, nil
		case "FIXED_LEN_BYTE_ARRAY", "BYTE_ARRAY":
			return newStatAgg[iceberg.Decimal](typ, truncLen), nil
		default:
			return nil, fmt.Errorf("unsupported physical type %s for decimal", physicalTypeStr)
		}

	default:
		return nil, fmt.Errorf("unsupported iceberg type: %s", typ)
	}
}

func (parquetFormat) PrimitiveTypeToPhysicalType(typ iceberg.PrimitiveType) string {
	switch typ.(type) {
	case iceberg.BooleanType:
		return "BOOLEAN"
	case iceberg.Int32Type:
		return "INT32"
	case iceberg.Int64Type:
		return "INT64"
	case iceberg.Float32Type:
		return "FLOAT"
	case iceberg.Float64Type:
		return "DOUBLE"
	case iceberg.DateType:
		return "INT32"
	case iceberg.TimeType:
		return "INT64"
	case iceberg.TimestampType, iceberg.TimestampTzType,
		iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return "INT64"
	case iceberg.StringType:
		return "BYTE_ARRAY"
	case iceberg.UUIDType:
		return "FIXED_LEN_BYTE_ARRAY"
	case iceberg.FixedType:
		return "FIXED_LEN_BYTE_ARRAY"
	case iceberg.BinaryType:
		return "BYTE_ARRAY"
	case iceberg.DecimalType:
		return "FIXED_LEN_BYTE_ARRAY"
	default:
		panic(fmt.Errorf("expected primitive type, got: %s", typ))
	}
}

func (parquetFormat) GetWriteProperties(props iceberg.Properties) any {
	pageVersion := props.Get(ParquetPageVersionKey, ParquetPageVersionDefault)

	var dpVersion parquet.DataPageVersion
	switch pageVersion {
	case "1":
		dpVersion = parquet.DataPageV1
	case "2":
		dpVersion = parquet.DataPageV2
	default:
		slog.Warn("unrecognized data page version, falling back to v2", "version", pageVersion)
		dpVersion = parquet.DataPageV2
	}

	writerProps := []parquet.WriterProperty{
		parquet.WithMaxRowGroupLength(int64(props.GetInt(ParquetRowGroupLimitKey,
			ParquetRowGroupLimitDefault))),
		parquet.WithDataPageSize(int64(props.GetInt(ParquetPageSizeBytesKey,
			ParquetPageSizeBytesDefault))),
		parquet.WithDataPageVersion(dpVersion),
		parquet.WithBatchSize(int64(props.GetInt(ParquetPageRowLimitKey,
			ParquetPageRowLimitDefault))),
		parquet.WithDictionaryPageSizeLimit(int64(props.GetInt(ParquetDictSizeBytesKey,
			ParquetDictSizeBytesDefault))),
	}

	compression := props.Get(ParquetCompressionKey, ParquetCompressionDefault)
	compressionLevel := props.GetInt(ParquetCompressionLevelKey,
		ParquetCompressionLevelDefault)

	var codec compress.Compression
	switch compression {
	case "snappy":
		codec = compress.Codecs.Snappy
	case "zstd":
		codec = compress.Codecs.Zstd
	case "uncompressed":
		codec = compress.Codecs.Uncompressed
	case "gzip":
		codec = compress.Codecs.Gzip
	case "brotli":
		codec = compress.Codecs.Brotli
	case "lz4":
		codec = compress.Codecs.Lz4
	case "lz4raw":
		codec = compress.Codecs.Lz4Raw
	case "lzo":
		codec = compress.Codecs.Lzo
	default:
		slog.Warn("unrecognized compression codec, falling back to uncompressed", "codec", compression)
	}

	writerProps = append(writerProps, parquet.WithCompression(codec),
		parquet.WithCompressionLevel(compressionLevel))

	// Bloom filter properties.
	// write.parquet.bloom-filter-max-bytes caps the per-column bloom filter size.
	bloomMaxBytes := props.GetInt(ParquetBloomFilterMaxBytesKey, ParquetBloomFilterMaxBytesDefault)
	writerProps = append(writerProps, parquet.WithMaxBloomFilterBytes(int64(bloomMaxBytes)))

	// write.parquet.bloom-filter-enabled.column.<col-name> enables bloom filters
	// for individual columns. Scan all properties for the prefix.
	prefix := ParquetBloomFilterColumnEnabledKeyPrefix + "."
	for key, val := range props {
		colName, ok := strings.CutPrefix(key, prefix)
		if !ok || colName == "" {
			continue
		}
		writerProps = append(writerProps, parquet.WithBloomFilterEnabledFor(colName, javaBool(val)))
	}

	// Dictionary encoding is on unless parquet.enable.dictionary disables it;
	// write.parquet.dict-encoding-enabled.column.<col-name> overrides per column.
	dictEnabled := ParquetDictEnabledDefault
	if val, ok := props[ParquetDictEnabledKey]; ok {
		dictEnabled = javaBool(val)
	}
	writerProps = append(writerProps, parquet.WithDictionaryDefault(dictEnabled))

	// The suffix is matched against the Parquet physical column path, not the
	// Iceberg field name. These differ for nested fields (Iceberg `items.element`
	// is `items.list.element` in Parquet), so a per-column override on one of
	// those has no effect. Same for the bloom filter prefix above.
	prefix = ParquetDictEncodingColumnEnabledKeyPrefix + "."
	for key, val := range props {
		colName, ok := strings.CutPrefix(key, prefix)
		if !ok || colName == "" {
			continue
		}
		writerProps = append(writerProps, parquet.WithDictionaryFor(colName, javaBool(val)))
	}

	return writerProps
}

// javaBool matches Java's Boolean.parseBoolean, which Iceberg's Java writer uses
// for these properties: only "true" (case-insensitive) is true; anything else,
// including "1" and "yes", is false. strconv.ParseBool differs ("1" → true).
func javaBool(val string) bool {
	return strings.EqualFold(val, "true")
}

func ValidateParquetWriteProperties(props iceberg.Properties) error {
	// A missing property uses the writer default, but an explicitly configured
	// empty value is invalid. This keeps validation from silently accepting a
	// value that GetInt would otherwise replace with its default.
	for _, key := range []string{
		ParquetRowGroupSizeBytesKey,
		ParquetRowGroupLimitKey,
		ParquetPageSizeBytesKey,
		ParquetPageRowLimitKey,
		ParquetDictSizeBytesKey,
		ParquetBloomFilterMaxBytesKey,
	} {
		value, ok := props[key]
		if !ok {
			continue
		}

		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: invalid %s value %q: %v", iceberg.ErrInvalidArgument, key, value, err)
		}
		if parsed <= 0 {
			return fmt.Errorf("%w: %s must be greater than 0, got %d", iceberg.ErrInvalidArgument, key, parsed)
		}
		if key == ParquetBloomFilterMaxBytesKey &&
			(parsed < parquetBloomFilterMaxBytesMin || parsed > parquetBloomFilterMaxBytesMax) {
			return fmt.Errorf("%w: %s must be between %d and %d bytes, got %d",
				iceberg.ErrInvalidArgument, key, parquetBloomFilterMaxBytesMin, parquetBloomFilterMaxBytesMax, parsed)
		}
		// Row-group size is kept as int64 all the way through the writer. The
		// other values are passed to Arrow APIs that take int and therefore
		// must fit the platform's int range.
		if key != ParquetRowGroupSizeBytesKey && strconv.IntSize == 32 && parsed > int64(^uint(0)>>1) {
			return fmt.Errorf("%w: %s value %d exceeds int range", iceberg.ErrInvalidArgument, key, parsed)
		}
	}

	if value, ok := props[ParquetCompressionLevelKey]; ok {
		level, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%w: invalid %s value %q: %v", iceberg.ErrInvalidArgument,
				ParquetCompressionLevelKey, value, err)
		}

		compression := props.Get(ParquetCompressionKey, ParquetCompressionDefault)
		if err := validateParquetCompressionLevel(compression, level); err != nil {
			return err
		}
	}

	return nil
}

func validateParquetCompressionLevel(compression string, level int) error {
	min, max := 0, 0
	switch compression {
	case "gzip":
		min, max = parquetGzipCompressionLevelMin, parquetGzipCompressionLevelMax
	case "zstd":
		min, max = parquetZstdCompressionLevelMin, parquetZstdCompressionLevelMax
	case "brotli":
		// Brotli uses -1 as the Iceberg default even though its underlying
		// library accepts explicit qualities only in the 0..11 range.
		if level == ParquetCompressionLevelDefault {
			return nil
		}
		min, max = parquetBrotliCompressionLevelMin, parquetBrotliCompressionLevelMax
	default:
		// The remaining codecs do not expose a tunable level in Arrow's
		// streaming writers, so the value is ignored after it is parsed.
		return nil
	}

	if level < min || level > max {
		return fmt.Errorf("%w: invalid %s value %d for %s; supported range is %d..%d (or %d for the default)",
			iceberg.ErrInvalidArgument, ParquetCompressionLevelKey, level, compression, min, max,
			ParquetCompressionLevelDefault)
	}

	return nil
}

// ParquetRowGroupTargetSizeBytes returns the configured uncompressed row-group
// size target. Iceberg Java rejects invalid and non-positive values rather than
// silently falling back to the default.
func ParquetRowGroupTargetSizeBytes(props iceberg.Properties) (int64, error) {
	value, ok := props[ParquetRowGroupSizeBytesKey]
	if !ok {
		return ParquetRowGroupSizeBytesDefault, nil
	}

	size, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid %s value %q: %v",
			iceberg.ErrInvalidArgument, ParquetRowGroupSizeBytesKey, value, err)
	}
	if size <= 0 {
		return 0, fmt.Errorf("%w: %s must be greater than 0, got %d",
			iceberg.ErrInvalidArgument, ParquetRowGroupSizeBytesKey, size)
	}

	return size, nil
}

func (p parquetFormat) WriteDataFile(ctx context.Context, fs iceio.WriteFileIO, partitionValues map[int]any, info WriteFileInfo, batches []arrow.RecordBatch) (iceberg.DataFile, error) {
	w, err := p.NewFileWriter(ctx, fs, partitionValues, info, batches[0].Schema())
	if err != nil {
		return nil, err
	}

	return writeDataFileBatches(w, batches)
}

func writeDataFileBatches(w FileWriter, batches []arrow.RecordBatch) (iceberg.DataFile, error) {
	for _, batch := range batches {
		if err := w.Write(batch); err != nil {
			if abortErr := w.Abort(); abortErr != nil {
				return nil, errors.Join(err, abortErr)
			}

			return nil, err
		}
	}

	return w.Close()
}

// ParquetFileWriter is an incremental single-file writer with open/write/close
// lifecycle. It writes Arrow record batches to a Parquet file and tracks bytes
// written for rolling file decisions.
type ParquetFileWriter struct {
	pqWriter         *pqarrow.FileWriter
	counter          *internal.CountingWriter
	fileCloser       io.Closer
	mem              memory.Allocator
	fs               iceio.WriteFileIO
	format           parquetFormat
	info             WriteFileInfo
	partition        map[int]any
	colMapping       map[string]int
	variantFieldIDs  map[int]struct{}
	geoCols          []geoColumn
	geoNormalizeCols []int
	geoAccs          map[int]*geoBoundsAccumulator
	arrowSchema      *arrow.Schema
	rowGroupBytes    int64
}

// geoColumn locates a top-level geometry/geography column: its position in the
// Arrow record batch and the Iceberg field ID its bounds are recorded under.
type geoColumn struct {
	colIdx  int
	fieldID int
}

// collectGeoColumns finds top-level WKB-encoded geo columns in the Arrow schema
// and pairs each with its Iceberg field ID. Geo bounds for columns nested inside
// structs/lists/maps are not yet computed, which diverges from Java/PyIceberg.
// TODO(#992): compute geo bounds for geo columns nested in structs/lists/maps.
func collectGeoColumns(sc *arrow.Schema, colMapping map[string]int) []geoColumn {
	var result []geoColumn
	for i, f := range sc.Fields() {
		if _, ok := f.Type.(*geoarrow.WKBType); !ok {
			continue
		}
		fieldID, ok := colMapping[f.Name]
		if !ok {
			continue
		}
		result = append(result, geoColumn{colIdx: i, fieldID: fieldID})
	}

	return result
}

// collectGeoNormalizationColumns finds top-level columns whose supported Arrow
// container types can contain a GeoArrow WKB value. The result is computed once
// per writer so batches without geo columns take a constant-time fast path.
func collectGeoNormalizationColumns(sc *arrow.Schema) []int {
	var result []int
	for i, f := range sc.Fields() {
		if arrowTypeContainsWKB(f.Type) {
			result = append(result, i)
		}
	}

	return result
}

func arrowTypeContainsWKB(dt arrow.DataType) bool {
	// Keep this traversal in sync with normalizeNestedArrayReachable. The table
	// write path does not currently pass dictionary- or run-end-encoded WKB
	// arrays; supporting those inputs requires unwrapping them in both places.
	switch t := dt.(type) {
	case *geoarrow.WKBType:
		return true
	case *arrow.StructType:
		for _, field := range t.Fields() {
			if arrowTypeContainsWKB(field.Type) {
				return true
			}
		}
	case *arrow.ListType, *arrow.LargeListType, *arrow.FixedSizeListType, *arrow.MapType:
		return arrowTypeContainsWKB(dt.(arrow.ListLikeType).Elem())
	}

	return false
}

// NewFileWriter creates a ParquetFileWriter that writes batches to a single
// Parquet file. Call Write to append batches, BytesWritten to check actual
// compressed file size, and Close to finalize and get the resulting DataFile.
func (p parquetFormat) NewFileWriter(ctx context.Context, fs iceio.WriteFileIO,
	partitionValues map[int]any, info WriteFileInfo, arrowSchema *arrow.Schema,
) (FileWriter, error) {
	rowGroupTargetSizeBytes := info.RowGroupBytes
	if rowGroupTargetSizeBytes == 0 {
		rowGroupTargetSizeBytes = ParquetRowGroupSizeBytesDefault
	}
	if rowGroupTargetSizeBytes < 0 {
		return nil, fmt.Errorf("%w: row-group target size must be greater than 0, got %d",
			iceberg.ErrInvalidArgument, rowGroupTargetSizeBytes)
	}

	fw, err := fs.Create(info.FileName)
	if err != nil {
		return nil, err
	}

	colMapping := info.ColMapping
	if colMapping == nil {
		colMapping, err = p.PathToIDMapping(info.FileSchema)
		if err != nil {
			fw.Close()

			return nil, err
		}
	}

	variantFieldIDs := info.VariantFieldIDs
	if variantFieldIDs == nil {
		variantFieldIDs = VariantFieldIDsFromSchema(info.FileSchema)
	}

	counter := &internal.CountingWriter{W: fw}
	mem := compute.GetAllocator(ctx)
	arrProps := writeArrowProps(mem)
	writerProps, err := getWriteProperties(info.WriteProps, arrowSchema, arrProps)
	if err != nil {
		fw.Close()

		return nil, err
	}

	writer, err := pqarrow.NewFileWriter(arrowSchema, counter, writerProps, arrProps)
	if err != nil {
		fw.Close()

		return nil, err
	}

	geoCols := collectGeoColumns(arrowSchema, colMapping)
	geoNormalizeCols := collectGeoNormalizationColumns(arrowSchema)
	geoAccs := make(map[int]*geoBoundsAccumulator, len(geoCols))
	for _, gc := range geoCols {
		// The accumulator must know whether the column is geometry or geography
		// so it can omit unsafe geography bounds (see geoBoundsAccumulator.Bounds).
		var isGeog bool
		if t, ok := info.FileSchema.FindTypeByID(gc.fieldID); ok {
			_, isGeog = t.(iceberg.GeographyType)
		}
		geoAccs[gc.fieldID] = newGeoBoundsAccumulator(isGeog)
	}

	return &ParquetFileWriter{
		pqWriter:         writer,
		counter:          counter,
		fileCloser:       fw,
		mem:              mem,
		fs:               fs,
		format:           p,
		info:             info,
		partition:        partitionValues,
		colMapping:       colMapping,
		variantFieldIDs:  variantFieldIDs,
		geoCols:          geoCols,
		geoNormalizeCols: geoNormalizeCols,
		geoAccs:          geoAccs,
		arrowSchema:      arrowSchema,
		rowGroupBytes:    rowGroupTargetSizeBytes,
	}, nil
}

// arrowSchemaHasShreddedDecimal reports whether any top-level shredded variant's
// typed_value carries a Decimal128 leaf (recursively).
func arrowSchemaHasShreddedDecimal(sc *arrow.Schema) bool {
	for _, f := range sc.Fields() {
		if vt, ok := f.Type.(*extensions.VariantType); ok {
			if tv := vt.TypedValue().Type; tv != nil && typeHasDecimal128(tv) {
				return true
			}
		}
	}

	return false
}

func typeHasDecimal128(dt arrow.DataType) bool {
	switch t := dt.(type) {
	case *arrow.Decimal128Type:
		return true
	case *arrow.StructType:
		for _, f := range t.Fields() {
			if typeHasDecimal128(f.Type) {
				return true
			}
		}
	case *arrow.ListType:
		return typeHasDecimal128(t.Elem())
	}

	return false
}

func writeArrowProps(mem memory.Allocator) pqarrow.ArrowWriterProperties {
	return pqarrow.NewArrowWriterProperties(pqarrow.WithAllocator(mem), pqarrow.WithStoreSchema())
}

// getWriteProperties requires explicit write properties so misconfigured writer
// plumbing fails fast instead of silently defaulting Parquet settings.
func getWriteProperties(writeProps any, arrowSchema *arrow.Schema, arrProps pqarrow.ArrowWriterProperties) (*parquet.WriterProperties, error) {
	if writeProps == nil {
		return nil, fmt.Errorf("%w: write properties are required", iceberg.ErrInvalidArgument)
	}

	writerProperties, ok := writeProps.([]parquet.WriterProperty)
	if !ok {
		return nil, fmt.Errorf("%w: invalid write properties type %T", iceberg.ErrInvalidArgument, writeProps)
	}

	// Clone: WriteProps is shared across concurrent partition writers; appending in place races.
	wp := slices.Clone(writerProperties)
	// Shredded decimals need the spec's INT32/INT64/FLBA-by-precision types; arrow-go
	// emits those only with StoreDecimalAsInteger, so gate it per file on one being present.
	if arrowSchemaHasShreddedDecimal(arrowSchema) {
		wp = append(wp, parquet.WithStoreDecimalAsInteger(true))
	}

	// Match Iceberg Java: apply parquet-mr's cost-based dictionary fallback to every leaf
	// column so high-cardinality columns fall back to PLAIN rather than keeping a dictionary.
	// arrow-go otherwise enables it only for uncompressed columns, so zstd (our default) would
	// retain dictionaries on all-distinct columns and roughly double their size.
	costFallback, err := dictCostFallbackProps(arrowSchema, wp, arrProps)
	if err != nil {
		return nil, err
	}
	wp = append(wp, costFallback...)

	return parquet.NewWriterProperties(wp...), nil
}

// dictCostFallbackProps returns a WithDictionaryCostFallbackFor(true) property per leaf, walking the arrow schema directly (extensions unwrapped) and falling back to pqarrow.ToParquet for list/map schemas.
func dictCostFallbackProps(arrowSchema *arrow.Schema, base []parquet.WriterProperty, arrProps pqarrow.ArrowWriterProperties) ([]parquet.WriterProperty, error) {
	if schemaHasListOrMap(arrowSchema) {
		return dictCostFallbackViaParquet(arrowSchema, base, arrProps)
	}

	var props []parquet.WriterProperty
	var walk func(prefix string, dt arrow.DataType)
	walk = func(prefix string, dt arrow.DataType) {
		if ext, ok := dt.(arrow.ExtensionType); ok {
			dt = ext.StorageType()
		}
		if d, ok := dt.(*arrow.DictionaryType); ok {
			dt = d.ValueType
		}
		if r, ok := dt.(*arrow.RunEndEncodedType); ok {
			dt = r.Encoded()
		}
		if st, ok := dt.(*arrow.StructType); ok {
			for _, f := range st.Fields() {
				walk(prefix+"."+f.Name, f.Type)
			}

			return
		}
		props = append(props, parquet.WithDictionaryCostFallbackFor(prefix, true))
	}
	for _, f := range arrowSchema.Fields() {
		walk(f.Name, f.Type)
	}

	return props, nil
}

// dictCostFallbackViaParquet is the authoritative path for list/map schemas, whose parquet leaf naming the direct walk does not reproduce.
func dictCostFallbackViaParquet(arrowSchema *arrow.Schema, base []parquet.WriterProperty, arrProps pqarrow.ArrowWriterProperties) ([]parquet.WriterProperty, error) {
	parquetSchema, err := pqarrow.ToParquet(arrowSchema, parquet.NewWriterProperties(base...), arrProps)
	if err != nil {
		return nil, err
	}

	props := make([]parquet.WriterProperty, 0, parquetSchema.NumColumns())
	for i := range parquetSchema.NumColumns() {
		props = append(props, parquet.WithDictionaryCostFallbackFor(parquetSchema.Column(i).Path(), true))
	}

	return props, nil
}

func schemaHasListOrMap(sc *arrow.Schema) bool {
	var has func(dt arrow.DataType) bool
	has = func(dt arrow.DataType) bool {
		if ext, ok := dt.(arrow.ExtensionType); ok {
			dt = ext.StorageType()
		}
		if d, ok := dt.(*arrow.DictionaryType); ok {
			dt = d.ValueType
		}
		if r, ok := dt.(*arrow.RunEndEncodedType); ok {
			dt = r.Encoded()
		}
		switch t := dt.(type) {
		case *arrow.ListType, *arrow.LargeListType, *arrow.FixedSizeListType,
			*arrow.ListViewType, *arrow.LargeListViewType, *arrow.MapType:
			return true
		case *arrow.StructType:
			for _, f := range t.Fields() {
				if has(f.Type) {
					return true
				}
			}
		}

		return false
	}
	for _, f := range sc.Fields() {
		if has(f.Type) {
			return true
		}
	}

	return false
}

// Write appends a record batch to the Parquet file.
func (w *ParquetFileWriter) Write(batch arrow.RecordBatch) error {
	writeBatch, err := w.normalizeGeoBatch(batch)
	if err != nil {
		return err
	}
	if writeBatch != batch {
		defer writeBatch.Release()
	}

	if err := w.accumulateGeoBounds(writeBatch); err != nil {
		return err
	}

	// Rotate before writing the next non-empty batch. Rotating immediately after
	// crossing the target would leave an empty trailing row group when the file
	// is closed without another write.
	if writeBatch.NumRows() > 0 &&
		w.pqWriter.RowGroupTotalBytesWritten() >= w.rowGroupBytes {
		if err := w.pqWriter.NewBufferedRowGroupChecked(); err != nil {
			return err
		}
	}

	return w.pqWriter.WriteBuffered(writeBatch)
}

// wkbStorage is the subset of the binary Arrow arrays that back a geoarrow WKB
// column (Binary and LargeBinary both satisfy it).
type wkbStorage interface {
	arrow.Array
	Value(int) []byte
}

func (w *ParquetFileWriter) normalizeGeoBatch(batch arrow.RecordBatch) (arrow.RecordBatch, error) {
	if len(w.geoNormalizeCols) == 0 {
		return batch, nil
	}

	columns := make(map[int]arrow.Array, len(w.geoNormalizeCols))
	for _, idx := range w.geoNormalizeCols {
		if idx >= int(batch.NumCols()) {
			continue
		}

		normalized, changed, err := normalizeNestedArray(batch.Column(idx), w.mem)
		if err != nil {
			for _, col := range columns {
				col.Release()
			}

			return nil, fmt.Errorf("normalizing geo values for column %d: %w", idx, err)
		}
		if changed {
			columns[idx] = normalized
		}
	}

	if len(columns) == 0 {
		return batch, nil
	}

	resultColumns := slices.Clone(batch.Columns())
	for idx, col := range columns {
		resultColumns[idx] = col
	}
	result := array.NewRecordBatch(batch.Schema(), resultColumns, batch.NumRows())
	for _, col := range columns {
		col.Release()
	}

	return result, nil
}

// normalizeNestedArray walks an Arrow array and replaces only nested WKB values
// that use EWKB encoding. Changed nested containers are rebuilt from their
// logical values with offset zero, so materialized children cannot be sliced a
// second time by an enclosing Arrow container. Recursive calls carry an
// optional reachability mask so values hidden by nullable ancestors are copied
// without being decoded. Such unreachable child slots may retain EWKB or
// malformed bytes, which is safe because they cannot be observed through the
// containing null value.
func normalizeNestedArray(arr arrow.Array, mem memory.Allocator) (arrow.Array, bool, error) {
	return normalizeNestedArrayReachable(arr, nil, mem)
}

func normalizeNestedArrayReachable(arr arrow.Array, active []bool, mem memory.Allocator) (arrow.Array, bool, error) {
	if active != nil && !hasReachableValues(active) {
		return nil, false, nil
	}

	if ext, ok := arr.(array.ExtensionArray); ok {
		if _, ok := ext.ExtensionType().(*geoarrow.WKBType); !ok {
			return nil, false, nil
		}

		return normalizeWKBArrayReachable(ext, active, mem)
	}

	// The table write path currently supplies unencoded nested arrays.
	switch nested := arr.(type) {
	case *array.Struct:
		return normalizeStruct(nested, active, mem)

	case *array.Map:
		return normalizeListChild(nested, active, mem, func(data arrow.ArrayData) arrow.Array {
			return array.NewMapData(data)
		})

	case *array.List:
		return normalizeListChild(nested, active, mem, func(data arrow.ArrayData) arrow.Array {
			return array.NewListData(data)
		})

	case *array.LargeList:
		return normalizeListChild(nested, active, mem, func(data arrow.ArrayData) arrow.Array {
			return array.NewLargeListData(data)
		})

	case *array.FixedSizeList:
		return normalizeListChild(nested, active, mem, func(data arrow.ArrayData) arrow.Array {
			return array.NewFixedSizeListData(data)
		})
	}

	return nil, false, nil
}

func isReachable(active []bool, idx int) bool {
	return active == nil || active[idx]
}

func hasReachableValues(active []bool) bool {
	for _, reachable := range active {
		if reachable {
			return true
		}
	}

	return false
}

func reachableWithValidity(active []bool, arr arrow.Array) []bool {
	if arr.NullN() == 0 {
		return active
	}

	reachable := make([]bool, arr.Len())
	for i := range reachable {
		reachable[i] = isReachable(active, i) && arr.IsValid(i)
	}

	return reachable
}

func normalizeStruct(nested *array.Struct, active []bool, mem memory.Allocator) (arrow.Array, bool, error) {
	children := make([]arrow.ArrayData, nested.NumField())
	changedChildren := make([]arrow.Array, 0, len(children))
	childActive := reachableWithValidity(active, nested)
	for idx := range children {
		field := nested.Field(idx)
		children[idx] = field.Data()
		normalized, changed, err := normalizeNestedArrayReachable(field, childActive, mem)
		if err != nil {
			for _, changedChild := range changedChildren {
				changedChild.Release()
			}

			return nil, false, err
		}
		if changed {
			children[idx] = normalized.Data()
			changedChildren = append(changedChildren, normalized)
		}
	}
	if len(changedChildren) == 0 {
		return nil, false, nil
	}
	defer func() {
		for _, changedChild := range changedChildren {
			changedChild.Release()
		}
	}()

	validity, nulls := rebasedValidity(nested, mem)
	if validity != nil {
		defer validity.Release()
	}
	buffers := slices.Clone(nested.Data().Buffers())
	if len(buffers) == 0 {
		buffers = []*memory.Buffer{validity}
	} else {
		buffers[0] = validity
	}
	data := array.NewData(nested.DataType(), nested.Len(), buffers, children, nulls, 0)
	result := array.NewStructData(data)
	data.Release()

	return result, true, nil
}

func normalizeListChild(list array.ListLike, active []bool, mem memory.Allocator,
	newArray func(arrow.ArrayData) arrow.Array,
) (arrow.Array, bool, error) {
	if list.Len() == 0 {
		return nil, false, nil
	}

	start, _ := list.ValueOffsets(0)
	_, end := list.ValueOffsets(list.Len() - 1)
	if start == end {
		return nil, false, nil
	}

	childActive := make([]bool, int(end-start))
	for i := range list.Len() {
		rowStart, rowEnd := list.ValueOffsets(i)
		if !isReachable(active, i) || list.IsNull(i) {
			continue
		}
		for j := rowStart; j < rowEnd; j++ {
			childActive[int(j-start)] = true
		}
	}
	if !hasReachableValues(childActive) {
		return nil, false, nil
	}

	values := array.NewSlice(list.ListValues(), start, end)
	defer values.Release()
	normalized, changed, err := normalizeNestedArrayReachable(values, childActive, mem)
	if err != nil || !changed {
		return nil, false, err
	}
	defer normalized.Release()

	children := slices.Clone(list.Data().Children())
	children[0] = normalized.Data()

	validity, nulls := rebasedValidity(list, mem)
	if validity != nil {
		defer validity.Release()
	}
	buffers := slices.Clone(list.Data().Buffers())
	if len(buffers) == 0 {
		buffers = []*memory.Buffer{validity}
	}
	buffers[0] = validity
	if list.DataType().ID() != arrow.FIXED_SIZE_LIST {
		offsets, err := rebasedListOffsets(list, start, mem)
		if err != nil {
			return nil, false, err
		}
		defer offsets.Release()
		if len(buffers) < 2 {
			return nil, false, errors.New("list array has no offsets buffer")
		}
		buffers[1] = offsets
	}

	newData := array.NewData(list.DataType(), list.Len(), buffers, children, nulls, 0)
	result := newArray(newData)
	newData.Release()

	return result, true, nil
}

func rebasedValidity(arr arrow.Array, mem memory.Allocator) (*memory.Buffer, int) {
	nulls := arr.NullN()
	if nulls == 0 {
		return nil, 0
	}

	data := mem.Allocate(int(bitutil.BytesForBits(int64(arr.Len()))))
	for i := range data {
		data[i] = 0
	}
	for i := range arr.Len() {
		if arr.IsValid(i) {
			bitutil.SetBit(data, i)
		}
	}

	return memory.NewBufferWithAllocator(data, mem), nulls
}

func rebasedListOffsets(list array.ListLike, first int64, mem memory.Allocator) (*memory.Buffer, error) {
	var (
		data []byte
		set  func(int, int64)
	)
	switch list.DataType().ID() {
	case arrow.LIST, arrow.MAP:
		data = mem.Allocate((list.Len() + 1) * 4)
		offsets := arrow.Int32Traits.CastFromBytes(data)
		set = func(idx int, value int64) {
			offsets[idx] = int32(value)
		}
	case arrow.LARGE_LIST:
		data = mem.Allocate((list.Len() + 1) * 8)
		offsets := arrow.Int64Traits.CastFromBytes(data)
		set = func(idx int, value int64) {
			offsets[idx] = value
		}
	default:
		return nil, fmt.Errorf("unsupported list type %s", list.DataType())
	}

	buffer := memory.NewBufferWithAllocator(data, mem)
	for i := range list.Len() {
		start, end := list.ValueOffsets(i)
		set(i, start-first)
		set(i+1, end-first)
	}

	return buffer, nil
}

func normalizeWKBArray(ext array.ExtensionArray, mem memory.Allocator) (arrow.Array, bool, error) {
	return normalizeWKBArrayReachable(ext, nil, mem)
}

func normalizeWKBArrayReachable(ext array.ExtensionArray, active []bool, mem memory.Allocator) (arrow.Array, bool, error) {
	storage, ok := ext.Storage().(wkbStorage)
	if !ok {
		return nil, false, nil
	}

	needsNormalization := false
	for i := range storage.Len() {
		if isReachable(active, i) && !storage.IsNull(i) && isEWKB(storage.Value(i)) {
			needsNormalization = true

			break
		}
	}
	if !needsNormalization {
		return nil, false, nil
	}

	builder := array.NewBinaryBuilder(mem, storage.DataType().(arrow.BinaryDataType))
	defer builder.Release()

	for i := range storage.Len() {
		if storage.IsNull(i) {
			builder.AppendNull()

			continue
		}
		if !isReachable(active, i) || !isEWKB(storage.Value(i)) {
			builder.Append(storage.Value(i))

			continue
		}

		value, err := normalizeWKB(storage.Value(i))
		if err != nil {
			return nil, false, err
		}
		builder.Append(value)
	}

	normalizedStorage := builder.NewArray()
	normalized := array.NewExtensionArrayWithStorage(ext.ExtensionType(), normalizedStorage)
	normalizedStorage.Release()

	return normalized, true, nil
}

// accumulateGeoBounds extends the per-field bounding boxes with the WKB values
// in this batch. Null rows are skipped; a malformed WKB value fails the write.
func (w *ParquetFileWriter) accumulateGeoBounds(batch arrow.RecordBatch) error {
	for _, gc := range w.geoCols {
		if gc.colIdx >= int(batch.NumCols()) {
			continue
		}
		ext, ok := batch.Column(gc.colIdx).(array.ExtensionArray)
		if !ok {
			continue
		}
		storage, ok := ext.Storage().(wkbStorage)
		if !ok {
			continue
		}

		acc, ok := w.geoAccs[gc.fieldID]
		if !ok {
			continue
		}
		for i := range storage.Len() {
			if storage.IsNull(i) {
				continue
			}
			if err := acc.AddWKB(storage.Value(i)); err != nil {
				return fmt.Errorf("computing geo bounds for field %d: %w", gc.fieldID, err)
			}
		}
	}

	return nil
}

// BytesWritten returns the number of bytes flushed to the output so far.
func (w *ParquetFileWriter) BytesWritten() int64 {
	return w.counter.Count
}

// Close finalizes the Parquet file and returns the resulting DataFile with
// accurate file statistics and size.
func (w *ParquetFileWriter) Close() (_ iceberg.DataFile, err error) {
	defer internal.CheckedClose(w.fileCloser, &err)

	if err = w.pqWriter.Close(); err != nil {
		return nil, err
	}

	filemeta, err := w.pqWriter.FileMetadata()
	if err != nil {
		return nil, err
	}

	stats := w.format.DataFileStatsFromMeta(filemeta, w.info.StatsCols, w.colMapping, w.variantFieldIDs, w.arrowSchema)
	stats.EqualityFieldIDs = w.info.EqualityFieldIDs

	if err = w.applyGeoBounds(stats); err != nil {
		return nil, err
	}

	return stats.ToDataFile(DataFileOpts{
		Schema:          w.info.FileSchema,
		Spec:            w.info.Spec,
		Path:            w.info.FileName,
		Format:          iceberg.ParquetFile,
		Content:         w.info.Content,
		FileSize:        w.counter.Count,
		PartitionValues: w.partition,
		SortOrderID:     w.info.SortOrderID,
	}), nil
}

func (w *ParquetFileWriter) Abort() error {
	closeErr := w.fileCloser.Close()
	removeErr := w.fs.Remove(w.info.FileName)
	if errors.Is(removeErr, fs.ErrNotExist) || os.IsNotExist(removeErr) {
		removeErr = nil
	}

	return errors.Join(closeErr, removeErr)
}

// applyGeoBounds injects the WKB single-point bounds accumulated during the
// write into the file statistics, so they flow through ToDataFile into the
// manifest entry like any other typed bound.
func (w *ParquetFileWriter) applyGeoBounds(stats *DataFileStatistics) error {
	for fieldID, acc := range w.geoAccs {
		// Honor the column's metrics mode: a column the caller never registered
		// (missing key) or one set to counts/none does not record bounds. Check
		// presence first — a missing key yields a zero-value mode of "" that would
		// otherwise fall through and write bounds the caller asked to skip.
		sc, ok := w.info.StatsCols[fieldID]
		if !ok || sc.Mode.Typ == MetricModeNone || sc.Mode.Typ == MetricModeCounts {
			continue
		}

		agg, err := acc.StatsAgg()
		if err != nil {
			return fmt.Errorf("encoding geo bounds for field %d: %w", fieldID, err)
		}
		if agg == nil {
			continue
		}
		if stats.ColAggs == nil {
			stats.ColAggs = make(map[int]StatsAgg)
		}
		stats.ColAggs[fieldID] = agg
	}

	return nil
}

// isGeoType reports whether the Iceberg type is a geometry or geography, whose
// bounds are computed from raw WKB rather than Parquet byte-array statistics.
func isGeoType(t iceberg.PrimitiveType) bool {
	switch t.(type) {
	case iceberg.GeometryType, iceberg.GeographyType:
		return true
	default:
		return false
	}
}

type decAsIntAgg[T int32 | int64] struct {
	*statsAggregator[T]
}

func (s *decAsIntAgg[T]) MinAsBytes() ([]byte, error) {
	if s.curMin == nil {
		return nil, nil
	}

	lit := iceberg.DecimalLiteral(iceberg.Decimal{
		Val:   decimal128.FromI64(int64(s.curMin.Value())),
		Scale: s.primitiveType.(iceberg.DecimalType).Scale(),
	})
	if s.truncLen > 0 {
		return s.toBytes((&iceberg.TruncateTransform{Width: s.truncLen}).
			Apply(iceberg.Optional[iceberg.Literal]{Valid: true, Val: lit}).Val)
	}

	return s.toBytes(lit)
}

func (s *decAsIntAgg[T]) MaxAsBytes() ([]byte, error) {
	if s.curMax == nil {
		return nil, nil
	}

	lit := iceberg.DecimalLiteral(iceberg.Decimal{
		Val:   decimal128.FromI64(int64(s.curMax.Value())),
		Scale: s.primitiveType.(iceberg.DecimalType).Scale(),
	})
	if s.truncLen <= 0 {
		return s.toBytes(lit)
	}

	return nil, fmt.Errorf("%s cannot be truncated for upper bound", s.primitiveType)
}

type wrappedBinaryStats struct {
	*metadata.ByteArrayStatistics
}

func (w *wrappedBinaryStats) Min() []byte {
	return w.ByteArrayStatistics.Min()
}

func (w *wrappedBinaryStats) Max() []byte {
	return w.ByteArrayStatistics.Max()
}

type wrappedStringStats struct {
	*metadata.ByteArrayStatistics
}

func (w *wrappedStringStats) Min() string {
	data := w.ByteArrayStatistics.Min()

	return unsafe.String(unsafe.SliceData(data), len(data))
}

func (w *wrappedStringStats) Max() string {
	data := w.ByteArrayStatistics.Max()

	return unsafe.String(unsafe.SliceData(data), len(data))
}

type wrappedUUIDStats struct {
	*metadata.FixedLenByteArrayStatistics
}

func (w *wrappedUUIDStats) HasMinMax() bool {
	const uuidByteWidth = 16

	return w.FixedLenByteArrayStatistics.HasMinMax() &&
		len(w.FixedLenByteArrayStatistics.Min()) == uuidByteWidth &&
		len(w.FixedLenByteArrayStatistics.Max()) == uuidByteWidth
}

func (w *wrappedUUIDStats) Min() uuid.UUID {
	if !w.HasMinMax() {
		return uuid.Nil
	}

	uid, err := uuid.FromBytes(w.FixedLenByteArrayStatistics.Min())
	if err != nil {
		panic(err)
	}

	return uid
}

func (w *wrappedUUIDStats) Max() uuid.UUID {
	if !w.HasMinMax() {
		return uuid.Nil
	}

	uid, err := uuid.FromBytes(w.FixedLenByteArrayStatistics.Max())
	if err != nil {
		panic(err)
	}

	return uid
}

type wrappedFLBAStats struct {
	*metadata.FixedLenByteArrayStatistics
	expectedLen int
}

func (w *wrappedFLBAStats) HasMinMax() bool {
	return w.FixedLenByteArrayStatistics.HasMinMax() &&
		len(w.FixedLenByteArrayStatistics.Min()) == w.expectedLen &&
		len(w.FixedLenByteArrayStatistics.Max()) == w.expectedLen
}

func (w *wrappedFLBAStats) Min() []byte {
	return w.FixedLenByteArrayStatistics.Min()
}

func (w *wrappedFLBAStats) Max() []byte {
	return w.FixedLenByteArrayStatistics.Max()
}

type wrappedDecStats struct {
	*metadata.FixedLenByteArrayStatistics
	expectedLen int
	scale       int
}

func (w wrappedDecStats) HasMinMax() bool {
	return w.FixedLenByteArrayStatistics.HasMinMax() &&
		len(w.FixedLenByteArrayStatistics.Min()) == w.expectedLen &&
		len(w.FixedLenByteArrayStatistics.Max()) == w.expectedLen
}

func (w wrappedDecStats) Min() iceberg.Decimal {
	if !w.HasMinMax() {
		return iceberg.Decimal{Scale: w.scale}
	}

	dec, err := BigEndianToDecimal(w.FixedLenByteArrayStatistics.Min())
	if err != nil {
		panic(err)
	}

	return iceberg.Decimal{Val: dec, Scale: w.scale}
}

func (w wrappedDecStats) Max() iceberg.Decimal {
	if !w.HasMinMax() {
		return iceberg.Decimal{Scale: w.scale}
	}

	dec, err := BigEndianToDecimal(w.FixedLenByteArrayStatistics.Max())
	if err != nil {
		panic(err)
	}

	return iceberg.Decimal{Val: dec, Scale: w.scale}
}

type wrappedDecByteArrayStats struct {
	*metadata.ByteArrayStatistics
	scale int
}

func (w wrappedDecByteArrayStats) Min() iceberg.Decimal {
	dec, err := BigEndianToDecimal(w.ByteArrayStatistics.Min())
	if err != nil {
		panic(err)
	}

	return iceberg.Decimal{Val: dec, Scale: w.scale}
}

func (w wrappedDecByteArrayStats) Max() iceberg.Decimal {
	dec, err := BigEndianToDecimal(w.ByteArrayStatistics.Max())
	if err != nil {
		panic(err)
	}

	return iceberg.Decimal{Val: dec, Scale: w.scale}
}

// parquetStatsColumn contains the immutable per-file resolution needed to
// interpret a physical Parquet column's statistics. The row-group metadata
// still carries the values that are accumulated, but the path-to-field lookup
// is independent of the row group and is done once from the resolved schema.
type parquetStatsColumn struct {
	path         string
	fieldID      int
	statsCol     StatisticsCollector
	variantChild bool
	skipStats    bool
	resolveErr   error
}

func resolveParquetStatsColumns(meta *metadata.FileMetaData, statsCols map[int]StatisticsCollector, colMapping map[string]int, variantFieldIDs map[int]struct{}) []parquetStatsColumn {
	columns := make([]parquetStatsColumn, meta.Schema.NumColumns())
	for pos := range meta.Schema.NumColumns() {
		column := meta.Schema.Column(pos)
		columnPath := column.ColumnPath()
		descriptor := parquetStatsColumn{path: column.Path()}

		fieldID, ok := colMapping[descriptor.path]
		if !ok {
			// Variant sub-columns (metadata, value, typed_value children) have
			// no Iceberg field ID (spec: "must not be assigned field IDs").
			// Walk up the resolved path once to find a variant ancestor.
			for depth := len(columnPath) - 1; depth >= 1; depth-- {
				ancestorPath := strings.Join(columnPath[:depth], ".")
				if ancestorID, hasAncestor := colMapping[ancestorPath]; hasAncestor {
					_, descriptor.variantChild = variantFieldIDs[ancestorID]

					break
				}
			}
			if !descriptor.variantChild {
				descriptor.resolveErr = fmt.Errorf("column chunk %q not found in column mapping", columnPath)
			}

			columns[pos] = descriptor

			continue
		}

		descriptor.fieldID = fieldID
		statsCol, ok := statsCols[fieldID]
		if !ok {
			// Only the reserved row-lineage metadata columns (_row_id,
			// _last_updated_sequence_number) may legitimately appear in a file
			// without a metrics-plan entry. Writers materialize them without
			// adding them to the table schema, so skip them entirely.
			if iceberg.IsMetadataColumn(fieldID) {
				descriptor.skipStats = true
			} else {
				descriptor.resolveErr = fmt.Errorf("field id %d (column %q) not found in the metrics plan", fieldID, descriptor.path)
			}
		} else {
			descriptor.statsCol = statsCol
		}
		columns[pos] = descriptor
	}

	return columns
}

func (p parquetFormat) DataFileStatsFromMeta(meta Metadata, statsCols map[int]StatisticsCollector, colMapping map[string]int, variantFieldIDs map[int]struct{}, arrowSchema *arrow.Schema) *DataFileStatistics {
	pqmeta := meta.(*metadata.FileMetaData)
	columns := resolveParquetStatsColumns(pqmeta, statsCols, colMapping, variantFieldIDs)
	var (
		colSizes        = make(map[int]int64)
		valueCounts     = make(map[int]int64)
		splitOffsets    = make([]int64, 0)
		nullValueCounts = make(map[int]int64)
		nanValueCounts  = make(map[int]int64)
		invalidateCol   = make(map[int]struct{})
		colAggs         = make(map[int]StatsAgg)
	)

	for rg := range pqmeta.NumRowGroups() {
		// reference: https://github.com/apache/iceberg-python/blob/main/pyiceberg/io/pyarrow.py#L2285
		rowGroup := pqmeta.RowGroup(rg)
		colChunk, err := rowGroup.ColumnChunk(0)
		if err != nil {
			panic(err)
		}

		dataOffset, dictOffset := colChunk.DataPageOffset(), colChunk.DictionaryPageOffset()
		if colChunk.HasDictionaryPage() && dictOffset < dataOffset {
			splitOffsets = append(splitOffsets, dictOffset)
		} else {
			splitOffsets = append(splitOffsets, dataOffset)
		}

		for pos := range rowGroup.NumColumns() {
			column := columns[pos]
			if column.resolveErr != nil {
				if _, err = rowGroup.ColumnChunk(pos); err != nil {
					panic(err)
				}

				panic(column.resolveErr)
			}
			if column.variantChild || column.skipStats || column.statsCol.Mode.Typ == MetricModeNone {
				continue
			}

			fieldID, statsCol := column.fieldID, column.statsCol
			colChunk, err = rowGroup.ColumnChunk(pos)
			if err != nil {
				panic(err)
			}

			if _, invalid := invalidateCol[fieldID]; invalid {
				continue
			}

			colSizes[fieldID] += colChunk.TotalCompressedSize()
			valueCounts[fieldID] += colChunk.NumValues()
			set, err := colChunk.StatsSet()
			if err != nil {
				panic(err)
			}

			if !set {
				invalidateCol[fieldID] = struct{}{}

				continue
			}

			stats, err := colChunk.Statistics()
			if err != nil || stats == nil {
				invalidateCol[fieldID] = struct{}{}

				continue
			}

			if stats.HasNullCount() {
				nullValueCounts[fieldID] += stats.NullCount()
			}

			if statsCol.Mode.Typ == MetricModeCounts || !stats.HasMinMax() {
				continue
			}

			// Geometry/Geography bounds are computed from raw WKB during the
			// write (Parquet byte-array min/max over WKB are meaningless) and
			// injected separately, so skip the generic min/max aggregator here
			// while still keeping the column's counts, sizes, and null counts.
			if isGeoType(statsCol.IcebergTyp) {
				continue
			}

			agg, ok := colAggs[fieldID]
			if !ok {
				agg, err = p.createStatsAgg(statsCol.IcebergTyp, stats.Type().String(), statsCol.Mode.Len)
				if err != nil {
					panic(err)
				}

				colAggs[fieldID] = agg
			}

			stats = wrapStatsForType(stats, statsCol.IcebergTyp)

			if !stats.HasMinMax() {
				invalidateCol[fieldID] = struct{}{}
				delete(colAggs, fieldID)

				continue
			}

			agg.Update(stats)
		}

	}

	slices.Sort(splitOffsets)
	maps.DeleteFunc(nullValueCounts, func(fieldID int, _ int64) bool {
		_, ok := invalidateCol[fieldID]

		return ok
	})
	maps.DeleteFunc(colAggs, func(fieldID int, _ StatsAgg) bool {
		_, ok := invalidateCol[fieldID]

		return ok
	})

	vlo, vup := p.collectVariantBoundsWithColumns(pqmeta, arrowSchema, colMapping, statsCols, columns)

	return &DataFileStatistics{
		RecordCount:        pqmeta.GetNumRows(),
		ColSizes:           colSizes,
		ValueCounts:        valueCounts,
		NullValueCounts:    nullValueCounts,
		NanValueCounts:     nanValueCounts,
		SplitOffsets:       splitOffsets,
		ColAggs:            colAggs,
		VariantLowerBounds: vlo,
		VariantUpperBounds: vup,
	}
}

// wrapStatsForType adapts Parquet stats so a StatsAgg reads Iceberg-typed min/max.
func wrapStatsForType(stats metadata.TypedStatistics, typ iceberg.PrimitiveType) metadata.TypedStatistics {
	switch t := typ.(type) {
	case iceberg.BinaryType:
		return &wrappedBinaryStats{stats.(*metadata.ByteArrayStatistics)}
	case iceberg.UUIDType:
		return &wrappedUUIDStats{stats.(*metadata.FixedLenByteArrayStatistics)}
	case iceberg.StringType:
		return &wrappedStringStats{stats.(*metadata.ByteArrayStatistics)}
	case iceberg.FixedType:
		return &wrappedFLBAStats{
			FixedLenByteArrayStatistics: stats.(*metadata.FixedLenByteArrayStatistics),
			expectedLen:                 t.Len(),
		}
	case iceberg.DecimalType:
		switch s := stats.(type) {
		case *metadata.FixedLenByteArrayStatistics:
			return &wrappedDecStats{
				FixedLenByteArrayStatistics: s,
				expectedLen:                 internal.DecimalRequiredBytes(t.Precision()),
				scale:                       t.Scale(),
			}
		case *metadata.ByteArrayStatistics:
			return &wrappedDecByteArrayStats{s, t.Scale()}
		}
	}

	return stats
}

// residualAllVariantNull reports whether a residual value column holds only variant-null.
func residualAllVariantNull(st metadata.TypedStatistics) bool {
	ba, ok := st.(*metadata.ByteArrayStatistics)
	if !ok {
		return false
	}

	return isVariantNull(ba.Min()) && isVariantNull(ba.Max())
}

// isVariantNull reports whether a serialized variant value is primitive null (byte 0x00).
func isVariantNull(b []byte) bool {
	return len(b) == 1 && b[0] == 0x00
}

// collectVariantBounds builds spec "Bounds for Variant" objects per shredded variant column.
func (p parquetFormat) collectVariantBounds(meta *metadata.FileMetaData, arrowSchema *arrow.Schema, colMapping map[string]int, statsCols map[int]StatisticsCollector) (lower, upper map[int][]byte) {
	if arrowSchema == nil {
		return nil, nil
	}

	columns := resolveParquetStatsColumns(meta, statsCols, colMapping, nil)

	return p.collectVariantBoundsWithColumns(meta, arrowSchema, colMapping, statsCols, columns)
}

func (p parquetFormat) collectVariantBoundsWithColumns(meta *metadata.FileMetaData, arrowSchema *arrow.Schema, colMapping map[string]int, statsCols map[int]StatisticsCollector, columns []parquetStatsColumn) (lower, upper map[int][]byte) {
	if arrowSchema == nil {
		return nil, nil
	}

	type leafAgg struct {
		leaf     variantLeaf
		parentID int
		agg      StatsAgg
		invalid  bool
	}

	byTyped := make(map[string]*leafAgg)
	residualParent := make(map[string]struct{}) // residual value columns to inspect
	residualBad := make(map[string]bool)        // residual value column has/may have non-null

	for _, f := range arrowSchema.Fields() {
		vt, ok := f.Type.(*extensions.VariantType)
		if !ok || vt.TypedValue().Type == nil {
			continue
		}
		parentID, ok := colMapping[f.Name]
		if !ok {
			continue
		}
		// Honor the parent variant column's metrics mode: none/counts skip min/max bounds.
		if sc, ok := statsCols[parentID]; !ok || sc.Mode.Typ == MetricModeNone || sc.Mode.Typ == MetricModeCounts {
			continue
		}
		for _, lf := range enumerateVariantLeaves([]string{f.Name}, vt.TypedValue()) {
			byTyped[lf.typedPath] = &leafAgg{leaf: lf, parentID: parentID}
			if lf.ownResidual != "" {
				residualParent[lf.ownResidual] = struct{}{}
			}
		}
	}
	if len(byTyped) == 0 {
		return nil, nil
	}

	for rg := range meta.NumRowGroups() {
		rowGroup := meta.RowGroup(rg)
		for pos := range rowGroup.NumColumns() {
			cc, err := rowGroup.ColumnChunk(pos)
			if err != nil {
				continue
			}
			path := columns[pos].path

			if _, isResidual := residualParent[path]; isResidual {
				// Non-null residual entries invalidate the bound unless they are all variant-null.
				set, serr := cc.StatsSet()
				st, terr := cc.Statistics()
				switch {
				case serr != nil || !set || terr != nil || st == nil || !st.HasNullCount():
					residualBad[path] = true
				case cc.NumValues()-st.NullCount() > 0:
					if !st.HasMinMax() || !residualAllVariantNull(st) {
						residualBad[path] = true
					}
				}

				continue
			}

			e, isTyped := byTyped[path]
			if !isTyped || e.invalid {
				continue
			}
			set, serr := cc.StatsSet()
			st, terr := cc.Statistics()
			switch {
			case serr != nil || !set || terr != nil || st == nil:
				e.invalid = true
			case st.HasMinMax():
				if e.agg == nil {
					agg, aerr := p.createStatsAgg(e.leaf.icebergType, st.Type().String(), 0)
					if aerr != nil {
						slog.Warn("variant bounds: no stats aggregator for shredded field", "path", e.leaf.jsonPath, "err", aerr)
						e.invalid = true

						break
					}
					e.agg = agg
				}
				e.agg.Update(wrapStatsForType(st, e.leaf.icebergType))
			case !st.HasNullCount() || cc.NumValues()-st.NullCount() > 0:
				e.invalid = true
			}
		}
	}

	perParent := make(map[int][]variantFieldBound)
	for _, e := range byTyped {
		if e.invalid || e.agg == nil {
			continue
		}
		if e.leaf.ownResidual != "" && residualBad[e.leaf.ownResidual] {
			continue
		}
		lo, hi := e.agg.Min(), e.agg.Max()
		if lo == nil || hi == nil || isNaNLiteral(lo) || isNaNLiteral(hi) {
			continue
		}
		lo, hi = truncateVariantBound(e.leaf.icebergType, lo, hi, statsCols[e.parentID].Mode.Len)
		perParent[e.parentID] = append(perParent[e.parentID], variantFieldBound{
			jsonPath:    e.leaf.jsonPath,
			icebergType: e.leaf.icebergType,
			lower:       lo,
			upper:       hi,
		})
	}
	if len(perParent) == 0 {
		return nil, nil
	}

	lower = make(map[int][]byte)
	upper = make(map[int][]byte)
	for parentID, fields := range perParent {
		lo, up, ok, err := serializeVariantBounds(fields)
		if err != nil {
			slog.Warn("variant bounds serialization skipped", "fieldID", parentID, "err", err)

			continue
		}
		if !ok {
			continue
		}
		lower[parentID] = lo
		upper[parentID] = up
	}

	return lower, upper
}

type ParquetFileSource struct {
	mem  memory.Allocator
	fs   iceio.IO
	file iceberg.DataFile
}

// RowGroupBloomPred holds the physical-encoded bytes for each literal in a
// bloom-filterable predicate on one field. A row group can be skipped when
// NONE of the bytes appear in the column's bloom filter.
type RowGroupBloomPred struct {
	FieldID   int
	PhysBytes [][]byte // one entry for EqualTo; one per value for In
}

// RowGroupDictionaryPred holds the physical-encoded bytes for each literal in
// a dictionary-prunable predicate on one field. A row group can be skipped
// when NONE of the bytes occur in its complete dictionary. It intentionally
// mirrors RowGroupBloomPred because both predicates use the same physical
// literal encoding.
type RowGroupDictionaryPred struct {
	FieldID   int
	PhysBytes [][]byte // one entry for EqualTo; one per value for In
}

// ParquetRowGroupTester combines stats-based, dictionary, and bloom filter row
// group pruning.
// Pass it as the tester argument to wrapPqArrowReader.GetRecords.
type ParquetRowGroupTester struct {
	StatsFn         func(*metadata.RowGroupMetaData, []int) (bool, error)
	BloomPreds      []RowGroupBloomPred      // nil = no bloom filter pass
	DictionaryPreds []RowGroupDictionaryPred // nil = no dictionary filter pass
	// RangeSet indicates that Start and Length came from an explicit scan task.
	// Without it, the zero value keeps the historical full-file behavior for
	// callers that construct a tester only for row-group pruning.
	RangeSet bool
	// When RangeSet is true, Start and Length select row groups whose absolute
	// Parquet row-group offset falls in [Start, Start+Length).
	Start, Length int64
	// PlanningSplitOffsets contains the split offsets used when the scan task
	// was planned. When it contains one valid offset per row group, range
	// selection uses these offsets instead of re-deriving them from the opened
	// Parquet footer. Sparse split-offset lists fall back to the footer offsets.
	PlanningSplitOffsets []int64
	// Survivors, if non-nil, is reset and then filled with one span per row group
	// that survives pruning, in file order, with positions relative to the full
	// file. It lets callers reconstruct each emitted row's original position even
	// when pruning skips groups. Set to nil when no group is skipped.
	Survivors *[]RowGroupSpan
}

// RowGroupSpan locates a surviving row group within the data file: the absolute
// position of its first row and the number of rows it holds.
type RowGroupSpan struct {
	FirstRowPos int64
	NumRows     int64
}

type wrapPqArrowReader struct {
	*pqarrow.FileReader
	rowGroupInfos []parquetRowGroupInfo
	// dictionarySource is shared with FileReader. The temporary reader created
	// from it wraps the source so it cannot close the main reader's input.
	dictionarySource parquet.ReaderAtSeeker
	dictionaryMem    memory.Allocator
}

func (w wrapPqArrowReader) Metadata() Metadata {
	return w.ParquetReader().MetaData()
}

func (w wrapPqArrowReader) SourceFileSize() int64 {
	return w.ParquetReader().MetaData().GetSourceFileSize()
}

func (w wrapPqArrowReader) Close() error {
	return w.ParquetReader().Close()
}

func (w wrapPqArrowReader) PrunedSchema(projectedIDs map[int]struct{}, mapping iceberg.NameMapping) (*arrow.Schema, []int, error) {
	return pruneParquetColumns(w.Manifest, projectedIDs, false, mapping)
}

// parquetRowGroupSplitOffset returns the same first-page offset used when
// DataFileStatistics.SplitOffsets is built. RowGroup.file_offset is optional
// in Parquet and Arrow returns zero when it is absent, so relying on it can
// discard every row group in a split task from an otherwise valid file.
func parquetRowGroupSplitOffset(rgMeta *metadata.RowGroupMetaData) int64 {
	if rgMeta.NumColumns() > 0 {
		if firstColumn, err := rgMeta.ColumnChunk(0); err == nil {
			dataOffset := firstColumn.DataPageOffset()
			if firstColumn.HasDictionaryPage() {
				dictOffset := firstColumn.DictionaryPageOffset()
				if dictOffset >= 0 && (dataOffset < 0 || dictOffset < dataOffset) {
					return dictOffset
				}
			}
			if dataOffset >= 0 {
				return dataOffset
			}
		}
	}

	return rgMeta.FileOffset()
}

func parquetRowGroupInfos(rdr *file.Reader) []parquetRowGroupInfo {
	result := make([]parquetRowGroupInfo, rdr.NumRowGroups())
	for i := range result {
		rgMeta := rdr.MetaData().RowGroup(i)
		result[i] = parquetRowGroupInfo{
			splitOffset: parquetRowGroupSplitOffset(rgMeta),
			numRows:     rgMeta.NumRows(),
		}
	}

	return result
}

func validateParquetRowGroupRange(fileSize, start, length int64) error {
	if start < 0 || length <= 0 || start > fileSize || length > fileSize-start {
		return fmt.Errorf(
			"%w: invalid parquet row-group range: start=%d, length=%d, file size=%d",
			iceberg.ErrInvalidArgument, start, length, fileSize)
	}

	return nil
}

func validParquetRowGroupOffsets(offsets []int64, numRowGroups int, fileSize int64) bool {
	if len(offsets) != numRowGroups {
		return false
	}

	for i, offset := range offsets {
		if offset < 0 || offset >= fileSize || (i > 0 && offset <= offsets[i-1]) {
			return false
		}
	}

	return true
}

func (w wrapPqArrowReader) GetRecords(ctx context.Context, cols []int, tester any) (array.RecordReader, error) {
	var rowGroupTester *ParquetRowGroupTester

	if tester != nil {
		var ok bool
		rowGroupTester, ok = tester.(*ParquetRowGroupTester)
		if !ok {
			return nil, fmt.Errorf("%w: invalid tester type", iceberg.ErrInvalidArgument)
		}
	}

	rangeSet := rowGroupTester != nil && rowGroupTester.RangeSet
	if rangeSet {
		if err := validateParquetRowGroupRange(
			w.SourceFileSize(), rowGroupTester.Start, rowGroupTester.Length); err != nil {
			return nil, err
		}
	}

	var rgList []int
	if rowGroupTester != nil && (rangeSet ||
		rowGroupTester.StatsFn != nil || len(rowGroupTester.DictionaryPreds) > 0 ||
		len(rowGroupTester.BloomPreds) > 0) {
		fileMeta := w.ParquetReader().MetaData()
		numRg := w.ParquetReader().NumRowGroups()
		planningOffsets := rowGroupTester.PlanningSplitOffsets
		if !validParquetRowGroupOffsets(planningOffsets, numRg, w.SourceFileSize()) {
			planningOffsets = nil
		}

		var (
			fieldIDToColIdx  map[int]int
			bfReader         *metadata.BloomFilterReader
			dictionaryReader *file.Reader
		)

		if len(rowGroupTester.BloomPreds) > 0 || len(rowGroupTester.DictionaryPreds) > 0 {
			fieldIDToColIdx = buildFieldIDToColIdx(fileMeta)
		}
		var dictionaryPredsByColumn map[int][]int
		if len(rowGroupTester.DictionaryPreds) > 0 {
			dictionaryPredsByColumn = groupRowGroupDictionaryPredicates(
				fieldIDToColIdx, rowGroupTester.DictionaryPreds)
			if len(dictionaryPredsByColumn) > 0 && w.dictionarySource != nil {
				// Dictionary inspection is an optional optimisation. If the
				// section-read reader cannot be created, use the main reader and
				// keep the existing conservative behaviour.
				dictionaryReader, _ = newParquetDictionaryReader(
					w.dictionarySource, fileMeta, w.dictionaryMem)
				if dictionaryReader != nil {
					// The temporary reader owns only its buffers. Its source is
					// wrapped so this cannot close the main reader's input.
					defer dictionaryReader.Close()
				}
			}
			if len(dictionaryPredsByColumn) > 0 && dictionaryReader == nil {
				dictionaryReader = w.ParquetReader()
			}
		}
		if len(rowGroupTester.BloomPreds) > 0 {
			bfReader = w.ParquetReader().GetBloomFilterReader()
		}

		if rowGroupTester.Survivors != nil {
			*rowGroupTester.Survivors = (*rowGroupTester.Survivors)[:0]
		}

		rangeStart := rowGroupTester.Start
		rangeEnd := rangeStart + rowGroupTester.Length
		rgList = make([]int, 0)
		var firstRowPos int64
		for rg := range numRg {
			pos := firstRowPos
			var (
				rgMeta  *metadata.RowGroupMetaData
				numRows int64
				offset  int64
			)
			if len(w.rowGroupInfos) == numRg {
				info := w.rowGroupInfos[rg]
				numRows = info.numRows
				offset = info.splitOffset
			} else {
				rgMeta = fileMeta.RowGroup(rg)
				numRows = rgMeta.NumRows()
			}
			firstRowPos += numRows

			use := true
			if rangeSet {
				if planningOffsets != nil {
					offset = planningOffsets[rg]
				} else if len(w.rowGroupInfos) != numRg {
					offset = parquetRowGroupSplitOffset(rgMeta)
				}
				use = offset >= rangeStart && offset < rangeEnd
			}
			if use && (rowGroupTester.StatsFn != nil || bfReader != nil) && rgMeta == nil {
				rgMeta = fileMeta.RowGroup(rg)
			}
			if use && rowGroupTester.StatsFn != nil {
				var err error
				use, err = rowGroupTester.StatsFn(rgMeta, cols)
				if err != nil {
					return nil, err
				}
			}

			if use && len(dictionaryPredsByColumn) > 0 {
				if rgMeta == nil {
					rgMeta = fileMeta.RowGroup(rg)
				}
				use = checkRowGroupDictionaries(
					dictionaryReader, fileMeta, rgMeta, rg,
					dictionaryPredsByColumn, rowGroupTester.DictionaryPreds)
			}

			if use && bfReader != nil {
				var err error
				use, err = checkRowGroupBloomFilters(bfReader, rg, fieldIDToColIdx, rowGroupTester.BloomPreds)
				if err != nil {
					return nil, err
				}
			}

			if !use {
				continue
			}

			rgList = append(rgList, rg)
			if rowGroupTester.Survivors != nil {
				*rowGroupTester.Survivors = append(*rowGroupTester.Survivors,
					RowGroupSpan{FirstRowPos: pos, NumRows: numRows})
			}
		}

		// No group was skipped: emitted rows are contiguous from zero, so drop
		// the spans and let position cursors take their contiguous fast path.
		if rowGroupTester.Survivors != nil && len(rgList) == numRg {
			*rowGroupTester.Survivors = nil
		}
	}

	return w.GetRecordReader(ctx, cols, rgList)
}

// nonClosingReaderAtSeeker gives a temporary Parquet reader access to a shared
// source without allowing that reader's Close method to close the source owned
// by the main Arrow reader.
type nonClosingReaderAtSeeker struct {
	parquet.ReaderAtSeeker
}

func (nonClosingReaderAtSeeker) Close() error {
	return nil
}

func newParquetDictionaryReader(
	source parquet.ReaderAtSeeker, fileMeta *metadata.FileMetaData, mem memory.Allocator,
) (*file.Reader, error) {
	readProps := parquet.NewReaderProperties(mem)
	readProps.BufferedStreamEnabled = true

	return file.NewParquetReader(nonClosingReaderAtSeeker{ReaderAtSeeker: source},
		file.WithMetadata(fileMeta), file.WithReadProps(readProps))
}

func groupRowGroupDictionaryPredicates(
	fieldIDToColIdx map[int]int,
	preds []RowGroupDictionaryPred,
) map[int][]int {
	predsByColumn := make(map[int][]int)
	for i, pred := range preds {
		if len(pred.PhysBytes) == 0 {
			continue
		}

		colIdx, ok := fieldIDToColIdx[pred.FieldID]
		if ok {
			predsByColumn[colIdx] = append(predsByColumn[colIdx], i)
		}
	}

	return predsByColumn
}

// checkRowGroupDictionaries checks each dictionary predicate against the
// complete dictionary for its column in row group rg. It returns false only
// when the column metadata proves that every data page is dictionary encoded
// and none of the predicate values occur in the dictionary. Missing encoding
// metadata, dictionary pages, unsupported physical types, malformed pages, or
// reader errors keep the row group so this optimisation cannot drop data.
func checkRowGroupDictionaries(
	rdr *file.Reader,
	fileMeta *metadata.FileMetaData,
	rgMeta *metadata.RowGroupMetaData,
	rg int,
	predsByColumn map[int][]int,
	preds []RowGroupDictionaryPred,
) bool {
	if len(preds) == 0 || len(predsByColumn) == 0 {
		return true
	}

	rgReader := rdr.RowGroup(rg)
	for colIdx, predIndexes := range predsByColumn {
		chunk, err := rgMeta.ColumnChunk(colIdx)
		if err != nil || chunk == nil || !parquetColumnUsesOnlyDictionaryData(chunk) {
			continue
		}

		column := fileMeta.Schema.Column(colIdx)
		pageRdr, err := rgReader.GetColumnPageReader(colIdx)
		if err != nil {
			continue
		}

		dictPage, dictErr := pageRdr.GetDictionaryPage()
		if dictErr != nil || dictPage == nil {
			if dictPage != nil {
				dictPage.Release()
			}
			_ = pageRdr.Close()

			continue
		}

		matches, known := dictionaryMatchesPredicates(
			dictPage, column.PhysicalType(), column.TypeLength(), preds, predIndexes)
		dictPage.Release()
		closeErr := pageRdr.Close()
		if !known {
			continue
		}
		if closeErr != nil {
			// A close error makes an otherwise conclusive dictionary result
			// unusable, so keep the row group conservatively.
			slog.Warn("dictionary row-group pruning skipped after page reader close error",
				"rowGroup", rg, "err", closeErr)

			continue
		}

		if !matches {
			return false
		}
	}

	return true
}

// parquetColumnUsesOnlyDictionaryData verifies that the optional encoding
// statistics identify a dictionary page and dictionary-encoded data pages,
// with no PLAIN fallback pages. EncodingStats is deliberately required: the
// column encoding list cannot distinguish the PLAIN dictionary page itself
// from a later PLAIN data page in older files.
func parquetColumnUsesOnlyDictionaryData(chunk *metadata.ColumnChunkMetaData) bool {
	if !chunk.HasDictionaryPage() {
		return false
	}

	stats := chunk.EncodingStats()
	if len(stats) == 0 {
		return false
	}

	var hasDictionaryPage, hasDictionaryData bool
	for _, stat := range stats {
		switch stat.PageType {
		case file.PageTypeDictionaryPage:
			if stat.Encoding != parquet.Encodings.Plain && stat.Encoding != parquet.Encodings.PlainDict {
				return false
			}
			hasDictionaryPage = true
		case file.PageTypeDataPage, file.PageTypeDataPageV2:
			if stat.Encoding != parquet.Encodings.RLEDict && stat.Encoding != parquet.Encodings.PlainDict {
				return false
			}
			hasDictionaryData = true
		default:
			return false
		}
	}

	return hasDictionaryPage && hasDictionaryData
}

// dictionaryMatchesPredicates decodes a PLAIN dictionary page and reports
// whether every requested predicate has at least one matching value. The
// returned known flag is false for unsupported or malformed input that
// prevents a pruning decision, which means the caller must retain the row
// group.
func dictionaryMatchesPredicates(
	page *file.DictionaryPage,
	physicalType parquet.Type,
	typeLen int,
	preds []RowGroupDictionaryPred,
	predIndexes []int,
) (bool, bool) {
	pageEncoding := parquet.Encoding(page.Encoding())
	if pageEncoding != parquet.Encodings.Plain && pageEncoding != parquet.Encodings.PlainDict {
		return false, false
	}

	width, variableWidth, ok := parquetDictionaryValueLayout(physicalType, typeLen)
	if !ok {
		return false, false
	}

	for _, predIndex := range predIndexes {
		usable := false
		for _, candidate := range preds[predIndex].PhysBytes {
			if variableWidth || len(candidate) == width {
				usable = true

				break
			}
		}
		if !usable {
			return false, false
		}
	}

	data := page.Data()
	offset := 0
	numValues := page.NumValues()
	if numValues < 0 {
		return false, false
	}
	if numValues == 0 {
		return false, len(data) == 0
	}

	if len(predIndexes) == 1 {
		pred := preds[predIndexes[0]]
		candidateSet := parquetDictionaryCandidateSet(
			physicalType, width, variableWidth, numValues, pred.PhysBytes)
		for range numValues {
			value, next, ok := nextParquetDictionaryValue(data, offset, physicalType, width)
			if !ok {
				return false, false
			}
			offset = next

			if candidateSet != nil {
				if _, ok := candidateSet[string(value)]; ok {
					return true, true
				}
			} else {
				for _, candidate := range pred.PhysBytes {
					if parquetDictionaryValueEqual(physicalType, value, candidate) {
						return true, true
					}
				}
			}
		}

		if offset != len(data) {
			return false, false
		}

		return false, true
	}

	// predIndexes contains indexes into the full preds slice. Keep matches
	// aligned with predIndexes so columns with a small predicate subset do not
	// allocate a slice for predicates belonging to other columns.
	matches := make([]bool, len(predIndexes))
	candidateSets := make([]map[string]struct{}, len(predIndexes))
	for matchIndex, predIndex := range predIndexes {
		candidateSets[matchIndex] = parquetDictionaryCandidateSet(
			physicalType, width, variableWidth, numValues, preds[predIndex].PhysBytes)
	}
	remaining := len(predIndexes)
	for range numValues {
		if remaining == 0 {
			break
		}

		value, next, ok := nextParquetDictionaryValue(data, offset, physicalType, width)
		if !ok {
			return false, false
		}
		offset = next

		for matchIndex, predIndex := range predIndexes {
			if matches[matchIndex] {
				continue
			}

			if candidateSets[matchIndex] != nil {
				if _, ok := candidateSets[matchIndex][string(value)]; ok {
					matches[matchIndex] = true
					remaining--
				}
			} else {
				for _, candidate := range preds[predIndex].PhysBytes {
					if parquetDictionaryValueEqual(physicalType, value, candidate) {
						matches[matchIndex] = true
						remaining--

						break
					}
				}
			}
		}
	}

	if remaining == 0 {
		// All predicates already match, so trailing dictionary entries cannot
		// change the decision and do not need to be decoded or validated.
		return true, true
	}

	if offset != len(data) {
		return false, false
	}

	return false, true
}

const (
	parquetDictionaryCandidateSetLimit     = 8
	parquetDictionaryCandidateSetMinValues = 16
)

func parquetDictionaryCandidateSet(
	physicalType parquet.Type,
	width int,
	variableWidth bool,
	numValues int32,
	candidates [][]byte,
) map[string]struct{} {
	if len(candidates) < parquetDictionaryCandidateSetLimit ||
		numValues < parquetDictionaryCandidateSetMinValues {
		return nil
	}

	switch physicalType {
	case parquet.Types.Int32, parquet.Types.Int64,
		parquet.Types.ByteArray, parquet.Types.FixedLenByteArray:
	default:
		// Float and Double need numeric comparison for signed zero and NaN.
		return nil
	}

	result := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if variableWidth || len(candidate) == width {
			result[string(candidate)] = struct{}{}
		}
	}

	return result
}

func parquetDictionaryValueLayout(physicalType parquet.Type, typeLen int) (width int, variableWidth, ok bool) {
	switch physicalType {
	case parquet.Types.Int32, parquet.Types.Float:
		return 4, false, true
	case parquet.Types.Int64, parquet.Types.Double:
		return 8, false, true
	case parquet.Types.ByteArray:
		return 0, true, true
	case parquet.Types.FixedLenByteArray:
		if typeLen > 0 {
			return typeLen, false, true
		}
	}

	return 0, false, false
}

func nextParquetDictionaryValue(data []byte, offset int, physicalType parquet.Type, width int) ([]byte, int, bool) {
	if physicalType == parquet.Types.ByteArray {
		if offset < 0 || offset > len(data) || len(data)-offset < 4 {
			return nil, 0, false
		}

		valueLen := int(binary.LittleEndian.Uint32(data[offset:]))
		valueStart := offset + 4
		if valueLen > len(data)-valueStart {
			return nil, 0, false
		}

		return data[valueStart : valueStart+valueLen], valueStart + valueLen, true
	}

	if offset < 0 || width < 0 || width > len(data)-offset {
		return nil, 0, false
	}

	return data[offset : offset+width], offset + width, true
}

func parquetDictionaryValueEqual(physicalType parquet.Type, value, candidate []byte) bool {
	if physicalType == parquet.Types.Float && len(value) == 4 && len(candidate) == 4 {
		left := math.Float32frombits(binary.LittleEndian.Uint32(value))
		right := math.Float32frombits(binary.LittleEndian.Uint32(candidate))
		if math.IsNaN(float64(right)) {
			// Keep groups for NaN literals because reader equality semantics
			// can differ.
			return true
		}
		if math.IsNaN(float64(left)) {
			return false
		}

		return left == right
	}

	if physicalType == parquet.Types.Double && len(value) == 8 && len(candidate) == 8 {
		left := math.Float64frombits(binary.LittleEndian.Uint64(value))
		right := math.Float64frombits(binary.LittleEndian.Uint64(candidate))
		if math.IsNaN(right) {
			// Keep groups for NaN literals because reader equality semantics
			// can differ.
			return true
		}
		if math.IsNaN(left) {
			return false
		}

		return left == right
	}

	return slices.Equal(value, candidate)
}

// buildFieldIDToColIdx maps each Iceberg field ID to its 0-based column index in
// the Parquet file schema. Used to translate bloom filter predicates (which carry
// field IDs) into the column positions required by BloomFilterReader.
func buildFieldIDToColIdx(meta *metadata.FileMetaData) map[int]int {
	sc := meta.Schema
	result := make(map[int]int, sc.NumColumns())
	for i := range sc.NumColumns() {
		fieldID := int(sc.Column(i).SchemaNode().FieldID())
		result[fieldID] = i
	}

	return result
}

// checkRowGroupBloomFilters checks each bloom predicate against the bloom filter
// for its column in row group rg. Returns false (skip) if ANY predicate has none
// of its values present in the bloom filter. Returns true (keep) on any error or
// missing bloom filter data — bloom filters are an optimisation, never a
// correctness gate.
func checkRowGroupBloomFilters(
	bfReader *metadata.BloomFilterReader,
	rg int,
	fieldIDToColIdx map[int]int,
	preds []RowGroupBloomPred,
) (bool, error) {
	rgBFReader, err := bfReader.RowGroup(rg)
	if err != nil {
		return false, err
	}

	if rgBFReader == nil {
		return true, nil
	}

	for _, pred := range preds {
		colIdx, ok := fieldIDToColIdx[pred.FieldID]
		if !ok {
			continue
		}

		bf, err := rgBFReader.GetColumnBloomFilter(colIdx)
		if err != nil || bf == nil {
			continue
		}

		hasher := bf.Hasher()
		anyFound := false
		for _, physBytes := range pred.PhysBytes {
			if bf.CheckHash(hasher.Sum64(physBytes)) {
				anyFound = true

				break
			}
		}

		if !anyFound {
			return false, nil
		}
	}

	return true, nil
}

func (pfs *ParquetFileSource) GetReader(ctx context.Context) (result FileReader, err error) {
	pf, err := pfs.fs.Open(pfs.file.FilePath())
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			if closeErr := pf.Close(); closeErr != nil {
				err = errors.Join(err, closeErr)
			}
		}
	}()

	readProps := parquet.NewReaderProperties(pfs.mem)
	var rdr *file.Reader
	var rowGroupInfos []parquetRowGroupInfo
	if cache := parquetMetadataCacheFromContext(ctx); cache != nil {
		entry := cache.entry(parquetMetadataCacheKey{
			path: pfs.file.FilePath(),
			size: pfs.file.FileSizeBytes(),
		})

		var parsed *file.Reader
		entry.once.Do(func() {
			parsed, entry.err = file.NewParquetReader(pf, file.WithReadProps(readProps))
			if entry.err == nil {
				entry.fileMetadata = parsed.MetaData()
				entry.rowGroupInfos = parquetRowGroupInfos(parsed)
			}
		})
		if entry.err != nil {
			return nil, entry.err
		}
		rowGroupInfos = entry.rowGroupInfos

		if parsed != nil {
			rdr = parsed
		} else {
			rdr, err = file.NewParquetReader(pf,
				file.WithMetadata(entry.fileMetadata), file.WithReadProps(readProps))
			if err != nil {
				return nil, err
			}
		}
	} else {
		rdr, err = file.NewParquetReader(pf, file.WithReadProps(readProps))
		if err != nil {
			return nil, err
		}
	}

	props := TablePropertiesFromContext(ctx)
	arrProps := pqarrow.ArrowReadProperties{
		Parallel:  true,
		BatchSize: int64(props.GetInt(ParquetBatchSizeKey, ParquetBatchSizeDefault)),
	}

	if pfs.file.ContentType() == iceberg.EntryContentPosDeletes {
		// for dictionary for filepath col
		arrProps.SetReadDict(0, true)
	}

	fr, err := pqarrow.NewFileReader(rdr, arrProps, pfs.mem)
	if err != nil {
		return nil, err
	}

	result = wrapPqArrowReader{
		FileReader:       fr,
		rowGroupInfos:    rowGroupInfos,
		dictionarySource: pf,
		dictionaryMem:    pfs.mem,
	}

	return result, nil
}

type manifestVisitor[T any] interface {
	Manifest(*pqarrow.SchemaManifest, []T, *iceberg.MappedField) T
	Field(pqarrow.SchemaField, T, *iceberg.MappedField) T
	Struct(pqarrow.SchemaField, []T, *iceberg.MappedField) T
	List(pqarrow.SchemaField, T, *iceberg.MappedField) T
	Map(pqarrow.SchemaField, T, T, *iceberg.MappedField) T
	Primitive(pqarrow.SchemaField, *iceberg.MappedField) T
}

func visitParquetManifest[T any](manifest *pqarrow.SchemaManifest, visitor manifestVisitor[T], mapping *iceberg.MappedField) (res T, err error) {
	if manifest == nil {
		err = fmt.Errorf("%w: cannot visit nil manifest", iceberg.ErrInvalidArgument)

		return res, err
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()

	var fieldMap *iceberg.MappedField

	results := make([]T, len(manifest.Fields))
	for i, f := range manifest.Fields {
		if mapping != nil {
			fieldMap = mapping.GetField(f.Field.Name)
		}
		res := visitManifestField(f, visitor, fieldMap)
		results[i] = visitor.Field(f, res, fieldMap)
	}

	return visitor.Manifest(manifest, results, mapping), nil
}

func visitParquetManifestStruct[T any](field pqarrow.SchemaField, visitor manifestVisitor[T], mapping *iceberg.MappedField) T {
	results := make([]T, len(field.Children))
	var fieldMap *iceberg.MappedField
	for i, f := range field.Children {
		if mapping != nil {
			fieldMap = mapping.GetField(f.Field.Name)
		}
		res := visitManifestField(f, visitor, fieldMap)
		results[i] = visitor.Field(f, res, fieldMap)
	}

	return visitor.Struct(field, results, mapping)
}

func visitManifestList[T any](field pqarrow.SchemaField, visitor manifestVisitor[T], mapping *iceberg.MappedField) T {
	elemField := field.Children[0]
	var elemMapping *iceberg.MappedField
	if mapping != nil {
		elemMapping = mapping.GetField("element")
	}
	res := visitManifestField(elemField, visitor, elemMapping)

	return visitor.List(field, res, mapping)
}

func visitManifestMap[T any](field pqarrow.SchemaField, visitor manifestVisitor[T], mapping *iceberg.MappedField) T {
	kvfield := field.Children[0]
	keyField, valField := kvfield.Children[0], kvfield.Children[1]
	var keyMapping, valMapping *iceberg.MappedField
	if mapping != nil {
		keyMapping = mapping.GetField("key")
		valMapping = mapping.GetField("value")
	}

	return visitor.Map(field, visitManifestField(keyField, visitor, keyMapping), visitManifestField(valField, visitor, valMapping), mapping)
}

func visitManifestField[T any](field pqarrow.SchemaField, visitor manifestVisitor[T], mapping *iceberg.MappedField) T {
	switch field.Field.Type.(type) {
	case *arrow.StructType:
		return visitParquetManifestStruct(field, visitor, mapping)
	case *arrow.MapType:
		return visitManifestMap(field, visitor, mapping)
	case arrow.ListLikeType:
		return visitManifestList(field, visitor, mapping)
	default:
		return visitor.Primitive(field, mapping)
	}
}

func pruneParquetColumns(manifest *pqarrow.SchemaManifest, selected map[int]struct{}, selectFullTypes bool, mapping iceberg.NameMapping) (*arrow.Schema, []int, error) {
	visitor := &pruneParquetSchema{
		selected:  selected,
		manifest:  manifest,
		fullTypes: selectFullTypes,
		indices:   []int{},
	}

	result, err := visitParquetManifest(manifest, visitor, &iceberg.MappedField{Fields: mapping})
	if err != nil {
		return nil, nil, err
	}

	return arrow.NewSchema(result.Type.(*arrow.StructType).Fields(), &result.Metadata),
		visitor.indices, nil
}

func getFieldID(f arrow.Field) *int {
	if !f.HasMetadata() {
		return nil
	}

	fieldIDStr, ok := f.Metadata.GetValue("PARQUET:field_id")
	if !ok {
		return nil
	}

	id, err := strconv.Atoi(fieldIDStr)
	if err != nil || id <= 0 {
		return nil
	}

	return &id
}

type pruneParquetSchema struct {
	selected  map[int]struct{}
	fullTypes bool
	manifest  *pqarrow.SchemaManifest

	indices []int
}

func (p *pruneParquetSchema) fieldID(field arrow.Field, mapping *iceberg.MappedField) int {
	// Embedded Parquet field IDs are authoritative. Name mappings are only a
	// fallback for files and fields that do not carry their own IDs.
	if id := getFieldID(field); id != nil {
		return *id
	}

	if mapping != nil {
		if mapping.FieldID != nil {
			return *mapping.FieldID
		}
	}

	panic(fmt.Errorf("%w: cannot convert %s to Iceberg field, missing field_id",
		iceberg.ErrInvalidSchema, field))
}

func (p *pruneParquetSchema) Manifest(manifest *pqarrow.SchemaManifest, fields []arrow.Field, _ *iceberg.MappedField) arrow.Field {
	finalFields := slices.DeleteFunc(fields, func(f arrow.Field) bool { return f.Type == nil })
	result := arrow.Field{
		Type: arrow.StructOf(finalFields...),
	}
	if manifest.SchemaMeta != nil {
		result.Metadata = *manifest.SchemaMeta
	}

	return result
}

func (p *pruneParquetSchema) Struct(field pqarrow.SchemaField, children []arrow.Field, _ *iceberg.MappedField) arrow.Field {
	selected, fields := []arrow.Field{}, field.Children
	sameType := true

	for i, t := range children {
		field := fields[i]
		if arrow.TypeEqual(field.Field.Type, t.Type) {
			selected = append(selected, *field.Field)
		} else if t.Type == nil {
			sameType = false
			// type has changed, create a new field with the projected type
			selected = append(selected, arrow.Field{
				Name:     field.Field.Name,
				Type:     field.Field.Type,
				Nullable: field.Field.Nullable,
				Metadata: field.Field.Metadata,
			})
		}
	}

	if len(selected) > 0 {
		if len(selected) == len(fields) && sameType {
			// nothing changed, return the original
			return *field.Field
		} else {
			result := *field.Field
			result.Type = arrow.StructOf(selected...)

			return result
		}
	}

	return arrow.Field{}
}

func (p *pruneParquetSchema) Field(field pqarrow.SchemaField, result arrow.Field, mapping *iceberg.MappedField) arrow.Field {
	_, ok := p.selected[p.fieldID(*field.Field, mapping)]
	if !ok {
		if result.Type != nil {
			return result
		}

		return arrow.Field{}
	}

	if p.fullTypes {
		return *field.Field
	}

	if _, ok := field.Field.Type.(*arrow.StructType); ok {
		result := *field.Field
		result.Type = p.projectSelectedStruct(result.Type)

		return result
	}

	// Variant is an extension type wrapping a struct (metadata + value).
	// Select the entire field including all children.
	if ext, ok := field.Field.Type.(arrow.ExtensionType); ok && extensions.IsVariantExtensionName(ext.ExtensionName()) {
		return p.projectVariant(field)
	}

	if !field.IsLeaf() {
		panic(errors.New("cannot explicitly project list or map types"))
	}

	p.indices = append(p.indices, field.ColIndex)

	return *field.Field
}

func (p *pruneParquetSchema) List(field pqarrow.SchemaField, elemResult arrow.Field, mapping *iceberg.MappedField) arrow.Field {
	var elemMapping *iceberg.MappedField
	if mapping != nil {
		elemMapping = mapping.GetField("element")
	}

	_, ok := p.selected[p.fieldID(*field.Children[0].Field, elemMapping)]
	if !ok {
		if elemResult.Type != nil {
			result := *field.Field
			result.Type = p.projectList(field.Field.Type.(arrow.ListLikeType), elemResult.Type)

			return result
		}

		return arrow.Field{}
	}

	if p.fullTypes {
		return *field.Field
	}

	_, ok = field.Children[0].Field.Type.(*arrow.StructType)
	if field.Children[0].Field.Type != nil && ok {
		result := *field.Field
		projected := p.projectSelectedStruct(elemResult.Type)
		result.Type = p.projectList(field.Field.Type.(arrow.ListLikeType), projected)

		return result
	}

	if !field.Children[0].IsLeaf() {
		panic(errors.New("cannot explicitly project list or map types"))
	}

	p.indices = append(p.indices, field.Children[0].ColIndex)

	return *field.Field
}

func (p *pruneParquetSchema) Map(field pqarrow.SchemaField, keyResult, valResult arrow.Field, mapping *iceberg.MappedField) arrow.Field {
	var valMapping *iceberg.MappedField
	if mapping != nil {
		valMapping = mapping.GetField("value")
	}

	_, ok := p.selected[p.fieldID(*field.Children[0].Children[1].Field, valMapping)]
	if !ok {
		if valResult.Type != nil {
			result := *field.Field
			result.Type = p.projectMap(field.Field.Type.(*arrow.MapType), valResult.Type)

			return result
		}

		return arrow.Field{}
	}

	if p.fullTypes {
		return *field.Field
	}

	_, ok = field.Children[0].Children[1].Field.Type.(*arrow.StructType)
	if ok {
		result := *field.Field
		projected := p.projectSelectedStruct(valResult.Type)
		result.Type = p.projectMap(field.Field.Type.(*arrow.MapType), projected)

		return result
	}

	if !field.Children[0].Children[1].IsLeaf() {
		panic("cannot explicitly project list or map types")
	}

	p.indices = append(p.indices, field.Children[0].Children[0].ColIndex)
	p.indices = append(p.indices, field.Children[0].Children[1].ColIndex)

	return *field.Field
}

func (p *pruneParquetSchema) Primitive(_ pqarrow.SchemaField, _ *iceberg.MappedField) arrow.Field {
	return arrow.Field{}
}

func (p *pruneParquetSchema) projectSelectedStruct(projected arrow.DataType) *arrow.StructType {
	if projected == nil {
		return &arrow.StructType{}
	}

	if ty, ok := projected.(*arrow.StructType); ok {
		return ty
	}

	panic("expected a struct")
}

func (p *pruneParquetSchema) collectLeafIndices(field pqarrow.SchemaField) {
	if field.IsLeaf() {
		p.indices = append(p.indices, field.ColIndex)

		return
	}

	for _, child := range field.Children {
		p.collectLeafIndices(child)
	}
}

func (p *pruneParquetSchema) projectVariant(field pqarrow.SchemaField) arrow.Field {
	p.collectLeafIndices(field)

	return *field.Field
}

func (p *pruneParquetSchema) projectList(listType arrow.ListLikeType, elemResult arrow.DataType) arrow.ListLikeType {
	if arrow.TypeEqual(listType.Elem(), elemResult) {
		return listType
	}

	origField := listType.ElemField()
	origField.Type = elemResult

	switch listType.(type) {
	case *arrow.ListType:
		return arrow.ListOfField(origField)
	case *arrow.LargeListType:
		return arrow.LargeListOfField(origField)
	case *arrow.ListViewType:
		return arrow.ListViewOfField(origField)
	}

	n := listType.(*arrow.FixedSizeListType).Len()

	return arrow.FixedSizeListOfField(n, origField)
}

func (p *pruneParquetSchema) projectMap(m *arrow.MapType, valResult arrow.DataType) *arrow.MapType {
	if arrow.TypeEqual(m.ItemType(), valResult) {
		return m
	}

	return arrow.MapOf(m.KeyType(), valResult)
}

type id2ParquetPath struct {
	fieldID int
	path    string
}

type id2ParquetPathVisitor struct {
	fieldID int
	path    []string
}

func (v *id2ParquetPathVisitor) Schema(_ *iceberg.Schema, res func() []id2ParquetPath) []id2ParquetPath {
	return res()
}

func (v *id2ParquetPathVisitor) Struct(_ iceberg.StructType, results []func() []id2ParquetPath) []id2ParquetPath {
	result := make([]id2ParquetPath, 0, len(results))
	for _, res := range results {
		result = append(result, res()...)
	}

	return result
}

func (v *id2ParquetPathVisitor) Field(field iceberg.NestedField, res func() []id2ParquetPath) []id2ParquetPath {
	v.fieldID = field.ID
	v.path = append(v.path, field.Name)
	result := res()
	v.path = v.path[:len(v.path)-1]

	return result
}

func (v *id2ParquetPathVisitor) List(listType iceberg.ListType, elemResult func() []id2ParquetPath) []id2ParquetPath {
	v.fieldID = listType.ElementID
	v.path = append(v.path, "list")
	result := elemResult()
	v.path = v.path[:len(v.path)-1]

	return result
}

func (v *id2ParquetPathVisitor) Map(m iceberg.MapType, keyResult func() []id2ParquetPath, valResult func() []id2ParquetPath) []id2ParquetPath {
	v.fieldID = m.KeyID
	v.path = append(v.path, "key_value")
	keyRes := keyResult()
	v.path = v.path[:len(v.path)-1]

	v.fieldID = m.ValueID
	v.path = append(v.path, "key_value")
	valRes := valResult()
	v.path = v.path[:len(v.path)-1]

	return append(keyRes, valRes...)
}

func (v *id2ParquetPathVisitor) Primitive(iceberg.PrimitiveType) []id2ParquetPath {
	return []id2ParquetPath{{fieldID: v.fieldID, path: strings.Join(v.path, ".")}}
}

func (v *id2ParquetPathVisitor) Variant(iceberg.VariantType) []id2ParquetPath {
	return []id2ParquetPath{{fieldID: v.fieldID, path: strings.Join(v.path, ".")}}
}
