package exprenv

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/inngest/expr"
	"github.com/karlseguin/ccache/v2"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
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

	defaultKeys = []string{
		"event",
		"async",
		"vars",
	}

	envSingleton *cel.Env
	envError     error
	envCreation  sync.Once
)

func init() {
	cache = ccache.New(ccache.Configure().MaxSize(CacheMaxSize))
	if e, err := Env(); err == nil {
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

// env creates a new environment in which we define a single user attribute as a map
// of string to dynamic types, plus functions to augment date handling
func Env(keys ...string) (*cel.Env, error) {
	envCreation.Do(func() {
		if len(keys) == 0 {
			keys = defaultKeys
		}

		vars := []*exprpb.Decl{}

		// declare top-level variable names as containers.
		for _, key := range keys {
			vars = append(
				vars,
				decls.NewVar(key, decls.NewMapType(decls.String, decls.Dyn)),
			)
		}

		// create a new environment in which we define a single user attribute as a map
		// of string to dynamic types.
		envSingleton, envError = cel.NewCustomEnv(
			cel.Lib(customLibrary{}),
			cel.Declarations(vars...),
		)
	})
	if envError != nil {
		return nil, fmt.Errorf("error initializing expression env: %w", envError)
	}
	return envSingleton, nil
}
