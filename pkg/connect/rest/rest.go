package rest

import (
	"github.com/inngest/inngest/pkg/connect/state"
	connpb "github.com/inngest/inngest/proto/gen/connect/v1"
)

type ShowConnsReply struct {
	Data []*connpb.ConnMetadata `json:"data"`
}

type WorkerGroup struct {
	state.WorkerGroup

	Synced bool     `json:"synced"`
	Conns  []string `json:"conns"`
}

type ShowWorkerGroupReply struct {
	Data *WorkerGroup `json:"data"`
}
