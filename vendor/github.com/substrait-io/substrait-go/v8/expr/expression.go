// SPDX-License-Identifier: Apache-2.0

package expr

import (
	"fmt"
	"slices"
	"strings"

	substraitgo "github.com/substrait-io/substrait-go/v8"
	"github.com/substrait-io/substrait-go/v8/extensions"
	"github.com/substrait-io/substrait-go/v8/types"
	proto "github.com/substrait-io/substrait-protobuf/go/substraitpb"
)

// MustExpr is a helper function to avoid having it get written and
// re-written by everyone. It takes an expression and an error
// and will panic if the error is non-nil, otherwise returning the
// expression.
//
// This is intended for situations where it's okay to panic, like
// testing, to make it more convenient when building expressions
// to use the `New...` functions that return some expression and
// potentially an error. For instance:
//
//	NewScalarFunc(reg, id, nil, MustExpr(NewRootFieldRef(...)),
//	    MustExpr(NewScalarFunc(...)))

type (
	// VirtualTableExpressionValue is a slice of other expression where each
	// element in the slice is a different field in the struct
	VirtualTableExpressionValue []Expression
)

func MustExpr(e Expression, err error) Expression {
	if err != nil {
		panic(err)
	}

	return e
}

func FuncArgFromProto(e *proto.FunctionArgument, baseSchema *types.RecordType, reg ExtensionRegistry) (types.FuncArg, error) {
	switch et := e.ArgType.(type) {
	case *proto.FunctionArgument_Enum:
		return types.Enum(et.Enum), nil
	case *proto.FunctionArgument_Type:
		return types.TypeFromProto(et.Type), nil
	case *proto.FunctionArgument_Value:
		return ExprFromProto(et.Value, baseSchema, reg)
	}
	return nil, substraitgo.ErrNotImplemented
}

func ExprFromProto(e *proto.Expression, baseSchema *types.RecordType, reg ExtensionRegistry) (Expression, error) {
	if e == nil {
		return nil, fmt.Errorf("%w: protobuf Expression is nil", substraitgo.ErrInvalidExpr)
	}

	switch et := e.RexType.(type) {
	case *proto.Expression_Literal_:
		return LiteralFromProto(et.Literal), nil
	case *proto.Expression_Selection:
		return FieldReferenceFromProto(et.Selection, baseSchema, reg)
	case *proto.Expression_ScalarFunction_:
		var err error
		args := make([]types.FuncArg, len(et.ScalarFunction.Arguments))
		for i, a := range et.ScalarFunction.Arguments {
			if args[i], err = FuncArgFromProto(a, baseSchema, reg); err != nil {
				return nil, err
			}
		}

		id, ok := reg.DecodeFunc(et.ScalarFunction.FunctionReference)
		if !ok {
			return nil, substraitgo.ErrNotFound
		}

		decl, ok := reg.LookupScalarFunction(et.ScalarFunction.FunctionReference)
		if !ok {
			return NewCustomScalarFunc(reg, extensions.NewScalarFuncVariant(id), types.TypeFromProto(et.ScalarFunction.OutputType), et.ScalarFunction.Options, args...)
		}

		return &ScalarFunction{
			funcRef:     et.ScalarFunction.FunctionReference,
			declaration: decl,
			args:        args,
			options:     et.ScalarFunction.Options,
			outputType:  types.TypeFromProto(et.ScalarFunction.OutputType),
		}, nil
	case *proto.Expression_WindowFunction_:
		var err error
		args := make([]types.FuncArg, len(et.WindowFunction.Arguments))
		for i, a := range et.WindowFunction.Arguments {
			if args[i], err = FuncArgFromProto(a, baseSchema, reg); err != nil {
				return nil, err
			}
		}

		parts := make([]Expression, len(et.WindowFunction.Partitions))
		for i, p := range et.WindowFunction.Partitions {
			if parts[i], err = ExprFromProto(p, baseSchema, reg); err != nil {
				return nil, err
			}
		}

		sorts := make([]SortField, len(et.WindowFunction.Sorts))
		for i, s := range et.WindowFunction.Sorts {
			if sorts[i], err = SortFieldFromProto(s, baseSchema, reg); err != nil {
				return nil, err
			}
		}

		id, ok := reg.DecodeFunc(et.WindowFunction.FunctionReference)
		if !ok {
			return nil, substraitgo.ErrNotFound
		}
		decl, ok := reg.LookupWindowFunction(et.WindowFunction.FunctionReference)
		if !ok {
			fn, err := NewCustomWindowFunc(reg, extensions.NewWindowFuncVariant(id), types.TypeFromProto(et.WindowFunction.OutputType),
				et.WindowFunction.Options, et.WindowFunction.Invocation, et.WindowFunction.Phase, args...)
			if err != nil {
				return nil, err
			}

			fn.Partitions = parts
			fn.Sorts = sorts
			fn.LowerBound = BoundFromProto(et.WindowFunction.LowerBound)
			fn.BoundsType = et.WindowFunction.BoundsType
			fn.UpperBound = BoundFromProto(et.WindowFunction.UpperBound)
			return fn, nil
		}

		return &WindowFunction{
			funcRef:     et.WindowFunction.FunctionReference,
			declaration: decl,
			args:        args,
			options:     et.WindowFunction.Options,
			outputType:  types.TypeFromProto(et.WindowFunction.OutputType),
			phase:       et.WindowFunction.Phase,
			invocation:  et.WindowFunction.Invocation,
			Partitions:  parts,
			Sorts:       sorts,
			BoundsType:  et.WindowFunction.BoundsType,
			LowerBound:  BoundFromProto(et.WindowFunction.LowerBound),
			UpperBound:  BoundFromProto(et.WindowFunction.UpperBound),
		}, nil
	case *proto.Expression_IfThen_:
		elseExpr, err := ExprFromProto(et.IfThen.Else, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		ifs := make([]IfThenPair, len(et.IfThen.Ifs))
		for i, clause := range et.IfThen.Ifs {
			ifs[i].If, err = ExprFromProto(clause.If, baseSchema, reg)
			if err != nil {
				return nil, err
			}

			ifs[i].Then, err = ExprFromProto(clause.Then, baseSchema, reg)
			if err != nil {
				return nil, err
			}
		}

		return &IfThen{
			ifs:        ifs,
			elseClause: elseExpr,
		}, nil
	case *proto.Expression_SwitchExpression_:
		matched, err := ExprFromProto(et.SwitchExpression.Match, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		elseExpr, err := ExprFromProto(et.SwitchExpression.Else, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		ifs := make([]struct {
			If   Literal
			Then Expression
		}, len(et.SwitchExpression.Ifs))
		for i, clause := range et.SwitchExpression.Ifs {
			ifs[i].If = LiteralFromProto(clause.If)
			ifs[i].Then, err = ExprFromProto(clause.Then, baseSchema, reg)
			if err != nil {
				return nil, err
			}
		}

		return &SwitchExpr{
			match:      matched,
			ifs:        ifs,
			elseClause: elseExpr,
		}, nil
	case *proto.Expression_SingularOrList_:
		val, err := ExprFromProto(et.SingularOrList.Value, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		opts := make([]Expression, len(et.SingularOrList.Options))
		for i, o := range et.SingularOrList.Options {
			opts[i], err = ExprFromProto(o, baseSchema, reg)
			if err != nil {
				return nil, err
			}
		}

		return &SingularOrList{
			Value:   val,
			Options: opts,
		}, nil
	case *proto.Expression_MultiOrList_:
		var err error
		val := make([]Expression, len(et.MultiOrList.Value))
		for i, v := range et.MultiOrList.Value {
			val[i], err = ExprFromProto(v, baseSchema, reg)
			if err != nil {
				return nil, err
			}
		}

		options := make([][]Expression, len(et.MultiOrList.Options))
		for i, opts := range et.MultiOrList.Options {
			options[i] = make([]Expression, len(opts.Fields))
			for j, o := range opts.Fields {
				options[i][j], err = ExprFromProto(o, baseSchema, reg)
				if err != nil {
					return nil, err
				}
			}
		}

		return &MultiOrList{
			Value:   val,
			Options: options,
		}, nil
	case *proto.Expression_Cast_:
		input, err := ExprFromProto(et.Cast.Input, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		return &Cast{
			Type:            types.TypeFromProto(et.Cast.Type),
			Input:           input,
			FailureBehavior: et.Cast.FailureBehavior,
		}, nil
	case *proto.Expression_Nested_:
		var err error
		nullable, typevar := et.Nested.Nullable, et.Nested.TypeVariationReference

		switch n := et.Nested.NestedType.(type) {
		case *proto.Expression_Nested_Map_:
			if len(n.Map.KeyValues) == 0 {
				return nil, fmt.Errorf("%w: use an empty map literal instead of NestedExpr map to preserve type info",
					substraitgo.ErrInvalidExpr)
			}

			keyValues := make([]struct{ Key, Value Expression }, len(n.Map.KeyValues))
			for i, kv := range n.Map.KeyValues {
				keyValues[i].Key, err = ExprFromProto(kv.Key, baseSchema, reg)
				if err != nil {
					return nil, err
				}

				keyValues[i].Value, err = ExprFromProto(kv.Value, baseSchema, reg)
				if err != nil {
					return nil, err
				}
			}

			return &MapExpr{
				Nullable:         nullable,
				TypeVariationRef: typevar,
				KeyValues:        keyValues,
			}, nil
		case *proto.Expression_Nested_Struct_:
			fields := make([]Expression, len(n.Struct.Fields))
			for i, f := range n.Struct.Fields {
				fields[i], err = ExprFromProto(f, baseSchema, reg)
				if err != nil {
					return nil, err
				}
			}

			return &StructExpr{
				Nullable:         nullable,
				TypeVariationRef: typevar,
				Fields:           fields,
			}, nil
		case *proto.Expression_Nested_List_:
			if len(n.List.Values) == 0 {
				return nil, fmt.Errorf("%w: use an empty list literal to preserve type info instead of nested expression",
					substraitgo.ErrInvalidExpr)
			}

			values := make([]Expression, len(n.List.Values))
			for i, v := range n.List.Values {
				values[i], err = ExprFromProto(v, baseSchema, reg)
				if err != nil {
					return nil, err
				}
			}

			return &ListExpr{
				Nullable:         nullable,
				TypeVariationRef: typevar,
				Values:           values,
			}, nil
		default:
			return nil, fmt.Errorf("%w: nested expression: %s",
				substraitgo.ErrInvalidExpr, n)
		}
	case *proto.Expression_DynamicParameter:
		if et.DynamicParameter == nil {
			return nil, fmt.Errorf("%w: dynamic parameter is nil", substraitgo.ErrInvalidExpr)
		}
		return &DynamicParameter{
			OutputType:         types.TypeFromProto(et.DynamicParameter.Type),
			ParameterReference: et.DynamicParameter.ParameterReference,
		}, nil
	case *proto.Expression_Enum_:
		return nil, fmt.Errorf("%w: deprecated", substraitgo.ErrNotImplemented)
	case *proto.Expression_Subquery_:
		if reg.subqueryConverter == nil {
			return nil, fmt.Errorf("%w: subquery expressions require a subquery converter to be configured", substraitgo.ErrNotImplemented)
		}
		return reg.SubqueryFromProto(et.Subquery, baseSchema, reg)
	case *proto.Expression_Lambda_:
		if et.Lambda.Parameters == nil {
			return nil, fmt.Errorf("%w: lambda parameters cannot be nil", substraitgo.ErrInvalidExpr)
		}
		if et.Lambda.Body == nil {
			return nil, fmt.Errorf("%w: lambda body cannot be nil", substraitgo.ErrInvalidExpr)
		}

		paramTypes := make([]types.Type, len(et.Lambda.Parameters.Types))
		for i, pt := range et.Lambda.Parameters.Types {
			paramTypes[i] = types.TypeFromProto(pt)
		}
		params := &types.StructType{
			Types:            paramTypes,
			TypeVariationRef: et.Lambda.Parameters.TypeVariationReference,
			Nullability:      et.Lambda.Parameters.Nullability,
		}

		if params.Nullability != types.NullabilityRequired {
			return nil, fmt.Errorf("%w: lambda parameters struct must have NULLABILITY_REQUIRED", substraitgo.ErrInvalidExpr)
		}

		body, err := ExprFromProto(et.Lambda.Body, baseSchema, reg)
		if err != nil {
			return nil, err
		}

		// TODO (#189): add validation and type resolution for lambda parameter references
		return &Lambda{Parameters: params, Body: body}, nil
	}
	return nil, fmt.Errorf("%w: ExprFromProto: %s", substraitgo.ErrNotImplemented, e)
}

type VisitFunc func(Expression) Expression

// Expression can be one of many different things as a generalized
// expression. It could be:
//
//   - A literal
//   - A Field Reference Selection
//   - A Scalar Function expression
//   - A Window Function expression
//   - An If-Then statement
//   - A Switch Expression
//   - A Singular Or List
//   - A Multiple Or List
//   - A Cast expression
//   - A Subquery
//   - A Nested expression
//   - A Dynamic Parameter
type Expression interface {
	// an Expression can also be a function argument
	types.FuncArg
	// an expression can also be the root of a reference
	RootRefType

	IsScalar() bool
	// GetType returns the output type of this expression
	GetType() types.Type
	// ToProto converts this Expression and its arguments
	// to the equivalent Protobuf objects.
	ToProto() *proto.Expression
	// Equals returns true if this expression and all of its
	// arguments and their children etc. are equal to the passed
	// in Expression.
	Equals(Expression) bool
	// Visit invokes the passed visit function for each child of the
	// expression. The visit function can return its input expression
	// as-is with no changes, or it can construct and return a
	// replacement expression. If any children have been replaced, Visit
	// will construct and return a new instance of this expression using
	// the new children. Callers can use the Visit method to traverse
	// and potentially rewrite the expression tree, in either pre or post
	// order. Here is a pre-order example:
	//
	//   func preOrderVisit(e Expression) Expression {
	//     // Replace some scalar function, leave everything else
	//     // as-is. This check is before the call to Visit, so
	//     // it's a pre-order traversal
	//     if f, ok := e.(*ScalarFunction); ok {
	//       return &ScalarFunction{
	//         ID: ExtID{URN: "some other urn", Name: "some other func"},
	//         Args: f.Args,
	//         Options: f.Options,
	//         OutputType: f.OutputType,
	//       }
	//     }
	//     return e.Visit(preOrderVisit)
	//   }
	//   newExpr := preOrderVisit(oldExpr)
	Visit(VisitFunc) Expression
}

type IfThenPair struct {
	If   Expression
	Then Expression
}

type IfThen struct {
	ifs        []IfThenPair
	elseClause Expression
}

// NewIfThen constructs a new IfThen expression, verifying it is valid.
//
// elseClause should not be nil as there is a required default via the else.
// The constructed expression would be interpreted as this pseudocode:
//
//	if <firstIf.If> then <firstIf.Then>
//	foreach e in elsifs:
//	 if <e.If> then <e.Then>
//	endforeach
//	else <elseClause>
//
// If elseClause is nil or all of the result expressions do not have the
// same result type, an error will be returned.
func NewIfThen(firstIf IfThenPair, elseClause Expression, elsifs ...IfThenPair) (*IfThen, error) {
	if elseClause == nil {
		return nil, fmt.Errorf("%w: must provide an else expression for IfThen", substraitgo.ErrInvalidExpr)
	}

	outputType := elseClause.GetType()
	if firstIf.If == nil {
		return nil, fmt.Errorf("%w: cannot have nil If expression", substraitgo.ErrInvalidExpr)
	}

	if firstIf.Then == nil || !outputType.Equals(firstIf.Then.GetType()) {
		return nil, fmt.Errorf("%w: type mismatch in If creation, expected %s",
			substraitgo.ErrInvalidExpr, outputType)
	}

	for _, ifthen := range elsifs {
		switch {
		case ifthen.If == nil:
			return nil, fmt.Errorf("%w: cannot have nil 'if' expression", substraitgo.ErrInvalidExpr)
		case ifthen.Then == nil:
			return nil, fmt.Errorf("%w: cannot have nil 'then' expression", substraitgo.ErrInvalidExpr)
		case !outputType.Equals(ifthen.Then.GetType()):
			return nil, fmt.Errorf("%w: type mismatch in IfThen expression, expected %s and got %s",
				substraitgo.ErrInvalidExpr, outputType, ifthen.Then.GetType())
		}
	}

	return &IfThen{
		ifs:        append([]IfThenPair{firstIf}, elsifs...),
		elseClause: elseClause,
	}, nil
}

// NIfs returns the number of If/then pairs are in this expression
// before the else clause. It should always be at least 1
func (ex *IfThen) NIfs() int { return len(ex.ifs) }

// IfPair returns the IfThenPair for the given index. It is not bounds-checked
func (ex *IfThen) IfPair(i int) IfThenPair { return ex.ifs[i] }
func (ex *IfThen) Else() Expression        { return ex.elseClause }

func (ex *IfThen) String() string {
	var b strings.Builder
	b.WriteString("<IfThen>(")
	for i, clause := range ex.ifs {
		if i != 0 {
			b.WriteString(": ")
		}
		b.WriteString("(" + clause.If.String() + ") ?")
		b.WriteString(clause.Then.String())
	}
	b.WriteString(")<Else>(")
	if ex.elseClause != nil {
		b.WriteString(ex.elseClause.String())
	}
	b.WriteString(")")
	return b.String()
}

func (ex *IfThen) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}

func (ex *IfThen) isRootRef() {}

func (ex *IfThen) IsScalar() bool {
	for _, clause := range ex.ifs {
		if clause.If != nil && !clause.If.IsScalar() {
			return false
		}
		if clause.Then != nil && !clause.Then.IsScalar() {
			return false
		}
	}
	return ex.elseClause == nil || ex.elseClause.IsScalar()
}

func (ex *IfThen) GetType() types.Type {
	return ex.elseClause.GetType()
}

func (ex *IfThen) ToProto() *proto.Expression {
	ifthenClauses := make([]*proto.Expression_IfThen_IfClause, len(ex.ifs))
	for i, c := range ex.ifs {
		ifthenClauses[i] = &proto.Expression_IfThen_IfClause{
			If:   c.If.ToProto(),
			Then: c.Then.ToProto(),
		}
	}

	var elseClause *proto.Expression
	if ex.elseClause != nil {
		elseClause = ex.elseClause.ToProto()
	}
	return &proto.Expression{
		RexType: &proto.Expression_IfThen_{
			IfThen: &proto.Expression_IfThen{
				Ifs:  ifthenClauses,
				Else: elseClause,
			},
		},
	}
}

func (ex *IfThen) Equals(other Expression) bool {
	rhs, ok := other.(*IfThen)
	if !ok {
		return false
	}

	switch {
	case ex.elseClause != nil && !ex.elseClause.Equals(rhs.elseClause):
		return false
	case ex.elseClause != nil && rhs.elseClause == nil:
		return false
	}

	return slices.EqualFunc(ex.ifs, rhs.ifs,
		func(l, r IfThenPair) bool {
			return l.If.Equals(r.If) && l.Then.Equals(r.Then)
		})
}

func (ex *IfThen) Visit(visit VisitFunc) Expression {
	var out *IfThen

	for i, clause := range ex.ifs {
		afterIf := visit(clause.If)
		afterThen := visit(clause.Then)

		if out == nil && (afterIf != clause.If || afterThen != clause.Then) {
			out = &IfThen{ifs: slices.Clone(ex.ifs)}
		}

		if out != nil {
			out.ifs[i].If = afterIf
			out.ifs[i].Then = afterThen
		}
	}

	afterElse := visit(ex.elseClause)
	if out == nil {
		if afterElse == ex.elseClause {
			return ex
		}

		out = &IfThen{ifs: slices.Clone(ex.ifs)}
	}

	out.elseClause = afterElse
	return out
}

type Cast struct {
	Type            types.Type
	Input           Expression
	FailureBehavior types.CastFailBehavior
}

func (ex *Cast) String() string {
	return fmt.Sprintf("cast(%s AS %s, fail: %s)",
		ex.Input, ex.Type, ex.FailureBehavior)
}

func (ex *Cast) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{Value: ex.ToProto()},
	}
}

func (ex *Cast) isRootRef() {}

func (ex *Cast) IsScalar() bool {
	return ex.Input.IsScalar()
}

func (ex *Cast) GetType() types.Type {
	return ex.Type
}

func (ex *Cast) ToProto() *proto.Expression {
	return &proto.Expression{
		RexType: &proto.Expression_Cast_{
			Cast: &proto.Expression_Cast{
				Type:            types.TypeToProto(ex.Type),
				Input:           ex.Input.ToProto(),
				FailureBehavior: ex.FailureBehavior,
			},
		},
	}
}

func (ex *Cast) Equals(other Expression) bool {
	rhs, ok := other.(*Cast)
	if !ok {
		return false
	}

	return ex.Type.Equals(rhs.Type) && ex.Input.Equals(rhs.Input) &&
		ex.FailureBehavior == rhs.FailureBehavior
}

func (ex *Cast) Visit(visit VisitFunc) Expression {
	newInput := visit(ex.Input)
	if newInput == ex.Input {
		return ex
	}
	newCast := *ex
	newCast.Input = newInput
	return &newCast
}

type DynamicParameter struct {
	OutputType         types.Type
	ParameterReference uint32
}

func (dp *DynamicParameter) String() string {
	return fmt.Sprintf("$%d:%s", dp.ParameterReference, dp.OutputType)
}

func (dp *DynamicParameter) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: dp.ToProto(),
		},
	}
}

func (dp *DynamicParameter) isRootRef() {}

func (dp *DynamicParameter) IsScalar() bool { return true }

func (dp *DynamicParameter) GetType() types.Type { return dp.OutputType }

func (dp *DynamicParameter) ToProto() *proto.Expression {
	return &proto.Expression{
		RexType: &proto.Expression_DynamicParameter{
			DynamicParameter: &proto.DynamicParameter{
				Type:               types.TypeToProto(dp.OutputType),
				ParameterReference: dp.ParameterReference,
			},
		},
	}
}

func (dp *DynamicParameter) Equals(other Expression) bool {
	rhs, ok := other.(*DynamicParameter)
	if !ok {
		return false
	}

	return dp.ParameterReference == rhs.ParameterReference &&
		dp.OutputType.Equals(rhs.OutputType)
}

func (dp *DynamicParameter) Visit(visit VisitFunc) Expression {
	return dp
}

type SwitchExpr struct {
	match Expression
	ifs   []struct {
		If   Literal
		Then Expression
	}
	elseClause Expression
}

// NewSwitch constructs a switch statement. Will return an error
// in the following cases:
//   - match is nil
//   - len(switchCases) < 1
//   - the type of the If literal in each switchCase != match.GetType()
//   - the GetType for each "Then" of switch cases aren't the same
//   - elseClause.GetType() != switchCases.Then.GetType() if elseClause != nil
//
// elseClause is allowed to be nil if there is no default case.
func NewSwitch(match Expression, elseClause Expression, switchCases ...struct {
	If   Literal
	Then Expression
}) (*SwitchExpr, error) {
	if match == nil {
		return nil, fmt.Errorf("%w: cannot construct Switch with nil match", substraitgo.ErrInvalidExpr)
	}

	// we don't care about the nullability for this comparison
	matchType := match.GetType().WithNullability(types.NullabilityUnspecified)
	if len(switchCases) == 0 {
		return nil, fmt.Errorf("%w: must have at least one switch case", substraitgo.ErrInvalidExpr)
	}

	// get the output type with the nullability unspecified as the nullability
	// of the output type will be determined based on if any of the returns
	// could return null or if none of them can. We don't care about the
	// nullability for the purposes of validity comparison here.
	var outputType types.Type
	if elseClause != nil {
		outputType = elseClause.GetType().WithNullability(types.NullabilityUnspecified)
	} else if switchCases[0].Then != nil {
		outputType = switchCases[0].Then.GetType().WithNullability(types.NullabilityUnspecified)
	}

	for _, c := range switchCases {
		switch {
		case c.If == nil:
			return nil, fmt.Errorf("%w: switch case must have a non-nil literal for If", substraitgo.ErrInvalidExpr)
		case c.Then == nil:
			return nil, fmt.Errorf("%w: switch case must have non-nil Then expression", substraitgo.ErrInvalidExpr)
		case !matchType.Equals(c.If.GetType().WithNullability(types.NullabilityUnspecified)):
			return nil, fmt.Errorf("%w: switch case If literal doesn't match type of match expression. expected %s, got %s",
				substraitgo.ErrInvalidExpr, matchType, c.If.GetType())
		case !outputType.Equals(c.Then.GetType().WithNullability(types.NullabilityUnspecified)):
			return nil, fmt.Errorf("%w: switch case result type doesn't match expected output type. expected %s, got %s",
				substraitgo.ErrInvalidExpr, outputType, c.Then.GetType())
		}
	}

	return &SwitchExpr{
		match:      match,
		ifs:        switchCases,
		elseClause: elseClause,
	}, nil
}

func (ex *SwitchExpr) MatchExpr() Expression { return ex.match }

// NCases returns the number of case statements in this switch, not including
// the existences of a default else clause
func (ex *SwitchExpr) NCases() int { return len(ex.ifs) }

// Case returns the pair of Literal and result Expression for the given
// index. It is not bounds checked.
func (ex *SwitchExpr) Case(i int) struct {
	If   Literal
	Then Expression
} {
	return ex.ifs[i]
}

func (ex *SwitchExpr) Else() Expression { return ex.elseClause }

func (ex *SwitchExpr) String() string {
	var b strings.Builder
	b.WriteString("CASE ")
	b.WriteString(ex.match.String())
	b.WriteString(":")
	for _, c := range ex.ifs {
		b.WriteString("\nWHEN ")
		b.WriteString(c.If.String())
		b.WriteString(" THEN ")
		b.WriteString(c.Then.String())
		b.WriteByte(';')
	}
	if ex.elseClause != nil {
		b.WriteString("Else ")
		b.WriteString(ex.elseClause.String())
	}
	return b.String()
}

func (ex *SwitchExpr) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}

func (ex *SwitchExpr) isRootRef() {}

func (ex *SwitchExpr) IsScalar() bool {
	if !ex.match.IsScalar() {
		return false
	}

	for _, c := range ex.ifs {
		if !c.Then.IsScalar() {
			return false
		}
	}

	if ex.elseClause != nil {
		return ex.elseClause.IsScalar()
	}

	return true
}

func (ex *SwitchExpr) GetType() types.Type {
	var t types.Type
	for _, cond := range ex.ifs {
		t = cond.Then.GetType()
		// check if any of the branches return a nullable type
		// only return a non-nullable if *all* branches return
		// non-nullable
		if t.GetNullability() == types.NullabilityNullable {
			break
		}
	}

	// if there's no else clause, then return a nullable type
	if ex.elseClause == nil {
		return t.WithNullability(types.NullabilityNullable)
	}

	// if any branch returns nullable, then return the nullable type
	// we don't care if the else clause is nullable
	if t.GetNullability() == types.NullabilityNullable {
		return t
	}

	// if no branch returns nullable, just return the else clause type
	return ex.elseClause.GetType()
}

func (ex *SwitchExpr) ToProto() *proto.Expression {
	var elseExpr *proto.Expression
	if ex.elseClause != nil {
		elseExpr = ex.elseClause.ToProto()
	}

	cases := make([]*proto.Expression_SwitchExpression_IfValue, len(ex.ifs))
	for i, c := range ex.ifs {
		cases[i] = &proto.Expression_SwitchExpression_IfValue{
			If:   c.If.ToProtoLiteral(),
			Then: c.Then.ToProto(),
		}
	}

	return &proto.Expression{
		RexType: &proto.Expression_SwitchExpression_{
			SwitchExpression: &proto.Expression_SwitchExpression{
				Match: ex.match.ToProto(),
				Ifs:   cases,
				Else:  elseExpr,
			},
		},
	}
}

func (ex *SwitchExpr) Equals(other Expression) bool {
	rhs, ok := other.(*SwitchExpr)
	if !ok {
		return false
	}

	switch {
	case !ex.match.Equals(rhs.match):
		return false
	case ex.elseClause != nil && !ex.elseClause.Equals(rhs.elseClause):
		return false
	case ex.elseClause == nil && rhs.elseClause != nil:
		return false
	}

	return slices.EqualFunc(ex.ifs, rhs.ifs,
		func(l, r struct {
			If   Literal
			Then Expression
		}) bool {
			return l.If.Equals(r.If) && l.Then.Equals(r.Then)
		})
}

func (ex *SwitchExpr) Visit(visit VisitFunc) Expression {
	var out *SwitchExpr
	if after := visit(ex.match); after != ex.match {
		out = &SwitchExpr{
			match: after,
			ifs: make([]struct {
				If   Literal
				Then Expression
			}, len(ex.ifs)),
			elseClause: ex.elseClause,
		}
	}

	for i, c := range ex.ifs {
		afterIf := visit(c.If)
		afterThen := visit(c.Then)

		if out == nil && (afterIf != c.If || afterThen != c.Then) {
			out = &SwitchExpr{
				match: ex.match,
				ifs: make([]struct {
					If   Literal
					Then Expression
				}, len(ex.ifs)),
				elseClause: ex.elseClause,
			}
			copy(out.ifs, ex.ifs[:i])
		}

		if out != nil {
			out.ifs[i].If = afterIf.(Literal)
			out.ifs[i].Then = afterThen
		}
	}

	afterElse := visit(ex.elseClause)
	if out == nil {
		if afterElse == ex.elseClause {
			return ex
		}

		out = &SwitchExpr{
			match: ex.match,
			ifs:   slices.Clone(ex.ifs),
		}
	}

	out.elseClause = afterElse
	return out
}

func exprEqual(l, r Expression) bool { return l.Equals(r) }

type SingularOrList struct {
	Value   Expression
	Options []Expression
}

func (ex *SingularOrList) String() string {
	var b strings.Builder
	b.WriteString(ex.Value.String())
	b.WriteString(" IN [")
	for i, o := range ex.Options {
		if i != 0 {
			b.WriteString(",")
		}
		b.WriteString(o.String())
	}
	b.WriteString("]")
	return b.String()
}

func (ex *SingularOrList) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}

func (ex *SingularOrList) isRootRef() {}

func (ex *SingularOrList) IsScalar() bool {
	if !ex.Value.IsScalar() {
		return false
	}

	for _, o := range ex.Options {
		if !o.IsScalar() {
			return false
		}
	}

	return true
}

func (ex *SingularOrList) GetType() types.Type {
	return &types.BooleanType{Nullability: types.NullabilityRequired}
}

func (ex *SingularOrList) ToProto() *proto.Expression {
	opts := make([]*proto.Expression, len(ex.Options))
	for i, o := range ex.Options {
		opts[i] = o.ToProto()
	}
	return &proto.Expression{
		RexType: &proto.Expression_SingularOrList_{
			SingularOrList: &proto.Expression_SingularOrList{
				Value:   ex.Value.ToProto(),
				Options: opts,
			},
		},
	}
}

func (ex *SingularOrList) Equals(other Expression) bool {
	rhs, ok := other.(*SingularOrList)
	if !ok {
		return false
	}

	if !ex.Value.Equals(rhs.Value) {
		return false
	}

	return slices.EqualFunc(ex.Options, rhs.Options, exprEqual)
}

func (ex *SingularOrList) Visit(visit VisitFunc) Expression {
	var out *SingularOrList
	if temp := visit(ex.Value); temp != ex.Value {
		out = &SingularOrList{
			Value:   temp,
			Options: make([]Expression, len(ex.Options)),
		}
	}

	for i, o := range ex.Options {
		temp := visit(o)
		if out == nil && temp != o {
			out = &SingularOrList{Value: ex.Value, Options: make([]Expression, len(ex.Options))}
			copy(out.Options, ex.Options[:i])
		}

		if out != nil {
			out.Options[i] = temp
		}
	}

	if out == nil {
		return ex
	}

	return out
}

type MultiOrList struct {
	Value   []Expression
	Options [][]Expression
}

func (ex *MultiOrList) String() string {
	var b strings.Builder
	writeList := func(list []Expression) {
		b.WriteByte('[')
		for i, v := range list {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(v.String())
		}
		b.WriteByte(']')
	}

	writeList(ex.Value)
	b.WriteString(" IN [")
	for i, opts := range ex.Options {
		if i != 0 {
			b.WriteString(", ")
		}
		writeList(opts)
	}
	b.WriteString("]")

	return b.String()
}

func (ex *MultiOrList) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}

func (ex *MultiOrList) isRootRef() {}

func (ex *MultiOrList) IsScalar() bool {
	for _, v := range ex.Value {
		if !v.IsScalar() {
			return false
		}
	}

	for _, opts := range ex.Options {
		for _, o := range opts {
			if !o.IsScalar() {
				return false
			}
		}
	}

	return true
}

func (ex *MultiOrList) GetType() types.Type {
	return &types.BooleanType{Nullability: types.NullabilityRequired}
}

func (ex *MultiOrList) ToProto() *proto.Expression {
	toSlice := func(exprs []Expression) (out []*proto.Expression) {
		out = make([]*proto.Expression, len(exprs))
		for i, e := range exprs {
			out[i] = e.ToProto()
		}
		return
	}

	opts := make([]*proto.Expression_MultiOrList_Record, len(ex.Options))
	for i, o := range ex.Options {
		opts[i] = &proto.Expression_MultiOrList_Record{
			Fields: toSlice(o),
		}
	}

	return &proto.Expression{
		RexType: &proto.Expression_MultiOrList_{
			MultiOrList: &proto.Expression_MultiOrList{
				Value:   toSlice(ex.Value),
				Options: opts,
			},
		},
	}
}

func (ex *MultiOrList) Equals(other Expression) bool {
	rhs, ok := other.(*MultiOrList)
	if !ok {
		return false
	}

	return slices.EqualFunc(ex.Value, rhs.Value, exprEqual) &&
		slices.EqualFunc(ex.Options, rhs.Options,
			func(l, r []Expression) bool {
				return slices.EqualFunc(l, r, exprEqual)
			})
}

func (ex *MultiOrList) Visit(visit VisitFunc) Expression {
	var out *MultiOrList
	for i, v := range ex.Value {
		after := visit(v)

		if out == nil && after != v {
			out = &MultiOrList{
				Value:   make([]Expression, len(ex.Value)),
				Options: make([][]Expression, len(ex.Options)),
			}
			copy(out.Value, ex.Value[:i])
		}

		if out != nil {
			out.Value[i] = after
		}
	}

	for i, opts := range ex.Options {
		if out != nil && len(out.Options[i]) == 0 {
			out.Options[i] = make([]Expression, len(ex.Options[i]))
		}

		for j, o := range opts {
			after := visit(o)

			if out == nil && after != o {
				out = &MultiOrList{
					Value:   slices.Clone(ex.Value),
					Options: make([][]Expression, len(ex.Options)),
				}
				for k := 0; k < i; k++ {
					out.Options[k] = slices.Clone(ex.Options[k])
				}

				for k := 0; k < j; k++ {
					out.Options[i][k] = ex.Options[i][k]
				}
			}

			if out != nil {
				out.Options[i][j] = after
			}
		}
	}

	if out == nil {
		return ex
	}

	return out
}

type NestedExpr interface {
	Expression

	IsNullable() bool
	TypeVariation() uint32
}

type MapExpr struct {
	Nullable         bool
	TypeVariationRef uint32
	KeyValues        []struct{ Key, Value Expression }
}

func (ex *MapExpr) IsNullable() bool      { return ex.Nullable }
func (ex *MapExpr) TypeVariation() uint32 { return ex.TypeVariationRef }

func (ex *MapExpr) String() string {
	var b strings.Builder
	b.WriteString("{")

	for i, kv := range ex.KeyValues {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(kv.Key.String())
		b.WriteString(" => ")
		b.WriteString(kv.Value.String())
	}

	b.WriteString("}")
	if ex.Nullable {
		b.WriteString("?")
	}
	fmt.Fprintf(&b, "(typeref=%d)", ex.TypeVariationRef)
	return b.String()
}

func (ex *MapExpr) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}

func (ex *MapExpr) isRootRef() {}

func (ex *MapExpr) IsScalar() bool {
	for _, kv := range ex.KeyValues {
		if !kv.Key.IsScalar() || !kv.Value.IsScalar() {
			return false
		}
	}

	return true
}

func (ex *MapExpr) GetType() types.Type {
	// there *should* be at least one element in the keyvalues,
	// otherwise an EmptyMap literal should be used in order
	// to ensure type information is not lost
	return &types.MapType{
		Nullability:      getNullability(ex.Nullable),
		TypeVariationRef: ex.TypeVariationRef,
		Key:              ex.KeyValues[0].Key.GetType(),
		Value:            ex.KeyValues[0].Value.GetType(),
	}
}

func (ex *MapExpr) ToProto() *proto.Expression {
	kvs := make([]*proto.Expression_Nested_Map_KeyValue, len(ex.KeyValues))
	for i, kv := range ex.KeyValues {
		kvs[i] = &proto.Expression_Nested_Map_KeyValue{
			Key:   kv.Key.ToProto(),
			Value: kv.Value.ToProto(),
		}
	}
	return &proto.Expression{
		RexType: &proto.Expression_Nested_{
			Nested: &proto.Expression_Nested{
				Nullable:               ex.Nullable,
				TypeVariationReference: ex.TypeVariationRef,
				NestedType: &proto.Expression_Nested_Map_{
					Map: &proto.Expression_Nested_Map{
						KeyValues: kvs,
					},
				},
			},
		},
	}
}

func (ex *MapExpr) Equals(other Expression) bool {
	rhs, ok := other.(*MapExpr)
	if !ok {
		return false
	}

	if ex.Nullable != rhs.Nullable || ex.TypeVariationRef != rhs.TypeVariationRef {
		return false
	}

	return slices.EqualFunc(ex.KeyValues, rhs.KeyValues,
		func(l, r struct{ Key, Value Expression }) bool {
			return l.Key.Equals(r.Key) && l.Value.Equals(r.Value)
		})
}

func (ex *MapExpr) Visit(visit VisitFunc) Expression {
	var out *MapExpr
	for i, kv := range ex.KeyValues {
		afterKey := visit(kv.Key)
		afterValue := visit(kv.Value)

		if out == nil && (afterKey != kv.Key || afterValue != kv.Value) {
			out = &MapExpr{
				Nullable:         ex.Nullable,
				TypeVariationRef: ex.TypeVariationRef,
				KeyValues: make([]struct {
					Key   Expression
					Value Expression
				}, len(ex.KeyValues)),
			}
			copy(out.KeyValues, ex.KeyValues[:i])
		}

		if out != nil {
			out.KeyValues[i].Key = afterKey
			out.KeyValues[i].Value = afterValue
		}
	}

	if out != nil {
		return out
	}

	return ex
}

type StructExpr struct {
	Nullable         bool
	TypeVariationRef uint32
	Fields           []Expression
}

func (ex *StructExpr) IsNullable() bool      { return ex.Nullable }
func (ex *StructExpr) TypeVariation() uint32 { return ex.TypeVariationRef }

func (ex *StructExpr) String() string {
	var b strings.Builder
	b.WriteString("{")
	for i, f := range ex.Fields {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(f.String())
	}
	b.WriteString("}")
	if ex.Nullable {
		b.WriteString("?")
	}
	fmt.Fprintf(&b, "(typeref=%d)", ex.TypeVariationRef)
	return b.String()
}

func (ex *StructExpr) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}
func (ex *StructExpr) isRootRef() {}

func (ex *StructExpr) IsScalar() bool {
	for _, f := range ex.Fields {
		if !f.IsScalar() {
			return false
		}
	}
	return true
}

func (ex *StructExpr) GetType() types.Type {
	typs := make([]types.Type, len(ex.Fields))
	for i, f := range ex.Fields {
		typs[i] = f.GetType()
	}
	return &types.StructType{
		Nullability:      getNullability(ex.Nullable),
		TypeVariationRef: ex.TypeVariationRef,
		Types:            typs,
	}
}

func (ex *StructExpr) ToProto() *proto.Expression {
	fields := make([]*proto.Expression, len(ex.Fields))
	for i, f := range ex.Fields {
		fields[i] = f.ToProto()
	}
	return &proto.Expression{
		RexType: &proto.Expression_Nested_{
			Nested: &proto.Expression_Nested{
				Nullable:               ex.Nullable,
				TypeVariationReference: ex.TypeVariationRef,
				NestedType: &proto.Expression_Nested_Struct_{
					Struct: &proto.Expression_Nested_Struct{
						Fields: fields,
					},
				},
			},
		},
	}
}

func (ex *StructExpr) Equals(other Expression) bool {
	rhs, ok := other.(*StructExpr)
	if !ok {
		return false
	}

	return ex.Nullable == rhs.Nullable &&
		ex.TypeVariationRef == rhs.TypeVariationRef &&
		slices.EqualFunc(ex.Fields, rhs.Fields, exprEqual)
}

func (ex *StructExpr) Visit(visit VisitFunc) Expression {
	var out *StructExpr
	for i, f := range ex.Fields {
		after := visit(f)
		if out == nil && after != f {
			out = &StructExpr{
				Nullable:         ex.Nullable,
				TypeVariationRef: ex.TypeVariationRef,
				Fields:           make([]Expression, len(ex.Fields)),
			}
			copy(out.Fields, ex.Fields[:i])
		}

		if out != nil {
			out.Fields[i] = after
		}
	}

	if out == nil {
		return ex
	}

	return out
}

type ListExpr struct {
	Nullable         bool
	TypeVariationRef uint32
	Values           []Expression
}

func NewListExpr(nullable bool, vals ...Expression) *ListExpr {
	return &ListExpr{
		Nullable: nullable,
		Values:   vals,
	}
}

func (ex *ListExpr) IsNullable() bool      { return ex.Nullable }
func (ex *ListExpr) TypeVariation() uint32 { return ex.TypeVariationRef }

func (ex *ListExpr) String() string {
	var b strings.Builder
	b.WriteString("[")
	for i, v := range ex.Values {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(v.String())
	}
	b.WriteString("]")
	if ex.Nullable {
		b.WriteString("?")
	}
	fmt.Fprintf(&b, "(typeref=%d)", ex.TypeVariationRef)
	return b.String()
}

func (ex *ListExpr) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: ex.ToProto(),
		},
	}
}
func (ex *ListExpr) isRootRef() {}

func (ex *ListExpr) IsScalar() bool {
	for _, v := range ex.Values {
		if !v.IsScalar() {
			return false
		}
	}
	return true
}

func (ex *ListExpr) GetType() types.Type {
	return &types.ListType{
		Nullability:      getNullability(ex.Nullable),
		TypeVariationRef: ex.TypeVariationRef,
		// to specify an empty list, use an emptylist literal
		// otherwise you will be missing type information.
		// thus we should assume there's at least one value
		Type: ex.Values[0].GetType(),
	}
}

func (ex *ListExpr) ToProto() *proto.Expression {
	vals := make([]*proto.Expression, len(ex.Values))
	for i, v := range ex.Values {
		vals[i] = v.ToProto()
	}
	return &proto.Expression{
		RexType: &proto.Expression_Nested_{
			Nested: &proto.Expression_Nested{
				Nullable:               ex.Nullable,
				TypeVariationReference: ex.TypeVariationRef,
				NestedType: &proto.Expression_Nested_List_{
					List: &proto.Expression_Nested_List{
						Values: vals,
					},
				},
			},
		},
	}
}

func (ex *ListExpr) Equals(other Expression) bool {
	rhs, ok := other.(*ListExpr)
	if !ok {
		return false
	}

	if ex.Nullable != rhs.Nullable || ex.TypeVariationRef != rhs.TypeVariationRef {
		return false
	}

	return slices.EqualFunc(ex.Values, rhs.Values, exprEqual)
}

func (ex *ListExpr) Visit(visit VisitFunc) Expression {
	var out *ListExpr
	for i, v := range ex.Values {
		after := visit(v)

		if out == nil && after != v {
			out = &ListExpr{
				Nullable:         ex.Nullable,
				TypeVariationRef: ex.TypeVariationRef,
				Values:           make([]Expression, len(ex.Values)),
			}
			copy(out.Values, ex.Values[:i])
		}

		if out != nil {
			out.Values[i] = after
		}
	}

	if out != nil {
		return out
	}

	return ex
}

type ExpressionReference struct {
	OutputNames []string
	// only one of these should ever be set at a time
	// this is ensured by the explicit setters rather than
	// making them public
	expr    Expression
	measure *AggregateFunction
}

func NewExpressionReference(names []string, ex Expression) ExpressionReference {
	return ExpressionReference{OutputNames: names, expr: ex}
}

func NewMeasureReference(names []string, measure *AggregateFunction) ExpressionReference {
	return ExpressionReference{OutputNames: names, measure: measure}
}

func (er *ExpressionReference) ToProto() *proto.ExpressionReference {
	out := &proto.ExpressionReference{OutputNames: er.OutputNames}
	switch {
	case er.expr != nil:
		out.ExprType = &proto.ExpressionReference_Expression{
			Expression: er.expr.ToProto(),
		}
	case er.measure != nil:
		out.ExprType = &proto.ExpressionReference_Measure{
			Measure: er.measure.ToProto(),
		}
	}

	return out
}

func (er *ExpressionReference) SetExpr(ex Expression) {
	er.expr = ex
	er.measure = nil
}

func (er *ExpressionReference) SetMeasure(m *AggregateFunction) {
	er.expr = nil
	er.measure = m
}

func (er *ExpressionReference) GetExpr() Expression            { return er.expr }
func (er *ExpressionReference) GetMeasure() *AggregateFunction { return er.measure }

type Extended struct {
	Version          *types.Version
	Extensions       extensions.Set
	ReferredExpr     []ExpressionReference
	BaseSchema       types.NamedStruct
	AdvancedExts     *extensions.AdvancedExtension
	ExpectedTypeURLs []string

	reg ExtensionRegistry
}

func ExtendedFromProto(ex *proto.ExtendedExpression, c *extensions.Collection) (*Extended, error) {
	extSet, err := extensions.GetExtensionSet(ex, c)
	if err != nil {
		return nil, err
	}
	var (
		base = types.NewNamedStructFromProto(ex.BaseSchema)
		reg  = NewExtensionRegistry(extSet, c)
		refs = make([]ExpressionReference, len(ex.ReferredExpr))
	)

	for i, r := range ex.ReferredExpr {
		refs[i].OutputNames = r.OutputNames
		switch et := r.ExprType.(type) {
		case *proto.ExpressionReference_Expression:
			thisType := types.NewRecordTypeFromStruct(base.Struct)
			expr, err := ExprFromProto(et.Expression, thisType, reg)
			if err != nil {
				return nil, err
			}
			refs[i].SetExpr(expr)
		case *proto.ExpressionReference_Measure:
			thisType := types.NewRecordTypeFromStruct(base.Struct)
			agg, err := NewAggregateFunctionFromProto(et.Measure, thisType, reg)
			if err != nil {
				return nil, err
			}
			refs[i].SetMeasure(agg)
		}
	}

	return &Extended{
		Version:          ex.Version,
		Extensions:       extSet,
		ReferredExpr:     refs,
		BaseSchema:       base,
		AdvancedExts:     ex.AdvancedExtensions,
		ExpectedTypeURLs: ex.ExpectedTypeUrls,
		reg:              reg,
	}, nil
}

func (ex *Extended) ToProto() *proto.ExtendedExpression {
	urns, decls := ex.reg.ExtensionsToProto()
	refs := make([]*proto.ExpressionReference, len(ex.ReferredExpr))
	for i, ref := range ex.ReferredExpr {
		refs[i] = ref.ToProto()
	}

	return &proto.ExtendedExpression{
		Version:            ex.Version,
		ExtensionUrns:      urns,
		Extensions:         decls,
		BaseSchema:         ex.BaseSchema.ToProto(),
		AdvancedExtensions: ex.AdvancedExts,
		ExpectedTypeUrls:   ex.ExpectedTypeURLs,
		ReferredExpr:       refs,
	}
}

func (s VirtualTableExpressionValue) ToProto() *proto.Expression_Nested_Struct {
	fields := make([]*proto.Expression, len(s))
	for i, f := range s {
		fields[i] = f.ToProto()
	}
	return &proto.Expression_Nested_Struct{
		Fields: fields,
	}
}
