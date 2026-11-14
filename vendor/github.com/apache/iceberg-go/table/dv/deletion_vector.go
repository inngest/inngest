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

package dv

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"math"
	"slices"
	"strconv"

	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	iceio "github.com/apache/iceberg-go/io"
	"github.com/apache/iceberg-go/puffin"
)

// dvCardinalityProperty is the spec-mandated blob property name carrying the
// number of deleted row positions encoded in the inner roaring bitmap. The
// writer attaches it; the reader uses it to detect truncated bitmaps whose
// CRC happens to validate (the CRC covers the bytes that ARE present, not
// the bytes that should have been).
const (
	dvCardinalityProperty        = "cardinality"
	dvReferencedDataFileProperty = "referenced-data-file"
)

var ErrInvalidDeletionVector = errors.New("invalid deletion vector")

const (
	// DVMagicNumber is the magic number for deletion vectors.
	// Spec bytes: D1 D3 39 64 (big-endian) = 0x6439D3D1 (little-endian uint32)
	DVMagicNumber uint32 = 0x6439D3D1

	dvLengthSize = 4 // length field
	dvMagicSize  = 4 // magic field
	dvCRCSize    = 4 // CRC-32 checksum
	dvMinSize    = dvLengthSize + dvMagicSize + dvCRCSize

	// maxCoalescedDVRangeSize caps a range only when it is extended with
	// another blob. A single blob may still be up to DefaultMaxBlobSize.
	maxCoalescedDVRangeSize int64 = 8 << 20
)

// DeserializeDV parses a deletion vector blob and returns a bitmap of deleted positions.
//
// The DV binary format is:
//   - Length (4 bytes, big-endian): size of magic + bitmap data, excluding CRC-32
//   - Magic  (4 bytes, little-endian): must be 0x6439D3D1
//   - Bitmap (variable): roaring bitmap in Iceberg portable format
//   - CRC-32 (4 bytes, big-endian): checksum over magic + bitmap
//
// If expectedCardinality >= 0, the bitmap's cardinality is validated against it.
func DeserializeDV(data []byte, expectedCardinality int64) (*RoaringPositionBitmap, error) {
	if len(data) < dvMinSize {
		return nil, fmt.Errorf("deletion vector payload too short: %d bytes (minimum %d)", len(data), dvMinSize)
	}

	// 1. Read and validate length
	length := binary.BigEndian.Uint32(data[0:dvLengthSize])
	expectedLength := uint32(len(data) - dvLengthSize - dvCRCSize)
	if length != expectedLength {
		return nil, fmt.Errorf("deletion vector length mismatch: got %d, expected %d", length, expectedLength)
	}

	// 2. Read and validate magic
	magic := binary.LittleEndian.Uint32(data[dvLengthSize : dvLengthSize+dvMagicSize])
	if magic != DVMagicNumber {
		return nil, fmt.Errorf("invalid deletion vector magic: 0x%08x, expected 0x%08x", magic, DVMagicNumber)
	}

	// 3. Verify CRC-32 over magic + bitmap (bytes 4 to len-4)
	bitmapDataStart := dvLengthSize
	bitmapDataEnd := len(data) - dvCRCSize
	computedCRC := crc32.ChecksumIEEE(data[bitmapDataStart:bitmapDataEnd])
	expectedCRC := binary.BigEndian.Uint32(data[bitmapDataEnd:])
	if computedCRC != expectedCRC {
		return nil, fmt.Errorf("deletion vector CRC mismatch: computed 0x%08x, expected 0x%08x", computedCRC, expectedCRC)
	}

	// 4. Deserialize roaring bitmap from the inner bytes (after length + magic, before CRC)
	roaringStart := dvLengthSize + dvMagicSize
	bitmap, err := DeserializeRoaringPositionBitmap(data[roaringStart:bitmapDataEnd])
	if err != nil {
		return nil, fmt.Errorf("deserialize deletion vector bitmap: %w", err)
	}

	// 5. Validate cardinality if requested
	if expectedCardinality >= 0 {
		actual := bitmap.Cardinality()
		if actual != expectedCardinality {
			return nil, fmt.Errorf("deletion vector cardinality mismatch: got %d, expected %d", actual, expectedCardinality)
		}
	}

	return bitmap, nil
}

// SerializeDV produces the spec-format DV binary envelope from a bitmap:
//
//   - Length (4 bytes, big-endian): size of magic + bitmap data, excluding CRC-32
//   - Magic  (4 bytes, little-endian): DVMagicNumber
//   - Bitmap (variable): roaring bitmap in Iceberg portable format
//   - CRC-32 (4 bytes, big-endian): checksum over magic + bitmap
//
// The bitmap is run-length encoded in place before it is written, matching
// Java's BitmapPositionDeleteIndex.serialize.
func SerializeDV(bitmap *RoaringPositionBitmap) ([]byte, error) {
	bitmap.RunLengthEncode()

	serializedSize := bitmap.serializedSize()
	innerLen := uint64(dvMagicSize) + serializedSize
	if innerLen > math.MaxUint32 {
		return nil, fmt.Errorf("deletion vector payload too large: %d bytes", innerLen)
	}

	totalSize := uint64(dvLengthSize+dvMagicSize+dvCRCSize) + serializedSize
	if totalSize > uint64(math.MaxInt) {
		return nil, fmt.Errorf("deletion vector serialized size %d exceeds maximum supported size %d", totalSize, math.MaxInt)
	}

	var buf bytes.Buffer
	buf.Grow(int(totalSize))

	var header [dvLengthSize + dvMagicSize]byte
	binary.LittleEndian.PutUint32(header[dvLengthSize:], DVMagicNumber)
	_, _ = buf.Write(header[:])

	if err := bitmap.Serialize(&buf); err != nil {
		return nil, fmt.Errorf("serialize roaring bitmap: %w", err)
	}

	bitmapDataEnd := buf.Len()
	innerLen = uint64(bitmapDataEnd - dvLengthSize)
	if innerLen > math.MaxUint32 {
		return nil, fmt.Errorf("deletion vector payload too large: %d bytes", innerLen)
	}

	crc := crc32.ChecksumIEEE(buf.Bytes()[dvLengthSize:bitmapDataEnd])

	var trailer [dvCRCSize]byte
	binary.BigEndian.PutUint32(trailer[:], crc)
	_, _ = buf.Write(trailer[:])

	out := buf.Bytes()
	binary.BigEndian.PutUint32(out[:dvLengthSize], uint32(innerLen))

	return out, nil
}

// ReadDV reads a deletion vector from a puffin file using the manifest entry metadata.
// ContentOffset and ContentSizeInBytes must be set on the DataFile (required by v3 spec).
//
// The decoded bitmap's cardinality is cross-validated against two independent
// sources, so a truncated or partially-overwritten blob whose CRC still
// validates over the bytes that are present is rejected:
//
//   - The manifest entry's record_count (dvFile.Count()). This is field 103, a
//     required non-nullable long, so it is always available — zero means an
//     empty deletion vector, not "unknown". Java's BitmapPositionDeleteIndex
//     validates against this value, so it is our primary expected cardinality.
//   - The puffin blob's spec-mandated `cardinality` property, when present.
//
// When both sources are available they must agree; a disagreement (e.g. a
// stale manifest record_count against a freshly written blob) is a writer bug
// and fails fast. The bitmap is then validated against the manifest count.
//
// ReadDV also requires the selected blob to be a deletion-vector blob whose
// referenced-data-file property matches the manifest. This is stricter than
// Java's DVUtil.readDV and PyIceberg, which read by offset and size without
// validating blob type or referenced-data-file, but prevents a valid Puffin
// blob for another data file from being applied here. A missing or empty
// referenced-data-file property is fatal because it cannot establish blob
// identity; a missing cardinality property is only warned about because the
// manifest record_count still bounds the decoded bitmap. The compression
// codec is also rejected because deletion-vector-v1 stores raw bytes here.
//
// Blobs missing the spec-required cardinality property are still validated
// against the manifest record_count and accepted with a slog warning rather
// than rejected — the Go writer always emits the property, but third-party
// writers may not, and the per-byte CRC check in DeserializeDV still applies.
func ReadDV(fs iceio.IO, dvFile iceberg.DataFile) (*RoaringPositionBitmap, error) {
	if err := validateDVFile(dvFile); err != nil {
		return nil, err
	}

	reader, f, err := openDVReader(fs, dvFile.FilePath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, _, manifestReferencedDataFile, contentOffset, contentSize := iceberginternal.BorrowedDataFilePointers(dvFile)
	offset, size := *contentOffset, *contentSize
	blob, err := findBlobMetadataByRange(reader.Blobs(), offset, size)
	if err != nil {
		return nil, fmt.Errorf("%w: DV file %s: %w", ErrInvalidDeletionVector, dvFile.FilePath(), err)
	}

	return readDV(reader, blob, dvFile, offset, manifestReferencedDataFile)
}

// ReadDVs reads multiple deletion vectors stored in the same Puffin file.
// The returned bitmaps have the same order as dvFiles. The Puffin file is
// opened and its footer is decoded once for the complete batch.
func ReadDVs(fs iceio.IO, dvFiles []iceberg.DataFile) ([]*RoaringPositionBitmap, error) {
	if len(dvFiles) == 0 {
		return nil, nil
	}
	if len(dvFiles) == 1 {
		bitmap, err := ReadDV(fs, dvFiles[0])
		if err != nil {
			return nil, err
		}

		return []*RoaringPositionBitmap{bitmap}, nil
	}

	filePath := dvFiles[0].FilePath()
	for i, dvFile := range dvFiles {
		if dvFile.FilePath() != filePath {
			return nil, fmt.Errorf("%w: deletion vector at index %d uses Puffin file %q, expected %q",
				iceberg.ErrInvalidArgument, i, dvFile.FilePath(), filePath)
		}
		if err := validateDVFile(dvFile); err != nil {
			return nil, err
		}
	}

	reader, f, err := openDVReader(fs, filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	blobsByOffset := indexBlobMetadataByOffset(reader.Blobs())
	type dvBlobRead struct {
		index               int
		blob                puffin.BlobMetadata
		offset              int64
		manifestCardinality int64
	}

	reads := make([]dvBlobRead, len(dvFiles))
	for i, dvFile := range dvFiles {
		_, _, manifestReferencedDataFile, contentOffset, contentSize := iceberginternal.BorrowedDataFilePointers(dvFile)
		offset, size := *contentOffset, *contentSize
		blob, err := findIndexedBlobMetadataByRange(blobsByOffset, offset, size)
		if err != nil {
			return nil, fmt.Errorf("%w: DV file %s: %w", ErrInvalidDeletionVector, dvFile.FilePath(), err)
		}

		manifestCardinality, err := validateDVBlobMetadata(blob, dvFile, offset, manifestReferencedDataFile)
		if err != nil {
			return nil, err
		}

		reads[i] = dvBlobRead{
			index:               i,
			blob:                blob,
			offset:              offset,
			manifestCardinality: manifestCardinality,
		}
	}

	slices.SortFunc(reads, func(a, b dvBlobRead) int {
		return cmp.Compare(a.offset, b.offset)
	})

	bitmaps := make([]*RoaringPositionBitmap, len(dvFiles))
	for start := 0; start < len(reads); {
		end := start + 1
		rangeStart := reads[start].offset
		rangeEnd := rangeStart + reads[start].blob.Length

		for end < len(reads) {
			next := reads[end]
			nextEnd := next.offset + next.blob.Length
			// Only coalesce contiguous or overlapping blobs. Gaps are left as
			// separate reads so the range does not fetch unrelated bytes.
			if next.offset > rangeEnd || nextEnd-rangeStart > maxCoalescedDVRangeSize {
				break
			}

			rangeEnd = max(rangeEnd, nextEnd)
			end++
		}

		rangeData := make([]byte, rangeEnd-rangeStart)
		if _, err := reader.ReadAt(rangeData, rangeStart); err != nil {
			return nil, fmt.Errorf("read DV blob range at offset %d: %w", rangeStart, err)
		}

		for _, read := range reads[start:end] {
			blobStart := read.offset - rangeStart
			blobEnd := blobStart + read.blob.Length
			bitmap, err := DeserializeDV(rangeData[blobStart:blobEnd], read.manifestCardinality)
			if err != nil {
				return nil, fmt.Errorf("deserialize DV blob at offset %d: %w", read.offset, err)
			}
			bitmaps[read.index] = bitmap
		}

		start = end
	}

	return bitmaps, nil
}

func validateDVFile(dvFile iceberg.DataFile) error {
	if dvFile.FileFormat() != iceberg.PuffinFile {
		return fmt.Errorf("expected PUFFIN format for deletion vector, got %s", dvFile.FileFormat())
	}

	_, _, manifestReferencedDataFile, contentOffset, contentSize := iceberginternal.BorrowedDataFilePointers(dvFile)
	if contentOffset == nil || contentSize == nil {
		return fmt.Errorf("DV file %s missing ContentOffset/ContentSizeInBytes", dvFile.FilePath())
	}
	if manifestReferencedDataFile == nil || *manifestReferencedDataFile == "" {
		return fmt.Errorf("%w: DV file %s missing or empty %s property", ErrInvalidDeletionVector, dvFile.FilePath(), dvReferencedDataFileProperty)
	}

	size := *contentSize
	if size < 0 || size > int64(puffin.DefaultMaxBlobSize) {
		return fmt.Errorf("DV blob size %d out of valid range [0, %d]", size, puffin.DefaultMaxBlobSize)
	}

	return nil
}

func openDVReader(fs iceio.IO, filePath string) (*puffin.Reader, iceio.File, error) {
	f, err := fs.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open DV file %s: %w", filePath, err)
	}

	reader, err := puffin.NewReader(f)
	if err != nil {
		_ = f.Close()

		return nil, nil, fmt.Errorf("create puffin reader for %s: %w", filePath, err)
	}

	return reader, f, nil
}

func readDV(reader *puffin.Reader, blob puffin.BlobMetadata, dvFile iceberg.DataFile, offset int64, manifestReferencedDataFile *string) (*RoaringPositionBitmap, error) {
	manifestCardinality, err := validateDVBlobMetadata(blob, dvFile, offset, manifestReferencedDataFile)
	if err != nil {
		return nil, err
	}

	blobData := make([]byte, blob.Length)
	if _, err := reader.ReadAt(blobData, offset); err != nil {
		return nil, fmt.Errorf("read DV blob at offset %d: %w", offset, err)
	}

	return DeserializeDV(blobData, manifestCardinality)
}

func validateDVBlobMetadata(blob puffin.BlobMetadata, dvFile iceberg.DataFile, offset int64, manifestReferencedDataFile *string) (int64, error) {
	if blob.Type != puffin.BlobTypeDeletionVector {
		return 0, fmt.Errorf("%w: DV file %s: blob at offset %d has type %q, expected %q", ErrInvalidDeletionVector,
			dvFile.FilePath(), offset, blob.Type, puffin.BlobTypeDeletionVector)
	}
	if blob.CompressionCodec != nil && *blob.CompressionCodec != "" {
		return 0, fmt.Errorf("%w: DV file %s: blob at offset %d uses unsupported compression codec %q",
			ErrInvalidDeletionVector, dvFile.FilePath(), offset, *blob.CompressionCodec)
	}

	referencedDataFile, ok := blob.Properties[dvReferencedDataFileProperty]
	if !ok || referencedDataFile == "" {
		return 0, fmt.Errorf("%w: DV file %s: blob at offset %d missing or empty %s property", ErrInvalidDeletionVector,
			dvFile.FilePath(), offset, dvReferencedDataFileProperty)
	}
	if referencedDataFile != *manifestReferencedDataFile {
		return 0, fmt.Errorf("%w: DV file %s: manifest referenced_data_file %q does not match puffin %s %q", ErrInvalidDeletionVector,
			dvFile.FilePath(), *manifestReferencedDataFile, dvReferencedDataFileProperty, referencedDataFile)
	}

	// Manifest record_count (field 103) is a required, non-nullable long, so it
	// is always present for a DV data file: zero means an empty deletion
	// vector, not "unknown". This is Java's primary expected cardinality.
	manifestCardinality := dvFile.Count()

	// The puffin blob independently declares its cardinality via the spec-
	// mandated property; when present it is a second source to cross-check.
	puffinCardinality, hasPuffinCardinality, err := blobCardinality(blob)
	if err != nil {
		return 0, fmt.Errorf("DV file %s: %w", dvFile.FilePath(), err)
	}

	// When both sources are available they must agree. A disagreement means a
	// writer set the manifest record_count and the blob property inconsistently
	// (e.g. a stale record_count after an incremental merge against a freshly
	// written blob) — fail fast rather than silently trusting one over the other.
	if hasPuffinCardinality && manifestCardinality != puffinCardinality {
		return 0, fmt.Errorf("DV file %s: manifest record_count %d disagrees with puffin cardinality property %d",
			dvFile.FilePath(), manifestCardinality, puffinCardinality)
	}

	if !hasPuffinCardinality {
		// Spec deviation: deletion-vector-v1 MUST carry a cardinality property.
		// The manifest record_count still bounds the decoded bitmap below, so
		// this is degraded (not absent) validation; flag it so operators can
		// spot non-conformant third-party writers.
		slog.Warn("DV blob missing spec-required cardinality property; validating against manifest record_count only",
			"dv_file", dvFile.FilePath(), "offset", offset)
	}

	return manifestCardinality, nil
}

func indexBlobMetadataByOffset(blobs []puffin.BlobMetadata) map[int64]puffin.BlobMetadata {
	indexed := make(map[int64]puffin.BlobMetadata, len(blobs))
	for _, blob := range blobs {
		if _, exists := indexed[blob.Offset]; !exists {
			indexed[blob.Offset] = blob
		}
	}

	return indexed
}

// findBlobMetadataByRange returns the footer entry identified by the
// manifest's content offset and size. On error it returns a zero-value
// puffin.BlobMetadata; callers must check the error before reading the value.
func findBlobMetadataByRange(blobs []puffin.BlobMetadata, offset, size int64) (puffin.BlobMetadata, error) {
	for _, blob := range blobs {
		if blob.Offset != offset {
			continue
		}

		return validateBlobMetadataSize(blob, offset, size)
	}

	return puffin.BlobMetadata{}, fmt.Errorf("no blob in puffin footer at offset %d, size %d", offset, size)
}

func findIndexedBlobMetadataByRange(blobsByOffset map[int64]puffin.BlobMetadata, offset, size int64) (puffin.BlobMetadata, error) {
	blob, ok := blobsByOffset[offset]
	if !ok {
		return puffin.BlobMetadata{}, fmt.Errorf("no blob in puffin footer at offset %d, size %d", offset, size)
	}

	return validateBlobMetadataSize(blob, offset, size)
}

func validateBlobMetadataSize(blob puffin.BlobMetadata, offset, size int64) (puffin.BlobMetadata, error) {
	if blob.Length != size {
		// Same starting offset, different length: the manifest entry
		// disagrees with the puffin footer on how big this blob is.
		// Surface that distinct condition rather than rolling it into
		// "no blob at offset" — different writer bug, different fix.
		return puffin.BlobMetadata{}, fmt.Errorf("blob at offset %d has length %d, manifest says %d", offset, blob.Length, size)
	}

	return blob, nil
}

// blobCardinality returns the cardinality declared by a Puffin blob. The bool
// indicates whether the property was present.
func blobCardinality(blob puffin.BlobMetadata) (int64, bool, error) {
	v, ok := blob.Properties[dvCardinalityProperty]
	if !ok {
		return 0, false, nil
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid %s property %q: %w", dvCardinalityProperty, v, err)
	}
	if parsed < 0 {
		// Negative is meaningless for a count of deleted positions, and -1
		// would disable cardinality validation in DeserializeDV.
		return 0, false, fmt.Errorf("%s property must be non-negative, got %d", dvCardinalityProperty, parsed)
	}

	return parsed, true, nil
}
