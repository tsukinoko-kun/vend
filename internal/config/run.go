package config

import (
	"errors"
	"os"
	"os/exec"

	"github.com/kballard/go-shellquote"
)

func run(command string, additionalArgs []string) error {
	args, err := shellquote.Split(os.ExpandEnv(command))
	if err != nil {
		return errors.Join(errors.New("unable to split command"), err)
	}

	if len(args) == 0 {
		return errors.New("no command provided")
	}

	cmd := exec.Command(args[0], append(args[1:], additionalArgs...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return errors.Join(errors.New("command execution failed"), err)
	}
	return nil
}
