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

import "github.com/apache/arrow-go/v18/internal/bitutils"

type byteArrayArrowHashOffset interface {
	~int32 | ~int64
}

func insertArrowOffsetHashes[T byteArrayArrowHashOffset](b BloomFilterBuilder, values []byte, offsets []T, numValid int64, validBits []byte, validBitsOffset int64) {
	if len(offsets) < 2 || numValid == 0 {
		return
	}

	h := b.Hasher()
	var (
		byteBatch [bloomFilterHashBatchSize][]byte
		hashBatch [bloomFilterHashBatchSize]uint64
		batchSize int
	)
	flush := func() {
		b.InsertBulk(sum64s(h, byteBatch[:batchSize], hashBatch[:batchSize]))
		batchSize = 0
	}
	if validBits == nil {
		for i := 0; i < len(offsets)-1; i++ {
			byteBatch[batchSize] = values[int(offsets[i]):int(offsets[i+1])]
			batchSize++
			if batchSize == len(byteBatch) {
				flush()
			}
		}
		if batchSize != 0 {
			flush()
		}
		return
	}

	visit := func(pos, length int64) {
		for i := pos; i < pos+length; i++ {
			byteBatch[batchSize] = values[int(offsets[i]):int(offsets[i+1])]
			batchSize++
			if batchSize == len(byteBatch) {
				flush()
			}
		}
	}

	bitutils.VisitSetBitRunsNoErr(validBits, validBitsOffset, int64(len(offsets)-1), visit)
	if batchSize != 0 {
		flush()
	}
}

// InsertArrowOffsetHashes inserts hashes for values described by 32-bit Arrow
// offsets without materializing parquet.ByteArray values.
func InsertArrowOffsetHashes(b BloomFilterBuilder, values []byte, offsets []int32) {
	numValues := 0
	if len(offsets) > 0 {
		numValues = len(offsets) - 1
	}
	insertArrowOffsetHashes(b, values, offsets, int64(numValues), nil, 0)
}

// InsertArrowOffsetHashes64 inserts hashes for values described by 64-bit Arrow
// offsets without materializing parquet.ByteArray values.
func InsertArrowOffsetHashes64(b BloomFilterBuilder, values []byte, offsets []int64) {
	numValues := 0
	if len(offsets) > 0 {
		numValues = len(offsets) - 1
	}
	insertArrowOffsetHashes(b, values, offsets, int64(numValues), nil, 0)
}

// InsertSpacedArrowOffsetHashes inserts hashes for valid values described by
// spaced 32-bit Arrow offsets without materializing parquet.ByteArray values.
func InsertSpacedArrowOffsetHashes(b BloomFilterBuilder, numValid int64, values []byte, offsets []int32, validBits []byte, validBitsOffset int64) {
	insertArrowOffsetHashes(b, values, offsets, numValid, validBits, validBitsOffset)
}

// InsertSpacedArrowOffsetHashes64 inserts hashes for valid values described by
// spaced 64-bit Arrow offsets without materializing parquet.ByteArray values.
func InsertSpacedArrowOffsetHashes64(b BloomFilterBuilder, numValid int64, values []byte, offsets []int64, validBits []byte, validBitsOffset int64) {
	insertArrowOffsetHashes(b, values, offsets, numValid, validBits, validBitsOffset)
}
