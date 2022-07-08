package devserver

import (
	"context"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/config"
	"github.com/inngest/inngest-cli/pkg/execution/executor"
	"github.com/inngest/inngest-cli/pkg/execution/runner"
	"github.com/inngest/inngest-cli/pkg/service"
)

// Create and start a new dev server (API, Exectutor, State, Logger, etc.)
func NewDevServer(ctx context.Context, c config.Config) error {
	api := api.NewService(c)
	runner := runner.NewService(c)
	exec := executor.NewService(c)
	return service.StartAll(ctx, api, runner, exec)
}
