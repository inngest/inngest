package types

import (
	"fmt"
	"strings"

	"github.com/substrait-io/substrait-go/v8/types/integer_parameters"
)

type Expr interface {
	Evaluate(symbolTable map[string]any) (any, error)
	String() string
}

type LiteralNumber struct {
	Value int64
}

func (l *LiteralNumber) Evaluate(map[string]any) (any, error) {
	return l.Value, nil
}

func (l *LiteralNumber) String() string {
	return fmt.Sprintf("%d", l.Value)
}

type BinaryOp int

const (
	Unknown BinaryOp = iota
	And
	Or
	Plus
	Minus
	Multiply
	Divide
	LT
	LTE
	GT
	GTE
	EQ
	NEQ
)

func (b BinaryOp) String() string {
	switch b {
	case Plus:
		return "+"
	case Minus:
		return "-"
	case Multiply:
		return "*"
	case Divide:
		return "/"
	case LT:
		return "<"
	case LTE:
		return "<="
	case GT:
		return ">"
	case GTE:
		return ">="
	case EQ:
		return "="
	case NEQ:
		return "!="
	case And:
		return "and"
	case Or:
		return "or"
	default:
		panic("Invalid binary operator")
	}
}

func GetBinaryOpType(op string) BinaryOp {
	op = strings.ToLower(op)
	switch op {
	case "+":
		return Plus
	case "-":
		return Minus
	case "*":
		return Multiply
	case "/":
		return Divide
	case "<":
		return LT
	case "<=":
		return LTE
	case ">":
		return GT
	case ">=":
		return GTE
	case "=":
		return EQ
	case "!=":
		return NEQ
	case "and":
		return And
	case "or":
		return Or
	default:
		panic(fmt.Sprintf("Invalid binary operator %s", op))
	}
}

type BinaryExpr struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (b BinaryExpr) Evaluate(symbolTable map[string]any) (any, error) {
	left, err := b.Left.Evaluate(symbolTable)
	if err != nil {
		return nil, err
	}
	right, err := b.Right.Evaluate(symbolTable)
	if err != nil {
		return nil, err
	}
	switch b.Op {
	case Plus:
		return left.(int64) + right.(int64), nil
	case Minus:
		return left.(int64) - right.(int64), nil
	case Multiply:
		return left.(int64) * right.(int64), nil
	case Divide:
		return left.(int64) / right.(int64), nil
	case LT:
		return left.(int64) < right.(int64), nil
	case LTE:
		return left.(int64) <= right.(int64), nil
	case GT:
		return left.(int64) > right.(int64), nil
	case GTE:
		return left.(int64) >= right.(int64), nil
	case EQ:
		return left.(int64) == right.(int64), nil
	case NEQ:
		return left.(int64) != right.(int64), nil
	case And:
		return left.(bool) && right.(bool), nil
	case Or:
		return left.(bool) || right.(bool), nil
	default:
		panic("Invalid binary operator")
	}
}

func (b BinaryExpr) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Op, b.Right)
}

type IfExpr struct {
	Condition Expr
	Then      Expr
	Else      Expr
	IsTernary bool
}

func (i IfExpr) String() string {
	if i.IsTernary {
		return fmt.Sprintf("%s ? %s : %s", i.Condition, i.Then, i.Else)
	}
	return fmt.Sprintf("if %s then %s else %s", i.Condition, i.Then, i.Else)
}

func (i IfExpr) Evaluate(symbolTable map[string]any) (any, error) {
	condition, err := i.Condition.Evaluate(symbolTable)
	if err != nil {
		return nil, err
	}
	if condition.(bool) {
		return i.Then.Evaluate(symbolTable)
	}
	return i.Else.Evaluate(symbolTable)
}

type NotExpr struct {
	Expr Expr
}

func (n NotExpr) String() string {
	return fmt.Sprintf("!%s", n.Expr)
}

func (n NotExpr) Evaluate(symbolTable map[string]any) (any, error) {
	result, err := n.Expr.Evaluate(symbolTable)
	if err != nil {
		return nil, err
	}
	return !result.(bool), nil
}

type FunctionCallExpr struct {
	Name string
	Args []Expr
}

func (f FunctionCallExpr) String() string {
	sb := strings.Builder{}
	sb.WriteString(f.Name)
	sb.WriteString("(")
	for i, arg := range f.Args {
		sb.WriteString(arg.String())
		if i < len(f.Args)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func (f FunctionCallExpr) Evaluate(symbolTable map[string]any) (any, error) {
	args := make([]any, len(f.Args))
	for i, arg := range f.Args {
		result, err := arg.Evaluate(symbolTable)
		if err != nil {
			return nil, err
		}
		args[i] = result
	}
	switch f.Name {
	case "abs":
		if args[0].(int64) < 0 {
			return -args[0].(int64), nil
		}
		return args[0].(int64), nil
	case "max":
		return max(args[0].(int64), args[1].(int64)), nil
	case "min":
		return min(args[0].(int64), args[1].(int64)), nil
	default:
		return nil, fmt.Errorf("unknown function %s", f.Name)
	}
}

type Assignment struct {
	Name  string
	Value Expr
}

func (a Assignment) Evaluate(symbolTable map[string]any) error {
	result, err := a.Value.Evaluate(symbolTable)
	if err != nil {
		return err
	}
	symbolTable[a.Name] = result
	return nil
}

func (a Assignment) String() string {
	return fmt.Sprintf("%s = %s\n", a.Name, a.Value)
}

type OutputDerivation struct {
	Assignments []Assignment
	FinalType   FuncDefArgType
}

func (m *OutputDerivation) SetNullability(n Nullability) FuncDefArgType {
	m.FinalType.SetNullability(n)
	return m
}

func (m *OutputDerivation) String() string {
	sb := strings.Builder{}
	for _, a := range m.Assignments {
		sb.WriteString(a.String())
	}

	return sb.String() + m.FinalType.String()
}

func (m *OutputDerivation) HasParameterizedParam() bool {
	return m.FinalType.HasParameterizedParam()
}

func (m *OutputDerivation) GetParameterizedParams() []interface{} {
	return m.FinalType.GetParameterizedParams()
}

func (m *OutputDerivation) MatchWithNullability(ot Type) bool {
	if m.FinalType.GetNullability() != ot.GetNullability() {
		return false
	}
	return m.MatchWithoutNullability(ot)
}

func (m *OutputDerivation) MatchWithoutNullability(ot Type) bool {
	return m.FinalType.MatchWithoutNullability(ot)
}

func (m *OutputDerivation) GetNullability() Nullability {
	return m.FinalType.GetNullability()
}

func (m *OutputDerivation) ShortString() string {
	return m.FinalType.ShortString()
}

type SymbolInfo struct {
	Name  string
	Value any
}

func buildTypeParametersNameValueMap(funcParameters []FuncDefArgType, argumentTypes []Type) (map[string]any, error) {
	symbolTable := make(map[string]any)
	for i, p := range funcParameters {
		paramNames := p.GetParameterizedParams()
		if len(paramNames) > 0 {
			paramValues := argumentTypes[i].GetParameters()
			if len(paramNames) != len(paramValues) {
				return nil, fmt.Errorf("function parameter %s has %d parameters, but %d were provided", p.String(), len(paramNames), len(paramValues))
			}
			for j, param := range paramNames {
				if intParam, ok := param.(*integer_parameters.VariableIntParam); ok {
					name := string(*intParam)
					if existingValue, ok := symbolTable[name]; ok && existingValue != paramValues[j] {
						return nil, fmt.Errorf("sync parameters %s has conflicting values: %v and %v", name, existingValue, paramValues[j])
					}
					symbolTable[name] = paramValues[j]
					continue
				}
			}
		}
	}
	return symbolTable, nil
}

func (m *OutputDerivation) ReturnType(funcParameters []FuncDefArgType, argumentTypes []Type) (Type, error) {
	// Build a symbol table of parameterized parameters of arguments
	symbolTable, err := buildTypeParametersNameValueMap(funcParameters, argumentTypes)
	if err != nil {
		return nil, err
	}

	// Evaluate assignments
	for _, a := range m.Assignments {
		err := a.Evaluate(symbolTable)
		if err != nil {
			return nil, err
		}
	}

	// make slice of parameters for final type
	parametrizedParams := m.FinalType.GetParameterizedParams()
	params := make([]interface{}, 0, len(parametrizedParams))
	for _, p := range parametrizedParams {
		switch param := p.(type) {
		case *integer_parameters.VariableIntParam:
			if paramValue, ok := symbolTable[string(*param)]; ok {
				params = append(params, paramValue)
			} else {
				return nil, fmt.Errorf("parameter %s is not defined", param)
			}
		case *integer_parameters.ConcreteIntParam:
			params = append(params, int64(*param))
		default:
			params = append(params, param)
		}
	}

	return m.FinalType.WithParameters(params)
}

func (m *OutputDerivation) WithParameters([]interface{}) (Type, error) {
	panic("WithParameters not to be called")
}

func AreSyncTypeParametersMatching(funcParameters []FuncDefArgType, argumentTypes []Type) bool {
	_, err := buildTypeParametersNameValueMap(funcParameters, argumentTypes)
	return err == nil
}
