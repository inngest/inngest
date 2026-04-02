// Package sort provides functions for sorting nodes and values.
package sort

import (
	"fmt"
	"math"
	"strings"

	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/config"
)

// UnsortedFieldsError will be returned by ParseWithConfig if
// Config.RequireFieldSortOrderToMatchAllFieldsInNode is set, and an unrecognized field is found
// while parsing.
type UnsortedFieldsError struct {
	UnsortedFields []unsortedField
}

// unsortedField records details about a single unsorted field.
type unsortedField struct {
	FieldName       string
	Line            int32
	ParentFieldName string
}

func (e *UnsortedFieldsError) Error() string {
	var errs []string
	for _, us := range e.UnsortedFields {
		errs = append(errs, fmt.Sprintf("  line: %d, parent field: %q, unsorted field: %q", us.Line, us.ParentFieldName, us.FieldName))
	}
	return fmt.Sprintf("fields parsed that were not specified in the parser.AddFieldSortOrder() call:\n%s", strings.Join(errs, "\n"))
}

func identityProjection(s string) string {
	return s
}

func dnsProjection(s string) string {
	parts := strings.Split(s, ".")
	// Reverse `parts`.
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".")
}

// nodeSortFunction sorts the given nodes, using the parent node as context. parent can be nil.
type nodeSortFunction func(parent *ast.Node, nodes []*ast.Node) error

// nodeFilterFunction filters the given nodes.
type nodeFilterFunction func(nodes []*ast.Node)

// valuesSortFunction sorts the given values.
type valuesSortFunction func(values []*ast.Value)

// Process sorts and filters the given nodes.
func Process(parent *ast.Node, nodes []*ast.Node, c config.Config) error {
	return process(parent, nodes, nodeSortFunctionConfig(c), nodeFilterFunctionConfig(c), valuesSortFunctionConfig(c), c)
}

// process sorts and filters the given nodes.
func process(parent *ast.Node, nodes []*ast.Node, sortFunction nodeSortFunction, filterFunction nodeFilterFunction, valuesSortFunction valuesSortFunction, c config.Config) error {
	if len(nodes) == 0 {
		return nil
	}
	if filterFunction != nil {
		filterFunction(nodes)
	}
	for _, nd := range nodes {
		err := process(nd, nd.Children, sortFunction, filterFunction, valuesSortFunction, c)
		if err != nil {
			return err
		}
		if valuesSortFunction != nil && nd.ValuesAsList {
			valuesSortFunction(nd.Values)
		}
	}
	if sortFunction != nil {
		if err := sortFunction(parent, nodes); err != nil {
			return err
		}
	}
	if c.UseShortRepeatedPrimitiveFields {
		groupRepeatedPrimitiveFields(nodes)
	}
	return nil
}

func isPrimitive(n *ast.Node) bool {
	return len(n.Children) == 0 && len(n.Values) == 1
}

func groupRepeatedPrimitiveFields(nodes []*ast.Node) {
	for i := 0; i < len(nodes); {
		node := nodes[i]
		if node.Deleted || !isPrimitive(node) {
			i++
			continue
		}
		j := i + 1
		for ; j < len(nodes); j++ {
			if nodes[j].Deleted || !isPrimitive(nodes[j]) || nodes[j].Name != node.Name || len(nodes[j].PreComments) > 0 || len(nodes[j].PostValuesComments) > 0 {
				break
			}
		}
		if j > i+1 {
			// Found group of repeated primitive fields: nodes[i...j-1]
			node.ValuesAsList = true
			node.ChildrenSameLine = true
			for k := i + 1; k < j; k++ {
				node.Values = append(node.Values, nodes[k].Values...)
				nodes[k].Deleted = true
			}
		}
		i = j
	}
}

// removeDuplicates marks duplicate key:value pairs from nodes as Deleted.
func removeDuplicates(nodes []*ast.Node) {
	type nameAndValue struct {
		name, value string
	}
	seen := make(map[nameAndValue]bool)
	for _, nd := range nodes {
		if len(nd.Values) == 1 {
			key := nameAndValue{nd.Name, nd.Values[0].Value}
			if _, value := seen[key]; value {
				// Name-Value pair found in the same nesting level, deleting.
				nd.Deleted = true
			} else {
				seen[key] = true
			}
		}
	}
}

// unsortedFieldCollector collects UnsortedFields during parsing.
type unsortedFieldCollector struct {
	fields map[string]unsortedField
}

// newUnsortedFieldCollector returns a new UnsortedFieldCollector.
func newUnsortedFieldCollector() *unsortedFieldCollector {
	return &unsortedFieldCollector{
		fields: make(map[string]unsortedField),
	}
}

// unsortedFieldCollectorFunc collects UnsortedFields during parsing.
type unsortedFieldCollectorFunc func(name string, line int32, parent string)

// collect collects the unsorted field.
func (ufc *unsortedFieldCollector) collect(name string, line int32, parent string) {
	ufc.fields[name] = unsortedField{name, line, parent}
}

// asError returns an error if any unsorted fields were collected.
func (ufc *unsortedFieldCollector) asError() error {
	if len(ufc.fields) == 0 {
		return nil
	}
	var fields []unsortedField
	for _, f := range ufc.fields {
		fields = append(fields, f)
	}
	return &UnsortedFieldsError{fields}
}

// nodeSortFunctionConfig returns a function that sorts nodes based on the config.
func nodeSortFunctionConfig(c config.Config) nodeSortFunction {
	var sorter ast.NodeLess = nil
	unsortedFieldCollector := newUnsortedFieldCollector()
	for name, fieldOrder := range c.FieldSortOrder {
		sorter = ast.ChainNodeLess(sorter, byFieldOrder(name, fieldOrder, unsortedFieldCollector.collect))
	}
	if c.SortFieldsByFieldName {
		sorter = ast.ChainNodeLess(sorter, ast.ByFieldName)
	}
	if c.SortFieldsByFieldNumber {
		sorter = ast.ChainNodeLess(sorter, ast.ByFieldNumber)
	}
	projection := identityProjection
	if c.DNSSortOrder {
		projection = dnsProjection
	}
	if c.SortRepeatedFieldsByContent {
		sorter = ast.ChainNodeLess(sorter, ast.ByFieldValue(projection))
	}
	for _, sf := range c.SortRepeatedFieldsBySubfield {
		field, subfieldPath := parseSubfieldSpec(sf)
		if len(subfieldPath) > 0 {
			sorter = ast.ChainNodeLess(sorter, ast.ByFieldSubfieldPath(field, subfieldPath,
				projection))
		}
	}
	if sorter != nil {
		return func(parent *ast.Node, ns []*ast.Node) error {
			ast.SortNodes(parent, ns, sorter, ast.ReverseOrdering(c.ReverseSort))
			if c.RequireFieldSortOrderToMatchAllFieldsInNode {
				return unsortedFieldCollector.asError()
			}
			return nil
		}
	}
	return nil
}

// Returns the field and subfield path parts of spec "{field}.{subfield1}.{subfield2}...".
// Spec without a dot is considered to be "{subfield}".
func parseSubfieldSpec(subfieldSpec string) (field string, subfieldPath []string) {
	parts := strings.Split(subfieldSpec, ".")
	if len(parts) == 1 {
		return "", parts
	}
	return parts[0], parts[1:]
}

// nodeFilterFunctionConfig returns a function that filters nodes based on the config.
func nodeFilterFunctionConfig(c config.Config) nodeFilterFunction {
	if c.RemoveDuplicateValuesForRepeatedFields {
		return removeDuplicates
	}
	return nil
}

// valuesSortFunctionConfig returns a function that sorts values based on the config.
func valuesSortFunctionConfig(c config.Config) valuesSortFunction {
	if c.SortRepeatedFieldsByContent {
		if c.ReverseSort {
			return ast.SortValuesReverse
		}
		return ast.SortValues
	}
	return nil
}

func getNodePriorityForByFieldOrder(parent, node *ast.Node, name string, priorities map[string]int, unsortedCollector unsortedFieldCollectorFunc) *int {
	if parent != nil && parent.Name != name {
		return nil
	}
	if parent == nil && name != config.RootName {
		return nil
	}
	// CommentOnly nodes don't set priority below, and default to MaxInt, which keeps them at the bottom
	prio := math.MaxInt

	// Unknown fields will get the int nil value of 0 from the order map, and bubble to the top.
	if !node.IsCommentOnly() {
		var ok bool
		prio, ok = priorities[node.Name]
		if !ok {
			parentName := config.RootName
			if parent != nil {
				parentName = parent.Name
			}
			unsortedCollector(node.Name, node.Start.Line, parentName)
		}
	}
	return &prio
}

// byFieldOrder returns a NodeLess function that orders fields within a node named name
// by the order specified in fieldOrder. Nodes sorted but not specified by the field order
// are bubbled to the top and reported to unsortedCollector.
func byFieldOrder(name string, fieldOrder []string, unsortedCollector unsortedFieldCollectorFunc) ast.NodeLess {
	priorities := make(map[string]int)
	for i, fieldName := range fieldOrder {
		priorities[fieldName] = i + 1
	}
	return func(parent, ni, nj *ast.Node, isWholeSlice bool) bool {
		if !isWholeSlice {
			return false
		}
		vi := getNodePriorityForByFieldOrder(parent, ni, name, priorities, unsortedCollector)
		vj := getNodePriorityForByFieldOrder(parent, nj, name, priorities, unsortedCollector)
		if vi == nil {
			return vj != nil
		}
		if vj == nil {
			return false
		}
		return *vi < *vj
	}
}
