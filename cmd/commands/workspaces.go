package commands

import (
	"github.com/google/uuid"
	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(workspacesRoot)
	workspacesRoot.AddCommand(workspacesList)
	workspacesRoot.AddCommand(workspacesSelect)
}

var workspacesRoot = &cobra.Command{
	Use:   "workspaces",
	Short: "Manages workspacess within your Inngest account",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var workspacesList = &cobra.Command{
	Use:   "list",
	Short: "Lists all workspaces within your Inngest account",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		state := inngest.RequireState(ctx)
		flows, err := state.Client.Workspaces(ctx)
		if err != nil {
			log.From(ctx).Fatal().Err(err).Msg("unable to fetch workspaces")
		}

		t := table.New(table.Row{"Selected", "ID", "Name", "Type"})
		for _, f := range flows {
			typ := "live"
			if f.Test {
				typ = "test"
			}

			selected := ""
			if state.SelectedWorkspace != nil && state.SelectedWorkspace.ID == f.ID {
				selected = "***"
			}

			t.AppendRow(table.Row{
				selected,
				f.ID,
				f.Name,
				typ,
			})
		}
		t.Render()
	},
}

var workspacesSelect = &cobra.Command{
	Use:   "select",
	Short: "Select a workspace for modification",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		state := inngest.RequireState(ctx)

		if len(args) == 0 {
			log.From(ctx).Fatal().Msg("No workspace ID passed to select. Usage: workspaces select [ID]")
		}

		id, err := uuid.Parse(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Invalid workspace ID")
		}

		flows, err := state.Client.Workspaces(ctx)
		if err != nil {
			log.From(ctx).Fatal().Err(err).Msg("unable to fetch workspaces")
		}

		var found *client.Workspace
		for _, f := range flows {
			if f.ID == id {
				found = &f
				break
			}
		}

		if found == nil {
			log.From(ctx).Fatal().Msg("Workspace not found")
		}

		if err := state.SetWorkspace(ctx, *found); err != nil {
			log.From(ctx).Fatal().Msgf("Error setting workspace: %s", err)
		}

		log.From(ctx).Info().Msg("Workspace selected")
	},
}
