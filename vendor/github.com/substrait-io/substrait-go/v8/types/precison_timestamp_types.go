// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"reflect"
	"time"

	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// TimePrecision is used to represent the precision of a timestamp
type TimePrecision int32

const (
	PrecisionUnknown TimePrecision = -1
	// below precision values are proto values
	PrecisionSeconds        TimePrecision = 0
	PrecisionDeciSeconds    TimePrecision = 1
	PrecisionCentiSeconds   TimePrecision = 2
	PrecisionMilliSeconds   TimePrecision = 3
	PrecisionEMinus4Seconds TimePrecision = 4 // 10^-4 of seconds
	PrecisionEMinus5Seconds TimePrecision = 5 // 10^-5 of seconds
	PrecisionMicroSeconds   TimePrecision = 6
	PrecisionEMinus7Seconds TimePrecision = 7 // 10^-7 of seconds
	PrecisionEMinus8Seconds TimePrecision = 8 // 10^-8 of seconds
	PrecisionNanoSeconds    TimePrecision = 9
)

func (m TimePrecision) ToProtoVal() int32 {
	return int32(m)
}

func SubSecondsToDuration(subSeconds int64, precision TimePrecision) time.Duration {
	switch precision {
	case PrecisionSeconds:
		return time.Duration(subSeconds) * time.Second
	case PrecisionDeciSeconds:
		return time.Duration(subSeconds) * time.Second / 10
	case PrecisionCentiSeconds:
		return time.Duration(subSeconds) * time.Second / 100
	case PrecisionMilliSeconds:
		return time.Duration(subSeconds) * time.Millisecond
	case PrecisionEMinus4Seconds:
		return time.Duration(subSeconds) * 100 * time.Microsecond
	case PrecisionEMinus5Seconds:
		return time.Duration(subSeconds) * 10 * time.Microsecond
	case PrecisionMicroSeconds:
		return time.Duration(subSeconds) * time.Microsecond
	case PrecisionEMinus7Seconds:
		return time.Duration(subSeconds) * 100 * time.Nanosecond
	case PrecisionEMinus8Seconds:
		return time.Duration(subSeconds) * 10 * time.Nanosecond
	case PrecisionNanoSeconds:
		return time.Duration(subSeconds) * time.Nanosecond
	default:
		panic(fmt.Sprintf("invalid precision %d", precision))
	}
}

func ProtoToTimePrecision(val int32) (TimePrecision, error) {
	if val < PrecisionSeconds.ToProtoVal() || val > PrecisionNanoSeconds.ToProtoVal() {
		return PrecisionUnknown, fmt.Errorf("invalid TimePrecision value %d", val)
	}
	return TimePrecision(val), nil
}

// PrecisionTimestampType this is used to represent a type of Precision timestamp
type PrecisionTimestampType struct {
	Precision        TimePrecision
	TypeVariationRef uint32
	Nullability      Nullability
}

// NewPrecisionTimestampType creates a type of new Precision timestamp.
// Created type has Nullability as Nullable
func NewPrecisionTimestampType(precision TimePrecision) *PrecisionTimestampType {
	return &PrecisionTimestampType{
		Precision:   precision,
		Nullability: NullabilityNullable,
	}
}

func (m *PrecisionTimestampType) GetPrecisionProtoVal() int32 {
	return m.Precision.ToProtoVal()
}

func (*PrecisionTimestampType) isRootRef() {}
func (m *PrecisionTimestampType) WithNullability(n Nullability) Type {
	return m.withNullability(n)
}

func (m *PrecisionTimestampType) GetParameters() []interface{} {
	return []interface{}{m.Precision}
}

func (m *PrecisionTimestampType) withNullability(n Nullability) *PrecisionTimestampType {
	return &PrecisionTimestampType{
		Precision:   m.Precision,
		Nullability: n,
	}
}

func (m *PrecisionTimestampType) GetType() Type                     { return m }
func (m *PrecisionTimestampType) GetNullability() Nullability       { return m.Nullability }
func (m *PrecisionTimestampType) GetTypeVariationReference() uint32 { return m.TypeVariationRef }
func (m *PrecisionTimestampType) Equals(rhs Type) bool {
	if o, ok := rhs.(*PrecisionTimestampType); ok {
		return *o == *m
	}
	return false
}

func (m *PrecisionTimestampType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: m.ToProto()},
	}
}

func (m *PrecisionTimestampType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_PrecisionTimestamp_{
		PrecisionTimestamp: &proto.Type_PrecisionTimestamp{
			Precision:              m.Precision.ToProtoVal(),
			Nullability:            m.Nullability,
			TypeVariationReference: m.TypeVariationRef}}}
}

func (*PrecisionTimestampType) ShortString() string {
	return GetShortTypeName(TypeNamePrecisionTimestamp)
}
func (m *PrecisionTimestampType) String() string {
	return fmt.Sprintf("%s%s<%d>", TypeNamePrecisionTimestamp, strNullable(m),
		m.Precision.ToProtoVal())
}

func (m *PrecisionTimestampType) ParameterString() string {
	return fmt.Sprintf("%d", m.Precision.ToProtoVal())
}

func (m *PrecisionTimestampType) BaseString() string {
	return typeNames[reflect.TypeOf(m)]
}

func (m *PrecisionTimestampType) GetPrecision() TimePrecision {
	return m.Precision
}

func (m *PrecisionTimestampType) GetReturnType(length int32, nullability Nullability) Type {
	out := *m
	out.Precision = TimePrecision(length)
	out.Nullability = nullability
	return &out
}

// PrecisionTimestampTzType this is used to represent a type of Precision timestamp with TimeZone
type PrecisionTimestampTzType struct {
	PrecisionTimestampType
}

// NewPrecisionTimestampTzType creates a type of new Precision timestamp with TimeZone.
// Created type has Nullability as Nullable
func NewPrecisionTimestampTzType(precision TimePrecision) *PrecisionTimestampTzType {
	return &PrecisionTimestampTzType{
		PrecisionTimestampType: PrecisionTimestampType{
			Precision:   precision,
			Nullability: NullabilityNullable,
		},
	}
}

func (m *PrecisionTimestampTzType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: m.ToProto()},
	}
}

func (m *PrecisionTimestampTzType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_PrecisionTimestampTz{
		PrecisionTimestampTz: &proto.Type_PrecisionTimestampTZ{
			Precision:              m.Precision.ToProtoVal(),
			Nullability:            m.Nullability,
			TypeVariationReference: m.TypeVariationRef}}}
}

func (m *PrecisionTimestampTzType) String() string {
	return fmt.Sprintf("%s%s<%d>", TypeNamePrecisionTimestampTz, strNullable(m),
		m.Precision.ToProtoVal())
}

func (m *PrecisionTimestampTzType) WithNullability(n Nullability) Type {
	return &PrecisionTimestampTzType{
		PrecisionTimestampType: *m.PrecisionTimestampType.withNullability(n),
	}
}

func (m *PrecisionTimestampTzType) GetParameters() []interface{} {
	return []interface{}{m.Precision}
}

func (m *PrecisionTimestampTzType) Equals(rhs Type) bool {
	if o, ok := rhs.(*PrecisionTimestampTzType); ok {
		return *o == *m
	}
	return false
}

func (m *PrecisionTimestampTzType) GetNullability() Nullability {
	return m.Nullability
}

func (*PrecisionTimestampTzType) ShortString() string {
	return GetShortTypeName(TypeNamePrecisionTimestampTz)
}

func (m *PrecisionTimestampTzType) BaseString() string {
	return typeNames[reflect.TypeOf(m)]
}

func (m *PrecisionTimestampTzType) GetReturnType(length int32, nullability Nullability) Type {
	out := *m
	out.Precision = TimePrecision(length)
	out.Nullability = nullability
	return &out
}

type PrecisionTimeType struct {
	Precision        TimePrecision
	TypeVariationRef uint32
	Nullability      Nullability
}

func NewPrecisionTimeType(precision TimePrecision) *PrecisionTimeType {
	return &PrecisionTimeType{
		Precision:   precision,
		Nullability: NullabilityNullable,
	}
}

func (m *PrecisionTimeType) GetPrecisionProtoVal() int32 {
	return m.Precision.ToProtoVal()
}

func (*PrecisionTimeType) isRootRef() {}
func (m *PrecisionTimeType) WithNullability(n Nullability) Type {
	return m.withNullability(n)
}

func (m *PrecisionTimeType) GetParameters() []interface{} {
	return []interface{}{m.Precision}
}

func (m *PrecisionTimeType) withNullability(n Nullability) *PrecisionTimeType {
	return &PrecisionTimeType{
		Precision:   m.Precision,
		Nullability: n,
	}
}

func (m *PrecisionTimeType) GetType() Type                     { return m }
func (m *PrecisionTimeType) GetNullability() Nullability       { return m.Nullability }
func (m *PrecisionTimeType) GetTypeVariationReference() uint32 { return m.TypeVariationRef }
func (m *PrecisionTimeType) Equals(rhs Type) bool {
	if o, ok := rhs.(*PrecisionTimeType); ok {
		return *o == *m
	}
	return false
}

func (m *PrecisionTimeType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: m.ToProto()},
	}
}

func (m *PrecisionTimeType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_PrecisionTime_{
		PrecisionTime: &proto.Type_PrecisionTime{
			Precision:              m.Precision.ToProtoVal(),
			Nullability:            m.Nullability,
			TypeVariationReference: m.TypeVariationRef}}}
}

func (*PrecisionTimeType) ShortString() string {
	return GetShortTypeName(TypeNamePrecisionTime)
}

func (m *PrecisionTimeType) String() string {
	return fmt.Sprintf("%s%s<%d>", TypeNamePrecisionTime, strNullable(m),
		m.Precision.ToProtoVal())
}

func (m *PrecisionTimeType) ParameterString() string {
	return fmt.Sprintf("%d", m.Precision.ToProtoVal())
}

func (m *PrecisionTimeType) BaseString() string {
	return typeNames[reflect.TypeOf(m)]
}

func (m *PrecisionTimeType) GetPrecision() TimePrecision {
	return m.Precision
}

func (m *PrecisionTimeType) GetReturnType(length int32, nullability Nullability) Type {
	out := *m
	out.Precision = TimePrecision(length)
	out.Nullability = nullability
	return &out
}
