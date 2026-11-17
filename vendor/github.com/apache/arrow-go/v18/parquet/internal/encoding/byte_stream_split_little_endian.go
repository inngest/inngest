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

//go:build 386 || amd64 || amd64p32 || alpha || arm || arm64 || loong64 || mipsle || mips64le || mips64p32le || nios2 || ppc64le || riscv || riscv64 || sh || wasm

package encoding

import (
	"unsafe"

	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/debug"
)

// decodeByteStreamSplitBatchWidth4InByteOrder decodes the batch of nValues raw bytes representing a 4-byte datatype provided
// by 'data', into the output buffer 'out' using BYTE_STREAM_SPLIT encoding. The values are expected to be in little-endian
// byte order and are be decoded into the 'out' array in machine's native endianness.
// 'out' must have space for at least len(data) bytes.
func decodeByteStreamSplitBatchWidth4InByteOrderDefault(data []byte, nValues, stride int, out []byte) {
	const width = 4
	debug.Assert(len(out) >= nValues*width, "not enough space in output buffer for decoding")
	// the beginning of the data slice can be truncated, but for valid encoding we need at least (width-1)*stride+nValues bytes
	debug.Assert(len(data) >= 3*stride+nValues, "not enough data for decoding")
	s0 := data[:nValues]
	s1 := data[stride : stride+nValues]
	s2 := data[2*stride : 2*stride+nValues]
	s3 := data[3*stride : 3*stride+nValues]
	out = out[:width*nValues]
	out32 := unsafe.Slice((*uint32)(unsafe.Pointer(&out[0])), nValues)
	for i := range nValues {
		out32[i] = uint32(s0[i]) | uint32(s1[i])<<8 | uint32(s2[i])<<16 | uint32(s3[i])<<24
	}
}

// decodeByteStreamSplitBatchWidth8InByteOrder decodes the batch of nValues raw bytes representing a 8-byte datatype provided
// by 'data', into the output buffer 'out' using BYTE_STREAM_SPLIT encoding. The values are expected to be in little-endian
// byte order and are be decoded into the 'out' array in machine's native endianness.
// 'out' must have space for at least len(data) bytes.
func decodeByteStreamSplitBatchWidth8InByteOrderDefault(data []byte, nValues, stride int, out []byte) {
	const width = 8
	debug.Assert(len(out) >= nValues*width, "not enough space in output buffer for decoding")
	debug.Assert(len(data) >= 7*stride+nValues, "not enough data for decoding")
	s0 := data[:nValues]
	s1 := data[stride : stride+nValues]
	s2 := data[2*stride : 2*stride+nValues]
	s3 := data[3*stride : 3*stride+nValues]
	s4 := data[4*stride : 4*stride+nValues]
	s5 := data[5*stride : 5*stride+nValues]
	s6 := data[6*stride : 6*stride+nValues]
	s7 := data[7*stride : 7*stride+nValues]
	out = out[:width*nValues]
	out64 := unsafe.Slice((*uint64)(unsafe.Pointer(&out[0])), nValues)
	for i := range nValues {
		out64[i] = uint64(s0[i]) | uint64(s1[i])<<8 | uint64(s2[i])<<16 | uint64(s3[i])<<24 |
			uint64(s4[i])<<32 | uint64(s5[i])<<40 | uint64(s6[i])<<48 | uint64(s7[i])<<56
	}
}

// decodeByteStreamSplitBatchFLBAWidth2 decodes the batch of nValues FixedLenByteArrays of length 2 provided by 'data',
// into the output slice 'out' using BYTE_STREAM_SPLIT encoding.
// 'out' must have space for at least nValues slices.
func decodeByteStreamSplitBatchFLBAWidth2(data []byte, nValues, stride int, out []parquet.FixedLenByteArray) {
	debug.Assert(len(out) >= nValues, "not enough space in output slice for decoding")
	debug.Assert(len(data) >= stride+nValues, "not enough data for decoding")
	s0 := data[:nValues]
	s1 := data[stride : stride+nValues]
	for i := range nValues {
		out16 := (*uint16)(unsafe.Pointer(&out[i][0]))
		*out16 = uint16(s0[i]) | uint16(s1[i])<<8
	}
}

// decodeByteStreamSplitBatchFLBAWidth4 decodes the batch of nValues FixedLenByteArrays of length 4 provided by 'data',
// into the output slice 'out' using BYTE_STREAM_SPLIT encoding.
// 'out' must have space for at least nValues slices.
func decodeByteStreamSplitBatchFLBAWidth4(data []byte, nValues, stride int, out []parquet.FixedLenByteArray) {
	debug.Assert(len(out) >= nValues, "not enough space in output slice for decoding")
	debug.Assert(len(data) >= 3*stride+nValues, "not enough data for decoding")
	s0 := data[:nValues]
	s1 := data[stride : stride+nValues]
	s2 := data[stride*2 : stride*2+nValues]
	s3 := data[stride*3 : stride*3+nValues]
	for i := range nValues {
		out32 := (*uint32)(unsafe.Pointer(&out[i][0]))
		*out32 = uint32(s0[i]) | uint32(s1[i])<<8 | uint32(s2[i])<<16 | uint32(s3[i])<<24
	}
}

// decodeByteStreamSplitBatchFLBAWidth8 decodes the batch of nValues FixedLenByteArrays of length 8 provided by 'data',
// into the output slice 'out' using BYTE_STREAM_SPLIT encoding.
// 'out' must have space for at least nValues slices.
func decodeByteStreamSplitBatchFLBAWidth8(data []byte, nValues, stride int, out []parquet.FixedLenByteArray) {
	debug.Assert(len(out) >= nValues, "not enough space in output slice for decoding")
	debug.Assert(len(data) >= 7*stride+nValues, "not enough data for decoding")
	s0 := data[:nValues]
	s1 := data[stride : stride+nValues]
	s2 := data[stride*2 : stride*2+nValues]
	s3 := data[stride*3 : stride*3+nValues]
	s4 := data[stride*4 : stride*4+nValues]
	s5 := data[stride*5 : stride*5+nValues]
	s6 := data[stride*6 : stride*6+nValues]
	s7 := data[stride*7 : stride*7+nValues]
	for i := range nValues {
		out64 := (*uint64)(unsafe.Pointer(&out[i][0]))
		*out64 = uint64(s0[i]) | uint64(s1[i])<<8 | uint64(s2[i])<<16 | uint64(s3[i])<<24 |
			uint64(s4[i])<<32 | uint64(s5[i])<<40 | uint64(s6[i])<<48 | uint64(s7[i])<<56
	}
}
