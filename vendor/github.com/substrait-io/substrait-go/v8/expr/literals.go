// SPDX-License-Identifier: Apache-2.0

//lint:file-ignore SA1019 Using a deprecated function, variable, constant or field

package expr

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	substraitgo "github.com/substrait-io/substrait-go/v8"
	"github.com/substrait-io/substrait-go/v8/types"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// PrimitiveLiteralValue is a type constraint that represents
// any of the non-nested literal types which are also easily comparable
// via ==
type PrimitiveLiteralValue interface {
	bool | int8 | int16 | ~int32 | ~int64 | float32 | float64 | ~string
}

type nestedLiteral interface {
	StructLiteralValue | ListLiteralValue
}

type WithTypeLiteral interface {
	// WithType attempts to convert a literal to a new type while preserving its value
	WithType(types.Type) (Literal, error)
}

// Easy type aliases for multi-value types that also
// saves us having to create new types / new objects at runtime
// when getting them from protobuf.
type (
	// StructLiteralValue is a slice of other literals where each
	// element in the slice is a different field in the struct
	StructLiteralValue []Literal
	// ListLiteralValue is a slice of other literals
	ListLiteralValue []Literal
	MapLiteralValue  []struct {
		Key   Literal
		Value Literal
	}
	// Null is a typed null value so it can be just a Type itself
	Null types.Type
)

// Deprecated: use VirtualTableExprFromLiteralProto
func StructLiteralFromProto(s *proto.Expression_Literal_Struct) StructLiteralValue {
	fields := make(StructLiteralValue, len(s.Fields))
	for i, f := range s.Fields {
		fields[i] = LiteralFromProto(f)
	}
	return fields
}

func (s StructLiteralValue) ToProto() *proto.Expression_Literal_Struct {
	fields := make([]*proto.Expression_Literal, len(s))
	for i, f := range s {
		fields[i] = f.ToProtoLiteral()
	}

	return &proto.Expression_Literal_Struct{
		Fields: fields,
	}
}

// Literal represents a specific literal of some type which could also
// be a typed null or a nested type like a struct/map/list.
//
// An empty map/empty list will have len(value) == 0
type Literal interface {
	// Literals are also Function arguments
	types.FuncArg
	RootRefType
	fmt.Stringer

	IsScalar() bool
	// GetType returns the full Type of the literal value
	GetType() types.Type
	// Equals only returns true if the rhs is a literal of the exact
	// same type and value.
	Equals(Expression) bool
	ToProto() *proto.Expression
	ToProtoLiteral() *proto.Expression_Literal
	Visit(VisitFunc) Expression
	// ValueString returns a human presentable representation of just the literal's value.
	ValueString() string
}

// A NullLiteral is a typed null, so it just contains its type with no value
type NullLiteral struct {
	Type types.Type
}

func NewNullLiteral(t types.Type) *NullLiteral {
	return &NullLiteral{Type: t.WithNullability(types.NullabilityNullable)}
}

func (*NullLiteral) IsScalar() bool { return true }

func (*NullLiteral) isRootRef() {}
func (n *NullLiteral) String() string {
	return "null(" + n.Type.String() + ")"
}
func (n *NullLiteral) ValueString() string {
	return "null"
}

func (n *NullLiteral) GetType() types.Type { return n.Type }
func (n *NullLiteral) ToProtoLiteral() *proto.Expression_Literal {
	return &proto.Expression_Literal{
		Nullable:               true,
		TypeVariationReference: n.Type.GetTypeVariationReference(),
		LiteralType:            &proto.Expression_Literal_Null{Null: types.TypeToProto(n.Type)},
	}
}

func (n *NullLiteral) ToProto() *proto.Expression {
	return &proto.Expression{
		RexType: &proto.Expression_Literal_{Literal: n.ToProtoLiteral()},
	}
}

func (n *NullLiteral) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: n.ToProto()},
	}
}

func (n *NullLiteral) Equals(rhs Expression) bool {
	if nl, ok := rhs.(*NullLiteral); ok {
		return nl.Type.Equals(n.Type)
	}

	return false
}

func (n *NullLiteral) Visit(VisitFunc) Expression {
	return n
}

// PrimitiveLiteral represents a non-nested, non-null literal value
// which can be compared easily using ==
type PrimitiveLiteral[T PrimitiveLiteralValue] struct {
	Value T
	Type  types.Type
}

func (*PrimitiveLiteral[T]) isRootRef() {}
func (t *PrimitiveLiteral[T]) String() string {
	return fmt.Sprintf("%s(%s)", t.Type.String(), t.ValueString())
}

func (t *PrimitiveLiteral[T]) ValueString() string {
	if lit, ok := any(t.Value).(types.TimePrinter); ok {
		return lit.ToTimeString()
	}
	return fmt.Sprintf("%v", t.Value)
}

func (t *PrimitiveLiteral[T]) IsoValueString() string {
	switch x := any(t.Value).(type) {
	case types.IsoTimePrinter:
		return x.ToIsoTimeString()
	}
	return t.ValueString()
}

func (t *PrimitiveLiteral[T]) GetType() types.Type { return t.Type }
func (t *PrimitiveLiteral[T]) ToProtoLiteral() *proto.Expression_Literal {
	lit := &proto.Expression_Literal{
		Nullable:               t.Type.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.Type.GetTypeVariationReference(),
	}

	switch v := any(t.Value).(type) {
	case bool:
		lit.LiteralType = &proto.Expression_Literal_Boolean{Boolean: v}
	case int8:
		lit.LiteralType = &proto.Expression_Literal_I8{I8: int32(v)}
	case int16:
		lit.LiteralType = &proto.Expression_Literal_I16{I16: int32(v)}
	case int32:
		lit.LiteralType = &proto.Expression_Literal_I32{I32: v}
	case int64:
		lit.LiteralType = &proto.Expression_Literal_I64{I64: v}
	case float32:
		lit.LiteralType = &proto.Expression_Literal_Fp32{Fp32: v}
	case float64:
		lit.LiteralType = &proto.Expression_Literal_Fp64{Fp64: v}
	case string:
		lit.LiteralType = &proto.Expression_Literal_String_{String_: v}
	case types.Timestamp:
		lit.LiteralType = &proto.Expression_Literal_Timestamp{Timestamp: int64(v)}
	case types.Date:
		lit.LiteralType = &proto.Expression_Literal_Date{Date: int32(v)}
	case types.Time:
		lit.LiteralType = &proto.Expression_Literal_Time{Time: int64(v)}
	case types.FixedChar:
		lit.LiteralType = &proto.Expression_Literal_FixedChar{FixedChar: string(v)}
	case types.TimestampTz:
		lit.LiteralType = &proto.Expression_Literal_TimestampTz{TimestampTz: int64(v)}
	default:
		panic("invalid primitive literal type")
	}

	return lit
}

func (t *PrimitiveLiteral[T]) ToProto() *proto.Expression {
	return &proto.Expression{
		RexType: &proto.Expression_Literal_{Literal: t.ToProtoLiteral()},
	}
}

func (t *PrimitiveLiteral[T]) Equals(rhs Expression) bool {
	if other, ok := rhs.(*PrimitiveLiteral[T]); ok {
		return t.Type.Equals(other.Type) && t.Value == other.Value
	}
	return false
}

func (t *PrimitiveLiteral[T]) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: t.ToProto()},
	}
}

func (t *PrimitiveLiteral[T]) Visit(VisitFunc) Expression { return t }
func (*PrimitiveLiteral[T]) IsScalar() bool               { return true }

func (t *PrimitiveLiteral[T]) WithType(newType types.Type) (Literal, error) {
	switch newType.(type) {
	case *types.FixedLenType[types.FixedChar]:
		if val, ok := any(t.Value).(types.FixedChar); ok {
			return &PrimitiveLiteral[types.FixedChar]{
				Value: val,
				Type:  newType,
			}, nil
		}
	}
	return nil, fmt.Errorf("invalid type %T for primitive literal", newType)
}

// NestedLiteral is either a Struct or List literal, both of which are
// represented as a slice of other literals.
type NestedLiteral[T nestedLiteral] struct {
	Value T
	Type  types.Type
}

func (*NestedLiteral[T]) isRootRef() {}
func (t *NestedLiteral[T]) String() string {
	return fmt.Sprintf("%s(%v)", t.Type, t.Value)
}
func (t *NestedLiteral[T]) ValueString() string {
	switch x := any(t.Value).(type) {
	case ListLiteralValue:
		var items []string
		for _, item := range x {
			items = append(items, item.ValueString())
		}
		return fmt.Sprintf("[%s]", strings.Join(items, ", "))
	}
	return fmt.Sprintf("%v", t.Value)
}
func (t *NestedLiteral[T]) GetType() types.Type { return t.Type }
func (t *NestedLiteral[T]) ToProtoLiteral() *proto.Expression_Literal {
	lit := &proto.Expression_Literal{
		Nullable:               t.Type.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.Type.GetTypeVariationReference(),
	}

	vals := make([]*proto.Expression_Literal, len(t.Value))
	for i, l := range t.Value {
		vals[i] = l.ToProtoLiteral()
	}

	switch any(t.Value).(type) {
	case StructLiteralValue:
		lit.LiteralType = &proto.Expression_Literal_Struct_{
			Struct: &proto.Expression_Literal_Struct{
				Fields: vals,
			},
		}
	case ListLiteralValue:
		if len(vals) == 0 {
			lit.LiteralType = &proto.Expression_Literal_EmptyList{
				EmptyList: types.TypeToProto(t.Type).GetList(),
			}
		} else {
			lit.LiteralType = &proto.Expression_Literal_List_{
				List: &proto.Expression_Literal_List{
					Values: vals,
				},
			}
		}
	}

	return lit
}

func (t *NestedLiteral[T]) ToProto() *proto.Expression {
	return &proto.Expression{
		RexType: &proto.Expression_Literal_{Literal: t.ToProtoLiteral()},
	}
}

func (t *NestedLiteral[T]) Equals(rhs Expression) bool {
	if other, ok := rhs.(*NestedLiteral[T]); ok {
		return t.Type.Equals(other.Type) && slices.EqualFunc(t.Value, other.Value, func(a, b Literal) bool {
			return a.Equals(b)
		})
	}
	return false
}

func (t *NestedLiteral[T]) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: t.ToProto()},
	}
}

func (t *NestedLiteral[T]) Visit(VisitFunc) Expression {
	return t
}
func (*NestedLiteral[T]) IsScalar() bool { return true }

func (t *NestedLiteral[T]) WithType(newType types.Type) (Literal, error) {
	switch newType.(type) {
	case *types.ListType:
		return &NestedLiteral[ListLiteralValue]{
			Value: ListLiteralValue(t.Value),
			Type:  newType,
		}, nil
	}
	return nil, fmt.Errorf("invalid type %T for nested literal", newType)
}

// MapLiteral is represented as a slice of Key/Value structs consisting
// of other literals.
type MapLiteral struct {
	Value MapLiteralValue
	Type  types.Type
}

func (*MapLiteral) isRootRef() {}
func (t *MapLiteral) String() string {
	return fmt.Sprintf("%s(%s)", t.Type, t.ValueString())
}
func (t *MapLiteral) ValueString() string {
	return fmt.Sprintf("%v", t.Value)
}
func (t *MapLiteral) GetType() types.Type { return t.Type }
func (t *MapLiteral) ToProtoLiteral() *proto.Expression_Literal {
	lit := &proto.Expression_Literal{
		Nullable:               t.Type.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.Type.GetTypeVariationReference(),
	}

	if len(t.Value) == 0 {
		lit.LiteralType = &proto.Expression_Literal_EmptyMap{
			EmptyMap: types.TypeToProto(t.Type).GetMap(),
		}
	} else {
		kv := make([]*proto.Expression_Literal_Map_KeyValue, len(t.Value))
		for i, v := range t.Value {
			kv[i].Key = v.Key.ToProtoLiteral()
			kv[i].Value = v.Value.ToProtoLiteral()
		}

		lit.LiteralType = &proto.Expression_Literal_Map_{
			Map: &proto.Expression_Literal_Map{KeyValues: kv},
		}
	}

	return lit
}

func (t *MapLiteral) ToProto() *proto.Expression {
	return &proto.Expression{RexType: &proto.Expression_Literal_{
		Literal: t.ToProtoLiteral(),
	}}
}

func (t *MapLiteral) Equals(rhs Expression) bool {
	if other, ok := rhs.(*MapLiteral); ok {
		return t.Type.Equals(other.Type) && slices.EqualFunc(t.Value, other.Value,
			func(a, b struct{ Key, Value Literal }) bool {
				return a.Key.Equals(b.Key) && a.Value.Equals(b.Value)
			})
	}
	return false
}

func (t *MapLiteral) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: t.ToProto()},
	}
}

func (t *MapLiteral) Visit(VisitFunc) Expression { return t }
func (*MapLiteral) IsScalar() bool               { return true }

// ByteSliceLiteral is any literal that is represented as a byte slice.
// As opposed to a string literal which can be compared with ==, a byte
// slice needs to use something like bytes.Equal
type ByteSliceLiteral[T ~[]byte] struct {
	Value T
	Type  types.Type
}

func (*ByteSliceLiteral[T]) isRootRef() {}
func (t *ByteSliceLiteral[T]) String() string {
	return fmt.Sprintf("%s(%s)", t.Type, t.ValueString())
}
func (t *ByteSliceLiteral[T]) ValueString() string {
	switch x := any(t.Value).(type) {
	case types.UUID:
		u, err := uuid.FromBytes(x[:])
		if err != nil {
			return fmt.Sprintf("%v", t.Value)
		}
		return u.String()
	case types.FixedBinary:
		return "0x" + hex.EncodeToString(x[:])
	default:
		return fmt.Sprintf("%v", t.Value)
	}
}

func (t *ByteSliceLiteral[T]) GetType() types.Type { return t.Type }
func (t *ByteSliceLiteral[T]) ToProtoLiteral() *proto.Expression_Literal {
	lit := &proto.Expression_Literal{
		Nullable:               t.Type.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.Type.GetTypeVariationReference(),
	}

	switch v := any(t.Value).(type) {
	case []byte:
		lit.LiteralType = &proto.Expression_Literal_Binary{Binary: v}
	case types.FixedBinary:
		lit.LiteralType = &proto.Expression_Literal_FixedBinary{FixedBinary: v}
	case types.UUID:
		lit.LiteralType = &proto.Expression_Literal_Uuid{Uuid: v}
	}

	return lit
}

func (t *ByteSliceLiteral[T]) ToProto() *proto.Expression {
	return &proto.Expression{RexType: &proto.Expression_Literal_{
		Literal: t.ToProtoLiteral(),
	}}
}

func (t *ByteSliceLiteral[T]) Equals(rhs Expression) bool {
	if other, ok := rhs.(*ByteSliceLiteral[T]); ok {
		return t.Type.Equals(other.Type) &&
			bytes.Equal(t.Value, other.Value)
	}

	return false
}

func (t *ByteSliceLiteral[T]) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: t.ToProto()},
	}
}

func (t *ByteSliceLiteral[T]) Visit(VisitFunc) Expression { return t }
func (*ByteSliceLiteral[T]) IsScalar() bool               { return true }

func (t *ByteSliceLiteral[T]) WithType(newType types.Type) (Literal, error) {
	switch newType.(type) {
	case *types.FixedBinaryType:
		return &ByteSliceLiteral[types.FixedBinary]{
			Value: types.FixedBinary(t.Value),
			Type:  newType,
		}, nil
	case *types.UUIDType:
		return &ByteSliceLiteral[types.UUID]{
			Value: types.UUID(t.Value),
			Type:  newType,
		}, nil
	}
	return nil, fmt.Errorf("byte slice literal withType is not supported for %T ", newType)
}

// ProtoLiteral is a literal that is represented using its protobuf
// message type such as a Decimal or UserDefinedType.
type ProtoLiteral struct {
	Value any
	Type  types.Type
}

func (t *ProtoLiteral) WithType(newType types.Type) (Literal, error) {
	switch typ := newType.(type) {
	case *types.DecimalType:
		return newDecimalWithType(t, typ)
	case *types.VarCharType:
		return newVarCharWithType(t, typ)
	case *types.PrecisionTimeType:
		return newPrecisionTimeWithType(t, typ)
	case *types.PrecisionTimestampType:
		return newPrecisionTimestampWithType(t, typ)
	case *types.PrecisionTimestampTzType:
		return newPrecisionTimestampTzWithType(t, typ)
	case *types.IntervalDayType:
		return newIntervalDayWithType(t, typ)
	}
	return nil, fmt.Errorf("protoLiteral withType is not supported for %T ", newType)
}

func (t *ProtoLiteral) ValueString() string {
	switch literalType := t.Type.(type) {
	case *types.PrecisionTimeType:
		tm := types.Time(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format("15:04:05.999999999")
	case *types.PrecisionTimestampType:
		tm := types.Timestamp(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format("2006-01-02 15:04:05.999999999")
	case *types.PrecisionTimestampTzType:
		tm := types.TimestampTz(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format(time.RFC3339Nano)
	case *types.DecimalType:
		return decimalBytesToString([16]byte(t.Value.([]byte)), literalType.Scale)
	case *types.IntervalYearType:
		x, _ := t.Value.(*proto.Expression_Literal_IntervalYearToMonth)
		// Validity is required by construction.
		return fmt.Sprintf("%d years, %d months", x.GetYears(), x.GetMonths())
	case *types.IntervalDayType:
		x, _ := t.Value.(*proto.Expression_Literal_IntervalDayToSecond)
		// Validity is required by construction.
		return fmt.Sprintf("%d days, %d seconds, %d subseconds", x.GetDays(), x.GetSeconds(), x.GetSubseconds())
	}
	return fmt.Sprintf("%s", t.Value)
}

// IsoValueString handles precision timestamp and interval literals to return a string in ISO 8601 format
func (t *ProtoLiteral) IsoValueString() string {
	switch literalType := t.Type.(type) {
	case *types.PrecisionTimeType:
		tm := types.Time(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format("15:04:05.999999999")
	case *types.PrecisionTimestampType:
		tm := types.Timestamp(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format("2006-01-02T15:04:05.999999999")
	case *types.PrecisionTimestampTzType:
		tm := types.TimestampTz(t.Value.(int64)).ToPrecisionTime(literalType.Precision)
		return tm.UTC().Format("2006-01-02T15:04:05.000-07:00")
	case *types.IntervalYearType:
		x, _ := t.Value.(*proto.Expression_Literal_IntervalYearToMonth)
		// Validity is required by construction.
		return fmt.Sprintf("P%dY%dM", x.GetYears(), x.GetMonths())
	case *types.IntervalDayType:
		x, _ := t.Value.(*proto.Expression_Literal_IntervalDayToSecond)
		// Validity is required by construction.
		sb := strings.Builder{}
		sb.WriteString("P")
		if x.GetDays() > 0 {
			sb.WriteString(fmt.Sprintf("%dD", x.GetDays()))
		}
		if x.GetSeconds() > 0 || x.GetSubseconds() > 0 {
			sb.WriteString("T")
			duration := time.Duration(x.GetSeconds()) * time.Second
			duration += types.SubSecondsToDuration(x.GetSubseconds(), literalType.Precision)
			sb.WriteString(strings.ToUpper(duration.String()))
		}
		return sb.String()
	}
	return t.ValueString()
}

func (*ProtoLiteral) isRootRef()            {}
func (t *ProtoLiteral) GetType() types.Type { return t.Type }
func (t *ProtoLiteral) String() string {
	return fmt.Sprintf("%s(%s)", t.Type, t.ValueString())
}
func (t *ProtoLiteral) ToProtoLiteral() *proto.Expression_Literal {
	lit := &proto.Expression_Literal{
		Nullable:               t.Type.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.Type.GetTypeVariationReference(),
	}

	switch literalType := t.Type.(type) {
	case *types.UserDefinedType:
		params := make([]*proto.Type_Parameter, len(literalType.TypeParameters))
		for i, p := range literalType.TypeParameters {
			params[i] = p.ToProto()
		}

		udt := &proto.Expression_Literal_UserDefined{
			TypeAnchorType: &proto.Expression_Literal_UserDefined_TypeReference{
				TypeReference: literalType.TypeReference},
			TypeParameters: params,
		}
		switch v := t.Value.(type) {
		case *proto.Expression_Literal_UserDefined_Value:
			udt.Val = v
		case *proto.Expression_Literal_UserDefined_Struct:
			udt.Val = v
		default:
			panic(fmt.Sprintf("unexpected UserDefined literal value type: %T", t.Value))
		}

		lit.LiteralType = &proto.Expression_Literal_UserDefined_{
			UserDefined: udt,
		}
	case *types.IntervalYearType:
		v := t.Value.(*types.IntervalYearToMonth)
		lit.LiteralType = &proto.Expression_Literal_IntervalYearToMonth_{
			IntervalYearToMonth: v,
		}
	case *types.IntervalDayType:
		v := t.Value.(*types.IntervalDayToSecond)
		lit.LiteralType = &proto.Expression_Literal_IntervalDayToSecond_{
			IntervalDayToSecond: v,
		}
	case *types.VarCharType:
		v := t.Value.(string)
		lit.LiteralType = &proto.Expression_Literal_VarChar_{
			VarChar: &proto.Expression_Literal_VarChar{
				Value:  v,
				Length: uint32(literalType.Length),
			},
		}
	case *types.DecimalType:
		v := t.Value.([]byte)
		lit.LiteralType = &proto.Expression_Literal_Decimal_{
			Decimal: &proto.Expression_Literal_Decimal{
				Value:     v,
				Precision: literalType.Precision,
				Scale:     literalType.Scale,
			},
		}
	case *types.PrecisionTimeType:
		v := t.Value.(int64)
		lit.LiteralType = &proto.Expression_Literal_PrecisionTime_{
			PrecisionTime: &proto.Expression_Literal_PrecisionTime{
				Precision: literalType.GetPrecisionProtoVal(),
				Value:     v,
			},
		}
	case *types.PrecisionTimestampType:
		v := t.Value.(int64)
		lit.LiteralType = &proto.Expression_Literal_PrecisionTimestamp_{
			PrecisionTimestamp: &proto.Expression_Literal_PrecisionTimestamp{
				Precision: literalType.GetPrecisionProtoVal(),
				Value:     v,
			},
		}
	case *types.PrecisionTimestampTzType:
		v := t.Value.(int64)
		lit.LiteralType = &proto.Expression_Literal_PrecisionTimestampTz{
			PrecisionTimestampTz: &proto.Expression_Literal_PrecisionTimestamp{
				Precision: literalType.GetPrecisionProtoVal(),
				Value:     v,
			},
		}
	}
	return lit
}

func (t *ProtoLiteral) ToProto() *proto.Expression {
	return &proto.Expression{RexType: &proto.Expression_Literal_{
		Literal: t.ToProtoLiteral(),
	}}
}

func (t *ProtoLiteral) Equals(rhs Expression) bool {
	if other, ok := rhs.(*ProtoLiteral); ok {
		return t.Type.Equals(other.Type) &&
			reflect.DeepEqual(t.Value, other.Value)
	}
	return false
}

func (t *ProtoLiteral) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: t.ToProto()},
	}
}

func (t *ProtoLiteral) Visit(VisitFunc) Expression { return t }
func (*ProtoLiteral) IsScalar() bool               { return true }

func newDecimalWithType(literal *ProtoLiteral, decType *types.DecimalType) (Literal, error) {
	litType, ok := literal.GetType().(*types.DecimalType)
	if !ok {
		return nil, fmt.Errorf("literal type is not decimal")
	}
	inDecimalBytes := [16]byte(literal.Value.([]byte))
	decimalBytes, precision, scale, err := modifyDecimalPrecisionAndScale(inDecimalBytes, litType.Scale, decType.Precision, decType.Scale)
	if err != nil {
		return nil, err
	}
	return NewLiteral[*types.Decimal](&types.Decimal{Value: decimalBytes[:16], Precision: precision, Scale: scale}, decType.GetNullability() == types.NullabilityNullable)
}

func newVarCharWithType(literal *ProtoLiteral, vcharType *types.VarCharType) (Literal, error) {
	if _, ok := literal.GetType().(*types.VarCharType); !ok {
		return nil, fmt.Errorf("literal type is not varchar")
	}
	if len(literal.Value.(string)) > int(vcharType.GetLength()) {
		return nil, fmt.Errorf("varchar literal value length is greater than type length")
	}
	return &ProtoLiteral{Value: literal.Value, Type: vcharType}, nil
}

func newPrecisionTimeWithType(literal *ProtoLiteral, ptType *types.PrecisionTimeType) (Literal, error) {
	if litType, ok := literal.GetType().(*types.PrecisionTimeType); ok {
		value := types.GetTimeValueByPrecision(types.Time(literal.Value.(int64)).ToPrecisionTime(litType.Precision), ptType.Precision)
		return &ProtoLiteral{Value: value, Type: ptType}, nil
	}
	return nil, fmt.Errorf("literal type is not precision time")
}

func newPrecisionTimestampWithType(literal *ProtoLiteral, ptsType *types.PrecisionTimestampType) (Literal, error) {
	if litType, ok := literal.GetType().(*types.PrecisionTimestampType); ok {
		value := types.GetTimeValueByPrecision(types.Timestamp(literal.Value.(int64)).ToPrecisionTime(litType.Precision), ptsType.Precision)
		return &ProtoLiteral{Value: value, Type: ptsType}, nil
	}
	return nil, fmt.Errorf("literal type is not precision timestamp")
}

func newPrecisionTimestampTzWithType(literal *ProtoLiteral, ptstzType *types.PrecisionTimestampTzType) (Literal, error) {
	if litType, ok := literal.GetType().(*types.PrecisionTimestampTzType); ok {
		value := types.GetTimeValueByPrecision(types.Timestamp(literal.Value.(int64)).ToPrecisionTime(litType.Precision), ptstzType.Precision)
		return &ProtoLiteral{Value: value, Type: ptstzType}, nil
	}
	return nil, fmt.Errorf("literal type is not precision timestamp tz")
}

func newIntervalDayWithType(literal *ProtoLiteral, intervalDayType *types.IntervalDayType) (Literal, error) {
	if _, ok := literal.GetType().(*types.IntervalDayType); ok {
		intervalValue := literal.Value.(*proto.Expression_Literal_IntervalDayToSecond)
		precisionDiff := intervalValue.GetPrecision() - intervalDayType.Precision.ToProtoVal()
		ss := intervalValue.GetSubseconds()
		if precisionDiff != 0 {
			factor := int64(math.Pow10(int(math.Abs(float64(precisionDiff)))))
			if precisionDiff > 0 {
				ss /= factor
			} else {
				ss *= factor
			}
		}
		return &ProtoLiteral{
			Value: &types.IntervalDayToSecond{
				Days:       intervalValue.GetDays(),
				Seconds:    intervalValue.GetSeconds(),
				Subseconds: ss,
				PrecisionMode: &proto.Expression_Literal_IntervalDayToSecond_Precision{
					Precision: intervalDayType.Precision.ToProtoVal(),
				},
			}, Type: intervalDayType,
		}, nil
	}
	return nil, fmt.Errorf("literal type is not interval day")
}

func getNullability(nullable bool) types.Nullability {
	if nullable {
		return types.NullabilityNullable
	}
	return types.NullabilityRequired
}

// A PrimitiveLiteral is a literal with one of the types specified in this interface
type newPrimitiveLiteralTypes interface {
	bool | int8 | int16 | ~int32 | ~int64 | float32 | float64 | string
}

func NewPrimitiveLiteral[T newPrimitiveLiteralTypes](val T, nullable bool) Literal {
	return &PrimitiveLiteral[T]{
		Value: val,
		Type: &types.PrimitiveType[T]{
			Nullability: getNullability(nullable),
		},
	}
}

func NewFixedCharLiteral(val types.FixedChar, nullable bool) *PrimitiveLiteral[types.FixedChar] {
	return &PrimitiveLiteral[types.FixedChar]{
		Value: val,
		Type: &types.FixedCharType{
			Nullability: getNullability(nullable),
			Length:      int32(len(val)),
		},
	}
}

// Convenience names so that there is StructLiteral, ListLiteral and MapLiteral
type (
	StructLiteral = NestedLiteral[StructLiteralValue]
	ListLiteral   = NestedLiteral[ListLiteralValue]
)

// NewNestedLiteral constructs a new literal value and marks whether the
// type should be considered nullable. This assumes that the passed in
// value is not empty, so len(v) MUST be > 0.
//
// For an Empty Map literal or an empty List literal, you need to use the
// corresponding NewEmptyMapLiteral and NewEmptyListLiteral functions which
// take the Type of the empty literal as an argument.
func NewNestedLiteral[T StructLiteralValue | MapLiteralValue | ListLiteralValue](val T, nullable bool) Literal {
	nullability := getNullability(nullable)

	switch v := any(val).(type) {
	case StructLiteralValue:
		typeList := make([]types.Type, len(v))
		for i, f := range v {
			typeList[i] = f.GetType()
		}
		return &NestedLiteral[StructLiteralValue]{
			Value: v,
			Type: &types.StructType{
				Nullability: nullability,
				Types:       typeList,
			}}
	case MapLiteralValue:
		return &MapLiteral{
			Value: v,
			Type: &types.MapType{
				Nullability: nullability,
				Key:         v[0].Key.GetType(),
				Value:       v[0].Value.GetType(),
			}}
	case ListLiteralValue:
		return &NestedLiteral[ListLiteralValue]{
			Value: v,
			Type: &types.ListType{
				Nullability: nullability,
				Type:        v[0].GetType(),
			}}
	}
	panic("should not get here")
}

// NewEmptyMapLiteral creates an empty map literal of the provided key/value
// types and marks the type as nullable or not.
func NewEmptyMapLiteral(key, val types.Type, nullable bool) *MapLiteral {
	return &MapLiteral{
		Type: &types.MapType{
			Nullability: getNullability(nullable),
			Key:         key,
			Value:       val,
		},
	}
}

// NewEmptyListLiteral creates an empty list literal of the
// type and marks the type as nullable or not.
func NewEmptyListLiteral(t types.Type, nullable bool) *ListLiteral {
	return &NestedLiteral[ListLiteralValue]{
		Type: &types.ListType{
			Nullability: getNullability(nullable),
			Type:        t,
		}}
}

func NewByteSliceLiteral[T []byte | types.UUID](val T, nullable bool) *ByteSliceLiteral[T] {
	return &ByteSliceLiteral[T]{
		Value: val,
		Type: &types.PrimitiveType[T]{
			Nullability: getNullability(nullable),
		},
	}
}

func NewFixedBinaryLiteral(val types.FixedBinary, nullable bool) *ByteSliceLiteral[types.FixedBinary] {
	return &ByteSliceLiteral[types.FixedBinary]{
		Value: val,
		Type: &types.FixedLenType[types.FixedBinary]{
			Length:      int32(len(val)),
			Nullability: getNullability(nullable),
		},
	}
}

type allLiteralTypes interface {
	PrimitiveLiteralValue | nestedLiteral | MapLiteralValue |
		[]byte | types.UUID | types.FixedBinary | *types.IntervalYearToMonth |
		*types.IntervalDayToSecond | *types.VarChar | *types.Decimal | *types.UserDefinedLiteral |
		*types.PrecisionTime | *types.PrecisionTimestamp | *types.PrecisionTimestampTz
}

func NewLiteral[T allLiteralTypes](val T, nullable bool) (Literal, error) {
	switch v := any(val).(type) {
	case bool:
		return NewPrimitiveLiteral(v, nullable), nil
	case int8:
		return NewPrimitiveLiteral(v, nullable), nil
	case int16:
		return NewPrimitiveLiteral(v, nullable), nil
	case int32:
		return NewPrimitiveLiteral(v, nullable), nil
	case int64:
		return NewPrimitiveLiteral(v, nullable), nil
	case float32:
		return NewPrimitiveLiteral(v, nullable), nil
	case float64:
		return NewPrimitiveLiteral(v, nullable), nil
	case string:
		return NewPrimitiveLiteral(v, nullable), nil
	case types.Timestamp:
		return NewPrimitiveLiteral(v, nullable), nil
	case types.TimestampTz:
		return NewPrimitiveLiteral(v, nullable), nil
	case types.Date:
		return NewPrimitiveLiteral(v, nullable), nil
	case types.Time:
		return NewPrimitiveLiteral(v, nullable), nil
	case types.FixedChar:
		return NewFixedCharLiteral(v, nullable), nil
	case types.UUID:
		return NewByteSliceLiteral(v, nullable), nil
	case []byte:
		return NewByteSliceLiteral(v, nullable), nil
	case types.FixedBinary:
		return NewFixedBinaryLiteral(v, nullable), nil
	case StructLiteralValue:
		return NewNestedLiteral(v, nullable), nil
	case ListLiteralValue:
		return NewNestedLiteral(v, nullable), nil
	case MapLiteralValue:
		return NewNestedLiteral(v, nullable), nil
	case *types.IntervalYearToMonth:
		return &ProtoLiteral{
			Value: v,
			Type: &types.IntervalYearType{
				Nullability: getNullability(nullable),
			},
		}, nil
	case *types.IntervalDayToSecond:
		return &ProtoLiteral{
			Value: v,
			Type: &types.IntervalDayType{
				Nullability: getNullability(nullable),
				Precision:   types.TimePrecision(v.GetPrecision()),
			},
		}, nil
	case *types.Decimal:
		if len(v.Value) != 16 {
			return nil, fmt.Errorf("decimal value must be 16 bytes")
		}
		return &ProtoLiteral{
			Value: v.Value,
			Type: &types.DecimalType{
				Nullability: getNullability(nullable),
				Precision:   v.Precision,
				Scale:       v.Scale,
			},
		}, nil
	case *types.UserDefinedLiteral:
		params := make([]types.TypeParam, len(v.TypeParameters))
		for i, p := range v.TypeParameters {
			params[i] = types.TypeParamFromProto(p)
		}

		return &ProtoLiteral{
			Value: v.Val,
			Type: &types.UserDefinedType{
				Nullability:    getNullability(nullable),
				TypeReference:  v.GetTypeReference(),
				TypeParameters: params,
			},
		}, nil
	case *types.VarChar:
		return &ProtoLiteral{
			Value: v.Value,
			Type: &types.VarCharType{
				Nullability: getNullability(nullable),
				Length:      int32(v.Length),
			},
		}, nil
	case *types.PrecisionTime:
		return NewPrecisionTimeLiteral(v.Value, types.TimePrecision(v.Precision), getNullability(nullable)), nil
	case *types.PrecisionTimestamp:
		return NewPrecisionTimestampLiteral(v.PrecisionTimestamp.Value, types.TimePrecision(v.PrecisionTimestamp.Precision), getNullability(nullable)), nil
	case *types.PrecisionTimestampTz:
		return NewPrecisionTimestampTzLiteral(v.PrecisionTimestampTz.Value, types.TimePrecision(v.PrecisionTimestampTz.Precision), getNullability(nullable)), nil
	}

	return nil, substraitgo.ErrNotImplemented
}

// LiteralFromProto constructs the appropriate Literal struct from
// a protobuf message.
func LiteralFromProto(l *proto.Expression_Literal) Literal {
	nullability := getNullability(l.Nullable)

	switch lit := l.LiteralType.(type) {
	case *proto.Expression_Literal_Boolean:
		return &PrimitiveLiteral[bool]{
			Value: lit.Boolean,
			Type: &types.BooleanType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_I8:
		return &PrimitiveLiteral[int8]{
			Value: int8(lit.I8),
			Type: &types.Int8Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_I16:
		return &PrimitiveLiteral[int16]{
			Value: int16(lit.I16),
			Type: &types.Int16Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_I32:
		return &PrimitiveLiteral[int32]{
			Value: lit.I32,
			Type: &types.Int32Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_I64:
		return &PrimitiveLiteral[int64]{
			Value: lit.I64,
			Type: &types.Int64Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Fp32:
		return &PrimitiveLiteral[float32]{
			Value: lit.Fp32,
			Type: &types.Float32Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Fp64:
		return &PrimitiveLiteral[float64]{
			Value: lit.Fp64,
			Type: &types.Float64Type{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_String_:
		return &PrimitiveLiteral[string]{
			Value: lit.String_,
			Type: &types.StringType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Binary:
		return &ByteSliceLiteral[[]byte]{
			Value: lit.Binary,
			Type: &types.BinaryType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Timestamp:
		return &PrimitiveLiteral[types.Timestamp]{
			Value: types.Timestamp(lit.Timestamp),
			Type: &types.TimestampType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Date:
		return &PrimitiveLiteral[types.Date]{
			Value: types.Date(lit.Date),
			Type: &types.DateType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Time:
		return &PrimitiveLiteral[types.Time]{
			Value: types.Time(lit.Time),
			Type: &types.TimeType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_IntervalDayToSecond_:
		precision := types.PrecisionMicroSeconds
		switch lit.IntervalDayToSecond.PrecisionMode.(type) {
		case *proto.Expression_Literal_IntervalDayToSecond_Precision:
			var err error
			precision, err = types.ProtoToTimePrecision(lit.IntervalDayToSecond.GetPrecision())
			if err != nil {
				return nil
			}
		}
		return &ProtoLiteral{
			Value: lit.IntervalDayToSecond,
			Type: &types.IntervalDayType{
				Precision:        precision,
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
			},
		}
	case *proto.Expression_Literal_FixedChar:
		return &PrimitiveLiteral[types.FixedChar]{
			Value: types.FixedChar(lit.FixedChar),
			Type: &types.FixedCharType{
				Length:           int32(len(lit.FixedChar)),
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_VarChar_:
		return &ProtoLiteral{
			Value: lit.VarChar.Value,
			Type: &types.VarCharType{
				Length:           int32(lit.VarChar.Length),
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
			},
		}
	case *proto.Expression_Literal_FixedBinary:
		return &ByteSliceLiteral[types.FixedBinary]{
			Value: lit.FixedBinary,
			Type: &types.FixedBinaryType{
				Length:           int32(len(lit.FixedBinary)),
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Decimal_:
		return &ProtoLiteral{
			Value: lit.Decimal.Value,
			Type: &types.DecimalType{
				Scale:            lit.Decimal.Scale,
				Precision:        lit.Decimal.Precision,
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
			},
		}
	case *proto.Expression_Literal_TimestampTz:
		return &PrimitiveLiteral[types.TimestampTz]{
			Value: types.TimestampTz(lit.TimestampTz),
			Type: &types.TimestampTzType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Uuid:
		return &ByteSliceLiteral[types.UUID]{
			Value: lit.Uuid,
			Type: &types.UUIDType{
				TypeVariationRef: l.TypeVariationReference,
				Nullability:      nullability,
			}}
	case *proto.Expression_Literal_Null:
		return &NullLiteral{Type: types.TypeFromProto(lit.Null)}
	case *proto.Expression_Literal_Struct_:
		typeList := make([]types.Type, len(lit.Struct.Fields))
		fields := make([]Literal, len(lit.Struct.Fields))
		for i, f := range lit.Struct.Fields {
			fields[i] = LiteralFromProto(f)
			typeList[i] = fields[i].GetType()
		}

		return &NestedLiteral[StructLiteralValue]{
			Value: StructLiteralValue(fields),
			Type: &types.StructType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				Types:            typeList,
			}}
	case *proto.Expression_Literal_Map_:
		ret := make(MapLiteralValue, len(lit.Map.KeyValues))
		for i, kv := range lit.Map.KeyValues {
			ret[i].Key = LiteralFromProto(kv.Key)
			ret[i].Value = LiteralFromProto(kv.Value)
		}
		return &MapLiteral{
			Value: ret,
			Type: &types.MapType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				Key:              ret[0].Key.GetType(),
				Value:            ret[0].Value.GetType(),
			}}
	case *proto.Expression_Literal_List_:
		ret := make(ListLiteralValue, len(lit.List.Values))
		for i, v := range lit.List.Values {
			ret[i] = LiteralFromProto(v)
		}
		return &NestedLiteral[ListLiteralValue]{
			Value: ListLiteralValue(ret),
			Type: &types.ListType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				Type:             ret[0].GetType(),
			}}
	case *proto.Expression_Literal_EmptyList:
		return &NestedLiteral[ListLiteralValue]{
			Value: nil,
			Type: &types.ListType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				Type:             types.TypeFromProto(lit.EmptyList.Type),
			}}
	case *proto.Expression_Literal_EmptyMap:
		return &MapLiteral{
			Value: nil,
			Type: &types.MapType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				Key:              types.TypeFromProto(lit.EmptyMap.Key),
				Value:            types.TypeFromProto(lit.EmptyMap.Value),
			}}
	case *proto.Expression_Literal_UserDefined_:
		params := make([]types.TypeParam, len(lit.UserDefined.TypeParameters))
		for i, p := range lit.UserDefined.TypeParameters {
			params[i] = types.TypeParamFromProto(p)
		}

		return &ProtoLiteral{
			Value: lit.UserDefined.Val,
			Type: &types.UserDefinedType{
				Nullability:      nullability,
				TypeVariationRef: l.TypeVariationReference,
				TypeReference:    lit.UserDefined.GetTypeReference(),
				TypeParameters:   params,
			},
		}
	case *proto.Expression_Literal_PrecisionTime_:
		precTime := lit.PrecisionTime
		precision, err := types.ProtoToTimePrecision(precTime.Precision)
		if err != nil {
			return nil
		}
		if precTime.Value < 0 {
			return nil
		}
		return NewPrecisionTimeLiteral(precTime.Value, precision, nullability)
	case *proto.Expression_Literal_PrecisionTimestamp_:
		precTimeStamp := lit.PrecisionTimestamp
		precision, err := types.ProtoToTimePrecision(precTimeStamp.Precision)
		if err != nil {
			return nil
		}
		if precTimeStamp.Value < 0 {
			return nil
		}
		return NewPrecisionTimestampLiteral(precTimeStamp.Value, precision, nullability)
	case *proto.Expression_Literal_PrecisionTimestampTz:
		precTimeStamp := lit.PrecisionTimestampTz
		precision, err := types.ProtoToTimePrecision(precTimeStamp.Precision)
		if err != nil {
			return nil
		}
		if precTimeStamp.Value < 0 {
			return nil
		}
		return NewPrecisionTimestampTzLiteral(precTimeStamp.Value, precision, nullability)
	case *proto.Expression_Literal_IntervalYearToMonth_:
		return intervalYearToMonthLiteralFromProto(l)
	case *proto.Expression_Literal_IntervalCompound_:
		return intervalCompoundLiteralFromProto(l)
	}
	panic("unimplemented literal type")
}

// NewPrecisionTimeLiteral creates a new PrecisionTime from an integer value, representing time since
// midnight, in the specified precision (nanoseconds to seconds).
func NewPrecisionTimeLiteral(value int64, precision types.TimePrecision, n types.Nullability) Literal {
	return &ProtoLiteral{
		Value: value,
		Type: &types.PrecisionTimeType{
			Precision:   precision,
			Nullability: n,
		},
	}
}

// NewPrecisionTimestampLiteral creates a new PrecisionTimestamp from an integer value, representing time since
// the unix epoch, in the specified precision (nanoseconds to seconds).
func NewPrecisionTimestampLiteral(value int64, precision types.TimePrecision, n types.Nullability) Literal {
	return &ProtoLiteral{
		Value: value,
		Type: &types.PrecisionTimestampType{
			Precision:   precision,
			Nullability: n,
		},
	}
}

// NewPrecisionTimestampTzLiteral it takes timestamp value which is in specified precision
// and nullable property (n) and returns a PrecisionTimestampTz Literal
func NewPrecisionTimestampTzLiteral(value int64, precision types.TimePrecision, n types.Nullability) Literal {
	return &ProtoLiteral{
		Value: value,
		Type: &types.PrecisionTimestampTzType{
			PrecisionTimestampType: types.PrecisionTimestampType{
				Precision:   precision,
				Nullability: n,
			},
		},
	}
}
