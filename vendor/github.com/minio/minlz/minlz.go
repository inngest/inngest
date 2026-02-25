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
	"hash/crc32"
)

const (
	// MaxBlockSize is the maximum value where MaxEncodedLen will return a valid block size.
	MaxBlockSize = 8 << 20

	// MinUserSkippableChunk is the lowest user defined skippable chunk ID.
	// All chunks IDs within this range will be ignored if not handled.
	MinUserSkippableChunk = 0x80

	// MaxUserSkippableChunk is the last user defined skippable chunk ID.
	MaxUserSkippableChunk = 0xbf

	// MinUserNonSkippableChunk is the lowest user defined non-skippable chunk ID.
	// All chunks IDs within this range will cause an error if not handled.
	MinUserNonSkippableChunk = 0xc0

	// MaxUserNonSkippableChunk is the last user defined non-skippable chunk ID.
	MaxUserNonSkippableChunk = 0xfd

	// ChunkTypePadding is a padding chunk.
	ChunkTypePadding = 0xfe

	// ChunkTypeStreamIdentifier is the Snappy/S2/MinLZ stream id chunk.
	ChunkTypeStreamIdentifier = 0xff

	// MaxUserChunkSize is the maximum possible size of a single chunk.
	MaxUserChunkSize = 1<<24 - 1 // 16777215
)

// debugging constants that will enable debug printing and extra checks.
const (
	debugValidateBlocks = false

	// Enable debug output for encoding.
	debugEncode = false

	// Enable debug output for Go decoding.
	debugDecode = false
)

/*
Each encoded block begins with the varint-encoded length of the decoded data,
followed by a sequence of chunks. Chunks begin and end on byte boundaries. The
first byte of each chunk is broken into its 2 least and 6 most significant bits
called l and m: l ranges in [0, 4) and m ranges in [0, 64). l is the chunk tag.
Zero means a literal tag. All other values mean a copy tag.

For literal tags:
  - If m < 60, the next 1 + m bytes are literal bytes.
  - Otherwise, let n be the little-endian unsigned integer denoted by the next
    m - 59 bytes. The next 1 + n bytes after that are literal bytes.
*/
const (
	tagLiteral    = 0x00
	tagRepeat     = 0x00 | (1 << 2)
	tagCopy1      = 0x01
	tagCopy2      = 0x02
	tagCopy3      = 0x03 | 4
	tagCopy2Fused = 0x03
)

const (
	checksumSize     = 4
	chunkHeaderSize  = 4
	magicChunk       = "\xff\x06\x00\x00" + magicBody
	magicBodySnappy  = "sNaPpY"
	magicBodyS2      = "S2sTwO"
	magicBody        = "MinLz"
	magicBodyLen     = len(magicBody) + 1
	magicChunkS2     = "\xff\x06\x00\x00" + magicBodyS2
	magicChunkSnappy = "\xff\x06\x00\x00" + magicBodySnappy
	maxBlockLog      = 23

	// maxBlockSize is the maximum size of the input to encodeBlock.
	//
	// For the framing format (Writer type instead of Encode function),
	// this is the maximum uncompressed size of a block.
	maxBlockSize = 1 << maxBlockLog

	// minBlockSize is the minimum size of block setting when creating a writer.
	minBlockSize = 4 << 10

	skippableFrameHeader = 4

	// Default block size
	defaultBlockSize = 2 << 20

	// maxSnappyBlockSize is the maximum snappy block size in streams.
	maxSnappyBlockSize = 1 << 16

	// maxS2BlockSize is the maximum s2 block size in streams.
	maxS2BlockSize = 4 << 20

	obufHeaderLen = checksumSize + chunkHeaderSize
)

// Internal chunk ids
const (
	chunkTypeLegacyCompressedData       = 0x00
	chunkTypeUncompressedData           = 0x01
	chunkTypeMinLZCompressedData        = 0x02
	chunkTypeMinLZCompressedDataCompCRC = 0x03
	chunkTypeEOF                        = 0x20
	maxNonSkippableChunk                = 0x3f
	chunkTypeIndex                      = 0x40 // chunk id of MinLZ index
	legacyIndexChunk                    = 0x99 // S2 index chunk id (now in user-skippable range)
)

var crcTable = crc32.MakeTable(crc32.Castagnoli)

// crc implements the checksum specified in section 3 of
// https://github.com/google/snappy/blob/master/framing_format.txt
func crc(b []byte) uint32 {
	c := crc32.Update(0, crcTable, b)
	return c>>15 | c<<17 + 0xa282ead8
}

type byter interface {
	Bytes() []byte
}

var _ byter = &bytes.Buffer{}
