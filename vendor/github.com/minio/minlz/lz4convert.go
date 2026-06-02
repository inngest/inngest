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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// lZ4Converter provides conversion from LZ4 blocks as defined here:
// https://github.com/lz4/lz4/blob/dev/doc/lz4_Block_format.md
type lZ4Converter struct {
}

// errDstTooSmall is returned when provided destination is too small.
var errDstTooSmall = errors.New("minlz: destination too small")

// errIncompressible is returned when the block is incompressible.
var errIncompressible = errors.New("minlz: incompressible")

// ConvertBlock will convert an LZ4 block and append it as an MinLZ
// block without a block length to dst.
// The uncompressed size is returned as well.
// dst must have capacity to contain the entire compressed block,
// which may exceed MaxEncodedLen().
func (l *lZ4Converter) ConvertBlock(dst, src []byte) ([]byte, int, error) {
	if len(src) == 0 {
		return dst, 0, nil
	}
	const debug = false
	const lz4MinMatch = 4
	const inlineLits = true

	// The block starts with the varint-encoded length of the decompressed bytes.
	s, d := 0, len(dst)
	dst = dst[:cap(dst)]
	if !debug && hasAsm {
		res, sz := cvtLZ4BlockAsm(dst[d:], src)
		if res < 0 {
			const (
				errCorrupt        = -1
				errDstTooSmallRet = -2
			)
			switch res {
			case errCorrupt:
				return nil, 0, ErrCorrupt
			case errDstTooSmallRet:
				return nil, 0, errDstTooSmall
			default:
				return nil, 0, fmt.Errorf("unexpected result: %d", res)
			}
		}
		if res < sz {
			return nil, 0, errIncompressible
		}
		if d+sz > len(dst) {
			return nil, 0, errDstTooSmall
		}
		return dst[:d+sz], res, nil
	}

	dLimit := len(dst) - 10
	dStart := d

	var lastOffset uint16
	lastOffset = 1
	var uncompressed int
	if debug {
		fmt.Printf("convert block start: len(src): %d, len(dst):%d \n", len(src), len(dst))
	}

	for {
		if s >= len(src) {
			return dst[:d], 0, ErrCorrupt
		}
		if uncompressed > MaxBlockSize {
			return dst[:d], 0, ErrTooLarge
		}
		// Read literal info
		token := src[s]
		ll := int(token >> 4)
		ml := int(lz4MinMatch + (token & 0xf))

		// If upper nibble is 15, literal length is extended
		if token >= 0xf0 {
			for {
				s++
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ll: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return dst[:d], 0, ErrCorrupt
				}
				val := src[s]
				ll += int(val)
				if val != 255 {
					break
				}
			}
		}
		// Skip past token
		if s+ll >= len(src) {
			if debug {
				fmt.Printf("error literals: s+ll (%d+%d) >= len(src) (%d)\n", s, ll, len(src))
			}
			return nil, 0, ErrCorrupt
		}
		s++
		var lits []byte
		if ll > 0 {
			if d+ll > dLimit {
				if debug {
					fmt.Printf("ERR: emit %d literals, d:%d, dLimit: %d\n", ll, d, dLimit)
				}
				return nil, 0, errDstTooSmall
			}
			if debug {
				fmt.Printf("emit %d literals, pos:%d\n", ll, uncompressed)
			}
			lits = src[s : s+ll]
			s += ll
		}

		// Check if we are done...
		if s == len(src) && ml == lz4MinMatch {
			if uncompressed+ll > MaxBlockSize {
				return dst[:d], 0, ErrTooLarge
			}
			uncompressed += ll
			d += emitLiteral(dst[d:], lits)
			break
		}
		// 2 byte offset
		if s >= len(src)-2 {
			if debug {
				fmt.Printf("s (%d) >= len(src)-2 (%d)", s, len(src)-2)
			}
			return nil, 0, ErrCorrupt
		}
		offset := load16(src, s)
		isRepeat := offset == lastOffset
		if len(lits) > 0 {
			// There are no offset >64K, so copy3 doesn't apply.
			if !inlineLits || len(lits) > maxCopy2Lits ||
				(offset <= 1024 && ml > copy2LitMaxLen) || // Comment out for speed.
				offset < 64 ||
				isRepeat {
				d += emitLiteral(dst[d:], lits)
				lits = nil
			}
			uncompressed += ll
		}

		s += 2
		if offset == 0 {
			if debug {
				fmt.Printf("error: offset 0, ml: %d, len(src)-s: %d\n", ml, len(src)-s)
			}
			return nil, 0, ErrCorrupt
		}
		if int(offset) > uncompressed {
			if debug {
				fmt.Printf("error: offset (%d)> uncompressed (%d)\n", offset, uncompressed)
			}
			return nil, 0, ErrCorrupt
		}

		if ml == lz4MinMatch+15 {
			for {
				if s >= len(src) {
					if debug {
						fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
					}
					return nil, 0, ErrCorrupt
				}
				val := src[s]
				s++
				ml += int(val)
				if val != 255 {
					if s >= len(src) {
						if debug {
							fmt.Printf("error reading ml: s (%d) >= len(src) (%d)\n", s, len(src))
						}
						return nil, 0, ErrCorrupt
					}
					break
				}
			}
		}
		if isRepeat {
			if debug {
				fmt.Printf("emit repeat, length: %d, offset: %d, pos:%d\n", ml, offset, uncompressed)
			}
			d += emitRepeat(dst[d:], ml)
		} else {
			if len(lits) > 0 {
				if debug {
					fmt.Printf("emit %d lits + copy, length: %d, offset: %d, pos:%d\n", len(lits), ml, offset, uncompressed)
				}
				d += emitCopyLits2(dst[d:], lits, int(offset), ml)
			} else {
				if debug {
					fmt.Printf("emit copy, length: %d, offset: %d, pos:%d\n", ml, offset, uncompressed)
				}
				d += emitCopy(dst[d:], int(offset), ml)
			}
			lastOffset = offset
		}
		uncompressed += ml
		if d > dLimit {
			return nil, 0, errDstTooSmall
		}
	}
	if uncompressed < d-dStart {
		return nil, 0, errIncompressible
	}
	return dst[:d], uncompressed, nil
}

func (l *lZ4Converter) ConvertStream(w io.Writer, r io.Reader) error {
	var tmp [4]byte
	const debug = false
	for {
		// Read magic
		_, err := io.ReadFull(r, tmp[:4])
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if binary.LittleEndian.Uint32(tmp[:4]) != 0x184D2204 {
			return fmt.Errorf("minlz: invalid lz4 magic: %x", tmp[:4])
		}

		// Read Frame Descriptor
		_, err = io.ReadFull(r, tmp[:2])
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if tmp[0]&(1<<3) != 0 {
			// Content Size - ignore
			var tmp2 [8]byte
			_, err = io.ReadFull(r, tmp2[:8])
			if err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
		}
		if tmp[0]&(1<<0) != 0 {
			// DictID - fail if set
			var tmp2 [4]byte
			_, err = io.ReadFull(r, tmp2[:2])
			if err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
			if tmp2 != [4]byte{0, 0, 0, 0} {
				return fmt.Errorf("minlz: dictID not supported")
			}
		}
		// Version
		if tmp[0]>>6 != 1 {
			return fmt.Errorf("minlz: unknown version: %d %d", tmp[0]>>6, tmp[0])
		}
		// Block Independence
		if tmp[0]&(1<<5) == 0 {
			return fmt.Errorf("minlz: block dependence not supported")
		}
		blockCrc := tmp[0]&(1<<4) != 0
		contentCrc := tmp[0]&(1<<2) != 0
		maxBlockSz := 0
		// Block Maximum Size
		bz := int(tmp[1] >> 4 & 0x7)
		switch bz {
		case 4:
			maxBlockSz = 64 << 10
		case 5:
			maxBlockSz = 256 << 10
		case 6:
			maxBlockSz = 1 << 20
		case 7:
			maxBlockSz = 4 << 20
		default:
			return fmt.Errorf("minlz: invalid block size: %d", bz)
		}
		// Header Checksum
		_, err = io.ReadFull(r, tmp[:1])
		if err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		var n int
		n, err = w.Write(makeHeader(maxBlockSz))
		if err != nil {
			return err
		}
		if n != len(magicChunk)+1 {
			return io.ErrShortWrite
		}

		block := make([]byte, maxBlockSz)
		dst := make([]byte, maxBlockSz)
		uncompSize := MaxEncodedLen(maxBlockSz)
		if debug {
			fmt.Println("hasCrc:", blockCrc)
		}
		for {
			// Read block size
			_, err := io.ReadFull(r, tmp[:4])
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			compressed := true
			blockSize := int(binary.LittleEndian.Uint32(tmp[:4]))
			if blockSize == 0 {
				if blockCrc {
					_, err = io.ReadFull(r, tmp[:4])
					if err != nil {
						if err == io.EOF {
							return io.ErrUnexpectedEOF
						}
						return err
					}
				}
				break
			}
			if blockSize>>31 != 0 {
				compressed = false
				blockSize &= (1 << 31) - 1
			}
			if blockSize > maxBlockSize {
				return fmt.Errorf("minlz: block size too large: %d", blockSize)
			}
			_, err = io.ReadFull(r, block[:blockSize])
			if err != nil {
				return err
			}
			// Read checksum (ignored)
			if blockCrc {
				_, err = io.ReadFull(r, tmp[:4])
				if err != nil {
					if err == io.EOF {
						return io.ErrUnexpectedEOF
					}
					return err
				}
			}
			if !compressed {
				var obuf [8]byte
				uncompressed := block[:blockSize]
				// Set to uncompressed.
				chunkType := uint8(chunkTypeUncompressedData)
				chunkLen := 4 + len(uncompressed)

				// Write as uncompressed.
				checksum := crc(uncompressed)
				obuf[0] = chunkType
				obuf[1] = uint8(chunkLen >> 0)
				obuf[2] = uint8(chunkLen >> 8)
				obuf[3] = uint8(chunkLen >> 16)
				obuf[4] = uint8(checksum >> 0)
				obuf[5] = uint8(checksum >> 8)
				obuf[6] = uint8(checksum >> 16)
				obuf[7] = uint8(checksum >> 24)
				_, err = w.Write(obuf[:8])
				if err != nil {
					return err
				}
				_, err = w.Write(uncompressed)
				if err != nil {
					return err
				}
				uncompSize += len(uncompressed)
				continue
			}
			// Convert block
			out, sz, err := l.ConvertBlock(dst[:0], block[:blockSize])
			if err != nil {
				return err
			}
			out = out[3:]
			if debug {
				fmt.Println(blockSize, "=>", len(out), "uncompressed:", sz, "ratio:", 100*float64(len(out))/float64(blockSize))
			}
			var obuf [8]byte
			chunkType := uint8(chunkTypeMinLZCompressedDataCompCRC)
			chunkLen := 4 + len(out)

			// Write block.
			checksum := crc(out)
			obuf[0] = chunkType
			obuf[1] = uint8(chunkLen >> 0)
			obuf[2] = uint8(chunkLen >> 8)
			obuf[3] = uint8(chunkLen >> 16)
			obuf[4] = uint8(checksum >> 0)
			obuf[5] = uint8(checksum >> 8)
			obuf[6] = uint8(checksum >> 16)
			obuf[7] = uint8(checksum >> 24)
			_, err = w.Write(obuf[:8])
			if err != nil {
				return err
			}
			_, err = w.Write(out)
			uncompSize += sz
		}
		if contentCrc {
			// Read content crc (ignored)
			_, err = io.ReadFull(r, tmp[:4])
			if err != nil {
				if err == io.EOF {
					return io.ErrUnexpectedEOF
				}
				return err
			}
		}
		var tmp [4 + binary.MaxVarintLen64]byte
		tmp[0] = chunkTypeEOF
		// Write uncompressed size.
		n = binary.PutUvarint(tmp[4:], uint64(uncompSize))
		tmp[1] = uint8(n)
		n += 4
		_, err = w.Write(tmp[:n])
		if err != nil {
			return err
		}
	}
}
