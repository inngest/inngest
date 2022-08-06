package cli

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngest/inngest/client"
)

var (
	listWidth           = 50
	newEventDescription = "A new event you're typing *right now*"
)

type EventList struct {
	l   sync.Mutex
	all map[string]Event

	sorted []Event
}

// Set replaces the stored event entirely
func (e *EventList) Set(evt Event) {
	e.l.Lock()
	defer e.l.Unlock()
	if e.all == nil {
		e.all = map[string]Event{}
	}
	e.all[evt.Event.Name] = evt
	// Horribly inefficient, but re-sort.
	e.Sort()
}

// Add adds an event to the list for the given workspace, upserting and merging
// the available workspace to the existing event list.
func (e *EventList) Add(evt client.Event, workspace *client.Workspace) {
	e.l.Lock()
	defer e.l.Unlock()

	if e.all == nil {
		e.all = map[string]Event{}
	}
	if _, ok := e.all[evt.Name]; !ok {
		e.all[evt.Name] = Event{
			Event: evt,
		}
	}

	found := e.all[evt.Name]
	if workspace == nil {
		found.Public = true
	} else {
		found.Workspaces = append(found.Workspaces, *workspace)
	}
	e.all[evt.Name] = found
}

func (e *EventList) Sort() {
	e.sorted = make([]Event, len(e.all))
	n := 0
	for _, evt := range e.all {
		e.sorted[n] = evt
		n++
	}
	sort.Slice(e.sorted, func(i, j int) bool {
		return e.sorted[i].Event.Name < e.sorted[j].Event.Name
	})
}

func (e *EventList) List() []Event {
	return e.sorted
}

// All returns all events.  This is not thread safe when adding events
// to the list and should be used when all events are loaded.
func (e *EventList) All() map[string]Event {
	return e.all
}

// Event lists the actual event and all workspaces that include this event.
type Event struct {
	// TODO: how do we want to handle multiple versions within the event browser?  plus
	// version discrepancies across workspaces.

	// event stores the client event.
	client.Event
	Public     bool
	Workspaces []client.Workspace
}

func NewEventBrowser(width, height int, evts *EventList, showNewEvent bool) (*EventBrowser, error) {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(1)
	l := list.New([]list.Item{}, delegate, listWidth, height)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.KeyMap = listKeyMap() // Remove the J/K keyboard navigation.

	v := viewport.New(width, height)
	v.KeyMap = viewportKeyMap()

	if evts == nil {
		evts = &EventList{}
	}

	return &EventBrowser{
		width:        width,
		height:       height,
		showNewEvent: showNewEvent,
		schemas:      evts,
		list:         l,
		viewport:     v,
	}, nil
}

// EventBrowser renders an interactive event browser.  It has two columns:  a left
// column which lists events, and a right content page which shows details for
// the currently selected event.
type EventBrowser struct {
	width  int
	height int

	// Whether to show a "new event" if the prefix doesn't match.
	showNewEvent bool

	schemas *EventList
	prefix  string

	// Renders the list on the left.
	list list.Model

	// Renders the detail view.  We use a viewport because the type will extend
	// beyond the height of the screen.
	viewport viewport.Model
}

var _ tea.Model = (*EventBrowser)(nil)

func (e *EventBrowser) Init() tea.Cmd {
	return nil
}

// UpdateSize updates the size of the event browser's rendering area.
func (e *EventBrowser) UpdateSize(width, height int) {
	if width < 100 {
		listWidth = 24
	} else {
		listWidth = 50
	}

	e.width = width
	e.height = height
	e.list.SetHeight(height)
	e.list.SetWidth(listWidth)
	e.viewport.Width = width
	e.viewport.Height = height
}

// Update handles incoming keypresses, mouse moves, resize events etc.
func (e *EventBrowser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	e.list, cmd = e.list.Update(msg)
	cmds = append(cmds, cmd)

	// Handle mouse comamnds in viewport.
	e.viewport, cmd = e.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return e, tea.Batch(cmds...)
}

// Selected returns the selected event via the list, or nil if no event is selected.
func (e EventBrowser) Selected() *Event {
	if e.list.SelectedItem() == nil {
		return nil
	}
	s := e.list.SelectedItem().(eventListItem)
	return &s.e
}

// UpdatePrefix updates the prefix we use to search and filter events.
func (e *EventBrowser) UpdatePrefix(s string) {
	e.prefix = s
}

func (e *EventBrowser) SetEvents(evts *EventList) {
	if evts == nil {
		evts = &EventList{}
	}
	e.schemas = evts
}

// View renders the list.
func (e *EventBrowser) View() string {
	// Filter the events by prefix.
	filtered := e.schemas
	if e.prefix != "" {
		filtered = &EventList{}
		for _, evt := range e.schemas.sorted {
			if strings.HasPrefix(strings.ToLower(evt.Event.Name), strings.ToLower(e.prefix)) {
				filtered.Set(evt)
			}
		}
	}

	// Render the event viewer.
	var selectedEvent *Event
	if e.list.SelectedItem() != nil {
		s := e.list.SelectedItem().(eventListItem)
		selectedEvent = &s.e
	}

	if filtered != nil && len(filtered.sorted) == 1 && filtered.sorted[0].Event.Name == e.prefix {
		// Ensure the item is selected if we have one match.
		e.list.Select(0)
	}

	// If there's no active event we're asking the user to define a new event.
	// Don't render the list & browser;  instead render a message saying we'll match on
	// a new custom event.
	//
	// We use len(filtered) here instead of selectedEvent so that we can show newly
	// filtered events when text is deleted via backspace.
	if e.prefix != "" && len(filtered.sorted) == 0 && !e.showNewEvent {
		msg := TextStyle.Copy().Foreground(Feint).Render("No event matched ")
		msg += BoldStyle.Copy().Render(e.prefix)
		msg += TextStyle.Copy().Foreground(Feint).Render(".  The function will be triggered using this unseen event.")
		return lipgloss.Place(
			e.width, 3,
			lipgloss.Center, lipgloss.Center,
			msg,
		)
	}

	list := e.renderList(filtered)
	detail := e.renderDetail(selectedEvent)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
}

func (e *EventBrowser) renderList(schemas *EventList) string {
	var found bool
	items := make([]list.Item, len(schemas.sorted))
	for n, evt := range schemas.sorted {
		if evt.Event.Name == e.prefix {
			found = true
		}
		items[n] = eventListItem{e: evt}
	}

	// If there's a prefix that doesn't match an event, see if we want
	// to show a "new event" at the top of the list.
	if e.showNewEvent && e.prefix != "" && !found {
		items = append([]list.Item{
			eventListItem{
				e: Event{
					Event: client.Event{
						Name:        e.prefix,
						Description: newEventDescription,
					},
				},
			},
		}, items...)
	}

	e.list.SetItems(items)
	left := lipgloss.NewStyle().
		Width(listWidth+2). // plus padding
		Padding(2, 2, 2, 0).
		Render(e.list.View())
	return left
}

func (e *EventBrowser) renderDetail(selected *Event) string {
	// This is the outer box.
	panel := lipgloss.NewStyle().
		Width(e.width - listWidth - 8). // list padding + inner padding
		Height(e.height).
		Padding(2)

	if selected == nil {
		return panel.Render("No event selected")
	}

	if len(selected.Versions) == 0 {
		return panel.Render("No event type recorded")
	}

	// Render the type, using terminal colouring
	buf := &bytes.Buffer{}
	if err := quick.Highlight(buf, selected.Versions[0].CueType, "go", "terminal256", "monokai"); err != nil {
		panic(err)
	}
	e.viewport.SetContent(buf.String())

	return panel.Render(e.viewport.View())
}

// eventListItem renders an event in the list.
type eventListItem struct {
	e Event
}

func (i eventListItem) Title() string       { return i.e.Name }
func (i eventListItem) FilterValue() string { return i.e.Name }
func (i eventListItem) Description() string {
	if len(i.e.Workspaces) == 0 && i.e.Public {
		return "Via an integration"
	}
	if len(i.e.Workspaces) == 0 {
		return "An unknown event"
	}

	workspaces := []string{}
	for _, w := range i.e.Workspaces {
		if w.Test {
			workspaces = append(workspaces, "test")
		} else {
			workspaces = append(workspaces, "prod")
		}
	}

	sort.Strings(workspaces)
	return fmt.Sprintf("Seen on %s", strings.Join(workspaces, ", "))
}

func listKeyMap() list.KeyMap {
	return list.KeyMap{
		CursorUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		CursorDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "prev page"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next page"),
		),
	}
}

func viewportKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+k", "ctrl+u"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+j", "ctrl+d"),
		),
	}
}
