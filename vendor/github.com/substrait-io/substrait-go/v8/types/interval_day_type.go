package types

import (
	"fmt"

	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// IntervalDayType this is used to represent a type of interval day.
type IntervalDayType struct {
	Precision        TimePrecision
	TypeVariationRef uint32
	Nullability      Nullability
}

func (m *IntervalDayType) GetPrecisionProtoVal() int32 {
	return m.Precision.ToProtoVal()
}

func (*IntervalDayType) isRootRef() {}
func (m *IntervalDayType) WithNullability(n Nullability) Type {
	m.Nullability = n
	return m
}

func (m *IntervalDayType) GetType() Type                     { return m }
func (m *IntervalDayType) GetNullability() Nullability       { return m.Nullability }
func (m *IntervalDayType) GetTypeVariationReference() uint32 { return m.TypeVariationRef }
func (m *IntervalDayType) Equals(rhs Type) bool {
	if o, ok := rhs.(*IntervalDayType); ok {
		return *o == *m
	}
	return false
}

func (m *IntervalDayType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: m.ToProto()},
	}
}

func (m *IntervalDayType) ToProto() *proto.Type {
	precisionVal := m.Precision.ToProtoVal()
	return &proto.Type{Kind: &proto.Type_IntervalDay_{
		IntervalDay: &proto.Type_IntervalDay{
			Precision:              &precisionVal,
			Nullability:            m.Nullability,
			TypeVariationReference: m.TypeVariationRef}}}
}

func (*IntervalDayType) ShortString() string { return shortTypeNames[TypeNameIntervalDay] }

func (m *IntervalDayType) String() string {
	return fmt.Sprintf("%s%s<%d>", TypeNameIntervalDay, strNullable(m),
		m.Precision.ToProtoVal())
}

func (m *IntervalDayType) ParameterString() string {
	return fmt.Sprintf("%d", m.Precision.ToProtoVal())
}

func (s *IntervalDayType) BaseString() string {
	return string(TypeNameIntervalDay)
}

func (m *IntervalDayType) GetPrecision() TimePrecision {
	return m.Precision
}

func (m *IntervalDayType) GetReturnType(length int32, nullability Nullability) Type {
	out := *m
	out.Precision = TimePrecision(length)
	out.Nullability = nullability
	return &out
}

func (m *IntervalDayType) GetParameters() []interface{} {
	return []interface{}{m.Precision}
}
