package memory

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestScopedKeyIgnoresEvaluatedHashWithoutExpression(t *testing.T) {
	accountID := uuid.New()
	entityID := uuid.New()

	for _, kind := range []byte{'n', 'r', 't'} {
		t.Run(string(kind), func(t *testing.T) {
			require.Equal(t,
				scopedKey(kind, accountID, 1, entityID, "", "first"),
				scopedKey(kind, accountID, 1, entityID, "", "second"),
			)
			require.NotEqual(t,
				scopedKey(kind, accountID, 1, entityID, "expression", "first"),
				scopedKey(kind, accountID, 1, entityID, "expression", "second"),
			)
		})
	}
}
