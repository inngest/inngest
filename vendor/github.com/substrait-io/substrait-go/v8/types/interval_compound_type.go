package types

import (
	"fmt"

	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// IntervalCompoundType this is used to represent a type of interval compound.
type IntervalCompoundType struct {
	precision        TimePrecision
	typeVariationRef uint32
	nullability      Nullability
}

// NewIntervalCompoundType creates a type of new interval compound.
func NewIntervalCompoundType() IntervalCompoundType {
	return IntervalCompoundType{}
}

func (m IntervalCompoundType) WithTypeVariationRef(typeVariationRef uint32) IntervalCompoundType {
	m.typeVariationRef = typeVariationRef
	return m
}

func (m IntervalCompoundType) GetPrecisionProtoVal() int32 {
	return m.precision.ToProtoVal()
}

func (IntervalCompoundType) isRootRef() {}
func (m IntervalCompoundType) WithNullability(n Nullability) Type {
	return IntervalCompoundType{
		precision:   m.precision,
		nullability: n,
	}
}

func (m IntervalCompoundType) WithPrecision(precision TimePrecision) IntervalCompoundType {
	return IntervalCompoundType{
		precision:   precision,
		nullability: m.nullability,
	}
}

func (m IntervalCompoundType) GetType() Type                     { return m }
func (m IntervalCompoundType) GetNullability() Nullability       { return m.nullability }
func (m IntervalCompoundType) GetTypeVariationReference() uint32 { return m.typeVariationRef }
func (m IntervalCompoundType) Equals(rhs Type) bool {
	if o, ok := rhs.(IntervalCompoundType); ok {
		return o == m
	}
	if o, ok := rhs.(*IntervalCompoundType); ok {
		return *o == m
	}
	return false
}

func (m IntervalCompoundType) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Type{Type: m.ToProto()},
	}
}

func (m IntervalCompoundType) ToProto() *proto.Type {
	return &proto.Type{Kind: &proto.Type_IntervalCompound_{
		IntervalCompound: &proto.Type_IntervalCompound{
			Precision:              m.precision.ToProtoVal(),
			Nullability:            m.nullability,
			TypeVariationReference: m.typeVariationRef}}}
}

func (IntervalCompoundType) ShortString() string { return shortTypeNames[TypeNameIntervalCompound] }
func (m IntervalCompoundType) String() string {
	return fmt.Sprintf("%s%s<%d>", TypeNameIntervalCompound, strNullable(m), m.precision.ToProtoVal())
}

func (m IntervalCompoundType) GetParameters() []interface{} {
	return []interface{}{m.precision}
}
