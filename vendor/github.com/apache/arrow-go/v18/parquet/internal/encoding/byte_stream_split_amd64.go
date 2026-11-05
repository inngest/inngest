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

//go:build !noasm

package encoding

import (
	"unsafe"

	"github.com/apache/arrow-go/v18/parquet/internal/debug"
	"golang.org/x/sys/cpu"
)

func init() {
	if cpu.X86.HasAVX2 {
		decodeByteStreamSplitBatchWidth4InByteOrder = decodeByteStreamSplitBatchWidth4AVX2
		decodeByteStreamSplitBatchWidth8InByteOrder = decodeByteStreamSplitBatchWidth8AVX2
		encodeByteStreamSplitWidth4Impl = encodeByteStreamSplitWidth4AVX2
		encodeByteStreamSplitWidth8Impl = encodeByteStreamSplitWidth8AVX2
	}
}

//go:noescape
func _encodeByteStreamSplitWidth4AVX2(in, out unsafe.Pointer, nValues int)

//go:noescape
func _encodeByteStreamSplitWidth8AVX2(in, out unsafe.Pointer, nValues int)

func encodeByteStreamSplitWidth4AVX2(data, in []byte) {
	if len(in) == 0 {
		return
	}
	debug.Assert(len(data) >= len(in), "not enough space in destination buffer for encoding")
	_encodeByteStreamSplitWidth4AVX2(unsafe.Pointer(&in[0]), unsafe.Pointer(&data[0]), len(in)/4)
}

func encodeByteStreamSplitWidth8AVX2(data, in []byte) {
	if len(in) == 0 {
		return
	}
	debug.Assert(len(data) >= len(in), "not enough space in destination buffer for encoding")
	_encodeByteStreamSplitWidth8AVX2(unsafe.Pointer(&in[0]), unsafe.Pointer(&data[0]), len(in)/8)
}

//go:noescape
func _decodeByteStreamSplitWidth4AVX2(data, out unsafe.Pointer, nValues, stride int)

//go:noescape
func _decodeByteStreamSplitWidth8AVX2(data, out unsafe.Pointer, nValues, stride int)

func decodeByteStreamSplitBatchWidth4AVX2(data []byte, nValues, stride int, out []byte) {
	if nValues == 0 {
		return
	}
	const width = 4
	debug.Assert(len(out) >= nValues*width, "not enough space in output buffer for decoding")
	debug.Assert(len(data) >= 3*stride+nValues, "not enough data for decoding")
	_decodeByteStreamSplitWidth4AVX2(unsafe.Pointer(&data[0]), unsafe.Pointer(&out[0]), nValues, stride)
}

func decodeByteStreamSplitBatchWidth8AVX2(data []byte, nValues, stride int, out []byte) {
	if nValues == 0 {
		return
	}
	const width = 8
	debug.Assert(len(out) >= nValues*width, "not enough space in output buffer for decoding")
	debug.Assert(len(data) >= 7*stride+nValues, "not enough data for decoding")
	_decodeByteStreamSplitWidth8AVX2(unsafe.Pointer(&data[0]), unsafe.Pointer(&out[0]), nValues, stride)
}
