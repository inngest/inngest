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

package substrait

import (
	_ "embed"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow/compute/exprs"
	"github.com/apache/iceberg-go"
	"github.com/substrait-io/substrait-go/v8/expr"
	"github.com/substrait-io/substrait-go/v8/extensions"
	"github.com/substrait-io/substrait-go/v8/types"
)

//go:embed functions_set.yaml
var funcsetYAML string

var (
	collection = extensions.GetDefaultCollectionWithNoError()
	funcSetURI = "extension:apache.iceberg:functions_set"
)

func init() {
	if !collection.URILoaded(funcSetURI) {
		if err := collection.Load(funcSetURI, strings.NewReader(funcsetYAML)); err != nil {
			panic(err)
		}
	}
}

func NewExtensionSet() exprs.ExtensionIDSet {
	return exprs.NewExtensionSetDefault(expr.NewEmptyExtensionRegistry(collection))
}

// ConvertExpr converts a bound expression to a Substrait expression so that it can
// be utilized for computation. Case sensitivity is applied when binding; the third
// argument is retained for API compatibility.
func ConvertExpr(schema *iceberg.Schema, e iceberg.BooleanExpression, _ bool) (*expr.ExtensionRegistry, expr.Expression, error) {
	base, err := ConvertSchema(schema)
	if err != nil {
		return nil, nil, err
	}

	reg := expr.NewEmptyExtensionRegistry(collection)

	bldr := expr.ExprBuilder{Reg: reg, BaseSchema: types.NewRecordTypeFromStruct(base.Struct)}
	b, err := iceberg.VisitExpr(e, &toSubstraitExpr{bldr: bldr})
	if err != nil {
		return nil, nil, err
	}

	out, err := b.BuildExpr()

	return &reg, out, err
}

// ConvertSchema converts an Iceberg schema to a substrait NamedStruct using
// the appropriate types and column names.
func ConvertSchema(schema *iceberg.Schema) (res types.NamedStruct, err error) {
	var typ types.Type

	typ, err = iceberg.Visit(schema, convertToSubstrait{})
	if err != nil {
		return res, err
	}

	val := typ.(*types.StructType)
	res.Struct = *val

	res.Names = make([]string, schema.NumFields())
	for i, f := range schema.Fields() {
		res.Names[i] = f.Name
	}

	return res, err
}

type convertToSubstrait struct{}

func (convertToSubstrait) Schema(_ *iceberg.Schema, result types.Type) types.Type {
	return result.WithNullability(types.NullabilityNullable)
}

func (convertToSubstrait) Struct(_ iceberg.StructType, results []types.Type) types.Type {
	return &types.StructType{
		Nullability: types.NullabilityUnspecified,
		Types:       results,
	}
}

func getNullability(required bool) types.Nullability {
	if required {
		return types.NullabilityRequired
	}

	return types.NullabilityNullable
}

func (convertToSubstrait) Field(field iceberg.NestedField, result types.Type) types.Type {
	return result.WithNullability(getNullability(field.Required))
}

func (c convertToSubstrait) List(list iceberg.ListType, elemResult types.Type) types.Type {
	return &types.ListType{
		Nullability: types.NullabilityUnspecified,
		Type:        c.Field(list.ElementField(), elemResult),
	}
}

func (c convertToSubstrait) Map(m iceberg.MapType, keyResult, valResult types.Type) types.Type {
	return &types.MapType{
		Nullability: types.NullabilityUnspecified,
		Key:         c.Field(m.KeyField(), keyResult),
		Value:       c.Field(m.ValueField(), valResult),
	}
}

func (convertToSubstrait) Primitive(iceberg.PrimitiveType) types.Type { panic("should not be called") }
func (convertToSubstrait) Variant(iceberg.VariantType) types.Type     { panic("should not be called") }

func (convertToSubstrait) VisitFixed(f iceberg.FixedType) types.Type {
	return &types.FixedBinaryType{Length: int32(f.Len())}
}

func (convertToSubstrait) VisitDecimal(d iceberg.DecimalType) types.Type {
	return &types.DecimalType{Precision: int32(d.Precision()), Scale: int32(d.Scale())}
}

func (convertToSubstrait) VisitBoolean() types.Type   { return &types.BooleanType{} }
func (convertToSubstrait) VisitInt32() types.Type     { return &types.Int32Type{} }
func (convertToSubstrait) VisitInt64() types.Type     { return &types.Int64Type{} }
func (convertToSubstrait) VisitFloat32() types.Type   { return &types.Float32Type{} }
func (convertToSubstrait) VisitFloat64() types.Type   { return &types.Float64Type{} }
func (convertToSubstrait) VisitDate() types.Type      { return &types.DateType{} }
func (convertToSubstrait) VisitTime() types.Type      { return &types.TimeType{} }
func (convertToSubstrait) VisitTimestamp() types.Type { return &types.TimestampType{} }
func (convertToSubstrait) VisitTimestampNs() types.Type {
	return &types.PrecisionTimestampType{Precision: types.PrecisionNanoSeconds}
}
func (convertToSubstrait) VisitTimestampTz() types.Type { return &types.TimestampTzType{} }
func (convertToSubstrait) VisitTimestampNsTz() types.Type {
	return &types.PrecisionTimestampTzType{
		PrecisionTimestampType: types.PrecisionTimestampType{Precision: types.PrecisionNanoSeconds},
	}
}
func (convertToSubstrait) VisitString() types.Type { return &types.StringType{} }
func (convertToSubstrait) VisitBinary() types.Type { return &types.BinaryType{} }
func (convertToSubstrait) VisitUUID() types.Type   { return &types.UUIDType{} }
func (convertToSubstrait) VisitUnknown() types.Type {
	// Unknown types cannot be stored in data files and have no Substrait equivalent
	// Returning nil indicates this type cannot be converted to Substrait
	return nil
}

func (convertToSubstrait) VisitGeometry(iceberg.GeometryType) types.Type {
	return &types.BinaryType{}
}

func (convertToSubstrait) VisitGeography(iceberg.GeographyType) types.Type {
	return &types.BinaryType{}
}

func (convertToSubstrait) VisitVariant() types.Type {
	// Variant has no Substrait equivalent today. We return BinaryType as a
	// structural placeholder so that schema conversion and expression binding
	// on sibling columns don't panic with a nil type. A proper user-defined
	// Substrait type will be needed once shredding enables predicate pushdown.
	return &types.BinaryType{}
}

var _ iceberg.SchemaVisitorPerPrimitiveType[types.Type] = (*convertToSubstrait)(nil)

var (
	boolURI    = extensions.SubstraitDefaultURNPrefix + "functions_boolean"
	compareURI = extensions.SubstraitDefaultURNPrefix + "functions_comparison"
	stringURI  = extensions.SubstraitDefaultURNPrefix + "functions_string"

	notID          = extensions.ID{URN: boolURI, Name: "not"}
	andID          = extensions.ID{URN: boolURI, Name: "and"}
	orID           = extensions.ID{URN: boolURI, Name: "or"}
	isNaNID        = extensions.ID{URN: compareURI, Name: "is_nan"}
	isNullID       = extensions.ID{URN: compareURI, Name: "is_null"}
	isNotNullID    = extensions.ID{URN: compareURI, Name: "is_not_null"}
	equalID        = extensions.ID{URN: compareURI, Name: "equal"}
	notEqualID     = extensions.ID{URN: compareURI, Name: "not_equal"}
	greaterEqualID = extensions.ID{URN: compareURI, Name: "gte"}
	greaterID      = extensions.ID{URN: compareURI, Name: "gt"}
	lessEqualID    = extensions.ID{URN: compareURI, Name: "lte"}
	lessID         = extensions.ID{URN: compareURI, Name: "lt"}
	startsWithID   = extensions.ID{URN: stringURI, Name: "starts_with"}
	isInID         = extensions.ID{URN: funcSetURI, Name: "is_in"}
)

type toSubstraitExpr struct {
	bldr expr.ExprBuilder
}

func (t *toSubstraitExpr) VisitTrue() expr.Builder {
	return t.bldr.Wrap(expr.NewLiteral(true, false))
}

func (t *toSubstraitExpr) VisitFalse() expr.Builder {
	return t.bldr.Wrap(expr.NewLiteral(false, false))
}

func (t *toSubstraitExpr) VisitNot(child expr.Builder) expr.Builder {
	return t.bldr.ScalarFunc(notID).Args(child.(expr.FuncArgBuilder))
}

func (t *toSubstraitExpr) VisitAnd(left, right expr.Builder) expr.Builder {
	return t.bldr.ScalarFunc(andID).Args(left.(expr.FuncArgBuilder),
		right.(expr.FuncArgBuilder))
}

func (t *toSubstraitExpr) VisitOr(left, right expr.Builder) expr.Builder {
	return t.bldr.ScalarFunc(orID).Args(left.(expr.FuncArgBuilder),
		right.(expr.FuncArgBuilder))
}

func (t *toSubstraitExpr) VisitUnbound(iceberg.UnboundPredicate) expr.Builder {
	panic("can only convert bound expressions to substrait")
}

func (t *toSubstraitExpr) VisitBound(pred iceberg.BoundPredicate) expr.Builder {
	if _, ok := pred.Term().(*iceberg.BoundTransform); ok {
		panic(fmt.Errorf("%w: transformed terms cannot be converted to substrait", iceberg.ErrNotImplemented))
	}

	return iceberg.VisitBoundPredicate(pred, t)
}

type substraitPrimitiveLiteralTypes interface {
	bool | ~int32 | ~int64 | float32 | float64 | string
}

func toPrimitiveSubstraitLiteral[T substraitPrimitiveLiteralTypes](v T) expr.Literal {
	return expr.NewPrimitiveLiteral(v, false)
}

func toByteSliceSubstraitLiteral[T []byte | types.UUID](v T) expr.Literal {
	return expr.NewByteSliceLiteral(v, false)
}

// toDecimalLiteral converts an iceberg.DecimalLiteral into a Substrait decimal
// literal. Precision is taken from the bound field's iceberg.DecimalType (typ),
// NOT from v.Type(). DecimalLiteral does not carry the originating column's
// precision, so v.Type() always reports a hardcoded precision of 9 (see
// https://github.com/apache/iceberg-go/issues/1028). If typ is not a
// DecimalType, this falls back to v.Type()'s (hardcoded) precision.
func toDecimalLiteral(typ iceberg.Type, v iceberg.DecimalLiteral) expr.Literal {
	precision := int32(9)
	if dt, ok := typ.(iceberg.DecimalType); ok {
		precision = int32(dt.Precision())
	}

	// decimal128.Num is {lo uint64, hi int64} — its raw in-memory bytes on
	// LE hosts are already the 16-byte little-endian two's-complement
	// representation Substrait wants. hostIsLE is a build-tag const, so the
	// compiler drops the unused branch on every known GOARCH.
	n := v.Value().Val
	buf := make([]byte, 16)
	if hostIsLE {
		copy(buf, unsafe.Slice((*byte)(unsafe.Pointer(&n)), 16))
	} else {
		binary.LittleEndian.PutUint64(buf[:8], n.LowBits())
		binary.LittleEndian.PutUint64(buf[8:], uint64(n.HighBits()))
	}

	result, _ := expr.NewLiteral(&types.Decimal{
		Scale:     int32(v.Scale),
		Value:     buf,
		Precision: int32(precision),
	}, false)

	return result
}

func toFixedLiteral(v iceberg.FixedLiteral) expr.Literal {
	return expr.NewFixedBinaryLiteral(types.FixedBinary(v), false)
}

func toSubstraitLiteral(typ iceberg.Type, lit iceberg.Literal) expr.Literal {
	switch lit := lit.(type) {
	case iceberg.BoolLiteral:
		return toPrimitiveSubstraitLiteral(bool(lit))
	case iceberg.Int32Literal:
		return toPrimitiveSubstraitLiteral(int32(lit))
	case iceberg.Int64Literal:
		return toPrimitiveSubstraitLiteral(int64(lit))
	case iceberg.Float32Literal:
		return toPrimitiveSubstraitLiteral(float32(lit))
	case iceberg.Float64Literal:
		return toPrimitiveSubstraitLiteral(float64(lit))
	case iceberg.StringLiteral:
		return toPrimitiveSubstraitLiteral(string(lit))
	case iceberg.TimestampLiteral:
		if typ.Equals(iceberg.PrimitiveTypes.TimestampTz) {
			return toPrimitiveSubstraitLiteral(types.TimestampTz(lit))
		}

		return toPrimitiveSubstraitLiteral(types.Timestamp(lit))
	case iceberg.TimestampNsLiteral:
		if typ.Equals(iceberg.PrimitiveTypes.TimestampTzNs) {
			return expr.NewPrecisionTimestampTzLiteral(
				int64(lit), types.PrecisionNanoSeconds, types.NullabilityRequired,
			)
		}

		return expr.NewPrecisionTimestampLiteral(
			int64(lit), types.PrecisionNanoSeconds, types.NullabilityRequired,
		)
	case iceberg.DateLiteral:
		return toPrimitiveSubstraitLiteral(types.Date(lit))
	case iceberg.TimeLiteral:
		return toPrimitiveSubstraitLiteral(types.Time(lit))
	case iceberg.BinaryLiteral:
		return toByteSliceSubstraitLiteral([]byte(lit))
	case iceberg.FixedLiteral:
		return toFixedLiteral(lit)
	case iceberg.UUIDLiteral:
		return toByteSliceSubstraitLiteral(types.UUID(lit[:]))
	case iceberg.DecimalLiteral:
		return toDecimalLiteral(typ, lit)
	}
	panic(fmt.Errorf("invalid literal type: %s", lit.Type()))
}

func toSubstraitLiteralSet(typ iceberg.Type, lits []iceberg.Literal) expr.ListLiteralValue {
	if len(lits) == 0 {
		return nil
	}

	sort.Slice(lits, func(i, j int) bool {
		return lits[i].String() < lits[j].String()
	})

	out := make([]expr.Literal, len(lits))
	for i, l := range lits {
		out[i] = toSubstraitLiteral(typ, l)
	}

	return out
}

func (t *toSubstraitExpr) getRef(ref iceberg.BoundReference) expr.Reference {
	path := ref.PosPath()
	out := expr.NewStructFieldRef(int32(path[0]))
	if len(path) == 1 {
		return out
	}

	cur := out
	for _, p := range path[1:] {
		next := expr.NewStructFieldRef(int32(p))
		cur.Child, cur = next, next
	}

	return out
}

func (t *toSubstraitExpr) makeSetFunc(id extensions.ID, term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) expr.Builder {
	val := toSubstraitLiteralSet(term.Type(), lits.Members())

	return t.bldr.ScalarFunc(id).Args(t.bldr.RootRef(t.getRef(term.Ref())),
		t.bldr.Literal(expr.NewNestedLiteral(val, false)))
}

func (t *toSubstraitExpr) VisitIn(term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) expr.Builder {
	return t.makeSetFunc(isInID, term, lits)
}

func (t *toSubstraitExpr) VisitNotIn(term iceberg.BoundTerm, lits iceberg.Set[iceberg.Literal]) expr.Builder {
	return t.bldr.ScalarFunc(notID).Args(t.makeSetFunc(isInID, term, lits).(expr.FuncArgBuilder))
}

func (t *toSubstraitExpr) makeRefFunc(id extensions.ID, term iceberg.BoundTerm) expr.Builder {
	return t.bldr.ScalarFunc(id).Args(t.bldr.RootRef(t.getRef(term.Ref())))
}

func (t *toSubstraitExpr) VisitIsNan(term iceberg.BoundTerm) expr.Builder {
	return t.makeRefFunc(isNaNID, term)
}

func (t *toSubstraitExpr) VisitNotNan(term iceberg.BoundTerm) expr.Builder {
	return t.bldr.ScalarFunc(notID).Args(
		t.makeRefFunc(isNaNID, term).(expr.FuncArgBuilder),
	)
}

func (t *toSubstraitExpr) VisitIsNull(term iceberg.BoundTerm) expr.Builder {
	return t.makeRefFunc(isNullID, term)
}

func (t *toSubstraitExpr) VisitNotNull(term iceberg.BoundTerm) expr.Builder {
	return t.makeRefFunc(isNotNullID, term)
}

func (t *toSubstraitExpr) makeLitFunc(id extensions.ID, term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.bldr.ScalarFunc(id).Args(t.bldr.RootRef(t.getRef(term.Ref())),
		t.bldr.Literal(toSubstraitLiteral(term.Type(), lit)))
}

func (t *toSubstraitExpr) VisitEqual(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(equalID, term, lit)
}

func (t *toSubstraitExpr) VisitNotEqual(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(notEqualID, term, lit)
}

func (t *toSubstraitExpr) VisitGreaterEqual(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(greaterEqualID, term, lit)
}

func (t *toSubstraitExpr) VisitGreater(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(greaterID, term, lit)
}

func (t *toSubstraitExpr) VisitLessEqual(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(lessEqualID, term, lit)
}

func (t *toSubstraitExpr) VisitLess(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(lessID, term, lit)
}

func (t *toSubstraitExpr) VisitStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.makeLitFunc(startsWithID, term, lit)
}

func (t *toSubstraitExpr) VisitNotStartsWith(term iceberg.BoundTerm, lit iceberg.Literal) expr.Builder {
	return t.bldr.ScalarFunc(notID).Args(
		t.makeLitFunc(startsWithID, term, lit).(expr.FuncArgBuilder),
	)
}

// toSubstraitExpr implements the optional BoundGeospatialExprVisitor as a
// backstop: bbox predicates are dropped to always-true during column-name
// translation and never reach substrait, but if one ever did these panic with a
// wrapped ErrNotImplemented (recovered into a typed, wrappable error by
// iceberg.VisitExpr) rather than a bare string.
var _ iceberg.BoundGeospatialExprVisitor[expr.Builder] = (*toSubstraitExpr)(nil)

func (t *toSubstraitExpr) VisitBBoxIntersects(iceberg.BoundTerm, iceberg.BoundingBox) expr.Builder {
	panic(fmt.Errorf("%w: geospatial bbox predicates cannot be converted to substrait", iceberg.ErrNotImplemented))
}

func (t *toSubstraitExpr) VisitBBoxNotIntersects(iceberg.BoundTerm, iceberg.BoundingBox) expr.Builder {
	panic(fmt.Errorf("%w: geospatial bbox predicates cannot be converted to substrait", iceberg.ErrNotImplemented))
}
