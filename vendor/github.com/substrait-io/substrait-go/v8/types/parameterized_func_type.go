// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"strings"
)

// ParameterizedFuncType is a parameterized function type
// example: func<any1 -> any2> or func<i32, i32 -> i64> or func<T -> R>
type ParameterizedFuncType struct {
	Nullability Nullability
	Parameters  []FuncDefArgType
	Return      FuncDefArgType
}

// SetNullability sets the nullability and returns the modified type
func (f *ParameterizedFuncType) SetNullability(n Nullability) FuncDefArgType {
	f.Nullability = n
	return f
}

// String returns the string representation like "func<any1 -> any2>"
func (f *ParameterizedFuncType) String() string {
	var b strings.Builder
	b.WriteString("func")
	b.WriteString(strFromNullability(f.Nullability))
	b.WriteByte('<')
	for i, p := range f.Parameters {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.String())
	}
	b.WriteString(" -> ")
	b.WriteString(f.Return.String())
	b.WriteByte('>')
	return b.String()
}

// HasParameterizedParam returns true if any parameter or return type has parameterized parameters
func (f *ParameterizedFuncType) HasParameterizedParam() bool {
	for _, pt := range f.Parameters {
		if pt.HasParameterizedParam() {
			return true
		}
	}
	return f.Return.HasParameterizedParam()
}

// GetParameterizedParams returns all parameterized parameters
func (f *ParameterizedFuncType) GetParameterizedParams() []interface{} {
	if !f.HasParameterizedParam() {
		return nil
	}
	var abstractParams []interface{}
	for _, pt := range f.Parameters {
		if pt.HasParameterizedParam() {
			abstractParams = append(abstractParams, pt)
		}
	}
	if f.Return.HasParameterizedParam() {
		abstractParams = append(abstractParams, f.Return)
	}
	return abstractParams
}

// MatchWithNullability checks if this parameterized type matches the given concrete type
// including nullability
func (f *ParameterizedFuncType) MatchWithNullability(ot Type) bool {
	if f.Nullability != ot.GetNullability() {
		return false
	}
	return f.MatchWithoutNullability(ot)
}

// MatchWithoutNullability checks if this parameterized type matches the given concrete type
// ignoring nullability
func (f *ParameterizedFuncType) MatchWithoutNullability(ot Type) bool {
	oft, ok := ot.(*FuncType)
	if !ok {
		return false
	}
	if len(f.Parameters) != len(oft.ParameterTypes) {
		return false
	}
	for i, pt := range f.Parameters {
		if !pt.MatchWithoutNullability(oft.ParameterTypes[i]) {
			return false
		}
	}
	return f.Return.MatchWithoutNullability(oft.ReturnType)
}

// GetNullability returns the nullability of this type
func (f *ParameterizedFuncType) GetNullability() Nullability {
	return f.Nullability
}

// ShortString returns the short string representation "func"
func (f *ParameterizedFuncType) ShortString() string {
	return "func"
}

// ReturnType resolves the parameterized type to a concrete type given function parameters and argument types
func (f *ParameterizedFuncType) ReturnType(funcParameters []FuncDefArgType, argumentTypes []Type) (Type, error) {
	// Build concrete parameter types
	concreteParams := make([]Type, len(f.Parameters))
	for i, pt := range f.Parameters {
		ct, err := pt.ReturnType(funcParameters, argumentTypes)
		if err != nil {
			return nil, fmt.Errorf("error resolving func parameter type %d: %w", i, err)
		}
		concreteParams[i] = ct
	}

	// Build concrete return type
	concreteReturn, err := f.Return.ReturnType(funcParameters, argumentTypes)
	if err != nil {
		return nil, fmt.Errorf("error resolving func return type: %w", err)
	}

	return &FuncType{
		Nullability:    f.Nullability,
		ParameterTypes: concreteParams,
		ReturnType:     concreteReturn,
	}, nil
}

// WithParameters returns a concrete type by substituting the given parameters
func (f *ParameterizedFuncType) WithParameters(params []interface{}) (Type, error) {
	// Build concrete parameter types by resolving each parameterized parameter type
	concreteParams := make([]Type, len(f.Parameters))
	for i, pt := range f.Parameters {
		ct, err := pt.WithParameters(params)
		if err != nil {
			return nil, fmt.Errorf("error resolving func parameter type %d with parameters: %w", i, err)
		}
		concreteParams[i] = ct
	}

	// Build concrete return type
	concreteReturn, err := f.Return.WithParameters(params)
	if err != nil {
		return nil, fmt.Errorf("error resolving func return type with parameters: %w", err)
	}

	return &FuncType{
		Nullability:    f.Nullability,
		ParameterTypes: concreteParams,
		ReturnType:     concreteReturn,
	}, nil
}
