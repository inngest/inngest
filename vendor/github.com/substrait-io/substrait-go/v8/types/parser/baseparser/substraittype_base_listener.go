// Code generated from SubstraitType.g4 by ANTLR 4.13.2. DO NOT EDIT.

package baseparser // SubstraitType
import "github.com/antlr4-go/antlr/v4"

// BaseSubstraitTypeListener is a complete listener for a parse tree produced by SubstraitTypeParser.
type BaseSubstraitTypeListener struct{}

var _ SubstraitTypeListener = &BaseSubstraitTypeListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseSubstraitTypeListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseSubstraitTypeListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseSubstraitTypeListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseSubstraitTypeListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterStartRule is called when production startRule is entered.
func (s *BaseSubstraitTypeListener) EnterStartRule(ctx *StartRuleContext) {}

// ExitStartRule is called when production startRule is exited.
func (s *BaseSubstraitTypeListener) ExitStartRule(ctx *StartRuleContext) {}

// EnterTypeStatement is called when production typeStatement is entered.
func (s *BaseSubstraitTypeListener) EnterTypeStatement(ctx *TypeStatementContext) {}

// ExitTypeStatement is called when production typeStatement is exited.
func (s *BaseSubstraitTypeListener) ExitTypeStatement(ctx *TypeStatementContext) {}

// EnterBoolean is called when production boolean is entered.
func (s *BaseSubstraitTypeListener) EnterBoolean(ctx *BooleanContext) {}

// ExitBoolean is called when production boolean is exited.
func (s *BaseSubstraitTypeListener) ExitBoolean(ctx *BooleanContext) {}

// EnterI8 is called when production i8 is entered.
func (s *BaseSubstraitTypeListener) EnterI8(ctx *I8Context) {}

// ExitI8 is called when production i8 is exited.
func (s *BaseSubstraitTypeListener) ExitI8(ctx *I8Context) {}

// EnterI16 is called when production i16 is entered.
func (s *BaseSubstraitTypeListener) EnterI16(ctx *I16Context) {}

// ExitI16 is called when production i16 is exited.
func (s *BaseSubstraitTypeListener) ExitI16(ctx *I16Context) {}

// EnterI32 is called when production i32 is entered.
func (s *BaseSubstraitTypeListener) EnterI32(ctx *I32Context) {}

// ExitI32 is called when production i32 is exited.
func (s *BaseSubstraitTypeListener) ExitI32(ctx *I32Context) {}

// EnterI64 is called when production i64 is entered.
func (s *BaseSubstraitTypeListener) EnterI64(ctx *I64Context) {}

// ExitI64 is called when production i64 is exited.
func (s *BaseSubstraitTypeListener) ExitI64(ctx *I64Context) {}

// EnterFp32 is called when production fp32 is entered.
func (s *BaseSubstraitTypeListener) EnterFp32(ctx *Fp32Context) {}

// ExitFp32 is called when production fp32 is exited.
func (s *BaseSubstraitTypeListener) ExitFp32(ctx *Fp32Context) {}

// EnterFp64 is called when production fp64 is entered.
func (s *BaseSubstraitTypeListener) EnterFp64(ctx *Fp64Context) {}

// ExitFp64 is called when production fp64 is exited.
func (s *BaseSubstraitTypeListener) ExitFp64(ctx *Fp64Context) {}

// EnterString is called when production string is entered.
func (s *BaseSubstraitTypeListener) EnterString(ctx *StringContext) {}

// ExitString is called when production string is exited.
func (s *BaseSubstraitTypeListener) ExitString(ctx *StringContext) {}

// EnterBinary is called when production binary is entered.
func (s *BaseSubstraitTypeListener) EnterBinary(ctx *BinaryContext) {}

// ExitBinary is called when production binary is exited.
func (s *BaseSubstraitTypeListener) ExitBinary(ctx *BinaryContext) {}

// EnterTimestamp is called when production timestamp is entered.
func (s *BaseSubstraitTypeListener) EnterTimestamp(ctx *TimestampContext) {}

// ExitTimestamp is called when production timestamp is exited.
func (s *BaseSubstraitTypeListener) ExitTimestamp(ctx *TimestampContext) {}

// EnterTimestampTz is called when production timestampTz is entered.
func (s *BaseSubstraitTypeListener) EnterTimestampTz(ctx *TimestampTzContext) {}

// ExitTimestampTz is called when production timestampTz is exited.
func (s *BaseSubstraitTypeListener) ExitTimestampTz(ctx *TimestampTzContext) {}

// EnterDate is called when production date is entered.
func (s *BaseSubstraitTypeListener) EnterDate(ctx *DateContext) {}

// ExitDate is called when production date is exited.
func (s *BaseSubstraitTypeListener) ExitDate(ctx *DateContext) {}

// EnterTime is called when production time is entered.
func (s *BaseSubstraitTypeListener) EnterTime(ctx *TimeContext) {}

// ExitTime is called when production time is exited.
func (s *BaseSubstraitTypeListener) ExitTime(ctx *TimeContext) {}

// EnterIntervalYear is called when production intervalYear is entered.
func (s *BaseSubstraitTypeListener) EnterIntervalYear(ctx *IntervalYearContext) {}

// ExitIntervalYear is called when production intervalYear is exited.
func (s *BaseSubstraitTypeListener) ExitIntervalYear(ctx *IntervalYearContext) {}

// EnterUuid is called when production uuid is entered.
func (s *BaseSubstraitTypeListener) EnterUuid(ctx *UuidContext) {}

// ExitUuid is called when production uuid is exited.
func (s *BaseSubstraitTypeListener) ExitUuid(ctx *UuidContext) {}

// EnterFixedChar is called when production fixedChar is entered.
func (s *BaseSubstraitTypeListener) EnterFixedChar(ctx *FixedCharContext) {}

// ExitFixedChar is called when production fixedChar is exited.
func (s *BaseSubstraitTypeListener) ExitFixedChar(ctx *FixedCharContext) {}

// EnterVarChar is called when production varChar is entered.
func (s *BaseSubstraitTypeListener) EnterVarChar(ctx *VarCharContext) {}

// ExitVarChar is called when production varChar is exited.
func (s *BaseSubstraitTypeListener) ExitVarChar(ctx *VarCharContext) {}

// EnterFixedBinary is called when production fixedBinary is entered.
func (s *BaseSubstraitTypeListener) EnterFixedBinary(ctx *FixedBinaryContext) {}

// ExitFixedBinary is called when production fixedBinary is exited.
func (s *BaseSubstraitTypeListener) ExitFixedBinary(ctx *FixedBinaryContext) {}

// EnterDecimal is called when production decimal is entered.
func (s *BaseSubstraitTypeListener) EnterDecimal(ctx *DecimalContext) {}

// ExitDecimal is called when production decimal is exited.
func (s *BaseSubstraitTypeListener) ExitDecimal(ctx *DecimalContext) {}

// EnterPrecisionIntervalDay is called when production precisionIntervalDay is entered.
func (s *BaseSubstraitTypeListener) EnterPrecisionIntervalDay(ctx *PrecisionIntervalDayContext) {}

// ExitPrecisionIntervalDay is called when production precisionIntervalDay is exited.
func (s *BaseSubstraitTypeListener) ExitPrecisionIntervalDay(ctx *PrecisionIntervalDayContext) {}

// EnterPrecisionIntervalCompound is called when production precisionIntervalCompound is entered.
func (s *BaseSubstraitTypeListener) EnterPrecisionIntervalCompound(ctx *PrecisionIntervalCompoundContext) {
}

// ExitPrecisionIntervalCompound is called when production precisionIntervalCompound is exited.
func (s *BaseSubstraitTypeListener) ExitPrecisionIntervalCompound(ctx *PrecisionIntervalCompoundContext) {
}

// EnterPrecisionTime is called when production precisionTime is entered.
func (s *BaseSubstraitTypeListener) EnterPrecisionTime(ctx *PrecisionTimeContext) {}

// ExitPrecisionTime is called when production precisionTime is exited.
func (s *BaseSubstraitTypeListener) ExitPrecisionTime(ctx *PrecisionTimeContext) {}

// EnterPrecisionTimestamp is called when production precisionTimestamp is entered.
func (s *BaseSubstraitTypeListener) EnterPrecisionTimestamp(ctx *PrecisionTimestampContext) {}

// ExitPrecisionTimestamp is called when production precisionTimestamp is exited.
func (s *BaseSubstraitTypeListener) ExitPrecisionTimestamp(ctx *PrecisionTimestampContext) {}

// EnterPrecisionTimestampTZ is called when production precisionTimestampTZ is entered.
func (s *BaseSubstraitTypeListener) EnterPrecisionTimestampTZ(ctx *PrecisionTimestampTZContext) {}

// ExitPrecisionTimestampTZ is called when production precisionTimestampTZ is exited.
func (s *BaseSubstraitTypeListener) ExitPrecisionTimestampTZ(ctx *PrecisionTimestampTZContext) {}

// EnterStruct is called when production struct is entered.
func (s *BaseSubstraitTypeListener) EnterStruct(ctx *StructContext) {}

// ExitStruct is called when production struct is exited.
func (s *BaseSubstraitTypeListener) ExitStruct(ctx *StructContext) {}

// EnterNStruct is called when production nStruct is entered.
func (s *BaseSubstraitTypeListener) EnterNStruct(ctx *NStructContext) {}

// ExitNStruct is called when production nStruct is exited.
func (s *BaseSubstraitTypeListener) ExitNStruct(ctx *NStructContext) {}

// EnterList is called when production list is entered.
func (s *BaseSubstraitTypeListener) EnterList(ctx *ListContext) {}

// ExitList is called when production list is exited.
func (s *BaseSubstraitTypeListener) ExitList(ctx *ListContext) {}

// EnterMap is called when production map is entered.
func (s *BaseSubstraitTypeListener) EnterMap(ctx *MapContext) {}

// ExitMap is called when production map is exited.
func (s *BaseSubstraitTypeListener) ExitMap(ctx *MapContext) {}

// EnterFunc is called when production func is entered.
func (s *BaseSubstraitTypeListener) EnterFunc(ctx *FuncContext) {}

// ExitFunc is called when production func is exited.
func (s *BaseSubstraitTypeListener) ExitFunc(ctx *FuncContext) {}

// EnterUserDefined is called when production userDefined is entered.
func (s *BaseSubstraitTypeListener) EnterUserDefined(ctx *UserDefinedContext) {}

// ExitUserDefined is called when production userDefined is exited.
func (s *BaseSubstraitTypeListener) ExitUserDefined(ctx *UserDefinedContext) {}

// EnterSingleFuncParam is called when production singleFuncParam is entered.
func (s *BaseSubstraitTypeListener) EnterSingleFuncParam(ctx *SingleFuncParamContext) {}

// ExitSingleFuncParam is called when production singleFuncParam is exited.
func (s *BaseSubstraitTypeListener) ExitSingleFuncParam(ctx *SingleFuncParamContext) {}

// EnterFuncParamsWithParens is called when production funcParamsWithParens is entered.
func (s *BaseSubstraitTypeListener) EnterFuncParamsWithParens(ctx *FuncParamsWithParensContext) {}

// ExitFuncParamsWithParens is called when production funcParamsWithParens is exited.
func (s *BaseSubstraitTypeListener) ExitFuncParamsWithParens(ctx *FuncParamsWithParensContext) {}

// EnterNumericLiteral is called when production numericLiteral is entered.
func (s *BaseSubstraitTypeListener) EnterNumericLiteral(ctx *NumericLiteralContext) {}

// ExitNumericLiteral is called when production numericLiteral is exited.
func (s *BaseSubstraitTypeListener) ExitNumericLiteral(ctx *NumericLiteralContext) {}

// EnterNumericParameterName is called when production numericParameterName is entered.
func (s *BaseSubstraitTypeListener) EnterNumericParameterName(ctx *NumericParameterNameContext) {}

// ExitNumericParameterName is called when production numericParameterName is exited.
func (s *BaseSubstraitTypeListener) ExitNumericParameterName(ctx *NumericParameterNameContext) {}

// EnterNumericExpression is called when production numericExpression is entered.
func (s *BaseSubstraitTypeListener) EnterNumericExpression(ctx *NumericExpressionContext) {}

// ExitNumericExpression is called when production numericExpression is exited.
func (s *BaseSubstraitTypeListener) ExitNumericExpression(ctx *NumericExpressionContext) {}

// EnterAnyType is called when production anyType is entered.
func (s *BaseSubstraitTypeListener) EnterAnyType(ctx *AnyTypeContext) {}

// ExitAnyType is called when production anyType is exited.
func (s *BaseSubstraitTypeListener) ExitAnyType(ctx *AnyTypeContext) {}

// EnterTypeDef is called when production typeDef is entered.
func (s *BaseSubstraitTypeListener) EnterTypeDef(ctx *TypeDefContext) {}

// ExitTypeDef is called when production typeDef is exited.
func (s *BaseSubstraitTypeListener) ExitTypeDef(ctx *TypeDefContext) {}

// EnterIfExpr is called when production IfExpr is entered.
func (s *BaseSubstraitTypeListener) EnterIfExpr(ctx *IfExprContext) {}

// ExitIfExpr is called when production IfExpr is exited.
func (s *BaseSubstraitTypeListener) ExitIfExpr(ctx *IfExprContext) {}

// EnterTypeLiteral is called when production TypeLiteral is entered.
func (s *BaseSubstraitTypeListener) EnterTypeLiteral(ctx *TypeLiteralContext) {}

// ExitTypeLiteral is called when production TypeLiteral is exited.
func (s *BaseSubstraitTypeListener) ExitTypeLiteral(ctx *TypeLiteralContext) {}

// EnterMultilineDefinition is called when production MultilineDefinition is entered.
func (s *BaseSubstraitTypeListener) EnterMultilineDefinition(ctx *MultilineDefinitionContext) {}

// ExitMultilineDefinition is called when production MultilineDefinition is exited.
func (s *BaseSubstraitTypeListener) ExitMultilineDefinition(ctx *MultilineDefinitionContext) {}

// EnterTernary is called when production Ternary is entered.
func (s *BaseSubstraitTypeListener) EnterTernary(ctx *TernaryContext) {}

// ExitTernary is called when production Ternary is exited.
func (s *BaseSubstraitTypeListener) ExitTernary(ctx *TernaryContext) {}

// EnterBinaryExpr is called when production BinaryExpr is entered.
func (s *BaseSubstraitTypeListener) EnterBinaryExpr(ctx *BinaryExprContext) {}

// ExitBinaryExpr is called when production BinaryExpr is exited.
func (s *BaseSubstraitTypeListener) ExitBinaryExpr(ctx *BinaryExprContext) {}

// EnterParenExpression is called when production ParenExpression is entered.
func (s *BaseSubstraitTypeListener) EnterParenExpression(ctx *ParenExpressionContext) {}

// ExitParenExpression is called when production ParenExpression is exited.
func (s *BaseSubstraitTypeListener) ExitParenExpression(ctx *ParenExpressionContext) {}

// EnterParameterName is called when production ParameterName is entered.
func (s *BaseSubstraitTypeListener) EnterParameterName(ctx *ParameterNameContext) {}

// ExitParameterName is called when production ParameterName is exited.
func (s *BaseSubstraitTypeListener) ExitParameterName(ctx *ParameterNameContext) {}

// EnterFunctionCall is called when production FunctionCall is entered.
func (s *BaseSubstraitTypeListener) EnterFunctionCall(ctx *FunctionCallContext) {}

// ExitFunctionCall is called when production FunctionCall is exited.
func (s *BaseSubstraitTypeListener) ExitFunctionCall(ctx *FunctionCallContext) {}

// EnterNotExpr is called when production NotExpr is entered.
func (s *BaseSubstraitTypeListener) EnterNotExpr(ctx *NotExprContext) {}

// ExitNotExpr is called when production NotExpr is exited.
func (s *BaseSubstraitTypeListener) ExitNotExpr(ctx *NotExprContext) {}

// EnterLiteralNumber is called when production LiteralNumber is entered.
func (s *BaseSubstraitTypeListener) EnterLiteralNumber(ctx *LiteralNumberContext) {}

// ExitLiteralNumber is called when production LiteralNumber is exited.
func (s *BaseSubstraitTypeListener) ExitLiteralNumber(ctx *LiteralNumberContext) {}
