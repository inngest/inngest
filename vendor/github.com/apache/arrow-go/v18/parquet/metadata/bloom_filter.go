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

package metadata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/endian"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/apache/arrow-go/v18/internal/bitutils"
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/debug"
	"github.com/apache/arrow-go/v18/parquet/internal/encryption"
	format "github.com/apache/arrow-go/v18/parquet/internal/gen-go/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/thrift"
	"github.com/apache/arrow-go/v18/parquet/internal/utils"
	"github.com/apache/arrow-go/v18/parquet/schema"
	"github.com/cespare/xxhash/v2"
)

const (
	bytesPerFilterBlock      = 32
	bitsSetPerBlock          = 8
	minimumBloomFilterBytes  = bytesPerFilterBlock
	bloomFilterHashBatchSize = 1024
	// currently using 128MB as maximum size, should probably be reconsidered
	maximumBloomFilterBytes = 128 * 1024 * 1024
)

var (
	salt = [bitsSetPerBlock]uint32{
		0x47b6137b, 0x44974d91, 0x8824ad5b, 0xa2b7289d,
		0x705495c7, 0x2df1424b, 0x9efc4947, 0x5c6bfb31}

	booleanFalseHash = xxhash.Sum64([]byte{0})
	booleanTrueHash  = xxhash.Sum64([]byte{1})

	defaultHashStrategy = format.BloomFilterHash{XXHASH: &format.XxHash{}}
	defaultAlgorithm    = format.BloomFilterAlgorithm{BLOCK: &format.SplitBlockAlgorithm{}}
	defaultCompression  = format.BloomFilterCompression{UNCOMPRESSED: &format.Uncompressed{}}
)

func optimalNumBytes(ndv uint32, fpp float64) uint32 {
	optimalBits := optimalNumBits(ndv, fpp)
	debug.Assert(bitutil.IsMultipleOf8(int64(optimalBits)), "optimal bits should be multiple of 8")
	return optimalBits >> 3
}

func optimalNumBits(ndv uint32, fpp float64) uint32 {
	debug.Assert(fpp > 0 && fpp < 1, "false positive prob must be in (0, 1)")
	var (
		m       = -8 * float64(ndv) / math.Log(1-math.Pow(fpp, 1.0/8.0))
		numBits uint32
	)

	if m < 0 || m > maximumBloomFilterBytes<<3 {
		numBits = maximumBloomFilterBytes << 3
	} else {
		numBits = uint32(m)
	}

	// round up to lower bound
	if numBits < minimumBloomFilterBytes<<3 {
		numBits = minimumBloomFilterBytes << 3
	}

	// get next power of 2 if bits is not power of 2
	if (numBits & (numBits - 1)) != 0 {
		numBits = uint32(bitutil.NextPowerOf2(int(numBits)))
	}
	return numBits
}

type Hasher interface {
	Sum64(b []byte) uint64
	Sum64s(b [][]byte) []uint64
}

type xxhasher struct{}

func (xxhasher) Sum64(b []byte) uint64 {
	return xxhash.Sum64(b)
}

func (xxhasher) Sum64s(b [][]byte) (vals []uint64) {
	vals = make([]uint64, len(b))
	for i, v := range b {
		vals[i] = xxhash.Sum64(v)
	}
	return
}

func (xxhasher) Sum64sInto(b [][]byte, vals []uint64) {
	for i, v := range b {
		vals[i] = xxhash.Sum64(v)
	}
}

func precomputedBooleanHashes(h Hasher) (falseHash, trueHash uint64, ok bool) {
	if _, ok := h.(xxhasher); !ok {
		return 0, 0, false
	}
	return booleanFalseHash, booleanTrueHash, true
}

type sum64sIntoHasher interface {
	Sum64sInto([][]byte, []uint64)
}

func sum64s(h Hasher, b [][]byte, vals []uint64) []uint64 {
	if into, ok := h.(sum64sIntoHasher); ok {
		into.Sum64sInto(b, vals)
		return vals[:len(b)]
	}
	return h.Sum64s(b)
}

func GetHash[T parquet.ColumnTypes](h Hasher, v T) uint64 {
	return h.Sum64(getBytes(v))
}

func GetHashes[T parquet.ColumnTypes](h Hasher, vals []T) []uint64 {
	out := make([]uint64, len(vals))
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		rawBatch  [bloomFilterHashBatchSize * arrow.Int64SizeBytes]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)

	for offset := 0; offset < len(vals); offset += bloomFilterHashBatchSize {
		end := min(offset+bloomFilterHashBatchSize, len(vals))
		n := end - offset
		getBytesSliceInto(byteBatch[:n], rawBatch[:], vals[offset:end])
		hashes := sum64s(h, byteBatch[:n], hashBatch[:n])
		copy(out[offset:end], hashes)
	}
	return out
}

// InsertHashes hashes values in bounded batches and inserts each batch into the bloom filter.
func InsertHashes[T parquet.ColumnTypes](b BloomFilterBuilder, vals []T) {
	if len(vals) == 0 {
		return
	}

	h := b.Hasher()
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		rawBatch  [bloomFilterHashBatchSize * arrow.Int64SizeBytes]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)

	for offset := 0; offset < len(vals); offset += bloomFilterHashBatchSize {
		end := min(offset+bloomFilterHashBatchSize, len(vals))
		n := end - offset
		getBytesSliceInto(byteBatch[:n], rawBatch[:], vals[offset:end])
		b.InsertBulk(sum64s(h, byteBatch[:n], hashBatch[:n]))
	}
}

func GetSpacedHashes[T parquet.ColumnTypes](h Hasher, numValid int64, vals []T, validBits []byte, validBitsOffset int64) []uint64 {
	if numValid == 0 {
		return []uint64{}
	}

	out := make([]uint64, 0, numValid)
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		rawBatch  [bloomFilterHashBatchSize * arrow.Int64SizeBytes]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)

	// TODO: replace with bitset run reader pool
	setReader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, int64(len(vals)))
	for {
		run := setReader.NextRun()
		if run.Length == 0 {
			break
		}

		runEnd := run.Pos + run.Length
		for pos := run.Pos; pos < runEnd; pos += bloomFilterHashBatchSize {
			end := min(pos+int64(bloomFilterHashBatchSize), runEnd)
			n := int(end - pos)
			getBytesSliceInto(byteBatch[:n], rawBatch[:], vals[pos:end])
			hashes := sum64s(h, byteBatch[:n], hashBatch[:n])
			out = append(out, hashes...)
		}
	}
	return out
}

// InsertSpacedHashes hashes valid values in bounded batches and inserts each batch into the bloom filter.
func InsertSpacedHashes[T parquet.ColumnTypes](b BloomFilterBuilder, numValid int64, vals []T, validBits []byte, validBitsOffset int64) {
	if numValid == 0 {
		return
	}

	h := b.Hasher()
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		rawBatch  [bloomFilterHashBatchSize * arrow.Int64SizeBytes]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)

	// TODO: replace with bitset run reader pool
	setReader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, int64(len(vals)))
	for {
		run := setReader.NextRun()
		if run.Length == 0 {
			break
		}

		runEnd := run.Pos + run.Length
		for pos := run.Pos; pos < runEnd; pos += bloomFilterHashBatchSize {
			end := min(pos+int64(bloomFilterHashBatchSize), runEnd)
			n := int(end - pos)
			getBytesSliceInto(byteBatch[:n], rawBatch[:], vals[pos:end])
			b.InsertBulk(sum64s(h, byteBatch[:n], hashBatch[:n]))
		}
	}
}

// GetHashesFromBitmap computes hashes for boolean values directly from a bitmap
// without converting to []bool, avoiding 8x memory overhead.
func GetHashesFromBitmap(h Hasher, bitmap []byte, bitmapOffset int64, numValues int64) []uint64 {
	if numValues == 0 {
		return []uint64{}
	}

	out := make([]uint64, numValues)
	if falseHash, trueHash, ok := precomputedBooleanHashes(h); ok {
		for i := range numValues {
			if bitutil.BitIsSet(bitmap, int(bitmapOffset+i)) {
				out[i] = trueHash
			} else {
				out[i] = falseHash
			}
		}
		return out
	}

	// Convert each bool bit to []byte for hashing
	// Reuse a single-byte slice to avoid allocating per value
	b := []byte{0}
	for i := range numValues {
		val := bitutil.BitIsSet(bitmap, int(bitmapOffset+i))
		// Hash the boolean as a single byte (0x00 or 0x01)
		if val {
			b[0] = 1
		} else {
			b[0] = 0
		}
		out[i] = h.Sum64(b)
	}
	return out
}

// GetSpacedHashesFromBitmap computes hashes for boolean values from a bitmap with validity information,
// only hashing valid (non-null) values. Returns hashes for numValid values.
// This avoids the 8x memory overhead of converting to []bool.
func GetSpacedHashesFromBitmap(h Hasher, numValid int64, bitmap []byte, bitmapOffset int64, numValues int64, validBits []byte, validBitsOffset int64) []uint64 {
	if numValid == 0 {
		return []uint64{}
	}

	out := make([]uint64, 0, numValid)
	falseHash, trueHash, usePrecomputedHashes := precomputedBooleanHashes(h)

	// Use SetBitRunReader to efficiently iterate over valid values
	setReader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, numValues)
	if usePrecomputedHashes {
		for {
			run := setReader.NextRun()
			if run.Length == 0 {
				break
			}

			for i := range run.Length {
				if bitutil.BitIsSet(bitmap, int(bitmapOffset+run.Pos+i)) {
					out = append(out, trueHash)
				} else {
					out = append(out, falseHash)
				}
			}
		}
		return out
	}

	// Reuse a single-byte slice to avoid allocating per value
	b := []byte{0}
	for {
		run := setReader.NextRun()
		if run.Length == 0 {
			break
		}

		// Hash each valid bit in this run
		for i := range run.Length {
			val := bitutil.BitIsSet(bitmap, int(bitmapOffset+run.Pos+i))
			if val {
				b[0] = 1
			} else {
				b[0] = 0
			}
			out = append(out, h.Sum64(b))
		}
	}
	return out
}

// InsertHashesFromBitmap hashes boolean values in bounded batches and inserts each batch into the bloom filter.
func InsertHashesFromBitmap(b BloomFilterBuilder, bitmap []byte, bitmapOffset int64, numValues int64) {
	if numValues == 0 {
		return
	}

	h := b.Hasher()
	falseHash, trueHash, usePrecomputedHashes := precomputedBooleanHashes(h)
	var (
		hashBatch [bloomFilterHashBatchSize]uint64
		value     [1]byte
	)

	if usePrecomputedHashes {
		for offset := int64(0); offset < numValues; offset += bloomFilterHashBatchSize {
			end := min(offset+int64(bloomFilterHashBatchSize), numValues)
			for i := offset; i < end; i++ {
				if bitutil.BitIsSet(bitmap, int(bitmapOffset+i)) {
					hashBatch[i-offset] = trueHash
				} else {
					hashBatch[i-offset] = falseHash
				}
			}
			b.InsertBulk(hashBatch[:end-offset])
		}
		return
	}

	for offset := int64(0); offset < numValues; offset += bloomFilterHashBatchSize {
		end := min(offset+int64(bloomFilterHashBatchSize), numValues)
		for i := offset; i < end; i++ {
			if bitutil.BitIsSet(bitmap, int(bitmapOffset+i)) {
				value[0] = 1
			} else {
				value[0] = 0
			}
			hashBatch[i-offset] = h.Sum64(value[:])
		}
		b.InsertBulk(hashBatch[:end-offset])
	}
}

// InsertSpacedHashesFromBitmap hashes valid boolean values in bounded batches and inserts each batch into the bloom filter.
func InsertSpacedHashesFromBitmap(b BloomFilterBuilder, numValid int64, bitmap []byte, bitmapOffset int64, numValues int64, validBits []byte, validBitsOffset int64) {
	if numValid == 0 {
		return
	}

	h := b.Hasher()
	falseHash, trueHash, usePrecomputedHashes := precomputedBooleanHashes(h)
	var (
		hashBatch [bloomFilterHashBatchSize]uint64
		value     [1]byte
	)

	setReader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, numValues)
	if usePrecomputedHashes {
		for {
			run := setReader.NextRun()
			if run.Length == 0 {
				break
			}

			runEnd := run.Pos + run.Length
			for pos := run.Pos; pos < runEnd; pos += bloomFilterHashBatchSize {
				end := min(pos+int64(bloomFilterHashBatchSize), runEnd)
				for i := pos; i < end; i++ {
					if bitutil.BitIsSet(bitmap, int(bitmapOffset+i)) {
						hashBatch[i-pos] = trueHash
					} else {
						hashBatch[i-pos] = falseHash
					}
				}
				b.InsertBulk(hashBatch[:end-pos])
			}
		}
		return
	}

	for {
		run := setReader.NextRun()
		if run.Length == 0 {
			break
		}

		runEnd := run.Pos + run.Length
		for pos := run.Pos; pos < runEnd; pos += bloomFilterHashBatchSize {
			end := min(pos+int64(bloomFilterHashBatchSize), runEnd)
			for i := pos; i < end; i++ {
				if bitutil.BitIsSet(bitmap, int(bitmapOffset+i)) {
					value[0] = 1
				} else {
					value[0] = 0
				}
				hashBatch[i-pos] = h.Sum64(value[:])
			}
			b.InsertBulk(hashBatch[:end-pos])
		}
	}
}

func getBytes[T parquet.ColumnTypes](v T) []byte {
	switch v := any(v).(type) {
	case int32:
		if endian.IsBigEndian {
			var out [arrow.Int32SizeBytes]byte
			binary.LittleEndian.PutUint32(out[:], uint32(v))
			return out[:]
		}
	case int64:
		if endian.IsBigEndian {
			var out [arrow.Int64SizeBytes]byte
			binary.LittleEndian.PutUint64(out[:], uint64(v))
			return out[:]
		}
	case float32:
		if endian.IsBigEndian {
			var out [arrow.Float32SizeBytes]byte
			binary.LittleEndian.PutUint32(out[:], math.Float32bits(v))
			return out[:]
		}
	case float64:
		if endian.IsBigEndian {
			var out [arrow.Float64SizeBytes]byte
			binary.LittleEndian.PutUint64(out[:], math.Float64bits(v))
			return out[:]
		}
	case parquet.ByteArray:
		return v
	case parquet.FixedLenByteArray:
		return v
	case parquet.Int96:
		return v[:]
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(&v)), unsafe.Sizeof(v))
}

func getBytesSliceInto[T parquet.ColumnTypes](b [][]byte, raw []byte, v []T) {
	switch v := any(v).(type) {
	case []int32:
		if endian.IsBigEndian {
			for i, vv := range v {
				value := raw[i*arrow.Int32SizeBytes : (i+1)*arrow.Int32SizeBytes]
				binary.LittleEndian.PutUint32(value, uint32(vv))
				b[i] = value
			}
			return
		}
	case []int64:
		if endian.IsBigEndian {
			for i, vv := range v {
				value := raw[i*arrow.Int64SizeBytes : (i+1)*arrow.Int64SizeBytes]
				binary.LittleEndian.PutUint64(value, uint64(vv))
				b[i] = value
			}
			return
		}
	case []float32:
		if endian.IsBigEndian {
			for i, vv := range v {
				value := raw[i*arrow.Float32SizeBytes : (i+1)*arrow.Float32SizeBytes]
				binary.LittleEndian.PutUint32(value, math.Float32bits(vv))
				b[i] = value
			}
			return
		}
	case []float64:
		if endian.IsBigEndian {
			for i, vv := range v {
				value := raw[i*arrow.Float64SizeBytes : (i+1)*arrow.Float64SizeBytes]
				binary.LittleEndian.PutUint64(value, math.Float64bits(vv))
				b[i] = value
			}
			return
		}
	case []parquet.ByteArray:
		for i, vv := range v {
			b[i] = vv
		}
		return
	case []parquet.FixedLenByteArray:
		for i, vv := range v {
			b[i] = vv
		}
		return
	case []parquet.Int96:
		for i, vv := range v {
			b[i] = vv[:]
		}
		return
	}

	var z T
	sz, ptr := int(unsafe.Sizeof(z)), unsafe.SliceData(v)
	rawValues := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), sz*len(v))
	for i := range b {
		b[i] = rawValues[i*sz : (i+1)*sz]
	}
}

type blockSplitBloomFilter struct {
	data     *memory.Buffer
	bitset32 []uint32

	hasher        Hasher
	algorithm     format.BloomFilterAlgorithm
	hashStrategy  format.BloomFilterHash
	compression   format.BloomFilterCompression
	cancelCleanup func()
}

func (b *blockSplitBloomFilter) getAlg() *format.BloomFilterAlgorithm {
	return &b.algorithm
}

func (b *blockSplitBloomFilter) getHashStrategy() *format.BloomFilterHash {
	return &b.hashStrategy
}

func (b *blockSplitBloomFilter) getCompression() *format.BloomFilterCompression {
	return &b.compression
}

func (b *blockSplitBloomFilter) CheckHash(hash uint64) bool {
	return checkHash(b.bitset32, hash)
}

func (b *blockSplitBloomFilter) CheckBulk(hashes []uint64) []bool {
	results := make([]bool, len(hashes))
	checkBulk(b.bitset32, hashes, results)
	return results
}

func (b *blockSplitBloomFilter) InsertHash(hash uint64) {
	insertHash(b.bitset32, hash)
}

func (b *blockSplitBloomFilter) InsertBulk(hashes []uint64) {
	insertBulk(b.bitset32, hashes)
}

func (b *blockSplitBloomFilter) Hasher() Hasher {
	return b.hasher
}

func (b *blockSplitBloomFilter) Size() int64 {
	return int64(len(b.bitset32) * 4)
}

func (b *blockSplitBloomFilter) Release() {
	if b.cancelCleanup != nil {
		b.cancelCleanup()
		b.cancelCleanup = nil
	}
	if b.data != nil {
		b.data.Release()
		b.data = nil
		b.bitset32 = nil
	}
}

func (b *blockSplitBloomFilter) WriteTo(w io.Writer, enc encryption.Encryptor) (int, error) {
	if enc != nil {
		n := enc.Encrypt(w, b.data.Bytes())
		return n, nil
	}
	return w.Write(b.data.Bytes())
}

func NewBloomFilter(numBytes, maxBytes uint32, mem memory.Allocator) BloomFilterBuilder {
	maxBytes = max(minimumBloomFilterBytes, min(maximumBloomFilterBytes, maxBytes))
	if numBytes < minimumBloomFilterBytes {
		numBytes = minimumBloomFilterBytes
	}

	if numBytes > maxBytes {
		numBytes = maxBytes
	}

	// get next power of 2 if it's not a power of 2
	if (numBytes & (numBytes - 1)) != 0 {
		numBytes = uint32(bitutil.NextPowerOf2(int(numBytes)))
	}

	buf := memory.NewResizableBuffer(mem)
	buf.ResizeNoShrink(int(numBytes))
	bf := &blockSplitBloomFilter{
		data:         buf,
		bitset32:     arrow.Uint32Traits.CastFromBytes(buf.Bytes()),
		hasher:       xxhasher{},
		algorithm:    format.BloomFilterAlgorithm{BLOCK: &format.SplitBlockAlgorithm{}},
		hashStrategy: format.BloomFilterHash{XXHASH: &format.XxHash{}},
		compression:  format.BloomFilterCompression{UNCOMPRESSED: &format.Uncompressed{}},
	}
	addCleanup(bf, nil)
	return bf
}

func NewBloomFilterFromNDVAndFPP(ndv uint32, fpp float64, maxBytes int64, mem memory.Allocator) BloomFilterBuilder {
	if fpp <= 0 || fpp >= 1 || math.IsNaN(fpp) {
		panic("parquet: bloom filter false-positive probability must be in (0, 1)")
	}
	maxBytes = max(minimumBloomFilterBytes, min(maximumBloomFilterBytes, maxBytes))
	numBytes := optimalNumBytes(ndv, fpp)
	return NewBloomFilter(numBytes, uint32(maxBytes), mem)
}

type BloomFilterBuilder interface {
	Hasher() Hasher
	Size() int64
	// Release immediately frees buffers owned by the builder. It is safe to
	// call more than once. Builders otherwise release their buffers during GC.
	Release()
	InsertHash(hash uint64)
	InsertBulk(hashes []uint64)
	WriteTo(io.Writer, encryption.Encryptor) (int, error)

	getAlg() *format.BloomFilterAlgorithm
	getHashStrategy() *format.BloomFilterHash
	getCompression() *format.BloomFilterCompression
}

type BloomFilter interface {
	Hasher() Hasher
	CheckHash(hash uint64) bool
	Size() int64
}

type TypedBloomFilter[T parquet.ColumnTypes] struct {
	BloomFilter
}

func (b *TypedBloomFilter[T]) Check(v T) bool {
	h := b.Hasher()
	return b.CheckHash(h.Sum64(getBytes(v)))
}

func validateBloomFilterHeader(hdr *format.BloomFilterHeader) error {
	if hdr == nil {
		return errors.New("bloom filter header must not be nil")
	}

	if !hdr.Algorithm.IsSetBLOCK() {
		return fmt.Errorf("unsupported bloom filter algorithm: %s", hdr.Algorithm)
	}

	if !hdr.Compression.IsSetUNCOMPRESSED() {
		return fmt.Errorf("unsupported bloom filter compression: %s", hdr.Compression)
	}

	if !hdr.Hash.IsSetXXHASH() {
		return fmt.Errorf("unsupported bloom filter hash strategy: %s", hdr.Hash)
	}

	if hdr.NumBytes < minimumBloomFilterBytes || hdr.NumBytes > maximumBloomFilterBytes {
		return fmt.Errorf("invalid bloom filter size: %d", hdr.NumBytes)
	}

	return nil
}

type BloomFilterReader struct {
	Input         parquet.ReaderAtSeeker
	FileMetadata  *FileMetaData
	Props         *parquet.ReaderProperties
	FileDecryptor encryption.FileDecryptor
	BufferPool    *sync.Pool
}

func (r *BloomFilterReader) RowGroup(i int) (*RowGroupBloomFilterReader, error) {
	if i < 0 || i >= len(r.FileMetadata.RowGroups) {
		return nil, fmt.Errorf("row group index %d out of range", i)
	}

	rgMeta := r.FileMetadata.RowGroup(i)
	return &RowGroupBloomFilterReader{
		input:          r.Input,
		rgMeta:         rgMeta,
		fileDecryptor:  r.FileDecryptor,
		rgOrdinal:      int16(i),
		bufferPool:     r.BufferPool,
		sourceFileSize: r.FileMetadata.sourceFileSize,
	}, nil
}

type RowGroupBloomFilterReader struct {
	input          parquet.ReaderAtSeeker
	rgMeta         *RowGroupMetaData
	fileDecryptor  encryption.FileDecryptor
	rgOrdinal      int16
	sourceFileSize int64

	bufferPool *sync.Pool
}

func (r *RowGroupBloomFilterReader) GetColumnBloomFilter(i int) (BloomFilter, error) {
	if i < 0 || i >= r.rgMeta.NumColumns() {
		return nil, fmt.Errorf("column index %d out of range", i)
	}

	col, err := r.rgMeta.ColumnChunk(i)
	if err != nil {
		return nil, err
	}

	var (
		decryptor           encryption.Decryptor
		header              format.BloomFilterHeader
		offset              int64
		bloomFilterReadSize int32 = 256
	)

	if offset = col.BloomFilterOffset(); offset <= 0 {
		return nil, nil
	}

	if col.BloomFilterLength() > 0 {
		bloomFilterReadSize = col.BloomFilterLength()
	}

	sectionRdr := io.NewSectionReader(r.input, offset, r.sourceFileSize-offset)
	cryptoMetadata := col.CryptoMetadata()
	if cryptoMetadata != nil {
		decryptor, err = encryption.GetColumnMetaDecryptor(cryptoMetadata, r.fileDecryptor)
		if err != nil {
			return nil, err
		}

		encryption.UpdateDecryptor(decryptor, r.rgOrdinal, int16(i),
			encryption.BloomFilterHeaderModule)
		hdr, err := decryptor.DecryptFrom(sectionRdr, maximumBloomFilterBytes)
		if err != nil {
			return nil, fmt.Errorf("decrypting bloom filter header: %w", err)
		}
		if _, err = thrift.DeserializeThrift(&header, hdr); err != nil {
			return nil, err
		}

		if err = validateBloomFilterHeader(&header); err != nil {
			return nil, err
		}

		encryption.UpdateDecryptor(decryptor, r.rgOrdinal, int16(i),
			encryption.BloomFilterBitsetModule)
		bitset, err := decryptor.DecryptFrom(sectionRdr, int64(header.NumBytes)+int64(decryptor.CiphertextSizeDelta()))
		if err != nil {
			return nil, fmt.Errorf("decrypting bloom filter bitset: %w", err)
		}
		if len(bitset) != int(header.NumBytes) {
			return nil, fmt.Errorf("wrong length of decrypted bloom filter bitset: %d vs %d",
				len(bitset), header.NumBytes)
		}

		return &blockSplitBloomFilter{
			data:         memory.NewBufferBytes(bitset),
			bitset32:     arrow.Uint32Traits.CastFromBytes(bitset),
			hasher:       xxhasher{},
			algorithm:    *header.Algorithm,
			hashStrategy: *header.Hash,
			compression:  *header.Compression,
		}, nil
	}

	headerBuf := r.bufferPool.Get().(*memory.Buffer)
	headerBuf.ResizeNoShrink(int(bloomFilterReadSize))
	if sectionSize := sectionRdr.Size(); sectionSize < int64(headerBuf.Len()) {
		headerBuf.ResizeNoShrink(int(sectionSize))
	}
	defer func() {
		if headerBuf != nil {
			headerBuf.ResizeNoShrink(0)
			r.bufferPool.Put(headerBuf)
		}
	}()

	if _, err = io.ReadFull(sectionRdr, headerBuf.Bytes()); err != nil {
		return nil, err
	}

	remaining, err := thrift.DeserializeThrift(&header, headerBuf.Bytes())
	if err != nil {
		return nil, err
	}
	headerSize := len(headerBuf.Bytes()) - int(remaining)

	if err = validateBloomFilterHeader(&header); err != nil {
		return nil, err
	}

	bloomFilterSz := header.NumBytes
	var bitset []byte
	if int(bloomFilterSz)+headerSize <= len(headerBuf.Bytes()) {
		// bloom filter data is entirely contained in the buffer we just read
		bitset = headerBuf.Bytes()[headerSize : headerSize+int(bloomFilterSz)]
	} else {
		buf := r.bufferPool.Get().(*memory.Buffer)
		buf.ResizeNoShrink(int(bloomFilterSz))
		filterBytesInHeader := headerBuf.Len() - headerSize
		if filterBytesInHeader > 0 {
			copy(buf.Bytes(), headerBuf.Bytes()[headerSize:])
		}

		if _, err = io.ReadFull(sectionRdr, buf.Bytes()[filterBytesInHeader:]); err != nil {
			buf.ResizeNoShrink(0)
			r.bufferPool.Put(buf)
			return nil, err
		}
		bitset = buf.Bytes()
		headerBuf.ResizeNoShrink(0)
		r.bufferPool.Put(headerBuf)
		headerBuf = buf
	}

	bf := &blockSplitBloomFilter{
		data:         headerBuf,
		bitset32:     arrow.GetData[uint32](bitset),
		hasher:       xxhasher{},
		algorithm:    *header.Algorithm,
		hashStrategy: *header.Hash,
		compression:  *header.Compression,
	}
	headerBuf = nil
	addCleanup(bf, r.bufferPool)
	return bf, nil
}

// VisitColumnBloomFilter invokes fn for the BloomFilter of the column at index i.
//
// Lifetime contract: The BloomFilter passed to fn (and its backing bitset)
// is recycled immediately after fn returns. Callers must not retain, store,
// or use the BloomFilter or its data outside the scope of the fn function.
func (r *RowGroupBloomFilterReader) VisitColumnBloomFilter(i int, fn func(BloomFilter) error) error {
	bf, err := r.GetColumnBloomFilter(i)
	if err != nil || bf == nil {
		return err
	}

	defer r.recycle(bf)

	return fn(bf)
}

func (r *RowGroupBloomFilterReader) recycle(bf BloomFilter) {
	if bf == nil {
		return
	}

	b, ok := bf.(*blockSplitBloomFilter)
	if !ok {
		return
	}

	b.bitset32 = nil

	if r.bufferPool != nil && b.cancelCleanup != nil {
		// Stop the GC cleanup so it can't return b.data to the pool a second time.
		b.cancelCleanup()
		b.cancelCleanup = nil
		b.data.ResizeNoShrink(0)
		r.bufferPool.Put(b.data)
	} else if b.data != nil {
		b.data.Release()
	}

	b.data = nil
}

type FileBloomFilterBuilder struct {
	Schema    *schema.Schema
	Encryptor encryption.FileEncryptor

	rgMetaBldrs  []*RowGroupMetaDataBuilder
	bloomFilters []map[string]BloomFilterBuilder
}

func (f *FileBloomFilterBuilder) AppendRowGroup(rgMeta *RowGroupMetaDataBuilder, filters map[string]BloomFilterBuilder) {
	f.rgMetaBldrs = append(f.rgMetaBldrs, rgMeta)
	f.bloomFilters = append(f.bloomFilters, filters)
}

func (f *FileBloomFilterBuilder) WriteTo(w utils.WriterTell) error {
	if len(f.rgMetaBldrs) == 0 || len(f.bloomFilters) == 0 {
		return nil
	}

	var (
		hdr        format.BloomFilterHeader
		serializer = thrift.NewThriftSerializer()
	)
	for rg, rgMeta := range f.rgMetaBldrs {
		if len(f.bloomFilters[rg]) == 0 {
			continue
		}

		for c, col := range rgMeta.colBuilders {
			colPath := col.column.Path()
			bf, ok := f.bloomFilters[rg][colPath]
			if !ok || bf == nil {
				continue
			}

			offset := w.Tell()
			col.chunk.MetaData.BloomFilterOffset = &offset
			var encryptor encryption.Encryptor
			if f.Encryptor != nil {
				encryptor = f.Encryptor.GetColumnMetaEncryptor(colPath)
			}

			if encryptor != nil {
				encryptor.UpdateAad(encryption.CreateModuleAad(
					encryptor.FileAad(), encryption.BloomFilterHeaderModule,
					int16(rg), int16(c), encryption.NonPageOrdinal))
			}

			hdr.NumBytes = int32(bf.Size())
			hdr.Algorithm = bf.getAlg()
			hdr.Hash = bf.getHashStrategy()
			hdr.Compression = bf.getCompression()

			_, err := serializer.Serialize(&hdr, w, encryptor)
			if err != nil {
				return err
			}

			if encryptor != nil {
				encryptor.UpdateAad(encryption.CreateModuleAad(
					encryptor.FileAad(), encryption.BloomFilterBitsetModule,
					int16(rg), int16(c), encryption.NonPageOrdinal))
			}

			if _, err = bf.WriteTo(w, encryptor); err != nil {
				return err
			}

			dataWritten := int32(w.Tell() - offset)
			col.chunk.MetaData.BloomFilterLength = &dataWritten
		}
	}
	return nil
}
