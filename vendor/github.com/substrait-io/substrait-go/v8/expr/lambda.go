package expr

import (
	"fmt"
	"strings"

	"github.com/substrait-io/substrait-go/v8/types"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// Lambda represents a lambda expression with parameters and a body.
type Lambda struct {
	Parameters *types.StructType // The formal lambda parameters, required to have NULLABILITY_REQUIRED
	Body       Expression
}

func (l *Lambda) String() string {
	var b strings.Builder
	b.WriteString("(")
	for i, t := range l.Parameters.Types {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "$%d: %s", i, t)
	}
	b.WriteString(") -> ")
	b.WriteString(l.Body.String())
	return b.String()
}

func (l *Lambda) isRootRef() {}

func (l *Lambda) IsScalar() bool {
	return false
}

func (l *Lambda) GetType() types.Type {
	return &types.FuncType{
		Nullability:    types.NullabilityRequired,
		ParameterTypes: l.Parameters.Types,
		ReturnType:     l.Body.GetType(),
	}
}

func (l *Lambda) Equals(other Expression) bool {
	rhs, ok := other.(*Lambda)
	if !ok {
		return false
	}
	return l.Parameters.Equals(rhs.Parameters) && l.Body.Equals(rhs.Body)
}

func (l *Lambda) ToProto() *proto.Expression {
	children := make([]*proto.Type, len(l.Parameters.Types))
	for i, c := range l.Parameters.Types {
		children[i] = types.TypeToProto(c)
	}
	paramsProto := &proto.Type_Struct{
		Types:                  children,
		TypeVariationReference: l.Parameters.TypeVariationRef,
		Nullability:            l.Parameters.Nullability,
	}

	return &proto.Expression{
		RexType: &proto.Expression_Lambda_{
			Lambda: &proto.Expression_Lambda{
				Parameters: paramsProto,
				Body:       l.Body.ToProto(),
			},
		},
	}
}

func (l *Lambda) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: l.ToProto()},
	}
}

func (l *Lambda) Visit(visit VisitFunc) Expression {
	newBody := visit(l.Body)
	if newBody == l.Body {
		return l
	}
	return &Lambda{Parameters: l.Parameters, Body: newBody}
}
