package cli

import (
	"context"
	"fmt"
	"math"
	"net/url"
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
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/scaffold"
	"github.com/inngest/inngestgo"
	humancron "github.com/lnquy/cron"
	cron "github.com/robfig/cron/v3"
	"golang.org/x/term"
)

const (
	stateAskName     = "state-name"
	stateAskRuntime  = "state-type"
	stateAskTrigger  = "state-trigger"
	stateAskEvent    = "state-event"
	stateAskCron     = "state-cron"
	stateAskURL      = "state-url"
	stateAskLanguage = "state-language"
	stateAskScaffold = "state-scaffold"

	// stateDone is triggered as soon as we're complete and are quitting the walkthrough.
	stateDone = "done"
	// stateQuit is used when terminating the walkthrough early
	stateQuit = "quit"

	eventPlaceholder = "What event name triggers this function?  Use your own event name or an event from an integration."
	cronPlaceholder  = "Specify the cron schedule for the function, eg. '0 * * * *' for every hour."
	urlPlaceholder   = "What's the URL we should call?"

	// anotherLanguage is the list item which is rendered at the bottom for a user
	// to select if we have no scaffolds for their language.
	anotherLanguage = "another language"

	runtimeHTTP   = "Call a URL"
	runtimeDocker = "New function"

	triggerTypeEvent     = "Event based"
	triggerTypeScheduled = "Scheduled"
)

type InitOpts struct {
	ShowWelcome bool

	// Event represents a pre-defined event name to use as the trigger.
	Event string
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
		transitions: 1,

		event: o.Event,
	}

	if o.Event != "" {
		f.event = o.Event
		f.triggerType = triggerTypeEvent
	}

	listHeight := height - f.eventBrowserOffset()

	f.triggerList = list.New([]list.Item{
		initListItem{
			name:        triggerTypeEvent,
			description: "Called every time a specific event is received",
		},
		initListItem{
			name:        triggerTypeScheduled,
			description: "Called automatically on a schedule",
		},
	}, list.NewDefaultDelegate(), width, listHeight)
	f.runtimeList = list.New([]list.Item{
		initListItem{
			name:        "New function",
			description: "Write a new function to be called",
		},
		initListItem{
			name:        runtimeHTTP,
			description: "Call an existing HTTP API as the function",
		},
	}, list.NewDefaultDelegate(), width, listHeight)

	f.languageList = list.New([]list.Item{}, languageDelegate, width, listHeight)
	f.scaffoldList = list.New([]list.Item{}, list.NewDefaultDelegate(), width, listHeight)

	f.textinput.Focus()
	f.textinput.CharLimit = 156
	f.textinput.Width = width
	f.textinput.Prompt = "→  "

	f.loading.Spinner = spinner.Dot
	f.loading.Style = lipgloss.NewStyle().Foreground(Primary)

	f.languageList.SetShowFilter(false)
	f.languageList.SetShowHelp(false)
	f.languageList.SetShowStatusBar(false)
	f.languageList.SetShowTitle(false)

	hideListChrome(&f.triggerList, &f.runtimeList, &f.languageList)

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

	// transitions records the number of questions we've aksed
	transitions int

	// triggerType is the type of trigger we're using, ie. cron or event.
	triggerType string
	// runtimeType is the type of function we're creating - either an HTTP call
	// or a new serverless function.
	runtimeType string
	// url is the URL to hit, if the runtime is HTTP.
	url string
	// name is the function name entered from the user
	name string
	// event is the event entered from the user
	event string
	// cron is the cron schedule if this is a scheduled function
	cron string
	// language is the language selected by the user
	language string
	// scaffold is the scaffold selected by the user
	scaffold *scaffold.Template

	events          []client.Event
	eventFetchError error

	humanCron string
	cronError error
	nextCron  time.Time

	urlError error

	// scaffolds are all scaffolds we have available, parsed after scaffolds
	// have updated.
	scaffolds *scaffold.Mapping
	// scaffoldCacheError is filled if pulling the cache of scaffolds fails.
	// this lets us render a warning if updating wasnt successful.
	scaffoldCacheError error
	// scaffoldDone records whether we have finished updating scaffolds.  this
	// lets us render spinners.
	scaffoldDone bool

	// these are models used to render helpers and sub-components.
	textinput    textinput.Model
	browser      *EventBrowser
	triggerList  list.Model
	runtimeList  list.Model
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
func (f *initModel) Function(ctx context.Context) (*function.Function, error) {
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

	switch f.triggerType {
	case triggerTypeEvent:
		fn.Triggers = []function.Trigger{
			{EventTrigger: &function.EventTrigger{Event: f.event, Definition: ed}},
		}
	case triggerTypeScheduled:
		fn.Triggers = []function.Trigger{
			{CronTrigger: &function.CronTrigger{Cron: f.cron}},
		}
	default:
		return nil, fmt.Errorf("Unknown trigger type: %s", f.triggerType)
	}

	// If this is an HTTP function, set the runtime.
	if f.runtimeType == runtimeHTTP {
		fn.Steps[function.DefaultStepName] = function.Step{
			ID:   function.DefaultStepName,
			Name: fn.Name,
			Path: function.DefaultStepPath,
			Runtime: inngest.RuntimeWrapper{
				Runtime: inngest.RuntimeHTTP{
					URL: f.url,
				},
			},
		}
	} else {
		fn.Steps[function.DefaultStepName] = function.Step{
			ID:   function.DefaultStepName,
			Name: fn.Name,
			Path: function.DefaultStepPath,
			Runtime: inngest.RuntimeWrapper{
				Runtime: inngest.RuntimeDocker{},
			},
		}
	}

	return fn, fn.Validate(ctx)
}

func (f *initModel) Template() *scaffold.Template {
	return f.scaffold
}

func (f *initModel) TelEvent() inngestgo.Event {
	return inngestgo.Event{
		Name: "cli/fn.initialized",
		Data: map[string]interface{}{
			"trigger":  f.triggerType,
			"event":    f.event,
			"cron":     f.cron,
			"runtime":  f.runtimeType,
			"language": f.language,
		},
		Timestamp: inngestgo.Now(),
		Version:   "2022-06-21",
	}
}

// the Y offset when rendering the event browser.
func (f *initModel) eventBrowserOffset() int {
	if f.showWelcome && f.height > 35 {
		return 25
	}
	return 17
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
			return f.events
		},
		func() tea.Msg {
			f.scaffoldCacheError = scaffold.UpdateCache(context.Background())
			f.scaffoldDone = true
			// This will be sent through to the Update function, which will add each
			// language to the languageList subcomponent ensuring its visible in the UI.
			return f.scaffolds
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

	if f.state == stateDone {
		// Ensure we always quit if someone forgot to return tea.Quit when updating
		// the state.
		return f, tea.Quit
	}

	// Globals.  These always run whenever changes happen.
	switch msg := msg.(type) {
	case *scaffold.Mapping:
		// The languages which are available to the scaffold have been updated.
		// Set the list items here, once.
		languages := []list.Item{}
		for k := range f.scaffolds.Languages {
			languages = append(languages, initListItem{name: k})
		}
		languages = append(languages, initListItem{name: anotherLanguage})
		f.languageList.SetItems(languages)

	case []client.Event:
		// We have received the events that are available for the current user. Set
		// the events in the event browser.
		f.browser.SetEvents(f.events)

	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		f.browser.UpdateSize(f.width, f.height-f.eventBrowserOffset()-2)
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

	originalState := f.state
	// Run the update events for each state.
	_, cmd = func() (tea.Model, tea.Cmd) {
		switch f.state {
		case stateAskName:
			return f.updateName(msg)
		case stateAskTrigger:
			return f.updateTrigger(msg)
		case stateAskRuntime:
			return f.updateRuntime(msg)
		case stateAskEvent:
			return f.updateEvent(msg)
		case stateAskCron:
			return f.updateCron(msg)
		case stateAskLanguage:
			return f.updateLanguage(msg)
		case stateAskURL:
			return f.updateURL(msg)
		}
		return f, nil
	}()
	if f.state != originalState {
		f.transitions++
	}

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

		// NOTE: Here we must check if we already have an event specified.  It's valid
		// to start the TUI with an --event flag, predefining this trigger name.
		if f.triggerType == "" {
			f.state = stateAskTrigger
		} else {
			f.state = stateAskRuntime
			// We're skipping two questions.  This isn't that nice.
			f.transitions += 2
		}
	}

	return f, cmd
}

func (f *initModel) updateCron(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	f.textinput.Placeholder = cronPlaceholder
	f.cron = f.textinput.Value()
	f.textinput, cmd = f.textinput.Update(msg)

	schedule, err := cron.
		NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).
		Parse(f.cron)
	if err != nil {
		f.cronError = fmt.Errorf("This isn't a valid cron schedule")
		f.humanCron = ""
		f.nextCron = time.Time{}
	} else {
		f.cronError = nil
		if desc, err := humancron.NewDescriptor(); err == nil {
			f.humanCron, _ = desc.ToDescription(f.cron, humancron.Locale_en)
		}
		f.nextCron = schedule.Next(time.Now())
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && f.cron != "" && f.cronError == nil {
		f.textinput.Placeholder = cronPlaceholder
		f.textinput.SetValue("")
		f.state = stateAskRuntime
	}

	return f, cmd
}

func (f *initModel) updateURL(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	f.textinput.Placeholder = urlPlaceholder
	f.url = f.textinput.Value()
	f.textinput, cmd = f.textinput.Update(msg)

	parsed, err := url.Parse(f.url)
	if err != nil || parsed.Host == "" {
		f.urlError = fmt.Errorf("This isn't a valid URL")
	} else {
		f.urlError = nil
	}

	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter && f.name != "" && f.cronError == nil {
		f.textinput.Placeholder = urlPlaceholder
		f.textinput.SetValue("")
		f.state = stateDone
		return f, tea.Quit
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

			// We have scaffolds with languages available, so move to the languages question.
			f.state = stateAskRuntime
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

func (f *initModel) updateTrigger(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// We only care about enter keypresses to select an item in the list.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
		// We've selected a trigger..
		f.triggerType = f.triggerList.SelectedItem().FilterValue()

		// Depending on the trigger, we're going to ask for the event type
		// or the schedule.
		switch f.triggerType {
		case triggerTypeEvent:
			f.textinput.Placeholder = eventPlaceholder
			f.state = stateAskEvent
			f.textinput.SetValue("")
		case triggerTypeScheduled:
			f.textinput.Placeholder = cronPlaceholder
			f.state = stateAskCron
			f.textinput.SetValue("")
		}
		return f, nil
	}

	f.triggerList, cmd = f.triggerList.Update(msg)
	cmds = append(cmds, cmd)
	return f, tea.Batch(cmds...)
}

func (f *initModel) updateRuntime(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// We only care about enter keypresses to select an item in the list.
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {

		f.runtimeType = f.runtimeList.SelectedItem().FilterValue()
		f.state = stateAskTrigger

		switch f.runtimeType {
		case runtimeHTTP:
			// Ask for the URL we're hitting.
			f.textinput.Placeholder = urlPlaceholder
			f.textinput.SetValue("")
			f.state = stateAskURL
		case runtimeDocker:
			// If we've attempted to update the scaffolds but have zero languages available,
			// quit early.
			if f.scaffolds == nil || (f.scaffoldDone && len(f.scaffolds.Languages) == 0) {
				f.state = stateDone
				return f, tea.Quit
			}
			f.state = stateAskLanguage
		}

		return f, nil
	}

	f.runtimeList, cmd = f.runtimeList.Update(msg)
	cmds = append(cmds, cmd)
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
	case stateAskTrigger:
		b.WriteString(f.renderTrigger())
	case stateAskRuntime:
		b.WriteString(f.renderRuntime())
	case stateAskCron:
		b.WriteString(f.renderCron())
	case stateAskEvent:
		b.WriteString(f.renderEvent())
	case stateAskLanguage:
		b.WriteString(f.renderLanguageSelection())
	case stateAskURL:
		b.WriteString(f.renderURL())
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
	n := 1
	write := func(s string) {
		b.WriteString(fmt.Sprintf("%d. %s", n, s))
		n++
	}

	write("Function name: " + BoldStyle.Render(f.name) + "\n")

	if f.triggerType != "" {
		write("Function trigger: " + BoldStyle.Render(f.triggerType) + "\n")
	}
	if f.cron != "" && f.state != stateAskCron {
		write("Cron schedule: " + BoldStyle.Render(f.cron) + " (" + f.humanCron + ")\n")
	}
	if f.runtimeType != "" {
		write("Function type: " + BoldStyle.Render(f.runtimeType) + "\n")
	}
	if f.event != "" && f.state != stateAskEvent {
		write(fmt.Sprintf("Event trigger: %s\n", f.event))
	}
	if f.language != "" && f.state != stateAskLanguage {
		write(fmt.Sprintf("Language: %s\n", f.language))
	}

	return b.String()
}

// renderName renders the question which asks for the function name.
func (f *initModel) renderName() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render(fmt.Sprintf("%d. Function name:", f.transitions)) + "\n")
	b.WriteString(f.textinput.View())
	return b.String()
}

// renderTrigger renders the trigger question, allowing users to select whether they want
// the function to run on a schedule or be event based.
func (f *initModel) renderTrigger() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render(fmt.Sprintf("%d. How should the function run?", f.transitions)) + "\n\n")
	b.WriteString(f.triggerList.View())
	return b.String()
}

func (f *initModel) renderRuntime() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render(fmt.Sprintf("%d. Do you want to write a new function or call a URL?", f.transitions)) + "\n\n")
	b.WriteString(f.runtimeList.View())
	return b.String()
}

func (f *initModel) renderCron() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render(fmt.Sprintf("%d. Cron schedule:", f.transitions)) + "\n")
	b.WriteString(f.textinput.View())
	if f.cronError != nil {
		b.WriteString("\n")
		b.WriteString(RenderWarning(f.cronError.Error()))
	}
	if !f.nextCron.IsZero() {
		b.WriteString("\n")
		dur := humanizeDuration(time.Until(f.nextCron))

		if f.humanCron != "" {
			b.WriteString(TextStyle.Copy().Foreground(Feint).Bold(true).Render(f.humanCron) + ". ")
		}
		b.WriteString(TextStyle.Copy().Foreground(Feint).Render("This would next run at: " + f.nextCron.Format(time.RFC3339) + " (in " + dur + ")\n"))
	}
	return b.String()
}

func (f *initModel) renderURL() string {
	b := &strings.Builder{}
	b.WriteString(BoldStyle.Render(fmt.Sprintf("%d. URL to call:", f.transitions)) + "\n")
	b.WriteString(f.textinput.View())
	if f.urlError != nil {
		b.WriteString("\n")
		b.WriteString(RenderWarning(f.urlError.Error()))
	}
	return b.String()
}

// renderEvent renders the question wihch asks for the event trigger, plus the
// event browser.
func (f *initModel) renderEvent() string {
	b := &strings.Builder{}

	b.WriteString(
		BoldStyle.Render(fmt.Sprintf("%d. Event trigger:", f.transitions)) +
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
	b.WriteString(
		BoldStyle.Render(fmt.Sprintf("%d. Which language would you like to use?", f.transitions)) +
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
	b.WriteString("\n")
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
	if w, err := clistate.Workspace(ctx); err == nil {
		workspaceID = &w.ID
	}

	c := clistate.Client(ctx)
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

func hideListChrome(lists ...*list.Model) {
	for _, l := range lists {
		l.SetShowFilter(false)
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		l.SetShowTitle(false)
	}
}

func humanizeDuration(duration time.Duration) string {
	days := int64(duration.Hours() / 24)
	hours := int64(math.Mod(duration.Hours(), 24))
	minutes := int64(math.Mod(duration.Minutes(), 60))
	seconds := int64(math.Mod(duration.Seconds(), 60))

	chunks := []struct {
		singularName string
		amount       int64
	}{
		{"day", days},
		{"hour", hours},
		{"minute", minutes},
		{"second", seconds},
	}

	parts := []string{}

	for _, chunk := range chunks {
		switch chunk.amount {
		case 0:
			continue
		case 1:
			parts = append(parts, fmt.Sprintf("%d %s", chunk.amount, chunk.singularName))
		default:
			parts = append(parts, fmt.Sprintf("%d %ss", chunk.amount, chunk.singularName))
		}
	}

	return strings.Join(parts, " ")
}
