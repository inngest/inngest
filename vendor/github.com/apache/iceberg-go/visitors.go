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
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/apache/iceberg-go/internal"
	"github.com/google/uuid"
)

// BooleanExprVisitor is an interface for recursively visiting the nodes of a
// boolean expression
type BooleanExprVisitor[T any] interface {
	VisitTrue() T
	VisitFalse() T
	VisitNot(childResult T) T
	VisitAnd(left, right T) T
	VisitOr(left, right T) T
	VisitUnbound(UnboundPredicate) T
	VisitBound(BoundPredicate) T
}

// BoundBooleanExprVisitor builds on BooleanExprVisitor by adding interface
// methods for visiting bound expressions, because we do casting of literals
// during binding you can assume that the BoundTerm and the Literal passed
// to a method have the same type. Sets passed to VisitIn and VisitNotIn are
// detached when dispatched through VisitBoundPredicate. The internal
// VisitBoundPredicateRef path may pass borrowed sets to trusted built-in
// visitors, which must not call Add or mutate their literal values.
type BoundBooleanExprVisitor[T any] interface {
	BooleanExprVisitor[T]

	VisitIn(BoundTerm, Set[Literal]) T
	VisitNotIn(BoundTerm, Set[Literal]) T
	VisitIsNan(BoundTerm) T
	VisitNotNan(BoundTerm) T
	VisitIsNull(BoundTerm) T
	VisitNotNull(BoundTerm) T
	VisitEqual(BoundTerm, Literal) T
	VisitNotEqual(BoundTerm, Literal) T
	VisitGreaterEqual(BoundTerm, Literal) T
	VisitGreater(BoundTerm, Literal) T
	VisitLessEqual(BoundTerm, Literal) T
	VisitLess(BoundTerm, Literal) T
	VisitStartsWith(BoundTerm, Literal) T
	VisitNotStartsWith(BoundTerm, Literal) T
}

// BoundGeospatialExprVisitor is an optional extension a BoundBooleanExprVisitor
// may implement to handle the geospatial bbox predicates (BBoxIntersects /
// BBoxNotIntersects). It is deliberately kept out of BoundBooleanExprVisitor:
// adding methods to that interface would break every existing implementation -
// including external ones (query-engine adapters, downstream visitors) - which
// would then fail to compile until they grew geo methods they may not care
// about. This mirrors Java, which dispatches geospatial predicates outside its
// BoundExpressionVisitor. VisitBoundPredicate routes a bbox predicate here only
// when the visitor implements this interface; a visitor that does not is treated
// as not supporting geospatial predicates.
type BoundGeospatialExprVisitor[T any] interface {
	VisitBBoxIntersects(BoundTerm, BoundingBox) T
	VisitBBoxNotIntersects(BoundTerm, BoundingBox) T
}

// boundSetExtremaExprVisitor is consulted only by the borrowed
// VisitBoundPredicateRef path. The table package's manifest evaluator is
// currently the only implementation.
type boundSetExtremaExprVisitor[T any] interface {
	VisitInWithExtrema(BoundTerm, Set[Literal], Literal, Literal) T
}

// VisitExpr is a convenience function to use a given visitor to visit all parts of
// a boolean expression in-order. Values returned from the methods are passed to the
// subsequent methods, effectively "bubbling up" the results.
func VisitExpr[T any](expr BooleanExpression, visitor BooleanExprVisitor[T]) (res T, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("error encountered during visitExpr: %s", e)
			case error:
				err = e
			}
		}
	}()

	return visitBoolExpr(expr, visitor), err
}

// VisitExprEvaluator traverses a boolean expression for a visitor whose
// boolean results represent expression truth. Unlike VisitExpr, it only visits
// the nodes required to determine the result: it skips the right side of an
// AND when the left side is false and the right side of an OR when the left
// side is true.
func VisitExprEvaluator(expr BooleanExpression, visitor BooleanExprVisitor[bool]) (res bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case string:
				err = fmt.Errorf("error encountered during visitExpr: %s", e)
			case error:
				err = e
			}
		}
	}()

	return visitEvaluator(expr, visitor), err
}

func visitBoolExpr[T any](e BooleanExpression, visitor BooleanExprVisitor[T]) T {
	switch e := e.(type) {
	case AlwaysFalse:
		return visitor.VisitFalse()
	case AlwaysTrue:
		return visitor.VisitTrue()
	case AndExpr:
		left, right := visitBoolExpr(e.left, visitor), visitBoolExpr(e.right, visitor)

		return visitor.VisitAnd(left, right)
	case OrExpr:
		left, right := visitBoolExpr(e.left, visitor), visitBoolExpr(e.right, visitor)

		return visitor.VisitOr(left, right)
	case NotExpr:
		child := visitBoolExpr(e.child, visitor)

		return visitor.VisitNot(child)
	case UnboundPredicate:
		return visitor.VisitUnbound(e)
	case BoundPredicate:
		return visitor.VisitBound(e)
	}
	panic(fmt.Errorf("%w: VisitBooleanExpression type %s", ErrNotImplemented, e))
}

func visitEvaluator(e BooleanExpression, visitor BooleanExprVisitor[bool]) bool {
	switch e := e.(type) {
	case AlwaysFalse:
		return visitor.VisitFalse()
	case AlwaysTrue:
		return visitor.VisitTrue()
	case AndExpr:
		left := visitEvaluator(e.left, visitor)
		if !left {
			return visitor.VisitFalse()
		}

		return visitor.VisitAnd(left, visitEvaluator(e.right, visitor))
	case OrExpr:
		left := visitEvaluator(e.left, visitor)
		if left {
			return visitor.VisitTrue()
		}

		return visitor.VisitOr(left, visitEvaluator(e.right, visitor))
	case NotExpr:
		return visitor.VisitNot(visitEvaluator(e.child, visitor))
	case UnboundPredicate:
		return visitor.VisitUnbound(e)
	case BoundPredicate:
		return visitor.VisitBound(e)
	}
	panic(fmt.Errorf("%w: VisitBooleanExpression type %s", ErrNotImplemented, e))
}

// VisitBoundPredicate uses a BoundBooleanExprVisitor to call the appropriate method
// based on the type of operation in the predicate. This is a convenience function
// for implementing the VisitBound method of a BoundBooleanExprVisitor by simply calling
// iceberg.VisitBoundPredicate(pred, this).
//
// If the predicate is a geospatial bbox op (OpBBoxIntersects/OpBBoxNotIntersects)
// and the visitor does not also implement BoundGeospatialExprVisitor[T], this
// panics with an error wrapping ErrNotImplemented. When reached through VisitExpr
// that panic is recovered into a returned error; a caller invoking this directly
// must implement the extension or recover the panic itself.
// Set predicates are dispatched with detached literal sets. Trusted built-in
// visitors that need the zero-copy path can use [VisitBoundPredicateRef].
func VisitBoundPredicate[T any](e BoundPredicate, visitor BoundBooleanExprVisitor[T]) T {
	return visitBoundPredicate(e, visitor, false)
}

// VisitBoundPredicateRef is the zero-copy dispatch path for trusted visitors
// inside this module. It is gated by an internal token so external visitors
// use VisitBoundPredicate and receive a detached set.
func VisitBoundPredicateRef[T any](e BoundPredicate, visitor BoundBooleanExprVisitor[T], _ internal.BoundPredicateRef) T {
	return visitBoundPredicate(e, visitor, true)
}

func visitBoundPredicate[T any](e BoundPredicate, visitor BoundBooleanExprVisitor[T], borrowed bool) T {
	switch e.Op() {
	case OpIn:
		literals := literalSetForVisit(e, borrowed)
		if borrowed {
			if extrema, ok := e.(boundSetExtremaRef); ok {
				minLit, maxLit, hasExtrema := extrema.boundSetExtremaRef()
				if hasExtrema {
					if extremaVisitor, ok := visitor.(boundSetExtremaExprVisitor[T]); ok {
						return extremaVisitor.VisitInWithExtrema(e.Term(), literals, minLit, maxLit)
					}
				}
			}
		}

		return visitor.VisitIn(e.Term(), literals)
	case OpNotIn:
		return visitor.VisitNotIn(e.Term(), literalSetForVisit(e, borrowed))
	case OpIsNan:
		return visitor.VisitIsNan(e.Term())
	case OpNotNan:
		return visitor.VisitNotNan(e.Term())
	case OpIsNull:
		return visitor.VisitIsNull(e.Term())
	case OpNotNull:
		return visitor.VisitNotNull(e.Term())
	case OpEQ:
		return visitor.VisitEqual(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpNEQ:
		return visitor.VisitNotEqual(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpGTEQ:
		return visitor.VisitGreaterEqual(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpGT:
		return visitor.VisitGreater(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpLTEQ:
		return visitor.VisitLessEqual(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpLT:
		return visitor.VisitLess(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpStartsWith:
		return visitor.VisitStartsWith(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpNotStartsWith:
		return visitor.VisitNotStartsWith(e.Term(), e.(BoundLiteralPredicate).Literal())
	case OpBBoxIntersects, OpBBoxNotIntersects:
		// Geospatial predicates are dispatched through the optional
		// BoundGeospatialExprVisitor extension so visitors that predate (or don't
		// care about) geo support are not forced to implement them. A visitor that
		// doesn't implement it doesn't support geo predicates; panic here is
		// recovered into an error by VisitExpr.
		gv, ok := visitor.(BoundGeospatialExprVisitor[T])
		if !ok {
			panic(fmt.Errorf("%w: visitor %T does not support geospatial bbox predicates",
				ErrNotImplemented, visitor))
		}
		bbox := e.(BoundBBoxPredicate).BBox()
		if e.Op() == OpBBoxIntersects {
			return gv.VisitBBoxIntersects(e.Term(), bbox)
		}

		return gv.VisitBBoxNotIntersects(e.Term(), bbox)
	}
	panic(fmt.Errorf("%w: unhandled bound predicate type: %s", ErrNotImplemented, e))
}

func literalSetForVisit(predicate BoundPredicate, borrowed bool) Set[Literal] {
	setPredicate, ok := predicate.(BoundSetPredicate)
	if !ok {
		panic(fmt.Errorf("%w: %s predicate %T does not implement BoundSetPredicate",
			ErrNotImplemented, predicate.Op(), predicate))
	}
	if borrowed {
		return boundSetLiteralsForVisit(predicate)
	}

	return setPredicate.Literals()
}

// BindExpr recursively binds each portion of an expression using the provided schema.
// Because the expression can end up being simplified to just AlwaysTrue/AlwaysFalse,
// this returns a BooleanExpression.
func BindExpr(s *Schema, expr BooleanExpression, caseSensitive bool) (BooleanExpression, error) {
	return VisitExpr(expr, &bindVisitor{schema: s, caseSensitive: caseSensitive})
}

type bindVisitor struct {
	schema        *Schema
	caseSensitive bool
}

func (*bindVisitor) VisitTrue() BooleanExpression  { return AlwaysTrue{} }
func (*bindVisitor) VisitFalse() BooleanExpression { return AlwaysFalse{} }
func (*bindVisitor) VisitNot(child BooleanExpression) BooleanExpression {
	return NewNot(child)
}

func (*bindVisitor) VisitAnd(left, right BooleanExpression) BooleanExpression {
	return NewAnd(left, right)
}

func (*bindVisitor) VisitOr(left, right BooleanExpression) BooleanExpression {
	return NewOr(left, right)
}

func (b *bindVisitor) VisitUnbound(pred UnboundPredicate) BooleanExpression {
	expr, err := pred.Bind(b.schema, b.caseSensitive)
	if err != nil {
		panic(err)
	}

	return expr
}

func (*bindVisitor) VisitBound(pred BoundPredicate) BooleanExpression {
	panic(fmt.Errorf("%w: found already bound predicate: %s", ErrInvalidArgument, pred))
}

// ExpressionEvaluator returns a function which can be used to evaluate a given expression
// as long as a structlike value is passed which operates like and matches the passed in
// schema.
func ExpressionEvaluator(s *Schema, unbound BooleanExpression, caseSensitive bool) (func(StructLike) (bool, error), error) {
	bound, err := BindExpr(s, unbound, caseSensitive)
	if err != nil {
		return nil, err
	}

	return (&exprEvaluator{bound: bound}).Eval, nil
}

type exprEvaluator struct {
	bound BooleanExpression
	st    StructLike
}

func (e *exprEvaluator) Eval(st StructLike) (bool, error) {
	// Use a per-call evaluator so concurrent callers (e.g. parallel
	// manifest scans in table.Scan.collectManifestEntries) do not race
	// on the shared `st` field. Mirrors the fix in inclusiveMetricsEval
	// from #445.
	ev := exprEvaluator{bound: e.bound, st: st}

	return VisitExprEvaluator(ev.bound, &ev)
}

func (e *exprEvaluator) VisitUnbound(UnboundPredicate) bool {
	panic("found unbound predicate when evaluating expression")
}

func (e *exprEvaluator) VisitBound(pred BoundPredicate) bool {
	return visitBoundPredicate(pred, e, true)
}

func (*exprEvaluator) VisitTrue() bool                { return true }
func (*exprEvaluator) VisitFalse() bool               { return false }
func (*exprEvaluator) VisitNot(child bool) bool       { return !child }
func (*exprEvaluator) VisitAnd(left, right bool) bool { return left && right }
func (*exprEvaluator) VisitOr(left, right bool) bool  { return left || right }

func (e *exprEvaluator) VisitIn(term BoundTerm, literals Set[Literal]) bool {
	v := term.evalToLiteral(e.st)
	if !v.Valid {
		return false
	}

	return literals.Contains(v.Val)
}

func (e *exprEvaluator) VisitNotIn(term BoundTerm, literals Set[Literal]) bool {
	return !e.VisitIn(term, literals)
}

func (e *exprEvaluator) VisitIsNan(term BoundTerm) bool {
	switch term.Type().(type) {
	case Float32Type:
		v := typedTermEval[float32](e.st, term)
		if !v.Valid {
			break
		}

		return math.IsNaN(float64(v.Val))
	case Float64Type:
		v := typedTermEval[float64](e.st, term)
		if !v.Valid {
			break
		}

		return math.IsNaN(v.Val)
	}

	return false
}

func (e *exprEvaluator) VisitNotNan(term BoundTerm) bool {
	return !e.VisitIsNan(term)
}

func (e *exprEvaluator) VisitIsNull(term BoundTerm) bool {
	return term.evalIsNull(e.st)
}

func (e *exprEvaluator) VisitNotNull(term BoundTerm) bool {
	return !term.evalIsNull(e.st)
}

func nullsFirstCmp[T LiteralType](cmp Comparator[T], v1, v2 Optional[T]) int {
	if !v1.Valid {
		if !v2.Valid {
			// both are null
			return 0
		}
		// v1 is null, v2 is not
		return -1
	}

	if !v2.Valid {
		return 1
	}

	return cmp(v1.Val, v2.Val)
}

func typedTermEval[T LiteralType](st StructLike, term BoundTerm) Optional[T] {
	v := term.evalToLiteral(st)
	if !v.Valid {
		return Optional[T]{}
	}

	lit, ok := v.Val.(TypedLiteral[T])
	if !ok {
		panic(fmt.Errorf("%w: term %s evaluated to literal type %s, expected %s",
			ErrType, term, v.Val.Type(), NewLiteral(*new(T)).Type()))
	}

	return Optional[T]{Valid: true, Val: lit.Value()}
}

func typedCmp[T LiteralType](st StructLike, term BoundTerm, lit Literal) int {
	v := typedTermEval[T](st, term)
	var l Optional[T]

	rhs := lit.(TypedLiteral[T])
	if lit != nil {
		l.Valid = true
		l.Val = rhs.Value()
	}

	return nullsFirstCmp(rhs.Comparator(), v, l)
}

func doCmp(st StructLike, term BoundTerm, lit Literal) int {
	// we already properly casted and converted everything during binding
	// so we can type assert based on the term type
	switch term.Type().(type) {
	case BooleanType:
		return typedCmp[bool](st, term, lit)
	case Int32Type:
		return typedCmp[int32](st, term, lit)
	case Int64Type:
		return typedCmp[int64](st, term, lit)
	case Float32Type:
		return typedCmp[float32](st, term, lit)
	case Float64Type:
		return typedCmp[float64](st, term, lit)
	case DateType:
		return typedCmp[Date](st, term, lit)
	case TimeType:
		return typedCmp[Time](st, term, lit)
	case TimestampType, TimestampTzType:
		return typedCmp[Timestamp](st, term, lit)
	case TimestampNsType, TimestampTzNsType:
		return typedCmp[TimestampNano](st, term, lit)
	case BinaryType, FixedType, GeographyType, GeometryType:
		return typedCmp[[]byte](st, term, lit)
	case StringType:
		return typedCmp[string](st, term, lit)
	case UUIDType:
		return typedCmp[uuid.UUID](st, term, lit)
	case DecimalType:
		return typedCmp[Decimal](st, term, lit)
	case VariantType:
		panic("variant values are not comparable")
	}
	panic(ErrType)
}

func (e *exprEvaluator) VisitEqual(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) == 0
}

func (e *exprEvaluator) VisitNotEqual(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) != 0
}

func (e *exprEvaluator) VisitGreater(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) > 0
}

func (e *exprEvaluator) VisitGreaterEqual(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) >= 0
}

func (e *exprEvaluator) VisitLess(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) < 0
}

func (e *exprEvaluator) VisitLessEqual(term BoundTerm, lit Literal) bool {
	return doCmp(e.st, term, lit) <= 0
}

func (e *exprEvaluator) VisitStartsWith(term BoundTerm, lit Literal) bool {
	var value, prefix string

	switch lit := lit.(type) {
	case TypedLiteral[string]:
		val := typedTermEval[string](e.st, term)
		if !val.Valid {
			return false
		}
		prefix, value = lit.Value(), val.Val
	case TypedLiteral[[]byte]:
		val := typedTermEval[[]byte](e.st, term)
		if !val.Valid {
			return false
		}
		prefix, value = string(lit.Value()), string(val.Val)
	}

	return strings.HasPrefix(value, prefix)
}

func (e *exprEvaluator) VisitNotStartsWith(term BoundTerm, lit Literal) bool {
	return !e.VisitStartsWith(term, lit)
}

// VisitBBoxIntersects evaluates a geospatial bbox predicate at the row level.
// Row-level geometric evaluation (parsing each row's WKB and testing its
// bounding box) is a query-engine concern - the Iceberg spec only requires
// bbox-based data skipping, which happens in the metrics/manifest evaluators.
// Here the predicate conservatively keeps every row so it never drops a row that
// an engine's own ST_Intersects would retain.
func (e *exprEvaluator) VisitBBoxIntersects(BoundTerm, BoundingBox) bool {
	return true
}

func (e *exprEvaluator) VisitBBoxNotIntersects(BoundTerm, BoundingBox) bool {
	// Keep every row, for the same reason as VisitBBoxIntersects above: row-level
	// geometric evaluation is a query-engine concern, and bounds alone cannot
	// prove a row does not intersect. Must stay true - returning false here would
	// silently drop rows an engine's own predicate would retain.
	return true
}

// exprEvaluator handles geospatial predicates, so it implements the optional
// BoundGeospatialExprVisitor. The assertion keeps that wiring compile-checked now
// that the geo methods are no longer part of BoundBooleanExprVisitor.
var _ BoundGeospatialExprVisitor[bool] = (*exprEvaluator)(nil)

// RewriteNotExpr rewrites a boolean expression to remove "Not" nodes from the expression
// tree. This is because Projections assume there are no "not" nodes.
//
// Not nodes will be replaced with simply calling `Negate` on the child in the tree.
func RewriteNotExpr(expr BooleanExpression) (BooleanExpression, error) {
	return VisitExpr(expr, rewriteNotVisitor{})
}

type rewriteNotVisitor struct{}

func (rewriteNotVisitor) VisitTrue() BooleanExpression  { return AlwaysTrue{} }
func (rewriteNotVisitor) VisitFalse() BooleanExpression { return AlwaysFalse{} }
func (rewriteNotVisitor) VisitNot(child BooleanExpression) BooleanExpression {
	return child.Negate()
}

func (rewriteNotVisitor) VisitAnd(left, right BooleanExpression) BooleanExpression {
	return NewAnd(left, right)
}

func (rewriteNotVisitor) VisitOr(left, right BooleanExpression) BooleanExpression {
	return NewOr(left, right)
}

func (rewriteNotVisitor) VisitUnbound(pred UnboundPredicate) BooleanExpression {
	return pred
}

func (rewriteNotVisitor) VisitBound(pred BoundPredicate) BooleanExpression {
	return pred
}

// ExtractFieldIDs returns a slice containing the field IDs which are referenced
// by any terms in the given expression. This enables retrieving exactly which
// fields are needed for an expression.
func ExtractFieldIDs(expr BooleanExpression) ([]int, error) {
	res, err := VisitExpr(expr, expressionFieldIDs{})
	if err != nil {
		return nil, err
	}

	out := make([]int, 0, len(res))

	return slices.AppendSeq(out, maps.Keys(res)), nil
}

type expressionFieldIDs struct{}

func (expressionFieldIDs) VisitTrue() map[int]struct{} {
	return map[int]struct{}{}
}

func (expressionFieldIDs) VisitFalse() map[int]struct{} {
	return map[int]struct{}{}
}

func (expressionFieldIDs) VisitNot(child map[int]struct{}) map[int]struct{} {
	return child
}

func (expressionFieldIDs) VisitAnd(left, right map[int]struct{}) map[int]struct{} {
	maps.Insert(left, maps.All(right))

	return left
}

func (expressionFieldIDs) VisitOr(left, right map[int]struct{}) map[int]struct{} {
	maps.Insert(left, maps.All(right))

	return left
}

func (expressionFieldIDs) VisitUnbound(UnboundPredicate) map[int]struct{} {
	panic("expression field IDs only works for bound expressions")
}

func (expressionFieldIDs) VisitBound(pred BoundPredicate) map[int]struct{} {
	return map[int]struct{}{
		pred.Ref().Field().ID: {},
	}
}

// TranslateColumnNames converts the names of columns in an expression by looking up
// the field IDs in the file schema. If columns don't exist they are replaced with
// AlwaysFalse or AlwaysTrue depending on the operator.
func TranslateColumnNames(expr BooleanExpression, fileSchema *Schema) (BooleanExpression, error) {
	res, extracts, err := TranslateColumnNamesForScan(expr, fileSchema)
	if err != nil {
		return nil, err
	}
	if len(extracts) > 0 {
		return nil, fmt.Errorf("%w: variant extract terms require TranslateColumnNamesForScan", ErrNotImplemented)
	}

	return res, nil
}

// VariantExtractColumn describes a synthetic column that materializes one variant extract term for filtering.
type VariantExtractColumn struct {
	Term       BoundExtract
	FieldID    int
	Name       string
	SourcePath []string
}

// TranslateColumnNamesForScan translates a bound filter to the file schema, mapping variant extract terms to synthetic reference columns.
func TranslateColumnNamesForScan(expr BooleanExpression, fileSchema *Schema) (BooleanExpression, []VariantExtractColumn, error) {
	tr := &scanTranslator{
		fileSchema: fileSchema,
		nextID:     fileSchema.HighestFieldID() + 1,
		byKey:      map[string]int{},
	}

	res, err := VisitExpr(expr, tr)
	if err != nil {
		return nil, nil, err
	}

	return res, tr.columns, nil
}

type scanTranslator struct {
	fileSchema *Schema
	nextID     int
	nameSeq    int
	byKey      map[string]int
	columns    []VariantExtractColumn
}

type defaultValueStruct []any

func (s defaultValueStruct) Size() int            { return len(s) }
func (s defaultValueStruct) Get(pos int) any      { return s[pos] }
func (s defaultValueStruct) Set(pos int, val any) { s[pos] = val }

func (*scanTranslator) VisitTrue() BooleanExpression  { return AlwaysTrue{} }
func (*scanTranslator) VisitFalse() BooleanExpression { return AlwaysFalse{} }
func (*scanTranslator) VisitNot(child BooleanExpression) BooleanExpression {
	return NewNot(child)
}

func (*scanTranslator) VisitAnd(left, right BooleanExpression) BooleanExpression {
	return NewAnd(left, right)
}

func (*scanTranslator) VisitOr(left, right BooleanExpression) BooleanExpression {
	return NewOr(left, right)
}

func (*scanTranslator) VisitUnbound(pred UnboundPredicate) BooleanExpression {
	panic(fmt.Errorf("%w: expected bound predicate, got: %s", ErrInvalidArgument, pred.Term()))
}

func (t *scanTranslator) extractRef(ext BoundExtract) Reference {
	key := ext.String()
	idx, ok := t.byKey[key]
	if !ok {
		sourcePath := t.fileSchema.columnPathSegments(ext.Ref().Field().ID)
		idx = len(t.columns)
		t.columns = append(t.columns, VariantExtractColumn{
			Term:       ext,
			FieldID:    t.nextID,
			Name:       t.syntheticName(),
			SourcePath: sourcePath,
		})
		t.nextID++
		t.byKey[key] = idx
	}

	return Reference(t.columns[idx].Name)
}

// syntheticName returns a derived-column name absent from the file schema.
func (t *scanTranslator) syntheticName() string {
	for {
		name := fmt.Sprintf("_variant_extract_%d", t.nameSeq)
		t.nameSeq++
		if _, found := t.fileSchema.FindFieldByName(name); !found {
			return name
		}
	}
}

func unbindPredicate(pred BoundPredicate, ref Reference) UnboundPredicate {
	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return p.AsUnbound(ref)
	case BoundLiteralPredicate:
		return p.AsUnbound(ref, p.Literal())
	case BoundSetPredicate:
		return p.AsUnbound(ref, boundSetLiteralsForVisit(p).Members())
	default:
		panic(fmt.Errorf("%w: unsupported predicate: %s", ErrNotImplemented, pred))
	}
}

func initialDefaultLiteral(field NestedField) (Literal, error) {
	switch typ := field.Type.(type) {
	case BinaryType, FixedType:
		if val, ok := field.InitialDefault.([]byte); ok {
			return BinaryLiteral(val).To(field.Type)
		}
		// Metadata defaults are spec hex strings. The shared decoder also accepts
		// legacy base64 written by iceberg-go v0.6.0.
		if val, ok := field.InitialDefault.(string); ok {
			fixedLen := -1
			if fixed, ok := typ.(FixedType); ok {
				fixedLen = fixed.Len()
			}
			decoded, err := internal.DecodeDefaultBytes(val, fixedLen)
			if err != nil {
				return nil, err
			}

			return BinaryLiteral(decoded).To(field.Type)
		}
	case DecimalType:
		if val, ok := field.InitialDefault.(Decimal); ok {
			return DecimalLiteral(val).To(field.Type)
		}
	}

	data, err := json.Marshal(field.InitialDefault)
	if err != nil {
		return nil, err
	}

	return decodeValue(data, field.Type)
}

func (t *scanTranslator) VisitBound(pred BoundPredicate) BooleanExpression {
	// A bbox predicate has no substrait/record-filter form; it is evaluated only
	// during metrics-based file pruning, so the record filter must conservatively
	// keep every row.
	if _, ok := pred.(BoundBBoxPredicate); ok {
		return AlwaysTrue{}
	}

	var ref Reference
	if ext, ok := pred.Term().(BoundExtract); ok {
		// If the variant column is absent from this file, the extracted path is null
		// for every row: IsNull matches, everything else cannot. Handle it here rather
		// than fabricating an all-null derived column downstream.
		if _, found := t.fileSchema.FindColumnName(ext.Ref().Field().ID); !found {
			if pred.Op() == OpIsNull {
				return AlwaysTrue{}
			}

			return AlwaysFalse{}
		}
		ref = t.extractRef(ext)
	} else {
		fileColName, found := t.fileSchema.FindColumnName(pred.Term().Ref().Field().ID)
		if !found {
			// in the case of schema evolution, the column might not be present
			// in the file schema when reading older data
			field := pred.Ref().Field()
			// A nested field can still be null when an optional parent is null, so
			// its default is not a file-wide constant. Preserve the existing
			// missing-column behavior until translation has row-level parent state.
			if field.InitialDefault == nil || len(pred.Ref().PosPath()) > 1 {
				if pred.Op() == OpIsNull {
					return AlwaysTrue{}
				}

				return AlwaysFalse{}
			}
			// The spec requires geo defaults to be null, but fail open for metadata
			// written by non-conforming V3 writers rather than aborting the scan.
			switch field.Type.(type) {
			case GeometryType, GeographyType:
				return AlwaysTrue{}
			}

			withContext := func(err error) error {
				return fmt.Errorf("initial-default for column %q (id %d): %w",
					field.Name, field.ID, err)
			}
			eval, err := ExpressionEvaluator(NewSchema(0, field),
				unbindPredicate(pred, Reference(field.Name)), true)
			if err != nil {
				panic(withContext(err))
			}

			lit, err := initialDefaultLiteral(field)
			if err != nil {
				panic(withContext(err))
			}

			matches, err := eval(defaultValueStruct{lit.Any()})
			if err != nil {
				panic(withContext(err))
			}
			if matches {
				return AlwaysTrue{}
			}

			return AlwaysFalse{}
		}
		ref = Reference(fileColName)
	}

	return unbindPredicate(pred, ref)
}

// sanitizedLiteralMask is the placeholder substituted for every literal value in
// a sanitized expression. It carries no information about the original value.
const sanitizedLiteralMask = "(redacted)"

// SanitizeExpression returns a copy of expr with every predicate literal replaced
// by an opaque placeholder while preserving the boolean structure, column
// references, and operations. It lets a filter be emitted somewhere untrusted
// (e.g. a metrics ScanReport shipped to a REST sink) without leaking the literal
// values a user scanned with, mirroring the intent of Java's ExpressionUtil.sanitize.
//
// The only guarantee is that no user value leaks: the exact placeholder text is
// not part of the contract and may change. In particular, unlike Java — which
// substitutes type-shaped placeholders (e.g. "(2-digit-int)") — every literal is
// masked with the same marker, since the goal is to withhold the values, not to
// preserve their shape or stay structurally comparable across clients. Set
// predicates (IN / NOT IN) keep their arity so the operation is not
// misrepresented, but the members are masked.
func SanitizeExpression(expr BooleanExpression) (BooleanExpression, error) {
	res, err := VisitExpr(expr, sanitizeVisitor{})
	if err != nil {
		return nil, err
	}

	return res.expr, nil
}

// sanitizedResult is the masked expression plus whether its subtree held a non-serializable term.
type sanitizedResult struct {
	expr              BooleanExpression
	hasUnserializable bool
}

type sanitizeVisitor struct{}

func (sanitizeVisitor) VisitTrue() sanitizedResult  { return sanitizedResult{expr: AlwaysTrue{}} }
func (sanitizeVisitor) VisitFalse() sanitizedResult { return sanitizedResult{expr: AlwaysFalse{}} }
func (sanitizeVisitor) VisitNot(child sanitizedResult) sanitizedResult {
	// Over-broad but never AlwaysFalse: a NOT over any unserializable descendant collapses the whole subtree, dropping serializable siblings from the report.
	if child.hasUnserializable {
		return sanitizedResult{expr: AlwaysTrue{}, hasUnserializable: true}
	}

	return sanitizedResult{expr: NewNot(child.expr)}
}

func (sanitizeVisitor) VisitAnd(left, right sanitizedResult) sanitizedResult {
	return sanitizedResult{expr: NewAnd(left.expr, right.expr), hasUnserializable: left.hasUnserializable || right.hasUnserializable}
}

func (sanitizeVisitor) VisitOr(left, right sanitizedResult) sanitizedResult {
	return sanitizedResult{expr: NewOr(left.expr, right.expr), hasUnserializable: left.hasUnserializable || right.hasUnserializable}
}

func (sanitizeVisitor) VisitUnbound(pred UnboundPredicate) sanitizedResult {
	// An extract term has no REST expression-JSON form; keeping it would fail json.Marshal. Collapse it like bbox (mirrors VisitBound).
	if _, ok := pred.Term().(*unboundExtract); ok {
		return sanitizedResult{expr: AlwaysTrue{}, hasUnserializable: true}
	}

	switch p := pred.(type) {
	case *unboundBBoxPredicate:
		// A bbox predicate carries query-box coordinates, not a per-row user
		// literal, and has no REST expression-JSON form (see MarshalJSON).
		// Collapse it to always-true so the surrounding expression still
		// sanitizes and serializes; nothing user-provided leaks.
		return sanitizedResult{expr: AlwaysTrue{}, hasUnserializable: true}
	case *unboundUnaryPredicate:
		// No literal to mask; the op and term are safe to keep as-is.
		return sanitizedResult{expr: pred}
	case *unboundLiteralPredicate:
		return sanitizedResult{expr: sanitizeLiteralPredicate(p.op, p.term)}
	case *unboundSetPredicate:
		return sanitizedResult{expr: sanitizeSetPredicate(p.op, p.term, p.lits.Len())}
	default:
		panic(fmt.Errorf("%w: unsupported predicate: %s", ErrNotImplemented, pred))
	}
}

func (sanitizeVisitor) VisitBound(pred BoundPredicate) sanitizedResult {
	// A bbox predicate carries query-box coordinates, not a per-row user literal,
	// and has no REST expression-JSON form (see MarshalJSON), so it cannot be
	// rebuilt as a reference predicate like the cases below. Collapse it to
	// always-true so the surrounding expression still sanitizes and serializes;
	// nothing user-provided leaks. Matched before ref is taken and on the exported
	// BoundBBoxPredicate interface (mirrors scanTranslator).
	if _, ok := pred.(BoundBBoxPredicate); ok {
		return sanitizedResult{expr: AlwaysTrue{}, hasUnserializable: true}
	}

	// An extract term has no reference form; rebuilding over its Ref() would leak the whole variant column name. Collapse it like bbox.
	if _, ok := pred.Term().(BoundExtract); ok {
		return sanitizedResult{expr: AlwaysTrue{}, hasUnserializable: true}
	}

	// Rebuild over the column name as an unbound reference: the sanitized form is
	// only serialized or logged, and an unbound predicate with a masked string
	// literal serializes without needing the field's type.
	ref := Reference(pred.Term().Ref().Field().Name)
	term := unboundTermForBound(pred.Term(), ref)
	switch p := pred.(type) {
	case BoundUnaryPredicate:
		return sanitizedResult{expr: UnaryPredicate(p.Op(), term)}
	case BoundLiteralPredicate:
		return sanitizedResult{expr: sanitizeLiteralPredicate(p.Op(), term)}
	case BoundSetPredicate:
		return sanitizedResult{expr: sanitizeSetPredicate(p.Op(), term, boundSetLiteralsForVisit(p).Len())}
	default:
		panic(fmt.Errorf("%w: unsupported predicate: %s", ErrNotImplemented, pred))
	}
}

func sanitizeLiteralPredicate(op Operation, term UnboundTerm) BooleanExpression {
	return LiteralPredicate(op, term, NewLiteral(sanitizedLiteralMask))
}

// sanitizeSetPredicate rebuilds a set predicate with count masked members. The
// masks are made distinct so the set does not collapse (which would turn IN into
// EQ); the suffix reveals only the number of values, never a value itself.
func sanitizeSetPredicate(op Operation, term UnboundTerm, count int) BooleanExpression {
	masks := make([]Literal, count)
	for i := range masks {
		masks[i] = NewLiteral(fmt.Sprintf("%s-%d", sanitizedLiteralMask, i+1))
	}

	return SetPredicate(op, term, masks)
}
