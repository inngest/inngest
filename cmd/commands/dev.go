package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngest-cli/pkg/devserver"
	log "github.com/inngest/inngest-cli/pkg/logger"
	"github.com/spf13/cobra"
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the inngest dev server",
		Example: "inngest dev -p 1234",
		Run:     doDev,
	}

	cmd.Flags().String("host", "", "host to run the API on")
	cmd.Flags().StringP("port", "p", "9999", "port to run the API on")
	cmd.Flags().String("dir", ".", "directory to load functions from")
	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	devserver, err := devserver.NewDevServer(devserver.Options{
		Hostname: cmd.Flag("host").Value.String(),
		Port:     cmd.Flag("port").Value.String(),
		Dir:      cmd.Flag("dir").Value.String(),
		Log:      log.Default(),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = devserver.Start(ctx)
	if err != nil {
		os.Exit(1)
	}
}
