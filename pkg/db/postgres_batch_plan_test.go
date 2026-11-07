package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	pgsqlc "github.com/inngest/inngest/pkg/db/postgres/sqlc"
	"github.com/stretchr/testify/require"
)

type postgresExplainDBTX struct {
	pgsqlc.DBTX
	query string
	plan  postgresPlan
}

func (db *postgresExplainDBTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db.query = query

	var raw []byte
	if err := db.DBTX.QueryRowContext(ctx, "EXPLAIN (FORMAT JSON) "+query, args...).Scan(&raw); err != nil {
		return nil, fmt.Errorf("explain generated query: %w", err)
	}

	var plans []postgresPlan
	if err := json.Unmarshal(raw, &plans); err != nil {
		return nil, fmt.Errorf("decode explain plan: %w", err)
	}
	if len(plans) != 1 {
		return nil, fmt.Errorf("expected one explain plan, got %d", len(plans))
	}
	db.plan = plans[0]

	return db.DBTX.QueryContext(ctx, query, args...)
}

type postgresPlan struct {
	Plan postgresPlanNode `json:"Plan"`
}

type postgresPlanNode struct {
	IndexCond string             `json:"Index Cond"`
	Plans     []postgresPlanNode `json:"Plans"`
}

func (p postgresPlanNode) hasIDIndexCondition() bool {
	if strings.Contains(strings.ToLower(p.IndexCond), "id") {
		return true
	}
	for _, child := range p.Plans {
		if child.hasIDIndexCondition() {
			return true
		}
	}
	return false
}

func TestPostgresBatchQueriesUseIDIndexes(t *testing.T) {
	if os.Getenv(envTestDatabase) != "postgres" {
		t.Skip("requires TEST_DATABASE=postgres")
	}

	adapter, cleanup := newTestAdapter(t)
	defer cleanup()

	conn := adapter.Conn()
	conn.SetMaxOpenConns(1)
	_, err := conn.ExecContext(t.Context(), "SET enable_seqscan = off")
	require.NoError(t, err)

	for _, test := range []struct {
		name string
		call func(context.Context, *pgsqlc.Queries) error
	}{
		{
			name: "apps",
			call: func(ctx context.Context, q *pgsqlc.Queries) error {
				_, err := q.GetAppsByIDs(ctx, []string{uuid.NewString()})
				return err
			},
		},
		{
			name: "functions",
			call: func(ctx context.Context, q *pgsqlc.Queries) error {
				_, err := q.GetFunctionsByIDs(ctx, []string{uuid.NewString()})
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			traced := &postgresExplainDBTX{DBTX: conn}
			err := test.call(t.Context(), pgsqlc.New(traced))
			require.NoError(t, err)
			require.NotEmpty(t, traced.query)
			require.True(t, traced.plan.Plan.hasIDIndexCondition(),
				"generated query %q should produce an index condition; plan: %+v",
				traced.query, traced.plan)
		})
	}
}
