package expressions

import (
	"context"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/interpreter"
	"github.com/pkg/errors"
)

// celProgram wraps a cel.Program used to evaluate expressions with a time decorator,
// allowing us to inspect the times referenced in the expression (and easily grab
// the earliest time >= now().
type celProgram struct {
	cel.Program
}

// buildProgram creates a reusable cel.Program from the AST.  It is data-independent:
// when evalUnknowns is true the unknownDecorator is attached (as a stateless wrapper),
// making the program safe to cache for the lifetime of the expression.
//
// evalUnknowns controls whether unknowns are treated as nulls (true, event matching)
// or left in the eval state (false, residual/interpolation via ResidualAst).
func buildProgram(ast *cel.Ast, env *cel.Env, evalUnknowns bool) (*celProgram, error) {
	var opts []cel.ProgramOption
	if evalUnknowns {
		// Event-matching path: OptTrackState/OptExhaustiveEval only needed for ResidualAst.
		opts = []cel.ProgramOption{
			cel.EvalOptions(cel.OptPartialEval),
			cel.CustomDecorator(unknownDecorator()),
		}
	} else {
		// Residual/interpolation path: OptTrackState for ResidualAst, OptExhaustiveEval so
		// unknowns in both branches of &&/|| are recorded.
		opts = []cel.ProgramOption{
			cel.EvalOptions(cel.OptExhaustiveEval, cel.OptTrackState, cel.OptPartialEval),
		}
	}
	prog, err := env.Program(ast, opts...)
	if err != nil {
		return nil, err
	}
	return &celProgram{Program: prog}, nil
}

// program builds a one-shot program+activation pair.  Used by the residual/interpolation
// path (evalUnknowns=false) where the program is not cached and unknowns must survive into
// the eval state so that ResidualAst can reduce the expression.
func program(
	ctx context.Context,
	ast *cel.Ast,
	env *cel.Env,
	data *Data,
	// evalUnknowns is used to evaluate unknowns, or leave them in the expression.  When
	// computing interpolated expressions to fill in `event.data.${vars}`, we're not actually
	// trying to evaluate unknowns: we're trying to leave them in so that we can compute
	// the interpolated expression.
	//
	// However, when we're ACTUALLY matching events, we want to evaluate any unknowns as if
	// the value was `null`.  Setting this to true forces the evaluation of unknown vars in
	// the program.
	evalUnknowns bool,
	attrs *UsedAttributes, // may be nil, will be computed if nil
) (*celProgram, interpreter.PartialActivation, error) {
	var err error

	if attrs == nil {
		if attrs, err = parseUsedAttributes(ctx, ast); err != nil {
			return nil, nil, err
		}
	}

	act, err := data.Partial(ctx, *attrs)
	if err != nil {
		return nil, nil, err
	}

	prog, err := buildProgram(ast, env, evalUnknowns)
	if err != nil {
		return nil, nil, err
	}
	return prog, act, nil
}

func eval(program *celProgram, activation interpreter.PartialActivation) (interface{}, error) {
	result, _, err := program.Eval(activation)
	if result == nil {
		return false, ErrNoResult
	}
	if types.IsUnknown(result) {
		// When evaluating to a strict result this should never happen.  We inject a decorator
		// to handle unknowns as values similar to null, and should always get a value.
		return false, nil
	}
	if types.IsError(result) {
		return false, errors.Wrapf(ErrInvalidResult, "invalid type comparison: %s", err.Error())
	}
	if err != nil {
		// This shouldn't be handled, as we should get an Error type in result above.
		return false, fmt.Errorf("error evaluating expression: %w", err)
	}
	return result.Value(), nil
}
