package expressions

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	pbexpr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

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

// ParseAttributes returns the attributes used within the expression.  This is necessary as we need to
// inspect all variables used in an expression in order to "fill in" blanks for unknowns.
func parseUsedAttributes(ctx context.Context, ast *cel.Ast) (*UsedAttributes, error) {
	attrs := &UsedAttributes{
		Root:   []string{},
		Fields: map[string][][]string{},
	}

	// Walk through the AST, looking for all instances of "select_expr" expression
	// kinds.  These elements are specifically selecting fields from parents, which
	// is exactly what we need to figure out the variables used within an expression.
	stack := []*pbexpr.Expr{ast.Expr()}
	for len(stack) > 0 {
		ast := stack[0]
		stack = stack[1:]

		// Depending on the item, add the following
		switch ast.ExprKind.(type) {
		case *pbexpr.Expr_ComprehensionExpr:
			// eg. "event.data.tags.exists(x, x == 'Open'), so put what we're iterating over
			// onto the stack to parse, ignoring this function call but adding the data.
			c := ast.GetComprehensionExpr()
			stack = append(stack, c.IterRange)
		case *pbexpr.Expr_CallExpr:
			// Everything is a function call:
			// - > evaluates to _>_ with two arguments, etc.
			// This means pop all args onto the stack so that we can find
			// all select expressions.
			stack = append(stack, ast.GetCallExpr().GetArgs()...)

		case *pbexpr.Expr_IdentExpr:
			name := ast.GetIdentExpr().Name
			attrs.add(name, nil)

		case *pbexpr.Expr_SelectExpr:
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
					return nil, fmt.Errorf("unknown number of callers for bracket notation: %d", len(args))
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

	return attrs, nil
}
