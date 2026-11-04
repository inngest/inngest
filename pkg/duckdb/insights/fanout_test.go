package insights

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestComprehensionFanOut pins checkComprehensionFanOut's rules: no
// comprehension nested in another's body/filter, and a body/filter can only
// reference its own variables.
func TestComprehensionFanOut(t *testing.T) {
	accepts := []string{
		"SELECT [x.key FOR x IN runs.sessions] AS k FROM runs",
		"SELECT [x::INTEGER FOR x IN ['1', '2']]",
		"SELECT [CAST(x AS DECIMAL(10, 2)) FOR x IN ['1', '2']]",
		"SELECT [TRY_CAST(x AS INTEGER) FOR x IN ['1', 'bad']]",
		"SELECT [DATE '2026-11-04' FOR x IN [1]]",
		"SELECT [CAST({'value': x} AS STRUCT(value INTEGER)) FOR x IN [1, 2]]",
		"SELECT [CAST([x] AS INTEGER[1]) FOR x IN [1, 2]]",
		"SELECT [x FOR x IN ['1', '2'] IF x::INTEGER > 1]",
		"SELECT [x.key FOR x IN sessions] AS k FROM runs",
		"SELECT [x FOR x IN event_ids IF x != ''] AS e FROM runs",
		"SELECT [upper(x) || '-' FOR x IN string_split(run_id, '-') IF length(x) > 2] AS p FROM runs",
		"SELECT [x FOR x IN [y FOR y IN event_ids]] AS e FROM runs",
		"SELECT [CASE WHEN x IS NULL THEN 0 ELSE 1 END FOR x IN event_ids] AS e FROM runs",
		"SELECT [x->>'key' FOR x IN sessions] AS k FROM runs",
		"SELECT [{'a': x, 'b': [x, 1]} FOR x IN event_ids IF x IN ('a', 'b')] AS e FROM runs",
		"WITH c AS (SELECT [x FOR x IN event_ids] AS e FROM runs) SELECT e FROM c",
	}
	for _, q := range accepts {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.NoError(t, err)
		})
	}

	const own = "a list comprehension can only reference its own variables"
	const nested = "a list comprehension can't be nested inside another list comprehension"
	rejects := map[string]string{
		"SELECT [CAST(run_id AS VARCHAR) FOR x IN event_ids] FROM runs":                                   own + `, not "run_id"`,
		"SELECT [TRY_CAST(run_id AS INTEGER) FOR x IN event_ids] FROM runs":                               own + `, not "run_id"`,
		"SELECT [x FOR x IN event_ids IF CAST(run_id AS VARCHAR) = x] FROM runs":                          own + `, not "run_id"`,
		"SELECT [CAST((SELECT max(run_id) FROM runs) AS VARCHAR) FOR x IN event_ids] FROM runs":           "can only use its own variables, literals",
		"SELECT [run_id FOR x IN string_split(run_id, '')] AS r FROM runs":                                own + `, not "run_id"`,
		"SELECT [x || run_id FOR x IN event_ids] AS r FROM runs":                                          own + `, not "run_id"`,
		"SELECT [x FOR x IN event_ids IF x <> run_id] AS r FROM runs":                                     own + `, not "run_id"`,
		"SELECT [runs.run_id FOR x IN event_ids] AS r FROM runs":                                          own + `, not "runs.run_id"`,
		"SELECT [runs.run_id FOR runs IN event_ids] AS r FROM runs":                                       own + `, not "runs.run_id"`,
		"SELECT [#1 FOR x IN event_ids] AS r FROM runs":                                                   "can only use its own variables, literals",
		"SELECT [x FOR x IN event_ids IF x = (SELECT max(run_id) FROM runs)] AS r FROM runs":              "can only use its own variables, literals",
		"SELECT [[y FOR y IN string_split(run_id, '')] FOR x IN string_split(run_id, '')] AS r FROM runs": nested,
		"SELECT [x FOR x IN event_ids IF len([y FOR y IN event_ids]) > 0] AS r FROM runs":                 nested,
		"SELECT [x FOR x IN [[y FOR y IN [z FOR z IN event_ids]] FOR w IN event_ids]] AS r FROM runs":     nested,
		"WITH c AS (SELECT [run_id FOR x IN event_ids] AS e FROM runs) SELECT e FROM c":                   own + `, not "run_id"`,
		"SELECT run_id FROM runs WHERE len([run_id FOR x IN event_ids]) > 0":                              own + `, not "run_id"`,
	}
	for q, want := range rejects {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.ErrorContains(t, err, want)
		})
	}
}

// TestComprehensionFanOutCaseInsensitive checks the fan-out rules on their
// own (not via validate): variables and table names match regardless of
// case, like DuckDB identifiers.
func TestComprehensionFanOutCaseInsensitive(t *testing.T) {
	const own = "a list comprehension can only reference its own variables"
	cases := map[string]string{
		"SELECT [X FOR x IN event_ids] AS e FROM runs":                   "",
		"SELECT [upper(x) FOR X IN event_ids IF x != ''] AS e FROM runs": "",
		"SELECT [X->>'key' FOR x IN sessions] AS k FROM runs":            "",
		"SELECT [Runs.run_id FOR runs IN event_ids] AS r FROM runs":      own + `, not "Runs.run_id"`,
		"SELECT [r.run_id FOR R IN event_ids] AS e FROM runs AS r":       own + `, not "r.run_id"`,
	}
	for q, want := range cases {
		t.Run(q, func(t *testing.T) {
			err := checkComprehensionFanOut(mustParse(t, q))
			if want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, want)
		})
	}
}
