package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pressly/goose/v3/database"
)

// duckdbStore is a hand-written goose database.Store for DuckDB.
//
// This doesn't use database.NewStoreFromQuerier (goose's usual, much
// shorter path for a new dialect) because of two independent
// incompatibilities discovered empirically against the real duckdb binary:
//
//  1. goose's closest built-in dialect, DialectSQLite3, generates
//     version-table DDL using "INTEGER PRIMARY KEY AUTOINCREMENT", which
//     DuckDB's parser rejects outright ("Parser Error: syntax error at or
//     near AUTOINCREMENT"). That alone rules out DialectSQLite3 and any
//     Querier reusing its DDL shape.
//  2. This POC's driver.Rows (rows.go, an earlier task, intentionally left
//     unmodified here) builds its Columns() list by iterating a Go map
//     (`for k := range rows[0]`), so the reported column order is not
//     guaranteed to match the SELECT list and can vary between calls.
//     goose's generic NewStoreFromQuerier-backed store scans multi-column
//     results (ListMigrations, GetMigration) positionally, trusting
//     Columns() order — which silently scans values into the wrong struct
//     field under this driver. Every multi-column query below is instead
//     scanned generically and its fields picked out by column NAME (see
//     ScanRowByName), which is immune to that reordering.
type duckdbStore struct {
	tableName string
}

var _ database.Store = (*duckdbStore)(nil)
var _ database.StoreExtender = (*duckdbStore)(nil)

func newDuckdbStore(tableName string) *duckdbStore {
	return &duckdbStore{tableName: tableName}
}

func (s *duckdbStore) Tablename() string { return s.tableName }

// CreateVersionTable creates the goose_db_version-equivalent bookkeeping
// table. There's no AUTOINCREMENT column (compare goose's sqlite3 dialect,
// which has a surrogate "id INTEGER PRIMARY KEY AUTOINCREMENT"): version_id
// is already the natural key, so no auto-incrementing key is needed.
func (s *duckdbStore) CreateVersionTable(ctx context.Context, db database.DBTxConn) error {
	q := fmt.Sprintf(`CREATE TABLE %s (
	version_id BIGINT NOT NULL,
	is_applied BOOLEAN NOT NULL,
	tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);`, s.tableName)
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("duckdb: create version table %q: %w", s.tableName, err)
	}
	return nil
}

func (s *duckdbStore) Insert(ctx context.Context, db database.DBTxConn, req database.InsertRequest) error {
	q := fmt.Sprintf(`INSERT INTO %s (version_id, is_applied) VALUES (?, ?);`, s.tableName)
	if _, err := db.ExecContext(ctx, q, req.Version, true); err != nil {
		return fmt.Errorf("duckdb: insert version %d: %w", req.Version, err)
	}
	return nil
}

func (s *duckdbStore) Delete(ctx context.Context, db database.DBTxConn, version int64) error {
	q := fmt.Sprintf(`DELETE FROM %s WHERE version_id = ?;`, s.tableName)
	if _, err := db.ExecContext(ctx, q, version); err != nil {
		return fmt.Errorf("duckdb: delete version %d: %w", version, err)
	}
	return nil
}

func (s *duckdbStore) GetMigration(
	ctx context.Context,
	db database.DBTxConn,
	version int64,
) (*database.GetMigrationResult, error) {
	q := fmt.Sprintf(`SELECT tstamp, is_applied FROM %s WHERE version_id = ? ORDER BY tstamp DESC LIMIT 1;`, s.tableName)
	rows, err := db.QueryContext(ctx, q, version)
	if err != nil {
		return nil, fmt.Errorf("duckdb: get migration %d: %w", version, err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %d", database.ErrVersionNotFound, version)
	}
	row, err := ScanRowByName(rows)
	if err != nil {
		return nil, err
	}

	applied, ok := row["is_applied"].(bool)
	if !ok {
		return nil, fmt.Errorf("duckdb: unexpected is_applied type %T", row["is_applied"])
	}
	ts, err := AsTimestamp(row["tstamp"])
	if err != nil {
		return nil, err
	}
	return &database.GetMigrationResult{Timestamp: ts, IsApplied: applied}, nil
}

func (s *duckdbStore) GetLatestVersion(ctx context.Context, db database.DBTxConn) (int64, error) {
	q := fmt.Sprintf(`SELECT MAX(version_id) FROM %s;`, s.tableName)
	var version sql.NullFloat64 // DuckDB's -jsonlines transport surfaces JSON numbers as float64.
	if err := db.QueryRowContext(ctx, q).Scan(&version); err != nil {
		return -1, fmt.Errorf("duckdb: get latest version: %w", err)
	}
	if !version.Valid {
		return -1, fmt.Errorf("duckdb: latest %w", database.ErrVersionNotFound)
	}
	return int64(version.Float64), nil
}

func (s *duckdbStore) ListMigrations(
	ctx context.Context,
	db database.DBTxConn,
) ([]*database.ListMigrationsResult, error) {
	q := fmt.Sprintf(`SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC;`, s.tableName)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("duckdb: list migrations: %w", err)
	}
	defer rows.Close()

	var out []*database.ListMigrationsResult
	for rows.Next() {
		row, err := ScanRowByName(rows)
		if err != nil {
			return nil, err
		}
		version, err := AsInt64(row["version_id"])
		if err != nil {
			return nil, err
		}
		applied, ok := row["is_applied"].(bool)
		if !ok {
			return nil, fmt.Errorf("duckdb: unexpected is_applied type %T", row["is_applied"])
		}
		out = append(out, &database.ListMigrationsResult{Version: version, IsApplied: applied})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// TableExists implements the optional database.StoreExtender method so
// goose can check for the version table directly. Without it, goose's
// fallback is to probe GetMigration(ctx, db, 0), which is one more
// multi-column, name-keyed scan than necessary during the common startup
// path — implementing this keeps that path on the fast, single-column
// EXISTS query instead. tableName is always the fixed internal
// "goose_db_version" constant, never external input, so direct string
// formatting here is safe.
func (s *duckdbStore) TableExists(ctx context.Context, db database.DBTxConn) (bool, error) {
	q := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '%s');`, s.tableName)
	var exists bool
	if err := db.QueryRowContext(ctx, q).Scan(&exists); err != nil {
		return false, fmt.Errorf("duckdb: check table exists: %w", err)
	}
	return exists, nil
}

// ScanRowByName scans the current row of rows generically and returns its
// values keyed by column name. This driver's Rows.Columns() order isn't
// guaranteed to match the query's SELECT list (see the duckdbStore doc
// comment above), so every multi-column caller in this file — and every
// caller in pkg/cqrs/duckdbquery — looks values up by name rather than by
// position.
func ScanRowByName(rows *sql.Rows) (map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	dest := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range dest {
		ptrs[i] = &dest[i]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return nil, fmt.Errorf("duckdb: scan row: %w", err)
	}
	out := make(map[string]any, len(cols))
	for i, c := range cols {
		out[c] = dest[i]
	}
	return out, nil
}

// AsInt64 converts a value read back from this driver into an int64.
// DuckDB's -jsonlines transport surfaces JSON numbers as float64.
func AsInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("duckdb: unexpected numeric type %T", v)
	}
}

// AsTimestamp converts a value read back from this driver into a time.Time.
// This driver has no read-side TIMESTAMP-to-time.Time conversion (see
// literal.go, which only handles the write side), so DuckDB's -jsonlines
// TIMESTAMP output arrives here as a plain string.
func AsTimestamp(v any) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case string:
		for _, layout := range []string{"2006-01-02 15:04:05.999999", "2006-01-02 15:04:05", time.RFC3339} {
			if ts, err := time.Parse(layout, t); err == nil {
				return ts, nil
			}
		}
		return time.Time{}, fmt.Errorf("duckdb: unparseable timestamp %q", t)
	default:
		return time.Time{}, fmt.Errorf("duckdb: unexpected timestamp type %T", v)
	}
}
