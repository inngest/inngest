package run

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
	"golang.org/x/sync/errgroup"
)

var (
	eventRegex  = regexp.MustCompile(`^event\..+`)
	outputRegex = regexp.MustCompile(`^output\.`)
	errorRegex  = regexp.MustCompile(`^error\.`)

	exprErrorRegex = regexp.MustCompile(`^ERROR: <input>:\d+:\d+:\s+`)
)

type ExprHandlerOpt func(ctx context.Context, h *ExpressionHandler) error
type ExprSQLConverter func(ctx context.Context, n *expr.Node) ([]sq.Expression, error)

func WithExpressionHandlerExpressions(cel []string) ExprHandlerOpt {
	return func(ctx context.Context, h *ExpressionHandler) error {
		if len(cel) == 0 {
			return nil
		}

		return h.add(ctx, cel)
	}
}

func WithExpressionHandlerBlob(exp string, delimiter string) ExprHandlerOpt {
	if delimiter == "" {
		delimiter = "\n"
	}
	cel := strings.Split(exp, delimiter)

	return func(ctx context.Context, h *ExpressionHandler) error {
		if exp == "" || len(cel) == 0 {
			return nil
		}

		return h.add(ctx, cel)
	}
}

func WithExpressionSQLConverter(c ExprSQLConverter) ExprHandlerOpt {
	return func(ctx context.Context, h *ExpressionHandler) error {
		h.SQLConverter = c
		return nil
	}
}

type ExpressionHandler struct {
	EventExprList  []string
	OutputExprList []string
	SQLConverter   ExprSQLConverter
}

func NewExpressionHandler(ctx context.Context, opts ...ExprHandlerOpt) (*ExpressionHandler, error) {
	h := &ExpressionHandler{
		EventExprList:  []string{},
		OutputExprList: []string{},
		SQLConverter:   EventFieldConverter,
	}

	for _, apply := range opts {
		if err := apply(ctx, h); err != nil {
			return nil, err
		}
	}

	return h, nil
}

// add adds the list of CEL strings passed in and store them in the list of expressions.
//
// currently not expecting to use nesting within a string, but if needed, this function
// should change to use recursion for importing the list of expressions instead.
func (h *ExpressionHandler) add(ctx context.Context, cel []string) error {
	parser := expressions.ParserSingleton()

	evtExprs := map[string]bool{}
	outputExprs := map[string]bool{}

	for _, e := range cel {
		// empty string, skip
		for len(e) == 0 {
			continue
		}

		// parse and validate
		tree, err := parser.Parse(ctx, expr.StringExpression(e))
		if err != nil {
			// reformat the error message to be more comprehensive when propagated back to the user
			errs := strings.Split(err.Error(), "\n")
			if len(errs) == 1 {
				return err
			}

			// Only take the first one, the rest is not needed.
			// then remove the prefix `ERROR : <input>:1:\d:` and use the rest of the error body
			msg := exprErrorRegex.ReplaceAllString(errs[0], "")
			return fmt.Errorf("%s\n | %s", msg, e)
		}
		if tree.HasMacros {
			return fmt.Errorf("macros are currently not supported")
		}
		// NOTE: if there are no predicates or AND or OR
		// it means an invalid syntax was used and it couldn't parse anything
		if !tree.Root.HasPredicate() && tree.Root.Ands == nil && tree.Root.Ors == nil {
			return fmt.Errorf("invalid syntax detected")
		}

		h.addToExprList(ctx, []*expr.Node{&tree.Root}, e, evtExprs, outputExprs)
	}

	for evt := range evtExprs {
		h.EventExprList = append(h.EventExprList, evt)
	}

	for output := range outputExprs {
		h.OutputExprList = append(h.OutputExprList, output)
	}

	return nil
}

func (h *ExpressionHandler) addToExprList(
	ctx context.Context,
	nodes []*expr.Node,
	cel string,
	evtDedup map[string]bool,
	outputDedup map[string]bool,
) {
	for _, n := range nodes {
		if n.HasPredicate() {
			switch {
			case eventRegex.MatchString(n.Predicate.Ident):
				if _, ok := evtDedup[cel]; !ok {
					evtDedup[cel] = true
				}
			case outputRegex.MatchString(n.Predicate.Ident), errorRegex.MatchString(n.Predicate.Ident):
				// Both output.* and error.* CEL filters look at spans.output in the database
				if _, ok := outputDedup[cel]; !ok {
					outputDedup[cel] = true
				}
			}
		}

		if n.Ands != nil {
			h.addToExprList(ctx, n.Ands, cel, evtDedup, outputDedup)
		}
		if n.Ors != nil {
			h.addToExprList(ctx, n.Ors, cel, evtDedup, outputDedup)
		}
	}
}

func (h *ExpressionHandler) HasFilters() bool {
	return h.HasEventFilters() || h.HasOutputFilters()
}

func (h *ExpressionHandler) HasEventFilters() bool {
	return len(h.EventExprList) > 0
}

func (h *ExpressionHandler) HasOutputFilters() bool {
	return len(h.OutputExprList) > 0
}

func (h *ExpressionHandler) ToSQLFilters(ctx context.Context) ([]sq.Expression, error) {
	filters := []sq.Expression{}
	parser := expressions.ParserSingleton()

	// used to dedup in case there's an expression that is included in both list
	dedup := map[string]bool{}
	exprs := []string{}
	for _, e := range h.EventExprList {
		if _, ok := dedup[e]; !ok {
			dedup[e] = true
			exprs = append(exprs, e)
		}
	}
	for _, e := range h.OutputExprList {
		if _, ok := dedup[e]; !ok {
			dedup[e] = true
			exprs = append(exprs, e)
		}
	}

	for _, exp := range exprs {
		tree, err := parser.Parse(ctx, expr.StringExpression(exp))
		if err != nil {
			return nil, fmt.Errorf("error evaluating event expression '%s': %w", exp, err)
		}

		expFilter, err := h.toSQLFilters(ctx, []*expr.Node{&tree.Root})
		if err != nil {
			return nil, err
		}
		filters = append(filters, expFilter...)
	}

	return filters, nil
}

func (h *ExpressionHandler) MatchEventExpressions(ctx context.Context, evt event.Event) (bool, error) {
	if !h.HasEventFilters() {
		return false, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]bool, len(h.EventExprList))
	data := evt.Map()

	for i, e := range h.EventExprList {
		idx := i
		exp := e

		eg.Go(func() error {
			eval, err := expressions.NewBooleanEvaluator(ctx, exp)
			if err != nil {
				return fmt.Errorf("error initializing expression evaluator for event: %w", err)
			}

			ok, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": data}))
			if err != nil {
				// if there's an error, it likely means the data being matched is not of the same structure
				// map[string]any vs int64
				res[idx] = false
				return nil
			}

			res[idx] = ok
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return allMatches(res), nil
}

func (h *ExpressionHandler) MatchOutputExpressions(ctx context.Context, output []byte) (bool, error) {
	if !h.HasOutputFilters() {
		return false, nil
	}
	// no output to match against, don't waste effort
	if string(output) == "" {
		return false, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]bool, len(h.OutputExprList))

	var result any
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("error deserializing output: %w", err)
	}

	var data map[string]any
	switch v := result.(type) {
	case map[string]any:
		data = map[string]any{"output": v}
	case []any:
		data = map[string]any{"output": v}
	case int64, float64, bool, string:
		data = map[string]any{"output": v}
	}

	for i, e := range h.OutputExprList {
		idx := i
		exp := e

		eg.Go(func() error {
			eval, err := expressions.NewBooleanEvaluator(ctx, exp)
			if err != nil {
				return fmt.Errorf("error initializing expression evaluator for output: %w", err)
			}

			ok, err := eval.Evaluate(ctx, expressions.NewData(data))
			if err != nil {
				// if there's an error, it likely means the data being matched is not of the same structure
				// map[string]any vs int64
				res[idx] = false
				return nil
			}

			res[idx] = ok
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return allMatches(res), nil
}

func allMatches(res []bool) bool {
	for _, v := range res {
		if !v {
			return false
		}
	}
	return true
}

// toSQLFilters parses the passed in nodes and converts them into SQL filter expressions
func (h *ExpressionHandler) toSQLFilters(ctx context.Context, nodes []*expr.Node) ([]sq.Expression, error) {
	filters := []sq.Expression{}

	for _, n := range nodes {
		res, err := h.SQLConverter(ctx, n)
		if err != nil {
			return nil, err
		}
		filters = append(filters, res...)

		// check for further nesting
		if n.Ands != nil {
			nested, err := h.toSQLFilters(ctx, n.Ands)
			if err != nil {
				return nil, err
			}

			switch len(nested) {
			case 0: // no op
			case 1:
				filters = append(filters, nested[0])
			default:
				filters = append(filters, sq.And(nested...))
			}
		}

		if n.Ors != nil {
			nested, err := h.toSQLFilters(ctx, n.Ors)
			if err != nil {
				return nil, err
			}

			switch len(nested) {
			case 0: // no op
			case 1:
				filters = append(filters, nested[0])
			default:
				filters = append(filters, sq.Or(nested...))
			}
		}
	}

	return filters, nil
}
