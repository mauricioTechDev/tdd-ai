package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the version",
	Example: `  tdd-ai version`,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "tdd-ai %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
