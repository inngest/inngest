// Copyright 2024 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package crbytes

import (
	"fmt"
	"unsafe"
)

// AllocAligned allocates a new byte slice of length n, ensuring the address of
// the beginning of the slice is word aligned. Go does not guarantee that a
// simple make([]byte, n) is aligned.
func AllocAligned(n int) []byte {
	if n == 0 {
		return nil
	}
	a := make([]uint64, (n+7)/8)
	b := unsafe.Slice((*byte)(unsafe.Pointer(&a[0])), n)

	// Verify alignment.
	ptr := uintptr(unsafe.Pointer(&b[0]))
	if ptr&7 != 0 {
		panic(fmt.Sprintf("allocated []uint64 slice not 8-aligned: pointer %p", &b[0]))
	}
	return b
}

// CopyAligned copies the provided byte slice into an aligned byte slice of the
// same length.
func CopyAligned(s []byte) []byte {
	dst := AllocAligned(len(s))
	copy(dst, s)
	return dst
}
