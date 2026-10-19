package imagecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/inngest/inngest/cmd/image/internal/docker"
	cliauth "github.com/inngest/inngest/cmd/internal/auth"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
)

const (
	pollInterval = 2 * time.Second

	buildReady  = "ready"
	buildFailed = "failed"
)

func pushCommand() *cli.Command {
	return &cli.Command{
		Name:      "push",
		Usage:     "Upload a local linux/amd64 image to Inngest",
		UsageText: "inngest image push name[:tag]",
		Description: `name is 1 to 63 characters matching ^[a-z0-9][a-z0-9._-]{0,62}$, with no
registry or '/', and not "default".  tag defaults to latest and is up to
128 characters matching ^[a-z0-9][a-z0-9._-]{0,127}$.

Launch the image by name (name:latest), name:tag,
name@sha256:<artifact digest>, or a bare sha256:<artifact digest>.`,
		Action: push,
	}
}

func push(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() != 1 {
		return errors.New("expected one image reference; usage: inngest image push name[:tag]")
	}
	name, imageTag, err := parseRef(cmd.Args().First())
	if err != nil {
		return err
	}

	stdout, stderr := writers(cmd)
	dc, err := docker.New(stdout, stderr)
	if err != nil {
		return err
	}

	api, err := newClient(ctx)
	if err != nil {
		return err
	}

	return pushImage(ctx, stdout, dc, api, name, imageTag)
}

// pushImage uploads the local image name:imageTag and waits until Inngest
// has converted it.
func pushImage(ctx context.Context, stdout io.Writer, dc *docker.Client, api *client, name, imageTag string) error {
	ref := name + ":" + imageTag
	image, err := dc.Inspect(ctx, ref)
	if err != nil {
		return fmt.Errorf("%w; build it with `inngest image build -t %s <context>`", err, ref)
	}
	if image.Platform() != platform {
		return fmt.Errorf("%s is a %s image; images must be %s", ref, image.Platform(), platform)
	}

	fmt.Fprintf(stdout, "Exporting image %s...\n\n", ref)
	archive, err := os.CreateTemp("", "inngest-build-*.tar")
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

	created, err := api.createBuild(ctx, &apiv2.CreateBuildRequest{
		Name:            name,
		Tag:             imageTag,
		Architecture:    "amd64",
		UploadSizeBytes: size,
		UploadSha256:    digest,
	})
	if err != nil {
		return fmt.Errorf("create build: %w", err)
	}
	buildID := created.GetData().GetId()

	fmt.Fprintf(stdout, "Uploading image %s (%d bytes)...\n\n", ref, size)
	if err := uploadArchive(ctx, created.GetUploadUrl(), created.GetUploadHeaders(), archive.Name()); err != nil {
		return fmt.Errorf("upload image: %w", err)
	}

	b, err := api.commitBuild(ctx, name, buildID)
	if err != nil {
		return fmt.Errorf("commit build %s: %w", buildID, err)
	}

	fmt.Fprintf(stdout, "Converting image in build %s...\n\n", buildID)
	b, err = waitForBuild(ctx, api, b)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Image %s is ready.  Artifact digest %s\n", ref, b.GetArtifactDigest())
	return nil
}

// waitForBuild polls the build until it is ready or failed.  stopping the
// CLI does not stop the build.  `inngest api get-build` shows its status.
func waitForBuild(ctx context.Context, api *client, b *apiv2.Build) (*apiv2.Build, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		switch b.GetStatus() {
		case buildReady:
			return b, nil
		case buildFailed:
			return nil, fmt.Errorf("build %s failed: %s", b.GetId(), b.GetError())
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		next, err := api.getBuild(ctx, b.GetId())
		if err != nil {
			return nil, fmt.Errorf("get build %s: %w", b.GetId(), err)
		}
		b = next
	}
}

// newClient returns an API client that uses the stored login.  it checks the
// login once, and deploy calls it before docker runs, so no image is built
// that cannot be uploaded.
func newClient(ctx context.Context) (*client, error) {
	manager, err := cliauth.NewManager()
	if err != nil {
		return nil, err
	}
	issuer, err := cliauth.Issuer()
	if err != nil {
		return nil, err
	}
	resource := cliauth.Resource(issuer)
	token := func(ctx context.Context) (string, error) {
		token, _, err := manager.AccessToken(ctx, resource)
		if errors.Is(err, cliauth.ErrNotLoggedIn) {
			return "", errors.New("authentication required to push images; log in with `inngest login`")
		}
		return token, err
	}
	if _, err := token(ctx); err != nil {
		return nil, err
	}
	return &client{
		baseURL: resource,
		token:   token,
		env:     os.Getenv("INNGEST_ENV"),
		http:    &http.Client{},
	}, nil
}
