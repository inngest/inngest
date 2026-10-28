package insights

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// nestedSubqueryShapes builds depth-deep FROM-clause subquery nestings in
// each shape deriveTable handles: a plain SELECT, a set operation whose
// leftmost operand holds the next level, and a subquery carrying its own
// WITH clause.
func nestedSubqueryShapes(depth int) map[string]string {
	plain, union, with := "SELECT run_id FROM runs", "SELECT run_id FROM runs", "SELECT run_id FROM runs"
	for i := range depth {
		plain = fmt.Sprintf("SELECT * FROM (%s) s%d", plain, i)
		union = fmt.Sprintf("SELECT * FROM (%s) s%d UNION ALL SELECT run_id FROM runs", union, i)
		with = fmt.Sprintf("WITH c%d AS (SELECT run_id FROM runs) SELECT * FROM (%s) s%d", i, with, i)
	}
	return map[string]string{"plain": plain, "union": union, "with": with}
}

// TestTranspileDeepNestingIsFast guards against deriveTable re-deriving a
// subquery's FROM clause after validating it, which made Transpile
// exponential in nesting depth (depth 16 took ~0.7s; depth 30 would take
// hours).
func TestTranspileDeepNestingIsFast(t *testing.T) {
	for shape, q := range nestedSubqueryShapes(30) {
		start := time.Now()
		_, err := Transpile(q, uuid.New(), uuid.New())
		require.NoError(t, err, shape)
		require.Less(t, time.Since(start), 200*time.Millisecond, shape)
	}
}

// TestTranspileAcceptsWithInsideSubquery: a FROM-clause subquery's own WITH
// clause used to fail as "unknown table", because deriveTable re-resolved
// the subquery's FROM clause without that WITH's CTEs.
func TestTranspileAcceptsWithInsideSubquery(t *testing.T) {
	for _, q := range []string{
		"SELECT * FROM (WITH c AS (SELECT run_id FROM runs) SELECT * FROM c) t",
		"SELECT * FROM (WITH c AS (SELECT run_id FROM runs) SELECT run_id FROM c UNION ALL SELECT run_id FROM c) t",
	} {
		tr, err := Transpile(q, uuid.New(), uuid.New())
		require.NoError(t, err, q)
		require.Equal(t, HintRunID, tr.ColumnPathHints[0][0].Hint, q)
	}
}

func BenchmarkTranspileNestedSubqueries(b *testing.B) {
	for _, depth := range []int{4, 16, 64} {
		for shape, q := range nestedSubqueryShapes(depth) {
			b.Run(fmt.Sprintf("%s/depth=%d", shape, depth), func(b *testing.B) {
				for b.Loop() {
					if _, err := Transpile(q, uuid.New(), uuid.New()); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
