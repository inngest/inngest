// Copyright 2023 CUE Labs AG
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

package ociregistry

import "iter"

// Seq is kept for backwards compatibility with existing implementations
//
// Deprecated: use iter.Seq2.
//
//go:fix inline
type Seq[T any] = iter.Seq2[T, error]

func All[T any](it iter.Seq2[T, error]) ([]T, error) {
	xs := []T{}
	for x, err := range it {
		if err != nil {
			return nil, err
		}
		xs = append(xs, x)
	}
	return xs, nil
}

func SliceSeq[T any](xs []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, x := range xs {
			if !yield(x, nil) {
				return
			}
		}
	}
}

// ErrorSeq returns an iterator that has no
// items and always returns the given error.
func ErrorSeq[T any](err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		yield(*new(T), err)
	}
}
