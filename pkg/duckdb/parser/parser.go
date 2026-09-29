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
	return &peg.Parser{Grammar: g, Primitives: Primitives(ks), SkipTrivia: SkipSQLTrivia, SymbolTokens: tokens}, ks, nil
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

// ParseString parses a single DuckDB SELECT statement into a typed AST. A
// trailing ';' and surrounding whitespace are tolerated; anything else
// after the statement is a parse error.
func ParseString(sql string) (stmt *SelectStatement, err error) {
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

	cst, parseErr := sess.Parse(trimmed, "SelectStatement")
	if parseErr != nil {
		pos := Position{}
		msg := parseErr.Error()
		var perr *peg.ParseError
		if errors.As(parseErr, &perr) {
			pos = positionAt(trimmed, perr.Pos)
			msg = perr.Message
		}
		return nil, &ParseError{Pos: pos, Message: msg}
	}

	defer func() {
		if r := recover(); r != nil {
			stmt, err = nil, fmt.Errorf("duckdb/parser: %v", r)
		}
	}()
	a := newAdapter(trimmed)
	// SelectStatement <- SelectStatementInternal
	return a.adaptSelectStatementInternal(body(cst)), nil
}
