package form

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type teaProgram[T any] struct {
	form *Form[T]

	// Store the current question.
	question Question[T]
}

func (t teaProgram[T]) Init() tea.Cmd {
	return nil
}

func (t teaProgram[T]) View() string {
	s := &strings.Builder{}

	if t.form.Intro != nil {
		s.WriteString(t.form.Intro(t.form.Model) + "\n")
	}

	q := t.form.Question(t.form.RootID)
	for q != nil {
		if _, err := q.Answer(t.form.Model); err == nil {
			s.WriteString(q.Render(t.form.Model))
		}
		q = t.form.Question(q.Next(t.form.Model))
	}

	if t.question != nil {
		s.WriteString(t.question.Render(t.form.Model))
	}

	return s.String()
}

func (t *teaProgram[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// With no question, quit rendering.
	if t.question == nil {
		return t, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		_, _ = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return t, tea.Quit
		}
	}

	// Ensure we update the current question.  The UpdateTea method here is a
	// pointer, so this should
	if t.question != nil {
		t.question, cmd = t.question.UpdateTea(t.form.Model, msg)
		cmds = append(cmds, cmd)
	}

	// Ensure we don't re-ask any answered questions.
	for {
		if t.question == nil {
			break
		}
		if _, err := t.question.Answer(t.form.Model); err != nil {
			// This isn't answered, so we can skip going to the next
			// question.
			break
		}
		// For each question that's answered, skip to the next question.
		t.question = t.form.Question(t.question.Next(t.form.Model))
	}

	if t.question == nil {
		// Ensure we handle nil questions when progressing to the end
		// of the form.
		return t, tea.Quit
	}

	return t, tea.Batch(cmds...)
}
