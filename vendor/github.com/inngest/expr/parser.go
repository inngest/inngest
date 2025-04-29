package expr

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

// TreeParser parses an expression into a tree, with a root node and branches for
// each subsequent OR or AND expression.
type TreeParser interface {
	Parse(ctx context.Context, eval Evaluable) (*ParsedExpression, error)
}

// CELCompiler represents a CEL compiler which takes an expression string
// and returns a CEL AST, any issues during parsing, and any lifted and replaced
// from the expression.
//
// By default, *cel.Env fulfils this interface.  In production, it's common
// to provide a caching layer on top of *cel.Env to optimize parsing, as it's
// the slowest part of the expression process.
type CELCompiler interface {
	// Compile calls Compile on the expression, parsing and validating the AST.
	// This returns the AST, issues during validation, and args lifted.
	Compile(expr string) (*cel.Ast, *cel.Issues, LiftedArgs)
	// Parse calls Parse on an expression, but does not check the expression
	// for valid variable names etc. within the env.
	Parse(expr string) (*cel.Ast, *cel.Issues, LiftedArgs)
}

// EnvCompiler turns a *cel.Env into a CELParser.
func EnvCompiler(env *cel.Env) CELCompiler {
	return envparser{env}
}

type envparser struct {
	env *cel.Env
}

func (e envparser) Parse(txt string) (*cel.Ast, *cel.Issues, LiftedArgs) {
	ast, iss := e.env.Parse(txt)
	return ast, iss, nil
}

func (e envparser) Compile(txt string) (*cel.Ast, *cel.Issues, LiftedArgs) {
	ast, iss := e.env.Compile(txt)
	return ast, iss, nil
}

// NewTreeParser returns a new tree parser for a given *cel.Env
func NewTreeParser(ep CELCompiler) TreeParser {
	parser := &parser{
		ep: ep,
	}
	return parser
}

type parser struct {
	ep CELCompiler

	// rander is a random reader set during testing.  it is never used outside
	// of the test package during Parse.  Instead,  a new deterministic random
	// reader is generated from the Evaluable identifier.
	rander RandomReader
}

func (p *parser) Parse(ctx context.Context, eval Evaluable) (*ParsedExpression, error) {
	expression := eval.GetExpression() // "event.data.id == '1'"
	if expression == "" {
		return &ParsedExpression{
			EvaluableID: eval.GetID(),
		}, nil
	}

	ast, issues, vars := p.ep.Parse(expression)
	if issues != nil {
		return nil, issues.Err()
	}

	r := p.rander
	if r == nil {
		// Create a new deterministic random reader based off of the evaluable's identifier.
		// This means that every time we parse an expression with the given identifier, the
		// group IDs will be deterministic as the randomness is sourced from the ID.
		//
		// We only overwrite this if rander is not nil so that we can inject rander during tests.
		id := eval.GetID()
		seed := int64(binary.NativeEndian.Uint64(id[:8]))
		r = rand.New(rand.NewSource(seed)).Read
	}

	node := newNode()
	_, hasMacros, err := navigateAST(
		expr{
			ast: ast.NativeRep().Expr(),
		},
		node,
		vars,
		r,
	)
	if err != nil {
		return nil, err
	}

	node.normalize()
	return &ParsedExpression{
		Root:        *node,
		Vars:        vars,
		EvaluableID: eval.GetID(),
		HasMacros:   hasMacros,
	}, nil
}

// ParsedExpression represents a parsed CEL expression into our higher-level AST.
//
// Expressions are simplified and canonicalized, eg. !(a == b) becomes a != b and
// !(b <= a) becomes (a > b).
type ParsedExpression struct {
	Root Node

	// Vars represents rewritten literals within the expression.
	//
	// This allows us to rewrite eg. `event.data.id == "foo"` into
	// `event.data.id == vars.a` such that multiple different literals
	// share the same expression.  Using the same expression allows us
	// to cache and skip CEL parsing, which is the slowest aspect of
	// expression matching.
	Vars LiftedArgs

	// Evaluable stores the original evaluable interface that was parsed.
	EvaluableID uuid.UUID

	HasMacros bool
}

// RootGroups returns the top-level matching groups within an expression.  This is a small
// utility to check the number of matching groups easily.
func (p ParsedExpression) RootGroups() []*Node {
	if len(p.Root.Ands) == 0 && len(p.Root.Ors) > 1 {
		return p.Root.Ors
	}
	return []*Node{&p.Root}
}

// PredicateGroup represents a group of predicates that must all pass in order to execute the
// given expression.  For example, this might contain two predicates representing an expression
// with two operators combined with "&&".
//
// MATCHING & EVALUATION
//
// A node evaluates to true if ALL of the following conditions are met:
//
// - All of the ANDS are truthy.
// - One or more of the ORs are truthy
//
// In essence, if there are ANDs and ORs, the ORs are implicitly added to ANDs:
//
//	(A && (B || C))
//
// This requres A *and* either B or C, and so we require all ANDs plus at least one node
// from OR to evaluate to true
type Node struct {
	GroupID groupID

	// Ands contains predicates at this level of the expression that are joined together
	// with an && operator.  All nodes in this set must evaluate to true in order for this
	// node in the expression to be truthy.
	//
	// Note that if any on of the Ors nodes evaluates to true, this node is truthy, regardless
	// of whether the Ands set evaluates to true.
	Ands []*Node `json:"and,omitempty"`

	// Ors represents matching OR groups within this expression.  For example, in
	// the expression `a == b && (c == 1 || d == 1)` the top-level predicate group will
	// have a child group containing the parenthesis sub-expression.
	//
	// At least one of the Or node's sub-trees must evaluate to true for the node to
	// be truthy, alongside all Ands.
	Ors []*Node `json:"or,omitempty"`

	// Predicate represents the predicate for this node.  This must evaluate to true in order
	// for the expression to be truthy.
	//
	// If this is nil, this is a parent container for a series of AND or Or checks.
	// a == b
	Predicate *Predicate
}

func (n Node) HasPredicate() bool {
	if n.Predicate == nil {
		return false
	}
	return n.Predicate.Operator != ""
}

func (n *Node) normalize() {
	if n.Predicate != nil {
		return
	}
	if len(n.Ands) == 0 {
		n.Ands = nil
	}
	if len(n.Ors) == 0 {
		n.Ors = nil
	}
	if len(n.Ands) == 1 && len(n.Ors) == 0 {
		// Check to see if the child is an orphan.
		child := n.Ands[0]
		if len(child.Ands) == 0 && len(child.Ors) == 0 && child.Predicate != nil {
			n.Predicate = child.Predicate
			n.Ands = nil
			return
		}
	}
}

func (n *Node) String() string {
	return n.string(0)
}

func (n *Node) string(depth int) string {
	builder := strings.Builder{}

	// If there are both ANDs and ORs in this node, wrap the entire
	// thing in parenthesis to minimize ambiguity.
	writeOuterParen := (len(n.Ands) >= 1 && len(n.Ors) >= 1 && depth > 0) ||
		(len(n.Ands) > 1 && depth > 0) // Chain multiple joined ands together when nesting.

	if writeOuterParen {
		builder.WriteString("(")
	}

	for i, and := range n.Ands {
		builder.WriteString(and.string(depth + 1))
		if i < len(n.Ands)-1 {
			// If this is not the last and, write an ampersand.
			builder.WriteString(" && ")
		}
	}

	// Tie the ANDs and ORs together with an and operand.
	if len(n.Ands) > 0 && len(n.Ors) > 0 {
		builder.WriteString(" && ")
	}

	// Write the "or" groups out, concatenated each with an Or operand..
	//
	// We skip this for the top-level node to remove extra meaningless
	// parens that wrap the entire expression

	writeOrParen := len(n.Ors) > 1 && depth > 0 || // Always chain nested ors
		len(n.Ors) > 1 && len(n.Ands) >= 1

	if writeOrParen {
		builder.WriteString("(")
	}
	for i, or := range n.Ors {
		builder.WriteString(or.string(depth + 1))
		if i < len(n.Ors)-1 {
			// If this is not the last and, write an Or operand..
			builder.WriteString(" || ")
		}
	}
	if writeOrParen {
		builder.WriteString(")")
	}

	// Write the actual clause.
	if n.Predicate != nil {
		builder.WriteString(n.Predicate.String())
	}

	// And finally, the outer paren.
	if writeOuterParen {
		builder.WriteString(")")
	}

	return builder.String()
}

func newNode() *Node {
	return &Node{}
}

// Predicate represents a predicate that must evaluate to true in order for an expression to
// be considered as viable when checking an event.
//
// This is equivalent to a CEL overload/function/macro.
type Predicate struct {
	// Literal represents the literal value that the operator compares against.  If two
	// variable are being compared, this is nil and LiteralIdent holds a pointer to the
	// name of the second variable.
	Literal any

	// Ident is the ident we're comparing to, eg. the variable.
	Ident string

	// LiteralIdent represents the second literal that we're comparing against,
	// eg. in the expression "event.data.a == event.data.b" this stores event.data.b
	LiteralIdent *string

	// Operator is the binary operator being used.  NOTE:  This always assumes that the
	// ident is to the left of the operator, eg "event.data.value > 100".  If the value
	// is to the left of the operator, the operator will be switched
	// (ie. 100 > event.data.value becomes event.data.value < 100)
	Operator string
}

func (p Predicate) String() string {
	lit := p.Literal
	if p.LiteralIdent != nil {
		lit = *p.LiteralIdent
	}

	switch str := p.Literal.(type) {
	case string:
		return fmt.Sprintf("%s %s %v", p.Ident, strings.ReplaceAll(p.Operator, "_", ""), strconv.Quote(str))
	case nil:
		if p.LiteralIdent == nil {
			// print `foo == null` instead of `foo == <nil>`, the Golang default.
			// We onyl do this if we're not comparing to an identifier.
			return fmt.Sprintf("%s %s null", p.Ident, strings.ReplaceAll(p.Operator, "_", ""))
		}
		return fmt.Sprintf("%s %s %v", p.Ident, strings.ReplaceAll(p.Operator, "_", ""), lit)
	default:
		return fmt.Sprintf("%s %s %v", p.Ident, strings.ReplaceAll(p.Operator, "_", ""), lit)
	}
}

func (p Predicate) LiteralAsString() string {
	str, _ := p.Literal.(string)
	return str
}

func (p Predicate) LiteralAsFloat64() (float64, error) {
	switch v := p.Literal.(type) {
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	}
	return 0, fmt.Errorf("not an int64 or float64")
}

// expr is wrapper around the CEL AST which stores parsing-related data.
type expr struct {
	ast celast.Expr

	// negated is true when this expr is part of a logical not branch,
	// ie !($expr)
	negated bool
}

// navigateAST iterates through an expression AST, parsing predicates into groups.
//
// It does this by iterating through the expression, amending the current `group` until
// an or expression is found.  When an or expression is found, we create another group which
// is mutated by the iteration.
func navigateAST(nav expr, parent *Node, vars LiftedArgs, rand RandomReader) ([]*Node, bool, error) {
	// on the very first call to navigateAST, ensure that we set the first node
	// inside the nodemap.
	result := []*Node{}

	hasMacros := false

	// Iterate through the stack, recursing down into each function call (eg. && branches).
	stack := []expr{nav}
	for len(stack) > 0 {
		item := stack[0]
		stack = stack[1:]

		switch item.ast.Kind() {
		case celast.SelectKind:
			c := item.ast.AsSelect()
			child := &Node{
				Predicate: &Predicate{
					Ident:    c.FieldName(),
					Operator: "select",
				},
			}
			child.normalize()
			result = append(result, child)
			hasMacros = true
		case celast.ComprehensionKind:
			// These are not supported.  A comprehension is eg. `.exists` and must
			// always run naively right now.
			c := item.ast.AsComprehension()
			child := &Node{
				Predicate: &Predicate{
					Ident:    c.IterVar(),
					Operator: "comprehension",
				},
			}
			child.normalize()
			result = append(result, child)
			hasMacros = true
		case celast.LiteralKind:
			// This is a literal. Do nothing, as this is always true.
		case celast.IdentKind:
			// This is a variable. Do nothing.
		case celast.CallKind:
			// Call kinds are the actual comparator operators, eg. >=, or &&.  These are specifically
			// what we're trying to parse, by taking the LHS and RHS of each opeartor then bringing
			// this up into a tree.

			fn := item.ast.AsCall().FunctionName()

			// Firstly, if this is a logical not, everything within this branch is negated:
			// !(a == b).  This flips the negated field, ie !(foo == bar) becomes foo != bar,
			// whereas !(!(foo == bar)) stays the same.
			if fn == operators.LogicalNot {
				// Immediately navigate into this single expression.
				astChild := item.ast.AsCall().Args()[0]
				stack = append(stack, expr{
					ast:     astChild,
					negated: !item.negated,
				})
				continue
			}

			if fn == operators.LogicalOr {
				for _, or := range peek(item, operators.LogicalOr) {
					var err error
					// Ors modify new nodes.  Assign a new Node to each
					// Or entry.
					newParent := newNode()

					// For each item in the stack, recurse into that AST.
					_, macros, err := navigateAST(or, newParent, vars, rand)
					if macros {
						hasMacros = true
					}
					if err != nil {
						return nil, hasMacros, err
					}

					// Ensure that we remove any redundant parents generated.
					newParent.normalize()
					if parent.Ors == nil {
						parent.Ors = []*Node{}
					}
					parent.Ors = append(parent.Ors, newParent)
				}
				continue
			}

			// For each &&, create a new child node in the .And field of the current
			// high-level AST.
			if item.ast.AsCall().FunctionName() == operators.LogicalAnd {
				stack = append(stack, peek(item, operators.LogicalAnd)...)
				continue
			}

			// This is a function call, ie. a binary op equality check with two
			// arguments, or a ternary operator.
			//
			// We assume that this is being called with an ident as a comparator.
			// Dependign on the LHS/RHS type, we want to organize the kind into
			// a specific type of tree.
			predicate := callToPredicate(item.ast, item.negated, vars)
			if predicate == nil {
				continue
			}

			child := &Node{
				Predicate: predicate,
			}
			child.normalize()
			result = append(result, child)
		}
	}

	parent.Ands = result

	// Add a group ID to the parent.
	total := len(parent.Ands)
	if parent.Predicate != nil {
		total += 1
	}
	if len(parent.Ors) >= 1 {
		total += 1
	}

	// For each AND, check to see if we have more than one string part, and check to see
	// whether we have a "!=" and an "==" chained together.  If so, this lets us optimize
	// != checks so that we only return the aggregate match if the other "==" also matches.
	//
	// This is necessary:  != returns basically every expression part, which is hugely costly
	// in terms of allocation.  We want to avoid that if poss.
	var (
		stringEq     uint8
		hasStringNeq bool
	)
	for _, item := range parent.Ands {
		if item.Predicate == nil {
			continue
		}
		if _, ok := item.Predicate.Literal.(string); !ok {
			continue
		}
		if item.Predicate.Operator == operators.Equals {
			stringEq++
		}
		if item.Predicate.Operator == operators.NotEquals {
			hasStringNeq = true
		}
	}

	flag := byte(OptimizeNone)
	if stringEq > 0 && hasStringNeq {
		// The flag is the number of string equality checks in the == group.
		flag = byte(stringEq)
	}

	// Create a new group ID which tracks the number of expressions that must match
	// within this group in order for the group to pass.
	//
	// This includes ALL ands, plus at least one OR.
	//
	// When checking an incoming event, we match the event against each node's
	// ident/variable.  Using the group ID, we can see if we've matched N necessary
	// items from the same identifier.  If so, the evaluation is true.
	parent.GroupID = newGroupIDWithReader(uint16(total), flag, rand)

	// For each sub-group, add the same group IDs to children if there's no nesting.
	//
	// We do this so that the parent node which contains all ANDs can correctly set
	// the same group ID for all child predicates.  This is necessasry;  if you compare
	// A && B && C, we want all of A/B/C to share the same group ID
	for n, item := range parent.Ands {
		if len(item.Ands) == 0 && len(item.Ors) == 0 && item.Predicate != nil {
			item.GroupID = parent.GroupID
			parent.Ands[n] = item
		}
	}
	for n, item := range parent.Ors {
		if len(item.Ands) == 0 && len(item.Ors) == 0 && item.Predicate != nil {
			item.GroupID = parent.GroupID
			parent.Ors[n] = item
		}
	}

	return result, hasMacros, nil
}

// peek recurses through nested operators (eg. a && b && c), grouping all operators
// together into an array.  This stops after exhausting matching operators.
func peek(nav expr, operator string) []expr {
	// Recurse into the children matching all consecutive child types,
	// eg. all ANDs, or all ORs.
	//
	// For each non-operator found, add it to a return list.
	stack := []expr{nav}
	result := []expr{}
	for len(stack) > 0 {
		item := stack[0]
		stack = stack[1:]

		if item.ast.AsCall().FunctionName() == operator {
			astChildren := item.ast.AsCall().Args()
			stack = append(
				stack,
				expr{
					ast:     astChildren[0],
					negated: nav.negated,
				},
				expr{
					ast:     astChildren[1],
					negated: nav.negated,
				},
			)
			continue
		}
		// This is not an AND or OR call, so don't recurse into it - leave this
		// as a result value for handling.
		//
		// In this case, we either have operators (>=) or OR tests.
		result = append(result, item)
	}
	return result
}

// callToPredicate transforms a function call within an expression (eg `>`) into
// a Predicate struct for our matching engine.  It ahandles normalization of
// LHS/RHS plus inversions.
func callToPredicate(item celast.Expr, negated bool, vars LiftedArgs) *Predicate {
	fn := item.AsCall().FunctionName()
	if fn == operators.LogicalAnd || fn == operators.LogicalOr {
		// Quit early, as we descend into these while iterating through the tree when calling this.
		return nil
	}

	// If this is in a negative expression (ie. `!(foo == bar)`), then invert the expression.
	if negated {
		fn = invert(fn)
	}

	args := item.AsCall().Args()
	if len(args) != 2 {
		return nil
	}

	var (
		identA, identB string
		literal        any
	)

	for _, item := range args {
		var ident string

		switch item.Kind() {
		case celast.CallKind:
			ident = parseArrayAccess(item)
			if ident == "" {
				// TODO: Panic or mark as non-exhaustive parse.
				return nil
			}
		case celast.IdentKind:
			ident = item.AsIdent()
		case celast.LiteralKind:
			literal = item.AsLiteral().Value()
		case celast.MapKind:
			literal = item.AsMap()
		case celast.SelectKind:
			// This is an expression, ie. "event.data.foo"  Iterate from the root field upwards
			// to get the full ident.
			ident = walkSelect(item)
		}

		if ident != "" {
			if identA == "" {
				// Set the first ident
				identA = ident
			} else {
				// Set the second.
				identB = ident
			}
		}
	}

	// If the literal is of type `structpb.NullValue`, replace this with a simple `nil`
	// to make nil checks easy.
	if _, ok := literal.(structpb.NullValue); ok {
		literal = nil
	}

	if identA != "" && identB != "" {
		// We're matching two variables together.  Check to see whether any
		// of these idents have variable data being passed in above.
		//
		// This happens when we use a parser which "lifts" variables out of
		// expressions to improve cache hits.
		//
		// Parsing can normalize `event.data.id == "1"` to
		// `event.data.id == vars.a` && vars["a"] = "1".
		//
		// In this case, check to see if we're using a lifted var and, if so,
		// use the variable as the ident directly.
		aIsVar := strings.HasPrefix(identA, VarPrefix)
		bIsVar := strings.HasPrefix(identB, VarPrefix)

		if aIsVar && bIsVar {
			// Someone is matching two literals together, so.... this,
			// is quite dumb.
			//
			// Do nothing but match on two vars.
			return &Predicate{
				LiteralIdent: &identB,
				Ident:        identA,
				Operator:     fn,
			}
		}

		if aIsVar && vars != nil {
			if val, ok := vars.Get(strings.TrimPrefix(identA, VarPrefix+".")); ok {
				// Normalize.
				literal = val
				identA = identB
				identB = ""
			}
		}

		if bIsVar && vars != nil {
			if val, ok := vars.Get(strings.TrimPrefix(identB, VarPrefix+".")); ok {
				// Normalize.
				literal = val
				identB = ""
			}
		}

		if identA != "" && identB != "" {
			// THese are still idents, so handle them as
			// variables being compared together.
			return &Predicate{
				LiteralIdent: &identB,
				Ident:        identA,
				Operator:     fn,
			}
		}
	}

	// if identA == "" || literal == nil {
	// 	return nil
	// }

	// We always assume that the ident is on the LHS.  In the case of comparisons,
	// we need to switch these and the operator if the literal is on the RHS.  This lets
	// us normalize all expressions and ensure correct ordering within Predicates.
	//
	// NOTE: If we passed the specific function into a predicate result we would not have to do this;
	// we could literally call the function with its binary args.  All we have is the AST, and
	// we don't want to pass the raw AST into Predicate as it contains too much data.

	switch fn {
	case operators.Equals, operators.NotEquals:
		// NOTE: NotEquals is _not_ supported.  This requires selecting all leaf nodes _except_
		// a given leaf, iterating over a tree.  We may as well execute every expressiona s the difference
		// is negligible.
	case operators.Greater, operators.GreaterEquals, operators.Less, operators.LessEquals:
		// We only support these operators for ints and floats, right now.
		// In the future we need to support scanning trees from a specific key
		// onwards.
		switch literal.(type) {
		case int64, float64:
			// Allowed
		case string:
			// Also allowed, eg. for matching datetime strings or filtering ULIDs after
			// a specific string.
		default:
			return nil
		}

		// Ensure we normalize `a > 100` and `100 < a` so that the literal is last.
		// This ensures we treat all expressions the same.
		if args[0].Kind() == celast.LiteralKind {
			// Switch the operators to ensure evaluation of predicates is correct and consistent.
			fn = normalize(fn)
		}
	default:
		return nil
	}

	return &Predicate{
		Literal:  literal,
		Ident:    identA,
		Operator: fn,
	}
}

func walkSelect(item celast.Expr) string {
	// This is an expression, ie. "event.data.foo"  Iterate from the root field upwards
	// to get the full ident.
	walked := ""
	for item.Kind() == celast.SelectKind {
		sel := item.AsSelect()
		if walked == "" {
			walked = sel.FieldName()
		} else {
			walked = sel.FieldName() + "." + walked
		}
		item = sel.Operand()
		if item.Kind() == celast.CallKind {
			arrayPrefix := parseArrayAccess(item)
			walked = arrayPrefix + "." + walked
		}
	}
	return strings.TrimPrefix(item.AsIdent()+"."+walked, ".")
}

func parseArrayAccess(item celast.Expr) string {
	// The only supported accessor here is _[_], which is an array index accessor.
	if item.AsCall().FunctionName() != operators.Index && item.AsCall().FunctionName() != operators.OptIndex {
		return ""
	}
	args := item.AsCall().Args()
	return fmt.Sprintf("%s[%v]", walkSelect(args[0]), args[1].AsLiteral().Value())
}

func invert(op string) string {
	switch op {
	case operators.Equals:
		return operators.NotEquals
	case operators.NotEquals:
		return operators.Equals
	case operators.Greater:
		// NOTE: Negating a > turns this into <=.  5 >= 5 == true, and only 5 < 5
		// negates this.
		return operators.LessEquals
	case operators.GreaterEquals:
		return operators.Less
	case operators.Less:
		return operators.GreaterEquals
	case operators.LessEquals:
		return operators.Greater
	default:
		return op
	}
}

func normalize(op string) string {
	switch op {
	case operators.Greater:
		return operators.Less
	case operators.GreaterEquals:
		return operators.LessEquals
	case operators.Less:
		return operators.Greater
	case operators.LessEquals:
		return operators.GreaterEquals
	default:
		return op
	}
}
