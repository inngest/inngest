package commands

import (
	"strings"

	"github.com/inngest/inngestctl/cmd/commands/internal/state"
	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func init() {
	rootCmd.AddCommand(workflowsRoot)
	workflowsRoot.AddCommand(workflowsList)
}

var workflowsRoot = &cobra.Command{
	Use:   "workflows",
	Short: "Manages workflows within your Inngest account",
	Run: func(cmd *cobra.Command, args []string) {
		workflowsList.Run(cmd, args)
	},
}

var workflowsList = &cobra.Command{
	Use:   "list",
	Short: "Lists all workflows within the current workspace",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		state := state.RequireState(ctx)

		if state.SelectedWorkspace == nil {
			log.From(ctx).Fatal().Err(errors.New("no workspace")).Msg("No workspace selected")
		}

		flows, err := state.Client.Workflows(ctx, state.SelectedWorkspace.ID)
		if err != nil {
			log.From(ctx).Fatal().Err(err).Msg("unable to fetch workspaces")
		}

		t := table.New(table.Row{"ID", "Name", "Live version", "Live since", "Triggers", "24h usage"})
		p := message.NewPrinter(language.English)
		for _, f := range flows {

			row := table.Row{
				f.ID,
				f.Name,
			}

			if f.Current == nil {
				row = append(row, "", "", "")
			} else {
				triggers := make([]string, len(f.Current.Triggers))
				for n, t := range f.Current.Triggers {
					triggers[n] = t.String()
				}

				row = append(row, f.Current.Version, f.Current.ValidFrom, strings.Join(triggers, ", "))
			}

			row = append(row, p.Sprintf("%d", f.Usage.Total))

			t.AppendRow(row)
		}
		t.Render()
	},
}
