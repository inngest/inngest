package expressions

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

var (
	defaultKeys = []string{
		"event",
		"steps",
		"response",
		"async",
		"user",
		"actions", // deprecated
		"action",  // deprecated
	}
)

// env creates a new environment in which we define a single user attribute as a map
// of string to dynamic types, plus functions to augment date handling
func env(keys ...string) (*cel.Env, error) {
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
	env, err := cel.NewCustomEnv(
		cel.Lib(customLibrary{}),
		cel.Declarations(vars...),
	)

	if err != nil {
		return nil, fmt.Errorf("error initializing expression env: %w", err)
	}

	return env, nil
}
