package cqrs

import (
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ShowConnsReply struct {
	Data []*connpb.ConnMetadata `json:"data"`
}
