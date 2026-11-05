package schema

import (
	"context"
	"database/sql"
	"reflect"
)

type Model interface {
	ScanRows(ctx context.Context, rows *sql.Rows) (int, error)
	Value() any
}

type Query interface {
	QueryAppender
	Operation() string
	GetModel() Model
	GetTableName() string
}

//------------------------------------------------------------------------------

type BeforeAppendModelHook interface {
	BeforeAppendModel(ctx context.Context, query Query) error
}

var beforeAppendModelHookType = reflect.TypeFor[BeforeAppendModelHook]()

//------------------------------------------------------------------------------

type BeforeScanRowHook interface {
	BeforeScanRow(context.Context) error
}

var beforeScanRowHookType = reflect.TypeFor[BeforeScanRowHook]()

//------------------------------------------------------------------------------

type AfterScanRowHook interface {
	AfterScanRow(context.Context) error
}

var afterScanRowHookType = reflect.TypeFor[AfterScanRowHook]()
