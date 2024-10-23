package expressions

import (
	"context"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/inngest/expr"
)

// Interpolated interpolates `event.data.foo` variables used within expressions.  Cancellation,
// for example, is defined without knowing the shape or values of `event.data.${var}`.  We only
// know this when instantiating a new function run.
//
// However, for fast parsing we want to interpolate these variables when saving the expressions,
// else we can't add the values to aggregate trees.
//
// If this returns an error, it also returns the original expression.  Note that if the
// expression doesn't contain `event.`, we won't interpolate at all.
func Interpolate(ctx context.Context, e string, data map[string]any) (string, error) {
	if !strings.Contains(e, "event.") {
		// Only interpolate events.
		return e, nil
	}

	// This is a one-off used to interpolate expressions, and so we do not need to cache
	// this.
	//
	// NOTE: If this used the lifting parser things would break massively!  We would
	// lift the same expression twice, resulting in two different values referencing
	// "vars.a".
	eval, err := cachedCompile(ctx, e)
	if err != nil {
		return e, err
	}

	ast, err := residual(ctx, eval.ast, eval.env, data)
	if err != nil {
		return e, err
	}

	// XXX: Here, any `event` variables that have yet to be defined are null.  We can
	// do two things:  something hacky, eg. regex, to replace, or we can parse the
	// expression ourselves and reformat using our expr library.  Let's do that.

	interpolated := ast.Source().Content()
	if !strings.Contains(interpolated, "event.") {
		// This doesn't contain event. vars, so ignore and return to optimize.
		return interpolated, nil
	}

	p := ParserSingleton()
	if p == nil {
		// Can't parse.
		return interpolated, nil
	}

	parsed, err := p.Parse(ctx, expr.StringExpression(interpolated))
	if err != nil {
		return interpolated, nil
	}

	// Walk the expression, and update the parsed value to null if there are
	// event vars.
	stack := []*expr.Node{&parsed.Root}
	for len(stack) > 0 {
		i := stack[0]
		stack = stack[1:]
		stack = append(stack, i.Ands...)
		stack = append(stack, i.Ors...)

		if i.Predicate == nil {
			continue
		}

		// event.data.foo == async.data.foo, but with a `null` (nil) event.
		if i.Predicate.LiteralIdent != nil && strings.HasPrefix(i.Predicate.Ident, "event.") && i.Predicate.Literal == nil {
			i.Predicate.Ident = *i.Predicate.LiteralIdent
			i.Predicate.LiteralIdent = nil
			i.Predicate.Literal = nil
			continue
		}

		// event.data.foo == "bar", but with a `null` (nil) event.
		if i.Predicate.LiteralIdent == nil && strings.HasPrefix(i.Predicate.Ident, "event.") {
			i.Predicate.Ident = "null"
			continue
		}
	}

	return parsed.Root.String(), nil
}

func residual(ctx context.Context, ast *cel.Ast, env *cel.Env, vars map[string]any) (*cel.Ast, error) {
	prog, act, err := program(ctx, ast, env, NewData(vars), false, nil)
	if err != nil {
		return nil, err
	}
	_, details, err := prog.Eval(act)
	if err != nil {
		return nil, err
	}
	return env.ResidualAst(ast, details)
}
