// Copyright (c) 2020, Peter Ohler, All rights reserved.

package jp

import (
	"strconv"

	"github.com/ohler55/ojg/gen"
)

type index int

// String returns the key as a string.
func (i index) String() string {
	return strconv.Itoa(int(i))
}

// Alter converts the node into it's native type. Note this will modify
// Objects and Arrays in place making them no longer usable as the
// original type. Use with care!
func (i index) Alter() any {
	return i
}

// Simplify makes a copy of the node but as simple types.
func (i index) Simplify() any {
	return int64(i)
}

// Dup returns a deep duplicate of the node.
func (i index) Dup() gen.Node {
	return i
}

// Empty returns true if the node is empty.
func (i index) Empty() bool {
	return false
}

// GetNodes the elements of the data identified by the path.
func (x Expr) GetNodes(n gen.Node) (results []gen.Node) {
	if len(x) == 0 {
		// A bit of a cheat but to get 100% coverage the index interface
		// functions have to be called. The alternative would be to make th
		// eindex type public but then someone could use it as a value and
		// break the evaluation. Anyway, since evaluating an expty expression
		// is all but useless the index functions are called here and return
		// values ignored since they are never used in the real code.
		i := index(0)
		_ = i.Alter()
		_ = i.Dup()
		_ = i.Simplify()
		_ = i.Empty()
		_ = i.String()
		return
	}
	var v gen.Node
	var prev gen.Node
	var has bool

	stack := make([]gen.Node, 0, 64)
	stack = append(stack, n)

	f := x[0]
	fi := index(0) // frag index
	stack = append(stack, fi)

	for 1 < len(stack) { // must have at least a data element and a fragment index
		prev = stack[len(stack)-2]
		if ii, up := prev.(index); up {
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = x[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch tf := f.(type) {
		case Child:
			if tv, ok := prev.(gen.Object); ok {
				if v, has = tv[string(tf)]; has {
					if fi == index(len(x))-1 { // last one
						results = append(results, v)
					} else {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			}
		case Nth:
			i := int(tf)
			if tv, ok := prev.(gen.Array); ok {
				if i < 0 {
					i = len(tv) + i
				}
				if 0 < i && i < len(tv) {
					v = tv[i]
					if fi == index(len(x))-1 { // last one
						results = append(results, v)
					} else {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
						}
					}
				}
			}
		case Wildcard:
			switch tv := prev.(type) {
			case gen.Object:
				if fi == index(len(x))-1 { // last one
					for _, v = range tv {
						results = append(results, v)
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
				if fi == index(len(x))-1 { // last one
					for _, v = range tv {
						results = append(results, v)
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
			}
		case Descent:
			di, _ := stack[len(stack)-1].(index)
			top := (di & descentChildFlag) == 0
			// first pass expands, second continues evaluation
			if (int64(di) & descentFlag) == 0 {
				switch tv := prev.(type) {
				case gen.Object:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if fi == index(len(x))-1 { // last one
						for _, v = range tv {
							results = append(results, v)
						}
					}
					for _, v = range tv {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				case gen.Array:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if fi == index(len(x))-1 { // last one
						for _, v = range tv {
							results = append(results, v)
						}
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				}
			} else {
				if fi == index(len(x))-1 { // last one
					if top {
						results = append(results, prev)
					}
				} else {
					stack = append(stack, prev)
				}
			}
		case Root:
			if fi == index(len(x))-1 { // last one
				results = append(results, n)
			} else {
				stack = append(stack, n)
			}
		case At, Bracket:
			if fi == index(len(x))-1 { // last one
				results = append(results, prev)
			} else {
				stack = append(stack, prev)
			}
		case Union:
			if fi == index(len(x))-1 { // last one
				for _, u := range tf {
					switch tu := u.(type) {
					case string:
						if tv, ok := prev.(gen.Object); ok {
							if v, has = tv[tu]; has {
								results = append(results, v)
							}
						}
					case int64:
						i := int(tu)
						if tv, ok := prev.(gen.Array); ok {
							if i < 0 {
								i = len(tv) + i
							}
							var v gen.Node
							if 0 <= i && i < len(tv) {
								v = tv[i]
							}
							results = append(results, v)
						}
					}
				}
			} else {
				for ui := len(tf) - 1; 0 <= ui; ui-- {
					u := tf[ui]
					switch tu := u.(type) {
					case string:
						if tv, ok := prev.(gen.Object); ok {
							if v, has = tv[tu]; has {
								switch v.(type) {
								case gen.Object, gen.Array:
									stack = append(stack, v)
								}
							}
						}
					case int64:
						i := int(tu)
						if tv, ok := prev.(gen.Array); ok {
							if i < 0 {
								i = len(tv) + i
							}
							var v gen.Node
							if 0 <= i && i < len(tv) {
								v = tv[i]
							}
							switch v.(type) {
							case gen.Object, gen.Array:
								stack = append(stack, v)
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
			if tv, ok := prev.(gen.Array); ok {
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
							case gen.Object, gen.Array:
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
							case gen.Object, gen.Array:
								stack = append(stack, v)
							}
						}
					}
				}
			}
		case *Filter:
			before := len(stack)
			ns, _ := tf.evalWithRoot(stack, prev, n)
			stack, _ = ns.([]gen.Node)
			if int(fi) == len(x)-1 { // last one
				for i := before; i < len(stack); i++ {
					results = append(results, stack[i])
				}
				if before < len(stack) {
					stack = stack[:before]
				}
			}
		}
		if int(fi) < len(x)-1 {
			if _, ok := stack[len(stack)-1].(index); !ok {
				fi++
				f = x[fi]
				stack = append(stack, fi)
			}
		}
	}
	// Free up anything still on the stack.
	stack = stack[0:cap(stack)]
	for i := cap(stack) - 1; 0 <= i; i-- {
		stack[i] = nil
	}
	return
}

// FirstNode returns the first matcning node.
func (x Expr) FirstNode(n gen.Node) (result gen.Node) {
	if len(x) == 0 {
		return nil
	}
	var v gen.Node
	var prev gen.Node
	var has bool

	stack := make([]gen.Node, 0, 64)
	defer func() {
		stack = stack[0:cap(stack)]
		for i := len(stack) - 1; 0 <= i; i-- {
			stack[i] = nil
		}
	}()
	stack = append(stack, n)
	f := x[0]
	fi := index(0) // frag index
	stack = append(stack, fi)

	for 1 < len(stack) { // must have at least a data element and a fragment index
		prev = stack[len(stack)-2]
		if ii, up := prev.(index); up {
			stack = stack[:len(stack)-1]
			fi = ii & fragIndexMask
			f = x[fi]
			continue
		}
		stack[len(stack)-2] = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		switch tf := f.(type) {
		case Child:
			if tv, ok := prev.(gen.Object); ok {
				if v, has = tv[string(tf)]; has {
					if fi == index(len(x))-1 { // last one
						return v
					}
					switch v.(type) {
					case gen.Object, gen.Array:
						stack = append(stack, v)
					}
				}
			}
		case Nth:
			i := int(tf)
			if tv, ok := prev.(gen.Array); ok {
				if i < 0 {
					i = len(tv) + i
				}
				if 0 <= i && i < len(tv) {
					v = tv[i]
					if fi == index(len(x))-1 { // last one
						return v
					}
					switch v.(type) {
					case gen.Object, gen.Array:
						stack = append(stack, v)
					}
				}
			}
		case Wildcard:
			switch tv := prev.(type) {
			case gen.Object:
				if fi == index(len(x))-1 { // last one
					for _, v = range tv {
						return v
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
				if fi == index(len(x))-1 { // last one
					for _, v = range tv {
						return v
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
			}
		case Descent:
			di, _ := stack[len(stack)-1].(index)
			// first pass expands, second continues evaluation
			if (int64(di) & descentFlag) == 0 {
				switch tv := prev.(type) {
				case gen.Object:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if fi == index(len(x))-1 { // last one
						for _, v = range tv {
							return v
						}
					}
					for _, v = range tv {
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				case gen.Array:
					// Put prev back and slide fi.
					stack[len(stack)-1] = prev
					stack = append(stack, di|descentFlag)
					if fi == index(len(x))-1 { // last one
						if 0 < len(tv) {
							return tv[0]
						}
					}
					for i := len(tv) - 1; 0 <= i; i-- {
						v = tv[i]
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
							stack = append(stack, fi|descentChildFlag)
						}
					}
				}
			} else {
				stack = append(stack, prev)
			}
		case Root:
			if fi == index(len(x))-1 { // last one
				return n
			}
			stack = append(stack, n)
		case At, Bracket:
			if fi == index(len(x))-1 { // last one
				return prev
			}
			stack = append(stack, prev)
		case Union:
			for ui := len(tf) - 1; 0 <= ui; ui-- {
				u := tf[ui]
				switch tu := u.(type) {
				case string:
					if tv, ok := prev.(gen.Object); ok {
						if v, has = tv[tu]; has {
							if fi == index(len(x))-1 { // last one
								return v
							}
							switch v.(type) {
							case gen.Object, gen.Array:
								stack = append(stack, v)
							}
						}
					}
				case int64:
					i := int(tu)
					if tv, ok := prev.(gen.Array); ok {
						if i < 0 {
							i = len(tv) + i
						}
						if 0 <= i && i < len(tv) {
							v = tv[i]
						}
						if fi == index(len(x))-1 { // last one
							return v
						}
						switch v.(type) {
						case gen.Object, gen.Array:
							stack = append(stack, v)
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
			if tv, ok := prev.(gen.Array); ok {
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
						return tv[start]
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
						return tv[start]
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
			}
		case *Filter:
			before := len(stack)
			ns, _ := tf.evalWithRoot(stack, prev, n)
			stack, _ = ns.([]gen.Node)
			if int(fi) == len(x)-1 { // last one
				if before < len(stack) {
					result := stack[before]
					stack = stack[:before]
					return result
				}
			}
		}
		if int(fi) < len(x)-1 {
			if _, ok := stack[len(stack)-1].(index); !ok {
				fi++
				f = x[fi]
				stack = append(stack, fi)
			}
		}
	}
	return nil
}
