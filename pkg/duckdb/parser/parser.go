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
)

func loadPegParser() (*peg.Parser, error) {
	pegParserOnce.Do(func() {
		g, kl, err := grammar.Load()
		if err != nil {
			pegParserErr = fmt.Errorf("duckdb/parser: loading vendored grammar: %w", err)
			return
		}
		ks := NewKeywordSets(kl.Reserved, kl.Unreserved, kl.ColumnName, kl.FuncName, kl.TypeName)
		pegParser = &peg.Parser{Grammar: g, Primitives: Primitives(ks), SkipTrivia: SkipSQLTrivia}
	})
	return pegParser, pegParserErr
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
