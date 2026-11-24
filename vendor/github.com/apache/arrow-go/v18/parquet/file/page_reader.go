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

package file

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"sort"
	"sync"

	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/utils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/compress"
	"github.com/apache/arrow-go/v18/parquet/internal/encoding/streaming"
	"github.com/apache/arrow-go/v18/parquet/internal/encryption"
	format "github.com/apache/arrow-go/v18/parquet/internal/gen-go/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/thrift"
	"github.com/apache/arrow-go/v18/parquet/metadata"
)

// PageReader is the interface used by the columnreader in order to read
// and handle DataPages and loop through them.
type PageReader interface {
	// Set the maximum Page header size allowed to be read
	SetMaxPageHeaderSize(int)
	// Return the current page, or nil if there are no more
	Page() Page
	// Fetch the next page, returns false if there are no more pages
	Next() bool
	// if Next returns false, Err will return the error encountered or
	// nil if there was no error and you just hit the end of the page
	Err() error
	// Reset allows reusing a page reader
	Reset(r parquet.BufferedReader, nrows int64, compressType compress.Compression, ctx *CryptoContext)

	// Get the dictionary page for this column chunk
	GetDictionaryPage() (*DictionaryPage, error)
	SeekToPageWithRow(rowIdx int64) error
	// Close releases the resources held by the reader.
	Close() error
}

type PageType = format.PageType

const (
	PageTypeDataPage       PageType = format.PageType_DATA_PAGE
	PageTypeDataPageV2     PageType = format.PageType_DATA_PAGE_V2
	PageTypeDictionaryPage PageType = format.PageType_DICTIONARY_PAGE
	PageTypeIndexPage      PageType = format.PageType_INDEX_PAGE
)

// Page is an interface for handling DataPages or Dictionary Pages
type Page interface {
	// Returns which kind of page this is
	Type() PageType
	// Get the raw bytes of this page.
	//
	// Note: with PageStreamingEnabled, a streaming data page's Data holds only the
	// rep/def level region, not the whole page; decode values through a column or
	// record reader instead.
	Data() []byte
	// return the encoding used for this page, Plain/RLE, etc.
	Encoding() format.Encoding
	// get the number of values in this page
	NumValues() int32
	// release this page object back into the page pool for re-use
	Release()
}

type page struct {
	buf *memory.Buffer
	typ format.PageType

	nvals    int32
	encoding format.Encoding
}

func (p *page) Type() PageType            { return p.typ }
func (p *page) Data() []byte              { return p.buf.Bytes() }
func (p *page) NumValues() int32          { return p.nvals }
func (p *page) Encoding() format.Encoding { return p.encoding }

// dataPageBase holds the members shared by DataPageV1/V2 but not dictionary
// pages: the rep/def level buffer accessor and, in streaming mode, the
// incremental value source.
type dataPageBase struct {
	page

	// valueBuf is only set for streaming mode to read values incrementally.
	// In such mode, buf only holds the rep+def level region.
	valueBuf streaming.ValueBuffer
	// clip bounds a read's batch to keep one Decode's aliased values near the buffer
	// size; 0 for non-streaming pages.
	clip int64
}

func (d *dataPageBase) valueBuffer() streaming.ValueBuffer { return d.valueBuf }
func (d *dataPageBase) levelBuffer() []byte                { return d.buf.Bytes() }

func (d *dataPageBase) valueClip() int64 { return d.clip }

func (d *dataPageBase) setStreamingValues(mem memory.Allocator, lenUncompressed int, levelBuf *memory.Buffer, valReader, rawSrc io.Reader, closer io.Closer) {
	d.buf = levelBuf
	valueBytes := lenUncompressed - levelBuf.Len()
	d.valueBuf = streaming.NewStreamBuffer(mem, valReader, valueBytes, drainAndClose(rawSrc, closer))
	d.clip = streaming.ClipBatch(valueBytes, int(d.nvals))
}

// releaseValueStream closes a streaming page's value source (draining + closing its
// underlying compressed stream so the file reader lands on the next page header even
// when values were skipped). It is a no-op for non-streaming pages.
func (d *dataPageBase) releaseValueStream() {
	if d.valueBuf != nil {
		_ = d.valueBuf.Close()
		d.valueBuf = nil
	}
	d.clip = 0
}

// DataPageConfig is a struct for passing configuration params to data page creation
// which can be expanded in the future without causing any breaking changes.
type DataPageConfig struct {
	Num              int32
	Encoding         parquet.Encoding
	UncompressedSize int32
	Stats            metadata.EncodedStatistics
	FirstRowIndex    int64
	SizeStats        SizeStatistics
}

// DataPage is the base interface for both DataPageV1 and DataPageV2 of the
// parquet spec.
type DataPage interface {
	Page
	UncompressedSize() int32
	Statistics() metadata.EncodedStatistics
	// FirstRowIndex returns the row ordinal within the row group
	// to the first row in the data page, or -1 if not set.
	FirstRowIndex() int64

	// valueBuffer returns the streaming value source, or nil for a non-streaming page.
	valueBuffer() streaming.ValueBuffer
	// valueClip bounds a read's batch on a streaming page, or 0 for a non-streaming page.
	valueClip() int64
}

// Create some pools to use for reusing the data page objects themselves so that
// we can avoid tight loops that are creating and destroying tons of individual
// objects. This combined with a Release function on the pages themselves
// which will put them back into the pool yields significant memory reduction
// and performance benefits

var dataPageV1Pool = sync.Pool{
	New: func() interface{} { return (*DataPageV1)(nil) },
}

var dataPageV2Pool = sync.Pool{
	New: func() interface{} { return (*DataPageV2)(nil) },
}

var dictPagePool = sync.Pool{
	New: func() interface{} { return (*DictionaryPage)(nil) },
}

// DataPageV1 represents a DataPage version 1 from the parquet.thrift file
type DataPageV1 struct {
	dataPageBase

	defLvlEncoding   format.Encoding
	repLvlEncoding   format.Encoding
	uncompressedSize int32
	statistics       metadata.EncodedStatistics
	firstRowIndex    int64
	sizeStatistics   SizeStatistics
}

// NewDataPageV1 returns a V1 data page with the given buffer as its data and the specified encoding information
//
// Will utilize objects that have been released back into the data page pool and
// re-use them if available as opposed to creating new objects. Calling Release on the
// data page object will release it back to the pool for re-use.
func NewDataPageV1(buffer *memory.Buffer, num int32, encoding, defEncoding, repEncoding parquet.Encoding, uncompressedSize int32) *DataPageV1 {
	dp := dataPageV1Pool.Get().(*DataPageV1)
	if dp == nil {
		return &DataPageV1{
			dataPageBase:     dataPageBase{page: page{buf: buffer, typ: format.PageType_DATA_PAGE, nvals: num, encoding: format.Encoding(encoding)}},
			defLvlEncoding:   format.Encoding(defEncoding),
			repLvlEncoding:   format.Encoding(repEncoding),
			uncompressedSize: uncompressedSize,
			firstRowIndex:    -1,
		}
	}

	dp.buf, dp.nvals = buffer, num
	dp.encoding = format.Encoding(encoding)
	dp.defLvlEncoding, dp.repLvlEncoding = format.Encoding(defEncoding), format.Encoding(repEncoding)
	dp.statistics.HasMax, dp.statistics.HasMin = false, false
	dp.statistics.HasNullCount, dp.statistics.HasDistinctCount = false, false
	dp.uncompressedSize = uncompressedSize
	dp.firstRowIndex = -1
	dp.sizeStatistics = SizeStatistics{}
	return dp
}

// NewDataPageV1WithStats is the same as NewDataPageV1, but also allows adding the stat info into the created page
func NewDataPageV1WithStats(buffer *memory.Buffer, num int32, encoding, defEncoding, repEncoding parquet.Encoding, uncompressedSize int32, stats metadata.EncodedStatistics) *DataPageV1 {
	ret := NewDataPageV1(buffer, num, encoding, defEncoding, repEncoding, uncompressedSize)
	ret.statistics = stats
	return ret
}

// NewDataPageV1WithConfig uses a DataPageConfig object to encapsulate some parameters for future expansion
// rather than continuing to add new functions to the API to avoid breaking changes.
func NewDataPageV1WithConfig(buffer *memory.Buffer, defEncoding, repEncoding parquet.Encoding, cfg DataPageConfig) *DataPageV1 {
	ret := NewDataPageV1WithStats(buffer, cfg.Num, cfg.Encoding, defEncoding, repEncoding, cfg.UncompressedSize, cfg.Stats)
	ret.firstRowIndex, ret.sizeStatistics = cfg.FirstRowIndex, cfg.SizeStats
	return ret
}

// Release this page back into the DataPage object pool so that it can be reused.
//
// After calling this function, the object should not be utilized anymore, otherwise
// conflicts can arise.
func (d *DataPageV1) Release() {
	if d.buf == nil {
		return
	}
	d.releaseValueStream()
	d.buf.Release()
	d.buf = nil
	dataPageV1Pool.Put(d)
}

func (d *DataPageV1) FirstRowIndex() int64 { return d.firstRowIndex }

// UncompressedSize returns the size of the data in this data page when uncompressed
func (d *DataPageV1) UncompressedSize() int32 { return d.uncompressedSize }

// Statistics returns the encoded statistics on this data page
func (d *DataPageV1) Statistics() metadata.EncodedStatistics { return d.statistics }

// DefinitionLevelEncoding returns the encoding utilized for the Definition Levels
func (d *DataPageV1) DefinitionLevelEncoding() parquet.Encoding {
	return parquet.Encoding(d.defLvlEncoding)
}

// RepetitionLevelEncoding returns the encoding utilized for the Repetition Levels
func (d *DataPageV1) RepetitionLevelEncoding() parquet.Encoding {
	return parquet.Encoding(d.repLvlEncoding)
}

// DataPageV2 is the representation of the V2 data page from the parquet.thrift spec
type DataPageV2 struct {
	dataPageBase

	nulls            int32
	nrows            int32
	defLvlByteLen    int32
	repLvlByteLen    int32
	compressed       bool
	uncompressedSize int32
	statistics       metadata.EncodedStatistics
	firstRowIndex    int64
	sizeStatistics   SizeStatistics
}

// NewDataPageV2 constructs a new V2 data page with the provided information and a buffer of the raw data.
func NewDataPageV2(buffer *memory.Buffer, numValues, numNulls, numRows int32, encoding parquet.Encoding, defLvlsByteLen, repLvlsByteLen, uncompressed int32, isCompressed bool) *DataPageV2 {
	dp := dataPageV2Pool.Get().(*DataPageV2)
	if dp == nil {
		return &DataPageV2{
			dataPageBase:     dataPageBase{page: page{buf: buffer, typ: format.PageType_DATA_PAGE_V2, nvals: numValues, encoding: format.Encoding(encoding)}},
			nulls:            numNulls,
			nrows:            numRows,
			defLvlByteLen:    defLvlsByteLen,
			repLvlByteLen:    repLvlsByteLen,
			compressed:       isCompressed,
			uncompressedSize: uncompressed,
			firstRowIndex:    -1,
		}
	}

	dp.buf, dp.nvals = buffer, numValues
	dp.encoding = format.Encoding(encoding)
	dp.nulls, dp.nrows = numNulls, numRows
	dp.defLvlByteLen, dp.repLvlByteLen = defLvlsByteLen, repLvlsByteLen
	dp.compressed, dp.uncompressedSize = isCompressed, uncompressed
	dp.statistics.HasMax, dp.statistics.HasMin = false, false
	dp.statistics.HasNullCount, dp.statistics.HasDistinctCount = false, false
	dp.firstRowIndex = -1
	dp.sizeStatistics = SizeStatistics{}
	return dp
}

// NewDataPageV2WithStats is the same as NewDataPageV2 but allows providing the encoded stats with the page.
func NewDataPageV2WithStats(buffer *memory.Buffer, numValues, numNulls, numRows int32, encoding parquet.Encoding, defLvlsByteLen, repLvlsByteLen, uncompressed int32, isCompressed bool, stats metadata.EncodedStatistics) *DataPageV2 {
	ret := NewDataPageV2(buffer, numValues, numNulls, numRows, encoding, defLvlsByteLen, repLvlsByteLen, uncompressed, isCompressed)
	ret.statistics = stats
	return ret
}

// NewDataPageV2WithConfig uses a DataPageConfig object to encapsulate some parameters for future expansion
// rather than continuing to add new functions to the API to avoid breaking changes.
func NewDataPageV2WithConfig(buffer *memory.Buffer, numNulls, numRows int32, defLvlsByteLen, repLvlsByteLen int32, isCompressed bool, cfg DataPageConfig) *DataPageV2 {
	ret := NewDataPageV2WithStats(buffer, cfg.Num, numNulls, numRows, cfg.Encoding, defLvlsByteLen, repLvlsByteLen, cfg.UncompressedSize, isCompressed, cfg.Stats)
	ret.firstRowIndex, ret.sizeStatistics = cfg.FirstRowIndex, cfg.SizeStats
	return ret
}

// Release this page back into the DataPage object pool so that it can be reused.
//
// After calling this function, the object should not be utilized anymore, otherwise
// conflicts can arise.
func (d *DataPageV2) Release() {
	if d.buf == nil {
		return
	}
	d.releaseValueStream()
	d.buf.Release()
	d.buf = nil
	dataPageV2Pool.Put(d)
}

func (d *DataPageV2) FirstRowIndex() int64 { return d.firstRowIndex }

// UncompressedSize is the size of the raw page when uncompressed. If `IsCompressed` is true, then
// the raw data in the buffer is expected to be compressed.
func (d *DataPageV2) UncompressedSize() int32 { return d.uncompressedSize }

// Statistics are the encoded statistics in the data page
func (d *DataPageV2) Statistics() metadata.EncodedStatistics { return d.statistics }

// NumNulls is the reported number of nulls in this datapage
func (d *DataPageV2) NumNulls() int32 { return d.nulls }

// NumRows is the number of rows recorded in the page header
func (d *DataPageV2) NumRows() int32 { return d.nrows }

// DefinitionLevelByteLen is the number of bytes in the buffer that are used to represent the definition levels
func (d *DataPageV2) DefinitionLevelByteLen() int32 { return d.defLvlByteLen }

// RepetitionLevelByteLen is the number of bytes in the buffer which are used to represent the repetition Levels
func (d *DataPageV2) RepetitionLevelByteLen() int32 { return d.repLvlByteLen }

// IsCompressed returns true if the data of this page is compressed
func (d *DataPageV2) IsCompressed() bool { return d.compressed }

// DictionaryPage represents the a page of data that uses dictionary encoding
type DictionaryPage struct {
	page

	sorted bool
}

// NewDictionaryPage constructs a new dictionary page with the provided data buffer and number of values.
func NewDictionaryPage(buffer *memory.Buffer, nvals int32, encoding parquet.Encoding) *DictionaryPage {
	dp := dictPagePool.Get().(*DictionaryPage)
	if dp == nil {
		return &DictionaryPage{
			page: page{
				buf:      buffer,
				typ:      format.PageType_DICTIONARY_PAGE,
				nvals:    nvals,
				encoding: format.Encoding(encoding),
			},
		}
	}

	dp.buf = buffer
	dp.nvals = nvals
	dp.encoding = format.Encoding(encoding)
	dp.sorted = false
	return dp
}

// Release this page back into the DataPage object pool so that it can be reused.
//
// After calling this function, the object should not be utilized anymore, otherwise
// conflicts can arise.
func (d *DictionaryPage) Release() {
	d.buf.Release()
	d.buf = nil
	dictPagePool.Put(d)
}

// IsSorted returns whether the dictionary itself is sorted
func (d *DictionaryPage) IsSorted() bool { return d.sorted }

type serializedPageReader struct {
	r             parquet.BufferedReader
	chunk         *metadata.ColumnChunkMetaData
	colIdx        int
	pgIndexReader *metadata.RowGroupPageIndexReader

	nrows    int64
	rowsSeen int64
	mem      memory.Allocator
	codec    compress.Codec

	// Streaming (PageStreamingEnabled) inputs from schema/properties.
	columnCanStream bool
	maxRepLevel     int16
	maxDefLevel     int16

	curPageHdr              *format.PageHeader
	pageOrd                 int16
	maxPageHeaderSize       int
	maxCompressedPageSize   int64
	maxUncompressedPageSize int64

	curPage           Page
	cryptoCtx         CryptoContext
	dataPageAad       string
	dataPageHeaderAad string

	baseOffset, dataOffset, dictOffset int64

	decompressBuffer *memory.Buffer
	dataPageBuffer   *memory.Buffer
	dictPageBuffer   *memory.Buffer
	err              error
}

func (p *serializedPageReader) Close() error {
	if p.curPage != nil {
		// free a direct PageReader user's streaming value stream
		p.curPage.Release()
		p.curPage = nil
	}
	if p.decompressBuffer != nil {
		p.decompressBuffer.Release()
		p.dictPageBuffer.Release()
		p.dataPageBuffer.Release()
	}
	return nil
}

func (p *serializedPageReader) init(compressType compress.Compression, ctx *CryptoContext) error {
	if p.mem == nil {
		p.mem = memory.NewGoAllocator()
	}
	p.decompressBuffer = memory.NewResizableBuffer(p.mem)
	p.dataPageBuffer = memory.NewResizableBuffer(p.mem)
	p.dictPageBuffer = memory.NewResizableBuffer(p.mem)
	p.decompressBuffer.ResizeNoShrink(defaultPageHeaderSize)

	codec, err := compress.GetCodec(compressType)
	if err != nil {
		return err
	}
	p.codec = codec
	if _, ok := codec.(compress.StreamingCodec); !ok {
		// A codec registered via compress.RegisterCodec need not implement
		// streaming; disable streaming for this column in that case.
		p.columnCanStream = false
	}

	if ctx != nil {
		p.cryptoCtx = *ctx
		p.initDecryption()
	}

	p.baseOffset = p.chunk.DataPageOffset()
	p.dataOffset = p.baseOffset
	if p.chunk.HasDictionaryPage() && p.chunk.DictionaryPageOffset() > 0 {
		p.baseOffset = p.chunk.DictionaryPageOffset()
		p.dictOffset = p.baseOffset
	}

	return nil
}

// NewPageReader returns a page reader for the data which can be read from the provided reader and compression.
//
// Deprecated: This function isn't properly safe for public API use and should not be utilized
// anymore. It will be removed from the public interface soon to prevent usage outside of this package.
func NewPageReader(r parquet.BufferedReader, nrows int64, compressType compress.Compression, mem memory.Allocator, ctx *CryptoContext) (PageReader, error) {
	if mem == nil {
		mem = memory.NewGoAllocator()
	}

	codec, err := compress.GetCodec(compressType)
	if err != nil {
		return nil, err
	}

	rdr := &serializedPageReader{
		r:                       r,
		maxPageHeaderSize:       defaultMaxPageHeaderSize,
		nrows:                   nrows,
		mem:                     mem,
		codec:                   codec,
		maxCompressedPageSize:   parquet.DefaultMaxCompressedPageSize,
		maxUncompressedPageSize: parquet.DefaultMaxUncompressedPageSize,

		decompressBuffer: memory.NewResizableBuffer(mem),
		dataPageBuffer:   memory.NewResizableBuffer(mem),
		dictPageBuffer:   memory.NewResizableBuffer(mem),
	}
	rdr.decompressBuffer.ResizeNoShrink(defaultPageHeaderSize)
	if ctx != nil {
		rdr.cryptoCtx = *ctx
		rdr.initDecryption()
	}
	return rdr, nil
}

func (p *serializedPageReader) Reset(r parquet.BufferedReader, nrows int64, compressType compress.Compression, ctx *CryptoContext) {
	if p.curPage != nil {
		p.curPage.Release()
	}
	p.rowsSeen, p.pageOrd, p.nrows = 0, 0, nrows
	p.curPageHdr, p.curPage, p.err = nil, nil, nil
	p.r = r
	p.columnCanStream = false

	p.codec, p.err = compress.GetCodec(compressType)
	if p.err != nil {
		return
	}
	if ctx != nil {
		p.cryptoCtx = *ctx
		p.initDecryption()
	} else {
		p.cryptoCtx = CryptoContext{}
		p.dataPageAad = ""
		p.dataPageHeaderAad = ""
	}
}

func (p *serializedPageReader) Err() error { return p.err }

func (p *serializedPageReader) SetMaxPageHeaderSize(sz int) {
	p.maxPageHeaderSize = sz
}

func (p *serializedPageReader) validatePageSizes(hdr *format.PageHeader) (int, int, error) {
	compressed := int64(hdr.GetCompressedPageSize())
	uncompressed := int64(hdr.GetUncompressedPageSize())
	if compressed < 0 || uncompressed < 0 {
		return 0, 0, errors.New("parquet: invalid page header (negative page size)")
	}
	if compressed > p.maxCompressedPageSize {
		return 0, 0, fmt.Errorf("parquet: compressed page size %d exceeds configured limit %d", compressed, p.maxCompressedPageSize)
	}
	if uncompressed > p.maxUncompressedPageSize {
		return 0, 0, fmt.Errorf("parquet: uncompressed page size %d exceeds configured limit %d", uncompressed, p.maxUncompressedPageSize)
	}
	return int(compressed), int(uncompressed), nil
}

func (p *serializedPageReader) initDecryption() {
	if p.cryptoCtx.DataDecryptor != nil {
		p.dataPageAad = encryption.CreateModuleAad(p.cryptoCtx.DataDecryptor.FileAad(), encryption.DataPageModule,
			p.cryptoCtx.RowGroupOrdinal, p.cryptoCtx.ColumnOrdinal, -1)
	}
	if p.cryptoCtx.MetaDecryptor != nil {
		p.dataPageHeaderAad = encryption.CreateModuleAad(p.cryptoCtx.MetaDecryptor.FileAad(), encryption.DataPageHeaderModule,
			p.cryptoCtx.RowGroupOrdinal, p.cryptoCtx.ColumnOrdinal, -1)
	}
}

func (p *serializedPageReader) updateDecryption(decrypt encryption.Decryptor, moduleType int8, pageAad string) {
	if p.cryptoCtx.StartDecryptWithDictionaryPage {
		aad := encryption.CreateModuleAad(decrypt.FileAad(), moduleType, p.cryptoCtx.RowGroupOrdinal, p.cryptoCtx.ColumnOrdinal, -1)
		decrypt.UpdateAad(aad)
	} else {
		pageaad := []byte(pageAad)
		encryption.QuickUpdatePageAad(pageaad, p.pageOrd)
		decrypt.UpdateAad(string(pageaad))
	}
}

func (p *serializedPageReader) Page() Page {
	return p.curPage
}

func (p *serializedPageReader) decompress(rd io.Reader, lenCompressed int, buf []byte) ([]byte, error) {
	p.decompressBuffer.ResizeNoShrink(lenCompressed)

	// Read directly into the memory.Buffer's backing slice
	n, err := io.ReadFull(rd, p.decompressBuffer.Bytes()[:lenCompressed])
	if err != nil {
		return nil, err
	}
	if n != lenCompressed {
		return nil, fmt.Errorf("parquet: expected to read %d compressed bytes, got %d", lenCompressed, n)
	}

	data := p.decompressBuffer.Bytes()[:lenCompressed]
	if p.cryptoCtx.DataDecryptor != nil {
		data, err = p.cryptoCtx.DataDecryptor.Decrypt(data)
		if err != nil {
			return nil, fmt.Errorf("decrypting page data: %w", err)
		}
	}

	decoded, err := compress.Decode(p.codec, buf, data)
	if err != nil {
		return nil, fmt.Errorf("parquet: could not decompress page: %w", err)
	}
	return decoded, nil
}

func (p *serializedPageReader) readV2Encrypted(rd io.Reader, lenCompressed int, lenUncompressed int, levelsBytelen int, compressed bool, buf []byte) (int, error) {
	// if encrypted, we need to decrypt before decompressing
	p.decompressBuffer.ResizeNoShrink(lenCompressed)

	n, err := io.ReadFull(rd, p.decompressBuffer.Bytes()[:lenCompressed])
	if err != nil {
		return n, err
	}
	if n != lenCompressed {
		return n, fmt.Errorf("parquet: expected to read %d compressed bytes, got %d", lenCompressed, n)
	}

	data, err := p.cryptoCtx.DataDecryptor.Decrypt(p.decompressBuffer.Bytes()[:lenCompressed])
	if err != nil {
		return n, fmt.Errorf("decrypting data page: %w", err)
	}

	// encrypted + uncompressed -> just copy the decrypted data to output buffer
	if !compressed {
		if len(data) != lenUncompressed {
			return n, fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, len(data))
		}

		copied := copy(buf, data)
		if copied != lenUncompressed {
			return copied, fmt.Errorf("parquet: expected to read %d uncompressed encrypted bytes, got %d", lenUncompressed, copied)
		}
		return copied, nil
	}

	// definition + repetition levels are always uncompressed
	if levelsBytelen > 0 {
		if len(data) < levelsBytelen {
			return n, fmt.Errorf("parquet: expected to read %d bytes for levels, got %d", levelsBytelen, len(data))
		}
		copied := copy(buf, data[:levelsBytelen])
		if copied != levelsBytelen {
			return copied, fmt.Errorf("parquet: expected to read %d bytes for levels, got %d", levelsBytelen, copied)
		}
		data = data[levelsBytelen:]
	}
	decoded, err := compress.Decode(p.codec, buf[levelsBytelen:], data)
	if err != nil {
		return n, fmt.Errorf("parquet: could not decompress data page: %w", err)
	}
	if len(decoded) != lenUncompressed-levelsBytelen {
		return levelsBytelen + len(decoded), fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, levelsBytelen+len(decoded))
	}
	return levelsBytelen + len(decoded), nil
}

func (p *serializedPageReader) readV2Unencrypted(rd io.Reader, lenCompressed int, lenUncompressed int, levelsBytelen int, compressed bool, buf []byte) (int, error) {
	if !compressed {
		// uncompressed, just read into the buffer
		n, err := io.ReadFull(rd, buf)
		if err != nil {
			return n, err
		}
		if n != lenUncompressed {
			return n, fmt.Errorf("parquet: expected to read %d uncompressed bytes, got %d", lenUncompressed, n)
		}
		return n, nil
	}

	decodedLen := 0
	// definition + repetition levels are always uncompressed
	if levelsBytelen > 0 {
		n, err := io.ReadFull(rd, buf[:levelsBytelen])
		if err != nil {
			return n, err
		}
		decodedLen += n
		if n != levelsBytelen {
			return n, fmt.Errorf("parquet: expected to read %d bytes for levels, got %d", levelsBytelen, n)
		}
	}
	values, err := p.decompress(p.r, lenCompressed-levelsBytelen, buf[levelsBytelen:])
	if err != nil {
		return decodedLen, err
	}
	decodedLen += len(values)
	if decodedLen != lenUncompressed {
		return decodedLen, fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, decodedLen)
	}
	return decodedLen, nil
}

type dataheader interface {
	IsSetStatistics() bool
	GetStatistics() *format.Statistics
}

func extractStats(dataHeader dataheader) (pageStats metadata.EncodedStatistics) {
	if dataHeader.IsSetStatistics() {
		stats := dataHeader.GetStatistics()
		if stats.IsSetMaxValue() {
			pageStats.SetMax(stats.GetMaxValue())
		} else if stats.IsSetMax() {
			pageStats.SetMax(stats.GetMax())
		}
		if stats.IsSetMinValue() {
			pageStats.SetMin(stats.GetMinValue())
		} else if stats.IsSetMin() {
			pageStats.SetMin(stats.GetMin())
		}

		if stats.IsSetNullCount() {
			pageStats.SetNullCount(stats.GetNullCount())
		}
		if stats.IsSetDistinctCount() {
			pageStats.SetDistinctCount(stats.GetDistinctCount())
		}
	}
	return
}

func (p *serializedPageReader) GetDictionaryPage() (*DictionaryPage, error) {
	if p.dictOffset > 0 {
		hdr := format.NewPageHeader()
		readBufSize := min(int(p.dataOffset-p.baseOffset), p.r.BufferSize())
		rd := utils.NewBufferedReader(
			io.NewSectionReader(p.r.Outer(), p.dictOffset-p.baseOffset, p.dataOffset-p.baseOffset),
			readBufSize)
		if err := p.readPageHeader(rd, hdr); err != nil {
			return nil, err
		}

		dictHeader := hdr.GetDictionaryPageHeader()
		if dictHeader == nil {
			return nil, errors.New("parquet: invalid dictionary page header")
		}

		p.cryptoCtx.StartDecryptWithDictionaryPage = true
		if p.cryptoCtx.DataDecryptor != nil {
			p.updateDecryption(p.cryptoCtx.DataDecryptor, encryption.DictPageModule, p.dataPageAad)
		}

		lenCompressed, lenUncompressed, err := p.validatePageSizes(hdr)
		if err != nil {
			return nil, err
		}

		p.cryptoCtx.StartDecryptWithDictionaryPage = false
		if dictHeader.GetNumValues() < 0 {
			return nil, errors.New("parquet: invalid page header (negative number of values)")
		}

		p.dictPageBuffer.ResizeNoShrink(lenUncompressed)
		buf := memory.NewBufferBytes(p.dictPageBuffer.Bytes())

		data, err := p.decompress(rd, lenCompressed, buf.Bytes())
		if err != nil {
			return nil, err
		}
		if len(data) != lenUncompressed {
			return nil, fmt.Errorf("parquet: metadata said %d bytes uncompressed dictionary page, got %d bytes", lenUncompressed, len(data))
		}

		return &DictionaryPage{
			page: page{
				buf:      buf,
				typ:      hdr.Type,
				nvals:    dictHeader.GetNumValues(),
				encoding: dictHeader.GetEncoding(),
			},
			sorted: dictHeader.IsSetIsSorted() && dictHeader.GetIsSorted(),
		}, nil
	}

	return nil, nil
}

func (p *serializedPageReader) readPageHeader(rd parquet.BufferedReader, hdr *format.PageHeader) error {
	allowedPgSz := defaultPageHeaderSize
	for {
		view, err := rd.Peek(allowedPgSz)
		if err != nil && err != io.EOF {
			return err
		}

		if len(view) == 0 {
			return io.EOF
		}

		extra := 0
		if p.cryptoCtx.MetaDecryptor != nil {
			p.updateDecryption(p.cryptoCtx.MetaDecryptor, encryption.DictPageHeaderModule, p.dataPageHeaderAad)
			ciphertext, frameErr := encryption.FramedCiphertext(view)
			if errors.Is(frameErr, io.ErrUnexpectedEOF) {
				allowedPgSz *= 2
				if allowedPgSz > p.maxPageHeaderSize {
					return errors.New("parquet: deserializing page header failed")
				}
				continue
			}
			if frameErr != nil {
				return fmt.Errorf("reading encrypted page header: %w", frameErr)
			}
			view, err = p.cryptoCtx.MetaDecryptor.Decrypt(ciphertext)
			if err != nil {
				return fmt.Errorf("decrypting page header: %w", err)
			}
			extra = p.cryptoCtx.MetaDecryptor.CiphertextSizeDelta()
		}

		remaining, err := thrift.DeserializeThrift(hdr, view)
		if err != nil {
			allowedPgSz *= 2
			if allowedPgSz > p.maxPageHeaderSize {
				return errors.New("parquet: deserializing page header failed")
			}
			continue
		}
		rd.Discard(len(view) - int(remaining) + extra)
		break
	}
	return nil
}

func (p *serializedPageReader) SeekToPageWithRow(rowIdx int64) error {
	if rowIdx < 0 || rowIdx >= p.nrows {
		return fmt.Errorf("parquet: cannot seek column reader to row index %d", rowIdx)
	}

	// Release before repositioning p.r: a streaming page's drain-on-release would
	// otherwise read from the new offset.
	if p.curPage != nil {
		p.curPage.Release()
		p.curPage = nil
	}

	var (
		oidx metadata.OffsetIndex
		err  error
	)

	if p.pgIndexReader != nil {
		oidx, err = p.pgIndexReader.GetOffsetIndex(p.colIdx)
		if err != nil {
			return err
		}
	}

	section := p.r.Outer()
	if oidx == nil {
		if _, err = section.Seek(p.dataOffset-p.baseOffset, io.SeekStart); err != nil {
			return err
		}
		p.r.Reset(section)

		p.rowsSeen = 0
		p.pageOrd = 0

		for p.Next() && p.rowsSeen < rowIdx {
		}
		return p.err
	}

	pages := oidx.GetPageLocations()
	index := sort.Search(len(pages), func(i int) bool {
		return pages[i].FirstRowIndex > rowIdx
	}) - 1

	if index < 0 {
		return fmt.Errorf("parquet: seek out of range")
	}

	if _, err = section.Seek(pages[index].GetOffset()-p.baseOffset, io.SeekStart); err != nil {
		return err
	}

	p.r.Reset(section)
	p.rowsSeen, p.pageOrd = pages[index].FirstRowIndex, int16(index)
	p.Next()
	return p.err
}

func (p *serializedPageReader) valueStream(limit io.Reader, compressed bool) (io.Reader, io.Closer) {
	if compressed {
		dec := p.codec.(compress.StreamingCodec).NewReader(limit)
		return dec, dec
	}
	return limit, nil
}

// drainAndClose returns the cleanup a streaming ValueBuffer runs on Close: consume
// any unread compressed bytes for the current page and close the decompressor.
func drainAndClose(rawSrc io.Reader, closer io.Closer) func() error {
	return func() error {
		_, _ = io.Copy(io.Discard, rawSrc)
		if closer != nil {
			return closer.Close()
		}
		return nil
	}
}

// streamingThreshold is the smallest data-page size worth streaming (var so tests can lower it).
var streamingThreshold = int64(1 << 20)

func (p *serializedPageReader) pageCanStream(enc format.Encoding, lenUncompressed int) bool {
	return p.columnCanStream && enc == format.Encoding_PLAIN && p.cryptoCtx.DataDecryptor == nil &&
		int64(lenUncompressed) > streamingThreshold
}

// readLevelData reads the V1 rep+def level region, capped at maxBytes (the
// uncompressed page size) so a bogus length prefix can't force a huge allocation.
func (p *serializedPageReader) readLevelData(r io.Reader, repEnc, defEnc format.Encoding, numValues, maxBytes int) (*memory.Buffer, error) {
	buf := memory.NewResizableBuffer(p.mem)
	pos := 0
	readOne := func(enc format.Encoding, maxLvl int16) error {
		switch enc {
		case format.Encoding_RLE:
			var hdr [4]byte
			if _, err := io.ReadFull(r, hdr[:]); err != nil {
				return err
			}
			n := int(binary.LittleEndian.Uint32(hdr[:]))
			if n < 0 || pos+4+n > maxBytes {
				return fmt.Errorf("parquet: invalid V1 level region length %d", n)
			}
			buf.Resize(pos + 4 + n)
			copy(buf.Bytes()[pos:], hdr[:])
			if _, err := io.ReadFull(r, buf.Bytes()[pos+4:pos+4+n]); err != nil {
				return err
			}
			pos += 4 + n
		case format.Encoding_BIT_PACKED:
			bitWidth := bits.Len64(uint64(maxLvl))
			nbits, ok := utils.Mul(numValues, bitWidth)
			if !ok {
				return fmt.Errorf("parquet: V1 level count %d too large (corrupt page?)", numValues)
			}
			n := int(bitutil.BytesForBits(int64(nbits)))
			if n < 0 || pos+n > maxBytes {
				return fmt.Errorf("parquet: invalid V1 level region length %d", n)
			}
			buf.Resize(pos + n)
			if _, err := io.ReadFull(r, buf.Bytes()[pos:pos+n]); err != nil {
				return err
			}
			pos += n
		default:
			return fmt.Errorf("parquet: unsupported V1 level encoding %s for streaming", enc)
		}
		return nil
	}

	if p.maxRepLevel > 0 {
		if err := readOne(repEnc, p.maxRepLevel); err != nil {
			buf.Release()
			return nil, err
		}
	}
	if p.maxDefLevel > 0 {
		if err := readOne(defEnc, p.maxDefLevel); err != nil {
			buf.Release()
			return nil, err
		}
	}
	return buf, nil
}

func (p *serializedPageReader) Next() bool {
	// Loop here because there may be unhandled page types that we skip until
	// finding a page that we do know what to do with
	if p.curPage != nil {
		p.curPage.Release()
	}
	p.curPage = nil
	p.curPageHdr = format.NewPageHeader()
	p.err = nil

	for p.rowsSeen < p.nrows {
		if err := p.readPageHeader(p.r, p.curPageHdr); err != nil {
			if err != io.EOF {
				p.err = err
			}

			return false
		}

		lenCompressed, lenUncompressed, err := p.validatePageSizes(p.curPageHdr)
		if err != nil {
			p.err = err
			return false
		}

		if p.cryptoCtx.DataDecryptor != nil {
			p.updateDecryption(p.cryptoCtx.DataDecryptor, encryption.DictPageModule, p.dataPageAad)
		}

		switch p.curPageHdr.GetType() {
		case format.PageType_DICTIONARY_PAGE:
			p.cryptoCtx.StartDecryptWithDictionaryPage = false
			dictHeader := p.curPageHdr.GetDictionaryPageHeader()
			if dictHeader.GetNumValues() < 0 {
				p.err = errors.New("parquet: invalid page header (negative number of values)")
				return false
			}

			p.dictPageBuffer.ResizeNoShrink(lenUncompressed)
			buf := memory.NewBufferBytes(p.dictPageBuffer.Bytes())

			data, err := p.decompress(p.r, lenCompressed, buf.Bytes())
			if err != nil {
				p.err = err
				return false
			}
			if len(data) != lenUncompressed {
				p.err = fmt.Errorf("parquet: metadata said %d bytes uncompressed dictionary page, got %d bytes", lenUncompressed, len(data))
				return false
			}

			// make dictionary page
			p.curPage = &DictionaryPage{
				page: page{
					buf:      buf,
					typ:      p.curPageHdr.Type,
					nvals:    dictHeader.GetNumValues(),
					encoding: dictHeader.GetEncoding(),
				},
				sorted: dictHeader.IsSetIsSorted() && dictHeader.GetIsSorted(),
			}

		case format.PageType_DATA_PAGE:
			p.pageOrd++
			dataHeader := p.curPageHdr.GetDataPageHeader()
			if dataHeader.GetNumValues() < 0 {
				p.err = errors.New("parquet: invalid page header (negative number of values)")
				return false
			}

			firstRowIdx := p.rowsSeen
			p.rowsSeen += int64(dataHeader.GetNumValues())

			dp := &DataPageV1{
				dataPageBase: dataPageBase{page: page{
					typ:      p.curPageHdr.Type,
					nvals:    dataHeader.GetNumValues(),
					encoding: dataHeader.GetEncoding(),
				}},
				defLvlEncoding:   dataHeader.GetDefinitionLevelEncoding(),
				repLvlEncoding:   dataHeader.GetRepetitionLevelEncoding(),
				uncompressedSize: int32(lenUncompressed),
				statistics:       extractStats(dataHeader),
				firstRowIndex:    firstRowIdx,
			}

			if p.pageCanStream(dataHeader.GetEncoding(), lenUncompressed) {
				limit := io.LimitReader(p.r, int64(lenCompressed))
				// V1 compresses the whole body, so the levels are read through the
				// codec too (a no-op passthrough for an uncompressed column).
				valReader, closer := p.valueStream(limit, true)
				levelBuf, err := p.readLevelData(valReader, dataHeader.GetRepetitionLevelEncoding(),
					dataHeader.GetDefinitionLevelEncoding(), int(dataHeader.GetNumValues()), lenUncompressed)
				if err != nil {
					_ = drainAndClose(limit, closer)()
					p.err = err
					return false
				}
				dp.setStreamingValues(p.mem, lenUncompressed, levelBuf, valReader, limit, closer)
			} else {
				p.dataPageBuffer.ResizeNoShrink(lenUncompressed)
				buf := memory.NewBufferBytes(p.dataPageBuffer.Bytes())
				data, err := p.decompress(p.r, lenCompressed, buf.Bytes())
				if err != nil {
					p.err = err
					return false
				}
				if len(data) != lenUncompressed {
					p.err = fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, len(data))
					return false
				}
				dp.buf = buf
			}
			p.curPage = dp
		case format.PageType_DATA_PAGE_V2:
			p.pageOrd++
			dataHeader := p.curPageHdr.GetDataPageHeaderV2()
			if dataHeader.GetNumValues() < 0 {
				p.err = errors.New("parquet: invalid page header (negative number of values)")
				return false
			}

			if dataHeader.GetDefinitionLevelsByteLength() < 0 || dataHeader.GetRepetitionLevelsByteLength() < 0 {
				p.err = errors.New("parquet: invalid page header (negative levels byte length)")
				return false
			}

			compressed := dataHeader.GetIsCompressed()
			firstRowIdx := p.rowsSeen
			p.rowsSeen += int64(dataHeader.GetNumRows())
			levelsBytelen, ok := utils.Add(int(dataHeader.GetDefinitionLevelsByteLength()), int(dataHeader.GetRepetitionLevelsByteLength()))
			if !ok {
				p.err = errors.New("parquet: levels size too large (corrupt file?)")
				return false
			}
			if levelsBytelen > lenCompressed {
				p.err = fmt.Errorf("parquet: V2 level region %d exceeds page body %d (corrupt page?)", levelsBytelen, lenCompressed)
				return false
			}

			dp := &DataPageV2{
				dataPageBase: dataPageBase{page: page{
					typ:      p.curPageHdr.Type,
					nvals:    dataHeader.GetNumValues(),
					encoding: dataHeader.GetEncoding(),
				}},
				nulls:            dataHeader.GetNumNulls(),
				nrows:            dataHeader.GetNumRows(),
				defLvlByteLen:    dataHeader.GetDefinitionLevelsByteLength(),
				repLvlByteLen:    dataHeader.GetRepetitionLevelsByteLength(),
				compressed:       compressed,
				uncompressedSize: int32(lenUncompressed),
				statistics:       extractStats(dataHeader),
				firstRowIndex:    firstRowIdx,
			}

			if p.pageCanStream(dataHeader.GetEncoding(), lenUncompressed) {
				// V2: rep/def levels are uncompressed and precede the separately
				// compressed value region. Read the level bytes directly off the front,
				// then stream the values from what remains.
				limit := io.LimitReader(p.r, int64(lenCompressed))
				levelBuf := memory.NewResizableBuffer(p.mem)
				levelBuf.Resize(levelsBytelen)
				if _, err := io.ReadFull(limit, levelBuf.Bytes()); err != nil {
					levelBuf.Release()
					p.err = err
					return false
				}
				valReader, closer := p.valueStream(limit, compressed)
				dp.setStreamingValues(p.mem, lenUncompressed, levelBuf, valReader, limit, closer)
			} else {
				p.dataPageBuffer.ResizeNoShrink(lenUncompressed)
				buf := memory.NewBufferBytes(p.dataPageBuffer.Bytes())
				if p.cryptoCtx.DataDecryptor != nil {
					got, err := p.readV2Encrypted(p.r, lenCompressed, lenUncompressed, levelsBytelen, compressed, buf.Bytes())
					if err != nil {
						p.err = err
						return false
					}
					if got != lenUncompressed {
						p.err = fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, got)
						return false
					}
				} else {
					got, err := p.readV2Unencrypted(p.r, lenCompressed, lenUncompressed, levelsBytelen, compressed, buf.Bytes())
					if err != nil {
						p.err = err
						return false
					}
					if got != lenUncompressed {
						p.err = fmt.Errorf("parquet: metadata said %d bytes uncompressed data page, got %d bytes", lenUncompressed, got)
						return false
					}
				}
				dp.buf = buf
			}
			p.curPage = dp
		default:
			// we don't know this page type, we're allowed to skip non-data pages
			continue
		}

		return true
	}

	return false
}
