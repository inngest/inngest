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

// IsNull is a convenience wrapper for calling UnaryPredicate(OpIsNull, t)
//
// Will panic if t is nil
func IsNull(t UnboundTerm) UnboundPredicate {
	return UnaryPredicate(OpIsNull, t)
}

// NotNull is a convenience wrapper for calling UnaryPredicate(OpNotNull, t)
//
// Will panic if t is nil
func NotNull(t UnboundTerm) UnboundPredicate {
	return UnaryPredicate(OpNotNull, t)
}

// IsNaN is a convenience wrapper for calling UnaryPredicate(OpIsNan, t)
//
// Will panic if t is nil
func IsNaN(t UnboundTerm) UnboundPredicate {
	return UnaryPredicate(OpIsNan, t)
}

// NotNaN is a convenience wrapper for calling UnaryPredicate(OpNotNan, t)
//
// Will panic if t is nil
func NotNaN(t UnboundTerm) UnboundPredicate {
	return UnaryPredicate(OpNotNan, t)
}

// IsIn is a convenience wrapper for constructing an unbound set predicate for
// OpIn. It returns a BooleanExpression instead of an UnboundPredicate because
// depending on the arguments, it can automatically reduce to AlwaysFalse or
// AlwaysTrue (if given no values for examples). It may also reduce to EqualTo
// if only one value is provided.
//
// Will panic if t is nil
func IsIn[T LiteralType](t UnboundTerm, vals ...T) BooleanExpression {
	lits := make([]Literal, 0, len(vals))
	for _, v := range vals {
		lits = append(lits, NewLiteral(v))
	}

	return SetPredicate(OpIn, t, lits)
}

// NotIn is a convenience wrapper for constructing an unbound set predicate for
// OpNotIn. It returns a BooleanExpression instead of an UnboundPredicate because
// depending on the arguments, it can automatically reduce to AlwaysFalse or
// AlwaysTrue (if given no values for examples). It may also reduce to NotEqualTo
// if only one value is provided.
//
// Will panic if t is nil
func NotIn[T LiteralType](t UnboundTerm, vals ...T) BooleanExpression {
	lits := make([]Literal, 0, len(vals))
	for _, v := range vals {
		lits = append(lits, NewLiteral(v))
	}

	return SetPredicate(OpNotIn, t, lits)
}

// EqualTo is a convenience wrapper for calling LiteralPredicate(OpEQ, t, NewLiteral(v))
//
// Will panic if t is nil
func EqualTo[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpEQ, t, NewLiteral(v))
}

// NotEqualTo is a convenience wrapper for calling LiteralPredicate(OpNEQ, t, NewLiteral(v))
//
// Will panic if t is nil
func NotEqualTo[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpNEQ, t, NewLiteral(v))
}

// GreaterThanEqual is a convenience wrapper for calling LiteralPredicate(OpGTEQ,
// t, NewLiteral(v))
//
// Will panic if t is nil
func GreaterThanEqual[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpGTEQ, t, NewLiteral(v))
}

// GreaterThan is a convenience wrapper for calling LiteralPredicate(OpGT,
// t, NewLiteral(v))
//
// Will panic if t is nil
func GreaterThan[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpGT, t, NewLiteral(v))
}

// LessThanEqual is a convenience wrapper for calling LiteralPredicate(OpLTEQ,
// t, NewLiteral(v))
//
// Will panic if t is nil
func LessThanEqual[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpLTEQ, t, NewLiteral(v))
}

// LessThan is a convenience wrapper for calling LiteralPredicate(OpLT,
// t, NewLiteral(v))
//
// Will panic if t is nil
func LessThan[T LiteralType](t UnboundTerm, v T) UnboundPredicate {
	return LiteralPredicate(OpLT, t, NewLiteral(v))
}

// StartsWith is a convenience wrapper for calling LiteralPredicate(OpStartsWith,
// t, NewLiteral(v))
//
// Will panic if t is nil
func StartsWith(t UnboundTerm, v string) UnboundPredicate {
	return LiteralPredicate(OpStartsWith, t, NewLiteral(v))
}

// NotStartsWith is a convenience wrapper for calling LiteralPredicate(OpNotStartsWith,
// t, NewLiteral(v))
//
// Will panic if t is nil
func NotStartsWith(t UnboundTerm, v string) UnboundPredicate {
	return LiteralPredicate(OpNotStartsWith, t, NewLiteral(v))
}
