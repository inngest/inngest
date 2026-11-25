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
// +build !noasm

package utils

import (
	"bytes"
	"io"
	"sync"
	"unsafe"
)

var bufferPool = sync.Pool{New: func() interface{} { return &bytes.Buffer{} }}

//go:noescape
func _unpack32_neon(in, out unsafe.Pointer, batchSize, nbits int) (num int)

func unpack32NEON(in io.Reader, out []uint32, nbits int) (int, error) {
	batch := len(out) / 32 * 32
	n := batch * nbits / 8
	if n <= 0 {
		return 0, nil
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()
	buffer.Grow(n)
	packed := buffer.AvailableBuffer()[:n]
	nread, err := io.ReadFull(in, packed)
	if err == io.ErrUnexpectedEOF && nread%(nbits*4) == 0 {
		err = io.EOF
	}
	completeBatch := nread * 8 / nbits / 32 * 32
	if completeBatch == 0 {
		return 0, err
	}

	var (
		input  = unsafe.Pointer(&packed[0])
		output = unsafe.Pointer(&out[0])
	)

	return _unpack32_neon(input, output, completeBatch, nbits), err
}
