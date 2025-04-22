package cmd

import (
	"vend/internal/update"

	"github.com/spf13/cobra"
)

var (
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "A brief description of your command",
		Run: func(cmd *cobra.Command, args []string) {
			update.Update(forceUpdate)
		},
	}
	forceUpdate bool
)

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update")
}
