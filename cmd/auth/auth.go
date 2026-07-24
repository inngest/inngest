package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	inncli "github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/inngest/client"
	"github.com/inngest/inngest/pkg/inngest/clistate"
	"github.com/urfave/cli/v3"
)

const expiredMsg = "The login code expired. Run `inngest auth login` to start again."

func Command() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Log in to and manage your Inngest Cloud account",
		Commands: []*cli.Command{
			{
				Name:   "login",
				Usage:  "Log in to your Inngest Cloud account",
				Action: login,
			},
			{
				Name:   "logout",
				Usage:  "Log out of your Inngest Cloud account",
				Action: logout,
			},
			{
				Name:   "whoami",
				Usage:  "Show the currently logged in account",
				Action: whoami,
			},
		},
	}
}

func login(ctx context.Context, cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	clientID := clistate.ClientID(ctx)
	if clientID == nil || *clientID == uuid.Nil {
		return errors.New("unable to read the CLI client ID from ~/.config/inngest")
	}

	api := client.New()
	start, err := api.StartDeviceLogin(ctx, *clientID)
	if err != nil {
		return fmt.Errorf("unable to start login: %w", err)
	}

	fmt.Println()
	fmt.Println(inncli.TextStyle.Render("To log in, open the following URL in your browser:"))
	fmt.Println(inncli.BoldStyle.Render("  " + start.VerificationURL))
	fmt.Println()
	fmt.Println(inncli.TextStyle.Render("And confirm that it shows this code:"))
	fmt.Println(inncli.BoldStyle.Render("  " + start.UserCode))
	fmt.Println()

	resp, err := pollForToken(ctx, api, *clientID, start)
	if err != nil {
		return err
	}

	state, _ := clistate.GetState(ctx)
	if state == nil {
		state = &clistate.State{ClientID: *clientID}
	}
	state.Credentials = []byte(resp.AccessToken)
	state.Account = client.Account{ID: resp.AccountID, Name: resp.AccountName}
	state.Env = resp.Env
	if err := state.Persist(ctx); err != nil {
		return fmt.Errorf("unable to save credentials: %w", err)
	}

	if resp.AccountName == "" {
		fmt.Println(inncli.BoldStyle.Render("Logged in."))
		return nil
	}
	msg := fmt.Sprintf("Logged in as %s.", resp.AccountName)
	if resp.Env != "" {
		msg = fmt.Sprintf("Logged in as %s, using the %s environment.", resp.AccountName, resp.Env)
	}
	fmt.Println(inncli.BoldStyle.Render(msg))
	return nil
}

func pollForToken(ctx context.Context, api client.Client, clientID uuid.UUID, start *client.StartDeviceLoginResponse) (*client.DeviceLoginResponse, error) {
	interval := time.Duration(start.Interval) * time.Second
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if start.ExpiresIn > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(start.ExpiresIn)*time.Second)
		defer cancel()
	}

	for {
		resp, err := poll(ctx, api, clientID, start.DeviceCode)
		if err != nil {
			// Ctrl-C or the expiry deadline is fatal; any other error (network
			// blip, rate limit, non-JSON gateway response) is transient — keep
			// polling until the deadline rather than failing the whole login.
			if ctx.Err() != nil {
				return nil, ctxErr(ctx)
			}
			if err := wait(ctx, interval); err != nil {
				return nil, err
			}
			continue
		}

		switch resp.Error {
		case "":
			if resp.AccessToken == "" {
				// Well-formed but missing a token: retry rather than persist an
				// empty credential.
				if err := wait(ctx, interval); err != nil {
					return nil, err
				}
				continue
			}
			return resp, nil
		case "authorization_pending", "server_error":
			if err := wait(ctx, interval); err != nil {
				return nil, err
			}
		case "expired_token":
			return nil, errors.New(expiredMsg)
		case "access_denied":
			return nil, errors.New("Login was denied. Run `inngest auth login` to try again.")
		default:
			return nil, fmt.Errorf("unable to log in: %s", resp.Error)
		}
	}
}

// wait sleeps for d, or returns a login error if the context is cancelled or
// the expiry deadline elapses first.
func wait(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctxErr(ctx)
	case <-time.After(d):
		return nil
	}
}

// poll runs a single PollDeviceLogin call in a goroutine so that ctx
// cancellation (Ctrl-C, code expiry) is honored while the server long-polls:
// the client's requests do not carry the context.
func poll(ctx context.Context, api client.Client, clientID, deviceCode uuid.UUID) (*client.DeviceLoginResponse, error) {
	type result struct {
		resp *client.DeviceLoginResponse
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		resp, err := api.PollDeviceLogin(ctx, clientID, deviceCode)
		ch <- result{resp, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctxErr(ctx)
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("unable to log in: %w", r.err)
		}
		return r.resp, nil
	}
}

func ctxErr(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New(expiredMsg)
	}
	return errors.New("login canceled")
}

func logout(ctx context.Context, cmd *cli.Command) error {
	state, err := clistate.GetState(ctx)
	if err == nil {
		state.Credentials = nil
		state.Account = client.Account{}
		state.Env = ""
		if err := state.Persist(ctx); err != nil {
			return fmt.Errorf("unable to clear credentials: %w", err)
		}
	}
	fmt.Println(inncli.TextStyle.Render("Logged out."))
	return nil
}

func whoami(ctx context.Context, cmd *cli.Command) error {
	state, err := clistate.GetState(ctx)
	if err != nil {
		fmt.Println(inncli.TextStyle.Render("Not logged in. Run `inngest auth login`."))
		return cli.Exit("", 1)
	}
	if len(state.Credentials) == 0 || state.Account.ID == uuid.Nil {
		fmt.Println(inncli.TextStyle.Render("Not logged in. Run `inngest auth login`."))
		return cli.Exit("", 1)
	}

	fmt.Println(inncli.TextStyle.Render(fmt.Sprintf("Logged in as %s.", state.Account.Name)))
	fmt.Println(inncli.FeintStyle.Render(fmt.Sprintf("Account ID: %s", state.Account.ID)))
	if state.Env != "" {
		fmt.Println(inncli.FeintStyle.Render(fmt.Sprintf("Environment: %s", state.Env)))
	}
	if os.Getenv(clistate.EnvApiKey) != "" {
		fmt.Println(inncli.RenderWarning(fmt.Sprintf("%s is set and overrides the stored credentials.", clistate.EnvApiKey)))
	}
	return nil
}
