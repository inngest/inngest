// Package expressions provides the ability to inspect and evaluate arbitrary
// user-defined expressions.  We use the Cel-Go package as a runtime to implement
// computationally bounded, non-turing complete expressions with a familiar c-like
// syntax.
//
// Unlike cel-go's defaults, this package handles unknowns similarly to null values,
// and allows arbitrary attributes within expressions without errors.  We also
// provide basic type coercion allowing eg. int <> float comparisons, which errors
// within cel by default.
//
// Expressions can be inspected to determine the variables that they reference,
// partially evaluated with missing data, and report timestamps used within the
// expression for future reference (eg. recomputing state at that time).
package expressions

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/karlseguin/ccache/v2"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var (
	CacheExtendTime = time.Minute * 10
	CacheTTL        = time.Minute * 30

	ErrNoResult      = errors.New("expression did not return true or false")
	ErrInvalidResult = errors.New("expression errored")
)

var (
	// cache is a global cache of precompiled expressions.
	cache *ccache.Cache
)

func init() {
	cache = ccache.New(ccache.Configure().MaxSize(10_000))
}

// Evaluable represents a cacheable, goroutine safe manager for evaluating a single
// precompiled expression with arbitrary data.
type Evaluable interface {
	// Evaluate tests the incoming Data against the expression that is
	// stored within the Evaluable implementation.
	//
	// Attributes that are present within the expression but missing from the
	// data should be treated as null values;  the expression must not error.
	Evaluate(ctx context.Context, data *Data) (bool, *time.Time, error)

	// UsedAttributes returns the attributes that are referenced within the
	// expression.
	UsedAttributes(ctx context.Context) *UsedAttributes

	// FilteredAttributes filters the given data to contain only attributes
	// referenced from the expression.
	FilteredAttributes(ctx context.Context, data *Data) *Data
}

// Evaluate is a helper function to create a new, cached expression evaluator to evaluate
// the given data immediately.
func Evaluate(ctx context.Context, expression string, input map[string]interface{}) (bool, *time.Time, error) {
	eval, err := NewExpressionEvaluator(ctx, expression)
	if err != nil {
		return false, nil, err
	}
	data := NewData(input)
	return eval.Evaluate(ctx, data)
}

// NewExpressionEvaluator returns a new Evaluable instance for a given expression. The
// instance can be used across many goroutines to evaluate the expression against any
// data. The Evaluable instance is loaded from the cache, or is cached if not found.
func NewExpressionEvaluator(ctx context.Context, expression string) (Evaluable, error) {
	sha := sum(expression)

	if eval := cache.Get(sha); eval != nil {
		eval.Extend(CacheExtendTime)
		return eval.Value().(*expressionEvaluator), nil
	}

	ctx, span := otel.Tracer("expressions").Start(ctx, "NewExpressionEvaluator")
	defer span.End()

	span.AddEvent("creating env")
	e, err := env()
	span.AddEvent("created env")
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	span.AddEvent("compiling")
	ast, issues := e.Compile(expression)
	span.AddEvent("compiled")
	if issues != nil {
		return nil, fmt.Errorf("error compiling expression: %w", issues.Err())
	}

	eval := &expressionEvaluator{
		ast:        ast,
		env:        e,
		expression: expression,
	}

	if err := eval.parseAttributes(ctx); err != nil {
		return nil, err
	}

	cache.Set(sha, eval, CacheTTL)
	return eval, nil
}

type expressionEvaluator struct {
	// TODO: Refactor unknownEval to remove the need tor attr.Eval(activation),
	// and make dateRefs thread safe.  We can then place a cel.Program on the
	// evaluator as it's thread safe.  Without these changes, Programs are bound
	// to specific expression & data combinations.
	ast *cel.Ast
	env *cel.Env

	// expression is the raw expression
	expression string

	// attrs is allows us to determine which attributes are used within an expression.
	// This is needed to create partial activations, and is also used to optimistically
	// load only necessary attributes.
	attrs *UsedAttributes
}

// Evaluate compiles an expression string against a set of variables, returning whether the
// expression evaluates to true, the next earliest time to re-test the evaluation (if dates are
// compared), and any errors.
func (e *expressionEvaluator) Evaluate(ctx context.Context, data *Data) (bool, *time.Time, error) {
	if data == nil {
		return false, nil, nil
	}

	act, err := data.Partial(ctx, *e.attrs)
	if err != nil {
		return false, nil, err
	}

	// We want to perform an exhaustive search and track the state of the search
	// to see if dates are compared, then return the minimum date compared.
	tr, td := timeDecorator(act)

	// Create the program, refusing to short circuit if a match is found.
	//
	// This will add all functions from functions.StandardOverloads as we
	// created the environment with our custom library.
	program, err := e.env.Program(
		e.ast,
		cel.EvalOptions(cel.OptExhaustiveEval, cel.OptTrackState, cel.OptPartialEval), // Exhaustive, always, right now.
		cel.CustomDecorator(unknownDecorator(act)),
		cel.CustomDecorator(td),
	)
	if err != nil {
		return false, nil, err
	}

	result, _, err := program.Eval(act)
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
		return false, nil, fmt.Errorf("error evaluating expression '%s': %w", e.expression, err)
	}

	b, ok := result.Value().(bool)
	if !ok {
		return false, nil, errors.Wrapf(ErrInvalidResult, "returned type %T (%s)", result, result)
	}

	// Find earliest date that we need to test against.
	earliest := tr.Next()
	return b, earliest, nil
}

// UsedAttributes returns the attributes used within the expression.
func (e *expressionEvaluator) UsedAttributes(ctx context.Context) *UsedAttributes {
	return e.attrs
}

// FilteredAttributes returns a new Data pointer with only the attributes
// used within the expression.
func (e *expressionEvaluator) FilteredAttributes(ctx context.Context, d *Data) *Data {
	if d == nil {
		return nil
	}

	filtered := map[string]interface{}{}

	current := filtered
	stack := e.attrs.FullPaths()
	for len(stack) > 0 {
		path := stack[0]
		stack = stack[1:]

		val, ok := d.Get(ctx, path)
		if !ok {
			continue
		}

		for n, part := range path {
			if n == len(path)-1 {
				// This is the value.
				current[part] = val
				continue
			}

			if _, ok := current[part]; !ok {
				current[part] = map[string]interface{}{}
			}
			current = current[part].(map[string]interface{})
		}

		current = filtered
	}

	// It is safe to set data directly here, as we've manually
	// created a map containing raw values from a previously set
	// Data field.  This prevents us from needlesly mapifying
	// data from a constructor.
	return &Data{data: filtered}
}

// ParseAttributes returns the attributes used within the expression.
func (e *expressionEvaluator) parseAttributes(ctx context.Context) error {
	if e.attrs != nil {
		return nil
	}

	attrs := &UsedAttributes{
		Root:   []string{},
		Fields: map[string][][]string{},
	}

	// Walk through the AST, looking for all instances of "select_expr" expression
	// kinds.  These elements are specifically selecting fields from parents, which
	// is exactly what we need to figure out the variables used within an expression.
	stack := []*expr.Expr{e.ast.Expr()}
	for len(stack) > 0 {
		ast := stack[0]
		stack = stack[1:]

		// Depending on the item, add the following
		switch ast.ExprKind.(type) {
		case *expr.Expr_ComprehensionExpr:
			// eg. "event.data.tags.exists(x, x == 'Open'), so put what we're iterating over
			// onto the stack to parse, ignoring this function call but adding the data.
			c := ast.GetComprehensionExpr()
			stack = append(stack, c.IterRange)
		case *expr.Expr_CallExpr:
			// Everything is a function call:
			// - > evaluates to _>_ with two arguments, etc.
			// This means pop all args onto the stack so that we can find
			// all select expressions.
			stack = append(stack, ast.GetCallExpr().GetArgs()...)

		case *expr.Expr_IdentExpr:
			name := ast.GetIdentExpr().Name
			attrs.add(name, nil)

		case *expr.Expr_SelectExpr:
			// Note that the select expression unravels from the deepest key first:
			// given "event.data.foo.bar", the current ast node will be for "foo"
			// and the field name will be for "bar".
			//
			// Iterate through all object selects until there are no more, adding
			// to the path.

			path := []string{}
			for ast.GetSelectExpr() != nil {
				path = append([]string{ast.GetSelectExpr().Field}, path...)
				ast = ast.GetSelectExpr().Operand
			}

			ident := ast.GetIdentExpr()
			caller := ast.GetCallExpr()

			if ident == nil && caller != nil && caller.Function == "_[_]" {
				// This might be square notation: "actions[1]".  This should
				// have two args:  the object (eg. actions), which is an
				// IdentExpr, and a ConstExpr containing the number.
				args := caller.GetArgs()
				if len(args) != 2 {
					return fmt.Errorf("unknown number of callers for bracket notation: %d", len(args))
				}

				// Functions have been rewritten to move "actions.1" into a string:
				// actions["1"]
				id := args[1].GetConstExpr().GetStringValue()
				path = append([]string{args[0].GetIdentExpr().GetName(), id}, path...)
			}

			if ident != nil {
				path = append([]string{ident.Name}, path...)
			}

			root := path[0]
			fields := path[1:]

			attrs.add(root, fields)
		}
	}

	e.attrs = attrs
	return nil
}

// UsedAttributes represents the evaluated expression's root and top-level fields used.
type UsedAttributes struct {
	// Root represents root-level variables used within the expression
	Root []string

	// Fields represent fields within each root-level variable accessed.
	//
	// For example, given an attribute of "event.data.index", this map holds
	// a key of "event" with a slice of [][]string{{"data", "index"}}
	Fields map[string][][]string

	// exists
	exists map[string]struct{}
}

// FullPaths returns a slice of path slices with the roots appended.
func (u UsedAttributes) FullPaths() [][]string {
	paths := [][]string{}
	for root, items := range u.Fields {
		for _, path := range items {
			path = append([]string{root}, path...)
			paths = append(paths, path)
		}
	}
	return paths
}

func (u *UsedAttributes) add(root string, path []string) {
	if u.exists == nil {
		u.exists = map[string]struct{}{}
	}

	if _, ok := u.Fields[root]; !ok {
		u.Root = append(u.Root, root)
		u.Fields[root] = [][]string{}
	}

	// Add this once.
	key := fmt.Sprintf("%s.%s", root, strings.Join(path, "."))
	if _, ok := u.exists[key]; !ok && len(path) > 0 {
		u.Fields[root] = append(u.Fields[root], path)
		// store this key so it's not duplicated.
		u.exists[key] = struct{}{}
	}
}

// sum returns a checksum of the given expression, used as the cache key.
func sum(expression string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(expression)))
}
