// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strings"
)

type UDTParameter interface {
	isTypeParameter() bool
	String() string
	toTypeParam() (TypeParam, error)
	MatchWithoutNullability(param TypeParam) bool
	MatchWithNullability(param TypeParam) bool
}

type DataTypeUDTParam struct {
	Type FuncDefArgType
}

func (d *DataTypeUDTParam) isTypeParameter() bool { return true }

func (d *DataTypeUDTParam) String() string {
	return d.Type.String()
}

func (d *DataTypeUDTParam) toTypeParam() (TypeParam, error) {
	return d.toTypeParamWithArgs(nil, nil)
}

func (d *DataTypeUDTParam) toTypeParamWithArgs(funcParams []FuncDefArgType, argTypes []Type) (TypeParam, error) {
	typ, err := d.Type.ReturnType(funcParams, argTypes)
	if err != nil {
		return nil, err
	}
	return &DataTypeParameter{Type: typ}, nil
}

func (d *DataTypeUDTParam) MatchWithNullability(param TypeParam) bool {
	if d.MatchWithoutNullability(param) {
		if dataParam, ok := param.(*DataTypeParameter); ok {
			return d.Type.GetNullability() == dataParam.Type.GetNullability()
		}
	}
	return false
}

func (d *DataTypeUDTParam) MatchWithoutNullability(param TypeParam) bool {
	if dataParam, ok := param.(*DataTypeParameter); ok {
		return d.Type.MatchWithoutNullability(dataParam.Type)
	}
	return false
}

type IntegerUDTParam struct {
	Integer int32
}

func (i *IntegerUDTParam) isTypeParameter() bool { return true }

func (i *IntegerUDTParam) String() string {
	return fmt.Sprintf("%d", i.Integer)
}

func (i *IntegerUDTParam) toTypeParam() (TypeParam, error) {
	return IntegerParameter(i.Integer), nil
}

func (i *IntegerUDTParam) MatchWithoutNullability(param TypeParam) bool {
	if intParam, ok := param.(IntegerParameter); ok {
		return i.Integer == int32(intParam)
	}
	return false
}

func (i *IntegerUDTParam) MatchWithNullability(param TypeParam) bool {
	return i.MatchWithoutNullability(param)
}

type StringUDTParam struct {
	StringVal string
}

func (s *StringUDTParam) isTypeParameter() bool { return true }

func (s *StringUDTParam) String() string {
	return s.StringVal
}

func (s *StringUDTParam) toTypeParam() (TypeParam, error) {
	return StringParameter(s.StringVal), nil
}

func (s *StringUDTParam) MatchWithoutNullability(param TypeParam) bool {
	if strParam, ok := param.(StringParameter); ok {
		return s.StringVal == string(strParam)
	}
	return false
}

func (s *StringUDTParam) MatchWithNullability(param TypeParam) bool {
	return s.MatchWithoutNullability(param)
}

// ParameterizedUserDefinedType is a parameter type struct
// example: U!point<Decimal(P,S), DECIMAL(P,S)> or U!square<INT8>.
type ParameterizedUserDefinedType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	TypeParameters   []UDTParameter
	Name             string
}

func (m *ParameterizedUserDefinedType) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *ParameterizedUserDefinedType) String() string {
	var parameterString string
	if len(m.TypeParameters) > 0 {
		sb := strings.Builder{}
		for i, typ := range m.TypeParameters {
			if i != 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(typ.String())
		}
		parameterString = fmt.Sprintf("<%s>", sb.String())
	}
	return fmt.Sprintf("u!%s%s%s", m.Name, strFromNullability(m.Nullability), parameterString)
}

func (m *ParameterizedUserDefinedType) HasParameterizedParam() bool {
	for _, typ := range m.TypeParameters {
		if param, ok := typ.(*DataTypeUDTParam); ok {
			return param.Type.HasParameterizedParam()
		}
	}
	return false
}

func (m *ParameterizedUserDefinedType) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	var abstractParams []interface{}
	for _, typ := range m.TypeParameters {
		if param, ok := typ.(*DataTypeUDTParam); ok && param.Type.HasParameterizedParam() {
			abstractParams = append(abstractParams, typ)
		}
	}
	return abstractParams
}

func (m *ParameterizedUserDefinedType) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	if udt, ok := ot.(*UserDefinedType); ok {
		if len(m.TypeParameters) != len(udt.TypeParameters) {
			return false
		}
		for i, param := range m.TypeParameters {
			if !param.MatchWithNullability(udt.TypeParameters[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (m *ParameterizedUserDefinedType) MatchWithoutNullability(ot Type) bool {
	if udt, ok := ot.(*UserDefinedType); ok {
		if len(m.TypeParameters) != len(udt.TypeParameters) {
			return false
		}
		for i, param := range m.TypeParameters {
			if !param.MatchWithoutNullability(udt.TypeParameters[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (m *ParameterizedUserDefinedType) GetNullability() Nullability {
	return m.Nullability
}

func (m *ParameterizedUserDefinedType) ShortString() string {
	return fmt.Sprintf("u!%s", m.Name)
}

func (m *ParameterizedUserDefinedType) ReturnType(funcParams []FuncDefArgType, argTypes []Type) (Type, error) {
	var types []TypeParam
	for _, udtParam := range m.TypeParameters {
		var param TypeParam
		var err error
		if dataParam, ok := udtParam.(*DataTypeUDTParam); ok {
			param, err = dataParam.toTypeParamWithArgs(funcParams, argTypes)
		} else {
			param, err = udtParam.toTypeParam()
		}
		if err != nil {
			return nil, err
		}
		types = append(types, param)
	}
	return &UserDefinedType{Nullability: m.Nullability, TypeParameters: types}, nil
}

func (m *ParameterizedUserDefinedType) WithParameters(params []interface{}) (Type, error) {
	if len(params) != len(m.TypeParameters) {
		return nil, fmt.Errorf("expected %d parameters, got %d", len(m.TypeParameters), len(params))
	}
	var typeParams []TypeParam
	for i, param := range params {
		if p, ok := param.(TypeParam); ok {
			typeParams = append(typeParams, p)
			continue
		}
		return nil, fmt.Errorf("unexpected type %T for parameter %d", param, i)
	}
	return &UserDefinedType{
		Nullability:      m.Nullability,
		TypeVariationRef: m.TypeVariationRef,
		TypeParameters:   typeParams,
	}, nil
}
