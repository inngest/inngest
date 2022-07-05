package commands

import (
	"fmt"

	"github.com/inngest/inngest-cli/pkg/api"
	"github.com/inngest/inngest-cli/pkg/server"
	"github.com/spf13/cobra"
)

const (
	ServeExecutor = "executor"
	ServeEventAPI = "events"
)

var (
	serveArgs = []string{ServeExecutor, ServeEventAPI}
)

func NewCmdServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "serve [component]",
		Short:     "Start an Inngest server",
		Example:   "inngestctl serve executor",
		RunE:      serve,
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: serveArgs,
	}
	return cmd
}

func serve(cmd *cobra.Command, args []string) error {
	var s server.Server

	switch args[0] {
	case ServeEventAPI:
		s = api.NewServer()
	default:
		return fmt.Errorf("Not implemented")
	}

	return server.Start(cmd.Context(), s)
}
