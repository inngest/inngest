package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngestctl/cmd/commands/internal/actions"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

func NewCmdActions() *cobra.Command {

	root := &cobra.Command{
		Use:    "actions",
		Hidden: true,
	}

	deploy := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy a serverless function",
		Example: "inngestctl actions deploy",
		Run:     runActionDeploy,
	}

	root.AddCommand(deploy)
	return root
}

func runActionDeploy(cmd *cobra.Command, args []string) {
	// This is a legacy command for deployin action from ./action.cue
	// configuration files.
	ctx := cmd.Context()
	state := state.RequireState(ctx)

	if len(args) == 0 {
		fmt.Println(cli.RenderError("No configuration provided"))
		os.Exit(1)
	}

	path, err := homedir.Expand(args[0])
	if err != nil {
		fmt.Println(cli.RenderError("Error reading configuration"))
		os.Exit(1)
	}

	byt, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(cli.RenderError("Error reading configuration"))
		os.Exit(1)
	}

	prefix := state.Account.Identifier.DSNPrefix
	if state.Account.Identifier.Domain != nil {
		prefix = *state.Account.Identifier.Domain
	}

	version, _, err := actions.Parse(prefix, string(byt))
	if err != nil {
		fmt.Println(cli.RenderError("Error parsing configuration"))
		os.Exit(1)
	}

	if err := deployAction(ctx, *version, false); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Error deploying action: %s", err)))
		os.Exit(1)
	}
	return
}
