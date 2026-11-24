package sqlschema

import (
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
)

// MigratorDialect is a Dialect that can create a Migrator for executing schema changes.
type MigratorDialect interface {
	schema.Dialect
	NewMigrator(db *bun.DB, schemaName string) Migrator
}

// Migrator renders schema-change operations as SQL.
type Migrator interface {
	AppendSQL(b []byte, operation any) ([]byte, error)
}

// migrator is a dialect-agnostic wrapper for sqlschema.MigratorDialect.
type migrator struct {
	Migrator
}

func NewMigrator(db *bun.DB, schemaName string) (Migrator, error) {
	md, ok := db.Dialect().(MigratorDialect)
	if !ok {
		return nil, fmt.Errorf("%q dialect does not implement sqlschema.Migrator", db.Dialect().Name())
	}
	return &migrator{
		Migrator: md.NewMigrator(db, schemaName),
	}, nil
}

// BaseMigrator can be embeded by dialect's Migrator implementations to re-use some of the existing bun queries.
type BaseMigrator struct {
	db *bun.DB
}

func NewBaseMigrator(db *bun.DB) *BaseMigrator {
	return &BaseMigrator{db: db}
}

func (m *BaseMigrator) AppendCreateTable(b []byte, model any) ([]byte, error) {
	return m.db.NewCreateTable().Model(model).AppendQuery(m.db.QueryGen(), b)
}

func (m *BaseMigrator) AppendDropTable(b []byte, schemaName, tableName string) ([]byte, error) {
	return m.db.NewDropTable().TableExpr("?.?", bun.Ident(schemaName), bun.Ident(tableName)).AppendQuery(m.db.QueryGen(), b)
}
