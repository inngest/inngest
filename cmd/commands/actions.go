package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pushOnly      bool
	includePublic bool
)

func init() {
	rootCmd.AddCommand(actionsRoot)
	actionsRoot.AddCommand(actionsList)
	actionsRoot.AddCommand(actionsDeploy)

	actionsDeploy.Flags().BoolVar(&pushOnly, "push-only", false, "Only push the action code;  do not create the action version")
	actionsList.Flags().BoolVar(&includePublic, "public", false, "Include publicly available actions")
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
	Short: "Lists all actions within your account",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		state := inngest.RequireState(ctx)
		_ = state

		actions, err := state.Client.Actions(ctx, includePublic)
		if err != nil {
			log.From(ctx).Fatal().Msg(err.Error())
		}

		fmt.Println("")
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
		fmt.Fprint(w, "DSN\tNAME\tLATEST VERSION\tPUBLISHED AT\tUNPUBLISHED AT\n")
		for _, a := range actions {
			if a.Latest == nil {
				fmt.Fprintf(w, "%s\t%s\t-\t-\n", a.DSN, a.Name)
				continue
			}

			published := "-"
			unpublished := "-"
			if a.Latest.ValidFrom != nil {
				published = a.Latest.ValidFrom.Format(time.RFC3339)
				if a.Latest.ValidFrom.After(time.Now()) {
					published = fmt.Sprintf("%s (scheduled)", published)
				}
			}
			if a.Latest.ValidTo != nil {
				unpublished = a.Latest.ValidTo.Format(time.RFC3339)
			}

			fmt.Fprintf(w, "%s\t%s\tv%d.%d\t%s\t%s\n", a.DSN, a.Name, a.Latest.VersionMajor, a.Latest.VersionMinor, published, unpublished)
		}

		w.Flush()
		fmt.Println("")
	},
}

var actionsDeploy = &cobra.Command{
	Use:   "deploy [~/path/to/action.cue]",
	Short: "Deploys an action to your account",
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
