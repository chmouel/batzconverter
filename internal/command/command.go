package command

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
)

type Runner interface {
	LookPath(file string) (string, error)
	Run(ctx context.Context, name string, args []string, env []string, stdin io.Reader) ([]byte, []byte, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (OSRunner) Run(ctx context.Context, name string, args []string, env []string, stdin io.Reader) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
