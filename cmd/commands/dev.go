package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngestctl/pkg/devserver"
	log "github.com/inngest/inngestctl/pkg/logger"
	"github.com/spf13/cobra"
)

var (
	prettyOutput bool
	jsonOutput   bool
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the inngest dev server",
		Example: "inngest dev -p 1234",
		Run:     doDev,
	}

	cmd.Flags().StringP("port", "p", "9999", "port to run the API on")
	cmd.Flags().String("dir", ".", "directory to load functions from")
	cmd.Flags().BoolVar(&prettyOutput, "pretty", false, "pretty print the JSON output")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "pretty print the JSON output")
	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	devserver, err := devserver.NewDevServer(devserver.Options{
		Port: cmd.Flag("port").Value.String(),
		Dir:  cmd.Flag("dir").Value.String(),
		Log:  log.Default(),
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
