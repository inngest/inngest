---
heading: "Rapidly building interactive CLIs in Go with Bubbletea"
subtitle: Our product is just different enough to make our CLI require really good interactivity.  We bundle an interactive event browser in our CLI.  Here's how it's built.
image: "/assets/blog/interactive-clis-with-bubbletea.jpg?v=2022-04-28"
date: 2022-04-15
---

In this post we’ll walk through our use of Bubbletea, an elm-inspired TUI interface for Golang. We’ll discuss why we chose it, some example code, and some thoughts. Let’s start with context — what we’re building and why.

### What we’re building

[We recently revamped our CLI](https://github.com/inngest/inngest-cli), making it easier develop, locally test, and [deploy serverless functions](https://www.inngest.com/). Our product is _just_ different enough to make our init experience require _really good interactivity_. You see, with Inngest functions are triggered by events instead of raw HTTP requests. An event is simple: fundamentally it has a `name` and some `data`. The general idea is that:

1. You send us an event
2. We store it for some amount of time (eg. from weeks to years)
3. We instantly trigger your serverless function, using the event as the payload

By using an _event_ instead of directly calling your functions we can do a bunch for you: fully type your payloads, enforce schemas, build audit trails, automatically retry functions, replay with historic events, coordinate between events in step functions... everything that was previously _really_ hard to build becomes simple (and free if you want to use us — sign up here).

Fully typing your payloads is important. It means we can build a _really good dev experience_ by ensuring that all data matches a schema, generating fake data for local testing, etc.

This all requires a solid CLI that walks you through scaffolding a new function, as you have to specify the event trigger ahead of time. This is the main difference to other platforms: instead of jumping straight into the code you **think about the data first**.

**Interactivity in the CLI**

Our CLI bundles an interactive event browser which allows you to specify a bunch of events from common sources, such as Github or Stripe. It also pulls in schemas for every custom event you send via the API. Here’s a demo:

<video controls>
	<source src="/assets/video/blog-bubbletea-interactivity.mp4" type="video/mp4" />
</video>

Building this historically would have been really, mind-numblingly, rage-quit levels of tediousness. It’s definitely possible... we use things like vim, emacs, htop, [or our favourite — btop](https://github.com/aristocratos/btop). But building interactivity in the terminal has never been _nice,_ hence abstraction city. I don’t know anyone that wants to develop with ncurses or termbox, painting character by character. And, if you do, I’m equal parts impressed and scared.

### **What is Bubbletea?**

There are a few Go libraries which make terminal interactivity easy, moving from low level to high level:

- [tcell](https://github.com/gdamore/tcell), which is a termbox like library for writing to terminals. This is fairly low-level; you draw boxes yourself. It’s super flexible, but still quite tedious to write.
- [tview](https://github.com/rivo/tview), a library for writing TUIs. This contains all of the code you need to bang out interactive interfaces with minimal code; it’s high level.
- [Bubbletea](https://github.com/charmbracelet/bubbletea), an elm-like library for terminal interfaces. It allows you to manage your UI state and rendering within _models_. So, state impacts rendering, and you get a reactive loop. Think React or, well, Elm, but using Go and for the terminal.

Bubbletea excels at creating complex TUIs with clean code. It uses similar architectural and mental models to other UI frameworks, which all converge on a reactive flow of message → state → render. It makes sense; it’s easy to understand; and it works.

With Bubbletea, you can be up and running with complex lists, surveys, questions, lists, and buffers in minutes. And it’s _so insanely easy to style_ in comparison to what you might be used to, thanks to the amazing work of [lipgloss](https://github.com/charmbracelet/lipgloss).

Because of its architecture, existing components, and ease of styling, Bubbletea was the clear winner for us to get started. Here’s how we built the event browser:

### Building a TUI-based event browser

Skip ahead: [you can view all of our code in our CLI here](https://github.com/inngest/inngest-cli/blob/main/cmd/commands/init.go). If you’re interested in the conclusions and want to gloss over the step-by-step guide, click here.

First, we need to launch the CLI, often using arguments, flags, etc. In the Go world, that means that you might well be using Steve Francia’s fantastic Cobra library - [https://github.com/spf13/cobra](https://github.com/spf13/cobra).

Let’s scaffold a command which will launch the event browser using Bubbletea:

```go
// NewCmdEventBrowser returns a *cobra.Command which can be added to the root
// command list in main.go
func NewCmdEventBrowser() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Scaffold a new function",
		Example: "inngest init",
		Run:     runBrowser,
	}
	return cmd
}

func runBrowser(cmd *cobra.Command, args []string) {
	// This is where we'll handle launching the event browser when invoked
}
```

It’s pretty easy to tie in Cobra with Bubbletea — we add a Bubbletea specific logic within `runBrowser`, which will be called by Cobra any time the `init` command is invoked. In Inngest, this same logic runs when you run `inngest init`.

Let’s get started with Bubbletea. Remember how Bubbletea uses an Elm-like architecture to render its UI? It renders UI based off of _application state - a Model_. To render anything we need to create a new Model. <b>A _Model_ is a struct that stores some application state</b>:

```go
package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// NewModel is an initializer which creates a new model for rendering
// our Bubbletea app.
func NewModel() (*model, error) {
	return model{}, nil
}

type model struct {
	// nameInput stores the event name we have from the text input component
	nameInput string
	// listinput stores the event name selected from the list, used as an
	// autocomplete.
	listInput string
	// event stores the final selected event.
	event string
}

// Ensure that model fulfils the tea.Model interface at compile time.
//
// This code isn't going to compile until later on when we add the required
// functions - we'll get to that in a second.
var _ tea.Model = (*model)(nil)
```

Now we have our app state with three fields: a struct member for an input field, for a list field (which records the selected list item), and a member which records the final name of the selected event.

That last line at the end of the model struct is useful when scaffolding models to ensure that you fulfill the `tea.Model` interface when there are no other type assertions (eg. passing model into a function which requires a `tea.Model`.

### Rendering a UI

Back at it, let’s start to render our UI. [Bubbletea calls the View function](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Model) of a `tea.Model` to render UI to the CLI. Let’s add one:

```go
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func NewModel() (*model, error) {
	return &model{}, nil
}

type model struct {
	nameInput string
	listInput string
	event     string
}

var _ tea.Model = (*model)(nil)

// View renders output to the CLI.
func (m model) View() string {
	if m.event != "" {
		// We have a final event selected.  Render a message which
		// confirms our selection.
		//
		// Bubbletea will handle writing this to the terminal, so all we need
		// to do is respond with a string.
		return fmt.Sprintf("You've selected: %s", m.event)
	}
	// If we have no final event we can render a text input and list.
	// We'll get to this in a bit, as Bubbletea has pre-made components
	// we can render.
	return "TODO"
}
```

This is the start of _state-dependent rendering_. By inspecting the application state at runtime we can decide what we want to render in the UI. In this case, once we have an event selected we don’t need to render a text input and autocomplete list.

We’re going to need to render some content to type in your desired event or select from a list, but for now let’s get this basic model rendered so we can see our “TODO” item.

A `tea.Model` has two other functions we need to implement to render our app. What happens when a key is pressed, the user clicks, or scrolls? Without handling these there’s no interactivity!

### Interactivity

Each model has an `Update` function which is called via Bubbletea itself. After all, Bubbletea is a framework: it calls us when there are updates. Let’s add the `Update` function so that we can handle keyboard inputs and model updates:

```go
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func NewModel() (*model, error) {
	return &model{}, nil
}

type model struct {
	nameInput string
	listInput string
	event     string
}

var _ tea.Model = (*model)(nil)

func (m *model) View() string {
	if m.event != "" {
		return fmt.Sprintf("You've selected: %s", m.event)
	}
	return "TODO" // We'll do this soon :)
}

// Update is called with a tea.Msg, representing something that happened within
// our application.
//
// This can be things like terminal resizing, keypresses, or custom IO.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Let's figure out what is in tea.Msg, and what we need to do.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// The terminal was resized.  We can access the new size with:
		_, _ = msg.Width, msg.Height
	case tea.KeyMsg:
		// msg is a keypress.  We can handle each key combo uniquely, and update
		// our state:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			// In this case, ctrl+c or ctrl+backslash quits the app by sending a
			// tea.Quit cmd.  This is a Bubbletea builtin which terminates the
			// overall framework which renders our model.
			//
			// Unfortunately, if you don't include this quitting can be, uh,
			// frustrating, as bubbletea catches every key combo by default.
			return m, tea.Quit
		}
	}
	// We return an updated model to Bubbletea for rendering here.  This allows
	// us to mutate state so that Bubbletea can render an updated view.
	//
	// We also return "commands".  A command is something that you need to do
	// after rendering.  Each command produces a tea.Msg which is its *result*.
	// Bubbletea calls this Update function again with the tea.Msg - this is our
	// render loop.
	//
	// For now, we have no commands to run given the message is not a keyboard
	// quit combo.
	return m, nil
}
```

A quick recap: the `Update` function is called with a `tea.Msg`, which can be anything at all. The `tea.Msg` argument represents something that happened to our app. Bubbletea automatically calls this with global events (keypresses, mouse clicks, resizes, etc.). It also calls `Update` any time a `tea.Cmd` generates a new message. This lets applications create their own cycles for interactivity. Bubbletea also [provides some utilities](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Cmd) to work with commands, eg. you can [batch](https://pkg.go.dev/github.com/charmbracelet/bubbletea#Batch) > 1 command together.

### Initialization

Okay, now we have one more function to implement in order to render our model: `Init() tea.Cmd`. This function is called just before the first render — similar to `componentWillMount` in React. It allows you run async logic and return a `tea.Msg` which will be passed into Update() for you to update your model’s state.

As an example, here we could fetch a bunch of events from a registry, return a new message containing the events, and store them in our model’s state. For now, we don’t need to do anything so we can return nil:

```go
package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func NewModel() (model, error) {
	return model{}, nil
}

type model struct {
	nameInput string
	listInput string
	event     string
}

var _ tea.Model = (*model)(nil)

func (m model) View() string {
	if m.event != "" {
		return fmt.Sprintf("You've selected: %s", m.event)
	}
	return "TODO" // We'll do this soon :)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		_, _ = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return m, tea.Quit
		}
	}
	return m, nil
}

// Init() is called to kick off the render cycle.  It allows you to
// perform IO after the app has loaded and rendered once, asynchronously.
// The tea.Cmd can return a tea.Msg which will be passed into Update() in order
// to update the model's state.
func (m model) Init() tea.Cmd {
	// We have nothing to do.  But, you could write a function which eg. calls
	// an HTTP endpoint to load events here, then return those events as a tea.Msg
	// so that our Update() function can store the events.
	return nil
}
```

This is literally _the_ benefit of nil interfaces in Go... which is a can of worms we won’t go into 🙃. On the note of Init, you might ask yourself “why not use a pointer reference to model and update state directly?”. It’s totally a fair question; pointer references mean you can mutate state at-will. However, [Bubbletea only re-renders to the UI after Update calls with messages](https://github.com/charmbracelet/bubbletea/blob/v0.20.0/tea.go#L491-L551). If we did that we wouldn’t be able to guarantee that the output is refreshed to our terminal.

Well — that’s... it! We can hop back to the Cobra entrypoint to render our CLI:

```go
func NewCmdEventBrowser() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Create a new event browser",
		Example: "inngest init",
		Run:     runBrowser,
	}
	return cmd
}

func runBrowser(cmd *cobra.Command, args []string) {
	// Create a new TUI model which will be rendered in Bubbletea.
	state, err := NewModel()
	if err != nil {
		fmt.Println(fmt.Sprintf("Error starting init command: %s\n", err))
		os.Exit(1)
	}
	// tea.NewProgram starts the Bubbletea framework which will render our
	// application using our state.
	if err := tea.NewProgram(state).Start(); err != nil {
		log.Fatal(err)
	}
}
```

It’s pretty basic and only renders “TODO” — but it covers every Bubbletea concept and allows us to build incredibly complex UIs in an easy, manageable way.

### Subcomponents in Bubbletea: adding text inputs

Now, we need to render some subcomponents, such as Bubbletea’s built in [text input](https://github.com/charmbracelet/bubbles/tree/master/textinput) and [list](https://github.com/charmbracelet/bubbles/tree/master/list) components. They contain pre-made models which have their own state — a text input needs to record what’s been typed, whether the cursor is displayed, etc. Each component is also a model, which means they also have a `View()` function which we can call to render it to the terminal, an `Update` function to update its local state, etc.

Let's add a text input to our application model:

```go
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func NewModel() (*model, error) {
	// We need to initialize a new text input model.
	ti := textinput.New()
	ti.CharLimit = 30
	ti.Placeholder = "Type in your event"
	// Nest the text input in our application state.
	return &model{input: ti}, nil
}

type model struct {
	nameInput string
	listInput string
	event     string
	// Add the text input to our main application state.  It's a subcomponent
	// which has its own state, etc.
	input textinput.Model
}

func (m model) Init() tea.Cmd {
	// Call Init() on our submodel.  If we had > 1 submodel and command, we would
	// create a slice of commands to batch:
	//
	// return tea.Batch(cmds...)
	cmd := m.input.Init()
	return cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		_, _ = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return m, tea.Quit
		}
	}
	// We call Bubbletea using our model as the top-level application.  Bubbletea
	// will call Update() in our model only.  It's up to us to call Update() on
	// our text input to update its state.  Without this, typing won't fill out
	// the text box.
	m.input, cmd = m.textinput.Update(msg)
	cmds = append(cmds, cmd)
	// store the text inputs value in our top-level state.
	m.nameInput = m.textinput.Value()
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.event != "" {
		return fmt.Sprintf("You've selected: %s", m.event)
	}

	b := &strings.Builder{}
	b.WriteString("Enter your event:\n")
	// render the text input.  All we need to do to show the full
	// input is call View() and return the string.
	b.WriteString(m.input.View())
	return b.String()
}
```

That’s it! We’ve rendered an interactive text input, and we’re controlling _how_ to render the input. Using Lipgloss, we can build flexbox-style layouts, change fonts, update sizes, etc to make the UI look however we like.

### Thoughts on Bubbletea

After building out our basic UI for creating event-driven serverless functions, we’re pretty impressed. For the first time it feels as if we can create maintainable, good looking TUI applications. It’s super easy to style, and the API for Bubbletea, Lipgloss, and the components (Bubbles) seem well thought out. The code is _much_ cleaner than before, and without the framework it would have taken days or weeks to develop something to the same standard. Not only that — the experience that it gives you as an end user is (hopefully) _great_.

There are a few gotchas, though. For example, having to handle `SIGINT` or `SIGQUIT` key combos yourself from Bubbletea kind of sucks. You could create a parent state wrapper which wraps your own custom State to listen for this key combo, or trap these signals yourself and quit Bubbletea from the outside. It’s also quite cumbersome to set up the variables to batch your `tea.Cmd` responses from Update. Overall, though, these are absolutely insignificant nits in a very clean and productive framework.

If you're interested in the final result and checking out how our CLI works, [you can see the source here](https://github.com/inngest/inngest-cli). We use it all the time when building new async functionality - it lets us build and test new serverless functions literally in under a minute.
