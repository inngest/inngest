package state

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/enums"
	"github.com/stretchr/testify/require"
)

func TestCustomConcurrency_ParseKey(t *testing.T) {
	r := require.New(t)
	someUUID := uuid.New()
	someHash := strconv.FormatUint(xxhash.Sum64String("abcdef0123456789"), 36)

	t.Run("function scope", func(t *testing.T) {
		sut := CustomConcurrency{
			Key:   fmt.Sprintf("f:%s:%s", someUUID.String(), someHash),
			Hash:  someHash,
			Limit: 20,
		}

		scope, id, hash, err := sut.ParseKey()
		r.NoError(err)
		r.Equal(enums.ConcurrencyScopeFn, scope)
		r.Equal(someUUID, id)
		r.Equal(someHash, hash)
	})

	t.Run("environment scope", func(t *testing.T) {
		sut := CustomConcurrency{
			Key:   fmt.Sprintf("e:%s:%s", someUUID.String(), someHash),
			Hash:  someHash,
			Limit: 20,
		}

		scope, id, hash, err := sut.ParseKey()
		r.NoError(err)
		r.Equal(enums.ConcurrencyScopeEnv, scope)
		r.Equal(someUUID, id)
		r.Equal(someHash, hash)
	})
}
