package debug

import (
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

type debugContextKey struct{}

var dbgCtxKey = debugContextKey{}

type DebugContext struct {
	Client dbgpb.DebugClient
	Conn   *grpc.ClientConn
}
