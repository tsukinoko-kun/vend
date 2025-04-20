package cmd

import (
	"fmt"
	"os"
	"path/filepath"
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

			dir := filepath.Dir(c.Location)
			if err := os.Chdir(dir); err != nil {
				fmt.Fprintf(os.Stderr, "failed to change directory into %s: %v\n", dir, err)
				return
			}

			if initFromGitsubmodules {
				if err := c.FromGitSubmodules(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to initialize vend project from .gitmodules file: %v\n", err)
					os.Exit(1)
				}
			}

			if err := c.Save(); err != nil {
				fmt.Fprintln(os.Stderr, "failed to save config:", err)
				return
			}
		},
	}
)

func init() {
	initCmd.Flags().BoolVar(&initFromGitsubmodules, "from-gitsubmodules", false, "Initialize vend project from .gitmodules file")
	rootCmd.AddCommand(initCmd)
}
