package executor

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/inngest/inngestgo"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestCancelExpressionGen(t *testing.T) {
	future, err := time.Parse(time.RFC3339, "2038-01-01T01:30:00.00Z")
	require.NoError(t, err)

	tests := []struct {
		EventID    ulid.ULID
		Expression *string
		Expected   string
	}{
		// Future
		{
			EventID:    ulid.MustNew(uint64(future.UnixMilli()), rand.Reader),
			Expression: nil,
			Expected:   "(async.ts == null || async.ts > 2145922200000)",
		},
		{
			EventID:    ulid.MustNew(uint64(future.UnixMilli()), rand.Reader),
			Expression: inngestgo.StrPtr("async.data.ok == true"),
			Expected:   "async.data.ok == true && (async.ts == null || async.ts > 2145922200000)",
		},
	}

	for _, test := range tests {
		actual := generateCancelExpression(test.EventID, test.Expression)
		require.Equal(t, test.Expected, actual)
	}
}
