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
		"SELECT range(length(run_id)) AS r FROM runs":                 "RANGE requires number literal arguments",
		"SELECT generate_series(0, 10000) AS r":                       "GENERATE_SERIES output size is limited",
		"SELECT generate_series(-300000000, 0) AS r":                  "GENERATE_SERIES output size is limited",
		"SELECT generate_series(1, 2, 0.0000001) AS r":                "GENERATE_SERIES output size is limited",
		"SELECT generate_series(1, length(run_id)) AS r FROM runs":    "GENERATE_SERIES requires number literal arguments",
	}
	for q, want := range rejects {
		t.Run(q, func(t *testing.T) {
			_, _, _, err := validate(mustParse(t, q))
			require.ErrorContains(t, err, want)
		})
	}
}
