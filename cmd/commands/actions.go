package commands

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pushOnly             bool
	includePublic        bool
	deployWithoutPublish bool
	versionRegex         = regexp.MustCompile(`^v?([0-9]+).([0-9]+)$`)
)

func init() {
	rootCmd.AddCommand(actionsRoot)
	actionsRoot.AddCommand(actionsList)
	actionsRoot.AddCommand(actionsDeploy)
	actionsRoot.AddCommand(actionsPublish)

	actionsDeploy.Flags().BoolVar(&pushOnly, "push-only", false, "Only push the action code;  do not create the action version")
	actionsDeploy.Flags().BoolVar(&deployWithoutPublish, "no-publish", false, "Do not publish this action after deploying.  This action will only be useable by test workflows.")
	actionsList.Flags().BoolVar(&includePublic, "public", false, "Include publicly available actions")
}

var actionsRoot = &cobra.Command{
	Use:   "actions",
	Short: "Manages actions within your account",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var actionsList = &cobra.Command{
	Use:   "list",
	Short: "Lists all actions within your account",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		state := inngest.RequireState(ctx)
		_ = state

		actions, err := state.Client.Actions(ctx, includePublic)
		if err != nil {
			log.From(ctx).Fatal().Msg(err.Error())
		}

		t := table.New(table.Row{"DSN", "Name", "Latest", "Published at", "Revoked at"})
		for _, a := range actions {
			if a.Latest == nil {
				t.AppendRow(table.Row{a.DSN, a.Name})
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
				_ = unpublished
			}

			t.AppendRow(table.Row{
				a.DSN,
				a.Name,
				fmt.Sprintf("v%d.%d", a.Latest.VersionMajor, a.Latest.VersionMinor),
				published,
				unpublished,
			})
		}
		t.Render()
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
		ctx := cmd.Context()
		state := inngest.RequireState(ctx)

		path, err := homedir.Expand(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Error finding configuration")
		}

		byt, err := os.ReadFile(path)
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}

		version, err := inngest.DeployAction(ctx, inngest.DeployActionOptions{
			PushOnly: pushOnly,
			Config:   string(byt),
			Client:   state.Client,
		})
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error deploying: %s", err)
		}

		if !deployWithoutPublish {
			actionsPublish.Run(cmd, []string{
				version.DSN,
				fmt.Sprintf("v%d.%d", version.Version.Major, version.Version.Minor),
			})
		}
	},
}

var actionsPublish = &cobra.Command{
	Use:   "publish [dsn] [version, eg. v1.12]",
	Short: "Pubishes a specific action version for use within workflows",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("An action DSN and version must be spplied, eg: $ inngestctl actions publish my-account/hello world v1.1")
		}
		// Check action version
		match := versionRegex.MatchString(args[1])
		if !match {
			return errors.New("Verion must be specified in the format of ${major}.${minor}, eg. v1.23 or 2.54")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		state := inngest.RequireState(ctx)

		v := versionRegex.FindStringSubmatch(args[1])
		major, err := strconv.Atoi(v[1])
		if err != nil {
			log.From(ctx).Fatal().Msg(err.Error())
		}
		minor, err := strconv.Atoi(v[2])
		if err != nil {
			log.From(ctx).Fatal().Msg(err.Error())
		}

		log.From(ctx).Info().Msg("Publishing action")
		_, err = state.Client.UpdateActionVersion(ctx, client.ActionVersionQualifier{
			DSN:          args[0],
			VersionMajor: major,
			VersionMinor: minor,
		}, true)
		if err != nil {
			log.From(ctx).Fatal().Msg(err.Error())
		}
		log.From(ctx).Info().Msg("Action published")
	},
}
