// Code generated from SubstraitType.g4 by ANTLR 4.13.2. DO NOT EDIT.

package baseparser // SubstraitType
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by SubstraitTypeParser.
type SubstraitTypeVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by SubstraitTypeParser#startRule.
	VisitStartRule(ctx *StartRuleContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#typeStatement.
	VisitTypeStatement(ctx *TypeStatementContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#boolean.
	VisitBoolean(ctx *BooleanContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#i8.
	VisitI8(ctx *I8Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#i16.
	VisitI16(ctx *I16Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#i32.
	VisitI32(ctx *I32Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#i64.
	VisitI64(ctx *I64Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#fp32.
	VisitFp32(ctx *Fp32Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#fp64.
	VisitFp64(ctx *Fp64Context) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#string.
	VisitString(ctx *StringContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#binary.
	VisitBinary(ctx *BinaryContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#timestamp.
	VisitTimestamp(ctx *TimestampContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#timestampTz.
	VisitTimestampTz(ctx *TimestampTzContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#date.
	VisitDate(ctx *DateContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#time.
	VisitTime(ctx *TimeContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#intervalYear.
	VisitIntervalYear(ctx *IntervalYearContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#uuid.
	VisitUuid(ctx *UuidContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#fixedChar.
	VisitFixedChar(ctx *FixedCharContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#varChar.
	VisitVarChar(ctx *VarCharContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#fixedBinary.
	VisitFixedBinary(ctx *FixedBinaryContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#decimal.
	VisitDecimal(ctx *DecimalContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#precisionIntervalDay.
	VisitPrecisionIntervalDay(ctx *PrecisionIntervalDayContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#precisionIntervalCompound.
	VisitPrecisionIntervalCompound(ctx *PrecisionIntervalCompoundContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#precisionTime.
	VisitPrecisionTime(ctx *PrecisionTimeContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#precisionTimestamp.
	VisitPrecisionTimestamp(ctx *PrecisionTimestampContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#precisionTimestampTZ.
	VisitPrecisionTimestampTZ(ctx *PrecisionTimestampTZContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#struct.
	VisitStruct(ctx *StructContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#nStruct.
	VisitNStruct(ctx *NStructContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#list.
	VisitList(ctx *ListContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#map.
	VisitMap(ctx *MapContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#func.
	VisitFunc(ctx *FuncContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#userDefined.
	VisitUserDefined(ctx *UserDefinedContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#singleFuncParam.
	VisitSingleFuncParam(ctx *SingleFuncParamContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#funcParamsWithParens.
	VisitFuncParamsWithParens(ctx *FuncParamsWithParensContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#numericLiteral.
	VisitNumericLiteral(ctx *NumericLiteralContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#numericParameterName.
	VisitNumericParameterName(ctx *NumericParameterNameContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#numericExpression.
	VisitNumericExpression(ctx *NumericExpressionContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#anyType.
	VisitAnyType(ctx *AnyTypeContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#typeDef.
	VisitTypeDef(ctx *TypeDefContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#IfExpr.
	VisitIfExpr(ctx *IfExprContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#TypeLiteral.
	VisitTypeLiteral(ctx *TypeLiteralContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#MultilineDefinition.
	VisitMultilineDefinition(ctx *MultilineDefinitionContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#Ternary.
	VisitTernary(ctx *TernaryContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#BinaryExpr.
	VisitBinaryExpr(ctx *BinaryExprContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#ParenExpression.
	VisitParenExpression(ctx *ParenExpressionContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#ParameterName.
	VisitParameterName(ctx *ParameterNameContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#FunctionCall.
	VisitFunctionCall(ctx *FunctionCallContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#NotExpr.
	VisitNotExpr(ctx *NotExprContext) interface{}

	// Visit a parse tree produced by SubstraitTypeParser#LiteralNumber.
	VisitLiteralNumber(ctx *LiteralNumberContext) interface{}
}
