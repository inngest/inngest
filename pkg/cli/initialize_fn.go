package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/state"
	"github.com/inngest/inngestctl/pkg/function"
	"github.com/inngest/inngestctl/pkg/scaffold"
	"golang.org/x/term"
)

const (
	stateAskName     = "name"
	stateAskEvent    = "event"
	stateAskLanguage = "language"
	stateAskScaffold = "scaffold"

	// stateDone is triggered as soon as we're complete and are quitting the walkthrough.
	stateDone = "done"
	// stateQuit is used when terminating the walkthrough early
	stateQuit = "quit"

	eventPlaceholder = "What event name triggers this function?  Use your own event name or an event from an integration."

	// anotherLanguage is the list item which is rendered at the bottom for a user
	// to select if we have no scaffolds for their language.
	anotherLanguage = "another language"
)

type InitOpts struct {
	ShowWelcome bool
}

// NewInitModel renders the UI for initializing a new function.
func NewInitModel(o InitOpts) (*initModel, error) {
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))

	languageDelegate := list.NewDefaultDelegate()
	languageDelegate.ShowDescription = false

	f := &initModel{
		width:       width,
		height:      height,
		showWelcome: o.ShowWelcome,
		events:      []client.Event{},
		state:       stateAskName,
		textinput:   textinput.New(),
		loading:     spinner.New(),
	}

	f.languageList = list.New([]list.Item{}, languageDelegate, width, height-f.eventBrowserOffset())
	f.scaffoldList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, height-f.eventBrowserOffset())

	f.textinput.Focus()
	f.textinput.CharLimit = 156
	f.textinput.Width = width
	f.textinput.Prompt = "→  "

	f.loading.Spinner = spinner.Dot
	f.loading.Style = lipgloss.NewStyle().Foreground(Primary)

	f.languageList.SetShowFilter(false)
	f.languageList.SetShowHelp(false)
	f.languageList.SetShowStatusBar(false)
	f.languageList.SetShowStatusBar(false)
	f.languageList.SetShowTitle(false)

	return f, nil
}

// initModel represehts the survey state when creating a new function.
type initModel struct {
	// The width of the terminal.  Necessary for styling content such
	// as the welcome message, the evnet browser, etc.
	width  int
	height int

	// whether to show the welcome message.
	showWelcome bool

	// The current state we're on.
	state string

	// name is the function name entered from the user
	name string
	// event is the event entered from the user
	event string
	// language is the language selected by the user
	language string
	// scaffold is the scaffold selected by the user
	scaffold *scaffold.Template

	events          []client.Event
	eventFetchError error

	// scaffolds are all scaffolds we have available, parsed after scaffolds
	// have updated.
	scaffolds *scaffold.Mapping
	// scaffoldCacheError is filled if pulling the cache of scaffolds fails.
	// this lets us render a warning if updating wasnt successful.
	scaffoldCacheError error
	// scaffoldDone records whether we have finished updating scaffolds.  this
	// lets us render spinners.
	scaffoldDone bool

	// these are models used to render helpers.
	textinput    textinput.Model
	browser      *EventBrowser
	languageList list.Model
	scaffoldList list.Model
	loading      spinner.Model
}

// Ensure that initModel fulfils the tea.Model interface.
var _ tea.Model = (*initModel)(nil)

// DidQuitEarly returns whether we quit the walthrough early.
func (f *initModel) DidQuitEarly() bool {
	return f.state == stateQuit
}

// Function returns the formatted function given answers from the TUI state.
// This returns an error if the function is not valid.
func (f *initModel) Function() (*function.Function, error) {
	// Attempt to find the schema that matches this event, and dump the
	// cue schema inside the function.
	var ed *function.EventDefinition
	for _, e := range f.events {
		if e.Name == f.event && len(e.Versions) > 0 {
			ed = &function.EventDefinition{
				Format: function.FormatCue,
				Synced: true,
				Def:    e.Versions[0].CueType,
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

func (f *initModel) Template() *scaffold.Template {
	return f.scaffold
}

// the Y offset when rendering the event browser.
func (f *initModel) eventBrowserOffset() int {
	if f.showWelcome && f.height > 35 {
		return 25
	}
	return 11
}

func (f *initModel) Init() tea.Cmd {
	// Remove the first N lines of the CLI height, which account for the header etc.
	f.browser, _ = NewEventBrowser(f.width, f.height-f.eventBrowserOffset(), f.events, true)
	return tea.Batch(
		f.loading.Tick,
		func() tea.Msg {
			schemas, err := fetchEvents()
			if err != nil {
				f.eventFetchError = err
			}

			sort.Slice(schemas, func(i, j int) bool {
				return schemas[i].Name < schemas[j].Name
			})
			f.events = schemas
			f.browser.SetEvents(f.events)

			// XXX: We could / should send a message here which contains the schemas
			// and/or error into Update directly.  However, because we have an initModel
			// singleton which is a pointer, it's safe to update our state here and return
			// a nil message which will trigger a re-render.  It's also easier and has less
			// allocations;  we're not creating a new struct which is passed via goroutines
			// to update the initModel members.
			return nil
		},
		func() tea.Msg {
			f.scaffoldCacheError = scaffold.UpdateCache(context.Background())
			f.scaffoldDone = true
			return nil
		},
	)
}

func (f *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// Base stuff we should always run.
	if !f.scaffoldDone {
		// Spin the scaffolding spinner if we're waiting.
		f.loading, cmd = f.loading.Update(msg)
		cmds = append(cmds, cmd)
	}

	if f.scaffoldDone && f.scaffolds == nil {
		f.scaffolds, _ = scaffold.Parse(context.Background())
	}

	// Globals.  These always run whenever changes happen.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.browser.UpdateSize(f.width, f.height-f.eventBrowserOffset())
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			f.state = stateQuit
			return f, tea.Quit
		}

		if msg.String() == "q" && f.state == stateAskLanguage {
			// If the user pressed "q" when waiting for scaffolds to be
			// loaded, quit and make the base inngest.json file.
			f.state = stateDone
			return f, tea.Quit
		}
	}

	// Run the update events for each state.
	_, cmd = func() (tea.Model, tea.Cmd) {
		switch f.state {
		case stateAskName:
			return f.updateName(msg)
		case stateAskEvent:
			return f.updateEvent(msg)
		case stateAskLanguage:
			return f.updateLanguage(msg)
		}
		return f, nil
	}()
	// Merge the async commands from each state into the top-level commands to run.
	cmds = append(cmds, cmd)

	// Return our updated state and all commands to run.
	return f, tea.Batch(cmds...)
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
			if sel := f.browser.Selected(); sel != nil {
				f.event = sel.Name
			}

			if f.event == "" {
				// There's no name, so don't do anything.
				return f, nil
			}

			// If we've attempted to update the scaffolds but have zero languages available,
			// quit early.
			if f.scaffolds == nil || (f.scaffoldDone && len(f.scaffolds.Languages) == 0) {
				f.state = stateDone
				return f, tea.Quit
			}

			// We have scaffolds with languages available, so move to the languages question.
			f.state = stateAskLanguage
			return f, nil

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

func (f *initModel) updateLanguage(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		// We've selected a language.
		f.language = f.languageList.SelectedItem().FilterValue()

		// We have no templates for unsupported languages, so send a done signal.
		if f.language == anotherLanguage {
			f.state = stateDone
			return f, tea.Quit
		}

		// If we only have one template for this language, use that and quit.
		if len(f.scaffolds.Languages[f.language]) == 1 {
			f.scaffold = &f.scaffolds.Languages[f.language][0]
			f.state = stateDone
			return f, tea.Quit
		}

		// Switch state to render a list to ask which scaffold to use.
		f.state = stateAskScaffold
	}

	f.languageList, cmd = f.languageList.Update(msg)
	cmds = append(cmds, cmd)
	return f, tea.Batch(cmds...)
}

// View is called to render the CLI's UI.
func (f *initModel) View() string {

	b := &strings.Builder{}

	if f.height > 35 {
		b.WriteString(f.renderIntro(f.showWelcome))
	}

	// If we have no workflow name, ask for it.
	switch f.state {
	case stateAskName:
		b.WriteString(f.renderName())
	case stateAskEvent:
		b.WriteString(f.renderEvent())
	case stateAskLanguage:
		b.WriteString(f.renderLanguageSelection())
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
	if f.language != "" && f.state != stateAskLanguage {
		b.WriteString(fmt.Sprintf("3. Language: %s\n", f.language))
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

	if f.eventFetchError != nil {
		b.WriteString("\n" + RenderWarning(fmt.Sprintf("We couldn't fetch your latest events from our API: %s", f.eventFetchError)) + "\n")
	}

	if f.height < 20 {
		b.WriteString("\n" + RenderWarning("Your TTY doesn't have enough height to render the event browser") + "\n")
		return b.String()
	}

	// Render two columns of text.
	headerMsg := lipgloss.JoinVertical(lipgloss.Center,
		BoldStyle.Copy().Foreground(Feint).Render("Event browser"),
		TextStyle.Copy().Foreground(Feint).Render("Showing events within your workspace, and their Cue schema. ")+
			BoldStyle.Copy().Foreground(Feint).Render("Tab: select. Ctrl-j/k: navigate code"),
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

func (f *initModel) renderLanguageSelection() string {
	if !f.scaffoldDone || f.scaffolds == nil || len(f.scaffolds.Languages) == 0 {
		return fmt.Sprintf("\n\n   %s Loading scaffold templates... Press q to quit and save your function without using a template.\n\n", f.loading.View())
	}

	b := &strings.Builder{}

	languages := []list.Item{}
	for k := range f.scaffolds.Languages {
		languages = append(languages, initListItem{name: k})
	}
	languages = append(languages, initListItem{name: anotherLanguage})
	f.languageList.SetItems(languages)

	b.WriteString(
		BoldStyle.Render("3. Which language would you like to use?") +
			TextStyle.Copy().Foreground(Feint).Render(" (q to quit without using a scaffold)") + "\n\n",
	)

	if f.scaffoldCacheError != nil {
		b.WriteString(RenderWarning(fmt.Sprintf("Couldn't update scaffolds: %s", f.scaffoldCacheError)) + "\n\n")
	}
	b.WriteString(f.languageList.View())

	return b.String()
}

func (f *initModel) renderIntro(welcome bool) string {
	b := &strings.Builder{}
	if welcome {
		b.WriteString(f.renderWelcome())
	}
	b.WriteString("\n\n")
	b.WriteString(BoldStyle.Render("Let's get you set up with a new serverless function."))
	b.WriteString("\n")
	b.WriteString(TextStyle.Copy().Foreground(Feint).Render("Answer these questions to get started."))
	b.WriteString("\n\n")
	b.WriteString(f.renderState())
	return b.String()
}

func (f *initModel) renderWelcome() string {
	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	dialog := lipgloss.Place(
		f.width, 7,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(TextStyle.Copy().Bold(true).PaddingLeft(3).PaddingRight(4).Render("👋 Welcome to Inngest!")),
		lipgloss.WithWhitespaceChars("⎼⎽"),
		lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#333333"}),
	)
	return dialog
}

func fetchEvents() ([]client.Event, error) {
	ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()

	var workspaceID *uuid.UUID
	if s, err := state.GetState(ctx); err == nil {
		workspaceID = &s.SelectedWorkspace.ID
	}

	c := state.Client(ctx)
	evts, err := c.AllEvents(ctx, &client.EventQuery{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		// Attempt to fetch unauthed events.
		return c.AllEvents(ctx, nil)
	}
	return evts, nil
}

type initListItem struct {
	name        string
	description string
}

func (i initListItem) Title() string       { return i.name }
func (i initListItem) Description() string { return i.description }
func (i initListItem) FilterValue() string { return i.name }
