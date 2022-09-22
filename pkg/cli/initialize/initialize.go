package initialize

import (
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngest/inngest"
	"github.com/inngest/inngest/inngest/client"
	"github.com/inngest/inngest/inngest/clistate"
	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/function"
	"github.com/inngest/inngest/pkg/scaffold"
	"github.com/inngest/inngestgo"
	"github.com/tidwall/sjson"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

var rootQuestion = questionName

const (
	// stateQuestions is used when rendering questions
	stateQuestions = "questions"
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

	// Cron represents the pre-defined cron schedule to use
	Cron string

	// Name represents a pre-defined function name.
	Name string

	// Language represents a pre-defined language to use within the scaffold.
	Language string

	// URL represents a pre-defined URL to hit
	URL string

	// Template represents the GitHub-hosted template to clone
	Template string
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
		events:      nil,
		textinput:   textinput.New(),
		loading:     spinner.New(),
		transitions: 1,

		question: rootQuestion,

		event:    o.Event,
		cron:     o.Cron,
		name:     o.Name,
		language: o.Language,
		url:      o.URL,
		state:    stateQuestions,
		template: o.Template,
	}

	if o.Cron != "" {
		f.triggerType = triggerTypeScheduled
	}
	if o.Event != "" {
		f.triggerType = triggerTypeEvent
	}
	if o.URL != "" {
		f.runtimeType = runtimeHTTP
	}
	if o.Language != "" {
		f.runtimeType = runtimeDocker
		// We can't be done here as we need the scaffolds to update.
		// This is handled during Update();  once scaffolds have loaded
		// we'll call quit and move the state to done.
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
	f.loading.Style = lipgloss.NewStyle().Foreground(cli.Primary)

	f.languageList.SetShowFilter(false)
	f.languageList.SetShowHelp(false)
	f.languageList.SetShowStatusBar(false)
	f.languageList.SetShowTitle(false)

	hideListChrome(&f.triggerList, &f.runtimeList, &f.languageList)

	return f, nil
}

// cloneSucceeded is a tea.Msg sent when a template clone succeeds.
type cloneSucceeded bool

// cloneError is a tea.Msg sent when a template clone fails.
type cloneError struct{ err error }

// initModel represehts the survey state when creating a new function.
type initModel struct {
	// The width of the terminal.  Necessary for styling content such
	// as the welcome message, the evnet browser, etc.
	width  int
	height int

	// whether to show the welcome message.
	showWelcome bool

	// Whether we're asking questions, done, or cancelled.
	state string

	// question represents the current question we're asking.
	question InitQuestion

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
	// eventFilter is the event entered from the user, but
	// not yet confirmed via an enter key.
	eventFilter string
	// event is the event trigger confirmed via the user
	event string
	// cron is the cron schedule if this is a scheduled function
	cron string
	// language is the language selected by the user
	language string
	// scaffold is the scaffold selected by the user
	scaffold *scaffold.Template
	// template is the remote GitHub-hosted template to use
	template        string
	cloningTemplate bool
	clonedTemplate  bool
	CloneError      error

	events          *cli.EventList
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
	browser      *cli.EventBrowser
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
	// If this is a template, load the function from the created template
	if f.shouldCloneTemplate() {
		if f.CloneError != nil {
			return nil, f.CloneError
		}

		if !f.clonedTemplate {
			return nil, fmt.Errorf("template not yet cloned")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}

		targetDir := filepath.Join(cwd, f.name)

		fn, err := function.Load(ctx, targetDir)
		if err != nil {
			return nil, err
		}

		return fn, fn.Validate(ctx)
	}

	// Attempt to find the schema that matches this event, and dump the
	// cue schema inside the function.
	var ed *function.EventDefinition

	if f.events != nil {
		if evt, ok := f.events.All()[f.event]; ok && len(evt.Event.Versions) > 0 {
			ed = &function.EventDefinition{
				Format: function.FormatCue,
				Synced: true,
				Def:    evt.Event.Versions[0].CueType,
			}
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
			Runtime: &inngest.RuntimeWrapper{
				Runtime: inngest.RuntimeHTTP{
					URL: f.url,
				},
			},
		}
	} else {
		fn.Steps[function.DefaultStepName] = function.Step{
			ID:      function.DefaultStepName,
			Name:    fn.Name,
			Path:    function.DefaultStepPath,
			Runtime: function.DefaultRuntime(),
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
	f.browser, _ = cli.NewEventBrowser(f.width, f.height-f.eventBrowserOffset(), f.events, true)

	// Ensure we're asking the correct question on initialization by iterating
	// through all questions that are already answered then progressing to the
	// next.
	for f.question != nil && f.question.Answered(f) {
		f.question = f.question.Next(f)
	}

	return tea.Batch(
		f.loading.Tick,
		func() tea.Msg {
			schemas, err := FetchEvents()
			if err != nil {
				f.eventFetchError = err
			}
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
	f.loading, cmd = f.loading.Update(msg)
	cmds = append(cmds, cmd)

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

		// If we already have a language selected, ensure that we select
		// this specific scaffold.
		//
		// If we've attempted to update the scaffolds but have zero languages available,
		// quit early.
		if f.language != "" && len(f.scaffolds.Languages[f.language]) == 1 {
			f.scaffold = &f.scaffolds.Languages[f.language][0]
		}

	case *cli.EventList:
		// We have received the events that are available for the current user. Set
		// the events in the event browser.
		f.browser.SetEvents(f.events)

	case cloneSucceeded:
		f.clonedTemplate = true
		f.state = stateDone
		return f, tea.Quit

	case cloneError:
		f.CloneError = msg.err
		f.state = stateDone
		return f, tea.Quit

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

		if msg.String() == "q" && f.language != "" {
			// If the user pressed "q" when waiting for scaffolds to be
			// loaded, quit and make the base inngest.json file.
			f.state = stateDone
			return f, tea.Quit
		}
	}

	// Ensure we update the current question.
	if f.question != nil {
		_, cmd = f.question.Update(f, msg)
		cmds = append(cmds, cmd)
	}

	// We may be skipping more than one question (eg. if the language is specified,
	// we'll skip the trigger question and language).  Skip all if possible.
	for f.question != nil && f.question.Answered(f) {
		f.question = f.question.Next(f)
	}

	// If we're ready to clone template, trigger it.
	if f.shouldCloneTemplate() && !f.cloningTemplate {
		f.cloningTemplate = true
		cmds = append(cmds, f.cloneTemplate(context.Background()))
	}

	// This is a separate if, as we want to capture the next question from
	// the above transition.
	//
	// We only quit if scaffolds have updated, which ensures that we've
	// selected the correct scaffold for our language.
	_, fErr := f.Function(context.Background())
	if f.question == nil && f.scaffoldDone && fErr == nil {
		f.state = stateDone
		return f, tea.Quit
	}

	// Merge the async commands from each state into the top-level commands to run.
	cmds = append(cmds, cmd)

	// Return our updated state and all commands to run.
	return f, tea.Batch(cmds...)
}

// View is called to render the CLI's UI.
func (f *initModel) View() string {

	b := &strings.Builder{}

	if f.height > 35 {
		b.WriteString(f.renderIntro(f.showWelcome))
	}

	// For each answered question, render the answered state.
	var q InitQuestion

	q = rootQuestion
	for q != nil && q.Answered(f) {
		b.WriteString(q.Render(f))
		q = q.Next(f)
	}

	// Render loading if we're waiting for a clone to complete
	if f.cloningTemplate {
		b.WriteString(f.renderCloningTemplate())
	}

	// If we have no workflow name, ask for it.
	if f.state == stateQuestions && f.question != nil {
		b.WriteString(f.question.Render(f))
	} else {
		// Done.  Add some padding to the final view for the parent.
		b.WriteString("\n")
	}

	return b.String()
}

func (f *initModel) renderIntro(welcome bool) string {
	b := &strings.Builder{}
	if welcome {
		b.WriteString(f.renderWelcome())
	}
	b.WriteString("\n")
	b.WriteString(cli.BoldStyle.Render("Let's get you set up with a new serverless function."))
	b.WriteString("\n")
	b.WriteString(cli.TextStyle.Copy().Foreground(cli.Feint).Render("Answer these questions to get started."))
	b.WriteString("\n\n")
	return b.String()
}

// Returns a string status regarding template cloning progress, success, or
// failure.
func (f *initModel) renderCloningTemplate() string {
	b := &strings.Builder{}

	if f.clonedTemplate {
		b.WriteString(fmt.Sprintf("Cloned template: %s\n", f.template))
	} else if f.CloneError != nil {
		b.WriteString(fmt.Sprintf("Failed to clone template: %s\n", f.template))
	} else {
		b.WriteString(fmt.Sprintf("%s Cloning template...\n", f.loading.View()))
	}

	return b.String()
}

func (f *initModel) renderWelcome() string {
	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cli.Primary).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	dialog := lipgloss.Place(
		f.width, 7,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(cli.TextStyle.Copy().Bold(true).PaddingLeft(3).PaddingRight(4).Render("👋 Welcome to Inngest!")),
		lipgloss.WithWhitespaceChars("⎼⎽"),
		lipgloss.WithWhitespaceForeground(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#333333"}),
	)
	return dialog
}

// Returns a tea.Cmd that will clone the template found in the initModel in to
// a temporary directory, check it's validity, then move it to the target
// directory given the initModel name.
func (f *initModel) cloneTemplate(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		cwd, err := os.Getwd()
		if err != nil {
			return cloneError{err}
		}

		targetPath := filepath.Join(cwd, f.name)

		if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
			return cloneError{fmt.Errorf("%s already exists", targetPath)}
		}

		repo, examplePath, _ := strings.Cut(f.template, "#")
		tmpPath, err := os.MkdirTemp("", "inngest-template-*")
		if err != nil {
			return cloneError{err}
		}

		cloneCmd := exec.Command("git", "clone", "https://"+repo+".git", "--depth", "1", tmpPath)
		if err = cloneCmd.Run(); err != nil {
			return cloneError{err}
		}

		// Now we've cloned, let's ensure the example path we've found is a valid
		// function to ensure people can't just use this to accidentally git clone
		// something that then doesn't work with inngest.
		tmpExamplePath := filepath.Join(tmpPath, examplePath)
		fn, err := function.Load(ctx, tmpExamplePath)
		if err != nil {
			return cloneError{fmt.Errorf("couldn't find valid function; are you sure this is an Inngest function? %s", err)}
		}

		// Set a new unique ID and name for the function
		newFnId, err := function.RandomID()
		if err != nil {
			return cloneError{fmt.Errorf("error creating new function ID: %s", err)}
		}

		fn.ID = newFnId
		fn.Name = f.name

		// Now we've made changes, validate that everything is still fine
		if err = fn.Validate(ctx); err != nil {
			return cloneError{fmt.Errorf("error validating function; are you sure this is an Inngest function? %s", err)}
		}

		// Make changes to the ID and Name in `inngest.json` in the tmp dir.
		// We do this directly (not via `function.MarshalJSON`) as we want to
		// preserve as much as possible of the original config file.
		//
		// If we overwrite using `function.MarshalJSON`, it may alter the structure
		// which may be confusing for users.
		tmpConfigPath := filepath.Join(tmpExamplePath, function.JsonConfigName)
		byt, err := os.ReadFile(tmpConfigPath)
		if err != nil {
			return cloneError{fmt.Errorf("error reading config file: %s", err)}
		}

		val, err := sjson.Set(string(byt), "name", fn.Name)
		if err != nil {
			return cloneError{fmt.Errorf("could not set new function name: %s", err)}
		}

		val, err = sjson.Set(val, "id", fn.ID)
		if err != nil {
			return cloneError{fmt.Errorf("could not set new function ID: %s", err)}
		}

		val, err = sjson.Delete(val, "description")
		if err != nil {
			return cloneError{fmt.Errorf("could not remove description")}
		}

		val, err = sjson.Delete(val, "tags")
		if err != nil {
			return cloneError{fmt.Errorf("could not remove tags")}
		}

		if err := os.WriteFile(filepath.Join(tmpExamplePath, function.JsonConfigName), []byte(val), 0644); err != nil {
			return cloneError{fmt.Errorf("error writing %s: %s", function.JsonConfigName, err)}
		}

		onlyOwnerWrite := 0755
		// Create every directory up-to-but-not-including the target dir
		if err = os.MkdirAll(filepath.Dir(targetPath), fs.FileMode(onlyOwnerWrite)); err != nil {
			return cloneError{err}
		}

		err = os.Rename(filepath.Join(tmpPath, examplePath), targetPath)
		if err != nil {
			return cloneError{fmt.Errorf("Error moving template from temp directory: %s", err)}
		}

		err = os.RemoveAll(tmpPath)
		if err != nil {
			fmt.Println("\n" + cli.RenderWarning(fmt.Sprintf("Failed to remove temporary dir after copy: %s", err)) + "\n")
		}

		fn, err = function.Load(ctx, targetPath)
		if err != nil {
			return cloneError{err}
		}

		if err = fn.Validate(ctx); err != nil {
			return cloneError{err}
		}

		return cloneSucceeded(true)
	}
}

// fetchEvents fetches all public events and events from each workspace.
func FetchEvents() (*cli.EventList, error) {
	ctx, done := context.WithTimeout(context.Background(), 20*time.Second)
	defer done()

	var eg errgroup.Group

	c := clistate.Client(ctx)
	evts := &cli.EventList{}

	// Fetch events from every workspace.
	workspaces, err := clistate.Client(ctx).Workspaces(ctx)
	if err != nil {
		// If this is an unauthorized error we can ignore it.
		if !strings.Contains(err.Error(), "unauthorized") {
			return nil, err
		}
		err = fmt.Errorf("You're not logged in so we're only showing public events")
	}

	eg.Go(func() error {
		// Find all public events.
		loaded, err := c.AllEvents(ctx, &client.EventQuery{
			SchemaSource: &client.SchemaSourceIntegration,
		})
		if err != nil {
			return err
		}
		for _, e := range loaded {
			evts.Add(e, nil)
		}
		return nil
	})

	for _, ws := range workspaces {
		w := ws
		eg.Go(func() error {
			loaded, err := c.AllEvents(ctx, &client.EventQuery{
				WorkspaceID: &w.ID,
			})
			if err != nil {
				return err
			}
			for _, e := range loaded {
				evts.Add(e, &w)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	evts.Sort()
	return evts, err
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

// Returns whether we should be - and can - clone a template based on current
// input.
func (f *initModel) shouldCloneTemplate() bool {
	return f.template != "" && f.name != ""
}
