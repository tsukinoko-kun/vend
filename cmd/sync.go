package cmd

import (
	"fmt"
	"os"
	"vend/internal/config"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Aliases: []string{"install"},
	Short:   "Install all sources",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error loading config:", err)
			return
		}

		fmt.Println("Installing sources...")

		c.Sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
