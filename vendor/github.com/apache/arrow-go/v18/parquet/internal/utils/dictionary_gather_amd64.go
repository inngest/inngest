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

//go:build amd64 && !noasm && !appengine
// +build amd64,!noasm,!appengine

package utils

import (
	"unsafe"

	"github.com/apache/arrow-go/v18/parquet"
	"golang.org/x/sys/cpu"
)

type dictionaryGatherFunc func(dictionary, output, indices unsafe.Pointer, length int)

const (
	dictionaryGather32MinValues = 32
	dictionaryGather64MinValues = 16
)

var (
	dictionaryGather32 dictionaryGatherFunc
	dictionaryGather64 dictionaryGatherFunc
)

func init() {
	if cpu.X86.HasAVX2 {
		dictionaryGather32 = _dictionary_gather_32_avx2
		dictionaryGather64 = _dictionary_gather_64_avx2
	}
}

// CopyDictionary reports whether a fixed-width dictionary copy was performed
// by the AVX2 implementation. The caller must validate dictionary indexes
// before calling this function.
func CopyDictionary[T parquet.ColumnTypes](out, dictionary []T, indices []IndexType) bool {
	if len(indices) == 0 || len(out) < len(indices) || len(dictionary) == 0 {
		return false
	}

	var gather dictionaryGatherFunc
	switch any(out).(type) {
	case []int32, []float32:
		if len(indices) < dictionaryGather32MinValues {
			return false
		}
		gather = dictionaryGather32
	case []int64, []float64:
		if len(indices) < dictionaryGather64MinValues {
			return false
		}
		gather = dictionaryGather64
	default:
		return false
	}
	if gather == nil {
		return false
	}

	gather(
		unsafe.Pointer(unsafe.SliceData(dictionary)),
		unsafe.Pointer(unsafe.SliceData(out)),
		unsafe.Pointer(unsafe.SliceData(indices)),
		len(indices),
	)
	return true
}
