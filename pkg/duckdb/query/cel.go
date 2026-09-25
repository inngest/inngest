package query

import (
	"context"
	"fmt"

	sq "github.com/doug-martin/goqu/v9"
)

// CELEventTableFilters, CELEventFilters, CELOutputFilters, and
// RenderWhereSQL are plugin points, not direct calls into
// pkg/duckdb/insights: this package (the general duckdb-backed CQRS read
// path) must build and function without the insights SQL execution feature
// present. A build that also includes pkg/duckdb/insights overrides all
// four vars (via an init() in that package) with the real CEL-to-SQL
// implementation.
//
// Every caller here invokes these unconditionally, whether or not the
// caller's own CEL expression list is empty -- the common case for a plain
// (non-CEL) query -- so each stub still has to handle "nothing to convert"
// correctly instead of erroring outright: only a genuinely non-empty input,
// which really does need the real insights implementation, is unimplemented.
var (
	CELEventTableFilters = func(ctx context.Context, exprs []string) ([]sq.Expression, error) {
		if len(exprs) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("duckdbquery: CEL event search unimplemented")
	}
	CELEventFilters = func(ctx context.Context, exprs []string) ([]sq.Expression, error) {
		if len(exprs) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("duckdbquery: CEL event filter unimplemented")
	}
	CELOutputFilters = func(ctx context.Context, exprs []string) ([]sq.Expression, error) {
		if len(exprs) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("duckdbquery: CEL output filter unimplemented")
	}
	RenderWhereSQL = func(filters []sq.Expression) (string, []any, error) {
		if len(filters) == 0 {
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("duckdbquery: CEL filter rendering unimplemented")
	}
)
