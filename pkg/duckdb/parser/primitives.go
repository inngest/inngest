package parser

import (
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

type KeywordCategory int

const (
	CategoryColumnName KeywordCategory = iota
	CategoryTypeName
	CategoryFuncName
)

type KeywordSets struct {
	Reserved, Unreserved, ColumnName, FuncName, TypeName map[string]struct{}
}

func toSet(words []string) map[string]struct{} {
	s := make(map[string]struct{}, len(words))
	for _, w := range words {
		s[strings.ToUpper(w)] = struct{}{}
	}
	return s
}

func NewKeywordSets(reserved, unreserved, columnName, funcName, typeName []string) *KeywordSets {
	return &KeywordSets{
		Reserved:   toSet(reserved),
		Unreserved: toSet(unreserved),
		ColumnName: toSet(columnName),
		FuncName:   toSet(funcName),
		TypeName:   toSet(typeName),
	}
}

func (ks *KeywordSets) isAnyKeyword(upper []byte) bool {
	for _, set := range []map[string]struct{}{ks.Reserved, ks.Unreserved, ks.ColumnName, ks.FuncName, ks.TypeName} {
		if _, ok := set[string(upper)]; ok {
			return true
		}
	}
	return false
}

// allowedAsIdentifier mirrors IdentifierMatcher::IsAllowedKeyword (DuckDB's
// matcher/identifier_matcher.hpp at the pinned commit): a word is fine as
// an identifier unless it's a keyword outside both the unreserved set and
// the specific category this identifier kind allows.
func (ks *KeywordSets) allowedAsIdentifier(upper []byte, allowed KeywordCategory) bool {
	if !ks.isAnyKeyword(upper) {
		return true
	}
	if _, ok := ks.Unreserved[string(upper)]; ok {
		return true
	}
	var categorySet map[string]struct{}
	switch allowed {
	case CategoryTypeName:
		categorySet = ks.TypeName
	case CategoryFuncName:
		categorySet = ks.FuncName
	default:
		categorySet = ks.ColumnName
	}
	_, ok := categorySet[string(upper)]
	return ok
}

func identStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func identCont(c byte) bool {
	return identStart(c) || (c >= '0' && c <= '9')
}

// upperASCII uppercases s (ASCII only — SQL keyword/identifier text always
// is) into buf and returns the filled prefix, falling back to a regular
// allocation only if s doesn't fit (identifiers/keywords are essentially
// never that long). Every caller uses the result as `set[string(result)]` —
// that exact shape is a compiler-recognized idiom for a map lookup that
// skips the allocation a plain strings.ToUpper + index would need, which
// matters here since identifier/keyword primitives run speculatively very
// often during backtracking.
func upperASCII(buf []byte, s string) []byte {
	if len(s) > len(buf) {
		return []byte(strings.ToUpper(s))
	}
	b := buf[:len(s)]
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b[i] = c
	}
	return b
}

// scanQuotedIdentifier scans a `"..."` token starting at pos (input[pos] ==
// '"'), unescaping "" -> " as it goes.
func scanQuotedIdentifier(input string, pos int) (decoded string, next int, ok bool) {
	var b strings.Builder
	i := pos + 1
	for {
		if i >= len(input) {
			return "", 0, false
		}
		if input[i] == '"' {
			if i+1 < len(input) && input[i+1] == '"' {
				b.WriteByte('"')
				i += 2
				continue
			}
			return b.String(), i + 1, true
		}
		b.WriteByte(input[i])
		i++
	}
}

func scanWord(input string, pos int) (word string, next int, ok bool) {
	if pos >= len(input) || !identStart(input[pos]) {
		return "", 0, false
	}
	i := pos + 1
	for i < len(input) && identCont(input[i]) {
		i++
	}
	return input[pos:i], i, true
}

func identifierPrimitive(allowed KeywordCategory, ks *KeywordSets) peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		if pos < len(input) && input[pos] == '"' {
			decoded, next, ok := scanQuotedIdentifier(input, pos)
			if !ok {
				return nil, 0, false
			}
			return newNode(peg.Node{Kind: peg.KindPrimitive, Text: input[pos:next], Value: decoded, HasValue: true, Start: pos, End: next}), next, true
		}
		word, next, ok := scanWord(input, pos)
		var buf [64]byte
		if !ok || !ks.allowedAsIdentifier(upperASCII(buf[:], word), allowed) {
			return nil, 0, false
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Text: word, Value: word, HasValue: true, Start: pos, End: next}), next, true
	}
}

func permissiveIdentifierPrimitive() peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		if pos < len(input) && input[pos] == '"' {
			decoded, next, ok := scanQuotedIdentifier(input, pos)
			if !ok {
				return nil, 0, false
			}
			return newNode(peg.Node{Kind: peg.KindPrimitive, Text: input[pos:next], Value: decoded, HasValue: true, Start: pos, End: next}), next, true
		}
		word, next, ok := scanWord(input, pos)
		if !ok {
			return nil, 0, false
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Text: word, Value: word, HasValue: true, Start: pos, End: next}), next, true
	}
}

// keywordCategoryPrimitive backs UnreservedKeyword/ColumnNameKeyword/
// FuncNameKeyword/TypeNameKeyword. In our vendored closure these are
// referenced *only* as identifier-fallback alternatives inside ColId/
// ColLabel/TypeFuncName (base.gram) — e.g. "next" is classified as an
// unreserved keyword, but `WITH next AS (...)` still needs the CTE's name
// to come back as "next", not "NEXT". So Value preserves the original
// casing like an identifier, even though membership is checked
// case-insensitively.
func keywordCategoryPrimitive(set map[string]struct{}) peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		word, next, ok := scanWord(input, pos)
		if !ok {
			return nil, 0, false
		}
		var buf [64]byte
		if _, ok := set[string(upperASCII(buf[:], word))]; !ok {
			return nil, 0, false
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Text: word, Value: word, HasValue: true, Start: pos, End: next}), next, true
	}
}

func numberLiteralPrimitive() peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		i := pos
		if i < len(input) && (input[i] == '+' || input[i] == '-') {
			i++
		}
		start := i
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			i++
		}
		hasDigits := i > start
		if i < len(input) && input[i] == '.' {
			i++
			for i < len(input) && input[i] >= '0' && input[i] <= '9' {
				i++
				hasDigits = true
			}
		}
		if !hasDigits {
			return nil, 0, false
		}
		if i < len(input) && (input[i] == 'e' || input[i] == 'E') {
			j := i + 1
			if j < len(input) && (input[j] == '+' || input[j] == '-') {
				j++
			}
			expStart := j
			for j < len(input) && input[j] >= '0' && input[j] <= '9' {
				j++
			}
			if j > expStart {
				i = j
			}
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Text: input[pos:i], Value: input[pos:i], HasValue: true, Start: pos, End: i}), i, true
	}
}

func stringLiteralPrimitive() peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		if pos >= len(input) || input[pos] != '\'' {
			return nil, 0, false
		}
		var b strings.Builder
		i := pos + 1
		for {
			if i >= len(input) {
				return nil, 0, false
			}
			if input[i] == '\'' {
				if i+1 < len(input) && input[i+1] == '\'' {
					b.WriteByte('\'')
					i += 2
					continue
				}
				return newNode(peg.Node{Kind: peg.KindPrimitive, Text: input[pos : i+1], Value: b.String(), HasValue: true, Start: pos, End: i + 1}), i + 1, true
			}
			b.WriteByte(input[i])
			i++
		}
	}
}

const operatorChars = "+-*/%^<>=~!@#&|?"

// operatorLiteralReserved holds operator-symbol tokens that OperatorLiteral
// must never claim, even though every one of their characters is in
// operatorChars — each is reserved for a different, non-adjacent grammar
// rule (LambdaArrowExpression's '->', not any of NamedOtherOperator's own
// sibling alternatives). Comparison/arithmetic/bitwise operators ('=', '+',
// '&', ...) don't need an entry here even though OperatorLiteral can also
// match them at the wrong grammar level (OtherOperatorExpression sits
// *inside* ComparisonExpression's own head-recursion, so it sees them
// first) — that misrouting is harmless because both paths build the exact
// same BinaryExpr{Op, Left, Right} shape. '->' is the one case with an
// observable difference: LambdaArrowExpression's handling isn't a generic
// BinaryExpr, so losing the token to OperatorLiteral silently drops lambda
// support instead of just taking a different internal path to the same AST.
var operatorLiteralReserved = map[string]struct{}{
	"->": {},
}

func operatorLiteralPrimitive() peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		i := pos
		for i < len(input) && strings.IndexByte(operatorChars, input[i]) >= 0 {
			i++
		}
		if i == pos {
			return nil, 0, false
		}
		text := input[pos:i]
		if _, reserved := operatorLiteralReserved[text]; reserved {
			return nil, 0, false
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Text: text, Value: text, HasValue: true, Start: pos, End: i}), i, true
	}
}

func endOfInputPrimitive() peg.Primitive {
	return func(newNode func(peg.Node) *peg.Node, input string, pos int) (*peg.Node, int, bool) {
		if pos != len(input) {
			return nil, 0, false
		}
		return newNode(peg.Node{Kind: peg.KindPrimitive, Start: pos, End: pos}), pos, true
	}
}

// SkipSQLTrivia skips whitespace, `--` line comments, and (possibly
// nested) `/* ... */` block comments.
func SkipSQLTrivia(input string, pos int) int {
	for {
		start := pos
		for pos < len(input) {
			switch input[pos] {
			case ' ', '\t', '\n', '\r':
				pos++
				continue
			}
			break
		}
		switch {
		case strings.HasPrefix(input[pos:], "--"):
			for pos < len(input) && input[pos] != '\n' {
				pos++
			}
		case strings.HasPrefix(input[pos:], "/*"):
			depth := 1
			pos += 2
			for pos < len(input) && depth > 0 {
				switch {
				case strings.HasPrefix(input[pos:], "/*"):
					depth++
					pos += 2
				case strings.HasPrefix(input[pos:], "*/"):
					depth--
					pos += 2
				default:
					pos++
				}
			}
		}
		if pos == start {
			return pos
		}
	}
}

// neverMatch is used for grammar rules our vendored closure references but
// never vendors (Statement, InsertColumnList — see Task 11's plan entry
// for why): "never matches" is exactly correct here, not a stand-in for
// missing behavior, since this package genuinely never supports either.
func neverMatch(_ func(peg.Node) *peg.Node, _ string, _ int) (*peg.Node, int, bool) {
	return nil, 0, false
}

func Primitives(ks *KeywordSets) map[string]peg.Primitive {
	col := identifierPrimitive(CategoryColumnName, ks)
	fn := identifierPrimitive(CategoryFuncName, ks)
	typ := identifierPrimitive(CategoryTypeName, ks)
	reserved := permissiveIdentifierPrimitive()
	return map[string]peg.Primitive{
		"Identifier":   col,
		"CatalogName":  col,
		"SchemaName":   col,
		"TableName":    col,
		"ColumnName":   col,
		"IndexName":    col,
		"SequenceName": col,
		"PragmaName":   col,
		"SettingName":  col,

		"FunctionName":      fn,
		"TableFunctionName": fn,
		"TypeName":          typ,

		"ReservedIdentifier":   reserved,
		"ReservedSchemaName":   reserved,
		"ReservedTableName":    reserved,
		"ReservedColumnName":   reserved,
		"ReservedIndexName":    reserved,
		"ReservedFunctionName": reserved,
		"ReservedTypeName":     reserved,
		"CopyOptionName":       reserved,
		"ReservedKeyword":      reserved,

		"UnreservedKeyword": keywordCategoryPrimitive(ks.Unreserved),
		"ColumnNameKeyword": keywordCategoryPrimitive(ks.ColumnName),
		"FuncNameKeyword":   keywordCategoryPrimitive(ks.FuncName),
		"TypeNameKeyword":   keywordCategoryPrimitive(ks.TypeName),

		"NumberLiteral":   numberLiteralPrimitive(),
		"StringLiteral":   stringLiteralPrimitive(),
		"OperatorLiteral": operatorLiteralPrimitive(),
		"EndOfInput":      endOfInputPrimitive(),

		"Statement":        neverMatch,
		"InsertColumnList": neverMatch,
	}
}
