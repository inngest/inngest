package commands

import (
	"syscall"

	"github.com/inngest/inngestctl/inngest"
	"github.com/inngest/inngestctl/inngest/client"
	"github.com/inngest/inngestctl/inngest/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	username string
	password string
)

func init() {
	rootCmd.AddCommand(login)

	login.Flags().StringVarP(&username, "username", "u", "", "your email address")
	login.Flags().StringVarP(&password, "password", "p", "", "your password (optional, read from TTY if blank)")
}

var login = &cobra.Command{
	Use:   "login",
	Short: "Logs in to your Inngest account",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		if username == "" {
			log.From(ctx).Fatal().Msgf("No username found.  Supply with the -u flag")
		}

		if password == "" {
			log.From(ctx).Info().Msg("Enter your password: ")
			byt, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.From(ctx).Fatal().Msgf("unable to read password: %s", err.Error())
			}
			password = string(byt)
		}

		log.From(ctx).Info().Msg("Logging in")
		jwt, err := client.New().Login(ctx, username, password)
		if err != nil {
			log.From(ctx).Fatal().Msgf("unable to log in: %s", err.Error())
		}

		state := inngest.State{Credentials: jwt}
		if err := state.Persist(ctx); err != nil {
			log.From(ctx).Fatal().Msgf("unable to log in: %s", err.Error())
		}

		log.From(ctx).Info().Msg("Logged in")
	},
}
