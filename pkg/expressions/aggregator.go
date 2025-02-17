package expressions

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/inngest/inngest/pkg/telemetry/metrics"
	"github.com/karlseguin/ccache/v2"
)

type EventEvaluable interface {
	expr.Evaluable
	GetEvent() *string
	GetWorkspaceID() uuid.UUID
}

func NewAggregator(
	ctx context.Context,
	size int64,
	concurrency int64,
	loader EvaluableLoader,
	log *slog.Logger,
) Aggregator {
	// use the package's singleton caching parser to create a new tree parser.
	// this uses lifted expression parsing with caching for speed.
	parser := expr.NewTreeParser(exprCompiler)
	if log == nil {
		log = logger.StdlibLogger(ctx)
	}
	return &aggregator{
		log:         log,
		concurrency: concurrency,
		records:     ccache.New(ccache.Configure().MaxSize(size).ItemsToPrune(uint32(size) / 4)),
		loader:      loader,
		parser:      parser,
		// use the package's exprEvaluator function as the actual logic which evaluates
		// expressions after the aggregate evaluator does matching.
		evaluator: exprEvaluator,
		mapLock:   &sync.Mutex{},
		locks:     map[string]*sync.Mutex{},
	}
}

// EvaluableLoader loads all Evaluables from a store since the given time, invoking the given do function for each
// event.
//
// It's expected that this wraps the state.PauseGetter interface, calling `do` for each item in the PauseIterator.
// The types are different as we must use the open source expr.Evaluable interfaces with aggregate evaluation.
type EvaluableLoader interface {
	LoadEvaluablesSince(ctx context.Context, workspaceID uuid.UUID, eventName string, since time.Time, do func(context.Context, expr.Evaluable) error) error
	EvaluablesByID(ctx context.Context, evaluableIDs ...uuid.UUID) ([]expr.Evaluable, error)
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
	LoadEventEvaluator(ctx context.Context, wsID uuid.UUID, eventName string, eventTS time.Time) (expr.AggregateEvaluator, error)

	// RemovePause is a shortcut to find an event evaluator _without_ refreshing new data, and to
	// remove the pause's expressions from any aggregate trees.
	//
	// This must be called by the executor when a pause is consumed.  Note that this is only to reduce
	// memory pressure;  a pause is consumed once atomically.  If removal fails, a dangling false positive
	// is left in the tree which increases the amount of work we have to do when matching but does NOT
	// impact execution.
	RemovePause(ctx context.Context, pause EventEvaluable) error
}

type aggregator struct {
	log *slog.Logger

	records *ccache.Cache

	concurrency int64
	loader      EvaluableLoader
	parser      expr.TreeParser
	evaluator   expr.ExpressionEvaluator

	mapLock *sync.Mutex
	locks   map[string]*sync.Mutex
}

func (a *aggregator) EvaluateAsyncEvent(ctx context.Context, event event.TrackedEvent) ([]expr.Evaluable, int32, error) {
	log := logger.StdlibLogger(ctx)

	log.Debug("loading evaluator")

	name := event.GetEvent().Name
	eval, err := a.LoadEventEvaluator(ctx, event.GetWorkspaceID(), name, event.GetEvent().Time())
	if err != nil {
		return nil, 0, fmt.Errorf("Could not load an event evaluator: %w", err)
	}

	if eval.SlowLen() > 100 {
		log.Warn(
			"evaluating aggregate pauses",
			"warning", "large number of slow pauses",
			"workspace_id", event.GetWorkspaceID(),
			"event", name,
			"error", err,
			"slow_expression_len", eval.SlowLen(),
			"mixed_expression_len", eval.MixedLen(),
			"fast_expression_len", eval.FastLen(),
		)

	} else {
		log.Debug(
			"evaluating aggregate pauses",
			"workspace_id", event.GetWorkspaceID(),
			"event", name,
			"error", err,
			"slow_expression_len", eval.SlowLen(),
			"mixed_expression_len", eval.MixedLen(),
			"fast_expression_len", eval.FastLen(),
		)
	}

	wsID := event.GetWorkspaceID()

	start := time.Now()
	found, evalCount, err := eval.Evaluate(ctx, map[string]any{
		"async": event.GetEvent().Map(),
	})
	metrics.HistogramAggregatePausesEvalDuration(ctx, time.Since(start).Milliseconds(), metrics.HistogramOpt{
		PkgName: pkgName,
		Tags: map[string]any{
			"workspaceID": wsID.String(),
			"success":     err == nil,
		},
	})
	if err != nil {
		log.Error(
			"error evaluating aggregate expressions",
			"workspace_id", event.GetWorkspaceID(),
			"event", name,
			"error", err,
		)
		return found, evalCount, err

	}

	metrics.IncrAggregatePausesFoundCounter(ctx, int64(len(found)), metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"workspaceID": wsID.String()},
	})
	metrics.IncrAggregatePausesEvaluatedCounter(ctx, int64(evalCount), metrics.CounterOpt{
		PkgName: pkgName,
		Tags:    map[string]any{"workspaceID": wsID.String()},
	})

	log.Debug(
		"evaluated aggregate expressions",
		"workspace_id", wsID,
		"event", name,
		"eval_count", evalCount,
		"matched_count", len(found),
		"total_count", eval.Len(),
		"found_count", len(found),
		"slow_expression_len", eval.SlowLen(),
		"mixed_expression_len", eval.MixedLen(),
		"fast_expression_len", eval.FastLen(),
	)

	return found, evalCount, err
}

func (a *aggregator) LoadEventEvaluator(ctx context.Context, wsID uuid.UUID, eventName string, eventTS time.Time) (expr.AggregateEvaluator, error) {
	key := wsID.String() + ":" + eventName

	a.mapLock.Lock()
	lock, ok := a.locks[key]
	if !ok {
		a.locks[key] = &sync.Mutex{}
		lock = a.locks[key]
	}
	a.mapLock.Unlock()
	// lock this key to prevent a possible race where multiple goroutines could see that
	// the bookkeeper is missing from the cache and try to create it at the same time,
	// meaning we would drop one of those bookkeepers:
	lock.Lock()
	defer lock.Unlock()

	var bk *bookkeeper

	val := a.records.Get(key)
	if val == nil {
		bk = &bookkeeper{
			wsID:  wsID,
			event: eventName,
			ae:    expr.NewAggregateEvaluator(a.parser, a.evaluator, a.loader.EvaluablesByID, a.concurrency),
			// updatedAt is a zero time.
		}

		// The time doesn't matter as ccache is an LRU which does not autoamtically GC expired
		// content;  it always serves stale content and only deletes when the cache is full.
		a.records.Set(key, bk, time.Hour*3)
	} else {
		bk = val.Value().(*bookkeeper)
		val.Extend(time.Hour * 3)
	}

	if bk.updatedAt.After(eventTS) {
		// We can use a stale executor here, as the pauses were updated after the event
		// was received.  This prevents spinning on locks.
		return bk.ae, nil
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
			"age_ms", time.Since(bk.updatedAt).Milliseconds(),
			"workspace_id", wsID,
			"event", eventName,
		)
		return bk.ae, nil
	}

	return bk.ae, nil
}

func (a *aggregator) getBookkeeper(ctx context.Context, wsID uuid.UUID, eventName string) *bookkeeper {
	key := wsID.String() + ":" + eventName
	var bk *bookkeeper
	val := a.records.Get(key)
	if val == nil {
		return nil
	}
	bk = val.Value().(*bookkeeper)
	return bk
}

func (a *aggregator) RemovePause(ctx context.Context, pause EventEvaluable) error {
	if pause.GetEvent() == nil {
		return fmt.Errorf("cannot remove non-pause evaluable")
	}

	bk := a.getBookkeeper(ctx, pause.GetWorkspaceID(), *pause.GetEvent())
	if bk == nil {
		return nil
	}

	return bk.ae.Remove(ctx, pause)
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
	count := 0

	err := l.LoadEvaluablesSince(ctx, b.wsID, b.event, b.updatedAt, func(ctx context.Context, eval expr.Evaluable) error {
		if eval == nil {
			return fmt.Errorf("adding nil pause")
		}
		_, err := b.ae.Add(ctx, eval)
		if err == nil {
			count++
		}
		return err
	})

	logger.StdlibLogger(ctx).Debug(
		"updated evaluator",
		"delta_ms", at.Sub(b.updatedAt).Milliseconds(),
		"count", count,
		"error", err,
	)

	if err == nil {
		b.updatedAt = at
	}
	return err
}
