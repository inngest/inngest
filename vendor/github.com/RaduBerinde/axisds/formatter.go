// Copyright 2025 Radu Berinde.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package axisds

import "fmt"

// BoundaryFormatter is used to print boundaries.
type BoundaryFormatter[B Boundary] func(b B) string

// MakeBoundaryFormatter creates a BoundaryFormatter[B] that uses fmt.Sprint().
func MakeBoundaryFormatter[B Boundary]() BoundaryFormatter[B] {
	return func(b B) string {
		return fmt.Sprint(b)
	}
}

// IntervalFormatter is used to print intervals.
type IntervalFormatter[B Boundary] func(start, end B) string

// MakeIntervalFormatter creates an IntervalFormatter[B] which uses the given
// formatter for B.
func MakeIntervalFormatter[B Boundary](bFmt BoundaryFormatter[B]) IntervalFormatter[B] {
	return func(start, end B) string {
		return fmt.Sprintf("[%s, %s)", bFmt(start), bFmt(end))
	}
}

// MakeEndpointIntervalFormatter creates an IntervalFormatter[Endpoint[B]] which
// uses the given formatter for B.
func MakeEndpointIntervalFormatter[B Boundary](
	bFmt BoundaryFormatter[B],
) IntervalFormatter[Endpoint[B]] {
	return func(start, end Endpoint[B]) string {
		c1, c2 := '[', ')'
		if start.PlusEpsilon {
			c1 = '('
		}
		if end.PlusEpsilon {
			c2 = ']'
		}
		return fmt.Sprintf("%c%s, %s%c", c1, bFmt(start.B), bFmt(end.B), c2)
	}
}
