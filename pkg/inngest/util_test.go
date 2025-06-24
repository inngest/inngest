package inngest

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeterministicSha1UUID(t *testing.T) {
	src := "yolo"

	id1 := DeterministicSha1UUID(src)
	id2 := DeterministicSha1UUID(src)

	require.Equal(t, id1, id2)
}

func TestDeterministicUUIDV7(t *testing.T) {
	t.Run("should generate deterministic UUID v7", func(t *testing.T) {
		seed := "1234567890123456"

		/*
		    0                   1                   2                   3
		    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |                            unixts                             |
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |unixts |       subsec_a        |  ver  |       subsec_b        |
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |var|                   subsec_seq_node                         |
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |                       subsec_seq_node                         |
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		*/

		id1, err := DeterministicUUIDV7(seed)
		require.NoError(t, err)

		id2, err := DeterministicUUIDV7(seed)
		require.NoError(t, err)

		// The second half is guaranteed to match (var and subsec_seq_node must be the same for the same seed)
		require.Equal(t, id1[8:], id2[8:])
	})
}
