package form

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/inngest/inngest/pkg/cli"
)

type InputQuestionOpts[T any] struct {
	QuestionOpts[T]

	Prompt      string
	Placeholder string

	// Render optionally handles rendering the text input manually.
	Render func(model T, textinput textinput.Model) string

	// TeaState allows overriding the textinput.Model used to render the
	// text input.
	//
	// This is an optional field.
	TeaState *textinput.Model
}

func NewInputQuestion[T any](id string, opts InputQuestionOpts[T]) Question[T] {
	var state textinput.Model

	if opts.TeaState != nil {
		state = *opts.TeaState
	} else {
		state = textinput.New()
		state.Placeholder = opts.Placeholder
		// Focus triggers updating this placeholder when rendering.
		state.Focus()
		state.Prompt = "→  "
	}

	return &inputQuestion[T]{
		id:    id,
		opts:  opts,
		state: &state,
	}
}

type inputQuestion[T any] struct {
	id   string
	opts InputQuestionOpts[T]

	state *textinput.Model
}

func (i inputQuestion[T]) ID() string {
	return i.id
}

func (i inputQuestion[T]) Answer(model T) (string, error) {
	return i.opts.Answer(model)
}

// Render renders the question.
func (i inputQuestion[T]) Render(model T) string {
	if i.opts.Render != nil {
		// Overriding render, so just use that function.
		return i.opts.Render(model, *i.state)
	}

	a, err := i.Answer(model)
	if err == nil {
		return fmt.Sprintf("%s: %s\n", i.opts.Prompt, cli.BoldStyle.Render(a))
	}

	b := &strings.Builder{}
	b.WriteString(cli.BoldStyle.Render(i.opts.Prompt+":") + "\n")
	b.WriteString(i.state.View())
	return b.String()
}

// UpdateTea updates the model T via Bubbletea UI
func (i inputQuestion[T]) UpdateTea(model T, msg tea.Msg) (Question[T], tea.Cmd) {
	state, cmd := i.state.Update(msg)
	i.state = &state

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		_ = i.UpdateAnswer(model, i.state.Value())
	}

	return i, cmd
}

func (i inputQuestion[T]) UpdateAnswer(model T, value interface{}) error {
	return i.opts.UpdateAnswer(model, value)
}

// Next returns the next question in the chain
func (i inputQuestion[T]) Next(model T) string {
	if i.opts.Next == nil {
		return ""
	}
	return i.opts.Next(model)
}
