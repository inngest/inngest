// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ohler55/ojg/alt"
	"github.com/ohler55/ojg/gen"
)

type delFlagType struct{}

var delFlag = &delFlagType{}

// MustDel removes matching nodes and panics on error.
func (x Expr) MustDel(data any) {
	if err := x.set(data, delFlag, "delete", false); err != nil {
		panic(err)
	}
}

// Del removes matching nodes.
func (x Expr) Del(data any) error {
	return x.set(data, delFlag, "delete", false)
}

// MustDelOne removes one matching node and panics on error.
func (x Expr) MustDelOne(data any) {
	if err := x.set(data, delFlag, "delete", true); err != nil {
		panic(err)
	}
}

// DelOne removes at most one node.
func (x Expr) DelOne(data any) error {
	return x.set(data, delFlag, "delete", true)
}

// MustSet all matching child node values. If the path to the child does not
// exist array and map elements are added. Panics on error.
func (x Expr) MustSet(data, value any) {
	if err := x.set(data, value, "set", false); err != nil {
		panic(err)
	}
}

// Set all matching child node values. An error is returned if it is not
// possible. If the path to the child does not exist array and map elements
// are added.
func (x Expr) Set(data, value any) error {
	return x.set(data, value, "set", false)
}

// SetOne child node value. An error is returned if it is not possible. If the
// path to the child does not exist array and map elements are added.
func (x Expr) SetOne(data, value any) error {
	return x.set(data, value, "set", true)
}

// MustSetOne child node value. If the path to the child does not exist array
// and map elements are added. Panics on error.
func (x Expr) MustSetOne(data, value any) {
	if err := x.set(data, value, "set", true); err != nil {
		panic(err)
	}
}

func (x Expr) set(data, value any, fun string, one bool) error {
	if len(x) == 0 {
		return fmt.Errorf("can not %s with an empty expression", fun)
	}
	switch x[len(x)-1].(type) {
	case Root, At, Bracket, Descent, Slice, *Filter:
		ta := strings.Split(fmt.Sprintf("%T", x[len(x)-1]), ".")
		return fmt.Errorf("can not %s with an expression ending with a %s", fun, ta[len(ta)-1])
	}
	var (
		v  any
		nv gen.Node
	)
	_, isNode := data.(gen.Node)
	nodeValue, ok := value.(gen.Node)
	if isNode && !ok {
		if value != nil {
			if v = alt.Generify(value); v == nil {
				return fmt.Errorf("can not %s a %T in a %T", fun, value, data)
			}
			nodeValue, _ = v.(gen.Node)
		}
	}
	var prev any
	stack := make([]any, 0, 64)
	stack = append(stack, data)

	f := x[0]
	fi := fragIndex(0) // frag index
	stack = append(stack, fi)

	for 1 < len(stack) {
		prev = stack[len(stack)-2]
		if ii, up := prev.(fragIndex); up {
			stack[len(stack)-1] = nil
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = x[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack[len(stack)-1] = nil
		stack = stack[:len(stack)-1]
		switch tf := f.(type) {
		case Child:
			var has bool
			switch tv := prev.(type) {
			case map[string]any:
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						delete(tv, string(tf))
					} else {
						tv[string(tf)] = value
					}
					if one {
						return nil
					}
				} else if v, has = tv[string(tf)]; has {
					switch v.(type) {
					case nil, gen.Bool, gen.Int, gen.Float, gen.String,
						bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						kind := reflect.Invalid
						if rt := reflect.TypeOf(v); rt != nil {
							kind = rt.Kind()
						}
						switch kind {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
						default:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						}
					}
				} else if value != delFlag {
					switch tc := x[fi+1].(type) {
					case Child:
						v = map[string]any{}
						tv[string(tf)] = v
						stack = append(stack, v)
					case Nth:
						if int(tc) < 0 {
							return fmt.Errorf("can not deduce the length of the array to add at '%s'", x[:fi+1])
						}
						v = make([]any, int(tc)+1)
						tv[string(tf)] = v
						stack = append(stack, v)
					default:
						return fmt.Errorf("can not deduce what element to add at '%s'", x[:fi+1])
					}
				}
			case Keyed:
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						tv.RemoveValueForKey(string(tf))
					} else {
						tv.SetValueForKey(string(tf), value)
					}
					if one {
						return nil
					}
				} else if v, has = tv.ValueForKey(string(tf)); has {
					switch v.(type) {
					case nil, gen.Bool, gen.Int, gen.Float, gen.String,
						bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						kind := reflect.Invalid
						if rt := reflect.TypeOf(v); rt != nil {
							kind = rt.Kind()
						}
						switch kind {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
						default:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						}
					}
				} else if value != delFlag {
					switch tc := x[fi+1].(type) {
					case Child:
						v = map[string]any{}
						tv.SetValueForKey(string(tf), v)
						stack = append(stack, v)
					case Nth:
						if int(tc) < 0 {
							return fmt.Errorf("can not deduce the length of the array to add at '%s'", x[:fi+1])
						}
						v = make([]any, int(tc)+1)
						tv.SetValueForKey(string(tf), v)
						stack = append(stack, v)
					default:
						return fmt.Errorf("can not deduce what element to add at '%s'", x[:fi+1])
					}
				}
			case gen.Object:
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						delete(tv, string(tf))
					} else {
						tv[string(tf)] = nodeValue
					}
					if one {
						return nil
					}
				} else if v, has = tv[string(tf)]; has {
					switch v.(type) {
					case gen.Object, gen.Array:
						stack = append(stack, v)
					default:
						return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
					}
				} else if value != delFlag {
					switch tc := x[fi+1].(type) {
					case Child:
						nv = gen.Object{}
						tv[string(tf)] = nv
						stack = append(stack, nv)
					case Nth:
						if int(tc) < 0 {
							return fmt.Errorf("can not deduce the length of the array to add at '%s'", x[:fi+1])
						}
						nv = make(gen.Array, int(tc)+1)
						tv[string(tf)] = nv
						stack = append(stack, nv)
					default:
						return fmt.Errorf("can not deduce what element to add at '%s'", x[:fi+1])
					}
				}
			default:
				if int(fi) == len(x)-1 { // last one
					if value != delFlag {
						if x.reflectSetChild(tv, string(tf), value) && one {
							return nil
						}
					}
				} else if v, has = x.reflectGetChild(tv, string(tf)); has {
					switch v.(type) {
					case nil, gen.Bool, gen.Int, gen.Float, gen.String,
						bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						kind := reflect.Invalid
						if rt := reflect.TypeOf(v); rt != nil {
							kind = rt.Kind()
						}
						switch kind {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
						default:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						}
					}
				}
			}
		case Nth:
			i := int(tf)
			switch tv := prev.(type) {
			case []any:
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					if int(fi) == len(x)-1 { // last one
						if value == delFlag {
							tv[i] = nil
						} else {
							tv[i] = value
						}
						if one {
							return nil
						}
					} else {
						v = tv[i]
						switch v.(type) {
						case bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64,
							nil, gen.Bool, gen.Int, gen.Float, gen.String:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							default:
								return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
							}
						}
					}
				} else {
					return fmt.Errorf("can not follow out of bounds array index at '%s'", x[:fi+1])
				}
			case Indexed:
				size := tv.Size()
				if i < 0 {
					i = size + i
				}
				if 0 <= i && i < size {
					if int(fi) == len(x)-1 { // last one
						if value == delFlag {
							tv.SetValueAtIndex(i, nil)
						} else {
							tv.SetValueAtIndex(i, value)
						}
						if one {
							return nil
						}
					} else {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64,
							nil, gen.Bool, gen.Int, gen.Float, gen.String:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							default:
								return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
							}
						}
					}
				} else {
					return fmt.Errorf("can not follow out of bounds array index at '%s'", x[:fi+1])
				}
			case gen.Array:
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					if int(fi) == len(x)-1 { // last one
						if value == delFlag {
							tv[i] = nil
						} else {
							tv[i] = nodeValue
						}
						if one {
							return nil
						}
					} else {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						default:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						}
					}
				} else {
					return fmt.Errorf("can not follow out of bounds array index at '%s'", x[:fi+1])
				}
			default:
				var has bool
				if int(fi) == len(x)-1 { // last one
					if value != delFlag {
						if x.reflectSetNth(tv, i, value) && one {
							return nil
						}
					}
				} else if v, has = x.reflectGetNth(tv, i); has {
					switch v.(type) {
					case bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64,
						nil, gen.Bool, gen.Int, gen.Float, gen.String:
						return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						kind := reflect.Invalid
						if rt := reflect.TypeOf(v); rt != nil {
							kind = rt.Kind()
						}
						switch kind {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
						default:
							return fmt.Errorf("can not follow a %T at '%s'", v, x[:fi+1])
						}
					}
				}
			}
		case Wildcard:
			switch tv := prev.(type) {
			case map[string]any:
				var k string
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						for k = range tv {
							delete(tv, k)
							if one {
								return nil
							}
						}
					} else {
						for k = range tv {
							tv[k] = value
							if one {
								return nil
							}
						}
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case []any:
				if int(fi) == len(x)-1 { // last one
					for i := range tv {
						if value == delFlag {
							tv[i] = nil
						} else {
							tv[i] = value
						}
						if one {
							return nil
						}
					}
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case Keyed:
				keys := tv.Keys()
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						for _, k := range keys {
							tv.RemoveValueForKey(k)
							if one {
								return nil
							}
						}
					} else {
						for _, k := range keys {
							tv.SetValueForKey(k, value)
							if one {
								return nil
							}
						}
					}
				} else {
					for _, k := range keys {
						v, _ = tv.ValueForKey(k)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if int(fi) == len(x)-1 { // last one
					for i := 0; i < size; i++ {
						if value == delFlag {
							tv.SetValueAtIndex(i, nil)
						} else {
							tv.SetValueAtIndex(i, value)
						}
						if one {
							return nil
						}
					}
				} else {
					for i := size - 1; 0 <= i; i-- {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case gen.Object:
				var k string
				if int(fi) == len(x)-1 { // last one
					if value == delFlag {
						for k = range tv {
							delete(tv, k)
							if one {
								return nil
							}
						}
					} else {
						for k = range tv {
							tv[k] = nodeValue
							if one {
								return nil
							}
						}
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			case gen.Array:
				if int(fi) == len(x)-1 { // last one
					for i := range tv {
						if value == delFlag {
							tv[i] = nil
							if one {
								return nil
							}
						} else {
							tv[i] = nodeValue
							if one {
								return nil
							}
						}
					}
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			default:
				if int(fi) != len(x)-1 {
					var va []any
					if one {
						if vv, has := x.reflectGetWildOne(tv); has {
							va = []any{vv}
						}
					} else {
						va = x.reflectGetWild(tv)
					}
					for _, v = range va {
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			}
		case Descent:
			di, _ := stack[len(stack)-1].(fragIndex)
			// first pass expands, second continues evaluation
			if (di & descentFlag) == 0 {
				switch tv := prev.(type) {
				case map[string]any:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for _, v = range tv {
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				case []any:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				case Keyed:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for _, k := range tv.Keys() {
						v, _ = tv.ValueForKey(k)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				case Indexed:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for i := tv.Size() - 1; 0 <= i; i-- {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				case gen.Object:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for _, v = range tv {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				case gen.Array:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				}
			} else {
				stack = append(stack, prev)
			}
		case Union:
			for _, u := range tf {
				switch tu := u.(type) {
				case string:
					var has bool
					switch tv := prev.(type) {
					case map[string]any:
						if int(fi) == len(x)-1 { // last one
							if value == delFlag {
								delete(tv, tu)
							} else {
								tv[tu] = value
							}
							if one {
								return nil
							}
						} else if v, has = tv[tu]; has {
							switch v.(type) {
							case nil, gen.Bool, gen.Int, gen.Float, gen.String,
								bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								kind := reflect.Invalid
								if rt := reflect.TypeOf(v); rt != nil {
									kind = rt.Kind()
								}
								switch kind {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					case Keyed:
						if int(fi) == len(x)-1 { // last one
							if value == delFlag {
								tv.RemoveValueForKey(tu)
							} else {
								tv.SetValueForKey(tu, value)
							}
							if one {
								return nil
							}
						} else if v, has = tv.ValueForKey(tu); has {
							switch v.(type) {
							case nil, gen.Bool, gen.Int, gen.Float, gen.String,
								bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								kind := reflect.Invalid
								if rt := reflect.TypeOf(v); rt != nil {
									kind = rt.Kind()
								}
								switch kind {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					case gen.Object:
						if int(fi) == len(x)-1 { // last one
							if value == delFlag {
								delete(tv, tu)
							} else {
								tv[tu] = nodeValue
							}
							if one {
								return nil
							}
						} else if v, has = tv[tu]; has {
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							}
						}
					default:
						var has bool
						if int(fi) == len(x)-1 { // last one
							if value != delFlag {
								if x.reflectSetChild(tv, tu, value) && one {
									return nil
								}
							}

						} else if v, has = x.reflectGetChild(tv, tu); has {
							switch v.(type) {
							case nil, gen.Bool, gen.Int, gen.Float, gen.String,
								bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								kind := reflect.Invalid
								if rt := reflect.TypeOf(v); rt != nil {
									kind = rt.Kind()
								}
								switch kind {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				case int64:
					i := int(tu)
					switch tv := prev.(type) {
					case []any:
						if i < 0 {
							i = len(tv) + i
						}
						if 0 <= i && i < len(tv) {
							v = tv[i]
							if int(fi) == len(x)-1 { // last one
								if value == delFlag {
									tv[i] = nil
								} else {
									tv[i] = value
								}
								if one {
									return nil
								}
							} else {
								switch v.(type) {
								case nil, gen.Bool, gen.Int, gen.Float, gen.String,
									bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
								case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
									stack = append(stack, v)
								default:
									kind := reflect.Invalid
									if rt := reflect.TypeOf(v); rt != nil {
										kind = rt.Kind()
									}
									switch kind {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					case Indexed:
						size := tv.Size()
						if i < 0 {
							i = size + i
						}
						if 0 <= i && i < size {
							v = tv.ValueAtIndex(i)
							if int(fi) == len(x)-1 { // last one
								if value == delFlag {
									tv.SetValueAtIndex(i, nil)
								} else {
									tv.SetValueAtIndex(i, value)
								}
								if one {
									return nil
								}
							} else {
								switch v.(type) {
								case nil, gen.Bool, gen.Int, gen.Float, gen.String,
									bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
								case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
									stack = append(stack, v)
								default:
									kind := reflect.Invalid
									if rt := reflect.TypeOf(v); rt != nil {
										kind = rt.Kind()
									}
									switch kind {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					case gen.Array:
						if i < 0 {
							i = len(tv) + i
						}
						if 0 <= i && i < len(tv) {
							v = tv[i]
						}
						if int(fi) == len(x)-1 { // last one
							if value == delFlag {
								tv[i] = nil
							} else {
								tv[i] = nodeValue
							}
							if one {
								return nil
							}
						} else {
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							}
						}
					default:
						var has bool
						if int(fi) == len(x)-1 { // last one
							if value != delFlag {
								if x.reflectSetNth(tv, i, value) && one {
									return nil
								}
							}
						} else if v, has = x.reflectGetNth(tv, i); has {
							switch v.(type) {
							case nil, gen.Bool, gen.Int, gen.Float, gen.String,
								bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								kind := reflect.Invalid
								if rt := reflect.TypeOf(v); rt != nil {
									kind = rt.Kind()
								}
								switch kind {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			}
		case Slice:
			start := 0
			end := -1
			step := 1
			if 0 < len(tf) {
				start = tf[0]
			}
			if 1 < len(tf) {
				end = tf[1]
			}
			if 2 < len(tf) {
				step = tf[2]
			}
			switch tv := prev.(type) {
			case []any:
				if start < 0 {
					start = len(tv) + start
				}
				if end < 0 {
					end = len(tv) + end
				}
				if start < 0 || end < 0 || len(tv) <= start || step == 0 {
					continue
				}
				if len(tv) <= end {
					end = len(tv) - 1
				}
				end = start + ((end - start) / step * step)
				if 0 < step {
					for i := end; start <= i; i -= step {
						v = tv[i]
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				} else {
					for i := end; i <= start; i -= step {
						v = tv[i]
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if start < 0 {
					start = size + start
				}
				if end < 0 {
					end = size + end
				}
				if start < 0 || end < 0 || size <= start || step == 0 {
					continue
				}
				if size <= end {
					end = size - 1
				}
				end = start + ((end - start) / step * step)
				if 0 < step {
					for i := end; start <= i; i -= step {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				} else {
					for i := end; i <= start; i -= step {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			case gen.Array:
				if start < 0 {
					start = len(tv) + start
				}
				if end < 0 {
					end = len(tv) + end
				}
				if start < 0 || end < 0 || len(tv) <= start || step == 0 {
					continue
				}
				if len(tv) <= end {
					end = len(tv) - 1
				}
				end = start + ((end - start) / step * step)
				if 0 < step {
					for i := end; start <= i; i -= step {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				} else {
					for i := end; i <= start; i -= step {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				}
			default:
				if int(fi) != len(x)-1 {
					for _, v = range x.reflectGetSlice(tv, start, end, step) {
						switch v.(type) {
						case nil, gen.Bool, gen.Int, gen.Float, gen.String,
							bool, string, float64, float32, int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							kind := reflect.Invalid
							if rt := reflect.TypeOf(v); rt != nil {
								kind = rt.Kind()
							}
							switch kind {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			}
		case *Filter:
			ns, _ := tf.evalWithRoot(stack, prev, data)
			stack, _ = ns.([]any)
		case Root:
			stack = append(stack, data)
		case At, Bracket:
			stack = append(stack, prev)
		}
		if int(fi) < len(x)-1 {
			if _, ok := stack[len(stack)-1].(fragIndex); !ok {
				fi++
				f = x[fi]
				stack = append(stack, fi)
			}
		}
	}
	return nil
}

func (x Expr) reflectSetChild(data any, key string, v any) bool {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rd = rd.Elem()
		}
		switch rt.Kind() {
		case reflect.Struct:
			rv := rd.FieldByNameFunc(func(k string) bool { return strings.EqualFold(k, key) })
			vv := reflect.ValueOf(v)
			if !vv.IsValid() && v == nil {
				vv = reflect.Zero(rv.Type())
			}
			vt := vv.Type()
			if rv.CanSet() && vt.AssignableTo(rv.Type()) {
				rv.Set(vv)
				return true
			}
		case reflect.Map:
			rk := reflect.ValueOf(key)
			vv := reflect.ValueOf(v)
			if !vv.IsValid() && v == nil {
				vv = reflect.Zero(rt.Elem())
			}
			vt := vv.Type()
			if vt.AssignableTo(rt.Elem()) {
				rd.SetMapIndex(rk, vv)
				return true
			}
		}
	}
	return false
}

func (x Expr) reflectSetNth(data any, i int, v any) bool {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			size := rd.Len()
			if i < 0 {
				i = size + i
			}
			if 0 <= i && i < size {
				rv := rd.Index(i)
				vv := reflect.ValueOf(v)
				vt := vv.Type()
				if rv.CanSet() && vt.AssignableTo(rv.Type()) {
					rv.Set(vv)
					return true
				}
			}
		}
	}
	return false
}
