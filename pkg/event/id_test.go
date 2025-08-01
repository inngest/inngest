package event

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/inngest/inngest/pkg/consts"
	"github.com/stretchr/testify/require"
)

func TestInternalIDSeedFromString(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		// A single idempotency key should always generate the same ULID.

		r := require.New(t)

		now := time.Now().UnixMilli()
		entropy := make([]byte, 10)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			now,
			base64.StdEncoding.EncodeToString(entropy),
		)

		seed := SeededIDFromString(idempotencyKey, 0)
		r.NotNil(seed)
		ulid1, err := seed.ToULID()
		r.NoError(err)
		ulid2, err := seed.ToULID()
		r.NoError(err)
		r.Equal(ulid1.String(), ulid2.String())
	})

	t.Run("index varies the ULID", func(t *testing.T) {
		// The same idempotency key with different indexes generates different
		// ULIDs.

		r := require.New(t)

		now := time.Now().UnixMilli()
		entropy := make([]byte, 10)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			now,
			base64.StdEncoding.EncodeToString(entropy),
		)

		seed0 := SeededIDFromString(idempotencyKey, 0)
		seed1 := SeededIDFromString(idempotencyKey, 1)
		r.NotNil(seed0)
		r.NotNil(seed1)
		ulid0, err := seed0.ToULID()
		r.NoError(err)
		ulid1, err := seed1.ToULID()
		r.NoError(err)
		r.NotEqual(ulid0.String(), ulid1.String())
	})

	t.Run("high index", func(t *testing.T) {
		// When the index is high (e.g. the max number events in a request), the
		// ULID is still unique for each event.

		r := require.New(t)

		now := time.Now().UnixMilli()
		entropy := make([]byte, 10)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			now,
			base64.StdEncoding.EncodeToString(entropy),
		)

		uniqueIDs := map[string]struct{}{}
		for i := 0; i < consts.MaxEvents; i++ {
			seed := SeededIDFromString(idempotencyKey, i)
			r.NotNil(seed)
			id, err := seed.ToULID()
			r.NoError(err)
			r.False(id.IsZero())
			uniqueIDs[id.String()] = struct{}{}
		}

		// No duplicates.
		r.Len(uniqueIDs, consts.MaxEvents)
	})

	t.Run("empty", func(t *testing.T) {
		r := require.New(t)

		seed := SeededIDFromString("", 0)
		r.Nil(seed)
	})

	t.Run("negative millis", func(t *testing.T) {
		r := require.New(t)

		entropy := make([]byte, 10)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			-1,
			base64.StdEncoding.EncodeToString(entropy),
		)

		seed := SeededIDFromString(idempotencyKey, 0)
		r.Nil(seed)
	})

	t.Run("too few bytes", func(t *testing.T) {
		r := require.New(t)

		now := time.Now().UnixMilli()
		entropy := make([]byte, 9)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			now,
			base64.StdEncoding.EncodeToString(entropy),
		)

		seed := SeededIDFromString(idempotencyKey, 0)
		r.Nil(seed)
	})

	t.Run("too many bytes", func(t *testing.T) {
		r := require.New(t)

		now := time.Now().UnixMilli()
		entropy := make([]byte, 11)
		_, err := rand.Read(entropy)
		r.NoError(err)
		idempotencyKey := fmt.Sprintf(
			"%d,%s",
			now,
			base64.StdEncoding.EncodeToString(entropy),
		)

		seed := SeededIDFromString(idempotencyKey, 0)
		r.Nil(seed)
	})
}
