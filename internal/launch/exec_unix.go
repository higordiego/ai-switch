//go:build darwin || linux || freebsd || openbsd || netbsd

package launch

import (
	"fmt"
	"os/exec"
	"syscall"
)

func Resolve(program string) (string, error) {
	path, err := exec.LookPath(program)
	if err != nil && program == "cursor-agent" {
		path, err = exec.LookPath("agent")
	}
	if err != nil {
		return "", fmt.Errorf("%s nao encontrado no PATH: %w", program, err)
	}
	return path, nil
}

func Execute(plan Plan) error {
	cmd, err := Command(plan)
	if err != nil {
		return err
	}
	// Replacing the process preserves its PTY, job control, signals and exit status.
	syscall.Umask(0077)
	return syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
}

// Command prepares the same isolated process used by Execute. It is also used
// by the interactive screen, which temporarily hands the terminal to a tool.
func Command(plan Plan) (*exec.Cmd, error) {
	path, err := Resolve(plan.Program)
	if err != nil {
		return nil, err
	}
	return &exec.Cmd{Path: path, Args: append([]string{plan.Program}, plan.Args...), Env: plan.Env}, nil
}
