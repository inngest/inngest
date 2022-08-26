package form

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type FormOpts[T any] struct {
	Questions []Question[T]
	RootID    string

	// Intro renders an intro to the form.
	Intro func(T) string
}

func NewForm[T any](model T, opts FormOpts[T]) (*Form[T], error) {
	form := &Form[T]{
		Model:    model,
		FormOpts: opts,
	}
	if form.Root() == nil {
		return nil, fmt.Errorf("root question not found: %s", opts.RootID)
	}
	return form, nil
}

type Form[T any] struct {
	FormOpts[T]

	// Model represents the model that this form updates.
	Model T

	root Question[T]
}

func (f *Form[T]) Root() Question[T] {
	if f.root == nil {
		f.root = f.Question(f.RootID)
	}
	return f.root
}

// Question returns a question by ID
func (f Form[T]) Question(id string) Question[T] {
	for _, q := range f.Questions {
		if q.ID() == id {
			return q
		}
	}
	return nil
}

func (f *Form[T]) Complete() (bool, error) {
	queue := []Question[T]{f.Root()}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if item == nil {
			continue
		}

		_, err := item.Answer(f.Model)
		if err == ErrUnanswered {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		next := item.Next(f.Model)
		queue = append(queue, f.Question(next))
	}
	return true, nil
}

func (f *Form[T]) Ask() error {
	if ok, err := f.Complete(); ok && err == nil {
		return nil
	}

	// Create a new tea program.
	p := &teaProgram[T]{
		form:     f,
		question: f.Root(),
	}

	if err := tea.NewProgram(p).Start(); err != nil {
		log.Fatal(err)
	}

	return nil
}
