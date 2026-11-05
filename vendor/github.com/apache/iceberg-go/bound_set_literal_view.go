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
	"slices"
)

func cloneBoundLiteral(lit Literal) Literal {
	switch lit := lit.(type) {
	case BinaryLiteral:
		return BinaryLiteral(slices.Clone(lit))
	case FixedLiteral:
		return FixedLiteral(slices.Clone(lit))
	case GeoLiteral:
		lit.val = slices.Clone(lit.val)

		return lit
	default:
		return lit
	}
}

func cloneBoundLiteralSet(lits Set[Literal]) Set[Literal] {
	cloned := newLiteralSet()
	for _, literal := range lits.Members() {
		cloned.Add(cloneBoundLiteral(literal))
	}

	return cloned
}

type boundSetLiteralRef interface {
	boundSetLiteralsRef() Set[Literal]
}

type boundSetExtremaRef interface {
	boundSetExtremaRef() (minLit, maxLit Literal, ok bool)
}

func (bsp *boundSetPredicate[T]) boundSetLiteralsRef() Set[Literal] {
	return bsp.lits
}

func (bsp *boundSetPredicate[T]) boundSetExtremaRef() (minLit, maxLit Literal, ok bool) {
	return bsp.minLiteral, bsp.maxLiteral, bsp.hasExtrema
}

func boundSetLiteralsForVisit(predicate BoundPredicate) Set[Literal] {
	if ref, ok := predicate.(boundSetLiteralRef); ok {
		return ref.boundSetLiteralsRef()
	}

	setPredicate, ok := predicate.(BoundSetPredicate)
	if !ok {
		panic(fmt.Errorf("%w: %s predicate %T does not implement BoundSetPredicate",
			ErrNotImplemented, predicate.Op(), predicate))
	}

	return setPredicate.Literals()
}
