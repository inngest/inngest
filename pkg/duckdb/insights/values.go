package insights

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strconv"
)

// maxSafeInteger is the largest integer a JSON consumer decoding numbers as
// IEEE754 doubles (JS, structpb.Value's number_value) round trips exactly:
// 2^53 - 1, Number.MAX_SAFE_INTEGER.
const maxSafeInteger = 1<<53 - 1

// JSONSafeValue converts one decoded result cell (Result.Rows) into a value
// every JSON encoder accepts and every JSON consumer reads back unchanged.
// It's the one cell encoding both the REST (pkg/api/v2) and GQL
// (pkg/coreapi) Insights endpoints use, so they agree on every value:
//
//   - NaN/+Inf/-Inf (which encoding/json rejects outright) become the
//     strings "NaN", "Infinity" and "-Infinity".
//   - Integers outside +-(2^53 - 1) become their decimal string, since a
//     double can't hold them exactly (9007199254740993 would otherwise
//     read back as 9007199254740992).
//   - Lists, structs and maps are converted recursively; a map's keys are
//     stringified, since JSON object keys must be strings.
//
// Anything else is returned unchanged.
func JSONSafeValue(v any) any {
	switch x := v.(type) {
	case nil, string, bool:
		return v
	case float64:
		return safeFloat(x)
	case float32:
		return safeFloat(float64(x))
	case int:
		return safeInt(int64(x))
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return safeInt(x)
	case uint:
		return safeUint(uint64(x))
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case uint64:
		return safeUint(x)
	case *big.Int:
		if x == nil {
			return nil
		}
		if x.IsInt64() {
			return safeInt(x.Int64())
		}
		return x.String()
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = JSONSafeValue(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			out[k] = JSONSafeValue(e)
		}
		return out
	}

	// Typed lists/maps (none today afaict, but a driver change could add
	// them) still get their elements converted.
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			// []byte already has a JSON encoding (base64).
			return v
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = JSONSafeValue(rv.Index(i).Interface())
		}
		return out
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = JSONSafeValue(iter.Value().Interface())
		}
		return out
	}
	return v
}

// JSONSafeRows applies JSONSafeValue to every cell of rows, returning a new
// slice (rows itself is left unchanged).
func JSONSafeRows(rows [][]any) [][]any {
	out := make([][]any, len(rows))
	for i, row := range rows {
		r := make([]any, len(row))
		for j, v := range row {
			r[j] = JSONSafeValue(v)
		}
		out[i] = r
	}
	return out
}

func safeFloat(f float64) any {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	return f
}

func safeInt(n int64) any {
	if n > maxSafeInteger || n < -maxSafeInteger {
		return strconv.FormatInt(n, 10)
	}
	return n
}

func safeUint(n uint64) any {
	if n > maxSafeInteger {
		return strconv.FormatUint(n, 10)
	}
	return n
}
