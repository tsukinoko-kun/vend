package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"vend/internal/config"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a script",
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

		switch len(args) {
		case 0:
			fmt.Fprintln(os.Stderr, "no script provided")
			return
		case 1:
			if err := c.Run(args[0], nil); err != nil {
				fmt.Fprintln(os.Stderr, "error running script:", err)
			}
			return
		default:
			if err := c.Run(args[0], args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, "error running script:", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
