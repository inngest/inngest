// Copyright (c) 2023, Peter Ohler, All rights reserved.

package ojg

// NumConvMethod specifies a json.Number conversion method. It is used by the
// parsers and the recomposer to convert a json.Number to either a float64 or
// to a string if used.
type NumConvMethod byte

const (
	// NumConvNone leaves a json.Number as is.
	NumConvNone = NumConvMethod(0)

	// NumConvFloat64 convert json.Number to a float64.
	NumConvFloat64 = NumConvMethod('f')

	// NumConvString convert json.Number to a string
	NumConvString = NumConvMethod('s')
)

// DefaultNumConvMethod is the default NumConvMethod for parsing and
// recompose.
var DefaultNumConvMethod = NumConvNone
