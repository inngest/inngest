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

package streaming

import (
	"io"

	"github.com/apache/arrow-go/v18/arrow/memory"
)

// ValueBuffer is the byte source a streaming decoder reads; bytes from Fill stay valid
// until the next Recycle, so a decoder can alias them instead of copying.
type ValueBuffer interface {
	// Fill returns >= need contiguous bytes from the cursor, valid until the next Recycle;
	// io.ErrUnexpectedEOF if fewer remain (incl. a need past the value region).
	Fill(need int) ([]byte, error)
	// Advance marks the first n bytes from the cursor as consumed.
	Advance(n int)
	// Skip discards the next n bytes without buffering them, for the discard/seek path.
	Skip(n int) error
	// Recycle reclaims everything handed out since the last Recycle; the caller must be
	// done with the previous batch's aliases. A decoder calls it when entering Decode.
	Recycle()
	io.Closer
}

// Decoder is a decoder that can read from a ValueBuffer via SetSource.
type Decoder interface {
	SetSource(nvals int, src ValueBuffer)
}

// DefaultBufferSize is the steady-state chunk size and the clip target.
const DefaultBufferSize = 1 << 20

// streamBuffer serves the value stream from reusable chunks; aliases stay valid until Recycle.
type streamBuffer struct {
	r         io.Reader
	onClose   func() error
	mem       memory.Allocator
	chunkSize int
	cur       []byte   // chunk currently being filled
	off, n    int      // consumed cursor / filled end; cur[off:n] is buffered read-ahead
	live      [][]byte // earlier chunks still backing this Decode's aliases
	remaining int      // value-region bytes not yet consumed; bounds Fill vs corrupt lengths
}

// NewStreamBuffer returns a ValueBuffer over r, bounded to valueBytes (the value region).
func NewStreamBuffer(mem memory.Allocator, r io.Reader, valueBytes int, onClose func() error) ValueBuffer {
	size := DefaultBufferSize
	if valueBytes > 0 && valueBytes < size {
		size = valueBytes
	}
	return &streamBuffer{mem: mem, r: r, onClose: onClose, chunkSize: size, cur: mem.Allocate(size), remaining: valueBytes}
}

// ClipBatch estimates rows per Decode that keep a batch's aliased values near the buffer
// size, for a page whose value region exceeds it. Row count is a conservative proxy for
// value count; skewed sizes may overshoot. Returns nvals (no clip) when the whole region
// already fits one buffer.
func ClipBatch(valueBytes, nvals int) int64 {
	if valueBytes <= DefaultBufferSize || nvals <= 0 {
		return int64(nvals)
	}
	avg := valueBytes / nvals
	if avg < 1 {
		avg = 1
	}
	if clip := int64(DefaultBufferSize / avg); clip > 1 {
		return clip
	}
	return 1
}

func (s *streamBuffer) Advance(n int) {
	s.off += n
	s.remaining -= n
}

func (s *streamBuffer) Skip(n int) error {
	if n > s.remaining {
		return io.ErrUnexpectedEOF
	}
	s.remaining -= n
	buffered := s.n - s.off
	if buffered >= n {
		s.off += n
		return nil
	}
	n -= buffered
	// Nothing here is aliased, so drop the cursor and discard the rest from the reader.
	s.off, s.n = 0, 0
	_, err := io.CopyN(io.Discard, s.r, int64(n))
	return err
}

func (s *streamBuffer) Fill(need int) ([]byte, error) {
	if s.n-s.off >= need {
		return s.cur[s.off:s.n], nil
	}
	if need > s.remaining {
		return nil, io.ErrUnexpectedEOF
	}
	if need > len(s.cur)-s.off {
		s.rotate(need)
	}
	for s.n-s.off < need {
		// Read ahead to the end of cur, bounded by the value region.
		end := min(len(s.cur), s.off+s.remaining)
		m, err := s.r.Read(s.cur[s.n:end])
		s.n += m
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	if s.n-s.off < need {
		return nil, io.ErrUnexpectedEOF
	}
	return s.cur[s.off:s.n], nil
}

// rotate retires the full chunk (still backing aliases) and starts fresh, carrying any unconsumed tail.
func (s *streamBuffer) rotate(need int) {
	tail := s.n - s.off
	s.live = append(s.live, s.cur)
	size := s.chunkSize
	if need > size {
		size = need
	}
	nc := s.mem.Allocate(size)
	copy(nc, s.cur[s.off:s.n])
	s.cur, s.off, s.n = nc, 0, tail
}

func (s *streamBuffer) Recycle() {
	for _, b := range s.live {
		s.mem.Free(b)
	}
	s.live = s.live[:0]
	// Compact once the consumed prefix passes half the chunk to preserve read-ahead
	// room while keeping compactions and rotations rare.
	if s.off*2 <= len(s.cur) {
		return
	}
	tail := s.n - s.off
	if len(s.cur) != s.chunkSize && tail <= s.chunkSize {
		// shrink the oversized chunk back to steady state
		nc := s.mem.Allocate(s.chunkSize)
		copy(nc, s.cur[s.off:s.n])
		s.mem.Free(s.cur)
		s.cur = nc
	} else {
		// compact in place
		copy(s.cur, s.cur[s.off:s.n])
	}
	s.off, s.n = 0, tail
}

func (s *streamBuffer) Close() error {
	for _, b := range s.live {
		s.mem.Free(b)
	}
	if s.cur != nil {
		s.mem.Free(s.cur)
	}
	s.live, s.cur = nil, nil
	if s.onClose != nil {
		return s.onClose()
	}
	return nil
}
