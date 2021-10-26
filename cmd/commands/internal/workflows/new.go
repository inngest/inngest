package workflows

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/inngest/inngestctl/inngest"
)

const (
	workflowComment = `// For documentation on workflow configuration, visit https://docs.inngest.com/docs/workflows`
)

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
		{ScheduleTrigger: &inngest.ScheduleTrigger{Cron: c.Cron}},
	}
}

func (c *Config) Configuration() (string, error) {
	output, err := inngest.FormatWorkflow(inngest.Workflow{
		Name:     c.Name,
		Triggers: c.Triggers(),
		Actions:  []inngest.Action{},
		Edges:    []inngest.Edge{},
	})
	if err != nil {
		return "", err
	}

	data := fmt.Sprintf("%s\n%s", workflowComment, output)
	return data, nil
}
