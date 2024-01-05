// Copyright (c) 2022, Peter Ohler, All rights reserved.

package jp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ohler55/ojg"
	"github.com/ohler55/ojg/gen"
)

// MustModify modifies matching nodes and panics on an expression error.
// Modified elements replace the original element in the data. The modified
// data is returned. Unless the data is a slice and modified the returned data
// will be the same object as the original. The modifier function will be
// called with the elements that match the path and should return the original
// element or if altered the altered value along with setting the returned
// changed value to true.
func (x Expr) MustModify(data any, modifier func(element any) (altered any, changed bool)) any {
	return x.modify(data, modifier, false)
}

// MustModifyOne modifies matching nodes and panics on an expression error.
// Modified elements replace the original element in the data. The modified
// data is returned. Unless the data is a slice and modified the returned data
// will be the same object as the original. The modifier function will be
// called with the elements that match the path and should return the original
// element or if altered the altered value along with setting the returned
// changed value to true. The function returns after the first modification.
func (x Expr) MustModifyOne(data any, modifier func(element any) (altered any, changed bool)) any {
	return x.modify(data, modifier, true)
}

// Modify modifies matching nodes and panics on an expression error.  Modified
// elements replace the original element in the data. The modified data is
// returned. Unless the data is a slice and modified the returned data will be
// the same object as the original. The modifier function will be called with
// the elements that match the path and should return the original element or
// if altered the altered value along with setting the returned changed value
// to true.
func (x Expr) Modify(data any, modifier func(element any) (altered any, changed bool)) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	result = x.modify(data, modifier, false)

	return
}

// ModifyOne modifies matching nodes and panics on an expression error.
// Modified elements replace the original element in the data. The modified
// data is returned. Unless the data is a slice and modified the returned data
// will be the same object as the original. The modifier function will be
// called with the elements that match the path and should return the original
// element or if altered the altered value along with setting the returned
// changed value to true. The function returns after the first modification.
func (x Expr) ModifyOne(data any, modifier func(element any) (altered any, changed bool)) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ojg.NewError(r)
		}
	}()
	result = x.modify(data, modifier, true)

	return
}

func (x Expr) modify(data any, modifier func(element any) (altered any, changed bool), one bool) any {
	if len(x) == 0 {
		panic("can not modify with an empty expression")
	}
	if _, ok := x[len(x)-1].(Descent); ok {
		ta := strings.Split(fmt.Sprintf("%T", x[len(x)-1]), ".")
		panic(fmt.Sprintf("can not modify with an expression where the last fragment is a %s",
			ta[len(ta)-1]))
	}
	wx := make(Expr, len(x)+1)
	copy(wx[1:], x)
	wx[0] = Nth(0)
	wrap := []any{data}
	var (
		v    any
		prev any
	)
	stack := make([]any, 0, 64)
	stack = append(stack, wrap)

	f := wx[0]
	fi := fragIndex(0) // frag index
	stack = append(stack, fi)

done:
	for 1 < len(stack) {
		prev = stack[len(stack)-2]
		if ii, up := prev.(fragIndex); up {
			stack[len(stack)-1] = nil
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = wx[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack[len(stack)-1] = nil
		stack = stack[:len(stack)-1]

		switch tf := f.(type) {
		case Child:
			var has bool
			key := string(tf)
			switch tv := prev.(type) {
			case map[string]any:
				if v, has = tv[key]; has {
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(v); changed {
							tv[key] = nv
							if one && changed {
								break done
							}
						}
					} else {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
				if v, has = tv[key]; has {
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(v); changed {
							tv[key] = nv.(gen.Node)
							if one && changed {
								break done
							}
						}
					} else {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			default:
				if v, has = wx.reflectGetChild(tv, key); has {
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(v); changed {
							wx.reflectSetChild(tv, key, nv)
							if one && changed {
								break done
							}
						}
					} else {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
		case Nth:
			i := int(tf)
			switch tv := prev.(type) {
			case []any:
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(tv[i]); changed {
							tv[i] = nv
							if one && changed {
								break done
							}
						}
					} else {
						v = tv[i]
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(tv[i]); changed {
							tv[i] = nv.(gen.Node)
							if one && changed {
								break done
							}
						}
					} else {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			default:
				var has bool
				if v, has = wx.reflectGetNth(tv, i); has {
					if int(fi) == len(wx)-1 { // last one
						if nv, changed := modifier(v); changed {
							wx.reflectSetNth(tv, i, nv)
							if one && changed {
								break done
							}
						}
					} else {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
		case Wildcard:
			switch tv := prev.(type) {
			case map[string]any:
				var k string
				if int(fi) == len(wx)-1 { // last one
					for k = range tv {
						if nv, changed := modifier(tv[k]); changed {
							tv[k] = nv
							if one && changed {
								break done
							}
						}
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
				if int(fi) == len(wx)-1 { // last one
					for i := range tv {
						if nv, changed := modifier(tv[i]); changed {
							tv[i] = nv
							if one && changed {
								break done
							}
						}
					}
				} else {
					for _, v = range tv {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
				if int(fi) == len(wx)-1 { // last one
					for k = range tv {
						if nv, changed := modifier(tv[k]); changed {
							tv[k] = nv.(gen.Node)
							if one && changed {
								break done
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
				if int(fi) == len(wx)-1 { // last one
					for i := range tv {
						if nv, changed := modifier(tv[i]); changed {
							tv[i] = nv.(gen.Node)
							if one && changed {
								break done
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
			default:
				if int(fi) == len(wx)-1 { // last one
					rv := reflect.ValueOf(tv)
					switch rv.Kind() {
					case reflect.Slice:
						cnt := rv.Len()
						for i := 0; i < cnt; i++ {
							iv := rv.Index(i)
							if nv, changed := modifier(iv.Interface()); changed {
								iv.Set(reflect.ValueOf(nv))
								if one && changed {
									break done
								}
							}
						}
					case reflect.Map:
						keys := rv.MapKeys()
						sort.Slice(keys, func(i, j int) bool {
							return strings.Compare(keys[i].String(), keys[j].String()) < 0
						})
						for _, k := range keys {
							ev := rv.MapIndex(k)
							if nv, changed := modifier(ev.Interface()); changed {
								rv.SetMapIndex(k, reflect.ValueOf(nv))
								if one && changed {
									break done
								}
							}
						}
					}
				} else {
					for _, v := range wx.reflectGetWild(tv) {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
		case Union:
			for _, u := range tf {
				switch tu := u.(type) {
				case string:
					var has bool
					switch tv := prev.(type) {
					case map[string]any:
						if v, has = tv[tu]; has {
							if int(fi) == len(wx)-1 { // last one
								if nv, changed := modifier(v); changed {
									tv[tu] = nv
									if one && changed {
										break done
									}
								}
							} else {
								switch v.(type) {
								case map[string]any, []any, gen.Object, gen.Array:
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
						if v, has = tv[tu]; has {
							if int(fi) == len(wx)-1 { // last one
								if nv, changed := modifier(v); changed {
									tv[tu] = nv.(gen.Node)
									if one && changed {
										break done
									}
								}
							} else {
								switch v.(type) {
								case gen.Object, gen.Array:
									stack = append(stack, v)
								}
							}
						}
					default:
						var has bool
						if v, has = wx.reflectGetChild(tv, tu); has {
							if int(fi) == len(wx)-1 { // last one
								if nv, changed := modifier(v); changed {
									wx.reflectSetChild(tv, tu, nv)
									if one && changed {
										break done
									}
								}
							} else {
								switch v.(type) {
								case map[string]any, []any, gen.Object, gen.Array:
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
				case int64:
					i := int(tu)
					switch tv := prev.(type) {
					case []any:
						if i < 0 {
							i = len(tv) + i
						}
						if 0 <= i && i < len(tv) {
							v = tv[i]
							if int(fi) == len(wx)-1 { // last one
								if nv, changed := modifier(v); changed {
									tv[i] = nv
									if one && changed {
										break done
									}
								}
							} else {
								switch v.(type) {
								case map[string]any, []any, gen.Object, gen.Array:
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
							if int(fi) == len(wx)-1 { // last one
								if nv, changed := modifier(tv[i]); changed {
									tv[i] = nv.(gen.Node)
									if one && changed {
										break done
									}
								}
							} else {
								v = tv[i]
								switch v.(type) {
								case gen.Object, gen.Array:
									stack = append(stack, v)
								}
							}
						}
					default:
						if int(fi) == len(wx)-1 { // last one
							rv := reflect.ValueOf(tv)
							if rv.Kind() == reflect.Slice {
								cnt := rv.Len()
								if i < 0 {
									i = cnt + i
								}
								if 0 <= i && i < cnt {
									iv := rv.Index(i)
									if nv, changed := modifier(iv.Interface()); changed {
										iv.Set(reflect.ValueOf(nv))
										if one && changed {
											break done
										}
									}
								}
							}
						} else {
							var has bool
							if v, has = wx.reflectGetNth(tv, i); has {
								switch v.(type) {
								case map[string]any, []any, gen.Object, gen.Array:
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
				if len(tv) <= end {
					end = len(tv) - 1
				}
				if start < 0 || end < 0 || len(tv) <= start || step == 0 {
					continue
				}
				if 0 < step {
					for i := start; i <= end; i += step {
						v = tv[i]
						if int(fi) == len(wx)-1 { // last one
							if nv, changed := modifier(v); changed {
								tv[i] = nv
								if one && changed {
									break done
								}
							}
						} else {
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array:
								stack = append(stack, v)
							}
						}
					}
				} else {
					for i := start; end <= i; i += step {
						v = tv[i]
						if int(fi) == len(wx)-1 { // last one
							if nv, changed := modifier(v); changed {
								tv[i] = nv
								if one && changed {
									break done
								}
							}
						} else {
							switch v.(type) {
							case map[string]any, []any, gen.Object, gen.Array:
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
				if len(tv) <= end {
					end = len(tv) - 1
				}
				if start < 0 || end < 0 || len(tv) <= start || step == 0 {
					continue
				}
				if 0 < step {
					for i := start; i <= end; i += step {
						if int(fi) == len(wx)-1 { // last one
							if nv, changed := modifier(tv[i]); changed {
								tv[i] = nv.(gen.Node)
								if one && changed {
									break done
								}
							}
						} else {
							v = tv[i]
							switch v.(type) {
							case gen.Object, gen.Array:
								stack = append(stack, v)
							}
						}
					}
				} else {
					for i := start; end <= i; i += step {
						if int(fi) == len(wx)-1 { // last one
							if nv, changed := modifier(tv[i]); changed {
								tv[i] = nv.(gen.Node)
								if one && changed {
									break done
								}
							}
						} else {
							v = tv[i]
							switch v.(type) {
							case gen.Object, gen.Array:
								stack = append(stack, v)
							}
						}
					}
				}
			default:
				if int(fi) == len(wx)-1 {
					rv := reflect.ValueOf(tv)
					if rv.Kind() == reflect.Slice {
						cnt := rv.Len()
						if start < 0 {
							start = cnt + start
						}
						if end < 0 {
							end = cnt + end
						}
						if cnt <= end {
							end = cnt - 1
						}
						if start < 0 || end < 0 || cnt <= start || step == 0 {
							continue
						}
						if 0 < step {
							for i := start; i <= end; i += step {
								iv := rv.Index(i)
								if nv, changed := modifier(iv.Interface()); changed {
									iv.Set(reflect.ValueOf(nv))
									if one && changed {
										break done
									}
								}
							}
						} else {
							for i := start; end <= i; i += step {
								iv := rv.Index(i)
								if nv, changed := modifier(iv.Interface()); changed {
									iv.Set(reflect.ValueOf(nv))
									if one && changed {
										break done
									}
								}
							}
						}
					}
				} else {
					for _, v := range wx.reflectGetSlice(tv, start, end, step) {
						switch v.(type) {
						case map[string]any, []any, gen.Object, gen.Array:
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
			if int(fi) == len(wx)-1 { // last one
				switch tv := prev.(type) {
				case []any:
					for i, vv := range tv {
						if tf.Match(vv) {
							if nv, changed := modifier(vv); changed {
								tv[i] = nv
								if one && changed {
									break done
								}
							}
						}
					}
				case gen.Array:
					for i, vv := range tv {
						if tf.Match(vv) {
							if nv, changed := modifier(vv); changed {
								tv[i] = nv.(gen.Node)
								if one && changed {
									break done
								}
							}
						}
					}
				default:
					rv := reflect.ValueOf(tv)
					switch rv.Kind() {
					case reflect.Slice:
						cnt := rv.Len()
						for i := 0; i < cnt; i++ {
							iv := rv.Index(i)
							vv := iv.Interface()
							if tf.Match(vv) {
								if nv, changed := modifier(vv); changed {
									iv.Set(reflect.ValueOf(nv))
									if one && changed {
										break done
									}
								}
							}
						}
					case reflect.Map:
						keys := rv.MapKeys()
						sort.Slice(keys, func(i, j int) bool {
							return strings.Compare(keys[i].String(), keys[j].String()) < 0
						})
						for _, k := range keys {
							ev := rv.MapIndex(k)
							vv := ev.Interface()
							if tf.Match(vv) {
								if nv, changed := modifier(vv); changed {
									rv.SetMapIndex(k, reflect.ValueOf(nv))
									if one && changed {
										break done
									}
								}
							}
						}
					}
				}
			} else {
				ns, _ := tf.evalWithRoot(stack, prev, data)
				stack, _ = ns.([]any)
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
						case map[string]any, []any, gen.Object, gen.Array:
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
						case map[string]any, []any, gen.Object, gen.Array:
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
						case map[string]any, []any, gen.Object, gen.Array:
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
						case map[string]any, []any, gen.Object, gen.Array:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				}
			} else {
				stack = append(stack, prev)
			}
		case Root:
			if int(fi) == len(wx)-1 { // last one
				if nv, changed := modifier(data); changed {
					wrap[0] = nv
					if one && changed {
						break done
					}
				}
			} else {
				stack = append(stack, data)
			}
		case At, Bracket:
			if int(fi) == len(wx)-1 { // last one
				if nv, changed := modifier(data); changed {
					wrap[0] = nv
					if one && changed {
						break done
					}
				}
			}
			stack = append(stack, prev)
		}
		if int(fi) < len(wx)-1 {
			if _, ok := stack[len(stack)-1].(fragIndex); !ok {
				fi++
				f = wx[fi]
				stack = append(stack, fi)
			}
		}
	}
	return wrap[0]
}
