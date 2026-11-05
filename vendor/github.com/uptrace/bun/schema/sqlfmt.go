package schema

import (
	"log/slog"
	"strings"

	"github.com/uptrace/bun/internal"
)

// QueryAppender is implemented by types that can append themselves to a SQL query.
type QueryAppender interface {
	AppendQuery(gen QueryGen, b []byte) ([]byte, error)
}

// ColumnsAppender is implemented by types that can append column definitions to a SQL query.
type ColumnsAppender interface {
	AppendColumns(gen QueryGen, b []byte) ([]byte, error)
}

//------------------------------------------------------------------------------

// Safe represents a safe SQL query.
type Safe string

var _ QueryAppender = (*Safe)(nil)

func (s Safe) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	return append(b, s...), nil
}

//------------------------------------------------------------------------------

// Name represents a single SQL name, for example, a column name.
type Name string

var _ QueryAppender = (*Name)(nil)

func (s Name) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	return gen.AppendName(b, string(s)), nil
}

//------------------------------------------------------------------------------

// Ident represents a SQL identifier, for example,
// a fully qualified column name such as `table_name.col_name`.
type Ident string

var _ QueryAppender = (*Ident)(nil)

func (s Ident) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	return gen.AppendIdent(b, string(s)), nil
}

//------------------------------------------------------------------------------

// QueryWithArgs is a query string paired with its arguments.
// NOTE: It should not be modified after creation.
type QueryWithArgs struct {
	Query string
	Args  []any
}

var _ QueryAppender = QueryWithArgs{}

func SafeQuery(query string, args []any) QueryWithArgs {
	if args == nil {
		args = make([]any, 0)
	} else if len(query) > 0 && strings.IndexByte(query, '?') == -1 {
		internal.Warn.Printf("query %q has %v args, but no placeholders", query, args)
	}
	return QueryWithArgs{
		Query: query,
		Args:  args,
	}
}

func UnsafeIdent(ident string) QueryWithArgs {
	return QueryWithArgs{Query: ident}
}

func (q QueryWithArgs) IsZero() bool {
	return q.Query == "" && q.Args == nil
}

func (q QueryWithArgs) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	if q.Args == nil {
		return gen.AppendIdent(b, q.Query), nil
	}
	return gen.AppendQuery(b, q.Query, q.Args...), nil
}

//------------------------------------------------------------------------------

// Order represents a SQL sort direction.
type Order string

const (
	OrderNone           Order = ""
	OrderAsc            Order = "ASC"
	OrderAscNullsFirst  Order = "ASC NULLS FIRST"
	OrderAscNullsLast   Order = "ASC NULLS LAST"
	OrderDesc           Order = "DESC"
	OrderDescNullsFirst Order = "DESC NULLS FIRST"
	OrderDescNullsLast  Order = "DESC NULLS LAST"
)

func (s Order) AppendQuery(gen QueryGen, b []byte) ([]byte, error) {
	return AppendOrder(b, s), nil
}

func AppendOrder(b []byte, sortDir Order) []byte {
	switch sortDir {
	case OrderAsc, OrderDesc,
		OrderAscNullsFirst, OrderAscNullsLast,
		OrderDescNullsFirst, OrderDescNullsLast:
		return append(b, sortDir...)
	case OrderNone:
		return b
	default:
		slog.Error("unsupported sort direction", slog.String("sort_dir", string(sortDir)))
		return b
	}
}

//------------------------------------------------------------------------------

// QueryWithSep is a QueryWithArgs with an additional separator.
type QueryWithSep struct {
	QueryWithArgs
	Sep string
}

func SafeQueryWithSep(query string, args []any, sep string) QueryWithSep {
	return QueryWithSep{
		QueryWithArgs: SafeQuery(query, args),
		Sep:           sep,
	}
}
