// Copyright 2024 The Cockroach Authors
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

// NB: this list of tags is taken from encoding/binary/native_endian_big.go
//go:build armbe || arm64be || m68k || mips || mips64 || mips64p32 || ppc || ppc64 || s390 || s390x || shbe || sparc || sparc64

package swiss

import "unsafe"

const bigEndian = true

// Get returns the i-th control byte.
func (g *ctrlGroup) Get(i uint32) ctrl {
	i = i ^ (groupSize - 1) // equivalent to (groupSize-1-i).
	return *(*ctrl)(unsafe.Add(unsafe.Pointer(g), i))
}

// Set sets the i-th control byte.
func (g *ctrlGroup) Set(i uint32, c ctrl) {
	i = i ^ (groupSize - 1) // equivalent to (groupSize-1-i).
	*(*ctrl)(unsafe.Add(unsafe.Pointer(g), i)) = c
}
