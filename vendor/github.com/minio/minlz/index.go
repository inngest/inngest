// Copyright 2025 MinIO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package minlz

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

const (
	IndexHeader     = "s2idx\x00"
	IndexTrailer    = "\x00xdi2s"
	maxIndexEntries = 1 << 16
	minIndexDist    = 1 << 20 // Minimum uncompressed distance between entries
)

// Index represents an S2/Snappy/MinLZ index.
type Index struct {
	// Total Uncompressed size.
	TotalUncompressed int64

	// Total Compressed size if known. Will be -1 if unknown.
	TotalCompressed int64

	// Offset pairs are pairs of Compressed -> Uncompressed positions.
	// Offsets are stream offsets from first stream byte.
	// It will be safe to start decompressing from any of these offsets.
	// The slice is sorted by offset.
	Offsets []OffsetPair

	estBlockUncomp int64
}

type OffsetPair struct {
	CompressedOffset   int64
	UncompressedOffset int64
}

func (i *Index) reset(maxBlock int) {
	if i == nil {
		return
	}
	for maxBlock < minIndexDist {
		maxBlock *= 2
	}
	i.estBlockUncomp = int64(maxBlock)
	i.TotalCompressed = -1
	i.TotalUncompressed = -1
	if len(i.Offsets) > 0 {
		i.Offsets = i.Offsets[:0]
	}
}

// allocInfos will allocate an empty slice of infos.
func (i *Index) allocInfos(n int) {
	if n > maxIndexEntries {
		panic("n > maxIndexEntries")
	}
	i.Offsets = make([]OffsetPair, 0, n)
}

// add an uncompressed and compressed pair.
// Entries must be sent in order.
func (i *Index) add(compressedOffset, uncompressedOffset int64) error {
	if i == nil {
		return nil
	}
	lastIdx := len(i.Offsets) - 1
	if lastIdx >= 0 {
		latest := i.Offsets[lastIdx]
		if uncompressedOffset-latest.UncompressedOffset < i.estBlockUncomp {
			// Don't add until we have i.estBlockUncomp
			return nil
		}
		if latest.UncompressedOffset > uncompressedOffset {
			return fmt.Errorf("internal error: Earlier uncompressed received (%d > %d)", latest.UncompressedOffset, uncompressedOffset)
		}
		if latest.CompressedOffset > compressedOffset {
			return fmt.Errorf("internal error: Earlier compressed received (%d > %d)", latest.UncompressedOffset, uncompressedOffset)
		}
	}
	i.Offsets = append(i.Offsets, OffsetPair{CompressedOffset: compressedOffset, UncompressedOffset: uncompressedOffset})
	if len(i.Offsets) > maxIndexEntries {
		// Keep memory from exploding.
		i.reduceLight()
	}
	return nil
}

// Find the offset at or before the wanted (uncompressed) offset.
// If offset is 0 or positive it is the offset from the beginning of the file.
// If the uncompressed size is known, the offset must be within the file.
// If an offset outside the file is requested io.ErrUnexpectedEOF is returned.
// If the offset is negative, it is interpreted as the distance from the end of the file,
// where -1 represents the last byte.
// If offset from the end of the file is requested, but size is unknown,
// ErrUnsupported will be returned.
func (i *Index) Find(offset int64) (compressedOff, uncompressedOff int64, err error) {
	if i.TotalUncompressed < 0 {
		return 0, 0, ErrCorrupt
	}
	if offset < 0 {
		offset = i.TotalUncompressed + offset
		if offset < 0 {
			return 0, 0, io.ErrUnexpectedEOF
		}
	}
	if offset > i.TotalUncompressed {
		return 0, 0, io.ErrUnexpectedEOF
	}
	if len(i.Offsets) > 200 {
		n := sort.Search(len(i.Offsets), func(n int) bool {
			return i.Offsets[n].UncompressedOffset > offset
		})
		if n == 0 {
			n = 1
		}
		return i.Offsets[n-1].CompressedOffset, i.Offsets[n-1].UncompressedOffset, nil
	}
	for _, info := range i.Offsets {
		if info.UncompressedOffset > offset {
			break
		}
		compressedOff = info.CompressedOffset
		uncompressedOff = info.UncompressedOffset
	}
	return compressedOff, uncompressedOff, nil
}

// reduce to stay below maxIndexEntries
func (i *Index) reduce() {
	if len(i.Offsets) < maxIndexEntries {
		return
	}

	// Algorithm, keep 1, remove removeN entries...
	removeN := (len(i.Offsets) + 1) / maxIndexEntries
	src := i.Offsets
	j := 0

	// Each block should be at least 1MB, but don't reduce below 1000 entries.
	for i.estBlockUncomp*(int64(removeN)+1) < minIndexDist && len(i.Offsets)/(removeN+1) > 1000 {
		removeN++
	}
	for idx := 0; idx < len(src); idx++ {
		i.Offsets[j] = src[idx]
		j++
		idx += removeN
	}
	i.Offsets = i.Offsets[:j]
	// Update maxblock estimate.
	i.estBlockUncomp += i.estBlockUncomp * int64(removeN)
}

// reduce to stay below maxIndexEntries
func (i *Index) reduceLight() {
	i.estBlockUncomp *= 2
	src := i.Offsets
	var j int
	for idx := 0; idx < len(src); idx++ {
		base := src[idx]
		i.Offsets[j] = base
		j++
		for idx < len(src) && src[idx].UncompressedOffset-base.UncompressedOffset < i.estBlockUncomp {
			idx++
		}
	}
	i.Offsets = i.Offsets[:j]
}

func (i *Index) appendTo(b []byte, uncompTotal, compTotal int64) []byte {
	if i == nil {
		return nil
	}
	i.reduce()
	var tmp [binary.MaxVarintLen64]byte

	initSize := len(b)
	// We make the start a skippable header+size.
	b = append(b, chunkTypeIndex, 0, 0, 0)
	b = append(b, []byte(IndexHeader)...)
	// Total Uncompressed size
	n := binary.PutVarint(tmp[:], uncompTotal)
	b = append(b, tmp[:n]...)
	// Total Compressed size
	n = binary.PutVarint(tmp[:], compTotal)
	b = append(b, tmp[:n]...)
	// Put EstBlockUncomp size
	n = binary.PutVarint(tmp[:], i.estBlockUncomp)
	b = append(b, tmp[:n]...)
	// Put length
	n = binary.PutVarint(tmp[:], int64(len(i.Offsets)))
	b = append(b, tmp[:n]...)

	// Check if we should add uncompressed offsets
	var hasUncompressed byte
	for idx, info := range i.Offsets {
		if idx == 0 {
			if info.UncompressedOffset != 0 {
				hasUncompressed = 1
				break
			}
			continue
		}
		if info.UncompressedOffset != i.Offsets[idx-1].UncompressedOffset+i.estBlockUncomp {
			hasUncompressed = 1
			break
		}
	}
	b = append(b, hasUncompressed)

	// Add each entry
	if hasUncompressed == 1 {
		for idx, info := range i.Offsets {
			uOff := info.UncompressedOffset
			if idx > 0 {
				prev := i.Offsets[idx-1]
				uOff -= prev.UncompressedOffset + (i.estBlockUncomp)
			}
			n = binary.PutVarint(tmp[:], uOff)
			b = append(b, tmp[:n]...)
		}
	}

	// Initial compressed size estimate.
	cPredict := i.estBlockUncomp / 2

	for idx, info := range i.Offsets {
		cOff := info.CompressedOffset
		if idx > 0 {
			prev := i.Offsets[idx-1]
			cOff -= prev.CompressedOffset + cPredict
			// Update compressed size prediction, with half the error.
			cPredict += cOff / 2
		}
		b = binary.AppendVarint(b, cOff)
	}

	// Add Total Size.
	// Stored as fixed size for easier reading.
	binary.LittleEndian.PutUint32(tmp[:], uint32(len(b)-initSize+4+len(IndexTrailer)))
	b = append(b, tmp[:4]...)
	// Trailer
	b = append(b, []byte(IndexTrailer)...)

	// Update size
	chunkLen := len(b) - initSize - skippableFrameHeader
	b[initSize+1] = uint8(chunkLen >> 0)
	b[initSize+2] = uint8(chunkLen >> 8)
	b[initSize+3] = uint8(chunkLen >> 16)
	//fmt.Printf("chunklen: 0x%x Uncomp:%d, Comp:%d\n", chunkLen, uncompTotal, compTotal)
	return b
}

// Load a binary index.
// A zero value Index can be used or a previous one can be reused.
func (i *Index) Load(b []byte) ([]byte, error) {
	if len(b) <= 4+len(IndexHeader)+len(IndexTrailer) {
		return b, io.ErrUnexpectedEOF
	}
	if b[0] != chunkTypeIndex && b[0] != legacyIndexChunk {
		return b, ErrCorrupt
	}
	chunkLen := int(b[1]) | int(b[2])<<8 | int(b[3])<<16
	b = b[4:]

	// Validate we have enough...
	if len(b) < chunkLen {
		return b, io.ErrUnexpectedEOF
	}
	if !bytes.Equal(b[:len(IndexHeader)], []byte(IndexHeader)) {
		return b, ErrUnsupported
	}
	b = b[len(IndexHeader):]

	// Total Uncompressed
	if v, n := binary.Varint(b); n <= 0 || v < 0 {
		return b, ErrCorrupt
	} else {
		i.TotalUncompressed = v
		b = b[n:]
	}

	// Total Compressed
	if v, n := binary.Varint(b); n <= 0 {
		return b, ErrCorrupt
	} else {
		i.TotalCompressed = v
		b = b[n:]
	}

	// Read EstBlockUncomp
	if v, n := binary.Varint(b); n <= 0 {
		return b, ErrCorrupt
	} else {
		if v < 0 {
			return b, ErrCorrupt
		}
		i.estBlockUncomp = v
		b = b[n:]
	}

	var entries int
	if v, n := binary.Varint(b); n <= 0 {
		return b, ErrCorrupt
	} else {
		if v < 0 || v > maxIndexEntries {
			return b, ErrCorrupt
		}
		entries = int(v)
		b = b[n:]
	}
	if cap(i.Offsets) < entries {
		i.allocInfos(entries)
	}
	i.Offsets = i.Offsets[:entries]

	if len(b) < 1 {
		return b, io.ErrUnexpectedEOF
	}
	hasUncompressed := b[0]
	b = b[1:]
	if hasUncompressed&1 != hasUncompressed {
		return b, ErrCorrupt
	}

	// Add each uncompressed entry
	for idx := range i.Offsets {
		var uOff int64
		if hasUncompressed != 0 {
			// Load delta
			if v, n := binary.Varint(b); n <= 0 {
				return b, ErrCorrupt
			} else {
				uOff = v
				b = b[n:]
			}
		}

		if idx > 0 {
			prev := i.Offsets[idx-1].UncompressedOffset
			uOff += prev + (i.estBlockUncomp)
			if uOff <= prev {
				return b, ErrCorrupt
			}
		}
		if uOff < 0 {
			return b, ErrCorrupt
		}
		i.Offsets[idx].UncompressedOffset = uOff
	}

	// Initial compressed size estimate.
	cPredict := i.estBlockUncomp / 2

	// Add each compressed entry
	for idx := range i.Offsets {
		var cOff int64
		if v, n := binary.Varint(b); n <= 0 {
			return b, ErrCorrupt
		} else {
			cOff = v
			b = b[n:]
		}

		if idx > 0 {
			// Update compressed size prediction, with half the error.
			cPredictNew := cPredict + cOff/2

			prev := i.Offsets[idx-1].CompressedOffset
			cOff += prev + cPredict
			if cOff <= prev {
				return b, ErrCorrupt
			}
			cPredict = cPredictNew
		}
		if cOff < 0 {
			return b, ErrCorrupt
		}
		i.Offsets[idx].CompressedOffset = cOff
	}
	if len(b) < 4+len(IndexTrailer) {
		return b, io.ErrUnexpectedEOF
	}
	// Skip size...
	b = b[4:]

	// Check trailer...
	if !bytes.Equal(b[:len(IndexTrailer)], []byte(IndexTrailer)) {
		return b, ErrCorrupt
	}
	return b[len(IndexTrailer):], nil
}

// LoadStream will load an index from the end of the supplied stream.
// ErrUnsupported will be returned if the signature cannot be found.
// ErrCorrupt will be returned if unexpected values are found.
// io.ErrUnexpectedEOF is returned if there are too few bytes.
// IO errors are returned as-is.
func (i *Index) LoadStream(rs io.ReadSeeker) error {
	// Go to end.
	_, err := rs.Seek(-10, io.SeekEnd)
	if err != nil {
		return err
	}
	var tmp [10]byte
	_, err = io.ReadFull(rs, tmp[:])
	if err != nil {
		return err
	}
	// Check trailer...
	if !bytes.Equal(tmp[4:4+len(IndexTrailer)], []byte(IndexTrailer)) {
		return ErrUnsupported
	}
	sz := binary.LittleEndian.Uint32(tmp[:4])
	if sz > MaxUserChunkSize+skippableFrameHeader {
		return ErrCorrupt
	}
	_, err = rs.Seek(-int64(sz), io.SeekEnd)
	if err != nil {
		return err
	}

	// Read index.
	buf := make([]byte, sz)
	_, err = io.ReadFull(rs, buf)
	if err != nil {
		return err
	}
	_, err = i.Load(buf)
	return err
}

// IndexStream will return an index for a stream.
// The stream structure will be checked, but
// data within blocks is not verified.
// The returned index can either be appended to the end of the stream
// or stored separately.
func IndexStream(r io.Reader) ([]byte, error) {
	var i Index
	var buf [MaxUserChunkSize]byte
	var readHeader bool
	for {
		_, err := io.ReadFull(r, buf[:4])
		if err != nil {
			if err == io.EOF {
				return i.appendTo(nil, i.TotalUncompressed, i.TotalCompressed), nil
			}
			return nil, err
		}
		// Start of this chunk.
		startChunk := i.TotalCompressed
		i.TotalCompressed += 4

		chunkType := buf[0]
		if !readHeader {
			if chunkType != ChunkTypeStreamIdentifier && chunkType != chunkTypeEOF {
				return nil, ErrCorrupt
			}
			readHeader = true
		}
		chunkLen := int(buf[1]) | int(buf[2])<<8 | int(buf[3])<<16
		if chunkLen < checksumSize {
			return nil, ErrCorrupt
		}

		i.TotalCompressed += int64(chunkLen)
		_, err = io.ReadFull(r, buf[:chunkLen])
		if err != nil {
			return nil, io.ErrUnexpectedEOF
		}

		switch chunkType {
		case chunkTypeLegacyCompressedData, chunkTypeMinLZCompressedData, chunkTypeMinLZCompressedDataCompCRC:
			// Section 4.2. Compressed data (chunk type 0x00).
			// Skip checksum.
			dLen, err := DecodedLen(buf[checksumSize:])
			if err != nil {
				return nil, err
			}
			if dLen > maxBlockSize {
				return nil, ErrCorrupt
			}
			if i.estBlockUncomp == 0 {
				// Use first block for estimate...
				i.estBlockUncomp = int64(dLen)
			}
			err = i.add(startChunk, i.TotalUncompressed)
			if err != nil {
				return nil, err
			}
			i.TotalUncompressed += int64(dLen)
			continue
		case chunkTypeUncompressedData:
			n2 := chunkLen - checksumSize
			if n2 > maxBlockSize {
				return nil, ErrCorrupt
			}
			if i.estBlockUncomp == 0 {
				// Use first block for estimate...
				i.estBlockUncomp = int64(n2)
			}
			err = i.add(startChunk, i.TotalUncompressed)
			if err != nil {
				return nil, err
			}
			i.TotalUncompressed += int64(n2)
			continue
		case ChunkTypeStreamIdentifier:
			// Section 4.1. Stream identifier (chunk type 0xff).
			if chunkLen != magicBodyLen {
				return nil, ErrCorrupt
			}

			if string(buf[:len(magicBody)]) != magicBody {
				if string(buf[:len(magicBodyS2)]) != magicBodyS2 {
					if string(buf[:magicBodyLen]) != magicBodySnappy {
						return nil, ErrCorrupt
					}
				}
			}
			continue
		case chunkTypeEOF:
			continue
		}

		if chunkType <= maxNonSkippableChunk {
			// Section 4.5. Reserved unskippable chunks (chunk types 0x03-0x3f).
			fmt.Println("UN:", chunkType)
			return nil, ErrUnsupported
		}
		// Skip user chunks and padding.
	}
}

// JSON returns the index as JSON text.
func (i *Index) JSON() []byte {
	type offset struct {
		CompressedOffset   int64 `json:"compressed"`
		UncompressedOffset int64 `json:"uncompressed"`
	}
	x := struct {
		TotalUncompressed int64    `json:"total_uncompressed"` // Total Uncompressed size if known. Will be -1 if unknown.
		TotalCompressed   int64    `json:"total_compressed"`   // Total Compressed size if known. Will be -1 if unknown.
		Offsets           []offset `json:"offsets"`
		EstBlockUncomp    int64    `json:"est_block_uncompressed"`
	}{
		TotalUncompressed: i.TotalUncompressed,
		TotalCompressed:   i.TotalCompressed,
		EstBlockUncomp:    i.estBlockUncomp,
	}
	for _, v := range i.Offsets {
		x.Offsets = append(x.Offsets, offset{CompressedOffset: v.CompressedOffset, UncompressedOffset: v.UncompressedOffset})
	}
	b, _ := json.MarshalIndent(x, "", "  ")
	return b
}

// RemoveIndexHeaders will trim all headers and trailers from a given index.
// This is expected to save 20 bytes.
// These can be restored using RestoreIndexHeaders.
// This removes a layer of security, but is the most compact representation.
// Returns nil if headers contains errors.
// The returned slice references the provided slice.
func RemoveIndexHeaders(b []byte) []byte {
	const save = 4 + len(IndexHeader) + len(IndexTrailer) + 4
	if len(b) <= save {
		return nil
	}
	if b[0] != chunkTypeIndex {
		return nil
	}
	chunkLen := int(b[1]) | int(b[2])<<8 | int(b[3])<<16
	b = b[4:]

	// Validate we have enough...
	if len(b) < chunkLen {
		return nil
	}
	b = b[:chunkLen]

	if !bytes.Equal(b[:len(IndexHeader)], []byte(IndexHeader)) {
		return nil
	}
	b = b[len(IndexHeader):]
	if !bytes.HasSuffix(b, []byte(IndexTrailer)) {
		return nil
	}
	b = bytes.TrimSuffix(b, []byte(IndexTrailer))

	if len(b) < 4 {
		return nil
	}
	return b[:len(b)-4]
}

// RestoreIndexHeaders will index restore headers removed by RemoveIndexHeaders.
// No error checking is performed on the input.
// If a 0 length slice is sent, it is returned without modification.
func RestoreIndexHeaders(in []byte) []byte {
	if len(in) == 0 {
		return in
	}
	b := make([]byte, 0, 4+len(IndexHeader)+len(in)+len(IndexTrailer)+4)
	b = append(b, chunkTypeIndex, 0, 0, 0)
	b = append(b, []byte(IndexHeader)...)
	b = append(b, in...)

	// Size of block as uint32
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)+4+len(IndexTrailer)))

	// Trailer
	b = append(b, []byte(IndexTrailer)...)

	chunkLen := len(b) - skippableFrameHeader
	b[1] = uint8(chunkLen >> 0)
	b[2] = uint8(chunkLen >> 8)
	b[3] = uint8(chunkLen >> 16)
	return b
}
