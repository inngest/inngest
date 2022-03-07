package commands

import (
	"fmt"
	"syscall"

	"github.com/inngest/inngestctl/cmd/commands/internal/state"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var (
	username string
	password string
)

func NewCmdLogin() *cobra.Command {
	login := &cobra.Command{
		Use:   "login",
		Short: "Logs in to your Inngest account",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			if username == "" {
				fmt.Printf("Your email: ")
				_, _ = fmt.Scanln(&username)
			}

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
			jwt, err := client.New(client.WithAPI(viper.GetString("api"))).Login(ctx, username, password)
			if err != nil {
				log.From(ctx).Fatal().Msgf("unable to log in: %s", err.Error())
			}

			// Fetch the account.
			client := client.New(
				client.WithAPI(viper.GetString("api")),
				client.WithCredentials(jwt),
			)
			account, err := client.Account(ctx)
			if err != nil {
				log.From(ctx).Fatal().Msgf("unable to fetch account: %s", err.Error())
			}

			// Find their workspaces, and select the default workspace.
			workspaces, err := client.Workspaces(ctx)
			if err != nil {
				log.From(ctx).Fatal().Msgf("unable to fetch workspaces: %s", err.Error())
			}

			state := state.State{
				Credentials: jwt,
				Account:     *account,
			}

			for _, item := range workspaces {
				if item.Name == "default" && !item.Test {
					_ = state.SetWorkspace(ctx, item)
				}
			}

			if err := state.Persist(ctx); err != nil {
				log.From(ctx).Fatal().Msgf("unable to log in: %s", err.Error())
			}

			log.From(ctx).Info().Msg("Logged in")
		},
	}

	login.Flags().StringVarP(&username, "username", "u", "", "your email address")
	login.Flags().StringVarP(&password, "password", "p", "", "your password (optional, read from TTY if blank)")
	return login
}
