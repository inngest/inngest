// Code generated from SubstraitType.g4 by ANTLR 4.13.2. DO NOT EDIT.

package baseparser // SubstraitType
import "github.com/antlr4-go/antlr/v4"

// SubstraitTypeListener is a complete listener for a parse tree produced by SubstraitTypeParser.
type SubstraitTypeListener interface {
	antlr.ParseTreeListener

	// EnterStartRule is called when entering the startRule production.
	EnterStartRule(c *StartRuleContext)

	// EnterTypeStatement is called when entering the typeStatement production.
	EnterTypeStatement(c *TypeStatementContext)

	// EnterBoolean is called when entering the boolean production.
	EnterBoolean(c *BooleanContext)

	// EnterI8 is called when entering the i8 production.
	EnterI8(c *I8Context)

	// EnterI16 is called when entering the i16 production.
	EnterI16(c *I16Context)

	// EnterI32 is called when entering the i32 production.
	EnterI32(c *I32Context)

	// EnterI64 is called when entering the i64 production.
	EnterI64(c *I64Context)

	// EnterFp32 is called when entering the fp32 production.
	EnterFp32(c *Fp32Context)

	// EnterFp64 is called when entering the fp64 production.
	EnterFp64(c *Fp64Context)

	// EnterString is called when entering the string production.
	EnterString(c *StringContext)

	// EnterBinary is called when entering the binary production.
	EnterBinary(c *BinaryContext)

	// EnterTimestamp is called when entering the timestamp production.
	EnterTimestamp(c *TimestampContext)

	// EnterTimestampTz is called when entering the timestampTz production.
	EnterTimestampTz(c *TimestampTzContext)

	// EnterDate is called when entering the date production.
	EnterDate(c *DateContext)

	// EnterTime is called when entering the time production.
	EnterTime(c *TimeContext)

	// EnterIntervalYear is called when entering the intervalYear production.
	EnterIntervalYear(c *IntervalYearContext)

	// EnterUuid is called when entering the uuid production.
	EnterUuid(c *UuidContext)

	// EnterFixedChar is called when entering the fixedChar production.
	EnterFixedChar(c *FixedCharContext)

	// EnterVarChar is called when entering the varChar production.
	EnterVarChar(c *VarCharContext)

	// EnterFixedBinary is called when entering the fixedBinary production.
	EnterFixedBinary(c *FixedBinaryContext)

	// EnterDecimal is called when entering the decimal production.
	EnterDecimal(c *DecimalContext)

	// EnterPrecisionIntervalDay is called when entering the precisionIntervalDay production.
	EnterPrecisionIntervalDay(c *PrecisionIntervalDayContext)

	// EnterPrecisionIntervalCompound is called when entering the precisionIntervalCompound production.
	EnterPrecisionIntervalCompound(c *PrecisionIntervalCompoundContext)

	// EnterPrecisionTime is called when entering the precisionTime production.
	EnterPrecisionTime(c *PrecisionTimeContext)

	// EnterPrecisionTimestamp is called when entering the precisionTimestamp production.
	EnterPrecisionTimestamp(c *PrecisionTimestampContext)

	// EnterPrecisionTimestampTZ is called when entering the precisionTimestampTZ production.
	EnterPrecisionTimestampTZ(c *PrecisionTimestampTZContext)

	// EnterStruct is called when entering the struct production.
	EnterStruct(c *StructContext)

	// EnterNStruct is called when entering the nStruct production.
	EnterNStruct(c *NStructContext)

	// EnterList is called when entering the list production.
	EnterList(c *ListContext)

	// EnterMap is called when entering the map production.
	EnterMap(c *MapContext)

	// EnterFunc is called when entering the func production.
	EnterFunc(c *FuncContext)

	// EnterUserDefined is called when entering the userDefined production.
	EnterUserDefined(c *UserDefinedContext)

	// EnterSingleFuncParam is called when entering the singleFuncParam production.
	EnterSingleFuncParam(c *SingleFuncParamContext)

	// EnterFuncParamsWithParens is called when entering the funcParamsWithParens production.
	EnterFuncParamsWithParens(c *FuncParamsWithParensContext)

	// EnterNumericLiteral is called when entering the numericLiteral production.
	EnterNumericLiteral(c *NumericLiteralContext)

	// EnterNumericParameterName is called when entering the numericParameterName production.
	EnterNumericParameterName(c *NumericParameterNameContext)

	// EnterNumericExpression is called when entering the numericExpression production.
	EnterNumericExpression(c *NumericExpressionContext)

	// EnterAnyType is called when entering the anyType production.
	EnterAnyType(c *AnyTypeContext)

	// EnterTypeDef is called when entering the typeDef production.
	EnterTypeDef(c *TypeDefContext)

	// EnterIfExpr is called when entering the IfExpr production.
	EnterIfExpr(c *IfExprContext)

	// EnterTypeLiteral is called when entering the TypeLiteral production.
	EnterTypeLiteral(c *TypeLiteralContext)

	// EnterMultilineDefinition is called when entering the MultilineDefinition production.
	EnterMultilineDefinition(c *MultilineDefinitionContext)

	// EnterTernary is called when entering the Ternary production.
	EnterTernary(c *TernaryContext)

	// EnterBinaryExpr is called when entering the BinaryExpr production.
	EnterBinaryExpr(c *BinaryExprContext)

	// EnterParenExpression is called when entering the ParenExpression production.
	EnterParenExpression(c *ParenExpressionContext)

	// EnterParameterName is called when entering the ParameterName production.
	EnterParameterName(c *ParameterNameContext)

	// EnterFunctionCall is called when entering the FunctionCall production.
	EnterFunctionCall(c *FunctionCallContext)

	// EnterNotExpr is called when entering the NotExpr production.
	EnterNotExpr(c *NotExprContext)

	// EnterLiteralNumber is called when entering the LiteralNumber production.
	EnterLiteralNumber(c *LiteralNumberContext)

	// ExitStartRule is called when exiting the startRule production.
	ExitStartRule(c *StartRuleContext)

	// ExitTypeStatement is called when exiting the typeStatement production.
	ExitTypeStatement(c *TypeStatementContext)

	// ExitBoolean is called when exiting the boolean production.
	ExitBoolean(c *BooleanContext)

	// ExitI8 is called when exiting the i8 production.
	ExitI8(c *I8Context)

	// ExitI16 is called when exiting the i16 production.
	ExitI16(c *I16Context)

	// ExitI32 is called when exiting the i32 production.
	ExitI32(c *I32Context)

	// ExitI64 is called when exiting the i64 production.
	ExitI64(c *I64Context)

	// ExitFp32 is called when exiting the fp32 production.
	ExitFp32(c *Fp32Context)

	// ExitFp64 is called when exiting the fp64 production.
	ExitFp64(c *Fp64Context)

	// ExitString is called when exiting the string production.
	ExitString(c *StringContext)

	// ExitBinary is called when exiting the binary production.
	ExitBinary(c *BinaryContext)

	// ExitTimestamp is called when exiting the timestamp production.
	ExitTimestamp(c *TimestampContext)

	// ExitTimestampTz is called when exiting the timestampTz production.
	ExitTimestampTz(c *TimestampTzContext)

	// ExitDate is called when exiting the date production.
	ExitDate(c *DateContext)

	// ExitTime is called when exiting the time production.
	ExitTime(c *TimeContext)

	// ExitIntervalYear is called when exiting the intervalYear production.
	ExitIntervalYear(c *IntervalYearContext)

	// ExitUuid is called when exiting the uuid production.
	ExitUuid(c *UuidContext)

	// ExitFixedChar is called when exiting the fixedChar production.
	ExitFixedChar(c *FixedCharContext)

	// ExitVarChar is called when exiting the varChar production.
	ExitVarChar(c *VarCharContext)

	// ExitFixedBinary is called when exiting the fixedBinary production.
	ExitFixedBinary(c *FixedBinaryContext)

	// ExitDecimal is called when exiting the decimal production.
	ExitDecimal(c *DecimalContext)

	// ExitPrecisionIntervalDay is called when exiting the precisionIntervalDay production.
	ExitPrecisionIntervalDay(c *PrecisionIntervalDayContext)

	// ExitPrecisionIntervalCompound is called when exiting the precisionIntervalCompound production.
	ExitPrecisionIntervalCompound(c *PrecisionIntervalCompoundContext)

	// ExitPrecisionTime is called when exiting the precisionTime production.
	ExitPrecisionTime(c *PrecisionTimeContext)

	// ExitPrecisionTimestamp is called when exiting the precisionTimestamp production.
	ExitPrecisionTimestamp(c *PrecisionTimestampContext)

	// ExitPrecisionTimestampTZ is called when exiting the precisionTimestampTZ production.
	ExitPrecisionTimestampTZ(c *PrecisionTimestampTZContext)

	// ExitStruct is called when exiting the struct production.
	ExitStruct(c *StructContext)

	// ExitNStruct is called when exiting the nStruct production.
	ExitNStruct(c *NStructContext)

	// ExitList is called when exiting the list production.
	ExitList(c *ListContext)

	// ExitMap is called when exiting the map production.
	ExitMap(c *MapContext)

	// ExitFunc is called when exiting the func production.
	ExitFunc(c *FuncContext)

	// ExitUserDefined is called when exiting the userDefined production.
	ExitUserDefined(c *UserDefinedContext)

	// ExitSingleFuncParam is called when exiting the singleFuncParam production.
	ExitSingleFuncParam(c *SingleFuncParamContext)

	// ExitFuncParamsWithParens is called when exiting the funcParamsWithParens production.
	ExitFuncParamsWithParens(c *FuncParamsWithParensContext)

	// ExitNumericLiteral is called when exiting the numericLiteral production.
	ExitNumericLiteral(c *NumericLiteralContext)

	// ExitNumericParameterName is called when exiting the numericParameterName production.
	ExitNumericParameterName(c *NumericParameterNameContext)

	// ExitNumericExpression is called when exiting the numericExpression production.
	ExitNumericExpression(c *NumericExpressionContext)

	// ExitAnyType is called when exiting the anyType production.
	ExitAnyType(c *AnyTypeContext)

	// ExitTypeDef is called when exiting the typeDef production.
	ExitTypeDef(c *TypeDefContext)

	// ExitIfExpr is called when exiting the IfExpr production.
	ExitIfExpr(c *IfExprContext)

	// ExitTypeLiteral is called when exiting the TypeLiteral production.
	ExitTypeLiteral(c *TypeLiteralContext)

	// ExitMultilineDefinition is called when exiting the MultilineDefinition production.
	ExitMultilineDefinition(c *MultilineDefinitionContext)

	// ExitTernary is called when exiting the Ternary production.
	ExitTernary(c *TernaryContext)

	// ExitBinaryExpr is called when exiting the BinaryExpr production.
	ExitBinaryExpr(c *BinaryExprContext)

	// ExitParenExpression is called when exiting the ParenExpression production.
	ExitParenExpression(c *ParenExpressionContext)

	// ExitParameterName is called when exiting the ParameterName production.
	ExitParameterName(c *ParameterNameContext)

	// ExitFunctionCall is called when exiting the FunctionCall production.
	ExitFunctionCall(c *FunctionCallContext)

	// ExitNotExpr is called when exiting the NotExpr production.
	ExitNotExpr(c *NotExprContext)

	// ExitLiteralNumber is called when exiting the LiteralNumber production.
	ExitLiteralNumber(c *LiteralNumberContext)
}
