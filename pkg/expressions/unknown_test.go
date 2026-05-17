package expressions

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandleUnknownCall tests handleUnknownCall — the path taken when one or more
// operands are unknown (missing from activation) or error type.
func TestHandleUnknownCall(t *testing.T) {
	ctx := context.Background()

	noData := map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{}}}

	tests := []struct {
		name     string
		expr     string
		data     map[string]interface{}
		expected interface{}
	}{
		// ── Equals ────────────────────────────────────────────────────────────────
		{
			name:     "unknown == string → false",
			expr:     `event.data.missing == "foo"`,
			data:     noData,
			expected: false,
		},
		{
			name:     "unknown == null → true (unknown treated as null)",
			expr:     `event.data.missing == null`,
			data:     noData,
			expected: true,
		},
		{
			name:     "unknown == 0 → false (unknown is not null literal)",
			expr:     `event.data.missing == 0`,
			data:     noData,
			expected: false,
		},
		{
			name:     "unknown == unknown → false (neither is null literal)",
			expr:     `event.data.a == event.data.b`,
			data:     noData,
			expected: false,
		},

		// ── NotEquals ─────────────────────────────────────────────────────────────
		{
			name:     "unknown != string → true",
			expr:     `event.data.missing != "foo"`,
			data:     noData,
			expected: true,
		},
		{
			name:     "unknown != null → false (unknown is null)",
			expr:     `event.data.missing != null`,
			data:     noData,
			expected: false,
		},
		{
			name:     "unknown != 0 → true (zero value does not equal unknown)",
			expr:     `event.data.missing != 0`,
			data:     noData,
			expected: true,
		},
		{
			name:     `unknown != "" → true`,
			expr:     `event.data.missing != ""`,
			data:     noData,
			expected: true,
		},

		// ── Less / LessEquals — direction matters ─────────────────────────────────
		// handleUnknownCall: if args[0] is unknown → true, else false
		{
			name:     "unknown LHS: unknown < 5 → true",
			expr:     `event.data.missing < 5`,
			data:     noData,
			expected: true,
		},
		{
			name:     "unknown LHS: unknown <= 5 → true",
			expr:     `event.data.missing <= 5`,
			data:     noData,
			expected: true,
		},
		{
			name:     "known LHS: 5 < unknown → false",
			expr:     `5 < event.data.missing`,
			data:     noData,
			expected: false,
		},
		{
			name:     "known LHS: 5 <= unknown → false",
			expr:     `5 <= event.data.missing`,
			data:     noData,
			expected: false,
		},
		{
			name: "known field > unknown → true",
			expr: `event.data.n > event.data.missing`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"n": 2}}},
			// args[0] = int(2), not unknown → Greater returns true
			expected: true,
		},

		// ── Greater / GreaterEquals — direction matters ───────────────────────────
		// handleUnknownCall: if args[0] is unknown → false, else true
		{
			name:     "unknown LHS: unknown > 5 → false",
			expr:     `event.data.missing > 5`,
			data:     noData,
			expected: false,
		},
		{
			name:     "unknown LHS: unknown >= 5 → false",
			expr:     `event.data.missing >= 5`,
			data:     noData,
			expected: false,
		},
		{
			name:     "known LHS: 5 > unknown → true",
			expr:     `5 > event.data.missing`,
			data:     noData,
			expected: true,
		},
		{
			name:     "known LHS: 5 >= unknown → true",
			expr:     `5 >= event.data.missing`,
			data:     noData,
			expected: true,
		},

		// ── Add with unknown ──────────────────────────────────────────────────────
		// handleUnknownCall Add: returns the first non-unknown arg in argument order
		{
			name:     "known + unknown → known (lhs preserved)",
			expr:     `event.data.name + event.data.missing`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"name": "hello"}}},
			expected: "hello",
		},
		{
			name:     "unknown + known → known (rhs preserved when lhs missing)",
			expr:     `event.data.missing + event.data.name`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"name": "hello"}}},
			expected: "hello",
		},
		{
			// Both args unknown → no non-unknown found → returns false
			name:     "unknown + unknown → false",
			expr:     `event.data.a + event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{}}},
			expected: false,
		},

		// ── size on unknown ───────────────────────────────────────────────────────
		{
			name:     "size(unknown) returns 0",
			expr:     `size(event.data.missing)`,
			data:     noData,
			expected: int64(0),
		},
		{
			name:     "size(unknown) > 0 → false",
			expr:     `size(event.data.missing) > 0`,
			data:     noData,
			expected: false,
		},
		{
			name:     "size(unknown) == 0 → true",
			expr:     `size(event.data.missing) == 0`,
			data:     noData,
			expected: true,
		},

		// ── @in with unknown (default branch) ─────────────────────────────────────
		{
			name:     "unknown in list → false",
			expr:     `event.data.missing in ["a", "b", "c"]`,
			data:     noData,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate(ctx, tt.expr, tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestNullTypeHandling tests the NullType path in unknownDecorator
// (argTypes.Exists(types.NullType) && argTypes.ArgLen() == 2),
// which coerces null to a zero value for ordered comparisons.
func TestNullTypeHandling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		expr     string
		data     map[string]interface{}
		expected interface{}
	}{
		// ── Equals with explicit null ─────────────────────────────────────────────
		{
			name:     "null == string → false",
			expr:     `null == "foo"`,
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			name:     "string == null → false",
			expr:     `"foo" == null`,
			data:     map[string]interface{}{},
			expected: false,
		},

		// ── NotEquals with explicit null ──────────────────────────────────────────
		{
			name:     "null != string → true",
			expr:     `null != "foo"`,
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			name:     "string != null → true",
			expr:     `"foo" != null`,
			data:     map[string]interface{}{},
			expected: true,
		},

		// ── null < "string" — ZeroValArgs with string zero value ────────────────────
		{
			// ZeroValArgs replaces null with String("") → "" < "foo" = true
			name:     `null < "foo" → true (null coerced to empty string)`,
			expr:     `null < "foo"`,
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			// "" < "aardvark" is false since "" < "a" is true actually... wait, "" < "foo" is true
			// ZeroValArgs: null → String("") → "foo" < "" = false
			name:     `"foo" < null → false (null coerced to empty string)`,
			expr:     `"foo" < null`,
			data:     map[string]interface{}{},
			expected: false,
		},
		// ── null compared with uint — zeroVal "uint" branch ─────────────────────────
		{
			// null < uint(5): ZeroValArgs → Uint(0) < Uint(5) = true
			name:     "null < uint(5) → true (null coerced to uint 0)",
			expr:     `null < uint(5)`,
			data:     map[string]interface{}{},
			expected: true,
		},
		// ── Less/Greater with null literal — ZeroValArgs replaces null with 0 ─────
		{
			// null < 5: ZeroValArgs → int(0) < int(5) = true
			name:     "null < 5 → true (null coerced to 0)",
			expr:     `null < 5`,
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			// 5 < null: ZeroValArgs → int(5) < int(0) = false
			name:     "5 < null → false (null coerced to 0)",
			expr:     `5 < null`,
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			// null > 0: ZeroValArgs → int(0) > int(0) = false
			name:     "null > 0 → false (null coerced to 0)",
			expr:     `null > 0`,
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			// 5 > null: ZeroValArgs → int(5) > int(0) = true
			name:     "5 > null → true (null coerced to 0)",
			expr:     `5 > null`,
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			// null <= 0: ZeroValArgs → int(0) <= int(0) = true
			name:     "null <= 0 → true (null coerced to 0)",
			expr:     `null <= 0`,
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			// null >= 1: ZeroValArgs → int(0) >= int(1) = false
			name:     "null >= 1 → false (null coerced to 0)",
			expr:     `null >= 1`,
			data:     map[string]interface{}{},
			expected: false,
		},
		{
			// null < 0.5: NonNull is DoubleType → Double(0) < 0.5 = true
			name:     "null < 0.5 → true (null coerced to 0.0)",
			expr:     `null < 0.5`,
			data:     map[string]interface{}{},
			expected: true,
		},
		// ── Null field values from activation ─────────────────────────────────────
		{
			name:     "float field > null → true",
			expr:     `event.data.float > null`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"float": 3.141}}},
			expected: true,
		},
		{
			name:     "int field > null → true",
			expr:     `event.data.n > null`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"n": 8}}},
			expected: true,
		},
		{
			name:     "float field <= null → false",
			expr:     `event.data.float <= null`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"float": 3.141}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate(ctx, tt.expr, tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestTruthyLogicalCoercion tests the evalTruthyLogical wrapper which handles ||
// and && with non-boolean operands using JS-like truthy coercion.
func TestTruthyLogicalCoercion(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		expr     string
		data     map[string]interface{}
		expected interface{}
	}{
		// ── Logical OR: returns first truthy, or last value if all falsy ───────────
		{
			name:     "OR both truthy strings → returns first",
			expr:     `event.data.a || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "first", "b": "second"}}},
			expected: "first",
		},
		{
			name:     "OR empty string (falsy) || truthy string → returns second",
			expr:     `event.data.a || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "", "b": "second"}}},
			expected: "second",
		},
		{
			name:     "OR missing || truthy → returns truthy",
			expr:     `event.data.missing || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"b": "found"}}},
			expected: "found",
		},
		{
			name:     "OR truthy || missing → returns truthy (lhs short-circuits)",
			expr:     `event.data.a || event.data.missing`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "found"}}},
			expected: "found",
		},
		{
			name:     "OR both missing → false (last unknown is falsy)",
			expr:     `event.data.a || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{}}},
			expected: false,
		},
		{
			name:     "OR int 0 (falsy) || string → returns string",
			expr:     `event.data.zero || event.data.name`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"zero": 0, "name": "hi"}}},
			expected: "hi",
		},
		// isTruthy for non-standard types in OR
		{
			// null is falsy → falls through to rhs
			name:     "OR null (falsy) || string → returns string",
			expr:     `null || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"b": "fallback"}}},
			expected: "fallback",
		},
		{
			// double 3.14 is truthy
			name:     "OR double 3.14 (truthy) || false → returns 3.14",
			expr:     `event.data.f || false`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"f": 3.14}}},
			expected: float64(3.14),
		},
		{
			// double 0.0 is falsy → falls through to rhs
			name:     "OR double 0.0 (falsy) || string → returns string",
			expr:     `event.data.f || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"f": 0.0, "b": "yes"}}},
			expected: "yes",
		},
		{
			// A non-empty list hits isTruthy default branch → truthy → OR returns
			// the list instead of false.
			name:     "OR non-empty list (truthy default branch) || false → returns list",
			expr:     `false || event.data.list`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"list": []string{"a"}}}},
			expected: []string{"a"},
		},
		{
			// uint(5) hits isTruthy UintType branch → non-zero → truthy
			name:     "OR uint(5) (truthy) || false → returns uint 5",
			expr:     `uint(5) || false`,
			data:     map[string]interface{}{},
			expected: uint64(5),
		},
		{
			// uint(0) hits isTruthy UintType branch → zero → falsy → falls through
			name:     "OR uint(0) (falsy) || string → returns string",
			expr:     `uint(0) || event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"b": "yes"}}},
			expected: "yes",
		},
		// Boolean operands delegate to native CEL
		{
			name:     "OR bool false || bool true → true",
			expr:     `event.data.flagA || event.data.flagB`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"flagA": false, "flagB": true}}},
			expected: true,
		},
		{
			name:     "OR bool false || bool false → false",
			expr:     `event.data.flagA || event.data.flagB`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"flagA": false, "flagB": false}}},
			expected: false,
		},
		{
			name:     "OR bool true || bool false → true",
			expr:     `event.data.flagA || event.data.flagB`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"flagA": true, "flagB": false}}},
			expected: true,
		},

		// ── Logical AND: returns first falsy, or last value if all truthy ──────────
		{
			name:     "AND both truthy strings → returns last",
			expr:     `event.data.a && event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "first", "b": "second"}}},
			expected: "second",
		},
		{
			name:     "AND empty string (falsy) && truthy → returns empty string",
			expr:     `event.data.a && event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "", "b": "second"}}},
			expected: "",
		},
		{
			name:     "AND truthy && empty string → returns empty string",
			expr:     `event.data.a && event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"a": "first", "b": ""}}},
			expected: "",
		},
		{
			name:     "AND missing (falsy) && truthy → returns false",
			expr:     `event.data.missing && event.data.b`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"b": "second"}}},
			expected: false,
		},
		// Boolean operands delegate to native CEL
		{
			name:     "AND bool true && bool false → false",
			expr:     `event.data.flagA && event.data.flagB`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"flagA": true, "flagB": false}}},
			expected: false,
		},
		{
			name:     "AND bool true && bool true → true",
			expr:     `event.data.flagA && event.data.flagB`,
			data:     map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"flagA": true, "flagB": true}}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate(ctx, tt.expr, tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestMacrosWithUnknowns tests that macros (exists, all) handle unknowns correctly.
// The key invariant: lambda variables produce "no such attribute" errors which must
// pass through the decorator unchanged, not be treated as unknown call failures.
func TestMacrosWithUnknowns(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		expr     string
		data     map[string]interface{}
		expected interface{}
	}{
		{
			name: "exists on real list with matching element → true",
			expr: `event.data.tags.exists(x, x == "a")`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"tags": []string{"a", "b", "c"}}}},
			expected: true,
		},
		{
			name: "exists on real list with no matching element → false",
			expr: `event.data.tags.exists(x, x == "z")`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"tags": []string{"a", "b", "c"}}}},
			expected: false,
		},
		{
			name: "exists on real list with OR inside lambda → true",
			expr: `event.data.tags.exists(x, x == "d" || x == "a")`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"tags": []string{"a", "b", "c"}}}},
			expected: true,
		},
		{
			name: "exists on unknown field → false",
			expr: `event.nonexistent.exists(x, x == "a")`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{}}},
			expected: false,
		},
		{
			name: "all on real list all matching → true",
			expr: `event.data.nums.all(x, x > 0)`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"nums": []int{1, 2, 3}}}},
			expected: true,
		},
		{
			name: "all on real list not all matching → false",
			expr: `event.data.nums.all(x, x > 1)`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"nums": []int{1, 2, 3}}}},
			expected: false,
		},
		{
			name: "contains on unknown nested field → false",
			expr: `event.data.some.unknown.contains("x")`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{}}},
			expected: false,
		},
		{
			name: "in operator with unknown field → false",
			expr: `event.data.nonexistent in ["LOL", "Issue", "Epic"]`,
			data: map[string]interface{}{"event": map[string]interface{}{"data": map[string]interface{}{"issue": "Bug"}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Evaluate(ctx, tt.expr, tt.data)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestUnknownDecoratorConcurrentSafety verifies that the same compiled expression
// can be evaluated concurrently with different data without data races or errors.
func TestUnknownDecoratorConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	// Use an expression that exercises unknown, null, and truthy paths together.
	expression := `event.data.missing < 5 && event.data.name != "" && (event.data.tag || event.data.fallback)`

	const goroutines = 100
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := map[string]interface{}{
				"event": map[string]interface{}{
					"data": map[string]interface{}{
						"name":     fmt.Sprintf("user-%d", n),
						"fallback": fmt.Sprintf("tag-%d", n),
					},
				},
			}
			_, err := Evaluate(ctx, expression, data)
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}

func TestEvalTruthyLogical_NaryOR(t *testing.T) {
	ctx := context.Background()

	// name == "alice" AND role == "admin" AND plan == "pro" AND (feature_a OR feature_b OR feature_c)
	expr := `event.data.user.name == "alice" && event.data.user.role == "admin" && event.data.user.plan == "pro" && (event.data.features.a || event.data.features.b || event.data.features.c)`

	eval := func(name, role, plan string, features map[string]interface{}) (interface{}, error) {
		return Evaluate(ctx, expr, map[string]interface{}{
			"event": map[string]interface{}{
				"data": map[string]interface{}{
					"user":     map[string]interface{}{"name": name, "role": role, "plan": plan},
					"features": features,
				},
			},
		})
	}

	// all ANDs satisfied, 3rd OR branch decides
	result, err := eval("alice", "admin", "pro", map[string]interface{}{"c": "on"})
	require.NoError(t, err)
	require.NotEqual(t, false, result)

	// all ANDs satisfied, 1st OR branch
	result, err = eval("alice", "admin", "pro", map[string]interface{}{"a": "on"})
	require.NoError(t, err)
	require.NotEqual(t, false, result)

	// all ANDs satisfied, 2nd OR branch
	result, err = eval("alice", "admin", "pro", map[string]interface{}{"b": "on"})
	require.NoError(t, err)
	require.NotEqual(t, false, result)

	// wrong name
	result, err = eval("bob", "admin", "pro", map[string]interface{}{"a": "on", "b": "on", "c": "on"})
	require.NoError(t, err)
	require.Equal(t, false, result)

	// wrong role
	result, err = eval("alice", "user", "pro", map[string]interface{}{"a": "on", "b": "on", "c": "on"})
	require.NoError(t, err)
	require.Equal(t, false, result)

	// wrong plan
	result, err = eval("alice", "admin", "free", map[string]interface{}{"a": "on", "b": "on", "c": "on"})
	require.NoError(t, err)
	require.Equal(t, false, result)

	// no features
	result, err = eval("alice", "admin", "pro", map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, false, result)
}
