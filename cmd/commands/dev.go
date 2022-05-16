package commands

import (
	"fmt"
	"os"

	"github.com/inngest/inngestctl/pkg/api"
	"github.com/spf13/cobra"
)

var (
	prettyOutput bool
)

func NewCmdDev() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run the inngest dev server",
		Example: "inngest dev -p 1234",
		Run:     doDev,
	}

	cmd.Flags().StringP("port", "p", "9999", "port to run the API on")
	cmd.Flags().BoolVar(&prettyOutput, "pretty", false, "pretty print the JSON output")
	return cmd
}

func doDev(cmd *cobra.Command, args []string) {
	err := api.NewAPI(api.Opts{
		Port:         cmd.Flag("port").Value.String(),
		PrettyOutput: prettyOutput,
		EventHandler: api.BasicEventHandler,
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
