package resolvers

import (
	"github.com/inngest/inngest/pkg/api"
	"github.com/inngest/inngest/pkg/cqrs"
	"github.com/inngest/inngest/pkg/execution"
	"github.com/inngest/inngest/pkg/execution/queue"
	"github.com/inngest/inngest/pkg/execution/runner"
	"github.com/inngest/inngest/pkg/history_reader"
)

type Resolver struct {
	Data          cqrs.Manager
	HistoryReader history_reader.Reader
	Runner        runner.Runner
	Queue         queue.JobQueueReader
	EventHandler  api.EventHandler
	Executor      execution.Executor
	ServerKind    string

	// LocalSigningKey is the key used to sign events for self-hosted services.
	LocalSigningKey string

	// RequireKeys defines whether event and signing keys are required for the
	// server to function. If this is true and signing keys are not defined,
	// the server will still boot but core actions such as syncing, runs, and
	// ingesting events will not work.
	RequireKeys bool
}
