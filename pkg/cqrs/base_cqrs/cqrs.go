package base_cqrs

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
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

func (w wrapper) GetSpansByRunID(ctx context.Context, runID ulid.ULID) (*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByRunID(ctx, runID.String())
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by run ID", "error", err)
		return nil, err
	}

	// We need an ordered map here because we loop over it later
	spanMap := orderedmap.NewOrderedMap[string, *cqrs.OtelSpan]()

	// A map of dynamic span IDs to the specific span ID that contains an
	// output
	outputDynamicRefs := make(map[string]string)
	var root *cqrs.OtelSpan

	for _, span := range spans {
		st := strings.Split(span.StartTime.(string), " m=")[0]
		startTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", st)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing start time", "error", err)
			return nil, err
		}

		et := strings.Split(span.EndTime.(string), " m=")[0]
		endTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", et)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing end time", "error", err)
			return nil, err
		}

		var parentSpanID *string
		if span.ParentSpanID.Valid {
			parentSpanID = &span.ParentSpanID.String
		}

		newSpan := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				SpanID:       span.DynamicSpanID.String,
				TraceID:      span.TraceID,
				ParentSpanID: parentSpanID,
				StartTime:    startTime,
				EndTime:      endTime,
				Name:         "",
				Attributes:   make(map[string]any),
			},
			Status:          enums.StepStatusRunning,
			RunID:           runID,
			MarkedAsDropped: false,
		}

		var outputSpanID *string
		var fragments []map[string]interface{}
		groupedAttrs := make(map[string]any)
		_ = json.Unmarshal([]byte(span.SpanFragments.(string)), &fragments)

		for _, fragment := range fragments {
			if name, ok := fragment["name"].(string); ok {
				if strings.HasPrefix(name, "executor.") {
					newSpan.Name = name
				}
			}

			if attrs, ok := fragment["attributes"].(string); ok {
				fragmentAttr := map[string]any{}
				if err := json.Unmarshal([]byte(attrs), &fragmentAttr); err != nil {
					logger.StdlibLogger(ctx).Error("error unmarshalling span attributes", "error", err)
					return nil, err
				}

				maps.Copy(groupedAttrs, fragmentAttr)

				if outputRef, ok := fragment["output_span_id"].(string); ok {
					outputSpanID = &outputRef
					outputDynamicRefs[span.DynamicSpanID.String] = outputRef
				}
			}
		}

		newSpan.Attributes, err = meta.ExtractTypedValues(ctx, groupedAttrs)
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
		if outputSpanID != nil && *outputSpanID != "" {
			newSpan.OutputID, err = encodeSpanOutputID(*outputSpanID)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error encoding span identifier", "error", err)
				return nil, err
			}
		}

		spanMap.Set(span.DynamicSpanID.String, newSpan)
	}

	for _, span := range spanMap.AllFromFront() {
		// If we have an output reference for this span, set the appropriate
		// target span ID here
		if spanRefStr := span.Attributes.StepOutputRef; spanRefStr != nil && *spanRefStr != "" {
			if targetSpanID, ok := outputDynamicRefs[*spanRefStr]; ok {
				// We've found the span ID that we need to target for
				// this span. So let's use it!
				span.OutputID, err = encodeSpanOutputID(targetSpanID)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error encoding span output ID", "error", err)
					return nil, err
				}
			}

		}

		if span.ParentSpanID == nil || *span.ParentSpanID == "" || *span.ParentSpanID == "0000000000000000" {
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

func (w wrapper) GetSpansByDebugRunID(ctx context.Context, debugRunID ulid.ULID) ([]*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByDebugRunID(ctx, sql.NullString{String: debugRunID.String(), Valid: true})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by debug run ID", "error", err)
		return nil, err
	}

	// We need an ordered map here because we loop over it later
	spanMap := orderedmap.NewOrderedMap[string, *cqrs.OtelSpan]()

	// A map of dynamic span IDs to the specific span ID that contains an output
	outputDynamicRefs := make(map[string]string)
	var roots []*cqrs.OtelSpan

	for _, span := range spans {
		st := strings.Split(span.StartTime.(string), " m=")[0]
		startTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", st)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing start time", "error", err)
			return nil, err
		}

		et := strings.Split(span.EndTime.(string), " m=")[0]
		endTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", et)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing end time", "error", err)
			return nil, err
		}

		var parentSpanID *string
		if span.ParentSpanID.Valid {
			parentSpanID = &span.ParentSpanID.String
		}

		// Parse the run ID from the span
		runID, err := ulid.Parse(span.RunID)
		if err != nil {
			logger.StdlibLogger(ctx).Error("error parsing run ID", "error", err, "runID", span.RunID)
			return nil, err
		}

		newSpan := &cqrs.OtelSpan{
			RawOtelSpan: cqrs.RawOtelSpan{
				SpanID:       span.DynamicSpanID.String,
				TraceID:      span.TraceID,
				ParentSpanID: parentSpanID,
				StartTime:    startTime,
				EndTime:      endTime,
				Name:         "",
				Attributes:   make(map[string]any),
			},
			Status:          enums.StepStatusRunning,
			RunID:           runID,
			MarkedAsDropped: false,
		}

		var outputSpanID *string
		var fragments []map[string]interface{}
		groupedAttrs := make(map[string]any)
		_ = json.Unmarshal([]byte(span.SpanFragments.(string)), &fragments)

		for _, fragment := range fragments {
			if name, ok := fragment["name"].(string); ok {
				if strings.HasPrefix(name, "executor.") {
					newSpan.Name = name
				}
			}

			if attrs, ok := fragment["attributes"].(string); ok {
				fragmentAttr := map[string]any{}
				if err := json.Unmarshal([]byte(attrs), &fragmentAttr); err != nil {
					logger.StdlibLogger(ctx).Error("error unmarshalling span attributes", "error", err)
					return nil, err
				}

				maps.Copy(groupedAttrs, fragmentAttr)

				if outputRef, ok := fragment["output_span_id"].(string); ok {
					outputSpanID = &outputRef
					outputDynamicRefs[span.DynamicSpanID.String] = outputRef
				}
			}
		}

		newSpan.Attributes, err = meta.ExtractTypedValues(ctx, groupedAttrs)
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
		if outputSpanID != nil && *outputSpanID != "" {
			newSpan.OutputID, err = encodeSpanOutputID(*outputSpanID)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error encoding span identifier", "error", err)
				return nil, err
			}
		}

		spanMap.Set(span.DynamicSpanID.String, newSpan)
	}

	for _, span := range spanMap.AllFromFront() {
		// If we have an output reference for this span, set the appropriate target span ID here
		if spanRefStr := span.Attributes.StepOutputRef; spanRefStr != nil && *spanRefStr != "" {
			if targetSpanID, ok := outputDynamicRefs[*spanRefStr]; ok {
				// We've found the span ID that we need to target for this span. So let's use it!
				span.OutputID, err = encodeSpanOutputID(targetSpanID)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error encoding span output ID", "error", err)
					return nil, err
				}
			}
		}

		if span.ParentSpanID == nil || *span.ParentSpanID == "" || *span.ParentSpanID == "0000000000000000" {
			root, _ := spanMap.Get(span.SpanID)
			roots = append(roots, root)
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

	// Sort each root span
	for _, root := range roots {
		sorter(root)
	}

	return roots, nil
}

func (w wrapper) GetSpansByDebugSessionID(ctx context.Context, debugSessionID ulid.ULID) ([]*cqrs.OtelSpan, error) {
	spans, err := w.q.GetSpansByDebugSessionID(ctx, sql.NullString{String: debugSessionID.String(), Valid: true})
	if err != nil {
		logger.StdlibLogger(ctx).Error("error getting spans by debug session ID", "error", err)
		return nil, err
	}

	// Group spans by debug_run_id to build separate trace trees
	spansByDebugRun := make(map[string][]*sqlc.GetSpansByDebugSessionIDRow)
	for _, span := range spans {
		if span.DebugRunID.Valid {
			spansByDebugRun[span.DebugRunID.String] = append(spansByDebugRun[span.DebugRunID.String], span)
		}
	}

	var allRoots []*cqrs.OtelSpan

	// Process each debug run separately
	for debugRunID, runSpans := range spansByDebugRun {
		spanMap := orderedmap.NewOrderedMap[string, *cqrs.OtelSpan]()
		outputDynamicRefs := make(map[string]string)
		var root *cqrs.OtelSpan

		for _, span := range runSpans {
			st := strings.Split(span.StartTime.(string), " m=")[0]
			startTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", st)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error parsing start time", "error", err)
				return nil, err
			}

			et := strings.Split(span.EndTime.(string), " m=")[0]
			endTime, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", et)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error parsing end time", "error", err)
				return nil, err
			}

			var parentSpanID *string
			if span.ParentSpanID.Valid {
				parentSpanID = &span.ParentSpanID.String
			}

			// Parse the run ID from the span
			runID, err := ulid.Parse(span.RunID)
			if err != nil {
				logger.StdlibLogger(ctx).Error("error parsing run ID", "error", err, "runID", span.RunID)
				return nil, err
			}

			newSpan := &cqrs.OtelSpan{
				RawOtelSpan: cqrs.RawOtelSpan{
					SpanID:       span.DynamicSpanID.String,
					TraceID:      span.TraceID,
					ParentSpanID: parentSpanID,
					StartTime:    startTime,
					EndTime:      endTime,
					Name:         "",
					Attributes:   make(map[string]any),
				},
				Status:          enums.StepStatusRunning,
				RunID:           runID,
				MarkedAsDropped: false,
			}

			var outputSpanID *string
			var fragments []map[string]interface{}
			groupedAttrs := make(map[string]any)
			_ = json.Unmarshal([]byte(span.SpanFragments.(string)), &fragments)

			for _, fragment := range fragments {
				if name, ok := fragment["name"].(string); ok {
					if strings.HasPrefix(name, "executor.") {
						newSpan.Name = name
					}
				}

				if attrs, ok := fragment["attributes"].(string); ok {
					fragmentAttr := map[string]any{}
					if err := json.Unmarshal([]byte(attrs), &fragmentAttr); err != nil {
						logger.StdlibLogger(ctx).Error("error unmarshalling span attributes", "error", err)
						return nil, err
					}

					maps.Copy(groupedAttrs, fragmentAttr)

					if outputRef, ok := fragment["output_span_id"].(string); ok {
						outputSpanID = &outputRef
						outputDynamicRefs[span.DynamicSpanID.String] = outputRef
					}
				}
			}

			newSpan.Attributes, err = meta.ExtractTypedValues(ctx, groupedAttrs)
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
			if outputSpanID != nil && *outputSpanID != "" {
				newSpan.OutputID, err = encodeSpanOutputID(*outputSpanID)
				if err != nil {
					logger.StdlibLogger(ctx).Error("error encoding span identifier", "error", err)
					return nil, err
				}
			}

			spanMap.Set(span.DynamicSpanID.String, newSpan)
		}

		for _, span := range spanMap.AllFromFront() {
			// If we have an output reference for this span, set the appropriate target span ID here
			if spanRefStr := span.Attributes.StepOutputRef; spanRefStr != nil && *spanRefStr != "" {
				if targetSpanID, ok := outputDynamicRefs[*spanRefStr]; ok {
					// We've found the span ID that we need to target for this span. So let's use it!
					span.OutputID, err = encodeSpanOutputID(targetSpanID)
					if err != nil {
						logger.StdlibLogger(ctx).Error("error encoding span output ID", "error", err)
						return nil, err
					}
				}
			}

			if span.ParentSpanID == nil || *span.ParentSpanID == "" || *span.ParentSpanID == "0000000000000000" {
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
					"debugRunID", debugRunID,
				)
			}
		}

		if root != nil {
			sorter(root)
			allRoots = append(allRoots, root)
		}
	}

	return allRoots, nil
}

func encodeSpanOutputID(spanID string) (*string, error) {
	p := true

	id := &cqrs.SpanIdentifier{
		SpanID:  spanID,
		Preview: &p,
	}

	encoded, err := id.Encode()
	if err != nil {
		return nil, err
	}

	return &encoded, nil
}

func sorter(span *cqrs.OtelSpan) {
	sort.Slice(span.Children, func(i, j int) bool {
		if !span.Children[i].StartTime.Equal(span.Children[j].StartTime) {
			return span.Children[i].StartTime.Before(span.Children[j].StartTime)
		}

		// sort based on SpanID if two spans have equal timestamps
		return span.Children[i].SpanID < span.Children[j].SpanID
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
	if opts.SpanID == "" {
		return nil, fmt.Errorf("spanID is required to retrieve output")
	}

	s, err := w.q.GetSpanOutput(ctx, opts.SpanID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving span output: %w", err)
	}

	so := &cqrs.SpanOutput{}
	var m map[string]any

	so.Data = []byte(fmt.Append(nil, s))
	if err := json.Unmarshal(so.Data, &m); err == nil && m != nil {
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
	res, err := w.GetTraceRuns(ctx, opt)
	if err != nil {
		return 0, err
	}

	return len(res), nil
}

func (w wrapper) GetTraceRuns(ctx context.Context, opt cqrs.GetTraceRunOpt) ([]*cqrs.TraceRun, error) {
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

		ID:         conn.Id,
		GatewayID:  conn.GatewayId,
		InstanceID: conn.InstanceId,
		Status:     int64(conn.Status),
		WorkerIp:   conn.WorkerIP,

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

		Id:         conn.ID,
		GatewayId:  conn.GatewayID,
		InstanceId: conn.InstanceID,
		Status:     connpb.ConnectionStatus(conn.Status),
		WorkerIP:   conn.WorkerIp,

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

			Id:         data.ID,
			GatewayId:  data.GatewayID,
			InstanceId: data.InstanceID,
			Status:     connpb.ConnectionStatus(data.Status),
			WorkerIP:   data.WorkerIp,

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
