package expressions

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter"
	"github.com/inngest/inngest/pkg/expressions/exprenv"
)

// unknownDecorator returns a data-independent decorator that wraps every InterpretableCall
// in a runtimeUnknownCall.  The actual type-dispatch logic (unknown/null/coercion handling)
// runs at eval time instead of plan time, which means the cel.Program can be built once and
// cached for the lifetime of the expression rather than rebuilt on every evaluation.
func unknownDecorator() interpreter.InterpretableDecorator {
	return func(i interpreter.Interpretable) (interpreter.Interpretable, error) {
		// Handle logical OR/AND nodes.  CEL represents || and && as special
		// evalOr/evalAnd (or evalExhaustiveOr/evalExhaustiveAnd) structs that
		// are NOT InterpretableCall.  They require boolean operands, but users
		// commonly write "event.data.a || event.data.b" expecting JS-like truthy
		// coercion.  We detect these nodes via reflection and wrap them to
		// implement truthy coercion when operands are non-boolean.
		if wrapped, ok := maybeWrapLogicalOp(i); ok {
			return wrapped, nil
		}

		// If this is a fold call, this is a macro (exists, has, etc), and is not an InterpretableCall
		call, ok := i.(interpreter.InterpretableCall)
		if !ok {
			return i, nil
		}

		return &runtimeUnknownCall{InterpretableCall: call}, nil
	}
}

// runtimeUnknownCall wraps an InterpretableCall and applies the same unknown/null/coercion
// handling logic as the old plan-time unknownDecorator, but at eval time using the live
// activation.  This is what allows cel.Program to be safely cached across evaluations.
type runtimeUnknownCall struct {
	interpreter.InterpretableCall
}

func (r *runtimeUnknownCall) Eval(act interpreter.Activation) ref.Val {
	fnVal := r.OverloadID()
	if fnVal == "" {
		fnVal = r.Function()
	}

	var argTypes argColl
	args := r.Args()
	for _, arg := range args {
		argTypes.add(arg.Eval(act))
	}

	if argTypes.TypeLen() == 1 && !argTypes.Exists(types.ErrType) && !argTypes.Exists(types.UnknownType) {
		// A single type used within the function with no error and unknown is
		// safe to call as usual.
		return r.InterpretableCall.Eval(act)
	}

	// For each function that we want to be heterogeneous, check the types here.
	//
	// Only run this in the case in which we have known types;  unknowns are handled
	// below.
	if argTypes.TypeLen() == 2 && !argTypes.Exists(types.UnknownType) {
		val := r.InterpretableCall.Eval(act)
		if !types.IsError(val) && !types.IsUnknown(val) {
			return val
		}

		if fnVal == operators.Add {
			// This allows concatenation of distinct types, eg string + number.
			return types.String(fmt.Sprintf("%v%v", args[0].Eval(act).Value(), args[1].Eval(act).Value()))
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
		for i := 0; i < argTypes.n && i < 2; i++ {
			if argTypes.a[i] != nil && argTypes.a[i].Type() == types.ErrType {
				if strings.HasPrefix(argTypes.a[i].(error).Error(), "no such attribute") {
					// This must be a macro call;  handle as usual
					return r.InterpretableCall.Eval(act)
				}
			}
		}
		// This is an unknown type.  Dependent on the function being called return
		// a concrete true or false value by default.
		result, _ := handleUnknownCall(r.InterpretableCall, argTypes)
		return result.Eval(act)
	}

	// Here we have multiple types called together.  If these are coercible, we'll
	// attempt to coerce them (eg. ints and floats).
	//
	// We can't create a custom null type, because Compare may run on String, Int, Double,
	// etc:  we'd have to wrap every type and add null checking.  This is a maintenance
	// burden and could be bug-prone.
	//
	// Therefore, we have to evaluate this here within a decorator.
	if argTypes.Exists(types.NullType) && argTypes.ArgLen() == 2 {
		switch r.Function() {
		case operators.Equals:
			return types.False
		case operators.NotEquals:
			return types.True
		}

		// Other operators, such as >, <=, depend on the argument order to evaluate
		// correctly.
		//
		// We must create a new zero type in place of the null argument,
		// then fetch the overload from the standard dispatcher and run the function.
		zeroArgs, err := argTypes.zeroValArgs()
		if err != nil {
			return r.InterpretableCall.Eval(act)
		}

		// Get the actual implementation which we've copied into overloads.go.
		fn := exprenv.GetBindings(r.Function(), nil)
		if fn == nil {
			return r.InterpretableCall.Eval(act)
		}
		return fn(zeroArgs[0], zeroArgs[1])
	}

	return r.InterpretableCall.Eval(act)
}

// By default, CEL tracks unknowns as a separate value.  This is fantastic, but when
// we're evaluating expressions we want to treat unknowns as nulls.
//
// This functionality adds custom logic for each overload to return a static ref.Val
// which is used in place of unknown.
func handleUnknownCall(i interpreter.InterpretableCall, args argColl) (interpreter.Interpretable, error) {
	switch i.Function() {
	case operators.Add:
		// Find the non-unknown type and return that
		for j := 0; j < args.n && j < 2; j++ {
			if types.IsUnknown(args.a[j]) {
				continue
			}
			return staticCall{result: args.a[j], InterpretableCall: i}, nil
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
		if args.a[0].Type() == types.UnknownType || args.a[0].Type() == types.ErrType {
			return staticCall{result: types.True, InterpretableCall: i}, nil
		}
		return staticCall{result: types.False, InterpretableCall: i}, nil

	case operators.Greater, operators.GreaterEquals:
		// If the first arg is unknown, return false:  unknown is not greater.
		if args.a[0].Type() == types.UnknownType || args.a[0].Type() == types.ErrType {
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
		return staticCall{result: types.False, InterpretableCall: i}, nil
	}
}

// maybeWrapLogicalOp detects CEL's internal evalOr/evalAnd (and their exhaustive
// variants) via reflection and wraps them to implement JS-like truthy coercion
// when operands are non-boolean.  Returns (wrapped, true) if the node was wrapped,
// or (nil, false) if the node is not a logical operator.
func maybeWrapLogicalOp(i interpreter.Interpretable) (interpreter.Interpretable, bool) {
	typeName := reflect.TypeOf(i).String()

	var isOr bool
	switch {
	case strings.Contains(typeName, "evalOr") || strings.Contains(typeName, "evalExhaustiveOr"):
		isOr = true
	case strings.Contains(typeName, "evalAnd") || strings.Contains(typeName, "evalExhaustiveAnd"):
		isOr = false
	default:
		return nil, false
	}

	// Extract the terms field via reflection + unsafe, since the CEL structs
	// are unexported.  The struct layout is:
	//   type eval{Exhaustive}{Or,And} struct {
	//       id    int64
	//       terms []interpreter.Interpretable
	//   }
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	termsField := v.FieldByName("terms")
	if !termsField.IsValid() {
		return nil, false
	}

	// Use unsafe to read the unexported field.
	terms := *(*[]interpreter.Interpretable)(unsafe.Pointer(termsField.UnsafeAddr()))

	return &evalTruthyLogical{
		inner: i,
		terms: terms,
		isOr:  isOr,
	}, true
}

// evalTruthyLogical wraps a CEL evalOr/evalAnd node to support JS-like truthy
// coercion for non-boolean operands.  If all operands are booleans, it delegates
// to the original CEL evaluation.
type evalTruthyLogical struct {
	inner interpreter.Interpretable
	terms []interpreter.Interpretable
	isOr  bool
}

func (e *evalTruthyLogical) ID() int64 {
	return e.inner.ID()
}

func (e *evalTruthyLogical) Eval(ctx interpreter.Activation) ref.Val {
	// CEL's || and && produce nested binary AST nodes by default; evalExhaustiveOr/And
	// inherits that 2-element terms slice.  This breaks if the env ever enables
	// cel.variadicLogicalOperatorASTs() which is not even exported can break this assumption.
	var vals [2]ref.Val
	allBool := true
	for i, term := range e.terms {
		vals[i] = term.Eval(ctx)
		if vals[i] == nil || vals[i].Type() != types.BoolType {
			allBool = false
		}
	}

	// If all operands are booleans, delegate to native CEL evaluation.
	if allBool {
		return e.inner.Eval(ctx)
	}

	// Apply JS-like truthy coercion.
	n := len(e.terms)
	if e.isOr {
		// Return the first truthy value, or the last value.
		for i := 0; i < n; i++ {
			if isTruthy(vals[i]) {
				return vals[i]
			}
		}
		return vals[n-1]
	}

	// AND: return the first falsy value, or the last value.
	for i := 0; i < n; i++ {
		if !isTruthy(vals[i]) {
			return vals[i]
		}
	}
	return vals[n-1]
}

// isTruthy returns whether a CEL value is "truthy" using JS-like semantics.
// Falsy values: unknown, error, null, false, empty string, 0, empty list/map.
func isTruthy(v ref.Val) bool {
	if v == nil || types.IsUnknown(v) || types.IsError(v) {
		return false
	}
	switch v.Type() {
	case types.NullType:
		return false
	case types.BoolType:
		return v.Value().(bool)
	case types.StringType:
		return v.Value().(string) != ""
	case types.IntType:
		return v.Value().(int64) != 0
	case types.UintType:
		return v.Value().(uint64) != 0
	case types.DoubleType:
		return v.Value().(float64) != 0
	default:
		return true
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

// argColl tracks the argument values and distinct types for a binary operator call.
// CEL operators are always binary (2 args), so fixed-size arrays are used throughout
// to avoid any heap allocation.
type argColl struct {
	a      [2]ref.Val // arguments in order
	n      int        // number of arguments added (0–2)
	t0, t1 ref.Type  // distinct types seen; t0 is set first
	nt     int        // number of distinct types (0–2)
}

func (t *argColl) add(val ref.Val) {
	if t.n < 2 {
		t.a[t.n] = val
	}
	t.n++
	if val == nil {
		return
	}
	typ := val.Type()
	switch t.nt {
	case 0:
		t.t0 = typ
		t.nt = 1
	case 1:
		if t.t0 != typ {
			t.t1 = typ
			t.nt = 2
		}
	}
}

func (t *argColl) TypeLen() int { return t.nt }
func (t *argColl) ArgLen() int  { return t.n }

func (t *argColl) Exists(typ ref.Type) bool {
	return (t.nt >= 1 && t.t0 == typ) || (t.nt >= 2 && t.t1 == typ)
}

// zeroValArgs returns the two args with any null replaced by the zero value of the
// other type.  Returns an error if there isn't exactly one non-null type present.
func (t *argColl) zeroValArgs() ([2]ref.Val, error) {
	var nonNull ref.Type
	nn := 0
	if t.nt >= 1 && t.t0 != types.NullType {
		nonNull = t.t0
		nn++
	}
	if t.nt >= 2 && t.t1 != types.NullType {
		nonNull = t.t1
		nn++
	}
	if nn != 1 {
		return t.a, fmt.Errorf("not exactly one other non-null type present")
	}
	result := t.a
	zero := zeroVal(nonNull)
	for i := 0; i < t.n && i < 2; i++ {
		if t.a[i] != nil && t.a[i].Type() == types.NullType {
			result[i] = zero
		}
	}
	return result, nil
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
