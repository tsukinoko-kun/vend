package cmd

import (
	"fmt"
	"vend/internal/update"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of vend",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(update.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
