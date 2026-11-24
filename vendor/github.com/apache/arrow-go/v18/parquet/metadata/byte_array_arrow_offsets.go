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

type byteArrayArrowOffset interface {
	~int32 | ~int64
}

func updateByteArrayStatisticsFromArrowOffsets[T byteArrayArrowOffset](s *ByteArrayStatistics, values []byte, offsets []T, numNull int64, validBits []byte, validBitsOffset int64, spaced bool) {
	numValues := 0
	if len(offsets) > 0 {
		numValues = len(offsets) - 1
	}

	s.IncNulls(numNull)
	if spaced {
		s.nvalues += int64(numValues) - numNull
	} else {
		s.nvalues += int64(numValues)
	}
	if numValues == 0 {
		return
	}

	min := s.defaultMin()
	max := s.defaultMax()
	if validBits == nil {
		for i := 0; i < numValues; i++ {
			value := parquet.ByteArray(values[int(offsets[i]):int(offsets[i+1])])
			min = s.minval(min, value)
			max = s.maxval(max, value)
		}
		s.SetMinMax(min, max)
		return
	}

	visit := func(pos, length int64) {
		for i := pos; i < pos+length; i++ {
			value := parquet.ByteArray(values[int(offsets[i]):int(offsets[i+1])])
			min = s.minval(min, value)
			max = s.maxval(max, value)
		}
	}
	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(numValues), visit)
	s.SetMinMax(min, max)
}

// UpdateFromArrowOffsets updates byte-array statistics from an Arrow value buffer
// and 32-bit offsets without materializing parquet.ByteArray values.
func (s *ByteArrayStatistics) UpdateFromArrowOffsets(values []byte, offsets []int32, numNull int64) {
	updateByteArrayStatisticsFromArrowOffsets(s, values, offsets, numNull, nil, 0, false)
}

// UpdateFromArrowOffsets64 updates byte-array statistics from an Arrow value buffer
// and 64-bit offsets without materializing parquet.ByteArray values.
func (s *ByteArrayStatistics) UpdateFromArrowOffsets64(values []byte, offsets []int64, numNull int64) {
	updateByteArrayStatisticsFromArrowOffsets(s, values, offsets, numNull, nil, 0, false)
}

// UpdateFromArrowOffsetsSpaced updates byte-array statistics from spaced Arrow
// values using a validity bitmap and 32-bit offsets.
func (s *ByteArrayStatistics) UpdateFromArrowOffsetsSpaced(values []byte, offsets []int32, validBits []byte, validBitsOffset, numNull int64) {
	updateByteArrayStatisticsFromArrowOffsets(s, values, offsets, numNull, validBits, validBitsOffset, true)
}

// UpdateFromArrowOffsetsSpaced64 updates byte-array statistics from spaced Arrow
// values using a validity bitmap and 64-bit offsets.
func (s *ByteArrayStatistics) UpdateFromArrowOffsetsSpaced64(values []byte, offsets []int64, validBits []byte, validBitsOffset, numNull int64) {
	updateByteArrayStatisticsFromArrowOffsets(s, values, offsets, numNull, validBits, validBitsOffset, true)
}
