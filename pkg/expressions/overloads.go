package expressions

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/inngest/inngest/pkg/dateutil"
	"github.com/xhit/go-str2duration/v2"

	// "github.com/google/cel-go/checker/decls"

	"github.com/google/cel-go/common/decls"
	"github.com/google/cel-go/common/functions"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/stdlib"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/parser"
)

type customLibrary struct{}

// EnvOptions returns options for the standard CEL function declarations and macros.
func (customLibrary) CompileOptions() []cel.EnvOption {
	stdFns := stdlib.Functions()

	// First, add all macros.
	envOpts := []cel.EnvOption{
		cel.Macros(parser.AllMacros...),
	}

	// Then, add all StdLib functions (eg. _||_, _>_) with disabled type guards
	for _, fn := range stdFns {
		copied := *fn

		if fn.Name() == operators.Add ||
			fn.Name() == operators.Equals ||
			fn.Name() == operators.NotEquals ||
			fn.Name() == operators.LessEquals ||
			fn.Name() == operators.Less ||
			fn.Name() == operators.Greater ||
			fn.Name() == operators.GreaterEquals {

			// These are functions in which we want to be loosely typed,
			// ie. they should work with heterogeneously typed inputs such as
			// adding a string and an int.
			//
			// CEL, by default, is strongly typed and does not allow for this.
			//
			// Instead of adding the default StdLib functions, we add our own
			// function kind with the default overload and an any type param.
			overloads, err := fn.Bindings()
			if err != nil || len(overloads) != 1 {
				continue
			}

			ol := overloads[0]

			var resultType *types.Type
			switch fn.Name() {
			case operators.Add:
				resultType = types.AnyType
			default:
				// Equals, NotEquals, etc. all return booleans.
				resultType = types.BoolType
			}

			custom := newFunctionEnvOption(
				fn.Name(),
				// Add a new overload for the stdlib, accepting any types and returning
				// the above  resultType.
				decls.Overload(
					fn.Name(),
					[]*types.Type{types.AnyType, types.AnyType},
					resultType,
					// OverloadIsNonStrict allows us to pass in unknown, null,
					// or error values to this type.
					// decls.OverloadIsNonStrict(),
					// Add the existing fn traits.
					decls.OverloadOperandTrait(ol.OperandTrait),
					// Ensure we pass in the actual function logic here.
					decls.BinaryBinding(getBindings(fn.Name(), ol.Binary)),
				),
				decls.DisableTypeGuards(true),
			)

			envOpts = append(envOpts, custom)
			continue
		}

		opt := cel.Function(
			fn.Name(),
			// Use the existing stdlib decl as-is.
			func(d *decls.FunctionDecl) (*decls.FunctionDecl, error) {
				return &copied, nil
			},
			decls.DisableTypeGuards(true),
		)
		envOpts = append(envOpts, opt)
	}

	// Then add custom functions.
	return append(envOpts, celDeclarations()...)
}

func getBindings(name string, existing functions.BinaryOp) functions.BinaryOp {
	switch name {
	case operators.Add:
		return func(lhs, rhs ref.Val) ref.Val {
			return lhs.(traits.Adder).Add(rhs)
		}
	case operators.Less:
		return func(lhs, rhs ref.Val) ref.Val {
			cmp := lhs.(traits.Comparer).Compare(rhs)
			if cmp == types.IntNegOne {
				return types.True
			}
			if cmp == types.IntOne || cmp == types.IntZero {
				return types.False
			}
			return cmp
		}
	case operators.LessEquals:
		return func(lhs, rhs ref.Val) ref.Val {
			cmp := lhs.(traits.Comparer).Compare(rhs)
			if cmp == types.IntNegOne || cmp == types.IntZero {
				return types.True
			}
			if cmp == types.IntOne {
				return types.False
			}
			return cmp
		}
	case operators.Greater:
		return func(lhs, rhs ref.Val) ref.Val {
			cmp := lhs.(traits.Comparer).Compare(rhs)
			if cmp == types.IntOne {
				return types.True
			}
			if cmp == types.IntNegOne || cmp == types.IntZero {
				return types.False
			}
			return cmp
		}
	case operators.GreaterEquals:
		return func(lhs, rhs ref.Val) ref.Val {
			cmp := lhs.(traits.Comparer).Compare(rhs)
			if cmp == types.IntOne || cmp == types.IntZero {
				return types.True
			}
			if cmp == types.IntNegOne {
				return types.False
			}
			return cmp
		}
	}
	return existing
}

// ProgramOptions returns function implementations for the standard CEL functions.
func (customLibrary) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{
		// Add our custom function implementations (overloads) into the fn.
		cel.Functions(celOverloads()...),
		cel.EvalOptions(cel.OptExhaustiveEval, cel.OptTrackState, cel.OptPartialEval),
	}
}

// newFunctioEnvOption creates a new *decls.FunctionDecl and wraps this within
// a cel.Function() call to create an EnvOption for modifying the cel environment.
//
// An alternative is to call decls.FunctioNDeclToExprDecl and use the
// *exprpb.Decl types directly within cel.Declarations as an EnvOption.
func newFunctionEnvOption(name string, opts ...decls.FunctionOpt) cel.EnvOption {
	fn, err := decls.NewFunction(name, opts...)
	if err != nil {
		panic(err.Error())
	}

	return cel.Function(
		name,
		// Use the existing stdlib decl as-is.
		func(d *decls.FunctionDecl) (*decls.FunctionDecl, error) {
			return fn, nil
		},
	)
}

func celDeclarations() []cel.EnvOption {
	custom := []cel.EnvOption{
		// Custom functions.  These are added for convenience to our users.
		newFunctionEnvOption(
			"lowercase",
			decls.Overload(
				"lowercase",
				[]*types.Type{types.StringType},
				types.StringType,
			),
		),
		newFunctionEnvOption(
			"uppercase",
			decls.Overload(
				"uppercase",
				[]*types.Type{types.StringType},
				types.StringType,
			),
		),
		newFunctionEnvOption(
			"b64decode",
			decls.Overload(
				"b64decode",
				[]*types.Type{types.StringType},
				types.StringType,
			),
		),
		newFunctionEnvOption(
			"json_parse",
			decls.Overload(
				"json_parse",
				[]*types.Type{types.StringType},
				types.AnyType,
			),
		),

		// Time functions
		newFunctionEnvOption(
			"date",
			decls.Overload(
				"date",
				[]*types.Type{types.AnyType},
				types.TimestampType,
			),
		),
		newFunctionEnvOption(
			"now",
			decls.Overload(
				"now",
				[]*types.Type{},
				types.TimestampType,
			),
		),
		newFunctionEnvOption(
			"now_plus",
			decls.Overload(
				"now_plus",
				[]*types.Type{types.StringType},
				types.TimestampType,
			),
		),
		newFunctionEnvOption(
			"now_minus",
			decls.Overload(
				"now_minus",
				[]*types.Type{types.StringType},
				types.TimestampType,
			),
		),
	}

	// return append(filtered, custom...)
	return custom
}

func celOverloads() []*functions.Overload {
	return []*functions.Overload{
		{
			Operator: "date",
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
