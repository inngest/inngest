package tel

import (
	"context"
	"os"
	"runtime"
	"sync"

	"github.com/inngest/inngest-cli/inngest/state"
	"github.com/inngest/inngest-cli/inngest/version"
	"github.com/inngest/inngestgo"
	"github.com/spf13/cobra"
)

const (
	EventName = "cli/telemetry.created"
	key       = "5_Jx-3FAkDMeddntV-KlZy1sbjY8UU1cqn2viGMPlv9Gq-0tWYaukPkUVbD04Zo-1SO2AF2dwnMv7rcHyhJzVQ"
)

var (
	client = inngestgo.NewClient(key)
	wg     *sync.WaitGroup
)

func init() {
	wg = &sync.WaitGroup{}
}

// Metadata holds telemetry context.
type Metadata struct {
	AccountID  string         `json:"account_id"`
	CLIVersion string         `json:"cli_version"`
	Cmd        string         `json:"cmd"`
	OS         string         `json:"os"` // the OS of the system
	Context    map[string]any `json:"context"`
}

func NewMetadata(ctx context.Context) *Metadata {
	var accountID string
	if state, err := state.GetState(ctx); err == nil {
		accountID = state.Account.ID.String()
	}
	return &Metadata{
		CLIVersion: version.Print(),
		OS:         runtime.GOOS,
		AccountID:  accountID,
	}
}

func (m *Metadata) SetCobraCmd(cmd *cobra.Command) {
	m.Cmd = cmd.CommandPath()
}

func (m *Metadata) Event() inngestgo.Event {
	return inngestgo.Event{
		Name: EventName,
		Data: map[string]any{
			"account_id":  m.AccountID,
			"cli_version": m.CLIVersion,
			"cmd":         m.Cmd,
			"os":          m.OS,
			"context":     m.Context,
		},
		Timestamp: inngestgo.Now(),
		Version:   "2022-06-01",
	}
}

func Send(ctx context.Context, m *Metadata) {
	if Disabled() || m == nil {
		return
	}

	wg.Add(1)
	go func() {
		_ = client.Send(ctx, m.Event())
		defer wg.Done()
	}()
}

func Wait() {
	wg.Wait()
}

// Disabled returns whether telemetry is disabled.
func Disabled() bool {
	return os.Getenv("DO_NOT_TRACK") != ""
}
