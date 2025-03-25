// Copyright (c) 2020, Peter Ohler, All rights reserved.

/*
Package alt contains functions and types for altering values.

# Conversions

Simple conversion from one to to another include converting to string, bool,
int64, float64, and time.Time. Each of these functions takes between one and
three arguments. The first is the value to convert. The second argument is the
value to return if the value can not be converted. For example, if the value
is an array then the second argument, the first default would be returned. If
the third argument is present then any input that is not the correct type will
cause the third default to be returned. The conversion functions are Int(),
FLoat(), Bool(), String(), and Time(). The reason for the defaults are to
allow a single return from a conversion unlike a type assertion.

	i := alt.Int("123", 0)

# Generify

It is often useful to work with generic values that can be converted to JSON
and also provide type safety so that code can be checked at compile
time. Those value types are defined in the gen package. The Genericer
interface defines the Generic() function as

	Generic() gen.Node

A Generify() function is used to convert values to gen.Node types.

	type Genny struct {
		val int
	}
	func (g *Genny) Generic() gen.Node {
	 	return gen.Object{"type": gen.String("genny"), "val": gen.Int(g.val)}
	}
	ga := []*Genny{&Genny{val: 3}}
	v := alt.Generify(ga)
	// v: [{"type":"Genny","val":3}]

# Decompose

The Decompose() functions creates a simple type converting non simple to
simple types using either the Simplify() interface or reflection. Unlike
Alter() a deep copy is returned leaving the original data unchanged.

	type Sample struct {
		Int int
		Str string
	}
	sample := Sample{Int: 3, Str: "three"}
	simple := alt.Decompose(&sample, &alt.Options{CreateKey: "^", FullTypePath: true})
	// simple: {"^":"github.com/ohler55/ojg/alt_test/Sample","int":3,"str":"three"}

# Recompose

Recompose simple data into more complex go types using either the Recompose()
function or the Recomposer struct that adds some efficiency by reusing
buffers. The package takes a best effort approach to recomposing matching
against not only json tags but also against member names and member names
starting with a lower case character.

	type Sample struct {
		Int int
		Str string
	}
	r, err := alt.NewRecomposer("^", map[any]alt.RecomposeFunc{&Sample{}: nil})
	var v any
	if err == nil {
		v, err = r.Recompose(map[string]any{"^": "Sample", "int": 3, "str": "three"})
	}
	// sample: {Int: 3, Str: "three"}

# Alter

The GenAlter() function converts a simple go data element into Node compliant
data. A best effort is made to convert values that are not simple into generic
Nodes. It modifies the values inplace if possible by altering the original.

	m := map[string]any{"a": 1, "b": 4, "c": 9}
	v := alt.GenAlter(m)
	// v:  gen.Object{"a": gen.Int(1), "b": gen.Int(4), "c": gen.Int(9)}, v)
*/
package alt
