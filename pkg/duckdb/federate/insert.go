package federate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/driver"
)

// InsertStream appends r's rows to t's physical lake table in catalog: one
// INSERT ... SELECT reading r over quack's SEND_DATA stream, so DuckLake
// writes the files itself (VARIANT shredding, its partitioning and sort
// order, file statistics) and commits them as one snapshot. Each column is
// cast to its lake type the way the federated views cast the delta (JSON
// text to VARIANT, list text to STRUCT lists). r must carry DeltaColumns(t).
// conn must be a quack-transport connection. It returns the rows inserted.
func InsertStream(ctx context.Context, conn *sql.Conn, catalog string, t Table, r BatchReader) (int64, error) {
	defer r.Close()

	phys, ok := Physical[t]
	if !ok {
		return 0, fmt.Errorf("federate: no physical table for %q", t)
	}
	if err := sameColumns(r.Schema(), DeltaColumns(t)); err != nil {
		return 0, fmt.Errorf("federate: inserting %s: %w", t, err)
	}
	// Read straight from the stream: a MATERIALIZED CTE (as queries use)
	// would buffer the whole partition first.
	st, scan, err := deltaScan(t, r, 0)
	if err != nil {
		return 0, err
	}
	cols := make([]string, len(phys.Columns))
	sel := make([]string, len(phys.Columns))
	for i, c := range phys.Columns {
		cols[i] = quoteIdent(c.Name)
		sel[i] = deltaSelect(c)
	}
	target := LakeSource{Catalog: catalog}.table(phys.Name)
	sqlText := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM (%s) AS %s",
		target, strings.Join(cols, ", "), strings.Join(sel, ", "), scan, quoteIdent(deltaCTEName(t)))

	rows, err := conn.QueryContext(driver.WithQuackStreams(ctx, st), sqlText)
	if err != nil {
		return 0, fmt.Errorf("federate: inserting %s: %w", t, err)
	}
	defer rows.Close()
	var n int64
	if rows.Next() {
		if err := rows.Scan(&n); err != nil {
			return 0, fmt.Errorf("federate: inserting %s: reading count: %w", t, err)
		}
	}
	if err := rows.Err(); err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("federate: inserting %s: %w", t, err)
	}
	return n, nil
}
