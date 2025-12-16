package debug

import (
	dbgpb "github.com/inngest/inngest/proto/gen/debug/v1"
	"google.golang.org/grpc"
)

type ContextKey struct{}

var CtxKey = ContextKey{}

type Context struct {
	Client dbgpb.DebugClient
	Conn   *grpc.ClientConn
}
