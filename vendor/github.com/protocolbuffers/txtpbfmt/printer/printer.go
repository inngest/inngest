// Package printer provides functions for printing formatted textproto messages.
package printer

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/config"
	"github.com/protocolbuffers/txtpbfmt/impl"
)

const indentSpaces = "  "

// Format formats a text proto file preserving comments.
func Format(in []byte) ([]byte, error) {
	return FormatWithConfig(in, config.Config{})
}

// FormatWithConfig functions similar to format, but allows the user to pass in
// additional configuration options.
func FormatWithConfig(in []byte, c config.Config) ([]byte, error) {
	if err := impl.AddMetaCommentsToConfig(in, &c); err != nil {
		return nil, err
	}
	if c.Disable {
		c.Infof("Ignored file with 'disable' comment.")
		return in, nil
	}
	nodes, err := impl.ParseWithMetaCommentConfig(in, c)
	if err != nil {
		return nil, err
	}
	return FormatNodes(nodes), nil
}

func removeDeleted(nodes []*ast.Node) []*ast.Node {
	var res []*ast.Node
	res = []*ast.Node{} // empty children is different from nil children
	// When removing a node which has an empty line before it, we should keep
	// the empty line before the next non-removed node to maintain the visual separation.
	// Consider the following:
	// foo: { name: "foo1" }
	// foo: { name: "foo2" }
	//
	// bar: { name: "bar1" }
	// bar: { name: "bar2" }
	//
	// If we decide to remove both foo2 and bar1, the result should still have one empty
	// line between foo1 and bar2.
	addEmptyLine := false
	for _, node := range nodes {
		if node.Deleted {
			if len(node.PreComments) > 0 && node.PreComments[0] == "" {
				addEmptyLine = true
			}
			continue
		}
		if len(node.Children) > 0 {
			node.Children = removeDeleted(node.Children)
		}
		if addEmptyLine && (len(node.PreComments) == 0 || node.PreComments[0] != "") {
			node.PreComments = append([]string{""}, node.PreComments...)
		}
		addEmptyLine = false
		res = append(res, node)
	}
	return res
}

// Debug returns a textual representation of the specified nodes for
// consumption by humans when debugging (e.g. in test failures). No guarantees
// are made about the specific output.
func Debug(nodes []*ast.Node, depth int) string {
	res := []string{""}
	prefix := strings.Repeat(".", depth)
	for _, nd := range nodes {
		var value string
		if nd.Deleted {
			res = append(res, "DELETED")
		}
		if nd.Children != nil { // Also for 0 children.
			value = fmt.Sprintf("children:%s", Debug(nd.Children, depth+1))
		} else {
			value = fmt.Sprintf("values: %v\n", nd.Values)
		}
		res = append(res,
			fmt.Sprintf("name: %q", nd.Name),
			fmt.Sprintf("PreComments: %q (len %d)", strings.Join(nd.PreComments, "\n"), len(nd.PreComments)),
			value)
	}
	return strings.Join(res, fmt.Sprintf("\n%s ", prefix))
}

// FormatNodes returns formatted nodes at the given indentation depth (0 = top-level) as bytes.
func FormatNodes(nodes []*ast.Node) []byte {
	return FormatNodesWithDepth(nodes, 0 /* depth */)
}

// FormatNodesWithDepth returns formatted nodes at the given indentation depth (0 = top-level) as bytes.
func FormatNodesWithDepth(nodes []*ast.Node, depth int) []byte {
	var result bytes.Buffer
	formatter{&result}.writeNodes(removeDeleted(nodes), depth, false /* isSameLine */, false /* asListItems */)
	return result.Bytes()
}

// stringWriter abstracts over bytes.Buffer and strings.Builder
type stringWriter interface {
	WriteString(s string) (int, error)
}

// formatter accumulates pretty-printed textproto contents into a stringWriter.
type formatter struct {
	stringWriter
}

func (f formatter) writeNode(nd *ast.Node, depth int, isSameLine, asListItems bool, index, lastNonCommentIndex int) {
	if len(nd.Raw) > 0 {
		f.WriteString(nd.Raw)
		return
	}
	indent := " "
	if !isSameLine {
		indent = strings.Repeat(indentSpaces, depth)
	}
	f.writePreComments(nd, indent, depth, index)

	if nd.IsCommentOnly() {
		// The comments have been printed already, no more work to do.
		return
	}
	f.WriteString(indent)
	// Node name may be empty in alternative-style textproto files, because they
	// contain a sequence of proto messages of the same type:
	//   { name: "first_msg" }
	//   { name: "second_msg" }
	// In all other cases, nd.Name is not empty and should be printed.
	if nd.Name != "" {
		f.writeNodeName(nd, indent)
	}

	f.writeNodeValues(nd, indent)

	f.writeNodeChildren(nd, depth, isSameLine)

	if asListItems && index < lastNonCommentIndex {
		f.WriteString(",")
	}

	f.writeNodeClosingBraceComment(nd)
}

func (f formatter) writePreComments(nd *ast.Node, indent string, depth int, index int) {
	for _, comment := range nd.PreComments {
		if len(comment) == 0 {
			if !(depth == 0 && index == 0) {
				f.WriteString("\n")
			}
			continue
		}
		f.WriteString(indent)
		f.WriteString(comment)
		f.WriteString("\n")
	}
}

func (f formatter) writeNodes(nodes []*ast.Node, depth int, isSameLine, asListItems bool) {
	lastNonCommentIndex := 0
	if asListItems {
		for i := len(nodes) - 1; i >= 0; i-- {
			if !nodes[i].IsCommentOnly() {
				lastNonCommentIndex = i
				break
			}
		}
	}

	for index, nd := range nodes {
		f.writeNode(nd, depth, isSameLine, asListItems, index, lastNonCommentIndex)
		if !isSameLine && len(nd.Raw) == 0 && !nd.IsCommentOnly() {
			f.WriteString("\n")
		}
	}
}

func (f formatter) writeNodeName(nd *ast.Node, indent string) {
	f.WriteString(nd.Name)
	if !nd.SkipColon {
		f.WriteString(":")
	}

	// The space after the name is required for one-liners and message fields:
	//   title: "there was a space here"
	//   metadata: { ... }
	// In other cases, there is a newline right after the colon, so no space required.
	if nd.Children != nil || (len(nd.Values) == 1 && len(nd.Values[0].PreComments) == 0) || nd.ValuesAsList {
		if nd.PutSingleValueOnNextLine {
			f.WriteString("\n" + indent + indentSpaces)
		} else {
			f.WriteString(" ")
		}
	}
}

func (f formatter) writeNodeValues(nd *ast.Node, indent string) {
	if nd.ValuesAsList { // For ValuesAsList option we will preserve even empty list  `field: []`
		f.writeValuesAsList(nd, nd.Values, indent+indentSpaces)
	} else if len(nd.Values) > 0 {
		f.writeValues(nd, nd.Values, indent+indentSpaces)
	}
}

func (f formatter) writeNodeChildren(nd *ast.Node, depth int, isSameLine bool) {
	if nd.Children != nil { // Also for 0 Children.
		if nd.ChildrenAsList {
			f.writeChildrenAsListItems(nd.Children, depth+1, isSameLine || nd.ChildrenSameLine)
		} else {
			f.writeChildren(nd.Children, depth+1, isSameLine || nd.ChildrenSameLine, nd.IsAngleBracket)
		}
	}
}

func (f formatter) writeNodeClosingBraceComment(nd *ast.Node) {
	if (nd.Children != nil || nd.ValuesAsList) && len(nd.ClosingBraceComment) > 0 {
		f.WriteString(indentSpaces)
		f.WriteString(nd.ClosingBraceComment)
	}
}

func (f formatter) writeValues(nd *ast.Node, vals []*ast.Value, indent string) {
	if len(vals) == 0 {
		// This should never happen: formatValues can be called only if there are some values.
		return
	}
	sep := "\n" + indent
	if len(vals) == 1 && len(vals[0].PreComments) == 0 {
		sep = ""
	}
	for _, v := range vals {
		f.WriteString(sep)
		for _, comment := range v.PreComments {
			f.WriteString(comment)
			f.WriteString(sep)
		}
		f.WriteString(v.Value)
		if len(v.InlineComment) > 0 {
			f.WriteString(indentSpaces)
			f.WriteString(v.InlineComment)
		}
	}
	for _, comment := range nd.PostValuesComments {
		f.WriteString(sep)
		f.WriteString(comment)
	}
}

func (f formatter) canWriteValuesAsListOnSameLine(nd *ast.Node, vals []*ast.Value) bool {
	if !nd.ChildrenSameLine || len(nd.PostValuesComments) > 0 {
		return false
	}
	// Parser found all children on a same line, but we need to check again.
	// It's possible that AST was modified after parsing.
	for _, val := range vals {
		if len(val.PreComments) > 0 || len(val.InlineComment) > 0 {
			return false
		}
	}
	return true
}

func (f formatter) writeValuesAsList(nd *ast.Node, vals []*ast.Value, indent string) {
	// Checks if it's possible to put whole list in a single line.
	sameLine := f.canWriteValuesAsListOnSameLine(nd, vals)
	sep := ""
	if !sameLine {
		sep = "\n" + indent
	}
	f.WriteString("[")

	for idx, v := range vals {
		for _, comment := range v.PreComments {
			f.WriteString(sep)
			f.WriteString(comment)
		}
		f.WriteString(sep)
		f.WriteString(v.Value)
		if idx < len(vals)-1 { // Don't put trailing comma that fails Python parser.
			f.WriteString(",")
			if sameLine {
				f.WriteString(" ")
			}
		}
		if len(v.InlineComment) > 0 {
			f.WriteString(indentSpaces)
			f.WriteString(v.InlineComment)
		}
	}
	for _, comment := range nd.PostValuesComments {
		f.WriteString(sep)
		f.WriteString(comment)
	}
	f.WriteString(strings.Replace(sep, indentSpaces, "", 1))
	f.WriteString("]")
}

// writeChildren writes the child nodes. The result always ends with a closing brace.
func (f formatter) writeChildren(children []*ast.Node, depth int, sameLine, isAngleBracket bool) {
	openBrace := "{"
	closeBrace := "}"
	if isAngleBracket {
		openBrace = "<"
		closeBrace = ">"
	}
	switch {
	case sameLine && len(children) == 0:
		f.WriteString(openBrace + closeBrace)
	case sameLine:
		f.WriteString(openBrace)
		f.writeNodes(children, depth, sameLine, false /* asListItems */)
		f.WriteString(" " + closeBrace)
	default:
		f.WriteString(openBrace + "\n")
		f.writeNodes(children, depth, sameLine, false /* asListItems */)
		f.WriteString(strings.Repeat(indentSpaces, depth-1))
		f.WriteString(closeBrace)
	}
}

// writeChildrenAsListItems writes the child nodes as list items.
func (f formatter) writeChildrenAsListItems(children []*ast.Node, depth int, sameLine bool) {
	openBrace := "["
	closeBrace := "]"
	switch {
	case sameLine && len(children) == 0:
		f.WriteString(openBrace + closeBrace)
	case sameLine:
		f.WriteString(openBrace)
		f.writeNodes(children, depth, sameLine, true /* asListItems */)
		f.WriteString(" " + closeBrace)
	default:
		f.WriteString(openBrace + "\n")
		f.writeNodes(children, depth, sameLine, true /* asListItems */)
		f.WriteString(strings.Repeat(indentSpaces, depth-1))
		f.WriteString(closeBrace)
	}
}
