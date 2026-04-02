// Package config contains the configuration for the formatter.
package config

import (
	"github.com/protocolbuffers/txtpbfmt/logger"
)

// Config can be used to pass additional config parameters to the formatter at
// the time of the API call.
type Config struct {
	// Do not apply any reformatting to this file.
	Disable bool

	// Expand all children irrespective of the initial state.
	ExpandAllChildren bool

	// Skip colons whenever possible.
	SkipAllColons bool

	// Allow unnamed nodes everywhere.
	// Default is to allow only top-level nodes to be unnamed.
	AllowUnnamedNodesEverywhere bool

	// Sort fields by field name.
	SortFieldsByFieldName bool

	// Sort fields by field number from proto definition.
	SortFieldsByFieldNumber bool

	// Path to protobuf descriptor file (.desc).
	ProtoDescriptor string

	// Full message type name for field number lookup (required, e.g. google.protobuf.Any).
	MessageFullName string

	// Sort adjacent scalar fields of the same field name by their contents.
	SortRepeatedFieldsByContent bool

	// Sort adjacent message fields of the given field name by the contents of the given subfield path.
	// Format: either "field_name.subfield_name.subfield_name2...subfield_nameN" or just
	// "subfield_name" (applies to all field names).
	SortRepeatedFieldsBySubfield []string

	// Sort the Sort* fields by descending order instead of ascending order.
	ReverseSort bool

	// Sort content fields in a way that's suitable for DNS names. It splits the
	// value around '.' characters, reverses the substrings, and concatenates to
	// generate the sort key.
	DNSSortOrder bool

	// Map from Node.Name to the order of all fields within that node. See AddFieldSortOrder().
	FieldSortOrder map[string][]string

	// RequireFieldSortOrderToMatchAllFieldsInNode will cause parsing to fail if a node was added via
	// AddFieldSortOrder() but 1+ fields under that node in the textproto aren't specified in the
	// field order. This won't fail for nodes that don't have a field order specified at all. Use this
	// to strictly enforce that your field order config always orders ALL the fields, and you're
	// willing for new fields in the textproto to break parsing in order to enforce it.
	RequireFieldSortOrderToMatchAllFieldsInNode bool

	// Remove lines that have the same field name and scalar value as another.
	RemoveDuplicateValuesForRepeatedFields bool

	// Permit usage of Python-style """ or ''' delimited strings.
	AllowTripleQuotedStrings bool

	// Max columns for string field values. If zero, no string wrapping will occur.
	// Strings that may contain HTML tags will never be wrapped.
	WrapStringsAtColumn int

	// Whether strings that appear to contain HTML tags should be wrapped
	// (requires WrapStringsAtColumn to be set).
	WrapHTMLStrings bool

	// Wrap string field values after each newline.
	// Should not be used with other Wrap* options.
	WrapStringsAfterNewlines bool

	// Wrap strictly at the column instead of a word boundary.
	WrapStringsWithoutWordwrap bool

	// Whether angle brackets used instead of curly braces should be preserved
	// when outputting a formatted textproto.
	PreserveAngleBrackets bool

	// Use single quotes around strings that contain double but not single quotes.
	SmartQuotes bool

	// Use a short representation for repeated primitive fields (`x: 1 x: 2` vs `x: [1, 2]`). If this
	// field is true, all repeated primitive fields will use the short representation; otherwise, the
	// latter will be used only if it's being used in the input textproto.
	UseShortRepeatedPrimitiveFields bool

	// Logger enables logging when it is non-nil.
	// If the log messages aren't going to be useful, it's best to leave Logger
	// set to nil, as otherwise log messages will be constructed.
	Logger logger.Logger
}

// Infof is used for informative messages, for testing or debugging.
func (c *Config) Infof(format string, args ...any) {
	if c.Logger != nil {
		c.Logger.Infof(format, args...)
	}
}

// InfoLevel returns true if the logger is set to non-nil.
func (c *Config) InfoLevel() bool {
	return c.Logger != nil
}

// RootName contains a constant that can be used to identify the root of all Nodes.
const RootName = "__ROOT__"

// AddFieldSortOrder adds a config rule for the given Node.Name, so that all contained field names
// are output in the provided order. To specify an order for top-level Nodes, use RootName as the
// nodeName.
func (c *Config) AddFieldSortOrder(nodeName string, fieldOrder ...string) {
	if c.FieldSortOrder == nil {
		c.FieldSortOrder = make(map[string][]string)
	}
	c.FieldSortOrder[nodeName] = fieldOrder
}
