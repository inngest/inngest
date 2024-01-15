package expressions

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"
)

// unknownDecorator returns a decorator for inspecting and handling unknowns at runtime.  This
// decorator is called before _any_ attribute or function is evaluated in CEL, allowing us to
// intercept and return our own values.
//
// For example, natively in CEL `size(null)` returns a "no such overload" error.  We intercept
// the evalutation of `size(null)` and return a new type (0) instead of the error.
func unknownDecorator(act interpreter.PartialActivation) interpreter.InterpretableDecorator {
	// Create a new dispatcher with all functions added
	dispatcher := interpreter.NewDispatcher()
	overloads := celOverloads()
	_ = dispatcher.Add(overloads...)

	return func(i interpreter.Interpretable) (interpreter.Interpretable, error) {
		// If this is a fold call, this is a macro (exists, has, etc), and is not an InterpretableCall
		call, ok := i.(interpreter.InterpretableCall)
		if !ok {
			return i, nil
		}

		fnVal := call.OverloadID()
		if fnVal == "" {
			fnVal = call.Function()
		}

		argTypes := &argColl{}

		args := call.Args()
		for _, arg := range args {
			// We want both attributes (variables) & consts to check for coercion.
			argTypes.Add(arg.Eval(act))
		}

		if argTypes.TypeLen() == 1 && !argTypes.Exists(types.ErrType) && !argTypes.Exists(types.UnknownType) {
			// A single type used within the function with no error and unknown is
			// safe to call as usual.
			return i, nil
		}

		// For each function that we wnat to be heterogeneous, check the types here.
		//
		// Only run this in the case in which we have known types;  unknowns are handled
		// below.
		if argTypes.TypeLen() == 2 && !argTypes.Exists(types.UnknownType) {
			// Check if the original function is a success.
			val := call.Eval(act)
			if !types.IsError(val) && !types.IsUnknown(val) {
				// Memoize this result and return it.
				return staticCall{result: val, InterpretableCall: call}, nil
			}

			switch fnVal {
			case operators.Add:
				//
				// This allows concatenation of distinct types, eg string + number.
				//
				str := types.String(fmt.Sprintf("%v%v", args[0].Eval(act).Value(), args[1].Eval(act).Value()))
				return staticCall{result: str, InterpretableCall: call}, nil
			}
		}

		if argTypes.Exists(types.ErrType) || argTypes.Exists(types.UnknownType) {
			// We work with unknown and error types, handling both as non-existent
			// types.
			//
			// Errors can appear when calling macros on unknowns (eg:
			// event.data.nonexistent.subkey.contains("something")).
			//
			// This could be because:
			//
			// 1. we're calling a macro on an unknown value. This happens before we can intercept
			// the InterpretableCall and will always happen.  That's fine, and this produces the
			// error "no such key".  These are the errors we want to intercept.
			//
			// 2. we're inside a macro and we're using the __result__ or lambda
			//    variable.  This error contains "no such attribute", and this is a usual
			//    part of runing macros.  XXX: Figure out why Eval() on macro variables fails:
			//    this is actually _not_ an error.
			for _, val := range argTypes.OfType(types.ErrType) {
				if strings.HasPrefix(val.(error).Error(), "no such attribute") {
					// This must be a macro call;  handle as usual
					return i, nil
				}
			}
			// This is an unknown type.  Dependent on the function being called return
			// a concrete true or false value by default.
			return handleUnknownCall(call, argTypes)
		}

		// Here we have multiple types called together.  If these are coercible, we'll
		// attempt to coerce them (eg. ints and floats).
		//
		// We can't create a custom null type, because Compare may run on String, Int, Double,
		// etc:  we'd have to wrap every type and add null checking.  This is a maintenance
		// en and could be bug-prone.
		//
		// Therefore, we have to evaluate this here within a decorator.
		if argTypes.Exists(types.NullType) && argTypes.ArgLen() == 2 {
			switch call.Function() {
			case operators.Equals:
				return staticCall{result: types.False, InterpretableCall: call}, nil
			case operators.NotEquals:
				return staticCall{result: types.True, InterpretableCall: call}, nil
			}

			// Other operators, such as >, <=, depend on the argument order to evaluate
			// correctly.
			//
			// We must create a new zero type in place of the null argument,
			// then fetch the overload from the standard dispatcher and run the function.
			args, err := argTypes.ZeroValArgs()
			if err != nil {
				return i, nil
			}

			// Get the actual implementation which we've copied into overloads.go.
			fn := getBindings(call.Function(), nil)
			if fn == nil {
				return i, nil
			}
			return staticCall{result: fn(args[0], args[1]), InterpretableCall: call}, nil
		}

		return i, nil
	}
}

// By default, CEL tracks unknowns as a separate value.  This is fantastic, but when
// we're evaluating expressions we want to treat unknowns as nulls.
//
// This functionality adds custom logic for each overload to return a static ref.Val
// which is used in place of unknown.
func handleUnknownCall(i interpreter.InterpretableCall, args *argColl) (interpreter.Interpretable, error) {
	switch i.Function() {
	case operators.Add:
		// Find the non-unknown type and return that
		for _, arg := range args.arguments {
			if types.IsUnknown(arg) {
				continue
			}
			return staticCall{result: arg, InterpretableCall: i}, nil
		}
		return staticCall{result: types.False, InterpretableCall: i}, nil
	case operators.Equals:
		// Comparing an unknown to null is true, else return false.
		result := types.False
		if args.Exists(types.NullType) {
			result = types.True
		}
		return staticCall{result: result, InterpretableCall: i}, nil

	case operators.NotEquals:
		if args.Exists(types.NullType) {
			// Unknowns are null, so this is false.
			return staticCall{result: types.False, InterpretableCall: i}, nil
		}
		// Are we comparing against a zero type (eg. empty string, 0).
		// The only item that should return true is not equals, as nil is always not equals
		return staticCall{result: types.True, InterpretableCall: i}, nil

	case operators.Less, operators.LessEquals:
		// Unknown is less than anything.
		if args.arguments[0].Type() == types.UnknownType || args.arguments[0].Type() == types.ErrType {
			return staticCall{result: types.True, InterpretableCall: i}, nil
		}
		return staticCall{result: types.False, InterpretableCall: i}, nil

	case operators.Greater, operators.GreaterEquals:
		// If the first arg is unkown, return false:  unknown is not greater.
		if args.arguments[0].Type() == types.UnknownType || args.arguments[0].Type() == types.ErrType {
			return staticCall{result: types.False, InterpretableCall: i}, nil
		}
		return staticCall{result: types.True, InterpretableCall: i}, nil

	case overloads.Size,
		overloads.SizeString,
		overloads.SizeBytes,
		overloads.SizeList,
		overloads.SizeStringInst,
		overloads.SizeBytesInst,
		overloads.SizeListInst,
		overloads.SizeMapInst:
		// Size on unknowns should always return zero to avoid type errors.
		return staticCall{result: types.IntZero, InterpretableCall: i}, nil
	default:

		// By default, return false, for eaxmple: "_<_", "@in", "@not_strictly_false"
		// return staticCall{result: types.False, InterpretableCall: call}, nil
		return staticCall{result: types.False, InterpretableCall: i}, nil
	}
}

// staticCall represents a wrapped interpreter.InterpretableCall function within
// an expression that always returns a static value.
type staticCall struct {
	interpreter.InterpretableCall
	result ref.Val
}

func (u staticCall) Eval(ctx interpreter.Activation) ref.Val {
	return u.result
}

// argColl inspects all of the types available within a function call in
// CEL, storing their type information.
type argColl struct {
	// types represents a map of types to their values used within the
	// function.
	types map[ref.Type][]ref.Val

	// arguments represents the function arguments, in order.
	arguments []ref.Val
}

// Add adds a new value to the type collection, storing its type in the map.
func (t *argColl) Add(vals ...ref.Val) {
	if t.types == nil {
		t.types = map[ref.Type][]ref.Val{}
	}

	for _, val := range vals {
		// Store the arguments in order (left and right hand side of operators)
		t.arguments = append(t.arguments, val)

		if val == nil {
			// XXX: We should probably handle this differently
			continue
		}

		typ := val.Type()
		coll, ok := t.types[typ]
		if !ok {
			t.types[typ] = []ref.Val{val}
			return
		}
		t.types[typ] = append(coll, val)
	}
}

func (t *argColl) TypeLen() int {
	return len(t.types)
}

func (t *argColl) ArgLen() int {
	return len(t.arguments)
}

func (t *argColl) Exists(typ ref.Type) bool {
	_, ok := t.types[typ]
	return ok
}

// OfType returns all arguments of the given type.
func (t *argColl) OfType(typ ref.Type) []ref.Val {
	coll, ok := t.types[typ]
	if !ok {
		return nil
	}
	return coll
}

// NonNull returns all non-null types as a slice.
func (t *argColl) NonNull() []ref.Type {
	coll := []ref.Type{}
	for typ := range t.types {
		if typ == types.NullType {
			continue
		}
		coll = append(coll, typ)
	}
	return coll
}

// ZeroValArgs returns all args with null types replaced as zero values
func (t *argColl) ZeroValArgs() ([]ref.Val, error) {
	typ := t.NonNull()
	if len(typ) != 1 {
		return t.arguments, fmt.Errorf("not exactly one other non-null type present")
	}

	coll := make([]ref.Val, len(t.arguments))
	for n, arg := range t.arguments {
		coll[n] = arg
		if arg.Type() == types.NullType {
			coll[n] = zeroVal(typ[0])
		}
	}

	return coll, nil
}

// zeroVal returns a zero value for common cel datatypes.  This helps us
// convert null values to a zero value of a specific type.
func zeroVal(t ref.Type) ref.Val {
	switch t.TypeName() {
	case "int":
		return types.IntZero
	case "uint":
		return types.Uint(0)
	case "double":
		return types.Double(0)
	case "string":
		return types.String("")
	}

	return types.NullValue
}
