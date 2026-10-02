// Package image provides local Docker export and isolated remote image builds.
package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	apiv2 "github.com/inngest/inngest/proto/gen/api/v2"
	"github.com/urfave/cli/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var namePattern = regexp.MustCompile(`^(inngest/)?[a-z0-9][a-z0-9._-]{0,62}$`)
var tagPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
var containerIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func Command() *cli.Command {
	flags := func() []cli.Flag {
		return []cli.Flag{
			&cli.StringFlag{Name: "tag", Aliases: []string{"t"}, Required: true, Usage: "Workspace image name and tag, e.g. app:latest"},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Value: "Dockerfile"},
			&cli.StringFlag{Name: "target", Usage: "Dockerfile build stage"},
			&cli.StringSliceFlag{Name: "build-arg", Usage: "Build argument NAME=value (repeatable)"},
			&cli.StringFlag{Name: "upload-id", Usage: "Reuse a UUID when retrying the exact same upload"},
			&cli.Int64Flag{Name: "expected-generation", Usage: "Override the current tag generation for compare-and-swap publication"},
			&cli.BoolFlag{Name: "immutable", Usage: "Prevent this tag from being moved after publication"},
			&cli.BoolFlag{Name: "no-wait", Usage: "Return the build ID after queueing"},
			&cli.DurationFlag{Name: "timeout", Value: 45 * time.Minute, Usage: "How long to wait for publication"},
		}
	}
	remoteFlags := append(flags(), &cli.BoolFlag{Name: "cache", Usage: "Reuse prior build results for digest-pinned bases"}, &cli.StringFlag{Name: "cache-epoch", Usage: "Change to invalidate prior build results"}, &cli.StringSliceFlag{Name: "secret", Usage: "Workspace secret name for a BuildKit secret mount"}, &cli.StringSliceFlag{Name: "registry-auth", Usage: "Registry host=workspace-secret; secret contains JSON username/password"})
	return &cli.Command{Name: "image", Usage: "Build and manage workspace images", Flags: []cli.Flag{
		&cli.StringFlag{Name: "api-url", Sources: cli.EnvVars("INNGEST_IMAGE_API_URL"), Usage: "Cloud API origin (defaults to the login issuer)"},
		&cli.StringFlag{Name: "api-key", Sources: cli.EnvVars("INNGEST_API_KEY")},
		&cli.StringFlag{Name: "env", Sources: cli.EnvVars("INNGEST_ENV")},
	}, Commands: []*cli.Command{
		{Name: "push", Usage: "Build locally with Docker, export and upload for conversion", ArgsUsage: "[context directory]", Flags: flags(), Action: func(ctx context.Context, cmd *cli.Command) error { return push(ctx, cmd, false) }},
		{Name: "build", Usage: "Upload a build context and build in an isolated remote VM", ArgsUsage: "[context directory]", Flags: remoteFlags, Action: func(ctx context.Context, cmd *cli.Command) error { return push(ctx, cmd, true) }},
		{Name: "list", Usage: "List accessible images", Flags: []cli.Flag{&cli.StringFlag{Name: "cursor"}}, Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := newClient(ctx, cmd)
			if err != nil {
				return err
			}
			result := &apiv2.ListImagesResponse{}
			if err = c.request(ctx, http.MethodGet, "/images?limit=100&cursor="+url.QueryEscape(cmd.String("cursor")), nil, result); err != nil {
				return err
			}
			return printJSON(cmd, result)
		}},
		{Name: "inspect", Usage: "Inspect image tags and immutable artifacts", ArgsUsage: "name or inngest/name", Flags: []cli.Flag{&cli.StringFlag{Name: "artifact-cursor"}}, Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if !namePattern.MatchString(name) {
				return errors.New("expected a workspace name or inngest/name")
			}
			c, err := newClient(ctx, cmd)
			if err != nil {
				return err
			}
			result := &apiv2.GetImageResponse{}
			if err = c.request(ctx, http.MethodGet, "/images/"+name+"?artifactCursor="+url.QueryEscape(cmd.String("artifact-cursor")), nil, result); err != nil {
				return err
			}
			return printJSON(cmd, result)
		}},
		{Name: "usage", Usage: "Show image storage and build time", Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := newClient(ctx, cmd)
			if err != nil {
				return err
			}
			result := &apiv2.GetImageUsageResponse{}
			if err = c.request(ctx, http.MethodGet, "/image-usage", nil, result); err != nil {
				return err
			}
			return printJSON(cmd, result)
		}},
		{Name: "build-status", ArgsUsage: "build-id", Action: func(ctx context.Context, cmd *cli.Command) error { return buildCommand(ctx, cmd, false) }},
		{Name: "cancel", ArgsUsage: "build-id", Action: func(ctx context.Context, cmd *cli.Command) error { return buildCommand(ctx, cmd, true) }},
	}}
}

func printJSON(cmd *cli.Command, message proto.Message) error {
	data, err := (protojson.MarshalOptions{Indent: "  "}).Marshal(message)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Writer, string(data))
	return err
}

func buildCommand(ctx context.Context, cmd *cli.Command, cancel bool) error {
	id, err := uuid.Parse(cmd.Args().First())
	if err != nil {
		return errors.New("expected a build UUID")
	}
	c, err := newClient(ctx, cmd)
	if err != nil {
		return err
	}
	if cancel {
		result := &apiv2.CancelImageBuildResponse{}
		if err = c.request(ctx, http.MethodPost, "/image-builds/"+id.String()+"/cancel", &apiv2.CancelImageBuildRequest{}, result); err != nil {
			return err
		}
		return printJSON(cmd, result)
	}
	result := &apiv2.GetImageBuildResponse{}
	if err = c.request(ctx, http.MethodGet, "/image-builds/"+id.String(), nil, result); err != nil {
		return err
	}
	return printJSON(cmd, result)
}

func parseTag(value string) (string, string, error) {
	name, tag, ok := strings.Cut(value, ":")
	if !ok {
		tag = "latest"
	}
	if len(name) > 255 || !namePattern.MatchString(name) || strings.Contains(name, "/") || !tagPattern.MatchString(tag) {
		return "", "", errors.New("tag must be a workspace image such as app:latest; inngest/* is reserved")
	}
	return name, tag, nil
}

func keyValues(values []string) (map[string]string, error) {
	result := map[string]string{}
	for _, value := range values {
		key, value, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, errors.New("expected NAME=value")
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate key %s", key)
		}
		result[key] = value
	}
	return result, nil
}

func push(ctx context.Context, cmd *cli.Command, remote bool) error {
	if cmd.Args().Len() > 1 {
		return errors.New("expected at most one context directory")
	}
	name, tag, err := parseTag(cmd.String("tag"))
	if err != nil {
		return err
	}
	directory := cmd.Args().First()
	if directory == "" {
		directory = "."
	}
	c, err := newClient(ctx, cmd)
	if err != nil {
		return err
	}
	id := uuid.New()
	if raw := cmd.String("upload-id"); raw != "" {
		id, err = uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			return errors.New("upload-id must be a UUID")
		}
	}
	// Persist this ID in CI output even when later upload/build work fails.
	fmt.Fprintln(cmd.ErrWriter, "Image upload:", id)
	request := &apiv2.PrepareImageUploadRequest{Id: id.String(), Name: name, Tag: tag, ImmutableTag: cmd.Bool("immutable"), Platform: "linux/amd64", Format: "rootfs.tar.gz", SourceType: "docker_export"}
	if cmd.IsSet("expected-generation") {
		request.ExpectedGeneration = cmd.Int64("expected-generation")
		if request.ExpectedGeneration < 0 {
			return errors.New("expected-generation cannot be negative")
		}
	} else {
		current := &apiv2.GetImageResponse{}
		err := c.request(ctx, http.MethodGet, "/images/"+name, nil, current)
		var status apiError
		if err != nil && (!errors.As(err, &status) || status.status != 404) {
			return err
		}
		for _, currentTag := range current.GetData().GetTags() {
			if currentTag.Name == tag {
				request.ExpectedGeneration = currentTag.Generation
				break
			}
		}
	}
	temp, err := os.MkdirTemp("", "inngest-image-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	file, err := os.CreateTemp(temp, "context-*.tar.gz")
	if err != nil {
		return err
	}
	defer file.Close()
	request.Sha256, request.SizeBytes, err = compressedArchive(file, func(out io.Writer) error {
		if remote {
			dockerfile, err := dockerfileText(directory, cmd.String("file"))
			if err != nil {
				return err
			}
			args, err := keyValues(cmd.StringSlice("build-arg"))
			if err != nil {
				return err
			}
			auth, err := keyValues(cmd.StringSlice("registry-auth"))
			if err != nil {
				return err
			}
			request.Format = "context.tar.gz"
			request.SourceType = "dockerfile"
			request.Recipe = &apiv2.ImageBuildRecipe{Dockerfile: dockerfile, Target: cmd.String("target"), BuildArgs: args, RegistryAuth: auth, Secrets: cmd.StringSlice("secret"), UseCache: cmd.Bool("cache"), CacheEpoch: cmd.String("cache-epoch")}
			return contextArchive(ctx, directory, cmd.String("file"), out)
		}
		config, err := exportDocker(ctx, runDocker, directory, cmd.String("file"), cmd.String("target"), cmd.StringSlice("build-arg"), temp, &boundedWriter{writer: out, remaining: maxExpandedBytes}, cmd.ErrWriter)
		request.Config = config
		return err
	})
	if err != nil {
		return err
	}
	grant := &apiv2.PrepareImageUploadResponse{}
	if err = c.request(ctx, http.MethodPost, "/image-uploads", request, grant); err != nil {
		return err
	}
	if grant.GetData().GetUploadId() != id.String() {
		return errors.New("image API returned a different upload ID")
	}
	if err = c.uploadGrant(ctx, grant.Data, file); err != nil {
		return err
	}
	complete := &apiv2.CompleteImageUploadResponse{}
	if err = c.request(ctx, http.MethodPost, "/image-uploads/"+id.String()+"/complete", &apiv2.CompleteImageUploadRequest{}, complete); err != nil {
		return err
	}
	if complete.GetBuildId() != id.String() {
		return errors.New("image API returned a different build ID")
	}
	if cmd.Bool("no-wait") {
		return printJSON(cmd, complete)
	}
	if cmd.Duration("timeout") <= 0 {
		return errors.New("timeout must be positive")
	}
	wait, cancel := context.WithTimeout(ctx, cmd.Duration("timeout"))
	defer cancel()
	return waitBuild(wait, c, id.String(), cmd.Writer, cmd.ErrWriter)
}

func waitBuild(ctx context.Context, c *client, id string, output, logs io.Writer) error {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	logOffset := 0
	for {
		result := &apiv2.GetImageBuildResponse{}
		if err := c.request(ctx, http.MethodGet, "/image-builds/"+id, nil, result); err != nil {
			return err
		}
		b := result.GetData()
		if b == nil {
			return errors.New("image API returned no build")
		}
		if len(b.Logs) > logOffset {
			fmt.Fprint(logs, b.Logs[logOffset:])
			logOffset = len(b.Logs)
		}
		switch b.State {
		case "succeeded":
			if b.ImageRef == "" {
				return errors.New("published build has no immutable image reference")
			}
			fmt.Fprintln(output, b.ImageRef)
			if !b.TagUpdated {
				fmt.Fprintln(logs, "Image published, but the tag changed during the build. Use the immutable reference above.")
			}
			return nil
		case "failed", "cancelled":
			return fmt.Errorf("image build %s: %s", b.State, b.ErrorMessage)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("stopped waiting; inspect build %s with inngest image build-status: %w", id, ctx.Err())
		case <-tick.C:
		}
	}
}
