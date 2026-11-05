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

	"github.com/apache/arrow-go/v18/parquet/variant"
	"github.com/google/uuid"
)

// BoundExtract is a bound variant sub-path term used for metrics pruning and residual evaluation.
type BoundExtract interface {
	BoundTerm

	Path() string
	// ExtractValue navigates v to this term's path and casts the leaf to the target type.
	ExtractValue(v variant.Value) (Literal, bool)
}

// Extract creates an unbound variant sub-path term for a dotted JSONPath.
func Extract(ref Reference, path string, typ PrimitiveType) UnboundTerm {
	return &unboundExtract{ref: ref, path: path, typ: typ}
}

type unboundExtract struct {
	ref  Reference
	path string
	typ  PrimitiveType
}

func (*unboundExtract) isTerm() {}
func (u *unboundExtract) String() string {
	return fmt.Sprintf("extract(%s, path=%s, type=%s)", u.ref, u.path, u.typ)
}

func (u *unboundExtract) Ref() Reference      { return u.ref }
func (u *unboundExtract) Path() string        { return u.path }
func (u *unboundExtract) Type() PrimitiveType { return u.typ }

// MarshalJSON reports that extract terms have no REST expression-JSON form (see ErrExtractNotSerializable), rather than silently encoding to {}.
func (*unboundExtract) MarshalJSON() ([]byte, error) { return nil, ErrExtractNotSerializable }

func (u *unboundExtract) Equals(other UnboundTerm) bool {
	rhs, ok := other.(*unboundExtract)
	if !ok {
		return false
	}

	sameType := (u.typ == nil && rhs.typ == nil) ||
		(u.typ != nil && rhs.typ != nil && u.typ.Equals(rhs.typ))

	return u.ref == rhs.ref && u.path == rhs.path && sameType
}

func (u *unboundExtract) Bind(schema *Schema, caseSensitive bool) (BoundTerm, error) {
	bound, err := u.ref.Bind(schema, caseSensitive)
	if err != nil {
		return nil, err
	}
	if _, ok := bound.Type().(VariantType); !ok {
		return nil, fmt.Errorf("%w: cannot bind extract, not a variant: %s", ErrInvalidArgument, u.ref)
	}
	if u.typ == nil {
		return nil, fmt.Errorf("%w: cannot bind extract, target type is required", ErrInvalidArgument)
	}
	if !isVariantExtractTarget(u.typ) {
		return nil, fmt.Errorf("%w: cannot bind extract, unsupported target type: %s", ErrInvalidArgument, u.typ)
	}

	fields, err := parseVariantPath(u.path)
	if err != nil {
		return nil, err
	}

	acc, ok := schema.accessorForField(bound.Ref().Field().ID)
	if !ok {
		return nil, ErrInvalidSchema
	}

	return createBoundExtract(bound.Ref(), fields, NormalizeVariantPath(fields), u.typ, acc), nil
}

// isVariantExtractTarget reports whether typ is a supported extract target (the set createBoundExtract handles).
func isVariantExtractTarget(typ PrimitiveType) bool {
	switch typ.(type) {
	case BooleanType, Int32Type, Int64Type, Float32Type, Float64Type,
		DateType, TimeType, TimestampType, TimestampTzType, TimestampNsType, TimestampTzNsType,
		StringType, FixedType, BinaryType, DecimalType, UUIDType:
		return true
	}

	return false
}

var _ BoundExtract = (*boundExtract[int32])(nil)

type boundExtract[T LiteralType] struct {
	ref    BoundReference
	fields []string
	path   string
	typ    PrimitiveType
	acc    accessor
}

func createBoundExtract(ref BoundReference, fields []string, path string, typ PrimitiveType, acc accessor) BoundTerm {
	switch typ.(type) {
	case BooleanType:
		return &boundExtract[bool]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case Int32Type:
		return &boundExtract[int32]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case Int64Type:
		return &boundExtract[int64]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case Float32Type:
		return &boundExtract[float32]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case Float64Type:
		return &boundExtract[float64]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case DateType:
		return &boundExtract[Date]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case TimeType:
		return &boundExtract[Time]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case TimestampType, TimestampTzType:
		return &boundExtract[Timestamp]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case TimestampNsType, TimestampTzNsType:
		return &boundExtract[TimestampNano]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case StringType:
		return &boundExtract[string]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case FixedType, BinaryType:
		return &boundExtract[[]byte]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case DecimalType:
		return &boundExtract[Decimal]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	case UUIDType:
		return &boundExtract[uuid.UUID]{ref: ref, fields: fields, path: path, typ: typ, acc: acc}
	}
	panic("unhandled variant extract target type: " + typ.String())
}

func (*boundExtract[T]) isTerm() {}
func (b *boundExtract[T]) String() string {
	return fmt.Sprintf("extract(%s, path=%s, type=%s)", b.ref, b.path, b.typ)
}

func (b *boundExtract[T]) Ref() BoundReference { return b.ref }
func (b *boundExtract[T]) Type() Type          { return b.typ }
func (b *boundExtract[T]) Path() string        { return b.path }

// VariantPath returns the term's member-name path for columnar extraction via compute.VariantGet.
func (b *boundExtract[T]) VariantPath() variant.VariantPath {
	var p variant.VariantPath
	for _, f := range b.fields {
		p = p.Field(f)
	}

	return p
}

// MarshalJSON reports that extract terms have no REST expression-JSON form (see ErrExtractNotSerializable), rather than silently encoding to {}.
func (*boundExtract[T]) MarshalJSON() ([]byte, error) { return nil, ErrExtractNotSerializable }

func (b *boundExtract[T]) Equals(other BoundTerm) bool {
	rhs, ok := other.(*boundExtract[T])
	if !ok {
		return false
	}

	return b.ref.Equals(rhs.ref) && b.path == rhs.path && b.typ.Equals(rhs.typ)
}

// navigateVariant walks the nested variant object by member names, returning the leaf value or invalid if the path is absent.
func navigateVariant(v variant.Value, fields []string) (variant.Value, bool) {
	for _, name := range fields {
		obj, ok := v.Value().(variant.ObjectValue)
		if !ok {
			return variant.Value{}, false
		}

		field, err := obj.ValueByKey(name)
		if err != nil {
			return variant.Value{}, false
		}

		v = field.Value
	}

	return v, true
}

// leafValue navigates the nested variant object from the value stored at this term's reference.
func (b *boundExtract[T]) leafValue(st StructLike) (variant.Value, bool) {
	raw := b.acc.Get(st)
	v, ok := raw.(variant.Value)
	if !ok {
		return variant.Value{}, false
	}

	return navigateVariant(v, b.fields)
}

// ExtractValue navigates v to this term's path and casts the leaf to the target type.
func (b *boundExtract[T]) ExtractValue(v variant.Value) (Literal, bool) {
	leaf, ok := navigateVariant(v, b.fields)
	if !ok {
		return nil, false
	}

	return CastVariantLiteral(leaf, b.typ)
}

func (b *boundExtract[T]) eval(st StructLike) Optional[T] {
	v, ok := b.leafValue(st)
	if !ok {
		return Optional[T]{}
	}

	result, ok := castVariantValue(v, b.typ)
	if !ok {
		return Optional[T]{}
	}

	val, ok := result.(T)
	if !ok {
		return Optional[T]{}
	}

	return Optional[T]{Valid: true, Val: val}
}

func (b *boundExtract[T]) evalToLiteral(st StructLike) Optional[Literal] {
	v := b.eval(st)
	if !v.Valid {
		return Optional[Literal]{}
	}

	lit := NewLiteral(v.Val)
	if !lit.Type().Equals(b.typ) {
		conv, err := lit.To(b.typ)
		if err != nil {
			return Optional[Literal]{}
		}

		lit = conv
	}

	return Optional[Literal]{Val: lit, Valid: true}
}

func (b *boundExtract[T]) evalIsNull(st StructLike) bool {
	return !b.eval(st).Valid
}
