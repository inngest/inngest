package workflows

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/internal/cuedefs"
)

const (
	workflowComment = `// For documentation on workflow configuration, visit https://docs.inngest.com/docs/workflows`
)

var slugifyRegex = regexp.MustCompile(`[^a-z0-9\\-_]+`)

var (
	questions = []*survey.Question{
		{
			Name:     "name",
			Prompt:   &survey.Input{Message: "Workflow name:"},
			Validate: survey.Required,
		},
		{
			Name: "triggerType",
			Prompt: &survey.Select{
				Message: "Trigger type:",
				Help:    "Is this workflow triggered via events or on a cron schedule?",
				Options: []string{"event", "cron"},
			},
			Validate: survey.Required,
		},
	}
)

type Config struct {
	Name        string
	ID          string
	TriggerType string
	EventName   string
	Cron        string
}

func (c *Config) Survey() error {
	fmt.Println("")
	if err := survey.Ask(questions, c); err != nil {
		return err
	}
	if err := c.TriggerQuestion(); err != nil {
		return err
	}
	fmt.Println("")
	return nil
}

func (c *Config) TriggerQuestion() error {
	if c.TriggerType == "event" {
		return survey.AskOne(&survey.Input{
			Message: "Triggering event name:",
		}, &c.EventName)
	}
	return survey.AskOne(&survey.Input{
		Help:    "eg. '0 * * * *'",
		Message: "Cron schedule:",
	}, &c.Cron)
}

func (c *Config) Triggers() []inngest.Trigger {
	if c.TriggerType == "event" {
		return []inngest.Trigger{
			{EventTrigger: &inngest.EventTrigger{Event: c.EventName}},
		}
	}
	return []inngest.Trigger{
		{CronTrigger: &inngest.CronTrigger{Cron: c.Cron}},
	}
}

func (c *Config) Configuration() (string, error) {
	output, err := cuedefs.FormatWorkflow(inngest.Workflow{
		Name:     c.Name,
		ID:       strings.ToLower(slugifyRegex.ReplaceAllString(c.Name, "-")),
		Triggers: c.Triggers(),
		Steps:    []inngest.Step{},
		Edges:    []inngest.Edge{},
	})
	if err != nil {
		return "", err
	}

	data := fmt.Sprintf("%s\n%s", workflowComment, output)
	return data, nil
}
