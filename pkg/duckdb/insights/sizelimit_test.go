package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSizeLimitedFunctions pins which calls the size guard (sizelimit.go)
// accepts and rejects. Every rejected shape was confirmed to blow past
// memory_limit at a large enough size (see maxSizeArgument).
func TestSizeLimitedFunctions(t *testing.T) {
	accepts := []string{
		"SELECT repeat('-', 20) AS bar",
		"SELECT repeat('ab', 5000) AS s",
		"SELECT repeat('x', -1) AS s",
		"SELECT lpad(run_id, 30, '0') AS s FROM runs",
		"SELECT rpad(run_id, 10000, ' ') AS s FROM runs",
		"SELECT format('{:>10} {:.2f} {{literal}}', run_id, 1.5) AS s FROM runs",
		"SELECT format('{}', run_id) AS s FROM runs",
		"SELECT printf('%-10s|%5.2f|100%%', run_id, 1.5) AS s FROM runs",
		"SELECT range(10) AS r",
		"SELECT range(0, 20000, 2) AS r",
		"SELECT range(10, 0, -1) AS r",
		"SELECT generate_series(1, 10000) AS r",
		"SELECT generate_series(-5, 5) AS r",
		"SELECT unnest(range(1000)) AS r",
	}
	for _, q := range accepts {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.NoError(t, err)
		})
	}

	rejects := map[string]string{
		"SELECT repeat('x', 1500000000) AS s":                         "REPEAT output size is limited",
		"SELECT repeat('ab', 5001) AS s":                              "REPEAT output size is limited",
		"SELECT repeat(run_id, 2) AS s FROM runs":                     "REPEAT requires a string literal",
		"SELECT repeat(repeat('x', 100), 100) AS s":                   "REPEAT requires a string literal",
		"SELECT repeat('x', length(run_id)) AS s FROM runs":           "REPEAT requires a number literal",
		"SELECT lpad('x', 1500000000, 'y') AS s":                      "LPAD output size is limited",
		"SELECT rpad('x', 10001, 'y') AS s":                           "RPAD output size is limited",
		"SELECT lpad('x', length(run_id) * 1000, 'y') AS s FROM runs": "LPAD requires a number literal",
		"SELECT format('{:>2000000000}', 'x') AS s":                   "FORMAT widths and precisions are limited",
		"SELECT format('{:.20000f}', 1.5) AS s":                       "FORMAT widths and precisions are limited",
		"SELECT format('{:>{}}', 'x', 2000000000) AS s":               "FORMAT doesn't support a width",
		"SELECT format(run_id, 1) AS s FROM runs":                     "FORMAT requires a string literal format",
		"SELECT printf('%1500000000s', 'x') AS s":                     "PRINTF widths and precisions are limited",
		"SELECT printf('%*s', 1500000000, 'x') AS s":                  "PRINTF doesn't support a width",
		"SELECT range(300000000) AS r":                                "RANGE output size is limited",
		"SELECT range(0, 10001) AS r":                                 "RANGE output size is limited",
		"SELECT range(length(run_id)) AS r FROM runs":                 "RANGE requires integer literal arguments",
		"SELECT generate_series(0, 10000) AS r":                       "GENERATE_SERIES output size is limited",
		"SELECT generate_series(-300000000, 0) AS r":                  "GENERATE_SERIES output size is limited",
		"SELECT generate_series(1, 2, 0.0000001) AS r":                "GENERATE_SERIES requires integer literal arguments",
		"SELECT generate_series(1, length(run_id)) AS r FROM runs":    "GENERATE_SERIES requires integer literal arguments",
	}
	for q, want := range rejects {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.ErrorContains(t, err, want)
		})
	}
}

// TestSizeLimitedComposition pins checkSizeComposition's rules: no guarded
// call inside a list comprehension or a CTE/subquery, and guarded calls add
// up across the statement.
func TestSizeLimitedComposition(t *testing.T) {
	accepts := []string{
		"SELECT [x * 2 for x in [1, 2, 3]] AS r",
		"SELECT [x || 'a' for x in string_split(run_id, '-')] AS r FROM runs",
		"SELECT [x.a for x in [{'a': 1}]] AS r",
		"SELECT range(5000) AS a, range(5000) AS b",
		"SELECT repeat('-', 20) AS a FROM runs UNION ALL SELECT repeat('=', 20) AS a FROM runs",
		"SELECT unnest(range(1000)) AS r",
		"SELECT range(10) AS r FROM runs WHERE run_id IN (SELECT run_id FROM runs)",
		"SELECT range(9223372036854770000, 9223372036854775000) AS r",
		"SELECT range(010) AS r",
		"WITH c AS (SELECT run_id FROM runs) SELECT lpad(run_id, 30, '0') AS r FROM c",
	}
	for _, q := range accepts {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.NoError(t, err)
		})
	}

	const inComprehension = "can't be used inside a list comprehension"
	const inSubquery = "can't be used inside a CTE or subquery"
	const inFrom = "can't be used in a FROM clause"
	const total = "output size is limited to 10000, counting every size-limited call"
	rejects := map[string]string{
		// The original repro.
		"SELECT len(flatten([range(10000) for x in range(10000)])) AS n":             "RANGE " + inComprehension,
		"SELECT [x for x in range(10)] AS r":                                         "RANGE " + inComprehension,
		"SELECT [range(10) for x in [1, 2]] AS r":                                    "RANGE " + inComprehension,
		"SELECT [x for x in [1, 2] if x < len(repeat('a', 3))] AS r":                 "REPEAT " + inComprehension,
		"SELECT [lpad(x, 5, '0') for x in string_split(run_id, '-')] AS r FROM runs": "LPAD " + inComprehension,
		"SELECT [[y for y in range(10)] for x in [1, 2]] AS r":                       "RANGE " + inComprehension,
		"SELECT [runs.run_id for runs in range(100)] AS r FROM runs":                 "RANGE " + inComprehension,
		// Float rounding near BIGINT's range used to compute this 100-element
		// series as 0 elements.
		"SELECT [range(10000) for x in range(9223372036854770000, 9223372036854770100)] AS r": "RANGE " + inComprehension,
		// A guarded value copied by name.
		"WITH t AS (SELECT range(10000) AS r) SELECT [x for x in t.r] AS c FROM t":                                 "RANGE " + inSubquery,
		"WITH c AS (SELECT range(10000) AS r) SELECT [r for y in string_split(run_id, '-')] AS x FROM c, runs":     "RANGE " + inSubquery,
		"WITH c AS (SELECT range(1000) AS r) SELECT list_value(r, r, r, r, r, r, r, r, r, r, r) AS x FROM c, runs": "RANGE " + inSubquery,
		"SELECT [#1 for y in string_split(run_id, '-')] AS x FROM (SELECT range(10000) AS r) s, runs":              "RANGE " + inFrom,
		"SELECT x FROM (SELECT repeat('a', 3) AS x) s":                                                             "REPEAT " + inFrom,
		"SELECT * FROM unnest(range(1000))":                                                                        "RANGE " + inFrom,
		"SELECT run_id FROM runs, unnest(range(10000)) u":                                                          "RANGE " + inFrom,
		"SELECT run_id FROM runs JOIN unnest([repeat('a', 3)]) u ON true":                                          "REPEAT " + inFrom,
		"SELECT run_id FROM runs WHERE run_id IN (SELECT repeat('a', 3))":                                          "REPEAT " + inSubquery,
		"SELECT (SELECT range(3)) AS r":                                                                            "RANGE " + inSubquery,
		// Sibling calls add up.
		"SELECT list_value(range(5000), range(5000), range(1)) AS r":                    "RANGE " + total,
		"SELECT rpad(run_id, 10000, ' ') AS a, lpad(run_id, 1, ' ') AS b FROM runs":     "LPAD " + total,
		"SELECT range(6000) AS a FROM runs UNION ALL SELECT range(6000) AS a FROM runs": "RANGE " + total,
		"SELECT format('{:>5000}', 1) AS a, printf('%5001s', 'x') AS b":                 "PRINTF " + total,
		"SELECT range(10000) AS r FROM runs WHERE len(range(1)) > 0":                    "RANGE " + total,
		"SELECT range(9223372036854770000, 9223372036854780001) AS r":                   "RANGE output size is limited to 10000",
	}
	for q, want := range rejects {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.ErrorContains(t, err, want)
		})
	}
}

// TestSizeChecksAreAllowedFunctions pins that every sizeChecks entry is an
// allowed function, so withSizeLimit's lookup by name can't silently miss.
func TestSizeChecksAreAllowedFunctions(t *testing.T) {
	for name := range sizeChecks {
		_, ok := allowedFunctions[name]
		require.True(t, ok, name)
	}
}
