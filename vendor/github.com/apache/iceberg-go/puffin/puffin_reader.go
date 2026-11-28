// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package puffin

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"sort"

	"github.com/pierrec/lz4/v4"
)

// ReaderAtSeeker combines io.ReaderAt and io.Seeker for reading Puffin files.
// This interface is implemented by *os.File, *bytes.Reader, and similar types.
type ReaderAtSeeker interface {
	io.ReaderAt
	io.Seeker
}

// Reader reads blobs and metadata from a Puffin file.
//
// Usage:
//
//	r, err := puffin.NewReader(file)
//	if err != nil {
//	    return err
//	}
//	for i := range r.Blobs() {
//	    blob, err := r.ReadBlob(i)
//	    // process blob.Data
//	}
type Reader struct {
	r             ReaderAtSeeker
	size          int64
	footer        Footer
	footerStart   int64 // cached after ReadFooter
	maxBlobSize   int64
	maxFooterSize int64
}

// BlobData pairs a blob's metadata with its content.
type BlobData struct {
	Metadata BlobMetadata
	Data     []byte
}

// ReaderOption configures a Reader.
type ReaderOption func(*Reader)

// WithMaxBlobSize sets the maximum blob size allowed when reading.
// This prevents OOM attacks from malicious files with huge blob lengths.
// Default is DefaultMaxBlobSize (256 MB).
func WithMaxBlobSize(size int64) ReaderOption {
	return func(r *Reader) {
		r.maxBlobSize = size
	}
}

// WithMaxFooterSize sets the maximum footer payload size allowed when reading.
// For compressed footer payloads, the limit applies after decompression. This
// bounds memory used by JSON decoding and decompression of untrusted Puffin
// metadata. The default is DefaultMaxFooterSize (64 MB). A non-positive size
// is rejected.
func WithMaxFooterSize(size int64) ReaderOption {
	return func(r *Reader) {
		r.maxFooterSize = size
	}
}

// NewReader creates a new Puffin file reader.
// The file size is auto-detected using Seek.
// It validates magic bytes and reads the footer eagerly.
// The caller is responsible for closing the underlying reader.
func NewReader(r ReaderAtSeeker, opts ...ReaderOption) (*Reader, error) {
	if r == nil {
		return nil, errors.New("puffin: reader is nil")
	}

	// Auto-detect file size
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("puffin: detect file size: %w", err)
	}

	// Minimum size: header magic + footer magic + footer trailer
	// [Magic] + zero for blob + [Magic] + [FooterPayloadSize (assuming ~0)] + [Flags] + [Magic]
	minSize := int64(MagicSize + MagicSize + footerTrailerSize)
	if size < minSize {
		return nil, fmt.Errorf("puffin: file too small (%d bytes, minimum %d)", size, minSize)
	}

	// Validate header magic
	var headerMagic [MagicSize]byte
	if _, err := r.ReadAt(headerMagic[:], 0); err != nil {
		return nil, fmt.Errorf("puffin: read header magic: %w", err)
	}
	if !bytes.Equal(headerMagic[:], magic[:]) {
		return nil, errors.New("puffin: invalid header magic")
	}

	pr := &Reader{
		r:             r,
		size:          size,
		maxBlobSize:   DefaultMaxBlobSize,
		maxFooterSize: DefaultMaxFooterSize,
	}

	for _, opt := range opts {
		opt(pr)
	}

	// Read footer
	if err := pr.readFooter(); err != nil {
		return nil, err
	}

	return pr, nil
}

// Blobs returns the blob metadata entries from the footer.
func (r *Reader) Blobs() []BlobMetadata {
	return cloneBlobMetadataList(r.footer.Blobs)
}

// Properties returns the file-level properties from the footer.
func (r *Reader) Properties() map[string]string {
	return maps.Clone(r.footer.Properties)
}

func cloneBlobMetadataList(blobs []BlobMetadata) []BlobMetadata {
	cloned := make([]BlobMetadata, len(blobs))
	for i, blob := range blobs {
		cloned[i] = cloneBlobMetadata(blob)
	}

	return cloned
}

func cloneBlobMetadata(blob BlobMetadata) BlobMetadata {
	blob.Fields = slices.Clone(blob.Fields)
	blob.Properties = maps.Clone(blob.Properties)
	if blob.CompressionCodec != nil {
		codec := *blob.CompressionCodec
		blob.CompressionCodec = &codec
	}

	return blob
}

// defaultFooterReadSize is the initial read size when reading the footer.
// We read more than strictly needed to hopefully get the entire footer
// in one read, reducing round-trips on cloud object storage.
const defaultFooterReadSize = 8 * 1024 // 8 KB

const (
	lz4FrameMagic                 uint32 = 0x184d2204
	lz4FrameHeaderPrefixSize             = 6 // magic (4) + FLG (1) + BD (1)
	lz4FrameHeaderSize                   = 7 // prefix + header checksum
	lz4FrameHeaderSizeWithContent        = 15
	lz4FrameContentSizeFlag       byte   = 1 << 3
	lz4FrameContentChecksumFlag   byte   = 1 << 2
	lz4FrameBlockChecksumFlag     byte   = 1 << 4
	lz4FrameVersionMask           byte   = 0xc0
	lz4FrameVersionOne            byte   = 0x40
	lz4FrameReservedFLGFlag       byte   = 1 << 1
	lz4FrameDictionaryIDFlag      byte   = 1
	lz4FrameReservedBDFlags       byte   = 0x8f
	lz4FrameBlockSizeMask         uint32 = 0x7fffffff
	lz4FrameReadBufferSize               = 4 << 20 // LZ4 maximum block size.
)

type countingReader struct {
	reader io.Reader
	count  int64
	err    error
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.count += int64(n)
	if err != nil && !errors.Is(err, io.EOF) {
		r.err = err
	}

	return n, err
}

type lz4FrameHeader struct {
	flags          byte
	headerSize     int
	contentSize    uint64
	blockMaxSize   int64
	hasContentSize bool
}

func lz4BlockMaxSize(bd byte) (int64, error) {
	switch (bd >> 4) & 0x7 {
	case 4:
		return 64 << 10, nil
	case 5:
		return 256 << 10, nil
	case 6:
		return 1 << 20, nil
	case 7:
		return 4 << 20, nil
	default:
		return 0, errors.New("invalid LZ4 frame block maximum size")
	}
}

// readLZ4FrameHeader reads and validates the LZ4 frame descriptor without
// consuming the payload. The content size is required by the Puffin format
// for compressed footer payloads.
func readLZ4FrameHeader(r io.ReaderAt) (lz4FrameHeader, error) {
	var header [lz4FrameHeaderSizeWithContent]byte
	if _, err := r.ReadAt(header[:lz4FrameHeaderSize], 0); err != nil {
		return lz4FrameHeader{}, err
	}

	if binary.LittleEndian.Uint32(header[:4]) != lz4FrameMagic {
		return lz4FrameHeader{}, errors.New("invalid LZ4 frame magic")
	}

	hasContentSize := header[4]&lz4FrameContentSizeFlag != 0
	if header[4]&lz4FrameVersionMask != lz4FrameVersionOne {
		return lz4FrameHeader{}, errors.New("invalid LZ4 frame version")
	}
	if header[4]&lz4FrameReservedFLGFlag != 0 {
		return lz4FrameHeader{}, errors.New("invalid LZ4 frame reserved flag")
	}
	if header[4]&lz4FrameDictionaryIDFlag != 0 {
		return lz4FrameHeader{}, errors.New("LZ4 frame dictionary ID is not supported")
	}
	if header[5]&lz4FrameReservedBDFlags != 0 {
		return lz4FrameHeader{}, errors.New("invalid LZ4 frame reserved block descriptor flags")
	}
	blockMaxSize, err := lz4BlockMaxSize(header[5])
	if err != nil {
		return lz4FrameHeader{}, err
	}
	if hasContentSize {
		if _, err := r.ReadAt(header[lz4FrameHeaderSize:], lz4FrameHeaderSize); err != nil {
			return lz4FrameHeader{}, err
		}
	}

	headerSize := lz4FrameHeaderSize
	if hasContentSize {
		headerSize = lz4FrameHeaderSizeWithContent
	}
	valid, err := lz4.ValidFrameHeader(header[:headerSize])
	if err != nil {
		return lz4FrameHeader{}, err
	}
	if !valid {
		return lz4FrameHeader{}, errors.New("invalid frame header")
	}

	frameHeader := lz4FrameHeader{
		flags:          header[4],
		headerSize:     headerSize,
		blockMaxSize:   blockMaxSize,
		hasContentSize: hasContentSize,
	}
	if hasContentSize {
		frameHeader.contentSize = binary.LittleEndian.Uint64(header[lz4FrameHeaderPrefixSize : lz4FrameHeaderPrefixSize+8])
	}

	return frameHeader, nil
}

func validateLZ4FrameEnvelope(
	r io.ReaderAt,
	payloadSize int64,
	flags byte,
	headerSize int,
	blockMaxSize int64,
) error {
	if payloadSize < int64(headerSize) {
		return errors.New("LZ4 frame is shorter than its header")
	}
	if payloadSize-int64(headerSize) < 4 {
		return errors.New("LZ4 frame is missing its end mark")
	}

	firstBlock := make([]byte, headerSize+4)
	if _, err := r.ReadAt(firstBlock, 0); err != nil {
		return fmt.Errorf("read LZ4 frame header and block header: %w", err)
	}

	blockSize := binary.LittleEndian.Uint32(firstBlock[headerSize:])
	offset := int64(headerSize + 4)
	reader := bufio.NewReaderSize(
		io.NewSectionReader(r, offset, payloadSize-offset),
		lz4FrameReadBufferSize,
	)
	var blockHeader [4]byte

	for blockSize != 0 {
		dataSize := int64(blockSize & lz4FrameBlockSizeMask)
		if dataSize > blockMaxSize {
			return errors.New("LZ4 frame block exceeds declared maximum")
		}
		if dataSize > payloadSize-offset {
			return errors.New("LZ4 frame block exceeds its payload")
		}
		if _, err := io.CopyN(io.Discard, reader, dataSize); err != nil {
			return fmt.Errorf("read LZ4 frame block: %w", err)
		}
		offset += dataSize

		if flags&lz4FrameBlockChecksumFlag != 0 {
			if payloadSize-offset < 4 {
				return errors.New("LZ4 frame block checksum is truncated")
			}
			if _, err := io.CopyN(io.Discard, reader, 4); err != nil {
				return fmt.Errorf("read LZ4 frame block checksum: %w", err)
			}
			offset += 4
		}

		if payloadSize-offset < int64(len(blockHeader)) {
			return errors.New("LZ4 frame is missing its end mark")
		}
		if _, err := io.ReadFull(reader, blockHeader[:]); err != nil {
			return fmt.Errorf("read LZ4 frame block header: %w", err)
		}
		offset += int64(len(blockHeader))

		blockSize = binary.LittleEndian.Uint32(blockHeader[:])
	}

	if flags&lz4FrameContentChecksumFlag != 0 {
		if payloadSize-offset < 4 {
			return errors.New("LZ4 frame content checksum is truncated")
		}
		if _, err := io.CopyN(io.Discard, reader, 4); err != nil {
			return fmt.Errorf("read LZ4 frame content checksum: %w", err)
		}
		offset += 4
	}

	if offset != payloadSize {
		return errors.New("LZ4 frame has trailing data")
	}

	return nil
}

// readFooter reads and parses the footer from the Puffin file.
func (r *Reader) readFooter() error {
	// Read a larger chunk from the end to minimize round-trips on cloud storage.
	// This often captures the entire footer in one read.
	readSize := min(int64(defaultFooterReadSize), r.size)
	buf := make([]byte, readSize)
	if _, err := r.r.ReadAt(buf, r.size-readSize); err != nil {
		return fmt.Errorf("puffin: read footer region: %w", err)
	}

	// Parse trailer from end of buffer: PayloadSize(4) + Flags(4) + Magic(4)
	trailer := buf[len(buf)-footerTrailerSize:]

	// Validate trailing magic
	if !bytes.Equal(trailer[8:12], magic[:]) {
		return errors.New("puffin: invalid trailing magic in footer")
	}

	// Extract payload size and flags
	payloadSize := int64(int32(binary.LittleEndian.Uint32(trailer[0:4])))
	flags := binary.LittleEndian.Uint32(trailer[4:8])

	// Check for unknown flags
	if flags&^uint32(FooterFlagCompressed) != 0 {
		return fmt.Errorf("puffin: unknown footer flags set: 0x%x", flags)
	}

	// Validate payload size
	if payloadSize < 0 {
		return fmt.Errorf("puffin: invalid footer payload size %d", payloadSize)
	}
	if r.maxFooterSize <= 0 {
		return fmt.Errorf("puffin: invalid maximum footer size %d", r.maxFooterSize)
	}

	// Calculate footer start position
	// Layout: [header magic (4)] [blobs...] [footer magic (4)] [JSON (payloadSize)] [trailer (12)]
	footerStart := r.size - footerTrailerSize - payloadSize - MagicSize
	if footerStart < MagicSize {
		return fmt.Errorf("puffin: footer payload size %d exceeds available space", payloadSize)
	}

	// Total footer size: magic(4) + payload + trailer(12)
	totalFooterSize := MagicSize + payloadSize + footerTrailerSize

	var payloadReader *io.SectionReader

	// Validate footer start magic
	if totalFooterSize <= readSize {
		// We already have the footer magic in buf
		footerOffset := len(buf) - int(totalFooterSize)
		if !bytes.Equal(buf[footerOffset:footerOffset+MagicSize], magic[:]) {
			return errors.New("puffin: invalid footer start magic")
		}

		payloadOffset := footerOffset + MagicSize
		payloadReader = io.NewSectionReader(
			bytes.NewReader(buf[payloadOffset:payloadOffset+int(payloadSize)]),
			0,
			payloadSize,
		)
	} else {
		// Footer is larger than our initial read, need to read magic separately
		var footerMagic [MagicSize]byte
		if _, err := r.r.ReadAt(footerMagic[:], footerStart); err != nil {
			return fmt.Errorf("puffin: read footer start magic: %w", err)
		}
		if !bytes.Equal(footerMagic[:], magic[:]) {
			return errors.New("puffin: invalid footer start magic")
		}

		payloadReader = io.NewSectionReader(r.r, footerStart+MagicSize, payloadSize)
	}
	if flags&FooterFlagCompressed == 0 && payloadSize > r.maxFooterSize {
		return fmt.Errorf("puffin: footer exceeds maximum size %d", r.maxFooterSize)
	}

	var footerReader io.Reader = payloadReader
	var compressedFooter *countingReader
	var compressedContentSize uint64
	var limitedFooter *io.LimitedReader
	if flags&FooterFlagCompressed != 0 {
		// Java, PyIceberg, and Rust currently reject compressed Puffin footers;
		// keep this flag reader-only until those implementations catch up.

		frameHeader, err := readLZ4FrameHeader(payloadReader)
		if err != nil {
			return fmt.Errorf("puffin: inspect compressed footer LZ4 frame: %w", err)
		}
		if !frameHeader.hasContentSize {
			return errors.New("puffin: compressed footer LZ4 frame is missing content size")
		}
		if frameHeader.contentSize > uint64(r.maxFooterSize) {
			return fmt.Errorf("puffin: footer exceeds maximum size %d", r.maxFooterSize)
		}
		if err := validateLZ4FrameEnvelope(
			payloadReader,
			payloadSize,
			frameHeader.flags,
			frameHeader.headerSize,
			frameHeader.blockMaxSize,
		); err != nil {
			return fmt.Errorf("puffin: inspect compressed footer LZ4 frame: %w", err)
		}

		compressedContentSize = frameHeader.contentSize
		compressedPayloadReader := bufio.NewReaderSize(
			io.NewSectionReader(payloadReader, 0, payloadSize),
			lz4FrameReadBufferSize,
		)
		compressedFooter = &countingReader{reader: lz4.NewReader(compressedPayloadReader)}
		footerReader = compressedFooter

		limitedFooterSize := r.maxFooterSize
		if limitedFooterSize < math.MaxInt64 {
			limitedFooterSize++
		}
		limitedFooter = &io.LimitedReader{R: footerReader, N: limitedFooterSize}
		footerReader = limitedFooter
	}

	decoder := json.NewDecoder(footerReader)
	var footer Footer
	if err := decoder.Decode(&footer); err != nil {
		return fmt.Errorf("puffin: decode footer JSON: %w", err)
	}
	if limitedFooter != nil && limitedFooter.N == 0 {
		buffered, err := io.ReadAll(decoder.Buffered())
		if err != nil {
			return fmt.Errorf("puffin: read buffered footer JSON: %w", err)
		}
		if len(bytes.TrimSpace(buffered)) > 0 {
			return errors.New("puffin: unexpected content after footer JSON")
		}
	}

	// FooterPayloadSize defines a single JSON footer object. Reject trailing
	// content deliberately, even though some other Iceberg implementations
	// accept padding or additional values inside the footer payload.
	if decoder.More() {
		return errors.New("puffin: unexpected content after footer JSON")
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if compressedFooter != nil && compressedFooter.err != nil {
			return fmt.Errorf("puffin: read compressed footer: %w", compressedFooter.err)
		}

		return errors.New("puffin: unexpected content after footer JSON")
	}
	if limitedFooter != nil && limitedFooter.N == 0 {
		return fmt.Errorf("puffin: footer exceeds maximum size %d", r.maxFooterSize)
	}
	if compressedFooter != nil && uint64(compressedFooter.count) != compressedContentSize {
		return fmt.Errorf(
			"puffin: compressed footer decoded size %d does not match LZ4 content size %d",
			compressedFooter.count,
			compressedContentSize,
		)
	}

	// Validate blob metadata
	if err := r.validateBlobs(footer.Blobs, footerStart); err != nil {
		return err
	}

	r.footer = footer
	r.footerStart = footerStart

	return nil
}

// ReadBlob reads the content of a specific blob by index.
// The footer is read automatically if not already cached.
func (r *Reader) ReadBlob(index int) (*BlobData, error) {
	footer := r.footer

	if index < 0 || index >= len(footer.Blobs) {
		return nil, fmt.Errorf("puffin: blob index %d out of range [0, %d)", index, len(footer.Blobs))
	}

	meta := footer.Blobs[index]
	data, err := r.readBlobData(meta)
	if err != nil {
		return nil, err
	}

	return &BlobData{Metadata: cloneBlobMetadata(meta), Data: data}, nil
}

// ReadBlobByMetadata reads a blob using its metadata directly.
// This is useful when you have metadata from an external source.
func (r *Reader) ReadBlobByMetadata(meta BlobMetadata) ([]byte, error) {
	return r.readBlobData(meta)
}

// readBlobData is the internal implementation for reading blob data.
func (r *Reader) readBlobData(meta BlobMetadata) ([]byte, error) {
	// Validate blob type
	if meta.Type == "" {
		return nil, errors.New("puffin: cannot read blob: type is required")
	}

	// Check for compressed blob (unsupported)
	if meta.CompressionCodec != nil && *meta.CompressionCodec != "" {
		return nil, fmt.Errorf("puffin: cannot read blob: compression %q not supported", *meta.CompressionCodec)
	}

	// Validate offset/length
	if err := r.validateRange(meta.Offset, meta.Length); err != nil {
		return nil, fmt.Errorf("puffin: blob: %w", err)
	}

	// Read blob data
	data := make([]byte, meta.Length)
	if _, err := r.r.ReadAt(data, meta.Offset); err != nil {
		return nil, fmt.Errorf("puffin: read blob data: %w", err)
	}

	return data, nil
}

// ReadAllBlobs reads all blobs from the file.
func (r *Reader) ReadAllBlobs() ([]*BlobData, error) {
	footer := r.footer

	if len(footer.Blobs) == 0 {
		return nil, nil
	}

	// Create index mapping to preserve original order
	type indexedBlob struct {
		index int
		meta  BlobMetadata
	}
	indexed := make([]indexedBlob, len(footer.Blobs))
	for i, meta := range footer.Blobs {
		indexed[i] = indexedBlob{index: i, meta: meta}
	}

	// Sort by offset for sequential I/O
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].meta.Offset < indexed[j].meta.Offset
	})

	// Read blobs in offset order, store in original order
	results := make([]*BlobData, len(footer.Blobs))
	for _, ib := range indexed {
		data, err := r.readBlobData(ib.meta)
		if err != nil {
			return nil, fmt.Errorf("puffin: read blob %d: %w", ib.index, err)
		}
		results[ib.index] = &BlobData{Metadata: cloneBlobMetadata(ib.meta), Data: data}
	}

	return results, nil
}

// ReadAt implements io.ReaderAt, reading from the blob data region.
// It validates that the read range is within the blob data region
// This is useful for deletion vector use case.
// offset/length pointing directly into the Puffin file in manifest.
func (r *Reader) ReadAt(p []byte, off int64) (n int, err error) {
	if err := r.validateRange(off, int64(len(p))); err != nil {
		return 0, fmt.Errorf("puffin: %w", err)
	}

	return r.r.ReadAt(p, off)
}

// validateRange validates offset and length for reading from the blob data region.
func (r *Reader) validateRange(offset, length int64) error {
	if length < 0 {
		return fmt.Errorf("invalid length %d", length)
	}
	if length > r.maxBlobSize {
		return fmt.Errorf("size %d exceeds limit %d", length, r.maxBlobSize)
	}
	if offset < MagicSize {
		return fmt.Errorf("invalid offset %d (before header)", offset)
	}

	end := offset + length
	if end < offset {
		return fmt.Errorf("offset+length overflow: offset=%d length=%d", offset, length)
	}
	if end > r.footerStart {
		return fmt.Errorf("extends into footer: offset=%d length=%d footerStart=%d",
			offset, length, r.footerStart)
	}

	return nil
}

// validateBlobs validates all blob metadata entries.
func (r *Reader) validateBlobs(blobs []BlobMetadata, footerStart int64) error {
	for i, blob := range blobs {
		// Type is required
		if blob.Type == "" {
			return fmt.Errorf("puffin: blob %d: type is required", i)
		}

		// Length must be non-negative
		if blob.Length < 0 {
			return fmt.Errorf("puffin: blob %d: invalid length %d", i, blob.Length)
		}

		// Offset must be after header magic
		if blob.Offset < MagicSize {
			return fmt.Errorf("puffin: blob %d: offset %d before header", i, blob.Offset)
		}

		// Check for overflow
		end := blob.Offset + blob.Length
		if end < blob.Offset {
			return fmt.Errorf("puffin: blob %d: offset+length overflow: offset=%d length=%d", i, blob.Offset, blob.Length)
		}

		// Blob must not extend into footer
		if end > footerStart {
			return fmt.Errorf("puffin: blob %d: extends into footer: offset=%d length=%d footerStart=%d",
				i, blob.Offset, blob.Length, footerStart)
		}

		// Spec invariant for deletion-vector-v1: snapshot-id and sequence-
		// number MUST be -1. The writer enforces this on the way out; the
		// reader symmetrically rejects malformed blobs on the way in so a
		// third-party tool that violates the contract can't sneak data
		// into the deserialization layer. Phrasing matches the writer's
		// "<type> requires X to be -1, got N" so both sides share a
		// grep-able canonical message.
		if blob.Type == BlobTypeDeletionVector {
			if blob.SnapshotID != -1 {
				return fmt.Errorf("puffin: blob %d: %s requires snapshot-id to be -1, got %d",
					i, BlobTypeDeletionVector, blob.SnapshotID)
			}
			if blob.SequenceNumber != -1 {
				return fmt.Errorf("puffin: blob %d: %s requires sequence-number to be -1, got %d",
					i, BlobTypeDeletionVector, blob.SequenceNumber)
			}
		}
	}

	return nil
}
