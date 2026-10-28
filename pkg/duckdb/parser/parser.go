package parser

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/inngest/inngest/pkg/duckdb/parser/grammar"
	"github.com/inngest/inngest/pkg/duckdb/parser/peg"
)

var (
	pegParser     *peg.Parser
	pegParserOnce sync.Once
	pegParserErr  error

	// keywordSets is the vendored grammar's keyword lists. Every keyword that
	// isn't purely unreserved — reserved, column-name, func-name and
	// type-name — is rejected as a bare identifier in at least one position
	// (see ColId/TableName in base.gram/common.gram), so Write must quote them.
	keywordSets *KeywordSets
)

func loadPegParser() (*peg.Parser, error) {
	pegParserOnce.Do(func() {
		pegParser, keywordSets, pegParserErr = newPegParser()
	})
	return pegParser, pegParserErr
}

// newPegParser builds a fresh peg.Parser over the vendored grammar. Most
// callers want the shared loadPegParser instead.
func newPegParser() (*peg.Parser, *KeywordSets, error) {
	g, kl, err := grammar.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("duckdb/parser: loading vendored grammar: %w", err)
	}
	tokens, err := symbolTokens(g)
	if err != nil {
		return nil, nil, fmt.Errorf("duckdb/parser: %w", err)
	}
	ks := NewKeywordSets(kl.Reserved, kl.Unreserved, kl.ColumnName, kl.FuncName, kl.TypeName)
	return &peg.Parser{Grammar: g, Primitives: Primitives(ks), SkipTrivia: SkipSQLTrivia, SymbolTokens: tokens, MaxDepth: maxRuleDepth, MaxEntries: maxParseEntries}, ks, nil
}

// isNonIdentKeyword reports whether s, spelled bare, would lex as a keyword
// that can't stand in for an identifier everywhere one is allowed.
func isNonIdentKeyword(s string) bool {
	if _, err := loadPegParser(); err != nil {
		return false
	}
	up := strings.ToUpper(s)
	for _, set := range []map[string]struct{}{keywordSets.Reserved, keywordSets.ColumnName, keywordSets.FuncName, keywordSets.TypeName} {
		if _, ok := set[up]; ok {
			return true
		}
	}
	return false
}

// isFuncNameIdent reports whether s, spelled bare, lexes as a FunctionName
// (an identifier, or a keyword the function-name category allows).
func isFuncNameIdent(s string) bool {
	if _, err := loadPegParser(); err != nil {
		return false
	}
	return keywordSets.allowedAsIdentifier([]byte(strings.ToUpper(s)), CategoryFuncName)
}

// isEmptyQuotedIdent reports whether src has an empty `""` identifier at
// pos, which the identifier primitives refuse to match (see
// scanQuotedIdentifier), so a parse that fails there gets DuckDB's own
// error message instead of a generic one.
func isEmptyQuotedIdent(src string, pos int) bool {
	return pos >= 0 && strings.HasPrefix(src[min(pos, len(src)):], `""`) && !strings.HasPrefix(src[min(pos, len(src)):], `"""`)
}

// MaxSQLBytes is the longest input ParseString accepts, rejected up front
// rather than parsed. Memory is bounded separately by maxParseEntries,
// which a dense input reaches well before this; this mainly bounds the
// work of scanning trivia (whitespace, comments) that creates no entries.
const MaxSQLBytes = 64 << 10

// maxRuleDepth caps nested grammar-rule invocations (peg.Parser.MaxDepth).
// Each level of parenthesized nesting costs about 20 rules and each rule a
// few KB of goroutine stack, so this allows roughly 200 levels of nesting —
// far beyond any realistic query — while keeping the stack in the low tens
// of MB instead of overflowing it, which Go can't recover from.
const maxRuleDepth = 4096

// maxParseEntries caps the packrat memo entries plus CST nodes one parse
// may create (peg.Parser.MaxEntries). Realistic queries use 10-30 per
// input byte, so this admits typical queries of tens of KB, while a dense
// worst case (`SELECT a, a, ...`, ~100 per byte) is cut off around 10KB
// instead of allocating ~10KB of memo/slab memory per input byte up to
// MaxSQLBytes (~2GB).
const maxParseEntries = 512 << 10

// errPrefix starts every non-ParseError error ParseString returns.
const errPrefix = "duckdb/parser: "

// ParseString parses a single DuckDB SELECT statement into a typed AST. A
// trailing ';' and surrounding whitespace are tolerated; anything else
// after the statement is a parse error, as is input longer than
// MaxSQLBytes or nested too deeply to parse safely.
func ParseString(sql string) (stmt *SelectStatement, err error) {
	if len(sql) > MaxSQLBytes {
		return nil, &ParseError{Pos: positionAt(sql, 0), Message: fmt.Sprintf("query is %d bytes, exceeding the %d-byte limit", len(sql), MaxSQLBytes)}
	}

	p, loadErr := loadPegParser()
	if loadErr != nil {
		return nil, loadErr
	}

	trimmed := strings.TrimSpace(sql)
	trimmed = strings.TrimSuffix(trimmed, ";")

	// A Session reuses its evaluator's scratch buffers across calls (see
	// Session's doc comment) instead of paying for fresh ones on every
	// parse. Safe here because cst is fully consumed (adapted into stmt,
	// which holds no *peg.Node references) before Release runs.
	sess := p.AcquireSession()
	defer sess.Release()

	// Registered before sess.Parse, not just before adapting, so a panic
	// from the peg engine or a primitive is also returned as an error.
	// An unsupported construct panics with a *ParseError (see
	// adapter.unsupported), returned as-is. The adapter's other panics
	// already carry the package prefix; only foreign ones (runtime errors,
	// the peg engine) need it added.
	defer func() {
		if r := recover(); r != nil {
			if perr, ok := r.(*ParseError); ok {
				stmt, err = nil, perr
				return
			}
			msg := fmt.Sprint(r)
			if !strings.HasPrefix(msg, errPrefix) {
				msg = errPrefix + msg
			}
			stmt, err = nil, errors.New(msg)
		}
	}()

	cst, parseErr := sess.Parse(trimmed, "SelectStatement")
	if parseErr != nil {
		pos := Position{}
		msg := parseErr.Error()
		var perr *peg.ParseError
		if errors.As(parseErr, &perr) {
			pos = positionAt(trimmed, perr.Pos)
			msg = perr.Message
		}
		if errors.Is(parseErr, peg.ErrTooDeep) {
			msg = "query is nested too deeply"
		} else if errors.Is(parseErr, peg.ErrTooLarge) {
			msg = "query is too large or complex to parse"
		} else if perr != nil && isEmptyQuotedIdent(trimmed, perr.Pos) {
			msg = "zero-length delimited identifier"
		}
		return nil, &ParseError{Pos: pos, Message: msg}
	}

	a := newAdapter(trimmed)
	// SelectStatement <- SelectStatementInternal
	return a.adaptSelectStatementInternal(body(cst)), nil
}
