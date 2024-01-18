package expressions

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/karlseguin/ccache/v2"
)

func NewAggregator(
	ctx context.Context,
	size int64,
	loader EvaluableLoader,
	log *slog.Logger,
) Aggregator {
	// use the package's singleton caching parser to create a new tree parser.
	// this uses lifted expression parsing with caching for speed.
	parser := expr.NewTreeParser(exprCompiler)
	if log == nil {
		log = logger.StdlibLogger(ctx)
	}
	return aggregator{
		log:     log,
		records: ccache.New(ccache.Configure().MaxSize(size).ItemsToPrune(200)),
		loader:  loader,
		parser:  parser,
		// use the package's exprEvaluator function as the actual logic which evaluates
		// expressions after the aggregate evaluator does matching.
		evaluator: exprEvaluator,
	}
}

// EvaluableLoader loads all Evaluables from a store since the given time, invoking the given do function for each
// event.
//
// It's expected that this wraps the state.PauseGetter interface, calling `do` for each item in the PauseIterator.
// The types are different as we must use the open source expr.Evaluable interfaces with aggregate evaluation.
type EvaluableLoader interface {
	LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error
}

// Aggregator manages a set of AggregateEvaluator instances to quickly evaluate expressions
// for incoming events.
//
// This is used across all pauses — `waitForEvent` and cancellations.
type Aggregator interface {
	// EvaluateAsyncEvent is a shorthand to evaluate Evaluable isntances tracked for a given event,
	// eg. all pauses stored in the state store.
	EvaluateAsyncEvent(ctx context.Context, event event.TrackedEvent) ([]expr.Evaluable, int32, error)

	// LoadEventEvaluator returns the aggregate evaluator for a given event.  This does a few
	// things under the hood:
	//
	// First, we must check to see if we have an existing AE present. If so, it loads pauses
	// since the AE was last updated and adds them to the AE.  This ensures that waitForEvent
	// expressions that were just created are evaluated properly.  It also minimizes reads
	// from the state store;  we only load a subset of all pauses in memory.
	//
	// If an AE is not present we must create a new AE for the event and load all pauses for
	// the given event into the tree.
	//
	// Finally, the Aggregator performs record-keeping, storing the size of AEs, usage statistics,
	// and LRU semantics to evict non-recently-used AEs under memory pressure.
	LoadEventEvaluator(ctx context.Context, wsID uuid.UUID, eventName string) (expr.AggregateEvaluator, error)

	// RemovePause is a shortcut to find an event evaluator _without_ refreshing new data, and to
	// remove the pause's expressions from any aggregate trees.
	//
	// This must be called by the executor when a pause is consumed.  Note that this is only to reduce
	// memory pressure;  a pause is consumed once atomically.  If removal fails, a dangling false positive
	// is left in the tree which increases the amount of work we have to do when matching but does NOT
	// impact execution.
	RemovePause(ctx context.Context, pause expr.Evaluable) error
}

type aggregator struct {
	log *slog.Logger

	records *ccache.Cache

	loader    EvaluableLoader
	parser    expr.TreeParser
	evaluator expr.ExpressionEvaluator
}

func (a aggregator) EvaluateAsyncEvent(ctx context.Context, event event.TrackedEvent) ([]expr.Evaluable, int32, error) {
	name := event.GetEvent().Name
	eval, err := a.LoadEventEvaluator(ctx, event.GetWorkspaceID(), name)
	if err != nil {
		return nil, 0, fmt.Errorf("Could not load an event evaluator: %w", err)
	}

	found, evalCount, err := eval.Evaluate(ctx, map[string]any{
		"async": event.GetEvent().Map(),
	})
	if err != nil {
		a.log.Error(
			"error evaluating aggregate expressions",
			"workspace_id", event.GetWorkspaceID(),
			"event", name,
			"error", err,
		)
		return found, evalCount, err

	}

	a.log.Debug(
		"evaluated aggregate expressions",
		"workspace_id", event.GetWorkspaceID(),
		"event", name,
		"eval_count", evalCount,
		"matched_count", len(found),
	)

	return found, evalCount, err
}

func (a aggregator) LoadEventEvaluator(ctx context.Context, wsID uuid.UUID, eventName string) (expr.AggregateEvaluator, error) {
	key := wsID.String() + ":" + eventName

	var bk *bookkeeper

	val := a.records.Get(key)
	if val == nil {
		bk = &bookkeeper{
			wsID:  wsID,
			event: eventName,
			ae:    expr.NewAggregateEvaluator(a.parser, a.evaluator),
			// updatedAt is a zero time.
		}

		// The time doesn't matter as ccache is an LRU which does not autoamtically GC expired
		// content;  it always serves stale content and only deletes when the cache is full.
		a.records.Set(key, bk, time.Hour*3)
	} else {
		bk = val.Value().(*bookkeeper)
		val.Extend(time.Hour * 3)
	}

	if err := bk.update(ctx, a.loader); err != nil {
		if bk.updatedAt.IsZero() {
			return nil, fmt.Errorf("could not load evaluables for aggregate evaluator")
		}
		// This means we had an error updating the aggregate evaluator with latest events;
		// matching will be stale.
		a.log.Warn(
			"using stale evaluator",
			"error", err,
			"age", time.Since(bk.updatedAt),
			"workspace_id", wsID,
			"event", eventName,
		)
		return bk.ae, nil
	}

	return bk.ae, nil
}

func (a aggregator) RemovePause(ctx context.Context, event expr.Evaluable) error {
	return fmt.Errorf("not implemented")
}

// bookkeeper manages an aggregator for an event name and records the time that the aggregator
// was last updated.  This allows us to fetch pauses stored since the last update time for
// a given workspace event.
type bookkeeper struct {
	wsID      uuid.UUID
	event     string
	ae        expr.AggregateEvaluator
	updatedAt time.Time
}

func (b *bookkeeper) update(ctx context.Context, l EvaluableLoader) error {
	at := time.Now()
	err := l.LoadEvaluablesSince(ctx, b.wsID, b.event, b.updatedAt, func(ctx context.Context, eval expr.Evaluable) error {
		_, err := b.ae.Add(ctx, eval)
		return err
	})
	if err == nil {
		b.updatedAt = at
	}
	return err
}
