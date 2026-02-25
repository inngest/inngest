package expr

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// VarPrefix is the lifted variable name used when extracting idents from an
	// expression.
	VarPrefix = "vars"
)

var (
	// replace is truly hack city.  these are 20 variable names for values that are
	// lifted out of expressions via liftLiterals.
	replace = []string{
		"a", "b", "c", "d", "e",
		"f", "g", "h", "i", "j",
		"k", "l", "m", "n", "o",
		"p", "q", "r", "s", "t",
		"u", "v", "w", "x", "y",
		"z",
	}
)

// LiftedArgs represents a set of variables that have been lifted from expressions and
// replaced with identifiers, eg `id == "foo"` becomes `id == vars.a`, with "foo" lifted
// as "vars.a".
type LiftedArgs interface {
	// Get a lifted variable argument from the parsed expression.
	Get(val string) (any, bool)
	// Return all lifted variables as a map.
	Map() map[string]any
}

// liftLiterals lifts quoted literals into variables, allowing us to normalize
// expressions to increase cache hit rates.
func liftLiterals(expr string) (string, LiftedArgs) {
	if strings.Contains(expr, VarPrefix+".") {
		// Do not lift an expression twice, else we run the risk of using
		// eg. `vars.a` to reference two separate strings, breaking the
		// expression.
		return expr, nil
	}

	lp := liftParser{expr: expr}
	return lp.lift()
}

type liftParser struct {
	expr string
	idx  int

	rewritten *strings.Builder

	// varCounter counts the number of variables lifted.
	varCounter int

	vars pointerArgMap

	// prevChar distinguishes a numeric literal start from a digit within an identifier.
	prevChar byte

	// bracketDepth: array indices must not be lifted; parseArrayAccess expects integer literals.
	bracketDepth int
}

func (l *liftParser) lift() (string, LiftedArgs) {
	l.vars = pointerArgMap{
		expr: l.expr,
		vars: map[string]argMapValue{},
	}

	l.rewritten = &strings.Builder{}

	comment := false

	for l.idx < len(l.expr) {
		char := l.expr[l.idx]

		l.idx++

		if comment && char == '\n' {
			comment = false
		}
		if comment && char != '\n' {
			continue
		}

		switch char {
		case '/':
			// if the next character is a slash, this is a comment line ("//")
			if len(l.expr) > l.idx && string(l.expr[l.idx]) == "/" {
				comment = true
				continue
			}
			// prevChar must be '/' so digits immediately after (e.g. x/2) are lifted, not skipped.
			l.rewritten.WriteByte(char)
			l.prevChar = char
		case '"':
			// Consume the string arg.
			val := l.consumeString('"')
			l.addLiftedVar(val)

		case '\'':
			val := l.consumeString('\'')
			l.addLiftedVar(val)
		case '.':
			// Leading-dot float (.5): if we wrote the dot then lifted the digit we'd produce ".vars.a".
			if !isIdentChar(l.prevChar) && l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
				l.consumeLeadingDotFloat()
			} else {
				l.rewritten.WriteByte(char)
				l.prevChar = char
			}
		case '[':
			l.bracketDepth++
			l.rewritten.WriteByte(char)
			l.prevChar = char
		case ']':
			l.bracketDepth--
			l.rewritten.WriteByte(char)
			l.prevChar = char
		default:
			if char >= '0' && char <= '9' && !isIdentChar(l.prevChar) {
				l.consumeNumeric(char)
			} else {
				l.rewritten.WriteByte(char)
				l.prevChar = char
			}
		}
	}

	return strings.TrimSpace(l.rewritten.String()), &l.vars
}

// isIdentChar returns true if c can be part of an identifier (a-z, A-Z, 0-9, _).
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// consumeLeadingDotFloat lifts a leading-dot float literal (.5, .5e2).
// The dot has already been consumed; l.idx points to the first digit.
func (l *liftParser) consumeLeadingDotFloat() {
	start := l.idx - 1 // include the leading dot

	for l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
		l.idx++
	}

	if l.idx < len(l.expr) && (l.expr[l.idx] == 'e' || l.expr[l.idx] == 'E') {
		l.idx++
		if l.idx < len(l.expr) && (l.expr[l.idx] == '+' || l.expr[l.idx] == '-') {
			l.idx++
		}
		for l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
			l.idx++
		}
	}

	numStr := l.expr[start:l.idx]
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		l.rewritten.WriteString(numStr)
		if len(numStr) > 0 {
			l.prevChar = numStr[len(numStr)-1]
		}
		return
	}
	l.addLiftedVar(argMapValue{parsed: f})
}

// consumeNumeric lifts a numeric literal so expressions differing only in value share
// the same CEL cache entry.
func (l *liftParser) consumeNumeric(first byte) {
	// Array index — parseArrayAccess expects an integer literal, not vars.X.
	if l.bracketDepth > 0 {
		l.rewritten.WriteByte(first)
		l.prevChar = first
		return
	}

	start := l.idx - 1 // first was already consumed (l.idx was incremented before the switch)

	// 0x/0b/0o prefix — base-10 parsing would give the wrong value.
	// TODO: we can lift those as well but not a priority?
	if first == '0' && l.idx < len(l.expr) {
		next := l.expr[l.idx]
		if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
			l.rewritten.WriteByte(first)
			l.prevChar = first
			return
		}
	}

	for l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
		l.idx++
	}

	// u/U suffix — lifting the digits alone leaves "u" in the expression, producing "vars.au" which is completely wrong
	// TODO: we can lift those as well but not a priority?
	if l.idx < len(l.expr) && (l.expr[l.idx] == 'u' || l.expr[l.idx] == 'U') {
		l.idx++ // consume the suffix as part of the token
		numStr := l.expr[start:l.idx]
		l.rewritten.WriteString(numStr)
		l.prevChar = numStr[len(numStr)-1]
		return
	}

	// Dot is fractional only when followed by a digit; trailing dot (1.) or field accessor (.field) are not.
	isFloat := false
	if l.idx < len(l.expr) && l.expr[l.idx] == '.' &&
		l.idx+1 < len(l.expr) && l.expr[l.idx+1] >= '0' && l.expr[l.idx+1] <= '9' {
		isFloat = true
		l.idx++ // consume '.'
		for l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
			l.idx++
		}
	}

	// Consume e/E exponent whole; leaving "e10" would produce "vars.ae10" (field access, not a number) which is wrong as well
	if l.idx < len(l.expr) && (l.expr[l.idx] == 'e' || l.expr[l.idx] == 'E') {
		l.idx++
		if l.idx < len(l.expr) && (l.expr[l.idx] == '+' || l.expr[l.idx] == '-') {
			l.idx++
		}
		for l.idx < len(l.expr) && l.expr[l.idx] >= '0' && l.expr[l.idx] <= '9' {
			l.idx++
		}
		numStr := l.expr[start:l.idx]
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			l.rewritten.WriteString(numStr)
			if len(numStr) > 0 {
				l.prevChar = numStr[len(numStr)-1]
			}
			return
		}
		l.addLiftedVar(argMapValue{parsed: f})
		return
	}

	numStr := l.expr[start:l.idx]
	if isFloat {
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			l.rewritten.WriteString(numStr)
			if len(numStr) > 0 {
				l.prevChar = numStr[len(numStr)-1]
			}
			return
		}
		l.addLiftedVar(argMapValue{parsed: f})
	} else {
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			l.rewritten.WriteString(numStr)
			if len(numStr) > 0 {
				l.prevChar = numStr[len(numStr)-1]
			}
			return
		}
		l.addLiftedVar(argMapValue{parsed: n})
	}
}

func (l *liftParser) addLiftedVar(val argMapValue) {
	if l.varCounter >= len(replace) {
		// Do nothing.
		v := val.get(l.expr)
		var s string
		switch typed := v.(type) {
		case string:
			s = strconv.Quote(typed)
		case int64:
			s = strconv.FormatInt(typed, 10)
		case float64:
			s = strconv.FormatFloat(typed, 'f', -1, 64)
		default:
			s = fmt.Sprintf("%v", v)
		}
		l.rewritten.WriteString(s)
		if len(s) > 0 {
			l.prevChar = s[len(s)-1]
		}
		return
	}

	letter := replace[l.varCounter]

	l.vars.vars[letter] = val
	l.varCounter++

	l.rewritten.WriteString(VarPrefix + "." + letter)
	l.prevChar = letter[0]
}

func (l *liftParser) consumeString(quoteChar byte) argMapValue {
	offset := l.idx
	length := 0

	for l.idx < len(l.expr) {
		char := l.expr[l.idx]

		if char == '\\' && l.peek() == quoteChar {
			// If we're escaping the quote character, ignore it.
			l.idx += 2
			length += 2
			continue
		}

		if char == quoteChar {
			// Skip over the end quote.
			l.idx++
			// Return the substring offset/length
			return argMapValue{offset: offset, length: length}
		}

		// Grab the next char for evaluation.
		l.idx++

		// Only now has the length of the inner quote increased.
		length++
	}

	// Should never happen:  we should always find the ending string quote, as the
	// expression should have already been validated.
	panic(fmt.Sprintf("unable to parse quoted string: `%s` (offset %d)", l.expr, offset))
}

func (l *liftParser) peek() byte {
	if (l.idx + 1) >= len(l.expr) {
		return 0x0
	}
	return l.expr[l.idx+1]
}

// pointerArgMap takes the original expression, and adds pointers to the original expression
// in order to grab variables.
//
// It does this by pointing to the offset and length of data within the expression, as opposed
// to extracting the value into a new string.  This greatly reduces memory growth & heap allocations.
type pointerArgMap struct {
	expr string
	vars map[string]argMapValue
}

func (p pointerArgMap) Map() map[string]any {
	res := map[string]any{}
	for k, v := range p.vars {
		res[k] = v.get(p.expr)
	}
	return res
}

func (p pointerArgMap) Get(key string) (any, bool) {
	val, ok := p.vars[key]
	if !ok {
		return nil, false
	}
	data := val.get(p.expr)
	return data, true
}

// argMapValue is either a string slice (offset/length into expr) or a pre-parsed numeric (parsed != nil).
type argMapValue struct {
	offset, length int
	parsed         any
}

func (a argMapValue) get(expr string) any {
	if a.parsed != nil {
		return a.parsed
	}
	return expr[a.offset : a.offset+a.length]
}

type regularArgMap map[string]any

func (p regularArgMap) Get(key string) (any, bool) {
	val, ok := p[key]
	return val, ok
}

func (p regularArgMap) Map() map[string]any {
	return p
}
