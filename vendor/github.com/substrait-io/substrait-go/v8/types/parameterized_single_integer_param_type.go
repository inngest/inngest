// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"reflect"

	"github.com/substrait-io/substrait-go/v8/types/integer_parameters"
)

type singleIntegerParamType interface {
	BaseString() string
	ShortString() string
	GetReturnType(length int32, nullability Nullability) Type
}

// parameterizedTypeSingleIntegerParam This is a generic type to represent parameterized type with a single integer parameter
type parameterizedTypeSingleIntegerParam[T singleIntegerParamType] struct {
	Nullability      Nullability
	TypeVariationRef uint32
	IntegerOption    integer_parameters.IntegerParameter
}

func (m *parameterizedTypeSingleIntegerParam[T]) SetNullability(n Nullability) FuncDefArgType {
	m.Nullability = n
	return m
}

func (m *parameterizedTypeSingleIntegerParam[T]) String() string {
	return fmt.Sprintf("%s%s%s", m.baseString(), strFromNullability(m.Nullability), m.parameterString())
}

func (m *parameterizedTypeSingleIntegerParam[T]) parameterString() string {
	return fmt.Sprintf("<%s>", m.IntegerOption.String())
}

func (m *parameterizedTypeSingleIntegerParam[T]) baseString() string {
	var t T
	tType := reflect.TypeOf(t)
	if tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	newInstance := reflect.New(tType).Interface().(T)
	return newInstance.BaseString()
}

func (m *parameterizedTypeSingleIntegerParam[T]) HasParameterizedParam() bool {
	_, ok1 := m.IntegerOption.(*integer_parameters.VariableIntParam)
	return ok1
}

func (m *parameterizedTypeSingleIntegerParam[T]) GetParameterizedParams() []interface{} {
	if !m.HasParameterizedParam() {
		return nil
	}
	return []interface{}{m.IntegerOption}
}

func (m *parameterizedTypeSingleIntegerParam[T]) MatchWithNullability(ot Type) bool {
	if m.Nullability != ot.GetNullability() {
		return false
	}
	return m.MatchWithoutNullability(ot)
}

func (m *parameterizedTypeSingleIntegerParam[T]) MatchWithoutNullability(ot Type) bool {
	if reflect.TypeFor[T]() != reflect.TypeOf(ot) {
		return false
	}
	if odt, ok := ot.(FixedType); ok {
		concreteLength := integer_parameters.NewConcreteIntParam(odt.GetLength())
		return m.IntegerOption.IsCompatible(concreteLength)
	}
	if odt, ok := ot.(timestampPrecisionType); ok {
		concreteLength := integer_parameters.NewConcreteIntParam(odt.GetPrecision().ToProtoVal())
		return m.IntegerOption.IsCompatible(concreteLength)
	}
	return false
}

func (m *parameterizedTypeSingleIntegerParam[T]) GetNullability() Nullability {
	return m.Nullability
}

func (m *parameterizedTypeSingleIntegerParam[T]) ShortString() string {
	newInstance := m.getNewInstance()
	return newInstance.ShortString()
}

func (m *parameterizedTypeSingleIntegerParam[T]) getNewInstance() T {
	var t T
	tType := reflect.TypeOf(t)
	if tType.Kind() == reflect.Ptr {
		tType = tType.Elem()
	}
	return reflect.New(tType).Interface().(T)
}

func (m *parameterizedTypeSingleIntegerParam[T]) ReturnType(params []FuncDefArgType, argumentTypes []Type) (Type, error) {
	concreteIntParam, ok := m.IntegerOption.(*integer_parameters.ConcreteIntParam)
	if !ok {
		derivation := OutputDerivation{FinalType: m}
		return derivation.ReturnType(params, argumentTypes)
	}
	t := m.getNewInstance()
	return t.GetReturnType(int32(*concreteIntParam), m.Nullability), nil
}

func (m *parameterizedTypeSingleIntegerParam[T]) WithParameters(params []interface{}) (Type, error) {
	if len(params) != 1 {
		if concreteIntParam, ok := m.IntegerOption.(*integer_parameters.ConcreteIntParam); ok {
			return m.getNewInstance().GetReturnType(int32(*concreteIntParam), m.Nullability), nil
		}
		return nil, fmt.Errorf("type must have 1 parameter")
	}
	switch params[0].(type) {
	case int64:
		return m.getNewInstance().GetReturnType(int32(params[0].(int64)), m.Nullability), nil
	case int32:
		return m.getNewInstance().GetReturnType(params[0].(int32), m.Nullability), nil
	case TimePrecision:
		return m.getNewInstance().GetReturnType(int32(params[0].(TimePrecision)), m.Nullability), nil
	default:
		return nil, fmt.Errorf("unknown parameter type for integer parameter")
	}
}
