package expr

import (
	"sync/atomic"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/karlseguin/ccache/v2"
)

var (
	CacheTime = time.Hour
)

// NewCachingCompiler returns a CELCompiler which lifts quoted literals out of the expression
// as variables and uses caching to cache expression parsing, resulting in improved
// performance when parsing expressions.
func NewCachingCompiler(env *cel.Env, cache *ccache.Cache) CELCompiler {
	return &cachingCompiler{
		cache: cache,
		env:   env,
	}
}

type cachingCompiler struct {
	// cache is a global cache of precompiled expressions.
	cache *ccache.Cache

	env *cel.Env

	hits   int64
	misses int64
}

// Parse calls
func (c *cachingCompiler) Parse(expr string) (*cel.Ast, *cel.Issues, LiftedArgs) {
	if c.cache == nil {
		c.cache = ccache.New(ccache.Configure())
	}

	expr, vars := liftLiterals(expr)

	if cached := c.cache.Get("cc:" + expr); cached != nil {
		cached.Extend(CacheTime)
		p := cached.Value().(parsedCELExpr)
		atomic.AddInt64(&c.hits, 1)
		return p.AST, p.ParseIssues, vars
	}

	ast, issues := c.env.Parse(expr)

	c.cache.Set("cc:"+expr, parsedCELExpr{
		Expr:        expr,
		AST:         ast,
		ParseIssues: issues,
	}, CacheTime)

	atomic.AddInt64(&c.misses, 1)
	return ast, issues, vars
}

func (c *cachingCompiler) Compile(expr string) (*cel.Ast, *cel.Issues, LiftedArgs) {
	ast, issues, args := c.Parse(expr)
	if issues != nil {
		return ast, issues, args
	}
	ast, issues = c.env.Check(ast)
	return ast, issues, args
}

func (c *cachingCompiler) Hits() int64 {
	return atomic.LoadInt64(&c.hits)
}

func (c *cachingCompiler) Misses() int64 {
	return atomic.LoadInt64(&c.misses)
}

type parsedCELExpr struct {
	Expr        string
	AST         *cel.Ast
	ParseIssues *cel.Issues
}
