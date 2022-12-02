package commands

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gosimple/slug"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/cli/form"
	"github.com/inngest/inngest/pkg/cli/form/formutil"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/scaffold"
	"github.com/spf13/cobra"
)

func NewCmdSteps() *cobra.Command {
	root := &cobra.Command{
		Use:    "step",
		Short:  "",
		Hidden: true,
	}

	add := &cobra.Command{
		Use:   "add",
		Short: "Adds a step to the current function",
		Run:   addStep,
	}

	root.AddCommand(add)
	return root
}

type stepModel struct {
	Name     string
	Language string
}

func addStep(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	model := &stepModel{}

	fn, err := function.Load(ctx, ".")
	if err != nil {
		fmt.Println(cli.RenderError(err.Error()))
		os.Exit(1)
	}

	go func() {
		// Fetch scaffolds.
		_ = scaffold.UpdateCache(ctx)
	}()

	stepForm, err := form.NewForm(model, form.FormOpts[*stepModel]{
		RootID: "name",
		Intro: func(model *stepModel) string {
			b := &strings.Builder{}
			b.WriteString("\n")
			b.WriteString(cli.BoldStyle.Render("Adding a step to your function"))
			b.WriteString("\n")
			b.WriteString(cli.TextStyle.Copy().Foreground(cli.Feint).Render("Answer these questions to add your step."))
			b.WriteString("\n")
			return b.String()
		},
		Questions: []form.Question[*stepModel]{
			form.NewInputQuestion(
				"name",
				form.InputQuestionOpts[*stepModel]{
					Prompt:      "Enter the step name",
					Placeholder: "eg. Send to slack",
					QuestionOpts: form.QuestionOpts[*stepModel]{
						Answer: func(model *stepModel) (string, error) {
							if model.Name == "" {
								return "", form.ErrUnanswered
							}
							return model.Name, nil
						},
						UpdateAnswer: func(model *stepModel, answer interface{}) error {
							switch v := answer.(type) {
							case string:
								if v != "" {
									model.Name = v
									return nil
								}
							}
							return form.ErrUnanswered
						},
						Next: func(model *stepModel) string {
							return "language"
						},
					},
				},
			),
			form.NewChoiceQuestion(
				"language",
				form.ChoiceQuestionOpts[*stepModel]{
					Prompt:     "Language",
					ItemGetter: &formutil.ScaffoldGetter{},
					PaddingY:   8,
					QuestionOpts: form.QuestionOpts[*stepModel]{
						Answer: func(model *stepModel) (string, error) {
							if model.Language == "" {
								return "", form.ErrUnanswered
							}
							return model.Language, nil
						},
						UpdateAnswer: func(model *stepModel, answer interface{}) error {
							fmt.Printf("%#v", answer)
							switch v := answer.(type) {
							case formutil.BasicListItem:
								model.Language = v.Name
								return nil
							}
							return form.ErrUnanswered
						},
					},
				},
			),
		},
	})
	if err != nil {
		panic(err)
	}

	// Render interactive UI to ask questions.
	if err := stepForm.Ask(); err != nil {
		fmt.Println(cli.RenderError(err.Error()))
		os.Exit(1)
	}

	id := slug.Make(model.Name)
	if _, ok := fn.Steps[id]; ok {
		fmt.Println(cli.RenderError(fmt.Sprintf("Step '%s' already exists in this function", id)))
		os.Exit(1)
	}

	step := function.Step{
		ID:      id,
		Name:    model.Name,
		Path:    function.FilePrefix + "./" + path.Join("steps", id),
		Runtime: function.DefaultRuntime(),
	}
	fn.Steps[id] = step
	if model.Language != "Another language" {
		mapping, err := scaffold.Parse(ctx)
		if err != nil {
			fmt.Println(cli.RenderError(err.Error()))
			os.Exit(1)
		}
		item := mapping.Languages[model.Language]
		if len(item) == 0 {
			// TODO
			os.Exit(1)
			return
		}
		if err := item[0].RenderStep(ctx, *fn, step); err != nil {
			fmt.Println(cli.RenderError(fmt.Sprintf("Step '%s' already exists in this function", id)))
			os.Exit(1)
		}
	}

	if err := fn.WriteToDisk(ctx); err != nil {
		fmt.Println(cli.RenderError(err.Error()))
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println(cli.BoldStyle.Copy().Foreground(cli.Green).Render("Step added"))
}
