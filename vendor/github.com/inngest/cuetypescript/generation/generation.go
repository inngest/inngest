package generation

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
)

var (
	ctxIndentLevel = "indent"
	ctxPath        = "path"

	nonAlphaRegexp = regexp.MustCompile("[^\\w]|_")
)

// MarshalString marshals a Cue string into a Typescript type string,
// returning an error.
func MarshalString(cuestr string) (string, error) {
	r := &cue.Runtime{}
	inst, err := r.Compile(".", cuestr)
	if err != nil {
		return "", fmt.Errorf("error generating inst: %w", err)
	}

	return MarshalCueValue(inst.Value())
}

// MarshalCueValue returns a typescript type given a cue value.
func MarshalCueValue(v cue.Value) (string, error) {
	if err := v.Validate(); err != nil {
		return "", err
	}

	// Assume that this is a top-level object containing all definitions,
	// and iterate over each definition
	it, err := v.Fields(cue.Definitions(true), cue.Concrete(false))
	if err != nil {
		return "", err
	}

	exprs := []*Expr{}

	for it.Next() {
		if len(exprs) > 0 {
			// Add two newlines between each field.
			exprs = append(exprs, []*Expr{
				{Data: Lit{Value: "\n"}},
				{Data: Lit{Value: "\n"}},
			}...)
		}

		result, err := generateExprs(context.Background(), it.Label(), it.Value())
		if err != nil {
			return "", err
		}
		exprs = append(exprs, result...)
	}

	// Add a final newline to terminate the file.
	exprs = append(exprs, []*Expr{{Data: Lit{Value: "\n"}}}...)

	str, err := FormatAST(exprs...)
	return str, err
}

// generateExprs creates a typescript expression for a top-level identifier.  This
// differs to the 'generateAST' function as it wraps the created AST within an Expr,
// representing a complete expression terminating with a semicolon.
//
// This is called when walking root-level fields.
func generateExprs(ctx context.Context, label string, v cue.Value) ([]*Expr, error) {
	label, err := formatLabel(label)
	if err != nil {
		return nil, err
	}

	exprs, ast, err := generateAST(ctx, label, v)
	if err != nil {
		return nil, err
	}

	for _, a := range ast {
		// Wrap the AST in an expression, indicating that the AST is a
		// fully defined typescript expression.
		if _, ok := a.(Local); ok {
			exprs = append(exprs, &Expr{Data: a})
			continue
		}

		if enum, ok := a.(Enum); ok {
			// Enums define their own top-level Local AST as they create
			// more than one export.
			enumExprs, err := enum.ExprAST()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, enumExprs...)
			continue
		}

		kind := LocalType
		if binding, ok := a.(Binding); ok && binding.Kind == BindingType {
			// If we're making a struct type, use an Interface declaration
			// instead of the default `const Event = type {` declaration.
			kind = LocalInterface
		}

		exprs = append(exprs, &Expr{
			Data: Local{
				Kind:     kind,
				Name:     label,
				IsExport: true,
				Value:    a,
			},
		})
	}

	return exprs, nil
}

// generateAST creates typescript AST for the given cue values
//
// If the value contains a field of enums, this may generate top-level expressions
// to add to the generated typescript file.
func generateAST(ctx context.Context, label string, v cue.Value) ([]*Expr, []AstKind, error) {
	// We have the cue's value, although this may represent many things.
	// Notably, v.IncompleteKind() returns cue.StringKind even if this field
	// represents a static string, a string type, or an enum of strings.
	//
	// In order to properly generate Typescript AST for the value we need to
	// walk Cue's AST.

	// Embed the current path in context.  This allows us to debug which fields have
	// issues nicely.
	ctx = withPath(ctx, label)

	switch v.IncompleteKind() {
	case cue.StructKind:
		exprs, ast, err := generateStructBinding(ctx, v)
		if err != nil {
			return nil, nil, err
		}
		return exprs, ast, nil
	case cue.ListKind:
		return generateArray(ctx, label, v)
	default:
		syn := v.Syntax(cue.All())
		switch ident := syn.(type) {
		case *ast.BinaryExpr:
			// This could be an enum, a basic lit with a constraint, or a type Ident
			// with a constraint.
			op, _ := v.Expr()
			if op == cue.OrOp {
				// This is an enum.  We're combining > 1 field using
				// the Or operator.
				ast, err := generateEnum(ctx, label, v)
				return nil, ast, err
			}

			if op == cue.AndOp {
				// Although it's possible to combine two structs via the AndOp,
				// those are handled within the IncompleteKind() check above.
				//
				// Because of this we can guarantee that this is a constrained
				// type check.
				ast, err := generateConstrainedIdent(ctx, label, v)
				return nil, ast, err
			}
		case *ast.BasicLit:
			// If this is "null", ensure that we generate a TS
			// ident of null. despite being a value, "null" should
			// be treated as a type.
			if err := v.Null(); err == nil {
				return nil, []AstKind{
					Type{Value: "null"},
				}, nil
			}

			// This is a const.
			scalar, err := generateScalar(ctx, label, v)
			if err != nil {
				return nil, nil, err
			}
			return nil, []AstKind{scalar}, nil
		case *ast.Ident:
			return nil, []AstKind{
				Type{Value: identToTS(ident.Name)},
			}, nil
		case *ast.UnaryExpr:
			// TODO: This is a constraint.  Map this to a function which
			// validates the type.
			return nil, nil, nil
		default:
			return nil, nil, fmt.Errorf("unhandled cue ident for '%s': %T (%s)", path(ctx), ident, ident)
		}
	}
	return nil, nil, fmt.Errorf("unhandled cue type: %v", v.IncompleteKind())
}

func generateConstrainedIdent(ctx context.Context, label string, v cue.Value) ([]AstKind, error) {
	// All types being constrained should share the same heirarchy - eg. uint refers to an
	// int and a number.  In Typescript we don't really care about refined types and can
	// use the first value, as typescript only uses "number" or "string" etc..
	//
	// XXX: Add constraints as comments above the identifier. We could also generate
	// functions which validate constraints with an aexpression
	_, vals := v.Expr()
	_, ast, err := generateAST(ctx, label, vals[0])
	return ast, err
}

// generateLocal returns a scalar identifier, such as a top-level const
// or top-level type.
func generateScalar(ctx context.Context, label string, v cue.Value) (AstKind, error) {
	var i interface{}
	if err := v.Decode(&i); err != nil {
		return nil, err
	}
	return Scalar{Value: i}, nil
}

// generateArray returns an array.  This will always produce a type definition, even if all
// values in the cue list are basic literal values (eg. instead of ["1", "2"] this will generate
// Array<string>).
//
// This may return top-level expressions if the array contains a struct with enums.
func generateArray(ctx context.Context, label string, v cue.Value) ([]*Expr, []AstKind, error) {
	members := []AstKind{}

	// Walk the iterator for all basic values first.
	iter, err := v.List()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid value generating array: %w", err)
	}

	for iter.Next() {
		// This is not called for type definitions;  only for concrete values.
		_, ast, err := generateAST(ctx, iter.Label(), iter.Value())

		if err != nil {
			return nil, nil, err
		}
		members = append(members, ast...)
	}

	// We can't use v.List() to create an iterator as iterators don't return types:
	// they only work with concrete values.
	//
	// Instead, take the Cue AST and walk it to determine the elements in the list,
	// and create TS AST from them.
	listLit, ok := v.Syntax(cue.All()).(*ast.ListLit)
	if !ok {
		return nil, nil, fmt.Errorf("unknown list ast type: %T", v.Syntax(cue.All()))
	}

	if len(listLit.Elts) == 0 {
		return nil, []AstKind{
			Binding{
				Kind: BindingTypedArray,
			},
		}, nil
	}

	elts := listLit.Elts
	if ellipsis, ok := listLit.Elts[0].(*ast.Ellipsis); ok {
		elts = []ast.Expr{ellipsis.Type}
	}

	// We only want the same type mapped once in an array. Typescript uses
	// the basic "number" type whereas cue has "int" and "float";  there's a chance
	// that we end up duplicating TS type names.
	mappedTypes := map[string]struct{}{}

	exprs := []*Expr{}
	for len(elts) > 0 {
		elt := elts[0]
		elts = elts[1:]

		switch a := elt.(type) {
		case *ast.BinaryExpr:
			elts = append(elts, a.X)
			elts = append(elts, a.Y)
			continue
		case *ast.BasicLit:
			// Basic value  This would already have been covered in the
			// iterator case above.
		case *ast.StructLit:
			value, err := astToValue(&cue.Runtime{}, a)
			if err != nil {
				return nil, nil, fmt.Errorf("error converting array struct to value: %w", err)
			}
			e, ast, err := generateStructBinding(ctx, value)
			if err != nil {
				return nil, nil, fmt.Errorf("error generating array struct type: %w", err)
			}
			exprs = append(exprs, e...)
			members = append(members, ast...)
		case *ast.Ident:
			typeName := identToTS(a.Name)
			if _, ok := mappedTypes[typeName]; ok {
				continue
			}
			members = append(members, Type{typeName})
			mappedTypes[typeName] = struct{}{}
		}
	}

	binding := Binding{
		Kind:    BindingTypedArray,
		Members: members,
	}

	return exprs, []AstKind{binding}, nil
}

// generateEnum creates an enum definition which should be epanded to its
// full Expr AST for a given value.
func generateEnum(ctx context.Context, label string, v cue.Value) ([]AstKind, error) {
	label, err := formatLabel(label)
	if err != nil {
		return nil, err
	}

	_, vals := v.Expr()
	members := make([]AstKind, len(vals))

	// Generate AST representing the value of each member in the enum.
	for n, val := range vals {
		_, ast, err := generateAST(ctx, label, val)
		if err != nil {
			return nil, fmt.Errorf("error generating ast for enum val: %w", err)
		}
		if len(ast) > 1 {
			return nil, fmt.Errorf("invalid ast generated for enum val: %v", ast)
		}
		members[n] = ast[0]
	}

	return []AstKind{Enum{
		Name:    label,
		Members: members,
	}}, nil
}

// generateStructBinding returns a binding representing a TypeScript object
// or interface.
//
// It does not wrap this within a Local as this function is used within top-level
// and nested structs;  nested structs are the Value of a KeyValue whereas
// top-level identifiers are values of a Local.
func generateStructBinding(ctx context.Context, v cue.Value) ([]*Expr, []AstKind, error) {
	it, err := v.Fields(cue.All())
	if err != nil {
		return nil, nil, err
	}

	expr := []*Expr{}

	members := []AstKind{}
	for it.Next() {
		if it.IsHidden() {
			continue
		}

		// Create the raw AST for each field's value
		newExpr, created, err := generateAST(withIncreasedIndentLevel(ctx), it.Label(), it.Value())
		expr = append(expr, newExpr...)
		if err != nil {
			return nil, nil, err
		}

		if len(created) == 0 {
			continue
		}

		// We may have generated top-level local definitions, which we should pull out
		// to the AST context and not use as a key-value.
		if local, ok := created[0].(Local); ok {
			// Add the fields to the top-level object being created.
			expr = append(expr, &Expr{Data: created[0]})
			// And add a reference to the type as the key value pair.
			created[0] = Type{Value: local.Name}
		}

		// A struct may contain enum definions.  If the enum is only merging
		// types (string | null) it's safe to inline.  If it contains values (
		// complex structs, "foo" | "bar") we want to drag this out.
		if enum, ok := created[0].(Enum); ok {

			if enum.IsScalarType() {
				// This type is safe to embed.
			} else {
				enumAst, err := enum.ExprAST()
				if err != nil {
					return nil, nil, err
				}
				expr = append(expr, enumAst...)
				// Add two newlines between each enum and struct visually.
				expr = append(expr, []*Expr{
					{Data: Lit{Value: "\n"}},
					{Data: Lit{Value: "\n"}},
				}...)
				// Use the enum name as the key's value.
				created[0] = Type{Value: enum.Name}
			}
		}

		// Wrap the AST value within a KeyValue.
		wrapped := make([]AstKind, len(created))
		for n, item := range created {
			wrapped[n] = KeyValue{
				Key:      it.Label(),
				Value:    item,
				Optional: it.IsOptional(),
			}
		}

		members = append(members, wrapped...)
	}

	// Add the struct to the generated AST fields.
	ast := []AstKind{Binding{
		Kind:        BindingType,
		Members:     members,
		IndentLevel: indentLevel(ctx),
	}}

	return expr, ast, nil
}

// identToTS returns Typescript type names from a given cue type name.
func identToTS(name string) string {
	switch name {
	case "<nil>":
		return "null"
	case "bool":
		return "boolean"
	case "float", "int", "number":
		return "number"
	case "_":
		return "unknown"
	case "[...]":
		return "Array<unknown>"
	case "{...}":
		return "{ [key: string]: unknown }"
	default:
		return name
	}
}

// indentLevel returns the current indent level from the context.  This is
// a quick and dirty way of formatting nested structs.
func indentLevel(ctx context.Context) int {
	indent, _ := ctx.Value(ctxIndentLevel).(int)
	return indent
}

// withIncreasedIndentLevel increases the indent level in the given context,
// returning a new context with the updated indent level.
func withIncreasedIndentLevel(ctx context.Context) context.Context {
	level := indentLevel(ctx) + 1
	return context.WithValue(ctx, ctxIndentLevel, level)
}

func astToValue(r *cue.Runtime, ast ast.Node) (cue.Value, error) {
	// XXX: We really need a better way to create a cue.Value from
	// an AST struct.
	byt, _ := format.Node(
		ast,
		format.TabIndent(false),
		format.UseSpaces(2),
	)
	inst, err := r.Compile(".", byt)
	if err != nil {
		return cue.Value{}, err
	}
	return inst.Value(), nil
}

func path(ctx context.Context) string {
	path, _ := ctx.Value(ctxPath).(string)
	return path
}

func withPath(ctx context.Context, path string) context.Context {
	existing, _ := ctx.Value(ctxPath).(string)
	if existing == "" {
		return context.WithValue(ctx, ctxPath, path)
	}
	return context.WithValue(ctx, ctxPath, existing+"."+path)
}

func formatLabel(label string) (string, error) {
	label = titleCaseName(label)

	if len(label) == 0 {
		return "", fmt.Errorf("unable to generate typescript for unnamed type")
	}

	// If the first letter is lowercase, convert this to title.
	if unicode.IsLower(rune(label[0])) {
		label = strings.Title(strings.ToLower(label))
	}

	return label, nil
}

func titleCaseName(name string) string {
	name = nonAlphaRegexp.ReplaceAllString(name, " ")
	name = strings.Title(name)
	return strings.ReplaceAll(name, " ", "")
}
