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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FunctionRunOpt struct {
	Cursor    string
	Items     int
	Status    []string
	TimeField models.RunsV2OrderByField
	Order     []models.RunsV2OrderBy
	Query     *string
	Start     time.Time
	End       time.Time
}

func (o FunctionRunOpt) OrderBy() string {
	if len(o.Order) == 0 {
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

func (c *Client) FunctionRuns(ctx context.Context, opts FunctionRunOpt) ([]FnRunEdge, FnRunPageInfo, int) {
	c.Helper()

	items := 40
	if opts.Items > 0 {
		items = opts.Items
	}

	cursor := "null"
	if opts.Cursor != "" {
		cursor = fmt.Sprintf(`"%s"`, opts.Cursor)
	}

	timeField := models.RunsV2OrderByFieldQueuedAt
	if opts.TimeField.IsValid() {
		timeField = opts.TimeField
	}

	query := fmt.Sprintf(`
	query GetFunctionRunsV2(
		$startTime: Time!,
		$endTime: Time!,
		$timeField: RunsV2OrderByField = QUEUED_AT,
		$status: [FunctionRunStatus!],
		$first: Int = 40,
		$query: String
	) {
		runs(
			first: $first,
			after: %s,
			filter: { from: $startTime, until: $endTime, status: $status, timeField: $timeField, query: $query },
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
			totalCount
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
			"query":     opts.Query,
		},
	})
	if len(resp.Errors) > 0 {
		c.Fatalf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		Runs struct {
			Edges      []FnRunEdge   `json:"edges"`
			PageInfo   FnRunPageInfo `json:"pageInfo"`
			TotalCount int           `json:"totalCount"`
		}
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		c.Fatal(err.Error())
	}

	return data.Runs.Edges, data.Runs.PageInfo, data.Runs.TotalCount
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
		c.Fatal(err.Error())
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
			t.Fatal(msg)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return run
}

// WaitForRunTraces waits for run traces with a matching status for a predefined timeout and interval.
// Once run traces are available, they are returned and tests continue. If run traces are missing or invalid, the test will fail.
func (c *Client) WaitForRunTraces(ctx context.Context, t *testing.T, runID *string, opts WaitForRunTracesOptions) *RunV2 {
	if opts.Interval == 0 {
		opts.Interval = 2 * time.Second
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}

	var traces *RunV2
	require.NotNil(t, runID)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		a := assert.New(t)
		a.NotEmpty(runID)

		run, err := c.RunTraces(ctx, *runID)
		a.NoError(err)
		a.NotNil(run)
		a.Equal(run.Status, opts.Status.String())

		if opts.ChildSpanCount > 0 {
			a.NotNil(run.Trace)
			a.True(run.Trace.IsRoot)
			a.Len(run.Trace.ChildSpans, opts.ChildSpanCount)
		}

		traces = run
	}, opts.Timeout, opts.Interval)

	return traces
}

type WaitForRunTracesOptions struct {
	Status   models.FunctionStatus
	Timeout  time.Duration
	Interval time.Duration

	ChildSpanCount int
}

func (c *Client) RunTraces(ctx context.Context, runID string) (*RunV2, error) {
	c.Helper()

	if runID == "" {
		return nil, nil
	}

	query := `
		query GetTraceRun($runID: String!) {
	  	run(runID: $runID) {
				status
				traceID
				isBatch
				batchCreatedAt
				cronSchedule
        endedAt

				trace {
					...TraceDetails
					childrenSpans {
						...TraceDetails
						childrenSpans {
							...TraceDetails
						}
					}
				}
			}
		}
		fragment TraceDetails on RunTraceSpan {
			name
			runID
			status
			attempts
			isRoot
			parentSpanID
			spanID
			startedAt
			endedAt
			duration
			outputID
			stepOp
			stepInfo {
				__typename
				... on InvokeStepInfo {
					triggeringEventID
					functionID
					timeout
					returnEventID
  				runID
					timedOut
				}
				... on SleepStepInfo {
					sleepUntil
				}
				... on WaitForEventStepInfo {
					eventName
					expression
					timeout
					foundEventID
					timedOut
				}
				... on RunStepInfo {
					type
				}
			}
		}
	`

	resp, err := c.DoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"runID": runID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("err with fnrun trace query: %w", err)
	}

	type response struct {
		Run RunV2
	}
	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		return nil, fmt.Errorf("could not unmarshal response data: %w", err)
	}

	return &data.Run, nil
}

type RunV2 struct {
	TraceID string `json:"traceID"`
	// RunID   string        `json:"runID"`
	Status         string        `json:"status"`
	Trace          *runTraceSpan `json:"trace,omitempty"`
	IsBatch        bool          `json:"isBatch"`
	BatchCreatedAt *time.Time    `json:"batchCreatedAt,omitempty"`
	CronSchedule   *string       `json:"cronSchedule,omitempty"`
	EndedAt        *time.Time    `json:"endedAt,omitempty"`
}

type runTraceSpan struct {
	Name         string         `json:"name"`
	RunID        string         `json:"runID"`
	Status       string         `json:"status"`
	Attempts     int            `json:"attempts"`
	IsRoot       bool           `json:"isRoot"`
	TraceID      string         `json:"traceID"`
	ParentSpanID string         `json:"parentSpanID"`
	SpanID       string         `json:"spanID"`
	Duration     int64          `json:"duration"`
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	EndedAt      *time.Time     `json:"endedAt,omitempty"`
	ChildSpans   []runTraceSpan `json:"childrenSpans"`
	OutputID     *string        `json:"outputID,omitempty"`
	StepOp       string         `json:"stepOp"`
	StepInfo     any            `json:"stepInfo,omitempty"`
}

func (c *Client) RunSpanOutput(ctx context.Context, outputID string) *models.RunTraceSpanOutput {
	c.Helper()

	if outputID == "" {
		return nil
	}

	query := `
		query GetTraceSpanOutput($outputID: String!) {
			output: runTraceSpanOutputByID(outputID: $outputID) {
				data
				error {
					name
					message
					stack
				}
			}
		}
	`

	resp := c.MustDoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"outputID": outputID,
		},
	})
	if len(resp.Errors) > 0 {
		c.Fatalf("err with span output query: %#v", resp.Errors)
	}

	type response struct {
		Output *models.RunTraceSpanOutput
	}
	data := response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		c.Fatal(err.Error())
	}

	return data.Output
}

func (c *Client) ExpectSpanOutput(t require.TestingT, expected string, output *models.RunTraceSpanOutput) {
	require.NotNil(t, output)
	require.NotNil(t, output.Data)
	require.Nil(t, output.Error)
	require.Contains(t, *output.Data, expected)
}

func (c *Client) ExpectSpanErrorOutput(t require.TestingT, msg string, stack string, output *models.RunTraceSpanOutput) {
	require.NotNil(t, output)
	require.Nil(t, output.Data)
	require.NotNil(t, output.Error)
	if msg != "" {
		require.Contains(t, output.Error.Message, msg)
	}
	if stack != "" {
		require.NotNil(t, output.Error.Stack)
		require.Contains(t, *output.Error.Stack, stack)
	}
}

func (c *Client) RunTrigger(ctx context.Context, runID string) *models.RunTraceTrigger {
	c.Helper()

	if runID == "" {
		return nil
	}

	query := `
		query GetTraceRunTrigger($runID: String!) {
			runTrigger(runID: $runID) {
				eventName
				IDs
				timestamp
				payloads
				isBatch
				batchID
				cron
			}
		}
	`

	resp := c.MustDoGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"runID": runID,
		},
	})
	if len(resp.Errors) > 0 {
		c.Fatalf("err with fnrun trace query: %#v", resp.Errors)
	}

	type response struct {
		RunTrigger *models.RunTraceTrigger
	}
	data := response{}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		c.Fatalf("err with run trigger query: %#v", err)
	}

	return data.RunTrigger
}

type runByEventID struct {
	ID string `json:"id"`
}

func (c *Client) RunsByEventID(ctx context.Context, eventID string) ([]runByEventID, error) {
	c.Helper()

	query := `
		query Q($eventID: ID!) {
			event(query: { eventId: $eventID }) {
				functionRuns {
					id
				}
			}
		}`

	resp := c.doGQL(ctx, graphql.RawParams{
		Query: query,
		Variables: map[string]any{
			"eventID": eventID,
		},
	})
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("err with gql: %#v", resp.Errors)
	}

	type response struct {
		Event struct {
			FunctionRuns []runByEventID `json:"functionRuns"`
		} `json:"event"`
	}

	data := &response{}
	if err := json.Unmarshal(resp.Data, data); err != nil {
		c.Fatal(err.Error())
	}

	return data.Event.FunctionRuns, nil
}
