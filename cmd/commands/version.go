package commands

import (
	"fmt"

	"github.com/inngest/inngest/pkg/inngest/version"
	"github.com/spf13/cobra"
)

func NewCmdVersion() *cobra.Command {
	return &cobra.Command{
		Use: "version",
		Short: fmt.Sprintf(
			"Shows the inngest CLI version (saving time, it's: %s)",
			version.Print(),
		),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.Print())
		},
	}
}
