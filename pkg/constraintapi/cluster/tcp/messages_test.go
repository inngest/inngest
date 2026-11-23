package tcp

import (
	"testing"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/constraintapi/cluster"
	"github.com/inngest/inngest/pkg/constraintapi/cluster/raftcoord"
	"github.com/stretchr/testify/require"
)

// TestCoordinatorMessagesRoundTrip checks the raftcoord messages on the
// wire.  the replication messages are checked in the cluster package.
func TestCoordinatorMessagesRoundTrip(t *testing.T) {
	msgs := []cluster.Message{
		raftcoord.ProposeRequest{Command: []byte(`{"kind":"init"}`)},
		raftcoord.ReportRequest{
			Report: cluster.Report{
				Node:   "node-b",
				BootID: uuid.NewString(),
				LSN: map[int]uint64{
					0: 17,
					5: 1 << 60,
				},
			},
		},
		raftcoord.Reply{
			Status: raftcoord.StatusRejected,
			Error:  "the table already has shards",
		},
		raftcoord.Reply{},
	}
	ids := map[byte]bool{}
	for _, m := range msgs {
		enc, err := cluster.EncodeMessage(nil, m)
		require.NoError(t, err)
		require.GreaterOrEqual(t, enc[0], cluster.MessageIDCoordinator)
		ids[enc[0]] = true
		got, err := cluster.DecodeMessage(enc)
		require.NoError(t, err)
		require.Equal(t, m, got)
	}
	require.Len(t, ids, 3)
}
