package run

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/cel-go/common/operators"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/event"
	"github.com/inngest/inngest/pkg/expressions"
	"golang.org/x/sync/errgroup"
)

var (
	eventRegex  = regexp.MustCompile(`^event\..+`)
	outputRegex = regexp.MustCompile(`^output`)
)

type ExprHandlerOpt func(ctx context.Context, h *ExpressionHandler) error

func WithExpressionHandlerExpressions(cel []string) ExprHandlerOpt {
	return func(ctx context.Context, h *ExpressionHandler) error {
		return h.add(ctx, cel)
	}
}

func WithExpressionHandlerBlob(exp string, delimiter string) ExprHandlerOpt {
	if delimiter == "" {
		delimiter = "\n"
	}
	cel := strings.Split(exp, delimiter)

	return func(ctx context.Context, h *ExpressionHandler) error {
		return h.add(ctx, cel)
	}
}

type ExpressionHandler struct {
	EventExprList  []string
	OutputExprList []string
}

func NewExpressionHandler(ctx context.Context, opts ...ExprHandlerOpt) (*ExpressionHandler, error) {
	h := &ExpressionHandler{
		EventExprList:  []string{},
		OutputExprList: []string{},
	}

	for _, apply := range opts {
		if err := apply(ctx, h); err != nil {
			return nil, err
		}
	}

	return h, nil
}

func (h *ExpressionHandler) add(ctx context.Context, cel []string) error {
	parser := expressions.ParserSingleton()

	for _, e := range cel {
		tree, err := parser.Parse(ctx, expr.StringExpression(e))
		if err != nil {
			return fmt.Errorf("error parsing expression '%s': %w", e, err)
		}

		if tree.Root.HasPredicate() {
			switch {
			case eventRegex.MatchString(tree.Root.Predicate.Ident):
				h.EventExprList = append(h.EventExprList, e)
			case outputRegex.MatchString(tree.Root.Predicate.Ident):
				h.OutputExprList = append(h.OutputExprList, e)
			}
		}
	}

	return nil
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

// create pre-filters for database queries
// - ULID
// - event id (idempotency key)
// - event name
// - version
// - timestamp
func (h *ExpressionHandler) ToSQLEventFilters(ctx context.Context) ([]sq.Expression, error) {
	filters := []sq.Expression{}
	parser := expressions.ParserSingleton()

	for _, exp := range h.EventExprList {
		tree, err := parser.Parse(ctx, expr.StringExpression(exp))
		if err != nil {
			return nil, fmt.Errorf("error evaluating event expression '%s': %w", exp, err)
		}

		expFilter, err := toSQLEventFilters(ctx, []*expr.Node{&tree.Root})
		if err != nil {
			return nil, err
		}
		filters = append(filters, expFilter...)
	}

	return filters, nil
}

func (h *ExpressionHandler) MatchEventExpressions(ctx context.Context, evt event.Event) (bool, error) {
	eg := errgroup.Group{}
	res := make([]bool, len(h.EventExprList))
	data := evt.Map()

	for i, exp := range h.EventExprList {
		idx := i

		eg.Go(func() error {
			eval, err := expressions.NewBooleanEvaluator(ctx, exp)
			if err != nil {
				return fmt.Errorf("error initializing expression evaluator for event: %w", err)
			}

			ok, _, err := eval.Evaluate(ctx, expressions.NewData(map[string]any{"event": data}))
			if err != nil {
				return fmt.Errorf("error evaluating event expression: %w", err)
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

// toSQLEventFilter parses the passed in nodes and converts them into SQL filter expressions
func toSQLEventFilters(ctx context.Context, nodes []*expr.Node) ([]sq.Expression, error) {
	filters := []sq.Expression{}

	for _, n := range nodes {
		if n.HasPredicate() {
			literal := n.Predicate.Literal

			switch n.Predicate.Ident {
			case "event.id":
				id, ok := literal.(string)
				if !ok {
					return nil, fmt.Errorf("expects 'event.id' to be a string: %v", literal)
				}
				switch n.Predicate.Operator {
				case operators.Equals:
					filters = append(filters, sq.C("event_id").Eq(id))
				case operators.NotEquals:
					filters = append(filters, sq.C("event_id").Neq(id))
				}
			case "event.name":
				name, ok := literal.(string)
				if !ok {
					return nil, fmt.Errorf("expects 'event.name' to be a string: %v", literal)
				}
				switch n.Predicate.Operator {
				case operators.Equals:
					filters = append(filters, sq.C("event_name").Eq(name))
				case operators.NotEquals:
					filters = append(filters, sq.C("event_name").Neq(name))
				}
			case "event.ts":
				ts, ok := literal.(int64)
				if !ok {
					return nil, fmt.Errorf("expects 'event.ts' to be an integer: %v", literal)
				}
				var f sq.Expression
				field := "event_ts"

				switch n.Predicate.Operator {
				case operators.Greater:
					f = sq.C(field).Gt(ts)
				case operators.GreaterEquals:
					f = sq.C(field).Gte(ts)
				case operators.Equals:
					f = sq.C(field).Eq(ts)
				case operators.Less:
					f = sq.C(field).Lt(ts)
				case operators.LessEquals:
					f = sq.C(field).Lte(ts)
				case operators.NotEquals:
					f = sq.C(field).Neq(ts)
				}
				if f != nil {
					filters = append(filters, f)
				}
			case "event.v":
				v, ok := literal.(string)
				if !ok {
					return nil, fmt.Errorf("expects 'event.v' to be a string: %v", literal)
				}
				switch n.Predicate.Operator {
				case operators.Equals:
					filters = append(filters, sq.C("event_v").Eq(v))
				case operators.NotEquals:
					filters = append(filters, sq.C("event_v").Neq(v))
				}
			}
		}

		// check for further nesting
		if n.Ands != nil {
			nested, err := toSQLEventFilters(ctx, n.Ands)
			if err != nil {
				return nil, err
			}
			filters = append(filters, sq.And(nested...))
		}

		if n.Ors != nil {
			nested, err := toSQLEventFilters(ctx, n.Ors)
			if err != nil {
				return nil, err
			}
			filters = append(filters, sq.Or(nested...))
		}
	}

	return filters, nil
}
