package cli

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngestctl/pkg/docker"
)

var (
	tickDelay    = 10 * time.Millisecond
	warningDelay = 8 * time.Second
)

// NewBuilder renders UI for building an image.
func NewBuilder(ctx context.Context, opts docker.BuildOpts) (*BuilderUI, error) {
	p := progress.New(progress.WithDefaultGradient())
	b, err := docker.NewBuilder(ctx, opts)
	return &BuilderUI{
		builder:  b,
		progress: p,
	}, err
}

type BuilderUI struct {
	builder  *docker.Builder
	buildErr error

	done bool

	// warning is shown if the build takes a long time, or it takes a while
	// to progress from 0
	warning string
	start   time.Time

	progress progress.Model
}

func (b *BuilderUI) Init() tea.Cmd {

	// Start the build.
	b.buildErr = b.builder.Start()
	b.start = time.Now()

	go func() {
		b.builder.Wait()
		<-time.After(tickDelay * 2)
		b.done = true
	}()

	return tea.Tick(tickDelay, b.tick)
}

type progressMsg float64

func (b *BuilderUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlBackslash:
			return b, tea.Quit
		}
	case progressMsg:
		cmds = append(cmds, tea.Tick(tickDelay, b.tick))
	}

	m, cmd := b.progress.Update(msg)
	b.progress = m.(progress.Model)
	cmds = append(cmds, cmd)

	if b.done == true {
		cmds = append(cmds, tea.Quit)
	}

	return b, tea.Batch(cmds...)
}

func (b *BuilderUI) tick(t time.Time) tea.Msg {
	taken := time.Now().Sub(b.start)

	if taken > warningDelay && b.builder.Progress() == 0 {
		b.warning = "This is taking some time.  Do you have internet?"
	}

	if taken > warningDelay*2 && b.builder.Progress() == 0 {
		b.warning = "Like, a really long time :("
	}

	if taken > warningDelay*4 && b.builder.Progress() == 0 {
		b.warning = "We need internet to pull image metadata.  Sorry, but it's not working now."
	}

	return progressMsg(b.builder.Progress())

}

func (b *BuilderUI) View() string {
	if b.buildErr != nil {
		return RenderError(b.buildErr.Error())
	}

	s := &strings.Builder{}

	header := lipgloss.Place(
		50, 4,
		lipgloss.Left, lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Top,
			b.progress.ViewAs(b.builder.Progress()),
			TextStyle.Copy().Foreground(Feint).Render(b.builder.ProgressText()),
		),
	)

	s.WriteString(header)

	if b.warning != "" {
		s.WriteString("\n")
		s.WriteString(TextStyle.Copy().Foreground(Orange).Render(b.warning))
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(s.String())
}
