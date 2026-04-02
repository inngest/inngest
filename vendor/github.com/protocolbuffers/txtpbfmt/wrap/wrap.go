// Package wrap provides functions for wrapping strings in textproto ASTs.
package wrap

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mitchellh/go-wordwrap"
	"github.com/protocolbuffers/txtpbfmt/ast"
	"github.com/protocolbuffers/txtpbfmt/config"
	"github.com/protocolbuffers/txtpbfmt/unquote"
)

var tagRegex = regexp.MustCompile(`<.*>`)

const indentSpaces = "  "

// Strings wraps the strings in the given nodes.
func Strings(nodes []*ast.Node, depth int, c config.Config) error {
	if c.WrapStringsAtColumn == 0 && !c.WrapStringsAfterNewlines {
		return nil
	}
	for _, nd := range nodes {
		if nd.ChildrenSameLine {
			continue
		}
		if err := wrapNodeStrings(nd, depth, c); err != nil {
			return err
		}
		if err := Strings(nd.Children, depth+1, c); err != nil {
			return err
		}
	}
	return nil
}

func wrapNodeStrings(nd *ast.Node, depth int, c config.Config) error {
	if c.WrapStringsAtColumn > 0 && needsWrappingAtColumn(nd, depth, c) {
		if err := wrapLinesAtColumn(nd, depth, c); err != nil {
			return err
		}
	}
	if c.WrapStringsAfterNewlines && needsWrappingAfterNewlines(nd, c) {
		if err := wrapLinesAfterNewlines(nd, c); err != nil {
			return err
		}
	}
	return nil
}

func shouldWrapString(v *ast.Value, maxLength int, c config.Config) bool {
	if len(v.Value) >= 3 && (strings.HasPrefix(v.Value, `'''`) || strings.HasPrefix(v.Value, `"""`)) {
		// Don't wrap triple-quoted strings
		return false
	}
	if len(v.Value) > 0 && v.Value[0] != '\'' && v.Value[0] != '"' {
		// Only wrap strings
		return false
	}
	return len(v.Value) > maxLength || c.WrapStringsWithoutWordwrap
}

func shouldNotWrapString(nd *ast.Node, c config.Config) bool {
	if !c.WrapHTMLStrings {
		for _, v := range nd.Values {
			if tagRegex.Match([]byte(v.Value)) {
				return true
			}
		}
	}
	return false
}

func needsWrappingAtColumn(nd *ast.Node, depth int, c config.Config) bool {
	// Even at depth 0 we have a 2-space indent when the wrapped string is rendered on the line below
	// the field name.
	const lengthBuffer = 2
	maxLength := c.WrapStringsAtColumn - lengthBuffer - (depth * len(indentSpaces))

	if shouldNotWrapString(nd, c) {
		return false
	}

	for _, v := range nd.Values {
		if shouldWrapString(v, maxLength, c) {
			return true
		}
	}
	return false
}

func wrapLinesWithoutWordwrap(str string, maxLength int) []string {
	// https://protobuf.dev/reference/protobuf/textformat-spec/#string.
	// String literals can contain octal, hex, unicode, and C-style escape
	// sequences: \a \b \f \n \r \t \v \? \' \"\ ? \\
	re := regexp.MustCompile(`\\[abfnrtv?\\'"]` +
		`|\\[0-7]{1,3}` +
		`|\\x[0-9a-fA-F]{1,2}` +
		`|\\u[0-9a-fA-F]{4}` +
		`|\\U000[0-9a-fA-F]{5}` +
		`|\\U0010[0-9a-fA-F]{4}` +
		`|.`)
	var lines []string
	var line strings.Builder
	for _, t := range re.FindAllString(str, -1) {
		if line.Len()+len(t) > maxLength {
			lines = append(lines, line.String())
			line.Reset()
		}
		line.WriteString(t)
	}
	lines = append(lines, line.String())
	return lines
}

func adjustLineLength(nd *ast.Node, v *ast.Value, line string, maxLength int, i int, numLines int) {
	lineLength := len(line)
	if v.InlineComment != "" {
		lineLength += len(indentSpaces) + len(v.InlineComment)
	}
	// field name and field value are inlined for single strings, adjust for that.
	if i == 0 && numLines == 1 {
		lineLength += len(nd.Name)
	}
	if lineLength > maxLength {
		// If there's an inline comment, promote it to a pre-comment which will
		// emit a newline.
		if v.InlineComment != "" {
			v.PreComments = append(v.PreComments, v.InlineComment)
			v.InlineComment = ""
		} else if i == 0 && len(v.PreComments) == 0 {
			// It's too long and we don't have any comments.
			nd.PutSingleValueOnNextLine = true
		}
	}
}

// If the Values of this Node constitute a string, and if Config.WrapStringsAtColumn > 0, then wrap
// the string so each line is within the specified columns. Wraps only the current Node (does not
// recurse into Children).
func wrapLinesAtColumn(nd *ast.Node, depth int, c config.Config) error {
	// This function looks at the unquoted ast.Value.Value string (i.e., with each Value's wrapping
	// quote chars removed). We need to remove these quotes, since otherwise they'll be re-flowed into
	// the body of the text.
	const lengthBuffer = 4 // Even at depth 0 we have a 2-space indent and a pair of quotes
	maxLength := c.WrapStringsAtColumn - lengthBuffer - (depth * len(indentSpaces))

	str, quote, err := unquote.Raw(nd)
	if err != nil {
		return fmt.Errorf("skipping string wrapping on node %q (error unquoting string): %v", nd.Name, err)
	}

	var lines []string
	if c.WrapStringsWithoutWordwrap {
		lines = wrapLinesWithoutWordwrap(str, maxLength)
	} else {
		// Remove one from the max length since a trailing space may be added below.
		wrappedStr := wordwrap.WrapString(str, uint(maxLength)-1)
		lines = strings.Split(wrappedStr, "\n")
	}

	newValues := make([]*ast.Value, 0, len(lines))
	// The Value objects have more than just the string in them. They also have any leading and
	// trailing comments. To maintain these comments we recycle the existing Value objects if
	// possible.
	var i int
	var line string
	for i, line = range lines {
		var v *ast.Value
		if i < len(nd.Values) {
			v = nd.Values[i]
		} else {
			v = &ast.Value{}
		}

		if !c.WrapStringsWithoutWordwrap && i < len(lines)-1 {
			line = line + " "
		}

		if c.WrapStringsWithoutWordwrap {
			adjustLineLength(nd, v, line, maxLength, i, len(lines))
		}

		v.Value = fmt.Sprintf(`%c%s%c`, quote, line, quote)
		newValues = append(newValues, v)
	}

	postWrapCollectComments(nd, i)

	nd.Values = newValues
	return nil
}

// N.b.: this will incorrectly match `\\\\x`, which hopefully is rare.
var byteEscapeRegex = regexp.MustCompile(`\\x`)

func needsWrappingAfterNewlines(nd *ast.Node, c config.Config) bool {
	for _, v := range nd.Values {
		if len(v.Value) >= 3 && (strings.HasPrefix(v.Value, `'''`) || strings.HasPrefix(v.Value, `"""`)) {
			// Don't wrap triple-quoted strings
			return false
		}
		if len(v.Value) > 0 && v.Value[0] != '\'' && v.Value[0] != '"' {
			// Only wrap strings
			return false
		}
		byteEscapeCount := len(byteEscapeRegex.FindAllStringIndex(v.Value, -1))
		if float64(byteEscapeCount) > float64(len(v.Value))*0.1 {
			// Only wrap UTF-8 looking strings (where less than ~10% of the characters are escaped).
			return false
		}
		// Check that there is at least one newline, *not* at the end of the string.
		if i := strings.Index(v.Value, `\n`); i >= 0 && i < len(v.Value)-3 {
			return true
		}
	}
	return false
}

// If the Values of this Node constitute a string, and if Config.WrapStringsAfterNewlines,
// then wrap the string so each line ends with a newline.
// Wraps only the current Node (does not recurse into Children).
func wrapLinesAfterNewlines(nd *ast.Node, c config.Config) error {
	str, quote, err := unquote.Raw(nd)
	if err != nil {
		return fmt.Errorf("skipping string wrapping on node %q (error unquoting string): %v", nd.Name, err)
	}

	wrappedStr := strings.ReplaceAll(str, `\n`, `\n`+"\n")
	// Avoid empty string at end after splitting in case str ended with an (escaped) newline.
	wrappedStr = strings.TrimSuffix(wrappedStr, "\n")
	lines := strings.Split(wrappedStr, "\n")
	newValues := make([]*ast.Value, 0, len(lines))
	// The Value objects have more than just the string in them. They also have any leading and
	// trailing comments. To maintain these comments we recycle the existing Value objects if
	// possible.
	var i int
	var line string
	for i, line = range lines {
		var v *ast.Value
		if i < len(nd.Values) {
			v = nd.Values[i]
		} else {
			v = &ast.Value{}
		}
		v.Value = fmt.Sprintf(`%c%s%c`, quote, line, quote)
		newValues = append(newValues, v)
	}

	postWrapCollectComments(nd, i)

	nd.Values = newValues
	return nil
}

func postWrapCollectComments(nd *ast.Node, i int) {
	for i++; i < len(nd.Values); i++ {
		// If this executes, then the text was wrapped into less lines of text (less Values) than
		// previously. If any of these had comments on them, we collect them so they are not lost.
		v := nd.Values[i]
		nd.PostValuesComments = append(nd.PostValuesComments, v.PreComments...)
		if len(v.InlineComment) > 0 {
			nd.PostValuesComments = append(nd.PostValuesComments, v.InlineComment)
		}
	}
}
