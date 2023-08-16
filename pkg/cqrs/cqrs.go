package cqrs

import (
	"context"
	"database/sql"
)

// DBWriter can be a *sql.DB or an *sql.TX, and is needed to allow
// transactions with stateless CQRS managers.
type DBWriter interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type Manager interface {
	AppManager
	FunctionManager
	ExecutionLoader
	EventWriter

	// Scoped allows creating a new manager using a transaction.
	WithTx(ctx context.Context) (TxManager, error)
}

type TxManager interface {
	Manager

	Commit(context.Context) error
	Rollback(context.Context) error
}
