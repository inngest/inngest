// Package parser edits text proto files, applies standard formatting
// and preserves comments.
//
// To disable a specific file from getting formatted, add '# txtpbfmt: disable'
// at the top of the file.
package parser

import (
	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/config"
	"github.com/protocolbuffers/txtpbfmt/impl"
	"github.com/protocolbuffers/txtpbfmt/printer"
	"github.com/protocolbuffers/txtpbfmt/sort"
)

// Config can be used to pass additional config parameters to the formatter at
// the time of the API call.
type Config = config.Config

// RootName contains a constant that can be used to identify the root of all Nodes.
const RootName = config.RootName

// UnsortedFieldsError will be returned by ParseWithConfig if
// Config.RequireFieldSortOrderToMatchAllFieldsInNode is set, and an unrecognized field is found
// while parsing.
type UnsortedFieldsError = sort.UnsortedFieldsError

// Format formats a text proto file preserving comments.
func Format(in []byte) ([]byte, error) {
	return printer.Format(in)
}

// FormatWithConfig functions similar to format, but allows the user to pass in
// additional configuration options.
func FormatWithConfig(in []byte, c config.Config) ([]byte, error) {
	return printer.FormatWithConfig(in, c)
}

// Parse returns a tree representation of a textproto file.
func Parse(in []byte) ([]*ast.Node, error) {
	return impl.Parse(in)
}

// ParseWithConfig functions similar to Parse, but allows the user to pass in
// additional configuration options.
func ParseWithConfig(in []byte, c config.Config) ([]*ast.Node, error) {
	return impl.ParseWithConfig(in, c)
}

// DebugFormat returns a textual representation of the specified nodes for
// consumption by humans when debugging (e.g. in test failures). No guarantees
// are made about the specific output.
func DebugFormat(nodes []*ast.Node, depth int) string {
	return printer.Debug(nodes, depth)
}

// Pretty formats the nodes at the given indentation depth (0 = top-level).
func Pretty(nodes []*ast.Node, depth int) string {
	return string(printer.FormatNodesWithDepth(nodes, depth))
}

// PrettyBytes returns formatted nodes at the given indentation depth (0 = top-level) as bytes.
func PrettyBytes(nodes []*ast.Node, depth int) []byte {
	return printer.FormatNodesWithDepth(nodes, depth)
}
