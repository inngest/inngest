// Package ast provides data structure representing textproto syntax tree.
package ast

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Position describes a position of a token in the input.
// Both byte-based and line/column-based positions are maintained
// because different downstream consumers need different formats
// and we don't want to keep the entire input in memory to be able
// to convert between the two.
// Fields Byte, Line and Column should be interpreted as
// ByteRange.start_byte, TextRange.start_line, and TextRange.start_column
// of devtools.api.Range proto.
type Position struct {
	Byte   uint32
	Line   int32
	Column int32
}

// Node represents a field with a value in a proto message, or a comment unattached to a field.
type Node struct {
	// Start describes the start position of the node.
	// For nodes that span entire lines, this is the first character
	// of the first line attributed to the node; possible a whitespace if the node is indented.
	// For nodes that are members of one-line message literals,
	// this is the first non-whitespace character encountered.
	Start Position
	// Lines of comments appearing before the field.
	// Each non-empty line starts with a # and does not contain the trailing newline.
	PreComments []string
	// Name of proto field (eg 'presubmit'). Will be an empty string for comment-only
	// nodes and unqualified messages, e.g.
	//     { name: "first_msg" }
	//     { name: "second_msg" }
	Name string
	// Values, for nodes that don't have children.
	Values []*Value
	// Children for nodes that have children.
	Children []*Node
	// Whether or not this node was deleted by edits.
	Deleted bool
	// Should the colon after the field name be omitted?
	// (e.g. "presubmit: {" vs "presubmit {")
	SkipColon bool
	// Whether or not all children are in the same line.
	// (eg "base { id: "id" }")
	ChildrenSameLine bool
	// Comment in the same line as the "}".
	ClosingBraceComment string
	// End holds the position suitable for inserting new items.
	// For multi-line nodes, this is the first character on the line with the closing brace.
	// For single-line nodes, this is the first character after the last item (usually a space).
	// For non-message nodes, this is Position zero value.
	End Position
	// Keep values in list (e.g "list: [1, 2]").
	ValuesAsList bool
	// Keep children in list (e.g "list: [ { value: 1 }, { value: 2 } ]").
	ChildrenAsList bool
	// Lines of comments appearing after last value inside list.
	// Each non-empty line starts with a # and does not contain the trailing newline.
	// e.g
	// field: [
	//   value
	//   # Comment
	// ]
	PostValuesComments []string
	// Whether the braces used for the children of this node are curly braces or angle brackets.
	IsAngleBracket bool
	// If this is not empty, it means that formatting was disabled for this node and it contains the
	// raw, unformatted node string.
	Raw string
	// Used when we want to break between the field name and values when a
	// single-line node exceeds the requested wrap column.
	PutSingleValueOnNextLine bool
	// Field number from proto definition (0 if unknown/not applicable).
	FieldNumber int32
}

// NodeLess is a sorting function that compares two *Nodes, possibly using the parent Node
// for context, returning whether a is less than b.
type NodeLess func(parent, a, b *Node, isWholeSlice bool) bool

// ChainNodeLess combines two NodeLess functions that returns the first comparison if values are
// not equal, else returns the second.
func ChainNodeLess(first, second NodeLess) NodeLess {
	if first == nil {
		return second
	}
	if second == nil {
		return first
	}
	return func(parent, a, b *Node, isWholeSlice bool) bool {
		if isALess := first(parent, a, b, isWholeSlice); isALess {
			return true
		}
		if isBLess := first(parent, b, a, isWholeSlice); isBLess {
			return false
		}
		return second(parent, a, b, isWholeSlice)
	}
}

type sortOptions struct {
	reverse bool
}

// A SortOption configures SortNodes.
type SortOption func(*sortOptions)

// ReverseOrdering controls whether to sort the Nodes in ascending or descending order. By default
// the Nodes are sorted in ascending order. By setting this option to true, the Nodes will be sorted
// in descending order.
//
// Default: false.
func ReverseOrdering(enabled bool) SortOption {
	return func(opts *sortOptions) {
		opts.reverse = enabled
	}
}

// SortNodes sorts nodes by the given less function.
func SortNodes(parent *Node, ns []*Node, less NodeLess, opts ...SortOption) {
	var options sortOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.reverse {
		sort.Stable(sort.Reverse(sortableNodes(parent, ns, less, true /* isWholeSlice */)))
	} else {
		sort.Stable(sortableNodes(parent, ns, less, true /* isWholeSlice */))
	}
	end := 0
	for begin := 0; begin < len(ns); begin = end {
		for end = begin + 1; end < len(ns) && ns[begin].Name == ns[end].Name; end++ {
		}
		if options.reverse {
			sort.Stable(sort.Reverse(sortableNodes(parent, ns[begin:end], less, false /* isWholeSlice */)))
		} else {
			sort.Stable(sortableNodes(parent, ns[begin:end], less, false /* isWholeSlice */))
		}
	}
}

// sortableNodes returns a sort.Interface that sorts nodes by the given less function.
func sortableNodes(parent *Node, ns []*Node, less NodeLess, isWholeSlice bool) sort.Interface {
	return sortable{parent, ns, less, isWholeSlice}
}

type sortable struct {
	parent       *Node
	ns           []*Node
	less         NodeLess
	isWholeSlice bool
}

func (s sortable) Len() int {
	return len(s.ns)
}

func (s sortable) Swap(i, j int) {
	s.ns[i], s.ns[j] = s.ns[j], s.ns[i]
}

func (s sortable) Less(i, j int) bool {
	if s.less == nil {
		return false
	}
	return s.less(s.parent, s.ns[i], s.ns[j], s.isWholeSlice)
}

// ByFieldName is a NodeLess function that orders nodes by their field name.
func ByFieldName(_, ni, nj *Node, isWholeSlice bool) bool {
	return ni.Name < nj.Name
}

func getFieldValueForByFieldValue(n *Node) *Value {
	if len(n.Values) != 1 {
		return nil
	}
	return n.Values[0]
}

// ByFieldValue returns a NodeLess function that orders adjacent scalar nodes
// with the same name by their scalar value. The values are passed through
// `projection` before sorting.
func ByFieldValue(projection func(string) string) NodeLess {
	return func(_, ni, nj *Node, isWholeSlice bool) bool {
		if isWholeSlice {
			return false
		}
		vi := getFieldValueForByFieldValue(ni)
		vj := getFieldValueForByFieldValue(nj)
		if vi == nil {
			return vj != nil
		}
		if vj == nil {
			return false
		}
		return projection(vi.Value) < projection(vj.Value)
	}
}

func getChildValueByFieldSubfield(field, subfield string, n *Node) *Value {
	if field != "" {
		if n.Name != field {
			return nil
		}
	}
	return n.getChildValue(subfield)
}

func getChildValueByFieldSubfieldPath(field string, subfieldPath []string, n *Node) *Value {
	if field != "" && n.Name != field {
		return nil
	}
	nodes := GetFromPath(n.Children, subfieldPath)
	if len(nodes) != 1 {
		return nil
	}
	if len(nodes[0].Values) != 1 {
		return nil
	}
	return nodes[0].Values[0]
}

// ByFieldSubfield returns a NodeLess function that orders adjacent message nodes with the given
// field name by the given subfield name value. If no field name is provided, it compares the
// subfields of any adjacent nodes with matching names.
func ByFieldSubfield(field, subfield string) NodeLess {
	return func(_, ni, nj *Node, isWholeSlice bool) bool {
		if isWholeSlice {
			return false
		}
		vi := getChildValueByFieldSubfield(field, subfield, ni)
		vj := getChildValueByFieldSubfield(field, subfield, nj)
		if vi == nil {
			return vj != nil
		}
		if vj == nil {
			return false
		}
		return vi.Value < vj.Value
	}
}

// ByFieldSubfieldPath returns a NodeLess function that orders adjacent message nodes with the given
// field name by the given subfield path value. If no field name is provided, it compares the
// subfields of any adjacent nodes with matching names. Values are passed
// through `projection` before sorting.
func ByFieldSubfieldPath(field string, subfieldPath []string, projection func(string) string) NodeLess {
	return func(_, ni, nj *Node, isWholeSlice bool) bool {
		if isWholeSlice {
			return false
		}
		vi := getChildValueByFieldSubfieldPath(field, subfieldPath, ni)
		vj := getChildValueByFieldSubfieldPath(field, subfieldPath, nj)
		if vi == nil {
			return vj != nil
		}
		if vj == nil {
			return false
		}
		return projection(vi.Value) < projection(vj.Value)
	}
}

// ByFieldNumber is a NodeLess function that orders fields by their field numbers.
// Field numbers are populated during parsing from descriptor information.
func ByFieldNumber(_, ni, nj *Node, isWholeSlice bool) bool {
	if !isWholeSlice {
		return false
	}

	numI, numJ := ni.FieldNumber, nj.FieldNumber

	// If both have field numbers, sort by field number
	if numI > 0 && numJ > 0 {
		return numI < numJ
	}

	// If only one has field number, prioritize it
	if numI > 0 && numJ == 0 {
		return true // ni has priority
	}
	if numI == 0 && numJ > 0 {
		return false // nj has priority
	}

	// If neither has field number, fall back to alphabetical order
	return ni.Name < nj.Name
}

// Formatter is a function that can format nodes in the AST.
type Formatter func([]*Node) error

var extraFormatters []Formatter

// RegisterFormatter registers an extra formatter that will be called after parsing.
func RegisterFormatter(f Formatter) {
	extraFormatters = append(extraFormatters, f)
}

// GetFormatters returns all registered formatters.
func GetFormatters() []Formatter {
	return extraFormatters
}

// getChildValue returns the Value of the child with the given field name,
// or nil if no single such child exists.
func (n *Node) getChildValue(field string) *Value {
	for _, c := range n.Children {
		if c.Name == field {
			if len(c.Values) != 1 {
				return nil
			}
			return c.Values[0]
		}
	}
	return nil
}

// IsCommentOnly returns true if this is a comment-only node. Even a node that
// only contains a blank line is considered a comment-only node in the sense
// that it has no proto content.
func (n *Node) IsCommentOnly() bool {
	return n.Name == "" && n.Children == nil
}

// IsBlankLine returns true if this is a blank line node.
func (n *Node) IsBlankLine() bool {
	return n.IsCommentOnly() && len(n.PreComments) == 1 && n.PreComments[0] == ""
}

type fixData struct {
	inline bool
}

// Fix fixes inconsistencies that may arise after manipulation.
//
// For example if a node is ChildrenSameLine but has non-inline children, or
// children with comments ChildrenSameLine will be set to false.
func (n *Node) Fix() {
	n.fix()
}

func isRealPosition(p Position) bool {
	return p.Byte != 0 || p.Line != 0 || p.Column != 0
}

func (n *Node) fix() fixData {
	isEmptyAndWasOriginallyInline := !(isRealPosition(n.Start) && isRealPosition(n.End) && n.End.Line-n.Start.Line > 0)
	d := fixData{
		// ChildrenSameLine may be false for cases with no children such as a
		// value `foo: false`. We don't want these to trigger expansion.
		inline: n.ChildrenSameLine || (len(n.Children) == 0 && isEmptyAndWasOriginallyInline && len(n.Values) <= 1),
	}

	for _, c := range n.Children {
		if c.Deleted {
			continue
		}

		cd := c.fix()
		if !cd.inline {
			d.inline = false
		}
	}

	for _, v := range n.Values {
		vd := v.fix()
		if !vd.inline {
			d.inline = false
		}
	}

	n.ChildrenSameLine = d.inline

	// textproto comments go until the end of the line, so we must force parents
	// to be multiline otherwise we will partially comment them out.
	if len(n.PreComments) > 0 || len(n.ClosingBraceComment) > 0 {
		d.inline = false
	}

	return d
}

// StringNode is a helper for constructing simple string nodes.
func StringNode(name, unquoted string) *Node {
	return &Node{Name: name, Values: []*Value{{Value: strconv.Quote(unquoted)}}}
}

// Value represents a field value in a proto message.
type Value struct {
	// Lines of comments appearing before the value (for multi-line strings).
	// Each non-empty line starts with a # and does not contain the trailing newline.
	PreComments []string
	// Node value (eg 'ERROR').
	Value string
	// Comment in the same line as the value.
	InlineComment string
}

func (v *Value) String() string {
	return fmt.Sprintf("{Value: %q, PreComments: %q, InlineComment: %q}", v.Value, strings.Join(v.PreComments, "\n"), v.InlineComment)
}

func (v *Value) fix() fixData {
	return fixData{
		inline: len(v.PreComments) == 0 && v.InlineComment == "",
	}
}

// SortValues sorts values by their value.
func SortValues(values []*Value) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Value < values[j].Value
	})
}

// SortValuesReverse reverse sorts values by their value.
func SortValuesReverse(values []*Value) {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Value > values[j].Value
	})
}

// GetFromPath returns all nodes with a given string path in the parse tree. See ast_test.go for examples.
func GetFromPath(nodes []*Node, path []string) []*Node {
	if len(path) == 0 {
		return nil
	}
	var res []*Node
	for _, node := range nodes {
		if node.Name == path[0] {
			if len(path) == 1 {
				res = append(res, node)
			} else {
				res = append(res, GetFromPath(node.Children, path[1:])...)
			}
		}
	}
	return res
}
