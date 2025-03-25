// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"reflect"
	"strconv"

	"github.com/ohler55/ojg/gen"
)

// Slice is a slice operation for a JSON path expression.
type Slice []int

// Append a fragment string representation of the fragment to the buffer
// then returning the expanded buffer.
func (f Slice) Append(buf []byte, _, _ bool) []byte {
	buf = append(buf, '[')
	if 0 < len(f) {
		for i, n := range f {
			if 0 < i {
				buf = append(buf, ':')
			}
			switch i {
			case 0:
				if n != 0 {
					buf = append(buf, strconv.FormatInt(int64(n), 10)...)
				}
			case 1:
				if n != maxEnd {
					buf = append(buf, strconv.FormatInt(int64(n), 10)...)
				}
			default:
				buf = append(buf, strconv.FormatInt(int64(n), 10)...)
			}
			if 2 <= i {
				break
			}
		}
		if len(f) == 1 {
			buf = append(buf, ':')
		}
	} else {
		buf = append(buf, ':')
	}
	buf = append(buf, ']')

	return buf
}

func (f Slice) remove(value any) (out any, changed bool) {
	out = value
	start := 0
	end := -1
	step := 1
	if 0 < len(f) {
		start = f[0]
	}
	if 1 < len(f) {
		end = f[1]
	}
	if 2 < len(f) {
		step = f[2]
	}
	switch tv := value.(type) {
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
			return
		}
		ns := make([]any, 0, len(tv))
		if 0 < step {
			for i, v := range tv {
				if inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, v)
				}
			}
		} else {
			// Walk in reverse to handle the just-one condition.
			for i := len(tv) - 1; 0 <= i; i-- {
				if inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, tv[i])
				}
			}
			for i := len(ns)/2 - 1; 0 <= i; i-- {
				ns[i], ns[len(ns)-i-1] = ns[len(ns)-i-1], ns[i]
			}
		}
		if changed {
			out = ns
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
		if start < 0 || end < 0 || len(tv) <= start || len(tv) <= end || step == 0 {
			return
		}
		ns := make(gen.Array, 0, len(tv))
		if 0 < step {
			for i, v := range tv {
				if inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, v)
				}
			}
		} else {
			for i := len(tv) - 1; 0 <= i; i-- {
				if inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, tv[i])
				}
			}
			for i := len(ns)/2 - 1; 0 <= i; i-- {
				ns[i], ns[len(ns)-i-1] = ns[len(ns)-i-1], ns[i]
			}
		}
		if changed {
			out = ns
		}
	case RemovableIndexed:
		size := tv.Size()
		if start < 0 {
			start = size + start
		}
		if end < 0 {
			end = size + end
		}
		if size <= end {
			end = size - 1
		}
		if start < 0 || end < 0 || size <= start || size <= end || step == 0 {
			return
		}
		for i := size - 1; 0 <= i; i-- {
			if inStep(i, start, end, step) {
				changed = true
				tv.RemoveValueAtIndex(i)
			}
		}
	default:
		rv := reflect.ValueOf(value)
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
				return
			}
			nc := 0
			for i := 0; i < cnt; i++ {
				if inStep(i, start, end, step) {
					changed = true
				} else {
					nc++
				}
			}
			if changed {
				changed = false
				ns := reflect.MakeSlice(rv.Type(), nc, nc)
				if 0 < step {
					ni := 0
					for i := 0; i < cnt; i++ {
						if inStep(i, start, end, step) {
							changed = true
						} else {
							ns.Index(ni).Set(rv.Index(i))
							ni++
						}
					}
				} else {
					ni := nc - 1
					for i := cnt - 1; 0 <= i; i-- {
						if inStep(i, start, end, step) {
							changed = true
						} else {
							ns.Index(ni).Set(rv.Index(i))
							ni--
						}
					}
				}
				out = ns.Interface()
			}
		}
	}
	return
}

func (f Slice) removeOne(value any) (out any, changed bool) {
	out = value
	start := 0
	end := -1
	step := 1
	if 0 < len(f) {
		start = f[0]
	}
	if 1 < len(f) {
		end = f[1]
	}
	if 2 < len(f) {
		step = f[2]
	}
	switch tv := value.(type) {
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
			return
		}
		ns := make([]any, 0, len(tv))
		if 0 < step {
			for i, v := range tv {
				if !changed && inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, v)
				}
			}
		} else {
			// Walk in reverse to handle the just-one condition.
			for i := len(tv) - 1; 0 <= i; i-- {
				if !changed && inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, tv[i])
				}
			}
			for i := len(ns)/2 - 1; 0 <= i; i-- {
				ns[i], ns[len(ns)-i-1] = ns[len(ns)-i-1], ns[i]
			}
		}
		if changed {
			out = ns
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
			return
		}
		ns := make(gen.Array, 0, len(tv))
		if 0 < step {
			for i, v := range tv {
				if !changed && inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, v)
				}
			}
		} else {
			// Walk in reverse to handle the just-one condition.
			for i := len(tv) - 1; 0 <= i; i-- {
				if !changed && inStep(i, start, end, step) {
					changed = true
				} else {
					ns = append(ns, tv[i])
				}
			}
			for i := len(ns)/2 - 1; 0 <= i; i-- {
				ns[i], ns[len(ns)-i-1] = ns[len(ns)-i-1], ns[i]
			}
		}
		if changed {
			out = ns
		}
	case RemovableIndexed:
		size := tv.Size()
		if start < 0 {
			start = size + start
		}
		if end < 0 {
			end = size + end
		}
		if size <= end {
			end = size - 1
		}
		if start < 0 || end < 0 || size <= start || size <= end || step == 0 {
			return
		}
		for i := 0; i < size; i++ {
			if inStep(i, start, end, step) {
				changed = true
				tv.RemoveValueAtIndex(i)
				break
			}
		}
	default:
		rv := reflect.ValueOf(value)
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
				return
			}
			nc := 0
			for i := 0; i < cnt; i++ {
				if !changed && inStep(i, start, end, step) {
					changed = true
				} else {
					nc++
				}
			}
			if changed {
				changed = false
				ns := reflect.MakeSlice(rv.Type(), nc, nc)
				if 0 < step {
					ni := 0
					for i := 0; i < cnt; i++ {
						if !changed && inStep(i, start, end, step) {
							changed = true
						} else {
							ns.Index(ni).Set(rv.Index(i))
							ni++
						}
					}
				} else {
					ni := nc - 1
					for i := cnt - 1; 0 <= i; i-- {
						if !changed && inStep(i, start, end, step) {
							changed = true
						} else {
							ns.Index(ni).Set(rv.Index(i))
							ni--
						}
					}
				}
				out = ns.Interface()
			}
		}
	}
	return
}

func inStep(i, start, end, step int) bool {
	if 0 < step {
		return start <= i && i <= end && (i-start)%step == 0
	}
	return end <= i && i <= start && (i-end)%-step == 0
}

func (f Slice) startEndStep(size int) (start, end, step int) {
	start = 0
	end = maxEnd
	step = 1
	if 0 < len(f) {
		start = f[0]
	}
	if 1 < len(f) {
		end = f[1]
	}
	if 2 < len(f) {
		step = f[2]
		if step == 0 {
			return
		}
	}
	if start < 0 {
		start = size + start
	} else if size <= start {
		start = size - 1
	}
	if start < 0 {
		start = 0
	}
	if end < 0 {
		end = size + end + 1
		if end < 0 && step < 0 {
			end = -1
		}
	} else if size < end {
		end = size
	}
	return
}

func (f Slice) locate(pp Expr, data any, rest Expr, max int) (locs []Expr) {
	switch td := data.(type) {
	case []any:
		start, end, step := f.startEndStep(len(td))
		if step == 0 {
			return
		}
		if 0 < step {
			if len(rest) == 0 { // last one
				for i := start; i < end; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; i < end; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td[i], rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		} else {
			if len(rest) == 0 { // last one
				for i := start; end < i; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; end < i; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td[i], rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		}
	case gen.Array:
		start, end, step := f.startEndStep(len(td))
		if step == 0 {
			return
		}
		if 0 < step {
			if len(rest) == 0 { // last one
				for i := start; i < end; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; i < end; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td[i], rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		} else {
			if len(rest) == 0 { // last one
				for i := start; end < i; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; end < i; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td[i], rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		}
	case Indexed:
		start, end, step := f.startEndStep(td.Size())
		if step == 0 {
			return
		}
		if 0 < step {
			if len(rest) == 0 { // last one
				for i := start; i < end; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; i < end; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td.ValueAtIndex(i), rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		} else {
			if len(rest) == 0 { // last one
				for i := start; end < i; i += step {
					locs = locateAppendFrag(locs, pp, Nth(i))
				}
			} else {
				cp := append(pp, nil) // place holder
				for i := start; end < i; i += step {
					cp[len(pp)] = Nth(i)
					locs = locateContinueFrag(locs, cp, td.ValueAtIndex(i), rest, max)
					if 0 < max && max <= len(locs) {
						break
					}
				}
			}
		}
	case nil:
		// no match
	default:
		rd := reflect.ValueOf(data)
		rt := rd.Type()
		switch rt.Kind() {
		case reflect.Slice, reflect.Array:
			start, end, step := f.startEndStep(rd.Len())
			if 0 < step {
				if len(rest) == 0 { // last one
					for i := start; i < end; i += step {
						rv := rd.Index(i)
						if rv.CanInterface() {
							locs = locateAppendFrag(locs, pp, Nth(i))
							if 0 < max && max <= len(locs) {
								break
							}
						}
					}
				} else {
					cp := append(pp, nil) // place holder
					for i := start; i < end; i += step {
						cp[len(pp)] = Nth(i)
						rv := rd.Index(i)
						if rv.CanInterface() {
							locs = locateContinueFrag(locs, cp, rv.Interface(), rest, max)
							if 0 < max && max <= len(locs) {
								break
							}
						}
					}
				}
			} else {
				if len(rest) == 0 { // last one
					for i := start; end < i; i += step {
						rv := rd.Index(i)
						if rv.CanInterface() {
							locs = locateAppendFrag(locs, pp, Nth(i))
							if 0 < max && max <= len(locs) {
								break
							}
						}
					}
				} else {
					cp := append(pp, nil) // place holder
					for i := start; end < i; i += step {
						cp[len(pp)] = Nth(i)
						rv := rd.Index(i)
						if rv.CanInterface() {
							locs = locateContinueFrag(locs, cp, rv.Interface(), rest, max)
							if 0 < max && max <= len(locs) {
								break
							}
						}
					}
				}
			}
		}
	}
	return
}
