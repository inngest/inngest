package inngest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	ErrNoState = fmt.Errorf("no Inngest state found")
)

// State persists across each cli invokation, allowing functionality such as workspace
// switching, etc.
type State struct {
	client.Client `json:"-"`

	SelectedWorkspace *Workspace `json:"workspace,omitempty"`
	Credentials       []byte     `json:"credentials"`
}

func (s State) Persist(ctx context.Context) error {
	path, err := homedir.Expand("~/.config/inngest")
	if err != nil {
		return fmt.Errorf("error reading ~/.config/inngest")
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("error creating ~/.config/inngest")
	}

	byt, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling state: %w", err)
	}

	path, err = homedir.Expand("~/.config/inngest/state")
	if err != nil {
		return fmt.Errorf("error reading ~/.config/inngest")
	}

	return ioutil.WriteFile(path, byt, 0600)
}

func (s *State) SetWorkspace(ctx context.Context, w client.Workspace) error {
	s.SelectedWorkspace = &Workspace{Workspace: w}
	return s.Persist(ctx)
}

// Workspace represents a single workspace within an Inngest account. The pertinent
// fields for the active workspace are marshalled into State.
type Workspace struct {
	client.Workspace

	IsOverridden bool `json:"-"`
}

func GetState(ctx context.Context) (*State, error) {
	path, err := homedir.Expand("~/.config/inngest")
	if err != nil {
		return nil, fmt.Errorf("error reading ~/.config/inngest")
	}

	dir := os.DirFS(path)
	byt, err := fs.ReadFile(dir, "state")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoState
	}

	state := &State{}
	if err := json.Unmarshal(byt, state); err != nil {
		return nil, fmt.Errorf("invalid state file: %w", err)
	}

	// add the client using our stored credentials.
	state.Client = client.New(
		client.WithCredentials(state.Credentials),
		client.WithAPI(viper.GetString("api")), // "INNGEST_API", set up by commands/root
	)

	wid := viper.GetString("workspace.id")
	if wid == "" {
		return state, nil
	}

	id, err := uuid.Parse(wid)
	if err != nil {
		log.From(ctx).Warn().Err(err).Msg("invalid WORKSPACE_ID uuid")
		return state, nil
	}

	state.SelectedWorkspace = &Workspace{Workspace: client.Workspace{ID: id}, IsOverridden: true}
	return state, nil
}

func RequireState(ctx context.Context) *State {
	state, err := GetState(ctx)
	if err == ErrNoState {
		log.From(ctx).Fatal().Msg("no Inngest state found. Run `inngestctl login` to log in.")
	}

	if err != nil {
		log.From(ctx).Fatal().Msgf("error reading state: %s", err.Error())
	}
	return state
}
