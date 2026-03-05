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

//go:build race

package race

import (
	"runtime"
	"unsafe"
)

const Enabled = true

func ReadSlice[T any](s []T) {
	if len(s) == 0 {
		return
	}
	runtime.RaceReadRange(unsafe.Pointer(&s[0]), len(s)*int(unsafe.Sizeof(s[0])))
}

func WriteSlice[T any](s []T) {
	if len(s) == 0 {
		return
	}
	runtime.RaceWriteRange(unsafe.Pointer(&s[0]), len(s)*int(unsafe.Sizeof(s[0])))
}
