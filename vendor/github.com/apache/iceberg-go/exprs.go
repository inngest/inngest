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
	"math"
	"reflect"

	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/google/uuid"
)

//go:generate stringer -type=Operation -linecomment

// Operation is an enum used for constants to define what operation a given
// expression or predicate is going to execute.
type Operation int

const (
	// do not change the order of these enum constants.
	// they are grouped for quick validation of operation type by
	// using <= and >= of the first/last operation in a group

	OpTrue  Operation = iota // True
	OpFalse                  // False
	// unary ops
	OpIsNull  // IsNull
	OpNotNull // NotNull
	OpIsNan   // IsNaN
	OpNotNan  // NotNaN
	// literal ops
	OpLT            // LessThan
	OpLTEQ          // LessThanEqual
	OpGT            // GreaterThan
	OpGTEQ          // GreaterThanEqual
	OpEQ            // Equal
	OpNEQ           // NotEqual
	OpStartsWith    // StartsWith
	OpNotStartsWith // NotStartsWith
	// set ops
	OpIn    // In
	OpNotIn // NotIn
	// boolean ops
	OpNot // Not
	OpAnd // And
	OpOr  // Or
	// geospatial ops. Kept after the boolean ops so the group ranges above
	// (used for quick op-kind validation) are undisturbed. These have their own
	// predicate constructor (BBoxIntersects) rather than belonging to the
	// unary/literal/set groups.
	OpBBoxIntersects    // BBoxIntersects
	OpBBoxNotIntersects // BBoxNotIntersects
)

// Negate returns the inverse operation for a given op
func (op Operation) Negate() Operation {
	switch op {
	case OpIsNull:
		return OpNotNull
	case OpNotNull:
		return OpIsNull
	case OpIsNan:
		return OpNotNan
	case OpNotNan:
		return OpIsNan
	case OpLT:
		return OpGTEQ
	case OpLTEQ:
		return OpGT
	case OpGT:
		return OpLTEQ
	case OpGTEQ:
		return OpLT
	case OpEQ:
		return OpNEQ
	case OpNEQ:
		return OpEQ
	case OpIn:
		return OpNotIn
	case OpNotIn:
		return OpIn
	case OpStartsWith:
		return OpNotStartsWith
	case OpNotStartsWith:
		return OpStartsWith
	case OpBBoxIntersects:
		return OpBBoxNotIntersects
	case OpBBoxNotIntersects:
		return OpBBoxIntersects
	default:
		panic("no negation for operation " + op.String())
	}
}

// FlipLR returns the correct operation to use if the left and right operands
// are flipped.
func (op Operation) FlipLR() Operation {
	switch op {
	case OpLT:
		return OpGT
	case OpLTEQ:
		return OpGTEQ
	case OpGT:
		return OpLT
	case OpGTEQ:
		return OpLTEQ
	case OpAnd:
		return OpAnd
	case OpOr:
		return OpOr
	default:
		panic("no left-right flip for operation: " + op.String())
	}
}

// BooleanExpression represents a full expression which will evaluate to a
// boolean value such as GreaterThan or StartsWith, etc.
type BooleanExpression interface {
	fmt.Stringer
	Op() Operation
	Negate() BooleanExpression
	Equals(BooleanExpression) bool
}

// AlwaysTrue is the boolean expression "True"
type AlwaysTrue struct{}

func (AlwaysTrue) String() string            { return "AlwaysTrue()" }
func (AlwaysTrue) Op() Operation             { return OpTrue }
func (AlwaysTrue) Negate() BooleanExpression { return AlwaysFalse{} }
func (AlwaysTrue) Equals(other BooleanExpression) bool {
	_, ok := other.(AlwaysTrue)

	return ok
}

// AlwaysFalse is the boolean expression "False"
type AlwaysFalse struct{}

func (AlwaysFalse) String() string            { return "AlwaysFalse()" }
func (AlwaysFalse) Op() Operation             { return OpFalse }
func (AlwaysFalse) Negate() BooleanExpression { return AlwaysTrue{} }
func (AlwaysFalse) Equals(other BooleanExpression) bool {
	_, ok := other.(AlwaysFalse)

	return ok
}

type NotExpr struct {
	child BooleanExpression
}

// NewNot creates a BooleanExpression representing a "Not" operation on the given
// argument. It will optimize slightly though:
//
// If the argument is AlwaysTrue or AlwaysFalse, the appropriate inverse expression
// will be returned directly. If the argument is itself a NotExpr, then the child
// will be returned rather than NotExpr(NotExpr(child)).
func NewNot(child BooleanExpression) BooleanExpression {
	if child == nil {
		panic(fmt.Errorf("%w: cannot create NotExpr with nil child",
			ErrInvalidArgument))
	}

	switch t := child.(type) {
	case NotExpr:
		return t.child
	case AlwaysTrue:
		return AlwaysFalse{}
	case AlwaysFalse:
		return AlwaysTrue{}
	}

	return NotExpr{child: child}
}

func (n NotExpr) String() string            { return "Not(child=" + n.child.String() + ")" }
func (NotExpr) Op() Operation               { return OpNot }
func (n NotExpr) Negate() BooleanExpression { return n.child }
func (n NotExpr) Equals(other BooleanExpression) bool {
	rhs, ok := other.(NotExpr)
	if !ok {
		return false
	}

	return n.child.Equals(rhs.child)
}

type AndExpr struct {
	left, right BooleanExpression
}

func newAnd(left, right BooleanExpression) BooleanExpression {
	if left == nil || right == nil {
		panic(fmt.Errorf("%w: cannot construct AndExpr with nil arguments",
			ErrInvalidArgument))
	}

	switch {
	case left == AlwaysFalse{} || right == AlwaysFalse{}:
		return AlwaysFalse{}
	case left == AlwaysTrue{}:
		return right
	case right == AlwaysTrue{}:
		return left
	}

	return AndExpr{left: left, right: right}
}

// NewAnd will construct a new AndExpr, allowing the caller to provide potentially
// more than just two arguments which will be folded to create an appropriate expression
// tree. i.e. NewAnd(a, b, c, d) becomes AndExpr(a, AndExpr(b, AndExpr(c, d)))
//
// Slight optimizations are performed on creation if either argument is AlwaysFalse
// or AlwaysTrue by performing reductions. If any argument is AlwaysFalse, then everything
// will get folded to a return of AlwaysFalse. If an argument is AlwaysTrue, then the other
// argument will be returned directly rather than creating an AndExpr.
//
// Will panic if any argument is nil
func NewAnd(left, right BooleanExpression, addl ...BooleanExpression) BooleanExpression {
	folded := newAnd(left, right)
	for _, a := range addl {
		folded = newAnd(folded, a)
	}

	return folded
}

func (a AndExpr) String() string {
	return "And(left=" + a.left.String() + ", right=" + a.right.String() + ")"
}

func (AndExpr) Op() Operation { return OpAnd }

func (a AndExpr) Equals(other BooleanExpression) bool {
	rhs, ok := other.(AndExpr)
	if !ok {
		return false
	}

	return (a.left.Equals(rhs.left) && a.right.Equals(rhs.right)) ||
		(a.left.Equals(rhs.right) && a.right.Equals(rhs.left))
}

func (a AndExpr) Negate() BooleanExpression {
	return NewOr(a.left.Negate(), a.right.Negate())
}

type OrExpr struct {
	left, right BooleanExpression
}

func newOr(left, right BooleanExpression) BooleanExpression {
	if left == nil || right == nil {
		panic(fmt.Errorf("%w: cannot construct OrExpr with nil arguments",
			ErrInvalidArgument))
	}

	switch {
	case left == AlwaysTrue{} || right == AlwaysTrue{}:
		return AlwaysTrue{}
	case left == AlwaysFalse{}:
		return right
	case right == AlwaysFalse{}:
		return left
	}

	return OrExpr{left: left, right: right}
}

// NewOr will construct a new OrExpr, allowing the caller to provide potentially
// more than just two arguments which will be folded to create an appropriate expression
// tree. i.e. NewOr(a, b, c, d) becomes OrExpr(a, OrExpr(b, OrExpr(c, d)))
//
// Slight optimizations are performed on creation if either argument is AlwaysFalse
// or AlwaysTrue by performing reductions. If any argument is AlwaysTrue, then everything
// will get folded to a return of AlwaysTrue. If an argument is AlwaysFalse, then the other
// argument will be returned directly rather than creating an OrExpr.
//
// Will panic if any argument is nil
func NewOr(left, right BooleanExpression, addl ...BooleanExpression) BooleanExpression {
	folded := newOr(left, right)
	for _, a := range addl {
		folded = newOr(folded, a)
	}

	return folded
}

func (o OrExpr) String() string {
	return "Or(left=" + o.left.String() + ", right=" + o.right.String() + ")"
}

func (OrExpr) Op() Operation { return OpOr }

func (o OrExpr) Equals(other BooleanExpression) bool {
	rhs, ok := other.(OrExpr)
	if !ok {
		return false
	}

	return (o.left.Equals(rhs.left) && o.right.Equals(rhs.right)) ||
		(o.left.Equals(rhs.right) && o.right.Equals(rhs.left))
}

func (o OrExpr) Negate() BooleanExpression {
	return NewAnd(o.left.Negate(), o.right.Negate())
}

// A Term is a simple expression that evaluates to a value
type Term interface {
	fmt.Stringer
	// requiring this method ensures that only types we define can be used
	// as a term.
	isTerm()
}

// UnboundTerm is an expression that evaluates to a value that isn't yet bound
// to a schema, thus it isn't yet known what the type will be.
type UnboundTerm interface {
	Term

	Equals(UnboundTerm) bool
	Bind(schema *Schema, caseSensitive bool) (BoundTerm, error)
}

// BoundTerm is a simple expression (typically a reference) that evaluates to a
// value and has been bound to a schema.
type BoundTerm interface {
	Term

	Equals(BoundTerm) bool
	Ref() BoundReference
	Type() Type

	evalToLiteral(StructLike) Optional[Literal]
	evalIsNull(StructLike) bool
}

// unbound is a generic interface representing something that is not yet bound
// to a particular type.
type unbound[B any] interface {
	Bind(schema *Schema, caseSensitive bool) (B, error)
}

// An UnboundPredicate represents a boolean predicate expression which has not
// yet been bound to a schema. Binding it will produce a BooleanExpression.
//
// BooleanExpression is used for the binding result because we may optimize and
// return AlwaysTrue / AlwaysFalse in some scenarios during binding which are
// not considered to be "Bound" as they do not have a bound Term or Reference.
type UnboundPredicate interface {
	BooleanExpression
	unbound[BooleanExpression]
	Term() UnboundTerm
}

// BoundPredicate is a boolean predicate expression which has been bound to a schema.
// The underlying reference and term can be retrieved from it.
type BoundPredicate interface {
	BooleanExpression
	Ref() BoundReference
	Term() BoundTerm
}

// Reference is a field name not yet bound to a particular field in a schema
type Reference string

func (r Reference) String() string {
	return "Reference(name='" + string(r) + "')"
}

func (Reference) isTerm() {}
func (r Reference) Equals(other UnboundTerm) bool {
	rhs, ok := other.(Reference)
	if !ok {
		return false
	}

	return r == rhs
}

func (r Reference) Bind(s *Schema, caseSensitive bool) (BoundTerm, error) {
	var (
		field NestedField
		found bool
	)

	if caseSensitive {
		field, found = s.FindFieldByName(string(r))
	} else {
		field, found = s.FindFieldByNameCaseInsensitive(string(r))
	}
	if !found {
		return nil, fmt.Errorf("%w: could not bind reference '%s', caseSensitive=%t",
			ErrInvalidSchema, string(r), caseSensitive)
	}

	acc, ok := s.accessorForField(field.ID)
	if !ok {
		return nil, ErrInvalidSchema
	}

	name, ok := s.FindColumnName(field.ID)
	if !ok {
		return nil, ErrInvalidSchema
	}

	return createBoundRef(field, acc, name), nil
}

// BoundReference is a named reference that has been bound to a particular field
// in a given schema.
type BoundReference interface {
	BoundTerm

	Field() NestedField
	Pos() int
	PosPath() []int
}

type boundRef[T LiteralType] struct {
	field    NestedField
	acc      accessor
	fullName string
}

func createBoundRef(field NestedField, acc accessor, name string) BoundReference {
	switch field.Type.(type) {
	case BooleanType:
		return &boundRef[bool]{field: field, acc: acc, fullName: name}
	case Int32Type:
		return &boundRef[int32]{field: field, acc: acc, fullName: name}
	case Int64Type:
		return &boundRef[int64]{field: field, acc: acc, fullName: name}
	case Float32Type:
		return &boundRef[float32]{field: field, acc: acc, fullName: name}
	case Float64Type:
		return &boundRef[float64]{field: field, acc: acc, fullName: name}
	case DateType:
		return &boundRef[Date]{field: field, acc: acc, fullName: name}
	case TimeType:
		return &boundRef[Time]{field: field, acc: acc, fullName: name}
	case TimestampType, TimestampTzType:
		return &boundRef[Timestamp]{field: field, acc: acc, fullName: name}
	case TimestampNsType, TimestampTzNsType:
		return &boundRef[TimestampNano]{field: field, acc: acc, fullName: name}
	case StringType:
		return &boundRef[string]{field: field, acc: acc, fullName: name}
	case FixedType, BinaryType, GeographyType, GeometryType:
		return &boundRef[[]byte]{field: field, acc: acc, fullName: name}
	case DecimalType:
		return &boundRef[Decimal]{field: field, acc: acc, fullName: name}
	case UUIDType:
		return &boundRef[uuid.UUID]{field: field, acc: acc, fullName: name}
	case VariantType:
		return &boundRef[variant.Value]{field: field, acc: acc, fullName: name}
	}
	panic("unhandled bound reference type: " + field.Type.String())
}

func (b *boundRef[T]) Pos() int { return b.acc.pos }

func (b *boundRef[T]) PosPath() []int {
	out, inner := []int{b.acc.pos}, &b.acc
	for inner.inner != nil {
		inner = inner.inner
		out = append(out, inner.pos)
	}

	return out
}

func (*boundRef[T]) isTerm() {}

func (b *boundRef[T]) String() string {
	return fmt.Sprintf("BoundReference(field=%s, accessor=%s)", b.field, &b.acc)
}

func (b *boundRef[T]) Equals(other BoundTerm) bool {
	rhs, ok := other.(*boundRef[T])
	if !ok {
		return false
	}

	return b.field.Equals(rhs.field)
}

func (b *boundRef[T]) Ref() BoundReference { return b }
func (b *boundRef[T]) Field() NestedField  { return b.field }
func (b *boundRef[T]) Type() Type          { return b.field.Type }

// referenceName preserves the canonical schema path without changing the
// leaf field's name (which is part of the bound reference's type metadata).
func (b *boundRef[T]) referenceName() string { return b.fullName }

func (b *boundRef[T]) eval(st StructLike) Optional[T] {
	switch v := b.acc.Get(st).(type) {
	case nil:
		return Optional[T]{}
	case T:
		return Optional[T]{Valid: true, Val: v}
	default:
		var z T
		typ, val := reflect.TypeOf(z), reflect.ValueOf(v)
		if !val.CanConvert(typ) {
			panic(fmt.Errorf("%w: cannot convert value '%+v' to expected type %s",
				ErrInvalidSchema, val.Interface(), typ.String()))
		}

		return Optional[T]{
			Valid: true,
			Val:   val.Convert(typ).Interface().(T),
		}
	}
}

func (b *boundRef[T]) evalToLiteral(st StructLike) Optional[Literal] {
	v := b.eval(st)
	if !v.Valid {
		return Optional[Literal]{}
	}

	lit := NewLiteral(v.Val)
	if !lit.Type().Equals(b.field.Type) {
		lit, _ = lit.To(b.field.Type)
	}

	return Optional[Literal]{Val: lit, Valid: true}
}

func (b *boundRef[T]) evalIsNull(st StructLike) bool {
	v := b.eval(st)

	return !v.Valid
}

// UnaryPredicate creates and returns an unbound predicate for the provided unary operation.
// Will panic if op is not a unary operation.
func UnaryPredicate(op Operation, t UnboundTerm) UnboundPredicate {
	if op < OpIsNull || op > OpNotNan {
		panic(fmt.Errorf("%w: invalid operation for unary predicate: %s",
			ErrInvalidArgument, op))
	}

	if t == nil {
		panic(fmt.Errorf("%w: cannot create unary predicate with nil term",
			ErrInvalidArgument))
	}

	return &unboundUnaryPredicate{op: op, term: t}
}

type unboundUnaryPredicate struct {
	op   Operation
	term UnboundTerm
}

func (up *unboundUnaryPredicate) String() string {
	return fmt.Sprintf("%s(term=%s)", up.op, up.term)
}

func (up *unboundUnaryPredicate) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*unboundUnaryPredicate)
	if !ok {
		return false
	}

	return up.op == rhs.op && up.term.Equals(rhs.term)
}

func (up *unboundUnaryPredicate) Op() Operation { return up.op }
func (up *unboundUnaryPredicate) Negate() BooleanExpression {
	return &unboundUnaryPredicate{op: up.op.Negate(), term: up.term}
}

func (up *unboundUnaryPredicate) Term() UnboundTerm { return up.term }
func (up *unboundUnaryPredicate) Bind(schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	bound, err := up.term.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}
	// fast case optimizations
	// An extract term's Ref() is the variant column, not the addressed sub-path; skip the required-field fast path.
	_, isExtract := bound.(BoundExtract)
	switch up.op {
	case OpIsNull:
		if transform, ok := bound.(*BoundTransform); ok {
			if _, alwaysNull := transform.transform.(VoidTransform); alwaysNull {
				return AlwaysTrue{}, nil
			}
		}
		if ref, ok := bound.(BoundReference); ok && !isExtract &&
			ref.Field().Required && !schema.FieldHasOptionalParent(ref.Field().ID) {
			return AlwaysFalse{}, nil
		}
	case OpNotNull:
		if transform, ok := bound.(*BoundTransform); ok {
			if _, alwaysNull := transform.transform.(VoidTransform); alwaysNull {
				return AlwaysFalse{}, nil
			}
		}
		if ref, ok := bound.(BoundReference); ok && !isExtract &&
			ref.Field().Required && !schema.FieldHasOptionalParent(ref.Field().ID) {
			return AlwaysTrue{}, nil
		}
	case OpIsNan:
		if !bound.Type().Equals(PrimitiveTypes.Float32) && !bound.Type().Equals(PrimitiveTypes.Float64) {
			return AlwaysFalse{}, nil
		}
	case OpNotNan:
		if !bound.Type().Equals(PrimitiveTypes.Float32) && !bound.Type().Equals(PrimitiveTypes.Float64) {
			return AlwaysTrue{}, nil
		}
	}

	return createBoundUnaryPredicate(up.op, bound), nil
}

// BoundUnaryPredicate is a bound predicate expression that has no arguments
type BoundUnaryPredicate interface {
	BoundPredicate

	AsUnbound(Reference) UnboundPredicate
}

func newBoundUnaryPred[T LiteralType](op Operation, term BoundTerm) BoundUnaryPredicate {
	return &boundUnaryPredicate[T]{op: op, term: term}
}

func createBoundUnaryPredicate(op Operation, term BoundTerm) BoundUnaryPredicate {
	switch term.Type().(type) {
	case BooleanType:
		return newBoundUnaryPred[bool](op, term)
	case Int32Type:
		return newBoundUnaryPred[int32](op, term)
	case Int64Type:
		return newBoundUnaryPred[int64](op, term)
	case Float32Type:
		return newBoundUnaryPred[float32](op, term)
	case Float64Type:
		return newBoundUnaryPred[float64](op, term)
	case DateType:
		return newBoundUnaryPred[Date](op, term)
	case TimeType:
		return newBoundUnaryPred[Time](op, term)
	case TimestampType, TimestampTzType:
		return newBoundUnaryPred[Timestamp](op, term)
	case TimestampNsType, TimestampTzNsType:
		return newBoundUnaryPred[TimestampNano](op, term)
	case StringType:
		return newBoundUnaryPred[string](op, term)
	case FixedType, BinaryType, GeographyType, GeometryType:
		return newBoundUnaryPred[[]byte](op, term)
	case DecimalType:
		return newBoundUnaryPred[Decimal](op, term)
	case UUIDType:
		return newBoundUnaryPred[uuid.UUID](op, term)
	case VariantType:
		return newBoundUnaryPred[variant.Value](op, term)
	}
	panic("unhandled bound reference type: " + term.Type().String())
}

type boundUnaryPredicate[T LiteralType] struct {
	op   Operation
	term BoundTerm
}

func (bp *boundUnaryPredicate[T]) AsUnbound(r Reference) UnboundPredicate {
	return &unboundUnaryPredicate{op: bp.op, term: unboundTermForBound(bp.term, r)}
}

func (bp *boundUnaryPredicate[T]) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*boundUnaryPredicate[T])
	if !ok {
		return false
	}

	return bp.op == rhs.op && bp.term.Equals(rhs.term)
}

func (bp *boundUnaryPredicate[T]) Op() Operation { return bp.op }
func (bp *boundUnaryPredicate[T]) Negate() BooleanExpression {
	return &boundUnaryPredicate[T]{op: bp.op.Negate(), term: bp.term}
}

func (bp *boundUnaryPredicate[T]) Term() BoundTerm     { return bp.term }
func (bp *boundUnaryPredicate[T]) Ref() BoundReference { return bp.term.Ref() }
func (bp *boundUnaryPredicate[T]) String() string {
	return fmt.Sprintf("Bound%s(term=%s)", bp.op, bp.term)
}

// LiteralPredicate constructs an unbound predicate for an operation that requires
// a single literal argument, such as LessThan or StartsWith.
//
// Panics if the operation provided is not a valid Literal operation,
// if the term is nil or if the literal is nil.
func LiteralPredicate(op Operation, t UnboundTerm, lit Literal) UnboundPredicate {
	switch {
	case op < OpLT || op > OpNotStartsWith:
		panic(fmt.Errorf("%w: invalid operation for LiteralPredicate: %s",
			ErrInvalidArgument, op))
	case t == nil:
		panic(fmt.Errorf("%w: cannot create literal predicate with nil term",
			ErrInvalidArgument))
	case lit == nil:
		panic(fmt.Errorf("%w: cannot create literal predicate with nil literal",
			ErrInvalidArgument))
	}

	return &unboundLiteralPredicate{op: op, term: t, lit: lit}
}

type unboundLiteralPredicate struct {
	op   Operation
	term UnboundTerm
	lit  Literal
}

func (ul *unboundLiteralPredicate) String() string {
	return fmt.Sprintf("%s(term=%s, literal=%s)", ul.op, ul.term, ul.lit)
}

func (ul *unboundLiteralPredicate) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*unboundLiteralPredicate)
	if !ok {
		return false
	}

	return ul.op == rhs.op && ul.term.Equals(rhs.term) && ul.lit.Equals(rhs.lit)
}

func (ul *unboundLiteralPredicate) Op() Operation { return ul.op }
func (ul *unboundLiteralPredicate) Negate() BooleanExpression {
	return &unboundLiteralPredicate{op: ul.op.Negate(), term: ul.term, lit: ul.lit}
}
func (ul *unboundLiteralPredicate) Term() UnboundTerm { return ul.term }
func (ul *unboundLiteralPredicate) Bind(schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	bound, err := ul.term.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}
	if (ul.op == OpStartsWith || ul.op == OpNotStartsWith) &&
		(!bound.Type().Equals(PrimitiveTypes.String) && !bound.Type().Equals(PrimitiveTypes.Binary)) {
		return nil, fmt.Errorf("%w: StartsWith and NotStartsWith must bind to String type, not %s",
			ErrType, bound.Type())
	}

	lit, err := ul.lit.To(bound.Type())
	if err != nil {
		return nil, err
	}

	switch lit.(type) {
	case AboveMaxLiteral:
		switch ul.op {
		case OpLT, OpLTEQ, OpNEQ:
			return AlwaysTrue{}, nil
		case OpGT, OpGTEQ, OpEQ:
			return AlwaysFalse{}, nil
		}
	case BelowMinLiteral:
		switch ul.op {
		case OpLT, OpLTEQ, OpEQ:
			return AlwaysFalse{}, nil
		case OpGT, OpGTEQ, OpNEQ:
			return AlwaysTrue{}, nil
		}
	}

	return createBoundLiteralPredicate(ul.op, bound, lit)
}

// BoundLiteralPredicate represents a bound boolean expression that utilizes a single
// literal as an argument, such as Equals or StartsWith.
type BoundLiteralPredicate interface {
	BoundPredicate

	Literal() Literal
	AsUnbound(Reference, Literal) UnboundPredicate
}

func newBoundLiteralPredicate[T LiteralType](op Operation, term BoundTerm, lit Literal) BoundPredicate {
	return &boundLiteralPredicate[T]{
		op: op, term: term,
		lit: lit.(TypedLiteral[T]),
	}
}

func createBoundLiteralPredicate(op Operation, term BoundTerm, lit Literal) (BoundPredicate, error) {
	finalLit, err := lit.To(term.Type())
	if err != nil {
		return nil, err
	}

	switch term.Type().(type) {
	case BooleanType:
		return newBoundLiteralPredicate[bool](op, term, finalLit), nil
	case Int32Type:
		return newBoundLiteralPredicate[int32](op, term, finalLit), nil
	case Int64Type:
		return newBoundLiteralPredicate[int64](op, term, finalLit), nil
	case Float32Type:
		return newBoundLiteralPredicate[float32](op, term, finalLit), nil
	case Float64Type:
		return newBoundLiteralPredicate[float64](op, term, finalLit), nil
	case DateType:
		return newBoundLiteralPredicate[Date](op, term, finalLit), nil
	case TimeType:
		return newBoundLiteralPredicate[Time](op, term, finalLit), nil
	case TimestampType, TimestampTzType:
		return newBoundLiteralPredicate[Timestamp](op, term, finalLit), nil
	case TimestampNsType, TimestampTzNsType:
		return newBoundLiteralPredicate[TimestampNano](op, term, finalLit), nil
	case StringType:
		return newBoundLiteralPredicate[string](op, term, finalLit), nil
	case FixedType, BinaryType, GeographyType, GeometryType:
		return newBoundLiteralPredicate[[]byte](op, term, finalLit), nil
	case DecimalType:
		return newBoundLiteralPredicate[Decimal](op, term, finalLit), nil
	case UUIDType:
		return newBoundLiteralPredicate[uuid.UUID](op, term, finalLit), nil
	case VariantType:
		return nil, fmt.Errorf("%w: ordered predicates are not supported on variant fields", ErrInvalidArgument)
	}

	return nil, fmt.Errorf("%w: could not create bound literal predicate for term type %s",
		ErrInvalidArgument, term.Type())
}

type boundLiteralPredicate[T LiteralType] struct {
	op   Operation
	term BoundTerm
	lit  TypedLiteral[T]
}

func (blp *boundLiteralPredicate[T]) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*boundLiteralPredicate[T])
	if !ok {
		return false
	}

	return blp.op == rhs.op && blp.term.Equals(rhs.term) && blp.lit.Equals(rhs.lit)
}

func (blp *boundLiteralPredicate[T]) Op() Operation { return blp.op }
func (blp *boundLiteralPredicate[T]) Negate() BooleanExpression {
	return &boundLiteralPredicate[T]{op: blp.op.Negate(), term: blp.term, lit: blp.lit}
}
func (blp *boundLiteralPredicate[T]) Term() BoundTerm     { return blp.term }
func (blp *boundLiteralPredicate[T]) Ref() BoundReference { return blp.term.Ref() }
func (blp *boundLiteralPredicate[T]) String() string {
	return fmt.Sprintf("Bound%s(term=%s, literal=%s)", blp.op, blp.term, blp.lit)
}
func (blp *boundLiteralPredicate[T]) Literal() Literal { return blp.lit }
func (blp *boundLiteralPredicate[T]) AsUnbound(r Reference, l Literal) UnboundPredicate {
	return &unboundLiteralPredicate{op: blp.op, term: unboundTermForBound(blp.term, r), lit: l}
}

// SetPredicate creates a boolean expression representing a predicate that uses a set
// of literals as the argument, like In or NotIn. Duplicate literals will be folded
// into a set, only maintaining the unique literals.
//
// Will panic if op is not a valid Set operation
func SetPredicate(op Operation, t UnboundTerm, lits []Literal) BooleanExpression {
	if op < OpIn || op > OpNotIn {
		panic(fmt.Errorf("%w: invalid operation for SetPredicate: %s",
			ErrInvalidArgument, op))
	}

	if t == nil {
		panic(fmt.Errorf("%w: cannot create set predicate with nil term",
			ErrInvalidArgument))
	}

	switch len(lits) {
	case 0:
		switch op {
		case OpIn:
			return AlwaysFalse{}
		case OpNotIn:
			return AlwaysTrue{}
		}
	case 1:
		switch op {
		case OpIn:
			return LiteralPredicate(OpEQ, t, lits[0])
		case OpNotIn:
			return LiteralPredicate(OpNEQ, t, lits[0])
		}
	}

	return &unboundSetPredicate{op: op, term: t, lits: newLiteralSet(lits...)}
}

type unboundSetPredicate struct {
	op   Operation
	term UnboundTerm
	lits Set[Literal]
}

func (usp *unboundSetPredicate) String() string {
	return fmt.Sprintf("%s(term=%s, {%v})", usp.op, usp.term, usp.lits.Members())
}

func (usp *unboundSetPredicate) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*unboundSetPredicate)
	if !ok {
		return false
	}

	return usp.op == rhs.op && usp.term.Equals(rhs.term) &&
		usp.lits.Equals(rhs.lits)
}

func (usp *unboundSetPredicate) Op() Operation { return usp.op }
func (usp *unboundSetPredicate) Negate() BooleanExpression {
	return &unboundSetPredicate{op: usp.op.Negate(), term: usp.term, lits: usp.lits}
}

func (usp *unboundSetPredicate) Term() UnboundTerm { return usp.term }
func (usp *unboundSetPredicate) Bind(schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	bound, err := usp.term.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}

	return createBoundSetPredicate(usp.op, bound, usp.lits)
}

// BoundSetPredicate is a bound expression that utilizes a set of literals such as In or NotIn
type BoundSetPredicate interface {
	BoundPredicate

	Literals() Set[Literal]
	AsUnbound(Reference, []Literal) UnboundPredicate
}

func createBoundSetPredicate(op Operation, term BoundTerm, lits Set[Literal]) (BooleanExpression, error) {
	boundType := term.Type()

	typedSet := make(literalSet, lits.Len())
	for _, v := range lits.Members() {
		casted, err := v.To(boundType)
		if err != nil {
			return nil, err
		}
		switch casted.(type) {
		case AboveMaxLiteral, BelowMinLiteral:
			continue
		}
		typedSet.Add(cloneBoundLiteral(casted))
	}

	switch typedSet.Len() {
	case 0:
		switch op {
		case OpIn:
			return AlwaysFalse{}, nil
		case OpNotIn:
			return AlwaysTrue{}, nil
		}
	case 1:
		switch op {
		case OpIn:
			return createBoundLiteralPredicate(OpEQ, term, typedSet.Members()[0])
		case OpNotIn:
			return createBoundLiteralPredicate(OpNEQ, term, typedSet.Members()[0])
		}
	}

	switch term.Type().(type) {
	case BooleanType:
		return newBoundSetPredicate[bool](op, term, typedSet), nil
	case Int32Type:
		return newBoundSetPredicate[int32](op, term, typedSet), nil
	case Int64Type:
		return newBoundSetPredicate[int64](op, term, typedSet), nil
	case Float32Type:
		return newBoundSetPredicate[float32](op, term, typedSet), nil
	case Float64Type:
		return newBoundSetPredicate[float64](op, term, typedSet), nil
	case DateType:
		return newBoundSetPredicate[Date](op, term, typedSet), nil
	case TimeType:
		return newBoundSetPredicate[Time](op, term, typedSet), nil
	case TimestampType, TimestampTzType:
		return newBoundSetPredicate[Timestamp](op, term, typedSet), nil
	case TimestampNsType, TimestampTzNsType:
		return newBoundSetPredicate[TimestampNano](op, term, typedSet), nil
	case StringType:
		return newBoundSetPredicate[string](op, term, typedSet), nil
	case BinaryType, FixedType, GeographyType, GeometryType:
		return newBoundSetPredicate[[]byte](op, term, typedSet), nil
	case DecimalType:
		return newBoundSetPredicate[Decimal](op, term, typedSet), nil
	case UUIDType:
		return newBoundSetPredicate[uuid.UUID](op, term, typedSet), nil
	}

	return nil, fmt.Errorf("%w: invalid bound type for set predicate - %s",
		ErrType, term.Type())
}

func newBoundSetPredicate[T LiteralType](op Operation, term BoundTerm, lits literalSet) *boundSetPredicate[T] {
	predicate := &boundSetPredicate[T]{op: op, term: term, lits: lits}
	if op == OpIn {
		predicate.minLiteral, predicate.maxLiteral, predicate.hasExtrema = literalSetExtrema[T](lits)
	}

	return predicate
}

type boundSetPredicate[T LiteralType] struct {
	op         Operation
	term       BoundTerm
	lits       Set[Literal]
	minLiteral Literal
	maxLiteral Literal
	hasExtrema bool
}

func literalSetExtrema[T LiteralType](lits literalSet) (minLit, maxLit Literal, ok bool) {
	var (
		cmp                Comparator[T]
		minValue, maxValue T
		valid              = true
	)

	lits.All(func(lit Literal) bool {
		typed, typeOK := lit.(TypedLiteral[T])
		if !typeOK {
			valid = false

			return false
		}

		value := typed.Value()
		if !ok {
			cmp = typed.Comparator()
			minLit, maxLit = lit, lit
			minValue, maxValue = value, value
			ok = true

			return true
		}

		if cmp(value, minValue) < 0 {
			minLit, minValue = lit, value
		}
		if cmp(value, maxValue) > 0 {
			maxLit, maxValue = lit, value
		}

		return true
	})

	if !valid {
		return nil, nil, false
	}

	return minLit, maxLit, ok
}

func (bsp *boundSetPredicate[T]) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*boundSetPredicate[T])
	if !ok {
		return false
	}

	return bsp.op == rhs.op && bsp.term.Equals(rhs.term) &&
		bsp.lits.Equals(rhs.lits)
}

func (bsp *boundSetPredicate[T]) Op() Operation { return bsp.op }
func (bsp *boundSetPredicate[T]) Negate() BooleanExpression {
	return &boundSetPredicate[T]{
		op:   bsp.op.Negate(),
		term: bsp.term,
		lits: bsp.lits,
	}
}
func (bsp *boundSetPredicate[T]) Term() BoundTerm     { return bsp.term }
func (bsp *boundSetPredicate[T]) Ref() BoundReference { return bsp.term.Ref() }
func (bsp *boundSetPredicate[T]) String() string {
	return fmt.Sprintf("Bound%s(term=%s, {%v})", bsp.op, bsp.term, bsp.lits.Members())
}

func (bsp *boundSetPredicate[T]) AsUnbound(r Reference, lits []Literal) UnboundPredicate {
	litSet := newLiteralSet(lits...)
	if litSet.Len() == 1 {
		switch bsp.op {
		case OpIn:
			return LiteralPredicate(OpEQ, unboundTermForBound(bsp.term, r), lits[0])
		case OpNotIn:
			return LiteralPredicate(OpNEQ, unboundTermForBound(bsp.term, r), lits[0])
		}
	}

	return &unboundSetPredicate{op: bsp.op, term: unboundTermForBound(bsp.term, r), lits: litSet}
}

func (bsp *boundSetPredicate[T]) Literals() Set[Literal] {
	return cloneBoundLiteralSet(bsp.lits)
}

type BoundTransform struct {
	transform Transform
	term      BoundTerm
}

func (*BoundTransform) isTerm() {}
func (b *BoundTransform) String() string {
	return fmt.Sprintf("BoundTransform(transform=%s, term=%s)",
		b.transform, b.term)
}

func (b *BoundTransform) Ref() BoundReference { return b.term.Ref() }
func (b *BoundTransform) Type() Type          { return b.transform.ResultType(b.term.Type()) }

func (b *BoundTransform) Equals(other BoundTerm) bool {
	rhs, ok := other.(*BoundTransform)
	if !ok {
		return false
	}

	return b.transform.Equals(rhs.transform) && b.term.Equals(rhs.term)
}

func (b *BoundTransform) evalToLiteral(st StructLike) Optional[Literal] {
	return b.transform.Apply(b.term.evalToLiteral(st))
}

func (b *BoundTransform) evalIsNull(st StructLike) bool {
	return !b.evalToLiteral(st).Valid
}

func unboundTermForBound(term BoundTerm, ref Reference) UnboundTerm {
	if transform, ok := term.(*BoundTransform); ok {
		return NewUnboundTransform(transform.transform, ref)
	}

	return ref
}

// UnboundTransform is a transform applied to a term, not yet bound to a schema;
// the unbound counterpart of BoundTransform. It's what a transform term in a
// REST expression (e.g. bucket[16](id)) parses into.
type UnboundTransform struct {
	transform Transform
	term      UnboundTerm
}

func NewUnboundTransform(transform Transform, term UnboundTerm) *UnboundTransform {
	return &UnboundTransform{transform: transform, term: term}
}

func (*UnboundTransform) isTerm() {}
func (u *UnboundTransform) String() string {
	if u == nil {
		return "UnboundTransform(<nil>)"
	}

	transform := "<nil>"
	if !isNilTransform(u.transform) {
		transform = u.transform.String()
	}

	return fmt.Sprintf("UnboundTransform(transform=%s, term=%s)", transform, u.term)
}

// Transform returns the transform applied to the term.
func (u *UnboundTransform) Transform() Transform { return u.transform }

func (u *UnboundTransform) Equals(other UnboundTerm) bool {
	rhs, ok := other.(*UnboundTransform)
	if !ok {
		return false
	}
	if u == nil || rhs == nil {
		return u == rhs
	}

	leftNil, rightNil := isNilTransform(u.transform), isNilTransform(rhs.transform)
	if leftNil || rightNil {
		if !leftNil || !rightNil || reflect.TypeOf(u.transform) != reflect.TypeOf(rhs.transform) {
			return false
		}
		if u.term == nil || rhs.term == nil {
			return u.term == nil && rhs.term == nil
		}

		return u.term.Equals(rhs.term)
	}

	return u.transform.Equals(rhs.transform) && u.term.Equals(rhs.term)
}

func (u *UnboundTransform) Bind(schema *Schema, caseSensitive bool) (BoundTerm, error) {
	if isNilTransform(u.transform) {
		return nil, fmt.Errorf("%w: transform cannot be nil", ErrInvalidArgument)
	}
	if isUnknownTransform(u.transform) {
		return nil, fmt.Errorf("%w: unknown transform %s cannot be evaluated", ErrNotImplemented, u.transform)
	}

	ref, ok := u.term.(Reference)
	if !ok {
		return nil, fmt.Errorf("%w: transform terms must wrap a direct reference, got %T",
			ErrInvalidArgument, u.term)
	}

	bound, err := ref.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}
	if !u.transform.CanTransform(bound.Type()) {
		return nil, fmt.Errorf("%w: transform %s cannot be applied to %s",
			ErrInvalidArgument, u.transform, bound.Type())
	}

	return &BoundTransform{transform: u.transform, term: bound}, nil
}

func isNilTransform(transform Transform) bool {
	if transform == nil {
		return true
	}

	value := reflect.ValueOf(transform)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return value.IsNil()
	default:
		return false
	}
}

func isUnknownTransform(transform Transform) bool {
	switch transform.(type) {
	case UnknownTransform, *UnknownTransform:
		return true
	default:
		return false
	}
}

// rejectTransformTerm guards predicates that still do not support transform
// terms, such as bounding-box predicates.
func rejectTransformTerm(term BoundTerm) error {
	if _, ok := term.(*BoundTransform); ok {
		return fmt.Errorf("%w: binding a predicate over a transform term is not supported", ErrNotImplemented)
	}

	return nil
}

// BoundingBox is a planar (XY) query bounding box used by the geospatial
// BBoxIntersects predicate. MinX/MinY are the lower-left corner and MaxX/MaxY
// the upper-right corner (X is longitude/easting, Y is latitude/northing).
//
// The box is two-dimensional: pruning only compares the X and Y extents of a
// geometry column's bounds, which is all the Iceberg spec requires for
// bbox-based data skipping. Z/M extents in a column's bounds are ignored.
//
// Z/M are intentionally omitted for now rather than reserved as fields: XY is
// sufficient for spec-compliant pruning, and adding optional MinZ/MaxZ/MinM/MaxM
// later is source-additive. The forward-compat contract is that Valid and Equals
// are defined over whichever extents the box carries - today X and Y - so adding
// higher dimensions later extends, rather than reinterprets, existing XY boxes.
type BoundingBox struct {
	MinX, MinY, MaxX, MaxY float64
}

func (b BoundingBox) String() string {
	return fmt.Sprintf("BoundingBox(minX=%g, minY=%g, maxX=%g, maxY=%g)",
		b.MinX, b.MinY, b.MaxX, b.MaxY)
}

// Equals reports whether two bounding boxes have identical extents.
func (b BoundingBox) Equals(other BoundingBox) bool {
	return b.MinX == other.MinX && b.MinY == other.MinY &&
		b.MaxX == other.MaxX && b.MaxY == other.MaxY
}

// Valid reports whether the box is well-formed: no coordinate is NaN and the
// minimum of each axis does not exceed its maximum. An infinite bound is allowed
// (an open half-plane). An invalid box would silently mis-prune - a NaN makes
// every intersection test false (pruning matching files), and an inverted box
// (min > max) reports the wrong overlap - so BBoxIntersects rejects one.
func (b BoundingBox) Valid() bool {
	if math.IsNaN(b.MinX) || math.IsNaN(b.MinY) ||
		math.IsNaN(b.MaxX) || math.IsNaN(b.MaxY) {
		return false
	}

	return b.MinX <= b.MaxX && b.MinY <= b.MaxY
}

// isGeoType reports whether t is a geometry or geography type, the only types a
// bbox predicate may bind to.
func isGeoType(t Type) bool {
	switch t.(type) {
	case GeometryType, GeographyType:
		return true
	default:
		return false
	}
}

// BBoxIntersects constructs an unbound geospatial predicate that matches rows
// whose geometry/geography value's bounding box intersects the query box. It is
// used to prune data files whose stored geo bounds cannot overlap the query
// region; the spec only requires bbox-based pruning, so full geometric predicate
// evaluation (ST_Intersects/ST_Within) remains a query-engine concern.
//
// Panics if the term is nil or the bbox is not Valid (NaN coordinate or an
// inverted min/max), either of which would cause silent mis-pruning. Panicking
// on invalid construction is consistent with the other predicate constructors in
// this package (UnaryPredicate, LiteralPredicate, SetPredicate, NewAnd, NewNot),
// which all panic rather than return an error; a caller building a box from
// untrusted input should validate it with BoundingBox.Valid first.
func BBoxIntersects(t UnboundTerm, bbox BoundingBox) UnboundPredicate {
	if t == nil {
		panic(fmt.Errorf("%w: cannot create bbox predicate with nil term",
			ErrInvalidArgument))
	}
	if !bbox.Valid() {
		panic(fmt.Errorf("%w: invalid bounding box %s (NaN coordinate or min > max)",
			ErrInvalidArgument, bbox))
	}

	return &unboundBBoxPredicate{op: OpBBoxIntersects, term: t, bbox: bbox}
}

type unboundBBoxPredicate struct {
	op   Operation
	term UnboundTerm
	bbox BoundingBox
}

func (u *unboundBBoxPredicate) String() string {
	return fmt.Sprintf("%s(term=%s, bbox=%s)", u.op, u.term, u.bbox)
}

func (u *unboundBBoxPredicate) Op() Operation     { return u.op }
func (u *unboundBBoxPredicate) Term() UnboundTerm { return u.term }
func (u *unboundBBoxPredicate) Negate() BooleanExpression {
	return &unboundBBoxPredicate{op: u.op.Negate(), term: u.term, bbox: u.bbox}
}

func (u *unboundBBoxPredicate) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*unboundBBoxPredicate)
	if !ok {
		return false
	}

	return u.op == rhs.op && u.term.Equals(rhs.term) && u.bbox.Equals(rhs.bbox)
}

func (u *unboundBBoxPredicate) Bind(schema *Schema, caseSensitive bool) (BooleanExpression, error) {
	bound, err := u.term.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}
	if err := rejectTransformTerm(bound); err != nil {
		return nil, err
	}

	if !isGeoType(bound.Type()) {
		return nil, fmt.Errorf("%w: BBoxIntersects must bind to a geometry or geography type, not %s",
			ErrType, bound.Type())
	}

	return &boundBBoxPredicate{op: u.op, term: bound, bbox: u.bbox}, nil
}

// BoundBBoxPredicate is a bound geospatial predicate that tests whether a
// geometry/geography column's bounds intersect a query bounding box.
type BoundBBoxPredicate interface {
	BoundPredicate

	BBox() BoundingBox
}

// boundBBoxPredicate carries a bound geo term plus the query box. Unlike the
// other bound predicates it intentionally has no AsUnbound(Reference) method, so
// it does NOT satisfy BoundUnaryPredicate: a bbox predicate must never be rebuilt
// as a generic unary predicate (it would reach substrait and error, or be
// dropped to AlwaysFalse when a column is absent). It has no record-filter or
// REST-JSON form at all, so the two visitors that would otherwise rebuild a bound
// predicate - scanTranslator.VisitBound and sanitizeVisitor.VisitBound -
// special-case *boundBBoxPredicate and collapse it to AlwaysTrue. Data-file
// pruning is done separately by inclusiveMetricsEval.
type boundBBoxPredicate struct {
	op   Operation
	term BoundTerm
	bbox BoundingBox
}

func (b *boundBBoxPredicate) String() string {
	return fmt.Sprintf("Bound%s(term=%s, bbox=%s)", b.op, b.term, b.bbox)
}

func (b *boundBBoxPredicate) Op() Operation       { return b.op }
func (b *boundBBoxPredicate) Term() BoundTerm     { return b.term }
func (b *boundBBoxPredicate) Ref() BoundReference { return b.term.Ref() }
func (b *boundBBoxPredicate) BBox() BoundingBox   { return b.bbox }

func (b *boundBBoxPredicate) Negate() BooleanExpression {
	return &boundBBoxPredicate{op: b.op.Negate(), term: b.term, bbox: b.bbox}
}

func (b *boundBBoxPredicate) Equals(other BooleanExpression) bool {
	rhs, ok := other.(*boundBBoxPredicate)
	if !ok {
		return false
	}

	return b.op == rhs.op && b.term.Equals(rhs.term) && b.bbox.Equals(rhs.bbox)
}
