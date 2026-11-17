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

package iceberg

import (
	"fmt"
	"hash/maphash"
	"maps"
	"runtime/debug"
)

const (
	icebergModulePath = "github.com/apache/iceberg-go"
	unknownVersion    = "(unknown version)"
)

var version = detectVersion()

func detectVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unknownVersion
	}

	return versionFromBuildInfo(info)
}

func versionFromBuildInfo(info *debug.BuildInfo) string {
	if info.Main.Path == icebergModulePath {
		return moduleVersion(&info.Main)
	}

	for _, dep := range info.Deps {
		if dep != nil && dep.Path == icebergModulePath {
			return moduleVersion(dep)
		}
	}

	return unknownVersion
}

func moduleVersion(module *debug.Module) string {
	if module.Replace != nil {
		module = module.Replace
	}
	if module.Version == "" || module.Version == "(devel)" {
		return unknownVersion
	}

	return module.Version
}

func Version() string { return version }

func fullVersion() string { return "Apache Iceberg Go " + Version() }

// Optional represents a typed value that could be null
type Optional[T any] struct {
	Val   T
	Valid bool
}

// StructLike represents a single row in a record.
type StructLike interface {
	// Size returns the number of columns in this row
	Size() int
	// Get returns the value in the requested column,
	// will panic if pos is out of bounds.
	Get(pos int) any
	// Set changes the value in the column indicated,
	// will panic if pos is out of bounds.
	Set(pos int, val any)
}

type accessor struct {
	pos   int
	inner *accessor
}

func (a *accessor) String() string {
	return fmt.Sprintf("Accessor(position=%d, inner=%s)", a.pos, a.inner)
}

func (a *accessor) Get(s StructLike) any {
	val, inner := s.Get(a.pos), a
	for val != nil && inner.inner != nil {
		inner = inner.inner
		val = val.(StructLike).Get(inner.pos)
	}

	return val
}

type Set[E any] interface {
	Add(...E)
	Contains(E) bool
	Members() []E
	Equals(Set[E]) bool
	Len() int
	All(func(E) bool) bool
}

var lzseed = maphash.MakeSeed()

type literalSet map[any]struct{ orig Literal }

func newLiteralSet(vals ...Literal) Set[Literal] {
	s := make(literalSet, len(vals))
	for _, v := range vals {
		s.addliteral(v)
	}

	return s
}

func (l literalSet) addliteral(v Literal) {
	switch v := v.(type) {
	case FixedLiteral:
		l[maphash.Bytes(lzseed, []byte(v))] = struct{ orig Literal }{v}
	case BinaryLiteral:
		l[maphash.Bytes(lzseed, []byte(v))] = struct{ orig Literal }{v}
	case GeoLiteral:
		// GeoLiteral holds a []byte, so it cannot be used as a map key directly.
		l[maphash.Bytes(lzseed, v.Value())] = struct{ orig Literal }{v}
	default:
		l[v] = struct{ orig Literal }{}
	}
}

func (l literalSet) Add(lits ...Literal) {
	for _, v := range lits {
		l.addliteral(v)
	}
}

func (l literalSet) Contains(lit Literal) bool {
	switch lit := lit.(type) {
	case BinaryLiteral:
		v, ok := l[maphash.Bytes(lzseed, []byte(lit))]
		if !ok {
			return false
		}

		return lit.Equals(v.orig)
	case FixedLiteral:
		v, ok := l[maphash.Bytes(lzseed, []byte(lit))]
		if !ok {
			return false
		}

		return lit.Equals(v.orig)
	case GeoLiteral:
		v, ok := l[maphash.Bytes(lzseed, lit.Value())]
		if !ok {
			return false
		}

		return lit.Equals(v.orig)
	default:
		_, ok := l[lit]

		return ok
	}
}

func (l literalSet) Members() []Literal {
	result := make([]Literal, 0, len(l))
	for k, v := range l {
		if k, ok := k.(Literal); ok {
			result = append(result, k)
		} else {
			result = append(result, v.orig)
		}
	}

	return result
}

func (l literalSet) Equals(other Set[Literal]) bool {
	rhs, ok := other.(literalSet)
	if !ok {
		return false
	}

	return maps.EqualFunc(l, rhs, func(v1, v2 struct{ orig Literal }) bool {
		switch {
		case v1.orig == nil:
			return v2.orig == nil
		case v2.orig == nil:
			return v1.orig == nil
		default:
			return v1.orig.Equals(v2.orig)
		}
	})
}

func (l literalSet) Len() int { return len(l) }

func (l literalSet) All(fn func(Literal) bool) bool {
	for k, v := range l {
		var e Literal
		if k, ok := k.(Literal); ok {
			e = k
		} else {
			e = v.orig
		}

		if !fn(e) {
			return false
		}
	}

	return true
}
