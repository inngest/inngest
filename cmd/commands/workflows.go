package commands

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/inngest/inngestctl/cmd/commands/internal/state"
	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/cmd/commands/internal/workflows"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	allWorkflows bool
	versionFlag  int
)

func init() {
	rootCmd.AddCommand(workflowsRoot)
	workflowsRoot.AddCommand(workflowsList)

	// Root by default calls list, so add the All flag to both.
	workflowsRoot.Flags().BoolVar(&allWorkflows, "all", false, "Show all workflows including drafts and archived flows (instead of only live flows)")
	workflowsList.Flags().BoolVar(&allWorkflows, "all", false, "Show all workflows including drafts and archived flows (instead of only live flows)")
	workflowConfig.Flags().IntVar(&versionFlag, "version", 0, "The version of the workflow to select")

	// Allow showing a single workflow's config
	workflowsRoot.AddCommand(workflowConfig)
	workflowsRoot.AddCommand(workflowVersions)
	workflowsRoot.AddCommand(workflowNew)
}

var workflowsRoot = &cobra.Command{
	Use:   "workflows",
	Short: "Manages workflows within your Inngest account",
}

var workflowsList = &cobra.Command{
	Use:   "list",
	Short: "Lists workflows within the current workspace, defaulting to live workflows.  Use --all for all workflows.",
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

var workflowVersions = &cobra.Command{
	Use:   "versions",
	Short: "Shows a workflow's version information.",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("No workflow specified.  Specify a workflow ID (or it's prefix)")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		workflow, err := findWorkflow(ctx, args[0])

		if err != nil {
			log.From(ctx).Fatal().Err(err).Msg("Unable to find workflow")
		}

		//  There isn't a live config for this workflow, so find the latest version and show that.
		versions := append(workflow.Drafts, workflow.Previous...)
		if workflow.Current != nil {
			versions = append(versions, *workflow.Current)
		}

		// Sort the versions by updated at, so that newer drafts show first.  Note that version
		// numbers don't always represent oldest -> newest, as you can publish an old draft.
		sort.SliceStable(versions, func(i, j int) bool {
			return versions[i].UpdatedAt.After(versions[j].UpdatedAt)
		})

		t := table.New(table.Row{"Version", "Live", "Live since", "Live until", "Last updated"})

		for _, v := range versions {
			live := workflow.Current != nil && workflow.Current.Version == v.Version

			row := table.Row{
				v.Version,
			}

			if live {
				row = append(row, "Yes")
			} else {
				row = append(row, "")
			}

			row = append(row, formatTime(v.ValidFrom), formatTime(v.ValidTo), formatTime(&v.UpdatedAt))

			t.AppendRow(row)
		}
		t.Render()
	},
}

var workflowConfig = &cobra.Command{
	Use:   "config",
	Short: "Shows a workflow's configuration given its ID or prefix",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("No workflow ID specified.  Specify a workflow ID or it's prefix")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		state := state.RequireState(ctx)

		workflow, err := findWorkflow(ctx, args[0])
		if err != nil {
			log.From(ctx).Fatal().Err(err).Msg("Unable to find workflow")
		}

		if versionFlag != 0 && workflow.Current != nil && workflow.Current.Version != versionFlag {
			// Request that specific version, as by default we only request the config for
			// the current version in the list.
			v, err := state.Client.WorkflowVersion(ctx, state.SelectedWorkspace.ID, workflow.ID, versionFlag)
			if err != nil {
				log.From(ctx).Fatal().Err(err).Msg("Unable to find workflow version")
			}
			fmt.Println(v.Config)
			return
		}

		// Show the current version by default
		if workflow.Current != nil {
			fmt.Println(workflow.Current.Config)
			return
		}

		log.From(ctx).Fatal().Err(err).Msg("No live version to show, and no version supplied with --version flag.  Show a specific version using the --version flag")
	},
}

var workflowNew = &cobra.Command{
	Use:   "new [name]",
	Short: "Creates a config file for a new workflow",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := workflows.Config{}
		if err := c.Survey(); err != nil {
			return err
		}

		data, err := c.Configuration()
		if err != nil {
			return err
		}

		ioutil.WriteFile("./workflow.cue", []byte(data), 0600)
		fmt.Println("Created a workflow configuration file: ./workflow.cue")
		fmt.Println("")
		fmt.Println("Edit this file with your configuration and deploy using `inngestctl workflows deploy`.")
		return nil
	},
}
