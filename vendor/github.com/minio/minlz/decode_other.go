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

//go:build !amd64 || appengine || !gc || noasm || purego

package minlz

// minLZDecode writes the decoding of src to dst. It assumes that the varint-encoded
// length of the decompressed bytes has already been read, and that len(dst)
// equals that length.
//
// It returns 0 on success or a decodeErrCodeXxx error code on failure.
func minLZDecode(dst, src []byte) int {
	return minLZDecodeGo(dst, src)
}
