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
// shorter path for a new dialect) because goose's closest built-in dialect,
// DialectSQLite3, generates version-table DDL using "INTEGER PRIMARY KEY
// AUTOINCREMENT", which DuckDB's parser rejects outright ("Parser Error:
// syntax error at or near AUTOINCREMENT", confirmed against the real duckdb
// binary). That rules out DialectSQLite3 and any Querier reusing its DDL
// shape, so every query below is instead hand-written and scanned
// positionally in the query's own column order (see pkg/db/duckdb/rows.go
// and quack_protocol.go, which guarantee Columns() matches that order).
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
	var rawTstamp any
	var applied bool
	if err := rows.Scan(&rawTstamp, &applied); err != nil {
		return nil, fmt.Errorf("duckdb: scan migration row: %w", err)
	}
	ts, err := AsTimestamp(rawTstamp)
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
		var rawVersion any
		var applied bool
		if err := rows.Scan(&rawVersion, &applied); err != nil {
			return nil, fmt.Errorf("duckdb: scan migration list row: %w", err)
		}
		version, err := AsInt64(rawVersion)
		if err != nil {
			return nil, err
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
// values keyed by column name, for callers that want name-keyed access to a
// query result (e.g. one whose column list isn't fixed at the call site)
// instead of writing out positional destinations.
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
