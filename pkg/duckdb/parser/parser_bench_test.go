package parser

import (
	"strings"
	"testing"

	"github.com/inngest/inngest/pkg/duckdb/parser/grammar"
)

// benchQueries spans the fixture categories in testdata/queries — from a
// trivial single-column select up to a query exercising several features
// at once (CTE, join, window function, aggregation) — so the benchmark
// reflects more than just the cheapest possible parse.
var benchQueries = []struct {
	name string
	sql  string
}{
	{"Trivial", "SELECT 1"},
	{"Simple", "SELECT a, b FROM orders WHERE a = 1"},
	{"Operators", "SELECT a FROM t WHERE a > 1 AND b < 2 OR c = 3 AND d BETWEEN 1 AND 10"},
	{"Join", "SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id"},
	{"Subquery", "SELECT a FROM t WHERE a IN (SELECT id FROM other WHERE other.id = t.id)"},
	{"WindowFunction", "SELECT row_number() OVER (PARTITION BY a ORDER BY b) FROM t"},
	{"CTE", "WITH cte AS (SELECT a FROM t WHERE a > 0) SELECT a FROM cte ORDER BY a LIMIT 10"},
	{"Complex", `
WITH ranked AS (
	SELECT
		o.id,
		o.customer_id,
		c.name,
		sum(o.amount) FILTER (WHERE o.status = 'paid') AS total,
		row_number() OVER (PARTITION BY o.customer_id ORDER BY o.created_at DESC) AS rn
	FROM orders o
	JOIN customers c ON c.id = o.customer_id
	LEFT JOIN refunds r ON r.order_id = o.id
	WHERE o.created_at > '2024-01-01' AND (o.status = 'paid' OR o.status = 'pending')
	GROUP BY o.id, o.customer_id, c.name
	HAVING sum(o.amount) > 100
)
SELECT * FROM ranked WHERE rn = 1 ORDER BY total DESC LIMIT 20
`},
}

// BenchmarkParseString measures steady-state parsing throughput via the
// public entry point — the vendored grammar is loaded once (sync.Once) on
// the first call, so after warmup this reflects real repeated-parse usage,
// not cold-start cost (see BenchmarkGrammarLoad for that separately).
func BenchmarkParseString(b *testing.B) {
	// Pay the one-time grammar-load cost before any sub-benchmark's timer
	// starts, so it doesn't bleed into the first table entry's numbers.
	if _, err := ParseString("SELECT 1"); err != nil {
		b.Fatal(err)
	}

	for _, q := range benchQueries {
		b.Run(q.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(q.sql)))
			for b.Loop() {
				if _, err := ParseString(q.sql); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPegParse measures the raw peg CST parse in isolation — the same
// peg.Parser.Parse call ParseString makes, but without the adapter walk that
// builds the typed AST afterward. Comparing this against BenchmarkParseString
// shows how much of ParseString's cost is grammar matching versus adaptation.
func BenchmarkPegParse(b *testing.B) {
	p, err := loadPegParser()
	if err != nil {
		b.Fatal(err)
	}

	for _, q := range benchQueries {
		sql := strings.TrimSuffix(strings.TrimSpace(q.sql), ";")
		b.Run(q.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(sql)))
			for b.Loop() {
				if _, err := p.Parse(sql, "SelectStatement"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkPegParseSession measures the same raw peg CST parse as
// BenchmarkPegParse, but through a pooled peg.Session (reused across every
// iteration) instead of a bare Parser.Parse call — the same reuse
// ParseString itself now does internally. Comparing this against
// BenchmarkPegParse isolates how much of a single parse's cost is the
// per-call node/child/env slab and memo-map allocation that a warm Session
// avoids.
func BenchmarkPegParseSession(b *testing.B) {
	p, err := loadPegParser()
	if err != nil {
		b.Fatal(err)
	}

	for _, q := range benchQueries {
		sql := strings.TrimSuffix(strings.TrimSpace(q.sql), ";")
		b.Run(q.name, func(b *testing.B) {
			sess := p.AcquireSession()
			defer sess.Release()
			b.ReportAllocs()
			b.SetBytes(int64(len(sql)))
			for b.Loop() {
				if _, err := sess.Parse(sql, "SelectStatement"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkGrammarLoad measures the one-time cost of parsing the vendored
// .gram files into a peg.Grammar (dsl_parser.go) plus reading the keyword
// lists — everything ParseString's sync.Once pays exactly once per
// process. grammar.Load itself has no memoization, so this can be run
// directly across b.N without any warmup exclusion.
func BenchmarkGrammarLoad(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, _, err := grammar.Load(); err != nil {
			b.Fatal(err)
		}
	}
}

// largeQuery repeats item as a comma-separated SELECT list until the query
// is about n bytes.
func largeQuery(n int, item string) string {
	var sb strings.Builder
	sb.WriteString("SELECT ")
	for sb.Len() < n-len(item)-2 {
		if sb.Len() > len("SELECT ") {
			sb.WriteString(", ")
		}
		sb.WriteString(item)
	}
	return sb.String()
}

var largeQueries = []struct {
	name string
	sql  string
}{
	{"Valid4KiB", largeQuery(4<<10, "a")},
	{"Valid16KiB", largeQuery(16<<10, "a")},
	{"Valid64KiB", largeQuery(64<<10, "a")},
	{"Failing64KiB", ") " + largeQuery(64<<10, "a")},
}

// BenchmarkParseStringLarge measures the memory a large (valid or failing)
// input costs through the pooled entry point, the case the packrat
// memo/slab budget (maxParseEntries) exists for.
func BenchmarkParseStringLarge(b *testing.B) {
	if _, err := ParseString("SELECT 1"); err != nil {
		b.Fatal(err)
	}
	for _, c := range largeQueries {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(c.sql)))
			for b.Loop() {
				_, _ = ParseString(c.sql)
			}
		})
	}
}

// BenchmarkPegParseLarge is BenchmarkParseStringLarge through a fresh
// evaluator every time (peg.Parser.Parse), so B/op includes all the
// buffer growth a cold parse of that input pays.
func BenchmarkPegParseLarge(b *testing.B) {
	p, err := loadPegParser()
	if err != nil {
		b.Fatal(err)
	}
	for _, c := range largeQueries {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(c.sql)))
			for b.Loop() {
				_, _ = p.Parse(c.sql, "SelectStatement")
			}
		})
	}
}
