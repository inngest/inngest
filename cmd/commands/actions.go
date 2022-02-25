package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/inngest/inngestctl/cmd/commands/internal/actions"
	"github.com/inngest/inngestctl/cmd/commands/internal/state"
	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	pushOnly      bool
	includePublic bool
	versionRegex  = regexp.MustCompile(`^v?([0-9]+).([0-9]+)$`)
	spacesRegex   = regexp.MustCompile(`\s`)
)

const (
	actionComment = `// For documentation on action configuration, visit https://docs.inngest.com/docs/actions`
)

func init() {
	rootCmd.AddCommand(actionsRoot)
	actionsRoot.AddCommand(actionsList)
	actionsRoot.AddCommand(actionsNew)
	actionsRoot.AddCommand(actionsValidate)
	actionsRoot.AddCommand(actionsDeploy)

	actionsDeploy.Flags().BoolVar(&pushOnly, "push-only", false, "Only push the action code;  do not create the action version")
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
		state := state.RequireState(ctx)
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

var actionsValidate = &cobra.Command{
	Use:   "validate [~/path/to/action.cue]",
	Short: "Ensures that the configuration is valid",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("No cue configuration found")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		prefix := ""
		if state, _ := state.GetState(ctx); state != nil {
			if state.Account.Identifier.Domain == nil {
				prefix = state.Account.Identifier.DSNPrefix
			} else {
				prefix = *state.Account.Identifier.Domain
			}
		}

		path, err := homedir.Expand(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Error finding configuration")
		}
		byt, err := os.ReadFile(path)
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}
		_, formatted, err := actions.Parse(prefix, string(byt))
		if err != nil {
			log.From(ctx).Fatal().Msgf("Invalid configuration: %s", err)
		}

		if formatted == string(byt) {
			log.From(ctx).Info().Msg("Valid action configuration")
			return
		}

		log.From(ctx).Info().Msg("Valid action configuration, with the following changes automatically applied on deploy:")
		fmt.Println(formatted)

	},
}

var actionsDeploy = &cobra.Command{
	Use:   "deploy [~/path/to/action.cue]",
	Short: "Pushes an action to your account and publishes the action for immediate use (skip publishing with --push-only)",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("No cue configuration found")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		state := state.RequireState(ctx)

		path, err := homedir.Expand(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Error finding configuration")
		}

		byt, err := os.ReadFile(path)
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}

		version, cueConfig, err := actions.Parse(state.Account.Identifier.DSNPrefix, string(byt))
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}

		version, err = inngest.DeployAction(ctx, inngest.DeployActionOptions{
			PushOnly: pushOnly,
			Config:   cueConfig,
			Client:   state.Client,
			Version:  version,
		})
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error deploying: %s.", err)
		}

		return
	},
}

var actionsNew = &cobra.Command{
	Use:   "new",
	Short: "Creates a config file for deploying a new serverless action",
	RunE: func(cmd *cobra.Command, args []string) error {
		prefix := ""
		if state, _ := state.GetState(cmd.Context()); state != nil {
			prefix = state.Account.Identifier.DSNPrefix
		}

		c := actions.Config{}
		if err := c.Survey(prefix); err != nil {
			return err
		}

		data, err := c.Configuration()
		if err != nil {
			return err
		}

		ioutil.WriteFile("./action.cue", []byte(data), 0600)
		fmt.Println("Created an action configuration file: ./action.cue")
		fmt.Println("")
		fmt.Println("Edit this file with your configuration and deploy using `inngestctl actions deploy`.")

		return nil
	},
}
