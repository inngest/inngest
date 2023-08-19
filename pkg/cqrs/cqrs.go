package cqrs

import (
	"context"
	"database/sql"
	"time"
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
	ExecutionLoader

	AppManager
	FunctionManager
	FunctionRunManager
	EventManager
	HistoryManager

	// Scoped allows creating a new manager using a transaction.
	WithTx(ctx context.Context) (TxManager, error)
}

type TxManager interface {
	Manager

	Commit(context.Context) error
	Rollback(context.Context) error
}

type Timebound struct {
	// After is the lower bound to load data from, exclusive.
	After *time.Time `json:"after,omitempty"`
	// Before is the upper bound to load data from, inclusive
	Before *time.Time `json:"before,omitempty"`
}
