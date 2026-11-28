// SPDX-License-Identifier: Apache-2.0

package integer_parameters

import "fmt"

// IntegerParameter represents a parameter type
// parameter can of concrete (38) or abstract type (P)
// or another parameterized type like VARCHAR<"L1">
type IntegerParameter interface {
	// IsCompatible is type compatible with other
	// compatible is other can be used in place of this type
	IsCompatible(other IntegerParameter) bool
	fmt.Stringer
}
