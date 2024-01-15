// Copyright (c) 2020, Peter Ohler, All rights reserved.

package gen

import "fmt"

// Builder is assists in build a more complex Node.
type Builder struct {
	stack  []Node
	starts []int
}

// Reset clears the the Builder of previous built nodes.
func (b *Builder) Reset() {
	if 0 < cap(b.stack) && 0 < len(b.stack) {
		b.stack = b.stack[:0]
		b.starts = b.starts[:0]
	} else {
		b.stack = make([]Node, 0, 64)
		b.starts = make([]int, 0, 16)
	}
}

// MustObject adds an object to the builder. A key is required if adding to a
// parent object.
func (b *Builder) MustObject(key ...string) {
	if err := b.Object(key...); err != nil {
		panic(err)
	}
}

// Object adds an object to the builder. A key is required if adding to a
// parent object.
func (b *Builder) Object(key ...string) error {
	newObj := Object{}
	if 0 < len(key) {
		if len(b.starts) == 0 || 0 <= b.starts[len(b.starts)-1] {
			return fmt.Errorf("can not use a key when pushing to an array")
		}
		if obj, _ := b.stack[len(b.stack)-1].(Object); obj != nil {
			obj[key[0]] = newObj
		}
	} else if 0 < len(b.starts) && b.starts[len(b.starts)-1] < 0 {
		return fmt.Errorf("must have a key when pushing to an object")
	}
	b.starts = append(b.starts, -1)
	b.stack = append(b.stack, newObj)

	return nil
}

// MustArray adds an array to the builder. A key is required if adding to a
// parent object.
func (b *Builder) MustArray(key ...string) {
	if err := b.Array(key...); err != nil {
		panic(err)
	}
}

// Array adds an array to the builder. A key is required if adding to a parent
// object.
func (b *Builder) Array(key ...string) error {
	if 0 < len(key) {
		if len(b.starts) == 0 || 0 <= b.starts[len(b.starts)-1] {
			return fmt.Errorf("can not use a key when pushing to an array")
		}
		b.stack = append(b.stack, Key(key[0]))
	} else if 0 < len(b.starts) && b.starts[len(b.starts)-1] < 0 {
		return fmt.Errorf("must have a key when pushing to an object")
	}
	b.starts = append(b.starts, len(b.stack))
	b.stack = append(b.stack, EmptyArray)

	return nil
}

// MustValue adds a Node to the builder. A key is required if adding to a
// parent object.
func (b *Builder) MustValue(value Node, key ...string) {
	if err := b.Value(value, key...); err != nil {
		panic(err)
	}
}

// Value adds a Node to the builder. A key is required if adding to a parent
// object.
func (b *Builder) Value(value Node, key ...string) error {
	switch {
	case 0 < len(key):
		if len(b.starts) == 0 || 0 <= b.starts[len(b.starts)-1] {
			return fmt.Errorf("can not use a key when pushing to an array")
		}
		if obj, _ := b.stack[len(b.stack)-1].(Object); obj != nil {
			obj[key[0]] = value
		}
	case 0 < len(b.starts) && b.starts[len(b.starts)-1] < 0:
		return fmt.Errorf("must have a key when pushing to an object")
	default:
		b.stack = append(b.stack, value)
	}
	return nil
}

// Pop close a parent Object or Array Node.
func (b *Builder) Pop() {
	if 0 < len(b.starts) {
		start := b.starts[len(b.starts)-1]
		if 0 <= start { // array
			start++
			size := len(b.stack) - start
			a := Array(make([]Node, size))
			copy(a, b.stack[start:len(b.stack)])
			b.stack = b.stack[:start]
			b.stack[start-1] = a
			if 2 < len(b.stack) {
				if k, ok := b.stack[len(b.stack)-2].(Key); ok {
					if obj, _ := b.stack[len(b.stack)-3].(Object); obj != nil {
						obj[string(k)] = a
						b.stack = b.stack[:len(b.stack)-2]
					}
				}
			}
		}
		b.starts = b.starts[:len(b.starts)-1]
	}
}

// PopAll close all parent Object or Array Nodes.
func (b *Builder) PopAll() {
	for 0 < len(b.starts) {
		b.Pop()
	}
}

// Result returns the current built Node.
func (b *Builder) Result() (result Node) {
	if 0 < len(b.stack) {
		result = b.stack[0]
	}
	return
}
