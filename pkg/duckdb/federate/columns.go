package federate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// EncodedDeltaStreamer is an optional DeltaStreamer extension: it streams a
// delta already encoded for quack (driver.EncodeQuackVectors), column by
// column, so the executor's streaming path sends it without boxing each
// value. The executor prefers it to Stream.
type EncodedDeltaStreamer interface {
	StreamEncoded(ctx context.Context, req DeltaRequest) (EncodedReader, error)
}

// EncodedReader is a delta stream of encoded batches. NextEncoded returns
// io.EOF once the stream is exhausted. Close is safe to call more than once.
type EncodedReader interface {
	Schema() []Column
	NextEncoded(ctx context.Context) (blob []byte, chunks uint64, rows int, err error)
	Close() error
}

// encodedDeltaStream is deltaStream for an EncodedReader.
func encodedDeltaStream(t Table, r EncodedReader, rowCap int, kind deltaKind) (driver.QuackStream, string, error) {
	cols := r.Schema()
	if len(cols) == 0 {
		return driver.QuackStream{}, "", fmt.Errorf("federate: delta for %s has no columns", t)
	}
	kinds, scan, id, err := deltaScanSQL(cols)
	if err != nil {
		return driver.QuackStream{}, "", err
	}
	def := fmt.Sprintf("%s AS MATERIALIZED (%s)", quoteIdent(deltaCTEName(t)), scan)
	count := &deltaCount{table: t, kind: kind, rowCap: rowCap}
	next := func(ctx context.Context) ([]byte, uint64, error) {
		blob, chunks, rows, err := r.NextEncoded(ctx)
		if errors.Is(err, io.EOF) {
			count.done(ctx)
		}
		if err != nil {
			return nil, 0, err // io.EOF ends the stream
		}
		if err := count.add(ctx, rows); err != nil {
			return nil, 0, fmt.Errorf("%w: %s delta has more than %d rows", err, t, rowCap)
		}
		return blob, chunks, nil
	}
	return driver.QuackStream{ID: id, Columns: kinds, NextEncoded: next}, def, nil
}

// deltaColumnsUsed works out which delta columns the query reads, by
// planning it with each source as its lake table alone, unbounded:
// everything above a source's UNION reads its lake and delta sides alike, so
// the delta needs exactly the columns the plan reads from the lake table
// (projected, or named in a filter pushed into the scan). The macros' own
// reads (raw_runs' collapse, for one) are included because they're part of
// the planned query.
//
// A table maps to nil when the plan has no scan of its lake table: stream
// every column. That covers a table the query never reads, too: a missing
// scan can't be told apart from one an optimization removed, so it stays
// conservative.
func deltaColumnsUsed(ctx context.Context, conn *sql.Conn, q Query, needed []Table) (map[Table][]Column, error) {
	if len(needed) == 0 {
		return nil, nil
	}
	sqlText, args, err := assemble(q, needed, map[Table]string{}, nil, nil)
	if err != nil {
		return nil, err
	}
	// With statistics, the planner can see a lake table (or its pruned
	// files) is empty and replace the scan with an empty result, hiding the
	// columns. Planning without those optimizations keeps the scan.
	if _, err := conn.ExecContext(ctx, "SET disabled_optimizers = 'statistics_propagation,empty_result_pullup';"); err != nil {
		return nil, nil
	}
	plan, err := explainJSON(ctx, conn, sqlText, args)
	_, _ = conn.ExecContext(context.Background(), "RESET disabled_optimizers;")
	if err != nil {
		// Planning is only an optimization: stream every column.
		return nil, nil
	}

	out := map[Table][]Column{}
	for _, t := range needed {
		names, found := scanColumns(plan, Physical[t].Name, DeltaColumns(t))
		if !found || len(names) == 0 {
			out[t] = nil
			continue
		}
		cols := []Column{}
		for _, c := range DeltaColumns(t) {
			if names[c.Name] {
				cols = append(cols, c)
			}
		}
		if len(cols) == 0 {
			// The scan reads rows but no columns (count(*) over a plain
			// union): stream one, so the delta still has its rows.
			cols = DeltaColumns(t)[:1]
		}
		out[t] = cols
	}
	return out, nil
}

type planNode struct {
	Name      string          `json:"name"`
	ExtraInfo json.RawMessage `json:"extra_info"`
	Children  []planNode      `json:"children"`
}

func explainJSON(ctx context.Context, conn *sql.Conn, sqlText string, args []any) ([]planNode, error) {
	rows, err := conn.QueryContext(ctx, "EXPLAIN (FORMAT JSON) "+sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var plan []planNode
	for rows.Next() {
		var key, value sql.NullString
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		var nodes []planNode
		if err := json.Unmarshal([]byte(value.String), &nodes); err != nil {
			return nil, fmt.Errorf("federate: parsing plan: %w", err)
		}
		plan = append(plan, nodes...)
	}
	if err := rows.Err(); err != nil && err != io.EOF {
		return nil, err
	}
	return plan, nil
}

// scanColumns returns the columns the plan reads from table: each scan of it
// contributes its projections, and any of cols named in its filters.
func scanColumns(plan []planNode, table string, cols []Column) (map[string]bool, bool) {
	names := map[string]bool{}
	found := false
	var walk func(n planNode)
	walk = func(n planNode) {
		if strings.Contains(n.Name, "SCAN") {
			var info map[string]any
			_ = json.Unmarshal(n.ExtraInfo, &info)
			if tbl, _ := info["Table"].(string); tableIs(tbl, table) {
				found = true
				for _, p := range stringsOf(info["Projections"]) {
					names[strings.Trim(p, `"`)] = true
				}
				for k, v := range info {
					if !strings.Contains(strings.ToLower(k), "filter") {
						continue
					}
					for _, f := range stringsOf(v) {
						for _, c := range cols {
							if identIn(f, c.Name) {
								names[c.Name] = true
							}
						}
					}
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, n := range plan {
		walk(n)
	}
	return names, found
}

// tableIs reports whether a plan's table reference (bare, or qualified as
// catalog.schema.x) names table.
func tableIs(ref, table string) bool {
	ref = strings.ReplaceAll(ref, `"`, "")
	return ref == table || strings.HasSuffix(ref, "."+table)
}

func stringsOf(v any) []string {
	switch x := v.(type) {
	case string:
		return strings.Split(x, "\n")
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// identIn reports whether name appears in expr as a whole identifier.
func identIn(expr, name string) bool {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_])"?` + regexp.QuoteMeta(name) + `"?($|[^A-Za-z0-9_])`).MatchString(expr)
}

func columnSet(cols []Column) map[string]bool {
	set := make(map[string]bool, len(cols))
	for _, c := range cols {
		set[c.Name] = true
	}
	return set
}
