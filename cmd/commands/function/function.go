package function

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/inngest/inngestctl/pkg/build"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	pushLong = `
Push a function to Inngest. This will push a draft of the function to Inngest. 

To make a function live use 'inngest function deploy'
`
	deployLong = `
Deploy a function to Inngest. This will push the function and make it the current live version
that incoming events will trigger. 
`
	pushExample = `
$ inngestctl function push
$ inngestctl f push
`
	deployExample = `
$ inngestctl function deploy
$ inngestctl f deploy
`
)

func NewCmdFunction() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "function",
		Aliases: []string{"fn", "f"},
		Short:   "Work with Inngest functions",
	}

	cmd.AddCommand(funcPush())
	cmd.AddCommand(funcDeploy())
	cmd.AddCommand(funcList())
	return cmd
}

func funcPush() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push",
		Short:   "Push a function to Inngest",
		Long:    pushLong,
		Example: pushExample,
		Run: func(cmd *cobra.Command, args []string) {
			pushFunction(false, cmd)
		},
	}
	return cmd
}

func funcDeploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy a function to Inngest",
		Long:    deployLong,
		Example: deployExample,
		Run: func(cmd *cobra.Command, args []string) {
			pushFunction(true, cmd)
		},
	}
	return cmd
}

// pushFunction takes a function, breaks it out into the workflow
// and action/s for the workflow, builds the images and pushes to
// inngest.
func pushFunction(publish bool, cmd *cobra.Command) {
	ctx := cmd.Context()
	state := state.RequireState(ctx)

	f, err := loadAndValidateFunc()
	if err != nil {
		fmt.Printf("Error loading Inngest file: %v", err)
		return
	}

	actions, err := f.GetActions(state)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, action := range actions {
		// If this is a push, then version the action
		if !publish {
			action.Version.Minor++
		}

		config, err := inngest.FormatAction(*action)
		if err != nil {
			fmt.Println(err)
			return
		}

		//TODO: (DR) We currently only support a single dockerfile/action so this
		// will use the dir the command is ran from, similar to what Load is doing for
		// the inngest file. Once we support multi-step functions this will need to pass in the dockerfile
		// location instead of hard coding to '.'.
		// Build right before deploy in case anything goes wrong processing the action
		build.Build(cmd, []string{".", "--tag", action.Runtime.RuntimeImage(), "--platform", "linux/amd64"})

		version, err := inngest.DeployAction(context.Background(), inngest.DeployActionOptions{
			PushOnly: !publish,
			Config:   config,
			Client:   state.Client,
			Version:  action,
		})
		if err != nil {
			if !strings.Contains(err.Error(), "This version has already published") {
				fmt.Println(err)
				return
			}
		}
		fmt.Printf("Successfully deployed action - Name:%v Version:%v\n", version.Name, version.Version)
	}

	workflow, err := f.GetWorkflow(state)
	if err != nil {
		fmt.Printf("f.Workflow %v", err)
		return
	}

	wConfig, err := inngest.FormatWorkflow(*workflow)
	if err != nil {
		fmt.Printf("inngest.FormatWorkflow %v", err)
		return
	}

	v, err := state.Client.DeployWorkflow(ctx, state.SelectedWorkspace.ID, wConfig, publish)
	if err != nil {
		fmt.Printf("state.Client.DeployWorkflow %v", err)
		return
	}

	if v.ValidFrom != nil && v.ValidFrom.Before(time.Now()) {
		fmt.Printf("Successfully deployed workflow version %d live\n", v.Version)
		return
	}

	fmt.Printf("Successfully deployed workflow version %d as a draft\n", v.Version)

}

func funcList() *cobra.Command {
	workflowsList := &cobra.Command{
		Use:   "list",
		Short: "Lists functions within the current workspace, defaulting to live functions.  Use --all for all functions.",
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

			allWorkflows := viper.GetBool("all")

			for _, f := range flows {
				if f.Current == nil && !allWorkflows {
					continue
				}

				row := table.Row{
					f.Slug,
					f.Name,
				}

				if f.Current == nil && allWorkflows {
					row = append(row, "", "", "", "")
					t.AppendRow(row)
					continue
				}

				triggers := make([]string, len(f.Current.Triggers))
				for n, t := range f.Current.Triggers {
					triggers[n] = t.String()
				}

				row = append(
					row,
					f.Current.Version,
					f.Current.ValidFrom,
					strings.Join(triggers, ", "),
					p.Sprintf("%d", f.Usage.Total),
				)
				t.AppendRow(row)
			}
			t.Render()
		},
	}
	workflowsList.Flags().Bool("all", false, "Show all functions including drafts and archived (instead of only live functions)")
	return workflowsList
}

func loadAndValidateFunc() (*function.Function, error) {
	// TODO: (DR) This should be a flag
	f, err := function.Load(".")
	if err != nil {
		if errors.Is(err, function.ErrNotFound) {
			return nil, errors.New("inngest file not found, please run 'inngestctl init' to create it")
		}
		return nil, err
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}

	return f, nil
}
