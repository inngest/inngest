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

// Boundary is the most basic unit used by this library. It represents a
// boundary on a 1D axis.
//
// The data structures in this package operate on half-open intervals
// like [startBoundary, endBoundary).
//
// Endpoint is a Boundary wrapper that creates more fine-grained intervals,
// allowing arbitrary types of intervals (in terms of inclusive vs exclusive
// endpoints).
//
// NOTE: it's equivalent to think of Boundaries as being infinitesimal and not
// corresponding to any valid value. In this case, there is no concept of
// inclusive or exclusive intervals. Endpoint can represent infinitesimal
// boundaries that are immediately before or after a value, allowing arbitrary
// types of intervals (with respect to valid values).
type Boundary any

// CompareFn is a function that compares two boundaries and returns -1, 0, or +1.
type CompareFn[B Boundary] func(x, y B) int

// Endpoint is a Boundary that extends a simpler boundary type to allow
// representing intervals with inclusive or exclusive end points.
type Endpoint[B Boundary] struct {
	B B
	// If PlusEpsilon is true, the boundary is considered to be infinitesimally
	// after B. When used as an interval ending point, it corresponds to an
	// inclusive end bound. When used as an interval starting point, it
	// corresponds to an exclusive start bound.
	PlusEpsilon bool
}

// InclusiveOrExclusive is used to specify the type of interval endpoint.
type InclusiveOrExclusive int8

const Inclusive InclusiveOrExclusive = 1
const Exclusive InclusiveOrExclusive = 2

// InclusiveIf returns Inclusive if the argument is true and Exclusive
// otherwise.
func InclusiveIf(inclusive bool) InclusiveOrExclusive {
	if inclusive {
		return Inclusive
	}
	return Exclusive
}

func MakeStartEndpoint[B Boundary](startBoundary B, startTyp InclusiveOrExclusive) Endpoint[B] {
	return Endpoint[B]{
		B:           startBoundary,
		PlusEpsilon: startTyp == Exclusive,
	}
}

func MakeEndEndpoint[B Boundary](endBoundary B, endTyp InclusiveOrExclusive) Endpoint[B] {
	return Endpoint[B]{
		B:           endBoundary,
		PlusEpsilon: endTyp == Inclusive,
	}
}

func MakeEndpoints[B Boundary](
	startBoundary B, startTyp InclusiveOrExclusive, endBoundary B, endTyp InclusiveOrExclusive,
) (start, end Endpoint[B]) {
	return MakeStartEndpoint(startBoundary, startTyp), MakeEndEndpoint(endBoundary, endTyp)
}

// EndpointCompareFn returns a CompareFn for Endpoint[B].
func EndpointCompareFn[B Boundary](bCmp CompareFn[B]) CompareFn[Endpoint[B]] {
	return func(x, y Endpoint[B]) int {
		if c := bCmp(x.B, y.B); c != 0 {
			return c
		}
		switch {
		case x.PlusEpsilon == y.PlusEpsilon:
			return 0
		case x.PlusEpsilon:
			return +1
		default:
			return -1
		}
	}
}
