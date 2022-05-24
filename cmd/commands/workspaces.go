package commands

import (
	"github.com/inngest/inngest-cli/cmd/commands/internal/table"
	"github.com/inngest/inngest-cli/inngest/log"
	"github.com/inngest/inngest-cli/inngest/state"
	"github.com/spf13/cobra"
)

func NewCmdWorkspaces() *cobra.Command {
	workspacesRoot := &cobra.Command{
		Use:    "workspaces",
		Short:  "Manages workspacess within your Inngest account",
		Run:    listWorkspaces,
		Hidden: true,
	}

	workspacesList := &cobra.Command{
		Use:   "list",
		Short: "Lists all workspaces within your Inngest account",
		Run:   listWorkspaces,
	}

	workspacesRoot.AddCommand(workspacesList)

	return workspacesRoot
}

func listWorkspaces(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	state := state.RequireState(ctx)
	flows, err := state.Client.Workspaces(ctx)
	if err != nil {
		log.From(ctx).Fatal().Err(err).Msg("unable to fetch workspaces")
	}

	t := table.New(table.Row{"ID", "Name", "Type"})
	for _, f := range flows {
		typ := "live"
		if f.Test {
			typ = "test"
		}

		t.AppendRow(table.Row{
			f.ID,
			f.Name,
			typ,
		})
	}
	t.Render()
}
