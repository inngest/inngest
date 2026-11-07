// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
)

// ParameterizedMapType is a struct having at least one of key or value of type ParameterizedAbstractType
// If All arguments are concrete they are represented by MapType
type ParameterizedMapType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Key              FuncDefArgType
	Value            FuncDefArgType
}

func (m *ParameterizedMapType) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *ParameterizedMapType) String() string {
	t := MapType{}
	parameterString := fmt.Sprintf("<%s, %s>", m.Key.String(), m.Value.String())
	return fmt.Sprintf("%s%s%s", t.BaseString(), strFromNullability(m.Nullability), parameterString)
}

func (m *ParameterizedMapType) HasParameterizedParam() bool {
	return m.Key.HasParameterizedParam() || m.Value.HasParameterizedParam()
}

func (m *ParameterizedMapType) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	var abstractParams []interface{}
	if m.Key.HasParameterizedParam() {
		abstractParams = append(abstractParams, m.Key)
	}
	if m.Value.HasParameterizedParam() {
		abstractParams = append(abstractParams, m.Value)
	}
	return abstractParams
}

func (m *ParameterizedMapType) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	if omt, ok := ot.(*MapType); ok {
		return m.Key.MatchWithNullability(omt.Key) && m.Value.MatchWithNullability(omt.Value)
	}
	return false
}

func (m *ParameterizedMapType) MatchWithoutNullability(ot Type) bool {
	if omt, ok := ot.(*MapType); ok {
		return m.Key.MatchWithoutNullability(omt.Key) && m.Value.MatchWithoutNullability(omt.Value)
	}
	return false
}

func (m *ParameterizedMapType) GetNullability() Nullability {
	return m.Nullability
}

func (m *ParameterizedMapType) ShortString() string {
	return "map"
}

func (m *ParameterizedMapType) ReturnType(funcParams []FuncDefArgType, argTypes []Type) (Type, error) {
	keyType, kerr := m.Key.ReturnType(funcParams, argTypes)
	if kerr != nil {
		return nil, fmt.Errorf("error in getting key type: %w", kerr)
	}
	valueType, verr := m.Value.ReturnType(funcParams, argTypes)
	if verr != nil {
		return nil, fmt.Errorf("error in getting value type: %w", verr)
	}

	return &MapType{Nullability: m.Nullability, Key: keyType, Value: valueType}, nil
}

func (m *ParameterizedMapType) WithParameters(params []interface{}) (Type, error) {
	if len(params) != 2 {
		if m.Key.HasParameterizedParam() || m.Value.HasParameterizedParam() {
			return nil, fmt.Errorf("map type must have 2 parameters")
		}
		return m.ReturnType(nil, nil)
	}
	if key, ok := params[0].(Type); ok {
		if value, ok := params[1].(Type); ok {
			return &MapType{Nullability: m.Nullability, Key: key, Value: value}, nil
		}
		return nil, fmt.Errorf("value must be a Type")
	}
	return nil, fmt.Errorf("key must be a Type")
}
