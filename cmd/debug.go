package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmdDebug(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "debug",
		Aliases: []string{"dbg"},
		Short:   "Debug commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Not implemented")
		},
	}

	return cmd
}
