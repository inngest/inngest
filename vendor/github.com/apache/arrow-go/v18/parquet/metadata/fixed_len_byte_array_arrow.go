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
	"github.com/apache/arrow-go/v18/internal/bitutils"
	"github.com/apache/arrow-go/v18/parquet"
)

func arrowFixedLenValueCount(values []byte, byteWidth int) int64 {
	if byteWidth <= 0 || len(values)%byteWidth != 0 {
		panic("parquet: Arrow fixed-length values are not aligned to the type length")
	}
	return int64(len(values) / byteWidth)
}

// UpdateFromArrowFixedWidth updates fixed-length byte-array statistics from
// values laid out in an Arrow value buffer.
func (s *FixedLenByteArrayStatistics) UpdateFromArrowFixedWidth(values []byte, byteWidth int, numNull int64) {
	nvalues := arrowFixedLenValueCount(values, byteWidth)
	s.IncNulls(numNull)
	s.nvalues += nvalues
	if nvalues == 0 {
		return
	}

	min, max := s.defaultMin(), s.defaultMax()
	for offset := 0; offset < len(values); offset += byteWidth {
		value := parquet.FixedLenByteArray(values[offset : offset+byteWidth])
		min = s.minval(min, value)
		max = s.maxval(max, value)
	}
	s.SetMinMax(min, max)
}

// UpdateFromArrowFixedWidthSpaced updates fixed-length byte-array statistics
// from an Arrow value buffer whose null positions are described by validBits.
func (s *FixedLenByteArrayStatistics) UpdateFromArrowFixedWidthSpaced(values []byte, byteWidth int, validBits []byte, validBitsOffset, numNull int64) {
	nvalues := arrowFixedLenValueCount(values, byteWidth)
	if validBits == nil {
		s.UpdateFromArrowFixedWidth(values, byteWidth, numNull)
		return
	}

	s.IncNulls(numNull)
	s.nvalues += nvalues - numNull
	if nvalues == 0 || nvalues == numNull {
		return
	}

	if s.bitSetReader == nil {
		s.bitSetReader = bitutils.NewSetBitRunReader(validBits, validBitsOffset, nvalues)
	} else {
		s.bitSetReader.Reset(validBits, validBitsOffset, nvalues)
	}

	min, max := s.defaultMin(), s.defaultMax()
	for {
		run := s.bitSetReader.NextRun()
		if run.Length == 0 {
			break
		}
		for pos := run.Pos; pos < run.Pos+run.Length; pos++ {
			start := int(pos) * byteWidth
			value := parquet.FixedLenByteArray(values[start : start+byteWidth])
			min = s.minval(min, value)
			max = s.maxval(max, value)
		}
	}
	s.SetMinMax(min, max)
}

// InsertArrowFixedLenHashes inserts hashes for fixed-length values laid out in
// an Arrow value buffer.
func InsertArrowFixedLenHashes(b BloomFilterBuilder, values []byte, byteWidth int) {
	if len(values) == 0 {
		return
	}
	arrowFixedLenValueCount(values, byteWidth)

	h := b.Hasher()
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)
	for offset := 0; offset < len(values); offset += bloomFilterHashBatchSize * byteWidth {
		end := min(offset+bloomFilterHashBatchSize*byteWidth, len(values))
		n := (end - offset) / byteWidth
		for i := 0; i < n; i++ {
			start := offset + i*byteWidth
			byteBatch[i] = values[start : start+byteWidth]
		}
		b.InsertBulk(sum64s(h, byteBatch[:n], hashBatch[:n]))
	}
}

// InsertSpacedArrowFixedLenHashes inserts hashes for valid fixed-length values
// from an Arrow value buffer.
func InsertSpacedArrowFixedLenHashes(b BloomFilterBuilder, numValid int64, values []byte, byteWidth int, validBits []byte, validBitsOffset int64) {
	if numValid == 0 {
		return
	}
	if validBits == nil {
		InsertArrowFixedLenHashes(b, values, byteWidth)
		return
	}

	nvalues := arrowFixedLenValueCount(values, byteWidth)
	h := b.Hasher()
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		hashBatch [bloomFilterHashBatchSize]uint64
	)
	setReader := bitutils.NewSetBitRunReader(validBits, validBitsOffset, nvalues)
	for {
		run := setReader.NextRun()
		if run.Length == 0 {
			break
		}
		for pos := run.Pos; pos < run.Pos+run.Length; pos += bloomFilterHashBatchSize {
			end := min(pos+int64(bloomFilterHashBatchSize), run.Pos+run.Length)
			n := int(end - pos)
			for i := 0; i < n; i++ {
				start := int(pos+int64(i)) * byteWidth
				byteBatch[i] = values[start : start+byteWidth]
			}
			b.InsertBulk(sum64s(h, byteBatch[:n], hashBatch[:n]))
		}
	}
}
