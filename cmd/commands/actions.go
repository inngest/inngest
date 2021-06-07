package commands

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(actionsRoot)
	actionsRoot.AddCommand(actionsList)
	actionsRoot.AddCommand(actionsDeploy)
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
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		if len(args) == 0 {
			log.From(ctx).Fatal().Msg("No cue configuration found")
		}

		state := inngest.RequireState(ctx)
		if state.SelectedWorkspace == nil {
			log.From(ctx).Fatal().Msg("You have no workspace selected")
		}

		path, err := homedir.Expand(args[0])
		if err != nil {
			log.From(ctx).Fatal().Msg("Error finding configuration")
		}

		byt, err := os.ReadFile(path)
		if err != nil {
			log.From(ctx).Fatal().Msgf("Error reading configuration: %s", err)
		}

		if err := inngest.DeployAction(ctx, inngest.DeployActionOptions{
			Config: string(byt),
			Client: state.Client,
		}); err != nil {
			log.From(ctx).Fatal().Msgf("Error deploying: %s", err)
		}
	},
}

func doDeploy(ctx context.Context, state *inngest.State) error {
	// TODO: build and push, plus just push.
	image := "127.0.0.1:9988/action-test-hello-world:latest"

	client, err := docker.NewClientWithOpts(docker.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}

	rc, err := client.ImagePush(ctx, image, types.ImagePushOptions{
		RegistryAuth: auth(state),
	})
	if err != nil {
		return err
	}
	defer rc.Close()

	err = jsonmessage.DisplayJSONMessagesStream(rc, os.Stderr, uintptr(syscall.Stderr), true, nil)
	if err != nil {
		var msgerr *jsonmessage.JSONError
		if errors.As(err, &msgerr) {
			return fmt.Errorf("%s", msgerr.Message)
		}
		return fmt.Errorf("error displaying push status: %w", err)
	}

	return nil
}

func auth(state *inngest.State) string {
	authConfig := types.AuthConfig{
		Username: "jwt",
		Password: string(state.Credentials),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON)
}
