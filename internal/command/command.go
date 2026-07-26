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
	cmd.Env = mergeEnvironment(os.Environ(), env)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func mergeEnvironment(base, overrides []string) []string {
	keys := make(map[string]struct{}, len(overrides))
	for _, value := range overrides {
		if separator := bytes.IndexByte([]byte(value), '='); separator >= 0 {
			keys[value[:separator]] = struct{}{}
		}
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, value := range base {
		separator := bytes.IndexByte([]byte(value), '=')
		if separator < 0 {
			result = append(result, value)
			continue
		}
		if _, replaced := keys[value[:separator]]; !replaced {
			result = append(result, value)
		}
	}
	return append(result, overrides...)
}
