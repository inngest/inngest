package actions

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/internal/cuedefs"
)

const (
	actionComment = `// For documentation on action configuration, visit https://docs.inngest.com/docs/actions`
)

var (
	spacesRegex = regexp.MustCompile(`\s`)
)

type Config struct {
	Name string
	DSN  string

	// No runtimes are configurable util we expose webassembly and/or creators
	// for node, python, golang.
	DockerImage string
}

func (c *Config) Survey(dsnPrefix string) error {
	fmt.Println("")

	err := survey.AskOne(
		&survey.Input{
			Message: "Action name:",
		},
		&c.Name,
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return err
	}

	err = survey.AskOne(
		&survey.Input{
			Message: "Unique action ID:",
			Default: spacesRegex.ReplaceAllString(strings.ToLower(c.Name), "-"),
		},
		&c.DSN,
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return err
	}

	c.DSN = fmt.Sprintf("%s/%s", dsnPrefix, c.DSN)

	err = survey.AskOne(
		&survey.Input{
			Message: "Docker image name:",
			Help:    "The docker image to use for this action.  This will be pushed to Inngest.",
		},
		&c.DockerImage,
		survey.WithValidator(survey.Required),
	)
	if err != nil {
		return err
	}

	fmt.Println("")
	return nil
}

func (c *Config) Configuration() (string, error) {
	output, err := cuedefs.FormatAction(inngest.ActionVersion{
		DSN:  c.DSN,
		Name: c.Name,
		Version: &inngest.VersionInfo{
			Major: 1,
			Minor: 1,
		},
		WorkflowMetadata: inngest.MetadataMap{},
		Response:         map[string]inngest.Response{},
		Runtime: inngest.RuntimeWrapper{
			Runtime: inngest.RuntimeDocker{
				Image: c.DockerImage,
			},
		},
	})
	if err != nil {
		return "", err
	}

	data := fmt.Sprintf("%s\n%s", actionComment, output)
	return data, nil
}
