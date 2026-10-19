package imagecmd

import (
	"context"
	"fmt"

	"github.com/inngest/inngest/cmd/image/internal/docker"
	"github.com/urfave/cli/v3"
)

func deployCommand() *cli.Command {
	return &cli.Command{
		Name:      "deploy",
		Usage:     "Build a linux/amd64 image with docker and push it to Inngest",
		UsageText: "inngest image deploy -t name[:tag] <context> [-- docker build args]",
		Flags: []cli.Flag{
			refFlag(true),
		},
		Action: deploy,
	}
}

func deploy(ctx context.Context, cmd *cli.Command) error {
	stdout, stderr := writers(cmd)

	dc, err := docker.New(stdout, stderr)
	if err != nil {
		return err
	}

	args, err := dockerArgs(cmd)
	if err != nil {
		return err
	}
	name, imageTag, err := parseRef(cmd.String("tag"))
	if err != nil {
		return err
	}

	api, err := newClient(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Running docker build...\n\n")
	if err := dc.Build(ctx, platform, append([]string{"-t", name + ":" + imageTag}, args...)); err != nil {
		return err
	}

	return pushImage(ctx, stdout, dc, api, name, imageTag)
}
