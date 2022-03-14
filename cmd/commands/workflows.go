package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/inngest/inngestctl/cmd/commands/internal/table"
	"github.com/inngest/inngestctl/cmd/commands/internal/workflows"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	allWorkflows    bool
	deployLive      bool
	canonicalFormat bool
	writeFormat     bool
	versionFlag     int
)

func NewCmdWorkflows() *cobra.Command {
	workflowsRoot := &cobra.Command{
		Use:   "workflows",
		Short: "Manages workflows within your Inngest account",
		// Hidden: true,
	}

	workflowsList := &cobra.Command{
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

	workflowFormat := &cobra.Command{
		Use:   "format",
		Short: "Formats a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, byt, err := readWorkflowFile(args)
			if err != nil {
				return err
			}

			parsed, err := inngest.ParseWorkflow(string(byt))
			if err != nil {
				return err
			}
			output, err := inngest.FormatWorkflow(*parsed)
			if err != nil {
				return err
			}

			if writeFormat {
				err = os.WriteFile(file, []byte(output), 0600)
				return err
			}

			fmt.Println(output)
			return nil
		},
	}

	workflowVersions := &cobra.Command{
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

	workflowConfig := &cobra.Command{
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

			printConfig := func(config string) {
				if canonicalFormat {
					parsed, err := inngest.ParseWorkflow(config)
					if err != nil {
						fmt.Println(config)
						return
					}
					if output, err := inngest.FormatWorkflow(*parsed); err == nil {
						fmt.Println(output)
						return
					}
				}
				fmt.Println(config)
			}

			if versionFlag != 0 && (workflow.Current == nil || workflow.Current.Version != versionFlag) {
				// Request that specific version, as by default we only request the config for
				// the current version in the list.
				v, err := state.Client.WorkflowVersion(ctx, state.SelectedWorkspace.ID, workflow.ID, versionFlag)
				if err != nil {
					log.From(ctx).Fatal().Err(err).Msg("Unable to find workflow version")
				}
				printConfig(v.Config)
				return
			}

			// Show the current version by default
			if workflow.Current != nil {
				printConfig(workflow.Current.Config)
				return
			}

			// XXX: If there's no current, show the latest version by default

			log.From(ctx).Fatal().Err(err).Msg("No live version to show, and no version supplied with --version flag.  Show a specific version using the --version flag")
		},
	}

	workflowNew := &cobra.Command{
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

			if err := ioutil.WriteFile("./workflow.cue", []byte(data), 0600); err != nil {
				fmt.Printf("Error writing workflow.cue file - error:%v", err)
			}
			fmt.Println("Created a workflow configuration file: ./workflow.cue")
			fmt.Println("")
			fmt.Println("Edit this file with your configuration and deploy using `inngestctl workflows deploy`.")
			return nil
		},
	}

	workflowDeploy := &cobra.Command{
		Use:          "deploy",
		Short:        "Deploys a workflow idempotently using a given config file or ./workflow.cue as a draft (use --live to push live)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			file, byt, err := readWorkflowFile(args)
			if err != nil {
				return err
			}

			s := state.RequireState(ctx)

			fmt.Printf("Deploying workflow %s...\n", file)
			v, err := s.Client.DeployWorkflow(ctx, s.SelectedWorkspace.ID, string(byt), deployLive)
			if err != nil {
				return fmt.Errorf("failed to deploy workflow: %w", err)
			}

			if v.ValidFrom != nil && v.ValidFrom.Before(time.Now()) {
				fmt.Printf("Deployed version %d live\n", v.Version)
				return nil
			}

			fmt.Printf("Deployed version %d as a draft\n", v.Version)
			return nil
		},
	}

	workflowValidate := &cobra.Command{
		Use:   "validate",
		Short: "Validates a workflow configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, byt, err := readWorkflowFile(args)
			if err != nil {
				return err
			}

			if _, err := inngest.ParseWorkflow(string(byt)); err != nil {
				return err
			}

			// XXX: Grab any event and action types from Inngest to perform static
			// typechecking of metadata

			fmt.Println("Workflow is valid")
			return nil
		},
	}

	workflowsRoot.AddCommand(workflowsList)

	// Allow showing a single workflow's config
	workflowsRoot.AddCommand(workflowConfig)
	workflowsRoot.AddCommand(workflowVersions)
	workflowsRoot.AddCommand(workflowNew)
	workflowsRoot.AddCommand(workflowDeploy)
	workflowsRoot.AddCommand(workflowFormat)
	workflowsRoot.AddCommand(workflowValidate)

	// Root by default calls list, so add the All flag to both.
	workflowsRoot.Flags().BoolVar(&allWorkflows, "all", false, "Show all workflows including drafts and archived flows (instead of only live flows)")
	workflowsList.Flags().BoolVar(&allWorkflows, "all", false, "Show all workflows including drafts and archived flows (instead of only live flows)")
	workflowConfig.Flags().IntVar(&versionFlag, "version", 0, "The version of the workflow to select")
	workflowConfig.Flags().BoolVar(&canonicalFormat, "format", false, "Whether to format the configuration")
	workflowDeploy.Flags().BoolVar(&deployLive, "live", false, "Deploy as the live, current version of the workflow")
	workflowFormat.Flags().BoolVar(&writeFormat, "write", false, "Edit the file in place with formatted config")

	return workflowsRoot
}

func readWorkflowFile(args []string) (string, []byte, error) {
	file := "./workflow.cue"
	if len(args) >= 1 {
		file = args[0]
	}

	byt, err := os.ReadFile(file)
	if err != nil {
		return file, nil, fmt.Errorf("unable to read workflow configuration file at '%s'", file)
	}
	return file, byt, nil
}
