package dashboards_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/duckdb/dashboards"
	"github.com/stretchr/testify/require"
)

// TestGetTimeoutOnlyReportsItsOwnDeadline: Get's own deadline is reported as
// its time limit, but a caller that gives up first just gets its own ctx
// error back.
func TestGetTimeoutOnlyReportsItsOwnDeadline(t *testing.T) {
	db, cleanup := newTestDuckDB(t)
	defer cleanup()
	scope := dashboards.Scope{AccountID: uuid.New(), EnvID: uuid.New()}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := dashboards.Get(ctx, db, scope, dashboards.KeySessionKeys)
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, err.Error(), "time limit")

	dashboards.SetGetTimeout(t, time.Nanosecond)
	_, err = dashboards.Get(context.Background(), db, scope, dashboards.KeySessionKeys)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "time limit")
}
