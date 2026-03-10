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
	"math"
	"runtime"
	"sync"

	"github.com/klauspost/compress/s2"
)

// ErrCantSeek is returned if the stream cannot be seeked.
type ErrCantSeek struct {
	Reason string
}

// Error returns the error as string.
func (e ErrCantSeek) Error() string {
	return fmt.Sprintf("minlz: Can't seek because %s", e.Reason)
}

// NewReader returns a new Reader that decompresses from r, using the framing
// format described at
// https://github.com/google/snappy/blob/master/framing_format.txt with S2 changes.
func NewReader(r io.Reader, opts ...ReaderOption) *Reader {
	nr := Reader{
		r:             r,
		maxBlock:      maxBlockSize,
		allowFallback: false,
	}
	for _, opt := range opts {
		if err := opt(&nr); err != nil {
			nr.err = err
			return &nr
		}
	}
	nr.maxBufSize = MaxEncodedLen(nr.maxBlock) + checksumSize
	nr.maxBlockOrg = nr.maxBlock
	nr.readHeader = nr.ignoreStreamID
	nr.paramsOK = true
	return &nr
}

// ReaderOption is an option for creating a decoder.
type ReaderOption func(*Reader) error

// ReaderMaxBlockSize allows controlling allocations if the stream
// has been compressed with a smaller WriterBlockSize, or with the default 1MB.
// Blocks must be this size or smaller to decompress,
// otherwise the decoder will return ErrUnsupported.
//
// For streams compressed with Snappy this can safely be set to 64KB (64 << 10).
//
// Default is the maximum limit of 8MB.
func ReaderMaxBlockSize(blockSize int) ReaderOption {
	return func(r *Reader) error {
		if blockSize > maxBlockSize || blockSize <= minBlockSize {
			return errors.New("minlz: invalid block size. Must be <= 8MB and >= 4KB")
		}
		r.maxBlock = blockSize
		return nil
	}
}

// ReaderIgnoreStreamIdentifier will make the reader skip the expected
// stream identifier at the beginning of the stream.
// This can be used when serving a stream that has been forwarded to a specific point.
// Validation of EOF length is also disabled.
func ReaderIgnoreStreamIdentifier() ReaderOption {
	return func(r *Reader) error {
		r.ignoreStreamID = true
		return nil
	}
}

// ReaderUserChunkCB will register a callback for chunks with the specified ID.
// ID must be a Reserved skippable chunks ID, 0x40-0xfd (inclusive).
// For each chunk with the ID, the callback is called with the content.
// Any returned non-nil error will abort decompression.
// Only one callback per ID is supported, latest sent will be used.
// Sending a nil function will disable previous callbacks.
// You can peek the stream, triggering the callback, by doing a Read with a 0
// byte buffer.
func ReaderUserChunkCB(id uint8, fn func(r io.Reader) error) ReaderOption {
	return func(r *Reader) error {
		if id < MinUserSkippableChunk || id > MaxUserNonSkippableChunk {
			return fmt.Errorf("ReaderUserChunkCB: Invalid id provided, must be 0x80-0xfd (inclusive)")
		}
		r.skippableCB[id-MinUserSkippableChunk] = fn
		return nil
	}
}

// ReaderIgnoreCRC will make the reader skip CRC calculation and checks.
func ReaderIgnoreCRC() ReaderOption {
	return func(r *Reader) error {
		r.ignoreCRC = true
		return nil
	}
}

// ReaderFallback will enable/disable S2/Snappy fallback.
func ReaderFallback(b bool) ReaderOption {
	return func(r *Reader) error {
		r.allowFallback = b
		return nil
	}
}

// Reader is an io.Reader that can read Snappy-compressed bytes.
type Reader struct {
	r           io.Reader
	err         error
	decoded     []byte
	buf         []byte
	tmp         [16]byte
	skippableCB [MaxUserNonSkippableChunk - MinUserSkippableChunk + 1]func(r io.Reader) error
	blockStart  int64 // Uncompressed offset at start of current.
	index       *Index

	// decoded[i:j] contains decoded bytes that have not yet been passed on.
	i, j int
	// maximum block size allowed.
	maxBlock    int
	maxBlockOrg int
	// maximum expected buffer size.
	maxBufSize     int
	readHeader     bool
	paramsOK       bool
	snappyFrame    bool
	ignoreStreamID bool
	ignoreCRC      bool
	allowFallback  bool
	wantEOF        bool
}

// GetBufferCapacity returns the capacity of the internal buffer.
// This might be useful to know when reusing the same reader in combination
// with the lazy buffer option.
func (r *Reader) GetBufferCapacity() int {
	return cap(r.buf)
}

// ensureBufferSize will ensure that the buffer can take at least n bytes.
// If false is returned the buffer exceeds maximum allowed size.
func (r *Reader) ensureBufferSize(n int) bool {
	if n > r.maxBufSize {
		r.err = ErrCorrupt
		return false
	}
	if cap(r.buf) >= n {
		return true
	}
	// Realloc buffer.
	r.buf = make([]byte, n, n)
	return true
}

// Reset discards any buffered data, resets all state, and switches the Snappy
// reader to read from r. This permits reusing a Reader rather than allocating
// a new one.
func (r *Reader) Reset(reader io.Reader) {
	if !r.paramsOK {
		return
	}
	r.index = nil
	r.r = reader
	r.err = nil
	r.i = 0
	r.j = 0
	r.blockStart = 0
	r.readHeader = r.ignoreStreamID
	r.wantEOF = false
	r.snappyFrame = false
	r.maxBlock = r.maxBlockOrg
	r.maxBufSize = MaxEncodedLen(r.maxBlock) + checksumSize
}

func (r *Reader) readFull(p []byte, allowEOF bool) (ok bool) {
	if _, r.err = io.ReadFull(r.r, p); r.err != nil {
		if r.err == io.ErrUnexpectedEOF || (r.err == io.EOF && !allowEOF) {
			r.err = ErrCorrupt
		}
		return false
	}
	return true
}

// skippable will skip n bytes.
// tmp is used as a temporary buffer for reading.
// The supplied slice does not need to be the size of the read.
func (r *Reader) skippable(tmp []byte, n int, allowEOF bool, id uint8) (ok bool) {
	if len(tmp) < 4096 {
		tmp = make([]byte, 4096)
	}
	if id <= maxNonSkippableChunk {
		r.err = fmt.Errorf("internal error: skippable id >= 0x40")
		return false
	}
	if id >= MinUserSkippableChunk && id <= MaxUserNonSkippableChunk {
		if fn := r.skippableCB[id-MinUserSkippableChunk]; fn != nil {
			rd := io.LimitReader(r.r, int64(n))
			r.err = fn(rd)
			if r.err != nil {
				return false
			}
			_, r.err = io.CopyBuffer(io.Discard, rd, tmp)
			return r.err == nil
		} else if id >= MinUserNonSkippableChunk && id <= MaxUserNonSkippableChunk {
			r.err = errors.New("un-skippable user chunk found")
			return false
		}
	}
	// Read and discard.
	for n > 0 {
		if n < len(tmp) {
			tmp = tmp[:n]
		}
		if _, r.err = io.ReadFull(r.r, tmp); r.err != nil {
			if errors.Is(r.err, io.ErrUnexpectedEOF) || (r.err == io.EOF && !allowEOF) {
				r.err = ErrCorrupt
			}
			return false
		}
		n -= len(tmp)
	}
	return true
}

// Read satisfies the io.Reader interface.
func (r *Reader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	const debug = false
	for {
		if r.i < r.j {
			n := copy(p, r.decoded[r.i:r.j])
			r.i += n
			return n, nil
		}
		if !r.readFull(r.tmp[:4], !r.wantEOF) {
			if debug {
				if r.err != io.EOF {
					fmt.Println("Readfull failed", r.err)
				}
			}
			return 0, r.err
		}
		chunkType := r.tmp[0]
		chunkLen := int(r.tmp[1]) | int(r.tmp[2])<<8 | int(r.tmp[3])<<16
		if debug {
			fmt.Printf("chunkType: 0x%x, chunkLen: %d\n", chunkType, chunkLen)
		}

		if !r.readHeader {
			if chunkType == ChunkTypeStreamIdentifier {
				r.readHeader = true
			} else if chunkType <= maxNonSkippableChunk && chunkType != chunkTypeEOF {
				if debug {
					fmt.Println("ERR: Header not found, got chunk", chunkType)
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
		}
		// The chunk types are specified at
		// https://github.com/google/snappy/blob/master/framing_format.txt
		switch chunkType {
		case chunkTypeMinLZCompressedData, chunkTypeMinLZCompressedDataCompCRC:
			r.blockStart += int64(r.j)
			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				if debug {
					fmt.Println("ERR: Read chunk too short, want checksum", chunkLen)
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err == nil {
					r.err = ErrTooLarge
				}
				return 0, r.err
			}
			buf := r.buf[:chunkLen]
			if !r.readFull(buf, false) {
				return 0, r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			n, hdrLen, err := decodedLen(buf)
			if err != nil {
				if debug {
					fmt.Println("ERR: decodedLen:", err)
				}
				r.err = err
				return 0, r.err
			}

			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}
			if n > len(r.decoded) {
				r.decoded = make([]byte, n)
			}
			buf = buf[hdrLen:]
			if n == 0 || n < len(buf) {
				if debug {
					fmt.Println("ERR: Invalid decompressed length:", n, "buf length:", len(buf))
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
			if ret := minLZDecode(r.decoded[:n], buf); ret != 0 {
				if debug {
					fmt.Println("ERR: Decoder returned error code:", ret)
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
			toCRC := r.decoded[:n]
			if chunkType == chunkTypeMinLZCompressedDataCompCRC {
				toCRC = buf
			}
			if !r.ignoreCRC && crc(toCRC) != checksum {
				if debug {
					fmt.Println("ERR: CRC mismatch")
				}
				r.err = ErrCRC
				return 0, r.err
			}
			r.i, r.j = 0, n
			continue

		case chunkTypeLegacyCompressedData:
			if !r.allowFallback {
				if debug {
					fmt.Println("ERR: Legacy compressed data not allowed")
				}
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.blockStart += int64(r.j)
			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err == nil {
					r.err = ErrTooLarge
				}
				return 0, r.err
			}
			buf := r.buf[:chunkLen]
			if !r.readFull(buf, false) {
				return 0, r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			n, err := DecodedLen(buf)
			if err != nil {
				r.err = err
				return 0, r.err
			}
			if r.snappyFrame && n > maxSnappyBlockSize || n > maxS2BlockSize {
				r.err = ErrCorrupt
				return 0, r.err
			}

			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}
			if n > len(r.decoded) {
				r.decoded = make([]byte, n)
			}
			if _, err := s2.Decode(r.decoded, buf); err != nil {
				r.err = err
				return 0, r.err
			}
			if !r.ignoreCRC && crc(r.decoded[:n]) != checksum {
				r.err = ErrCRC
				return 0, r.err
			}
			r.i, r.j = 0, n
			continue

		case chunkTypeUncompressedData:
			r.blockStart += int64(r.j)
			// Section 4.3. Uncompressed data (chunk type 0x01).
			if chunkLen < checksumSize {
				if debug {
					fmt.Println("chunkLen < checksumSize", r.err)
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err == nil {
					r.err = ErrTooLarge
				}
				return 0, r.err
			}
			buf := r.buf[:checksumSize]
			if !r.readFull(buf, false) {
				if debug {
					fmt.Println("Readfull failed", r.err)
				}
				return 0, r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			// Read directly into r.decoded instead of via r.buf.
			n := chunkLen - checksumSize
			if r.snappyFrame && n > maxSnappyBlockSize {
				if debug {
					fmt.Println("ERR: Snappy block too big")
				}
				r.err = ErrCorrupt
				return 0, r.err
			}
			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}
			if n > len(r.decoded) {
				r.decoded = make([]byte, n)
			}
			if !r.readFull(r.decoded[:n], false) {
				if debug {
					fmt.Println("Readfull2 failed", r.err)
				}
				return 0, r.err
			}
			if !r.ignoreCRC && crc(r.decoded[:n]) != checksum {
				r.err = ErrCRC
				return 0, r.err
			}

			r.i, r.j = 0, n
			continue
		case chunkTypeEOF:
			if debug {
				fmt.Println("EOF chunk", chunkLen)
			}
			if chunkLen > binary.MaxVarintLen64 {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if chunkLen != 0 {
				buf := r.tmp[:chunkLen]
				if !r.readFull(buf, false) {
					return 0, r.err
				}
				if !r.ignoreStreamID {
					wantSize, n := binary.Uvarint(buf[:chunkLen])
					if n != chunkLen {
						if debug {
							fmt.Println("ERR: EOF chunk length mismatch", n, chunkLen)
						}
						r.err = ErrCorrupt
						return 0, r.err
					}
					if wantSize != uint64(r.blockStart+int64(r.j)) {
						if debug {
							fmt.Println("ERR: EOF data length mismatch", wantSize, r.blockStart+int64(r.j))
						}
						r.err = ErrCorrupt
						return 0, r.err
					}
					if debug {
						fmt.Println("EOF length verified", wantSize, "==", r.blockStart+int64(r.j), r.blockStart, r.j)
					}
				}
			}
			r.wantEOF = false
			r.readHeader = false
			continue
		case ChunkTypeStreamIdentifier:
			// Section 4.1. Stream identifier (chunk type 0xff).
			if chunkLen != magicBodyLen {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if !r.readFull(r.tmp[:magicBodyLen], false) {
				return 0, r.err
			}
			r.blockStart = 0
			r.i, r.j = 0, 0
			if string(r.tmp[:len(magicBody)]) == magicBody {
				if !r.minLzHeader(r.tmp[:magicBodyLen]) {
					return 0, r.err
				}
				continue
			}

			if !r.allowFallback {
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.maxBlock = r.maxBlockOrg
			if string(r.tmp[:magicBodyLen]) != magicBodyS2 && string(r.tmp[:magicBodyLen]) != magicBodySnappy {
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.snappyFrame = string(r.tmp[:magicBodyLen]) == magicBodySnappy
			continue
		}

		if chunkType <= maxNonSkippableChunk {
			// Section 4.5. Reserved unskippable chunks (chunk types 0x02-0x7f).
			// fmt.Printf("ERR chunktype: 0x%x\n", chunkType)
			r.err = ErrUnsupported
			return 0, r.err
		}

		// Handle skippable chunks
		if !r.skippable(r.buf, chunkLen, false, chunkType) {
			return 0, r.err
		}
	}
}

// WriteTo writes data to w until there's no more data to write or
// when an error occurs. The return value n is the number of bytes
// written. Any error encountered during the write is also returned.
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) {
	if r.i > 0 || r.j > 0 {
		if r.i != r.j {
			missing := r.decoded[r.i:r.j]
			n2, err := w.Write(missing)
			if err == nil && n2 != len(missing) {
				err = io.ErrShortWrite
			}
			n += int64(n2)
			if err != nil {
				r.err = err
				return n, r.err
			}
		}
		r.blockStart += int64(r.j)
		r.i, r.j = 0, 0
	}
	n2, err := r.DecodeConcurrent(w, runtime.NumCPU())
	return n + n2, err
}

// DecodeConcurrent will decode the full stream to w.
// This function should not be combined with reading, seeking or other operations.
// Up to 'concurrent' goroutines will be used.
// If <= 0, min(runtime.NumCPU, runtime.GOMAXPROCS, 8) will be used.
// On success the number of bytes decompressed nil and is returned.
// This is mainly intended for bigger streams, since it will cause more allocations.
func (r *Reader) DecodeConcurrent(w io.Writer, concurrent int) (written int64, err error) {
	if r.i > 0 || r.j > 0 {
		return 0, errors.New("DecodeConcurrent called after Read")
	}
	if concurrent <= 0 {
		concurrent = min(runtime.NumCPU(), runtime.GOMAXPROCS(0), 8)
	}
	if concurrent == 1 {
		if rf, ok := w.(io.ReaderFrom); ok {
			return rf.ReadFrom(r)
		}
		buf := make([]byte, 128<<10)
		return io.CopyBuffer(w, r, buf)
	}

	const debug = false
	// Write to output
	var errMu sync.Mutex
	var aErr error
	setErr := func(e error) (ok bool) {
		errMu.Lock()
		defer errMu.Unlock()
		if e == nil {
			return aErr == nil
		}
		if aErr == nil {
			aErr = e
		}
		return false
	}
	hasErr := func() (ok bool) {
		errMu.Lock()
		v := aErr != nil
		errMu.Unlock()
		return v
	}

	var aWritten int64
	toRead := make(chan []byte, concurrent+1)
	writtenBlocks := make(chan []byte, concurrent+1)
	queue := make(chan chan io.Writer, concurrent)
	reUse := make(chan chan io.Writer, concurrent)
	for i := 0; i < concurrent; i++ {
		toRead <- nil // We do not know max block size yet, so don't alloc yet
		writtenBlocks <- nil
		reUse <- make(chan io.Writer, 1)
	}
	// Add extra in+out block, so we can read ahead by one.
	toRead <- nil
	writtenBlocks <- nil

	// Writer.
	// We let the goroutine that did the decompression do the writing.
	// We are more likely that decompressed data will be in local cache.
	var wg sync.WaitGroup
	wg.Add(1)
	writeBuf := func(buf []byte, entry chan io.Writer) {
		// Wait until our turn
		w := <-entry
		defer func() {
			if buf != nil {
				writtenBlocks <- buf
			}
			reUse <- entry

			// Take next top entry from queue.
			next, ok := <-queue
			if !ok {
				wg.Done()
				return
			}
			// Forward writer
			next <- w
		}()
		n, err := w.Write(buf)
		if err != nil {
			setErr(err)
			return
		}
		want := len(buf)
		if n != want {
			setErr(io.ErrShortWrite)
			return
		}
		aWritten += int64(n)
	}

	// Seed writer
	seed := <-reUse
	go writeBuf(nil, seed)
	seed <- w

	// Cleanup
	defer func() {
		if r.err != nil {
			setErr(r.err)
		} else if err != nil {
			setErr(err)
		}
		close(queue)
		wg.Wait()
		if err == nil {
			err = aErr
		}
		written = aWritten
	}()

	// Reader
	for !hasErr() {
		if !r.readFull(r.tmp[:4], !r.wantEOF) {
			if r.err == io.EOF {
				r.err = nil
			}
			return 0, r.err
		}
		chunkType := r.tmp[0]
		chunkLen := int(r.tmp[1]) | int(r.tmp[2])<<8 | int(r.tmp[3])<<16
		if !r.readHeader {
			if chunkType == ChunkTypeStreamIdentifier {
				r.readHeader = true
			} else if chunkType <= maxNonSkippableChunk && chunkType != chunkTypeEOF {
				r.err = ErrCorrupt
				return 0, r.err
			}
		}

		// The chunk types are specified at
		// https://github.com/google/snappy/blob/master/framing_format.txt
		switch chunkType {
		case chunkTypeLegacyCompressedData:
			if !r.allowFallback {
				if debug {
					fmt.Println("ERR: Legacy compressed data not allowed")
				}
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.blockStart += int64(r.j)
			r.j = 0
			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if chunkLen > r.maxBufSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			orgBuf := <-toRead
			if cap(orgBuf) < chunkLen {
				orgBuf = make([]byte, r.maxBufSize)
			}
			buf := orgBuf[:chunkLen]

			if !r.readFull(buf, false) {
				return 0, r.err
			}

			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			n, err := DecodedLen(buf)
			if err != nil {
				r.err = err
				return 0, r.err
			}
			if r.snappyFrame && n > maxSnappyBlockSize {
				r.err = ErrCorrupt
				return 0, r.err
			}

			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}
			wg.Add(1)

			decoded := <-writtenBlocks
			if cap(decoded) < n {
				decoded = make([]byte, r.maxBlock)
			}
			entry := <-reUse
			queue <- entry
			r.blockStart += int64(r.j)
			go func() {
				defer wg.Done()
				decoded = decoded[:n]
				_, err := s2.Decode(decoded, buf)
				toRead <- orgBuf
				if err != nil {
					writtenBlocks <- decoded
					setErr(err)
					writeBuf(nil, entry)
					return
				}
				if !r.ignoreCRC && crc(decoded) != checksum {
					writtenBlocks <- decoded
					setErr(ErrCRC)
					writeBuf(nil, entry)
					return
				}
				writeBuf(decoded, entry)
			}()
			continue
		case chunkTypeMinLZCompressedData, chunkTypeMinLZCompressedDataCompCRC:
			r.blockStart += int64(r.j)
			r.j = 0

			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if chunkLen > r.maxBufSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			orgBuf := <-toRead
			if cap(orgBuf) < chunkLen {
				orgBuf = make([]byte, r.maxBufSize)
			}
			buf := orgBuf[:chunkLen]

			if !r.readFull(buf, false) {
				return 0, r.err
			}

			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			n, hdrSize, err := decodedLen(buf)
			if err != nil {
				r.err = err
				return 0, r.err
			}

			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}
			r.blockStart += int64(n)
			buf = buf[hdrSize:]
			if n == 0 || n < len(buf) {
				r.err = ErrCorrupt
				return 0, r.err
			}

			wg.Add(1)

			decoded := <-writtenBlocks
			if cap(decoded) < n {
				decoded = make([]byte, r.maxBlock)
			}
			entry := <-reUse
			queue <- entry
			go func() {
				defer wg.Done()
				decoded = decoded[:n]
				ret := minLZDecode(decoded, buf)

				toRead <- orgBuf
				if ret != 0 {
					if debug {
						fmt.Println("ERR: Decoder returned error code:", ret)
					}
					writtenBlocks <- decoded
					setErr(ErrCorrupt)
					writeBuf(nil, entry)
					return
				}
				toCRC := decoded
				if chunkType == chunkTypeMinLZCompressedDataCompCRC {
					toCRC = buf
				}
				if !r.ignoreCRC && crc(toCRC) != checksum {
					if debug {
						fmt.Println("ERR: CRC mismatch", crc(decoded), checksum)
					}
					writtenBlocks <- decoded
					setErr(ErrCRC)
					writeBuf(nil, entry)
					return
				}
				writeBuf(decoded, entry)
			}()
			continue
		case chunkTypeUncompressedData:
			// Section 4.3. Uncompressed data (chunk type 0x01).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if chunkLen > r.maxBufSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			r.blockStart += int64(r.j)
			r.j = 0
			// Grab write buffer
			orgBuf := <-writtenBlocks
			if cap(orgBuf) < chunkLen {
				orgBuf = make([]byte, r.maxBufSize)
			}
			buf := orgBuf[:checksumSize]
			if !r.readFull(buf, false) {
				return 0, r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			// Read content.
			n := chunkLen - checksumSize
			r.blockStart += int64(n)

			if r.snappyFrame && n > maxSnappyBlockSize {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if n > r.maxBlock {
				r.err = ErrTooLarge
				return 0, r.err
			}

			// Read uncompressed
			buf = orgBuf[:n]
			if !r.readFull(buf, false) {
				return 0, r.err
			}

			if !r.ignoreCRC && crc(buf) != checksum {
				r.err = ErrCRC
				return 0, r.err
			}
			entry := <-reUse
			queue <- entry
			go writeBuf(buf, entry)
			continue
		case chunkTypeEOF:
			if chunkLen != 0 {
				if chunkLen > binary.MaxVarintLen64 {
					r.err = ErrCorrupt
					return 0, r.err
				}

				buf := r.tmp[:chunkLen]
				if !r.readFull(buf, false) {
					return 0, r.err
				}
				if !r.ignoreStreamID {
					wantSize, n := binary.Uvarint(buf[:chunkLen])
					if n != chunkLen {
						if debug {
							fmt.Println("ERR: EOF chunk length mismatch", n, chunkLen)
						}
						r.err = ErrCorrupt
						return 0, r.err
					}
					if wantSize != uint64(r.blockStart+int64(r.j)) {
						if debug {
							fmt.Println("ERR: EOF data length mismatch", wantSize, r.blockStart+int64(r.j))
						}
						r.err = ErrCorrupt
						return 0, r.err
					}
				}
			}
			r.wantEOF = false
			r.readHeader = false
			continue
		case ChunkTypeStreamIdentifier:
			// Section 4.1. Stream identifier (chunk type 0xff).
			if chunkLen != magicBodyLen {
				r.err = ErrCorrupt
				return 0, r.err
			}
			if !r.readFull(r.tmp[:magicBodyLen], false) {
				return 0, r.err
			}
			r.blockStart = 0
			r.i, r.j = 0, 0
			if string(r.tmp[:len(magicBody)]) == magicBody {
				if !r.minLzHeader(r.tmp[:magicBodyLen]) {
					return 0, r.err
				}
				continue
			}
			if !r.allowFallback {
				if debug {
					fmt.Println("!fallback")
				}
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.maxBlock = r.maxBlockOrg

			if string(r.tmp[:magicBodyLen]) != magicBodyS2 && string(r.tmp[:magicBodyLen]) != magicBodySnappy {
				r.err = ErrUnsupported
				return 0, r.err
			}
			r.snappyFrame = string(r.tmp[:magicBodyLen]) == magicBodySnappy
			continue
		}

		if chunkType <= maxNonSkippableChunk {
			if debug {
				fmt.Printf("ERR chunktype: 0x%x\n", chunkType)
			}
			// Section 4.5. Reserved unskippable chunks (chunk types 0x04-0x3f).
			r.err = ErrUnsupported
			return 0, r.err
		}

		// Section 4.4 Padding (chunk type 0xfe).
		// Section 4.6. Reserved skippable chunks (chunk types 0x40-0xfd).
		if !r.skippable(r.buf, chunkLen, false, chunkType) {
			return 0, r.err
		}
	}
	return 0, r.err
}

func (r *Reader) minLzHeader(hdr []byte) (ok bool) {
	if len(hdr) < magicBodyLen {
		r.err = ErrCorrupt
		return false
	}
	// Upper 2 bits most be 0
	if hdr[magicBodyLen-1]&(3<<6) != 0 {
		r.err = ErrCorrupt
		return false
	}
	n := hdr[magicBodyLen-1]&15 + 10
	if n > maxBlockLog {
		r.err = ErrCorrupt
		return false
	}
	r.maxBlock = 1 << n
	r.maxBufSize = MaxEncodedLen(r.maxBlock) + checksumSize
	if r.maxBlock > r.maxBlockOrg {
		r.err = ErrTooLarge
		return false
	}
	if !r.ensureBufferSize(MaxEncodedLen(r.maxBlock) + checksumSize) {
		if r.err == nil {
			r.err = ErrTooLarge
		}
		return false
	}
	if len(r.decoded) < r.maxBlock {
		r.decoded = make([]byte, 0, n)
	}
	r.snappyFrame = false
	r.wantEOF = true
	return true
}

// Skip will skip n bytes forward in the decompressed output.
// For larger skips this consumes less CPU and is faster than reading output and discarding it.
// CRC is not checked on skipped blocks.
// io.ErrUnexpectedEOF is returned if the stream ends before all bytes have been skipped.
// If a decoding error is encountered subsequent calls to Read will also fail.
func (r *Reader) Skip(n int64) error {
	if n < 0 {
		return errors.New("attempted negative skip")
	}
	if r.err != nil {
		return r.err
	}

	for n > 0 {
		if r.i < r.j {
			// Skip in buffer.
			// decoded[i:j] contains decoded bytes that have not yet been passed on.
			left := int64(r.j - r.i)
			if left >= n {
				tmp := int64(r.i) + n
				if tmp > math.MaxInt32 {
					return errors.New("minlz: internal overflow in skip")
				}
				r.i = int(tmp)
				return nil
			}
			n -= int64(r.j - r.i)
			r.i = r.j
		}

		// Buffer empty; read blocks until we have content.
		if !r.readFull(r.tmp[:4], !r.wantEOF) {
			if r.err == io.EOF {
				r.err = io.ErrUnexpectedEOF
			}
			return r.err
		}
		chunkType := r.tmp[0]
		if !r.readHeader {
			if chunkType == ChunkTypeStreamIdentifier {
				r.readHeader = true
			} else if chunkType <= maxNonSkippableChunk && chunkType != chunkTypeEOF {
				r.err = ErrCorrupt
				return r.err
			}
		}

		chunkLen := int(r.tmp[1]) | int(r.tmp[2])<<8 | int(r.tmp[3])<<16

		// The chunk types are specified at
		// https://github.com/google/snappy/blob/master/framing_format.txt
		switch chunkType {
		case chunkTypeMinLZCompressedData, chunkTypeMinLZCompressedDataCompCRC:
			r.blockStart += int64(r.j)
			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err == nil {
					r.err = ErrTooLarge
				}
				return r.err
			}
			buf := r.buf[:chunkLen]
			if !r.readFull(buf, false) {
				return r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			dLen, hdrSize, err := decodedLen(buf)
			if err != nil {
				r.err = err
				return r.err
			}
			if dLen > r.maxBlock {
				r.err = ErrTooLarge
				return r.err
			}
			if dLen == 0 || dLen < len(buf)-hdrSize {
				r.err = ErrCorrupt
				return r.err
			}
			// Check if destination is within this block
			if int64(dLen) > n {
				if len(r.decoded) < dLen {
					r.decoded = make([]byte, dLen)
				}
				buf = buf[hdrSize:]
				if ret := minLZDecode(r.decoded[:dLen], buf); ret != 0 {
					r.err = ErrTooLarge
					return r.err
				}
				toCRC := r.decoded[:dLen]
				if chunkType == chunkTypeMinLZCompressedDataCompCRC {
					toCRC = buf
				}
				if !r.ignoreCRC && crc(toCRC) != checksum {
					r.err = ErrCRC
					return r.err
				}
			} else {
				// Skip block completely
				n -= int64(dLen)
				r.blockStart += int64(dLen)
				dLen = 0
			}
			r.i, r.j = 0, dLen
			continue
		case chunkTypeLegacyCompressedData:
			if !r.allowFallback {
				r.err = ErrUnsupported
				return r.err
			}

			r.blockStart += int64(r.j)
			// Section 4.2. Compressed data (chunk type 0x00).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err == nil {
					r.err = ErrTooLarge
				}
				return r.err
			}
			buf := r.buf[:chunkLen]
			if !r.readFull(buf, false) {
				return r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			buf = buf[checksumSize:]

			dLen, err := DecodedLen(buf)
			if err != nil {
				r.err = err
				return r.err
			}
			if dLen > r.maxBlock {
				r.err = ErrCorrupt
				return r.err
			}
			// Check if destination is within this block
			if int64(dLen) > n {
				if len(r.decoded) < dLen {
					r.decoded = make([]byte, dLen)
				}
				if _, err := s2.Decode(r.decoded, buf); err != nil {
					r.err = err
					return r.err
				}
				if crc(r.decoded[:dLen]) != checksum {
					r.err = ErrCorrupt
					return r.err
				}
			} else {
				// Skip block completely
				n -= int64(dLen)
				r.blockStart += int64(dLen)
				dLen = 0
			}
			r.i, r.j = 0, dLen
			continue
		case chunkTypeUncompressedData:
			r.blockStart += int64(r.j)
			// Section 4.3. Uncompressed data (chunk type 0x01).
			if chunkLen < checksumSize {
				r.err = ErrCorrupt
				return r.err
			}
			if !r.ensureBufferSize(chunkLen) {
				if r.err != nil {
					r.err = ErrTooLarge
				}
				return r.err
			}
			buf := r.buf[:checksumSize]
			if !r.readFull(buf, false) {
				return r.err
			}
			checksum := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
			// Read directly into r.decoded instead of via r.buf.
			n2 := chunkLen - checksumSize
			if n2 > len(r.decoded) {
				if n2 > r.maxBlock {
					r.err = ErrCorrupt
					return r.err
				}
				r.decoded = make([]byte, n2)
			}
			if !r.readFull(r.decoded[:n2], false) {
				return r.err
			}
			if int64(n2) < n {
				if crc(r.decoded[:n2]) != checksum {
					r.err = ErrCorrupt
					return r.err
				}
			}
			r.i, r.j = 0, n2
			continue
		case chunkTypeEOF:
			if chunkLen != 0 {
				if chunkLen > binary.MaxVarintLen64 {
					r.err = ErrCorrupt
					return r.err
				}

				buf := r.tmp[:chunkLen]
				if !r.readFull(buf, false) {
					return r.err
				}
				if !r.ignoreStreamID {
					wantSize, n := binary.Uvarint(buf[:chunkLen])
					if n != chunkLen {
						r.err = ErrCorrupt
						return r.err
					}
					if wantSize != uint64(r.blockStart+int64(r.j)) {
						r.err = ErrCorrupt
						return r.err
					}
				}
			}
			r.wantEOF = false
			r.readHeader = false
			continue
		case ChunkTypeStreamIdentifier:
			// Section 4.1. Stream identifier (chunk type 0xff).
			if chunkLen != magicBodyLen {
				r.err = ErrCorrupt
				return r.err
			}
			if !r.readFull(r.tmp[:magicBodyLen], false) {
				return r.err
			}
			r.blockStart = 0
			r.i, r.j = 0, 0
			if string(r.tmp[:len(magicBody)]) == magicBody {
				if !r.minLzHeader(r.tmp[:magicBodyLen]) {
					return r.err
				}
				continue
			}
			if !r.allowFallback {
				r.err = ErrUnsupported
				return r.err
			}
			r.maxBlock = r.maxBlockOrg
			if string(r.tmp[:magicBodyLen]) != magicBodyS2 && string(r.tmp[:magicBodyLen]) != magicBodySnappy {
				r.err = ErrUnsupported
				return r.err
			}
			r.snappyFrame = string(r.tmp[:magicBodyLen]) == magicBodySnappy

			continue
		}

		if chunkType <= maxNonSkippableChunk {
			// Section 4.5. Reserved unskippable chunks (chunk types 0x02-0x7f).
			r.err = ErrUnsupported
			return r.err
		}
		// Section 4.4 Padding (chunk type 0xfe).
		// Section 4.6. Reserved skippable chunks (chunk types 0x80-0xfd).
		if !r.skippable(r.buf, chunkLen, false, chunkType) {
			return r.err
		}
	}
	return nil
}

// ReadSeeker provides random or forward seeking in compressed content.
// See Reader.ReadSeeker
type ReadSeeker struct {
	*Reader
	seek     io.Seeker
	readAtMu sync.Mutex
}

// ReadSeeker will return an io.ReadSeeker and io.ReaderAt
// compatible version of the reader.
// The original input must support the io.Seeker interface.
// A custom index can be specified which will be used if supplied.
// When using a custom index, it will not be read from the input stream.
// The ReadAt position will affect regular reads and the current position of Seek.
// So using Read after ReadAt will continue from where the ReadAt stopped.
// No functions should be used concurrently.
// The returned ReadSeeker contains a shallow reference to the existing Reader,
// meaning changes performed to one is reflected in the other.
func (r *Reader) ReadSeeker(index []byte) (*ReadSeeker, error) {
	// Read index if provided.
	if len(index) != 0 {
		if r.index == nil {
			r.index = &Index{}
		}
		if _, err := r.index.Load(index); err != nil {
			return nil, ErrCantSeek{Reason: "loading index returned: " + err.Error()}
		}
	}

	// Check if input is seekable
	rs, ok := r.r.(io.ReadSeeker)
	if !ok {
		return nil, ErrCantSeek{Reason: "input stream isn't seekable"}
	}

	if r.index != nil {
		// Seekable and index, ok...
		return &ReadSeeker{Reader: r, seek: rs}, nil
	}

	// Load from stream.
	r.index = &Index{}

	// Read current position.
	pos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, ErrCantSeek{Reason: "seeking input returned: " + err.Error()}
	}
	err = r.index.LoadStream(rs)
	if err != nil {
		if err == ErrUnsupported {
			return nil, ErrCantSeek{Reason: "input stream does not contain an index"}
		}
		return nil, ErrCantSeek{Reason: "reading index returned: " + err.Error()}
	}

	// reset position.
	_, err = rs.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, ErrCantSeek{Reason: "seeking input returned: " + err.Error()}
	}
	return &ReadSeeker{Reader: r, seek: rs}, nil
}

// Seek allows seeking in compressed data.
func (r *ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if r.err != nil {
		if !errors.Is(r.err, io.EOF) {
			return 0, r.err
		}
		// Reset on EOF
		r.err = nil
	}

	// Calculate absolute offset.
	absOffset := offset

	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		absOffset = r.blockStart + int64(r.i) + offset
	case io.SeekEnd:
		if r.index == nil {
			return 0, ErrUnsupported
		}
		absOffset = r.index.TotalUncompressed + offset
	default:
		r.err = ErrUnsupported
		return 0, r.err
	}

	if absOffset < 0 {
		return 0, errors.New("seek before start of file")
	}

	if !r.readHeader {
		// Make sure we read the header.
		// Seek to start, since we may be at EOF.
		_, r.err = r.seek.Seek(0, io.SeekStart)
		if r.err != nil {
			return 0, r.err
		}
		_, r.err = r.Read([]byte{})
		if r.err != nil {
			return 0, r.err
		}
	}

	// If we are inside current block no need to seek.
	// This includes no offset changes.
	if absOffset >= r.blockStart && absOffset < r.blockStart+int64(r.j) {
		r.i = int(absOffset - r.blockStart)
		return r.blockStart + int64(r.i), nil
	}

	// We can seek and we have an index.
	c, u, err := r.index.Find(absOffset)
	if err != nil {
		return r.blockStart + int64(r.i), err
	}

	// Seek to next block
	_, err = r.seek.Seek(c, io.SeekStart)
	if err != nil {
		return 0, err
	}

	r.i = r.j                     // Remove rest of current block.
	r.blockStart = u - int64(r.j) // Adjust current block start for accounting.
	if u < absOffset {
		// Forward inside block
		return absOffset, r.Skip(absOffset - u)
	}
	if u > absOffset {
		return 0, fmt.Errorf("minlz seek: (internal error) u (%d) > absOffset (%d)", u, absOffset)
	}
	return absOffset, nil
}

// ReadAt reads len(p) bytes into p starting at offset off in the
// underlying input source. It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered.
//
// When ReadAt returns n < len(p), it returns a non-nil error
// explaining why more bytes were not returned. In this respect,
// ReadAt is stricter than Read.
//
// Even if ReadAt returns n < len(p), it may use all of p as scratch
// space during the call. If some data is available but not len(p) bytes,
// ReadAt blocks until either all the data is available or an error occurs.
// In this respect ReadAt is different from Read.
//
// If the n = len(p) bytes returned by ReadAt are at the end of the
// input source, ReadAt may return either err == EOF or err == nil.
//
// If ReadAt is reading from an input source with a seek offset,
// ReadAt should not affect nor be affected by the underlying
// seek offset.
//
// Clients of ReadAt can execute parallel ReadAt calls on the
// same input source. This is however not recommended.
func (r *ReadSeeker) ReadAt(p []byte, offset int64) (int, error) {
	r.readAtMu.Lock()
	defer r.readAtMu.Unlock()
	_, err := r.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	n := 0
	for n < len(p) {
		n2, err := r.Read(p[n:])
		if err != nil {
			// This will include io.EOF
			return n + n2, err
		}
		n += n2
	}
	return n, nil
}

// Index will return the index used.
func (r *ReadSeeker) Index() *Index {
	return r.index
}

// ReadByte satisfies the io.ByteReader interface.
func (r *Reader) ReadByte() (byte, error) {
	if r.err != nil {
		return 0, r.err
	}
	if r.i < r.j {
		c := r.decoded[r.i]
		r.i++
		return c, nil
	}
	var tmp [1]byte
	for i := 0; i < 10; i++ {
		n, err := r.Read(tmp[:])
		if err != nil {
			return 0, err
		}
		if n == 1 {
			return tmp[0], nil
		}
	}
	return 0, io.ErrNoProgress
}

// UserChunkCB will register a callback for chunks with the specified ID.
// ID must be a reserved user chunks ID, 0x80-0xfd (inclusive).
// For each chunk with the ID, the callback is called with the content.
// Any returned non-nil error will abort decompression.
// Only one callback per ID is supported, latest sent will be used.
// Sending a nil function will disable previous callbacks.
// You can peek the stream, triggering the callback, by doing a Read with a 0
// byte buffer.
func (r *Reader) UserChunkCB(id uint8, fn func(r io.Reader) error) error {
	if id < MinUserSkippableChunk || id > MaxUserNonSkippableChunk {
		return fmt.Errorf("ReaderUserChunkCB: Invalid id provided, must be 0x80-0xfe (inclusive)")
	}
	r.skippableCB[id-MinUserSkippableChunk] = fn
	return nil
}
