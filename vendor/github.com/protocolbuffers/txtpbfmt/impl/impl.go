// Package impl edits text proto files, applies standard formatting
// and preserves comments.
package impl

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/config"
	"github.com/protocolbuffers/txtpbfmt/descriptor"
	"github.com/protocolbuffers/txtpbfmt/quote"
	"github.com/protocolbuffers/txtpbfmt/sort"
	"github.com/protocolbuffers/txtpbfmt/wrap"
)

type parser struct {
	in     []byte
	index  int
	length int
	// Maps the index of '{' characters on 'in' that have the matching '}' on
	// the same line to 'true'.
	bracketSameLine map[int]bool
	config          config.Config
	line, column    int // current position, 1-based.
}

var defConfig = config.Config{}

type bracketState struct {
	insideComment            bool
	insideString             bool
	insideTemplate           bool
	insideTripleQuotedString bool
	stringDelimiter          string
	isEscapedChar            bool
}

func (s *bracketState) processChar(c byte, i int, in []byte, allowTripleQuotedStrings bool) {
	switch c {
	case '#':
		if !s.insideString {
			s.insideComment = true
		}
	case '%':
		if !s.insideComment && !s.insideString {
			s.insideTemplate = !s.insideTemplate
		}
	case '"', '\'':
		if s.insideComment {
			return
		}
		s.handleQuotes(c, i, in, allowTripleQuotedStrings)
	}
}

func (s *bracketState) handleQuotes(c byte, i int, in []byte, allowTripleQuotedStrings bool) {
	delim := string(c)
	tripleQuoted := false
	if allowTripleQuotedStrings && i+3 <= len(in) {
		triple := string(in[i : i+3])
		if triple == `"""` || triple == `'''` {
			delim = triple
			tripleQuoted = true
		}
	}

	if s.insideString {
		if s.stringDelimiter == delim && (s.insideTripleQuotedString || !s.isEscapedChar) {
			s.insideString = false
			s.insideTripleQuotedString = false
		}
	} else {
		s.insideString = true
		s.insideTripleQuotedString = tripleQuoted
		s.stringDelimiter = delim
	}
}

// Return the byte-positions of each bracket which has the corresponding close on the
// same line as a set.
func sameLineBrackets(in []byte, allowTripleQuotedStrings bool) (map[int]bool, error) {
	line := 1
	type bracket struct {
		index int
		line  int
	}
	var open []bracket // Stack.
	res := map[int]bool{}
	state := bracketState{}
	for i, c := range in {
		state.processChar(c, i, in, allowTripleQuotedStrings)
		switch c {
		case '\n':
			line++
			state.insideComment = false
		case '{', '<':
			if state.insideComment || state.insideString || state.insideTemplate {
				continue
			}
			open = append(open, bracket{index: i, line: line})
		case '}', '>':
			if state.insideComment || state.insideString || state.insideTemplate {
				continue
			}
			if len(open) == 0 {
				return nil, fmt.Errorf("too many '}' or '>' at line %d, index %d", line, i)
			}
			last := len(open) - 1
			br := open[last]
			open = open[:last]
			if br.line == line {
				res[br.index] = true
			}
		}
		if state.isEscapedChar {
			state.isEscapedChar = false
		} else if c == '\\' && state.insideString && !state.insideTripleQuotedString {
			state.isEscapedChar = true
		}

	}
	if state.insideString {
		return nil, fmt.Errorf("unterminated string literal")
	}
	return res, nil
}

var (
	spaceSeparators = []byte(" \t\n\r")
	valueSeparators = []byte(" \t\n\r{}:,[]<>;#")
)

// Parse returns a tree representation of a textproto file.
func Parse(in []byte) ([]*ast.Node, error) {
	return ParseWithConfig(in, defConfig)
}

// ParseWithConfig functions similar to Parse, but allows the user to pass in
// additional configuration options.
func ParseWithConfig(in []byte, c config.Config) ([]*ast.Node, error) {
	if err := AddMetaCommentsToConfig(in, &c); err != nil {
		return nil, err
	}
	return ParseWithMetaCommentConfig(in, c)
}

// ParseWithMetaCommentConfig parses in textproto with MetaComments already added to configuration.
func ParseWithMetaCommentConfig(in []byte, c config.Config) ([]*ast.Node, error) {
	p, err := newParser(in, c)
	if err != nil {
		return nil, err
	}

	// Load descriptor if field number sorting is enabled
	var rootDesc protoreflect.MessageDescriptor
	if c.SortFieldsByFieldNumber {
		if c.ProtoDescriptor == "" {
			return nil, fmt.Errorf("proto_descriptor is required when using sort_fields_by_field_number")
		}

		loader, err := descriptor.NewLoader(c.ProtoDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to create descriptor loader: %v", err)
		}

		// Get root message descriptor
		rootDesc, err = loader.GetRootMessageDescriptor(c.MessageFullName)
		if err != nil {
			return nil, fmt.Errorf("failed to get root message descriptor: %v", err)
		}
	}

	if p.config.InfoLevel() {
		p.config.Infof("p.in: %q", string(p.in))
		p.config.Infof("p.length: %v", p.length)
	}
	// Although unnamed nodes aren't strictly allowed, some formats represent a
	// list of protos as a list of unnamed top-level nodes.
	nodes, _, err := p.parse( /*isRoot=*/ true, rootDesc)
	if err != nil {
		return nil, err
	}
	if p.index < p.length {
		return nil, fmt.Errorf("parser didn't consume all input. Stopped at %s", p.errorContext())
	}
	for _, f := range ast.GetFormatters() {
		if err := f(nodes); err != nil {
			return nil, err
		}
	}
	if err := wrap.Strings(nodes, 0, c); err != nil {
		return nil, err
	}
	if err := sort.Process( /*parent=*/ nil, nodes, c); err != nil {
		return nil, err
	}
	return nodes, nil
}

// There are two types of MetaComment, one in the format of <key>=<val> and the other one doesn't
// have the equal sign. Currently there are only two MetaComments that are in the former format:
//
//	"sort_repeated_fields_by_subfield": If this appears multiple times, then they will all be added
//	to the config and the order is preserved.
//	"wrap_strings_at_column": The <val> is expected to be an integer. If it is not, then it will be
//	ignored. If this appears multiple times, only the last one saved.
func addToConfig(metaComment string, c *config.Config) error {
	// Test if a MetaComment is in the format of <key>=<val>.
	key, val, hasEqualSign := strings.Cut(metaComment, "=")
	switch key {
	case "allow_triple_quoted_strings":
		c.AllowTripleQuotedStrings = true
	case "allow_unnamed_nodes_everywhere":
		c.AllowUnnamedNodesEverywhere = true
	case "disable":
		c.Disable = true
	case "expand_all_children":
		c.ExpandAllChildren = true
	case "preserve_angle_brackets":
		c.PreserveAngleBrackets = true
	case "remove_duplicate_values_for_repeated_fields":
		c.RemoveDuplicateValuesForRepeatedFields = true
	case "skip_all_colons":
		c.SkipAllColons = true
	case "smartquotes":
		c.SmartQuotes = true
	case "sort_fields_by_field_name":
		c.SortFieldsByFieldName = true
	case "sort_repeated_fields_by_content":
		c.SortRepeatedFieldsByContent = true
	case "sort_repeated_fields_by_subfield":
		// Take all the subfields and the subfields in order as tie breakers.
		if !hasEqualSign {
			return fmt.Errorf("format should be %s=<string>, got: %s", key, metaComment)
		}
		c.SortRepeatedFieldsBySubfield = append(c.SortRepeatedFieldsBySubfield, val)
	case "reverse_sort":
		c.ReverseSort = true
	case "dns_sort_order":
		c.DNSSortOrder = true
	case "wrap_strings_at_column":
		// If multiple of this MetaComment exists in the file, take the last one.
		if !hasEqualSign {
			return fmt.Errorf("format should be %s=<int>, got: %s", key, metaComment)
		}
		i, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return fmt.Errorf("error parsing %s value %q (skipping): %v", key, val, err)
		}
		c.WrapStringsAtColumn = i
	case "wrap_html_strings":
		c.WrapHTMLStrings = true
	case "wrap_strings_after_newlines":
		c.WrapStringsAfterNewlines = true
	case "wrap_strings_without_wordwrap":
		c.WrapStringsWithoutWordwrap = true
	case "use_short_repeated_primitive_fields":
		c.UseShortRepeatedPrimitiveFields = true
	case "on": // This doesn't change the overall config.
	case "off": // This doesn't change the overall config.
	default:
		return fmt.Errorf("unrecognized MetaComment: %s", metaComment)
	}
	return nil
}

// AddMetaCommentsToConfig parses MetaComments and adds them to the configuration.
func AddMetaCommentsToConfig(in []byte, c *config.Config) error {
	scanner := bufio.NewScanner(bytes.NewReader(in))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		if line[0] != byte('#') {
			break // only process the leading comment block
		}

		// Look for comment lines in the format of "<key>:<value>", and process the lines with <key>
		// equals to "txtpbfmt". It's assumed that the MetaComments are given in the format of:
		// # txtpbfmt: <MetaComment 1>[, <MetaComment 2> ...]
		key, value, hasColon := strings.Cut(line[1:], ":") // Ignore the first '#'.
		if hasColon && strings.TrimSpace(key) == "txtpbfmt" {
			for _, s := range strings.Split(strings.TrimSpace(value), ",") {
				metaComment := strings.TrimSpace(s)
				if err := addToConfig(metaComment, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func newParser(in []byte, c config.Config) (*parser, error) {
	var bracketSameLine map[int]bool
	if c.ExpandAllChildren {
		bracketSameLine = map[int]bool{}
	} else {
		var err error
		if bracketSameLine, err = sameLineBrackets(in, c.AllowTripleQuotedStrings); err != nil {
			return nil, err
		}
	}
	if len(in) > 0 && in[len(in)-1] != '\n' {
		in = append(in, '\n')
	}
	parser := &parser{
		in:              in,
		index:           0,
		length:          len(in),
		bracketSameLine: bracketSameLine,
		config:          c,
		line:            1,
		column:          1,
	}
	return parser, nil
}

// getFieldNumber returns the field number for a given field name in the descriptor.
func getFieldNumber(desc protoreflect.MessageDescriptor, fieldName string) int32 {
	if desc == nil {
		return 0
	}

	field := desc.Fields().ByTextName(fieldName)
	if field == nil {
		return 0
	}
	return int32(field.Number())
}

// findChildDescriptor finds the descriptor for a nested message field.
func (p *parser) findChildDescriptor(desc protoreflect.MessageDescriptor, fieldName string) protoreflect.MessageDescriptor {
	if desc == nil {
		return nil
	}

	field := desc.Fields().ByTextName(fieldName)
	if field == nil {
		return nil
	}
	if field.Kind() == protoreflect.MessageKind {
		return field.Message()
	}
	return nil
}

func (p *parser) nextInputIs(b byte) bool {
	return p.index < p.length && p.in[p.index] == b
}

func (p *parser) consume(b byte) bool {
	if !p.nextInputIs(b) {
		return false
	}
	p.index++
	p.column++
	if b == '\n' {
		p.line++
		p.column = 1
	}
	return true
}

// consumeString consumes the given string s, which should not have any newlines.
func (p *parser) consumeString(s string) bool {
	if p.index+len(s) > p.length {
		return false
	}
	if string(p.in[p.index:p.index+len(s)]) != s {
		return false
	}
	p.index += len(s)
	p.column += len(s)
	return true
}

// loopDetector detects if the parser is in an infinite loop (ie failing to
// make progress).
type loopDetector struct {
	lastIndex int
	count     int
	parser    *parser
}

func (p *parser) getLoopDetector() *loopDetector {
	return &loopDetector{lastIndex: p.index, parser: p}
}

func (l *loopDetector) iter() error {
	if l.parser.index == l.lastIndex {
		l.count++
		if l.count < 2 {
			return nil
		}
		return fmt.Errorf("parser failed to make progress at %s", l.parser.errorContext())
	}
	l.lastIndex = l.parser.index
	l.count = 0
	return nil
}

func (p parser) errorContext() string {
	index := p.index
	if index >= p.length {
		index = p.length - 1
	}
	// Provide the surrounding input as context.
	lastContentIndex := index + 20
	if lastContentIndex >= p.length {
		lastContentIndex = p.length - 1
	}
	previousContentIndex := index - 20
	if previousContentIndex < 0 {
		previousContentIndex = 0
	}
	before := string(p.in[previousContentIndex:index])
	after := string(p.in[index:lastContentIndex])
	return fmt.Sprintf("index %v\nposition %+v\nbefore: %q\nafter: %q\nbefore+after: %q", index, p.position(), before, after, before+after)
}

func (p *parser) position() ast.Position {
	return ast.Position{
		Byte:   uint32(p.index),
		Line:   int32(p.line),
		Column: int32(p.column),
	}
}

// Modifies the parser by rewinding to the given position.
// A position can be snapshotted by using the `position()` function above.
func (p *parser) rollbackPosition(pos ast.Position) {
	p.index = int(pos.Byte)
	p.line = int(pos.Line)
	p.column = int(pos.Column)
}

func (p *parser) consumeOptionalSeparator() error {
	if p.index > 0 && !p.isBlankSep(p.index-1) {
		// If an unnamed field immediately follows non-whitespace, we require a separator character first (key_one:,:value_two instead of key_one::value_two)
		if p.consume(':') {
			return fmt.Errorf("parser encountered unexpected character ':' (should be whitespace, ',', or ';')")
		}
	}

	_ = p.consume(';') // Ignore optional ';'.
	_ = p.consume(',') // Ignore optional ','.

	return nil
}

// parse parses a text proto.
// It assumes the text to be either conformant with the standard text proto
// (i.e. passes proto.UnmarshalText() without error) or the alternative textproto
// format (sequence of messages, each of which passes proto.UnmarshalText()).
// endPos is the position of the first character on the first line
// after parsed nodes: that's the position to append more children.
func (p *parser) parse(isRoot bool, desc protoreflect.MessageDescriptor) (result []*ast.Node, endPos ast.Position, err error) {
	var res []*ast.Node
	res = []*ast.Node{} // empty children is different from nil children
	for ld := p.getLoopDetector(); p.index < p.length; {
		if err := ld.iter(); err != nil {
			return nil, ast.Position{}, err
		}

		// p.parse is often invoked with the index pointing at the newline character
		// after the previous item. We should still report that this item starts in
		// the next line.
		p.consume('\r')
		p.consume('\n')
		startPos := p.position()

		fmtDisabled, err := p.readFormatterDisabledBlock()
		if err != nil {
			return nil, startPos, err
		}
		if len(fmtDisabled) > 0 {
			res = append(res, &ast.Node{
				Start: startPos,
				Raw:   fmtDisabled,
			})
			continue
		}

		// Read PreComments.
		comments, blankLines := p.skipWhiteSpaceAndReadComments(true /* multiLine */)

		// Handle blank lines.
		if blankLines > 0 {
			if p.config.InfoLevel() {
				p.config.Infof("blankLines: %v", blankLines)
			}
			// Here we collapse the leading blank lines into one blank line.
			comments = append([]string{""}, comments...)
		}

		for p.nextInputIs('%') {
			comments = append(comments, p.readTemplate())
			c, _ := p.skipWhiteSpaceAndReadComments(false)
			comments = append(comments, c...)
		}

		if end, endPos, err := p.handleEndOfMessage(startPos, comments, &res); end {
			return res, endPos, err
		}

		nd := &ast.Node{
			Start:       startPos,
			PreComments: comments,
		}
		if p.config.InfoLevel() {
			p.config.Infof("PreComments: %q", strings.Join(nd.PreComments, "\n"))
		}

		// Skip white-space other than '\n', which is handled below.
		for p.consume(' ') || p.consume('\t') {
		}

		// Handle multiple comment blocks.
		// <example>
		// # comment block 1
		// # comment block 1
		//
		// # comment block 2
		// # comment block 2
		// </example>
		// Each block that ends on an empty line (instead of a field) gets its own
		// 'empty' node.
		if p.nextInputIs('\n') {
			res = append(res, nd)
			continue
		}

		// Handle end of file.
		if end, err := p.handleEndOfFile(nd, &res); end {
			if err != nil {
				return nil, ast.Position{}, err
			}
			break
		}

		if err := p.parseFieldName(nd, isRoot); err != nil {
			return nil, ast.Position{}, err
		}

		// Set field number from descriptor if available
		nd.FieldNumber = getFieldNumber(desc, nd.Name)

		// Skip separator.
		preCommentsBeforeColon, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)
		nd.SkipColon = !p.consume(':')
		previousPos := p.position()
		preCommentsAfterColon, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)

		if err := p.parseFieldValue(nd, desc, preCommentsBeforeColon, preCommentsAfterColon, previousPos); err != nil {
			return nil, ast.Position{}, err
		}

		if p.config.InfoLevel() && p.index < p.length {
			p.config.Infof("p.in[p.index]: %q", string(p.in[p.index]))
		}
		res = append(res, nd)
	}
	return res, p.position(), nil
}

func (p *parser) parseFieldValue(nd *ast.Node, desc protoreflect.MessageDescriptor, preCommentsBeforeColon, preCommentsAfterColon []string, previousPos ast.Position) error {
	if p.consume('{') || p.consume('<') {
		if err := p.parseMessage(nd, desc); err != nil {
			return err
		}
	} else if p.consume('[') {
		if err := p.parseList(nd, preCommentsBeforeColon, preCommentsAfterColon); err != nil {
			return err
		}
		if nd.ValuesAsList {
			return nil
		}
	} else {
		// Rewind comments.
		p.rollbackPosition(previousPos)
		// Handle Values.
		var err error
		nd.Values, err = p.readValues()
		if err != nil {
			return err
		}
		if err := p.consumeOptionalSeparator(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) handleEndOfFile(nd *ast.Node, res *[]*ast.Node) (bool, error) {
	if p.index >= p.length {
		nd.End = p.position()
		if len(nd.PreComments) > 0 {
			*res = append(*res, nd)
		}
		return true, nil
	}
	return false, nil
}

func (p *parser) handleEndOfMessage(startPos ast.Position, comments []string, res *[]*ast.Node) (bool, ast.Position, error) {
	if endPos := p.position(); p.consume('}') || p.consume('>') || p.consume(']') {
		// Handle comments after last child.

		if len(comments) > 0 {
			*res = append(*res, &ast.Node{Start: startPos, PreComments: comments})
		}

		// endPos points at the closing brace, but we should rather return the position
		// of the first character after the previous item. Therefore let's rewind a bit:
		for endPos.Byte > 0 && p.in[endPos.Byte-1] == ' ' {
			endPos.Byte--
			endPos.Column--
		}

		if err := p.consumeOptionalSeparator(); err != nil {
			return true, ast.Position{}, err
		}

		// Done parsing children.
		return true, endPos, nil
	}
	return false, ast.Position{}, nil
}

func (p *parser) parseFieldName(nd *ast.Node, isRoot bool) error {
	if p.consume('[') {
		// Read Name (of proto extension).
		nd.Name = fmt.Sprintf("[%s]", p.readExtension())
		_ = p.consume(']') // Ignore the ']'.
	} else {
		// Read Name.
		nd.Name = p.readFieldName()
		if nd.Name == "" && !isRoot && !p.config.AllowUnnamedNodesEverywhere {
			return fmt.Errorf("Failed to find a FieldName at %s", p.errorContext())
		}
	}
	if p.config.InfoLevel() {
		p.config.Infof("name: %q", nd.Name)
	}
	return nil
}

func (p *parser) parseMessage(nd *ast.Node, desc protoreflect.MessageDescriptor) error {
	if p.config.SkipAllColons {
		nd.SkipColon = true
	}
	nd.ChildrenSameLine = p.bracketSameLine[p.index-1]
	nd.IsAngleBracket = p.config.PreserveAngleBrackets && p.in[p.index-1] == '<'
	// Recursive call to parse child nodes.
	childDesc := p.findChildDescriptor(desc, nd.Name)
	nodes, lastPos, err := p.parse( /*isRoot=*/ false, childDesc)
	if err != nil {
		return err
	}
	nd.Children = nodes
	nd.End = lastPos

	nd.ClosingBraceComment = p.readInlineComment()
	return nil
}

func (p *parser) parseList(nd *ast.Node, preCommentsBeforeColon, preCommentsAfterColon []string) error {
	openBracketLine := p.line

	// Skip separator.
	preCommentsAfterListStart := p.readContinuousBlocksOfComments()

	var preComments []string
	preComments = append(preComments, preCommentsBeforeColon...)
	preComments = append(preComments, preCommentsAfterColon...)
	preComments = append(preComments, preCommentsAfterListStart...)

	if p.nextInputIs('{') {
		// Handle list of nodes.
		return p.parseListOfNodes(nd, preComments, openBracketLine)
	} else {
		// Handle list of values.
		return p.parseListOfValues(nd, preComments, openBracketLine)
	}
}

func (p *parser) parseListOfNodes(nd *ast.Node, preComments []string, openBracketLine int) error {
	nd.ChildrenAsList = true

	nodes, lastPos, err := p.parse( /*isRoot=*/ true, nil)
	if err != nil {
		return err
	}
	if len(nodes) > 0 {
		nodes[0].PreComments = preComments
	}

	nd.Children = nodes
	nd.End = lastPos
	nd.ClosingBraceComment = p.readInlineComment()
	nd.ChildrenSameLine = openBracketLine == p.line
	return nil
}

func (p *parser) parseListOfValues(nd *ast.Node, preComments []string, openBracketLine int) error {
	nd.ValuesAsList = true // We found values in list - keep it as list.

	for ld := p.getLoopDetector(); !p.consume(']') && p.index < p.length; {
		if err := ld.iter(); err != nil {
			return err
		}

		// Read each value in the list.
		vals, err := p.readValues()
		if err != nil {
			return err
		}
		if len(vals) != 1 {
			return fmt.Errorf("multiple-string value not supported (%v). Please add comma explicitly, see http://b/162070952", vals)
		}
		if len(preComments) > 0 {
			// If we read preComments before readValues(), they should go first,
			// but avoid copy overhead if there are none.
			vals[0].PreComments = append(preComments, vals[0].PreComments...)
		}

		// Skip separator.
		_, _ = p.skipWhiteSpaceAndReadComments(false /* multiLine */)
		if p.consume(',') {
			vals[0].InlineComment = p.readInlineComment()
		}

		nd.Values = append(nd.Values, vals...)

		preComments, _ = p.skipWhiteSpaceAndReadComments(true /* multiLine */)
	}
	nd.ChildrenSameLine = openBracketLine == p.line

	// Handle comments after last line (or for empty list)
	nd.PostValuesComments = preComments
	nd.ClosingBraceComment = p.readInlineComment()

	if err := p.consumeOptionalSeparator(); err != nil {
		return err
	}
	return nil
}

func (p *parser) readFieldName() string {
	i := p.index
	for ; i < p.length && !p.isValueSep(i); i++ {
	}
	return p.advance(i)
}

func (p *parser) readExtension() string {
	i := p.index
	for ; i < p.length && (p.isBlankSep(i) || !p.isValueSep(i)); i++ {
	}
	return removeBlanks(p.advance(i))
}

func removeBlanks(in string) string {
	s := []byte(in)
	for _, b := range spaceSeparators {
		s = bytes.Replace(s, []byte{b}, nil, -1)
	}
	return string(s)
}

func (p *parser) readContinuousBlocksOfComments() []string {
	var preComments []string
	for {
		comments, blankLines := p.skipWhiteSpaceAndReadComments(true)
		if len(comments) == 0 {
			break
		}
		if blankLines > 0 && len(preComments) > 0 {
			comments = append([]string{""}, comments...)
		}
		preComments = append(preComments, comments...)
	}

	return preComments
}

func (p *parser) consumeWhitespace() (int, error) {
	start := p.index
	for p.index < p.length && p.isBlankSep(p.index) {
		if p.consume('\n') || (p.consume('\r') && p.consume('\n')) {
			// Include up to one blank line before the 'off' directive.
			start = p.index - 1
		} else if p.consume(' ') || p.consume('\t') {
			// Do nothing. Side-effect is to advance p.index.
		} else {
			return 0, fmt.Errorf("unhandled isBlankSep at %s", p.errorContext())
		}
	}
	return start, nil
}

// Returns the exact text within the block flanked by "# txtpbfmt: off" and "# txtpbfmt: on".
// The 'off' directive must be on its own line, and it cannot be preceded by a comment line. Any
// preceding whitespace on this line and up to one blank line will be retained.
// The 'on' directive must followed by a line break. Only full nodes of a AST can be
// within this block. Partially disabled sections, like just the first line of a for loop without
// body or closing brace, are not supported. Value lists are not supported. No parsing happens
// within this block, and as parsing errors will be ignored, please exercise caution.
func (p *parser) readFormatterDisabledBlock() (string, error) {
	previousPos := p.position()
	start, err := p.consumeWhitespace()
	if err != nil {
		return "", err
	}
	if !p.consumeString("# txtpbfmt: off") {
		// Directive not found. Rollback to start.
		p.rollbackPosition(previousPos)
		return "", nil
	}
	if !p.consume('\n') {
		return "", fmt.Errorf("txtpbfmt off should be followed by newline at %s", p.errorContext())
	}
	for ; p.index < p.length; p.index++ {
		if p.consumeString("# txtpbfmt: on") {
			if !p.consume('\n') {
				return "", fmt.Errorf("txtpbfmt on should be followed by newline at %s", p.errorContext())
			}
			// Retain up to one blank line.
			p.consume('\n')
			return string(p.in[start:p.index]), nil
		}
	}
	// We reached the end of the file without finding the 'on' directive.
	p.rollbackPosition(previousPos)
	return "", fmt.Errorf("unterminated txtpbfmt off at %s", p.errorContext())
}

// skipWhiteSpaceAndReadComments has multiple cases:
//   - (1) reading a block of comments followed by a blank line
//   - (2) reading a block of comments followed by non-blank content
//   - (3) reading the inline comments between the current char and the end of
//     the current line
//
// In both cases (1) and (2), there can also be blank lines before the comment
// starts.
//
// Lines of comments and number of blank lines before the comment will be
// returned. If there is no comment, the returned slice will be empty.
func (p *parser) skipWhiteSpaceAndReadComments(multiLine bool) ([]string, int) {
	i := p.index
	var foundComment, insideComment bool
	commentBegin := 0
	var comments []string
	// Number of blanks lines *before* the comment (if any) starts.
	blankLines := 0
	for ; i < p.length; i++ {
		if p.in[i] == '#' && !insideComment {
			insideComment = true
			foundComment = true
			commentBegin = i
		} else if p.in[i] == '\n' {
			if insideComment {
				comments = append(comments, string(p.in[commentBegin:i])) // Exclude the '\n'.
				insideComment = false
			} else if foundComment {
				i-- // Put back the last '\n' so the caller can detect that we're on case (1).
				break
			} else {
				blankLines++
			}
			if !multiLine {
				break
			}
		}
		if !insideComment && !p.isBlankSep(i) {
			break
		}
	}
	sep := p.advance(i)
	if p.config.InfoLevel() {
		p.config.Infof("sep: %q\np.index: %v", string(sep), p.index)
		if p.index < p.length {
			p.config.Infof("p.in[p.index]: %q", string(p.in[p.index]))
		}
	}
	return comments, blankLines
}

func (p *parser) isBlankSep(i int) bool {
	return bytes.Contains(spaceSeparators, p.in[i:i+1])
}

func (p *parser) isValueSep(i int) bool {
	return bytes.Contains(valueSeparators, p.in[i:i+1])
}

func (p *parser) advance(i int) string {
	if i > p.length {
		i = p.length
	}
	res := p.in[p.index:i]
	p.index = i
	strRes := string(res)
	newlines := strings.Count(strRes, "\n")
	if newlines == 0 {
		p.column += len(strRes)
	} else {
		p.column = len(strRes) - strings.LastIndex(strRes, "\n")
		p.line += newlines
	}
	return string(res)
}

func (p *parser) readValues() ([]*ast.Value, error) {
	var values []*ast.Value
	var previousPos ast.Position
	preComments, _ := p.skipWhiteSpaceAndReadComments(true /* multiLine */)
	if p.nextInputIs('%') {
		values = append(values, p.populateValue(p.readTemplate(), nil))
		previousPos = p.position()
	}
	if v, err := p.readTripleQuotedStringValue(); err != nil {
		return nil, err
	} else {
		if v != nil {
			values = append(values, v)
			previousPos = p.position()
		}
	}
	for p.consume('"') || p.consume('\'') {
		// Handle string value.
		v, err := p.readSingleQuotedStringValue(preComments)
		if err != nil {
			return nil, err
		}
		values = append(values, v)
		previousPos = p.position()
		preComments, _ = p.skipWhiteSpaceAndReadComments(true /* multiLine */)
	}
	if previousPos != (ast.Position{}) {
		// Rewind comments.
		p.rollbackPosition(previousPos)
	} else {
		i := p.index
		// Handle other values.
		values = append(values, p.readOtherValue(i, preComments))
	}
	if p.config.InfoLevel() {
		p.config.Infof("values: %v", values)
	}
	return values, nil
}

func (p *parser) readTripleQuotedStringValue() (*ast.Value, error) {
	if !p.config.AllowTripleQuotedStrings {
		return nil, nil
	}
	return p.readTripleQuotedString()
}

func (p *parser) readSingleQuotedStringValue(preComments []string) (*ast.Value, error) {
	stringBegin := p.index - 1 // Index of the quote.
	i := p.index
	for ; i < p.length; i++ {
		if p.in[i] == '\\' {
			i++ // Skip escaped char.
			continue
		}
		if p.in[i] == '\n' {
			p.index = i
			return nil, fmt.Errorf("found literal (unescaped) new line in string at %s", p.errorContext())
		}
		if p.in[i] == p.in[stringBegin] {
			var vl string
			if p.config.SmartQuotes {
				vl = quote.Smart(p.advance(i))
			} else {
				vl = quote.Fix(p.advance(i))
			}
			_ = p.advance(i + 1) // Skip the quote.
			return p.populateValue(vl, preComments), nil
		}
	}
	if i == p.length {
		p.index = i
		return nil, fmt.Errorf("unfinished string at %s", p.errorContext())
	}
	return nil, nil
}

func (p *parser) readOtherValue(i int, preComments []string) *ast.Value {
	for ; i < p.length; i++ {
		if p.isValueSep(i) {
			break
		}
	}
	vl := p.advance(i)
	return p.populateValue(vl, preComments)
}

func (p *parser) readTripleQuotedString() (*ast.Value, error) {
	start := p.index
	stringBegin := p.index
	delimiter := `"""`
	if !p.consumeString(delimiter) {
		delimiter = `'''`
		if !p.consumeString(delimiter) {
			return nil, nil
		}
	}

	for {
		if p.consumeString(delimiter) {
			break
		}
		if p.index == p.length {
			p.index = start
			return nil, fmt.Errorf("unfinished string at %s", p.errorContext())
		}
		p.index++
	}

	v := p.populateValue(string(p.in[stringBegin:p.index]), nil)

	return v, nil
}

func (p *parser) populateValue(vl string, preComments []string) *ast.Value {
	if p.config.InfoLevel() {
		p.config.Infof("value: %q", vl)
	}
	return &ast.Value{
		Value:         vl,
		InlineComment: p.readInlineComment(),
		PreComments:   preComments,
	}
}

func (p *parser) readInlineComment() string {
	inlineComment, _ := p.skipWhiteSpaceAndReadComments(false /* multiLine */)
	if p.config.InfoLevel() {
		p.config.Infof("inlineComment: %q", strings.Join(inlineComment, "\n"))
	}
	if len(inlineComment) > 0 {
		return inlineComment[0]
	}
	return ""
}

func (p *parser) readStringInTemplate(i int) int {
	stringBegin := i - 1 // Index of quote.
	for ; i < p.length; i++ {
		if p.in[i] == '\\' {
			i++ // Skip escaped char.
			continue
		}
		if p.in[i] == p.in[stringBegin] {
			i++ // Skip end quote.
			break
		}
	}
	return i
}

func (p *parser) readTemplate() string {
	if !p.nextInputIs('%') {
		return ""
	}
	i := p.index + 1
	for ; i < p.length; i++ {
		if p.in[i] == '"' || p.in[i] == '\'' {
			i++
			i = p.readStringInTemplate(i)
		}
		if i < p.length && p.in[i] == '%' {
			i++
			break
		}
	}
	return p.advance(i)
}
