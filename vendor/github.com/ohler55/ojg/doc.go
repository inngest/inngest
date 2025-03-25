// Copyright (c) 2020, Peter Ohler, All rights reserved.

/*
Package ojg is a collection of JSON tools including a validators, parsers, a
full JSONPath implementation, data conversion utilities, and a simple type
assembler. Most of the tools are designed for simple types although used in
complex ways. Simple types in this context are data objects composed of these
types.

	bool
	int64
	float64
	string
	time.Time
	[]any
	map[string]any

# Oj

Package oj contains functions and types for parsing JSON as well as support
for building simple types. Included in the oj package are:

	Parser for parsing JSON strings and streams into simple types.

	Validator for validating JSON strings and streams.

	Builder for building simple types.

	Writer for writing data as JSON.

# Gen

Package gen provides type safe generic types. They are type safe in that array
and objects can only be constructed of other types in the package. The basic
types are:

	Bool
	Int
	Float
	String
	Time

The collection types are Array and Object. All the types implement the Node
interface which is a relatively simple interface defined primarily to restrict
what can be in the collection types. The Node interface should not be used to
define new generic types.

Also included in the package are a builder and parser that behave like the
parser and builder in the oj package except for gen types.

# Jp

Package jp provides JSONPath implementation that operations on simple go
types, generic (gen package), and public struct with public members. Get, set,
and delete operations can be evaluated on data. When needed reflection is used
to follow a path.

# Alt

The alt package contains functions and types for altering values. It includes functions for:

	Decompose() a value into simple types of bool, int64, float64, string,
		time.Time, []any and map[string]any.

	Recompose() takes simple data type and converts it back into a complex type.

	Alter() is the same as decompose except it alters the value in place.

	Generify() converts a simple value into a gen.Node.

# Asm

The asm package provides a means of building JSON or the corresponding simple
types based on a JSON script represented by the Plan type.

# Cmd oj

The oj command is a general purpose tool for processing JSON
documents. Features include reformatting JSON, colorizing JSON, extracting
parts of a JSON document, and filtering. JSONPath is used for both extracting
and filtering.
*/
package ojg
