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
	"fmt"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/inngest/expr"
	"github.com/karlseguin/ccache/v2"
	"github.com/pkg/errors"
)

const (
	pkgName = "expressions.inngest"
)

var (
	CacheExtendTime = time.Minute * 30
	CacheTTL        = time.Minute * 30
	// cache is a global cache of precompiled expressions.
	cache *ccache.Cache

	// On average, 20 compiled expressions fit into 1mb of ram.
	CacheMaxSize int64 = 50_000

	exprCompiler expr.CELCompiler
	treeParser   expr.TreeParser
)

func init() {
	cache = ccache.New(ccache.Configure().MaxSize(CacheMaxSize))
	if e, err := env(); err == nil {
		exprCompiler = expr.NewCachingCompiler(e, cache)
		treeParser = expr.NewTreeParser(exprCompiler)
	}
}

func CompilerSingleton() expr.CELCompiler {
	return exprCompiler
}

func ParserSingleton() expr.TreeParser {
	return treeParser
}

var (
	ErrNoResult      = errors.New("expression did not return true or false")
	ErrInvalidResult = errors.New("expression errored")
)

// Evaluator represents a cacheable, goroutine safe manager for evaluating a single
// precompiled expression with arbitrary data.
type Evaluator interface {
	// Evaluate tests the incoming Data against the expression that is
	// stored within the BooleanEvaluator implementation.
	//
	// Attributes that are present within the expression but missing from the
	// data should be treated as null values;  the expression must not error.
	Evaluate(ctx context.Context, data *Data) (interface{}, *time.Time, error)

	// UsedAttributes returns the attributes that are referenced within the
	// expression.
	UsedAttributes(ctx context.Context) *UsedAttributes

	// FilteredAttributes filters the given data to contain only attributes
	// referenced from the expression.
	FilteredAttributes(ctx context.Context, data *Data) *Data
}

// BooleanEvaluator representsn Evaluator which evaluates an expression returning
// booleans only.
type BooleanEvaluator interface {
	// Evaluate tests the incoming Data against the expression that is
	// stored within the BooleanEvaluator implementation.
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

func exprEvaluator(ctx context.Context, e expr.Evaluable, input map[string]any) (bool, error) {
	eval, err := NewBooleanEvaluator(ctx, e.GetExpression())
	if err != nil {
		return false, err
	}
	data := NewData(input)
	ok, _, err := eval.Evaluate(ctx, data)
	return ok, err
}

// Evaluate is a helper function to create a new, cached expression evaluator to evaluate
// the given data immediately.
func Evaluate(ctx context.Context, expression string, input map[string]interface{}) (interface{}, *time.Time, error) {
	eval, err := NewExpressionEvaluator(ctx, expression)
	if err != nil {
		return false, nil, err
	}
	data := NewData(input)
	return eval.Evaluate(ctx, data)
}

func EvaluateBoolean(ctx context.Context, expression string, input map[string]interface{}) (bool, *time.Time, error) {
	eval, err := NewBooleanEvaluator(ctx, expression)
	if err != nil {
		return false, nil, err
	}
	data := NewData(input)
	return eval.Evaluate(ctx, data)
}

// NewExpressionEvaluator returns a new BooleanEvaluator instance for a given expression. The
// instance can be used across many goroutines to evaluate the expression against any
// data. The Evaluable instance is loaded from the cache, or is cached if not found.
//
// NOTE: This does NOT validate that the expression uses known variables.  Validation is SLOW,
// NOT THREAD SAFE:  It is expected that you call Validate() separaetly.  We do NOT bundle it
// here because evaluating expressions needs to be fast in the hot path.
func NewExpressionEvaluator(ctx context.Context, expression string) (Evaluator, error) {
	// Use the lifting expression parser in order to compile our env,
	// if it's not nil.
	if exprCompiler != nil {
		ast, issues, vars := exprCompiler.Parse(expression)
		if issues != nil {
			return nil, NewCompileError(issues.Err())
		}
		e, err := env()
		if err != nil {
			return nil, err
		}
		eval := &expressionEvaluator{
			ast:        ast,
			env:        e,
			expression: expression,
			liftedVars: vars.Map(),
		}
		if err := eval.parseAttributes(ctx); err != nil {
			return nil, err
		}
		return eval, nil
	}

	// Use default parsing, if the exprCompiler isn't specified.
	return cachedCompile(ctx, expression)
}

func cachedCompile(ctx context.Context, expression string) (*expressionEvaluator, error) {
	// NOTE: We use an "eval:" prefix to avoid any conflicts with the `exprCompiler` singleton which
	// may use the expression as a key.
	if eval := cache.Get("eval:" + expression); eval != nil {
		eval.Extend(CacheExtendTime)
		return eval.Value().(*expressionEvaluator), nil
	}

	e, err := env()
	if err != nil {
		return nil, err
	}
	ast, issues := e.Parse(expression)
	if issues != nil {
		return nil, NewCompileError(issues.Err())
	}
	eval := &expressionEvaluator{
		ast:        ast,
		env:        e,
		expression: expression,
	}

	if err := eval.parseAttributes(ctx); err != nil {
		return nil, err
	}

	cache.Set("eval:"+expression, eval, CacheTTL)
	return eval, nil
}

func NewBooleanEvaluator(ctx context.Context, expression string) (BooleanEvaluator, error) {
	e, err := NewExpressionEvaluator(ctx, expression)
	return booleanEvaluator{Evaluator: e}, err
}

type booleanEvaluator struct {
	Evaluator
}

func (b booleanEvaluator) Evaluate(ctx context.Context, data *Data) (bool, *time.Time, error) {
	val, time, err := b.Evaluator.Evaluate(ctx, data)
	if err != nil {
		return false, time, err
	}
	result, ok := val.(bool)
	if !ok {
		return false, nil, errors.Wrapf(ErrInvalidResult, "returned type %T (%s)", val, val)
	}
	return result, time, err
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

	// liftedVars are vars lifted from the expression, if parsed with a lifting
	// parser.
	liftedVars map[string]any

	// attrs is allows us to determine which attributes are used within an expression.
	// This is needed to create partial activations, and is also used to optimistically
	// load only necessary attributes.
	attrs *UsedAttributes
}

// Validate calls parse and check on an ASTs using NON CACHING parsing.  This MUST be non-caching
// as calling Check on an AST is not thread safe.
func Validate(ctx context.Context, expression string) error {
	// Compile the expression as new.
	env, err := env()
	if err != nil {
		return err
	}
	if _, issues := env.Compile(expression); issues != nil {
		return fmt.Errorf("error validating expression: %w", NewCompileError(issues.Err()))
	}
	return nil
}

// Evaluate compiles an expression string against a set of variables, returning whether the
// expression evaluates to true, the next earliest time to re-test the evaluation (if dates are
// compared), and any errors.
func (e *expressionEvaluator) Evaluate(ctx context.Context, data *Data) (interface{}, *time.Time, error) {
	if data == nil {
		return false, nil, nil
	}

	if len(e.liftedVars) > 0 {
		// Clone the data to remove concurrent writes.
		data = data.Clone()
		data.Add(map[string]any{
			"vars": e.liftedVars,
		})
	}

	program, act, err := program(ctx, e.ast, e.env, data, true, e.attrs)

	if err != nil {
		return nil, nil, err
	}

	return eval(program, act)
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

	attrs, err := parseUsedAttributes(ctx, e.ast)

	if err != nil {
		return err
	}
	e.attrs = attrs
	return nil
}

type CompileError struct {
	Err error
}

func NewCompileError(err error) *CompileError {
	return &CompileError{Err: err}
}

func (c *CompileError) Error() string {
	return fmt.Sprintf("error compiling expression: %s", c.Err)
}

func (c *CompileError) Unwrap() error {
	return c.Err
}

func (c *CompileError) Message() string {
	return c.Err.Error()
}

func (c *CompileError) Is(tgt error) bool {
	_, ok := tgt.(*CompileError)
	return ok
}
