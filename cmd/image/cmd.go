package imagecmd

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v3"
)

// platform is the only platform that compute runs images on.
const platform = "linux/amd64"

// the API rejects names and tags that do not match these patterns.
var (
	namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,62}$`)
	tagPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "image",
		Usage: "Build and deploy images for Inngest compute",
		Commands: []*cli.Command{
			buildCommand(),
			pushCommand(),
			deployCommand(),
		},
	}
}

func stdio(cmd *cli.Command) (io.Reader, io.Writer, io.Writer) {
	var stdin io.Reader = os.Stdin
	var stdout, stderr io.Writer = os.Stdout, os.Stderr
	if r := cmd.Root().Reader; r != nil {
		stdin = r
	}
	if w := cmd.Root().Writer; w != nil {
		stdout = w
	}
	if w := cmd.Root().ErrWriter; w != nil {
		stderr = w
	}
	return stdin, stdout, stderr
}

func validateName(name string) error {
	if !namePattern.MatchString(name) || name == "default" {
		return fmt.Errorf("invalid image name %q; use 1 to 63 lowercase letters, digits, '.', '_' or '-', with no registry or '/'", name)
	}
	return nil
}

func validateTag(tag string) error {
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid image tag %q; use up to 128 lowercase letters, digits, '.', '_' or '-'", tag)
	}
	return nil
}

// parseRef splits name[:tag] and checks both against the API's patterns.
func parseRef(ref string) (string, string, error) {
	name, tag, found := strings.Cut(ref, ":")
	if !found {
		tag = "latest"
	}
	if err := validateName(name); err != nil {
		return "", "", err
	}
	if err := validateTag(tag); err != nil {
		return "", "", err
	}
	return name, tag, nil
}

// refFlag is -t name[:tag], the same form as `docker build -t`.
func refFlag(required bool) *cli.StringFlag {
	return &cli.StringFlag{
		Name:     "tag",
		Aliases:  []string{"t"},
		Required: required,
		Usage: "Image name and optional tag as name[:tag].  " +
			"name is 1 to 63 characters matching ^[a-z0-9][a-z0-9._-]{0,62}$, with no registry or '/', and not \"default\".  " +
			"tag defaults to latest and is up to 128 characters matching ^[a-z0-9][a-z0-9._-]{0,127}$.  " +
			"Launch the image by name (name:latest), name:tag, name@sha256:<artifact digest>, or a bare sha256:<artifact digest>",
		Validator: func(ref string) error {
			_, _, err := parseRef(ref)
			return err
		},
	}
}

// dockerArgs returns the build context and the docker build args that
// follow "--".
func dockerArgs(cmd *cli.Command) ([]string, error) {
	args := cmd.Args().Slice()
	if len(args) == 0 {
		return nil, fmt.Errorf("missing build context; expected at least a path to a directory containing a Dockerfile")
	}
	return args, nil
}
