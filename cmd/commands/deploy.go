package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/inngest/inngestctl/internal/cuedefs"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/spf13/cobra"
)

var (
	ErrAlreadyDeployed = fmt.Errorf("This action has already been deployed")
)

func NewCmdDeploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy a function to Inngest",
		Example: "inngestctl deploy",
		Run:     doDeploy,
	}
	return cmd
}

func doDeploy(cmd *cobra.Command, args []string) {
	fmt.Println(cli.EnvString())

	if err := deployFunction(cmd, args); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
}

func deployFunction(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	fn, err := function.Load(ctx, ".")
	if err != nil {
		return err
	}

	actions, _, err := fn.Actions(ctx)
	if err != nil {
		return err
	}

	for _, a := range actions {
		if err := deployAction(ctx, a); err != nil {
			return err
		}
	}

	return deployWorkflow(ctx, fn)
}

func deployWorkflow(ctx context.Context, fn *function.Function) error {
	s := state.RequireState(ctx)

	ws, err := state.Workspace(ctx)
	if err != nil {
		return err
	}

	wflow, err := fn.Workflow(ctx)
	if err != nil {
		return err
	}

	config, err := cuedefs.FormatWorkflow(*wflow)
	if err != nil {
		return err
	}

	fmt.Println(cli.BoldStyle.Render(fmt.Sprintf("Deploying workflow %s...", wflow.Name)))
	v, err := s.Client.DeployWorkflow(ctx, ws.ID, config, true)
	if err != nil {
		return fmt.Errorf("failed to deploy workflow: %w", err)
	}

	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render(fmt.Sprintf("Workflow deployed as version %d", v.Version)))
	return nil
}

// deployAction deploys a given action to Inngest, creating a new version, pushing the image,
// the setting the action to "published" once pushed.
func deployAction(ctx context.Context, a inngest.ActionVersion) error {
	var err error

	state := state.RequireState(ctx)

	// Ensure we normalize the DSN before building.
	a = normalizeDSN(ctx, a)

	tag := a.DSN

	if a.Runtime.RuntimeType() == inngest.RuntimeTypeDocker {
		fmt.Println(cli.BoldStyle.Render(fmt.Sprintf("Building action %s...", tag)))

		// Build the image.  We always need to do this first to ensure we have
		// an up-to-date image and checksum for the action.
		ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
			QuitOnComplete: true,
			BuildOpts: []dockerdriver.BuildOpts{
				dockerdriver.BuildOpts{
					Path:     ".",
					Tag:      tag,
					Platform: "linux/amd64",
				},
			},
		})
		if err != nil {
			return err
		}
		if err := tea.NewProgram(ui).Start(); err != nil {
			return err
		}
		if ui.Error() != nil {
			// We don't want to repeat the docker build error in
			// the UI.
			return fmt.Errorf("Exiting after a build error")
		}
		fmt.Println("")
	}

	// configure version information, ensuring that we skip redeploying actions that are
	// already live.
	a, err = configureVersionInfo(ctx, a)
	if err == ErrAlreadyDeployed {
		fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render("This action has already been deployed."))
		return nil
	}
	if err != nil {
		return fmt.Errorf("error preparing action: %w", err)
	}

	config, err := cuedefs.FormatAction(a)
	if err != nil {
		return err
	}

	fmt.Println(cli.BoldStyle.Render(fmt.Sprintf("Deploying action version %s...", a.Version.String())))

	// Create the action in the API.
	if _, err = state.Client.CreateAction(ctx, config); err != nil {
		return fmt.Errorf("error creating action: %w", err)
	}

	if a.Runtime.RuntimeType() == inngest.RuntimeTypeDocker {
		// Push the docker image.
		switch a.Runtime.RuntimeType() {
		case "docker":
			if _, err = dockerdriver.Push(ctx, a, state.Client.Credentials()); err != nil {
				return fmt.Errorf("error pushing action: %w", err)
			}
		default:
			return fmt.Errorf("unknown runtime type: %s", a.Runtime.RuntimeType())
		}

		// Publish
		_, err = state.Client.UpdateActionVersion(ctx, client.ActionVersionQualifier{
			DSN:          a.DSN,
			VersionMajor: a.Version.Major,
			VersionMinor: a.Version.Minor,
		}, true)
	}

	if err == nil {
		fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render("Action deployed"))
	}

	return err

}

// normalizeDSN ensures that the action DSN has an account identifier prefix added.
func normalizeDSN(ctx context.Context, a inngest.ActionVersion) inngest.ActionVersion {
	state := state.RequireState(ctx)
	// Add your account identifier locally, before finding action versions.
	prefix := ""
	if state.Account.Identifier.Domain == nil {
		prefix = state.Account.Identifier.DSNPrefix
	} else {
		prefix = *state.Account.Identifier.Domain
	}
	if !strings.Contains(a.DSN, "/") {
		a.DSN = fmt.Sprintf("%s/%s", prefix, a.DSN)
	}
	return a
}

func configureVersionInfo(ctx context.Context, a inngest.ActionVersion) (inngest.ActionVersion, error) {
	state := state.RequireState(ctx)

	// If we're publishing a specific version, attempt to find it.  Else, load the latest
	// action version.  This automatically happens depending on whether a.Version is nil.
	found, err := state.Action(ctx, a.DSN, a.Version)
	// When deploying without specifying an action, allow "action not found"
	// errors.
	if a.Version == nil && err != nil && err.Error() == "action not found" {
		a.Version = &inngest.VersionInfo{
			Major: 1,
			Minor: 1,
		}
		return a, nil
	}
	if err != nil {
		return a, err
	}

	// If we're requesting that we deploy a specific version, check that it doesn't
	// exist.
	if a.Version != nil && found != nil && err == nil {
		return a, fmt.Errorf("Version %s of the action already exists", a.Version.String())
	}

	// Are the runtimes the same?
	if a.Runtime.Runtime != found.Runtime.Runtime {
		a.Version = &inngest.VersionInfo{
			Major: found.Version.Major,
			Minor: found.Version.Minor + 1,
		}
		return a, nil
	}

	switch a.Runtime.RuntimeType() {
	case inngest.RuntimeTypeHTTP:
		// The URL must be the same, which is covered in the check above.
		return a, ErrAlreadyDeployed

	case inngest.RuntimeTypeDocker:
		digest, err := dockerdriver.Digest(ctx, a)
		if err != nil {
			return a, err
		}

		// If the found action version already deployed has the same digest, we can skip
		// deploying altogether.
		if found != nil && found.ImageSha256 != nil && *found.ImageSha256 == digest {
			// XXX: Ensure that the image is live.
			return a, ErrAlreadyDeployed
		}

		if found != nil {
			// Deploy the next minor version.
			a.Version = &inngest.VersionInfo{
				Major: found.Version.Major,
				Minor: found.Version.Minor + 1,
			}
		}
	}

	return a, nil
}
