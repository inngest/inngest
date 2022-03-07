package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/event-schemas/events"
	"github.com/inngest/inngestctl/pkg/function"
	"golang.org/x/term"
)

const (
	stateAskName  = "name"
	stateAskEvent = "event"
	stateDone     = "done"

	eventPlaceholder = "What event name triggers this function?  Use your own event name or an event from an integration."

	// the Y offset when rendering the event browser.
	eventBrowserOffset = 25
)

// NewInitModel renders the UI for initializing a new function.
func NewInitModel() (*initModel, error) {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))

	schemas, err := fetchEvents()
	if err != nil {
		fmt.Println(RenderWarning("We couldn't fetch your latest events from our API"))
	}

	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	f := &initModel{
		width:     width,
		height:    height,
		events:    schemas,
		state:     stateAskName,
		textinput: textinput.New(),
	}
	f.textinput.Focus()
	f.textinput.CharLimit = 156
	f.textinput.Width = width
	f.textinput.Prompt = "→  "

	return f, nil
}

// initModel represehts the survey state when creating a new function.
type initModel struct {
	// The width of the terminal.  Necessary for styling content such
	// as the welcome message, the evnet browser, etc.
	width  int
	height int

	events []events.Event

	// The current state we're on.
	state string

	name  string
	event string

	textinput textinput.Model
	browser   *EventBrowser
}

// Ensure that initModel fulfils the tea.Model interface.
var _ tea.Model = (*initModel)(nil)

// DidQuitEarly returns whether we quit the walthrough early.
func (f *initModel) DidQuitEarly() bool {
	return false
}

// Function returns the formatted function given answers from the TUI state.
// This returns an error if the function is not valid.
func (f *initModel) Function() (*function.Function, error) {
	// Attempt to find the schema that matches this event, and dump the
	// cue schema inside the function.
	var ed *function.EventDefinition
	for _, e := range f.events {
		if e.Name == f.event {
			ed = &function.EventDefinition{
				Format: function.FormatCue,
				Synced: true,
				Def:    e.Cue,
			}
			break
		}
	}

	fn, err := function.New()
	if err != nil {
		return nil, err
	}

	fn.Name = f.name
	fn.Triggers = []function.Trigger{
		{EventTrigger: &function.EventTrigger{Event: f.event, Definition: ed}},
	}

	return fn, fn.Validate()
}

func (f *initModel) Init() tea.Cmd {
	// Remove the first N lines of the CLI height, which account for the header etc.
	f.browser, _ = NewEventBrowser(f.width, f.height-eventBrowserOffset, f.events)
	return nil
}

func (f *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Globals.  These always run whenever changes happen.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.browser.UpdateSize(f.width, f.height-eventBrowserOffset)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return f, tea.Quit
		}
	}

	switch f.state {
	case stateAskName:
		return f.updateName(msg)
	case stateAskEvent:
		return f.updateEvent(msg)
	}

	return f, nil
}

func (f *initModel) updateName(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	f.textinput.Placeholder = "What should this function be called?"
	f.name = f.textinput.Value()
	f.textinput, cmd = f.textinput.Update(msg)

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && f.name != "" {
		f.textinput.Placeholder = eventPlaceholder
		f.textinput.SetValue("")
		f.state = stateAskEvent
	}

	return f, cmd
}

func (f *initModel) updateEvent(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	f.textinput.Placeholder = eventPlaceholder
	f.event = f.textinput.Value()
	f.textinput, cmd = f.textinput.Update(msg)
	cmds = append(cmds, cmd)
	f.browser.UpdatePrefix(f.event)

	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyEnter:
			if f.event == "" {
				// There's no name, so don't do anything.
				return f, nil
			}

			// We're done.  Quit this TUI to resume the CLI functionality with state.
			f.state = stateDone
			return f, tea.Quit

		case tea.KeyUp, tea.KeyDown:
			// Send this to the event browser to navigate the filter list.
			_, cmd = f.browser.Update(msg)
			cmds = append(cmds, cmd)
		case tea.KeyTab:
			// Select the event from the browser.
			if evt := f.browser.Selected(); evt != nil {
				f.event = evt.Name
				f.browser.UpdatePrefix(f.event)
				f.textinput.SetValue(f.event)
			}
		}
		// If we're holding ctrl, send this to the filter list.  This allows us to use
		// ctrl+j/k to navigate the code viewer.
		if strings.Contains(key.String(), "ctrl+") {
			_, cmd = f.browser.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return f, tea.Batch(cmds...)
}

// View is called to render the CLI's UI.
func (f *initModel) View() string {

	b := &strings.Builder{}

	_, _ = b.WriteString(f.renderWelcome())
	b.WriteString("\n\n")
	b.WriteString(BoldStyle.Render("Let's get you set up with a new serverless function."))
	b.WriteString("\n")
	b.WriteString(TextStyle.Copy().Foreground(Feint).Render("Answer these questions to get started."))
	b.WriteString("\n\n")
	b.WriteString(f.renderState())

	// If we have no workflow name, ask for it.
	switch f.state {
	case stateAskName:
		b.WriteString(f.renderName())
	case stateAskEvent:
		b.WriteString(f.renderEvent())
	case stateDone:
		// Done.  Add some padding to the final view for the parent.
		b.WriteString("\n")
	}

	return b.String()
}

// renderState renders the already answered questions.
func (f *initModel) renderState() string {
	if f.state == stateAskName {
		return ""
	}
	b := &strings.Builder{}
	b.WriteString(fmt.Sprintf("1. Function name: %s\n", f.name))
	if f.event != "" && f.state != stateAskEvent {
		b.WriteString(fmt.Sprintf("2. Event: %s\n", f.event))
	}
	return b.String()
}

// renderName renders the question which asks for the function name.
func (f *initModel) renderName() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render("1. Function name:") + "\n")
	b.WriteString(f.textinput.View())
	return b.String()
}

// renderEvent renders the question wihch asks for the event trigger, plus the
// event browser.
func (f *initModel) renderEvent() string {
	b := &strings.Builder{}

	b.WriteString(
		BoldStyle.Render("2. Event trigger:") +
			TextStyle.Copy().Foreground(Feint).Render(" (enter to continue)") + "\n",
	)
	b.WriteString(f.textinput.View() + "\n\n\n")

	// Render two columns of text.
	headerMsg := lipgloss.JoinVertical(lipgloss.Center,
		BoldStyle.Copy().Foreground(Feint).Render("Event browser"),
		TextStyle.Copy().Foreground(Feint).Render("Showing events within your workspace, and their Cue schema. ")+
			BoldStyle.Copy().Foreground(Feint).Render("Tab: select.  Ctrl-j/k: navigate code"),
	)

	// Place the header in the center of the screen.
	header := lipgloss.Place(
		f.width, 3,
		lipgloss.Center, lipgloss.Center,
		headerMsg,
	)

	b.WriteString(header)

	b.WriteString(f.browser.View())

	return b.String()
}

func (f *initModel) renderWelcome() string {
	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	dialog := lipgloss.Place(
		f.width, 7,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(TextStyle.Copy().Bold(true).PaddingLeft(3).PaddingRight(4).Render("👋 Welcome to Inngest!")),
		lipgloss.WithWhitespaceChars("⎻⎼⎽⎼"),
		lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#333333"}),
	)
	return dialog
}

func fetchEvents() ([]events.Event, error) {
	// Fetch our events.
	//
	// XXX: This should be moved to our GQL endpoint and should request
	// the users events, once merged.
	resp, err := http.Get("https://schemas.inngest.com/generated.json")
	if err != nil {
		return nil, fmt.Errorf("error fetching events: %w", err)
	}
	defer resp.Body.Close()
	// Update the event browser directly here.  This runs before View() and
	// we don't need to run through the update flow.
	byt, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error fetching events: %w", err)
	}
	schemas := []events.Event{}
	err = json.Unmarshal(byt, &schemas)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling events: %w", err)
	}
	return schemas, nil
}
