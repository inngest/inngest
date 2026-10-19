package imagecmd

import (
	"context"

	"github.com/inngest/inngest/cmd/image/internal/docker"
	"github.com/urfave/cli/v3"
)

func buildCommand() *cli.Command {
	return &cli.Command{
		Name:      "build",
		Usage:     "Build a linux/amd64 image with docker without uploading it",
		UsageText: "inngest image build [-t name[:tag]] <context> [-- docker build args]",
		Flags: []cli.Flag{
			refFlag(false),
		},
		Action: build,
	}
}

func build(ctx context.Context, cmd *cli.Command) error {
	stdin, stdout, stderr := stdio(cmd)

	dc, err := docker.New(stdin, stdout, stderr)
	if err != nil {
		return err
	}

	args, err := dockerArgs(cmd)
	if err != nil {
		return err
	}
	if ref := cmd.String("tag"); ref != "" {
		name, imageTag, err := parseRef(ref)
		if err != nil {
			return err
		}
		args = append([]string{"-t", name + ":" + imageTag}, args...)
	}

	return dc.Build(ctx, platform, args)
}
