// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strings"
)

// ParameterizedStructType is a parameter type struct
// example: Struct<Decimal(P,S), INT8> or Struct<INT8>.
type ParameterizedStructType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Types            []FuncDefArgType
}

func (m *ParameterizedStructType) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *ParameterizedStructType) String() string {
	sb := strings.Builder{}
	for i, typ := range m.Types {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(typ.String())
	}
	t := StructType{}
	parameterString := fmt.Sprintf("<%s>", sb.String())
	return fmt.Sprintf("%s%s%s", t.BaseString(), strFromNullability(m.Nullability), parameterString)
}

func (m *ParameterizedStructType) HasParameterizedParam() bool {
	for _, typ := range m.Types {
		if typ.HasParameterizedParam() {
			return true
		}
	}
	return false
}

func (m *ParameterizedStructType) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	var abstractParams []interface{}
	for _, typ := range m.Types {
		if typ.HasParameterizedParam() {
			abstractParams = append(abstractParams, typ)
		}
	}
	return abstractParams
}

func (m *ParameterizedStructType) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	if omt, ok := ot.(*StructType); ok {
		if len(m.Types) != len(omt.Types) {
			return false
		}
		for i, typ := range m.Types {
			if !typ.MatchWithNullability(omt.Types[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (m *ParameterizedStructType) MatchWithoutNullability(ot Type) bool {
	if omt, ok := ot.(*StructType); ok {
		if len(m.Types) != len(omt.Types) {
			return false
		}
		for i, typ := range m.Types {
			if !typ.MatchWithoutNullability(omt.Types[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (m *ParameterizedStructType) GetNullability() Nullability {
	return m.Nullability
}

func (m *ParameterizedStructType) ShortString() string {
	return "struct"
}

func (m *ParameterizedStructType) ReturnType(funcParams []FuncDefArgType, argTypes []Type) (Type, error) {
	var types []Type
	for _, typ := range m.Types {
		retType, err := typ.ReturnType(funcParams, argTypes)
		if err != nil {
			return nil, fmt.Errorf("error in struct field type: %w", err)
		}
		types = append(types, retType)

	}
	return &StructType{Nullability: m.Nullability, Types: types}, nil
}

func (m *ParameterizedStructType) WithParameters(params []interface{}) (Type, error) {
	if len(params) != len(m.Types) {
		return nil, fmt.Errorf("expected %d parameters, got %d", len(m.Types), len(params))
	}
	var types []Type
	for i, typ := range m.Types {
		t, ok := params[i].(Type)
		if !ok {
			return nil, fmt.Errorf("expected parameter to be of type Type, got %T", params[i])
		}
		itype, err := typ.WithParameters(t.GetParameters())
		if err != nil {
			return nil, err
		}
		types = append(types, itype)
	}
	return &StructType{Nullability: m.Nullability, Types: types}, nil
}
