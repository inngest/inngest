// pkg/duckdb/parser/adapter_expr.go
package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

// adapter walks a peg.Node CST (from parsing against the vendored DuckDB
// grammar) into the typed AST. src is the original SQL text — kept around
// for Position computation and for raw-text extraction (Type is stored as
// unparsed source text; see CastExpr's doc note in this task's plan entry).
type adapter struct {
	src string
	// newlines holds the byte offset of every '\n' in src, ascending. a.pos
	// is called twice per adapted node (start and end), so on anything but
	// a trivial query this table (built once) plus a binary search per call
	// is the difference between O(nodes) and O(nodes*offset) — positionAt's
	// plain per-call scan from byte 0 doesn't get to reuse anything across
	// calls, and it's the far more common case here.
	newlines []int
}

func newAdapter(src string) *adapter {
	a := &adapter{src: src}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			a.newlines = append(a.newlines, i)
		}
	}
	return a
}

// positionAt scans src from the start on every call — fine for the rare,
// one-off error-position lookup it backs (see parser.go's ParseString), but
// not for adapter.pos's per-node hot path; see newAdapter's doc comment.
func positionAt(src string, offset int) Position {
	line, col := 1, 1
	for i := 0; i < offset && i < len(src); i++ {
		if src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return Position{Offset: offset, Line: line, Column: col}
}

func (a *adapter) pos(offset int) Position {
	// Match positionAt's loop, whose `i < offset` bound treats any
	// offset <= 0 as "no characters scanned yet" (line 1, column 1).
	clamped := offset
	if clamped > len(a.src) {
		clamped = len(a.src)
	}
	if clamped < 0 {
		clamped = 0
	}
	// idx = count of newlines strictly before clamped, since sort.Search
	// finds the first newline at or past it.
	idx := sort.Search(len(a.newlines), func(i int) bool { return a.newlines[i] >= clamped })
	lastNewline := -1
	if idx > 0 {
		lastNewline = a.newlines[idx-1]
	}
	return Position{Offset: offset, Line: idx + 1, Column: clamped - lastNewline}
}

func (a *adapter) at(n *peg.Node) baseExpr {
	return baseExpr{start: a.pos(n.Start), end: a.pos(n.End)}
}

func (a *adapter) rawText(n *peg.Node) string { return strings.TrimSpace(a.src[n.Start:n.End]) }

// body unwraps a KindRule wrapper to reveal the Expr its body produced.
func body(n *peg.Node) *peg.Node {
	if n.Kind == peg.KindRule && len(n.Children) == 1 {
		return n.Children[0]
	}
	return n
}

// choice unwraps a KindChoice wrapper to whichever alternative matched.
// (A grammar rule with only one alternative collapses at DSL-parse time —
// Task 2's parseChoice — so some "choices" arrive already unwrapped; choice
// is a no-op in that case.)
func choice(n *peg.Node) *peg.Node {
	if n.Kind == peg.KindChoice && len(n.Children) == 1 {
		return n.Children[0]
	}
	return n
}

// present reports whether a KindOptional node matched, returning its inner
// (not-yet-body-unwrapped) child. Panics if n isn't actually KindOptional —
// deliberately loud rather than silently treating a mismatched Kind the
// same as "absent": that mismatch means the vendored grammar's cardinality
// at this position changed (e.g. a re-vendor turned '?' into '*'/'+' — see
// Task 15) and this call site wasn't updated, which would otherwise show
// up as silently-wrong output (an identifier that's just missing from the
// AST) instead of a loud, findable failure.
func present(n *peg.Node) (*peg.Node, bool) {
	if n.Kind != peg.KindOptional {
		panic(fmt.Sprintf("duckdb/parser: present() called on a %v node, not KindOptional — the vendored grammar's cardinality at this position likely changed; see Task 15's plan entry", n.Kind))
	}
	if len(n.Children) == 0 {
		return nil, false
	}
	return n.Children[0], true
}

// repeatChildren returns a KindRepeat node's matched elements (each still
// wrapped in whatever the repeated sub-expression itself produces — call
// body() on each if it's a plain rule reference). Panics on a Kind
// mismatch for the same reason present() does.
func repeatChildren(n *peg.Node) []*peg.Node {
	if n.Kind != peg.KindRepeat {
		panic(fmt.Sprintf("duckdb/parser: repeatChildren() called on a %v node, not KindRepeat — the vendored grammar's cardinality at this position likely changed; see Task 15's plan entry", n.Kind))
	}
	return n.Children
}

// literalText descends through KindRule/KindOptional/KindChoice wrapper
// layers to a leaf KindLiteral/KindPrimitive node and returns its matched
// text — a primitive's decoded Value if it has one (so a quoted identifier
// or string literal comes back unescaped), uppercased if the raw text is
// keyword-shaped (so 'and'/'AND'/'And' all normalize the same way), and
// verbatim for symbol operators. Does not handle a multi-element Seq (no
// caller needs that — the one grammar rule shaped that way, IsDistinctFromOp,
// is adapted with its own explicit field access instead).
func literalText(n *peg.Node) string {
	for {
		switch n.Kind {
		case peg.KindRule, peg.KindOptional, peg.KindChoice:
			if len(n.Children) == 0 {
				return ""
			}
			n = n.Children[0]
		default:
			if n.HasValue {
				return n.Value
			}
			if isWordLiteral(n.Text) {
				return strings.ToUpper(n.Text)
			}
			return n.Text
		}
	}
}

func isWordLiteral(s string) bool {
	return len(s) > 0 && identStart(s[0])
}

// foldLeftAssoc walks a "Head TailRule*" shape: head is already adapted,
// tails is the KindRepeat node of Tail matches. combine receives the
// accumulated left side and one raw Tail node and returns the new
// accumulated Expr — used directly by levels whose Tail shape needs custom
// handling (IsDistinctFrom, Comparison), and via foldBinaryLevel for the
// levels whose Tail is uniformly "Op RHS".
func foldLeftAssoc(head Expr, tails *peg.Node, combine func(left Expr, tail *peg.Node) Expr) Expr {
	result := head
	for _, t := range tails.Children {
		result = combine(result, t)
	}
	return result
}

// foldBinaryLevel handles the 9 precedence levels whose grammar shape is
// exactly "Operand TailRule*" with Tail exactly "OpRule Operand" — LogicalOr,
// LogicalAnd, OtherOperator, Bitwise, Additive, Multiplicative,
// Exponentiation, Collate, AtTimeZone. adaptOperand adapts one operand at
// this level (i.e. the next-higher-precedence level's adapter).
func (a *adapter) foldBinaryLevel(n *peg.Node, adaptOperand func(*peg.Node) Expr) Expr {
	seq := body(n)
	head := adaptOperand(seq.Children[0])
	return foldLeftAssoc(head, seq.Children[1], func(left Expr, tail *peg.Node) Expr {
		tseq := body(tail)
		op := literalText(tseq.Children[0])
		right := adaptOperand(tseq.Children[1])
		return &BinaryExpr{baseExpr: a.at(tail), Op: op, Left: left, Right: right}
	})
}

// --- the 16-level precedence chain, root to leaf ---

func (a *adapter) adaptExpression(n *peg.Node) Expr {
	return a.adaptLambdaArrowExpression(body(n))
}

func (a *adapter) adaptLambdaArrowExpression(n *peg.Node) Expr {
	seq := body(n)
	head := a.adaptLogicalOrExpression(seq.Children[0])
	return foldLeftAssoc(head, seq.Children[1], func(left Expr, tail *peg.Node) Expr {
		tseq := body(tail)
		right := a.adaptLogicalOrExpression(tseq.Children[1])
		if id, ok := left.(*Ident); ok && len(id.Parts) == 1 {
			return &LambdaExpr{baseExpr: a.at(tail), Params: []string{id.Parts[0]}, Body: right}
		}
		return &BinaryExpr{baseExpr: a.at(tail), Op: "->", Left: left, Right: right}
	})
}

func (a *adapter) adaptLogicalOrExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptLogicalAndExpression)
}

func (a *adapter) adaptLogicalAndExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptLogicalNotExpression)
}

func (a *adapter) adaptLogicalNotExpression(n *peg.Node) Expr {
	// LogicalNotExpression <- NotExpression? IsExpression ; NotExpression <- NotKeyword+
	seq := body(n)
	result := a.adaptIsExpression(seq.Children[1])
	if notOpt, ok := present(seq.Children[0]); ok {
		count := len(body(notOpt).Children)
		for i := 0; i < count; i++ {
			result = &UnaryExpr{baseExpr: a.at(seq.Children[0]), Op: "NOT", X: result}
		}
	}
	return result
}

func (a *adapter) adaptIsExpression(n *peg.Node) Expr {
	// IsExpression <- IsDistinctFromExpression IsTest*
	seq := body(n)
	result := a.adaptIsDistinctFromExpression(seq.Children[0])
	for _, testNode := range seq.Children[1].Children {
		// IsTest <- IsLiteral / NotNull / IsNull
		alt := choice(body(testNode))
		switch alt.Name {
		case "IsLiteral":
			// IsLiteral <- 'IS' 'NOT'? IsLiteralValue
			lseq := body(alt)
			_, not := present(lseq.Children[1])
			value := literalText(lseq.Children[2])
			result = &IsExpr{baseExpr: a.at(testNode), X: result, Not: not, Value: value}
		case "NotNull":
			result = &NullTest{baseExpr: a.at(testNode), X: result, Not: true}
		default: // IsNull
			result = &NullTest{baseExpr: a.at(testNode), X: result, Not: false}
		}
	}
	return result
}

func (a *adapter) adaptIsDistinctFromExpression(n *peg.Node) Expr {
	// IsDistinctFromExpression <- ComparisonExpression IsDistinctFromTail*
	seq := body(n)
	head := a.adaptComparisonExpression(seq.Children[0])
	return foldLeftAssoc(head, seq.Children[1], func(left Expr, tail *peg.Node) Expr {
		// IsDistinctFromTail <- IsDistinctFromOp ComparisonExpression
		// IsDistinctFromOp <- 'IS' 'NOT'? 'DISTINCT' 'FROM'
		tseq := body(tail)
		opSeq := body(tseq.Children[0])
		_, not := present(opSeq.Children[1])
		right := a.adaptComparisonExpression(tseq.Children[1])
		return &DistinctFromExpr{baseExpr: a.at(tail), Left: left, Right: right, Not: not}
	})
}

func (a *adapter) adaptComparisonExpression(n *peg.Node) Expr {
	// ComparisonExpression <- BetweenInLikeExpression ComparisonExpressionTail*
	seq := body(n)
	head := a.adaptBetweenInLikeExpression(seq.Children[0])
	return foldLeftAssoc(head, seq.Children[1], func(left Expr, tail *peg.Node) Expr {
		// ComparisonExpressionTail <- ComparisonOperator NotExpression? BetweenInLikeExpression
		// The optional NotExpression here is unused/vestigial upstream syntax — not modeled (see this task's scope note).
		tseq := body(tail)
		op := literalText(tseq.Children[0])
		right := a.adaptBetweenInLikeExpression(tseq.Children[len(tseq.Children)-1])
		return &BinaryExpr{baseExpr: a.at(tail), Op: op, Left: left, Right: right}
	})
}

func (a *adapter) adaptBetweenInLikeExpression(n *peg.Node) Expr {
	// BetweenInLikeExpression <- OtherOperatorExpression BetweenInLikeOp?
	seq := body(n)
	x := a.adaptOtherOperatorExpression(seq.Children[0])
	opNode, ok := present(seq.Children[1])
	if !ok {
		return x
	}
	// BetweenInLikeOp <- 'NOT'? BetweenInLikeOpExpression
	opSeq := body(opNode)
	_, not := present(opSeq.Children[0])
	inner := choice(body(opSeq.Children[1])) // BetweenInLikeOpExpression <- BetweenClause / InClause / LikeClause
	switch inner.Name {
	case "BetweenClause":
		// 'BETWEEN' OtherOperatorExpression 'AND' OtherOperatorExpression
		bseq := body(inner)
		low := a.adaptOtherOperatorExpression(bseq.Children[1])
		high := a.adaptOtherOperatorExpression(bseq.Children[3])
		return &BetweenExpr{baseExpr: a.at(n), X: x, Low: low, High: high, Not: not}
	case "InClause":
		// 'IN' InExpression ; InExpression <- InExpressionList / InSelectStatement / InContainsExpression
		iseq := body(inner)
		in := choice(body(iseq.Children[1]))
		switch in.Name {
		case "InExpressionList":
			return &InExpr{baseExpr: a.at(n), X: x, Not: not, List: a.adaptExprList(body(in))}
		case "InSelectStatement":
			return &InExpr{baseExpr: a.at(n), X: x, Not: not, Subquery: a.adaptParenSelect(body(in))}
		default: // InContainsExpression <- OtherOperatorExpression
			rhs := a.adaptOtherOperatorExpression(body(in))
			return &InExpr{baseExpr: a.at(n), X: x, Not: not, List: []Expr{rhs}}
		}
	default: // LikeClause <- LikeVariations OtherOperatorExpression EscapeClause?
		lseq := body(inner)
		op := a.adaptLikeVariations(lseq.Children[0])
		pattern := a.adaptOtherOperatorExpression(lseq.Children[1])
		var escape Expr
		if escNode, ok := present(lseq.Children[2]); ok {
			// EscapeClause <- 'ESCAPE' ComparisonExpression
			escape = a.adaptComparisonExpression(body(escNode).Children[1])
		}
		return &LikeExpr{baseExpr: a.at(n), X: x, Pattern: pattern, Not: not, Op: op, Escape: escape}
	}
}

func (a *adapter) adaptLikeVariations(n *peg.Node) string {
	alt := choice(body(n))
	if alt.Name == "SimilarToToken" {
		return "SIMILAR TO"
	}
	return literalText(alt)
}

func (a *adapter) adaptOtherOperatorExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptBitwiseExpression)
}
func (a *adapter) adaptBitwiseExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptAdditiveExpression)
}
func (a *adapter) adaptAdditiveExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptMultiplicativeExpression)
}
func (a *adapter) adaptMultiplicativeExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptExponentiationExpression)
}
func (a *adapter) adaptExponentiationExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptCollateExpression)
}
func (a *adapter) adaptCollateExpression(n *peg.Node) Expr {
	return a.foldBinaryLevel(n, a.adaptAtTimeZoneExpression)
}
func (a *adapter) adaptAtTimeZoneExpression(n *peg.Node) Expr {
	// AtTimeZoneExpressionTail <- AtTimeZoneOperator PrefixExpression ;
	// AtTimeZoneOperator <- 'AT' 'TIME' 'ZONE' — a 3-token Seq, not the
	// single leaf literalText() (used by foldBinaryLevel) is documented to
	// handle, so it silently comes back "" here. AtTimeZoneOperator only
	// ever matches that one fixed phrase, so hardcode the normalized text
	// instead — the same fix adaptLikeVariations applies for SIMILAR TO.
	seq := body(n)
	head := a.adaptPrefixExpression(seq.Children[0])
	return foldLeftAssoc(head, seq.Children[1], func(left Expr, tail *peg.Node) Expr {
		right := a.adaptPrefixExpression(body(tail).Children[1])
		return &BinaryExpr{baseExpr: a.at(tail), Op: "AT TIME ZONE", Left: left, Right: right}
	})
}

func (a *adapter) adaptPrefixExpression(n *peg.Node) Expr {
	// PrefixExpression <- PrefixOperator* BaseExpression
	seq := body(n)
	prefixes := body(seq.Children[0]).Children
	result := a.adaptBaseExpression(seq.Children[1])
	for i := len(prefixes) - 1; i >= 0; i-- {
		result = &UnaryExpr{baseExpr: a.at(prefixes[i]), Op: literalText(prefixes[i]), X: result}
	}
	return result
}

func (a *adapter) adaptBaseExpression(n *peg.Node) Expr {
	// BaseExpression <- SingleExpression IndirectionList?
	seq := body(n)
	result := a.adaptSingleExpression(seq.Children[0])
	if list, ok := present(seq.Children[1]); ok {
		for _, ind := range body(list).Children {
			result = a.adaptIndirection(result, ind)
		}
	}
	return result
}

func (a *adapter) adaptIndirection(x Expr, n *peg.Node) Expr {
	// Indirection <- CastOperator / DotOperator / SliceExpression / PostfixOperator
	alt := choice(body(n))
	switch alt.Name {
	case "CastOperator":
		// '::' Type
		seq := body(alt)
		return &CastExpr{baseExpr: a.at(n), X: x, Type: a.rawText(seq.Children[1])}
	case "DotOperator":
		inner := choice(body(alt)) // DotMethodOperator / DotColumnOperator
		if inner.Name == "DotMethodOperator" {
			// '.' MethodExpression ; MethodExpression <- ColLabel MethodExpressionArguments
			mseq := body(body(inner).Children[1])
			name := literalText(mseq.Children[0])
			args, orderBy, distinct, all := a.adaptMethodArgs(mseq.Children[1])
			return &FunctionExpr{baseExpr: a.at(n), Name: []string{name}, Receiver: x, Args: args, OrderBy: orderBy, Distinct: distinct, All: all}
		}
		// DotColumnOperator <- '.' ColLabel
		return &DotExpr{baseExpr: a.at(n), X: x, Field: literalText(body(inner).Children[1])}
	case "SliceExpression":
		return a.adaptSliceExpression(x, alt)
	default: // PostfixOperator <- '!'
		return &UnaryExpr{baseExpr: a.at(n), Op: "!", X: x, Postfix: true}
	}
}

func (a *adapter) adaptSliceExpression(x Expr, n *peg.Node) *SliceExpr {
	// SliceExpression <- '[' SliceBound ']' ; SliceBound <- Expression? EndSliceBound? StepSliceBound?
	seq := body(n)
	bseq := body(seq.Children[1])
	s := &SliceExpr{baseExpr: a.at(n), X: x}
	if e, ok := present(bseq.Children[0]); ok {
		s.Start = a.adaptExpression(e)
	}
	if e, ok := present(bseq.Children[1]); ok {
		// EndSliceBound <- ':' EndSliceValue? ; EndSliceValue <- Expression / EndSliceMinus
		if v, ok := present(body(e).Children[1]); ok {
			inner := choice(body(v))
			if inner.Name != "EndSliceMinus" {
				s.Stop = a.adaptExpression(inner)
			}
		}
	}
	if e, ok := present(bseq.Children[2]); ok {
		// StepSliceBound <- ':' Expression?
		s.HasStep = true
		if v, ok := present(body(e).Children[1]); ok {
			s.Step = a.adaptExpression(v)
		}
	}
	return s
}

// --- SingleExpression: every leaf alternative except SpecialFunctionExpression ---

func (a *adapter) adaptSingleExpression(n *peg.Node) Expr {
	alt := choice(body(n))
	switch alt.Name {
	case "ParensExpression":
		// ParensExpression <- Parens(Expression) — grouping parens are
		// transparent, no wrapper node. alt's own body is a single
		// ExprCall (Parens(...)), so body(alt) only unwraps to the
		// "Parens" KindRule node itself; a second body() reaches its Seq.
		return a.adaptExpression(body(body(alt)).Children[1])
	case "LiteralExpression":
		return a.adaptLiteralExpression(alt)
	case "Parameter":
		return a.adaptParameter(alt)
	case "SubqueryExpression":
		return a.adaptSubqueryExpression(alt)
	case "SpecialFunctionExpression":
		panic("duckdb/parser: COALESCE/EXTRACT/POSITION/SUBSTRING/TRIM/OVERLAY/NULLIF/ROW/UNPACK/TRY/COLUMNS/the LAMBDA keyword form are out of scope for now — see this task's scope note")
	case "ParenthesisExpression":
		return a.adaptParenthesisExpression(alt)
	case "IntervalLiteral":
		return a.adaptIntervalLiteral(alt)
	case "TypeLiteral":
		return a.adaptTypeLiteral(alt)
	case "CaseExpression":
		return a.adaptCaseExpression(alt)
	case "StarExpression":
		return a.adaptStarExpression(alt)
	case "CastExpression":
		return a.adaptCastExpressionCall(alt)
	case "GroupingExpression":
		return a.adaptGroupingExpression(alt)
	case "MapExpression":
		return a.adaptMapExpression(alt)
	case "FunctionExpression":
		return a.adaptFunctionExpression(alt)
	case "ColumnReference":
		return a.adaptColumnReference(alt)
	case "ListComprehensionExpression":
		return a.adaptListComprehensionExpression(alt)
	case "ListExpression":
		return a.adaptListExpression(alt)
	case "StructExpression":
		return a.adaptStructExpression(alt)
	case "PositionalExpression":
		// '#' NumberLiteral
		return &PositionalExpr{baseExpr: a.at(alt), Index: literalText(body(alt).Children[1])}
	default: // DefaultExpression <- 'DEFAULT'
		return &DefaultExpr{baseExpr: a.at(alt)}
	}
}

func (a *adapter) adaptLiteralExpression(n *peg.Node) Expr {
	// LiteralExpression <- StringLiteral / NumberLiteral / ConstantLiteral
	alt := choice(body(n))
	switch alt.Name {
	case "StringLiteral":
		return &Literal{baseExpr: a.at(alt), Kind: LitString, Text: alt.Value}
	case "NumberLiteral":
		return &Literal{baseExpr: a.at(alt), Kind: LitNumber, Text: alt.Text}
	default: // ConstantLiteral <- NullLiteral / TrueLiteral / FalseLiteral
		switch choice(body(alt)).Name {
		case "TrueLiteral":
			return &Literal{baseExpr: a.at(alt), Kind: LitTrue}
		case "FalseLiteral":
			return &Literal{baseExpr: a.at(alt), Kind: LitFalse}
		default:
			return &Literal{baseExpr: a.at(alt), Kind: LitNull}
		}
	}
}

func (a *adapter) adaptParameter(n *peg.Node) Expr {
	// Parameter <- QuestionMarkNumberedParameter / AnonymousParameter / NumberedParameter / ColLabelParameter
	alt := choice(body(n))
	switch alt.Name {
	case "QuestionMarkNumberedParameter", "NumberedParameter":
		// '?' NumberLiteral , or '$' NumberLiteral
		return &Parameter{baseExpr: a.at(alt), Kind: ParamNumbered, Value: literalText(body(alt).Children[1])}
	case "AnonymousParameter":
		return &Parameter{baseExpr: a.at(alt), Kind: ParamAnonymous}
	default: // ColLabelParameter <- '$' ColLabel
		return &Parameter{baseExpr: a.at(alt), Kind: ParamNamed, Value: literalText(body(alt).Children[1])}
	}
}

func (a *adapter) adaptSubqueryExpression(n *peg.Node) Expr {
	// SubqueryExpression <- SubqueryNot? SubqueryExists? SubqueryReference
	seq := body(n)
	_, not := present(seq.Children[0])
	_, exists := present(seq.Children[1])
	sub := a.adaptParenSelect(body(seq.Children[2])) // SubqueryReference <- Parens(SelectStatementInternal)
	return &SubqueryExpr{baseExpr: a.at(n), Select: sub, Exists: exists, Not: not}
}

func (a *adapter) adaptParenthesisExpression(n *peg.Node) Expr {
	// ParenthesisExpression <- Parens(List(Expression)?) — n's own body is
	// a single ExprCall (Parens(...)), so body(n) only unwraps to the
	// "Parens" KindRule node itself; a second body() reaches its Seq.
	seq := body(body(n))
	inner, ok := present(seq.Children[1])
	if !ok {
		return &ListExpr{baseExpr: a.at(n), Paren: true}
	}
	return &ListExpr{baseExpr: a.at(n), Paren: true, Elems: a.adaptExprList2(inner)}
}

func (a *adapter) adaptIntervalLiteral(n *peg.Node) Expr {
	// IntervalLiteral <- 'INTERVAL' IntervalParameter Interval?
	seq := body(n)
	paramAlt := choice(body(seq.Children[1])) // IntervalParameter <- IntervalStringParameter / NumberLiteral / ParensExpression
	var value Expr
	switch paramAlt.Name {
	case "IntervalStringParameter":
		value = &Literal{baseExpr: a.at(paramAlt), Kind: LitString, Text: body(paramAlt).Value}
	case "NumberLiteral":
		value = &Literal{baseExpr: a.at(paramAlt), Kind: LitNumber, Text: paramAlt.Text}
	default: // ParensExpression <- Parens(Expression)
		value = a.adaptExpression(body(body(paramAlt)).Children[1])
	}
	unit := ""
	if u, ok := present(seq.Children[2]); ok {
		unit = a.rawText(u)
	}
	return &IntervalExpr{baseExpr: a.at(n), Value: value, Unit: unit}
}

func (a *adapter) adaptTypeLiteral(n *peg.Node) Expr {
	// TypeLiteral <- Type StringLiteral
	seq := body(n)
	return &TypeLiteral{baseExpr: a.at(n), Type: a.rawText(seq.Children[0]), Value: seq.Children[1].Value}
}

func (a *adapter) adaptCaseExpression(n *peg.Node) Expr {
	// 'CASE' Expression? CaseWhenThen+ CaseElse? 'END'
	seq := body(n)
	c := &CaseExpr{baseExpr: a.at(n)}
	if op, ok := present(seq.Children[1]); ok {
		c.Operand = a.adaptExpression(op)
	}
	for _, wt := range body(seq.Children[2]).Children {
		wseq := body(wt) // 'WHEN' Expression 'THEN' Expression
		c.Whens = append(c.Whens, &CaseWhen{baseExpr: a.at(wt), Cond: a.adaptExpression(wseq.Children[1]), Result: a.adaptExpression(wseq.Children[3])})
	}
	if el, ok := present(seq.Children[3]); ok {
		c.Else = a.adaptExpression(body(el).Children[1]) // 'ELSE' Expression
	}
	return c
}

func (a *adapter) adaptStarExpression(n *peg.Node) Expr {
	// StarExpression <- StarQualifierList? '*' ExcludeList? ReplaceList? RenameList?
	seq := body(n)
	s := &StarExpr{baseExpr: a.at(n)}
	if q, ok := present(seq.Children[0]); ok {
		for _, cd := range body(q).Children { // StarQualifierList <- ColIdDot+ ; ColIdDot <- ColId '.'
			s.Qualifier = append(s.Qualifier, literalText(body(cd).Children[0]))
		}
	}
	if ex, ok := present(seq.Children[2]); ok {
		// ExcludeList <- ExcludeOrExcept ExcludeNames
		s.Exclude = a.adaptExcludeNames(body(ex).Children[1])
	}
	return s
}

func (a *adapter) adaptExcludeNames(n *peg.Node) []string {
	// ExcludeNames <- ExcludeNameList / ExcludeNameSingle
	alt := choice(body(n))
	if alt.Name == "ExcludeNameSingle" {
		return []string{a.rawText(body(alt))}
	}
	// ExcludeNameList <- Parens(List(ExcludeName)) — alt's own body is a
	// single ExprCall (Parens(...)), so body(alt) only unwraps to the
	// "Parens" KindRule node itself; a second body() reaches its Seq.
	listSeq := body(body(body(alt)).Children[1])
	names := []string{a.rawText(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		names = append(names, a.rawText(body(tail).Children[1]))
	}
	return names
}

func (a *adapter) adaptCastExpressionCall(n *peg.Node) Expr {
	// CastExpression <- CastOrTryCast Parens(CastArguments) ; CastArguments <- Expression 'AS' Type
	seq := body(n)
	tryCast := choice(body(seq.Children[0])).Name == "TryCastKeyword"
	argsSeq := body(body(seq.Children[1]).Children[1])
	return &CastExpr{baseExpr: a.at(n), X: a.adaptExpression(argsSeq.Children[0]), Type: a.rawText(argsSeq.Children[2]), TryCast: tryCast}
}

func (a *adapter) adaptGroupingExpression(n *peg.Node) Expr {
	// GroupingExpression <- GroupingOrGroupingId Parens(List(Expression)?)
	seq := body(n)
	args, _ := a.adaptOptExprList(seq.Children[1])
	return &GroupingExpr{baseExpr: a.at(n), Args: args}
}

func (a *adapter) adaptOptExprList(n *peg.Node) ([]Expr, bool) {
	parensSeq := body(n)
	inner, ok := present(parensSeq.Children[1])
	if !ok {
		return nil, false
	}
	return a.adaptExprList2(inner), true
}

func (a *adapter) adaptMapExpression(n *peg.Node) Expr {
	// MapExpression <- 'MAP' MapStructExpression ; MapStructExpression <- '{' List(MapStructField)? '}'
	seq := body(n)
	mseq := body(seq.Children[1])
	m := &MapExpr{baseExpr: a.at(n)}
	if fields, ok := present(mseq.Children[1]); ok {
		listSeq := body(fields)
		add := func(fn *peg.Node) {
			fseq := body(fn) // MapStructField <- Expression ':' Expression
			m.Keys = append(m.Keys, a.adaptExpression(fseq.Children[0]))
			m.Values = append(m.Values, a.adaptExpression(fseq.Children[2]))
		}
		add(listSeq.Children[0])
		for _, tail := range listSeq.Children[1].Children {
			add(body(tail).Children[1])
		}
	}
	return m
}

func (a *adapter) adaptFunctionExpression(n *peg.Node) Expr {
	// FunctionExpression <- FunctionIdentifier FunctionExpressionArguments WithinGroupClause? FilterClause? ExportClause? OverClause?
	seq := body(n)
	f := &FunctionExpr{baseExpr: a.at(n), Name: a.adaptFunctionIdentifier(seq.Children[0])}
	// FunctionExpressionArguments <- Parens(FunctionExpressionArgumentList)
	// seq.Children[1] (FunctionExpressionArguments) has a single-ExprCall
	// body, so it takes two body() calls to reach the Parens Seq, then
	// Children[1] for the list node, then a third body() for its own Seq.
	argsSeq := body(body(body(seq.Children[1])).Children[1])
	f.Args, f.OrderBy, f.Distinct, f.All, f.IgnoreNulls, f.RespectNulls = a.adaptCallArgList(argsSeq)
	if wg, ok := present(seq.Children[2]); ok {
		// WithinGroupClause <- 'WITHIN' 'GROUP' Parens(OrderByClause)
		f.Within = a.adaptOrderByClause(body(body(wg).Children[2]).Children[1])
	}
	if fc, ok := present(seq.Children[3]); ok {
		// FilterClause <- 'FILTER' FilterClauseExpression
		// FilterClauseExpression <- Parens(FilterClauseContents) ; FilterClauseContents <- 'WHERE'? Expression
		parensNode := body(body(fc).Children[1])
		contentsNode := body(parensNode).Children[1]
		f.Filter = a.adaptExpression(body(contentsNode).Children[1])
	}
	if _, ok := present(seq.Children[4]); ok {
		f.Export = true
	}
	if ov, ok := present(seq.Children[5]); ok {
		f.Over = a.adaptOverClause(ov)
	}
	return f
}

func (a *adapter) adaptFunctionIdentifier(n *peg.Node) []string {
	// FunctionIdentifier <- CatalogReservedSchemaFunctionName / SchemaReservedFunctionName / FunctionNameAsQualifiedName
	alt := choice(body(n))
	switch alt.Name {
	case "CatalogReservedSchemaFunctionName":
		// CatalogQualification ReservedSchemaQualification* ReservedFunctionName
		// (Task 15's second-commit re-vendor found ReservedSchemaQualification
		// widened from '?' to '*' upstream — DuckDB now allows nested schema
		// chains here, e.g. catalog.schema1.schema2.func(...).)
		seq := body(alt)
		parts := []string{literalText(body(seq.Children[0]).Children[0])}
		for _, s := range repeatChildren(seq.Children[1]) {
			parts = append(parts, literalText(body(s).Children[0]))
		}
		return append(parts, literalText(seq.Children[2]))
	case "SchemaReservedFunctionName":
		// SchemaQualification ReservedFunctionName
		seq := body(alt)
		return []string{literalText(body(seq.Children[0]).Children[0]), literalText(seq.Children[1])}
	default: // FunctionNameAsQualifiedName <- FunctionName
		return []string{literalText(body(alt))}
	}
}

// adaptCallArgList reads a "DistinctOrAll? ArgList? OrderByClause?
// IgnoreOrRespectNulls?" shaped node — the body of
// FunctionExpressionArgumentList or MethodExpressionArgumentList (identical
// shape, different arg-list rule name underneath, both List(FunctionArgument)).
func (a *adapter) adaptCallArgList(seq *peg.Node) (args []Expr, orderBy *OrderByClause, distinct, all, ignoreNulls, respectNulls bool) {
	if d, ok := present(seq.Children[0]); ok {
		switch choice(body(d)).Name {
		case "DistinctKeyword":
			distinct = true
		case "AllKeyword":
			all = true
		}
	}
	if argsNode, ok := present(seq.Children[1]); ok {
		listSeq := body(body(argsNode)) // List(FunctionArgument)
		args = []Expr{a.adaptFunctionArgument(listSeq.Children[0])}
		for _, tail := range listSeq.Children[1].Children {
			args = append(args, a.adaptFunctionArgument(body(tail).Children[1]))
		}
	}
	if o, ok := present(seq.Children[2]); ok {
		orderBy = a.adaptOrderByClause(o)
	}
	if ig, ok := present(seq.Children[3]); ok {
		switch choice(body(ig)).Name {
		case "IgnoreNulls":
			ignoreNulls = true
		case "RespectNulls":
			respectNulls = true
		}
	}
	return
}

func (a *adapter) adaptFunctionArgument(n *peg.Node) Expr {
	// FunctionArgument <- NamedFunctionArgument / PositionalFunctionArgument
	alt := choice(body(n))
	if alt.Name == "PositionalFunctionArgument" {
		return a.adaptExpression(body(alt)) // PositionalFunctionArgument <- Expression
	}
	// NamedFunctionArgument <- NamedParameter ; NamedParameter <- TypeFuncName Type? NamedParameterAssignment Expression
	npSeq := body(body(alt))
	return &NamedArg{baseExpr: a.at(n), Name: literalText(npSeq.Children[0]), Value: a.adaptExpression(npSeq.Children[3])}
}

func (a *adapter) adaptMethodArgs(n *peg.Node) (args []Expr, orderBy *OrderByClause, distinct, all bool) {
	// MethodExpressionArguments <- Parens(MethodExpressionArgumentList) —
	// same triple-unwrap as adaptFunctionExpression's FunctionExpressionArguments.
	seq := body(body(body(n)).Children[1])
	args, orderBy, distinct, all, _, _ = a.adaptCallArgList(seq)
	return
}

func (a *adapter) adaptColumnReference(n *peg.Node) Expr {
	// ColumnReference <- NestedSchemaTableColumnName / CatalogReservedSchemaTableColumnName / SchemaReservedTableColumnName / TableReservedColumnName / NestedColumnName
	// (NestedSchemaTableColumnName is new as of Task 15's second-commit
	// re-vendor: a 5+-component reference where the qualification is
	// deeper than catalog+schema, e.g. catalog.schema1.schema2.schema3.col.)
	alt := choice(body(n))
	switch alt.Name {
	case "NestedSchemaTableColumnName":
		// CatalogQualification ReservedSchemaQualification ReservedSchemaQualification ReservedSchemaQualification+ ReservedColumnName
		seq := body(alt)
		parts := []string{
			literalText(body(seq.Children[0]).Children[0]),
			literalText(body(seq.Children[1]).Children[0]),
			literalText(body(seq.Children[2]).Children[0]),
		}
		for _, s := range repeatChildren(seq.Children[3]) {
			parts = append(parts, literalText(body(s).Children[0]))
		}
		return &Ident{baseExpr: a.at(alt), Parts: append(parts, literalText(seq.Children[4]))}
	case "CatalogReservedSchemaTableColumnName":
		// CatalogQualification ReservedSchemaQualification ReservedTableQualification ReservedColumnName
		seq := body(alt)
		return &Ident{baseExpr: a.at(alt), Parts: []string{
			literalText(body(seq.Children[0]).Children[0]),
			literalText(body(seq.Children[1]).Children[0]),
			literalText(body(seq.Children[2]).Children[0]),
			literalText(seq.Children[3]),
		}}
	case "SchemaReservedTableColumnName":
		// SchemaQualification ReservedTableQualification ReservedColumnName
		seq := body(alt)
		return &Ident{baseExpr: a.at(alt), Parts: []string{
			literalText(body(seq.Children[0]).Children[0]),
			literalText(body(seq.Children[1]).Children[0]),
			literalText(seq.Children[2]),
		}}
	case "TableReservedColumnName":
		// TableQualification ReservedColumnName
		seq := body(alt)
		return &Ident{baseExpr: a.at(alt), Parts: []string{
			literalText(body(seq.Children[0]).Children[0]),
			literalText(seq.Children[1]),
		}}
	default: // NestedColumnName <- IdentifierDot* ColumnName
		seq := body(alt)
		var parts []string
		for _, d := range body(seq.Children[0]).Children {
			parts = append(parts, literalText(body(d).Children[0])) // IdentifierDot <- Identifier '.'
		}
		parts = append(parts, literalText(seq.Children[1]))
		return &Ident{baseExpr: a.at(alt), Parts: parts}
	}
}

func (a *adapter) adaptListComprehensionExpression(n *peg.Node) Expr {
	// '[' Expression 'FOR' List(ColIdOrString) 'IN' Expression ListComprehensionFilter? ']'
	seq := body(n)
	l := &ListComprehensionExpr{baseExpr: a.at(n), Expr: a.adaptExpression(seq.Children[1]), Source: a.adaptExpression(seq.Children[5])}
	listSeq := body(seq.Children[3])
	l.Vars = []string{literalText(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		l.Vars = append(l.Vars, literalText(body(tail).Children[1]))
	}
	if f, ok := present(seq.Children[6]); ok {
		l.Filter = a.adaptExpression(body(f).Children[1]) // ListComprehensionFilter <- 'IF' Expression
	}
	return l
}

func (a *adapter) adaptListExpression(n *peg.Node) Expr {
	// ListExpression <- ArrayBoundedListExpression / ArrayParensSelect
	alt := choice(body(n))
	if alt.Name == "ArrayParensSelect" {
		// 'ARRAY' Parens(SelectStatementInternal)
		sub := a.adaptParenSelect(body(body(alt).Children[1]))
		return &SubqueryExpr{baseExpr: a.at(n), Select: sub}
	}
	// ArrayBoundedListExpression <- 'ARRAY'? BoundedListExpression ; BoundedListExpression <- '[' List(Expression)? ']'
	bseq := body(body(alt).Children[1])
	var elems []Expr
	if list, ok := present(bseq.Children[1]); ok {
		elems = a.adaptExprList2(list)
	}
	return &ListExpr{baseExpr: a.at(n), Elems: elems}
}

func (a *adapter) adaptStructExpression(n *peg.Node) Expr {
	// StructExpression <- '{' List(StructField)? '}'
	seq := body(n)
	s := &StructExpr{baseExpr: a.at(n)}
	if fields, ok := present(seq.Children[1]); ok {
		listSeq := body(fields)
		add := func(fn *peg.Node) {
			fseq := body(fn) // StructField <- ColIdOrString ':' Expression
			s.Fields = append(s.Fields, &StructField{baseExpr: a.at(fn), Key: literalText(fseq.Children[0]), Value: a.adaptExpression(fseq.Children[2])})
		}
		add(listSeq.Children[0])
		for _, tail := range listSeq.Children[1].Children {
			add(body(tail).Children[1])
		}
	}
	return s
}

// --- shared expression-list readers ---

// adaptExprList2 reads a bare List(Expression) node (no surrounding Parens).
func (a *adapter) adaptExprList2(n *peg.Node) []Expr {
	listSeq := body(n)
	exprs := []Expr{a.adaptExpression(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		exprs = append(exprs, a.adaptExpression(body(tail).Children[1]))
	}
	return exprs
}

// adaptExprList reads a Parens(List(Expression)) node.
func (a *adapter) adaptExprList(n *peg.Node) []Expr {
	return a.adaptExprList2(body(n).Children[1])
}

// --- window functions (OVER, named or inline; ORDER BY; frame clauses) ---

func (a *adapter) adaptOverClause(n *peg.Node) *WindowSpec {
	// OverClause <- 'OVER' WindowFrame ; WindowFrame <- ParensIdentifier / WindowFrameDefinition / IdentifierWindowFrame
	seq := body(n)
	alt := choice(body(seq.Children[1]))
	switch alt.Name {
	case "ParensIdentifier": // Parens(Identifier) — alt's body is a single ExprCall; double-unwrap to its Seq.
		return &WindowSpec{baseExpr: a.at(n), Name: literalText(body(body(alt)).Children[1])}
	case "IdentifierWindowFrame": // <- Identifier
		return &WindowSpec{baseExpr: a.at(n), Name: literalText(body(alt))}
	default:
		return a.adaptWindowFrameDefinition(alt)
	}
}

func (a *adapter) adaptWindowFrameDefinition(n *peg.Node) *WindowSpec {
	// WindowFrameDefinition <- WindowFrameNameContentsParens / WindowFrameContentsParens
	alt := choice(body(n))
	spec := &WindowSpec{baseExpr: a.at(n)}
	var contents *peg.Node
	if alt.Name == "WindowFrameNameContentsParens" {
		// WindowFrameNameContentsParens <- Parens(WindowFrameNameContents)
		// WindowFrameNameContents <- BaseWindowName? WindowFrameContents
		// alt's own body is a single ExprCall (Parens(...)), so body(alt)
		// only unwraps to the "Parens" KindRule node itself; a second
		// body() reaches its Seq.
		ncSeq := body(body(body(alt)).Children[1])
		if nameNode, ok := present(ncSeq.Children[0]); ok {
			spec.Name = literalText(nameNode) // BaseWindowName <- Identifier
		}
		contents = ncSeq.Children[1]
	} else {
		// WindowFrameContentsParens <- Parens(WindowFrameContents) — same double-unwrap as above.
		contents = body(body(alt)).Children[1]
	}
	// WindowFrameContents <- WindowPartition? OrderByClause? FrameClause?
	cseq := body(contents)
	if p, ok := present(cseq.Children[0]); ok {
		// WindowPartition <- 'PARTITION' 'BY' List(Expression)
		spec.PartitionBy = a.adaptExprList2(body(p).Children[2])
	}
	if o, ok := present(cseq.Children[1]); ok {
		spec.OrderBy = a.adaptOrderByClause(o)
	}
	if f, ok := present(cseq.Children[2]); ok {
		spec.Frame = a.adaptFrameClause(f)
	}
	return spec
}

func (a *adapter) adaptOrderByClause(n *peg.Node) *OrderByClause {
	// OrderByClause <- 'ORDER' 'BY' OrderByExpressions ; OrderByExpressions <- OrderByAll / OrderByExpressionList
	seq := body(n)
	alt := choice(body(seq.Children[2]))
	if alt.Name == "OrderByAll" {
		// OrderByAll <- 'ALL' DescOrAsc? NullsFirstOrLast?
		aseq := body(alt)
		item := &OrderByItem{baseExpr: a.at(alt)}
		if d, ok := present(aseq.Children[1]); ok {
			item.Desc = choice(body(d)).Name == "DescendingOrder"
		}
		if nf, ok := present(aseq.Children[2]); ok {
			item.HasNulls = true
			item.NullsFirst = choice(body(nf)).Name == "NullsFirst"
		}
		return &OrderByClause{baseExpr: a.at(n), All: true, Items: []*OrderByItem{item}}
	}
	// OrderByExpressionList <- List(OrderByExpression)
	listSeq := body(body(alt))
	items := []*OrderByItem{a.adaptOrderByExpression(listSeq.Children[0])}
	for _, tail := range listSeq.Children[1].Children {
		items = append(items, a.adaptOrderByExpression(body(tail).Children[1]))
	}
	return &OrderByClause{baseExpr: a.at(n), Items: items}
}

func (a *adapter) adaptOrderByExpression(n *peg.Node) *OrderByItem {
	// OrderByExpression <- Expression DescOrAsc? NullsFirstOrLast?
	seq := body(n)
	item := &OrderByItem{baseExpr: a.at(n), X: a.adaptExpression(seq.Children[0])}
	if d, ok := present(seq.Children[1]); ok {
		item.Desc = choice(body(d)).Name == "DescendingOrder"
	}
	if nf, ok := present(seq.Children[2]); ok {
		item.HasNulls = true
		item.NullsFirst = choice(body(nf)).Name == "NullsFirst"
	}
	return item
}

func (a *adapter) adaptFrameClause(n *peg.Node) *FrameClause {
	// FrameClause <- Framing FrameExtent WindowExcludeClause?
	seq := body(n)
	fc := &FrameClause{baseExpr: a.at(n), Unit: literalText(seq.Children[0])}
	alt := choice(body(seq.Children[1])) // FrameExtent <- BetweenFrameExtent / SingleFrameExtent
	if alt.Name == "BetweenFrameExtent" {
		// 'BETWEEN' FrameBound 'AND' FrameBound
		bseq := body(alt)
		fc.Start = a.adaptFrameBound(bseq.Children[1])
		fc.EndBound = a.adaptFrameBound(bseq.Children[3])
	} else {
		fc.Start = a.adaptFrameBound(body(alt)) // SingleFrameExtent <- FrameBound
	}
	if exc, ok := present(seq.Children[2]); ok {
		// WindowExcludeClause <- 'EXCLUDE' WindowExcludeElement
		fc.Exclude = literalText(body(exc).Children[1])
	}
	return fc
}

func (a *adapter) adaptFrameBound(n *peg.Node) *FrameBound {
	// FrameBound <- FrameUnbounded / FrameCurrentRow / FrameExpression
	alt := choice(body(n))
	switch alt.Name {
	case "FrameUnbounded":
		// 'UNBOUNDED' PrecedingOrFollowing
		if choice(body(body(alt).Children[1])).Name == "PrecedingFrame" {
			return &FrameBound{baseExpr: a.at(n), Kind: FrameUnboundedPreceding}
		}
		return &FrameBound{baseExpr: a.at(n), Kind: FrameUnboundedFollowing}
	case "FrameCurrentRow":
		return &FrameBound{baseExpr: a.at(n), Kind: FrameCurrentRow}
	default: // FrameExpression <- Expression PrecedingOrFollowing
		seq := body(alt)
		expr := a.adaptExpression(seq.Children[0])
		if choice(body(seq.Children[1])).Name == "PrecedingFrame" {
			return &FrameBound{baseExpr: a.at(n), Kind: FramePreceding, Expr: expr}
		}
		return &FrameBound{baseExpr: a.at(n), Kind: FrameFollowing, Expr: expr}
	}
}
