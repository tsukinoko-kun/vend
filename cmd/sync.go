package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

		dir := filepath.Dir(c.Location)
		if err := os.Chdir(dir); err != nil {
			fmt.Fprintf(os.Stderr, "failed to change directory into %s: %v\n", dir, err)
			return
		}

		c.Sync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
