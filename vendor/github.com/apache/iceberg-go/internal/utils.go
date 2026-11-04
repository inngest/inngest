// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package internal

import (
	"cmp"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"iter"
	"slices"
	"sync/atomic"

	"golang.org/x/exp/constraints"
)

// EncodeDefaultBytes returns the Iceberg JSON single-value representation for
// binary and fixed defaults.
func EncodeDefaultBytes(value []byte) string {
	return hex.EncodeToString(value)
}

// DecodeDefaultBytes decodes the Iceberg hex representation for binary and
// fixed defaults. Base64 is accepted only for compatibility with metadata
// written by iceberg-go v0.6.0. Hex wins when both encodings are valid.
// fixedLen is -1 for binary and the required width for fixed values.
func DecodeDefaultBytes(value string, fixedLen int) ([]byte, error) {
	hexValue, hexErr := hex.DecodeString(value)
	if hexErr == nil && (fixedLen < 0 || len(hexValue) == fixedLen) {
		return hexValue, nil
	}

	base64Value, base64Err := base64.StdEncoding.DecodeString(value)
	if base64Err == nil && (fixedLen < 0 || len(base64Value) == fixedLen) {
		return base64Value, nil
	}

	if hexErr == nil {
		return nil, fmt.Errorf("hex value has %d bytes, expected %d", len(hexValue), fixedLen)
	}
	if base64Err == nil {
		return nil, fmt.Errorf("legacy base64 value has %d bytes, expected %d", len(base64Value), fixedLen)
	}

	return nil, fmt.Errorf("invalid hex or legacy base64 value: hex: %v; base64: %v", hexErr, base64Err)
}

// SchemaRef marks schema access that may return references to internal state.
// It is restricted to packages within this module by Go's internal package rules.
type SchemaRef struct{}

// PartitionSpecRef marks partition-spec access that may return references to
// internal state. It is restricted to packages within this module by Go's
// internal package rules.
type PartitionSpecRef struct{}

// FloorDiv performs floored integer division, rounding toward negative infinity.
// This matches Java's Math.floorDiv behavior for negative dividends.
func FloorDiv[T constraints.Integer](a, b T) T {
	d := a / b
	if (a^b) < 0 && d*b != a {
		d--
	}

	return d
}

// Helper function to find the difference between two slices (a - b).
func Difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	diff := make([]string, 0)
	for _, item := range a {
		if !m[item] {
			diff = append(diff, item)
		}
	}

	return diff
}

type Bin[T any] struct {
	binWeight    int64
	targetWeight int64
	items        []T
}

func (b *Bin[T]) Weight() int64            { return b.binWeight }
func (b *Bin[T]) CanAdd(weight int64) bool { return b.binWeight+weight <= b.targetWeight }
func (b *Bin[T]) Add(item T, weight int64) {
	b.binWeight += weight
	b.items = append(b.items, item)
}

func PackingIterator[T any](itr iter.Seq[T], targetWeight int64, lookback int, weightFunc func(T) int64, largestBinFirst bool) iter.Seq[[]T] {
	bins := make([]Bin[T], 0)
	findBin := func(weight int64) *Bin[T] {
		for i := range bins {
			if bins[i].CanAdd(weight) {
				return &bins[i]
			}
		}

		return nil
	}

	removeBin := func() Bin[T] {
		if largestBinFirst {
			maxBin := slices.MaxFunc(bins, func(a, b Bin[T]) int {
				return cmp.Compare(a.Weight(), b.Weight())
			})
			i := slices.IndexFunc(bins, func(e Bin[T]) bool {
				return e.Weight() == maxBin.Weight()
			})

			bins = slices.Delete(bins, i, i+1)

			return maxBin
		}

		out := bins[0]
		if lookback == 1 {
			// Keep the two-bin buffer for the next FIFO eviction.
			if len(bins) == 2 {
				bins[0] = bins[1]
				bins[1] = Bin[T]{}
			} else {
				bins[0] = Bin[T]{}
			}
			bins = bins[:len(bins)-1]
		} else {
			bins = bins[1:]
		}

		return out
	}

	return func(yield func([]T) bool) {
		for item := range itr {
			w := weightFunc(item)
			bin := findBin(w)
			if bin != nil {
				bin.Add(item, w)
			} else {
				bin := Bin[T]{targetWeight: targetWeight}
				bin.Add(item, w)
				bins = append(bins, bin)

				if len(bins) > lookback {
					if !yield(removeBin().items) {
						return
					}
				}
			}
		}

		for len(bins) > 0 {
			if !yield(removeBin().items) {
				return
			}
		}
	}
}

type SlicePacker[T any] struct {
	TargetWeight    int64
	Lookback        int
	LargestBinFirst bool
}

func (s *SlicePacker[T]) Pack(items []T, weightFunc func(T) int64) [][]T {
	return slices.Collect(PackingIterator(slices.Values(items), s.TargetWeight,
		s.Lookback, weightFunc, s.LargestBinFirst))
}

func (s *SlicePacker[T]) PackEnd(items []T, weightFunc func(T) int64) [][]T {
	packed := slices.Collect(PackingIterator(func(yield func(T) bool) {
		for i := len(items); i > 0; i-- {
			if !yield(items[i-1]) {
				return
			}
		}
	}, s.TargetWeight, s.Lookback, weightFunc, s.LargestBinFirst))
	slices.Reverse(packed)

	for _, items := range packed {
		slices.Reverse(items)
	}

	if packed == nil {
		return make([][]T, 0)
	}

	return packed
}

type CountingWriter struct {
	Count int64
	W     io.Writer
}

func (w *CountingWriter) Write(p []byte) (int, error) {
	n, err := w.W.Write(p)
	w.Count += int64(n)

	return n, err
}

func RecoverError(err *error) {
	if r := recover(); r != nil {
		switch e := r.(type) {
		case string:
			*err = fmt.Errorf("error encountered during arrow schema visitor: %s", e)
		case error:
			*err = fmt.Errorf("error encountered during arrow schema visitor: %w", e)
		}
	}
}

func SingleErrorIter[T any](err error) iter.Seq2[T, error] {
	var z T

	return func(yield func(T, error) bool) {
		_ = yield(z, err)
	}
}

func Counter(start int) iter.Seq[int] {
	var current atomic.Int64
	current.Store(int64(start) - 1)

	return func(yield func(int) bool) {
		for {
			if !yield(int(current.Add(1))) {
				return
			}
		}
	}
}

// CheckedClose is a helper function to close a resource and return an error if it fails.
// It is intended to be used in a defer statement.
func CheckedClose(c io.Closer, err *error) {
	*err = errors.Join(*err, c.Close())
}

// SliceEqualHelper compares the equality of two slices whose elements have an Equals method
func SliceEqualHelper[T interface{ Equals(T) bool }](s1, s2 []T) bool {
	return slices.EqualFunc(s1, s2, func(t1, t2 T) bool {
		return t1.Equals(t2)
	})
}
