package devserver

import (
	"context"
	"encoding/json"
	"strings"

	apiv2 "github.com/inngest/inngest/pkg/api/v2"
	"github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/inngest"
	"github.com/inngest/inngest/pkg/util"
)

type runsReader struct {
	q db.Querier
}

func NewRunsReader(q db.Querier) apiv2.RunsReader {
	return &runsReader{q: q}
}

func (r *runsReader) GetRuns(ctx context.Context, opts apiv2.GetRunsOpts) (*apiv2.GetRunsResult, error) {
	rows, err := r.q.GetRuns(ctx, db.GetRunsParams{
		EventID:       opts.EventID,
		Cursor:        opts.Cursor,
		Limit:         int64(opts.Limit + 1),
		IncludeOutput: opts.IncludeOutput,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(rows) > opts.Limit
	if hasMore {
		rows = rows[:opts.Limit]
	}

	runs := make([]*apiv2.RunListItem, 0, len(rows))
	for _, row := range rows {
		runs = append(runs, runListItemFromRow(row, opts.IncludeOutput))
	}

	return &apiv2.GetRunsResult{
		Runs:    runs,
		HasMore: hasMore,
	}, nil
}

func runListItemFromRow(row *db.RunListItemRow, includeOutput bool) *apiv2.RunListItem {
	fn := inngest.Function{}
	_ = json.Unmarshal([]byte(row.FunctionConfig), &fn)

	functionName := fn.Name
	if functionName == "" {
		functionName = row.FunctionName
	}

	appID := row.AppName
	if appID == "" {
		appID = row.FunctionAppID.String()
	}

	run := &apiv2.RunListItem{
		RunID:        row.FunctionRun.RunID,
		RunStartedAt: row.FunctionRun.RunStartedAt,
		EventID:      row.FunctionRun.EventID,
		FunctionID:   publicRunListFunctionID(row.AppName, row.FunctionSlug, fn.Slug),
		FunctionName: functionName,
		AppID:        appID,
	}

	if !row.FunctionRun.BatchID.IsZero() {
		run.BatchID = &row.FunctionRun.BatchID
	}
	if row.FunctionRun.Cron.Valid {
		run.Cron = &row.FunctionRun.Cron.String
	}
	if row.FunctionFinish.Status.Valid {
		run.Status, _ = enums.RunStatusString(row.FunctionFinish.Status.String)
		if row.FunctionFinish.CreatedAt.Valid {
			run.EndedAt = &row.FunctionFinish.CreatedAt.Time
		}
		if includeOutput && len(row.Output) > 0 {
			run.Output = publicRunOutput(row.Output)
		}
	}

	return run
}

func publicRunListFunctionID(appID string, storedFunctionID string, configFunctionID string) string {
	if configFunctionID != "" && configFunctionID != storedFunctionID {
		return configFunctionID
	}

	functionID := configFunctionID
	if functionID == "" {
		functionID = storedFunctionID
	}
	if appID != "" {
		return strings.TrimPrefix(functionID, appID+"-")
	}
	return functionID
}

func publicRunOutput(raw []byte) json.RawMessage {
	output := util.EnsureJSON(json.RawMessage(raw))

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(output, &envelope); err == nil {
		if data, ok := envelope["data"]; ok {
			output = data
		}
	}

	var opcodes []struct {
		Op   string          `json:"op"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(output, &opcodes); err != nil {
		return util.EnsureJSON(output)
	}

	for i := len(opcodes) - 1; i >= 0; i-- {
		if opcodes[i].Op == enums.OpcodeRunComplete.String() || opcodes[i].Op == enums.OpcodeSyncRunComplete.String() {
			return util.EnsureJSON(opcodes[i].Data)
		}
	}

	return util.EnsureJSON(output)
}
