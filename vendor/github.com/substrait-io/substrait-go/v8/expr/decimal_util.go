package expr

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

var decimalPattern = regexp.MustCompile(`^[+-]?\d*(\.\d*)?([eE][+-]?\d*)?$`)

// DecimalStringToBytes converts a decimal string to a 16-byte byte array.
// 16-byte bytes represents a little-endian 128-bit integer, to be divided by 10^Scale to get the decimal value.
// This function also returns the precision and scale of the decimal value.
// The precision is the total number of digits in the decimal value. The precision is limited to 38 digits.
// The scale is the number of digits to the right of the decimal point. The scale is limited to the precision.
func DecimalStringToBytes(decimalStr string) ([16]byte, int32, int32, error) {
	var result [16]byte

	strings.Trim(decimalStr, " ")
	if !decimalPattern.MatchString(decimalStr) {
		return result, 0, 0, fmt.Errorf("invalid decimal string")
	}

	// Parse the decimal string using apd
	dec, cond, err := apd.NewFromString(decimalStr)
	if err != nil || cond.Any() {
		return result, 0, 0, fmt.Errorf("invalid decimal string %s: %v", decimalStr, err)
	}

	return DecimalToBytes(dec)
}

// DecimalToBytes converts apd.Decimal to a 16-byte byte array.
// 16-byte bytes represents a little-endian 128-bit integer, to be divided by 10^Scale to get the decimal value.
// This function also returns the precision and scale of the decimal value.
func DecimalToBytes(dec *apd.Decimal) ([16]byte, int32, int32, error) {
	var (
		result    [16]byte
		precision int32
		scale     int32
	)

	if dec.Exponent > 0 {
		precision = int32(apd.NumDigits(&dec.Coeff)) + dec.Exponent
		scale = 0
	} else {
		scale = -dec.Exponent
		precision = max(int32(apd.NumDigits(&dec.Coeff)), scale+1)
	}
	if precision > 38 {
		return result, precision, scale, fmt.Errorf("number %s exceeds maximum precision of 38 (%d)", dec.String(), precision)
	}

	coefficient := dec.Coeff
	if dec.Exponent > 0 {
		// Multiply coefficient by 10^exponent.
		multiplier := apd.NewBigInt(1).Exp(apd.NewBigInt(10), apd.NewBigInt(int64(dec.Exponent)), nil)
		coefficient.Mul(&dec.Coeff, multiplier)
	}
	// Convert the coefficient to a byte array.
	byteArray := coefficient.Bytes()
	if len(byteArray) > 16 {
		return result, 0, 0, fmt.Errorf("number exceeds 16 bytes")
	}
	copy(result[16-len(byteArray):], byteArray)

	// Handle the sign by taking the two's complement for negative numbers.
	if dec.Negative {
		negate(result[:])
	}

	// Reverse the byte array to little-endian.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, precision, scale, nil
}

// negate flips the sign of a two-complements value by modifying it in place.
func negate(bytes []byte) {
	for i := range bytes {
		bytes[i] = ^bytes[i]
	}
	carry := byte(1)
	for i := len(bytes) - 1; i >= 0; i-- {
		bytes[i] += carry
		if bytes[i] != 0 {
			break
		}
	}
}

func decimalBytesToString(decimalBytes [16]byte, scale int32) string {
	isNegative := decimalBytes[15]&0x80 != 0

	// Reverse the byte array to big-endian.
	processingValue := make([]byte, 16)
	for i := len(processingValue) - 1; i >= 0; i = i - 1 {
		processingValue[i] = decimalBytes[15-i]
	}
	if isNegative {
		negate(processingValue[:])
	}

	// Convert into an apd.BigInt so it can handle the rendering.
	intValue := new(big.Int).SetBytes(processingValue[:])
	if isNegative {
		intValue.Neg(intValue)
	}
	apdBigInt := new(apd.BigInt).SetMathBigInt(intValue)
	return apd.NewWithBigInt(apdBigInt, -scale).String()
}

func modifyDecimalPrecisionAndScale(decimalBytes [16]byte, scale, targetPrecision, targetScale int32) ([16]byte, int32, int32, error) {
	var result [16]byte
	if targetPrecision > 38 {
		return result, 0, 0, fmt.Errorf("target precision %d exceeds maximum allowed precision of 38", targetPrecision)
	}

	isNegative := decimalBytes[15]&0x80 != 0

	// Reverse the byte array to convert from little-endian to big-endian.
	processingValue := make([]byte, 16)
	for i := 0; i < 16; i++ {
		processingValue[i] = decimalBytes[15-i]
	}
	if isNegative {
		negate(processingValue[:])
	}

	// Convert the bytes into a big.Int and wrap it into an apd.Decimal.
	intValue := new(big.Int).SetBytes(processingValue[:])
	apdBigInt := new(apd.BigInt).SetMathBigInt(intValue)
	dec := apd.NewWithBigInt(apdBigInt, -scale)

	// Normalize the decimal by removing trailing zeros.
	dec.Reduce(dec)

	err2 := validatePrecisionAndScale(dec, targetPrecision, targetScale)
	if err2 != nil {
		return result, 0, 0, err2
	}

	// After ensuring the targetScale is sufficient for dec, adjust the scale to the target scale
	ctx := apd.BaseContext.WithPrecision(uint32(targetPrecision))
	_, err := ctx.Quantize(dec, dec, -targetScale)
	if err != nil {
		return result, 0, 0, fmt.Errorf("error adjusting scale: %v", err)
	}

	// Convert the adjusted decimal coefficient to a byte array.
	byteArray := dec.Coeff.Bytes()
	if len(byteArray) > 16 {
		return result, 0, 0, fmt.Errorf("number exceeds 16 bytes")
	}
	copy(result[16-len(byteArray):], byteArray)

	// Handle the sign by applying two's complement for negative numbers.
	if isNegative {
		negate(result[:])
	}

	// Reverse the byte array back to little-endian.
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, targetPrecision, targetScale, nil
}

func validatePrecisionAndScale(dec *apd.Decimal, targetPrecision int32, targetScale int32) error {
	// Validate the minimum precision and scale.
	minPrecision, minScale := getMinimumPrecisionAndScale(dec)
	if targetScale < minScale {
		return fmt.Errorf("number %v exceeds target scale %d, minimum scale needed is %d", dec.String(), targetScale, minScale)
	}
	if targetPrecision < minPrecision {
		return fmt.Errorf("number %s exceeds target precision %d, minimum precision needed is %d with target scale %d", dec.String(), targetPrecision, minPrecision, targetScale)
	}
	if targetPrecision-targetScale < minPrecision-minScale {
		return fmt.Errorf("number %v exceeds target precision %d with target scale %d, minimum precision needed is %d with minimum scale %d", dec.String(), targetPrecision, targetScale, minPrecision, minScale)
	}
	return nil
}

func getMinimumPrecisionAndScale(dec *apd.Decimal) (precision int32, scale int32) {
	if dec.Exponent > 0 {
		precision = int32(apd.NumDigits(&dec.Coeff)) + dec.Exponent
		scale = 0
	} else {
		scale = -dec.Exponent
		precision = max(int32(apd.NumDigits(&dec.Coeff)), scale)
	}
	return precision, scale
}
