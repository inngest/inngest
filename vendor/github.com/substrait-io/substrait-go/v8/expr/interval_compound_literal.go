package expr

import (
	"errors"
	"fmt"

	"github.com/substrait-io/substrait-go/v8/types"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// IntervalCompoundLiteral creates an interval compound literal
type IntervalCompoundLiteral struct {
	Years              int32
	Months             int32
	Days               int32
	Seconds            int32
	SubSeconds         int64
	SubSecondPrecision types.TimePrecision
	Nullability        types.Nullability
}

func (m IntervalCompoundLiteral) getType() types.Type {
	return types.NewIntervalCompoundType().WithPrecision(m.SubSecondPrecision).WithNullability(m.Nullability)
}

func (m IntervalCompoundLiteral) ToProtoLiteral() *proto.Expression_Literal {
	t := m.getType()
	intrCompPB := &proto.Expression_Literal_IntervalCompound{}

	if m.Years != 0 || m.Months != 0 {
		yearToMonthProto := &proto.Expression_Literal_IntervalYearToMonth{
			Years:  m.Years,
			Months: m.Months,
		}
		intrCompPB.IntervalYearToMonth = yearToMonthProto
	}

	if m.Days != 0 || m.Seconds != 0 || m.SubSeconds != 0 {
		dayToSecondProto := &proto.Expression_Literal_IntervalDayToSecond{
			Days:          m.Days,
			Seconds:       m.Seconds,
			PrecisionMode: &proto.Expression_Literal_IntervalDayToSecond_Precision{Precision: m.SubSecondPrecision.ToProtoVal()},
			Subseconds:    m.SubSeconds,
		}
		intrCompPB.IntervalDayToSecond = dayToSecondProto
	}

	return &proto.Expression_Literal{
		LiteralType:            &proto.Expression_Literal_IntervalCompound_{IntervalCompound: intrCompPB},
		Nullable:               t.GetNullability() == types.NullabilityNullable,
		TypeVariationReference: t.GetTypeVariationReference(),
	}
}

func (m IntervalCompoundLiteral) ToProto() *proto.Expression {
	return &proto.Expression{RexType: &proto.Expression_Literal_{
		Literal: m.ToProtoLiteral(),
	}}
}

func intervalCompoundLiteralFromProto(l *proto.Expression_Literal) Literal {
	icLiteral := IntervalCompoundLiteral{Nullability: getNullability(l.Nullable)}
	yearToMonth := l.GetIntervalCompound().GetIntervalYearToMonth()
	if yearToMonth != nil {
		icLiteral.Years = yearToMonth.Years
		icLiteral.Months = yearToMonth.Months
	}
	dayToSecond := l.GetIntervalCompound().GetIntervalDayToSecond()
	if dayToSecond == nil {
		// no day to second part
		return icLiteral
	}
	err := validateIntervalDayToSecondProto(dayToSecond)
	if err != nil {
		return nil
	}

	// get subSecond/precision value from proto. To get value it takes care of deprecated microseconds
	precision, subSeconds, err := intervalCompoundPrecisionSubSecondsFromProto(dayToSecond)
	if err != nil {
		return nil
	}
	icLiteral.Days = dayToSecond.Days
	icLiteral.Seconds = dayToSecond.Seconds
	icLiteral.SubSeconds = subSeconds
	icLiteral.SubSecondPrecision = precision
	return icLiteral
}

func (IntervalCompoundLiteral) isRootRef()            {}
func (m IntervalCompoundLiteral) GetType() types.Type { return m.getType() }
func (m IntervalCompoundLiteral) String() string {
	return fmt.Sprintf("%s(%s)", m.getType(), m.ValueString())
}
func (m IntervalCompoundLiteral) ValueString() string {
	return fmt.Sprintf("%d years, %d months, %d days, %d seconds, %d subseconds",
		m.Years, m.Months, m.Days, m.Seconds, m.SubSeconds)
}
func (m IntervalCompoundLiteral) Equals(rhs Expression) bool {
	if other, ok := rhs.(IntervalCompoundLiteral); ok {
		return m.getType().Equals(other.GetType()) && (m == other)
	}
	return false
}

func (m IntervalCompoundLiteral) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: m.ToProto()},
	}
}

func (m IntervalCompoundLiteral) Visit(VisitFunc) Expression { return m }
func (IntervalCompoundLiteral) IsScalar() bool               { return true }

func validateIntervalDayToSecondProto(idts *proto.Expression_Literal_IntervalDayToSecond) error {
	if idts.PrecisionMode == nil {
		// error, precision mode must be set for intervalCompound
		return errors.New("missing precision mode for interval compound")
	}
	if _, ok := idts.PrecisionMode.(*proto.Expression_Literal_IntervalDayToSecond_Microseconds); ok {
		// if microsecond precision then subseconds must be set to zero
		if idts.Subseconds > 0 {
			return errors.New("both deprecated microseconds and subseconds can't be non zero")
		}
	}
	return nil
}

func intervalCompoundPrecisionSubSecondsFromProto(protoVal *proto.Expression_Literal_IntervalDayToSecond) (types.TimePrecision, int64, error) {
	var precisionVal int32
	var subSecondVal int64
	switch pmt := protoVal.PrecisionMode.(type) {
	case *proto.Expression_Literal_IntervalDayToSecond_Precision:
		precisionVal = pmt.Precision
		subSecondVal = protoVal.Subseconds
	case *proto.Expression_Literal_IntervalDayToSecond_Microseconds:
		// deprecated field microsecond is set, treat its value subsecond
		precisionVal = types.PrecisionMicroSeconds.ToProtoVal()
		subSecondVal = int64(pmt.Microseconds)
	}
	precision, err := types.ProtoToTimePrecision(precisionVal)
	if err != nil {
		return types.PrecisionUnknown, 0, err
	}
	return precision, subSecondVal, nil
}
