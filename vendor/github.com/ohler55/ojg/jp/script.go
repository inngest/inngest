// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
)

type nothing int

const userOpCode = 'U'

var (
	// Lower precedence is evaluated first.
	eq     = &op{prec: 3, code: '=', name: "==", cnt: 2}
	neq    = &op{prec: 3, code: 'n', name: "!=", cnt: 2}
	lt     = &op{prec: 3, code: '<', name: "<", cnt: 2}
	gt     = &op{prec: 3, code: '>', name: ">", cnt: 2}
	lte    = &op{prec: 3, code: 'l', name: "<=", cnt: 2}
	gte    = &op{prec: 3, code: 'g', name: ">=", cnt: 2}
	or     = &op{prec: 4, code: '|', name: "||", cnt: 2}
	and    = &op{prec: 4, code: '&', name: "&&", cnt: 2}
	not    = &op{prec: 0, code: '!', name: "!", cnt: 1}
	add    = &op{prec: 2, code: '+', name: "+", cnt: 2}
	sub    = &op{prec: 2, code: '-', name: "-", cnt: 2}
	mult   = &op{prec: 1, code: '*', name: "*", cnt: 2}
	divide = &op{prec: 1, code: '/', name: "/", cnt: 2}
	get    = &op{prec: 0, code: 'G', name: "get", cnt: 1}
	in     = &op{prec: 3, code: 'i', name: "in", cnt: 2}
	empty  = &op{prec: 3, code: 'e', name: "empty", cnt: 2}
	rx     = &op{prec: 3, code: '~', name: "~=", cnt: 2}
	rxa    = &op{prec: 3, code: '~', name: "=~", cnt: 2}
	has    = &op{prec: 3, code: 'h', name: "has", cnt: 2}
	exists = &op{prec: 3, code: 'x', name: "exists", cnt: 2}
	// functions
	length = &op{prec: 0, code: 'L', name: "length", cnt: 1}
	count  = &op{prec: 0, code: 'C', name: "count", cnt: 1, getLeft: true}
	match  = &op{prec: 0, code: 'M', name: "match", cnt: 2}
	search = &op{prec: 0, code: 'S', name: "search", cnt: 2}

	// group is for an equation inside () so it represents the (). It should
	// not be in the opMap.
	group = &op{prec: 0, code: '(', name: "(", cnt: 1}

	opMap = map[string]*op{
		eq.name:     eq,
		neq.name:    neq,
		lt.name:     lt,
		gt.name:     gt,
		lte.name:    lte,
		gte.name:    gte,
		or.name:     or,
		and.name:    and,
		not.name:    not,
		add.name:    add,
		sub.name:    sub,
		mult.name:   mult,
		divide.name: divide,
		in.name:     in,
		empty.name:  empty,
		has.name:    has,
		exists.name: exists,
		rx.name:     rx,
		rxa.name:    rx,

		length.name: length,
		count.name:  count,
		match.name:  match,
		search.name: search,
	}
	// Nothing can be used in scripts to indicate no value as in a script such
	// as [?(@.x == Nothing)] this indicates there was no value as @.x. It is
	// the same as [?(@.x has false)] or [?(@.x exists false)].
	Nothing = nothing(0)
)

type op struct {
	name     string
	uniFun   func(arg any) any
	duoFun   func(left, right any) any
	prec     byte
	cnt      byte
	code     byte
	getLeft  bool
	getRight bool
}

type precBuf struct {
	prec byte
	buf  []byte
}

type multivalue []any

// Script represents JSON Path script used in filters as well.
type Script struct {
	template []any
}

// NewScript parses the string argument and returns a script or an error.
func NewScript(str string) (s *Script, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	s = MustNewScript(str)
	return
}

// MustNewScript parses the string argument and returns a script or an error.
func MustNewScript(str string) (s *Script) {
	return MustParseEquation(str).Script()
}

// Append a string representation of the fragment to the buffer and then
// return the expanded buffer.
func (s *Script) Append(buf []byte) []byte {
	buf = append(buf, '(')
	if 0 < len(s.template) {
		bstack := make([]any, len(s.template))
		copy(bstack, s.template)

		for i := len(bstack) - 1; 0 <= i; i-- {
			o, _ := bstack[i].(*op)
			if o == nil {
				if i == 0 {
					buf = s.appendValue(buf, bstack[i], 0)
				}
				continue
			}
			var (
				left  any
				right any
			)
			if 1 < len(bstack)-i {
				left = bstack[i+1]
			}
			if 2 < len(bstack)-i {
				right = bstack[i+2]
			}
			bstack[i] = s.appendOp(o, left, right)
			if i+int(o.cnt)+1 <= len(bstack) {
				copy(bstack[i+1:], bstack[i+int(o.cnt)+1:])
			}
		}
		if pb, _ := bstack[0].(*precBuf); pb != nil {
			buf = append(buf, pb.buf...)
		}
	}
	buf = append(buf, ')')

	return buf
}

// String representation of the script.
func (s *Script) String() string {
	return string(s.Append([]byte{}))
}

// Match returns true if the script returns true when evaluated against the
// data argument.
func (s *Script) Match(data any) bool {
	stack := []any{}
	if node, ok := data.(gen.Node); ok {
		ns, _ := s.evalWithRoot(stack, gen.Array{node}, data)
		stack, _ = ns.([]any)
	} else {
		ns, _ := s.evalWithRoot(stack, []any{data}, data)
		stack, _ = ns.([]any)
	}
	return 0 < len(stack)
}

// Eval is primarily used by the Expr parser but is public for testing.
func (s *Script) Eval(stack, data any) any {
	ns, _ := s.evalWithRoot(stack, data, nil)
	return ns
}

func (s *Script) evalWithRoot(stack, data, root any) (any, Expr) {
	// Checking the type each iteration adds 2.5% but allows code not to be
	// duplicated and not to call a separate function. Using just one more
	// function call for each iteration adds 6.5%.
	var (
		dlen    int
		locKeys Expr
		locs    Expr
	)
	switch td := data.(type) {
	case []any:
		dlen = len(td)
	case gen.Array:
		dlen = len(td)
	case map[string]any:
		dlen = len(td)
		da := make([]any, 0, dlen)
		for k, v := range td {
			da = append(da, v)
			locKeys = append(locKeys, Child(k))
		}
		data = da
	case Indexed:
		dlen = td.Size()
	case Keyed:
		keys := td.Keys()
		dlen = len(keys)
		da := make([]any, dlen)
		for i, k := range keys {
			da[i], _ = td.ValueForKey(k)
			locKeys = append(locKeys, Child(k))
		}
		data = da
	case gen.Object:
		dlen = len(td)
		da := make(gen.Array, 0, dlen)
		for k, v := range td {
			da = append(da, v)
			locKeys = append(locKeys, Child(k))
		}
		data = da
	default:
		rv := reflect.ValueOf(td)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return stack, locs
		}
		dlen = rv.Len()
		da := make([]any, 0, dlen)
		for i := 0; i < dlen; i++ {
			da = append(da, rv.Index(i).Interface())
			locKeys = append(locKeys, Nth(i))
		}
		data = da
	}
	sstack := make([]any, len(s.template))
	var v any
	for vi := dlen - 1; 0 <= vi; vi-- {
		switch td := data.(type) {
		case []any:
			v = td[vi]
		case Indexed:
			v = td.ValueAtIndex(vi)
		case gen.Array:
			v = td[vi]
		}
		// Eval script for each member of the list.
		copy(sstack, s.template)
		var multi bool
		// resolve all expr members
		for i, ev := range sstack {
			if 0 < i {
				if o, ok := sstack[i-1].(*op); ok && o.getLeft {
					var x Expr
					if x, ok = ev.(Expr); ok {
						ev = x.Get(v)
					} else {
						ev = nil
					}
					sstack[i] = ev
				}
				// TBD one more for getRight once function extensions are supported
			}
			// Normalize into nil, bool, int64, float64, and string early so
			// that each comparison doesn't have to.
		Normalize:
			switch x := ev.(type) {
			case Expr:
				var has bool
				dv := v
				switch x[0].(type) {
				case At:
					// The most common pattern is [?(@.child == value)] where
					// the operation and value vary but the @.child is the
					// most widely used. For that reason an optimization is
					// included for that condition of a one level child lookup
					// path.
					if m, ok := v.(map[string]any); ok && len(x) == 2 {
						var c Child
						if c, ok = x[1].(Child); ok {
							if ev, has = m[string(c)]; has {
								sstack[i] = ev
								goto Normalize
							} else {
								sstack[i] = Nothing
							}
						}
					}
				case Root:
					dv = root
				}
				if x.Normal() {
					if ev, has = x.FirstFound(dv); has {
						sstack[i] = ev
						goto Normalize
					} else {
						sstack[i] = Nothing
					}
				} else {
					values := x.Get(dv)
					switch len(values) {
					case 0:
						sstack[i] = Nothing
					case 1:
						sstack[i] = normalize(values[0])
					default:
						multi = true
						mval := make(multivalue, len(values))
						for gi, gv := range values {
							mval[gi] = normalize(gv)
						}
						sstack[i] = mval
					}
				}
			case int:
				sstack[i] = int64(x)
			case int8:
				sstack[i] = int64(x)
			case int16:
				sstack[i] = int64(x)
			case int32:
				sstack[i] = int64(x)
			case uint:
				sstack[i] = int64(x)
			case uint8:
				sstack[i] = int64(x)
			case uint16:
				sstack[i] = int64(x)
			case uint32:
				sstack[i] = int64(x)
			case uint64:
				sstack[i] = int64(x)
			case float32:
				sstack[i] = float64(x)
			case gen.Bool:
				sstack[i] = bool(x)
			case gen.String:
				sstack[i] = string(x)
			case gen.Int:
				sstack[i] = int64(x)
			case gen.Float:
				sstack[i] = float64(x)

			default:
				// Any other type are already simplified or are not
				// handled and will fail later.
			}
		}
		var match bool
		if multi {
			max := 1
			for _, v := range sstack {
				if mv, ok := v.(multivalue); ok {
					max *= len(mv)
				}
			}
			for mi := 0; mi < max; mi++ {
				xstack := evalStack(expandStack(sstack, mi))
				if match, _ = xstack[0].(bool); match {
					break
				}
			}
		} else {
			sstack = evalStack(sstack)
			match, _ = sstack[0].(bool)
		}
		if match {
			switch tstack := stack.(type) {
			case []any:
				tstack = append(tstack, v)
				if 0 < len(locKeys) {
					locs = append(locs, locKeys[vi])
				} else {
					locs = append(locs, Nth(vi))
				}
				stack = tstack
			case []gen.Node:
				if n, ok := v.(gen.Node); ok {
					tstack = append(tstack, n)
					if 0 < len(locKeys) {
						locs = append(locs, locKeys[vi])
					} else {
						locs = append(locs, Nth(vi))
					}
					stack = tstack
				}
			}
		}
	}
	for i := range sstack {
		sstack[i] = nil
	}
	return stack, locs
}

func normalize(v any) any {
	switch tv := v.(type) {
	case int:
		v = int64(tv)
	case int8:
		v = int64(tv)
	case int16:
		v = int64(tv)
	case int32:
		v = int64(tv)
	case uint:
		v = int64(tv)
	case uint8:
		v = int64(tv)
	case uint16:
		v = int64(tv)
	case uint32:
		v = int64(tv)
	case uint64:
		v = int64(tv)
	case float32:
		v = float64(tv)
	case gen.Bool:
		v = bool(tv)
	case gen.String:
		v = string(tv)
	case gen.Int:
		v = int64(tv)
	case gen.Float:
		v = float64(tv)
	}
	return v
}

func expandStack(stack []any, mi int) []any {
	nstack := make([]any, len(stack))
	for i, v := range stack {
		if mv, ok := v.(multivalue); ok {
			nstack[i] = mv[mi%len(mv)]
			mi /= len(mv)
		} else {
			nstack[i] = v
		}
	}
	return nstack
}

func evalStack(sstack []any) []any {
	for i := len(sstack) - 1; 0 <= i; i-- {
		o, _ := sstack[i].(*op)
		if o == nil {
			// a value, not an op
			continue
		}
		var (
			left  any
			right any
		)
		if 1 < len(sstack)-i {
			left = sstack[i+1]
		}
		if 2 < len(sstack)-i {
			right = sstack[i+2]
		}
		switch o.code {
		case group.code:
			sstack[i] = left
		case eq.code:
			if left == right {
				sstack[i] = true
			} else {
				sstack[i] = false
				switch tl := left.(type) {
				case int64:
					if tr, ok := right.(float64); ok {
						sstack[i] = ok && float64(tl) == tr
					}
				case float64:
					tr, ok := right.(int64)
					sstack[i] = ok && tl == float64(tr)
				}
			}
		case neq.code:
			if left == right {
				sstack[i] = false
			} else {
				sstack[i] = true
				switch tl := left.(type) {
				case int64:
					if tr, ok := right.(float64); ok {
						sstack[i] = ok && float64(tl) != tr
					}
				case float64:
					tr, ok := right.(int64)
					sstack[i] = ok && tl != float64(tr)
				}
			}
		case lt.code:
			sstack[i] = false
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl < tr
				case float64:
					sstack[i] = float64(tl) < tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl < float64(tr)
				case float64:
					sstack[i] = tl < tr
				}
			case string:
				tr, ok := right.(string)
				sstack[i] = ok && tl < tr
			}
		case gt.code:
			sstack[i] = false
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl > tr
				case float64:
					sstack[i] = float64(tl) > tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl > float64(tr)
				case float64:
					sstack[i] = tl > tr
				}
			case string:
				tr, ok := right.(string)
				sstack[i] = ok && tl > tr
			}
		case lte.code:
			sstack[i] = false
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl <= tr
				case float64:
					sstack[i] = float64(tl) <= tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl <= float64(tr)
				case float64:
					sstack[i] = tl <= tr
				}
			case string:
				tr, ok := right.(string)
				sstack[i] = ok && tl <= tr
			}
		case gte.code:
			sstack[i] = false
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl >= tr
				case float64:
					sstack[i] = float64(tl) >= tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl >= float64(tr)
				case float64:
					sstack[i] = tl >= tr
				}
			case string:
				tr, ok := right.(string)
				sstack[i] = ok && tl >= tr
			}
		case or.code:
			// If one is a boolean true then true.
			lb, _ := left.(bool)
			rb, _ := right.(bool)
			sstack[i] = lb || rb
		case and.code:
			// If both are a boolean true then true else false.
			lb, _ := left.(bool)
			rb, _ := right.(bool)
			sstack[i] = lb && rb
		case not.code:
			lb, _ := left.(bool)
			sstack[i] = !lb
		case add.code:
			sstack[i] = Nothing
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl + tr
				case float64:
					sstack[i] = float64(tl) + tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl + float64(tr)
				case float64:
					sstack[i] = tl + tr
				}
			case string:
				if tr, ok := right.(string); ok {
					sstack[i] = tl + tr
				}
			}
		case sub.code:
			sstack[i] = Nothing
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl - tr
				case float64:
					sstack[i] = float64(tl) - tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl - float64(tr)
				case float64:
					sstack[i] = tl - tr
				}
			}
		case mult.code:
			sstack[i] = Nothing
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl * tr
				case float64:
					sstack[i] = float64(tl) * tr
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					sstack[i] = tl * float64(tr)
				case float64:
					sstack[i] = tl * tr
				}
			}
		case divide.code:
			sstack[i] = Nothing
			switch tl := left.(type) {
			case int64:
				switch tr := right.(type) {
				case int64:
					if tr != 0 {
						sstack[i] = tl / tr
					}
				case float64:
					if tr != 0.0 {
						sstack[i] = float64(tl) / tr

					}
				}
			case float64:
				switch tr := right.(type) {
				case int64:
					if tr != 0 {
						sstack[i] = tl / float64(tr)
					}
				case float64:
					if tr != 0.0 {
						sstack[i] = tl / tr
					}
				}
			}
		case in.code:
			sstack[i] = false
			if list, ok := right.([]any); ok {
				for _, ev := range list {
					if left == ev {
						sstack[i] = true
						break
					}
				}
			}
		case empty.code:
			sstack[i] = false
			if boo, ok := right.(bool); ok {
				switch tl := left.(type) {
				case string:
					sstack[i] = boo == (len(tl) == 0)
				case []any:
					sstack[i] = boo == (len(tl) == 0)
				case map[string]any:
					sstack[i] = boo == (len(tl) == 0)
				}
			}
		case has.code, exists.code:
			sstack[i] = false
			if boo, ok := right.(bool); ok {
				sstack[i] = boo == (left != Nothing)
			}
		case rx.code:
			sstack[i] = false
			ls, ok := left.(string)
			if !ok {
				break
			}
			switch tr := right.(type) {
			case string:
				if rx, err := regexp.Compile(tr); err == nil {
					sstack[i] = rx.MatchString(ls)
				}
			case *regexp.Regexp:
				sstack[i] = tr.MatchString(ls)
			}
		case length.code:
			sstack[i] = Nothing
			switch tl := left.(type) {
			case string:
				sstack[i] = int64(len(tl))
			case []any:
				sstack[i] = int64(len(tl))
			case map[string]any:
				sstack[i] = int64(len(tl))
			}
		case count.code:
			sstack[i] = Nothing
			if nl, ok := left.([]any); ok {
				sstack[i] = int64(len(nl))
			}
		case match.code:
			sstack[i] = Nothing
			if ls, ok := left.(string); ok {
				if rs, _ := right.(string); 0 < len(rs) {
					if rs[0] != '^' {
						rs = "^" + rs
					}
					if rs[len(rs)-1] != '$' {
						rs += "$"
					}
					if rx, err := regexp.Compile(rs); err == nil {
						sstack[i] = rx.MatchString(ls)
					}
				}
			}
		case search.code:
			sstack[i] = Nothing
			if ls, ok := left.(string); ok {
				if rs, _ := right.(string); 0 < len(rs) {
					if rx, err := regexp.Compile(rs); err == nil {
						sstack[i] = rx.MatchString(ls)
					}
				}
			}
		default:
			if o.uniFun != nil {
				sstack[i] = o.uniFun(left)
			} else if o.duoFun != nil {
				sstack[i] = o.duoFun(left, right)
			}
		}
		if i+int(o.cnt)+1 <= len(sstack) {
			copy(sstack[i+1:], sstack[i+int(o.cnt)+1:])
		}
	}
	return sstack
}

// Inspect the script.
func (s *Script) Inspect() *Form {
	f, _ := nextForm(s.template)

	return f.(*Form)
}

func nextForm(st []any) (any, []any) {
	var v any
	if 0 < len(st) {
		v = st[0]
		st = st[1:]
		if ov, ok := v.(*op); ok {
			f := Form{Op: ov.name}
			f.Left, st = nextForm(st)
			f.Right, st = nextForm(st)
			v = &f
		}
	}
	return v, st
}

func (s *Script) appendOp(o *op, left, right any) (pb *precBuf) {
	pb = &precBuf{prec: o.prec}
	switch o.code {
	case not.code:
		pb.buf = append(pb.buf, o.name...)
		pb.buf = s.appendValue(pb.buf, left, o.prec)
	case group.code:
		pb.buf = s.appendValue(pb.buf, left, o.prec)
	case length.code, count.code:
		pb.buf = append(pb.buf, o.name...)
		pb.buf = append(pb.buf, '(')
		pb.buf = s.appendValue(pb.buf, left, o.prec)
		pb.buf = append(pb.buf, ')')
	case match.code, search.code:
		pb.buf = append(pb.buf, o.name...)
		pb.buf = append(pb.buf, '(')
		pb.buf = s.appendValue(pb.buf, left, o.prec)
		pb.buf = append(pb.buf, ',', ' ')
		pb.buf = s.appendValue(pb.buf, right, o.prec)
		pb.buf = append(pb.buf, ')')
	case userOpCode:
		pb.buf = append(pb.buf, o.name...)
		pb.buf = append(pb.buf, '(')
		pb.buf = s.appendValue(pb.buf, left, o.prec)
		if 1 < o.cnt {
			pb.buf = append(pb.buf, ',', ' ')
			pb.buf = s.appendValue(pb.buf, right, o.prec)
		}
		pb.buf = append(pb.buf, ')')
	default:
		pb.buf = s.appendValue(pb.buf, left, o.prec)
		pb.buf = append(pb.buf, ' ')
		pb.buf = append(pb.buf, o.name...)
		pb.buf = append(pb.buf, ' ')
		pb.buf = s.appendValue(pb.buf, right, o.prec)
	}
	return
}

func (s *Script) appendValue(buf []byte, v any, prec byte) []byte {
	switch tv := v.(type) {
	case nil:
		buf = append(buf, "null"...)
	case nothing:
		buf = append(buf, "Nothing"...)
	case string:
		buf = AppendString(buf, tv, '\'')
	case int64:
		buf = append(buf, strconv.FormatInt(tv, 10)...)
	case float64:
		buf = append(buf, strconv.FormatFloat(tv, 'g', -1, 64)...)
	case bool:
		if tv {
			buf = append(buf, "true"...)
		} else {
			buf = append(buf, "false"...)
		}
	case []any:
		buf = append(buf, '[')
		for i, v := range tv {
			if 0 < i {
				buf = append(buf, ',')
			}
			buf = s.appendValue(buf, v, prec)
		}
		buf = append(buf, ']')
	case Expr:
		buf = tv.Append(buf)
	case *regexp.Regexp:
		buf = AppendString(buf, tv.String(), '/')
	case *precBuf:
		if prec < tv.prec {
			buf = append(buf, '(')
			buf = append(buf, tv.buf...)
			buf = append(buf, ')')
		} else {
			buf = append(buf, tv.buf...)
		}
	}
	return buf
}

var builtInNames = map[string]bool{
	"==":     true,
	"!=":     true,
	"<":      true,
	">":      true,
	"<=":     true,
	">=":     true,
	"||":     true,
	"&&":     true,
	"!":      true,
	"+":      true,
	"-":      true,
	"*":      true,
	"/":      true,
	"get":    true,
	"in":     true,
	"empty":  true,
	"~=":     true,
	"=~":     true,
	"has":    true,
	"exists": true,
	"length": true,
	"count":  true,
	"match":  true,
	"search": true,
	"true":   true,
	"false":  true,
	"null":   true,
}

// RegisterUnaryFunction registers a unary function for scripts. The 'get'
// argument if true indicates a get operation to provide the argument to the
// provided function otherwise the first match is used. Names must be alpha
// characters only.
func RegisterUnaryFunction(name string, get bool, f func(arg any) any) {
	name = strings.ToLower(name)
	if builtInNames[name] {
		panic(fmt.Errorf("operation %s can not be replaced", name))
	}
	opMap[name] = &op{
		name:    name,
		uniFun:  f,
		code:    userOpCode,
		cnt:     1,
		getLeft: get,
	}
}

// RegisterBinaryFunction registers a function that takes two argument for
// scripts. The 'getLeft' and 'getRight' arguments if true indicates a get
// operation to provide the argument to the provided function otherwise the
// first match is used. Names must be alpha characters only.
func RegisterBinaryFunction(name string, getLeft, getRight bool, f func(left, right any) any) {
	name = strings.ToLower(name)
	if builtInNames[name] {
		panic(fmt.Errorf("operation %s can not be replaced", name))
	}
	opMap[name] = &op{
		name:     name,
		duoFun:   f,
		code:     userOpCode,
		cnt:      2,
		getLeft:  getLeft,
		getRight: getRight,
	}
}
