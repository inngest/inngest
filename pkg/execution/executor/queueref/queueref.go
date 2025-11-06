package queueref

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"

	"github.com/inngest/inngest/pkg/execution/queue"
)

// QueueRef is a tuple containing the shard ID and job ID
// referencing the currently executed job.  This is used to
// reset queue item attempts after a successful checkpoint.
//
// - The first part is the job ID
// - The second part is the shard ID
type QueueRef [2]string

func Decode(input string) QueueRef {
	byt, _ := base64.StdEncoding.DecodeString(input)
	parts := strings.Split(string(byt), "::")
	if len(parts) == 2 {
		return QueueRef{parts[0], parts[1]}
	}
	return QueueRef{}
}

func FromCtx(ctx context.Context) QueueRef {
	return QueueRef{
		queue.JobIDFromContext(ctx),
		queue.ShardIDFromContext(ctx),
	}
}

func StringFromCtx(ctx context.Context) string {
	return FromCtx(ctx).String()
}

func (q QueueRef) String() string {
	return base64.StdEncoding.EncodeToString(
		bytes.Join([][]byte{
			[]byte(q[0]),
			[]byte(q[1]),
		}, []byte("::")),
	)
}

func (q QueueRef) JobID() string {
	return q[0]
}

func (q QueueRef) ShardID() string {
	return q[1]
}
