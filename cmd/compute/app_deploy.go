package computecmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/inngest/inngest/cmd/compute/internal/docker"
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
		Usage:     "Build apps from inngest.toml and deploy them to Inngest compute",
		UsageText: "inngest deploy [app...] [-- docker build args]",
		Description: `Reads inngest.toml in the current directory, builds the app's Dockerfile
for linux/amd64 with docker, uploads the image, and waits for Inngest to
deploy it.  The current directory is the docker build context.  Without
inngest.toml, it asks a few questions in a terminal and writes the file.

app is the name of an [apps.<name>] section.  It can be omitted when the
file has one app.  Several apps deploy one at a time, in the order given;
when one fails, the rest still deploy, and the command exits with an error.
Run ` + "`inngest deploy example-config`" + ` for a commented example.

Arguments after -- go to docker build for every app, as in
` + "`inngest image build`" + `, such as ` + "`inngest deploy web worker -- --build-arg VERSION=1`" + `.

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
	// the arguments before the first docker build flag are app names.  the
	// rest go to docker build, as in `inngest image build`.  urfave removes
	// the -- before them.
	args := cmd.Args().Slice()
	n := slices.IndexFunc(args, func(arg string) bool { return strings.HasPrefix(arg, "-") })
	if n < 0 {
		n = len(args)
	}
	appNames, args := args[:n], args[n:]

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
	// check every app before building any, so that a mistake in the last
	// app does not stop the deploy after the first apps deployed.
	names, err := selectApps(apps, appNames)
	if err != nil {
		return err
	}
	for _, name := range names {
		if _, err := os.Stat(apps[name].dockerfile()); err != nil {
			return fmt.Errorf("[apps.%s] dockerfile %s: %w; set dockerfile in %s to the app's Dockerfile", name, apps[name].dockerfile(), err, appsFile)
		}
	}

	dc, err := docker.New(stdin, stdout, stderr)
	if err != nil {
		return err
	}
	api, err := newClient(ctx)
	if err != nil {
		return err
	}
	if len(names) == 1 {
		_, err := deployApp(ctx, stdout, dc, api, names[0], apps[names[0]], args)
		return err
	}

	// one failed app does not stop the rest.
	ids := make([]string, len(names))
	errs := make([]error, len(names))
	for i, name := range names {
		ids[i], errs[i] = deployApp(ctx, stdout, dc, api, name, apps[name], args)
	}
	var failed []string
	fmt.Fprintf(stdout, "\nSummary:\n")
	for i, name := range names {
		if errs[i] != nil {
			failed = append(failed, name)
			fmt.Fprintf(stdout, "  %s: failed: %v\n", name, errs[i])
			continue
		}
		fmt.Fprintf(stdout, "  %s: deploy %s is ready\n", name, ids[i])
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d apps failed to deploy: %s", len(failed), len(names), strings.Join(failed, ", "))
	}
	return nil
}

var errNoAppsFile = errors.New(appsFile + ` not found in the current directory.  inngest deploy reads the apps to deploy from it.
Create it with one of:
  inngest deploy example-config > ` + appsFile + `   then edit the file
  inngest deploy                                 in a terminal, which asks a few questions and writes the file`)

// deployApp builds the app's image with the extra docker build args, uploads
// it, and waits until Inngest has built and published the app.  it returns
// the deploy ID.
func deployApp(ctx context.Context, stdout io.Writer, dc *docker.Client, api *client, name string, a *app, dockerArgs []string) (string, error) {
	// the local tag is new on every run, so that concurrent deploys of an app
	// do not upload each other's image.  the app name is in the tag, where
	// every app name is valid.
	suffix := make([]byte, 8)
	_, _ = rand.Read(suffix)
	ref := "inngest-deploy:" + name + "-" + hex.EncodeToString(suffix)

	fmt.Fprintf(stdout, "Building app %s from %s...\n\n", name, a.dockerfile())
	if err := dc.Build(ctx, platform, append([]string{"-t", ref, "-f", a.dockerfile(), "."}, dockerArgs...)); err != nil {
		return "", err
	}
	defer func() { _ = dc.Remove(ctx, ref) }()
	image, err := dc.Inspect(ctx, ref)
	if err != nil {
		return "", err
	}
	if image.Platform() != platform {
		return "", fmt.Errorf("%s built a %s image; apps must be %s", a.dockerfile(), image.Platform(), platform)
	}

	fmt.Fprintf(stdout, "Exporting image %s...\n\n", ref)
	archive, err := os.CreateTemp("", "inngest-deploy-*.tar")
	if err != nil {
		return "", fmt.Errorf("create image archive: %w", err)
	}
	_ = archive.Close()
	defer os.Remove(archive.Name())

	if err := dc.Save(ctx, ref, archive.Name()); err != nil {
		return "", err
	}
	size, digest, err := fileDigest(archive.Name())
	if err != nil {
		return "", fmt.Errorf("hash image archive: %w", err)
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
		return "", err
	}
	deployID := created.GetData().GetId()

	fmt.Fprintf(stdout, "Uploading app %s (%d bytes) for deploy %s...\n\n", name, size, deployID)
	if err := uploadArchive(ctx, created.GetUploadUrl(), created.GetUploadHeaders(), archive.Name()); err != nil {
		return "", fmt.Errorf("upload app: %w", err)
	}

	d, err := api.commitDeploy(ctx, name, deployID)
	if err != nil {
		return "", fmt.Errorf("commit deploy %s: %w", deployID, err)
	}

	fmt.Fprintf(stdout, "Inngest is building app %s in deploy %s...\n", name, deployID)
	// the build and the sync share one deployWait.
	start := time.Now()
	d, err = waitForDeploy(ctx, stdout, api, d, start, deployDone)
	if err != nil {
		return "", err
	}
	if a.Serve == nil {
		fmt.Fprintf(stdout, "\nDeploy %s of app %s is ready.  Inngest is starting its workers.\n", deployID, name)
		return deployID, nil
	}
	if d.GetSync() == nil {
		// an API without sync reports does not say how the sync ends.
		fmt.Fprintf(stdout, "\nDeploy %s is built.  Inngest is starting app %s to sync its functions.  A failed sync sends an inngest/deploy.sync_failed event.\n", deployID, name)
		return deployID, nil
	}

	fmt.Fprintf(stdout, "\nBuilt.  Inngest is starting app %s to sync its functions...\n", name)
	d, err = waitForDeploy(ctx, stdout, api, d, start, syncDone)
	if err != nil {
		return "", err
	}
	result := d.GetSync()
	if result.GetStatus() != syncSynced {
		// the error is one line, so the summary of several apps stays readable.
		err := fmt.Errorf("app %s did not sync: %s", name, result.GetError())
		if result.GetUrl() != "" {
			err = fmt.Errorf("%w; see %s", err, result.GetUrl())
		}
		return "", err
	}
	fmt.Fprintf(stdout, "\nApp %s is synced and ready to run functions.\n", name)
	if result.GetUrl() != "" {
		fmt.Fprintf(stdout, "See the sync at %s\n", result.GetUrl())
	}
	return deployID, nil
}

// the sync statuses of a deploy.
const (
	syncSyncing = "syncing"
	syncSynced  = "synced"
)

const (
	// deployWait limits the wait for one deploy.  Inngest stops a build after
	// 35 minutes and a sync after a few more.
	deployWait = 45 * time.Minute
	// progressInterval is how often the wait says that it is still waiting.
	progressInterval = 30 * time.Second
)

// deployDone reports whether the build of a deploy has ended, and the error of
// a failed build.
func deployDone(d *apiv2.Deploy) (bool, error) {
	switch d.GetStatus() {
	case buildReady:
		return true, nil
	case buildFailed:
		return true, fmt.Errorf("deploy %s failed: %s", d.GetId(), d.GetError())
	}
	return false, nil
}

// syncDone reports whether the sync of a ready deploy has ended.  a status
// other than syncing ends the wait, so a status that this CLI does not know
// fails at once.  the caller reads a failed sync's error.
func syncDone(d *apiv2.Deploy) (bool, error) {
	return d.GetSync().GetStatus() != syncSyncing, nil
}

// waitForDeploy polls the deploy until done says it has ended, and says how
// long it has waited since start every progressInterval.  stopping the CLI
// does not stop the deploy.
func waitForDeploy(ctx context.Context, stdout io.Writer, api *client, d *apiv2.Deploy, start time.Time, done func(*apiv2.Deploy) (bool, error)) (*apiv2.Deploy, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	progress := time.NewTicker(progressInterval)
	defer progress.Stop()
	for {
		if ok, err := done(d); ok {
			return d, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-progress.C:
			waited := time.Since(start).Round(time.Second)
			if waited >= deployWait {
				return nil, fmt.Errorf("stopped waiting for deploy %s after %s; Inngest may still finish it", d.GetId(), waited)
			}
			fmt.Fprintf(stdout, "  still waiting (%s)\n", waited)
			continue
		case <-ticker.C:
		}
		next, err := api.getDeploy(ctx, d.GetId())
		if err != nil {
			return nil, fmt.Errorf("get deploy %s: %w", d.GetId(), err)
		}
		d = next
	}
}
