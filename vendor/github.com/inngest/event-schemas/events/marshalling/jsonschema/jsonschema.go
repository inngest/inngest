package jsonschema

import (
	"encoding/json"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/ast/astutil"
	"cuelang.org/go/cue/format"
	cuejson "cuelang.org/go/encoding/json"
	"cuelang.org/go/encoding/jsonschema"
	"cuelang.org/go/encoding/openapi"
)

var (
	c = &openapi.Config{
		PkgName: "",
		Version: "3.0.0",
	}
)

// UnmarshalString returns a cue type for an event from a JSON schema definition.
func UnmarshalString(schema string) (string, error) {
	// Decode the schema into a cue.Instance.
	r := &cue.Runtime{}
	in, err := cuejson.Decode(r, "", []byte(schema))
	if err != nil {
		return "", err
	}

	expr, err := jsonschema.Extract(in, &jsonschema.Config{})
	if err != nil {
		return "", err
	}
	if expr == nil {
		return "", fmt.Errorf("no definition generated from json schema")
	}
	if err := astutil.Sanitize(expr); err != nil {
		return "", err
	}

	// By default, this returns event data as top-level values, ie. not wrapped
	// in an object.  By compiling the file we can extract the top-level implicit
	// object as a cue.Value and format that node.
	//
	// This gives us an outer object:
	// {
	//    name: "..."
	// }
	instance, err := r.CompileFile(expr)
	if err != nil {
		return "", err
	}

	return formatValue(instance.Value())
}

// MarshalString generates OpenAPI schemas given cue configuration.  Schemas are
// generated for each top-level identifier;  many schemas are generated:
//
//	#Event: {
//		name: string
//	}
//
// Cue types without identifiers will have no schemas generated.
func MarshalString(cuestr string) (Schemas, error) {
	r := &cue.Runtime{}
	inst, err := r.Compile(".", cuestr)
	if err != nil {
		return Schemas{}, fmt.Errorf("error generating inst: %w", err)
	}

	byt, err := openapi.Gen(inst, c)
	if err != nil {
		return Schemas{}, fmt.Errorf("error generating config: %w", err)
	}

	genned := &genned{}
	if err := json.Unmarshal(byt, genned); err != nil {
		return Schemas{}, fmt.Errorf("error unmarshalling genned schema: %w", err)
	}

	return Schemas{All: genned.Components.Schemas}, err
}

// MarshalCueValue generates an openAPI schema for the given cue value,
// utilizing Cue's OpenAPI integration package.  This returns a single schema
// for the given Cue value - the value must be a Cue struct containing type
// definitions.
func MarshalCueValue(v cue.Value) (map[string]interface{}, error) {
	// We need to transform the value to a *cue.Instance.
	// TODO: A bvetter way other than formatting and re-parsing to generate
	// the instance.
	val, err := formatValue(v, cue.Attributes(true))
	if err != nil {
		return nil, fmt.Errorf("error formatting instance value: %w", err)
	}

	schemas, err := MarshalString(fmt.Sprintf("#event: %s", val))
	if err != nil {
		return nil, err
	}

	return schemas.Find("event"), nil
}

// Schemas stores all schemas generated for a cue file.
type Schemas struct {
	// All stores all generated schemas, in a map.
	All map[string]map[string]interface{}
}

// Find returns a schema for the given identifier
func (s Schemas) Find(identifier string) map[string]interface{} {
	val, _ := s.All[identifier]
	return val
}

// genned represents the generated data from Cue's openapi package.  We care
// only about extracting the event schema from the generated package;  the
// rest is discarded.
type genned struct {
	Components struct {
		// Schemas lists all top-level
		Schemas map[string]map[string]interface{}
	}
}

// formatValue formats a given cue value as well-defined cue config.
func formatValue(input cue.Value, opts ...cue.Option) (string, error) {
	opts = append([]cue.Option{
		cue.Docs(true),
		cue.Optional(true),
		cue.Definitions(true),
		cue.ResolveReferences(true),
	}, opts...)

	syn := input.Syntax(opts...)
	return formatNode(syn)
}

func formatNode(input ast.Node, opts ...format.Option) (string, error) {
	out, err := format.Node(
		input,
		format.TabIndent(false),
		format.UseSpaces(2),
	)
	return string(out), err
}
