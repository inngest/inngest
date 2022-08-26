package cli

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/go-multierror"
	"github.com/inngest/inngest/pkg/execution/driver/dockerdriver"
	"github.com/inngest/inngest/pkg/logger"
	"github.com/spf13/viper"
)

var (
	tickDelay    = 10 * time.Millisecond
	warningDelay = 8 * time.Second
)

// BuilderUIOpts configure the builder when creating Docker images for one or
// more functions.
type BuilderUIOpts struct {
	BuildOpts      []dockerdriver.BuildOpts
	QuitOnComplete bool
}

// NewBuilder renders UI for building an image.
func NewBuilder(ctx context.Context, opts BuilderUIOpts) (*BuilderUI, error) {
	p := progress.New(progress.WithDefaultGradient())

	instances := []*instance{}
	for _, opt := range opts.BuildOpts {
		b, err := dockerdriver.NewBuilder(ctx, opt)
		if err != nil {
			// If we're returning an error, err on the side of caution and assume that
			// the user may not have Docker installed.
			//
			// Provide some simple hints to point to user towards installing Docker,
			// and ideally our justification for requiring it.
			fmt.Println("\n" + RenderWarning("Looks like there was a problem loading Docker.\n\nDocker is a required dependency for the CLI in order to build, run, and deploy your functions.\n\nYou can find more information on installing Docker on your system here: https://docs.docker.com/get-docker/."))

			return nil, err
		}
		instances = append(instances, &instance{
			builder: b,
		})
	}

	return &BuilderUI{
		opts:      opts,
		Instances: instances,
		progress:  p,
	}, nil
}

func (b *BuilderUI) Start(ctx context.Context) error {
	// Depending on the type of output here, we want to either
	// create an interactive UI for building images or show JSON output
	// as we build.
	if viper.GetBool("json") {
		return b.StartJSON(ctx)
	}
	if err := tea.NewProgram(b).Start(); err != nil {
		return err
	}
	return nil
}

func (b *BuilderUI) StartJSON(ctx context.Context) error {
	wg := &sync.WaitGroup{}
	var builderr error

	l := logger.Default().With().Str("caller", "builder").Logger()

	for _, opt := range b.opts.BuildOpts {
		b, err := dockerdriver.NewBuilder(ctx, opt)
		if err != nil {
			return err
		}
		wg.Add(1)
		go func(opt dockerdriver.BuildOpts) {
			l.Info().Interface("opts", opt).Msg("building image")
			err := b.Run()
			wg.Done()
			if err != nil {
				l.Error().
					Interface("opts", opt).
					Err(err).
					Msg("error building image")
				builderr = multierror.Append(builderr, err)
				return
			}
			l.Info().Interface("opts", opt).Msg("successfully built image")
		}(opt)
	}

	wg.Wait()
	return builderr
}

// instance represents a single builder instance used to compile a single
// step in a function.
type instance struct {
	builder *dockerdriver.Builder
	err     error
}

type BuilderUI struct {
	opts BuilderUIOpts

	// Instances represents single build instances for each function being built.
	Instances []*instance

	// progress is the top-level progress to show for the entire build system.
	progress progress.Model

	warning string
}

func (b *BuilderUI) Error() error {
	var err error
	for _, i := range b.Instances {
		if i.err != nil {
			err = multierror.Append(err, i.err)
		}
		if i.builder.Error() != nil {
			err = multierror.Append(err, i.builder.Error())
		}
	}
	return err
}

func (b *BuilderUI) Done() bool {
	for _, i := range b.Instances {
		if !i.builder.Done() {
			return false
		}
	}
	return true
}

func (b *BuilderUI) Init() tea.Cmd {
	for _, fn := range b.Instances {
		fn.err = fn.builder.Start()
	}
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

	if b.Done() && b.opts.QuitOnComplete {
		cmds = append(cmds, tea.Quit)
	}

	return b, tea.Batch(cmds...)
}

func (b *BuilderUI) tick(t time.Time) tea.Msg {
	if len(b.Instances) == 0 {
		return nil
	}

	// Get the earliest start time.
	taken := time.Since(b.Instances[0].builder.StartAt)

	if taken > warningDelay && b.Instances[0].builder.Progress() == 0 {
		b.warning = "This is taking some time.  Do you have internet?"
	}

	if taken > warningDelay*2 && b.Instances[0].builder.Progress() == 0 {
		b.warning = "Like, a really long time :("
	}

	if taken > warningDelay*4 && b.Instances[0].builder.Progress() == 0 {
		b.warning = "We need internet to pull image metadata.  Sorry, but it's not working now."
	}

	if len(b.Instances) == 1 {
		return progressMsg(b.Instances[0].builder.Progress())
	}

	// This just ticks - we don't store the progress here, instead we capture the
	// progress directly in View using `ViewAs`.
	return progressMsg(0)

}

func (b *BuilderUI) View() string {
	if len(b.Instances) == 0 {
		return ""
	}
	// If we're rendering a single instance, use the progress directly from
	// the docker builder.
	if len(b.Instances) == 1 {
		return b.renderSingleBuild()
	}

	// Return how many steps we're building.

	s := &strings.Builder{}

	s.WriteString(FeintStyle.Render(fmt.Sprintf("Building %d steps", len(b.Instances))) + "\n")
	output := ""

	// To calculate the overall progress, we're going to add up the total number of steps
	// for each build running vs the number of steps complete.
	complete := float64(0)
	for _, i := range b.Instances {
		complete += i.builder.Progress()
	}
	progress := complete / (float64(len(b.Instances)) * 100.0)

	header := lipgloss.Place(
		50, 2,
		lipgloss.Left, lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Top,
			b.progress.ViewAs(progress),
			output,
		),
	)
	s.WriteString(header)
	s.WriteString("\n")

	// For each step, add the current progress.
	for n, i := range b.Instances {
		text := i.builder.ProgressText()
		if i.builder.Done() {
			text = "Build complete"
		}
		s.WriteString(
			FeintStyle.Render(fmt.Sprintf("Step %d: %s", n+1, text)) + "\n",
		)
	}

	if b.warning != "" {
		s.WriteString("\n")
		s.WriteString(TextStyle.Copy().Foreground(Orange).Render(b.warning))
	}

	return lipgloss.NewStyle().Padding(1, 0).Render(s.String())
}

func (b *BuilderUI) renderSingleBuild() string {
	instance := b.Instances[0]

	s := &strings.Builder{}

	s.WriteString(FeintStyle.Render("Building a single step") + "\n")

	output := instance.builder.Output(1)
	if err := instance.builder.Error(); err != nil {
		output = "\n" + RenderError(err.Error())
	} else {
		output = TextStyle.Copy().Foreground(Feint).Render(output)
	}

	header := lipgloss.Place(
		50, 3,
		lipgloss.Left, lipgloss.Center,
		lipgloss.JoinVertical(
			lipgloss.Top,
			b.progress.ViewAs(instance.builder.Progress()/100),
			FeintStyle.Render(instance.builder.ProgressText()),
			output,
		),
	)

	s.WriteString(header)

	if b.warning != "" {
		s.WriteString("\n")
		s.WriteString(TextStyle.Copy().Foreground(Orange).Render(b.warning))
	}

	return lipgloss.NewStyle().Padding(1, 0).Render(s.String())
}
