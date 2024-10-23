// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"
	"strings"

	"github.com/ohler55/ojg/gen"
)

const (
	fragIndexMask    = 0x0000ffff
	descentFlag      = 0x00010000
	descentChildFlag = 0x00020000

	// The standard math package fails to compile on 32bit architectures (ARM)
	// with an int overflow. Most likley due to math.MaxInt64 being defined as
	// 1<<63 - 1 which default to integer values. Since arrays are not likely
	// to be over 2147483647 on a 32 bit system that is set as the max end
	// specifier for a array range.
	maxEnd = 2147483647
)

type fragIndex int

// The easy way to implement the Get is to have each fragment handle the
// getting using recursion. The overhead of a go function call is rather high
// though so instead a pseudo call stack is implemented here that grows and
// shrinks as the getting takes place. The fragment index if placed on the
// stack as well mostly for a small degree of simplicity in what a few people
// might find a complex approach to the solution. Its at least twice as fast
// as the recursive function call approach and in some cases such as the
// recursive descent more than an order of magnitude faster.

// Get the elements of the data identified by the path.
func (x Expr) Get(data any) (results []any) {
	if len(x) == 0 {
		return
	}
	var v any
	var prev any
	var has bool

	stack := make([]any, 0, 64)
	stack = append(stack, data)

	f := x[0]
	fi := fragIndex(0) // frag index
	stack = append(stack, fi)

	for 1 < len(stack) { // must have at least a data element and a fragment index
		prev = stack[len(stack)-2]
		if ii, up := prev.(fragIndex); up {
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = x[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		has = false
		switch tf := f.(type) {
		case Child:
			switch tv := prev.(type) {
			case map[string]any:
				v, has = tv[string(tf)]
			case gen.Object:
				v, has = tv[string(tf)]
			case Keyed:
				v, has = tv.ValueForKey(string(tf))
			default:
				v, has = x.reflectGetChild(tv, string(tf))
			}
			if has {
				if int(fi) == len(x)-1 { // last one
					results = append(results, v)
				} else {
					switch v.(type) {
					case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
						int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						if rt := reflect.TypeOf(v); rt != nil {
							switch rt.Kind() {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
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
					v = tv[i]
					has = true
				}
			case gen.Array:
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					v = tv[i]
					has = true
				}
			case Indexed:
				if i < 0 {
					i = tv.Size() + i
				}
				if 0 <= i && i < tv.Size() {
					v = tv.ValueAtIndex(i)
					has = true
				}
			default:
				v, has = x.reflectGetNth(tv, i)
			}
			if has {
				if int(fi) == len(x)-1 { // last one
					results = append(results, v)
				} else {
					switch v.(type) {
					case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
						int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						if rt := reflect.TypeOf(v); rt != nil {
							switch rt.Kind() {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			}
		case Wildcard:
			switch tv := prev.(type) {
			case map[string]any:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						results = append(results, v)
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case []any:
				if int(fi) == len(x)-1 { // last one
					results = append(results, tv...)
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case Keyed:
				keys := tv.Keys()
				if int(fi) == len(x)-1 { // last one
					for _, k := range keys {
						v, _ := tv.ValueForKey(k)
						results = append(results, v)
					}
				} else {
					for i := len(keys) - 1; 0 <= i; i-- {
						v, _ := tv.ValueForKey(keys[i])
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if int(fi) == len(x)-1 { // last one
					for i := 0; i < size; i++ {
						results = append(results, tv.ValueAtIndex(i))
					}
				} else {
					for i := size - 1; 0 <= i; i-- {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case gen.Object:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						results = append(results, v)
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				}
			case gen.Array:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						results = append(results, v)
					}
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				}
			default:
				got := x.reflectGetWild(tv)
				if int(fi) == len(x)-1 { // last one
					for i := len(got) - 1; 0 <= i; i-- {
						results = append(results, got[i])
					}
				} else {
					for _, v := range got {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			}
		case Descent:
			di, _ := stack[len(stack)-1].(fragIndex)
			top := (di & descentChildFlag) == 0
			// first pass expands, second continues evaluation
			if (di & descentFlag) == 0 {
				switch tv := prev.(type) {
				case map[string]any:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for _, v = range tv {
							results = append(results, v)
						}
					}
					for _, v = range tv {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
									stack = append(stack, fi|descentChildFlag)
								}
							}
						}
					}
				case []any:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						results = append(results, tv...)
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
									stack = append(stack, fi|descentChildFlag)
								}
							}
						}
					}
				case Keyed:
					keys := tv.Keys()
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for _, k := range keys {
							v, _ := tv.ValueForKey(k)
							results = append(results, v)
						}
					}
					for i := len(keys) - 1; 0 <= i; i-- {
						v, _ := tv.ValueForKey(keys[i])
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
									stack = append(stack, fi|descentChildFlag) // TBD ??
								}
							}
						}
					}
				case Indexed:
					size := tv.Size()
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for i := 0; i < size; i++ {
							results = append(results, tv.ValueAtIndex(i))
						}
					}
					for i := size - 1; 0 <= i; i-- {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
									stack = append(stack, fi|descentChildFlag)
								}
							}
						}
					}
				case gen.Object:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for _, v = range tv {
							results = append(results, v)
						}
					}
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
					if int(fi) == len(x)-1 { // last one
						for _, v = range tv {
							results = append(results, v)
						}
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				default:
					got := x.reflectGetWild(tv)
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for i := len(got) - 1; 0 <= i; i-- {
							results = append(results, got[i])
						}
					} else {
						for _, v := range got {
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
								stack = append(stack, fi|descentChildFlag)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
										stack = append(stack, fi|descentChildFlag)
									}
								}
							}
						}
					}
				}
			} else {
				if int(fi) == len(x)-1 { // last one
					if top {
						results = append(results, prev)
					}
				} else {
					stack = append(stack, prev)
				}
			}
		case Root:
			if int(fi) == len(x)-1 { // last one
				results = append(results, data)
			} else {
				stack = append(stack, data)
			}
		case At, Bracket:
			if int(fi) == len(x)-1 { // last one
				results = append(results, prev)
			} else {
				stack = append(stack, prev)
			}
		case Union:
			if int(fi) == len(x)-1 { // last one
				for _, u := range tf {
					has = false
					switch tu := u.(type) {
					case string:
						switch tv := prev.(type) {
						case map[string]any:
							v, has = tv[tu]
						case Keyed:
							v, has = tv.ValueForKey(tu)
						case gen.Object:
							v, has = tv[tu]
						default:
							v, has = x.reflectGetChild(tv, tu)
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
								has = true
							}
						case Indexed:
							if i < 0 {
								i = tv.Size() + i
							}
							if 0 <= i && i < tv.Size() {
								v = tv.ValueAtIndex(i)
								has = true
							}
						case gen.Array:
							if i < 0 {
								i = len(tv) + i
							}
							if 0 <= i && i < len(tv) {
								v = tv[i]
								has = true
							}
						default:
							v, has = x.reflectGetNth(tv, i)
						}
					}
					if has {
						results = append(results, v)
					}
				}
			} else {
				for ui := len(tf) - 1; 0 <= ui; ui-- {
					u := tf[ui]
					has = false
					switch tu := u.(type) {
					case string:
						switch tv := prev.(type) {
						case map[string]any:
							v, has = tv[tu]
						case Keyed:
							v, has = tv.ValueForKey(tu)
						case gen.Object:
							v, has = tv[tu]
						default:
							v, has = x.reflectGetChild(tv, tu)
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
								has = true
							}
						case Indexed:
							if i < 0 {
								i = tv.Size() + i
							}
							if 0 <= i && i < tv.Size() {
								v = tv.ValueAtIndex(i)
								has = true
							}
						case gen.Array:
							if i < 0 {
								i = len(tv) + i
							}
							if 0 <= i && i < len(tv) {
								v = tv[i]
								has = true
							}
						default:
							v, has = x.reflectGetNth(tv, i)
						}
					}
					if has {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
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
			end := maxEnd
			step := 1
			if 0 < len(tf) {
				start = tf[0]
			}
			if 1 < len(tf) {
				end = tf[1]
			}
			if 2 < len(tf) {
				step = tf[2]
				if step == 0 {
					continue
				}
			}
			switch tv := prev.(type) {
			case []any:
				if start < 0 {
					start = len(tv) + start
					if start < 0 {
						start = 0
					}
				}
				if end < 0 {
					end = len(tv) + end
				}
				if len(tv) <= start {
					continue
				}
				if len(tv) < end {
					end = len(tv)
				}
				if 0 < step {
					if int(fi) == len(x)-1 { // last one
						for i := start; i < end; i += step {
							results = append(results, tv[i])
						}
					} else {
						end = start + (end-start-1)/step*step
						for i := end; start <= i; i -= step {
							v = tv[i]
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 { // last one
						for i := start; end < i; i += step {
							results = append(results, tv[i])
						}
					} else {
						end = start - (start-end-1)/step*step
						for i := end; i <= start; i -= step {
							v = tv[i]
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if start < 0 {
					start = size + start
					if start < 0 {
						start = 0
					}
				}
				if end < 0 {
					end = size + end
				}
				if size <= start {
					continue
				}
				if size < end {
					end = size
				}
				if 0 < step {
					if int(fi) == len(x)-1 { // last one
						for i := start; i < end; i += step {
							results = append(results, tv.ValueAtIndex(i))
						}
					} else {
						end = start + (end-start-1)/step*step
						for i := end; start <= i; i -= step {
							v = tv.ValueAtIndex(i)
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 { // last one
						for i := start; end < i; i += step {
							results = append(results, tv.ValueAtIndex(i))
						}
					} else {
						end = start - (start-end-1)/step*step
						for i := end; i <= start; i -= step {
							v = tv.ValueAtIndex(i)
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					}
				}
			case gen.Array:
				if start < 0 {
					start = len(tv) + start
					if start < 0 {
						start = 0
					}
				}
				if end < 0 {
					end = len(tv) + end
				}
				if len(tv) <= start {
					continue
				}
				if 0 < step {
					if len(tv) < end {
						end = len(tv)
					}
					if int(fi) == len(x)-1 { // last one
						for i := start; i < end; i += step {
							results = append(results, tv[i])
						}
					} else {
						end = start + (end-start-1)/step*step
						for i := end; start <= i; i -= step {
							v = tv[i]
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							}
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 { // last one
						for i := start; end < i; i += step {
							results = append(results, tv[i])
						}
					} else {
						end = start - (start-end-1)/step*step
						for i := end; i <= start; i -= step {
							v = tv[i]
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							}
						}
					}
				}
			default:
				got := x.reflectGetSlice(tv, start, end, step)
				if int(fi) == len(x)-1 { // last one
					for i := len(got) - 1; 0 <= i; i-- {
						results = append(results, got[i])
					}
				} else {
					for _, v := range got {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			}
		case *Filter:
			before := len(stack)
			ns, _ := tf.evalWithRoot(stack, prev, data)
			stack, _ = ns.([]any)
			if int(fi) == len(x)-1 { // last one
				for i := len(stack) - 1; before <= i; i-- {
					results = append(results, stack[i])
				}
				if before < len(stack) {
					stack = stack[:before]
				}
			}
		}
		if int(fi) < len(x)-1 {
			if _, ok := stack[len(stack)-1].(fragIndex); !ok {
				fi++
				f = x[fi]
				stack = append(stack, fi)
			}
		}
	}
	// Free up anything still on the stack.
	stack = stack[0:cap(stack)]
	for i := len(stack) - 1; 0 <= i; i-- {
		stack[i] = nil
	}
	return
}

// First element of the data identified by the path.
func (x Expr) First(data any) any {
	first, _ := x.FirstFound(data)
	return first
}

// FirstFound element of the data identified by the path.
func (x Expr) FirstFound(data any) (any, bool) {
	if len(x) == 0 {
		return nil, false
	}
	var (
		v    any
		prev any
		has  bool
	)
	stack := make([]any, 0, 64)
	defer func() {
		stack = stack[0:cap(stack)]
		for i := len(stack) - 1; 0 <= i; i-- {
			stack[i] = nil
		}
	}()
	stack = append(stack, data)
	f := x[0]
	fi := fragIndex(0) // frag index
	stack = append(stack, fi)

	for 1 < len(stack) { // must have at least a data element and a fragment index
		prev = stack[len(stack)-2]
		if ii, up := prev.(fragIndex); up {
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = x[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		has = false
		switch tf := f.(type) {
		case Child:
			switch tv := prev.(type) {
			case map[string]any:
				v, has = tv[string(tf)]
			case Keyed:
				v, has = tv.ValueForKey(string(tf))
			case gen.Object:
				v, has = tv[string(tf)]
			default:
				v, has = x.reflectGetChild(tv, string(tf))
			}
			if has {
				if int(fi) == len(x)-1 { // last one
					return v, true
				}
				switch v.(type) {
				case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
					int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
				case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
					stack = append(stack, v)
				default:
					if rt := reflect.TypeOf(v); rt != nil {
						switch rt.Kind() {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
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
					v = tv[i]
					has = true
				}
			case Indexed:
				if i < 0 {
					i = tv.Size() + i
				}
				if 0 <= i && i < tv.Size() {
					v = tv.ValueAtIndex(i)
					has = true
				}
			case gen.Array:
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					v = tv[i]
					has = true
				}
			default:
				v, has = x.reflectGetNth(tv, i)
			}
			if has {
				if int(fi) == len(x)-1 { // last one
					return v, true
				}
				switch v.(type) {
				case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
					int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
				case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
					stack = append(stack, v)
				default:
					if rt := reflect.TypeOf(v); rt != nil {
						switch rt.Kind() {
						case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
							stack = append(stack, v)
						}
					}
				}
			}
		case Wildcard:
			switch tv := prev.(type) {
			case map[string]any:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						return v, true
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case []any:
				if int(fi) == len(x)-1 { // last one
					if 0 < len(tv) {
						return tv[0], true
					}
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case Keyed:
				keys := tv.Keys()
				if int(fi) == len(x)-1 { // last one
					if 0 < len(keys) {
						return tv.ValueForKey(keys[0])
					}
				} else {
					for i := len(keys) - 1; 0 <= i; i-- {
						v, _ := tv.ValueForKey(keys[i])
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if int(fi) == len(x)-1 { // last one
					if 0 < size {
						return tv.ValueAtIndex(0), true
					}
				} else {
					for i := size - 1; 0 <= i; i-- {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case gen.Object:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						return v, true
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				}
			case gen.Array:
				if int(fi) == len(x)-1 { // last one
					for _, v = range tv {
						return v, true
					}
				} else {
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						}
					}
				}
			default:
				if v, has = x.reflectGetWildOne(tv); has {
					if int(fi) == len(x)-1 { // last one
						return v, true
					}
					switch v.(type) {
					case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
						int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						if rt := reflect.TypeOf(v); rt != nil {
							switch rt.Kind() {
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
					if int(fi) == len(x)-1 { // last one
						for _, v = range tv {
							return v, true
						}
					}
					for _, v = range tv {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				case []any:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						if 0 < len(tv) {
							return tv[0], true
						}
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				case Keyed:
					keys := tv.Keys()
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						if 0 < len(keys) {
							return tv.ValueForKey(keys[0])
						}
					}
					for i := len(keys) - 1; 0 <= i; i-- {
						v, _ := tv.ValueForKey(keys[i])
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				case Indexed:
					size := tv.Size()
					if int(fi) == len(x)-1 { // last one
						if 0 < size {
							return tv.ValueAtIndex(0), true
						}
					} else {
						for i := size - 1; 0 <= i; i-- {
							v = tv.ValueAtIndex(i)
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
									}
								}
							}
						}
					}
				case gen.Object:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						for _, v = range tv {
							return v, true
						}
					}
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
					if int(fi) == len(x)-1 { // last one
						if 0 < len(tv) {
							return tv[0], true
						}
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				default:
					got := x.reflectGetWild(tv)
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if int(fi) == len(x)-1 { // last one
						if 0 < len(got) {
							return got[0], true
						}
					} else {
						for _, v := range got {
							switch v.(type) {
							case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
								int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
							case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
								stack = append(stack, v)
								stack = append(stack, fi|descentChildFlag)
							default:
								if rt := reflect.TypeOf(v); rt != nil {
									switch rt.Kind() {
									case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
										stack = append(stack, v)
										stack = append(stack, fi|descentChildFlag)
									}
								}
							}
						}
					}

				}
			} else {
				stack = append(stack, prev)
			}
		case Root:
			if int(fi) == len(x)-1 { // last one
				return data, true
			}
			stack = append(stack, data)
		case At, Bracket:
			if int(fi) == len(x)-1 { // last one
				return prev, true
			}
			stack = append(stack, prev)
		case Union:
			if int(fi) == len(x)-1 { // last one
				for _, u := range tf {
					has = false
					switch tu := u.(type) {
					case string:
						switch tv := prev.(type) {
						case map[string]any:
							v, has = tv[tu]
						case Keyed:
							v, has = tv.ValueForKey(tu)
						case gen.Object:
							v, has = tv[tu]
						default:
							v, has = x.reflectGetChild(tv, tu)
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
								has = true
							}
						case Indexed:
							if i < 0 {
								i = tv.Size() + i
							}
							if 0 <= i && i < tv.Size() {
								v = tv.ValueAtIndex(i)
								has = true
							}
						case gen.Array:
							if i < 0 {
								i = len(tv) + i
							}
							if 0 <= i && i < len(tv) {
								v = tv[i]
								has = true
							}
						default:
							v, has = x.reflectGetNth(tv, i)
						}
					}
					if has {
						return v, true
					}
				}
			} else {
				for ui := len(tf) - 1; 0 <= ui; ui-- {
					u := tf[ui]
					has = false
					switch tu := u.(type) {
					case string:
						switch tv := prev.(type) {
						case map[string]any:
							v, has = tv[tu]
						case Keyed:
							v, has = tv.ValueForKey(tu)
						case gen.Object:
							v, has = tv[tu]
						default:
							v, has = x.reflectGetChild(tv, tu)
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
								has = true
							}
						case Indexed:
							if i < 0 {
								i = tv.Size() + i
							}
							if 0 <= i && i < tv.Size() {
								v = tv.ValueAtIndex(i)
								has = true
							}
						case gen.Array:
							if i < 0 {
								i = len(tv) + i
							}
							if 0 <= i && i < len(tv) {
								v = tv[i]
								has = true
							}
						default:
							v, has = x.reflectGetNth(tv, i)
						}
					}
					if has {
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
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
			end := maxEnd
			step := 1
			if 0 < len(tf) {
				start = tf[0]
			}
			if 1 < len(tf) {
				end = tf[1]
			}
			if 2 < len(tf) {
				step = tf[2]
				if step == 0 {
					continue
				}
			}
			switch tv := prev.(type) {
			case []any:
				if start < 0 {
					start = len(tv) + start
					if start < 0 {
						start = 0
					}
				}
				if len(tv) <= start {
					continue
				}
				if end < 0 {
					end = len(tv) + end
				}
				if len(tv) < end {
					end = len(tv)
				}
				if 0 < step {
					if int(fi) == len(x)-1 && start < end { // last one
						return tv[start], true
					}
					end = start + (end-start-1)/step*step
					for i := end; start <= i; i -= step {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 && end < start { // last one
						return tv[start], true
					}
					end = start - (start-end-1)/step*step
					for i := end; i <= start; i -= step {
						v = tv[i]
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case Indexed:
				size := tv.Size()
				if start < 0 {
					start = size + start
					if start < 0 {
						start = 0
					}
				}
				if size <= start {
					continue
				}
				if end < 0 {
					end = size + end
				}
				if size < end {
					end = size
				}
				if 0 < step {
					if int(fi) == len(x)-1 && start < end { // last one
						return tv.ValueAtIndex(start), true
					}
					end = start + (end-start-1)/step*step
					for i := end; start <= i; i -= step {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 && end < start { // last one
						return tv.ValueAtIndex(start), true
					}
					end = start - (start-end-1)/step*step
					for i := end; i <= start; i -= step {
						v = tv.ValueAtIndex(i)
						switch v.(type) {
						case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
							int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
						case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
							stack = append(stack, v)
						default:
							if rt := reflect.TypeOf(v); rt != nil {
								switch rt.Kind() {
								case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
									stack = append(stack, v)
								}
							}
						}
					}
				}
			case gen.Array:
				if start < 0 {
					start = len(tv) + start
					if start < 0 {
						start = 0
					}
				}
				if len(tv) <= start {
					continue
				}
				if end < 0 {
					end = len(tv) + end
				}
				if len(tv) < end {
					end = len(tv)
				}
				if 0 < step {
					if int(fi) == len(x)-1 && start < end { // last one
						return tv[start], true
					}
					end = start + (end-start-1)/step*step
					for i := end; start <= i; i -= step {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				} else {
					if end < -1 {
						end = -1
					}
					if int(fi) == len(x)-1 && end < start { // last one
						return tv[start], true
					}
					end = start - (start-end-1)/step*step
					for i := end; i <= start; i -= step {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			default:
				if v, has = x.reflectGetNth(tv, start); has {
					if int(fi) == len(x)-1 { // last one
						return v, true
					}
					switch v.(type) {
					case nil, bool, string, float64, float32, gen.Bool, gen.Float, gen.String,
						int, uint, int8, int16, int32, int64, uint8, uint16, uint32, uint64, gen.Int:
					case map[string]any, []any, gen.Object, gen.Array, Keyed, Indexed:
						stack = append(stack, v)
					default:
						if rt := reflect.TypeOf(v); rt != nil {
							switch rt.Kind() {
							case reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Array, reflect.Map:
								stack = append(stack, v)
							}
						}
					}
				}
			}
		case *Filter:
			before := len(stack)
			ns, _ := tf.evalWithRoot(stack, prev, data)
			stack, _ = ns.([]any)
			if int(fi) == len(x)-1 { // last one
				if before < len(stack) {
					result := stack[len(stack)-1]
					stack = stack[:before]
					return result, true
				}
			}
		}
		if int(fi) < len(x)-1 {
			if _, ok := stack[len(stack)-1].(fragIndex); !ok {
				fi++
				f = x[fi]
				stack = append(stack, fi)
			}
		}
	}
	return nil, false
}

func (x Expr) reflectGetChild(data any, key string) (v any, has bool) {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rd = rd.Elem()
		}
		switch rt.Kind() {
		case reflect.Struct:
			rv := reflectGetFieldByKey(rd, key)
			if rv.IsValid() && rv.CanInterface() {
				v = rv.Interface()
				has = true
			}
		case reflect.Map:
			mkt := rt.Key()
			kt := reflect.TypeOf(key)
			if !kt.ConvertibleTo(mkt) {
				break
			}
			rv := rd.MapIndex(reflect.ValueOf(key).Convert(mkt))
			if rv.IsValid() && rv.CanInterface() {
				v = rv.Interface()
				has = true
			}
		}
	}
	return
}

func reflectGetFieldByKey(structValue reflect.Value, key string) reflect.Value {
	if f, ok := reflectGetStructFieldByNameOrJsonTag(structValue, key); ok {
		return structValue.FieldByIndex(f.Index)
	}
	return reflect.Value{}
}

func reflectGetStructFieldByNameOrJsonTag(structValue reflect.Value, key string) (result reflect.StructField, ok bool) {

	// match by field name or by json tag
	match := func(f reflect.StructField) bool {
		if strings.EqualFold(f.Name, key) {
			return true
		}
		tagValue := f.Tag.Get("json")
		jsonKey, _, _ := strings.Cut(tagValue, ",")
		jsonKey = strings.Trim(jsonKey, " ")
		if strings.EqualFold(jsonKey, key) {
			return true
		}
		return false
	}

	type fieldScan struct {
		structVal reflect.Value
		index     []int
	}

	// -----------------------------------------------------------------
	// The algorithm is breadth first search, one depth level at a time.
	// Based on the original: ((Struct) reflect.Value).FieldByNameFunc()
	// -----------------------------------------------------------------

	// The 'current' and 'next' slices are work queues:
	// 'current' lists the fields to visit on this depth level,
	// and 'next' lists the fields on the next lower level.
	current := []fieldScan{}
	next := []fieldScan{{structVal: structValue}}

	// 'nextCount' records the number of times an embedded struct type has been
	// encountered and considered for queueing in the 'next' slice.
	// We only queue the first one, but we increment the count on each.
	// If a struct type T can be reached more than once at a given depth level,
	// then it annihilates itself and need not be considered at all when we
	// process that next depth level.
	var nextCount map[reflect.Type]int

	// 'visited' records the structs that have been considered already.
	// Note that embedded pointer fields can create cycles in the graph of
	// reachable embedded types; 'visited' avoids following those cycles.
	// It also avoids duplicated effort: if we didn't find the field in an
	// embedded type T at level 2, we won't find it in one at level 4 either.
	visited := map[reflect.Type]bool{}

	for len(next) > 0 {
		current, next = next, current[:0]
		count := nextCount
		nextCount = nil

		// Process all the fields at this depth, now listed in 'current'.
		// The loop queues embedded fields found in 'next', for processing during the next
		// iteration. The multiplicity of the 'current' field counts is recorded
		// in 'count'; the multiplicity of the 'next' field counts is recorded in 'nextCount'.
		for i := range current {
			scan := current[i]
			sVal := scan.structVal
			sTyp := sVal.Type()
			if visited[sTyp] {
				// We've looked through this type before, at a higher level.
				// That higher level would shadow the lower level we're now at,
				// so this one can't be useful to us. Ignore it.
				continue
			}
			visited[sTyp] = true

			for i := 0; i < sVal.NumField(); i++ {
				fieldValue := sVal.Field(i)
				structField := sTyp.Field(i)

				var nestedTyp reflect.Type
				if structField.Anonymous {
					// Embedded field of type T or *T.
					nestedTyp = fieldValue.Type()

					if nestedTyp.Kind() == reflect.Interface {
						fieldValue = reflect.ValueOf(fieldValue.Interface())
						nestedTyp = fieldValue.Type()
					}

					if nestedTyp.Kind() == reflect.Ptr {
						fieldValue = fieldValue.Elem()
						nestedTyp = nestedTyp.Elem()
					}
				}

				// Does it match?
				if match(structField) {
					// Potential match
					if count[sTyp] > 1 || ok {
						// Name appeared multiple times at this level: annihilate.
						return reflect.StructField{}, false
					}
					result = structField
					result.Index = nil
					result.Index = append(result.Index, scan.index...)
					result.Index = append(result.Index, i)
					ok = true
					continue
				}

				// Queue embedded struct fields for processing with next level,
				// but only if we haven't seen a match yet at this level and only
				// if the embedded types haven't already been queued.
				if ok || nestedTyp == nil || nestedTyp.Kind() != reflect.Struct {
					continue
				}

				// here we are sure that the nested type is indeed a Struct
				nestedStructVal := fieldValue
				nestedStructTyp := nestedStructVal.Type()

				if nextCount[nestedStructTyp] > 0 {
					nextCount[nestedStructTyp] = 2 // exact multiple doesn't matter
					continue
				}
				if nextCount == nil {
					nextCount = map[reflect.Type]int{}
				}
				nextCount[nestedStructTyp] = 1
				if count[sTyp] > 1 {
					nextCount[nestedStructTyp] = 2 // exact multiple doesn't matter
				}
				var index []int
				index = append(index, scan.index...)
				index = append(index, i)
				next = append(next, fieldScan{structVal: nestedStructVal, index: index})
			}
		}
		if ok {
			break
		}
	}
	return
}

func (x Expr) reflectGetNth(data any, i int) (v any, has bool) {
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
				if rv.CanInterface() {
					v = rv.Interface()
					has = true
				}
			}
		}
	}
	return
}

func (x Expr) reflectGetWild(data any) (va []any) {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rd = rd.Elem()
		}
		switch rt.Kind() {
		case reflect.Struct:
			for i := rd.NumField() - 1; 0 <= i; i-- {
				rv := rd.Field(i)
				if rv.CanInterface() {
					va = append(va, rv.Interface())
				}
			}
		case reflect.Slice, reflect.Array:
			// Iterate in reverse order as that puts values on the stack in reverse.
			for i := rd.Len() - 1; 0 <= i; i-- {
				rv := rd.Index(i)
				if rv.CanInterface() {
					va = append(va, rv.Interface())
				}
			}
		}
	}
	return
}

func (x Expr) reflectGetWildOne(data any) (any, bool) {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		if rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
			rd = rd.Elem()
		}
		switch rt.Kind() {
		case reflect.Struct:
			for i := rd.NumField() - 1; 0 <= i; i-- {
				rv := rd.Field(i)
				if rv.CanInterface() {
					return rv.Interface(), true
				}
			}
		case reflect.Slice, reflect.Array:
			size := rd.Len()
			if 0 < size {
				rv := rd.Index(0)
				if rv.CanInterface() {
					return rv.Interface(), true
				}
			}
		}
	}
	return nil, false
}

func (x Expr) reflectGetSlice(data any, start, end, step int) (va []any) {
	if !isNil(data) {
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			size := rd.Len()
			if start < 0 {
				start = size + start
				if start < 0 {
					start = 0
				}
			}
			if end < 0 {
				end = size + end
				if end < -1 {
					end = -1
				}
			}
			if size < end {
				end = size
			}
			if 0 <= start && start < size {
				if 0 < step {
					for i := start; i < end; i += step {
						rv := rd.Index(i)
						if rv.CanInterface() {
							va = append([]any{rv.Interface()}, va...)
						}
					}
				} else {
					for i := start; end < i; i += step {
						rv := rd.Index(i)
						if rv.CanInterface() {
							va = append([]any{rv.Interface()}, va...)
						}
					}
				}
			}
		}
	}
	return
}
