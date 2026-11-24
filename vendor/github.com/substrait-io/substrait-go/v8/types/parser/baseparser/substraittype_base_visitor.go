// Code generated from SubstraitType.g4 by ANTLR 4.13.2. DO NOT EDIT.

package baseparser // SubstraitType
import "github.com/antlr4-go/antlr/v4"

type BaseSubstraitTypeVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseSubstraitTypeVisitor) VisitStartRule(ctx *StartRuleContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTypeStatement(ctx *TypeStatementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitBoolean(ctx *BooleanContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitI8(ctx *I8Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitI16(ctx *I16Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitI32(ctx *I32Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitI64(ctx *I64Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFp32(ctx *Fp32Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFp64(ctx *Fp64Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitString(ctx *StringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitBinary(ctx *BinaryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTimestamp(ctx *TimestampContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTimestampTz(ctx *TimestampTzContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitDate(ctx *DateContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTime(ctx *TimeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitIntervalYear(ctx *IntervalYearContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitUuid(ctx *UuidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFixedChar(ctx *FixedCharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitVarChar(ctx *VarCharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFixedBinary(ctx *FixedBinaryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitDecimal(ctx *DecimalContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitPrecisionIntervalDay(ctx *PrecisionIntervalDayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitPrecisionIntervalCompound(ctx *PrecisionIntervalCompoundContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitPrecisionTime(ctx *PrecisionTimeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitPrecisionTimestamp(ctx *PrecisionTimestampContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitPrecisionTimestampTZ(ctx *PrecisionTimestampTZContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitStruct(ctx *StructContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitNStruct(ctx *NStructContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitList(ctx *ListContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitMap(ctx *MapContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFunc(ctx *FuncContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitUserDefined(ctx *UserDefinedContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitSingleFuncParam(ctx *SingleFuncParamContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFuncParamsWithParens(ctx *FuncParamsWithParensContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitNumericLiteral(ctx *NumericLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitNumericParameterName(ctx *NumericParameterNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitNumericExpression(ctx *NumericExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitAnyType(ctx *AnyTypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTypeDef(ctx *TypeDefContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitIfExpr(ctx *IfExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTypeLiteral(ctx *TypeLiteralContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitMultilineDefinition(ctx *MultilineDefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitTernary(ctx *TernaryContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitBinaryExpr(ctx *BinaryExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitParenExpression(ctx *ParenExpressionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitParameterName(ctx *ParameterNameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitFunctionCall(ctx *FunctionCallContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitNotExpr(ctx *NotExprContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseSubstraitTypeVisitor) VisitLiteralNumber(ctx *LiteralNumberContext) interface{} {
	return v.VisitChildren(ctx)
}
