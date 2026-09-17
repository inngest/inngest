// pkg/duckdb/parser/ast.go
package parser

import (
	"fmt"
	"strings"
)

type Expr interface {
	Node
	isExpr()
}

// baseExpr carries a node's full source span. Every concrete AST type
// embeds this (directly for Expr types, or via baseTableRefNode/
// groupByItemNode for TableRef/GroupByItem types) so any node can be
// located in the original text via src[n.Pos().Offset:n.End().Offset] —
// see Node's doc comment in Task 6's visitor.go.
type baseExpr struct{ start, end Position }

func (b baseExpr) Pos() Position { return b.start }
func (b baseExpr) End() Position { return b.end }
func (baseExpr) isExpr()         {}

// nonNil filters absent optional children out of a Children() result. Only
// safe to call with arguments whose static field type is already an
// interface (Expr, Node, TableRef) — e.g. c.Operand where Operand Expr — not
// with a concrete pointer type (e.g. *OrderByClause). A nil *T boxed into
// the Node parameter here produces a NON-nil interface value (the classic
// Go gotcha: the interface's type word is set even though its data word is
// nil), so this filter would wrongly treat a nil *OrderByClause as present.
// Concrete-pointer-typed fields must be nil-checked before appending
// instead (see SelectStatement.Children below for the pattern).
func nonNil(nodes ...Node) []Node {
	var out []Node
	for _, n := range nodes {
		if n != nil {
			out = append(out, n)
		}
	}
	return out
}

// --- literals, names, parameters ---

type Ident struct {
	baseExpr
	Parts []string
}

func (i *Ident) Children() []Node { return nil }
func (i *Ident) String() string   { return fmt.Sprintf("Ident(%s)", strings.Join(i.Parts, ".")) }

type LiteralKind int

const (
	LitString LiteralKind = iota
	LitNumber
	LitNull
	LitTrue
	LitFalse
)

type Literal struct {
	baseExpr
	Kind LiteralKind
	Text string
}

func (l *Literal) Children() []Node { return nil }
func (l *Literal) String() string   { return fmt.Sprintf("Literal(%d,%s)", l.Kind, l.Text) }

type ParamKind int

const (
	ParamAnonymous ParamKind = iota
	ParamNumbered
	ParamNamed
)

type Parameter struct {
	baseExpr
	Kind  ParamKind
	Value string
}

func (p *Parameter) Children() []Node { return nil }
func (p *Parameter) String() string   { return fmt.Sprintf("Parameter(%d,%s)", p.Kind, p.Value) }

// --- operators ---

type BinaryExpr struct {
	baseExpr
	Op          string
	Left, Right Expr
}

func (b *BinaryExpr) Children() []Node { return []Node{b.Left, b.Right} }
func (b *BinaryExpr) String() string   { return "BinaryExpr(" + b.Op + ")" }

type UnaryExpr struct {
	baseExpr
	Op      string
	X       Expr
	Postfix bool
}

func (u *UnaryExpr) Children() []Node { return []Node{u.X} }
func (u *UnaryExpr) String() string   { return "UnaryExpr(" + u.Op + ")" }

type NullTest struct {
	baseExpr
	X   Expr
	Not bool
}

func (n *NullTest) Children() []Node { return []Node{n.X} }
func (n *NullTest) String() string   { return fmt.Sprintf("NullTest(not=%v)", n.Not) }

type IsExpr struct {
	baseExpr
	X     Expr
	Not   bool
	Value string
}

func (i *IsExpr) Children() []Node { return []Node{i.X} }
func (i *IsExpr) String() string   { return fmt.Sprintf("IsExpr(not=%v,%s)", i.Not, i.Value) }

type DistinctFromExpr struct {
	baseExpr
	Left, Right Expr
	Not         bool
}

func (d *DistinctFromExpr) Children() []Node { return []Node{d.Left, d.Right} }
func (d *DistinctFromExpr) String() string   { return fmt.Sprintf("DistinctFromExpr(not=%v)", d.Not) }

type BetweenExpr struct {
	baseExpr
	X, Low, High Expr
	Not          bool
}

func (b *BetweenExpr) Children() []Node { return []Node{b.X, b.Low, b.High} }
func (b *BetweenExpr) String() string   { return fmt.Sprintf("BetweenExpr(not=%v)", b.Not) }

type InExpr struct {
	baseExpr
	X        Expr
	Not      bool
	List     []Expr
	Subquery *SelectStatement
}

func (i *InExpr) Children() []Node {
	children := []Node{i.X}
	for _, e := range i.List {
		children = append(children, e)
	}
	if i.Subquery != nil {
		children = append(children, i.Subquery)
	}
	return children
}
func (i *InExpr) String() string { return fmt.Sprintf("InExpr(not=%v)", i.Not) }

type LikeExpr struct {
	baseExpr
	X, Pattern Expr
	Not        bool
	Op         string
	Escape     Expr
}

func (l *LikeExpr) Children() []Node { return nonNil(l.X, l.Pattern, l.Escape) }
func (l *LikeExpr) String() string   { return fmt.Sprintf("LikeExpr(%s,not=%v)", l.Op, l.Not) }

type CastExpr struct {
	baseExpr
	X       Expr
	Type    string
	TryCast bool
}

func (c *CastExpr) Children() []Node { return []Node{c.X} }
func (c *CastExpr) String() string   { return fmt.Sprintf("CastExpr(%s,try=%v)", c.Type, c.TryCast) }

type CaseWhen struct {
	baseExpr
	Cond, Result Expr
}

func (w *CaseWhen) Children() []Node { return []Node{w.Cond, w.Result} }
func (w *CaseWhen) String() string   { return "CaseWhen" }

type CaseExpr struct {
	baseExpr
	Operand Expr
	Whens   []*CaseWhen
	Else    Expr
}

func (c *CaseExpr) Children() []Node {
	children := nonNil(c.Operand)
	for _, w := range c.Whens {
		children = append(children, w)
	}
	return append(children, nonNil(c.Else)...)
}
func (c *CaseExpr) String() string { return "CaseExpr" }

type StarExpr struct {
	baseExpr
	Qualifier []string
	Exclude   []string
}

func (s *StarExpr) Children() []Node { return nil }
func (s *StarExpr) String() string {
	return fmt.Sprintf("StarExpr(%s,exclude=%v)", strings.Join(s.Qualifier, "."), s.Exclude)
}

type SubqueryExpr struct {
	baseExpr
	Select *SelectStatement
	Exists bool
	Not    bool
}

func (s *SubqueryExpr) Children() []Node { return []Node{s.Select} }
func (s *SubqueryExpr) String() string {
	return fmt.Sprintf("SubqueryExpr(exists=%v,not=%v)", s.Exists, s.Not)
}

type ListExpr struct {
	baseExpr
	Elems []Expr
	Paren bool
}

func (l *ListExpr) Children() []Node {
	children := make([]Node, len(l.Elems))
	for i, e := range l.Elems {
		children[i] = e
	}
	return children
}
func (l *ListExpr) String() string { return fmt.Sprintf("ListExpr(paren=%v)", l.Paren) }

type StructField struct {
	baseExpr
	Key   string
	Value Expr
}

func (f *StructField) Children() []Node { return []Node{f.Value} }
func (f *StructField) String() string   { return "StructField(" + f.Key + ")" }

type StructExpr struct {
	baseExpr
	Fields []*StructField
}

func (s *StructExpr) Children() []Node {
	children := make([]Node, len(s.Fields))
	for i, f := range s.Fields {
		children[i] = f
	}
	return children
}
func (s *StructExpr) String() string { return "StructExpr" }

type MapExpr struct {
	baseExpr
	Keys, Values []Expr
}

func (m *MapExpr) Children() []Node {
	var children []Node
	for i := range m.Keys {
		children = append(children, m.Keys[i], m.Values[i])
	}
	return children
}
func (m *MapExpr) String() string { return "MapExpr" }

type IntervalExpr struct {
	baseExpr
	Value Expr
	Unit  string
}

func (i *IntervalExpr) Children() []Node { return []Node{i.Value} }
func (i *IntervalExpr) String() string   { return "IntervalExpr(" + i.Unit + ")" }

type TypeLiteral struct {
	baseExpr
	Type  string
	Value string
}

func (t *TypeLiteral) Children() []Node { return nil }
func (t *TypeLiteral) String() string   { return fmt.Sprintf("TypeLiteral(%s,%s)", t.Type, t.Value) }

type NamedArg struct {
	baseExpr
	Name  string
	Value Expr
}

func (a *NamedArg) Children() []Node { return []Node{a.Value} }
func (a *NamedArg) String() string   { return "NamedArg(" + a.Name + ")" }

type FunctionExpr struct {
	baseExpr
	Name         []string
	Receiver     Expr
	Distinct     bool
	All          bool
	Args         []Expr
	OrderBy      *OrderByClause
	IgnoreNulls  bool
	RespectNulls bool
	Filter       Expr
	Within       *OrderByClause
	Export       bool
	Over         *WindowSpec
}

func (f *FunctionExpr) Children() []Node {
	children := nonNil(f.Receiver)
	for _, a := range f.Args {
		children = append(children, a)
	}
	if f.OrderBy != nil {
		children = append(children, f.OrderBy)
	}
	children = append(children, nonNil(f.Filter)...)
	if f.Within != nil {
		children = append(children, f.Within)
	}
	if f.Over != nil {
		children = append(children, f.Over)
	}
	return children
}
func (f *FunctionExpr) String() string {
	return fmt.Sprintf("FunctionExpr(%s,distinct=%v)", strings.Join(f.Name, "."), f.Distinct)
}

type WindowSpec struct {
	baseExpr
	Name        string
	PartitionBy []Expr
	OrderBy     *OrderByClause
	Frame       *FrameClause
}

func (w *WindowSpec) Children() []Node {
	var children []Node
	for _, p := range w.PartitionBy {
		children = append(children, p)
	}
	if w.OrderBy != nil {
		children = append(children, w.OrderBy)
	}
	if w.Frame != nil {
		children = append(children, w.Frame)
	}
	return children
}
func (w *WindowSpec) String() string { return "WindowSpec(" + w.Name + ")" }

type FrameClause struct {
	baseExpr
	Unit     string
	Start    *FrameBound
	EndBound *FrameBound // named EndBound, not End — End() is Node's span-end method
	Exclude  string
}

func (f *FrameClause) Children() []Node {
	var children []Node
	if f.Start != nil {
		children = append(children, f.Start)
	}
	if f.EndBound != nil {
		children = append(children, f.EndBound)
	}
	return children
}
func (f *FrameClause) String() string { return "FrameClause(" + f.Unit + ")" }

type FrameBoundKind int

const (
	FrameUnboundedPreceding FrameBoundKind = iota
	FrameUnboundedFollowing
	FrameCurrentRow
	FramePreceding
	FrameFollowing
)

type FrameBound struct {
	baseExpr
	Kind FrameBoundKind
	Expr Expr
}

func (b *FrameBound) Children() []Node { return nonNil(b.Expr) }
func (b *FrameBound) String() string   { return fmt.Sprintf("FrameBound(%d)", b.Kind) }

type DotExpr struct {
	baseExpr
	X     Expr
	Field string
}

func (d *DotExpr) Children() []Node { return []Node{d.X} }
func (d *DotExpr) String() string   { return "DotExpr(" + d.Field + ")" }

type SliceExpr struct {
	baseExpr
	X, Start, Stop, Step Expr // Stop, not End — End() is Node's span-end method
	HasStep              bool
}

func (s *SliceExpr) Children() []Node { return nonNil(s.X, s.Start, s.Stop, s.Step) }
func (s *SliceExpr) String() string   { return "SliceExpr" }

type LambdaExpr struct {
	baseExpr
	Params []string
	Body   Expr
}

func (l *LambdaExpr) Children() []Node { return []Node{l.Body} }
func (l *LambdaExpr) String() string   { return "LambdaExpr(" + strings.Join(l.Params, ",") + ")" }

type PositionalExpr struct {
	baseExpr
	Index string
}

func (p *PositionalExpr) Children() []Node { return nil }
func (p *PositionalExpr) String() string   { return "PositionalExpr(#" + p.Index + ")" }

type DefaultExpr struct{ baseExpr }

func (d *DefaultExpr) Children() []Node { return nil }
func (d *DefaultExpr) String() string   { return "DefaultExpr" }

type ListComprehensionExpr struct {
	baseExpr
	Expr   Expr
	Vars   []string
	Source Expr
	Filter Expr
}

func (l *ListComprehensionExpr) Children() []Node {
	return nonNil(l.Expr, l.Source, l.Filter)
}
func (l *ListComprehensionExpr) String() string {
	return "ListComprehensionExpr(" + strings.Join(l.Vars, ",") + ")"
}

type GroupingExpr struct {
	baseExpr
	Args []Expr
}

func (g *GroupingExpr) Children() []Node {
	children := make([]Node, len(g.Args))
	for i, e := range g.Args {
		children[i] = e
	}
	return children
}
func (g *GroupingExpr) String() string { return "GroupingExpr" }

// --- ORDER BY (shared by ResultModifiers, window frames, WITHIN GROUP) ---

type OrderByItem struct {
	baseExpr
	X          Expr
	Desc       bool
	HasNulls   bool
	NullsFirst bool
}

func (o *OrderByItem) Children() []Node { return []Node{o.X} }
func (o *OrderByItem) String() string {
	return fmt.Sprintf("OrderByItem(desc=%v,nulls=%v/%v)", o.Desc, o.HasNulls, o.NullsFirst)
}

type OrderByClause struct {
	baseExpr
	Items []*OrderByItem
	All   bool
}

func (o *OrderByClause) Children() []Node {
	children := make([]Node, len(o.Items))
	for i, it := range o.Items {
		children[i] = it
	}
	return children
}
func (o *OrderByClause) String() string { return "OrderByClause" }

// --- SELECT (this task's cut is intentionally narrow — see adapter_select.go) ---

type SelectItem struct {
	baseExpr
	Expr  Expr
	Alias string
}

func (s *SelectItem) Children() []Node { return []Node{s.Expr} }
func (s *SelectItem) String() string   { return "SelectItem(" + s.Alias + ")" }

// TableRef is implemented by every FROM-clause table reference. Only
// *BaseTableRef exists as of this task — Task 10 adds the rest (joins,
// subqueries, table functions, VALUES, parenthesized refs).
type TableRef interface {
	Node
	isTableRef()
}

type baseTableRefNode struct{ baseExpr }

func (baseTableRefNode) isTableRef() {}

type BaseTableRef struct {
	baseTableRefNode
	Name  []string
	Alias string
}

func (b *BaseTableRef) Children() []Node { return nil }
func (b *BaseTableRef) String() string {
	return fmt.Sprintf("BaseTableRef(%s,alias=%s)", strings.Join(b.Name, "."), b.Alias)
}

type FromClause struct {
	baseExpr
	Refs []TableRef
}

func (f *FromClause) Children() []Node {
	children := make([]Node, len(f.Refs))
	for i, r := range f.Refs {
		children[i] = r
	}
	return children
}
func (f *FromClause) String() string { return "FromClause" }

// SelectStatement is the root AST type ParseString (Task 8) returns. Fields
// this task's adapter never populates (GroupBy, Having, etc.) stay nil —
// Children() skips them via nonNil, so a fixture dump never shows an empty
// placeholder for a clause that wasn't present in the source.
type SelectStatement struct {
	baseExpr
	With     *WithClause
	Columns  []*SelectItem
	Distinct *DistinctClause
	From     *FromClause
	Where    Expr
	GroupBy  *GroupByClause
	Having   Expr
	Windows  []*WindowDef
	Qualify  Expr
	OrderBy  *OrderByClause
	Limit    *LimitClause

	// Set when this node is one UNION/INTERSECT/EXCEPT combination instead
	// of a simple select — see this task's Interfaces note above.
	SetOp     SetOp
	SetAll    bool
	SetByName bool
	SetLeft   *SelectStatement
	SetRight  *SelectStatement

	// Set only for a VALUES (...), (...) statement.
	Values [][]Expr
}

func (s *SelectStatement) Children() []Node {
	children := make([]Node, len(s.Columns))
	for i, c := range s.Columns {
		children[i] = c
	}
	// All of these are concrete *T or Expr fields — nil-check each on its
	// own static type, never through the generic nonNil helper (see its
	// doc comment on why that would be unsafe for concrete pointers).
	if s.With != nil {
		children = append(children, s.With)
	}
	if s.Distinct != nil {
		children = append(children, s.Distinct)
	}
	if s.From != nil {
		children = append(children, s.From)
	}
	if s.Where != nil {
		children = append(children, s.Where)
	}
	if s.GroupBy != nil {
		children = append(children, s.GroupBy)
	}
	if s.Having != nil {
		children = append(children, s.Having)
	}
	for _, w := range s.Windows {
		children = append(children, w)
	}
	if s.Qualify != nil {
		children = append(children, s.Qualify)
	}
	if s.OrderBy != nil {
		children = append(children, s.OrderBy)
	}
	if s.Limit != nil {
		children = append(children, s.Limit)
	}
	if s.SetLeft != nil {
		children = append(children, s.SetLeft)
	}
	if s.SetRight != nil {
		children = append(children, s.SetRight)
	}
	for _, row := range s.Values {
		for _, e := range row {
			children = append(children, e)
		}
	}
	return children
}
func (s *SelectStatement) String() string {
	if s.SetOp == SetOpNone {
		if s.Values != nil {
			return fmt.Sprintf("SelectStatement(VALUES,rows=%d)", len(s.Values))
		}
		return "SelectStatement"
	}
	return fmt.Sprintf("SelectStatement(%s,all=%v,byName=%v)", s.SetOp, s.SetAll, s.SetByName)
}

// --- GROUP BY / HAVING / DISTINCT / LIMIT (Task 9 additions to ast.go) ---

type GroupByItem interface {
	Node
	isGroupByItem()
}

type groupByItemNode struct{ baseExpr }

func (groupByItemNode) isGroupByItem() {}

type GroupByExprItem struct {
	groupByItemNode
	X Expr
}

func (g *GroupByExprItem) Children() []Node { return []Node{g.X} }
func (g *GroupByExprItem) String() string   { return "GroupByExprItem" }

type GroupByEmpty struct{ groupByItemNode }

func (g *GroupByEmpty) Children() []Node { return nil }
func (g *GroupByEmpty) String() string   { return "GroupByEmpty" }

type GroupByCube struct {
	groupByItemNode
	Items []Expr
}

func (g *GroupByCube) Children() []Node {
	children := make([]Node, len(g.Items))
	for i, e := range g.Items {
		children[i] = e
	}
	return children
}
func (g *GroupByCube) String() string { return "GroupByCube" }

type GroupByRollup struct {
	groupByItemNode
	Items []Expr
}

func (g *GroupByRollup) Children() []Node {
	children := make([]Node, len(g.Items))
	for i, e := range g.Items {
		children[i] = e
	}
	return children
}
func (g *GroupByRollup) String() string { return "GroupByRollup" }

type GroupingSets struct {
	groupByItemNode
	Sets []GroupByItem
}

func (g *GroupingSets) Children() []Node {
	children := make([]Node, len(g.Sets))
	for i, s := range g.Sets {
		children[i] = s
	}
	return children
}
func (g *GroupingSets) String() string { return "GroupingSets" }

type GroupByClause struct {
	baseExpr
	All   bool
	Items []GroupByItem
}

func (g *GroupByClause) Children() []Node {
	children := make([]Node, len(g.Items))
	for i, it := range g.Items {
		children[i] = it
	}
	return children
}
func (g *GroupByClause) String() string { return fmt.Sprintf("GroupByClause(all=%v)", g.All) }

type DistinctClause struct {
	baseExpr
	On []Expr
}

func (d *DistinctClause) Children() []Node {
	children := make([]Node, len(d.On))
	for i, e := range d.On {
		children[i] = e
	}
	return children
}
func (d *DistinctClause) String() string { return "DistinctClause" }

type LimitClause struct {
	baseExpr
	Limit   Expr
	All     bool
	Percent bool
	Offset  Expr
}

func (l *LimitClause) Children() []Node { return nonNil(l.Limit, l.Offset) }
func (l *LimitClause) String() string {
	return fmt.Sprintf("LimitClause(all=%v,percent=%v)", l.All, l.Percent)
}

type TableFunctionRef struct {
	baseTableRefNode
	Lateral        bool
	Name           []string
	Args           []Expr
	WithOrdinality bool
	Alias          string
}

func (t *TableFunctionRef) Children() []Node {
	children := make([]Node, len(t.Args))
	for i, e := range t.Args {
		children[i] = e
	}
	return children
}
func (t *TableFunctionRef) String() string {
	return fmt.Sprintf("TableFunctionRef(%s,alias=%s)", strings.Join(t.Name, "."), t.Alias)
}

type TableSubqueryRef struct {
	baseTableRefNode
	Lateral bool
	Select  *SelectStatement
	Alias   string
}

func (t *TableSubqueryRef) Children() []Node { return []Node{t.Select} }
func (t *TableSubqueryRef) String() string   { return "TableSubqueryRef(alias=" + t.Alias + ")" }

type ParensTableRef struct {
	baseTableRefNode
	Ref   TableRef
	Alias string
}

func (p *ParensTableRef) Children() []Node { return []Node{p.Ref} }
func (p *ParensTableRef) String() string   { return "ParensTableRef(alias=" + p.Alias + ")" }

type JoinRef struct {
	baseTableRefNode
	Left, Right TableRef
	Type        string
	On          Expr
	Using       []string
}

func (j *JoinRef) Children() []Node {
	children := []Node{j.Left, j.Right}
	if j.On != nil {
		children = append(children, j.On)
	}
	return children
}
func (j *JoinRef) String() string { return fmt.Sprintf("JoinRef(%s,using=%v)", j.Type, j.Using) }

type SetOp int

const (
	SetOpNone SetOp = iota
	SetOpUnion
	SetOpIntersect
	SetOpExcept
)

func (op SetOp) String() string {
	switch op {
	case SetOpUnion:
		return "UNION"
	case SetOpIntersect:
		return "INTERSECT"
	case SetOpExcept:
		return "EXCEPT"
	default:
		return "NONE"
	}
}

type CTE struct {
	baseExpr
	Name         string
	ColumnNames  []string
	Materialized *bool
	Select       *SelectStatement
}

func (c *CTE) Children() []Node { return []Node{c.Select} }
func (c *CTE) String() string   { return "CTE(" + c.Name + ")" }

type WithClause struct {
	baseExpr
	Recursive bool
	CTEs      []*CTE
}

func (w *WithClause) Children() []Node {
	children := make([]Node, len(w.CTEs))
	for i, c := range w.CTEs {
		children[i] = c
	}
	return children
}
func (w *WithClause) String() string { return fmt.Sprintf("WithClause(recursive=%v)", w.Recursive) }

type WindowDef struct {
	baseExpr
	Name string
	Spec *WindowSpec
}

func (w *WindowDef) Children() []Node { return []Node{w.Spec} }
func (w *WindowDef) String() string   { return "WindowDef(" + w.Name + ")" }

type PivotColumn struct {
	baseExpr
	Header Expr
	In     []string
}

func (p *PivotColumn) Children() []Node { return []Node{p.Header} }
func (p *PivotColumn) String() string   { return fmt.Sprintf("PivotColumn(in=%v)", p.In) }

type PivotRef struct {
	baseTableRefNode
	Source  TableRef
	Columns []*SelectItem
	For     []*PivotColumn
	GroupBy []string
	Alias   string
}

func (p *PivotRef) Children() []Node {
	children := []Node{p.Source}
	for _, c := range p.Columns {
		children = append(children, c)
	}
	for _, f := range p.For {
		children = append(children, f)
	}
	return children
}
func (p *PivotRef) String() string {
	return fmt.Sprintf("PivotRef(groupBy=%v,alias=%s)", p.GroupBy, p.Alias)
}

type UnpivotRef struct {
	baseTableRefNode
	Source       TableRef
	IncludeNulls bool
	ExcludeNulls bool
	Header       []string // the new VALUE column name(s) — "amount" in "UNPIVOT (amount FOR quarter IN (...))"
	// Names holds the new NAME column identifier for each "FOR name IN
	// (...)" group ("quarter" above) — one entry per UnpivotValueList
	// match. Columns is every group's target columns flattened together
	// (a simplification: with more than one group this loses which
	// columns belong to which Names entry — fine for the common single-
	// group case this package's fixtures cover; split Columns into
	// per-group slices if a real multi-group query ever needs it).
	Names   []string
	Columns []*SelectItem
	Alias   string
}

func (u *UnpivotRef) Children() []Node {
	children := []Node{u.Source}
	for _, c := range u.Columns {
		children = append(children, c)
	}
	return children
}
func (u *UnpivotRef) String() string {
	return fmt.Sprintf("UnpivotRef(header=%v,names=%v,alias=%s)", u.Header, u.Names, u.Alias)
}
