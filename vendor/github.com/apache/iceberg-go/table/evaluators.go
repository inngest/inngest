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

package table

import (
	"encoding"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/apache/arrow-go/v18/parquet"
	"github.com/apache/arrow-go/v18/parquet/metadata"
	parquetschema "github.com/apache/arrow-go/v18/parquet/schema"
	"github.com/apache/iceberg-go"
	iceberginternal "github.com/apache/iceberg-go/internal"
	"github.com/apache/iceberg-go/table/internal"
	"github.com/google/uuid"
)

const (
	rowsMightMatch, rowsMustMatch      = true, true
	rowsCannotMatch, rowsMightNotMatch = false, false
	inPredicateLimit                   = 200
)

// newManifestEvaluator returns a function that can be used to evaluate whether a particular
// manifest file has rows that might or might not match a given partition filter by using
// the stats provided in the partitions (UpperBound/LowerBound/ContainsNull/ContainsNaN).
func newManifestEvaluator(spec iceberg.PartitionSpec, schema *iceberg.Schema, partitionFilter iceberg.BooleanExpression, caseSensitive bool) (func(iceberg.ManifestFile) (bool, error), error) {
	partType := spec.PartitionType(schema)
	partSchema := iceberg.NewSchema(0, partType.FieldList...)
	filter, err := iceberg.RewriteNotExpr(partitionFilter)
	if err != nil {
		return nil, err
	}

	boundFilter, err := iceberg.BindExpr(partSchema, filter, caseSensitive)
	if err != nil {
		return nil, err
	}

	return (&manifestEvalVisitor{partitionFilter: boundFilter}).Eval, nil
}

type manifestEvalVisitor struct {
	partitionFields []iceberg.FieldSummary
	partitionFilter iceberg.BooleanExpression
}

func (m *manifestEvalVisitor) Eval(manifest iceberg.ManifestFile) (bool, error) {
	parts := manifestFilePartitions(manifest)
	if len(parts) == 0 {
		return rowsMightMatch, nil
	}

	// Use a per-call visitor so concurrent callers of the same cached Eval
	// closure (e.g. the errgroup in classifyFilesForFilteredDeletions and
	// the parallel manifest scans in scanner.collectManifestEntries) do
	// not race on partitionFields. Mirrors the exprEvaluator fix above.
	ev := manifestEvalVisitor{
		partitionFields: parts,
		partitionFilter: m.partitionFilter,
	}

	result, err := iceberg.VisitExprEvaluator(ev.partitionFilter, &ev)
	if errors.Is(err, iceberg.ErrInvalidFixedLength) {
		return rowsMightMatch, nil
	}

	return result, err
}

func removeBoundCmp[T iceberg.LiteralType](bound iceberg.Literal, vals []iceberg.Literal, cmpToDelete int) []iceberg.Literal {
	val := bound.(iceberg.TypedLiteral[T])
	cmp := val.Comparator()

	return slices.DeleteFunc(vals, func(v iceberg.Literal) bool {
		return cmp(val.Value(), v.(iceberg.TypedLiteral[T]).Value()) == cmpToDelete
	})
}

func removeBoundCheck(bound iceberg.Literal, vals []iceberg.Literal, toDelete int) []iceberg.Literal {
	switch bound.Type().(type) {
	case iceberg.BooleanType:
		return removeBoundCmp[bool](bound, vals, toDelete)
	case iceberg.Int32Type:
		return removeBoundCmp[int32](bound, vals, toDelete)
	case iceberg.Int64Type:
		return removeBoundCmp[int64](bound, vals, toDelete)
	case iceberg.Float32Type:
		return removeBoundCmp[float32](bound, vals, toDelete)
	case iceberg.Float64Type:
		return removeBoundCmp[float64](bound, vals, toDelete)
	case iceberg.DateType:
		return removeBoundCmp[iceberg.Date](bound, vals, toDelete)
	case iceberg.TimeType:
		return removeBoundCmp[iceberg.Time](bound, vals, toDelete)
	case iceberg.TimestampType, iceberg.TimestampTzType:
		return removeBoundCmp[iceberg.Timestamp](bound, vals, toDelete)
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return removeBoundCmp[iceberg.TimestampNano](bound, vals, toDelete)
	case iceberg.BinaryType, iceberg.FixedType:
		return removeBoundCmp[[]byte](bound, vals, toDelete)
	case iceberg.StringType:
		return removeBoundCmp[string](bound, vals, toDelete)
	case iceberg.UUIDType:
		return removeBoundCmp[uuid.UUID](bound, vals, toDelete)
	case iceberg.DecimalType:
		return removeBoundCmp[iceberg.Decimal](bound, vals, toDelete)
	}
	panic("unrecognized type")
}

func allBoundCmp[T iceberg.LiteralType](bound iceberg.Literal, set iceberg.Set[iceberg.Literal], want int) bool {
	val := bound.(iceberg.TypedLiteral[T])
	cmp := val.Comparator()

	return set.All(func(e iceberg.Literal) bool {
		return cmp(val.Value(), e.(iceberg.TypedLiteral[T]).Value()) == want
	})
}

func allBoundCheck(bound iceberg.Literal, set iceberg.Set[iceberg.Literal], want int) bool {
	switch bound.Type().(type) {
	case iceberg.BooleanType:
		return allBoundCmp[bool](bound, set, want)
	case iceberg.Int32Type:
		return allBoundCmp[int32](bound, set, want)
	case iceberg.Int64Type:
		return allBoundCmp[int64](bound, set, want)
	case iceberg.Float32Type:
		return allBoundCmp[float32](bound, set, want)
	case iceberg.Float64Type:
		return allBoundCmp[float64](bound, set, want)
	case iceberg.DateType:
		return allBoundCmp[iceberg.Date](bound, set, want)
	case iceberg.TimeType:
		return allBoundCmp[iceberg.Time](bound, set, want)
	case iceberg.TimestampType, iceberg.TimestampTzType:
		return allBoundCmp[iceberg.Timestamp](bound, set, want)
	case iceberg.TimestampNsType, iceberg.TimestampTzNsType:
		return allBoundCmp[iceberg.TimestampNano](bound, set, want)
	case iceberg.BinaryType, iceberg.FixedType:
		return allBoundCmp[[]byte](bound, set, want)
	case iceberg.StringType:
		return allBoundCmp[string](bound, set, want)
	case iceberg.UUIDType:
		return allBoundCmp[uuid.UUID](bound, set, want)
	case iceberg.DecimalType:
		return allBoundCmp[iceberg.Decimal](bound, set, want)
	}
	panic(iceberg.ErrType)
}

func (m *manifestEvalVisitor) VisitIn(term iceberg.BoundTerm, literals iceberg.Set[iceberg.Literal]) bool {
	return m.visitIn(term, literals, nil, nil)
}

func (m *manifestEvalVisitor) VisitInWithExtrema(term iceberg.BoundTerm, literals iceberg.Set[iceberg.Literal], minLit, maxLit iceberg.Literal) bool {
	return m.visitIn(term, literals, minLit, maxLit)
}

func (m *manifestEvalVisitor) visitIn(term iceberg.BoundTerm, literals iceberg.Set[iceberg.Literal], minLit, maxLit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.LowerBound == nil {
		return rowsCannotMatch
	}

	lower, err := iceberg.LiteralFromBytes(term.Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	if maxLit != nil {
		if getCmpLiteral(lower)(lower, maxLit) > 0 {
			return rowsCannotMatch
		}
	} else {
		if literals.Len() > inPredicateLimit {
			return rowsMightMatch
		}
		if allBoundCheck(lower, literals, 1) {
			return rowsCannotMatch
		}
	}

	if field.UpperBound != nil {
		upper, err := iceberg.LiteralFromBytes(term.Type(), *field.UpperBound)
		if err != nil {
			panic(err)
		}

		if minLit != nil {
			if getCmpLiteral(upper)(upper, minLit) < 0 {
				return rowsCannotMatch
			}
		} else if allBoundCheck(upper, literals, -1) {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitNotIn(term iceberg.BoundTerm, literals iceberg.Set[iceberg.Literal]) bool {
	// because the bounds are not necessarily a min or max value, this cannot be answered using them
	// notIn(col, {X, ...}) with (X, Y) doesn't guarantee that X is a value in col
	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitIsNan(term iceberg.BoundTerm) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.ContainsNaN != nil && !*field.ContainsNaN {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitNotNan(term iceberg.BoundTerm) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.ContainsNaN != nil && *field.ContainsNaN && !field.ContainsNull && field.LowerBound == nil {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitIsNull(term iceberg.BoundTerm) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if !field.ContainsNull {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitNotNull(term iceberg.BoundTerm) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	// ContainsNull encodes whether at least one partition value is null
	// lowerBound is null if all partition values are null
	allNull := field.ContainsNull && field.LowerBound == nil
	if allNull && (term.Ref().Type().Equals(iceberg.PrimitiveTypes.Float32) || term.Ref().Type().Equals(iceberg.PrimitiveTypes.Float64)) {
		// floating point types may include NaN values, which we check separately
		// in case bounds don't include NaN values, ContainsNaN needsz to be checked
		allNull = field.ContainsNaN != nil && !*field.ContainsNaN
	}

	if allNull {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func getCmp[T iceberg.LiteralType](b iceberg.TypedLiteral[T]) func(iceberg.Literal, iceberg.Literal) int {
	cmp := b.Comparator()

	return func(l1, l2 iceberg.Literal) int {
		return cmp(l1.(iceberg.TypedLiteral[T]).Value(), l2.(iceberg.TypedLiteral[T]).Value())
	}
}

func getCmpLiteral(boundary iceberg.Literal) func(iceberg.Literal, iceberg.Literal) int {
	switch l := boundary.(type) {
	case iceberg.TypedLiteral[bool]:
		return getCmp(l)
	case iceberg.TypedLiteral[int32]:
		return getCmp(l)
	case iceberg.TypedLiteral[int64]:
		return getCmp(l)
	case iceberg.TypedLiteral[float32]:
		return getCmp(l)
	case iceberg.TypedLiteral[float64]:
		return getCmp(l)
	case iceberg.TypedLiteral[iceberg.Date]:
		return getCmp(l)
	case iceberg.TypedLiteral[iceberg.Time]:
		return getCmp(l)
	case iceberg.TypedLiteral[iceberg.Timestamp]:
		return getCmp(l)
	case iceberg.TypedLiteral[iceberg.TimestampNano]:
		return getCmp(l)
	case iceberg.TypedLiteral[[]byte]:
		return getCmp(l)
	case iceberg.TypedLiteral[string]:
		return getCmp(l)
	case iceberg.TypedLiteral[uuid.UUID]:
		return getCmp(l)
	case iceberg.TypedLiteral[iceberg.Decimal]:
		return getCmp(l)
	case iceberg.GeoLiteral:
		// Geometry/Geography bounds are coordinate tuples with no total order,
		// so they must never reach an ordering comparison. Reject explicitly
		// instead of falling through to the generic type panic, so the failure
		// names the real cause if a geo term is ever routed here.
		panic(fmt.Errorf("%w: geometry/geography has no ordering, cannot compare %s bounds",
			iceberg.ErrType, boundary.Type()))
	}
	panic(iceberg.ErrType)
}

func (m *manifestEvalVisitor) VisitEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.LowerBound == nil || field.UpperBound == nil {
		// values are all null and literal cannot contain null
		return rowsCannotMatch
	}

	lower, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	cmp := getCmpLiteral(lower)
	if cmp(lower, lit) == 1 {
		return rowsCannotMatch
	}

	upper, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.UpperBound)
	if err != nil {
		panic(err)
	}

	if cmp(lit, upper) == 1 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitNotEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	// because bounds are not necessarily a min or max, this cannot be answered
	// using them. notEq(col, X) with (X, Y) doesn't guarantee X is a value in col
	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitGreaterEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.UpperBound == nil {
		return rowsCannotMatch
	}

	upper, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.UpperBound)
	if err != nil {
		panic(err)
	}

	if getCmpLiteral(upper)(lit, upper) == 1 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitGreater(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.UpperBound == nil {
		return rowsCannotMatch
	}

	upper, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.UpperBound)
	if err != nil {
		panic(err)
	}

	if getCmpLiteral(upper)(lit, upper) >= 0 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitLessEqual(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.LowerBound == nil {
		return rowsCannotMatch
	}

	lower, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	if getCmpLiteral(lower)(lit, lower) == -1 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitLess(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.LowerBound == nil {
		return rowsCannotMatch
	}

	lower, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	if getCmpLiteral(lower)(lit, lower) <= 0 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	var prefix string
	if val, ok := lit.(iceberg.TypedLiteral[string]); ok {
		prefix = val.Value()
	} else {
		prefix = string(lit.(iceberg.TypedLiteral[[]byte]).Value())
	}

	lenPrefix := len(prefix)

	if field.LowerBound == nil {
		return rowsCannotMatch
	}

	lower, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	// truncate lower bound so that it's length is not greater than the length of prefix
	var v string
	switch l := lower.(type) {
	case iceberg.TypedLiteral[string]:
		v = l.Value()
		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}
	case iceberg.TypedLiteral[[]byte]:
		v = string(l.Value())
		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}
	}

	if v > prefix {
		return rowsCannotMatch
	}

	if field.UpperBound == nil {
		return rowsCannotMatch
	}

	upper, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.UpperBound)
	if err != nil {
		panic(err)
	}

	switch u := upper.(type) {
	case iceberg.TypedLiteral[string]:
		v = u.Value()
		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}
	case iceberg.TypedLiteral[[]byte]:
		v = string(u.Value())
		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}
	}

	if v < prefix {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitNotStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) bool {
	pos := term.Ref().Pos()
	field := m.partitionFields[pos]

	if field.ContainsNull || field.LowerBound == nil || field.UpperBound == nil {
		return rowsMightMatch
	}

	// NotStartsWith will match unless ALL values must start with the prefix.
	// this happens when the lower and upper bounds BOTH start with the prefix
	lower, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.LowerBound)
	if err != nil {
		panic(err)
	}

	upper, err := iceberg.LiteralFromBytes(term.Ref().Type(), *field.UpperBound)
	if err != nil {
		panic(err)
	}

	var prefix, lowerBound, upperBound string
	if val, ok := lit.(iceberg.TypedLiteral[string]); ok {
		prefix = val.Value()
		lowerBound, upperBound = lower.(iceberg.TypedLiteral[string]).Value(), upper.(iceberg.TypedLiteral[string]).Value()
	} else {
		prefix = string(lit.(iceberg.TypedLiteral[[]byte]).Value())
		lowerBound = string(lower.(iceberg.TypedLiteral[[]byte]).Value())
		upperBound = string(upper.(iceberg.TypedLiteral[[]byte]).Value())
	}

	lenPrefix := len(prefix)
	if len(lowerBound) < lenPrefix {
		return rowsMightMatch
	}

	if lowerBound[:lenPrefix] == prefix {
		// if upper is shorter then upper can't start with the prefix
		if len(upperBound) < lenPrefix {
			return rowsMightMatch
		}

		if upperBound[:lenPrefix] == prefix {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

// Manifest (partition-summary) pruning does not apply to geo predicates: no
// partition transform accepts a geometry/geography source (Transform.CanTransform
// rejects them), so a geo column is never a partition field and a bbox predicate
// always projects to AlwaysTrue before reaching this evaluator. These methods
// exist only to satisfy the visitor interface and conservatively keep every
// manifest. Data-file pruning is handled by inclusiveMetricsEval.
func (m *manifestEvalVisitor) VisitBBoxIntersects(iceberg.BoundTerm, iceberg.BoundingBox) bool {
	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitBBoxNotIntersects(iceberg.BoundTerm, iceberg.BoundingBox) bool {
	return rowsMightMatch
}

// These evaluators handle geospatial predicates, so they implement the optional
// iceberg.BoundGeospatialExprVisitor. The assertions keep that wiring
// compile-checked now that the geo methods live on that extension interface
// rather than on BoundBooleanExprVisitor.
var (
	_ iceberg.BoundGeospatialExprVisitor[bool]                         = (*manifestEvalVisitor)(nil)
	_ iceberg.BoundGeospatialExprVisitor[bool]                         = (*inclusiveMetricsEval)(nil)
	_ iceberg.BoundGeospatialExprVisitor[bool]                         = (*strictMetricsEval)(nil)
	_ iceberg.BoundGeospatialExprVisitor[[]internal.RowGroupBloomPred] = (*bloomPredicateCollector)(nil)
)

func (m *manifestEvalVisitor) VisitTrue() bool {
	return rowsMightMatch
}

func (m *manifestEvalVisitor) VisitFalse() bool {
	return rowsCannotMatch
}

func (m *manifestEvalVisitor) VisitUnbound(iceberg.UnboundPredicate) bool {
	panic("need bound predicate")
}

func (m *manifestEvalVisitor) VisitBound(pred iceberg.BoundPredicate) bool {
	if isTransformedTerm(pred.Term()) {
		return rowsMightMatch
	}

	return iceberg.VisitBoundPredicateRef(pred, m, iceberginternal.BoundPredicateRef{})
}

func (m *manifestEvalVisitor) VisitNot(child bool) bool       { return !child }
func (m *manifestEvalVisitor) VisitAnd(left, right bool) bool { return left && right }
func (m *manifestEvalVisitor) VisitOr(left, right bool) bool  { return left || right }

type projectionEvaluator struct {
	spec          iceberg.PartitionSpec
	schema        *iceberg.Schema
	caseSensitive bool
}

func (*projectionEvaluator) VisitTrue() iceberg.BooleanExpression  { return iceberg.AlwaysTrue{} }
func (*projectionEvaluator) VisitFalse() iceberg.BooleanExpression { return iceberg.AlwaysFalse{} }
func (*projectionEvaluator) VisitNot(child iceberg.BooleanExpression) iceberg.BooleanExpression {
	panic(fmt.Errorf("%w: cannot project 'not' expression, should be rewritten %s",
		iceberg.ErrInvalidArgument, child))
}

func (*projectionEvaluator) VisitAnd(left, right iceberg.BooleanExpression) iceberg.BooleanExpression {
	return iceberg.NewAnd(left, right)
}

func (*projectionEvaluator) VisitOr(left, right iceberg.BooleanExpression) iceberg.BooleanExpression {
	return iceberg.NewOr(left, right)
}

func (*projectionEvaluator) VisitUnbound(pred iceberg.UnboundPredicate) iceberg.BooleanExpression {
	panic(fmt.Errorf("%w: cannot project unbound predicate: %s", iceberg.ErrInvalidArgument, pred))
}

type inclusiveProjection struct{ projectionEvaluator }

func (p *inclusiveProjection) Project(expr iceberg.BooleanExpression) (iceberg.BooleanExpression, error) {
	expr, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	bound, err := iceberg.BindExpr(p.schema, expr, p.caseSensitive)
	if err != nil {
		return nil, err
	}

	return iceberg.VisitExpr(bound, p)
}

func (p *inclusiveProjection) VisitBound(pred iceberg.BoundPredicate) iceberg.BooleanExpression {
	parts := p.spec.FieldsBySourceID(pred.Term().Ref().Field().ID)

	var result iceberg.BooleanExpression = iceberg.AlwaysTrue{}
	for _, part := range parts {
		// consider (d = 2019-01-01) with bucket(7, d) and bucket(5, d)
		// projections: b1 = bucket(7, '2019-01-01') = 5, b2 = bucket(5, '2019-01-01') = 0
		// any value where b1 != 5 or any value where b2 != 0 cannot be the '2019-01-01'
		//
		// similarly, if partitioning by day(ts) and hour(ts), the more restrictive
		// projection should be used. ts = 2019-01-01T01:00:00 produces day=2019-01-01 and
		// hour=2019-01-01-01. the value will be in 2019-01-01-01 and not in 2019-01-01-02.
		inclProjection, err := part.Transform.Project(part.Name, pred)
		if err != nil {
			panic(err)
		}
		if inclProjection != nil {
			result = iceberg.NewAnd(result, inclProjection)
		}
	}

	return result
}

func newInclusiveProjection(s *iceberg.Schema, spec iceberg.PartitionSpec, caseSensitive bool) func(iceberg.BooleanExpression) (iceberg.BooleanExpression, error) {
	return (&inclusiveProjection{
		projectionEvaluator: projectionEvaluator{
			schema:        s,
			spec:          spec,
			caseSensitive: caseSensitive,
		},
	}).Project
}

type metricsEvaluator struct {
	valueCounts map[int]int64
	nullCounts  map[int]int64
	nanCounts   map[int]int64
	lowerBounds map[int][]byte
	upperBounds map[int][]byte
}

func isTransformedTerm(term iceberg.BoundTerm) bool {
	_, ok := term.(*iceberg.BoundTransform)

	return ok
}

func (m *metricsEvaluator) VisitTrue() bool  { return rowsMightMatch }
func (m *metricsEvaluator) VisitFalse() bool { return rowsCannotMatch }
func (m *metricsEvaluator) VisitNot(child bool) bool {
	panic(fmt.Errorf("%w: NOT should be rewritten %v", iceberg.ErrInvalidArgument, child))
}
func (m *metricsEvaluator) VisitAnd(left, right bool) bool { return left && right }
func (m *metricsEvaluator) VisitOr(left, right bool) bool  { return left || right }

func (m *metricsEvaluator) containsNullsOnly(id int) bool {
	valCount, ok := m.valueCounts[id]
	if !ok {
		return false
	}

	nullCount, ok := m.nullCounts[id]
	if !ok {
		return false
	}

	return valCount == nullCount
}

func (m *metricsEvaluator) containsNansOnly(id int) bool {
	nanCount, ok := m.nanCounts[id]
	if !ok {
		return false
	}

	valCount, ok := m.valueCounts[id]
	if !ok {
		return false
	}

	return nanCount == valCount
}

func (m *metricsEvaluator) isNan(v iceberg.Literal) bool {
	switch v := v.(type) {
	case iceberg.Float32Literal:
		return math.IsNaN(float64(v))
	case iceberg.Float64Literal:
		return math.IsNaN(float64(v))
	default:
		return false
	}
}

func newInclusiveMetricsEvaluator(s *iceberg.Schema, expr iceberg.BooleanExpression,
	caseSensitive bool, includeEmptyFiles bool,
) (func(iceberg.DataFile) (bool, error), error) {
	rewritten, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	bound, err := iceberg.BindExpr(s, rewritten, caseSensitive)
	if err != nil {
		return nil, err
	}

	return (&inclusiveMetricsEval{
		includeEmptyFiles: includeEmptyFiles,
		expr:              bound,
	}).Eval, nil
}

func newParquetRowGroupStatsEvaluator(fileSchema *iceberg.Schema, expr iceberg.BooleanExpression,
	includeEmptyFiles bool,
) (func(*metadata.RowGroupMetaData, []int) (bool, error), error) {
	rewritten, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	return newParquetRowGroupStatsEvaluatorFromRewritten(rewritten, includeEmptyFiles), nil
}

func newParquetRowGroupStatsEvaluatorFromRewritten(
	expr iceberg.BooleanExpression, includeEmptyFiles bool,
) func(*metadata.RowGroupMetaData, []int) (bool, error) {
	return (&inclusiveMetricsEval{
		includeEmptyFiles: includeEmptyFiles,
		expr:              expr,
	}).TestRowGroup
}

type inclusiveMetricsEval struct {
	metricsEvaluator
	expr              iceberg.BooleanExpression
	includeEmptyFiles bool
}

// intBackedDecimal reports whether a column's statistics are a decimal that
// Parquet stores in an INT32 or INT64. Parquet's plain encoding for those is
// little-endian, while an Iceberg bound is big-endian two's complement, so the
// stat bytes have to be reversed before they can be used as a bound. Read
// as-is, they decode to an unrelated value and can prune row groups that match.
// A FIXED_LEN_BYTE_ARRAY-backed decimal is big-endian in Parquet too, so its
// bounds are already in Iceberg's form.
func intBackedDecimal(descr *parquetschema.Column) bool {
	switch descr.PhysicalType() {
	case parquet.Types.Int32, parquet.Types.Int64:
		_, ok := descr.LogicalType().(parquetschema.DecimalLogicalType)

		return ok
	default:
		return false
	}
}

// parquetStatsBounds converts a Parquet column chunk's min/max statistics into
// Iceberg bound bytes. Any evaluator that populates bounds from raw Parquet
// statistics must go through here rather than calling EncodeMin/EncodeMax
// directly, or int-backed decimals silently prune rows that match.
func parquetStatsBounds(stats metadata.TypedStatistics) (lower, upper []byte) {
	lower, upper = stats.EncodeMin(), stats.EncodeMax()
	if intBackedDecimal(stats.Descr()) {
		// EncodeMin/EncodeMax return fresh buffers, so reversing in place is safe.
		// The spec wants minimum-width bounds, but these stay in memory and the
		// decoder accepts any width, so the 4- or 8-byte form needs no trimming.
		slices.Reverse(lower)
		slices.Reverse(upper)
	}

	return lower, upper
}

func (m *inclusiveMetricsEval) TestRowGroup(rgmeta *metadata.RowGroupMetaData, colIndices []int) (bool, error) {
	if !m.includeEmptyFiles && rgmeta.NumRows() == 0 {
		return rowsCannotMatch, nil
	}

	if m.valueCounts == nil {
		m.valueCounts = make(map[int]int64, len(colIndices))
	} else {
		clear(m.valueCounts)
	}
	if m.nullCounts == nil {
		m.nullCounts = make(map[int]int64, len(colIndices))
	} else {
		clear(m.nullCounts)
	}
	m.nanCounts = nil
	if m.lowerBounds != nil {
		clear(m.lowerBounds)
	}
	if m.upperBounds != nil {
		clear(m.upperBounds)
	}

	for _, c := range colIndices {
		colMeta, err := rgmeta.ColumnChunk(c)
		if err != nil {
			return false, err
		}

		if ok, err := colMeta.StatsSet(); !ok || err != nil {
			continue
		}

		stats, err := colMeta.Statistics()
		if err != nil {
			return false, err
		}

		if stats == nil {
			continue
		}

		fieldID := int(stats.Descr().SchemaNode().FieldID())
		m.valueCounts[fieldID] = stats.NumValues()
		if stats.HasNullCount() {
			m.nullCounts[fieldID] = stats.NullCount()
		}
		if stats.HasMinMax() {
			if m.lowerBounds == nil {
				// Lower and upper bounds are always allocated together.
				m.lowerBounds = make(map[int][]byte, len(colIndices))
				m.upperBounds = make(map[int][]byte, len(colIndices))
			}

			lower, upper := parquetStatsBounds(stats)
			m.lowerBounds[fieldID] = lower
			m.upperBounds[fieldID] = upper
		}
	}

	result, err := iceberg.VisitExprEvaluator(m.expr, m)
	if errors.Is(err, iceberg.ErrInvalidFixedLength) {
		return rowsMightMatch, nil
	}

	return result, err
}

func (m *inclusiveMetricsEval) Eval(file iceberg.DataFile) (bool, error) {
	if !m.includeEmptyFiles && file.Count() == 0 {
		return rowsCannotMatch, nil
	}

	// avoid race condition while maintaining existing state
	ev := inclusiveMetricsEval{
		includeEmptyFiles: m.includeEmptyFiles,
		expr:              m.expr,
	}

	ev.valueCounts, ev.nullCounts, ev.nanCounts, ev.lowerBounds, ev.upperBounds = dataFileStats(file)

	result, err := iceberg.VisitExprEvaluator(m.expr, &ev)
	if errors.Is(err, iceberg.ErrInvalidFixedLength) {
		return rowsMightMatch, nil
	}

	return result, err
}

func (m *inclusiveMetricsEval) mayContainNull(fieldID int) bool {
	if m.nullCounts == nil {
		return true
	}

	_, ok := m.nullCounts[fieldID]

	return ok
}

func (m *inclusiveMetricsEval) VisitUnbound(iceberg.UnboundPredicate) bool {
	panic("need bound predicate")
}

func (m *inclusiveMetricsEval) VisitBound(pred iceberg.BoundPredicate) bool {
	if isTransformedTerm(pred.Term()) {
		return rowsMightMatch
	}

	return iceberg.VisitBoundPredicateRef(pred, m, iceberginternal.BoundPredicateRef{})
}

func (m *inclusiveMetricsEval) VisitIsNull(t iceberg.BoundTerm) bool {
	if _, ok := t.(iceberg.BoundExtract); ok {
		return rowsMightMatch
	}

	fieldID := t.Ref().Field().ID
	if cnt, exists := m.nullCounts[fieldID]; exists && cnt == 0 {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitNotNull(t iceberg.BoundTerm) bool {
	// no need to check whether the field is required because binding evaluates
	// that case if the column has no non-null values, the expression cannot match
	fieldID := t.Ref().Field().ID
	if m.containsNullsOnly(fieldID) {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitIsNan(t iceberg.BoundTerm) bool {
	if _, ok := t.(iceberg.BoundExtract); ok {
		return rowsMightMatch
	}

	fieldID := t.Ref().Field().ID
	if cnt, exists := m.nanCounts[fieldID]; exists && cnt == 0 {
		return rowsCannotMatch
	}
	// when there's no nancounts information but we already know the column
	// contains null it's guaranteed that there's no nan value
	if m.containsNullsOnly(fieldID) {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitNotNan(t iceberg.BoundTerm) bool {
	if _, ok := t.(iceberg.BoundExtract); ok {
		return rowsMightMatch
	}

	fieldID := t.Ref().Field().ID

	if m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	return rowsMightMatch
}

// boundFor decodes the file bound for term t from raw: a scalar for a reference, or the
// variant sub-path value for an extract; ok is false when raw is nil or not castable.
func (m *inclusiveMetricsEval) boundFor(t iceberg.BoundTerm, raw []byte) (iceberg.Literal, bool) {
	if raw == nil {
		return nil, false
	}

	if ext, ok := t.(iceberg.BoundExtract); ok {
		lit, found, err := internal.VariantBoundLiteral(raw, ext.Path(), ext.Type().(iceberg.PrimitiveType))
		if err != nil {
			// Best-effort pruning: an undecodable bound means we cannot prune, not that the scan fails.
			return nil, false
		}

		return lit, found
	}

	field := t.Ref().Field()
	if _, ok := field.Type.(iceberg.PrimitiveType); !ok {
		panic(fmt.Errorf("%w: expected iceberg.PrimitiveType, got %s",
			iceberg.ErrInvalidTypeString, field.Type))
	}

	lit, err := iceberg.LiteralFromBytes(field.Type, raw)
	if err != nil {
		panic(err)
	}

	return lit, true
}

func (m *inclusiveMetricsEval) VisitLess(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if lowerBound, ok := m.boundFor(t, m.lowerBounds[fieldID]); ok {
		if m.isNan(lowerBound) {
			// nan indicates unreliable bounds
			return rowsMightMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, lit) >= 0 {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitLessEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if lowerBound, ok := m.boundFor(t, m.lowerBounds[fieldID]); ok {
		if m.isNan(lowerBound) {
			// nan indicates unreliable bounds
			return rowsMightMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, lit) > 0 {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitGreater(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if upperBound, ok := m.boundFor(t, m.upperBounds[fieldID]); ok {
		if getCmpLiteral(upperBound)(upperBound, lit) <= 0 {
			if m.isNan(upperBound) {
				return rowsMightMatch
			}

			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitGreaterEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if upperBound, ok := m.boundFor(t, m.upperBounds[fieldID]); ok {
		if getCmpLiteral(upperBound)(upperBound, lit) < 0 {
			if m.isNan(upperBound) {
				return rowsMightMatch
			}

			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if lowerBound, ok := m.boundFor(t, m.lowerBounds[fieldID]); ok {
		if m.isNan(lowerBound) {
			return rowsMightMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, lit) == 1 {
			return rowsCannotMatch
		}
	}

	if upperBound, ok := m.boundFor(t, m.upperBounds[fieldID]); ok {
		if m.isNan(upperBound) {
			return rowsMightMatch
		}

		if getCmpLiteral(upperBound)(upperBound, lit) == -1 {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitNotEqual(iceberg.BoundTerm, iceberg.Literal) bool {
	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitIn(t iceberg.BoundTerm, s iceberg.Set[iceberg.Literal]) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsCannotMatch
	}

	if s.Len() > inPredicateLimit {
		// skip evaluating the predicate if the number of values is too big
		return rowsMightMatch
	}

	values := s.Members()
	if lowerBound, ok := m.boundFor(t, m.lowerBounds[fieldID]); ok {
		if m.isNan(lowerBound) {
			return rowsMightMatch
		}

		values = removeBoundCheck(lowerBound, values, 1)
		if len(values) == 0 {
			return rowsCannotMatch
		}
	}

	if upperBound, ok := m.boundFor(t, m.upperBounds[fieldID]); ok {
		if m.isNan(upperBound) {
			return rowsMightMatch
		}

		values = removeBoundCheck(upperBound, values, -1)
		if len(values) == 0 {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitNotIn(iceberg.BoundTerm, iceberg.Set[iceberg.Literal]) bool {
	// because the bounds are not necessarily a min or max value, this cannot be
	// answered using them. notIn(col, {X, ...}) with (XX, Y) doesn't guarantee that
	// X is a value in col
	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitStartsWith(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNullsOnly(fieldID) {
		return rowsCannotMatch
	}

	var prefix string
	if val, ok := lit.(iceberg.TypedLiteral[string]); ok {
		prefix = val.Value()
	} else {
		prefix = string(lit.(iceberg.TypedLiteral[[]byte]).Value())
	}

	lenPrefix := len(prefix)

	if lowerBound, ok := m.boundFor(t, m.lowerBounds[fieldID]); ok {
		var v string
		switch l := lowerBound.(type) {
		case iceberg.TypedLiteral[string]:
			v = l.Value()
		case iceberg.TypedLiteral[[]byte]:
			v = string(l.Value())
		}

		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}

		if len(v) > 0 && v > prefix {
			return rowsCannotMatch
		}
	}

	if upperBound, ok := m.boundFor(t, m.upperBounds[fieldID]); ok {
		var v string
		switch u := upperBound.(type) {
		case iceberg.TypedLiteral[string]:
			v = u.Value()
		case iceberg.TypedLiteral[[]byte]:
			v = string(u.Value())
		}

		if len(v) > lenPrefix {
			v = v[:lenPrefix]
		}

		if len(v) > 0 && v < prefix {
			return rowsCannotMatch
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitNotStartsWith(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	if _, ok := t.(iceberg.BoundExtract); ok {
		return rowsMightMatch
	}

	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNull(fieldID) {
		return rowsMightMatch
	}

	if _, ok := field.Type.(iceberg.PrimitiveType); !ok {
		panic(fmt.Errorf("%w: expected iceberg.PrimitiveType, got %s",
			iceberg.ErrInvalidTypeString, field.Type))
	}

	// not_starts_with will match unless all values must start with the prefix.
	// this happens when the lower and upper bounds both start with the prefix
	lowerBoundBytes, upperBoundBytes := m.lowerBounds[fieldID], m.upperBounds[fieldID]
	if lowerBoundBytes != nil && upperBoundBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBoundBytes)
		if err != nil {
			panic(err)
		}

		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBoundBytes)
		if err != nil {
			panic(err)
		}

		var prefix, lower, upper string
		if val, ok := lit.(iceberg.TypedLiteral[string]); ok {
			prefix = val.Value()
			lower, upper = lowerBound.(iceberg.TypedLiteral[string]).Value(), upperBound.(iceberg.TypedLiteral[string]).Value()
		} else {
			prefix = string(lit.(iceberg.TypedLiteral[[]byte]).Value())
			lower, upper = string(lowerBound.(iceberg.TypedLiteral[[]byte]).Value()), string(upperBound.(iceberg.TypedLiteral[[]byte]).Value())
		}

		lenPrefix := len(prefix)
		if len(lower) < lenPrefix {
			return rowsMightMatch
		}

		if lower[:lenPrefix] == prefix {
			if len(upper) < lenPrefix {
				return rowsMightMatch
			}

			if upper[:lenPrefix] == prefix {
				return rowsCannotMatch
			}
		}
	}

	return rowsMightMatch
}

func (m *inclusiveMetricsEval) VisitBBoxIntersects(t iceberg.BoundTerm, bbox iceberg.BoundingBox) bool {
	fieldID := t.Ref().Field().ID

	// If the column is entirely null, no geometry can intersect the query box.
	if m.containsNullsOnly(fieldID) {
		return rowsCannotMatch
	}

	// Only geometry bounds are safe to prune with a planar min/max compare.
	// Geography bounds are geodesic and may cross the antimeridian (lower_x >
	// upper_x; spec Appendix D), which scalar XY intersection would mis-handle
	// and wrongly prune. iceberg-go emits no geography bounds, but files written
	// by other engines can, so guard on the column type rather than on presence.
	if _, isGeography := t.Ref().Field().Type.(iceberg.GeographyType); isGeography {
		return rowsMightMatch
	}

	// Prune only when both geo bounds are present and decode cleanly. A missing
	// or malformed bound leaves the file unprunable, which is always safe.
	minX, minY, maxX, maxY, ok := internal.GeoBoundsXY(m.lowerBounds[fieldID], m.upperBounds[fieldID])
	if !ok {
		return rowsMightMatch
	}

	if internal.BBoxIntersectsXY(minX, minY, maxX, maxY, bbox.MinX, bbox.MinY, bbox.MaxX, bbox.MaxY) {
		return rowsMightMatch
	}

	return rowsCannotMatch
}

func (m *inclusiveMetricsEval) VisitBBoxNotIntersects(iceberg.BoundTerm, iceberg.BoundingBox) bool {
	// A file whose bounds intersect the query box may still hold only geometries
	// outside it, so not-intersects cannot be answered from bounds alone.
	return rowsMightMatch
}

func newStrictMetricsEvaluator(s *iceberg.Schema, expr iceberg.BooleanExpression,
	caseSensitive bool, includeEmptyFiles bool,
) (func(iceberg.DataFile) (bool, error), error) {
	rewritten, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	bound, err := iceberg.BindExpr(s, rewritten, caseSensitive)
	if err != nil {
		return nil, err
	}

	return (&strictMetricsEval{
		includeEmptyFiles: includeEmptyFiles,
		expr:              bound,
	}).Eval, nil
}

type strictMetricsEval struct {
	metricsEvaluator
	expr              iceberg.BooleanExpression
	includeEmptyFiles bool
}

func (m *strictMetricsEval) Eval(file iceberg.DataFile) (bool, error) {
	if !m.includeEmptyFiles && file.Count() <= 0 {
		return rowsMustMatch, nil
	}

	// avoid race condition while maintaining existing state
	ev := strictMetricsEval{
		includeEmptyFiles: m.includeEmptyFiles,
		expr:              m.expr,
	}

	ev.valueCounts, ev.nullCounts, ev.nanCounts, ev.lowerBounds, ev.upperBounds = dataFileStats(file)

	result, err := iceberg.VisitExprEvaluator(m.expr, &ev)
	if errors.Is(err, iceberg.ErrInvalidFixedLength) {
		return rowsMightNotMatch, nil
	}

	return result, err
}

func (m *strictMetricsEval) VisitUnbound(iceberg.UnboundPredicate) bool {
	panic("need bound predicate")
}

func (m *strictMetricsEval) VisitBound(pred iceberg.BoundPredicate) bool {
	if _, ok := pred.Term().(iceberg.BoundExtract); ok {
		return rowsMightNotMatch
	}
	if isTransformedTerm(pred.Term()) {
		return rowsMightNotMatch
	}

	return iceberg.VisitBoundPredicateRef(pred, m, iceberginternal.BoundPredicateRef{})
}

func (m *strictMetricsEval) VisitIsNull(t iceberg.BoundTerm) bool {
	fieldID := t.Ref().Field().ID
	if m.containsNullsOnly(fieldID) {
		return rowsMustMatch
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitNotNull(t iceberg.BoundTerm) bool {
	fieldID := t.Ref().Field().ID
	if cnt, exists := m.nullCounts[fieldID]; exists && cnt == 0 {
		return rowsMustMatch
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitIsNan(t iceberg.BoundTerm) bool {
	fieldID := t.Ref().Field().ID

	if m.containsNansOnly(fieldID) {
		return rowsMustMatch
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitNotNan(t iceberg.BoundTerm) bool {
	fieldID := t.Ref().Field().ID

	if cnt, exists := m.nanCounts[fieldID]; exists && cnt == 0 {
		return rowsMustMatch
	}

	if m.containsNullsOnly(fieldID) {
		return rowsMustMatch
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitLess(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	if upperBoundBytes := m.upperBounds[fieldID]; upperBoundBytes != nil {
		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBoundBytes)
		if err != nil {
			panic(err)
		}

		if getCmpLiteral(upperBound)(upperBound, lit) < 0 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitLessEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	if upperBoundBytes := m.upperBounds[fieldID]; upperBoundBytes != nil {
		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBoundBytes)
		if err != nil {
			panic(err)
		}

		if getCmpLiteral(upperBound)(upperBound, lit) <= 0 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitGreater(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	if lowerBoundBytes := m.lowerBounds[fieldID]; lowerBoundBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBoundBytes)
		if err != nil {
			panic(err)
		}

		if m.isNan(lowerBound) {
			// NaN indicates unreliable bounds.
			return rowsMightNotMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, lit) > 0 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitGreaterEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	if lowerBoundBytes := m.lowerBounds[fieldID]; lowerBoundBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBoundBytes)
		if err != nil {
			panic(err)
		}

		if m.isNan(lowerBound) {
			// NaN indicates unreliable bounds.
			return rowsMightNotMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, lit) >= 0 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	lowerBytes := m.lowerBounds[fieldID]
	upperBytes := m.upperBounds[fieldID]

	if lowerBytes != nil && upperBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBytes)
		if err != nil {
			panic(err)
		}
		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBytes)
		if err != nil {
			panic(err)
		}
		if getCmpLiteral(lowerBound)(lowerBound, lit) != 0 || getCmpLiteral(upperBound)(upperBound, lit) != 0 {
			return rowsMightNotMatch
		} else {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitNotEqual(t iceberg.BoundTerm, lit iceberg.Literal) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsMustMatch
	}

	var cmp func(iceberg.Literal, iceberg.Literal) int
	if lowerBoundBytes := m.lowerBounds[fieldID]; lowerBoundBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBoundBytes)
		if err != nil {
			panic(err)
		}

		if m.isNan(lowerBound) {
			return rowsMightNotMatch
		}

		cmp = getCmpLiteral(lowerBound)
		if cmp(lowerBound, lit) == 1 {
			return rowsMustMatch
		}
	}

	if upperBoundBytes := m.upperBounds[fieldID]; upperBoundBytes != nil {
		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBoundBytes)
		if err != nil {
			panic(err)
		}

		if m.isNan(upperBound) {
			return rowsMightNotMatch
		}

		cmp = getCmpLiteral(upperBound)
		if cmp(upperBound, lit) == -1 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitIn(t iceberg.BoundTerm, s iceberg.Set[iceberg.Literal]) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.mayContainNulls(field) || m.mayContainNans(field) {
		return rowsMightNotMatch
	}

	lowerBytes := m.lowerBounds[fieldID]
	upperBytes := m.upperBounds[fieldID]

	if lowerBytes != nil && upperBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBytes)
		if err != nil {
			panic(err)
		}
		if !s.Contains(lowerBound) {
			return rowsMightNotMatch
		}

		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBytes)
		if err != nil {
			panic(err)
		}
		if !s.Contains(upperBound) {
			return rowsMightNotMatch
		}

		if getCmpLiteral(lowerBound)(lowerBound, upperBound) != 0 {
			return rowsMightNotMatch
		}

		return rowsMustMatch
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitNotIn(t iceberg.BoundTerm, s iceberg.Set[iceberg.Literal]) bool {
	field := t.Ref().Field()
	fieldID := field.ID

	if m.containsNullsOnly(fieldID) || m.containsNansOnly(fieldID) {
		return rowsMustMatch
	}

	values := s.Members()
	if lowerBoundBytes := m.lowerBounds[fieldID]; lowerBoundBytes != nil {
		lowerBound, err := iceberg.LiteralFromBytes(field.Type, lowerBoundBytes)
		if err != nil {
			panic(err)
		}

		if m.isNan(lowerBound) {
			return rowsMightNotMatch
		}

		values = removeBoundCheck(lowerBound, values, 1)
		if len(values) == 0 {
			return rowsMustMatch
		}
	}

	if upperBoundBytes := m.upperBounds[fieldID]; upperBoundBytes != nil {
		upperBound, err := iceberg.LiteralFromBytes(field.Type, upperBoundBytes)
		if err != nil {
			panic(err)
		}

		values = removeBoundCheck(upperBound, values, -1)
		if len(values) == 0 {
			return rowsMustMatch
		}
	}

	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitStartsWith(iceberg.BoundTerm, iceberg.Literal) bool {
	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitNotStartsWith(iceberg.BoundTerm, iceberg.Literal) bool {
	return rowsMightNotMatch
}

// Strict evaluation asks whether every row must match. Bounds can prove a bbox
// predicate false for a file (disjoint bounds) but never that every geometry
// intersects the query box, so neither variant can assert rowsMustMatch.
func (m *strictMetricsEval) VisitBBoxIntersects(iceberg.BoundTerm, iceberg.BoundingBox) bool {
	return rowsMightNotMatch
}

func (m *strictMetricsEval) VisitBBoxNotIntersects(iceberg.BoundTerm, iceberg.BoundingBox) bool {
	return rowsMightNotMatch
}

func (m *strictMetricsEval) mayContainNulls(field iceberg.NestedField) bool {
	cnt, exists := m.nullCounts[field.ID]
	if !exists {
		return !field.Required
	}

	return cnt > 0
}

func (m *strictMetricsEval) mayContainNans(field iceberg.NestedField) bool {
	switch field.Type.(type) {
	case iceberg.Float32Type, iceberg.Float64Type:
		cnt, exists := m.nanCounts[field.ID]
		if !exists {
			return true
		}

		return cnt > 0
	default:
		return false
	}
}

// literalToPhysBytes converts an Iceberg literal to the physical byte
// representation that the Parquet bloom filter writer hashes, using the
// literal's MarshalBinary encoding. Float types are normalized before
// encoding (-0.0 → 0.0, NaN → canonical NaN) to match Parquet/Java writer
// behavior and avoid false-negative bloom filter results.
func literalToPhysBytes(lit iceberg.Literal) ([]byte, bool) {
	switch v := lit.(type) {
	case iceberg.Float32Literal:
		// Normalize: -0.0 and 0.0 have different bit patterns, so without this
		// a query for 0.0 won't match a row group where -0.0 was written.
		f := float32(v)
		if f == 0 {
			f = 0
		} else if math.IsNaN(float64(f)) {
			f = float32(math.NaN())
		}

		lit = iceberg.Float32Literal(f)

	case iceberg.Float64Literal:
		f := float64(v)
		if f == 0 {
			f = 0
		} else if math.IsNaN(f) {
			f = math.NaN()
		}

		lit = iceberg.Float64Literal(f)

	case iceberg.BoolLiteral:
		// Low cardinality; bloom filters not useful.
		return nil, false

	case iceberg.DecimalLiteral:
		// Physical representation depends on precision (INT32/INT64/FIXED_LEN_BYTE_ARRAY).
		// TODO: add decimal bloom filter support once schema-level precision lookup is wired in.
		_ = v

		return nil, false

	case iceberg.TimestampNsLiteral:
		// Not yet handled by all expression machinery; skip for safety.
		_ = v

		return nil, false
	}

	m, ok := lit.(encoding.BinaryMarshaler)
	if !ok {
		return nil, false
	}

	b, err := m.MarshalBinary()
	if err != nil {
		return nil, false
	}

	return b, true
}

// bloomPredicateCollector walks a bound expression and collects EqualTo/In
// predicates as RowGroupBloomPred values. Only conjuncts (And chains) are
// collected; Or suppresses collection because a row group can only be skipped
// when ALL disjuncts are absent — which we do not evaluate here.
type bloomPredicateCollector struct{}

func (c *bloomPredicateCollector) VisitTrue() []internal.RowGroupBloomPred  { return nil }
func (c *bloomPredicateCollector) VisitFalse() []internal.RowGroupBloomPred { return nil }
func (c *bloomPredicateCollector) VisitNot(_ []internal.RowGroupBloomPred) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitAnd(left, right []internal.RowGroupBloomPred) []internal.RowGroupBloomPred {
	return append(left, right...)
}

func (c *bloomPredicateCollector) VisitOr(_, _ []internal.RowGroupBloomPred) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitUnbound(_ iceberg.UnboundPredicate) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitBound(pred iceberg.BoundPredicate) []internal.RowGroupBloomPred {
	if isTransformedTerm(pred.Term()) {
		return nil
	}

	return iceberg.VisitBoundPredicateRef(pred, c, iceberginternal.BoundPredicateRef{})
}

func (c *bloomPredicateCollector) VisitEqual(t iceberg.BoundTerm, lit iceberg.Literal) []internal.RowGroupBloomPred {
	field := t.Ref().Field()
	if _, ok := field.Type.(iceberg.PrimitiveType); !ok {
		return nil
	}

	b, ok := literalToPhysBytes(lit)
	if !ok {
		return nil
	}

	return []internal.RowGroupBloomPred{{FieldID: field.ID, PhysBytes: [][]byte{b}}}
}

func (c *bloomPredicateCollector) VisitIn(t iceberg.BoundTerm, s iceberg.Set[iceberg.Literal]) []internal.RowGroupBloomPred {
	if s.Len() > inPredicateLimit {
		return nil
	}

	field := t.Ref().Field()
	if _, ok := field.Type.(iceberg.PrimitiveType); !ok {
		return nil
	}

	physBytes := make([][]byte, 0, s.Len())
	for _, lit := range s.Members() {
		b, ok := literalToPhysBytes(lit)
		if !ok {
			return nil
		}

		physBytes = append(physBytes, b)
	}

	return []internal.RowGroupBloomPred{{FieldID: field.ID, PhysBytes: physBytes}}
}

func (c *bloomPredicateCollector) VisitNotIn(_ iceberg.BoundTerm, _ iceberg.Set[iceberg.Literal]) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitIsNan(_ iceberg.BoundTerm) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitNotNan(_ iceberg.BoundTerm) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitIsNull(_ iceberg.BoundTerm) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitNotNull(_ iceberg.BoundTerm) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitNotEqual(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitGreaterEqual(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitGreater(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitLessEqual(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitLess(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitStartsWith(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitNotStartsWith(_ iceberg.BoundTerm, _ iceberg.Literal) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitBBoxIntersects(_ iceberg.BoundTerm, _ iceberg.BoundingBox) []internal.RowGroupBloomPred {
	return nil
}

func (c *bloomPredicateCollector) VisitBBoxNotIntersects(_ iceberg.BoundTerm, _ iceberg.BoundingBox) []internal.RowGroupBloomPred {
	return nil
}

// newBloomFilterPredicates walks expr and returns bloom-filter-checkable
// predicates for row group pruning. Returns nil (no predicates) for
// AlwaysTrue or any expression with no EqualTo/In conjuncts.
func newBloomFilterPredicates(expr iceberg.BooleanExpression) ([]internal.RowGroupBloomPred, error) {
	if expr == nil {
		return nil, nil
	}

	rewritten, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	return newBloomFilterPredicatesFromRewritten(rewritten)
}

func newBloomFilterPredicatesFromRewritten(expr iceberg.BooleanExpression) ([]internal.RowGroupBloomPred, error) {
	if expr == nil {
		return nil, nil
	}

	return iceberg.VisitExpr(expr, &bloomPredicateCollector{})
}

// newDictionaryPredicates reuses the same conservative EqualTo/In collector
// as Bloom filters. The physical literal bytes are also the representation
// needed to compare values decoded from a PLAIN dictionary page.
func newDictionaryPredicates(expr iceberg.BooleanExpression) ([]internal.RowGroupDictionaryPred, error) {
	if expr == nil {
		return nil, nil
	}

	rewritten, err := iceberg.RewriteNotExpr(expr)
	if err != nil {
		return nil, err
	}

	return newDictionaryPredicatesFromRewritten(rewritten)
}

func newDictionaryPredicatesFromRewritten(expr iceberg.BooleanExpression) ([]internal.RowGroupDictionaryPred, error) {
	bloomPreds, err := newBloomFilterPredicatesFromRewritten(expr)
	if err != nil {
		return nil, err
	}

	if len(bloomPreds) == 0 {
		return nil, nil
	}

	dictionaryPreds := make([]internal.RowGroupDictionaryPred, len(bloomPreds))
	for i, pred := range bloomPreds {
		dictionaryPreds[i] = internal.RowGroupDictionaryPred(pred)
	}

	return dictionaryPreds, nil
}
