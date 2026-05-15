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

//go:build !appengine && !noasm && gc && !purego

package minlz

import (
	"sync"

	"github.com/minio/minlz/internal/race"
)

const hasAsm = true

var encPools [7]sync.Pool

// encodeBlock encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src)) &&
//	minNonLiteralBlockSize <= len(src) && len(src) <= maxBlockSize
func encodeBlock(dst, src []byte) (d int) {
	race.ReadSlice(src)
	race.WriteSlice(dst)

	switch {
	case len(src) > 2<<20:
		const sz, pool = 131072, 0
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm(dst, src, tmp)
	case len(src) > 512<<10:
		const sz, pool = 131072, 0
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm2MB(dst, src, tmp)
	case len(src) > 64<<10:
		const sz, pool = 65536, 2
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm512K(dst, src, tmp)
	case len(src) > 16<<10:
		const sz, pool = 16384, 3
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm64K(dst, src, tmp)
	case len(src) > 4<<10:
		const sz, pool = 8192, 4
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm16K(dst, src, tmp)
	case len(src) > 1<<10:
		const sz, pool = 2048, 5
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm4K(dst, src, tmp)
	case len(src) > minNonLiteralBlockSize:
		const sz, pool = 1024, 6
		tmp, ok := encPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encPools[pool].Put(tmp)
		return encodeBlockAsm1K(dst, src, tmp)
	}
	return 0
}

var encBetterPools [6]sync.Pool

// encodeBlockBetter encodes a non-empty src to a guaranteed-large-enough dst. It
// assumes that the varint-encoded length of the decompressed bytes has already
// been written.
//
// It also assumes that:
//
//	len(dst) >= MaxEncodedLen(len(src)) &&
//	minNonLiteralBlockSize <= len(src) && len(src) <= maxBlockSize
func encodeBlockBetter(dst, src []byte) (d int) {
	race.ReadSlice(src)
	race.WriteSlice(dst)

	switch {
	case len(src) > 2<<20:
		const sz, pool = 589824, 0
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm(dst, src, tmp)
	case len(src) > 512<<10:
		const sz, pool = 589824, 0
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm2MB(dst, src, tmp)
	case len(src) > 64<<10:
		const sz, pool = 294912, 1
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm512K(dst, src, tmp)
	case len(src) > 16<<10:
		const sz, pool = 73728, 2
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm64K(dst, src, tmp)
	case len(src) > 4<<10:
		const sz, pool = 36864, 3
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm16K(dst, src, tmp)
	case len(src) > 1<<10:
		const sz, pool = 10240, 4
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm4K(dst, src, tmp)
	case len(src) > minNonLiteralBlockSize:
		const sz, pool = 4608, 5
		tmp, ok := encBetterPools[pool].Get().(*[sz]byte)
		if !ok {
			tmp = &[sz]byte{}
		}
		race.WriteSlice(tmp[:])
		defer encBetterPools[pool].Put(tmp)
		return encodeBetterBlockAsm1K(dst, src, tmp)
	}
	return 0
}
