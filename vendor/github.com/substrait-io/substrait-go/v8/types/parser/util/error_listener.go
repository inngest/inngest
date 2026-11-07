package util

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
)

type VisitErrorListener interface {
	ReportVisitError(ctx antlr.ParserRuleContext, err error)
	ReportPanicError(err error)
	ErrorCount() int
	GetErrors() []string
}

type SimpleErrorListener struct {
	errorCount int
	errors     []string
}

func (l *SimpleErrorListener) ReportVisitError(ctx antlr.ParserRuleContext, err error) {
	l.errorCount++
	l.errors = append(l.errors, fmt.Sprintf("Visit error at line %d: %s", ctx.GetStart().GetLine(), err))
}

func (l *SimpleErrorListener) ReportPanicError(err error) {
	l.errorCount++
	l.errors = append(l.errors, fmt.Sprintf("Tree Visit panic error %s", err))
}

func (l *SimpleErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	l.errorCount++
	l.errors = append(l.errors, fmt.Sprintf("Syntax error at line %d:%d: %s ", line, column, msg))
}

func (l *SimpleErrorListener) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
}

func (l *SimpleErrorListener) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs *antlr.ATNConfigSet) {
}

func (l *SimpleErrorListener) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs *antlr.ATNConfigSet) {
}

func (l *SimpleErrorListener) ErrorCount() int {
	return l.errorCount
}

func (l *SimpleErrorListener) GetErrors() []string {
	return l.errors
}

func NewSimpleErrorListener() *SimpleErrorListener {
	return new(SimpleErrorListener)
}

func TransformPanicToError(err *error, input, ctxStr string, errorListener VisitErrorListener) {
	if r := recover(); r != nil {
		switch t := r.(type) {
		case string:
			*err = fmt.Errorf("failed %s %s with error: %s", ctxStr, input, t)
		case error:
			*err = t
		default:
			*err = fmt.Errorf("failed %s %s with unknown panic", ctxStr, input)
		}
		if errorListener != nil {
			errorListener.ReportPanicError(*err)
		}
	}
}
