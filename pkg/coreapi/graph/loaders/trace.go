package loader

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
	"github.com/inngest/inngest/pkg/consts"
	"github.com/inngest/inngest/pkg/coreapi/graph/models"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/oklog/ulid/v2"
)

type TraceRequestKey struct {
	*cqrs.TraceRunIdentifier
}

func (k *TraceRequestKey) Raw() any {
	return k
}

func (k *TraceRequestKey) String() string {
	return fmt.Sprintf("%s-%s", k.TraceID, k.RunID)
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

			_, err = tb.Build()
			if err != nil {
				res.Error = fmt.Errorf("error run details: %w", err)
				return
			}

			// TODO: prime tree
		}(ctx, results[i])
	}

	wg.Wait()

	return results
}

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

func (tb *TraceTreeBuilder) Build() (*models.RunTraceSpan, error) {
	return nil, fmt.Errorf("not implemented")
}
