package insights

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/inngest/inngest/pkg/duckdb/parser"
)

// maxSizeArgument caps the output a single call to one of the size-guarded
// functions below (repeat, lpad, rpad, format, printf, range,
// generate_series) can ask for: characters for the string functions,
// elements for range/generate_series.
//
// These are guarded statically because DuckDB allocates their output
// outside memory_limit (driver.ResourceLimits), so the sandbox can't bound
// them. Measured against duckdb w/ memory_limit=200MB, threads=2:
//
//   - repeat('x', 1500000000) -> ~3GB RSS, lpad/rpad -> ~5GB,
//     format('{:>1500000000}', 'x') / printf('%1500000000s', 'x') -> ~4.5GB
//   - range(300000000) / generate_series(300000000) -> ~22GB
//   - per row it multiplies by the chunk size: lpad(col, 100000, 'y') over
//     200k rows -> ~250MB, range(<col> + 100000) -> ~3.8GB, while 10000 stays
//     at ~55MB for strings
//
// string_split isn't guarded: its output is bounded by its input, which
// (now that repeat/lpad/... can't inflate a string) is bounded by the data
// itself.
//
// This is a deliberate exception to checkFunction's "only reject what
// DuckDB would reject" rule: these calls would run, just not safely.
//
// It also bounds how these calls compose (nested in a list comprehension,
// copying row data per element, adding up across the statement), which the
// per-call checks can't see. See checkSizeComposition.
const maxSizeArgument = 10_000

// sizeLimitDiagnostic is the DiagnosticError a size guard rejects a call
// with, promoted by checkFunction like the argument-count diagnostics.
func sizeLimitDiagnostic(name, problem string) Diagnostic {
	return Diagnostic{
		Severity: DiagnosticError,
		Code:     "size-limit",
		Message:  fmt.Sprintf("%s %s", strings.ToUpper(name), problem),
	}
}

// sizeCheck returns the output size (characters or elements, see
// maxSizeArgument) a call to one size-guarded function asks for, or a
// non-empty problem (for sizeLimitDiagnostic) when args ask for too much or
// can't be bounded statically. It can assume a valid argument count.
type sizeCheck func(args []parser.Expr) (size float64, problem string)

// sizeChecks is every size-guarded function, keyed by its allowedFunctions
// name. withSizeLimit looks a call up here, and so does
// checkSizeComposition.
var sizeChecks = map[string]sizeCheck{
	"repeat":          repeatSizeLimit,
	"lpad":            padSizeLimit,
	"rpad":            padSizeLimit,
	"format":          formatSizeLimit(false),
	"printf":          formatSizeLimit(true),
	"range":           seriesSizeLimit(false),
	"generate_series": seriesSizeLimit(true),
}

// withSizeLimit wraps next with name's sizeChecks entry. It goes inside
// withArity, so the check can assume a valid argument count.
func withSizeLimit(next functionTypeResolver) functionTypeResolver {
	return func(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
		if check, ok := sizeChecks[name]; ok {
			if _, problem := check(args); problem != "" {
				return ColumnTypeUnknown, nil, []Diagnostic{sizeLimitDiagnostic(name, problem)}
			}
		}
		return next(name, args, scope)
	}
}

// numberLiteral returns e's value if it's a (possibly signed) number
// literal. Anything else, including a cast or arithmetic on literals, is
// non-literal: a size guard can't bound it, so it's rejected.
func numberLiteral(e parser.Expr) (float64, bool) {
	sign := 1.0
	if u, ok := e.(*parser.UnaryExpr); ok && !u.Postfix && (u.Op == "-" || u.Op == "+") {
		if u.Op == "-" {
			sign = -1
		}
		e = u.X
	}
	lit, ok := e.(*parser.Literal)
	if !ok || lit.Kind != parser.LitNumber {
		return 0, false
	}
	text := strings.ReplaceAll(lit.Text, "_", "")
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		i, ierr := strconv.ParseInt(text, 0, 64) // 0x... literals
		if ierr != nil {
			return 0, false
		}
		v = float64(i)
	}
	return sign * v, !math.IsNaN(v) && !math.IsInf(v, 0)
}

func stringLiteral(e parser.Expr) (string, bool) {
	lit, ok := e.(*parser.Literal)
	if !ok || lit.Kind != parser.LitString {
		return "", false
	}
	return lit.Text, true
}

var (
	errSizeNotLiteral = fmt.Sprintf("requires a number literal size/count argument (at most %d)", maxSizeArgument)
	errSizeTooLarge   = fmt.Sprintf("output size is limited to %d", maxSizeArgument)
)

// repeatSizeLimit: repeat(string, count) produces len(string) * count
// characters, so both must be literals. A column (or a nested repeat) as
// the string would multiply by an unbounded length.
func repeatSizeLimit(args []parser.Expr) (float64, string) {
	s, ok := stringLiteral(args[0])
	if !ok {
		return 0, fmt.Sprintf("requires a string literal as its string argument (its output is len(string) * count, limited to %d)", maxSizeArgument)
	}
	count, ok := numberLiteral(args[1])
	if !ok {
		return 0, errSizeNotLiteral
	}
	size := max(0, float64(len(s))*count)
	if size > maxSizeArgument {
		return 0, errSizeTooLarge
	}
	return size, ""
}

// padSizeLimit: lpad/rpad(string, count, character) always produce exactly
// count characters (truncating a longer string), so only count matters.
func padSizeLimit(args []parser.Expr) (float64, string) {
	count, ok := numberLiteral(args[1])
	if !ok {
		return 0, errSizeNotLiteral
	}
	if count > maxSizeArgument {
		return 0, errSizeTooLarge
	}
	return max(0, count), ""
}

// seriesSizeLimit is range/generate_series(start[, stop][, step]):
// every argument must be an integer literal and the element count at most
// maxSizeArgument. inclusive is generate_series' inclusive stop.
// Timestamp series are rejected too (their bounds aren't number
// literals), since a small interval over a long span is just as unbounded,
// and so are decimal bounds, which DuckDB rejects anyway.
//
// The count is computed exactly (big.Int), never in float64: bounds near
// BIGINT's range are farther apart than a float64 ulp, so float math could
// round a real 500-element series down to 0, and checkSizeComposition
// treats a 0-length comprehension source as never evaluating its body.
func seriesSizeLimit(inclusive bool) sizeCheck {
	return func(args []parser.Expr) (float64, string) {
		vals := make([]*big.Int, len(args))
		for i, a := range args {
			v, ok := integerLiteral(a)
			if !ok {
				return 0, fmt.Sprintf("requires integer literal arguments (at most %d elements)", maxSizeArgument)
			}
			vals[i] = v
		}
		start, stop, step := big.NewInt(0), vals[0], big.NewInt(1)
		if len(vals) >= 2 {
			start, stop = vals[0], vals[1]
		}
		if len(vals) == 3 {
			step = vals[2]
		}
		if step.Sign() == 0 {
			return 0, "" // DuckDB rejects this itself
		}
		// Normalize to an ascending span w/ a positive step.
		span := new(big.Int).Sub(stop, start)
		absStep := new(big.Int).Abs(step)
		if step.Sign() < 0 {
			span.Neg(span)
		}
		n := new(big.Int)
		switch {
		case inclusive && span.Sign() >= 0:
			n.Quo(span, absStep).Add(n, big.NewInt(1)) // floor(span/step) + 1
		case !inclusive && span.Sign() > 0:
			n.Add(span, absStep).Sub(n, big.NewInt(1)).Quo(n, absStep) // ceil(span/step)
		}
		if n.Cmp(big.NewInt(maxSizeArgument)) > 0 {
			return 0, errSizeTooLarge
		}
		return float64(n.Int64()), ""
	}
}

// integerLiteral returns e's exact value if it's a (possibly signed)
// integer literal (decimal, or 0x/0b/0o, w/ optional "_" separators).
func integerLiteral(e parser.Expr) (*big.Int, bool) {
	neg := false
	if u, ok := e.(*parser.UnaryExpr); ok && !u.Postfix && (u.Op == "-" || u.Op == "+") {
		neg = u.Op == "-"
		e = u.X
	}
	lit, ok := e.(*parser.Literal)
	if !ok || lit.Kind != parser.LitNumber {
		return nil, false
	}
	text := strings.ReplaceAll(lit.Text, "_", "")
	base := 10 // DuckDB reads "010" as 10, not octal
	if len(text) > 1 && text[0] == '0' && strings.ContainsAny(text[1:2], "xXbBoO") {
		base = 0
	}
	v, ok := new(big.Int).SetString(text, base)
	if !ok {
		return nil, false
	}
	if neg {
		v.Neg(v)
	}
	return v, true
}

// formatSizeLimit is format/printf(format, ...): the format string must be
// a literal, every width/precision in it at most maxSizeArgument, and none
// may come from an argument ("{:>{}}", "%*s"). printf selects printf-style
// specs ("%-10.3f"), otherwise fmt-style ("{:>10}").
//
// Its size (for checkSizeComposition) is the sum of its widths/precisions,
// the part of the output the format asks for on its own (the rest is the
// format's own text and the arguments, bounded by the query text and data).
func formatSizeLimit(printf bool) sizeCheck {
	return func(args []parser.Expr) (float64, string) {
		f, ok := stringLiteral(args[0])
		if !ok {
			return 0, fmt.Sprintf("requires a string literal format (widths are limited to %d)", maxSizeArgument)
		}
		specs, dynamic := fmtSpecs(f)
		if printf {
			specs, dynamic = printfSpecs(f)
		}
		if dynamic {
			return 0, fmt.Sprintf("doesn't support a width or precision taken from an argument (widths are limited to %d)", maxSizeArgument)
		}
		size := 0.0
		for _, spec := range specs {
			width := maxDigitRun(spec)
			if width > maxSizeArgument {
				return 0, fmt.Sprintf("widths and precisions are limited to %d", maxSizeArgument)
			}
			size += width
		}
		return size, ""
	}
}

// fmtSpecs returns the inside of every "{...}" replacement field in an
// fmt-style format string ("{{"/"}}" are escapes), and whether any has a
// nested (argument-supplied) width/precision.
func fmtSpecs(f string) (specs []string, dynamic bool) {
	for i := 0; i < len(f); i++ {
		if f[i] != '{' {
			continue
		}
		if i+1 < len(f) && f[i+1] == '{' {
			i++
			continue
		}
		end := strings.IndexByte(f[i+1:], '}')
		if end < 0 {
			return specs, dynamic // malformed, DuckDB rejects it
		}
		spec := f[i+1 : i+1+end]
		if strings.ContainsRune(spec, '{') {
			dynamic = true
		}
		specs = append(specs, spec)
		i += end + 1
	}
	return specs, dynamic
}

// printfSpecs returns the flags/width/precision part of every "%..."
// conversion in a printf-style format string ("%%" is an escape), and
// whether any takes its width/precision from an argument ("*").
func printfSpecs(f string) (specs []string, dynamic bool) {
	for i := 0; i < len(f); i++ {
		if f[i] != '%' {
			continue
		}
		j := i + 1
		for j < len(f) && strings.IndexByte("-+ #0123456789.*$'", f[j]) >= 0 {
			j++
		}
		spec := f[i+1 : j]
		if spec == "" && j < len(f) && f[j] == '%' {
			i = j
			continue
		}
		if strings.ContainsRune(spec, '*') {
			dynamic = true
		}
		specs = append(specs, spec)
		i = j
	}
	return specs, dynamic
}

// maxDigitRun returns the largest decimal number appearing in s, or 0.
// Saturates rather than overflowing for an absurdly long run.
func maxDigitRun(s string) float64 {
	best, cur := 0.0, 0.0
	for i := 0; i <= len(s); i++ {
		if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			cur = cur*10 + float64(s[i]-'0')
			continue
		}
		best = max(best, cur)
		cur = 0
	}
	return best
}

// checkSizeComposition bounds what the per-call guards (sizeChecks) can't see
// on their own: how many copies of a guarded call's output get produced at
// once. Measured against the pinned duckdb w/ memory_limit=2GiB, threads=4
// (driver.DefaultResourceLimits):
//
//   - A list comprehension evaluates its body/filter once per source element,
//     outside memory_limit: len([range(10000) for x in range(10000)]) -> ~7GB
//     RSS (~18GB w/ flatten), and per row (a whole 2048-row vector at once)
//     [range(10000) || [i] for x in range(100)] -> >21GB.
//   - Row data gets copied the same way: [s for x in range(10000)] over a ~1KB
//     string column -> >14GB, and a CTE column holding a range(10000) can be
//     copied by name the same way.
//   - Sibling calls add up: list_value() of 10 per-row range(10000)s -> ~4.2GB
//     (1 -> ~470MB, 3 -> ~1.5GB).
//
// A single guarded call evaluated per row streams in vectors, so it stays
// bounded however many rows there are (sum(len(range(10000) || [i])) over 1M
// rows -> ~470MB), and aggregating its output (list, string_agg) is tracked
// by memory_limit (it fails w/ an Out of Memory Error instead).
//
// Rather than track sizes through the expression tree (which has to get
// every way of naming or copying a value right), the rule is deliberately
// blunt and fail-closed:
//
//  1. A guarded call can't appear anywhere inside a list comprehension
//     (source, body, or filter), so nothing evaluates it, or a list it
//     produced, once per element.
//  2. A guarded call can only appear in the statement's own top-level SELECT
//     (or a UNION/INTERSECT/EXCEPT operand of it), never in a CTE, a FROM
//     subquery, or an expression subquery. So its output can't be referenced
//     (and copied) by name: every copy is its own call site in the query text.
//  3. The sizes of every guarded call in the statement add up to at most
//     maxSizeArgument, so sibling calls (and repeating a call in the query
//     text) can't add up past what one call may ask for.
//
// This rejects some harmless shapes ([repeat('-', 3) for x in tags], two
// rpad(col, 10000, ' ') columns, a repeat() in a CTE), which is the trade-off
// for not having to reason about where a guarded value flows.
func checkSizeComposition(stmt *parser.SelectStatement) error {
	c := &sizeComposition{}
	c.walkTopLevel(stmt)
	return c.err
}

// sizeComposition walks a statement for checkSizeComposition.
type sizeComposition struct {
	// total is the statement's guarded output so far (rule 3).
	total float64
	err   error
}

// walkTopLevel walks stmt as a top-level SELECT: its own expressions may have
// guarded calls (rule 2), and so may its set-op operands.
func (c *sizeComposition) walkTopLevel(stmt *parser.SelectStatement) {
	if stmt.SetOp != parser.SetOpNone {
		if stmt.With != nil {
			c.walkNested(stmt.With)
		}
		c.walkTopLevel(stmt.SetLeft)
		c.walkTopLevel(stmt.SetRight)
		// ORDER BY/LIMIT etc. on the set-op node itself.
		for _, n := range stmt.Children() {
			if c.err != nil {
				return
			}
			if n != parser.Node(stmt.SetLeft) && n != parser.Node(stmt.SetRight) && n != parser.Node(stmt.With) {
				parser.Walk(c, n)
			}
		}
		return
	}
	for _, n := range stmt.Children() {
		if c.err != nil {
			return
		}
		parser.Walk(c, n)
	}
}

// walkNested rejects any guarded call under n (rule 2).
func (c *sizeComposition) walkNested(n parser.Node) {
	if c.err == nil {
		c.err = rejectSizeLimitedCalls(n, "can't be used inside a CTE or subquery, only in the query's top-level SELECT")
	}
}

func (c *sizeComposition) Visit(n parser.Node) parser.Visitor {
	if c.err != nil {
		return nil
	}
	switch x := n.(type) {
	case *parser.SelectStatement, *parser.WithClause, *parser.TableSubqueryRef, *parser.SubqueryExpr:
		c.walkNested(x)
		return nil
	case *parser.ListComprehensionExpr:
		c.err = rejectSizeLimitedCalls(x, "can't be used inside a list comprehension")
		return nil
	case *parser.FunctionExpr:
		name := strings.ToLower(strings.Join(x.Name, "."))
		if check, ok := sizeChecks[name]; ok {
			size, problem := check(x.Args)
			if problem != "" {
				return nil // validate already rejected it
			}
			c.total += size
			if c.total > maxSizeArgument {
				c.err = sizeCompositionError(x, fmt.Sprintf("%s output size is limited to %d, counting every size-limited call in the query together", strings.ToUpper(name), maxSizeArgument))
				return nil
			}
		}
	}
	return c
}

// rejectSizeLimitedCalls returns an error naming the first guarded call
// under n, if any.
func rejectSizeLimitedCalls(n parser.Node, problem string) error {
	var err error
	parser.Walk(visitFunc(func(n parser.Node) bool {
		if f, ok := n.(*parser.FunctionExpr); ok {
			name := strings.ToLower(strings.Join(f.Name, "."))
			if _, ok := sizeChecks[name]; ok {
				err = sizeCompositionError(f, fmt.Sprintf("%s %s (output size is limited to %d)", strings.ToUpper(name), problem, maxSizeArgument))
			}
		}
		return err == nil
	}), n)
	return err
}

// visitFunc is a parser.Visitor that keeps descending while it returns true.
type visitFunc func(parser.Node) bool

func (f visitFunc) Visit(n parser.Node) parser.Visitor {
	if f(n) {
		return f
	}
	return nil
}

func sizeCompositionError(n parser.Node, msg string) error {
	return &ValidationError{Pos: n.Pos(), End: n.End(), Message: msg}
}
