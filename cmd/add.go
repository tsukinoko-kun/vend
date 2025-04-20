package cmd

import (
	"fmt"
	"os"
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

		for _, source := range args {
			c.Add(source)
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
