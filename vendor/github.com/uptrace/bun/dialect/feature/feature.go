// Package feature defines flags that represent optional SQL dialect capabilities.
// Each dialect (PostgreSQL, MySQL, SQLite, MSSQL) declares which features it supports
// by combining flags with the bitwise OR operator.
package feature

import (
	"fmt"
	"strconv"

	"github.com/uptrace/bun/internal"
)

// Feature is a bit flag representing an optional SQL dialect capability.
type Feature = internal.Flag

const (
	// Common query features.

	// CTE enables Common Table Expressions (WITH ... AS ...) syntax.
	CTE Feature = 1 << iota
	// WithValues enables WITH ... (VALUES ...) syntax for inline value lists.
	WithValues
	// Returning enables the RETURNING clause to return rows affected by DML statements.
	Returning
	// Output enables the OUTPUT clause, the MSSQL equivalent of RETURNING.
	Output
	// DefaultPlaceholder enables the DEFAULT keyword as a placeholder in INSERT value lists.
	DefaultPlaceholder
	// DoubleColonCast enables PostgreSQL-style :: type cast syntax.
	DoubleColonCast
	// ValuesRow enables VALUES ROW(...) syntax.
	ValuesRow
	// CompositeIn enables WHERE (A, B) IN ((N, NN), (N, NN), ...) composite comparison syntax.
	CompositeIn

	// SELECT features.

	// OffsetFetch enables OFFSET ... FETCH NEXT syntax (MSSQL).
	OffsetFetch
	// SelectExists enables EXISTS subquery expressions.
	SelectExists

	// INSERT features.

	// InsertReturning enables INSERT ... RETURNING syntax.
	InsertReturning
	// InsertTableAlias enables table alias support in INSERT statements.
	InsertTableAlias
	// InsertOnConflict enables INSERT ... ON CONFLICT syntax (PostgreSQL, SQLite).
	InsertOnConflict
	// InsertOnDuplicateKey enables INSERT ... ON DUPLICATE KEY syntax (MySQL).
	InsertOnDuplicateKey
	// InsertIgnore enables INSERT IGNORE syntax to silently skip conflicting rows (MySQL).
	InsertIgnore

	// UPDATE features.

	// UpdateFromTable enables UPDATE ... FROM ... syntax for joining tables in updates.
	UpdateFromTable
	// UpdateMultiTable enables multi-table UPDATE syntax (MySQL).
	UpdateMultiTable
	// UpdateTableAlias enables table alias support in UPDATE statements.
	UpdateTableAlias
	// UpdateOrderLimit enables UPDATE ... ORDER BY ... LIMIT syntax.
	UpdateOrderLimit

	// DELETE features.

	// DeleteReturning enables DELETE ... RETURNING syntax.
	DeleteReturning
	// DeleteTableAlias enables table alias support in DELETE statements.
	DeleteTableAlias
	// DeleteOrderLimit enables DELETE ... ORDER BY ... LIMIT syntax.
	DeleteOrderLimit

	// MERGE features.

	// Merge enables MERGE ... USING ... ON ... WHEN syntax for upsert operations.
	Merge
	// MergeReturning enables MERGE ... RETURNING syntax.
	MergeReturning

	// Table DDL features.

	// TableCascade enables CASCADE support for DROP TABLE and related operations.
	TableCascade
	// TableIdentity enables table-level IDENTITY property (MSSQL).
	TableIdentity
	// TableTruncate enables TRUNCATE TABLE support.
	TableTruncate
	// TableNotExists enables IF NOT EXISTS / IF EXISTS syntax for CREATE TABLE and DROP TABLE.
	TableNotExists
	// AlterColumnExists enables ADD/DROP COLUMN IF NOT EXISTS / IF EXISTS syntax.
	AlterColumnExists
	// CreateIndexIfNotExists enables CREATE INDEX IF NOT EXISTS syntax.
	CreateIndexIfNotExists

	// Column definition features.

	// AutoIncrement enables AUTO_INCREMENT syntax for auto-generated columns (MySQL).
	AutoIncrement
	// Identity enables IDENTITY column syntax for auto-generated columns (MSSQL).
	Identity
	// GeneratedIdentity enables GENERATED ALWAYS AS IDENTITY syntax (PostgreSQL).
	GeneratedIdentity

	// Dialect-specific features.

	// FKDefaultOnAction indicates that FK ON UPDATE/ON DELETE has the default value NO ACTION.
	FKDefaultOnAction
	// MSSavepoint enables Microsoft SQL Server savepoint support.
	MSSavepoint
)

// NotSupportError is returned when an operation requires a feature
// that the current dialect does not support.
type NotSupportError struct {
	Flag Feature
}

func (err *NotSupportError) Error() string {
	name, ok := flag2str[err.Flag]
	if !ok {
		name = strconv.FormatInt(int64(err.Flag), 10)
	}
	return fmt.Sprintf("bun: feature %s is not supported by current dialect", name)
}

// NewNotSupportError returns a NotSupportError for the given feature flag.
func NewNotSupportError(flag Feature) *NotSupportError {
	return &NotSupportError{Flag: flag}
}

var flag2str = map[Feature]string{
	// Common query features.
	CTE:                "CTE",
	WithValues:         "WithValues",
	Returning:          "Returning",
	Output:             "Output",
	DefaultPlaceholder: "DefaultPlaceholder",
	DoubleColonCast:    "DoubleColonCast",
	ValuesRow:          "ValuesRow",
	CompositeIn:        "CompositeIn",

	// SELECT features.
	OffsetFetch:  "OffsetFetch",
	SelectExists: "SelectExists",

	// INSERT features.
	InsertReturning:      "InsertReturning",
	InsertTableAlias:     "InsertTableAlias",
	InsertOnConflict:     "InsertOnConflict",
	InsertOnDuplicateKey: "InsertOnDuplicateKey",
	InsertIgnore:         "InsertIgnore",

	// UPDATE features.
	UpdateFromTable:  "UpdateFromTable",
	UpdateMultiTable: "UpdateMultiTable",
	UpdateTableAlias: "UpdateTableAlias",
	UpdateOrderLimit: "UpdateOrderLimit",

	// DELETE features.
	DeleteReturning:  "DeleteReturning",
	DeleteTableAlias: "DeleteTableAlias",
	DeleteOrderLimit: "DeleteOrderLimit",

	// MERGE features.
	Merge:          "Merge",
	MergeReturning: "MergeReturning",

	// Table DDL features.
	TableCascade:           "TableCascade",
	TableIdentity:          "TableIdentity",
	TableTruncate:          "TableTruncate",
	TableNotExists:         "TableNotExists",
	AlterColumnExists:      "AlterColumnExists",
	CreateIndexIfNotExists: "CreateIndexIfNotExists",

	// Column definition features.
	AutoIncrement:     "AutoIncrement",
	Identity:          "Identity",
	GeneratedIdentity: "GeneratedIdentity",

	// Dialect-specific features.
	FKDefaultOnAction: "FKDefaultOnAction",
	MSSavepoint:       "MSSavepoint",
}
