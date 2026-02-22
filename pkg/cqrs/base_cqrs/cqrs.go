package base_cqrs

import (
	"cmp"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	sq "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	sqexp "github.com/doug-martin/goqu/v9/exp"
	"github.com/elliotchance/orderedmap/v3"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/cqrs"
	sqlc_postgres "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/postgres"
	sqlc "github.com/inngest/inngest/pkg/cqrs/base_cqrs/sqlc/sqlite"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/inngest/inngest/pkg/execution/history"
	"github.com/inngest/inngest/pkg/execution/state"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/run"
	"github.com/inngest/inngest/pkg/tracing/meta"
	"github.com/inngest/inngest/pkg/tracing/metadata"
	"github.com/inngest/inngest/pkg/util"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
	"github.com/oklog/ulid/v2"
)

const (
	forceHTTPS = false
)

var (
	// end represents a ulid ending with 'Z', eg. a far out cursor.
	endULID = ulid.ULID([16]byte{'Z'})
	nilUUID = uuid.UUID{}
)

func NewQueries(db *sql.DB, driver string, o sqlc_postgres.NewNormalizedOpts) (q sqlc.Querier) {
	if driver == "postgres" {
		q = sqlc_postgres.NewNormalized(db, o)
	} else {
		q = sqlc.New(db)
	}

	return q
}

func NewCQRS(db *sql.DB, driver string, o sqlc_postgres.NewNormalizedOpts) cqrs.Manager {
	// Force goqu to use prepared statements for consistency with sqlc
	sq.SetDefaultPrepared(true)
	return wrapper{
		driver: driver,
		q:      NewQueries(db, driver, o),
		db:     db,
		opts:   o,
	}
}

type wrapper struct {
	driver string
	q      sqlc.Querier
	db     *sql.DB
	tx     *sql.Tx
	opts   sqlc_postgres.NewNormalizedOpts
}

func (w wrapper) isPostgres() bool {
	return w.driver == "postgres"
}

func (w wrapper) dialect() string {
	if w.isPostgres() {
		return "postgres"
	}

	return "sqlite3"
}

// spanRunsAdapter encapsulates all database-specific logic for GetSpanRuns.
// This only makes sense because spans and events have very similar structure in SQLite and postgres
// if we ever diverge more significantly, we should fork the query paths at a higher point and share less code
type spanRunsAdapter struct {
	dialect        string
	celConverter   run.ExprSQLConverter
	eventIdsExpr   sqexp.Expression
	buildEventJoin func(q *sq.SelectDataset) *sq.SelectDataset
	parseEventIDs  func(raw *string) []string
	parseTime      func(s string) (time.Time, error)
}

var sqliteSpanRunsAdapter = spanRunsAdapter{
	dialect:      "sqlite3",
	celConverter: run.SpanEventSQLiteConverter,
	eventIdsExpr: sq.L("MAX(spans.event_ids)").As("event_ids"),
	buildEventJoin: func(q *sq.SelectDataset) *sq.SelectDataset {
		// SQLite: json_each for unnesting
		// json_each('') errors with "malformed JSON", so we use NULLIF to convert empty strings
		// to NULL. json_each(NULL) safely returns no rows.
		return q.InnerJoin(sq.L("json_each(NULLIF(spans.event_ids, '')) AS je"), sq.On(sq.L("1=1"))).
			InnerJoin(sq.L("events"), sq.On(sq.L("je.value = events.event_id")))
	},
	parseEventIDs: func(raw *string) []string {
		// SQLite: plain JSON array
		var ids []string
		if raw != nil && *raw != "" {
			// Ignore error: return empty slice on parse failure
		_ = json.Unmarshal([]byte(*raw), &ids)
		}
		return ids
	},
	parseTime: func(s string) (time.Time, error) {
		// SQLite: we currently store the literal go time.Time string
		// strip monotonic clock suffix if present
		if idx := strings.Index(s, " m="); idx != -1 {
			s = s[:idx]
		}
		return time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", s)
	},
}

var postgresSpanRunsAdapter = spanRunsAdapter{
	dialect:      "postgres",
	celConverter: run.SpanEventPostgresConverter,
	// PostgreSQL: cast JSONB to text first (no MAX for JSONB)
	eventIdsExpr: sq.L("MAX(spans.event_ids::text)").As("event_ids"),
	buildEventJoin: func(q *sq.SelectDataset) *sq.SelectDataset {
		// PostgreSQL: jsonb_array_elements_text for unnesting
		// event_ids is JSONB containing a JSON string (double-encoded), e.g. "[\"uuid\"]" or ""
		// Extract string with #>>'{}', use NULLIF to handle empty strings, then parse as JSON
		return q.InnerJoin(
			sq.L("jsonb_array_elements_text(NULLIF(spans.event_ids#>>'{}', '')::jsonb) AS eid(event_id)"),
			sq.On(sq.L("true")),
		).InnerJoin(sq.T("events"), sq.On(sq.L("eid.event_id = events.event_id")))
	},
	parseEventIDs: func(raw *string) []string {
		// PostgreSQL: double-encoded JSON (a JSON string containing a JSON array)
		var ids []string
		if raw != nil && *raw != "" {
			var innerStr string
			if err := json.Unmarshal([]byte(*raw), &innerStr); err == nil {
				// Ignore error: return empty slice on parse failure
			_ = json.Unmarshal([]byte(innerStr), &ids)
			}
		}
		return ids
	},
	parseTime: func(s string) (time.Time, error) {
		return time.Parse(time.RFC3339Nano, s)
	},
}

func (w wrapper) spanRunsAdapter() spanRunsAdapter {
	if w.isPostgres() {
		return postgresSpanRunsAdapter
	}
	return sqliteSpanRunsAdapter
}

type normalizedSpan interface {
	GetTraceID() string
	GetRunID() string
	GetDynamicSpanID() sql.NullString
	GetParentSpanID() sql.NullString
	GetStartTime() interface{}
	GetEndTime() interface{}
	GetSpanFragments() any
}

func (w wrapper) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByRunID(ctx, runID.String())
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by run ID", "error", err)
		return nil, err
	}

	return mapRootSpansFromRows(ctx, spans, false)
}

func (w wrapper) GetSpansByDebugRunID(ctx context.Context, debugRunID ulid.ULID) ([]*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByDebugRunID(ctx, sql.NullString{String: debugRunID.String(), Valid: true})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by debug run ID", "error", err)
		return nil, err
	}

	if len(spans) == 0 {
		return nil, nil
	}

	return buildDebugRunSpan(ctx, spans)
}

func (w wrapper) GetSpansByDebugSessionID(ctx context.Context, debugSessionID ulid.ULID) ([][]*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByDebugSessionID(ctx, sql.NullString{String: debugSessionID.String(), Valid: true})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by debug session ID", "error", err)
		return nil, err
	}

	if len(spans) == 0 {
		return nil, nil
	}

	spansByDebugSession := make(map[string][]*sqlc.GetSpansByDebugSessionIDRow)
	for _, span := range spans {
		if span.DebugRunID.Valid {
			spansByDebugSession[span.DebugRunID.String] = append(spansByDebugSession[span.DebugRunID.String], span)
		}
	}

	var allDebugRuns [][]*cqrs.OtelSpan

	for _, runSpans := range spansByDebugSession {
		debugRunSpans, err := buildDebugRunSpan(ctx, runSpans)
		if err != nil {
			return nil, err
		}
		allDebugRuns = append(allDebugRuns, debugRunSpans)
	}

	return allDebugRuns, nil
}

var _ normalizedSpan = (*sqlc.GetRunSpanByRunIDRow)(nil)

func (w wrapper) GetRunSpanByRunID(ctx context.Context, runID ulid.ULID, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	// Ignore the workspace ID for now.
	span, err := w.q.GetRunSpanByRunID(ctx, sqlc.GetRunSpanByRunIDParams{
		RunID:     runID.String(),
		AccountID: accountID.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting span by run ID", "error", err)
		return nil, err
	}

	return w.mapSpanWithOpts(ctx, span, runID, accountID, workspaceID, opt)
}

func (w wrapper) GetStepSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	// Ignore the workspace ID for now.
	span, err := w.q.GetStepSpanByStepID(ctx, sqlc.GetStepSpanByStepIDParams{
		RunID:     runID.String(),
		StepID:    stepID,
		AccountID: accountID.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting step span by step ID", "error", err)
		return nil, err
	}

	return w.mapSpanWithOpts(ctx, span, runID, accountID, workspaceID, opt)
}

func (w wrapper) GetExecutionSpanByStepIDAndAttempt(ctx context.Context, runID ulid.ULID, stepID string, attempt int, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	// Ignore the workspace ID for now.
	span, err := w.q.GetExecutionSpanByStepIDAndAttempt(ctx, sqlc.GetExecutionSpanByStepIDAndAttemptParams{
		RunID:       runID.String(),
		StepID:      stepID,
		StepAttempt: int64(attempt),
		AccountID:   accountID.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting step execution span by step ID and attempt", "error", err)
		return nil, err
	}

	return w.mapSpanWithOpts(ctx, span, runID, accountID, workspaceID, opt)
}

func (w wrapper) GetLatestExecutionSpanByStepID(ctx context.Context, runID ulid.ULID, stepID string, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	// Ignore the workspace ID for now.
	span, err := w.q.GetLatestExecutionSpanByStepID(ctx, sqlc.GetLatestExecutionSpanByStepIDParams{
		RunID:     runID.String(),
		StepID:    stepID,
		AccountID: accountID.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting step execution span by step ID and attempt", "error", err)
		return nil, err
	}

	return w.mapSpanWithOpts(ctx, span, runID, accountID, workspaceID, opt)
}

func (w wrapper) GetSpanBySpanID(ctx context.Context, runID ulid.ULID, spanID string, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	// Ignore the workspace ID for now.
	span, err := w.q.GetSpanBySpanID(ctx, sqlc.GetSpanBySpanIDParams{
		RunID:     runID.String(),
		SpanID:    spanID,
		AccountID: accountID.String(),
	})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting step span by step ID", "error", err)
		return nil, err
	}

	return w.mapSpanWithOpts(ctx, span, runID, accountID, workspaceID, opt)
}

func (w wrapper) mapSpanWithOpts(ctx context.Context, span normalizedSpan, runID ulid.ULID, accountID, workspaceID uuid.UUID, opt cqrs.GetTraceSpanOpt) (*cqrs.OtelSpan, error) {
	spans := []normalizedSpan{span}
	if opt.IncludeMetadata {
		metadataSpans, err := w.q.GetMetadataSpansByParentSpanID(ctx, sqlc.GetMetadataSpansByParentSpanIDParams{
			RunID:        runID.String(),
			ParentSpanID: span.GetDynamicSpanID(),
			AccountID:    accountID.String(),
		})
		if err != nil {
			logger.StdlibLogger(ctx).Error("error getting metadata spans by parent span ID", "error", err)
			return nil, err
		}

		log.Println(len(metadataSpans))

		for _, mspan := range metadataSpans {
			spans = append(spans, mspan)
		}
	}

	return mapRootSpansFromRows(ctx, spans, true)
}

type IODynamicRef struct {
	OutputRef string
	InputRef  string
}

type spanRollupInfo struct {
	metadataByParent map[string][]*cqrs.SpanMetadata
	dynamicRefs      map[string]*IODynamicRef
}

func mapSpanFromRow[T normalizedSpan](ctx context.Context, span T, info *spanRollupInfo) (*cqrs.OtelSpan, error) {
	// Use interface methods to get the fields directly
	traceID := span.GetTraceID()
	runIDStr := span.GetRunID()
	dynamicSpanID := span.GetDynamicSpanID()
	parentSpanID := span.GetParentSpanID()
	startTime := span.GetStartTime()
	endTime := span.GetEndTime()
	spanFragments := span.GetSpanFragments()

	var parsedStartTime time.Time
	switch v := startTime.(type) {
	case time.Time:
		parsedStartTime = v
	case string:
		st := strings.Split(v, " m=")[0]
		parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", st)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing start time", "error", err)
			return nil, err
		}
		parsedStartTime = parsed
	default:
		return nil, fmt.Errorf("unexpected start time type: %T", startTime)
	}

	var parsedEndTime time.Time
	switch v := endTime.(type) {
	case time.Time:
		parsedEndTime = v
	case string:
		et := strings.Split(v, " m=")[0]
		parsed, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", et)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing end time", "error", err)
			return nil, err
		}
		parsedEndTime = parsed
	default:
		return nil, fmt.Errorf("unexpected end time type: %T", endTime)
	}

	var parentSpanIDPtr *string
	if parentSpanID.Valid {
		parentSpanIDPtr = &parentSpanID.String
	}

	runID, err := ulid.Parse(runIDStr)
	if err != nil {
		logger.StdlibLogger(ctx).Error("error parsing run ID from span", "error", err)
		return nil, err
	}

	newSpan := &cqrs.OtelSpan{
		RawOtelSpan: cqrs.RawOtelSpan{
			SpanID:       dynamicSpanID.String,
			TraceID:      traceID,
			ParentSpanID: parentSpanIDPtr,
			StartTime:    parsedStartTime,
			// NOTE:
			// The end time is only valid if this span denotes a step end, or the run end.
			// EG. if this is an "Executor.run" span, this would never have an end time.
			// However, this is the actual span commit time.  We must handle this when we
			// parse the spans.
			EndTime:    parsedEndTime,
			Name:       "",
			Attributes: make(map[string]any),
		},
		Status:          enums.StepStatusRunning,
		RunID:           runID,
		MarkedAsDropped: false,
	}

	var (
		outputSpanID *string
		inputSpanID  *string
		fragments    []map[string]any

		isMetadata bool
	)

	var spanFragmentsBytes []byte
	switch v := spanFragments.(type) {
	case string:
		spanFragmentsBytes = []byte(v)
	case json.RawMessage:
		spanFragmentsBytes = []byte(v)
	case []byte:
		spanFragmentsBytes = v
	default:
		return nil, fmt.Errorf("unexpected span fragments type: %T", spanFragments)
	}

	_ = json.Unmarshal(spanFragmentsBytes, &fragments)

fragmentLoop:
	for _, fragment := range fragments {
		if name, ok := fragment["name"].(string); ok {
			switch {
			case strings.HasPrefix(name, "executor."):
				newSpan.Name = name
			case name == meta.SpanNameMetadata:
				newSpan.Name = name
				isMetadata = true
				break fragmentLoop
			}
		}

		if attrs, ok := fragment["attributes"].(string); ok {
			fragmentAttr := map[string]any{}
			if err := json.Unmarshal([]byte(attrs), &fragmentAttr); err != nil {
				logger.StdlibLogger(ctx).Error("error unmarshalling span attributes", "error", err)
				return nil, err
			}

			maps.Copy(newSpan.RawOtelSpan.Attributes, fragmentAttr)

			if outputRef, ok := fragment["output_span_id"].(string); ok && info != nil {
				outputSpanID = &outputRef
				if io, ok := info.dynamicRefs[dynamicSpanID.String]; ok && io != nil {
					io.OutputRef = outputRef
				} else {
					info.dynamicRefs[dynamicSpanID.String] = &IODynamicRef{OutputRef: outputRef}
				}
			}

			if inputRef, ok := fragment["input_span_id"].(string); ok && info != nil {
				inputSpanID = &inputRef
				if io, ok := info.dynamicRefs[dynamicSpanID.String]; ok && io != nil {
					io.InputRef = inputRef
				} else {
					info.dynamicRefs[dynamicSpanID.String] = &IODynamicRef{InputRef: inputRef}
				}
			}
		}
	}

	if info != nil && isMetadata && parentSpanIDPtr != nil {
		metadata, err := rollupSpanMetadataFromFragments(ctx, fragments, parsedEndTime)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error rolling up metadata span", "error", err)
		} else {
			info.metadataByParent[*parentSpanIDPtr] = append(info.metadataByParent[*parentSpanIDPtr], metadata)
		}
	}

	newSpan.Attributes, err = meta.ExtractTypedValues(ctx, newSpan.RawOtelSpan.Attributes)
	if err != nil {
		return nil, fmt.Errorf("error extracting typed values from span attributes: %w", err)
	}

	if newSpan.Attributes.DynamicStatus != nil {
		newSpan.Status = *newSpan.Attributes.DynamicStatus
	}

	if newSpan.Attributes.AppID != nil {
		newSpan.AppID = *newSpan.Attributes.AppID
	}

	if newSpan.Attributes.FunctionID != nil {
		newSpan.FunctionID = *newSpan.Attributes.FunctionID
	}

	if newSpan.Attributes.RunID != nil {
		newSpan.RunID = *newSpan.Attributes.RunID
	}

	if newSpan.Attributes.DebugRunID != nil {
		newSpan.DebugRunID = *newSpan.Attributes.DebugRunID
	}

	if newSpan.Attributes.DebugSessionID != nil {
		newSpan.DebugSessionID = *newSpan.Attributes.DebugSessionID
	}

	if newSpan.Attributes.StartedAt != nil {
		newSpan.StartTime = *newSpan.Attributes.StartedAt
	}

	if newSpan.Attributes.EndedAt != nil {
		newSpan.EndTime = *newSpan.Attributes.EndedAt
	}

	if newSpan.Attributes.DropSpan != nil && *newSpan.Attributes.DropSpan {
		newSpan.MarkedAsDropped = true
	}

	// If this span has finished, set a preliminary output ID.
	if (outputSpanID != nil && *outputSpanID != "") || (inputSpanID != nil && *inputSpanID != "") {
		newSpan.OutputID, err = encodeSpanOutputID(outputSpanID, inputSpanID)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error encoding span identifier", "error", err)
			return nil, err
		}
	}

	return newSpan, nil
}

// Uses generics to accept slices of any type that implements normalizedSpan interface
// implicitRoot causes the first span to be used as the root if no explicit root is found
func mapRootSpansFromRows[T normalizedSpan](ctx context.Context, spans []T, implicitRoot bool) (*cqrs.OtelSpan, error) {
	// ordered map is required by subsequent gql mapping
	spanMap := orderedmap.NewOrderedMap[string, *cqrs.OtelSpan]()

	metadataByParent := make(map[string][]*cqrs.SpanMetadata)

	// A map of dynamic span IDs to the specific span ID that contains I/O
	dynamicRefs := make(map[string]*IODynamicRef)

	var root *cqrs.OtelSpan
	var runID ulid.ULID
	var err error

	for _, span := range spans {
		info := spanRollupInfo{
			dynamicRefs:      dynamicRefs,
			metadataByParent: metadataByParent,
		}
		newSpan, err := mapSpanFromRow(ctx, span, &info)
		if err != nil {
			return nil, err
		}

		if newSpan.Name == meta.SpanNameMetadata {
			continue
		}

		spanMap.Set(newSpan.SpanID, newSpan)
	}

	// Build a reverse lookup map for output references
	outputDynamicRefs := make(map[string]*string)
	for spanID, ioRef := range dynamicRefs {
		if ioRef != nil && ioRef.OutputRef != "" {
			outputDynamicRefs[ioRef.OutputRef] = &spanID
		}
	}

	for _, span := range spanMap.AllFromFront() {
		// If we have an output reference for this span, set the appropriate
		// target span ID here
		if spanRefStr := span.Attributes.StepOutputRef; spanRefStr != nil && *spanRefStr != "" {
			if targetSpanID, ok := outputDynamicRefs[*spanRefStr]; ok {
				// We've found the span ID that we need to target for
				// this span. So let's use it!
				span.OutputID, err = encodeSpanOutputID(targetSpanID, nil)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error encoding span output ID", "error", err)
					return nil, err
				}
			}
		}

		if metadata, ok := metadataByParent[span.SpanID]; ok {
			span.Metadata = metadata
		}

		if (span.Attributes.IsUserland == nil || !*span.Attributes.IsUserland) && (span.ParentSpanID == nil || *span.ParentSpanID == "" || *span.ParentSpanID == "0000000000000000") || (implicitRoot && root == nil) {
			root, _ = spanMap.Get(span.SpanID)
			continue
		}

		if parent, ok := spanMap.Get(*span.ParentSpanID); ok {
			// This is wrong. Either do it properly in DB or infer it
			// correctly here. e.g. if child failed but more attempts coming,
			// still running
			if span.Status != enums.StepStatusUnknown && span.Status != enums.StepStatusRunning && (parent.Status == enums.StepStatusUnknown || parent.Status == enums.StepStatusRunning) {
				parent.Status = span.Status
			}

			item, _ := spanMap.Get(span.SpanID)
			parent.Children = append(parent.Children, item)
		} else {
			logger.StdlibLogger(ctx).Warn(
				"lost lineage detected",
				"spanID", span.SpanID,
				"parentSpanID", span.ParentSpanID,
			)
		}
	}

	if root == nil {
		return nil, fmt.Errorf("no root span found for run %s", runID.String())
	}

	sorter(root)

	return root, nil
}

func rollupSpanMetadataFromFragments(ctx context.Context, fragments []map[string]any, updatedAt time.Time) (*cqrs.SpanMetadata, error) {
	ret := &cqrs.SpanMetadata{
		Values:    metadata.Values{},
		UpdatedAt: updatedAt,
	}

	for _, fragment := range fragments {
		attrs, ok := fragment["attributes"].(string)
		if !ok {
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span kind, no attributes")
			continue
		}

		var fragmentAttr struct {
			Scope  *metadata.Scope  `json:"_inngest.metadata.scope"`
			Kind   *metadata.Kind   `json:"_inngest.metadata.kind"`
			Op     *metadata.Opcode `json:"_inngest.metadata.op"`
			Values *string          `json:"_inngest.metadata.values"`
		}
		if err := json.Unmarshal([]byte(attrs), &fragmentAttr); err != nil {
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span attributes", "error", err)
			return nil, err
		}

		switch {
		case fragmentAttr.Scope == nil:
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span kind")
			continue // TODO: err
		case fragmentAttr.Kind == nil:
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span kind")
			continue // TODO: err
		case fragmentAttr.Op == nil:
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span op")
			continue
		case fragmentAttr.Values == nil:
			logger.StdlibLogger(ctx).Error("error unmarshalling metadata span metadata")
			continue
		}

		if ret.Kind == "" && *fragmentAttr.Kind != "" {
			ret.Kind = *fragmentAttr.Kind
		} else if ret.Kind != *fragmentAttr.Kind {
			logger.StdlibLogger(ctx).Warn(
				"mismatch in metadata kind during rollup, skipping",
				"kinds", []metadata.Kind{ret.Kind, *fragmentAttr.Kind},
			)
			continue
		}

		if ret.Scope == enums.MetadataScopeUnknown && *fragmentAttr.Scope != enums.MetadataScopeUnknown {
			ret.Scope = *fragmentAttr.Scope
		} else if ret.Scope != *fragmentAttr.Scope {
			logger.StdlibLogger(ctx).Warn(
				"mismatch in metadata scope during rollup, skipping",
				"scopes", []metadata.Scope{ret.Scope, *fragmentAttr.Scope},
			)
			continue
		}

		var fragmentMetadata metadata.Values
		err := json.Unmarshal([]byte(*fragmentAttr.Values), &fragmentMetadata)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error unmarshalling span metadata", "error", err)
			return nil, err
		}

		err = ret.Values.Combine(fragmentMetadata, *fragmentAttr.Op)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error rolling up metadata span metadata", "error", err)
			return nil, err
		}
	}

	return ret, nil
}

func encodeSpanOutputID(outputSpanID *string, inputSpanID *string) (*string, error) {
	p := true
	osid := ""
	if outputSpanID != nil {
		osid = *outputSpanID
	}

	id := &cqrs.SpanIdentifier{
		SpanID:      osid,
		InputSpanID: inputSpanID,
		Preview:     &p,
	}

	encoded, err := id.Encode()
	if err != nil {
		return nil, err
	}

	return &encoded, nil
}

// group by run id, sort by started at, let the frontend handle overlay.
func buildDebugRunSpan[T normalizedSpan](ctx context.Context, spans []T) ([]*cqrs.OtelSpan, error) {
	if len(spans) == 0 {
		return nil, nil
	}

	spansByRunID := make(map[string][]T)
	for _, span := range spans {
		runID := span.GetRunID()
		spansByRunID[runID] = append(spansByRunID[runID], span)
	}

	runSpans := make([]*cqrs.OtelSpan, 0, len(spansByRunID))
	for _, runSpansGroup := range spansByRunID {
		runSpan, err := mapRootSpansFromRows(ctx, runSpansGroup, false)
		if err != nil {
			return nil, err
		}
		if runSpan != nil {
			runSpans = append(runSpans, runSpan)
		}
	}

	if len(runSpans) == 0 {
		return nil, nil
	}

	return runSpans, nil
}

func sorter(span *cqrs.OtelSpan) {
	sort.Slice(span.Children, func(i, j int) bool {
		if !span.Children[i].StartTime.Equal(span.Children[j].StartTime) {
			return span.Children[i].StartTime.Before(span.Children[j].StartTime)
		}

		// sort based on SpanID if two spans have equal timestamps
		return span.Children[i].SpanID < span.Children[j].SpanID
	})

	slices.SortFunc(span.Metadata, func(a, b *cqrs.SpanMetadata) int {
		return cmp.Or(
			cmp.Compare(a.Scope, b.Scope),
			cmp.Compare(a.Kind, b.Kind))
	})

	for _, child := range span.Children {
		sorter(child)
	}
}

// LoadFunction implements the state.FunctionLoader interface.
func (w wrapper) LoadFunction(ctx context.Context, envID, fnID uuid.UUID) (*state.ExecutorFunction, error) {
	// XXX: This doesn't store versions, as the dev server is currently ignorant to version.s
	fn, err := w.GetFunctionByInternalUUID(ctx, fnID)
	if err != nil {
		return nil, err
	}
	def, err := fn.InngestFunction()
	if err != nil {
		return nil, err
	}

	return &state.ExecutorFunction{
		Function: def,
		Paused:   false, // dev server does not support pausing
	}, nil
}

func (w wrapper) WithTx(ctx context.Context) (cqrs.TxManager, error) {
	if w.tx != nil {
		// Already in a tx else DB would be present.
		return w, nil
	}
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	var q sqlc.Querier
	if w.isPostgres() {
		q = sqlc_postgres.NewNormalized(tx, w.opts)
	} else {
		q = sqlc.New(tx)
	}

	return &wrapper{
		driver: w.driver,
		q:      q,
		tx:     tx,
		opts:   w.opts,
	}, nil
}

func (w wrapper) Commit(ctx context.Context) error {
	return w.tx.Commit()
}

func (w wrapper) Rollback(ctx context.Context) error {
	return w.tx.Rollback()
}

func (w wrapper) GetLatestQueueSnapshot(ctx context.Context) (*cqrs.QueueSnapshot, error) {
	chunks, err := w.q.GetLatestQueueSnapshotChunks(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting queue snapshot: %w", err)
	}

	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk.Data...)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var snapshot cqrs.QueueSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue snapshot: %w", err)
	}

	return &snapshot, nil
}

func (w wrapper) GetQueueSnapshot(ctx context.Context, snapshotID cqrs.SnapshotID) (*cqrs.QueueSnapshot, error) {
	chunks, err := w.q.GetQueueSnapshotChunks(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("error getting queue snapshot: %w", err)
	}

	var data []byte
	for _, chunk := range chunks {
		data = append(data, chunk.Data...)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var snapshot cqrs.QueueSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("error unmarshalling queue snapshot: %w", err)
	}

	return &snapshot, nil
}

func (w wrapper) InsertQueueSnapshot(ctx context.Context, params cqrs.InsertQueueSnapshotParams) (cqrs.SnapshotID, error) {
	var snapshotID cqrs.SnapshotID

	byt, err := json.Marshal(params.Snapshot)
	if err != nil {
		return snapshotID, fmt.Errorf("error marshalling snapshot: %w", err)
	}

	var chunks [][]byte
	for len(byt) > 0 {
		if len(byt) > consts.StartMaxQueueChunkSize {
			chunks = append(chunks, byt[:consts.StartMaxQueueChunkSize])
			byt = byt[consts.StartMaxQueueChunkSize:]
		} else {
			chunks = append(chunks, byt)
			break
		}
	}

	snapshotID = ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)

	tx, err := w.WithTx(ctx)
	if err != nil {
		return snapshotID, fmt.Errorf("error starting transaction: %w", err)
	}

	// Insert each chunk of the snapshot.
	for i, chunk := range chunks {
		err = tx.InsertQueueSnapshotChunk(ctx, cqrs.InsertQueueSnapshotChunkParams{
			SnapshotID: snapshotID,
			ChunkID:    i,
			Chunk:      chunk,
		})
		if err != nil {
			return snapshotID, fmt.Errorf("error inserting queue snapshot chunk: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return snapshotID, fmt.Errorf("error committing transaction: %w", err)
	}

	// Asynchronously remove old snapshots.
	go func() {
		_, _ = w.q.DeleteOldQueueSnapshots(ctx, consts.StartMaxQueueSnapshots)
	}()

	return snapshotID, nil
}

func (w wrapper) InsertQueueSnapshotChunk(ctx context.Context, params cqrs.InsertQueueSnapshotChunkParams) error {
	err := w.q.InsertQueueSnapshotChunk(ctx, sqlc.InsertQueueSnapshotChunkParams{
		SnapshotID: params.SnapshotID.String(),
		ChunkID:    int64(params.ChunkID),
		Data:       params.Chunk,
	})
	if err != nil {
		return fmt.Errorf("error inserting queue snapshot chunk: %w", err)
	}

	return nil
}

//
// Apps
//

// GetApps returns apps that have not been deleted.
func (w wrapper) GetApps(ctx context.Context, envID uuid.UUID, filter *cqrs.FilterAppParam) ([]*cqrs.App, error) {
	data, err := w.q.GetApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get apps: %w", err)
	}
	if filter == nil {
		return SQLiteToCQRSList(data, sqliteApp), nil
	}

	filtered := []*cqrs.App{}
	for _, app := range data {
		if filter.Method != nil && filter.Method.String() != app.Method {
			continue
		}
		filtered = append(filtered, SQLiteToCQRS(app, sqliteApp))
	}

	return filtered, nil
}

func (w wrapper) GetAppByChecksum(ctx context.Context, envID uuid.UUID, checksum string) (*cqrs.App, error) {
	app, err := w.q.GetAppByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

func (w wrapper) GetAppByID(ctx context.Context, id uuid.UUID) (*cqrs.App, error) {
	app, err := w.q.GetAppByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

func (w wrapper) GetAppByURL(ctx context.Context, envID uuid.UUID, url string) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	url = util.NormalizeAppURL(url, forceHTTPS)

	app, err := w.q.GetAppByURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

func (w wrapper) GetAppByName(ctx context.Context, envID uuid.UUID, name string) (*cqrs.App, error) {
	app, err := w.q.GetAppByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

// GetAllApps returns all apps.
func (w wrapper) GetAllApps(ctx context.Context, envID uuid.UUID) ([]*cqrs.App, error) {
	apps, err := w.q.GetAllApps(ctx)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRSList(apps, sqliteApp), nil
}

// InsertApp creates a new app.
func (w wrapper) UpsertApp(ctx context.Context, arg cqrs.UpsertAppParams) (*cqrs.App, error) {
	// Normalize the URL before inserting into the DB.
	arg.Url = util.NormalizeAppURL(arg.Url, forceHTTPS)

	if arg.Method == "" {
		arg.Method = enums.AppMethodServe.String()
	}

	app, err := w.q.UpsertApp(ctx, sqlc.UpsertAppParams{
		ID:          arg.ID,
		Name:        arg.Name,
		SdkLanguage: arg.SdkLanguage,
		SdkVersion:  arg.SdkVersion,
		Framework:   arg.Framework,
		Metadata:    arg.Metadata,
		Status:      arg.Status,
		Error:       arg.Error,
		Checksum:    arg.Checksum,
		Url:         arg.Url,
		Method:      arg.Method,
		AppVersion:  sql.NullString{String: arg.AppVersion, Valid: arg.AppVersion != ""},
	})
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(app, sqliteApp), nil
}

func (w wrapper) UpdateAppError(ctx context.Context, arg cqrs.UpdateAppErrorParams) (*cqrs.App, error) {
	// Use the direct SQL UPDATE query instead of load-then-upsert
	app, err := w.q.UpdateAppError(ctx, sqlc.UpdateAppErrorParams{
		ID:    arg.ID,
		Error: arg.Error,
	})
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

func (w wrapper) UpdateAppURL(ctx context.Context, arg cqrs.UpdateAppURLParams) (*cqrs.App, error) {
	// Normalize the URL before updating in the DB.
	arg.Url = util.NormalizeAppURL(arg.Url, forceHTTPS)

	// Use the direct SQL UPDATE query instead of delete-and-reinsert
	app, err := w.q.UpdateAppURL(ctx, sqlc.UpdateAppURLParams{
		ID:  arg.ID,
		Url: arg.Url,
	})
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRS(app, sqliteApp), nil
}

// DeleteApp deletes an app
func (w wrapper) DeleteApp(ctx context.Context, id uuid.UUID) error {
	return w.q.DeleteApp(ctx, id)
}

//
// Functions
//

func (w wrapper) GetAppFunctions(ctx context.Context, appID uuid.UUID) ([]*cqrs.Function, error) {
	fns, err := w.q.GetAppFunctions(ctx, appID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(fns, sqliteFunction), nil
}

func (w wrapper) GetFunctionByExternalID(ctx context.Context, wsID uuid.UUID, appID, fnSlug string) (*cqrs.Function, error) {
	fn, err := w.q.GetFunctionBySlug(ctx, fnSlug)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(fn, sqliteFunction), nil
}

func (w wrapper) GetFunctionByInternalUUID(ctx context.Context, fnID uuid.UUID) (*cqrs.Function, error) {
	fn, err := w.q.GetFunctionByID(ctx, fnID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(fn, sqliteFunction), nil
}

func (w wrapper) GetFunctions(ctx context.Context) ([]*cqrs.Function, error) {
	fns, err := w.q.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(fns, sqliteFunction), nil
}

func (w wrapper) GetFunctionsByAppInternalID(ctx context.Context, appID uuid.UUID) ([]*cqrs.Function, error) {
	fns, err := w.q.GetAppFunctions(ctx, appID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(fns, sqliteFunction), nil
}

func (w wrapper) GetFunctionsByAppExternalID(ctx context.Context, workspaceID uuid.UUID, appID string) ([]*cqrs.Function, error) {
	// Ingore the workspace ID for now.
	fns, err := w.q.GetAppFunctionsBySlug(ctx, appID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(fns, sqliteFunction), nil
}

func (w wrapper) InsertFunction(ctx context.Context, params cqrs.InsertFunctionParams) (*cqrs.Function, error) {
	fn, err := w.q.InsertFunction(ctx, sqlc.InsertFunctionParams{
		ID:        params.ID,
		AppID:     params.AppID,
		Name:      params.Name,
		Slug:      params.Slug,
		Config:    params.Config,
		CreatedAt: params.CreatedAt,
	})
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(fn, sqliteFunction), nil
}

func (w wrapper) DeleteFunctionsByAppID(ctx context.Context, appID uuid.UUID) error {
	return w.q.DeleteFunctionsByAppID(ctx, appID)
}

func (w wrapper) DeleteFunctionsByIDs(ctx context.Context, ids []uuid.UUID) error {
	return w.q.DeleteFunctionsByIDs(ctx, ids)
}

func (w wrapper) UpdateFunctionConfig(ctx context.Context, arg cqrs.UpdateFunctionConfigParams) (*cqrs.Function, error) {
	fn, err := w.q.UpdateFunctionConfig(ctx, sqlc.UpdateFunctionConfigParams{
		ID:     arg.ID,
		Config: arg.Config,
	})
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(fn, sqliteFunction), nil
}

//
// Events
//

func (w wrapper) InsertEvent(ctx context.Context, e cqrs.Event) error {
	data, err := json.Marshal(e.EventData)
	if err != nil {
		return err
	}
	user, err := json.Marshal(e.EventUser)
	if err != nil {
		return err
	}
	evt := sqlc.InsertEventParams{
		InternalID: e.ID,
		ReceivedAt: time.Now(),
		EventID:    e.EventID,
		EventName:  e.EventName,
		EventData:  string(data),
		EventUser:  string(user),
		EventV: sql.NullString{
			Valid:  e.EventVersion != "",
			String: e.EventVersion,
		},
		EventTs: time.UnixMilli(e.EventTS),
	}
	return w.q.InsertEvent(ctx, evt)
}

func (w wrapper) InsertEventBatch(ctx context.Context, eb cqrs.EventBatch) error {
	evtIDs := make([]string, len(eb.Events))
	for i, evt := range eb.Events {
		evtIDs[i] = evt.GetInternalID().String()
	}

	batch := sqlc.InsertEventBatchParams{
		ID:          eb.ID,
		AccountID:   eb.AccountID,
		WorkspaceID: eb.WorkspaceID,
		AppID:       eb.AppID,
		WorkflowID:  eb.FunctionID,
		RunID:       eb.RunID,
		StartedAt:   eb.StartedAt(),
		ExecutedAt:  eb.ExecutedAt(),
		EventIds:    []byte(strings.Join(evtIDs, ",")),
	}

	return w.q.InsertEventBatch(ctx, batch)
}

func (w wrapper) GetEventByInternalID(ctx context.Context, internalID ulid.ULID) (*cqrs.Event, error) {
	obj, err := w.q.GetEventByInternalID(ctx, internalID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(obj, sqliteEvent), nil
}

func (w wrapper) GetEventBatchesByEventID(ctx context.Context, eventID ulid.ULID) ([]*cqrs.EventBatch, error) {
	batches, err := w.q.GetEventBatchesByEventID(ctx, eventID.String())
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(batches, sqliteEventBatch), nil
}

func (w wrapper) GetEventBatchByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.EventBatch, error) {
	obj, err := w.q.GetEventBatchByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRS(obj, sqliteEventBatch), nil
}

func (w wrapper) GetEventsByInternalIDs(ctx context.Context, ids []ulid.ULID) ([]*cqrs.Event, error) {
	objs, err := w.q.GetEventsByInternalIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	return SQLiteToCQRSList(objs, sqliteEvent), nil
}

func (w wrapper) GetEventsByExpressions(ctx context.Context, cel []string) ([]*cqrs.Event, error) {
	expHandler, err := run.NewExpressionHandler(ctx,
		run.WithExpressionHandlerExpressions(cel),
	)
	if err != nil {
		return nil, err
	}
	prefilters, err := expHandler.ToSQLFilters(ctx)
	if err != nil {
		return nil, err
	}

	sql, args, err := sq.Dialect(w.dialect()).
		From("events").
		Select(
			"internal_id",
			"account_id",
			"workspace_id",
			"source",
			"source_id",
			"received_at",
			"event_id",
			"event_name",
			"event_data",
			"event_user",
			"event_v",
			"event_ts",
		).
		Where(prefilters...).
		Order(sq.C("received_at").Desc()).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.Event{}
	for rows.Next() {
		data := sqlc.Event{}
		if err := rows.Scan(
			&data.InternalID,
			&data.AccountID,
			&data.WorkspaceID,
			&data.Source,
			&data.SourceID,
			&data.ReceivedAt,
			&data.EventID,
			&data.EventName,
			&data.EventData,
			&data.EventUser,
			&data.EventV,
			&data.EventTs,
		); err != nil {
			return nil, err
		}

		evt, err := data.ToCQRS()
		if err != nil {
			return nil, fmt.Errorf("error deserializing event: %w", err)
		}

		ok, err := expHandler.MatchEventExpressions(ctx, evt.GetEvent())
		if err != nil {
			return nil, err
		}
		if ok {
			res = append(res, evt)
		}
	}

	return res, nil
}

func (w wrapper) GetEvent(ctx context.Context, internalID ulid.ULID, accountID uuid.UUID, workspaceID uuid.UUID) (*cqrs.Event, error) {
	return w.GetEventByInternalID(ctx, internalID)
}

func (w wrapper) GetEvents(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts *cqrs.WorkspaceEventsOpts) ([]*cqrs.Event, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	if opts.Cursor == nil {
		opts.Cursor = &endULID
	}

	builder := newEventsQueryBuilder(ctx, *opts)
	filter := builder.filter
	order := builder.order

	sql, args, err := sq.Dialect(w.dialect()).
		From("events").
		Select(
			"internal_id",
			"account_id",
			"workspace_id",
			"source",
			"source_id",
			"received_at",
			"event_id",
			"event_name",
			"event_data",
			"event_user",
			"event_v",
			"event_ts",
		).
		Where(filter...).
		Order(order...).
		Limit(uint(opts.Limit)).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*cqrs.Event, 0, opts.Limit)
	for rows.Next() {
		data := sqlc.Event{}
		if err := rows.Scan(
			&data.InternalID,
			&data.AccountID,
			&data.WorkspaceID,
			&data.Source,
			&data.SourceID,
			&data.ReceivedAt,
			&data.EventID,
			&data.EventName,
			&data.EventData,
			&data.EventUser,
			&data.EventV,
			&data.EventTs,
		); err != nil {
			return nil, err
		}
		out = append(out, SQLiteToCQRS(&data, sqliteEvent))
	}

	return out, nil
}

func (w wrapper) GetEventsCount(ctx context.Context, accountID uuid.UUID, workspaceID uuid.UUID, opts cqrs.WorkspaceEventsOpts) (int64, error) {
	if err := opts.Validate(); err != nil {
		return 0, err
	}

	// We don't want to consider cursor pagination for total count, so overwrite input param
	opts.Cursor = &endULID

	builder := newEventsQueryBuilder(ctx, opts)
	filter := builder.filter
	// ignore builder.order for count queries, it will error on Postgres

	sql, args, err := sq.Dialect(w.dialect()).
		From("events").
		Select(sq.COUNT("*").As("count")).
		Where(filter...).
		ToSQL()
	if err != nil {
		return 0, err
	}

	var count int64
	err = w.db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

type eventsQueryBuilder struct {
	filter []sq.Expression
	order  []sqexp.OrderedExpression
}

func newEventsQueryBuilder(ctx context.Context, opt cqrs.WorkspaceEventsOpts) *eventsQueryBuilder {
	filter := []sq.Expression{}

	filter = append(filter, sq.C("received_at").Lte(opt.Newest))
	filter = append(filter, sq.C("received_at").Gte(opt.Oldest))
	if opt.Cursor != nil {
		filter = append(filter, sq.C("internal_id").Lt(*opt.Cursor))
	}

	if len(opt.Names) > 0 {
		filter = append(filter, sq.C("event_name").In(opt.Names))
	}
	if !opt.IncludeInternalEvents {
		filter = append(filter, sq.C("event_name").NotLike("inngest/%"))
	}

	order := []sqexp.OrderedExpression{}
	order = append(order, sq.C("internal_id").Desc())

	return &eventsQueryBuilder{
		filter: filter,
		order:  order,
	}
}

func (w wrapper) GetEventsIDbound(
	ctx context.Context,
	ids cqrs.IDBound,
	limit int,
	includeInternal bool,
) ([]*cqrs.Event, error) {
	if ids.Before == nil {
		ids.Before = &endULID
	}

	if ids.After == nil {
		ids.After = &ulid.Zero
	}

	evts, err := w.q.GetEventsIDbound(ctx, sqlc.GetEventsIDboundParams{
		After:           *ids.After,
		Before:          *ids.Before,
		IncludeInternal: strconv.FormatBool(includeInternal),
		Limit:           int64(limit),
	})
	if err != nil {
		return []*cqrs.Event{}, err
	}

	return SQLiteToCQRSList(evts, sqliteEvent), nil
}

//
// Function runs
//

func (w wrapper) InsertFunctionRun(ctx context.Context, e cqrs.FunctionRun) error {
	run := sqlc.InsertFunctionRunParams{
		RunID:           e.RunID,
		RunStartedAt:    e.RunStartedAt,
		FunctionID:      e.FunctionID,
		FunctionVersion: e.FunctionVersion,
		TriggerType:     "event",
		EventID:         e.EventID,
		WorkspaceID:     e.WorkspaceID,
	}

	// Handle nullable fields
	if e.BatchID != nil {
		run.BatchID = *e.BatchID
	}
	if e.OriginalRunID != nil {
		run.OriginalRunID = *e.OriginalRunID
	}
	if e.Cron != nil {
		run.Cron = sql.NullString{
			Valid:  true,
			String: *e.Cron,
		}
	}

	return w.q.InsertFunctionRun(ctx, run)
}

func (w wrapper) GetFunctionRunsFromEvents(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	eventIDs []ulid.ULID,
) ([]*cqrs.FunctionRun, error) {
	runs, err := w.q.GetFunctionRunsFromEvents(ctx, eventIDs)
	if err != nil {
		return nil, err
	}
	result := []*cqrs.FunctionRun{}
	for _, item := range runs {
		result = append(result, toCQRSRun(item.FunctionRun, item.FunctionFinish))
	}
	return result, nil
}

func (w wrapper) GetFunctionRun(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	id ulid.ULID,
) (*cqrs.FunctionRun, error) {
	item, err := w.q.GetFunctionRun(ctx, id)
	if err != nil {
		return nil, err
	}
	return toCQRSRun(item.FunctionRun, item.FunctionFinish), nil
}

func (w wrapper) GetFunctionRunsTimebound(ctx context.Context, t cqrs.Timebound, limit int) ([]*cqrs.FunctionRun, error) {
	after := time.Time{}                           // after the beginning of time, eg all
	before := time.Now().Add(time.Hour * 24 * 365) // before 1 year in the future, eg all
	if t.After != nil {
		after = *t.After
	}
	if t.Before != nil {
		before = *t.Before
	}

	runs, err := w.q.GetFunctionRunsTimebound(ctx, sqlc.GetFunctionRunsTimeboundParams{
		Before: before,
		After:  after,
		Limit:  int64(limit),
	})
	if err != nil {
		return nil, err
	}
	result := []*cqrs.FunctionRun{}
	for _, item := range runs {
		result = append(result, toCQRSRun(item.FunctionRun, item.FunctionFinish))
	}
	return result, nil
}

func (w wrapper) GetFunctionRunFinishesByRunIDs(
	ctx context.Context,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
	runIDs []ulid.ULID,
) ([]*cqrs.FunctionRunFinish, error) {
	finish, err := w.q.GetFunctionRunFinishesByRunIDs(ctx, runIDs)
	if err != nil {
		return nil, err
	}
	return SQLiteToCQRSList(finish, sqliteFunctionFinish), nil
}

//
// History
//

func (w wrapper) InsertHistory(ctx context.Context, h history.History) error {
	params, err := convertHistoryToWriter(h)
	if err != nil {
		return err
	}
	return w.q.InsertHistory(ctx, *params)
}

func (w wrapper) GetFunctionRunHistory(ctx context.Context, runID ulid.ULID) ([]*history.History, error) {
	_, err := w.q.GetFunctionRunHistory(ctx, runID)
	// TODO: Convert history
	return nil, err
}

func toCQRSRun(run sqlc.FunctionRun, finish sqlc.FunctionFinish) *cqrs.FunctionRun {
	copied := cqrs.FunctionRun{
		RunID:           run.RunID,
		RunStartedAt:    run.RunStartedAt,
		FunctionID:      run.FunctionID,
		FunctionVersion: run.FunctionVersion,
		EventID:         run.EventID,
		WorkspaceID:     run.WorkspaceID,
	}
	if !run.BatchID.IsZero() {
		copied.BatchID = &run.BatchID
	}
	if !run.OriginalRunID.IsZero() {
		copied.OriginalRunID = &run.OriginalRunID
	}
	if run.Cron.Valid {
		copied.Cron = &run.Cron.String
	}
	if finish.Status.Valid {
		copied.Status, _ = enums.RunStatusString(finish.Status.String)
		copied.Output = util.EnsureJSON(json.RawMessage(finish.Output.String))
		copied.EndedAt = &finish.CreatedAt.Time
	}
	return &copied
}

//
// Trace
//

func (w wrapper) InsertSpan(ctx context.Context, span *cqrs.Span) error {
	params := &sqlc.InsertTraceParams{
		Timestamp:       span.Timestamp,
		TimestampUnixMs: span.Timestamp.UnixMilli(),
		TraceID:         span.TraceID,
		SpanID:          span.SpanID,
		SpanName:        span.SpanName,
		SpanKind:        span.SpanKind,
		ServiceName:     span.ServiceName,
		ScopeName:       span.ScopeName,
		ScopeVersion:    span.ScopeVersion,
		Duration:        int64(span.Duration / time.Millisecond),
		StatusCode:      span.StatusCode,
	}

	if span.RunID != nil {
		params.RunID = *span.RunID
	}
	if span.ParentSpanID != nil {
		params.ParentSpanID = sql.NullString{String: *span.ParentSpanID, Valid: true}
	}
	if span.TraceState != nil {
		params.TraceState = sql.NullString{String: *span.TraceState, Valid: true}
	}
	if byt, err := json.Marshal(span.ResourceAttributes); err == nil {
		params.ResourceAttributes = byt
	}
	if byt, err := json.Marshal(span.SpanAttributes); err == nil {
		params.SpanAttributes = byt
	}
	if byt, err := json.Marshal(span.Events); err == nil {
		params.Events = byt
	}
	if byt, err := json.Marshal(span.Links); err == nil {
		params.Links = byt
	}
	if span.StatusMessage != nil {
		params.StatusMessage = sql.NullString{String: *span.StatusMessage, Valid: true}
	}

	return w.q.InsertTrace(ctx, *params)
}

func (w wrapper) InsertTraceRun(ctx context.Context, run *cqrs.TraceRun) error {
	runid, err := ulid.Parse(run.RunID)
	if err != nil {
		return fmt.Errorf("error parsing runID as ULID: %w", err)
	}

	params := sqlc.InsertTraceRunParams{
		AccountID:   run.AccountID,
		WorkspaceID: run.WorkspaceID,
		AppID:       run.AppID,
		FunctionID:  run.FunctionID,
		TraceID:     []byte(run.TraceID),
		SourceID:    run.SourceID,
		RunID:       runid,
		QueuedAt:    run.QueuedAt.UnixMilli(),
		StartedAt:   run.StartedAt.UnixMilli(),
		EndedAt:     run.EndedAt.UnixMilli(),
		Status:      run.Status.ToCode(),
		TriggerIds:  []byte{},
		Output:      run.Output,
		IsDebounce:  run.IsDebounce,
		HasAi:       run.HasAI,
	}

	if run.BatchID != nil {
		params.BatchID = *run.BatchID
	}
	if run.CronSchedule != nil {
		params.CronSchedule = sql.NullString{String: *run.CronSchedule, Valid: true}
	}
	if len(run.TriggerIDs) > 0 {
		params.TriggerIds = []byte(strings.Join(run.TriggerIDs, ","))
	}

	return w.q.InsertTraceRun(ctx, params)
}

type traceRunCursorFilter struct {
	ID    string
	Value int64
}

func (w wrapper) GetTraceSpansByRun(ctx context.Context, id cqrs.TraceRunIdentifier) ([]*cqrs.Span, error) {
	spans, err := w.q.GetTraceSpans(ctx, sqlc.GetTraceSpansParams{
		TraceID: id.TraceID,
		RunID:   id.RunID,
	})
	if err != nil {
		return nil, err
	}

	res := []*cqrs.Span{}
	seen := map[string]bool{}
	for _, s := range spans {
		// identifier to used for checking if this span is seen already
		m := map[string]any{
			"ts":  s.Timestamp.UnixMilli(),
			"tid": s.TraceID,
			"sid": s.SpanID,
		}
		byt, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		ident := base64.StdEncoding.EncodeToString(byt)
		if _, ok := seen[ident]; ok {
			// already seen, so continue
			continue
		}

		span := &cqrs.Span{
			Timestamp:    s.Timestamp,
			TraceID:      string(s.TraceID),
			SpanID:       string(s.SpanID),
			SpanName:     s.SpanName,
			SpanKind:     s.SpanKind,
			ServiceName:  s.ServiceName,
			ScopeName:    s.ScopeName,
			ScopeVersion: s.ScopeVersion,
			Duration:     time.Duration(s.Duration * int64(time.Millisecond)),
			StatusCode:   s.StatusCode,
			RunID:        &s.RunID,
		}

		if s.StatusMessage.Valid {
			span.StatusMessage = &s.StatusMessage.String
		}

		if s.ParentSpanID.Valid {
			span.ParentSpanID = &s.ParentSpanID.String
		}
		if s.TraceState.Valid {
			span.TraceState = &s.TraceState.String
		}

		var resourceAttr, spanAttr map[string]string
		if err := json.Unmarshal(s.ResourceAttributes, &resourceAttr); err == nil {
			span.ResourceAttributes = resourceAttr
		}
		if err := json.Unmarshal(s.SpanAttributes, &spanAttr); err == nil {
			span.SpanAttributes = spanAttr
		}

		res = append(res, span)
		seen[ident] = true
	}

	return res, nil
}

func (w wrapper) FindOrBuildTraceRun(ctx context.Context, opts cqrs.FindOrCreateTraceRunOpt) (*cqrs.TraceRun, error) {
	run, err := w.GetTraceRun(ctx, cqrs.TraceRunIdentifier{RunID: opts.RunID})
	if err == nil {
		return run, nil
	}

	new := cqrs.TraceRun{
		AccountID:   opts.AccountID,
		WorkspaceID: opts.WorkspaceID,
		AppID:       opts.AppID,
		FunctionID:  opts.FunctionID,
		RunID:       opts.RunID.String(),
		TraceID:     opts.TraceID,
		QueuedAt:    ulid.Time(opts.RunID.Time()),
		TriggerIDs:  []string{},
		Status:      enums.RunStatusUnknown,
	}

	return &new, nil
}

func (w wrapper) GetTraceRunsByTriggerID(ctx context.Context, triggerID ulid.ULID) ([]*cqrs.TraceRun, error) {
	// convert sqlc.TraceRun{} to cqrs.TraceRun{}
	sqlcTraceRuns, err := w.q.GetTraceRunsByTriggerId(ctx, triggerID.String())
	if err != nil {
		return nil, err
	}
	cqrsTraceRuns := make([]*cqrs.TraceRun, len(sqlcTraceRuns))
	// dedupe this conversion
	for i, run := range sqlcTraceRuns {
		start := time.UnixMilli(run.StartedAt)
		end := time.UnixMilli(run.EndedAt)
		triggerIDS := strings.Split(string(run.TriggerIds), ",")

		var (
			isBatch bool
			batchID *ulid.ULID
			cron    *string
		)

		if !run.BatchID.IsZero() {
			isBatch = true
			batchID = &run.BatchID
		}

		if run.CronSchedule.Valid {
			cron = &run.CronSchedule.String
		}

		cqrsTraceRuns[i] = &cqrs.TraceRun{
			AccountID:    run.AccountID,
			WorkspaceID:  run.WorkspaceID,
			AppID:        run.AppID,
			FunctionID:   run.FunctionID,
			TraceID:      string(run.TraceID),
			RunID:        run.RunID.String(),
			QueuedAt:     time.UnixMilli(run.QueuedAt),
			StartedAt:    start,
			EndedAt:      end,
			Duration:     end.Sub(start),
			SourceID:     run.SourceID,
			TriggerIDs:   triggerIDS,
			Output:       run.Output,
			Status:       enums.RunCodeToStatus(run.Status),
			BatchID:      batchID,
			IsBatch:      isBatch,
			CronSchedule: cron,
			HasAI:        run.HasAi,
		}
	}
	return cqrsTraceRuns, nil
}

func (w wrapper) GetTraceRun(ctx context.Context, id cqrs.TraceRunIdentifier) (*cqrs.TraceRun, error) {
	run, err := w.q.GetTraceRun(ctx, id.RunID)
	if err != nil {
		return nil, err
	}

	start := time.UnixMilli(run.StartedAt)
	end := time.UnixMilli(run.EndedAt)
	triggerIDS := strings.Split(string(run.TriggerIds), ",")

	var (
		isBatch bool
		batchID *ulid.ULID
		cron    *string
	)

	if !run.BatchID.IsZero() {
		isBatch = true
		batchID = &run.BatchID
	}

	if run.CronSchedule.Valid {
		cron = &run.CronSchedule.String
	}

	trun := cqrs.TraceRun{
		AccountID:    run.AccountID,
		WorkspaceID:  run.WorkspaceID,
		AppID:        run.AppID,
		FunctionID:   run.FunctionID,
		TraceID:      string(run.TraceID),
		RunID:        id.RunID.String(),
		QueuedAt:     time.UnixMilli(run.QueuedAt),
		StartedAt:    start,
		EndedAt:      end,
		Duration:     end.Sub(start),
		SourceID:     run.SourceID,
		TriggerIDs:   triggerIDS,
		Output:       run.Output,
		Status:       enums.RunCodeToStatus(run.Status),
		BatchID:      batchID,
		IsBatch:      isBatch,
		CronSchedule: cron,
		HasAI:        run.HasAi,
	}

	return &trun, nil
}

func (w wrapper) GetSpanOutput(ctx context.Context, opts cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	ids := []string{}
	if opts.SpanID != "" {
		ids = append(ids, opts.SpanID)
	}
	if opts.InputSpanID != nil && *opts.InputSpanID != "" {
		ids = append(ids, *opts.InputSpanID)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("span ID or input span ID is required to retrieve output")
	}

	rows, err := w.q.GetSpanOutput(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("error retrieving span output: %w", err)
	}

	so := &cqrs.SpanOutput{}

	for _, row := range rows {
		if row.Input != nil {
			so.Input = []byte(fmt.Append(nil, row.Input))
		}

		if row.Output != nil {
			var m map[string]any

			so.Data = []byte(fmt.Append(nil, row.Output))
			if err := json.Unmarshal(so.Data, &m); err == nil && m != nil {
				// NOTE: By default, we wrap errors and data.  However, unforutnately
				// step.waitForEvent is _not_ wrapped, so we check to see if there's
				// both "data" and "name";  if so, we return the data wholesale.
				if isWaitForEventOutput(m) {
					return so, nil
				}

				if errData, ok := m["error"]; ok {
					so.IsError = true
					so.Data, _ = json.Marshal(errData)
				} else if successData, ok := m["data"]; ok {
					so.Data, _ = json.Marshal(successData)
				} else {
					sanitizedSpanID := strings.ReplaceAll(opts.SpanID, "\n", "")
					sanitizedSpanID = strings.ReplaceAll(sanitizedSpanID, "\r", "")

					logger.StdlibLogger(ctx).Error("span output is not keyed, assuming success", "spanID", sanitizedSpanID)
				}
			}
		}
	}

	return so, nil
}

func (w wrapper) LegacyGetSpanOutput(ctx context.Context, opts cqrs.SpanIdentifier) (*cqrs.SpanOutput, error) {
	if opts.TraceID == "" {
		return nil, fmt.Errorf("traceID is required to retrieve output")
	}
	if opts.SpanID == "" {
		return nil, fmt.Errorf("spanID is required to retrieve output")
	}

	// query spans in descending order
	spans, err := w.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{
		TraceID: opts.TraceID,
		SpanID:  opts.SpanID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving spans for output: %w", err)
	}

	for _, s := range spans {
		var evts []cqrs.SpanEvent
		err := json.Unmarshal(s.Events, &evts)
		if err != nil {
			return nil, fmt.Errorf("error parsing span outputs: %w", err)
		}

		var (
			input      []byte
			spanOutput *cqrs.SpanOutput
		)

		for _, evt := range evts {
			if spanOutput == nil {
				_, isFnOutput := evt.Attributes[consts.OtelSysFunctionOutput]
				_, isStepOutput := evt.Attributes[consts.OtelSysStepOutput]
				if isFnOutput || isStepOutput {
					var isError bool
					switch strings.ToUpper(s.StatusCode) {
					case "ERROR", "STATUS_CODE_ERROR":
						isError = true
					}

					spanOutput = &cqrs.SpanOutput{
						Data:             []byte(evt.Name),
						Timestamp:        evt.Timestamp,
						Attributes:       evt.Attributes,
						IsError:          isError,
						IsFunctionOutput: isFnOutput,
						IsStepOutput:     isStepOutput,
					}
				}
			}

			if _, isInput := evt.Attributes[consts.OtelSysStepInput]; isInput && input == nil {
				input = []byte(evt.Name)
			}

			if spanOutput != nil && input != nil {
				break
			}
		}

		if spanOutput != nil {
			spanOutput.Input = input
			return spanOutput, nil
		}
	}

	return nil, fmt.Errorf("no output found")
}

func (w wrapper) GetSpanStack(ctx context.Context, opts cqrs.SpanIdentifier) ([]string, error) {
	if opts.TraceID == "" {
		return nil, fmt.Errorf("traceID is required to retrieve stack")
	}
	if opts.SpanID == "" {
		return nil, fmt.Errorf("spanID is required to retrieve stack")
	}

	// query spans in descending order
	spans, err := w.q.GetTraceSpanOutput(ctx, sqlc.GetTraceSpanOutputParams{
		TraceID: opts.TraceID,
		SpanID:  opts.SpanID,
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving spans for stack: %w", err)
	}

	for _, s := range spans {
		var evts []cqrs.SpanEvent
		err := json.Unmarshal(s.Events, &evts)
		if err != nil {
			return nil, fmt.Errorf("error parsing span outputs: %w", err)
		}

		for _, evt := range evts {
			if _, isStackEvt := evt.Attributes[consts.OtelSysStepStack]; isStackEvt {
				// Data is kept in the `Name` field
				return strings.Split(evt.Name, ","), nil
			}
		}
	}

	return nil, fmt.Errorf("no stack found")
}

type runsQueryBuilder struct {
	filter       []sq.Expression
	order        []sqexp.OrderedExpression
	cursor       *cqrs.TracePageCursor
	cursorLayout *cqrs.TracePageCursor
}

func newRunsQueryBuilder(ctx context.Context, opt cqrs.GetTraceRunOpt) *runsQueryBuilder {
	l := logger.StdlibLogger(ctx)

	// filters
	filter := []sq.Expression{}
	if len(opt.Filter.AppID) > 0 {
		filter = append(filter, sq.C("app_id").In(opt.Filter.AppID))
	}
	if len(opt.Filter.FunctionID) > 0 {
		filter = append(filter, sq.C("function_id").In(opt.Filter.FunctionID))
	}
	if len(opt.Filter.Status) > 0 {
		status := []int64{}
		for _, s := range opt.Filter.Status {
			switch s {
			case enums.RunStatusUnknown, enums.RunStatusOverflowed:
				continue
			}
			status = append(status, s.ToCode())
		}
		filter = append(filter, sq.C("status").In(status))
	}
	// Skipped runs should only be visible in event-scoped queries, not the runs list
	filter = append(filter, sq.C("status").Neq(enums.RunStatusSkipped.ToCode()))
	tsfield := strings.ToLower(opt.Filter.TimeField.String())
	filter = append(filter, sq.C(tsfield).Gte(opt.Filter.From.UnixMilli()))

	until := opt.Filter.Until
	if until.UnixMilli() <= 0 {
		until = time.Now()
	}
	filter = append(filter, sq.C(tsfield).Lt(until.UnixMilli()))

	// Layout to be used for the response cursors
	resCursorLayout := cqrs.TracePageCursor{
		Cursors: map[string]cqrs.TraceCursor{},
	}

	reqcursor := &cqrs.TracePageCursor{}
	if opt.Cursor != "" {
		if err := reqcursor.Decode(opt.Cursor); err != nil {
			l.Error("error decoding function run cursor", "error", err, "cursor", opt.Cursor)
		}
	}

	// order by
	//
	// When going through the sorting fields, construct
	// - response pagination cursor layout
	// - update filter with op against sorted fields for pagination
	sortOrder := []enums.TraceRunTime{}
	sortDir := map[enums.TraceRunTime]enums.TraceRunOrder{}
	cursorFilter := map[enums.TraceRunTime]traceRunCursorFilter{}
	for _, f := range opt.Order {
		sortDir[f.Field] = f.Direction
		found := false
		for _, field := range sortOrder {
			if f.Field == field {
				found = true
				break
			}
		}
		if !found {
			sortOrder = append(sortOrder, f.Field)
		}

		rc := reqcursor.Find(f.Field.String())
		if rc != nil {
			cursorFilter[f.Field] = traceRunCursorFilter{ID: reqcursor.ID, Value: rc.Value}
		}
		resCursorLayout.Add(f.Field.String())
	}

	order := []sqexp.OrderedExpression{}
	for _, f := range sortOrder {
		var o sqexp.OrderedExpression
		field := strings.ToLower(f.String())
		if d, ok := sortDir[f]; ok {
			switch d {
			case enums.TraceRunOrderAsc:
				o = sq.C(field).Asc()
			case enums.TraceRunOrderDesc:
				o = sq.C(field).Desc()
			default:
				l.Error("invalid direction specified for sorting", "field", field, "direction", d.String())
				continue
			}

			order = append(order, o)
		}
	}
	order = append(order, sq.C("run_id").Asc())

	// cursor filter
	for k, cf := range cursorFilter {
		ord, ok := sortDir[k]
		if !ok {
			continue
		}

		var compare sq.Expression
		field := strings.ToLower(k.String())
		switch ord {
		case enums.TraceRunOrderAsc:
			compare = sq.C(field).Gt(cf.Value)
		case enums.TraceRunOrderDesc:
			compare = sq.C(field).Lt(cf.Value)
		default:
			continue
		}

		filter = append(filter, sq.Or(
			compare,
			sq.And(
				sq.C(field).Eq(cf.Value),
				sq.C("run_id").Gt(cf.ID),
			),
		))
	}

	return &runsQueryBuilder{
		filter:       filter,
		order:        order,
		cursor:       reqcursor,
		cursorLayout: &resCursorLayout,
	}
}

func (w wrapper) GetTraceRunsCount(ctx context.Context, opt cqrs.GetTraceRunOpt) (int, error) {
	// explicitly set it to zero so it would not attempt to paginate
	opt.Items = 0
	var (
		res []*cqrs.TraceRun
		err error
	)
	if opt.Preview {
		res, err = w.GetSpanRuns(ctx, opt)
	} else {
		res, err = w.GetTraceRuns(ctx, opt)
	}
	if err != nil {
		return 0, err
	}

	return len(res), nil
}

func (w wrapper) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	if opt.Preview {
		return w.GetSpanRuns(ctx, opt)
	}

	l := logger.StdlibLogger(ctx)

	// use evtIDs as post query filter
	evtIDs := []string{}
	expHandler, err := run.NewExpressionHandler(ctx,
		run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"),
	)
	if err != nil {
		return nil, err
	}
	if expHandler.HasEventFilters() {
		evts, err := w.GetEventsByExpressions(ctx, expHandler.EventExprList)
		if err != nil {
			return nil, err
		}
		for _, e := range evts {
			evtIDs = append(evtIDs, e.ID.String())
		}
	}

	builder := newRunsQueryBuilder(ctx, opt)
	filter := builder.filter
	order := builder.order
	reqcursor := builder.cursor
	resCursorLayout := builder.cursorLayout

	// read from database
	// TODO:
	// change this to a continuous loop with limits instead of just attempting to grab everything.
	// might not matter though since this is primarily meant for local
	// development
	sql, args, err := sq.Dialect(w.dialect()).
		From("trace_runs").
		Select(
			"app_id",
			"function_id",
			"trace_id",
			"run_id",
			"queued_at",
			"started_at",
			"ended_at",
			"status",
			"source_id",
			"trigger_ids",
			"output",
			"batch_id",
			"is_debounce",
			"cron_schedule",
			"has_ai",
		).
		Where(filter...).
		Order(order...).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.TraceRun{}
	var count uint
	for rows.Next() {
		data := sqlc.TraceRun{}
		err := rows.Scan(
			&data.AppID,
			&data.FunctionID,
			&data.TraceID,
			&data.RunID,
			&data.QueuedAt,
			&data.StartedAt,
			&data.EndedAt,
			&data.Status,
			&data.SourceID,
			&data.TriggerIds,
			&data.Output,
			&data.BatchID,
			&data.IsDebounce,
			&data.CronSchedule,
			&data.HasAi,
		)
		if err != nil {
			return nil, err
		}

		// filter out runs that doesn't have the event IDs
		if len(evtIDs) > 0 && !data.HasEventIDs(evtIDs) {
			continue
		}

		// the cursor target should be skipped
		if reqcursor.ID == data.RunID.String() {
			continue
		}

		if expHandler.HasOutputFilters() {
			ok, err := expHandler.MatchOutputExpressions(ctx, data.Output)
			if err != nil {
				l.Error("error inspecting run for output match",
					"error", err,
					"output", string(data.Output),
					"acctID", data.AccountID,
					"wsID", data.WorkspaceID,
					"appID", data.AppID,
					"wfID", data.FunctionID,
					"runID", data.RunID,
				)
				continue
			}
			if !ok {
				continue
			}
		}

		// copy layout
		pc := resCursorLayout
		// construct the needed fields to generate a cursor representing this run
		pc.ID = data.RunID.String()
		for k := range pc.Cursors {
			switch k {
			case strings.ToLower(enums.TraceRunTimeQueuedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.QueuedAt}
			case strings.ToLower(enums.TraceRunTimeStartedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.StartedAt}
			case strings.ToLower(enums.TraceRunTimeEndedAt.String()):
				pc.Cursors[k] = cqrs.TraceCursor{Field: k, Value: data.EndedAt}
			default:
				l.Warn("unknown field registered as cursor", "field", k)
				delete(pc.Cursors, k)
			}
		}

		cursor, err := pc.Encode()
		if err != nil {
			l.Error("error encoding cursor", "error", err, "page_cursor", pc)
		}
		var cron *string
		if data.CronSchedule.Valid {
			cron = &data.CronSchedule.String
		}
		var batchID *ulid.ULID
		isBatch := !data.BatchID.IsZero()
		if isBatch {
			batchID = &data.BatchID
		}

		res = append(res, &cqrs.TraceRun{
			AppID:        data.AppID,
			FunctionID:   data.FunctionID,
			TraceID:      string(data.TraceID),
			RunID:        data.RunID.String(),
			QueuedAt:     time.UnixMilli(data.QueuedAt),
			StartedAt:    time.UnixMilli(data.StartedAt),
			EndedAt:      time.UnixMilli(data.EndedAt),
			SourceID:     data.SourceID,
			TriggerIDs:   data.EventIDs(),
			Triggers:     [][]byte{},
			Output:       data.Output,
			Status:       enums.RunCodeToStatus(data.Status),
			IsBatch:      isBatch,
			BatchID:      batchID,
			IsDebounce:   data.IsDebounce,
			HasAI:        data.HasAi,
			CronSchedule: cron,
			Cursor:       cursor,
		})
		count++
		// enough items, don't need to proceed anymore
		if opt.Items > 0 && count >= opt.Items {
			break
		}
	}

	return res, nil
}

// OTel traces are hard coded to true for dev server until we move
// entitlements here.
func (w wrapper) OtelTracesEnabled(ctx context.Context, accountID uuid.UUID) (bool, error) {
	return true, nil
}

// For the API - CQRS return
func (w wrapper) GetEventRuns(
	ctx context.Context,
	eventID ulid.ULID,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
) ([]*cqrs.FunctionRun, error) {
	runs, err := w.q.GetFunctionRunsFromEvents(ctx, []ulid.ULID{eventID})
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

func (w wrapper) GetRun(
	ctx context.Context,
	runID ulid.ULID,
	accountID uuid.UUID,
	workspaceID uuid.UUID,
) (*cqrs.FunctionRun, error) {
	return w.GetFunctionRun(ctx, accountID, workspaceID, runID)
}

//
// Connect
//

func (w wrapper) InsertWorkerConnection(ctx context.Context, conn *cqrs.WorkerConnection) error {
	appVersion := sql.NullString{}
	if conn.AppVersion != nil {
		appVersion.Valid = true
		appVersion.String = *conn.AppVersion
	}

	var lastHeartbeatAt, disconnectedAt sql.NullInt64

	if conn.LastHeartbeatAt != nil {
		lastHeartbeatAt = sql.NullInt64{
			Int64: conn.LastHeartbeatAt.UnixMilli(),
			Valid: true,
		}
	}

	if conn.DisconnectedAt != nil {
		lastHeartbeatAt = sql.NullInt64{
			Int64: conn.DisconnectedAt.UnixMilli(),
			Valid: true,
		}
	}

	var disconnectReason sql.NullString
	if conn.DisconnectReason != nil {
		disconnectReason = sql.NullString{
			String: *conn.DisconnectReason,
			Valid:  true,
		}
	}

	params := sqlc.InsertWorkerConnectionParams{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.WorkspaceID,
		AppID:       conn.AppID,
		AppName:     conn.AppName,

		ID:                   conn.Id,
		GatewayID:            conn.GatewayId,
		InstanceID:           conn.InstanceId,
		Status:               int64(conn.Status),
		WorkerIp:             conn.WorkerIP,
		MaxWorkerConcurrency: conn.MaxWorkerConcurrency,

		ConnectedAt:     conn.ConnectedAt.UnixMilli(),
		LastHeartbeatAt: lastHeartbeatAt,
		DisconnectedAt:  disconnectedAt,
		RecordedAt:      conn.RecordedAt.UnixMilli(),
		InsertedAt:      time.Now().UnixMilli(),

		DisconnectReason: disconnectReason,

		GroupHash:     []byte(conn.GroupHash),
		SdkLang:       conn.SDKLang,
		SdkVersion:    conn.SDKVersion,
		SdkPlatform:   conn.SDKPlatform,
		SyncID:        conn.SyncID,
		AppVersion:    appVersion,
		FunctionCount: int64(conn.FunctionCount),

		CpuCores: int64(conn.CpuCores),
		MemBytes: conn.MemBytes,
		Os:       conn.Os,
	}

	return w.q.InsertWorkerConnection(ctx, params)
}

type WorkerConnectionCursorFilter struct {
	ID    string
	Value int64
}

func (w wrapper) GetWorkerConnection(ctx context.Context, id cqrs.WorkerConnectionIdentifier) (*cqrs.WorkerConnection, error) {
	conn, err := w.q.GetWorkerConnection(ctx, sqlc.GetWorkerConnectionParams{
		AccountID:    id.AccountID,
		WorkspaceID:  id.WorkspaceID,
		ConnectionID: id.ConnectionID,
	})
	if err != nil {
		return nil, err
	}

	connectedAt := time.UnixMilli(conn.ConnectedAt)

	var disconnectedAt, lastHeartbeatAt *time.Time
	if conn.DisconnectedAt.Valid {
		disconnectedAt = ptr.Time(time.UnixMilli(conn.DisconnectedAt.Int64))
	}

	if conn.LastHeartbeatAt.Valid {
		lastHeartbeatAt = ptr.Time(time.UnixMilli(conn.LastHeartbeatAt.Int64))
	}

	var appVersion *string
	if conn.AppVersion.Valid {
		appVersion = &conn.AppVersion.String
	}

	var disconnectReason *string
	if conn.DisconnectReason.Valid {
		disconnectReason = &conn.DisconnectReason.String
	}

	workerConn := cqrs.WorkerConnection{
		AccountID:   conn.AccountID,
		WorkspaceID: conn.WorkspaceID,
		AppID:       conn.AppID,

		Id:                   conn.ID,
		GatewayId:            conn.GatewayID,
		InstanceId:           conn.InstanceID,
		Status:               connpb.ConnectionStatus(conn.Status),
		WorkerIP:             conn.WorkerIp,
		MaxWorkerConcurrency: conn.MaxWorkerConcurrency,

		LastHeartbeatAt: lastHeartbeatAt,
		ConnectedAt:     connectedAt,
		DisconnectedAt:  disconnectedAt,
		RecordedAt:      time.UnixMilli(conn.RecordedAt),
		InsertedAt:      time.UnixMilli(conn.InsertedAt),

		DisconnectReason: disconnectReason,

		GroupHash:     string(conn.GroupHash),
		SDKLang:       conn.SdkLang,
		SDKVersion:    conn.SdkVersion,
		SDKPlatform:   conn.SdkPlatform,
		SyncID:        conn.SyncID,
		AppVersion:    appVersion,
		FunctionCount: int(conn.FunctionCount),

		CpuCores: int32(conn.CpuCores),
		MemBytes: conn.MemBytes,
		Os:       conn.Os,
	}

	return &workerConn, nil
}

type workerConnectionsQueryBuilder struct {
	filter       []sq.Expression
	order        []sqexp.OrderedExpression
	cursor       *cqrs.WorkerConnectionPageCursor
	cursorLayout *cqrs.WorkerConnectionPageCursor
}

func newWorkerConnectionsQueryBuilder(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) *workerConnectionsQueryBuilder {
	l := logger.StdlibLogger(ctx)

	// filters
	filter := []sq.Expression{}
	if len(opt.Filter.AppID) > 0 {
		filter = append(filter, sq.C("app_id").In(opt.Filter.AppID))
	}
	if len(opt.Filter.Status) > 0 {
		status := []int64{}
		for _, s := range opt.Filter.Status {
			status = append(status, int64(s))
		}
		filter = append(filter, sq.C("status").In(status))
	}
	tsfield := strings.ToLower(opt.Filter.TimeField.String())
	filter = append(filter, sq.C(tsfield).Gte(opt.Filter.From.UnixMilli()))

	until := opt.Filter.Until
	if until.UnixMilli() <= 0 {
		until = time.Now()
	}
	filter = append(filter, sq.C(tsfield).Lt(until.UnixMilli()))

	// Layout to be used for the response cursors
	resCursorLayout := cqrs.WorkerConnectionPageCursor{
		Cursors: map[string]cqrs.WorkerConnectionCursor{},
	}

	reqcursor := &cqrs.WorkerConnectionPageCursor{}
	if opt.Cursor != "" {
		if err := reqcursor.Decode(opt.Cursor); err != nil {
			l.Error("error decoding worker connection history cursor", "error", err, "cursor", opt.Cursor)
		}
	}

	// order by
	//
	// When going through the sorting fields, construct
	// - response pagination cursor layout
	// - update filter with op against sorted fields for pagination
	sortOrder := []enums.WorkerConnectionTimeField{}
	sortDir := map[enums.WorkerConnectionTimeField]enums.WorkerConnectionSortOrder{}
	cursorFilter := map[enums.WorkerConnectionTimeField]WorkerConnectionCursorFilter{}
	for _, f := range opt.Order {
		sortDir[f.Field] = f.Direction
		found := false
		for _, field := range sortOrder {
			if f.Field == field {
				found = true
				break
			}
		}
		if !found {
			sortOrder = append(sortOrder, f.Field)
		}

		rc := reqcursor.Find(f.Field.String())
		if rc != nil {
			cursorFilter[f.Field] = WorkerConnectionCursorFilter{ID: reqcursor.ID, Value: rc.Value}
		}
		resCursorLayout.Add(f.Field.String())
	}

	order := []sqexp.OrderedExpression{}
	for _, f := range sortOrder {
		var o sqexp.OrderedExpression
		field := strings.ToLower(f.String())
		if d, ok := sortDir[f]; ok {
			switch d {
			case enums.WorkerConnectionSortOrderAsc:
				o = sq.C(field).Asc()
			case enums.WorkerConnectionSortOrderDesc:
				o = sq.C(field).Desc()
			default:
				l.Error("invalid direction specified for sorting", "field", field, "direction", d.String())
				continue
			}

			order = append(order, o)
		}
	}
	order = append(order, sq.C("id").Asc())

	// cursor filter
	for k, cf := range cursorFilter {
		ord, ok := sortDir[k]
		if !ok {
			continue
		}

		var compare sq.Expression
		field := strings.ToLower(k.String())
		switch ord {
		case enums.WorkerConnectionSortOrderAsc:
			compare = sq.C(field).Gt(cf.Value)
		case enums.WorkerConnectionSortOrderDesc:
			compare = sq.C(field).Lt(cf.Value)
		default:
			continue
		}

		filter = append(filter, sq.Or(
			compare,
			sq.And(
				sq.C(field).Eq(cf.Value),
				sq.C("id").Gt(cf.ID),
			),
		))
	}

	return &workerConnectionsQueryBuilder{
		filter:       filter,
		order:        order,
		cursor:       reqcursor,
		cursorLayout: &resCursorLayout,
	}
}

func (w wrapper) GetWorkerConnectionsCount(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) (int, error) {
	// explicitly set it to zero so it would not attempt to paginate
	opt.Items = 0
	res, err := w.GetWorkerConnections(ctx, opt)
	if err != nil {
		return 0, err
	}

	return len(res), nil
}

func (w wrapper) GetWorkerConnections(ctx context.Context, opt cqrs.GetWorkerConnectionOpt) ([]*cqrs.WorkerConnection, error) {
	l := logger.StdlibLogger(ctx)

	builder := newWorkerConnectionsQueryBuilder(ctx, opt)
	filter := builder.filter
	order := builder.order
	reqcursor := builder.cursor
	resCursorLayout := builder.cursorLayout

	// read from database
	// TODO:
	// change this to a continuous loop with limits instead of just attempting to grab everything.
	// might not matter though since this is primarily meant for local development
	sql, args, err := sq.Dialect(w.dialect()).
		From("worker_connections").
		Select(
			"account_id",
			"workspace_id",
			"app_id",
			"app_name",

			"id",
			"gateway_id",
			"instance_id",
			"status",
			"worker_ip",
			"max_worker_concurrency",

			"connected_at",
			"last_heartbeat_at",
			"disconnected_at",
			"recorded_at",
			"inserted_at",

			"disconnect_reason",

			"group_hash",
			"sdk_lang",
			"sdk_version",
			"sdk_platform",
			"sync_id",
			"app_version",
			"function_count",

			"cpu_cores",
			"mem_bytes",
			"os",
		).
		Where(filter...).
		Order(order...).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := w.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	res := []*cqrs.WorkerConnection{}
	var count uint
	for rows.Next() {
		data := sqlc.WorkerConnection{}
		err := rows.Scan(
			&data.AccountID,
			&data.WorkspaceID,
			&data.AppID,
			&data.AppName,

			&data.ID,
			&data.GatewayID,
			&data.InstanceID,
			&data.Status,
			&data.WorkerIp,
			&data.MaxWorkerConcurrency,

			&data.ConnectedAt,
			&data.LastHeartbeatAt,
			&data.DisconnectedAt,
			&data.RecordedAt,
			&data.InsertedAt,

			&data.DisconnectReason,

			&data.GroupHash,
			&data.SdkLang,
			&data.SdkVersion,
			&data.SdkPlatform,
			&data.SyncID,
			&data.AppVersion,
			&data.FunctionCount,

			&data.CpuCores,
			&data.MemBytes,
			&data.Os,
		)
		if err != nil {
			return nil, err
		}

		// the cursor target should be skipped
		if reqcursor.ID == data.ID.String() {
			continue
		}

		// copy layout
		pc := resCursorLayout
		// construct the needed fields to generate a cursor representing this run
		pc.ID = data.ID.String()
		for k := range pc.Cursors {
			switch k {
			case strings.ToLower(enums.WorkerConnectionTimeFieldConnectedAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.ConnectedAt}
			case strings.ToLower(enums.WorkerConnectionTimeFieldLastHeartbeatAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.LastHeartbeatAt.Int64}
			case strings.ToLower(enums.WorkerConnectionTimeFieldDisconnectedAt.String()):
				pc.Cursors[k] = cqrs.WorkerConnectionCursor{Field: k, Value: data.DisconnectedAt.Int64}
			default:
				l.Warn("unknown field registered as cursor", "field", k)
				delete(pc.Cursors, k)
			}
		}

		cursor, err := pc.Encode()
		if err != nil {
			l.Error("error encoding cursor", "error", err, "page_cursor", pc)
		}

		connectedAt := time.UnixMilli(data.ConnectedAt)

		var disconnectedAt, lastHeartbeatAt *time.Time
		if data.DisconnectedAt.Valid {
			disconnectedAt = ptr.Time(time.UnixMilli(data.DisconnectedAt.Int64))
		}
		if data.LastHeartbeatAt.Valid {
			lastHeartbeatAt = ptr.Time(time.UnixMilli(data.LastHeartbeatAt.Int64))
		}

		var appVersion *string
		if data.AppVersion.Valid {
			appVersion = &data.AppVersion.String
		}

		var disconnectReason *string
		if data.DisconnectReason.Valid {
			disconnectReason = &data.DisconnectReason.String
		}

		res = append(res, &cqrs.WorkerConnection{
			AccountID:   data.AccountID,
			WorkspaceID: data.WorkspaceID,
			AppID:       data.AppID,
			AppName:     data.AppName,

			Id:                   data.ID,
			GatewayId:            data.GatewayID,
			InstanceId:           data.InstanceID,
			Status:               connpb.ConnectionStatus(data.Status),
			WorkerIP:             data.WorkerIp,
			MaxWorkerConcurrency: data.MaxWorkerConcurrency,

			LastHeartbeatAt: lastHeartbeatAt,
			ConnectedAt:     connectedAt,
			DisconnectedAt:  disconnectedAt,
			RecordedAt:      time.UnixMilli(data.RecordedAt),
			InsertedAt:      time.UnixMilli(data.InsertedAt),

			DisconnectReason: disconnectReason,

			GroupHash:     string(data.GroupHash),
			SDKLang:       data.SdkLang,
			SDKVersion:    data.SdkVersion,
			SDKPlatform:   data.SdkPlatform,
			SyncID:        data.SyncID,
			FunctionCount: int(data.FunctionCount),
			AppVersion:    appVersion,

			CpuCores: int32(data.CpuCores),
			MemBytes: data.MemBytes,
			Os:       data.Os,

			Cursor: cursor,
		})
		count++
		// enough items, don't need to proceed anymore
		if opt.Items > 0 && count >= opt.Items {
			break
		}
	}

	return res, nil
}

// GetSpanRuns retrieves a list of span-based runs using the same filtering
// logic as GetTraceRuns but working against the spans table with executor.run +
// EXTEND span grouping
func (w wrapper) GetSpanRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
	l := logger.StdlibLogger(ctx)
	adapter := w.spanRunsAdapter()

	builder := newSpanRunsQueryBuilder(ctx, opt)

	// Parse CEL expressions using adapter's converter
	var celFilters []sq.Expression
	var useJoin bool
	if opt.Filter.CEL != "" {
		expHandler, err := run.NewExpressionHandler(ctx,
			run.WithExpressionHandlerBlob(opt.Filter.CEL, "\n"),
			run.WithExpressionSQLConverter(adapter.celConverter),
		)
		if err != nil {
			return nil, err
		}
		if expHandler.HasFilters() {
			celFilters, err = expHandler.ToSQLFilters(ctx)
			if err != nil {
				return nil, err
			}
			useJoin = needsEventJoin(opt.Filter.CEL)
		}
	}

	selectCols := []interface{}{
		"spans.run_id",
		"spans.dynamic_span_id",
		"spans.account_id",
		"spans.app_id",
		"spans.function_id",
		"spans.trace_id",
		sq.L("MIN(spans.start_time)").As("start_time"),
		sq.L("MAX(spans.end_time)").As("end_time"),
		// subselect for argmax(status, end_time)
		// not the most efficient but it'll do for now
		sq.L(`(SELECT s2.status FROM spans s2
			WHERE s2.run_id = spans.run_id AND s2.dynamic_span_id = spans.dynamic_span_id
			ORDER BY s2.end_time DESC LIMIT 1)`).As("status"),
		adapter.eventIdsExpr, // DB-specific due to storage differences
	}

	groupByCols := []interface{}{
		"spans.run_id",
		"spans.dynamic_span_id",
		"spans.account_id",
		"spans.app_id",
		"spans.function_id",
		"spans.trace_id",
	}

	// Build ORDER BY for aggregated columns
	var orderExprs []sqexp.OrderedExpression
	for _, o := range opt.Order {
		var aggExpr sqexp.LiteralExpression
		switch o.Field {
		case enums.TraceRunTimeQueuedAt, enums.TraceRunTimeStartedAt:
			aggExpr = sq.L("MIN(spans.start_time)")
		case enums.TraceRunTimeEndedAt:
			aggExpr = sq.L("MAX(spans.end_time)")
		default:
			aggExpr = sq.L("MIN(spans.start_time)")
		}
		if o.Direction == enums.TraceRunOrderAsc {
			orderExprs = append(orderExprs, aggExpr.Asc())
		} else {
			orderExprs = append(orderExprs, aggExpr.Desc())
		}
	}
	if len(orderExprs) == 0 {
		orderExprs = append(orderExprs, sq.L("MIN(spans.start_time)").Desc())
	}
	// always add run_id at the end for stable sorting
	orderExprs = append(orderExprs, sq.C("run_id").Asc())

	q := sq.Dialect(adapter.dialect).From("spans")
	if useJoin {
		// database specific join syntax needed because event_ids is an array of ids to the events table,
		// so we need to unpack that and perform the join before the spans are grouped back together by run_id
		q = adapter.buildEventJoin(q)
	}

	allFilters := append(builder.filter, celFilters...)
	q = q.Select(selectCols...).
		Where(sq.L("spans.dynamic_span_id").In(
			sq.Dialect(adapter.dialect).Select("dynamic_span_id").Distinct().From("spans").Where(sq.C("name").Eq(meta.SpanNameRun)),
		)).
		Where(allFilters...).
		GroupBy(groupByCols...).
		Order(orderExprs...)

	if opt.Items > 0 {
		q = q.Limit(opt.Items + 1) // fetch one more item than requested to determine hasNextPage
	}

	sqlQuery, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	l.Debug("GetSpanRuns query", "sql", sqlQuery, "args", args)

	rows, err := w.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		l.Debug("GetSpanRuns query error", "error", err)
		return nil, err
	}
	defer rows.Close()

	return w.convertSpanRunRows(ctx, rows, builder.cursorLayout, adapter, opt.Items)
}

// convertSpanRunRows converts database rows to TraceRun structs
func (w wrapper) convertSpanRunRows(
	ctx context.Context,
	rows *sql.Rows,
	cursorLayout *cqrs.TracePageCursor,
	adapter spanRunsAdapter,
	itemLimit uint,
) ([]*cqrs.TraceRun, error) {
	l := logger.StdlibLogger(ctx)

	type runRow struct {
		RunID         string
		DynamicSpanID string
		AccountID     string
		AppID         string
		FunctionID    string
		TraceID       string
		StartTime     string
		EndTime       *string
		Status        *string
		EventIDs      *string
	}

	res := []*cqrs.TraceRun{}
	var count uint

	for rows.Next() {
		var row runRow
		err := rows.Scan(
			&row.RunID,
			&row.DynamicSpanID,
			&row.AccountID,
			&row.AppID,
			&row.FunctionID,
			&row.TraceID,
			&row.StartTime,
			&row.EndTime,
			&row.Status,
			&row.EventIDs,
		)
		if err != nil {
			return nil, err
		}

		// Parse times using adapter, times are stored differently across SQLite and Postgres
		startTime, err := adapter.parseTime(row.StartTime)
		if err != nil {
			l.Debug("invalid start_time", "start_time", row.StartTime, "error", err)
			continue
		}
		var endTime *time.Time
		if row.EndTime != nil && *row.EndTime != "" {
			if t, err := adapter.parseTime(*row.EndTime); err == nil {
				endTime = &t
			}
		}

		// Parse UUIDs
		accountUUID, err := uuid.Parse(row.AccountID)
		if err != nil {
			l.Debug("invalid account ID", "account_id", row.AccountID, "error", err)
			continue
		}
		appUUID, err := uuid.Parse(row.AppID)
		if err != nil {
			l.Debug("invalid app ID", "app_id", row.AppID, "error", err)
			continue
		}
		functionUUID, err := uuid.Parse(row.FunctionID)
		if err != nil {
			l.Debug("invalid function ID", "function_id", row.FunctionID, "error", err)
			continue
		}

		// Parse status
		status := enums.RunStatusRunning
		if row.Status != nil && *row.Status != "" {
			if stepStatus, err := enums.StepStatusString(*row.Status); err == nil && stepStatus != enums.StepStatusUnknown {
				status = enums.StepStatusToRunStatus(stepStatus)
			}
		}

		// Parse event IDs using adapter due to differences in column type and serialization
		triggerIDs := adapter.parseEventIDs(row.EventIDs)

		// Calculate duration
		var duration time.Duration
		if endTime != nil {
			duration = endTime.Sub(startTime)
		}

		// Build cursor for pagination
		var cursor string
		if cursorLayout != nil {
			c := &cqrs.TracePageCursor{
				ID:      row.RunID,
				Cursors: map[string]cqrs.TraceCursor{},
			}
			for field := range cursorLayout.Cursors {
				switch field {
				case "start_time":
					c.Cursors[field] = cqrs.TraceCursor{Field: field, Value: startTime.UnixMicro()}
				case "end_time":
					if endTime != nil {
						c.Cursors[field] = cqrs.TraceCursor{Field: field, Value: endTime.UnixMicro()}
					}
				}
			}
			if encoded, err := c.Encode(); err == nil {
				cursor = encoded
			}
		}

		traceRun := &cqrs.TraceRun{
			AccountID:   accountUUID,
			WorkspaceID: accountUUID,
			AppID:       appUUID,
			FunctionID:  functionUUID,
			TraceID:     row.TraceID,
			RunID:       row.RunID,
			QueuedAt:    startTime,
			StartedAt:   startTime,
			Duration:    duration,
			Status:      status,
			Cursor:      cursor,
			TriggerIDs:  triggerIDs,
		}

		if endTime != nil {
			traceRun.EndedAt = *endTime
		}

		res = append(res, traceRun)
		count++

		// We have filled a page's worth of requests, so break
		if itemLimit > 0 && count >= itemLimit {
			break
		}
	}

	return res, nil
}

// newSpanRunsQueryBuilder creates a query builder for span-based runs Similar
// to newRunsQueryBuilder but adapted for spans table structure
func newSpanRunsQueryBuilder(ctx context.Context, opt cqrs.GetTraceRunOpt) *runsQueryBuilder {
	l := logger.StdlibLogger(ctx)

	// filters
	filter := []sq.Expression{}
	//
	// debug runs are a special kind of run that should not be included in the main runs list
	filter = append(filter, sq.C("debug_run_id").IsNull())
	if len(opt.Filter.AppID) > 0 {
		filter = append(filter, sq.C("app_id").In(opt.Filter.AppID))
	}
	if len(opt.Filter.FunctionID) > 0 {
		filter = append(filter, sq.C("function_id").In(opt.Filter.FunctionID))
	}
	if len(opt.Filter.Status) > 0 {
		statusStrings := make([]string, 0, len(opt.Filter.Status))
		for _, s := range opt.Filter.Status {
			statusStrings = append(statusStrings, s.String())
		}
		filter = append(filter, sq.C("status").In(statusStrings))
	}
	// Skipped runs should only be visible in event-scoped queries, not the runs list.
	// status is nullable in spans, so we must also accept NULL.
	filter = append(filter, sq.Or(
		sq.C("status").IsNull(),
		sq.C("status").Neq(enums.RunStatusSkipped.String()),
	))

	// Map time fields - spans use start_time/end_time instead of
	// queued_at/started_at/ended_at
	var tsfield string
	switch opt.Filter.TimeField {
	case enums.TraceRunTimeQueuedAt, enums.TraceRunTimeStartedAt:
		tsfield = "start_time"
	case enums.TraceRunTimeEndedAt:
		tsfield = "end_time"
	default:
		tsfield = "start_time"
	}

	// Convert times to UTC to match spans storage format in SQLite
	// We currently store SQLite timestamps as Go's time.Time string: "2025-07-13 19:32:24.939517 +0000 UTC m=+..."
	// SQLite compares these as strings, so filter times must also serialize with "+0000 UTC" suffix to correctly use
	// lexicographic comparisons.
	// The UTC conversion was not strictly necessary for Postgres because the timestamp columns are timestamptz, so
	// type and timezone conversion were handled for us
	filter = append(filter, sq.C(tsfield).Gte(opt.Filter.From.UTC()))
	filter = append(filter, sq.C(tsfield).Lt(opt.Filter.Until.UTC()))

	// cursor
	resCursorLayout := &cqrs.TracePageCursor{
		Cursors: map[string]cqrs.TraceCursor{},
	}

	// decode request cursor if there's one
	var reqCursor *cqrs.TracePageCursor
	if len(opt.Cursor) > 0 {
		reqCursor = &cqrs.TracePageCursor{Cursors: map[string]cqrs.TraceCursor{}}
		if err := reqCursor.Decode(opt.Cursor); err != nil {
			l.Debug("cursor decode failed", "error", err)
			reqCursor = nil
		}
	}

	// orders
	order := []sqexp.OrderedExpression{}
	for _, o := range opt.Order {
		// Map enum field names to column names
		var field string
		switch o.Field {
		case enums.TraceRunTimeQueuedAt, enums.TraceRunTimeStartedAt:
			field = "start_time"
		case enums.TraceRunTimeEndedAt:
			field = "end_time"
		default:
			field = "start_time"
		}

		resCursorLayout.Add(field)

		switch o.Direction {
		case enums.TraceRunOrderAsc:
			order = append(order, sq.C(field).Asc())
		case enums.TraceRunOrderDesc:
			order = append(order, sq.C(field).Desc())
		}
	}

	// Always add run_id as final sort field for stable pagination
	order = append(order, sq.C("run_id").Asc())
	resCursorLayout.Add("run_id")

	// cursor-based pagination filter
	if reqCursor != nil {
		cursorFilters := []sq.Expression{}
		for i, o := range opt.Order {
			// Map field names same as above
			var field string
			switch o.Field {
			case enums.TraceRunTimeQueuedAt, enums.TraceRunTimeStartedAt:
				field = "start_time"
			case enums.TraceRunTimeEndedAt:
				field = "end_time"
			default:
				field = "start_time"
			}

			if cursor := reqCursor.Find(field); cursor != nil {
				// Build cursor condition for this field
				// Convert int64 microseconds to time.Time in UTC for spans table comparison
				cursorTime := time.UnixMicro(cursor.Value).UTC()
				var baseCondition sq.Expression
				if o.Direction == enums.TraceRunOrderAsc {
					baseCondition = sq.C(field).Gt(cursorTime)
				} else {
					baseCondition = sq.C(field).Lt(cursorTime)
				}

				// Build compound condition for tie-breaking
				equalityConditions := []sq.Expression{sq.C(field).Eq(cursorTime)}

				// Add conditions for all subsequent fields in sort order
				for j := i + 1; j < len(opt.Order); j++ {
					var nextField string
					switch opt.Order[j].Field {
					case enums.TraceRunTimeQueuedAt, enums.TraceRunTimeStartedAt:
						nextField = "start_time"
					case enums.TraceRunTimeEndedAt:
						nextField = "end_time"
					default:
						nextField = "start_time"
					}

					if nextCursor := reqCursor.Find(nextField); nextCursor != nil {
						nextCursorTime := time.UnixMicro(nextCursor.Value).UTC()
						if opt.Order[j].Direction == enums.TraceRunOrderAsc {
							equalityConditions = append(equalityConditions, sq.C(nextField).Gt(nextCursorTime))
						} else {
							equalityConditions = append(equalityConditions, sq.C(nextField).Lt(nextCursorTime))
						}
					}
				}

				// Add run_id tie-breaker
				if reqCursor.ID != "" {
					equalityConditions = append(equalityConditions, sq.C("run_id").Gt(reqCursor.ID))
				}

				// Combine: (field > cursor_value) OR (field = cursor_value AND next_conditions)
				tieBreakingCondition := sq.And(equalityConditions...)
				cursorFilters = append(cursorFilters, sq.Or(baseCondition, tieBreakingCondition))
			}
		}

		if len(cursorFilters) > 0 {
			filter = append(filter, sq.Or(cursorFilters...))
		}
	}

	return &runsQueryBuilder{
		filter:       filter,
		order:        order,
		cursor:       reqCursor,
		cursorLayout: resCursorLayout,
	}
}

// needsEventJoin checks if CEL expression references event.* fields
func needsEventJoin(cel string) bool {
	return strings.Contains(cel, "event.")
}

func isWaitForEventOutput(o map[string]any) bool {
	_, name := o["name"]
	_, data := o["data"]
	_, ts := o["ts"]
	return name && data && ts
}
