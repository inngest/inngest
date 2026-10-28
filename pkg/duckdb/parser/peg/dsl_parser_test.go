package peg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGrammarSelfContained(t *testing.T) {
	g, err := ParseGrammar(`
# a comment
Greeting <- Hello / Goodbye
Hello <- 'hello' Name
Goodbye <- 'bye'? Name
Name <- [A-Za-z]+
List(D) <- D (',' D)* ','?
Parens(D) <- '(' D ')'
Sum <- Parens(List(Name))
NotAKeyword <- !Hello Name
Token <- < Name Name >
`)
	require.NoError(t, err)
	require.Len(t, g.Rules, 9)

	greeting := g.Rules["Greeting"]
	require.Equal(t, ExprChoice, greeting.Expr.Kind)
	require.Len(t, greeting.Expr.Children, 2)
	require.Equal(t, ExprRef, greeting.Expr.Children[0].Kind)
	require.Equal(t, "Hello", greeting.Expr.Children[0].Ref)

	hello := g.Rules["Hello"]
	require.Equal(t, ExprSeq, hello.Expr.Kind)
	require.Equal(t, ExprLiteral, hello.Expr.Children[0].Kind)
	require.Equal(t, "hello", hello.Expr.Children[0].Literal)
	require.Equal(t, ExprRef, hello.Expr.Children[1].Kind)

	goodbye := g.Rules["Goodbye"]
	require.Equal(t, ExprOptional, goodbye.Expr.Children[0].Kind)

	name := g.Rules["Name"]
	require.Equal(t, ExprPlus, name.Expr.Kind)
	require.Equal(t, ExprClass, name.Expr.Children[0].Kind)
	require.False(t, name.Expr.Children[0].Class.Negated)
	require.Equal(t, []ClassRange{{Lo: 'A', Hi: 'Z'}, {Lo: 'a', Hi: 'z'}}, name.Expr.Children[0].Class.Ranges)

	list := g.Rules["List"]
	require.Equal(t, "D", list.Param)

	sum := g.Rules["Sum"]
	require.Equal(t, ExprCall, sum.Expr.Kind)
	require.Equal(t, "Parens", sum.Expr.Ref)
	require.Len(t, sum.Expr.Children, 1)
	arg := sum.Expr.Children[0]
	require.Equal(t, ExprCall, arg.Kind)
	require.Equal(t, "List", arg.Ref)

	notAKeyword := g.Rules["NotAKeyword"]
	require.Equal(t, ExprSeq, notAKeyword.Expr.Kind)
	require.Equal(t, ExprNot, notAKeyword.Expr.Children[0].Kind)

	token := g.Rules["Token"]
	require.Equal(t, ExprToken, token.Expr.Kind)
}

func TestParseGrammarRejectsDuplicateRule(t *testing.T) {
	_, err := ParseGrammar("A <- 'x'\nA <- 'y'\n")
	require.Error(t, err)
}

func TestClassMatches(t *testing.T) {
	digits := Class{Ranges: []ClassRange{{Lo: '0', Hi: '9'}}}
	require.True(t, digits.Matches('5'))
	require.False(t, digits.Matches('a'))

	notQuote := Class{Negated: true, Ranges: []ClassRange{{Lo: '\'', Hi: '\''}}}
	require.True(t, notQuote.Matches('x'))
	require.False(t, notQuote.Matches('\''))
}
