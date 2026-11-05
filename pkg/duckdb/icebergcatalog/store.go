package icebergcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// store keeps catalog state in the Iceberg JDBC catalog layout
// (`iceberg_tables` + `iceberg_namespace_properties`, same as Java's
// JdbcCatalog and iceberg-go's catalog/sql), so the backing db stays
// readable by those. Every query is a fixed, parameterized statement; `$N`
// placeholders work on both pgx and modernc sqlite.
//
// NOTE: we always write the V1 layout (w/ `iceberg_type`). Legacy V0 tables
// (no `iceberg_type` column) aren't migrated.
type store struct {
	db   *sql.DB
	name string
}

var (
	errNoSuchTable            = errors.New("no such table")
	errNoSuchNamespace        = errors.New("no such namespace")
	errTableAlreadyExists     = errors.New("table already exists")
	errNamespaceAlreadyExists = errors.New("namespace already exists")
	errNamespaceNotEmpty      = errors.New("namespace not empty")
)

const (
	createTablesDDL = `CREATE TABLE IF NOT EXISTS iceberg_tables (
	catalog_name VARCHAR(255) NOT NULL,
	table_namespace VARCHAR(255) NOT NULL,
	table_name VARCHAR(255) NOT NULL,
	iceberg_type VARCHAR(5),
	metadata_location VARCHAR(1000),
	previous_metadata_location VARCHAR(1000),
	PRIMARY KEY (catalog_name, table_namespace, table_name)
)`
	createNamespacePropsDDL = `CREATE TABLE IF NOT EXISTS iceberg_namespace_properties (
	catalog_name VARCHAR(255) NOT NULL,
	namespace VARCHAR(255) NOT NULL,
	property_key VARCHAR(255) NOT NULL,
	property_value VARCHAR(1000),
	PRIMARY KEY (catalog_name, namespace, property_key)
)`
	// tableTypeFilter matches table rows (views share iceberg_tables).
	tableTypeFilter = `(iceberg_type = 'TABLE' OR iceberg_type IS NULL)`
)

// namespaceExistsProperty is the marker row JDBC catalogs write so a
// namespace w/o properties still exists. It's reserved: clients can't set,
// remove or see it.
const namespaceExistsProperty = "exists"

func newStore(ctx context.Context, db *sql.DB, name string) (*store, error) {
	for _, ddl := range []string{createTablesDDL, createNamespacePropsDDL} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return nil, fmt.Errorf("creating catalog tables: %w", err)
		}
	}
	return &store{db: db, name: name}, nil
}

// encodedNamespacePrefix + namespaceKey match iceberg-go's catalog/sql
// encoding: levels joined w/ `.` like Java, unless a level itself contains
// a `.` (then JSON, so it round-trips).
const encodedNamespacePrefix = "__iceberg_namespace_v1__:"

func namespaceKey(ns []string) string {
	legacy := strings.Join(ns, ".")
	if !strings.Contains(legacy, encodedNamespacePrefix) && !slices.ContainsFunc(ns, func(p string) bool { return strings.Contains(p, ".") }) {
		return legacy
	}
	return encodedNamespacePrefix + string(mustMarshal(ns))
}

func namespaceFromKey(key string) []string {
	if rest, ok := strings.CutPrefix(key, encodedNamespacePrefix); ok {
		var ns []string
		if err := json.Unmarshal([]byte(rest), &ns); err == nil && len(ns) > 0 && namespaceKey(ns) == key {
			return ns
		}
	}
	return strings.Split(key, ".")
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *store) namespaceExists(ctx context.Context, q querier, ns []string) (bool, error) {
	var one int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM iceberg_namespace_properties WHERE catalog_name = $1 AND namespace = $2
UNION ALL SELECT 1 FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2 LIMIT 1`, s.name, namespaceKey(ns)).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *store) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func checkReserved(keys ...string) error {
	if slices.Contains(keys, namespaceExistsProperty) {
		return fmt.Errorf("%w: namespace property %q is reserved", errBadRequest, namespaceExistsProperty)
	}
	return nil
}

func (s *store) createNamespace(ctx context.Context, ns []string, props map[string]string) error {
	for k := range props {
		if err := checkReserved(k); err != nil {
			return err
		}
	}
	rows := map[string]string{namespaceExistsProperty: "true"}
	for k, v := range props {
		rows[k] = v
	}
	return s.tx(ctx, func(tx *sql.Tx) error {
		if ok, err := s.namespaceExists(ctx, tx, ns); err != nil || ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNamespaceAlreadyExists, strings.Join(ns, "."))
			}
			return err
		}
		for k, v := range rows {
			if _, err := tx.ExecContext(ctx, `INSERT INTO iceberg_namespace_properties (catalog_name, namespace, property_key, property_value) VALUES ($1, $2, $3, $4)`,
				s.name, namespaceKey(ns), k, v); err != nil {
				// Most likely a concurrent create won the PK.
				return fmt.Errorf("%w: %s (%v)", errNamespaceAlreadyExists, strings.Join(ns, "."), err)
			}
		}
		return nil
	})
}

func (s *store) loadNamespace(ctx context.Context, ns []string) (map[string]string, error) {
	if ok, err := s.namespaceExists(ctx, s.db, ns); err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(ns, "."))
		}
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT property_key, property_value FROM iceberg_namespace_properties WHERE catalog_name = $1 AND namespace = $2`, s.name, namespaceKey(ns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	props := map[string]string{}
	for rows.Next() {
		var k string
		var v sql.NullString
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		if k != namespaceExistsProperty {
			props[k] = v.String
		}
	}
	return props, rows.Err()
}

func (s *store) dropNamespace(ctx context.Context, ns []string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if ok, err := s.namespaceExists(ctx, tx, ns); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(ns, "."))
			}
			return err
		}
		// Single statement so a table created concurrently either lands
		// first (and we refuse) or after (and fails its namespace check).
		res, err := tx.ExecContext(ctx, `DELETE FROM iceberg_namespace_properties WHERE catalog_name = $1 AND namespace = $2
AND NOT EXISTS (SELECT 1 FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2)`, s.name, namespaceKey(ns))
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNamespaceNotEmpty, strings.Join(ns, "."))
			}
			return err
		}
		return nil
	})
}

type propertiesSummary struct {
	Removed []string `json:"removed"`
	Updated []string `json:"updated"`
	Missing []string `json:"missing"`
}

func (s *store) updateNamespace(ctx context.Context, ns []string, removals []string, updates map[string]string) (propertiesSummary, error) {
	sum := propertiesSummary{Removed: []string{}, Updated: []string{}, Missing: []string{}}
	if err := checkReserved(removals...); err != nil {
		return sum, err
	}
	for k := range updates {
		if err := checkReserved(k); err != nil {
			return sum, err
		}
		if slices.Contains(removals, k) {
			return sum, fmt.Errorf("%w: property %q is both updated and removed", errBadRequest, k)
		}
	}
	err := s.tx(ctx, func(tx *sql.Tx) error {
		if ok, err := s.namespaceExists(ctx, tx, ns); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(ns, "."))
			}
			return err
		}
		key := namespaceKey(ns)
		for _, k := range removals {
			res, err := tx.ExecContext(ctx, `DELETE FROM iceberg_namespace_properties WHERE catalog_name = $1 AND namespace = $2 AND property_key = $3`, s.name, key, k)
			if err != nil {
				return err
			}
			if n, _ := res.RowsAffected(); n > 0 {
				sum.Removed = append(sum.Removed, k)
			} else {
				sum.Missing = append(sum.Missing, k)
			}
		}
		for k, v := range updates {
			if _, err := tx.ExecContext(ctx, `INSERT INTO iceberg_namespace_properties (catalog_name, namespace, property_key, property_value) VALUES ($1, $2, $3, $4)
ON CONFLICT (catalog_name, namespace, property_key) DO UPDATE SET property_value = excluded.property_value`, s.name, key, k, v); err != nil {
				return err
			}
			sum.Updated = append(sum.Updated, k)
		}
		return nil
	})
	return sum, err
}

// listNamespaces returns parent's direct children (top-level namespaces for
// an empty parent), like Java's JdbcCatalog.
func (s *store) listNamespaces(ctx context.Context, parent []string) ([][]string, error) {
	if len(parent) > 0 {
		if ok, err := s.namespaceExists(ctx, s.db, parent); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(parent, "."))
			}
			return nil, err
		}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT namespace FROM iceberg_namespace_properties WHERE catalog_name = $1
UNION SELECT table_namespace FROM iceberg_tables WHERE catalog_name = $1`, s.name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seen := map[string]bool{}
	out := [][]string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		ns := namespaceFromKey(key)
		if len(ns) <= len(parent) || !slices.Equal(ns[:len(parent)], parent) {
			continue
		}
		child := ns[:len(parent)+1]
		if k := namespaceKey(child); !seen[k] {
			seen[k] = true
			out = append(out, child)
		}
	}
	return out, rows.Err()
}

func (s *store) listTables(ctx context.Context, ns []string) ([]identifier, error) {
	if ok, err := s.namespaceExists(ctx, s.db, ns); err != nil || !ok {
		if err == nil {
			err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(ns, "."))
		}
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT table_name FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2 AND `+tableTypeFilter+` ORDER BY table_name`, s.name, namespaceKey(ns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []identifier{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, identifier{Namespace: ns, Name: name})
	}
	return out, rows.Err()
}

// tableLocation is a table's current metadata location.
func (s *store) tableLocation(ctx context.Context, id identifier) (string, error) {
	var loc sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT metadata_location FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2 AND table_name = $3 AND `+tableTypeFilter,
		s.name, namespaceKey(id.Namespace), id.Name).Scan(&loc)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !loc.Valid) {
		return "", fmt.Errorf("%w: %s", errNoSuchTable, id)
	}
	return loc.String, err
}

// createTable registers a new table at metadataLoc.
func (s *store) createTable(ctx context.Context, id identifier, metadataLoc string) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if ok, err := s.namespaceExists(ctx, tx, id.Namespace); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(id.Namespace, "."))
			}
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO iceberg_tables (catalog_name, table_namespace, table_name, iceberg_type, metadata_location) VALUES ($1, $2, $3, 'TABLE', $4)`,
			s.name, namespaceKey(id.Namespace), id.Name, metadataLoc); err != nil {
			// The PK is the only constraint, so this is (almost certainly) a
			// table or view w/ the same name.
			return fmt.Errorf("%w: %s (%v)", errTableAlreadyExists, id, err)
		}
		return nil
	})
}

// swapTable is the optimistic commit: it moves a table's pointer from prev
// to next only if it's still at prev.
func (s *store) swapTable(ctx context.Context, id identifier, prev, next string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE iceberg_tables SET metadata_location = $1, previous_metadata_location = $2, iceberg_type = 'TABLE'
WHERE catalog_name = $3 AND table_namespace = $4 AND table_name = $5 AND metadata_location = $2 AND `+tableTypeFilter,
		next, prev, s.name, namespaceKey(id.Namespace), id.Name)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err == nil {
			err = fmt.Errorf("%w: %s metadata location moved underneath us", errCommitFailed, id)
		}
		return err
	}
	return nil
}

func (s *store) dropTable(ctx context.Context, id identifier) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2 AND table_name = $3 AND `+tableTypeFilter,
		s.name, namespaceKey(id.Namespace), id.Name)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err == nil {
			err = fmt.Errorf("%w: %s", errNoSuchTable, id)
		}
		return err
	}
	return nil
}

func (s *store) renameTable(ctx context.Context, from, to identifier) error {
	return s.tx(ctx, func(tx *sql.Tx) error {
		if ok, err := s.namespaceExists(ctx, tx, to.Namespace); err != nil || !ok {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchNamespace, strings.Join(to.Namespace, "."))
			}
			return err
		}
		var one int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM iceberg_tables WHERE catalog_name = $1 AND table_namespace = $2 AND table_name = $3`,
			s.name, namespaceKey(to.Namespace), to.Name).Scan(&one)
		if err == nil {
			return fmt.Errorf("%w: %s", errTableAlreadyExists, to)
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE iceberg_tables SET table_namespace = $1, table_name = $2
WHERE catalog_name = $3 AND table_namespace = $4 AND table_name = $5 AND `+tableTypeFilter,
			namespaceKey(to.Namespace), to.Name, s.name, namespaceKey(from.Namespace), from.Name)
		if err != nil {
			return fmt.Errorf("%w: %s (%v)", errTableAlreadyExists, to, err)
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			if err == nil {
				err = fmt.Errorf("%w: %s", errNoSuchTable, from)
			}
			return err
		}
		return nil
	})
}
