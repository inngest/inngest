package loader

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/oklog/ulid/v2"
)

type TraceRequestKey struct {
	*cqrs.TraceRunIdentifier
}

func (k *TraceRequestKey) Raw() any {
	return k
}

func (k *TraceRequestKey) String() string {
	return fmt.Sprintf("%s:%s", k.TraceID, k.RunID)
}

type traceReader struct {
	loaders *loaders
	reader  cqrs.TraceReader
}

func (tr *traceReader) GetRunTrace(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		results[i] = &dataloader.Result{}

		req, ok := key.Raw().(*TraceRequestKey)
		if !ok {
			results[i].Error = fmt.Errorf("unexpected type %T", key.Raw())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result) {
			defer wg.Done()

			spans, err := tr.reader.GetTraceSpansByRun(ctx, *req.TraceRunIdentifier)
			if err != nil {
				res.Error = fmt.Errorf("error retrieving span: %w", err)
				return
			}
			if len(spans) < 1 {
				return
			}

			// TODO: build tree from spans
			tb, err := NewTraceTreeBuilder(TraceTreeBuilderOpts{
				AccountID:   req.AccountID,
				WorkspaceID: req.WorkspaceID,
				AppID:       req.AppID,
				FunctionID:  req.FunctionID,
				RunID:       req.RunID,
				Spans:       spans,
			})
			if err != nil {
				res.Error = err
				return
			}

			tree, err := tb.Build()
			if err != nil {
				res.Error = fmt.Errorf("error run details: %w", err)
				return
			}

			res.Data = tree
			var primeTree func(context.Context, []*models.RunTraceSpan)
			primeTree = func(ctx context.Context, tspans []*models.RunTraceSpan) {
				for _, span := range tspans {
					if span != nil {
						if span.SpanID != "" {
							tr.loaders.RunSpanLoader.Prime(
								ctx,
								&SpanRequestKey{
									TraceRunIdentifier: req.TraceRunIdentifier,
									SpanID:             span.SpanID,
								},
								span,
							)
						}

						if span.ChildrenSpans != nil && len(span.ChildrenSpans) > 0 {
							primeTree(ctx, span.ChildrenSpans)
						}
					}
				}
			}

			primeTree(ctx, []*models.RunTraceSpan{tree})
		}(ctx, results[i])
	}

	wg.Wait()

	return results
}

type SpanRequestKey struct {
	*cqrs.TraceRunIdentifier `json:"trident,omitempty"`
	SpanID                   string `json:"sid"`
}

func (k *SpanRequestKey) Raw() any {
	return k
}

func (k *SpanRequestKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.TraceID, k.RunID, k.SpanID)
}

func (tr *traceReader) GetSpanRun(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	results := make([]*dataloader.Result, len(keys))

	var wg sync.WaitGroup
	for i, key := range keys {
		results[i] = &dataloader.Result{}

		req, ok := key.Raw().(*SpanRequestKey)
		if !ok {
			results[i].Error = fmt.Errorf("unexpected type for span %T", key.Raw())
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, res *dataloader.Result) {
			defer wg.Done()

			// If we're here, we're requested a span ID that wasn't primed by
			// GetRunTrace. Span IDs can sometimes be virtualized based on the
			// entire trace, so here we refetch the entire trace for each key and
			// pick out the spans we need.
			//
			// Because this is calling another loader, duplicate requests will
			// still be filtered out.
			rootSpan, err := LoadOne[models.RunTraceSpan](
				ctx,
				tr.loaders.RunTraceLoader,
				&TraceRequestKey{TraceRunIdentifier: req.TraceRunIdentifier},
			)
			if err != nil {
				res.Error = fmt.Errorf("failed to get run trace: %w", err)
			}

			var findNestedSpan func([]*models.RunTraceSpan) *models.RunTraceSpan
			findNestedSpan = func(spans []*models.RunTraceSpan) *models.RunTraceSpan {
				for _, span := range spans {
					if span == nil {
						continue
					}
					if span.SpanID == req.SpanID {
						return span
					}

					if len(span.ChildrenSpans) > 0 {
						nestedSpan := findNestedSpan(span.ChildrenSpans)
						if nestedSpan != nil {
							return nestedSpan
						}
					}
				}
				return nil
			}

			res.Data = findNestedSpan([]*models.RunTraceSpan{rootSpan})
		}(ctx, results[i])
	}

	wg.Wait()
	return results
}

// TraceTreeBuilder builds the span tree used for the API
type TraceTreeBuilder struct {
	// root is the root span of the tree
	root *cqrs.Span
	// trigger is the trigger span of the tree
	trigger *cqrs.Span

	// spans is used for quick access seen spans,
	// and also assigning children span to them.
	// - key: spanID
	spans map[string]*cqrs.Span
	// groups is used for grouping spans together
	// - key: groupID
	groups map[string][]*cqrs.Span

	// identifiers
	accID uuid.UUID
	wsID  uuid.UUID
	appID uuid.UUID
	fnID  uuid.UUID
	runID ulid.ULID
}

type TraceTreeBuilderOpts struct {
	AccountID   uuid.UUID
	WorkspaceID uuid.UUID
	AppID       uuid.UUID
	FunctionID  uuid.UUID
	RunID       ulid.ULID
	Spans       []*cqrs.Span
}

type SpanConverter func(*cqrs.Span)

func NewTraceTreeBuilder(opts TraceTreeBuilderOpts) (*TraceTreeBuilder, error) {
	ttb := &TraceTreeBuilder{
		accID:  opts.AccountID,
		wsID:   opts.WorkspaceID,
		appID:  opts.AppID,
		fnID:   opts.FunctionID,
		runID:  opts.RunID,
		spans:  map[string]*cqrs.Span{},
		groups: map[string][]*cqrs.Span{},
	}

	for _, s := range opts.Spans {
		if s.ScopeName == consts.OtelScopeFunction {
			ttb.root = s
		}

		if s.ScopeName == consts.OtelScopeTrigger {
			ttb.trigger = s
		}

		ttb.spans[s.SpanID] = s
		if s.ParentSpanID != nil {
			if groupID := s.GroupID(); groupID != nil {
				if _, exists := ttb.groups[*groupID]; !exists {
					ttb.groups[*groupID] = []*cqrs.Span{}
				}
				ttb.groups[*groupID] = append(ttb.groups[*groupID], s)
			}
		}
	}

	// loop through again to construct parent/child relationship
	for _, s := range opts.Spans {
		if s.ParentSpanID != nil {
			if parent, ok := ttb.spans[*s.ParentSpanID]; ok {
				if parent.Children == nil {
					parent.Children = []*cqrs.Span{}
				}
				parent.Children = append(parent.Children, s)
			}
		}
	}

	if ttb.root == nil {
		return nil, fmt.Errorf("no function run span found")
	}
	if ttb.trigger == nil {
		return nil, fmt.Errorf("no trigger span found")
	}

	return ttb, nil
}

// Build goes through the tree and construct the trace for API to be consumed
func (tb *TraceTreeBuilder) Build() (*models.RunTraceSpan, error) {
	root, err := tb.toRunTraceSpan(tb.root)
	if err != nil {
		return nil, fmt.Errorf("error converting function span: %w", err)
	}

	// sort it in asc order before proceeding
	spans := tb.root.Children
	sort.Slice(spans, func(i, j int) bool {
		return spans[i].Timestamp.UnixMilli() < spans[j].Timestamp.UnixMilli()
	})

	// these are the execution or steps for the function run
	for _, span := range spans {
		tspan, err := tb.toRunTraceSpan(span)
		if err != nil {
			return nil, fmt.Errorf("error converting execution span: %w", err)
		}
		root.ChildrenSpans = append(root.ChildrenSpans, tspan)
	}

	return root, nil
}

func (tb *TraceTreeBuilder) toRunTraceSpan(s *cqrs.Span) (*models.RunTraceSpan, error) {
	var (
		appID  uuid.UUID
		fnID   uuid.UUID
		runID  ulid.ULID
		status models.RunTraceSpanStatus
		stepOp *models.StepOp
	)

	if s.RunID != nil {
		runID = *s.RunID
	}

	if id := s.AppID(); id != nil {
		appID = *id
	}
	if id := s.FunctionID(); id != nil {
		fnID = *id
	}

	// TODO: assign step status
	if s.ScopeName == consts.OtelScopeFunction {
		fnstatus := s.FunctionStatus()
		switch fnstatus {
		case enums.RunStatusRunning:
			status = models.RunTraceSpanStatusRunning
		case enums.RunStatusCompleted:
			status = models.RunTraceSpanStatusCompleted
		case enums.RunStatusCancelled:
			status = models.RunTraceSpanStatusCancelled
		case enums.RunStatusFailed, enums.RunStatusOverflowed:
			status = models.RunTraceSpanStatusFailed
		default:
			return nil, fmt.Errorf("unexpected run status: %v", fnstatus.String())
		}
	}

	res := models.RunTraceSpan{
		AppID:         appID,
		FunctionID:    fnID,
		RunID:         runID,
		TraceID:       s.TraceID,
		ParentSpanID:  s.ParentSpanID,
		SpanID:        s.SpanID,
		IsRoot:        s.ParentSpanID == nil,
		Name:          s.SpanName,
		Status:        status,
		QueuedAt:      ulid.Time(runID.Time()),
		ChildrenSpans: []*models.RunTraceSpan{},
		StepOp:        stepOp,
	}

	return &res, nil
}
