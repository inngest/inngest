package expressions

import (
	"context"
	"fmt"
	"time"

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
	*timeRefs
}

// EarliestTimeReference returns the earliest time referenced in the
// expression that is >= now().
func (c celProgram) EarliestTimeReference() *time.Time {
	if c.timeRefs == nil {
		return nil
	}
	return c.timeRefs.Next()
}

// program takes a compiled AST, a cel.Env, and the data that's used in the expression and
// returns a program and partial that's used as input to evaluate the expression.
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
	// However, when we're ACTAULLY matching events, we want to evaluate any unknows as if
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

	// We want to perform an exhaustive search and track the state of the search
	// to see if dates are compared, then return the minimum date compared.
	tr, td := timeDecorator(act)

	opts := []cel.ProgramOption{
		cel.EvalOptions(cel.OptExhaustiveEval, cel.OptTrackState, cel.OptPartialEval), // Exhaustive, always, right now.
		cel.CustomDecorator(td),
	}

	if evalUnknowns {
		opts = append(opts, cel.CustomDecorator(unknownDecorator(act)))
	}

	// Create the program, refusing to short circuit if a match is found.
	//
	// This will add all functions from functions.StandardOverloads as we
	// created the environment with our custom library.
	prog, err := env.Program(ast, opts...)

	return &celProgram{Program: prog, timeRefs: tr}, act, err
}

func eval(program *celProgram, activation interpreter.PartialActivation) (interface{}, *time.Time, error) {
	result, _, err := program.Eval(activation)
	if result == nil {
		return false, nil, ErrNoResult
	}
	if types.IsUnknown(result) {
		// When evaluating to a strict result this should never happen.  We inject a decorator
		// to handle unknowns as values similar to null, and should always get a value.
		return false, nil, nil
	}
	if types.IsError(result) {
		return false, nil, errors.Wrapf(ErrInvalidResult, "invalid type comparison: %s", err.Error())
	}
	if err != nil {
		// This shouldn't be handled, as we should get an Error type in result above.
		return false, nil, fmt.Errorf("error evaluating expression: %w", err)
	}
	return result.Value(), program.EarliestTimeReference(), nil
}
