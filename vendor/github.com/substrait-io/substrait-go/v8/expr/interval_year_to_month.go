package expr

import (
	"fmt"

	"github.com/substrait-io/substrait-go/v8/types"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// IntervalYearToMonthLiteral implements Literal interface for interval year to month type
type IntervalYearToMonthLiteral struct {
	Years       int32
	Months      int32
	Nullability types.Nullability
}

func (m IntervalYearToMonthLiteral) getType() types.Type {
	return types.NewIntervalYearToMonthType().WithNullability(m.Nullability)
}

func (m IntervalYearToMonthLiteral) ToProtoLiteral() *proto.Expression_Literal {
	t := m.getType()
	return &proto.Expression_Literal{
		LiteralType: &proto.Expression_Literal_IntervalYearToMonth_{
			IntervalYearToMonth: &proto.Expression_Literal_IntervalYearToMonth{
				Years:  m.Years,
				Months: m.Months,
			},
		},
		Nullable:               t.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.GetTypeVariationReference(),
	}
}

func (m IntervalYearToMonthLiteral) ToProto() *proto.Expression {
	return &proto.Expression{RexType: &proto.Expression_Literal_{
		Literal: m.ToProtoLiteral(),
	}}
}

func intervalYearToMonthLiteralFromProto(l *proto.Expression_Literal) Literal {
	return IntervalYearToMonthLiteral{
		Years:       l.GetIntervalYearToMonth().Years,
		Months:      l.GetIntervalYearToMonth().Months,
		Nullability: getNullability(l.Nullable),
	}
}

func (IntervalYearToMonthLiteral) isRootRef()            {}
func (m IntervalYearToMonthLiteral) GetType() types.Type { return m.getType() }
func (m IntervalYearToMonthLiteral) String() string {
	return fmt.Sprintf("%s(%s)", m.getType(), m.ValueString())
}
func (m IntervalYearToMonthLiteral) ValueString() string {
	return fmt.Sprintf("%d years, %d months", m.Years, m.Months)
}
func (m IntervalYearToMonthLiteral) Equals(rhs Expression) bool {
	if other, ok := rhs.(IntervalYearToMonthLiteral); ok {
		return m.getType().Equals(other.GetType()) && (m == other)
	}
	return false
}

func (m IntervalYearToMonthLiteral) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: m.ToProto()},
	}
}

func (m IntervalYearToMonthLiteral) Visit(VisitFunc) Expression { return m }
func (IntervalYearToMonthLiteral) IsScalar() bool               { return true }
