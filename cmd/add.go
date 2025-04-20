package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"vend/internal/config"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a source",
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

		for _, source := range args {
			if err := c.Add(source); err != nil {
				fmt.Fprintf(os.Stderr, "error adding source %s: %v\n", source, err)
			}
		}

		if err := c.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "error saving config:", err)
			return
		}

		c.Sync()
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
