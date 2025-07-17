package event_cel

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	sq "github.com/doug-martin/goqu/v9"
	"github.com/google/cel-go/common/operators"
	"github.com/inngest/expr"
	"github.com/inngest/inngest/pkg/expressions"
)

var (
	eventRegex = regexp.MustCompile(`^event\..+`)

	exprErrorRegex = regexp.MustCompile(`^ERROR: <input>:\d+:\d+:\s+`)
)

type ExprHandlerOpt func(ctx context.Context, h *ExpressionHandler) error
type ExprSQLConverter func(ctx context.Context, n *expr.Node) ([]sq.Expression, error)

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
	EventExprList []string
	SQLConverter  ExprSQLConverter
}

func NewExpressionHandler(ctx context.Context, opts ...ExprHandlerOpt) (*ExpressionHandler, error) {
	h := &ExpressionHandler{
		EventExprList: []string{},
		SQLConverter:  SQLiteConverter,
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

		err = h.addToExprList(ctx, []*expr.Node{&tree.Root}, e, evtExprs)
		if err != nil {
			return err
		}
	}

	for evt := range evtExprs {
		h.EventExprList = append(h.EventExprList, evt)
	}

	return nil
}

func (h *ExpressionHandler) addToExprList(
	ctx context.Context,
	nodes []*expr.Node,
	cel string,
	evtDedup map[string]bool,
) error {
	for _, n := range nodes {
		if n.HasPredicate() {
			switch {
			case eventRegex.MatchString(n.Predicate.Ident):
				if _, ok := evtDedup[cel]; !ok {
					evtDedup[cel] = true
				}
			default:
				return fmt.Errorf("unsupported filter %s", n.Predicate.Ident)
			}
		}

		if n.Ands != nil {
			err := h.addToExprList(ctx, n.Ands, cel, evtDedup)
			if err != nil {
				return err
			}
		}
		if n.Ors != nil {
			err := h.addToExprList(ctx, n.Ors, cel, evtDedup)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *ExpressionHandler) HasFilters() bool {
	return h.HasDataFilters()
}

func (h *ExpressionHandler) HasDataFilters() bool {
	return len(h.EventExprList) > 0
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

	for _, exp := range exprs {
		tree, err := parser.Parse(ctx, expr.StringExpression(exp))
		if err != nil {
			return nil, fmt.Errorf("error evaluating expression '%s': %w", exp, err)
		}

		expFilter, err := h.toSQLFilters(ctx, []*expr.Node{&tree.Root})
		if err != nil {
			return nil, err
		}
		filters = append(filters, expFilter...)
	}

	return filters, nil
}

// toSQLFilter parses the passed in nodes and converts them into SQL filter expressions
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

func SQLiteConverter(ctx context.Context, n *expr.Node) ([]sq.Expression, error) {
	filters := []sq.Expression{}
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
			default:
				return nil, fmt.Errorf("unsupported operator %s", n.Predicate.Operator)
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
			default:
				return nil, fmt.Errorf("unsupported operator %s", n.Predicate.Operator)
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
			default:
				return nil, fmt.Errorf("unsupported operator %s", n.Predicate.Operator)
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
			default:
				return nil, fmt.Errorf("unsupported operator %s", n.Predicate.Operator)
			}
		default:
			return nil, fmt.Errorf("unsupported field %s", n.Predicate.Ident)
		}
	}

	return filters, nil
}
