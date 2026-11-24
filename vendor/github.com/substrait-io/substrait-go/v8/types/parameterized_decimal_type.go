// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"

	"github.com/substrait-io/substrait-go/v8/types/integer_parameters"
)

// ParameterizedDecimalType is a decimal type which to hold function arguments
// example: Decimal<P,S> or Decimal<P,0> or Decimal(10, 2)
type ParameterizedDecimalType struct {
	Nullability      Nullability
	TypeVariationRef uint32
	Precision        integer_parameters.IntegerParameter
	Scale            integer_parameters.IntegerParameter
}

func (m *ParameterizedDecimalType) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *ParameterizedDecimalType) String() string {
	t := DecimalType{}
	parameterString := fmt.Sprintf("<%s,%s>", m.Precision.String(), m.Scale.String())
	return fmt.Sprintf("%s%s%s", t.BaseString(), strFromNullability(m.Nullability), parameterString)
}

func (m *ParameterizedDecimalType) HasParameterizedParam() bool {
	_, ok1 := m.Precision.(*integer_parameters.VariableIntParam)
	_, ok2 := m.Scale.(*integer_parameters.VariableIntParam)
	return ok1 || ok2
}

func (m *ParameterizedDecimalType) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	var params []interface{}
	params = append(params, m.Precision)
	params = append(params, m.Scale)
	return params
}

func (m *ParameterizedDecimalType) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	return m.MatchWithoutNullability(ot)
}

func (m *ParameterizedDecimalType) MatchWithoutNullability(ot Type) bool {
	if odt, ok := ot.(*DecimalType); ok {
		concretePrecision := integer_parameters.NewConcreteIntParam(odt.Precision)
		concreteScale := integer_parameters.NewConcreteIntParam(odt.Scale)
		return m.Precision.IsCompatible(concretePrecision) && m.Scale.IsCompatible(concreteScale)
	}
	return false
}

func (m *ParameterizedDecimalType) GetNullability() Nullability {
	return m.Nullability
}

func (m *ParameterizedDecimalType) ShortString() string {
	return "dec"
}

func (m *ParameterizedDecimalType) ReturnType(parameters []FuncDefArgType, argumentTypes []Type) (Type, error) {
	precision, perr := m.Precision.(*integer_parameters.ConcreteIntParam)
	scale, serr := m.Scale.(*integer_parameters.ConcreteIntParam)
	if !perr || !serr {
		derivation := OutputDerivation{FinalType: m}
		return derivation.ReturnType(parameters, argumentTypes)
	}
	return &DecimalType{Nullability: m.Nullability, Precision: int32(*precision), Scale: int32(*scale)}, nil
}

func (m *ParameterizedDecimalType) WithParameters(params []interface{}) (Type, error) {
	if len(params) != 2 {
		p, pOk := m.Precision.(*integer_parameters.ConcreteIntParam)
		s, sOk := m.Scale.(*integer_parameters.ConcreteIntParam)
		if pOk && sOk {
			return &DecimalType{Nullability: m.Nullability, Precision: int32(*p), Scale: int32(*s)}, nil
		}
		return nil, fmt.Errorf("decimal type must have 2 parameters")
	}
	if precision, ok := params[0].(int64); ok {
		if scale, ok := params[1].(int64); ok {
			return &DecimalType{Nullability: m.Nullability, Precision: int32(precision), Scale: int32(scale)}, nil
		}
		return nil, fmt.Errorf("scale must be an integer")
	}
	return nil, fmt.Errorf("precision must be an integer, but got %t", params[0])
}
