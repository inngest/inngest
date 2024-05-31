package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
)

type FunctionRunOpt struct {
	Cursor    string
	Items     int
	Status    []string
	TimeField models.FunctionRunTimeFieldV2
	Order     []models.RunsV2OrderBy
	Start     time.Time
	End       time.Time
}

func (o FunctionRunOpt) OrderBy() string {
	if o.Order == nil || len(o.Order) == 0 {
		return fmt.Sprintf("[ { field: %s, direction: %s } ]", models.RunsV2OrderByFieldQueuedAt, models.RunsOrderByDirectionDesc)
	}

	orderby := []string{}
	for _, o := range o.Order {
		order := fmt.Sprintf("{ field: %s, direction: %s }", o.Field, o.Direction)
		orderby = append(orderby, order)
	}

	res := "[ "
	res += strings.Join(orderby, ",")
	res += " ]"

	return res
}

type FnRunEdge struct {
	Cursor string
	Node   struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		TraceID   string    `json:"traceID"`
		QueuedAt  time.Time `json:"queuedAt"`
		StartedAt time.Time `json:"startedAt"`
		EndedAt   time.Time `json:"endedAt"`
	}
}

type FnRunPageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor,omitempty"`
}

func (c *Client) FunctionRuns(ctx context.Context, opts FunctionRunOpt) ([]FnRunEdge, FnRunPageInfo) {
	c.Helper()

	items := 40
	if opts.Items > 0 {
		items = opts.Items
	}

	cursor := "null"
	if opts.Cursor != "" {
		cursor = fmt.Sprintf(`"%s"`, opts.Cursor)
	}

	timeField := models.FunctionRunTimeFieldV2QueuedAt
	if opts.TimeField.IsValid() {
		timeField = opts.TimeField
	}

	query := fmt.Sprintf(`
	query GetFunctionRunsV2(
		$startTime: Time!,
		$endTime: Time!,
		$timeField: FunctionRunTimeFieldV2 = QUEUED_AT,
		$status: [FunctionRunStatus!],
		$first: Int = 40
	) {
		runs(
			first: $first,
			after: %s,
			filter: { from: $startTime, until: $endTime, status: $status, timeField: $timeField },
			orderBy: %s
		) {
			edges {
				cursor
				node {
					id
					status
 					traceID
 					startedAt
					queuedAt
 					endedAt
				}
			}
			pageInfo {
				startCursor
				endCursor
				hasNextPage
			}
		}
	}`,
		cursor,
		opts.OrderBy(),
	)

	resp := c.MustDoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"startTime": opts.Start,
			"endTime":   opts.End,
			"timeField": timeField,
			"status":    opts.Status,
			"first":     items,
		},
	})
	if len(resp.Errors) > 0 {
		c.Fatalf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		Runs struct {
			Edges    []FnRunEdge   `json:"edges"`
			PageInfo FnRunPageInfo `json:"pageInfo"`
		}
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		c.Fatalf(err.Error())
	}

	return data.Runs.Edges, data.Runs.PageInfo
}

type Run struct {
	Output string `json:"output"`
	Status string `json:"status"`
}

func (c *Client) Run(ctx context.Context, runID string) Run {
	c.Helper()

	query := `
		query GetRun($runID: ID!) {
			functionRun(query: { functionRunId: $runID }) {
				output
				status
			}
		}`

	resp := c.MustDoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"runID": runID,
		},
	})
	if len(resp.Errors) > 0 {
		c.Fatalf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		FunctionRun Run `json:"functionRun"`
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		c.Fatalf(err.Error())
	}

	return data.FunctionRun
}

func (c *Client) WaitForRunStatus(
	ctx context.Context,
	t *testing.T,
	expectedStatus string,
	runID *string,
) Run {
	t.Helper()

	start := time.Now()
	var run Run
	for {
		if runID != nil && *runID != "" {
			run = c.Run(ctx, *runID)
			if run.Status == expectedStatus {
				break
			}
		}

		if time.Since(start) > 5*time.Second {
			var msg string
			if runID == nil || *runID == "" {
				msg = "Run ID is empty"
			} else {
				msg = fmt.Sprintf("Expected status %s, got %s", expectedStatus, run.Status)
			}
			t.Fatalf(msg)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return run
}
