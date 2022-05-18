package expressions

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// env creates a new environment in which we define a single user attribute as a map
// of string to dynamic types, plus functions to augment date handling
func env() (*cel.Env, error) {
	decls := []*exprpb.Decl{
		// declare top-level containers - user and event are possible.
		decls.NewVar("response", decls.NewMapType(decls.String, decls.Dyn)), // used in edges
		decls.NewVar("async", decls.NewMapType(decls.String, decls.Dyn)),    // used in async edges for the new async data.
		decls.NewVar("user", decls.NewMapType(decls.String, decls.Dyn)),     // User information
		decls.NewVar("event", decls.NewMapType(decls.String, decls.Dyn)),    // Event information
		decls.NewVar("action", decls.NewMapType(decls.String, decls.Dyn)),   // Action results
		decls.NewVar("actions", decls.NewMapType(decls.String, decls.Dyn)),  // A common typo for action;  allow both.
	}

	// create a new environment in which we define a single user attribute as a map
	// of string to dynamic types.
	env, err := cel.NewCustomEnv(
		cel.Lib(customLibrary{}),
		cel.Declarations(decls...),
		// XXX: We can add a ref.TypeAdapter here which can convert
		// actions.JSON etc. into ref.Value types
	)

	if err != nil {
		return nil, fmt.Errorf("error initializing expression env: %w", err)
	}

	return env, nil
}
