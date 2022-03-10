package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/inngest/inngestctl/inngest/state"
)

// findWorkflow finds a workflow given a UUID or a UUID prefix.
func findWorkflow(ctx context.Context, idOrPrefix string) (*client.Workflow, error) {
	state := state.RequireState(ctx)

	id, err := uuid.Parse(idOrPrefix)
	if err == nil {
		return state.Client.Workflow(ctx, state.SelectedWorkspace.ID, id)
	}

	flows, err := state.Client.Workflows(ctx, state.SelectedWorkspace.ID)
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
		return nil, fmt.Errorf("No workflow in workspace '%s' found for ID: %s", state.SelectedWorkspace.Name, idOrPrefix)
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
