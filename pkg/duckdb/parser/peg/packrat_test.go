package peg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const testGrammar = `
Greeting <- Hello / Goodbye
Hello <- 'hello' Name
Goodbye <- 'bye'? Name
Name <- Word
List(D) <- D (',' D)* ','?
Parens(D) <- '(' D ')'
Sum <- Parens(List(Name))
`

func wordPrimitive(newNode func(Node) *Node, input string, pos int) (*Node, int, bool) {
	start := pos
	for pos < len(input) && ((input[pos] >= 'a' && input[pos] <= 'z') || (input[pos] >= 'A' && input[pos] <= 'Z')) {
		pos++
	}
	if pos == start {
		return nil, 0, false
	}
	return newNode(Node{Kind: KindPrimitive, Text: input[start:pos], Start: start, End: pos}), pos, true
}

func skipASCIIWhitespace(input string, pos int) int {
	for pos < len(input) && (input[pos] == ' ' || input[pos] == '\t' || input[pos] == '\n' || input[pos] == '\r') {
		pos++
	}
	return pos
}

func newTestParser(t *testing.T) *Parser {
	g, err := ParseGrammar(testGrammar)
	require.NoError(t, err)
	return &Parser{Grammar: g, Primitives: map[string]Primitive{"Word": wordPrimitive}, SkipTrivia: skipASCIIWhitespace}
}

func TestParseChoiceAndLiteral(t *testing.T) {
	p := newTestParser(t)
	n, err := p.Parse("hello world", "Greeting")
	require.NoError(t, err)
	require.Equal(t, KindRule, n.Kind)
	require.Equal(t, "Greeting", n.Name)
	choice := n.Children[0]
	require.Equal(t, KindChoice, choice.Kind)
	require.Equal(t, 0, choice.Alt) // matched Hello, not Goodbye
}

func TestParseOptional(t *testing.T) {
	p := newTestParser(t)
	n, err := p.Parse("world", "Goodbye")
	require.NoError(t, err) // 'bye'? absent, just Name
	seq := n.Children[0]
	require.Equal(t, KindSeq, seq.Kind)
	opt := seq.Children[0]
	require.Equal(t, KindOptional, opt.Kind)
	require.Empty(t, opt.Children)
}

func TestParseParameterizedRules(t *testing.T) {
	p := newTestParser(t)
	n, err := p.Parse("(alice, bob, carol)", "Sum")
	require.NoError(t, err)
	require.Equal(t, "Sum", n.Name)
	// Sum's body is the ExprCall Parens(List(Name)) directly (a single-
	// element rule body isn't Seq-wrapped — Task 2's dsl_parser.go
	// collapses a length-1 sequence to its sole element), so n.Children[0]
	// is the "Parens" KindRule wrapper node itself; unwrap once more to
	// reach its actual '(' List(Name) ')' Seq.
	parensNode := n.Children[0]
	require.Equal(t, KindRule, parensNode.Kind)
	require.Equal(t, "Parens", parensNode.Name)
	parensSeq := parensNode.Children[0]
	require.Equal(t, KindSeq, parensSeq.Kind)
	require.Len(t, parensSeq.Children, 3)
	require.Equal(t, KindLiteral, parensSeq.Children[0].Kind)
	listNode := parensSeq.Children[1]
	require.Equal(t, "List", listNode.Name)
}

func TestParseRejectsTrailingGarbage(t *testing.T) {
	p := newTestParser(t)
	_, err := p.Parse("hello world!!!", "Greeting")
	require.Error(t, err)
	var perr *ParseError
	require.ErrorAs(t, err, &perr)
}

func TestParseMemoizesRepeatedRuleAtSamePosition(t *testing.T) {
	// Several alternatives all try Shared at the same starting position
	// before falling through to Word — not exponential (this engine
	// doesn't support true left recursion; neither does DuckDB's own
	// grammar need it, see Task 3's evaluator doc comment), but it does
	// exercise the same (rule, pos) memo key being looked up repeatedly
	// and confirms the cached result is reused consistently rather than
	// corrupting state across repeated evaluations.
	g, err := ParseGrammar(`
A <- (Shared 'x') / (Shared 'y') / (Shared 'z') / (Shared 'q') / Word
Shared <- Word
`)
	require.NoError(t, err)
	p := &Parser{Grammar: g, Primitives: map[string]Primitive{"Word": wordPrimitive}, SkipTrivia: skipASCIIWhitespace}
	n, err := p.Parse("hello", "A")
	require.NoError(t, err)
	require.NotNil(t, n)
}
