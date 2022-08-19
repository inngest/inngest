package initialize

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngest/pkg/cli"
	humancron "github.com/lnquy/cron"
	"github.com/robfig/cron/v3"
)

var (
	// questionName renders a text input to ask for the function name.
	questionName = question{
		answered: func(m *initModel) bool {
			return m.name != ""
		},
		render: func(m *initModel) string {
			if m.name != "" {
				return "Function name: " + cli.BoldStyle.Render(m.name) + "\n"
			}

			b := &strings.Builder{}
			b.WriteString(cli.BoldStyle.Render("Function name:") + "\n")
			b.WriteString(m.textinput.View())
			return b.String()
		},
		update: func(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
			var cmd tea.Cmd
			m.textinput.Placeholder = "What should this function be called?"
			m.textinput, cmd = m.textinput.Update(msg)
			value := m.textinput.Value()

			if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && value != "" {
				m.name = value
				m.textinput.Placeholder = eventPlaceholder
				m.textinput.SetValue("")
			}
			return m, cmd
		},
		next: func(m *initModel) InitQuestion {
			if m.template != "" {
				return nil
			}

			return questionTrigger
		},
	}

	// questionTrigger asks the user to select between a scheduled and event trigger.
	questionTrigger = question{
		answered: func(m *initModel) bool {
			return m.triggerType != ""
		},
		render: func(m *initModel) string {
			if m.triggerType != "" {
				return "Function trigger: " + cli.BoldStyle.Render(m.triggerType) + "\n"
			}

			b := &strings.Builder{}
			b.WriteString(cli.BoldStyle.Render("How should the function run?") + "\n\n")
			b.WriteString(m.triggerList.View())
			return b.String()
		},
		update: updateTrigger,
		next: func(m *initModel) InitQuestion {
			switch m.triggerType {
			case triggerTypeEvent:
				return questionEventName
			case triggerTypeScheduled:
				return questionSchedule
			default:
				return nil
			}
		},
	}

	// questionEventName asks for the event trigger name, rendering an event browser.
	questionEventName = question{
		answered: func(m *initModel) bool {
			return m.event != ""
		},
		render: renderEvent,
		update: updateEvent,
		next: func(m *initModel) InitQuestion {
			return questionRuntime
		},
	}

	// questionSchedule asks for the event trigger name, rendering an event browser.
	questionSchedule = question{
		answered: func(m *initModel) bool {
			return m.cron != ""
		},
		render: renderSchedule,
		update: updateSchedule,
		next: func(m *initModel) InitQuestion {
			return questionRuntime
		},
	}

	// questionRuntime asks for the event trigger name, rendering an event browser.
	questionRuntime = question{
		answered: func(m *initModel) bool {
			return m.runtimeType != ""
		},
		render: renderRuntime,
		update: updateRuntime,
		next: func(m *initModel) InitQuestion {
			switch m.runtimeType {
			case runtimeHTTP:
				// Ask for URL
				return questionURL
			case runtimeDocker:
				// Ask for Language
				return questionLanguage
			default:
				return nil
			}
		},
	}

	// questionLanguage asks for the event trigger name, rendering an event browser.
	questionLanguage = question{
		name: "language",
		answered: func(m *initModel) bool {
			return m.language != ""
		},
		render: renderLanguage,
		update: updateLanguage,
		next: func(m *initModel) InitQuestion {
			return questionScaffold
		},
	}

	// questionScaffold asks for the scaffold to use
	questionScaffold = question{
		name: "scaffold",
		answered: func(m *initModel) bool {
			return m.scaffold != nil
		},
		render: renderScaffold,
		update: updateScaffold,
		next: func(m *initModel) InitQuestion {
			return nil
		},
	}

	// questionURL asks for the event trigger name, rendering an event browser.
	questionURL = question{
		answered: func(m *initModel) bool {
			return m.url != ""
		},
		render: renderURL,
		update: updateURL,
		next: func(m *initModel) InitQuestion {
			return nil
		},
	}
)

type InitQuestion interface {
	// Answered returns whether this question has an answer.  If so,
	// the controlling model should skip to the next question in the
	// chain via Next()
	Answered(m *initModel) bool

	// Render renders the question.
	Render(m *initModel) string

	// Update renders the question.
	Update(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd)

	// Next retunrs the next question in the chain
	Next(m *initModel) InitQuestion
}

// question represents an abstract question which is used for single
// initialization and zero allocation after init.
type question struct {
	// name is a local identifier, used when debugging.
	name     string
	answered func(m *initModel) bool
	render   func(m *initModel) string
	update   func(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd)
	next     func(m *initModel) InitQuestion
}

func (q question) Answered(m *initModel) bool                            { return q.answered(m) }
func (q question) Render(m *initModel) string                            { return q.render(m) }
func (q question) Update(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) { return q.update(m, msg) }
func (q question) Next(m *initModel) InitQuestion                        { return q.next(m) }

// updateTrigger updates the trigger within the given model.
func updateTrigger(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// We only care about enter keypresses to select an item in the list.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		// We've selected a trigger..
		m.triggerType = m.triggerList.SelectedItem().FilterValue()

		// Depending on the trigger, we're going to ask for the event type
		// or the schedule.
		switch m.triggerType {
		case triggerTypeEvent:
			m.textinput.Placeholder = eventPlaceholder
			m.textinput.SetValue("")
		case triggerTypeScheduled:
			m.textinput.Placeholder = cronPlaceholder
			m.textinput.SetValue("")
		}
		return m, nil
	}

	m.triggerList, cmd = m.triggerList.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func renderEvent(m *initModel) string {
	if m.event != "" {
		return "Event name: " + cli.BoldStyle.Render(m.event) + "\n"
	}

	b := &strings.Builder{}

	b.WriteString(
		cli.BoldStyle.Render(fmt.Sprintf("%d. Event trigger:", m.transitions)) +
			cli.TextStyle.Copy().Foreground(cli.Feint).Render(" (enter to continue)") + "\n",
	)
	b.WriteString(m.textinput.View() + "\n")

	if m.eventFetchError != nil {
		b.WriteString("\n" + cli.RenderWarning(m.eventFetchError.Error()) + "\n\n")
	}

	if m.height < 20 {
		b.WriteString("\n" + cli.RenderWarning("Your TTY doesn't have enough height to render the event browser") + "\n")
		return b.String()
	}

	b.WriteString("\n")

	// Render two columns of text.
	headerMsg := lipgloss.JoinVertical(lipgloss.Center,
		cli.BoldStyle.Copy().Foreground(cli.Feint).Render("Event browser"),
		cli.TextStyle.Copy().Foreground(cli.Feint).Render("Showing events within your workspace, and their Cue schema. ")+
			cli.BoldStyle.Copy().Foreground(cli.Feint).Render("Tab: select. Ctrl-j/k: navigate code"),
	)

	// Place the header in the center of the screen.
	header := lipgloss.Place(
		m.width, 3,
		lipgloss.Center, lipgloss.Center,
		headerMsg,
	)

	b.WriteString(header)
	b.WriteString(m.browser.View())
	return b.String()
}

func updateEvent(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.textinput.Placeholder = eventPlaceholder
	m.textinput, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)

	m.eventFilter = m.textinput.Value()
	m.browser.UpdatePrefix(m.eventFilter)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			// Use the input value by default
			m.event = m.textinput.Value()
			if sel := m.browser.Selected(); sel != nil {
				m.event = sel.Name
			}
			if m.textinput.Value() == "" || m.event == "" {
				// There's no name, so don't do anything.
				return m, nil
			}

			return m, nil
		case tea.KeyUp, tea.KeyDown:
			// Send this to the event browser to navigate the filter list.
			_, cmd = m.browser.Update(msg)
			cmds = append(cmds, cmd)
		case tea.KeyTab:
			// Select the event from the browser.
			if evt := m.browser.Selected(); evt != nil {
				m.event = evt.Name
				m.browser.UpdatePrefix(m.event)
				m.textinput.SetValue(m.event)
			}
		}
		// If we're holding ctrl, send this to the filter list.  This allows us to use
		// ctrl+j/k to navigate the code viewer.
		if strings.Contains(key.String(), "ctrl+") {
			_, cmd = m.browser.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func renderSchedule(m *initModel) string {
	if m.cron != "" {
		return "Cron schedule: " + cli.BoldStyle.Render(m.cron) + " (" + m.humanCron + ")\n"
	}

	b := &strings.Builder{}
	b.WriteString(cli.BoldStyle.Render("Cron schedule:") + "\n")
	b.WriteString(m.textinput.View())
	if m.cronError != nil {
		b.WriteString("\n")
		b.WriteString(cli.RenderWarning(m.cronError.Error()))
	}
	if !m.nextCron.IsZero() {
		b.WriteString("\n")
		dur := humanizeDuration(time.Until(m.nextCron))

		if m.humanCron != "" {
			b.WriteString(cli.TextStyle.Copy().Foreground(cli.Feint).Bold(true).Render(m.humanCron) + ". ")
		}
		b.WriteString(cli.TextStyle.Copy().Foreground(cli.Feint).Render("This would next run at: " + m.nextCron.Format(time.RFC3339) + " (in " + dur + ")\n"))
	}
	return b.String()
}

func updateSchedule(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textinput.Placeholder = cronPlaceholder
	m.textinput, cmd = m.textinput.Update(msg)

	value := m.textinput.Value()

	schedule, err := cron.
		NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).
		Parse(value)
	if err != nil {
		m.cronError = fmt.Errorf("This isn't a valid cron schedule")
		m.humanCron = ""
		m.nextCron = time.Time{}
	} else {
		m.cronError = nil
		if desc, err := humancron.NewDescriptor(); err == nil {
			m.humanCron, _ = desc.ToDescription(m.cron, humancron.Locale_en)
		}
		m.nextCron = schedule.Next(time.Now())
	}

	// Confirming the cron should update the cron model value.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && value != "" && m.cronError == nil {
		m.cron = value
		m.textinput.Placeholder = cronPlaceholder
		m.textinput.SetValue("")
	}

	return m, cmd
}

func renderRuntime(m *initModel) string {
	if m.runtimeType != "" {
		return "Function type: " + cli.BoldStyle.Render(m.runtimeType) + "\n"
	}

	b := &strings.Builder{}
	b.WriteString(cli.BoldStyle.Render("Do you want to write a new function or call a URL?") + "\n\n")
	b.WriteString(m.runtimeList.View())
	return b.String()
}

func updateRuntime(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// We only care about enter keypresses to select an item in the list.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		m.runtimeType = m.runtimeList.SelectedItem().FilterValue()

		switch m.runtimeType {
		case runtimeHTTP:
			// Ask for the URL we're hitting.
			m.textinput.Placeholder = urlPlaceholder
			m.textinput.SetValue("")
		case runtimeDocker:
			// If we've attempted to update the scaffolds but have zero languages available,
			// quit early.
			if m.scaffolds == nil || (m.scaffoldDone && len(m.scaffolds.Languages) == 0) {
				m.state = stateDone
				return m, tea.Quit
			}
		}
		return m, nil
	}

	m.runtimeList, cmd = m.runtimeList.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func renderLanguage(m *initModel) string {
	if m.language != "" {
		return "Language: " + cli.BoldStyle.Render(m.language) + "\n"
	}

	if !m.scaffoldDone || m.scaffolds == nil || len(m.scaffolds.Languages) == 0 {
		return fmt.Sprintf("\n\n   %s Loading scaffold templates... Press q to quit and save your function without using a template.\n\n", m.loading.View())
	}
	b := &strings.Builder{}
	b.WriteString(
		cli.BoldStyle.Render("Which language would you like to use?") +
			cli.TextStyle.Copy().Foreground(cli.Feint).Render(" (q to quit without using a scaffold)") + "\n\n",
	)
	if m.scaffoldCacheError != nil {
		b.WriteString(cli.RenderWarning(fmt.Sprintf("Couldn't update scaffolds: %s", m.scaffoldCacheError)) + "\n\n")
	}
	b.WriteString(m.languageList.View())
	return b.String()
}

func updateLanguage(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		// We've selected a language.
		m.language = m.languageList.SelectedItem().FilterValue()
		// We have no templates for unsupported languages, so send a done signal.
		if m.language == anotherLanguage {
			m.state = stateDone
			return m, tea.Quit
		}
		// If we only have one template for this language, use that and quit.
		if len(m.scaffolds.Languages[m.language]) == 1 {
			m.scaffold = &m.scaffolds.Languages[m.language][0]
			m.state = stateDone
			return m, tea.Quit
		}
	}

	m.languageList, cmd = m.languageList.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func renderURL(m *initModel) string {
	if m.url != "" {
		return "URL: " + cli.BoldStyle.Render(m.url) + "\n"
	}

	b := &strings.Builder{}
	b.WriteString(cli.BoldStyle.Render("URL to call:") + "\n")
	b.WriteString(m.textinput.View())
	if m.urlError != nil {
		b.WriteString("\n")
		b.WriteString(cli.RenderWarning(m.urlError.Error()))
	}
	return b.String()
}

func updateURL(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textinput.Placeholder = urlPlaceholder
	m.textinput, cmd = m.textinput.Update(msg)

	value := m.textinput.Value()

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		m.urlError = fmt.Errorf("This isn't a valid URL")
	} else {
		m.urlError = nil
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && m.name != "" && m.cronError == nil {
		m.url = value
		m.textinput.Placeholder = urlPlaceholder
		m.textinput.SetValue("")
		m.state = stateDone
		return m, tea.Quit
	}

	return m, cmd
}

func renderScaffold(m *initModel) string {
	// TODO: Allow selecting multiple scaffolds when necessary.
	return ""
}

func updateScaffold(m *initModel, msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.scaffoldDone {
		return m, nil
	}
	if m.language != "" && len(m.scaffolds.Languages[m.language]) == 1 {
		m.scaffold = &m.scaffolds.Languages[m.language][0]
		m.state = stateDone
	}

	// TODO: When allowing multiple scaffolds, allow selecting the scaffold
	// from a scaffold list here.
	return m, tea.Quit
}
