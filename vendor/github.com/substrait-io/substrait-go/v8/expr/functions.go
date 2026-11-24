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
	pb "google.golang.org/protobuf/proto"
)

func FuncArgsEqual(a, b types.FuncArg) bool {
	if a == b {
		return true
	}

	switch lhs := a.(type) {
	case Expression:
		rhs, ok := b.(Expression)
		if !ok {
			return false
		}

		return lhs.Equals(rhs)
	case types.Type:
		rhs, ok := b.(types.Type)
		if !ok {
			return false
		}

		return lhs.Equals(rhs)
	case types.Enum:
		rhs, ok := b.(types.Enum)
		if !ok {
			return false
		}

		return lhs == rhs
	}

	return false
}

type (
	SortField struct {
		Expr Expression
		Kind types.SortKind
	}

	Bound interface {
		fmt.Stringer
		ToProto() *proto.Expression_WindowFunction_Bound
	}

	PrecedingBound int64
	FollowingBound int64
	CurrentRow     struct{}
	Unbounded      struct{}
)

func (s *SortField) ToProto() *proto.SortField {
	ret := &proto.SortField{Expr: s.Expr.ToProto()}
	switch k := s.Kind.(type) {
	case types.SortDirection:
		ret.SortKind = &proto.SortField_Direction{
			Direction: proto.SortField_SortDirection(k)}
	case types.FunctionRef:
		ret.SortKind = &proto.SortField_ComparisonFunctionReference{
			ComparisonFunctionReference: uint32(k)}
	}

	return ret
}

func SortFieldFromProto(
	f *proto.SortField, baseSchema *types.RecordType, reg ExtensionRegistry,
) (sf SortField, err error) {
	sf.Expr, err = ExprFromProto(f.Expr, baseSchema, reg)
	if err != nil {
		return
	}

	switch k := f.SortKind.(type) {
	case *proto.SortField_Direction:
		sf.Kind = types.SortDirection(k.Direction)
	case *proto.SortField_ComparisonFunctionReference:
		sf.Kind = types.FunctionRef(k.ComparisonFunctionReference)
	default:
		err = substraitgo.ErrNotImplemented
	}
	return
}

func (fb PrecedingBound) ToProto() *proto.Expression_WindowFunction_Bound {
	return &proto.Expression_WindowFunction_Bound{
		Kind: &proto.Expression_WindowFunction_Bound_Preceding_{
			Preceding: &proto.Expression_WindowFunction_Bound_Preceding{Offset: int64(fb)},
		},
	}
}

func (fb PrecedingBound) String() string {
	return fmt.Sprintf("%d PRECEDING", fb)
}

func (fb FollowingBound) ToProto() *proto.Expression_WindowFunction_Bound {
	return &proto.Expression_WindowFunction_Bound{
		Kind: &proto.Expression_WindowFunction_Bound_Following_{
			Following: &proto.Expression_WindowFunction_Bound_Following{Offset: int64(fb)},
		},
	}
}

func (fb FollowingBound) String() string {
	return fmt.Sprintf("%d FOLLOWING", fb)
}

func (CurrentRow) ToProto() *proto.Expression_WindowFunction_Bound {
	return &proto.Expression_WindowFunction_Bound{
		Kind: &proto.Expression_WindowFunction_Bound_CurrentRow_{
			CurrentRow: &proto.Expression_WindowFunction_Bound_CurrentRow{},
		},
	}
}

func (CurrentRow) String() string {
	return "CURRENT ROW"
}

func (Unbounded) ToProto() *proto.Expression_WindowFunction_Bound {
	return &proto.Expression_WindowFunction_Bound{
		Kind: &proto.Expression_WindowFunction_Bound_Unbounded_{
			Unbounded: &proto.Expression_WindowFunction_Bound_Unbounded{},
		}}
}

func (Unbounded) String() string {
	return "UNBOUNDED"
}

func BoundFromProto(b *proto.Expression_WindowFunction_Bound) Bound {
	if b == nil {
		return nil
	}

	switch t := b.Kind.(type) {
	case *proto.Expression_WindowFunction_Bound_Preceding_:
		return PrecedingBound(t.Preceding.Offset)
	case *proto.Expression_WindowFunction_Bound_CurrentRow_:
		return CurrentRow{}
	case *proto.Expression_WindowFunction_Bound_Following_:
		return FollowingBound(t.Following.Offset)
	case *proto.Expression_WindowFunction_Bound_Unbounded_:
		return Unbounded{}
	}

	// bound is optional
	return nil
}

type FunctionInvocation interface {
	CompoundName() string
	ID() extensions.ID
	GetOptions() []*types.FunctionOption
	GetArgTypes() []types.Type
}

type ScalarFunction struct {
	funcRef     uint32
	declaration *extensions.ScalarFunctionVariant

	args       []types.FuncArg
	options    []*types.FunctionOption
	outputType types.Type
}

// NewCustomScalarFunc doesn't validate that the ID can be found already
// in the registry with LookupScalarFunction and will construct the function
// as provided as long as the outputType is non-nil. In this case, the registry
// is only used to provide an anchor / function reference that can be used
// when serializing this expression to Protobuf. Guaranteeing that you have
// a valid expression returned.
//
// Currently an error is only returned if outputType == nil
func NewCustomScalarFunc(
	reg ExtensionRegistry, v *extensions.ScalarFunctionVariant, outputType types.Type,
	opts []*types.FunctionOption, args ...types.FuncArg,
) (*ScalarFunction, error) {
	if outputType == nil {
		return nil, fmt.Errorf("%w: must provide non-nil output type", substraitgo.ErrInvalidType)
	}

	return &ScalarFunction{
		funcRef:     reg.GetFuncAnchor(v.ID()),
		declaration: v,
		options:     opts,
		args:        args,
		outputType:  outputType,
	}, nil
}

type variant interface {
	*extensions.ScalarFunctionVariant | *extensions.AggregateFunctionVariant | *extensions.WindowFunctionVariant
	ResolveType([]types.Type, extensions.Set) (types.Type, error)
}

func resolveVariant[T variant](
	id extensions.ID, reg ExtensionRegistry, getter func(extensions.ID) (T, bool),
	args []types.FuncArg,
) (T, types.Type, error) {
	argTypes := make([]types.Type, 0, len(args))
	for _, arg := range args {
		switch a := arg.(type) {
		case types.Enum:
			argTypes = append(argTypes, types.CommonEnumType)
		case Expression:
			argTypes = append(argTypes, a.GetType())
		}
	}

	decl, found := getter(id)
	if !found {
		if strings.IndexByte(id.Name, ':') == -1 {
			sigs := make([]string, len(argTypes))
			for i, t := range argTypes {
				if t == types.CommonEnumType {
					// enum value
					sigs[i] = extensions.EnumTypeString
				} else if ud, ok := t.(*types.UserDefinedType); ok {
					id, found := reg.DecodeType(ud.TypeReference)
					if !found {
						return nil, nil, fmt.Errorf("%w: could not find type for reference %d",
							substraitgo.ErrNotFound, ud.TypeReference)
					}
					sigs[i] = "u!" + id.Name
				} else {
					sigs[i] = t.ShortString()
				}
			}
			id.Name += ":" + strings.Join(sigs, "_")
			if decl, found = getter(id); !found {
				return nil, nil, fmt.Errorf("%w: could not find matching function for id: %s",
					substraitgo.ErrNotFound, id)
			}
		} else {
			return nil, nil, fmt.Errorf("%w: could not find matching function for id: %s",
				substraitgo.ErrNotFound, id)
		}
	}

	outType, err := decl.ResolveType(argTypes, reg.Set)
	if err != nil {
		return nil, nil, err
	}

	return decl, outType, nil
}

// NewScalarFunc validates that the specified ID can be found in the
// registry via LookupScalarFunction and retrieves a function anchor to use.
//
// If the name in the ID is not currently a compound signature and cannot
// be found in the registry, we'll attempt to construct the compound signature
// based on the types of the provided arguments and look it up that way.
// If both attempts fail to lookup the function, a substraitgo.ErrNotFound
// will be returned.
//
// Currently the options are not validated against the function declaration
// but the number of arguments and their types will be validated in order to
// resolve the output type.
func NewScalarFunc(
	reg ExtensionRegistry, id extensions.ID, opts []*types.FunctionOption, args ...types.FuncArg,
) (*ScalarFunction, error) {
	decl, outType, err := resolveVariant(id, reg, reg.c.GetScalarFunc, args)
	if err != nil {
		return nil, err
	}

	// We use the fully qualified ID for resolving an anchor, to make sure we
	// are using the correct compound name
	ref := reg.GetFuncAnchor(decl.ID())

	return &ScalarFunction{
		funcRef:     ref,
		declaration: decl,
		outputType:  outType,
		options:     opts,
		args:        args,
	}, nil
}

func (s *ScalarFunction) Name() string                           { return s.declaration.Name() }
func (s *ScalarFunction) CompoundName() string                   { return s.declaration.CompoundName() }
func (s *ScalarFunction) ID() extensions.ID                      { return s.declaration.ID() }
func (s *ScalarFunction) Variadic() *extensions.VariadicBehavior { return s.declaration.Variadic() }
func (s *ScalarFunction) SessionDependant() bool                 { return s.declaration.SessionDependent() }
func (s *ScalarFunction) Deterministic() bool                    { return s.declaration.Deterministic() }
func (s *ScalarFunction) NArgs() int                             { return len(s.args) }
func (s *ScalarFunction) Arg(i int) types.FuncArg                { return s.args[i] }
func (s *ScalarFunction) FuncRef() uint32                        { return s.funcRef }
func (s *ScalarFunction) IsScalar() bool {
	for _, arg := range s.args {
		if ex, ok := arg.(Expression); ok {
			if !ex.IsScalar() {
				return false
			}
		}
	}
	return true
}

func (*ScalarFunction) isRootRef() {}

func (s *ScalarFunction) String() string {
	var b strings.Builder

	b.WriteString(s.Name())
	b.WriteByte('(')

	for i, arg := range s.args {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.String())
	}

	if len(s.options) > 0 {
		b.WriteString(", {")
		for i, o := range s.options {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(o.Name)
			b.WriteString(": [")
			b.WriteString(strings.Join(o.Preference, ","))
			b.WriteString("]")
		}
		b.WriteString("}")
	}

	b.WriteString(") => ")
	if s.outputType != nil {
		b.WriteString(s.outputType.String())
	} else {
		b.WriteString("?")
	}

	return b.String()
}

func (s *ScalarFunction) GetOption(name string) []string {
	for _, o := range s.options {
		if name == o.Name {
			return o.GetPreference()
		}
	}
	return nil
}

func (s *ScalarFunction) GetOptions() []*types.FunctionOption { return s.options }

func getArgTypes(args []types.FuncArg) []types.Type {
	argTypes := make([]types.Type, len(args))
	for i, arg := range args {
		switch a := arg.(type) {
		case types.Enum:
			argTypes[i] = types.CommonEnumType
		case Expression:
			argTypes[i] = a.GetType()
		case types.Type:
			argTypes[i] = a
		}
	}
	return argTypes
}

func (s *ScalarFunction) GetArgTypes() []types.Type {
	return getArgTypes(s.args)
}

func (s *ScalarFunction) GetType() types.Type { return s.outputType }
func (s *ScalarFunction) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: s.ToProto(),
		},
	}
}

func (s *ScalarFunction) ToProto() *proto.Expression {
	args := make([]*proto.FunctionArgument, len(s.args))
	for i, a := range s.args {
		args[i] = a.ToProtoFuncArg()
	}

	return &proto.Expression{
		RexType: &proto.Expression_ScalarFunction_{
			ScalarFunction: &proto.Expression_ScalarFunction{
				FunctionReference: s.funcRef,
				Options:           s.options,
				OutputType:        types.TypeToProto(s.outputType),
				Arguments:         args,
			},
		},
	}
}

func (s *ScalarFunction) Equals(rhs Expression) bool {
	other, ok := rhs.(*ScalarFunction)
	if !ok {
		return false
	}

	switch {
	case s.funcRef != other.funcRef:
		return false
	case !s.outputType.Equals(other.outputType):
		return false
	}

	for i := range s.args {
		if !FuncArgsEqual(s.args[i], other.args[i]) {
			return false
		}
	}

	return slices.EqualFunc(s.options, other.options, func(a, b *types.FunctionOption) bool {
		return pb.Equal(a, b)
	})
}

func (s *ScalarFunction) Visit(visit VisitFunc) Expression {
	var args []types.FuncArg
	for i, arg := range s.args {
		var after types.FuncArg
		switch t := arg.(type) {
		case Expression:
			after = visit(t)
		default:
			after = arg
		}

		if args == nil && arg != after {
			args = make([]types.FuncArg, len(s.args))
			for j := 0; j < i; j++ {
				args[j] = s.args[j]
			}
		}

		if args != nil {
			args[i] = after
		}
	}

	if args == nil {
		return s
	}

	out := *s
	out.args = args
	return &out
}

type WindowFunction struct {
	funcRef     uint32
	declaration *extensions.WindowFunctionVariant

	args       []types.FuncArg
	options    []*types.FunctionOption
	outputType types.Type

	phase      types.AggregationPhase
	Sorts      []SortField
	invocation types.AggregationInvocation
	Partitions []Expression

	BoundsType             types.BoundsType
	LowerBound, UpperBound Bound
}

func NewCustomWindowFunc(
	reg ExtensionRegistry, v *extensions.WindowFunctionVariant, outputType types.Type,
	opts []*types.FunctionOption, invoke types.AggregationInvocation, phase types.AggregationPhase,
	args ...types.FuncArg,
) (*WindowFunction, error) {
	if outputType == nil {
		return nil, fmt.Errorf("%w: must provide non-nil output type", substraitgo.ErrInvalidExpr)
	}

	return &WindowFunction{
		funcRef:     reg.GetFuncAnchor(v.ID()),
		declaration: v,
		outputType:  outputType,
		options:     opts,
		args:        args,
		invocation:  invoke,
		phase:       phase,
	}, nil
}

func NewWindowFunc(
	reg ExtensionRegistry, id extensions.ID, opts []*types.FunctionOption,
	invoke types.AggregationInvocation, phase types.AggregationPhase, args ...types.FuncArg,
) (*WindowFunction, error) {
	decl, outType, err := resolveVariant(id, reg, reg.c.GetWindowFunc, args)
	if err != nil {
		return nil, err
	}

	if decl.Decomposability() == extensions.DecomposeNone && phase != types.AggPhaseInitialToResult {
		return nil, fmt.Errorf("%w: non-decomposable window or agg function '%s' must use InitialToResult phase",
			substraitgo.ErrInvalidExpr, id)
	}

	// We use the fully qualified ID for resolving an anchor, to make sure we
	// are using the correct compound name
	ref := reg.GetFuncAnchor(decl.ID())

	return &WindowFunction{
		funcRef:     ref,
		declaration: decl,
		outputType:  outType,
		options:     opts,
		args:        args,
		invocation:  invoke,
		phase:       phase,
	}, nil
}

func (w *WindowFunction) Name() string                            { return w.declaration.Name() }
func (w *WindowFunction) CompoundName() string                    { return w.declaration.CompoundName() }
func (w *WindowFunction) ID() extensions.ID                       { return w.declaration.ID() }
func (w *WindowFunction) Variadic() *extensions.VariadicBehavior  { return w.declaration.Variadic() }
func (w *WindowFunction) SessionDependant() bool                  { return w.declaration.SessionDependent() }
func (w *WindowFunction) Deterministic() bool                     { return w.declaration.Deterministic() }
func (w *WindowFunction) NArgs() int                              { return len(w.args) }
func (w *WindowFunction) Arg(i int) types.FuncArg                 { return w.args[i] }
func (w *WindowFunction) Phase() types.AggregationPhase           { return w.phase }
func (w *WindowFunction) Invocation() types.AggregationInvocation { return w.invocation }
func (w *WindowFunction) Decomposable() extensions.DecomposeType {
	return w.declaration.Decomposability()
}
func (w *WindowFunction) Ordered() bool { return w.declaration.Ordered() }
func (w *WindowFunction) MaxSet() int   { return w.declaration.MaxSet() }
func (w *WindowFunction) IntermediateType() (types.FuncDefArgType, error) {
	return w.declaration.Intermediate()
}
func (w *WindowFunction) WindowType() extensions.WindowType { return w.declaration.WindowType() }
func (*WindowFunction) IsScalar() bool                      { return false }

func (*WindowFunction) isRootRef() {}

func (w *WindowFunction) String() string {
	var b strings.Builder

	b.WriteString(w.declaration.Name())
	b.WriteByte('(')

	for i, arg := range w.args {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.String())
	}

	if len(w.Sorts) > 0 {
		b.WriteString("; sort: [")
		for i, s := range w.Sorts {
			if i != 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, "{expr: %s, %s}", s.Expr, s.Kind.String())
		}
		b.WriteString("]")
	}

	if len(w.options) > 0 {
		b.WriteString("; [options: {")
		for i, opt := range w.options {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s => %v", opt.Name, opt.Preference)
		}
		b.WriteString("}]")
	}

	if len(w.Partitions) > 0 {
		b.WriteString("; partitions: [")
		for i, part := range w.Partitions {
			if i != 0 {
				b.WriteString(", ")
			}
			b.WriteString(part.String())
		}
		b.WriteString("]")
	}

	if w.LowerBound != nil || w.UpperBound != nil {
		b.WriteString("; bounds: [")
		if w.LowerBound != nil {
			b.WriteString(w.LowerBound.String())
		}
		if w.UpperBound != nil {
			b.WriteString(", ")
			b.WriteString(w.UpperBound.String())
		}
		b.WriteString("]")
	}

	fmt.Fprintf(&b, "; phase: %s, invocation: %s) => %s",
		w.phase, w.invocation, w.outputType)

	return b.String()
}

func (w *WindowFunction) GetOptions() []*types.FunctionOption { return w.options }

func (w *WindowFunction) GetArgTypes() []types.Type {
	return getArgTypes(w.args)
}

func (w *WindowFunction) GetType() types.Type { return w.outputType }

// validate checks that the WindowFunction is valid according to Substrait
// specification rules. Currently validates that RANGE bounds type requires
// exactly one sort field.
func (w *WindowFunction) validate() error {
	if w.BoundsType == types.BoundsTypeRange && len(w.Sorts) != 1 {
		return fmt.Errorf("%w: RANGE bounds type requires exactly one sort field, got %d",
			substraitgo.ErrInvalidExpr, len(w.Sorts))
	}
	return nil
}

func (w *WindowFunction) Equals(other Expression) bool {
	rhs, ok := other.(*WindowFunction)
	if !ok {
		return false
	}

	switch {
	case w.funcRef != rhs.funcRef:
		return false
	case !w.outputType.Equals(rhs.outputType):
		return false
	case w.phase != rhs.phase || w.invocation != rhs.invocation:
		return false
	case w.LowerBound != rhs.LowerBound || w.UpperBound != rhs.UpperBound:
		return false
	case !slices.EqualFunc(w.options, rhs.options, func(l, r *types.FunctionOption) bool {
		return l.Name == r.Name && slices.Equal(l.Preference, r.Preference)
	}):
		return false
	case !slices.EqualFunc(w.Partitions, rhs.Partitions, exprEqual):
		return false
	case !slices.EqualFunc(w.Sorts, rhs.Sorts, func(l, r SortField) bool {
		return l.Expr.Equals(r.Expr) && l.Kind == r.Kind
	}):
		return false
	case !slices.EqualFunc(w.args, rhs.args, FuncArgsEqual):
		return false
	}

	return true
}

func (w *WindowFunction) ToProto() *proto.Expression {
	var (
		args       []*proto.FunctionArgument
		sorts      []*proto.SortField
		parts      []*proto.Expression
		upperBound *proto.Expression_WindowFunction_Bound
		lowerBound *proto.Expression_WindowFunction_Bound
	)

	if len(w.args) > 0 {
		args = make([]*proto.FunctionArgument, len(w.args))
		for i, a := range w.args {
			args[i] = a.ToProtoFuncArg()
		}
	}

	if len(w.Sorts) > 0 {
		sorts = make([]*proto.SortField, len(w.Sorts))
		for i, s := range w.Sorts {
			sorts[i] = s.ToProto()
		}
	}

	if len(w.Partitions) > 0 {
		parts = make([]*proto.Expression, len(w.Partitions))
		for i, p := range w.Partitions {
			parts[i] = p.ToProto()
		}
	}

	if w.UpperBound != nil {
		upperBound = w.UpperBound.ToProto()
	}

	if w.LowerBound != nil {
		lowerBound = w.LowerBound.ToProto()
	}

	return &proto.Expression{
		RexType: &proto.Expression_WindowFunction_{
			WindowFunction: &proto.Expression_WindowFunction{
				FunctionReference: w.funcRef,
				Arguments:         args,
				Options:           w.options,
				OutputType:        types.TypeToProto(w.outputType),
				Phase:             w.phase,
				Sorts:             sorts,
				Invocation:        w.invocation,
				Partitions:        parts,
				BoundsType:        w.BoundsType,
				LowerBound:        lowerBound,
				UpperBound:        upperBound,
			},
		},
	}
}

func (w *WindowFunction) ToProtoFuncArg() *proto.FunctionArgument {
	return &proto.FunctionArgument{
		ArgType: &proto.FunctionArgument_Value{
			Value: w.ToProto(),
		},
	}
}

func (w *WindowFunction) Visit(visit VisitFunc) Expression {
	var args []types.FuncArg
	for i, arg := range w.args {
		var after types.FuncArg
		switch t := arg.(type) {
		case Expression:
			after = visit(t)
		default:
			after = arg
		}

		if args == nil && arg != after {
			args = make([]types.FuncArg, len(w.args))
			for j := 0; j < i; j++ {
				args[j] = w.args[j]
			}
		}

		if args != nil {
			args[i] = after
		}
	}

	if args == nil {
		return w
	}

	out := *w
	out.args = args
	return &out
}

type AggregateFunction struct {
	funcRef     uint32
	declaration *extensions.AggregateFunctionVariant

	args       []types.FuncArg
	options    []*types.FunctionOption
	outputType types.Type
	phase      types.AggregationPhase
	invocation types.AggregationInvocation
	Sorts      []SortField
}

func NewAggregateFunc(
	reg ExtensionRegistry, id extensions.ID, opts []*types.FunctionOption,
	invoke types.AggregationInvocation, phase types.AggregationPhase, sorts []SortField,
	args ...types.FuncArg,
) (*AggregateFunction, error) {
	decl, outType, err := resolveVariant(id, reg, reg.c.GetAggregateFunc, args)
	if err != nil {
		return nil, err
	}

	// We use the fully qualified ID for resolving an anchor, to make sure we
	// are using the correct compound name
	ref := reg.GetFuncAnchor(decl.ID())

	return &AggregateFunction{
		funcRef:     ref,
		declaration: decl,
		outputType:  outType,
		options:     opts,
		args:        args,
		invocation:  invoke,
		phase:       phase,
		Sorts:       sorts,
	}, nil
}

func NewCustomAggregateFunc(
	reg ExtensionRegistry, v *extensions.AggregateFunctionVariant, outputType types.Type,
	opts []*types.FunctionOption, invoke types.AggregationInvocation, phase types.AggregationPhase,
	sorts []SortField, args ...types.FuncArg,
) (*AggregateFunction, error) {
	if outputType == nil {
		return nil, fmt.Errorf("%w: must provide non-nil output type", substraitgo.ErrInvalidExpr)
	}

	return &AggregateFunction{
		funcRef:    reg.GetFuncAnchor(v.ID()),
		outputType: outputType,
		options:    opts,
		args:       args,
		invocation: invoke,
		phase:      phase,
		Sorts:      sorts,
	}, nil
}

func NewAggregateFunctionFromProto(
	agg *proto.AggregateFunction, baseSchema *types.RecordType, reg ExtensionRegistry,
) (*AggregateFunction, error) {
	if agg.OutputType == nil {
		return nil, fmt.Errorf("%w: missing output type", substraitgo.ErrInvalidExpr)
	}

	var err error
	args := make([]types.FuncArg, len(agg.Arguments))
	for i, a := range agg.Arguments {
		if args[i], err = FuncArgFromProto(a, baseSchema, reg); err != nil {
			return nil, err
		}
	}

	sorts := make([]SortField, len(agg.Sorts))
	for i, s := range agg.Sorts {
		if sorts[i], err = SortFieldFromProto(s, baseSchema, reg); err != nil {
			return nil, err
		}
	}

	id, ok := reg.DecodeFunc(agg.FunctionReference)
	if !ok {
		return nil, substraitgo.ErrNotFound
	}
	decl, ok := reg.LookupAggregateFunction(agg.FunctionReference)
	if !ok {
		return NewCustomAggregateFunc(reg, extensions.NewAggFuncVariant(id), types.TypeFromProto(agg.OutputType), agg.Options, agg.Invocation, agg.Phase, sorts, args...)
	}

	return &AggregateFunction{
		funcRef:     agg.FunctionReference,
		declaration: decl,
		args:        args,
		options:     agg.Options,
		outputType:  types.TypeFromProto(agg.OutputType),
		phase:       agg.Phase,
		invocation:  agg.Invocation,
		Sorts:       sorts,
	}, nil
}

func (a *AggregateFunction) Name() string                            { return a.declaration.Name() }
func (a *AggregateFunction) CompoundName() string                    { return a.declaration.CompoundName() }
func (a *AggregateFunction) ID() extensions.ID                       { return a.declaration.ID() }
func (a *AggregateFunction) Variadic() *extensions.VariadicBehavior  { return a.declaration.Variadic() }
func (a *AggregateFunction) SessionDependant() bool                  { return a.declaration.SessionDependent() }
func (a *AggregateFunction) Deterministic() bool                     { return a.declaration.Deterministic() }
func (a *AggregateFunction) NArgs() int                              { return len(a.args) }
func (a *AggregateFunction) Arg(i int) types.FuncArg                 { return a.args[i] }
func (a *AggregateFunction) Phase() types.AggregationPhase           { return a.phase }
func (a *AggregateFunction) Invocation() types.AggregationInvocation { return a.invocation }
func (a *AggregateFunction) Decomposable() extensions.DecomposeType {
	return a.declaration.Decomposability()
}
func (a *AggregateFunction) Ordered() bool { return a.declaration.Ordered() }
func (a *AggregateFunction) MaxSet() int   { return a.declaration.MaxSet() }
func (a *AggregateFunction) IntermediateType() (types.FuncDefArgType, error) {
	return a.declaration.Intermediate()
}

func (a *AggregateFunction) Clone() *AggregateFunction {
	newA := *a
	if a.args != nil {
		newA.args = make([]types.FuncArg, len(a.args))
		copy(newA.args, a.args)
	}
	if a.options != nil {
		newA.options = make([]*types.FunctionOption, len(a.options))
		copy(newA.options, a.options)
	}
	if a.Sorts != nil {
		newA.Sorts = make([]SortField, len(a.Sorts))
		copy(newA.Sorts, a.Sorts)
	}
	return &newA
}

// SetArg sets the specified argument to the provided value.  The index is not checked for validity.
func (a *AggregateFunction) SetArg(i int, arg types.FuncArg) {
	a.args[i] = arg
}

func (a *AggregateFunction) String() string {
	var b strings.Builder

	b.WriteString(a.declaration.Name())
	b.WriteByte('(')

	for i, arg := range a.args {
		if i != 0 {
			b.WriteString(", ")
		}
		b.WriteString(arg.String())
	}

	b.WriteString(") => ")
	b.WriteString(a.outputType.String())

	return b.String()
}

func (a *AggregateFunction) GetOption(name string) []string {
	for _, o := range a.options {
		if name == o.Name {
			return o.GetPreference()
		}
	}
	return nil
}

func (a *AggregateFunction) GetOptions() []*types.FunctionOption { return a.options }

func (a *AggregateFunction) GetArgTypes() []types.Type {
	return getArgTypes(a.args)
}

func (a *AggregateFunction) GetType() types.Type { return a.outputType }

func (a *AggregateFunction) ToProto() *proto.AggregateFunction {
	var (
		args  []*proto.FunctionArgument
		sorts []*proto.SortField
	)
	if len(a.args) > 0 {
		args = make([]*proto.FunctionArgument, len(a.args))
		for i, arg := range a.args {
			args[i] = arg.ToProtoFuncArg()
		}
	}

	if len(a.Sorts) > 0 {
		sorts = make([]*proto.SortField, len(a.Sorts))
		for i, s := range a.Sorts {
			sorts[i] = s.ToProto()
		}
	}

	return &proto.AggregateFunction{
		FunctionReference: a.funcRef,
		Arguments:         args,
		Options:           a.options,
		OutputType:        types.TypeToProto(a.outputType),
		Phase:             a.phase,
		Sorts:             sorts,
		Invocation:        a.invocation,
	}
}
