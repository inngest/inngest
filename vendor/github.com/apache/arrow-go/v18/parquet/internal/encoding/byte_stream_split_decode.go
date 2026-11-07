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

package encoding

import (
	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/internal/debug"
)

var (
	decodeByteStreamSplitBatchWidth4InByteOrder func(data []byte, nValues, stride int, out []byte) = decodeByteStreamSplitBatchWidth4InByteOrderDefault
	decodeByteStreamSplitBatchWidth8InByteOrder func(data []byte, nValues, stride int, out []byte) = decodeByteStreamSplitBatchWidth8InByteOrderDefault
)

// decodeByteStreamSplitBatchFLBA decodes the batch of nValues FixedLenByteArrays provided by 'data',
// into the output slice 'out' using BYTE_STREAM_SPLIT encoding.
// 'out' must have space for at least nValues slices.
func decodeByteStreamSplitBatchFLBA(data []byte, nValues, stride, width int, out []parquet.FixedLenByteArray) {
	debug.Assert(len(out) >= nValues, "not enough space in output slice for decoding")
	debug.Assert(len(data) >= (width-1)*stride+nValues, "not enough data for decoding")
	for stream := 0; stream < width; stream++ {
		for element := 0; element < nValues; element++ {
			encLoc := stride*stream + element
			out[element][stream] = data[encLoc]
		}
	}
}
