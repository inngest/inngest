package schema

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
	"github.com/uptrace/bun/internal"
	"github.com/uptrace/bun/internal/parser"
)

var nopQueryGen = QueryGen{
	dialect: newNopDialect(),
}

type QueryGen struct {
	dialect Dialect
	args    *namedArgList
}

func NewQueryGen(dialect Dialect) QueryGen {
	return QueryGen{
		dialect: dialect,
	}
}

func NewNopQueryGen() QueryGen {
	return nopQueryGen
}

func (f QueryGen) IsNop() bool {
	return f.dialect.Name() == dialect.Invalid
}

func (f QueryGen) Dialect() Dialect {
	return f.dialect
}

func (f QueryGen) IdentQuote() byte {
	return f.dialect.IdentQuote()
}

func (gen QueryGen) Append(b []byte, v any) []byte {
	switch v := v.(type) {
	case nil:
		return dialect.AppendNull(b)
	case bool:
		return dialect.AppendBool(b, v)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint:
		return strconv.AppendInt(b, int64(v), 10)
	case uint32:
		return gen.Dialect().AppendUint32(b, v)
	case uint64:
		return gen.Dialect().AppendUint64(b, v)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return gen.Dialect().AppendString(b, v)
	case time.Time:
		return gen.Dialect().AppendTime(b, v)
	case []byte:
		return gen.Dialect().AppendBytes(b, v)
	case QueryAppender:
		return AppendQueryAppender(gen, b, v)
	default:
		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Ptr && vv.IsNil() {
			return dialect.AppendNull(b)
		}
		appender := Appender(gen.Dialect(), vv.Type())
		return appender(gen, b, vv)
	}
}

func (f QueryGen) AppendName(b []byte, name string) []byte {
	return dialect.AppendName(b, name, f.IdentQuote())
}

func (f QueryGen) AppendIdent(b []byte, ident string) []byte {
	return dialect.AppendIdent(b, ident, f.IdentQuote())
}

func (f QueryGen) AppendValue(b []byte, v reflect.Value) []byte {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return dialect.AppendNull(b)
	}
	appender := Appender(f.dialect, v.Type())
	return appender(f, b, v)
}

func (f QueryGen) HasFeature(feature feature.Feature) bool {
	return f.dialect.Features().Has(feature)
}

func (f QueryGen) WithArg(arg NamedArgAppender) QueryGen {
	return QueryGen{
		dialect: f.dialect,
		args:    f.args.WithArg(arg),
	}
}

func (f QueryGen) WithNamedArg(name string, value any) QueryGen {
	return QueryGen{
		dialect: f.dialect,
		args:    f.args.WithArg(&namedArg{name: name, value: value}),
	}
}

func (f QueryGen) FormatQuery(query string, args ...any) string {
	if f.IsNop() || (args == nil && f.args == nil) || strings.IndexByte(query, '?') == -1 {
		return query
	}
	return internal.String(f.AppendQuery(nil, query, args...))
}

func (f QueryGen) AppendQuery(dst []byte, query string, args ...any) []byte {
	if f.IsNop() || (args == nil && f.args == nil) || strings.IndexByte(query, '?') == -1 {
		return append(dst, query...)
	}
	return f.append(dst, parser.NewString(query), args)
}

func (f QueryGen) append(dst []byte, p *parser.Parser, args []any) []byte {
	var namedArgs NamedArgAppender
	if len(args) == 1 {
		if v, ok := args[0].(NamedArgAppender); ok {
			namedArgs = v
		} else if v, ok := newStructArgs(f, args[0]); ok {
			namedArgs = v
		}
	}

	var argIndex int
	for p.Valid() {
		b, ok := p.ReadSep('?')
		if !ok {
			dst = append(dst, b...)
			continue
		}
		if len(b) > 0 && b[len(b)-1] == '\\' {
			dst = append(dst, b[:len(b)-1]...)
			dst = append(dst, '?')
			continue
		}
		dst = append(dst, b...)

		name, numeric := p.ReadIdentifier()
		if name != "" {
			if numeric {
				idx, err := strconv.Atoi(name)
				if err != nil {
					goto restore_arg
				}

				if idx >= len(args) {
					goto restore_arg
				}

				dst = f.appendArg(dst, args[idx])
				continue
			}

			if namedArgs != nil {
				dst, ok = namedArgs.AppendNamedArg(f, dst, name)
				if ok {
					continue
				}
			}

			dst, ok = f.args.AppendNamedArg(f, dst, name)
			if ok {
				continue
			}

		restore_arg:
			dst = append(dst, '?')
			dst = append(dst, name...)
			continue
		}

		if argIndex >= len(args) {
			dst = append(dst, '?')
			continue
		}

		arg := args[argIndex]
		argIndex++

		dst = f.appendArg(dst, arg)
	}

	return dst
}

func (gen QueryGen) appendArg(b []byte, arg any) []byte {
	switch arg := arg.(type) {
	case QueryAppender:
		bb, err := arg.AppendQuery(gen, b)
		if err != nil {
			return dialect.AppendError(b, err)
		}
		return bb
	default:
		return gen.Append(b, arg)
	}
}

//------------------------------------------------------------------------------

type NamedArgAppender interface {
	AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool)
}

type namedArgList struct {
	arg  NamedArgAppender
	next *namedArgList
}

func (l *namedArgList) WithArg(arg NamedArgAppender) *namedArgList {
	return &namedArgList{
		arg:  arg,
		next: l,
	}
}

func (l *namedArgList) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	for l != nil && l.arg != nil {
		if b, ok := l.arg.AppendNamedArg(gen, b, name); ok {
			return b, true
		}
		l = l.next
	}
	return b, false
}

//------------------------------------------------------------------------------

type namedArg struct {
	name  string
	value any
}

var _ NamedArgAppender = (*namedArg)(nil)

func (a *namedArg) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	if a.name == name {
		return gen.appendArg(b, a.value), true
	}
	return b, false
}

//------------------------------------------------------------------------------

type structArgs struct {
	table *Table
	strct reflect.Value
}

var _ NamedArgAppender = (*structArgs)(nil)

func newStructArgs(gen QueryGen, strct any) (*structArgs, bool) {
	v := reflect.ValueOf(strct)
	if !v.IsValid() {
		return nil, false
	}

	v = reflect.Indirect(v)
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	return &structArgs{
		table: gen.Dialect().Tables().Get(v.Type()),
		strct: v,
	}, true
}

func (m *structArgs) AppendNamedArg(gen QueryGen, b []byte, name string) ([]byte, bool) {
	return m.table.AppendNamedArg(gen, b, name, m.strct)
}
