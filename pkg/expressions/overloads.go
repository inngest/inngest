package expressions

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	"github.com/google/cel-go/parser"
	"github.com/inngest/inngest/pkg/dateutil"
	str2duration "github.com/xhit/go-str2duration/v2"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type customLibrary struct{}

// EnvOptions returns options for the standard CEL function declarations and macros.
func (customLibrary) CompileOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.Declarations(celDeclarations()...),
		cel.Macros(parser.AllMacros...),
	}
}

// ProgramOptions returns function implementations for the standard CEL functions.
func (customLibrary) ProgramOptions() []cel.ProgramOption {
	overloads := celOverloads()

	return []cel.ProgramOption{
		// Always inject standard overloads into our program
		cel.Functions(overloads...),
		cel.EvalOptions(cel.OptExhaustiveEval, cel.OptTrackState, cel.OptPartialEval),
	}
}

func celDeclarations() []*exprpb.Decl {
	// Take the standard overloads from checker.  We'll filter this to add our own
	// heterogeneous comparison types that allow any type conversions.
	filtered := []*exprpb.Decl{}
	for _, d := range checker.StandardTypes() {
		if _, ok := d.DeclKind.(*exprpb.Decl_Function); ok {
			// This is a function that we will overload directly in the cel env to
			// provide easier type handling.
			if d.Name == operators.Add ||
				d.Name == operators.Equals ||
				d.Name == operators.NotEquals ||
				d.Name == operators.LessEquals ||
				d.Name == operators.Less ||
				d.Name == operators.Greater ||
				d.Name == operators.GreaterEquals {
				continue
			}
		}

		filtered = append(filtered, d)
	}

	custom := []*exprpb.Decl{
		// We add replicas of functions.StandardOverloads to the cel environment here.
		// Even though the StdLib in cel adds functions.StandardOverloads to programs,
		// programs are not used at compile time, and so doing things like "3 < 3.141"
		// straight up breaks when generating an AST for the program.
		//
		// We do not need to re-implement the functionality for these overloads - they
		// are already implemented within functions/standard using traits.
		decls.NewFunction(
			operators.Add,
			decls.NewOverload("add_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Any)),
		decls.NewFunction(
			operators.Equals,
			decls.NewOverload("equals_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),
		decls.NewFunction(
			operators.NotEquals,
			decls.NewOverload("not_equals_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),
		decls.NewFunction(
			operators.LessEquals,
			decls.NewOverload("less_equals_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),
		decls.NewFunction(
			operators.Less,
			decls.NewOverload("less_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),
		decls.NewFunction(
			operators.Greater,
			decls.NewOverload("greater_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),
		decls.NewFunction(
			operators.GreaterEquals,
			decls.NewOverload("greater_equals_any",
				[]*expr.Type{decls.Any, decls.Any}, decls.Bool)),

		// Custom functions.  These are added for convenience.
		decls.NewFunction(
			"lowercase",
			decls.NewOverload(
				"lowercase",
				[]*expr.Type{decls.String},
				decls.String,
			),
		),
		decls.NewFunction(
			"uppercase",
			decls.NewOverload(
				"uppercase",
				[]*expr.Type{decls.String},
				decls.String,
			),
		),
		decls.NewFunction(
			"b64decode",
			decls.NewOverload(
				"b64decode",
				[]*expr.Type{decls.String},
				decls.String,
			),
		),
		decls.NewFunction(
			"json_parse",
			decls.NewOverload(
				"json_parse",
				[]*expr.Type{decls.String},
				decls.Any,
			),
		),

		// Time functions
		decls.NewFunction(
			"date",
			decls.NewOverload(
				"to_date",
				[]*expr.Type{decls.Any},
				decls.Timestamp,
			),
		),
		decls.NewFunction(
			"now",
			decls.NewOverload(
				"now",
				[]*expr.Type{},
				decls.Timestamp,
			),
		),
		decls.NewFunction(
			"now_plus",
			decls.NewOverload(
				"now_plus",
				[]*expr.Type{decls.String},
				decls.Timestamp,
			),
		),
		decls.NewFunction(
			"now_minus",
			decls.NewOverload(
				"now_minus",
				[]*expr.Type{decls.String},
				decls.Timestamp,
			),
		),
	}

	return append(filtered, custom...)
}

func celOverloads() []*functions.Overload {
	return []*functions.Overload{
		{
			Operator: "to_date",
			Unary: func(i ref.Val) ref.Val {
				switch i.Type().TypeName() {
				case "string":
					t, err := dateutil.ParseString(i.Value().(string))
					if err == nil {
						return types.Timestamp{Time: t}
					}
				case "int":
					t, err := dateutil.ParseInt(i.Value().(int64))
					if err == nil {
						return types.Timestamp{Time: t}
					}
				}
				return nil
			},
		},
		{
			Operator: "lowercase",
			Unary: func(i ref.Val) ref.Val {
				str, _ := i.Value().(string)
				return types.String(strings.ToLower(str))
			},
		},
		{
			Operator: "uppercase",
			Unary: func(i ref.Val) ref.Val {
				str, _ := i.Value().(string)
				return types.String(strings.ToUpper(str))
			},
		},
		{
			Operator: "b64decode",
			Unary: func(i ref.Val) ref.Val {
				str, _ := i.Value().(string)
				byt, _ := base64.StdEncoding.DecodeString(str)
				return types.String(byt)
			},
		},
		{
			Operator: "json_parse",
			Unary: func(i ref.Val) ref.Val {
				str, _ := i.Value().(string)
				mapped := map[string]interface{}{}
				_ = json.Unmarshal([]byte(str), &mapped)
				return types.NewStringInterfaceMap(types.DefaultTypeAdapter, mapped)
			},
		},
		{
			Operator: "now",
			Function: func(args ...ref.Val) ref.Val {
				t := time.Now()
				return types.Timestamp{Time: t}
			},
		},
		{
			Operator: "now_minus",
			Unary: func(i ref.Val) ref.Val {
				if i.Type().TypeName() != "string" {
					// cel should already take care of this for us
					return nil
				}
				now := time.Now()
				duration, err := str2duration.ParseDuration(i.Value().(string))
				if err != nil {
					return nil
				}
				t := now.Add(-1 * duration)
				return types.Timestamp{Time: t}
			},
		},
		{
			Operator: "now_plus",
			Unary: func(i ref.Val) ref.Val {
				if i.Type().TypeName() != "string" {
					// cel should already take care of this for us
					return nil
				}

				now := time.Now()
				duration, err := str2duration.ParseDuration(i.Value().(string))
				if err != nil {
					return nil
				}
				t := now.Add(duration)
				return types.Timestamp{Time: t}
			},
		},
	}
}
