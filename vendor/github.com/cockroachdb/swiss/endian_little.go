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

// NB: this list of tags is taken from encoding/binary/native_endian_little.go
//go:build 386 || amd64 || amd64p32 || alpha || arm || arm64 || loong64 || mipsle || mips64le || mips64p32le || nios2 || ppc64le || riscv || riscv64 || sh || wasm

package swiss

import "unsafe"

const bigEndian = false

// Get returns the i-th control byte.
func (g *ctrlGroup) Get(i uint32) ctrl {
	return *(*ctrl)(unsafe.Add(unsafe.Pointer(g), i))
}

// Set sets the i-th control byte.
func (g *ctrlGroup) Set(i uint32, c ctrl) {
	*(*ctrl)(unsafe.Add(unsafe.Pointer(g), i)) = c
}
