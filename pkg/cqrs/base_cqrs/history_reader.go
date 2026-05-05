package base_cqrs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	dbpkg "github.com/inngest/inngest/pkg/db"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/history_reader"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/usage"
	"github.com/oklog/ulid/v2"
)

func NewHistoryReader(adapter dbpkg.Adapter) history_reader.Reader {
	return &reader{
		q: adapter.Q(),
	}
}

type reader struct {
	q dbpkg.Querier
}

func (r *reader) CountRuns(
	ctx context.Context,
	opts history_reader.CountRunOpts,
) (int, error) {
	count, err := r.q.HistoryCountRuns(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count runs: %w", err)
	}

	return int(count), nil
}

func (r *reader) GetRun(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) (history_reader.Run, error) {
	rawRun, err := r.q.GetFunctionRun(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return history_reader.Run{}, history_reader.ErrNotFound
		}

		return history_reader.Run{}, fmt.Errorf("failed to get run: %w", err)
	}

	run, err := sqlToRun(&rawRun.FunctionRun, &rawRun.FunctionFinish)
	if err != nil {
		return history_reader.Run{}, fmt.Errorf("failed to convert run: %w", err)
	}

	return *run, nil
}

func (r *reader) GetFunctionRun(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	runID ulid.ULID,
) (*cqrs.FunctionRun, error) {
	run, err := r.GetRun(ctx, runID, history_reader.GetRunOpts{
		AccountID:   accountID,
		WorkspaceID: &workspaceID,
	})
	if err != nil {
		return nil, err
	}
	return run.ToCQRS(), nil
}

// For the API - CQRS return
func (r *reader) GetFunctionRunsFromEvents(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	eventIDs []ulid.ULID,
) ([]*cqrs.FunctionRun, error) {
	runs, err := r.q.GetFunctionRunsFromEvents(ctx, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get function runs: %w", err)
	}

	result := []*cqrs.FunctionRun{}
	for _, rawRun := range runs {
		run, err := sqlToRun(&rawRun.FunctionRun, &rawRun.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, run.ToCQRS())
	}

	return result, nil
}

func (r *reader) GetRunHistory(
	ctx context.Context,
	runID ulid.ULID,
	opts history_reader.GetRunOpts,
) ([]*history_reader.RunHistory, error) {
	rows, err := r.q.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, history_reader.ErrNotFound
		}

		return nil, fmt.Errorf("failed to get run history: %w", err)
	}

	var items []*history_reader.RunHistory
	for _, row := range rows {
		historyItem, err := sqlToRunHistory(row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert history item: %w", err)
		}
		items = append(items, historyItem)
	}

	return items, nil
}

func (r *reader) GetRunHistoryItemOutput(
	ctx context.Context,
	historyID ulid.ULID,
	opts history_reader.GetHistoryOutputOpts,
) (*string, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	item, err := r.q.GetHistoryItem(ctx, historyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, history_reader.ErrNotFound
		}

		return nil, fmt.Errorf("failed to get history item: %w", err)
	}

	if !item.Result.Valid {
		return nil, history_reader.ErrNotFound
	}

	var (
		result            *string
		execHistoryResult history.Result
	)

	if err := json.Unmarshal([]byte(item.Result.String), &execHistoryResult); err == nil {
		result = &execHistoryResult.Output
	}

	return result, nil
}

func (r *reader) GetRuns(
	ctx context.Context,
	opts history_reader.GetRunsOpts,
) ([]history_reader.Run, error) {
	runs, err := r.q.GetFunctionRuns(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get runs: %w", err)
	}

	var result []history_reader.Run
	for _, run := range runs {
		r, err := sqlToRun(&run.FunctionRun, &run.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, *r)
	}

	return result, nil
}

func (r *reader) GetRunsByEventID(
	ctx context.Context,
	eventID ulid.ULID,
	opts history_reader.GetRunsByEventIDOpts,
) ([]history_reader.Run, error) {
	runs, err := r.q.GetFunctionRunsFromEvents(ctx, []ulid.ULID{eventID})
	if err != nil {
		return nil, fmt.Errorf("failed to get runs by event ID: %w", err)
	}

	var result []history_reader.Run
	for _, run := range runs {
		r, err := sqlToRun(&run.FunctionRun, &run.FunctionFinish)
		if err != nil {
			return nil, fmt.Errorf("failed to convert run: %w", err)
		}

		result = append(result, *r)
	}

	return result, nil
}

func (r *reader) GetSkippedRunsByEventID(
	_ context.Context,
	_ ulid.ULID,
	_ history_reader.GetRunsByEventIDOpts,
) ([]history_reader.SkippedRun, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) GetUsage(
	ctx context.Context,
	opts history_reader.GetUsageOpts,
) ([]usage.UsageSlot, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) GetReplayRuns(ctx context.Context, opts history_reader.GetReplayRunsOpts) ([]history_reader.ReplayRun, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) CountReplayRuns(ctx context.Context, opts history_reader.CountReplayRunsOpts) (history_reader.ReplayRunCounts, error) {
	return history_reader.ReplayRunCounts{}, errors.New("not implemented")
}

func (r *reader) GetActiveRunIDs(
	ctx context.Context,
	opts history_reader.GetActiveRunIDsOpts,
) ([]ulid.ULID, error) {
	return nil, errors.New("not implemented")
}

func (r *reader) CountActiveRuns(
	ctx context.Context,
	opts history_reader.CountActiveRunsOpts,
) (int, error) {
	return 0, errors.New("not implemented")
}

func (w wrapper) GetRunDefers(ctx context.Context, runID ulid.ULID) ([]cqrs.RunDefer, error) {
	rows, err := w.q.GetFunctionRunHistory(ctx, runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get run history: %w", err)
	}

	log := logger.StdlibLogger(ctx)

	type entry struct {
		userID    string
		fnSlug    string
		input     json.RawMessage
		cancelled bool
	}
	order := []string{}
	byHashed := map[string]*entry{}

	for _, row := range rows {
		if row == nil || !row.Result.Valid || row.Result.String == "" {
			continue
		}

		var res history.Result
		if err := json.Unmarshal([]byte(row.Result.String), &res); err != nil {
			log.Warn("failed to unmarshal history result while extracting defers",
				"run_id", runID.String(), "history_id", row.ID.String(), "error", err)
			continue
		}
		if res.RawOutput == nil {
			continue
		}

		opcodes, err := decodeOpcodes(res.RawOutput)
		if err != nil {
			log.Warn("failed to decode raw_output opcodes while extracting defers",
				"run_id", runID.String(), "history_id", row.ID.String(), "error", err)
			continue
		}

		for _, op := range opcodes {
			switch op.Op {
			case enums.OpcodeDeferAdd:
				if op.ID == "" {
					log.Warn("DeferAdd opcode missing hashed step ID; skipping",
						"run_id", runID.String(), "history_id", row.ID.String())
					continue
				}
				addOpts, err := op.DeferAddOpts()
				if err != nil {
					log.Warn("failed to parse DeferAdd opts; skipping",
						"run_id", runID.String(), "hashed_id", op.ID, "error", err)
					continue
				}
				userID := op.ID
				if op.Userland != nil && op.Userland.ID != "" {
					userID = op.Userland.ID
				}
				existing, ok := byHashed[op.ID]
				if !ok {
					existing = &entry{}
					byHashed[op.ID] = existing
					order = append(order, op.ID)
				}
				existing.userID = userID
				existing.fnSlug = addOpts.FnSlug
				if len(addOpts.Input) > 0 {
					existing.input = addOpts.Input
				}
			case enums.OpcodeDeferCancel:
				cancelOpts, err := op.DeferCancelOpts()
				if err != nil {
					log.Warn("failed to parse DeferCancel opts; skipping",
						"run_id", runID.String(), "error", err)
					continue
				}
				if existing, ok := byHashed[cancelOpts.TargetHashedID]; ok {
					existing.cancelled = true
				}
			}
		}
	}

	if len(order) == 0 {
		return nil, nil
	}

	hashToEventID := make(map[string]ulid.ULID, len(order))
	eventIDs := make([]string, 0, len(order))
	for _, h := range order {
		if byHashed[h].cancelled {
			continue
		}
		evtID, err := event.DeferredScheduleEventID(runID, h)
		if err != nil {
			log.Warn("failed to compute deferred schedule event ID; skipping run lookup",
				"run_id", runID.String(), "hashed_id", h, "error", err)
			continue
		}
		hashToEventID[h] = evtID
		eventIDs = append(eventIDs, evtID.String())
	}

	var runsByUserEventID map[string]*cqrs.FunctionRun
	if len(eventIDs) > 0 {
		runsByUserEventID, err = w.fetchRunsByUserEventIDs(ctx, eventIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch child runs for defers: %w", err)
		}
	}

	out := make([]cqrs.RunDefer, 0, len(order))
	for _, h := range order {
		e := byHashed[h]
		rd := cqrs.RunDefer{
			ID:     e.userID,
			FnSlug: e.fnSlug,
			Input:  e.input,
		}
		if e.cancelled {
			rd.Status = cqrs.RunDeferStatusAborted
		} else {
			rd.Status = cqrs.RunDeferStatusScheduled
			if evtID, ok := hashToEventID[h]; ok {
				rd.Run = runsByUserEventID[evtID.String()]
			}
		}
		out = append(out, rd)
	}

	return out, nil
}

// fetchRunsByUserEventIDs resolves a slice of user-facing event_id strings to
// the child function runs they triggered. Hand-rolled because the prepared
// GetFunctionRunsFromEvents query matches function_runs.event_id, which stores
// the events.internal_id ULID — not the user-facing events.event_id text the
// deterministic deferred.schedule IDs use.
func (w wrapper) fetchRunsByUserEventIDs(
	ctx context.Context,
	userEventIDs []string,
) (map[string]*cqrs.FunctionRun, error) {
	out := map[string]*cqrs.FunctionRun{}
	if len(userEventIDs) == 0 {
		return out, nil
	}

	placeholders := strings.Repeat(",?", len(userEventIDs))[1:]
	args := make([]any, len(userEventIDs))
	for i, id := range userEventIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT
	e.event_id,
	fr.run_id,
	fr.run_started_at,
	fr.function_id,
	fr.function_version,
	fr.event_id,
	fr.batch_id,
	fr.original_run_id,
	fr.cron,
	fr.workspace_id,
	ff.run_id,
	ff.status,
	ff.output,
	ff.completed_step_count,
	ff.created_at
FROM events AS e
INNER JOIN function_runs AS fr ON fr.event_id = e.internal_id
LEFT JOIN function_finishes AS ff ON ff.run_id = fr.run_id
WHERE e.event_id IN (%s)
`, placeholders)

	rows, err := w.adapter.Conn().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query child runs by user event id: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userEventID string
		var fr dbpkg.FunctionRun
		var ff dbpkg.FunctionFinish
		if err := rows.Scan(
			&userEventID,
			&fr.RunID,
			&fr.RunStartedAt,
			&fr.FunctionID,
			&fr.FunctionVersion,
			&fr.EventID,
			&fr.BatchID,
			&fr.OriginalRunID,
			&fr.Cron,
			&fr.WorkspaceID,
			&ff.RunID,
			&ff.Status,
			&ff.Output,
			&ff.CompletedStepCount,
			&ff.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan child run row: %w", err)
		}
		out[userEventID] = toCQRSRun(fr, ff)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate child run rows: %w", err)
	}
	return out, nil
}

func (w wrapper) GetRunDeferredFrom(ctx context.Context, runID ulid.ULID) (*cqrs.RunDeferredFrom, error) {
	log := logger.StdlibLogger(ctx)

	// Join on events.internal_id — see fetchRunsByUserEventIDs for the why.
	const query = `
SELECT e.event_data
FROM function_runs AS fr
INNER JOIN events AS e ON fr.event_id = e.internal_id
WHERE fr.run_id = ? AND e.event_name = ?
LIMIT 1
`

	var eventData string
	row := w.adapter.Conn().QueryRowContext(ctx, query, runID, consts.FnDeferScheduleName)
	if err := row.Scan(&eventData); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query deferred trigger event: %w", err)
	}

	var envelope struct {
		Inngest *event.DeferredScheduleMetadata `json:"_inngest"`
	}
	if err := json.Unmarshal([]byte(eventData), &envelope); err != nil {
		log.Warn("failed to unmarshal deferred schedule event data",
			"run_id", runID.String(), "error", err)
		return nil, nil
	}
	if envelope.Inngest == nil {
		return nil, nil
	}
	meta := envelope.Inngest

	parentRunID, err := ulid.Parse(meta.ParentRunID)
	if err != nil {
		log.Warn("invalid parent_run_id in deferred schedule metadata",
			"run_id", runID.String(),
			"parent_run_id", meta.ParentRunID,
			"error", err)
		return nil, nil
	}

	out := &cqrs.RunDeferredFrom{
		ParentRunID:  parentRunID,
		ParentFnSlug: meta.ParentFnSlug,
	}

	parent, err := w.GetFunctionRun(ctx, uuid.Nil, uuid.Nil, parentRunID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Warn("failed to load parent run for deferred-from linkage",
			"run_id", runID.String(),
			"parent_run_id", parentRunID.String(),
			"error", err)
	} else if parent != nil {
		out.ParentRun = parent
	}

	return out, nil
}

// decodeOpcodes coerces history.Result.RawOutput (any) into a typed opcode
// slice. RawOutput's runtime shape depends on how Result was loaded — see
// marshalJSONAsString for the same coercion used at write time.
func decodeOpcodes(raw any) ([]state.GeneratorOpcode, error) {
	if raw == nil {
		return nil, nil
	}
	str, err := marshalJSONAsString(raw)
	if err != nil {
		return nil, fmt.Errorf("re-marshal raw_output: %w", err)
	}
	if str == "" {
		return nil, nil
	}
	var ops []state.GeneratorOpcode
	if err := json.Unmarshal([]byte(str), &ops); err != nil {
		return nil, fmt.Errorf("raw_output is not a recognized opcode shape: %w", err)
	}
	return ops, nil
}

func sqlToRun(item *dbpkg.FunctionRun, finish *dbpkg.FunctionFinish) (*history_reader.Run, error) {
	if item == nil {
		return nil, history_reader.ErrNotFound
	}

	var (
		endedAt       *time.Time
		batchID       *ulid.ULID
		output        *string
		status        = enums.RunStatusRunning
		originalRunID *ulid.ULID
		cron          *string
	)

	if finish != nil && finish.Status.Valid {
		status, _ = enums.RunStatusString(finish.Status.String)
		output = &finish.Output.String
		endedAt = &finish.CreatedAt.Time
	}

	if !item.BatchID.IsZero() {
		batchID = &item.BatchID
	}

	if !item.OriginalRunID.IsZero() {
		originalRunID = &item.OriginalRunID
	}

	if item.Cron.Valid && item.Cron.String != "" {
		cron = &item.Cron.String
	}

	return &history_reader.Run{
		AccountID:       nilUUID,
		BatchID:         batchID,
		EndedAt:         endedAt,
		EventID:         item.EventID,
		ID:              item.RunID,
		OriginalRunID:   originalRunID,
		Output:          output,
		StartedAt:       item.RunStartedAt,
		Status:          status,
		WorkflowID:      item.FunctionID,
		WorkspaceID:     nilUUID,
		WorkflowVersion: int(item.FunctionVersion),
		Cron:            cron,
	}, nil
}

func sqlToRunHistory(item *dbpkg.History) (*history_reader.RunHistory, error) {
	historyType, err := enums.HistoryTypeString(item.Type)
	if err != nil {
		return nil, fmt.Errorf("invalid history type: %w", err)
	}

	var (
		cancel *history_reader.RunHistoryCancel

		groupID *uuid.UUID

		stepName *string

		url *string

		stepType *enums.HistoryStepType

		result            *history_reader.RunHistoryResult
		execHistoryResult *history.Result

		sleep            *history_reader.RunHistorySleep
		execHistorySleep *history.Sleep

		waitForEvent            *history_reader.RunHistoryWaitForEvent
		execHistoryWaitForEvent *history.WaitForEvent

		waitResult            *history_reader.RunHistoryWaitResult
		execHistoryWaitResult *history.WaitResult

		invokeFunction            *history_reader.RunHistoryInvokeFunction
		execHistoryInvokeFunction *history.InvokeFunction

		invokeFunctionResult            *history_reader.RunHistoryInvokeFunctionResult
		execHistoryInvokeFunctionResult *history.InvokeFunctionResult
	)

	if item.CancelRequest.Valid {
		_ = json.Unmarshal([]byte(item.CancelRequest.String), &cancel)
	}

	if item.GroupID.Valid {
		if gid, err := uuid.Parse(item.GroupID.String); err == nil {
			groupID = &gid
		}
	}

	if item.StepName.Valid {
		stepName = &item.StepName.String
	}

	if item.Url.Valid {
		url = &item.Url.String
	}

	if item.StepType.Valid {
		if st, err := enums.HistoryStepTypeString(item.StepType.String); err == nil {
			stepType = &st
		}
	}

	if item.Result.Valid {
		if err := json.Unmarshal([]byte(item.Result.String), &execHistoryResult); err == nil && execHistoryResult != nil {
			result = &history_reader.RunHistoryResult{
				DurationMS:  execHistoryResult.DurationMS,
				ErrorCode:   execHistoryResult.ErrorCode,
				Framework:   execHistoryResult.Framework,
				Platform:    execHistoryResult.Platform,
				SDKLanguage: execHistoryResult.SDKLanguage,
				SDKVersion:  execHistoryResult.SDKVersion,
				SizeBytes:   execHistoryResult.SizeBytes,
			}
		}
	}

	if item.Sleep.Valid {
		if err := json.Unmarshal([]byte(item.Sleep.String), &execHistorySleep); err == nil && execHistorySleep != nil {
			sleep = &history_reader.RunHistorySleep{
				Until: execHistorySleep.Until,
			}
		}
	}

	if item.WaitForEvent.Valid {
		if err := json.Unmarshal([]byte(item.WaitForEvent.String), &execHistoryWaitForEvent); err == nil && execHistoryWaitForEvent != nil {
			waitForEvent = &history_reader.RunHistoryWaitForEvent{
				EventName:  execHistoryWaitForEvent.EventName,
				Expression: execHistoryWaitForEvent.Expression,
				Timeout:    execHistoryWaitForEvent.Timeout,
			}
		}
	}

	if item.WaitResult.Valid {
		if err := json.Unmarshal([]byte(item.WaitResult.String), &execHistoryWaitResult); err == nil && execHistoryWaitResult != nil {
			waitResult = &history_reader.RunHistoryWaitResult{
				EventID: execHistoryWaitResult.EventID,
				Timeout: execHistoryWaitResult.Timeout,
			}
		}
	}

	if item.InvokeFunction.Valid {
		if err := json.Unmarshal([]byte(item.InvokeFunction.String), &execHistoryInvokeFunction); err == nil && execHistoryInvokeFunction != nil {
			invokeFunction = &history_reader.RunHistoryInvokeFunction{
				CorrelationID: execHistoryInvokeFunction.CorrelationID,
				EventID:       execHistoryInvokeFunction.EventID,
				FunctionID:    execHistoryInvokeFunction.FunctionID,
				Timeout:       execHistoryInvokeFunction.Timeout,
			}
		}
	}

	if item.InvokeFunctionResult.Valid {
		if err := json.Unmarshal([]byte(item.InvokeFunctionResult.String), &execHistoryInvokeFunctionResult); err == nil && execHistoryInvokeFunctionResult != nil {
			invokeFunctionResult = &history_reader.RunHistoryInvokeFunctionResult{
				EventID: execHistoryInvokeFunctionResult.EventID,
				RunID:   execHistoryInvokeFunctionResult.RunID,
				Timeout: execHistoryInvokeFunctionResult.Timeout,
			}
		}
	}

	return &history_reader.RunHistory{
		Attempt:              item.Attempt,
		Cancel:               cancel,
		CreatedAt:            item.CreatedAt,
		FunctionVersion:      item.FunctionVersion,
		GroupID:              groupID,
		ID:                   item.ID,
		InvokeFunction:       invokeFunction,
		InvokeFunctionResult: invokeFunctionResult,
		Result:               result,
		RunID:                item.RunID,
		Sleep:                sleep,
		StepName:             stepName,
		StepType:             stepType,
		Type:                 historyType,
		URL:                  url,
		WaitForEvent:         waitForEvent,
		WaitResult:           waitResult,
	}, nil
}
