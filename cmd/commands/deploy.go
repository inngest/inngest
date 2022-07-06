package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/inngest/inngest-cli/inngest"
	"github.com/inngest/inngest-cli/inngest/client"
	"github.com/inngest/inngest-cli/inngest/state"
	"github.com/inngest/inngest-cli/internal/cuedefs"
	"github.com/inngest/inngest-cli/pkg/cli"
	"github.com/inngest/inngest-cli/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest-cli/pkg/function"
	"github.com/spf13/cobra"
)

var (
	ErrAlreadyDeployed = fmt.Errorf("This action has already been deployed")
)

func NewCmdDeploy() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "deploy",
		Short:   "Deploy a function to Inngest",
		Example: "inngest deploy",
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

	// Build all steps.
	steps, err := dockerdriver.FnBuildOpts(ctx, *fn, "--platform", "linux/amd64")
	if err != nil {
		return err
	}
	ui, err := cli.NewBuilder(ctx, cli.BuilderUIOpts{
		QuitOnComplete: true,
		BuildOpts:      steps,
	})
	if err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}
	if err := ui.Start(ctx); err != nil {
		fmt.Println("\n" + cli.RenderError(err.Error()) + "\n")
		os.Exit(1)
	}

	// Push each action
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

	fmt.Println(cli.BoldStyle.Render(fmt.Sprintf("Deploying function %s...", wflow.Name)))
	v, err := s.Client.DeployWorkflow(ctx, ws.ID, config, true)
	if err != nil {
		return fmt.Errorf("failed to deploy function: %w", err)
	}

	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render(fmt.Sprintf("Function deployed as version %d", v.Version)))
	return nil
}

// deployAction deploys a given action to Inngest, creating a new version, pushing the image,
// the setting the action to "published" once pushed.
func deployAction(ctx context.Context, a inngest.ActionVersion) error {
	var err error

	state := state.RequireState(ctx)

	// Ensure we normalize the DSN before building.
	a = normalizeDSN(ctx, a)

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
	if a.Runtime.RuntimeType() != found.Runtime.RuntimeType() {
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
