package stdoutlogger

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/inngest/inngestctl/pkg/cli"
	"github.com/inngest/inngestctl/pkg/event"
	"github.com/inngest/inngestctl/pkg/logger"
)

func NewLogger(o logger.Options) logger.Logger {
	return &stdoutLogger{
		Pretty: o.Pretty,
	}
}

type stdoutLogger struct {
	Pretty bool
}

func (l *stdoutLogger) Log(m logger.Message) {
	category := strings.TrimSpace(fmt.Sprintf("%s %s", m.Object, m.Action))

	prefixColor := map[string]lipgloss.Color{
		"REGISTRY": lipgloss.Color("28"),
		"API":      lipgloss.Color("32"),
		"EVENT":    cli.Fuschia,
		"FUNCTION": cli.Iris,
		"EXECUTOR": cli.Green,
		"ERROR":    cli.Red,
	}

	prefix := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(cli.White).
		Background(prefixColor[m.Object]).
		Bold(true).
		Render(category)
	message := cli.TextStyle.Render(m.Msg)

	context, err := formatContext(m.Context, l.Pretty)
	if err != nil {
		l.Log(logger.Message{
			Object: "ERROR",
			Msg:    err.Error(),
		})
		return
	}

	additional := lipgloss.NewStyle().Foreground(cli.Feint).Render(string(context))

	os.Stdout.Write([]byte(prefix + " " + message + " " + additional + "\n"))
}

func formatContext(context interface{}, pretty bool) (string, error) {
	switch c := context.(type) {
	case string:
		return c, nil
	case *event.Event:
		return marshal(c, pretty)
	}
	return "", nil
}

func marshal(context interface{}, pretty bool) (string, error) {
	if pretty {
		b, err := json.MarshalIndent(context, "  ", "  ")
		return "\n  " + string(b), err
	} else {
		b, err := json.Marshal(context)
		return string(b), err
	}
}
