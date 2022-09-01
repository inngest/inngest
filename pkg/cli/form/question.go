package form

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	ErrUnanswered = fmt.Errorf("this question is unanswered")
)

type Question[T any] interface {
	// ID allows us to reference questions, eg. for setting the root
	ID() string

	// Answer returns the answer as a string.  If the question is
	// unanswered, this must return the error ErrUnanswered.
	Answer(T) (string, error)

	// Render renders the question.
	Render(T) string

	// UpdateTea updates the model T via Bubbletea UI, returning any
	// commands necessary to re-render the state.
	UpdateTea(model T, msg tea.Msg) (Question[T], tea.Cmd)

	// UpdateAnswer is called when the qestion is submitted.
	UpdateAnswer(model T, answer interface{}) error

	// Next returns the next question in the chain
	Next(T) string
}

type QuestionOpts[T any] struct {
	// Answer returns the current answer as a string for display, or
	// ErrUnanswered if this question has no answer.
	Answer func(T) (string, error)

	// UpdateAnswer is called with the answer from the question to
	// update the model.
	//
	// This must return a custom error if the answer is invalid,
	// or ErrUnanswered if there's no answer.
	UpdateAnswer func(T, interface{}) error

	// Next returns the ID of the next question in the chain, if applicable.
	Next func(T) string

	// UpdateTea is called when receing a tea.Msg to update any internal
	// tea models.
	UpdateTea func(T, msg tea.Msg) (Question[T], tea.Cmd)
}

// NewQuestion returns a question that custom-renders everything with no base.
func NewQuestion[T any](id string, opts QuestionOpts[T], render func(T) string) Question[T] {
	return &question[T]{
		id:     id,
		opts:   opts,
		render: render,
	}
}

type question[T any] struct {
	id     string
	opts   QuestionOpts[T]
	render func(T) string
}

func (i question[T]) ID() string {
	return i.id
}

func (i question[T]) Answer(model T) (string, error) {
	return i.opts.Answer(model)
}

// Render renders the question.
func (i question[T]) Render(model T) string {
	return i.render(model)
}

// UpdateTea updates the model T via Bubbletea UI
func (i question[T]) UpdateTea(model T, msg tea.Msg) (Question[T], tea.Cmd) {
	return i, nil
}

func (i question[T]) UpdateAnswer(model T, value interface{}) error {
	return i.opts.UpdateAnswer(model, value)
}

// Next returns the next question in the chain
func (i question[T]) Next(model T) string {
	if i.opts.Next == nil {
		return ""
	}
	return i.opts.Next(model)
}
