package cmd

import (
	"fmt"
	"os"
	"vend/internal/config"

	"github.com/spf13/cobra"
)

var (
	initFromGitsubmodules = false

	initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize a new vend project",
		Run: func(cmd *cobra.Command, args []string) {
			c := config.New()
			if initFromGitsubmodules {
				if err := c.FromGitSubmodules(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to initialize vend project from .gitmodules file: %v\n", err)
					os.Exit(1)
				}
			}
			c.Save()
		},
	}
)

func init() {
	initCmd.Flags().BoolVar(&initFromGitsubmodules, "from-gitsubmodules", false, "Initialize vend project from .gitmodules file")
	rootCmd.AddCommand(initCmd)
}
