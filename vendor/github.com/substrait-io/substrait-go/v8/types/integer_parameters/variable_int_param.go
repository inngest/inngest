// SPDX-License-Identifier: Apache-2.0

package integer_parameters

// VariableIntParam represents an integer parameter for a parameterized type
// Example: VARCHAR(L1) -> L1 is an VariableIntParam
// DECIMAL<P, 0> --> P Is an VariableIntParam
type VariableIntParam string

func NewVariableIntParam(s string) IntegerParameter {
	m := VariableIntParam(s)
	return &m
}

func (m *VariableIntParam) IsCompatible(o IntegerParameter) bool {
	switch o.(type) {
	case *VariableIntParam, *ConcreteIntParam:
		return true
	default:
		return false
	}
}

func (m *VariableIntParam) String() string {
	return string(*m)
}

func (m *VariableIntParam) GetAbstractParamName() string {
	return string(*m)
}
