package imagecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/inngest/inngest/cmd/image/internal/docker"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
)

// deployTag is the tag of every deploy.  deploys of an app differ by ID.
const deployTag = "latest"

// DeployCommand is `inngest deploy`, which deploys an app from inngest.toml.
// `inngest image deploy` is different: it builds and pushes a sandbox image.
func DeployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Build an app from inngest.toml and deploy it to Inngest compute",
		UsageText: "inngest deploy [app] [-- docker build args]",
		Description: `Reads inngest.toml in the current directory, builds the app's Dockerfile
for linux/amd64 with docker, uploads the image, and waits for Inngest to
deploy it.  The current directory is the docker build context.  Without
inngest.toml, it asks a few questions in a terminal and writes the file.

app is the name of an [apps.<name>] section.  It can be omitted when the
file has one app.  Run ` + "`inngest deploy example-config`" + ` for a commented example.

Arguments after -- go to docker build, as in ` + "`inngest image build`" + `, such as
` + "`inngest deploy web -- --build-arg VERSION=1`" + `.

Log in with ` + "`inngest login`" + ` first.  INNGEST_ENV selects the environment.`,
		Commands: []*cli.Command{
			{
				Name:  "example-config",
				Usage: "Print a commented example inngest.toml",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					_, stdout, _ := stdio(cmd)
					_, err := io.WriteString(stdout, exampleConfig)
					return err
				},
			},
		},
		Action: deployAppCommand,
	}
}

func deployAppCommand(ctx context.Context, cmd *cli.Command) error {
	// the first argument is the app name unless it is a docker build flag.
	// the rest go to docker build, as in `inngest image build`.
	args := cmd.Args().Slice()
	var appName string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		appName, args = args[0], args[1:]
	}

	stdin, stdout, stderr := stdio(cmd)
	f, err := os.Open(appsFile)
	if errors.Is(err, fs.ErrNotExist) {
		if isTerminal(stdin, stdout) {
			return writeAppsFile(stdin, stdout)
		}
		return errNoAppsFile
	}
	if err != nil {
		return err
	}
	apps, err := parseApps(f)
	_ = f.Close()
	if err != nil {
		return err
	}
	name, a, err := selectApp(apps, appName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(a.dockerfile()); err != nil {
		return fmt.Errorf("[apps.%s] dockerfile %s: %w; set dockerfile in %s to the app's Dockerfile", name, a.dockerfile(), err, appsFile)
	}

	dc, err := docker.New(stdin, stdout, stderr)
	if err != nil {
		return err
	}
	api, err := newClient(ctx)
	if err != nil {
		return err
	}
	return deployApp(ctx, stdout, dc, api, name, a, args)
}

var errNoAppsFile = errors.New(appsFile + ` not found in the current directory.  inngest deploy reads the apps to deploy from it.
Create it with one of:
  inngest deploy example-config > ` + appsFile + `   then edit the file
  inngest deploy                                 in a terminal, which asks a few questions and writes the file`)

// deployApp builds the app's image with the extra docker build args, uploads
// it, and waits until Inngest has built and published the app.
func deployApp(ctx context.Context, stdout io.Writer, dc *docker.Client, api *client, name string, a *app, dockerArgs []string) error {
	// the local tag has a prefix so that it does not replace a local image
	// with the app's name.
	ref := "inngest-deploy/" + name + ":" + deployTag

	fmt.Fprintf(stdout, "Building app %s from %s...\n\n", name, a.dockerfile())
	if err := dc.Build(ctx, platform, append([]string{"-t", ref, "-f", a.dockerfile(), "."}, dockerArgs...)); err != nil {
		return err
	}
	image, err := dc.Inspect(ctx, ref)
	if err != nil {
		return err
	}
	if image.Platform() != platform {
		return fmt.Errorf("%s built a %s image; apps must be %s", a.dockerfile(), image.Platform(), platform)
	}

	fmt.Fprintf(stdout, "Exporting image %s...\n\n", ref)
	archive, err := os.CreateTemp("", "inngest-deploy-*.tar")
	if err != nil {
		return fmt.Errorf("create image archive: %w", err)
	}
	_ = archive.Close()
	defer os.Remove(archive.Name())

	if err := dc.Save(ctx, ref, archive.Name()); err != nil {
		return err
	}
	size, digest, err := fileDigest(archive.Name())
	if err != nil {
		return fmt.Errorf("hash image archive: %w", err)
	}

	created, err := api.createDeploy(ctx, &apiv2.CreateDeployRequest{
		Name:            name,
		Tag:             deployTag,
		Architecture:    "amd64",
		UploadSizeBytes: size,
		UploadSha256:    digest,
		Config:          a.deployConfig(),
	})
	if err != nil {
		return fmt.Errorf("create deploy: %w", err)
	}
	deployID := created.GetData().GetId()

	fmt.Fprintf(stdout, "Uploading app %s (%d bytes) for deploy %s...\n\n", name, size, deployID)
	if err := uploadArchive(ctx, created.GetUploadUrl(), created.GetUploadHeaders(), archive.Name()); err != nil {
		return fmt.Errorf("upload app: %w", err)
	}

	d, err := api.commitDeploy(ctx, name, deployID)
	if err != nil {
		return fmt.Errorf("commit deploy %s: %w", deployID, err)
	}

	fmt.Fprintf(stdout, "Inngest is building app %s in deploy %s...\n\n", name, deployID)
	d, err = waitForDeploy(ctx, api, d)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Deploy %s is ready.  Artifact digest %s\n", deployID, d.GetArtifactDigest())
	if a.Serve != nil {
		// the deploy API does not report the sync, which runs after the
		// deploy is ready.
		fmt.Fprintf(stdout, "Inngest now starts the app and syncs its functions.  A failed sync sends an inngest/deploy.sync_failed event.\n")
	}
	return nil
}

// waitForDeploy polls the deploy until it is ready or failed.  stopping the
// CLI does not stop the deploy.
func waitForDeploy(ctx context.Context, api *client, d *apiv2.Deploy) (*apiv2.Deploy, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		switch d.GetStatus() {
		case buildReady:
			return d, nil
		case buildFailed:
			return nil, fmt.Errorf("deploy %s failed: %s", d.GetId(), d.GetError())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		next, err := api.getDeploy(ctx, d.GetId())
		if err != nil {
			return nil, fmt.Errorf("get deploy %s: %w", d.GetId(), err)
		}
		d = next
	}
}
