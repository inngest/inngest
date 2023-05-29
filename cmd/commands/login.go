package commands

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/inngest/inngest/pkg/cli"
	"github.com/inngest/inngest/pkg/inngest/client"
	"github.com/inngest/inngest/pkg/inngest/clistate"
	"github.com/inngest/inngest/pkg/inngest/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	username string
	password string
	token    string
)

func NewCmdLogin() *cobra.Command {
	login := &cobra.Command{
		Use:   "login",
		Short: "Logs in to your Inngest account",
		Run:   doLogin,
	}
	login.Flags().StringVarP(&username, "username", "u", "", "Username (email address).  If blank will use a device login flow.")
	login.Flags().StringVarP(&password, "password", "p", "", "Password (optional, only used when username is provided and read from TTY if blank)")
	login.Flags().StringVarP(&token, "token", "t", "", "Token (optional, if specified will log in using a static API key)")
	return login
}

func doLogin(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	if username != "" {
		userPassAuth(ctx)
	} else if token != "" {
		tokenAuth(ctx)
	} else {
		DeviceAuth(ctx)
	}

	if err := fetchAccount(cmd.Context()); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("unable to log in: %s", err.Error())))
		os.Exit(1)
	}

	fmt.Println(cli.BoldStyle.Render("Logged in."))
	fmt.Println("")
}

func tokenAuth(ctx context.Context) {
	state := clistate.RequireState(ctx)
	state.Credentials = []byte(token)

	if err := state.Persist(ctx); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", err)))
		os.Exit(1)
	}
}

func DeviceAuth(ctx context.Context) {
	state := clistate.RequireState(ctx)
	start, err := state.Client.StartDeviceLogin(ctx, state.ClientID)
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("unable to log in: %s", err.Error())))
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Println(cli.TextStyle.Render("To sign in to the CLI, open the following URL:"))
	fmt.Println(cli.BoldStyle.Render(start.VerificationURL))
	fmt.Println("")
	fmt.Println(cli.TextStyle.Render("And enter the code:"))
	fmt.Println(cli.BoldStyle.Render(start.UserCode))
	fmt.Println("")

	var resp *client.DeviceLoginResponse
	for resp == nil || (resp != nil && resp.Error == "authorization_pending") {
		<-time.After(time.Second)
		resp, err = state.Client.PollDeviceLogin(ctx, state.ClientID, start.DeviceCode)
		if err != nil {
			fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", err.Error())))
			os.Exit(1)
		}
	}
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", err.Error())))
		os.Exit(1)
	}
	if resp.Error != "" {
		fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", resp.Error)))
		os.Exit(1)
	}

	state.Credentials = []byte(resp.AccessToken)
	if err := state.Persist(ctx); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", resp.Error)))
		os.Exit(1)
	}
}

func userPassAuth(ctx context.Context) {
	if password == "" {
		fmt.Printf("Enter your password: ")
		byt, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.From(ctx).Fatal().Msgf("unable to read password: %s", err.Error())
		}
		password = string(byt)
		fmt.Println("")
	}

	fmt.Println("Logging in...")
	jwt, err := client.New().Login(ctx, username, password)
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("unable to log in: %s", err.Error())))
		os.Exit(1)
	}
	state := clistate.State{
		Credentials: jwt,
	}
	if err := state.Persist(ctx); err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("Unable to log in: %s", err)))
		os.Exit(1)
	}
}

func fetchAccount(ctx context.Context) error {
	state := clistate.RequireState(ctx)
	account, err := state.Client.Account(ctx)
	if err != nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("unable to log in: %s", err.Error())))
		os.Exit(1)
	}
	if account == nil {
		fmt.Println(cli.RenderError(fmt.Sprintf("unable to log in: %s", err.Error())))
		os.Exit(1)
	}
	state.Account = *account
	return state.Persist(ctx)
}
