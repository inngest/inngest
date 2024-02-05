package clistate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/google/uuid"
	"github.com/inngest/inngest/pkg/inngest/client"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	ErrNoState = fmt.Errorf("no Inngest state found")

	prodFlag    = "prod"
	prodEnvVars = []string{"ENV", "NODE_ENV", "ENVIRONMENT"}
)

const (
	SettingRanInit = "ranInit"
	EnvApiKey      = "INNGEST_AUTH_TOKEN"
)

func init() {
	ctx := context.Background()
	state, err := GetState(ctx)
	// Always create a new client ID when the CLI runs.
	if err == ErrNoState {
		_ = State{ClientID: uuid.New()}.Persist(ctx)
		return
	}
	if err != nil {
		_ = State{ClientID: uuid.New()}.Persist(ctx)
		return
	}
	// Ensure we have a client ID.
	if state.ClientID == uuid.Nil {
		state.ClientID = uuid.New()
		_ = state.Persist(ctx)
	}
}

func SaveSetting(ctx context.Context, key string, value interface{}) error {
	s, _ := GetState(ctx)
	if s == nil {
		s = &State{Settings: make(map[string]interface{})}
	}
	if s.Settings == nil {
		s.Settings = make(map[string]interface{})
	}
	s.Settings[key] = value
	return s.Persist(ctx)
}

func GetSetting(ctx context.Context, key string) interface{} {
	s, _ := GetState(ctx)
	if s == nil {
		return nil
	}
	setting, ok := s.Settings[key]
	if !ok {
		return nil
	}
	return setting
}

func Clear(ctx context.Context) error {
	return (State{ClientID: uuid.New()}).Persist(ctx)
}

// State persists across each cli invokation, allowing functionality such as workspace
// switching, etc.
type State struct {
	client.Client `json:"-"`

	ClientID    uuid.UUID              `json:"clientID"`
	Credentials []byte                 `json:"credentials"`
	Account     client.Account         `json:"account"`
	Settings    map[string]interface{} `json:"settings"`
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

// Client returns an API client, attempting to use authentication from
// state if found.
func Client(ctx context.Context) client.Client {
	state, _ := GetState(ctx)
	if state != nil {
		return state.Client
	}
	return client.New()
}

func ClientID(ctx context.Context) *uuid.UUID {
	state, err := GetState(ctx)
	if err != nil {
		return nil
	}
	return &state.ClientID
}

func AccountID(ctx context.Context) *uuid.UUID {
	state, err := GetState(ctx)
	if err != nil || state.Account.ID == (uuid.UUID{}) {
		return nil
	}
	return &state.Account.ID
}

func AccountIdentifier(ctx context.Context) (string, error) {
	if os.Getenv("INNGEST_DSN_PREFIX") != "" {
		return os.Getenv("INNGEST_DSN_PREFIX"), nil
	}

	state, err := GetState(ctx)
	if err != nil {
		return "", err
	}

	// Add your account identifier locally, before finding action versions.
	if state.Account.Identifier.Domain == nil {
		return state.Account.Identifier.DSNPrefix, nil
	}

	return *state.Account.Identifier.Domain, nil
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

	// If we've been given an API key to use, use that instead of whatever we find
	// in the config file above.
	envApiKey := os.Getenv(EnvApiKey)
	if envApiKey != "" {
		state.Credentials = []byte(envApiKey)
	}

	// add the client using our stored credentials.
	state.Client = client.New(
		client.WithCredentials(state.Credentials),
		client.WithAPI(viper.GetString("api")), // "INNGEST_API", set up by commands/root
	)

	return state, nil
}

// IsProd returns whether we're accessing a production environment for the current
// command.  There are two ways to specify production:  a global --prod flag, or
// by setting the "ENV", "NODE_ENV", or "ENVIRONMENT" env vars to "production".
func IsProd() bool {
	if viper.GetBool(prodFlag) {
		return true
	}
	for _, f := range prodEnvVars {
		if os.Getenv(f) == "production" {
			return true
		}
	}
	return false
}

func RequireState(ctx context.Context) *State {
	state, err := GetState(ctx)
	if err == ErrNoState {
		fmt.Println("\nRun `inngestctl login` and log in before running this command.")
		os.Exit(1)
	}

	if err != nil {
		log.From(ctx).Fatal().Msgf("error reading state: %s", err.Error())
	}

	return state
}
