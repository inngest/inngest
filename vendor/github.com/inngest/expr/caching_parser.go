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

// NewCachingParser returns a CELParser which lifts quoted literals out of the expression
// as variables and uses caching to cache expression parsing, resulting in improved
// performance when parsing expressions.
func NewCachingParser(env *cel.Env, cache *ccache.Cache) CELParser {
	return &cachingParser{
		cache: cache,
		env:   env,
	}
}

type cachingParser struct {
	// cache is a global cache of precompiled expressions.
	cache *ccache.Cache

	env *cel.Env

	hits   int64
	misses int64
}

func (c *cachingParser) Parse(expr string) (*cel.Ast, *cel.Issues, LiftedArgs) {
	if c.cache == nil {
		c.cache = ccache.New(ccache.Configure())
	}

	expr, vars := liftLiterals(expr)

	if cached := c.cache.Get(expr); cached != nil {
		cached.Extend(CacheTime)
		p := cached.Value().(ParsedCelExpr)
		atomic.AddInt64(&c.hits, 1)
		return p.AST, p.Issues, vars
	}

	ast, issues := c.env.Parse(expr)

	c.cache.Set(expr, ParsedCelExpr{
		Expr:   expr,
		AST:    ast,
		Issues: issues,
	}, CacheTime)

	atomic.AddInt64(&c.misses, 1)
	return ast, issues, vars
}

func (c *cachingParser) Hits() int64 {
	return atomic.LoadInt64(&c.hits)
}

func (c *cachingParser) Misses() int64 {
	return atomic.LoadInt64(&c.misses)
}

type ParsedCelExpr struct {
	Expr   string
	AST    *cel.Ast
	Issues *cel.Issues
}
