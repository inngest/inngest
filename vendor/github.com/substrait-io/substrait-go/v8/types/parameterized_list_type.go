// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
)

// ParameterizedListType is a list type having parameter of ParameterizedAbstractType
// basically a list of which type is another abstract parameter
// example: List<Decimal(P,S)>. Kindly note concrete types List<Decimal(38, 0)> is not represented by this type
// Concrete type is represented by ListType
type ParameterizedListType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Type             FuncDefArgType
}

func (m *ParameterizedListType) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *ParameterizedListType) String() string {
	t := ListType{}
	parameterString := fmt.Sprintf("<%s>", m.Type)
	return fmt.Sprintf("%s%s%s", t.BaseString(), strFromNullability(m.Nullability), parameterString)
}

func (m *ParameterizedListType) HasParameterizedParam() bool {
	return m.Type.HasParameterizedParam()
}

func (m *ParameterizedListType) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	return []interface{}{m.Type}
}

func (m *ParameterizedListType) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	if olt, ok := ot.(*ListType); ok {
		result := m.Type.MatchWithNullability(olt.Type)
		return result
	}
	return false
}

func (m *ParameterizedListType) MatchWithoutNullability(ot Type) bool {
	if olt, ok := ot.(*ListType); ok {
		return m.Type.MatchWithoutNullability(olt.Type)
	}
	return false
}

func (m *ParameterizedListType) GetNullability() Nullability {
	return m.Nullability
}

func (m *ParameterizedListType) ShortString() string {
	return "list"
}

func (m *ParameterizedListType) ReturnType(
	funcParams []FuncDefArgType, argTypes []Type,
) (Type, error) {
	elemType, err := m.Type.ReturnType(funcParams, argTypes)
	if err != nil {
		return nil, err
	}
	return &ListType{Nullability: m.Nullability, Type: elemType}, nil
}

func (m *ParameterizedListType) WithParameters(params []interface{}) (Type, error) {
	if len(params) != 1 {
		return nil, fmt.Errorf("expected 1 parameter, got %d", len(params))
	}
	if t, ok := params[0].(Type); ok {
		return &ListType{Nullability: m.Nullability, Type: t}, nil
	}
	return nil, fmt.Errorf("expected parameter to be of type Type, got %T", params[0])
}
