package commands

import (
	"context"
	"os"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pushOnly bool
)

func init() {
	rootCmd.AddCommand(actionsRoot)
	actionsRoot.AddCommand(actionsList)
	actionsRoot.AddCommand(actionsDeploy)

	actionsDeploy.Flags().BoolVar(&pushOnly, "push-only", false, "Only push the action code;  do not create the action version")
}

var actionsRoot = &cobra.Command{
	Use:   "actions",
	Short: "Manages actions within your selected workspace",
	Run: func(cmd *cobra.Command, args []string) {
		// With no arguments provided, default to listing the
		// available actions.
		actionsList.Run(cmd, args)
	},
}

var actionsList = &cobra.Command{
	Use:   "list",
	Short: "Lists all actions within your selected workspace",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		state := inngest.RequireState(ctx)
		err := state.Client.Actions(ctx, state.SelectedWorkspace.ID)
		if err != nil {
			log.From(ctx).Error().Err(err)
		}

		// TODO: List all actions
	},
}

var actionsDeploy = &cobra.Command{
	Use:   "deploy [~/path/to/action.cue]",
	Short: "Deploys an action to your selected workspace",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("No cue configuration found")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		state := inngest.RequireState(ctx)

		path, err := homedir.Expand(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Error finding configuration")
		}

		byt, err := os.ReadFile(path)
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}

		if err := inngest.DeployAction(ctx, inngest.DeployActionOptions{
			PushOnly: pushOnly,
			Config:   string(byt),
			Client:   state.Client,
		}); err != nil {
			log.From(ctx).Fatal().Msgf("Error deploying: %s", err)
		}
	},
}
