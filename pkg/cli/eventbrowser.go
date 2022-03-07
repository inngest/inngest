package cli

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/quick"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/event-schemas/events"
)

var listWidth = 50

func NewEventBrowser(width, height int, evts []events.Event) (*EventBrowser, error) {
	delegate := list.NewDefaultDelegate()
	delegate.SetSpacing(1)
	l := list.New([]list.Item{}, delegate, listWidth, height)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	l.KeyMap = listKeyMap() // Remove the J/K keyboard navigation.

	v := viewport.New(width, height)
	v.KeyMap = viewportKeyMap()

	return &EventBrowser{
		width:    width,
		height:   height,
		schemas:  evts,
		list:     l,
		viewport: v,
	}, nil
}

// EventBrowser renders an interactive event browser.  It has two columns:  a left
// column which lists events, and a right content page which shows details for
// the currently selected event.
type EventBrowser struct {
	width  int
	height int

	schemas []events.Event
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
	e.width = width
	e.height = height
	// the list's width is fixed.
	e.list.SetHeight(height)
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
func (e EventBrowser) Selected() *events.Event {
	if e.list.SelectedItem() == nil {
		return nil
	}
	s := e.list.SelectedItem().(listItem)
	return &s.e
}

// UpdatePrefix updates the prefix we use to search and filter events.
func (e *EventBrowser) UpdatePrefix(s string) {
	e.prefix = s
}

// View renders the list.
func (e *EventBrowser) View() string {
	// Filter the events by prefix.
	filtered := e.schemas
	if e.prefix != "" {
		filtered = []events.Event{}
		for _, evt := range e.schemas {
			if strings.HasPrefix(strings.ToLower(evt.Name), strings.ToLower(e.prefix)) {
				filtered = append(filtered, evt)
			}
		}
	}

	// Render the event viewer.
	var selectedEvent *events.Event
	if e.list.SelectedItem() != nil {
		s := e.list.SelectedItem().(listItem)
		selectedEvent = &s.e
	}

	if len(filtered) == 1 && filtered[0].Name == e.prefix {
		// Ensure the item is selected if we have one match.
		e.list.Select(0)
	}

	// If there's no active event we're asking the user to define a new event.
	// Don't render the list & browser;  instead render a message saying we'll match on
	// a new custom event.
	//
	// We use len(filtered) here instead of selectedEvent so that we can show newly
	// filtered events when text is deleted via backspace.
	if e.prefix != "" && len(filtered) == 0 {
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

func (e *EventBrowser) renderList(schemas []events.Event) string {
	items := make([]list.Item, len(schemas))
	for n, evt := range schemas {
		items[n] = listItem{e: evt}
	}
	e.list.SetItems(items)
	left := lipgloss.NewStyle().
		Width(listWidth + 4). // plus padding
		Padding(2).
		Render(e.list.View())
	return left
}

func (e *EventBrowser) renderDetail(selected *events.Event) string {
	// This is the outer box.
	panel := lipgloss.NewStyle().
		Width(e.width - listWidth - 8). // list padding + inner padding
		Height(e.height).
		Padding(2)

	if selected == nil {
		return panel.Render("No event selected")
	}

	// Render the type, using terminal colouring
	buf := &bytes.Buffer{}
	if err := quick.Highlight(buf, selected.Cue, "go", "terminal256", "monokai"); err != nil {
		panic(err)
	}
	e.viewport.SetContent(buf.String())

	return panel.Render(e.viewport.View())
}

// listItem renders an event in the list.
type listItem struct {
	e events.Event
}

func (i listItem) Title() string       { return i.e.Name }
func (i listItem) Description() string { return i.e.Description }
func (i listItem) FilterValue() string { return i.e.Name }

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
