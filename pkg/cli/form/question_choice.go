package form

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngest/pkg/cli"
	"golang.org/x/term"
)

type ItemGetter interface {
	Items() []list.Item
}

type ChoiceQuestionOpts[T any] struct {
	QuestionOpts[T]

	Prompt string

	// OffsetY represents the list offset from the Y position.
	PaddingY int

	// ItemGetter returns all available items to the list
	// items on each render of the list.
	//
	// This is required.
	ItemGetter ItemGetter

	// TeaState allows overriding the list.Model used to render the list, if desired.
	//
	// This is an optional field.  It is a pointer, and can be mutated outside
	// of the Tea framework.
	TeaState *list.Model

	// Render optionally handles rendering the text list manually.
	Render func(model T, textinput list.Model) string
}

func NewChoiceQuestion[T any](id string, opts ChoiceQuestionOpts[T]) Question[T] {
	var state list.Model

	if opts.TeaState != nil {
		state = *opts.TeaState
	} else {
		width, height, _ := term.GetSize(int(os.Stdout.Fd()))

		d := list.NewDefaultDelegate()
		d.ShowDescription = false

		state = list.New(opts.ItemGetter.Items(), d, width, height-opts.PaddingY)
		state.SetShowFilter(false)
		state.SetShowHelp(false)
		state.SetShowStatusBar(false)
		state.SetShowTitle(false)

	}

	return &ChoiceQuestion[T]{
		id:    id,
		opts:  opts,
		state: &state,
	}
}

type ChoiceQuestion[T any] struct {
	id   string
	opts ChoiceQuestionOpts[T]

	state *list.Model
}

func (i ChoiceQuestion[T]) ID() string {
	return i.id
}

func (i ChoiceQuestion[T]) Answer(model T) (string, error) {
	return i.opts.Answer(model)
}

// Render renders the question.
func (i ChoiceQuestion[T]) Render(model T) string {
	if i.opts.Render != nil {
		// Overriding render, so just use that function.
		return i.opts.Render(model, *i.state)
	}

	a, err := i.Answer(model)
	if err == nil {
		return fmt.Sprintf("%s: %s\n", i.opts.Prompt, cli.BoldStyle.Render(a))
	}

	b := &strings.Builder{}
	b.WriteString(cli.BoldStyle.Render(i.opts.Prompt+":") + "\n\n")
	b.WriteString(i.state.View())
	return b.String()
}

// UpdateTea updates the model T via Bubbletea UI
func (i ChoiceQuestion[T]) UpdateTea(model T, msg tea.Msg) (Question[T], tea.Cmd) {
	state, cmd := i.state.Update(msg)
	i.state = &state

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		i.state.SetWidth(msg.Width)
		i.state.SetHeight(msg.Height - i.opts.PaddingY)
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		_ = i.UpdateAnswer(model, i.state.SelectedItem())
	}

	return i, cmd
}

func (i ChoiceQuestion[T]) UpdateAnswer(model T, value interface{}) error {
	return i.opts.UpdateAnswer(model, value)
}

// Next returns the next question in the chain
func (i ChoiceQuestion[T]) Next(model T) string {
	if i.opts.Next == nil {
		return ""
	}
	return i.opts.Next(model)
}
