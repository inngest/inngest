package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngest-cli/inngest/client"
	"github.com/inngest/inngest-cli/inngest/clistate"
	"github.com/inngest/inngest-cli/inngest/log"
)

// findWorkflow finds a workflow given a UUID or a UUID prefix.
func findWorkflow(ctx context.Context, idOrPrefix string) (*client.Workflow, error) {
	s := clistate.RequireState(ctx)

	ws, err := clistate.Workspace(ctx)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idOrPrefix)
	if err == nil {
		return s.Client.Workflow(ctx, ws.ID, id)
	}

	flows, err := s.Client.Workflows(ctx, ws.ID)
	if err != nil {
		log.From(ctx).Fatal().Err(err).Msg("unable to fetch workspaces")
	}

	candidates := []*client.Workflow{}
	for _, f := range flows {
		copied := f
		if f.Slug == idOrPrefix {
			return &copied, nil
		}

		if strings.HasPrefix(f.Slug, idOrPrefix) {
			candidates = append(candidates, &copied)
		}
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("No workflow in workspace '%s' found for ID: %s", ws.Name, idOrPrefix)
	}

	return nil, fmt.Errorf("More than one workflow found with the prefix: %s", idOrPrefix)
}

// formatTime is a helper function which formats the time for human output.
func formatTime(d *time.Time) string {
	if d == nil || d.IsZero() {
		return ""
	}
	return d.Format(time.UnixDate)
}
