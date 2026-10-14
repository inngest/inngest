package devserver

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/api/v2/apiv2base"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/generated"
	"github.com/inngest/inngest/pkg/coreapi/graph/resolvers"
	"github.com/inngest/inngest/pkg/cqrs"
	cqrsmanager "github.com/inngest/inngest/pkg/cqrs/manager"
	dbpkg "github.com/inngest/inngest/pkg/db"
	dbsqlite "github.com/inngest/inngest/pkg/db/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/stretchr/testify/require"
)

// Exercise legacy wire cursors through both public run lists and the real
// SQLite readers, including GraphQL's millisecond-based ended-at fast path.
func TestRunListsAcceptLegacyCursors(t *testing.T) {
	db, err := dbsqlite.Open(t.Context(), dbsqlite.Options{ForTest: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	adapter := dbsqlite.New(db)
	store := cqrsmanager.New(adapter)
	appID, fnID := uuid.New(), uuid.New()
	boundary := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	firstID, secondID := "01J00000000000000000000001", "01J00000000000000000000002"
	olderID, newerID := "01J00000000000000000000003", "01J00000000000000000000004"
	oldestID, newestID := "01J00000000000000000000005", "01J00000000000000000000006"
	for _, row := range []struct {
		id     string
		offset time.Duration
	}{
		{firstID, 0}, {secondID, 0}, {olderID, -time.Millisecond}, {newerID, time.Millisecond},
		{oldestID, -2 * time.Millisecond}, {newestID, 2 * time.Millisecond},
	} {
		at := boundary.Add(row.offset)
		require.NoError(t, adapter.Q().InsertSpan(t.Context(), dbpkg.InsertSpanParams{
			SpanID: row.id, TraceID: row.id, RunID: row.id,
			DynamicSpanID: sql.NullString{String: row.id, Valid: true},
			Name:          meta.SpanNameRun, Status: sql.NullString{String: enums.StepStatusCompleted.String(), Valid: true},
			StartTime: at, EndTime: at.Add(time.Second),
			AccountID: consts.DevServerAccountID.String(), EnvID: consts.DevServerEnvID.String(),
			AppID: appID.String(), FunctionID: fnID.String(),
		}))
		require.NoError(t, store.InsertTraceRun(t.Context(), &cqrs.TraceRun{
			RunID: row.id, TraceID: row.id, AccountID: consts.DevServerAccountID, WorkspaceID: consts.DevServerEnvID,
			AppID: appID, FunctionID: fnID, Status: enums.RunStatusCompleted,
			QueuedAt: at, StartedAt: at, EndedAt: at.Add(time.Second),
		}))
	}
	rest, err := apiv2.NewHTTPHandler(t.Context(), apiv2.ServiceOptions{Runs: NewRunProvider(store, nil)}, apiv2.HTTPHandlerOptions{}, apiv2base.NewBase())
	require.NoError(t, err)
	graphql := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &resolvers.Resolver{Data: store}}))

	for _, transport := range []string{"REST", "GraphQL"} {
		for _, field := range []string{"queuedAt", "startedAt", "endedAt"} {
			for _, direction := range []string{"ASC", "DESC"} {
				t.Run(transport+"/"+field+"/"+direction, func(t *testing.T) {
					at, column := boundary, "start_time"
					if field == "endedAt" {
						at, column = boundary.Add(time.Second), "end_time"
					}
					value := at.UnixMicro()
					if transport == "GraphQL" && field == "endedAt" {
						column, value = "ended_at", at.UnixMilli()
					}
					legacyJSON := fmt.Sprintf(`{"id":%q,"c":{%q:{"f":%q,"v":%d}}}`, firstID, column, column, value)
					legacy := base64.StdEncoding.EncodeToString([]byte(legacyJSON))
					withZero := base64.StdEncoding.EncodeToString([]byte(strings.Replace(legacyJSON, `"v":`, `"ns":0,"v":`, 1)))
					fetch := func(cursor string) ([]string, string) {
						t.Helper()
						from, until := boundary.Add(-time.Minute).Format(time.RFC3339), boundary.Add(time.Minute).Format(time.RFC3339)
						rec := httptest.NewRecorder()
						if transport == "REST" {
							query := url.Values{"cursor": {cursor}, "limit": {"2"}, "timeField": {field}, "order": {direction}, "from": {from}, "until": {until}}
							rest.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v2/runs?"+query.Encode(), nil))
							require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
							var response struct {
								Data []struct{ ID string }
								Page struct{ Cursor string }
							}
							require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
							ids := make([]string, 0, len(response.Data))
							for _, run := range response.Data {
								ids = append(ids, run.ID)
							}
							return ids, response.Page.Cursor
						}
						gqlField := map[string]string{"queuedAt": "QUEUED_AT", "startedAt": "STARTED_AT", "endedAt": "ENDED_AT"}[field]
						body, err := json.Marshal(map[string]any{
							"query":     `query($after: String!, $from: Time!, $until: Time!) { runs(first: 2, after: $after, orderBy: [{field: ` + gqlField + `, direction: ` + direction + `}], filter: {from: $from, until: $until, timeField: ` + gqlField + `}) { edges { node { id } } pageInfo { endCursor } } }`,
							"variables": map[string]any{"after": cursor, "from": from, "until": until},
						})
						require.NoError(t, err)
						req := httptest.NewRequest(http.MethodPost, "/v0/gql", bytes.NewReader(body))
						req.Header.Set("Content-Type", "application/json")
						graphql.ServeHTTP(rec, req)
						require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
						var response struct {
							Errors []json.RawMessage `json:"errors"`
							Data   struct {
								Runs struct {
									Edges    []struct{ Node struct{ ID string } }
									PageInfo struct{ EndCursor string }
								}
							}
						}
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
						require.Empty(t, response.Errors, rec.Body.String())
						ids := make([]string, 0, len(response.Data.Runs.Edges))
						for _, edge := range response.Data.Runs.Edges {
							ids = append(ids, edge.Node.ID)
						}
						return ids, response.Data.Runs.PageInfo.EndCursor
					}
					ids, cursor := fetch(legacy)
					pageEnd, lastID, offset := newerID, newestID, time.Millisecond
					if direction == "DESC" {
						pageEnd, lastID, offset = olderID, oldestID, -time.Millisecond
					}
					require.Equal(t, []string{secondID, pageEnd}, ids)
					zeroIDs, zeroCursor := fetch(withZero)
					require.Equal(t, ids, zeroIDs)
					require.Equal(t, cursor, zeroCursor)
					raw, err := base64.StdEncoding.DecodeString(cursor)
					require.NoError(t, err)
					endValue := at.Add(offset).UnixMicro()
					if transport == "GraphQL" && field == "endedAt" {
						endValue = at.Add(offset).UnixMilli()
					}
					require.JSONEq(t, fmt.Sprintf(`{"id":%q,"c":{%q:{"f":%q,"v":%d}}}`, pageEnd, column, column, endValue), string(raw))
					ids, cursor = fetch(cursor)
					require.Equal(t, []string{lastID}, ids)
					// REST omits the continuation when this final page has no more rows.
					if cursor != "" {
						ids, _ = fetch(cursor)
						require.Empty(t, ids)
					}
				})
			}
		}
	}
}
