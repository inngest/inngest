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

// runCELMatcher re-checks a run candidate against a run.ExpressionHandler's
// expressions, with the same evaluator (and the same "an evaluation error
// is a non-match" rule) as ExpressionHandler.MatchEventExpressions/
// MatchOutputExpressions.
//
// Those two can't be called directly here, though: each binds only one
// namespace, so an expression mixing event.* and output.* (which
// celExprsToSQL deliberately supports) could never match, and
// MatchOutputExpressions binds output to the whole stored value, while
// inngest.runs.output holds a {"data":T}/{"error":T} envelope whose data
// and error halves the DuckDB SQL scope exposes as output.* and error.*.
// So every expression is evaluated with event bound to one triggering
// event (the run matches when some single triggering event satisfies every
// expression, the same "one event matches all event expressions" rule
// pkg/cqrs/manager applies via GetEventsByExpressions) and output/error
// bound to the envelope's halves.
type runCELMatcher struct {
	evals []expressions.BooleanEvaluator
	// hasEvent and hasOutput mirror pkg/cqrs/manager never matching a run
	// with no triggering event against an event.* expression, or a run
	// with no output against an output.*/error.* one.
	hasEvent, hasOutput bool
}

// newRunCELMatcher returns nil when h has nothing to match.
func newRunCELMatcher(ctx context.Context, h *run.ExpressionHandler) (*runCELMatcher, error) {
	if !h.HasFilters() {
		return nil, nil
	}
	m := &runCELMatcher{hasEvent: h.HasEventFilters(), hasOutput: h.HasOutputFilters()}
	seen := map[string]bool{}
	for _, list := range [][]string{h.EventExprList, h.OutputExprList} {
		for _, e := range list {
			if seen[e] {
				continue
			}
			seen[e] = true
			eval, err := expressions.NewBooleanEvaluator(ctx, e)
			if err != nil {
				return nil, fmt.Errorf("duckdbquery: initializing CEL evaluator for %q: %w", e, err)
			}
			m.evals = append(m.evals, eval)
		}
	}
	return m, nil
}

// match reports whether a run with the given inputs (inngest.runs.inputs,
// a JSON array of triggering events) and output (inngest.runs.output)
// satisfies every expression.
func (m *runCELMatcher) match(ctx context.Context, inputs, output []byte) (bool, error) {
	if m.hasOutput && len(output) == 0 {
		return false, nil
	}

	vars := map[string]any{}
	if len(output) > 0 {
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
	}

	var evts []json.RawMessage
	if len(inputs) > 0 {
		if err := json.Unmarshal(inputs, &evts); err != nil {
			return false, fmt.Errorf("duckdbquery: decoding run inputs: %w", err)
		}
	}
	if len(evts) == 0 {
		// e.g. a cron run, with no event to bind at all
		if m.hasEvent {
			return false, nil
		}
		return m.matchVars(ctx, vars)
	}
	for _, raw := range evts {
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		vars["event"] = evt.Map()
		ok, err := m.matchVars(ctx, vars)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}

func (m *runCELMatcher) matchVars(ctx context.Context, vars map[string]any) (bool, error) {
	data := expressions.NewData(vars)
	for _, eval := range m.evals {
		ok, err := eval.Evaluate(ctx, data)
		if err != nil || !ok {
			return false, nil
		}
	}
	return true, nil
}
