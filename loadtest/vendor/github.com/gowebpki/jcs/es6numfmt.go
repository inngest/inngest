// Copyright 2021 Bret Jordan & Benedikt Thoma, All rights reserved.
// Copyright 2006-2019 WebPKI.org (http://webpki.org).
//
// Use of this source code is governed by an Apache 2.0 license that can be
// found in the LICENSE file in the root of the source tree.

package jcs

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

const invalidPattern uint64 = 0x7ff0000000000000

// NumberToJSON converts numbers in IEEE-754 double precision into the
// format specified for JSON in EcmaScript Version 6 and forward.
// The core application for this is canonicalization per RFC 8785:
func NumberToJSON(ieeeF64 float64) (res string, err error) {
	ieeeU64 := math.Float64bits(ieeeF64)

	// Special case: NaN and Infinity are invalid in JSON
	if (ieeeU64 & invalidPattern) == invalidPattern {
		return "null", errors.New("Invalid JSON number: " + strconv.FormatUint(ieeeU64, 16))
	}

	// Special case: eliminate "-0" as mandated by the ES6-JSON/JCS specifications
	if ieeeF64 == 0 { // Right, this line takes both -0 and 0
		return "0", nil
	}

	// Deal with the sign separately
	var sign string = ""
	if ieeeF64 < 0 {
		ieeeF64 = -ieeeF64
		sign = "-"
	}

	// ES6 has a unique "g" format
	var format byte = 'e'
	if ieeeF64 < 1e+21 && ieeeF64 >= 1e-6 {
		format = 'f'
	}

	// The following should (in "theory") do the trick:
	es6Formatted := strconv.FormatFloat(ieeeF64, format, -1, 64)

	// Ryu version
	exponent := strings.IndexByte(es6Formatted, 'e')
	if exponent > 0 {
		// Go outputs "1e+09" which must be rewritten as "1e+9"
		if es6Formatted[exponent+2] == '0' {
			es6Formatted = es6Formatted[:exponent+2] + es6Formatted[exponent+3:]
		}
	}
	return sign + es6Formatted, nil
}
