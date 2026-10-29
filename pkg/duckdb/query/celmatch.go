package query

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
	"github.com/inngest/inngest/pkg/run"
)

// The SQL that CELEventFilters/CELOutputFilters/CELEventTableFilters build
// from a CEL expression is only a superset prefilter: inngest/expr's tree
// (which it's converted from) can't represent every expression, so any
// predicate that can't be converted soundly is left unconstrained. Like
// pkg/cqrs/manager, every candidate row is re-checked in Go, and since
// that breaks LIMIT pushdown, candidates are fetched in keyset-paginated
// batches until the requested page is full or the rows run out.

// celCandidateBatch is how many candidate rows a CEL-filtered query fetches
// per round trip for a page of items: a few pages' worth (so a selective
// post-filter doesn't take a round trip per page), but at least
// celMinCandidateBatch.
func celCandidateBatch(items uint) uint {
	return max(items*4, celMinCandidateBatch)
}

const celMinCandidateBatch = 256

// runCELMatcher re-checks a run candidate the same way pkg/cqrs/manager's
// getTraceRunsFromTable does, with the same evaluator (and the same "an
// evaluation error is a non-match" rule) as
// ExpressionHandler.MatchEventExpressions/MatchOutputExpressions:
//
//   - every expression in EventExprList is evaluated with only event bound,
//     and the run matches when some single triggering event satisfies all of
//     them (the manager's GetEventsByExpressions + HasEventIDs), so a run
//     with no triggering event (e.g. a cron run) never matches.
//   - every expression in OutputExprList is evaluated with only output/
//     error bound, and a run with no output yet never matches.
//
// An expression mixing event.* and output.* lands in both lists, so like
// the manager, event.name == "x" || output.ok == true needs both a
// triggering event named "x" and a true output.ok (each half sees the other
// namespace as null), and a mixed && never matches.
//
// MatchEventExpressions/MatchOutputExpressions can't be called directly
// here, though: they re-create every evaluator per call, and
// MatchOutputExpressions binds output to the whole stored value, while
// inngest.runs.output holds a {"data":T}/{"error":T} envelope whose data
// and error halves the DuckDB SQL scope exposes as output.* and error.*.
type runCELMatcher struct {
	eventEvals, outputEvals []expressions.BooleanEvaluator
}

// newRunCELMatcher returns nil when h has nothing to match.
func newRunCELMatcher(ctx context.Context, h *run.ExpressionHandler) (*runCELMatcher, error) {
	if !h.HasFilters() {
		return nil, nil
	}
	m := &runCELMatcher{}
	var err error
	if m.eventEvals, err = newCELEvaluators(ctx, h.EventExprList); err != nil {
		return nil, err
	}
	if m.outputEvals, err = newCELEvaluators(ctx, h.OutputExprList); err != nil {
		return nil, err
	}
	return m, nil
}

func newCELEvaluators(ctx context.Context, exprs []string) ([]expressions.BooleanEvaluator, error) {
	out := make([]expressions.BooleanEvaluator, 0, len(exprs))
	for _, e := range exprs {
		eval, err := expressions.NewBooleanEvaluator(ctx, e)
		if err != nil {
			return nil, fmt.Errorf("duckdbquery: initializing CEL evaluator for %q: %w", e, err)
		}
		out = append(out, eval)
	}
	return out, nil
}

// match reports whether a run with the given inputs (inngest.runs.inputs,
// a JSON array of triggering events) and output (inngest.runs.output)
// passes every expression.
func (m *runCELMatcher) match(ctx context.Context, inputs, output []byte) (bool, error) {
	if len(m.outputEvals) > 0 {
		if len(output) == 0 {
			return false, nil
		}
		vars := map[string]any{}
		var envelope map[string]any
		// A non-object output has no data/error half to bind, so output.*
		// and error.* just see nothing, same as the SQL scope's $.data/
		// $.error paths would.
		if err := json.Unmarshal(output, &envelope); err == nil {
			if v, ok := envelope["data"]; ok {
				vars["output"] = v
			}
			if v, ok := envelope["error"]; ok {
				vars["error"] = v
			}
		}
		if !matchAll(ctx, m.outputEvals, vars) {
			return false, nil
		}
	}

	if len(m.eventEvals) == 0 {
		return true, nil
	}
	var evts []json.RawMessage
	if len(inputs) > 0 {
		if err := json.Unmarshal(inputs, &evts); err != nil {
			return false, fmt.Errorf("duckdbquery: decoding run inputs: %w", err)
		}
	}
	for _, raw := range evts {
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if matchAll(ctx, m.eventEvals, map[string]any{"event": evt.Map()}) {
			return true, nil
		}
	}
	return false, nil
}

func matchAll(ctx context.Context, evals []expressions.BooleanEvaluator, vars map[string]any) bool {
	data := expressions.NewData(vars)
	for _, eval := range evals {
		ok, err := eval.Evaluate(ctx, data)
		if err != nil || !ok {
			return false
		}
	}
	return true
}
