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

import "encoding/binary"

// commonPrefixGeneric is used for architectures without a native
// implementation. It is defined here rather than common_generic.go so that the
// benchmarking code can have access to it even when there's a native
// implementation available.
func commonPrefixGeneric(a, b []byte) int {
	asUint64 := func(data []byte, i int) uint64 {
		return binary.LittleEndian.Uint64(data[i:])
	}
	var shared int
	n := min(len(a), len(b))
	for shared < n-7 && asUint64(a, shared) == asUint64(b, shared) {
		shared += 8
	}
	for shared < n && a[shared] == b[shared] {
		shared++
	}
	return shared
}
