package expressions

import (
	"context"
	"strings"

	"github.com/google/cel-go/cel"
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
func Interpolate(ctx context.Context, expr string, data map[string]any) (string, error) {
	if !strings.Contains(expr, "event.") {
		// Don't interpolate anything in the event, as it isn't being referenced.
		return expr, nil
	}

	// This is a one-off used to interpolate expressions, and so we do not need to cache
	// this.
	//
	// NOTE: If this used the lifting parser things would break massively!  We would
	// lift the same expression twice, resulting in two different values referencing
	// "vars.a".
	eval, err := cachedCompile(ctx, expr)
	if err != nil {
		return expr, err
	}

	ast, err := residual(ctx, eval.ast, eval.env, data)
	if err != nil {
		return expr, err
	}

	return ast.Source().Content(), nil
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
