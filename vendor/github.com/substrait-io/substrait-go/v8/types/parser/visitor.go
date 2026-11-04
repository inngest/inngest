package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/substrait-io/substrait-go/v8/types"
	"github.com/substrait-io/substrait-go/v8/types/integer_parameters"
	baseparser2 "github.com/substrait-io/substrait-go/v8/types/parser/baseparser"
	"github.com/substrait-io/substrait-go/v8/types/parser/util"
)

type TypeVisitor struct {
	baseparser2.SubstraitTypeVisitor
	ErrorListener util.VisitErrorListener
}

var _ baseparser2.SubstraitTypeVisitor = &TypeVisitor{}

func (v *TypeVisitor) Visit(tree antlr.ParseTree) interface{} {
	if tree == nil {
		return nil
	}
	return tree.Accept(v)
}

func (v *TypeVisitor) VisitStartRule(ctx *baseparser2.StartRuleContext) interface{} {
	return v.Visit(ctx.Expr())
}

func (v *TypeVisitor) VisitTypeStatement(ctx *baseparser2.TypeStatementContext) interface{} {
	return v.Visit(ctx.TypeDef())
}

func (v *TypeVisitor) VisitParenExpression(ctx *baseparser2.ParenExpressionContext) interface{} {
	return v.Visit(ctx.Expr())
}

func (v *TypeVisitor) VisitTypeLiteral(ctx *baseparser2.TypeLiteralContext) interface{} {
	return v.Visit(ctx.TypeDef())
}

func (v *TypeVisitor) VisitLiteralNumber(ctx *baseparser2.LiteralNumberContext) interface{} {
	num, err := strconv.ParseInt(ctx.Number().GetText(), 10, 0)
	if err != nil {
		v.ErrorListener.ReportVisitError(ctx, fmt.Errorf("error parsing number: %s", err))
		return 0
	}
	return &types.LiteralNumber{Value: num}
}

func (v *TypeVisitor) VisitFunctionCall(ctx *baseparser2.FunctionCallContext) interface{} {
	args := make([]types.Expr, 0, len(ctx.AllExpr()))
	for _, expr := range ctx.AllExpr() {
		args = append(args, v.Visit(expr).(types.Expr))
	}
	return &types.FunctionCallExpr{
		Name: strings.ToLower(ctx.Identifier().GetText()),
		Args: args,
	}
}

func (v *TypeVisitor) VisitBinaryExpr(ctx *baseparser2.BinaryExprContext) interface{} {
	return &types.BinaryExpr{
		Op:    types.GetBinaryOpType(ctx.GetOp().GetText()),
		Left:  v.Visit(ctx.GetLeft()).(types.Expr),
		Right: v.Visit(ctx.GetRight()).(types.Expr),
	}
}

func (v *TypeVisitor) VisitIfExpr(ctx *baseparser2.IfExprContext) interface{} {
	return &types.IfExpr{
		Condition: v.Visit(ctx.GetIfExpr()).(types.Expr),
		Then:      v.Visit(ctx.GetThenExpr()).(types.Expr),
		Else:      v.Visit(ctx.GetElseExpr()).(types.Expr),
	}
}

func (v *TypeVisitor) VisitNotExpr(ctx *baseparser2.NotExprContext) interface{} {
	return &types.NotExpr{
		Expr: v.Visit(ctx.Expr()).(types.Expr),
	}
}

func (v *TypeVisitor) VisitTernary(ctx *baseparser2.TernaryContext) interface{} {
	return &types.IfExpr{
		Condition: v.Visit(ctx.GetIfExpr()).(types.Expr),
		Then:      v.Visit(ctx.GetThenExpr()).(types.Expr),
		Else:      v.Visit(ctx.GetElseExpr()).(types.Expr),
		IsTernary: true,
	}
}

func (v *TypeVisitor) VisitTypeDef(ctx *baseparser2.TypeDefContext) interface{} {
	if ctx.ParameterizedType() != nil {
		return v.Visit(ctx.ParameterizedType())
	}
	if ctx.ScalarType() != nil {
		nullability := types.NullabilityRequired
		if ctx.GetIsnull() != nil {
			nullability = types.NullabilityNullable
		}

		scalarTypeExpr := v.Visit(ctx.ScalarType())
		scalarType := scalarTypeExpr.(types.Type)
		return scalarType.WithNullability(nullability)
	}

	return v.Visit(ctx.AnyType())
}

func (v *TypeVisitor) VisitAnyType(ctx *baseparser2.AnyTypeContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	name := "any"
	if ctx.AnyVar() != nil {
		name = ctx.AnyVar().GetText()
	}
	return &types.AnyType{Name: name, Nullability: nullability}
}

func (v *TypeVisitor) VisitBoolean(*baseparser2.BooleanContext) interface{} {
	return &types.BooleanType{}
}

func (v *TypeVisitor) VisitI8(*baseparser2.I8Context) interface{} {
	return &types.Int8Type{}
}

func (v *TypeVisitor) VisitI16(*baseparser2.I16Context) interface{} {
	return &types.Int16Type{}
}

func (v *TypeVisitor) VisitI32(*baseparser2.I32Context) interface{} {
	return &types.Int32Type{}
}

func (v *TypeVisitor) VisitI64(*baseparser2.I64Context) interface{} {
	return &types.Int64Type{}
}

func (v *TypeVisitor) VisitFp32(*baseparser2.Fp32Context) interface{} {
	return &types.Float32Type{}
}

func (v *TypeVisitor) VisitFp64(*baseparser2.Fp64Context) interface{} {
	return &types.Float64Type{}
}

func (v *TypeVisitor) VisitString(*baseparser2.StringContext) interface{} {
	return &types.StringType{}
}

func (v *TypeVisitor) VisitBinary(*baseparser2.BinaryContext) interface{} {
	return &types.BinaryType{}
}

func (v *TypeVisitor) VisitTimestamp(*baseparser2.TimestampContext) interface{} {
	return &types.TimestampType{}
}

func (v *TypeVisitor) VisitTimestampTz(*baseparser2.TimestampTzContext) interface{} {
	return &types.TimestampTzType{}
}

func (v *TypeVisitor) VisitDate(*baseparser2.DateContext) interface{} {
	return &types.DateType{}
}

func (v *TypeVisitor) VisitTime(*baseparser2.TimeContext) interface{} {
	return &types.TimeType{}
}

func (v *TypeVisitor) VisitIntervalYear(*baseparser2.IntervalYearContext) interface{} {
	return &types.IntervalYearType{}
}

func (v *TypeVisitor) VisitUuid(*baseparser2.UuidContext) interface{} {
	return &types.UUIDType{}
}

func (v *TypeVisitor) VisitUserDefined(ctx *baseparser2.UserDefinedContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	var params []types.UDTParameter
	for _, expr := range ctx.AllExpr() {
		paramExpr := v.Visit(expr)
		switch param := paramExpr.(type) {
		case types.FuncDefArgType:
			params = append(params, &types.DataTypeUDTParam{Type: param})
		case *types.LiteralNumber:
			params = append(params, &types.IntegerUDTParam{Integer: int32(param.Value)})
		case types.StringParameter:
			params = append(params, &types.StringUDTParam{StringVal: string(param)})
		default:
			// TODO handle other user defined type parameters
			v.ErrorListener.ReportVisitError(ctx, fmt.Errorf(
				"User defined type parameter is not a FuncDefArgType/int/string, type %T ", param))
		}
	}
	name := ctx.Identifier().GetText()
	return &types.ParameterizedUserDefinedType{Name: name, Nullability: nullability, TypeParameters: params}
}

func (v *TypeVisitor) VisitFixedChar(ctx *baseparser2.FixedCharContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetLength()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedFixedCharType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitVarChar(ctx *baseparser2.VarCharContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetLength()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedVarCharType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitFixedBinary(ctx *baseparser2.FixedBinaryContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetLength()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedFixedBinaryType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitDecimal(ctx *baseparser2.DecimalContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	precision := v.Visit(ctx.GetPrecision()).(integer_parameters.IntegerParameter)
	scale := v.Visit(ctx.GetScale()).(integer_parameters.IntegerParameter)

	return &types.ParameterizedDecimalType{Precision: precision, Scale: scale, Nullability: nullability}
}

func (v *TypeVisitor) VisitPrecisionTime(ctx *baseparser2.PrecisionTimeContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetPrecision()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedPrecisionTimeType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitPrecisionTimestamp(ctx *baseparser2.PrecisionTimestampContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetPrecision()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedPrecisionTimestampType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitPrecisionTimestampTZ(ctx *baseparser2.PrecisionTimestampTZContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetPrecision()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedPrecisionTimestampTzType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitPrecisionIntervalDay(ctx *baseparser2.PrecisionIntervalDayContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	length := v.Visit(ctx.GetPrecision()).(integer_parameters.IntegerParameter)
	return &types.ParameterizedIntervalDayType{IntegerOption: length, Nullability: nullability}
}

func (v *TypeVisitor) VisitStruct(ctx *baseparser2.StructContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}
	var fieldTypes []types.FuncDefArgType
	for _, expr := range ctx.AllExpr() {
		fieldTypes = append(fieldTypes, v.Visit(expr).(types.FuncDefArgType))
	}
	return &types.ParameterizedStructType{Types: fieldTypes, Nullability: nullability}
}

func (v *TypeVisitor) VisitList(ctx *baseparser2.ListContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}
	elementType := v.Visit(ctx.Expr()).(types.FuncDefArgType)
	return &types.ParameterizedListType{Type: elementType, Nullability: nullability}
}

func (v *TypeVisitor) VisitNStruct(*baseparser2.NStructContext) interface{} {
	panic("implement me")
}

func (v *TypeVisitor) VisitMap(ctx *baseparser2.MapContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}
	keyType, keyOk := v.Visit(ctx.GetKey()).(types.FuncDefArgType)
	valueType, valueOk := v.Visit(ctx.GetValue()).(types.FuncDefArgType)
	if !keyOk || !valueOk {
		v.ErrorListener.ReportVisitError(ctx, fmt.Errorf("map key or value type is not a FuncDefArgType"))
	}
	return &types.ParameterizedMapType{Key: keyType, Value: valueType, Nullability: nullability}
}

func (v *TypeVisitor) VisitFunc(ctx *baseparser2.FuncContext) interface{} {
	nullability := types.NullabilityRequired
	if ctx.GetIsnull() != nil {
		nullability = types.NullabilityNullable
	}

	// Parse function parameters
	var paramTypes []types.FuncDefArgType
	if ctx.GetParams() != nil {
		paramsResult := v.Visit(ctx.GetParams())
		switch p := paramsResult.(type) {
		case []types.FuncDefArgType:
			paramTypes = p
		case types.FuncDefArgType:
			paramTypes = []types.FuncDefArgType{p}
		}
	}

	// Parse return type
	returnType := v.Visit(ctx.GetReturnType()).(types.FuncDefArgType)

	return &types.ParameterizedFuncType{
		Nullability: nullability,
		Parameters:  paramTypes,
		Return:      returnType,
	}
}

func (v *TypeVisitor) VisitSingleFuncParam(ctx *baseparser2.SingleFuncParamContext) interface{} {
	return v.Visit(ctx.Expr())
}

func (v *TypeVisitor) VisitFuncParamsWithParens(ctx *baseparser2.FuncParamsWithParensContext) interface{} {
	params := make([]types.FuncDefArgType, 0, len(ctx.AllExpr()))
	for _, expr := range ctx.AllExpr() {
		params = append(params, v.Visit(expr).(types.FuncDefArgType))
	}
	return params
}

func (v *TypeVisitor) VisitParameterName(ctx *baseparser2.ParameterNameContext) interface{} {
	return types.StringParameter(ctx.Identifier().GetText())
}

func (v *TypeVisitor) VisitNumericLiteral(ctx *baseparser2.NumericLiteralContext) interface{} {
	num, err := strconv.ParseInt(ctx.Number().GetText(), 10, 0)
	if err != nil {
		v.ErrorListener.ReportVisitError(ctx, fmt.Errorf("error parsing type parameter as number: %s", err))
		return integer_parameters.NewConcreteIntParam(0)
	}
	return integer_parameters.NewConcreteIntParam(int32(num))
}

func (v *TypeVisitor) VisitNumericParameterName(ctx *baseparser2.NumericParameterNameContext) interface{} {
	return integer_parameters.NewVariableIntParam(ctx.GetText())
}

func (v *TypeVisitor) VisitNumericExpression(ctx *baseparser2.NumericExpressionContext) interface{} {
	// TODO handle numeric expression
	return v.Visit(ctx.Expr())
}

func (v *TypeVisitor) VisitMultilineDefinition(ctx *baseparser2.MultilineDefinitionContext) interface{} {
	assignments := make([]types.Assignment, 0)
	for i, expr := range ctx.AllExpr() {
		parsedExpr := v.Visit(expr)
		assignment := types.Assignment{
			Name:  ctx.Identifier(i).GetText(),
			Value: parsedExpr.(types.Expr),
		}
		assignments = append(assignments, assignment)
	}
	return &types.OutputDerivation{
		Assignments: assignments,
		FinalType:   v.Visit(ctx.GetFinalType()).(types.FuncDefArgType),
	}
}
