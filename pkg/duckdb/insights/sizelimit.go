package insights

import (
	"fmt"
	"math"
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

// withSizeLimit wraps next with check, which returns a non-empty problem
// (for sizeLimitDiagnostic) when args ask for too much output. It goes
// inside withArity, so check can assume a valid argument count.
func withSizeLimit(check func(args []parser.Expr) string, next functionTypeResolver) functionTypeResolver {
	return func(name string, args []parser.Expr, scope *tableScope) (ColumnType, []PathHint, []Diagnostic) {
		if problem := check(args); problem != "" {
			return ColumnTypeUnknown, nil, []Diagnostic{sizeLimitDiagnostic(name, problem)}
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
func repeatSizeLimit(args []parser.Expr) string {
	s, ok := stringLiteral(args[0])
	if !ok {
		return fmt.Sprintf("requires a string literal as its string argument (its output is len(string) * count, limited to %d)", maxSizeArgument)
	}
	count, ok := numberLiteral(args[1])
	if !ok {
		return errSizeNotLiteral
	}
	if float64(len(s))*count > maxSizeArgument {
		return errSizeTooLarge
	}
	return ""
}

// padSizeLimit: lpad/rpad(string, count, character) always produce exactly
// count characters (truncating a longer string), so only count matters.
func padSizeLimit(args []parser.Expr) string {
	count, ok := numberLiteral(args[1])
	if !ok {
		return errSizeNotLiteral
	}
	if count > maxSizeArgument {
		return errSizeTooLarge
	}
	return ""
}

// seriesSizeLimit is range/generate_series(start[, stop][, step]):
// every argument must be a number literal and the element count at most
// maxSizeArgument. inclusive is generate_series' inclusive stop.
// Timestamp series are rejected too (their bounds aren't number
// literals), since a small interval over a long span is just as unbounded.
func seriesSizeLimit(inclusive bool) func(args []parser.Expr) string {
	return func(args []parser.Expr) string {
		vals := make([]float64, len(args))
		for i, a := range args {
			v, ok := numberLiteral(a)
			if !ok {
				return fmt.Sprintf("requires number literal arguments (at most %d elements)", maxSizeArgument)
			}
			vals[i] = v
		}
		start, stop, step := 0.0, vals[0], 1.0
		if len(vals) >= 2 {
			start, stop = vals[0], vals[1]
		}
		if len(vals) == 3 {
			step = vals[2]
		}
		if step == 0 {
			return "" // DuckDB rejects this itself
		}
		n := math.Ceil((stop - start) / step)
		if inclusive {
			n = math.Floor((stop-start)/step) + 1
		}
		if n > maxSizeArgument {
			return errSizeTooLarge
		}
		return ""
	}
}

// formatSizeLimit is format/printf(format, ...): the format string must be
// a literal, every width/precision in it at most maxSizeArgument, and none
// may come from an argument ("{:>{}}", "%*s"). printf selects printf-style
// specs ("%-10.3f"), otherwise fmt-style ("{:>10}").
func formatSizeLimit(printf bool) func(args []parser.Expr) string {
	return func(args []parser.Expr) string {
		f, ok := stringLiteral(args[0])
		if !ok {
			return fmt.Sprintf("requires a string literal format (widths are limited to %d)", maxSizeArgument)
		}
		specs, dynamic := fmtSpecs(f)
		if printf {
			specs, dynamic = printfSpecs(f)
		}
		if dynamic {
			return fmt.Sprintf("doesn't support a width or precision taken from an argument (widths are limited to %d)", maxSizeArgument)
		}
		for _, spec := range specs {
			if maxDigitRun(spec) > maxSizeArgument {
				return fmt.Sprintf("widths and precisions are limited to %d", maxSizeArgument)
			}
		}
		return ""
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
