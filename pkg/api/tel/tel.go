package tel

import (
	"context"
	"os"
	"runtime"
	"sync"

	"github.com/inngest/inngest/pkg/inngest/clistate"
	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/inngest/inngestgo"
	"github.com/spf13/cobra"
)

const (
	EventName   = "cli/command.executed"
	CIEventName = "cli/ci.command.executed"
)

var (
	key = "5_Jx-3FAkDMeddntV-KlZy1sbjY8UU1cqn2viGMPlv9Gq-0tWYaukPkUVbD04Zo-1SO2AF2dwnMv7rcHyhJzVQ"
)

var (
	client = inngestgo.NewClient(inngestgo.ClientOpts{
		EventKey: &key,
	})
	wg *sync.WaitGroup
)

func init() {
	wg = &sync.WaitGroup{}
}

// Metadata holds telemetry context.
type Metadata struct {
	AccountID  string         `json:"account_id"`
	DeviceID   string         `json:"device_id"`
	CLIVersion string         `json:"cli_version"`
	Cmd        string         `json:"cmd"`
	OS         string         `json:"os"` // the OS of the system
	Context    map[string]any `json:"context"`
}

func NewMetadata(ctx context.Context) *Metadata {
	var accountID string
	var deviceID string
	// Set account ID if not empty
	if clistate.AccountID(ctx) != nil {
		accountID = clistate.AccountID(ctx).String()
	}
	if clistate.ClientID(ctx) != nil {
		deviceID = clistate.ClientID(ctx).String()
	}
	return &Metadata{
		CLIVersion: version.Print(),
		OS:         runtime.GOOS,
		AccountID:  accountID,
		DeviceID:   deviceID,
		Context:    map[string]any{},
	}
}

func (m *Metadata) SetCobraCmd(cmd *cobra.Command) {
	m.Cmd = cmd.CommandPath()
}

func (m *Metadata) Event() inngestgo.Event {
	name := EventName
	if isCI() {
		name = CIEventName
	}
	return inngestgo.Event{
		Name: name,
		Data: map[string]any{
			"account_id":  m.AccountID,
			"device_id":   m.DeviceID,
			"cli_version": m.CLIVersion,
			"cmd":         m.Cmd,
			"os":          m.OS,
			"context":     m.Context,
		},
		Timestamp: inngestgo.Now(),
		Version:   "2022-12-16",
	}
}

func SendMetadata(ctx context.Context, m *Metadata) {
	Send(ctx, m.Event())
}
func SendEvent(ctx context.Context, name string, m *Metadata) {
	if isCI() {
		return
	}
	evt := m.Event()
	evt.Name = name
	Send(ctx, evt)
}

func Send(ctx context.Context, e inngestgo.Event) {
	if Disabled() {
		return
	}

	wg.Add(1)
	go func() {
		_, _ = client.Send(ctx, e)
		defer wg.Done()
	}()
}

func Wait() {
	wg.Wait()
}

func isCI() bool {
	return os.Getenv("CI") == "true"
}

// Disabled returns whether telemetry is disabled.
func Disabled() bool {
	if version.Version == "dev" && version.Hash == "" {
		return true
	}
	return os.Getenv("DO_NOT_TRACK") != ""
}
