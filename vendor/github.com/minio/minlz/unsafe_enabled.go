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

// We enable 64 bit LE platforms:

//go:build (amd64 || arm64 || ppc64le || riscv64) && !nounsafe && !purego && !appengine

package minlz

import (
	"unsafe"
)

func load8(b []byte, i int) byte {
	return *(*byte)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

func load16(b []byte, i int) uint16 {
	//return binary.LittleEndian.Uint16(b[i:])
	//return *(*uint16)(unsafe.Pointer(&b[i]))
	return *(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

func load32(b []byte, i int) uint32 {
	//return binary.LittleEndian.Uint32(b[i:])
	//return *(*uint32)(unsafe.Pointer(&b[i]))
	return *(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

func load64(b []byte, i int) uint64 {
	//return binary.LittleEndian.Uint64(b[i:])
	//return *(*uint64)(unsafe.Pointer(&b[i]))
	return *(*uint64)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), i))
}

func store8(b []byte, idx int, v uint8) {
	*(*uint8)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), idx)) = v
}

func store16(b []byte, idx int, v uint16) {
	//binary.LittleEndian.PutUint16(b, v)
	*(*uint16)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), idx)) = v
}

func store32(b []byte, idx int, v uint32) {
	//binary.LittleEndian.PutUint32(b, v)
	*(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(b)), idx)) = v
}
